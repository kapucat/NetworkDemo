package main

import (
	"log"      // ログ出力用
	"math"     // 数学関数用
	"net/http" // HTTPサーバー用
	"strings"  // 文字列操作用
	"sync"     // 同期用
	"time"     // 時間操作用

	"github.com/gorilla/websocket" // WebSocket用
)

// WebSocket通信を扱うためのGo言語のコードです。WebSocketは、クライアントとサーバー間で双方向通信を可能にするプロトコル。
// このコードでは、WebSocket接続の管理、メッセージの送受信、クライアントの登録・解除などの機能が実装されている。
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

// Clientは、WebSocket接続を表す構造体です。各クライアントは、Hubに登録され、メッセージの送受信を行います。
type Client struct {
	hub          *Hub                  // Hubへの参照。クライアントはHubを通じて他のクライアントと通信する。
	connection   *websocket.Conn       // WebSocket接続。クライアントとの通信に使用される。
	send         chan WebSocketMessage // 送信キュー。クライアントに送信するメッセージを格納するチャネル。
	remoteAddr   string                // クライアントのリモートアドレス。接続元のIPアドレスやポート番号を保持する。
	accountID    int64                 // クライアントのアカウントID。認証されたユーザーの識別子として使用される。
	loginName    string                // クライアントのログイン名。認証されたユーザーの表示名として使用される。
	hasMoveState bool                  // プレイヤーの移動状態を保持するフラグ。初回の移動メッセージを受信したかどうかを示す。
	lastMoveX    float64               // プレイヤーの最後のX座標。移動メッセージの検証に使用される。
	lastMoveY    float64               // プレイヤーの最後のY座標。移動メッセージの検証に使用される。
	lastMoveZ    float64               // プレイヤーの最後のZ座標。移動メッセージの検証に使用される。
	lastMoveAt   time.Time             // プレイヤーの最後の移動時刻。移動メッセージの検証に使用される。
	sendMu       sync.Mutex            // 送信チャネルの操作を保護するためのミューテックス。複数のゴルーチンからの同時アクセスを防ぐ。
	sendClosed   bool                  // 送信チャネルが閉じられたかどうかを示すフラグ。チャネルが閉じられた場合、送信操作は行わない。
}

// sendSafelyは、クライアントにメッセージを送信する際に、送信チャネルが閉じられていないかを確認し、安全に送信を行います。
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

// closeSendは、クライアントの送信チャネルを閉じるためのメソッドです。送信チャネルがすでに閉じられている場合は何も行いません。
func (client *Client) closeSend() {
	client.sendMu.Lock()
	defer client.sendMu.Unlock()

	if client.sendClosed {
		return
	}

	client.sendClosed = true
	close(client.send)
}

// 定数の定義
const (
	maxPlayerCoordinate = 1000.0                // プレイヤーの座標の最大値。これを超える座標は無効とする。
	maxPlayerSpeed      = 8.0                   // プレイヤーの最大移動速度（単位: メートル/秒）。これを超える移動は無効とする。
	minMoveInterval     = 50 * time.Millisecond // プレイヤーの移動メッセージの最小間隔。これを下回る間隔での移動は無効とする。
	pongWait            = 60 * time.Second      // Pongメッセージの待機時間。これを超えると接続が切断される。
	pingPeriod          = 30 * time.Second      // Pingメッセージの送信間隔。これにより接続が生きているかを確認する。
	writeWait           = 10 * time.Second      // WebSocketへの書き込みのタイムアウト時間。これを超えると書き込みが失敗する。
)

// isFiniteNumberは、与えられた浮動小数点数が有限の数値であるかどうかを判定する関数です。NaNや無限大の場合はfalseを返します。
func isFiniteNumber(value float64) bool {
	return !math.IsNaN(value) &&
		!math.IsInf(value, 0)
}

// validatePlayerMoveは、クライアントの移動メッセージが有効かどうかを検証するメソッドです。座標の範囲や移動速度などをチェックします。
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

// Hubは、WebSocketクライアントの管理を行う構造体です。クライアントの登録・解除、メッセージのブロードキャストなどを担当します。
type Hub struct {
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan WebSocketMessage
}

// newHubは、新しいHubインスタンスを作成する関数です。クライアントのマップやチャネルを初期化します。
func newHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan WebSocketMessage),
	}
}

// websocketHubは、グローバルなHubインスタンスです。すべてのWebSocketクライアントはこのHubを通じて管理されます。
var websocketHub = newHub()

// runは、Hubのメインループを実行するメソッドです。クライアントの登録・解除やメッセージのブロードキャストを処理します。
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

		// クライアントの登録解除処理。Hubからクライアントを削除し、送信チャネルを閉じる。
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

		// ブロードキャストメッセージの処理。Hubに登録されているすべてのクライアントにメッセージを送信する。
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

// websocketUpgraderは、WebSocket接続のアップグレードを行うための設定を持つ構造体です。バッファサイズやオリジンチェックなどを設定します。
var websocketUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,

	// Unity Editor・Windowsアプリからの接続確認用。
	// 公開時は接続元を制限する。
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// websocketHandlerは、WebSocket接続のハンドラ関数です。クライアントの認証、接続のアップグレード、クライアントの登録などを行います。
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

	// 認証に失敗した場合は、ログに記録してHTTP 401 Unauthorizedを返す。
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

	// WebSocket接続のアップグレードを試みる。失敗した場合は、ログに記録して処理を終了する。
	connection, err := websocketUpgrader.Upgrade(
		w,
		r,
		nil,
	)

	// 接続のアップグレードに失敗した場合は、ログに記録して処理を終了する。
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
		send:       make(chan WebSocketMessage, 256),
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

// readPumpは、クライアントからのメッセージを受信するためのメソッドです。接続が切断されるまでループし、受信したメッセージを処理します。
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

	// Pongメッセージを受信した際に呼び出されるハンドラを設定する。Pongを受信したら、読み取りのタイムアウトを延長する。
	client.connection.SetPongHandler(
		func(string) error {
			client.connection.SetReadDeadline(
				time.Now().Add(pongWait),
			)

			return nil
		},
	)

	// クライアントからのメッセージを受信するループ。接続が切断されるまで継続する。
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

		// チャットメッセージの処理。メッセージが空でないか、長さが200文字以内かをチェックし、条件を満たす場合はブロードキャストする。
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

		// プレイヤーの移動メッセージの処理。移動が有効かどうかを検証し、条件を満たす場合はブロードキャストする。
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

// writePumpは、クライアントへのメッセージ送信を行うためのメソッドです。送信キューからメッセージを取り出し、WebSocket接続に書き込みます。
func (client *Client) writePump() {
	ticker :=
		time.NewTicker(pingPeriod)

	defer func() {
		ticker.Stop()
	}()

	for {
		select {

		// クライアントの送信キューからメッセージを受信し、WebSocket接続に書き込む。送信キューが閉じられている場合は、接続を閉じる。
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

		// 定期的にPingメッセージを送信して、接続が生きているか確認する。Pongが返ってこない場合は接続を切断する。
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
