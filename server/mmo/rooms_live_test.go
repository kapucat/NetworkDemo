package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// Explicit opt-in: this creates disposable accounts only on the selected exhibition server.
func TestLiveRoomProtocol(t *testing.T) {
	if os.Getenv("NETWORKDEMO_ROOM_LIVE_TEST") != "https://49.212.160.97" {
		t.Skip("live test not requested")
	}
	base := os.Getenv("NETWORKDEMO_ROOM_LIVE_TEST")
	type account struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	}
	accounts := []account{}
	client := &http.Client{Timeout: 15 * time.Second}
	post := func(path string, body any, target any) {
		t.Helper()
		data, _ := json.Marshal(body)
		res, err := client.Post(base+path, "application/json", bytes.NewReader(data))
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		if res.StatusCode < 200 || res.StatusCode >= 300 {
			t.Fatalf("request %s returned %d", path, res.StatusCode)
		}
		if err = json.NewDecoder(res.Body).Decode(target); err != nil {
			t.Fatal(err)
		}
	}
	var nonce [6]byte
	rand.Read(nonce[:])
	prefix := "roomqa_" + hex.EncodeToString(nonce[:])[:10]
	tokens := []string{}
	for i := 0; i < 3; i++ {
		name := fmt.Sprint(prefix, i)
		var passwordBytes [16]byte
		rand.Read(passwordBytes[:])
		password := hex.EncodeToString(passwordBytes[:])
		body := map[string]string{"login_name": name, "password": password}
		var registered AccountRegisterResponse
		post("/api/accounts/register", body, &registered)
		accounts = append(accounts, account{registered.AccountID, name})
		data, _ := json.Marshal(accounts)
		if err := os.WriteFile("../live-accounts.json", data, 0600); err != nil {
			t.Fatal(err)
		}
		var logged AccountLoginResponse
		post("/api/accounts/login", body, &logged)
		tokens = append(tokens, logged.SessionToken)
	}
	dial := func(token string, roomMode bool) roomSocket {
		t.Helper()
		url := "wss://49.212.160.97/ws?token=" + token
		if roomMode {
			url += "&rooms=1"
		}
		conn, _, err := websocket.DefaultDialer.Dial(url, nil)
		if err != nil {
			t.Fatal("public websocket handshake failed: ", err)
		}
		t.Cleanup(func() { conn.Close() })
		s := roomSocket{conn, make(chan WebSocketMessage, 128)}
		go func() {
			for {
				var m WebSocketMessage
				if conn.ReadJSON(&m) != nil {
					return
				}
				s.messages <- m
			}
		}()
		if roomMode {
			s.wait(t, "room.list")
		} else {
			s.wait(t, "connected")
		}
		return s
	}
	a, b, c := dial(tokens[0], true), dial(tokens[1], true), dial(tokens[2], true)
	legacy := dial(tokens[0], false)
	a.send(t, WebSocketMessage{Type: "room.create", Message: "公開確認A"})
	id := a.wait(t, "room.state").Room.ID
	b.send(t, WebSocketMessage{Type: "room.join", RoomID: id})
	b.wait(t, "room.state")
	c.send(t, WebSocketMessage{Type: "room.create", Message: "公開確認B"})
	idB := c.wait(t, "room.state").Room.ID
	captureAdminRoom(t, id, "waiting", 2)
	for _, s := range []roomSocket{a, b} {
		s.send(t, WebSocketMessage{Type: "room.ready", RoomID: id, Ready: true})
	}
	for {
		if a.wait(t, "room.state").Room.Phase == "countdown" {
			break
		}
	}
	b.send(t, WebSocketMessage{Type: "room.ready", RoomID: id, Ready: false})
	for {
		if a.wait(t, "room.state").Room.Phase == "waiting" {
			break
		}
	}
	b.send(t, WebSocketMessage{Type: "room.ready", RoomID: id, Ready: true})
	for {
		if a.wait(t, "room.state").Room.Phase == "playing" {
			break
		}
	}
	for {
		if b.wait(t, "room.state").Room.Phase == "playing" {
			break
		}
	}
	captureAdminRoom(t, id, "playing", 2)
	a.send(t, WebSocketMessage{Type: "chat", RoomID: id, Message: "room A chat"})
	b.wait(t, "chat.message")
	a.send(t, WebSocketMessage{Type: "player.move", RoomID: id, X: 1, Y: 0, Z: 1})
	b.wait(t, "player.move")
	legacy.send(t, WebSocketMessage{Type: "chat", Message: "legacy chat"})
	legacy.wait(t, "chat.message")
	c.send(t, WebSocketMessage{Type: "room.ping", ClientUnixMS: 123})
	until := time.NewTimer(5 * time.Second)
	defer until.Stop()
	barrier := false
	for !barrier {
		select {
		case m := <-c.messages:
			if m.Type == "chat.message" || m.Type == "player.move" || m.Type == "player.join" {
				t.Fatal("cross-room or legacy data leaked")
			}
			barrier = m.Type == "room.pong" && m.ClientUnixMS == 123
		case <-until.C:
			t.Fatal("missing pong")
		}
	}
	b.send(t, WebSocketMessage{Type: "room.leave", RoomID: id})
	b.wait(t, "room.left")
	b.send(t, WebSocketMessage{Type: "room.join", RoomID: idB})
	if b.wait(t, "room.state").Room.ID != idB {
		t.Fatal("room switch failed")
	}
	captureAdminRoom(t, idB, "waiting", 2)
	for _, s := range []roomSocket{b, c} {
		s.send(t, WebSocketMessage{Type: "room.ready", RoomID: idB, Ready: true})
	}
	for {
		if b.wait(t, "room.state").Room.Phase == "countdown" {
			break
		}
	}
	c.conn.Close()
	for {
		m := b.wait(t, "room.state")
		if m.Room.Phase == "waiting" && len(m.Room.Members) == 1 {
			break
		}
	}
	captureAdminRoom(t, idB, "waiting", 1)
	t.Log("Public HTTPS login / WSS rooms / readiness cancellation / synchronized start / scoped chat & movement / legacy compatibility / room switching / disconnect cancellation passed")
}
