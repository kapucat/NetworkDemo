package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestAdminRoomSnapshotLifecycle(t *testing.T) {
	h := newHub()
	a := roomTestClient(h, 41)
	b := roomTestClient(h, 42)
	now := time.Now().UTC()
	empty := h.snapshotRooms(now)
	if empty.Version != 1 || empty.ServerID == "" || len(empty.Rooms) != 0 {
		t.Fatal("empty snapshot")
	}
	h.handleRoom(roomRequest{a, WebSocketMessage{Type: "room.create", Message: "日本語room"}}, now)
	h.handleRoom(roomRequest{b, WebSocketMessage{Type: "room.join", RoomID: a.roomID}}, now)
	s := h.snapshotRooms(now)
	if len(s.Rooms) != 1 || s.Rooms[0].Capacity != 4 || len(s.Rooms[0].Members) != 2 || s.Rooms[0].Phase != "waiting" || s.Rooms[0].StartAt != nil {
		t.Fatal("membership snapshot")
	}
	s.Rooms[0].Members[0].Name = "modified"
	s.Rooms[0].Members[0].Ready = true
	fresh := h.snapshotRooms(now)
	if fresh.Rooms[0].Members[0].Name == "modified" || fresh.Rooms[0].Members[0].Ready {
		t.Fatal("snapshot aliases live state")
	}
	for _, c := range []*Client{a, b} {
		h.handleRoom(roomRequest{c, WebSocketMessage{Type: "room.ready", RoomID: a.roomID, Ready: true}}, now)
	}
	s = h.snapshotRooms(now)
	if s.Rooms[0].Phase != "countdown" || s.Rooms[0].StartAt == nil || !s.Rooms[0].StartAt.Equal(now.Add(roomCountdown)) {
		t.Fatal("countdown snapshot")
	}
	h.tickRooms(now.Add(roomCountdown))
	if h.snapshotRooms(now).Rooms[0].Phase != "playing" {
		t.Fatal("playing snapshot")
	}
	h.leaveRoom(b, now)
	if len(h.snapshotRooms(now).Rooms[0].Members) != 1 {
		t.Fatal("leave not reflected")
	}
	h.leaveRoom(a, now)
	if len(h.snapshotRooms(now).Rooms) != 0 {
		t.Fatal("deleted room retained")
	}
	data, _ := json.Marshal(s)
	for _, secret := range []string{"password_hash", "session_token", "remoteAddr", "Chats"} {
		if strings.Contains(string(data), secret) {
			t.Fatal("unrelated data leaked")
		}
	}
}

func TestAdminRoomRPCAndHandler(t *testing.T) {
	h := newHub()
	c := roomTestClient(h, 1)
	go h.run()
	h.roomRequests <- roomRequest{c, WebSocketMessage{Type: "room.create", Message: "RPC"}}
	for {
		m := <-c.send
		if m.Type == "room.state" {
			break
		}
	}
	var wg sync.WaitGroup
	for i := 0; i < 24; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			s, err := h.readRooms(ctx)
			if err != nil || len(s.Rooms) != 1 || s.Rooms[0].Name != "RPC" {
				t.Error("RPC snapshot", err)
			}
		}()
	}
	wg.Wait()
	for _, method := range []string{"GET", "POST"} {
		response := httptest.NewRecorder()
		h.roomsHandler(response, httptest.NewRequest(method, "/rooms", nil))
		expected := http.StatusOK
		if method == "POST" {
			expected = http.StatusMethodNotAllowed
		}
		if response.Code != expected || response.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("handler", method, response.Code)
		}
	}
}

func TestAdminRoomsCanceledRequest(t *testing.T) {
	h := newHub()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if _, err := h.readRooms(ctx); err == nil {
		t.Fatal("stopped hub must time out")
	}
	req := httptest.NewRequest("GET", "/rooms", nil).WithContext(ctx)
	response := httptest.NewRecorder()
	h.roomsHandler(response, req)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatal("unavailable hub response")
	}
}
