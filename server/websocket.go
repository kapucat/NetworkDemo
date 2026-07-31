package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

type WebSocketMessage struct {
	Type      string `json:"type"`
	LoginName string `json:"login_name,omitempty"`
	Message   string `json:"message"`
	SentAt    string `json:"sent_at"`
}

var websocketUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,

	// Unity Editor・Windowsアプリからの接続確認用。
	// ブラウザ版を公開する際は接続元を制限する。
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

	connection, err := websocketUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("websocket upgrade failed:", err)
		return
	}
	defer connection.Close()

	log.Printf(
		"websocket connected: remote=%s",
		r.RemoteAddr,
	)

	connectedMessage := WebSocketMessage{
		Type:    "connected",
		Message: "websocket connected",
		SentAt:  time.Now().UTC().Format(time.RFC3339),
	}

	if err := connection.WriteJSON(connectedMessage); err != nil {
		log.Println("websocket initial write failed:", err)
		return
	}

	for {
		var receivedMessage WebSocketMessage

		err := connection.ReadJSON(&receivedMessage)
		if err != nil {
			if websocket.IsUnexpectedCloseError(
				err,
				websocket.CloseNormalClosure,
				websocket.CloseGoingAway,
			) {
				log.Println("websocket read failed:", err)
			}

			break
		}

		receivedMessage.Message =
			strings.TrimSpace(receivedMessage.Message)

		if receivedMessage.Message == "" {
			continue
		}

		if len([]rune(receivedMessage.Message)) > 200 {
			errorMessage := WebSocketMessage{
				Type:    "error",
				Message: "message is too long",
				SentAt:  time.Now().UTC().Format(time.RFC3339),
			}

			if err := connection.WriteJSON(errorMessage); err != nil {
				log.Println("websocket error write failed:", err)
				break
			}

			continue
		}

		log.Printf(
			"websocket message received: type=%s login_name=%s message=%s",
			receivedMessage.Type,
			receivedMessage.LoginName,
			receivedMessage.Message,
		)

		echoMessage := WebSocketMessage{
			Type:      "chat",
			LoginName: receivedMessage.LoginName,
			Message:   receivedMessage.Message,
			SentAt:    time.Now().UTC().Format(time.RFC3339),
		}

		if err := connection.WriteJSON(echoMessage); err != nil {
			log.Println("websocket write failed:", err)
			break
		}
	}

	log.Printf(
		"websocket disconnected: remote=%s",
		r.RemoteAddr,
	)
}

// encoding/jsonがプロジェクト内で利用可能かを
// コンパイル時に明確に確認するための宣言。
var _ = json.Valid
