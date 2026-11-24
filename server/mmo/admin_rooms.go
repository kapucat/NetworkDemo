package main

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"time"
)

type adminRoomMember struct {
	AccountID int64
	Name      string
	Ready     bool
}
type adminRoom struct {
	ID       string
	Name     string
	Phase    string
	Capacity int
	StartAt  *time.Time
	Members  []adminRoomMember
}
type adminRoomSnapshot struct {
	Version  int
	ServerID string
	At       time.Time
	Rooms    []adminRoom
}
type adminRoomRequest struct {
	ctx   context.Context
	reply chan adminRoomSnapshot
}

// Only call from the Hub owner goroutine. Returned values never alias live state.
func (h *Hub) snapshotRooms(now time.Time) adminRoomSnapshot {
	out := adminRoomSnapshot{Version: 1, ServerID: mmoAdmin.instance, At: now.UTC(), Rooms: []adminRoom{}}
	for _, r := range h.rooms {
		room := adminRoom{ID: r.id, Name: r.name, Phase: r.phase, Capacity: roomCapacity, Members: []adminRoomMember{}}
		if !r.startAt.IsZero() {
			at := r.startAt.UTC()
			room.StartAt = &at
		}
		for c, ready := range r.members {
			room.Members = append(room.Members, adminRoomMember{c.accountID, c.loginName, ready})
		}
		sort.Slice(room.Members, func(i, j int) bool { return room.Members[i].AccountID < room.Members[j].AccountID })
		out.Rooms = append(out.Rooms, room)
	}
	sort.Slice(out.Rooms, func(i, j int) bool { return out.Rooms[i].ID < out.Rooms[j].ID })
	return out
}
func (h *Hub) readRooms(ctx context.Context) (adminRoomSnapshot, error) {
	req := adminRoomRequest{ctx: ctx, reply: make(chan adminRoomSnapshot, 1)}
	select {
	case h.adminRoomRequests <- req:
	case <-ctx.Done():
		return adminRoomSnapshot{}, ctx.Err()
	}
	select {
	case s := <-req.reply:
		return s, nil
	case <-ctx.Done():
		return adminRoomSnapshot{}, ctx.Err()
	}
}
func (h *Hub) roomsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	snapshot, err := h.readRooms(ctx)
	if err != nil {
		http.Error(w, "room monitor unavailable", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(snapshot)
}
