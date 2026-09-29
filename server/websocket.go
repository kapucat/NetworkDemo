package main

import (
	"log"
	"net/http"
	"strings"
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

			log.Printf(
				"websocket client registered: remote=%s total=%d",
				client.remoteAddr,
				len(hub.clients),
			)

		case client := <-hub.unregister:
			if _, exists := hub.clients[client]; exists {
				delete(hub.clients, client)
				close(client.send)

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
					close(client.send)

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

	client.send <- connectedMessage

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
				client.send <- WebSocketMessage{
					Type:    "error",
					Message: "message is too long",
					SentAt: time.Now().
						UTC().
						Format(time.RFC3339),
				}
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
			client.send <- WebSocketMessage{
				Type:    "error",
				Message: "unsupported message type",
				SentAt: time.Now().
					UTC().
					Format(time.RFC3339),
			}
		}
	}
}

func (client *Client) writePump() {
	for message := range client.send {
		err := client.connection.WriteJSON(
			message,
		)
		if err != nil {
			log.Println(
				"websocket write failed:",
				err,
			)
			return
		}
	}
}
