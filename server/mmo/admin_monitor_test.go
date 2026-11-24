package main

import (
	"github.com/gorilla/websocket"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestAdminMonitorLifecycle(t *testing.T) {
	m := newAdminMonitor()
	a := &Client{accountID: 1, loginName: "user"}
	b := &Client{accountID: 1, loginName: "user"}
	now := time.Now().UTC()
	m.add(a, now)
	m.add(b, now)
	s := m.snapshot(now)
	if len(s.Connections) != 2 || s.Connections[0].ID == s.Connections[1].ID {
		t.Fatal("same account connections must remain distinct")
	}
	if s.Connections[0].HasPosition || s.Connections[0].SpeedKnown {
		t.Fatal("initial position/speed unknown")
	}
	m.move(a, WebSocketMessage{X: 1}, now)
	m.move(a, WebSocketMessage{X: 3}, now.Add(time.Second))
	s = m.snapshot(now.Add(time.Second))
	if s.Connections[0].Speed != 2 || !s.Connections[0].SpeedKnown {
		t.Fatal("observed speed")
	}
	if m.snapshot(now.Add(4 * time.Second)).Connections[0].SpeedKnown {
		t.Fatal("stale speed")
	}
	for i := 0; i < 60; i++ {
		m.chat(a, "日本語😀", now)
	}
	s = m.snapshot(now)
	if len(s.Connections[0].Chats) != 50 {
		t.Fatal("bounded history")
	}
	s.Connections[0].Chats[0].Message = "changed"
	if m.snapshot(now).Connections[0].Chats[0].Message != "日本語😀" {
		t.Fatal("snapshot aliases live state")
	}
	r := httptest.NewRecorder()
	m.handler(r, httptest.NewRequest("GET", "/snapshot", nil))
	body := r.Body.String()
	for _, secret := range []string{"password", "token", "remoteAddr"} {
		if strings.Contains(body, secret) {
			t.Fatal("sensitive field", secret)
		}
	}
	r = httptest.NewRecorder()
	m.handler(r, httptest.NewRequest("POST", "/snapshot", nil))
	if r.Code != 405 {
		t.Fatal("read only")
	}
	m.remove(a)
	m.remove(b)
	if len(m.snapshot(now).Connections) != 0 {
		t.Fatal("disconnect removes history and state")
	}
}

func TestAdminReadPumpIntegration(t *testing.T) {
	original := mmoAdmin
	mmoAdmin = newAdminMonitor()
	defer func() { mmoAdmin = original }()
	hub := newHub()
	go hub.run()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocketUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		c := &Client{hub: hub, connection: conn, accountID: 123, loginName: "monitor_test", send: make(chan WebSocketMessage, 256), moveWake: make(chan struct{}, 1)}
		hub.register <- c
		go c.writePump()
		c.readPump()
	}))
	defer server.Close()
	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), nil)
	if err != nil {
		t.Fatal(err)
	}
	wait := func(test func() bool) {
		t.Helper()
		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			if test() {
				return
			}
			time.Sleep(5 * time.Millisecond)
		}
		t.Fatal("monitor hook timed out")
	}
	wait(func() bool { return len(mmoAdmin.snapshot(time.Now()).Connections) == 1 })
	conn.WriteJSON(WebSocketMessage{Type: "player.move", X: 1, Y: 2, Z: 3})
	conn.WriteJSON(WebSocketMessage{Type: "chat.send", Message: "日本語のチャット😀"})
	wait(func() bool {
		s := mmoAdmin.snapshot(time.Now())
		return len(s.Connections) == 1 && s.Connections[0].HasPosition && len(s.Connections[0].Chats) == 1
	})
	conn.WriteJSON(WebSocketMessage{Type: "player.move", X: 2001})
	time.Sleep(20 * time.Millisecond)
	if mmoAdmin.snapshot(time.Now()).Connections[0].X != 1 {
		t.Fatal("rejected movement leaked into monitor")
	}
	conn.Close()
	wait(func() bool { return len(mmoAdmin.snapshot(time.Now()).Connections) == 0 })
}
func TestAdminMonitorConcurrent(t *testing.T) {
	m := newAdminMonitor()
	c := &Client{accountID: 1, loginName: "user"}
	m.add(c, time.Now())
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				m.move(c, WebSocketMessage{X: float64(j)}, time.Now())
				m.chat(c, "test", time.Now())
				m.snapshot(time.Now())
				m.touch(c, time.Now())
			}
		}()
	}
	wg.Wait()
}
