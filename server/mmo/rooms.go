package main

import (
	"crypto/rand"
	"encoding/hex"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const roomCapacity = 4
const maxRooms = 50
const roomCountdown = 3 * time.Second

type roomMember struct {
	AccountID int64  `json:"account_id"`
	LoginName string `json:"login_name"`
	Ready     bool   `json:"ready"`
}
type roomSummary struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Count    int    `json:"count"`
	Capacity int    `json:"capacity"`
	Phase    string `json:"phase"`
}
type roomState struct {
	ID            string       `json:"id"`
	Name          string       `json:"name"`
	Phase         string       `json:"phase"`
	StartAtUnixMS int64        `json:"start_at_unix_ms"`
	Members       []roomMember `json:"members"`
}

// All room state is owned exclusively by the Hub event loop.
type gameRoom struct {
	id, name, phase string
	startAt         time.Time
	members         map[*Client]bool
}
type roomRequest struct {
	client  *Client
	message WebSocketMessage
}

func (h *Hub) roomSend(c *Client, m WebSocketMessage) {
	if !c.sendSafely(m) && c.connection != nil {
		_ = c.connection.Close()
	}
}
func (h *Hub) roomError(c *Client, text string) {
	h.roomSend(c, WebSocketMessage{Type: "room.error", Message: text})
}
func (h *Hub) roomList(c *Client) {
	rooms := make([]roomSummary, 0, len(h.rooms))
	for _, r := range h.rooms {
		rooms = append(rooms, roomSummary{r.id, r.name, len(r.members), roomCapacity, r.phase})
	}
	sort.Slice(rooms, func(i, j int) bool { return rooms[i].ID < rooms[j].ID })
	h.roomSend(c, WebSocketMessage{Type: "room.list", Rooms: rooms, ServerUnixMS: time.Now().UnixMilli()})
}
func (h *Hub) publishRoomList() {
	for c := range h.clients {
		if c.roomsMode && c.roomID == "" {
			h.roomList(c)
		}
	}
}
func (r *gameRoom) snapshot() *roomState {
	s := &roomState{ID: r.id, Name: r.name, Phase: r.phase, Members: make([]roomMember, 0, len(r.members))}
	if !r.startAt.IsZero() {
		s.StartAtUnixMS = r.startAt.UnixMilli()
	}
	for c, ready := range r.members {
		s.Members = append(s.Members, roomMember{c.accountID, c.loginName, ready})
	}
	sort.Slice(s.Members, func(i, j int) bool { return s.Members[i].AccountID < s.Members[j].AccountID })
	return s
}
func (h *Hub) publishRoom(r *gameRoom, now time.Time) {
	m := WebSocketMessage{Type: "room.state", RoomID: r.id, Room: r.snapshot(), ServerUnixMS: now.UnixMilli()}
	for c := range r.members {
		h.roomSend(c, m)
	}
}
func (h *Hub) leaveRoom(c *Client, now time.Time) {
	r := h.rooms[c.roomID]
	c.roomID = ""
	if r == nil {
		return
	}
	delete(r.members, c)
	c.hasMoveState = false
	for peer := range r.members {
		peer.discardMove(c.accountID)
	}
	if len(r.members) == 0 {
		delete(h.rooms, r.id)
	} else {
		if r.phase == "countdown" {
			r.phase = "waiting"
			r.startAt = time.Time{}
			for peer := range r.members {
				r.members[peer] = false
			}
		}
		h.publishRoom(r, now)
	}
	h.roomSend(c, WebSocketMessage{Type: "room.left", RoomID: r.id})
	h.publishRoomList()
}
func (h *Hub) tickRooms(now time.Time) {
	for _, r := range h.rooms {
		if r.phase == "countdown" && !now.Before(r.startAt) {
			r.phase = "playing"
			h.publishRoom(r, now)
			h.publishRoomList()
		}
	}
}
func (h *Hub) handleRoom(req roomRequest, now time.Time) {
	c, m := req.client, req.message
	if !h.clients[c] || !c.roomsMode {
		return
	}
	// Movement has its own rate/distance validation. Bound lobby commands separately.
	if m.Type != "player.move" && m.Type != "chat" && m.Type != "chat.send" {
		elapsed := now.Sub(c.roomCommandAt).Seconds()
		c.roomCommandTokens = min(12, c.roomCommandTokens+max(0, elapsed)*6)
		c.roomCommandAt = now
		if c.roomCommandTokens < 1 {
			h.roomError(c, "操作が多すぎます。少し待ってください。")
			return
		}
		c.roomCommandTokens--
	}
	switch m.Type {
	case "room.ping":
		h.roomSend(c, WebSocketMessage{Type: "room.pong", ClientUnixMS: m.ClientUnixMS, ServerUnixMS: now.UnixMilli()})
	case "room.list":
		h.roomList(c)
	case "room.create", "room.join":
		if c.roomID != "" {
			h.roomError(c, "先に現在の部屋から退出してください。")
			return
		}
		for other := range h.clients {
			if other != c && other.roomsMode && other.accountID == c.accountID && other.roomID != "" {
				h.roomError(c, "同じアカウントは既に部屋に参加しています。")
				return
			}
		}
		var r *gameRoom
		if m.Type == "room.create" {
			name := strings.TrimSpace(m.Message)
			if utf8.RuneCountInString(name) < 1 || utf8.RuneCountInString(name) > 20 || strings.IndexFunc(name, unicode.IsControl) >= 0 {
				h.roomError(c, "部屋名は1～20文字で入力してください。")
				return
			}
			if len(h.rooms) >= maxRooms {
				h.roomError(c, "部屋数の上限です。")
				return
			}
			var b [8]byte
			if _, err := rand.Read(b[:]); err != nil {
				h.roomError(c, "部屋を作成できませんでした。")
				return
			}
			id := hex.EncodeToString(b[:])
			r = &gameRoom{id: id, name: name, phase: "waiting", members: make(map[*Client]bool)}
			h.rooms[id] = r
		} else {
			r = h.rooms[m.RoomID]
			if r == nil || r.phase != "waiting" || len(r.members) >= roomCapacity {
				h.roomError(c, "その部屋には参加できません。部屋一覧を更新してください。")
				h.roomList(c)
				return
			}
		}
		c.roomID = r.id
		c.hasMoveState = false
		r.members[c] = false
		// A roster change requires everyone to confirm readiness again.
		for peer := range r.members {
			r.members[peer] = false
		}
		h.publishRoom(r, now)
		h.publishRoomList()
	case "room.leave":
		if m.RoomID != c.roomID {
			return
		}
		h.leaveRoom(c, now)
		h.roomList(c)
	case "room.ready":
		r := h.rooms[c.roomID]
		if r == nil || m.RoomID != r.id || r.phase == "playing" {
			h.roomError(c, "準備状態を変更できません。")
			return
		}
		if r.phase == "countdown" && m.Ready {
			return
		}
		r.members[c] = m.Ready
		all := len(r.members) >= 2
		for _, ready := range r.members {
			all = all && ready
		}
		if all {
			r.phase = "countdown"
			r.startAt = now.Add(roomCountdown)
		} else {
			r.phase = "waiting"
			r.startAt = time.Time{}
		}
		h.publishRoom(r, now)
		h.publishRoomList()
	case "player.move", "chat", "chat.send":
		r := h.rooms[c.roomID]
		if r == nil || m.RoomID != r.id {
			return
		}
		h.tickRooms(now)
		if m.Type == "player.move" {
			if r.phase != "playing" || !c.validatePlayerMove(m) {
				return
			}
			m.Type = "player.move"
			mmoAdmin.move(c, m, now)
		} else {
			m.Message = strings.TrimSpace(m.Message)
			if m.Message == "" {
				return
			}
			if utf8.RuneCountInString(m.Message) > 200 {
				h.roomError(c, "チャットは200文字以内です。")
				return
			}
			if now.Sub(c.roomChatAt) < 150*time.Millisecond {
				return
			}
			c.roomChatAt = now
			m.Type = "chat.message"
			mmoAdmin.chat(c, m.Message, now)
		}
		// Never relay client-supplied identity or nested room state.
		out := WebSocketMessage{Type: m.Type, AccountID: c.accountID, LoginName: c.loginName, RoomID: r.id,
			Message: m.Message, X: m.X, Y: m.Y, Z: m.Z, RotationY: m.RotationY, SentAt: now.UTC().Format(time.RFC3339Nano)}
		for peer := range r.members {
			if out.Type == "player.move" {
				peer.enqueueMove(out)
			} else {
				h.roomSend(peer, out)
			}
		}
	default:
		h.roomError(c, "対応していない部屋操作です。")
	}
}
