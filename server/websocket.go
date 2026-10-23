package main

import (
	"log"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type WebSocketMessage struct {
	Type      string  `json:"type"`
	AccountID int64   `json:"account_id,omitempty"`
	LoginName string  `json:"login_name,omitempty"`
	Message   string  `json:"message,omitempty"`
	X         float64 `json:"x,omitempty"`
	Y         float64 `json:"y,omitempty"`
	Z         float64 `json:"z,omitempty"`
	RotationY float64 `json:"rotation_y,omitempty"`
	SentAt    string  `json:"sent_at,omitempty"`
}

type Client struct {
	hub        *Hub
	connection *websocket.Conn
	send       chan WebSocketMessage
	remoteAddr string
	accountID  int64
	loginName  string

	hasMoveState bool
	lastMoveX    float64
	lastMoveY    float64
	lastMoveZ    float64
	lastMoveAt   time.Time

	sendMu     sync.Mutex
	sendClosed bool
}

func (client *Client) sendSafely(
	message WebSocketMessage,
) bool {
	client.sendMu.Lock()
	defer client.sendMu.Unlock()

	if client.sendClosed {
		return false
	}

	select {
	case client.send <- message:
		return true
	default:
		return false
	}
}

func (client *Client) closeSend() {
	client.sendMu.Lock()
	defer client.sendMu.Unlock()

	if client.sendClosed {
		return
	}

	client.sendClosed = true
	close(client.send)
}

const (
	maxPlayerCoordinate = 1000.0
	maxPlayerSpeed      = 8.0
	minMoveInterval     = 50 * time.Millisecond

	pongWait   = 60 * time.Second
	pingPeriod = 30 * time.Second
	writeWait  = 10 * time.Second
)

func isFiniteNumber(value float64) bool {
	return !math.IsNaN(value) &&
		!math.IsInf(value, 0)
}

func (client *Client) validatePlayerMove(
	message WebSocketMessage,
) bool {
	if !isFiniteNumber(message.X) ||
		!isFiniteNumber(message.Y) ||
		!isFiniteNumber(message.Z) ||
		!isFiniteNumber(message.RotationY) {
		return false
	}

	if math.Abs(message.X) > maxPlayerCoordinate ||
		math.Abs(message.Y) > maxPlayerCoordinate ||
		math.Abs(message.Z) > maxPlayerCoordinate {
		return false
	}

	now := time.Now()

	if !client.hasMoveState {
		client.hasMoveState = true
		client.lastMoveX = message.X
		client.lastMoveY = message.Y
		client.lastMoveZ = message.Z
		client.lastMoveAt = now

		return true
	}

	elapsed :=
		now.Sub(client.lastMoveAt)

	if elapsed < minMoveInterval {
		return false
	}

	deltaX :=
		message.X - client.lastMoveX

	deltaY :=
		message.Y - client.lastMoveY

	deltaZ :=
		message.Z - client.lastMoveZ

	distance :=
		math.Sqrt(
			deltaX*deltaX +
				deltaY*deltaY +
				deltaZ*deltaZ,
		)

	maxDistance :=
		maxPlayerSpeed*elapsed.Seconds() +
			0.5

	if distance > maxDistance {
		return false
	}

	client.lastMoveX = message.X
	client.lastMoveY = message.Y
	client.lastMoveZ = message.Z
	client.lastMoveAt = now

	return true
}

type Hub struct {
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan WebSocketMessage
}

func newHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan WebSocketMessage),
	}
}

var websocketHub = newHub()

func (hub *Hub) run() {
	for {
		select {
		case client := <-hub.register:
			// すでに接続しているプレイヤーを
			// 新しく入ってきたクライアントへ通知する。
			for existingClient := range hub.clients {
				client.send <- WebSocketMessage{
					Type:      "player.join",
					AccountID: existingClient.accountID,
					LoginName: existingClient.loginName,
					SentAt: time.Now().
						UTC().
						Format(time.RFC3339),
				}
			}

			hub.clients[client] = true

			log.Printf(
				"websocket client registered: remote=%s total=%d",
				client.remoteAddr,
				len(hub.clients),
			)

		case client := <-hub.unregister:
			if _, exists := hub.clients[client]; exists {
				delete(hub.clients, client)
				client.closeSend()
				log.Printf(
					"websocket client unregistered: remote=%s total=%d",
					client.remoteAddr,
					len(hub.clients),
				)
			}

		case message := <-hub.broadcast:
			for client := range hub.clients {
				select {
				case client.send <- message:

				default:
					// 送信キューが詰まったクライアントは
					// Hubから外して接続を閉じる。
					delete(hub.clients, client)
					client.closeSend()
					if err := client.connection.Close(); err != nil {
						log.Println(
							"websocket close failed:",
							err,
						)
					}
				}
			}
		}
	}
}

var websocketUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,

	// Unity Editor・Windowsアプリからの接続確認用。
	// 公開時は接続元を制限する。
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// GET /ws
func websocketHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(
			w,
			"method not allowed",
			http.StatusMethodNotAllowed,
		)
		return
	}

	sessionToken :=
		strings.TrimSpace(
			r.URL.Query().Get("token"),
		)

	if sessionToken == "" {
		http.Error(
			w,
			"unauthorized",
			http.StatusUnauthorized,
		)
		return
	}

	accountID, loginName, err :=
		validateSessionToken(sessionToken)

	if err != nil {
		log.Printf(
			"websocket authentication failed: remote=%s error=%v",
			r.RemoteAddr,
			err,
		)

		http.Error(
			w,
			"unauthorized",
			http.StatusUnauthorized,
		)
		return
	}

	connection, err := websocketUpgrader.Upgrade(
		w,
		r,
		nil,
	)
	if err != nil {
		log.Println(
			"websocket upgrade failed:",
			err,
		)
		return
	}

	client := &Client{
		hub:        websocketHub,
		connection: connection,
		send:       make(chan WebSocketMessage, 64),
		remoteAddr: r.RemoteAddr,
		accountID:  accountID,
		loginName:  loginName,
	}

	log.Printf(
		"websocket authenticated: remote=%s account_id=%d login_name=%s",
		client.remoteAddr,
		client.accountID,
		client.loginName,
	)

	client.hub.register <- client

	go client.writePump()

	client.hub.broadcast <- WebSocketMessage{
		Type:      "player.join",
		AccountID: client.accountID,
		LoginName: client.loginName,
		SentAt: time.Now().
			UTC().
			Format(time.RFC3339),
	}

	connectedMessage := WebSocketMessage{
		Type:    "connected",
		Message: "websocket connected",
		SentAt: time.Now().
			UTC().
			Format(time.RFC3339),
	}

	client.sendSafely(connectedMessage)
	// この処理は切断されるまで継続する。
	client.readPump()
}

func (client *Client) readPump() {
	defer func() {
		client.hub.unregister <- client

		client.hub.broadcast <- WebSocketMessage{
			Type:      "player.leave",
			AccountID: client.accountID,
			LoginName: client.loginName,
			SentAt: time.Now().
				UTC().
				Format(time.RFC3339),
		}

		if err := client.connection.Close(); err != nil {
			log.Println(
				"websocket close failed:",
				err,
			)
		}

		log.Printf(
			"websocket disconnected: remote=%s",
			client.remoteAddr,
		)
	}()

	client.connection.SetReadDeadline(
		time.Now().Add(pongWait),
	)

	client.connection.SetPongHandler(
		func(string) error {
			client.connection.SetReadDeadline(
				time.Now().Add(pongWait),
			)

			return nil
		},
	)

	for {
		var receivedMessage WebSocketMessage

		err := client.connection.ReadJSON(
			&receivedMessage,
		)
		if err != nil {
			if websocket.IsUnexpectedCloseError(
				err,
				websocket.CloseNormalClosure,
				websocket.CloseGoingAway,
			) {
				log.Println(
					"websocket read failed:",
					err,
				)
			}

			return
		}

		switch receivedMessage.Type {

		case "chat", "chat.send":
			message :=
				strings.TrimSpace(
					receivedMessage.Message,
				)

			if message == "" {
				continue
			}

			if len([]rune(message)) > 200 {
				client.sendSafely(WebSocketMessage{Type: "error",
					Message: "message is too long",
					SentAt: time.Now().
						UTC().
						Format(time.RFC3339),
				})
				continue
			}

			client.hub.broadcast <- WebSocketMessage{
				Type:      "chat.message",
				AccountID: client.accountID,
				LoginName: client.loginName,
				Message:   message,
				SentAt: time.Now().
					UTC().
					Format(time.RFC3339),
			}

		case "player.move":
			if !client.validatePlayerMove(
				receivedMessage,
			) {
				log.Printf(
					"invalid player move: "+
						"account_id=%d "+
						"x=%.3f y=%.3f z=%.3f rotation_y=%.3f",
					client.accountID,
					receivedMessage.X,
					receivedMessage.Y,
					receivedMessage.Z,
					receivedMessage.RotationY,
				)

				client.sendSafely(WebSocketMessage{Type: "error",
					Message: "invalid player move",
					SentAt: time.Now().
						UTC().
						Format(time.RFC3339),
				})

				continue
			}

			client.hub.broadcast <- WebSocketMessage{
				Type:      "player.move",
				AccountID: client.accountID,
				LoginName: client.loginName,
				X:         receivedMessage.X,
				Y:         receivedMessage.Y,
				Z:         receivedMessage.Z,
				RotationY: receivedMessage.RotationY,
				SentAt: time.Now().
					UTC().
					Format(time.RFC3339),
			}

		default:
			client.sendSafely(WebSocketMessage{Type: "error",
				Message: "unsupported message type",
				SentAt: time.Now().
					UTC().
					Format(time.RFC3339),
			})
		}
	}
}

func (client *Client) writePump() {
	ticker :=
		time.NewTicker(pingPeriod)

	defer func() {
		ticker.Stop()
	}()

	for {
		select {
		case message, ok := <-client.send:
			client.connection.SetWriteDeadline(
				time.Now().Add(writeWait),
			)

			if !ok {
				client.connection.WriteMessage(
					websocket.CloseMessage,
					[]byte{},
				)

				return
			}

			err :=
				client.connection.WriteJSON(
					message,
				)

			if err != nil {
				log.Println(
					"websocket write failed:",
					err,
				)

				return
			}

		case <-ticker.C:
			client.connection.SetWriteDeadline(
				time.Now().Add(writeWait),
			)

			err :=
				client.connection.WriteMessage(
					websocket.PingMessage,
					nil,
				)

			if err != nil {
				log.Println(
					"websocket ping failed:",
					err,
				)

				return
			}
		}
	}
}
