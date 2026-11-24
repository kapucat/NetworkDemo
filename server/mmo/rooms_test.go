package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func roomTestClient(h *Hub, id int64) *Client {
	c := &Client{hub: h, accountID: id, loginName: fmt.Sprint("user", id), roomsMode: true,
		send: make(chan WebSocketMessage, 256), moveWake: make(chan struct{}, 1), roomCommandTokens: 12, roomCommandAt: time.Now()}
	h.clients[c] = true
	return c
}
func roomDo(h *Hub, c *Client, kind string, now time.Time) {
	h.handleRoom(roomRequest{c, WebSocketMessage{Type: kind, RoomID: c.roomID}}, now)
}
func TestRoomLifecycleAndIsolation(t *testing.T) {
	h := newHub()
	a := roomTestClient(h, 1)
	b := roomTestClient(h, 2)
	other := roomTestClient(h, 3)
	now := time.Now()
	h.handleRoom(roomRequest{a, WebSocketMessage{Type: "room.create", Message: "部屋A"}}, now)
	id := a.roomID
	if id == "" {
		t.Fatal("no room")
	}
	h.handleRoom(roomRequest{b, WebSocketMessage{Type: "room.join", RoomID: id}}, now)
	h.handleRoom(roomRequest{other, WebSocketMessage{Type: "room.create", Message: "部屋B"}}, now)
	for _, c := range []*Client{a, b} {
		h.handleRoom(roomRequest{c, WebSocketMessage{Type: "room.ready", RoomID: id, Ready: true}}, now)
	}
	r := h.rooms[id]
	if r.phase != "countdown" {
		t.Fatal(r.phase)
	}
	h.handleRoom(roomRequest{a, WebSocketMessage{Type: "player.move", RoomID: id, X: 1}}, now)
	if a.hasMoveState {
		t.Fatal("movement before start")
	}
	h.tickRooms(now.Add(3 * time.Second))
	if r.phase != "playing" {
		t.Fatal("not started")
	}
	h.handleRoom(roomRequest{a, WebSocketMessage{Type: "player.move", RoomID: id, AccountID: 999, X: 1}}, now.Add(4*time.Second))
	if other.moveCount() != 0 || b.moveCount() != 1 {
		t.Fatal("cross-room movement")
	}
	move, _ := b.takeMove()
	if move.AccountID != 1 || move.RoomID != id {
		t.Fatal("spoofed identity")
	}
	h.handleRoom(roomRequest{a, WebSocketMessage{Type: "chat", RoomID: id, Message: "hello"}}, now.Add(4*time.Second))
	for len(other.send) > 0 {
		if (<-other.send).Type == "chat.message" {
			t.Fatal("cross-room chat")
		}
	}
	h.leaveRoom(a, now)
	if r.phase != "playing" || len(r.members) != 1 {
		t.Fatal("playing disconnect")
	}
	h.leaveRoom(b, now)
	if h.rooms[id] != nil {
		t.Fatal("empty room retained")
	}
}
func TestRoomCancelCapacityAndDuplicate(t *testing.T) {
	h := newHub()
	a := roomTestClient(h, 1)
	now := time.Now()
	h.handleRoom(roomRequest{a, WebSocketMessage{Type: "room.create", Message: "room"}}, now)
	r := h.rooms[a.roomID]
	h.handleRoom(roomRequest{a, WebSocketMessage{Type: "room.ready", RoomID: r.id, Ready: true}}, now)
	if r.phase != "waiting" {
		t.Fatal("one person started")
	}
	duplicate := roomTestClient(h, 1)
	h.handleRoom(roomRequest{duplicate, WebSocketMessage{Type: "room.join", RoomID: r.id}}, now)
	if duplicate.roomID != "" {
		t.Fatal("duplicate account")
	}
	b := roomTestClient(h, 2)
	h.handleRoom(roomRequest{b, WebSocketMessage{Type: "room.join", RoomID: r.id}}, now)
	if r.members[a] {
		t.Fatal("membership did not reset ready")
	}
	for _, c := range []*Client{a, b} {
		h.handleRoom(roomRequest{c, WebSocketMessage{Type: "room.ready", RoomID: r.id, Ready: true}}, now)
	}
	h.leaveRoom(b, now)
	h.tickRooms(now.Add(10 * time.Second))
	if r.phase != "waiting" || !r.startAt.IsZero() || r.members[a] {
		t.Fatal("countdown not cancelled")
	}
	for i := int64(3); i <= 6; i++ {
		c := roomTestClient(h, i)
		h.handleRoom(roomRequest{c, WebSocketMessage{Type: "room.join", RoomID: r.id}}, now)
	}
	if len(r.members) != 4 {
		t.Fatal("capacity")
	}
	h.handleRoom(roomRequest{a, WebSocketMessage{Type: "room.ready", RoomID: "wrong", Ready: true}}, now)
	if r.members[a] {
		t.Fatal("stale room command accepted")
	}
}
func TestRoomInputBounds(t *testing.T) {
	h := newHub()
	c := roomTestClient(h, 1)
	now := time.Now()
	for _, name := range []string{"", strings.Repeat("あ", 21), "name\nother"} {
		h.handleRoom(roomRequest{c, WebSocketMessage{Type: "room.create", Message: name}}, now)
		if c.roomID != "" {
			t.Fatal("bad name accepted")
		}
	}
	roomDo(h, c, "room.ready", now)
	if len(h.rooms) != 0 {
		t.Fatal("unexpected room")
	}
}

type roomSocket struct {
	conn     *websocket.Conn
	messages chan WebSocketMessage
}

func (s roomSocket) wait(t *testing.T, kind string) WebSocketMessage {
	t.Helper()
	timeout := time.NewTimer(6 * time.Second)
	defer timeout.Stop()
	for {
		select {
		case m := <-s.messages:
			if m.Type == kind {
				return m
			}
		case <-timeout.C:
			t.Fatal("timeout waiting " + kind)
			return WebSocketMessage{}
		}
	}
}
func (s roomSocket) send(t *testing.T, m WebSocketMessage) {
	t.Helper()
	if err := s.conn.WriteJSON(m); err != nil {
		t.Fatal(err)
	}
}
func TestRoomsOverWebSocket(t *testing.T) {
	h := newHub()
	go h.run()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocketUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		var id int64
		fmt.Sscan(r.URL.Query().Get("id"), &id)
		c := &Client{hub: h, connection: conn, accountID: id, loginName: fmt.Sprint(id), roomsMode: true, send: make(chan WebSocketMessage, 256), moveWake: make(chan struct{}, 1)}
		h.register <- c
		go c.writePump()
		c.readPump()
	}))
	defer srv.Close()
	dial := func(id int64) roomSocket {
		conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http")+fmt.Sprint("?id=", id), nil)
		if err != nil {
			t.Fatal(err)
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
		s.wait(t, "room.list")
		return s
	}
	a, b, c := dial(101), dial(102), dial(103)
	a.send(t, WebSocketMessage{Type: "room.create", Message: "A"})
	id := a.wait(t, "room.state").Room.ID
	b.send(t, WebSocketMessage{Type: "room.join", RoomID: id})
	b.wait(t, "room.state")
	c.send(t, WebSocketMessage{Type: "room.create", Message: "B"})
	other := c.wait(t, "room.state").Room.ID
	for _, s := range []roomSocket{a, b} {
		s.send(t, WebSocketMessage{Type: "room.ready", RoomID: id, Ready: true})
	}
	for {
		if a.wait(t, "room.state").Room.Phase == "playing" {
			break
		}
	}
	a.send(t, WebSocketMessage{Type: "chat", RoomID: id, Message: "in room A"})
	if b.wait(t, "chat.message").Message != "in room A" {
		t.Fatal("chat missing")
	}
	a.send(t, WebSocketMessage{Type: "player.move", RoomID: id, X: 3})
	if b.wait(t, "player.move").AccountID != 101 {
		t.Fatal("movement missing")
	}
	c.send(t, WebSocketMessage{Type: "room.ping", ClientUnixMS: 123})
	for {
		m := c.wait(t, "room.pong")
		if m.ClientUnixMS == 123 {
			break
		}
	}
	for len(c.messages) > 0 {
		m := <-c.messages
		if m.Type == "chat.message" || m.Type == "player.move" {
			t.Fatal("room leak")
		}
	}
	b.send(t, WebSocketMessage{Type: "room.leave", RoomID: id})
	b.wait(t, "room.left")
	b.send(t, WebSocketMessage{Type: "room.join", RoomID: other})
	if b.wait(t, "room.state").Room.ID != other {
		t.Fatal("switch room")
	}
}
