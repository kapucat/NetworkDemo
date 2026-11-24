package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"sort"
	"sync"
	"time"
)

type adminChat struct {
	At      time.Time
	Message string
}
type adminConnection struct {
	ID           uint64
	AccountID    int64
	Name         string
	LastReceived time.Time
	HasPosition  bool
	X, Y, Z      float64
	Speed        float64
	SpeedKnown   bool
	MoveAt       time.Time
	Chats        []adminChat
}
type adminSnapshot struct {
	Version     int
	ServerID    string
	At          time.Time
	Connections []adminConnection
}
type adminMonitor struct {
	instance string
	mu       sync.Mutex
	next     uint64
	clients  map[*Client]adminConnection
}

func newAdminMonitor() *adminMonitor {
	return &adminMonitor{instance: newMonitorInstance(), clients: make(map[*Client]adminConnection)}
}

var mmoAdmin = newAdminMonitor()

func (m *adminMonitor) add(c *Client, now time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.clients[c]; exists {
		return
	}
	m.next++
	m.clients[c] = adminConnection{ID: m.next, AccountID: c.accountID, Name: c.loginName, LastReceived: now, Chats: []adminChat{}}
}
func (m *adminMonitor) remove(c *Client) { m.mu.Lock(); defer m.mu.Unlock(); delete(m.clients, c) }
func (m *adminMonitor) touch(c *Client, now time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.clients[c]
	if !ok {
		return
	}
	s.LastReceived = now
	m.clients[c] = s
}

// 検証を通過した移動だけを記録する。readPump所有の検証用フィールドは読まない。
func (m *adminMonitor) move(c *Client, v WebSocketMessage, now time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.clients[c]
	if !ok {
		return
	}
	s.SpeedKnown = false
	if s.HasPosition && now.After(s.MoveAt) {
		dt := now.Sub(s.MoveAt).Seconds()
		dx, dy, dz := v.X-s.X, v.Y-s.Y, v.Z-s.Z
		speed := math.Sqrt(dx*dx+dy*dy+dz*dz) / dt
		if !math.IsNaN(speed) && !math.IsInf(speed, 0) {
			s.Speed = speed
			s.SpeedKnown = true
		}
	}
	s.X = v.X
	s.Y = v.Y
	s.Z = v.Z
	s.HasPosition = true
	s.MoveAt = now
	s.LastReceived = now
	m.clients[c] = s
}
func (m *adminMonitor) chat(c *Client, text string, now time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.clients[c]
	if !ok {
		return
	}
	s.Chats = append(s.Chats, adminChat{At: now, Message: text})
	if len(s.Chats) > 50 {
		s.Chats = append([]adminChat(nil), s.Chats[len(s.Chats)-50:]...)
	}
	s.LastReceived = now
	m.clients[c] = s
}
func (m *adminMonitor) snapshot(now time.Time) adminSnapshot {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := adminSnapshot{Version: 1, ServerID: m.instance, At: now, Connections: []adminConnection{}}
	for _, s := range m.clients {
		s.Chats = append([]adminChat{}, s.Chats...)
		// 停止後も直前の観測速度を「現在速度」として表示しない。
		if now.Sub(s.MoveAt) > 2*time.Second {
			s.SpeedKnown = false
		}
		out.Connections = append(out.Connections, s)
	}
	sort.Slice(out.Connections, func(i, j int) bool { return out.Connections[i].ID < out.Connections[j].ID })
	return out
}
func (m *adminMonitor) handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", 405)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(m.snapshot(time.Now().UTC()))
}
func serveMmoAdmin() {
	mux := http.NewServeMux()
	mux.HandleFunc("/snapshot", mmoAdmin.handler)
	mux.HandleFunc("/rooms", websocketHub.roomsHandler)
	mux.HandleFunc("/kick", mmoAdmin.kickHandler)
	mux.HandleFunc("/kick-status", mmoAdmin.kickHandler)
	mux.HandleFunc("/accounts", mmoAdmin.accountsHandler)
	server := http.Server{Addr: "127.0.0.1:8091", Handler: mux, ReadHeaderTimeout: 2 * time.Second, ReadTimeout: 5 * time.Second, WriteTimeout: 20 * time.Second, IdleTimeout: 5 * time.Second}
	if err := server.ListenAndServe(); err != nil {
		fmt.Fprintln(os.Stderr, "MMO admin monitor unavailable")
	}
}

// 同一コンテナ内だけで取得する固定コマンド。公開APIのMuxへ登録しない。
func adminSnapshotCommand() bool {
	if len(os.Args) != 2 {
		return false
	}
	path := ""
	switch os.Args[1] {
	case "--admin-rooms":
		path = "/rooms"
	case "--admin-snapshot":
		path = "/snapshot"
	case "--admin-kick":
		path = "/kick"
	case "--admin-kick-status":
		path = "/kick-status"
	case "--admin-accounts":
		path = "/accounts"
	default:
		return false
	}
	client := http.Client{Timeout: 20 * time.Second, Transport: &http.Transport{Proxy: nil}}
	var r *http.Response
	var err error
	if path == "/snapshot" || path == "/rooms" {
		r, err = client.Get("http://127.0.0.1:8091" + path)
	} else {
		body, readErr := io.ReadAll(io.LimitReader(os.Stdin, 65537))
		if readErr != nil || len(body) > 65536 {
			os.Exit(1)
		}
		r, err = client.Post("http://127.0.0.1:8091"+path, "application/json", bytes.NewReader(body))
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "monitor unavailable")
		os.Exit(1)
	}
	defer r.Body.Close()
	if r.StatusCode != 200 {
		os.Exit(1)
	}
	data, err := io.ReadAll(io.LimitReader(r.Body, 16*1024*1024+1))
	if err != nil || len(data) > 16*1024*1024 || !json.Valid(data) {
		os.Exit(1)
	}
	os.Stdout.Write(data)
	return true
}
