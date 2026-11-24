package main

import (
	"errors"
	"github.com/gorilla/websocket"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

type memoryKickJournal struct {
	requests              map[string]adminKickRequest
	results               map[string]adminKickResult
	failBegin, failFinish bool
}

func newMemoryKickJournal() *memoryKickJournal {
	return &memoryKickJournal{requests: map[string]adminKickRequest{}, results: map[string]adminKickResult{}}
}
func (j *memoryKickJournal) begin(r adminKickRequest) (adminKickResult, bool, error) {
	if j.failBegin {
		return adminKickResult{}, false, errors.New("save failed")
	}
	if stored, ok := j.requests[r.OperationID]; ok {
		if !reflect.DeepEqual(stored, r) {
			return adminKickResult{}, false, errors.New("mismatch")
		}
		return j.results[r.OperationID], false, nil
	}
	j.requests[r.OperationID] = r
	j.results[r.OperationID] = adminKickResult{OperationID: r.OperationID, Status: "Started"}
	return j.results[r.OperationID], true, nil
}
func (j *memoryKickJournal) finish(id, status string, n int) (adminKickResult, error) {
	if j.failFinish {
		return adminKickResult{}, errors.New("finish failed")
	}
	r := adminKickResult{OperationID: id, Status: status, Count: n}
	j.results[id] = r
	return r, nil
}
func (j *memoryKickJournal) lookup(id string) (adminKickRequest, adminKickResult, error) {
	r, ok := j.results[id]
	if !ok {
		return adminKickRequest{}, adminKickResult{OperationID: id, Status: "Unknown"}, nil
	}
	return j.requests[id], r, nil
}
func kickRequest(m *adminMonitor, op string, ids ...uint64) adminKickRequest {
	return adminKickRequest{Version: 1, OperationID: op, ServerID: m.instance, AccountID: 42, ConnectionIDs: ids, Reason: "接続終了のテスト😀"}
}

const kickTestID = "11111111-2222-3333-4444-555555555555"

func TestAdminKickTargetsAndReplay(t *testing.T) {
	m := newAdminMonitor()
	j := newMemoryKickJournal()
	a := &Client{accountID: 42}
	b := &Client{accountID: 42}
	other := &Client{accountID: 99}
	m.add(a, time.Now())
	m.add(b, time.Now())
	m.add(other, time.Now())
	r := kickRequest(m, kickTestID, 1, 2)
	result := m.executeKick(r, j)
	if result.Status != "Completed" || result.Count != 2 || len(m.snapshot(time.Now()).Connections) != 1 {
		t.Fatal("selected targets only")
	}
	fresh := &Client{accountID: 42}
	m.add(fresh, time.Now())
	result = m.executeKick(r, j)
	if result.Count != 2 || len(m.snapshot(time.Now()).Connections) != 2 {
		t.Fatal("replay cut new connection")
	}
	r.Reason = "changed"
	if m.executeKick(r, j).Error == "" {
		t.Fatal("different reason replay")
	}
}
func TestAdminKickChangedInstanceAndTargets(t *testing.T) {
	m := newAdminMonitor()
	j := newMemoryKickJournal()
	a := &Client{accountID: 42}
	m.add(a, time.Now())
	r := kickRequest(m, kickTestID, 1, 2)
	if m.executeKick(r, j).Status != "Changed" || len(m.snapshot(time.Now()).Connections) != 1 {
		t.Fatal("missing target must abort all")
	}
	r.OperationID = "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	r.ConnectionIDs = []uint64{1}
	r.ServerID = newMonitorInstance()
	if m.executeKick(r, j).Status != "Changed" || len(m.snapshot(time.Now()).Connections) != 1 {
		t.Fatal("boot ID prevents reused connection IDs")
	}
}
func TestAdminKickFailureAndRecovery(t *testing.T) {
	m := newAdminMonitor()
	j := newMemoryKickJournal()
	m.add(&Client{accountID: 42}, time.Now())
	r := kickRequest(m, kickTestID, 1)
	j.failBegin = true
	if m.executeKick(r, j).Error == "" || len(m.snapshot(time.Now()).Connections) != 1 {
		t.Fatal("journal failure must prevent cut")
	}
	j.failBegin = false
	j.failFinish = true
	if m.executeKick(r, j).Error == "" || len(m.snapshot(time.Now()).Connections) != 0 {
		t.Fatal("finish response loss")
	}
	j.failFinish = false
	if m.lookupKick(kickTestID, j).Status != "TargetsGone" {
		t.Fatal("recover target absence")
	}
}
func TestAdminKickConcurrentReplay(t *testing.T) {
	m := newAdminMonitor()
	j := newMemoryKickJournal()
	m.add(&Client{accountID: 42}, time.Now())
	r := kickRequest(m, kickTestID, 1)
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if m.executeKick(r, j).Count != 1 {
				t.Error("replay result")
			}
		}()
	}
	wg.Wait()
	if len(j.requests) != 1 {
		t.Fatal("duplicate journal")
	}
}
func TestAdminKickWebSocketCloses(t *testing.T) {
	m := newAdminMonitor()
	hub := newHub()
	go hub.run()
	done := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocketUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		c := &Client{accountID: 42, connection: conn}
		m.add(c, time.Now())
		defer close(done)
		defer m.remove(c)
		defer conn.Close()
		for {
			if _, _, err = conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
	defer server.Close()
	conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	deadline := time.Now().Add(time.Second)
	for len(m.snapshot(time.Now()).Connections) == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if m.executeKick(kickRequest(m, kickTestID, 1), newMemoryKickJournal()).Status != "Completed" {
		t.Fatal("cut failed")
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("socket was not closed")
	}
	conn.SetReadDeadline(time.Now().Add(time.Second))
	if _, _, err = conn.ReadMessage(); !websocket.IsCloseError(err, websocket.ClosePolicyViolation) {
		t.Fatalf("expected admin close code 1008, got %v", err)
	}
}
