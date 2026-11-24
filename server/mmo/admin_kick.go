package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/gorilla/websocket"
	"net/http"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

func newMonitorInstance() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic("monitor instance initialization failed")
	}
	return hex.EncodeToString(b[:])
}

type adminKickRequest struct {
	Version       int
	OperationID   string
	ServerID      string
	AccountID     int64
	ConnectionIDs []uint64
	Reason        string
}
type adminKickResult struct {
	OperationID string
	Status      string
	Count       int
	Error       string `json:",omitempty"`
}
type kickJournal interface {
	begin(adminKickRequest) (adminKickResult, bool, error)
	finish(string, string, int) (adminKickResult, error)
	lookup(string) (adminKickRequest, adminKickResult, error)
}
type databaseKickJournal struct{}

var mmoKickJournal kickJournal = databaseKickJournal{}
var kickOperationMu sync.Mutex
var operationPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func (databaseKickJournal) begin(request adminKickRequest) (result adminKickResult, fresh bool, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return
	}
	defer tx.Rollback()
	payload, err := json.Marshal(request)
	if err != nil {
		return
	}
	inserted, err := tx.ExecContext(ctx, `INSERT INTO mmo_admin.connection_operations(operation_id,request,status) VALUES($1,$2::jsonb,'Started') ON CONFLICT DO NOTHING`, request.OperationID, string(payload))
	if err != nil {
		return
	}
	n, err := inserted.RowsAffected()
	if err != nil {
		return
	}
	fresh = n == 1
	var raw []byte
	result.OperationID = request.OperationID
	err = tx.QueryRowContext(ctx, `SELECT request,status,affected_count FROM mmo_admin.connection_operations WHERE operation_id=$1 FOR UPDATE`, request.OperationID).Scan(&raw, &result.Status, &result.Count)
	if err != nil {
		return
	}
	var stored adminKickRequest
	err = json.Unmarshal(raw, &stored)
	if err != nil {
		return
	}
	if !reflect.DeepEqual(stored, request) {
		err = errors.New("operation mismatch")
		return
	}
	err = tx.Commit()
	return
}
func (databaseKickJournal) finish(id, status string, count int) (adminKickResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result := adminKickResult{OperationID: id}
	err := db.QueryRowContext(ctx, `UPDATE mmo_admin.connection_operations SET status=$2,affected_count=$3,updated_at=now() WHERE operation_id=$1 AND status='Started' RETURNING status,affected_count`, id, status, count).Scan(&result.Status, &result.Count)
	if err == sql.ErrNoRows {
		_, result, err = databaseKickJournal{}.lookup(id)
	}
	return result, err
}
func (databaseKickJournal) lookup(id string) (adminKickRequest, adminKickResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var request adminKickRequest
	result := adminKickResult{OperationID: id}
	var raw []byte
	err := db.QueryRowContext(ctx, `SELECT request,status,affected_count FROM mmo_admin.connection_operations WHERE operation_id=$1`, id).Scan(&raw, &result.Status, &result.Count)
	if err == sql.ErrNoRows {
		result.Status = "Unknown"
		return request, result, nil
	}
	if err == nil {
		err = json.Unmarshal(raw, &request)
	}
	return request, result, err
}
func normalizeKick(request *adminKickRequest) bool {
	request.OperationID = strings.ToLower(request.OperationID)
	request.Reason = strings.TrimSpace(request.Reason)
	if request.Version != 1 || !operationPattern.MatchString(request.OperationID) || len(request.ServerID) != 32 || request.AccountID <= 0 || len(request.ConnectionIDs) < 1 || len(request.ConnectionIDs) > 200 || utf8.RuneCountInString(request.Reason) < 1 || utf8.RuneCountInString(request.Reason) > 500 {
		return false
	}
	sort.Slice(request.ConnectionIDs, func(i, j int) bool { return request.ConnectionIDs[i] < request.ConnectionIDs[j] })
	for i, id := range request.ConnectionIDs {
		if id == 0 || (i > 0 && id == request.ConnectionIDs[i-1]) {
			return false
		}
	}
	return true
}
func (m *adminMonitor) kickTargets(r adminKickRequest) ([]*Client, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if r.ServerID != m.instance {
		return nil, false
	}
	targets := make([]*Client, 0, len(r.ConnectionIDs))
	for _, id := range r.ConnectionIDs {
		var found *Client
		for client, state := range m.clients {
			if state.ID == id && state.AccountID == r.AccountID {
				found = client
				break
			}
		}
		if found == nil {
			return nil, false
		}
		targets = append(targets, found)
	}
	return targets, true
}
func (m *adminMonitor) targetsGone(r adminKickRequest) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if r.ServerID != m.instance {
		return true
	}
	for _, state := range m.clients {
		for _, id := range r.ConnectionIDs {
			if state.ID == id {
				return false
			}
		}
	}
	return true
}
func (m *adminMonitor) executeKick(request adminKickRequest, journal kickJournal) adminKickResult {
	if !normalizeKick(&request) {
		return adminKickResult{Error: "INVALID"}
	}
	kickOperationMu.Lock()
	defer kickOperationMu.Unlock()
	result, fresh, err := journal.begin(request)
	if err != nil {
		return adminKickResult{OperationID: request.OperationID, Error: "UNCERTAIN"}
	}
	if !fresh {
		return result
	} // 応答切断後の再送も、同じ操作では再度ソケットを切らない。
	targets, ok := m.kickTargets(request)
	status, count := "Changed", 0
	if ok {
		for _, client := range targets {
			if client.connection != nil {
				client.connection.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "Closed by administrator"), time.Now().Add(100*time.Millisecond))
				client.connection.Close()
			}
			m.remove(client)
		}
		status, count = "Completed", len(targets)
	}
	result, err = journal.finish(request.OperationID, status, count)
	if err != nil {
		return adminKickResult{OperationID: request.OperationID, Error: "UNCERTAIN"}
	}
	return result
}
func (m *adminMonitor) lookupKick(id string, journal kickJournal) adminKickResult {
	id = strings.ToLower(id)
	if !operationPattern.MatchString(id) {
		return adminKickResult{Error: "INVALID"}
	}
	kickOperationMu.Lock()
	defer kickOperationMu.Unlock()
	request, result, err := journal.lookup(id)
	if err != nil {
		return adminKickResult{OperationID: id, Error: "UNCERTAIN"}
	}
	if result.Status == "Started" && m.targetsGone(request) {
		result, err = journal.finish(id, "TargetsGone", len(request.ConnectionIDs))
		if err != nil {
			return adminKickResult{OperationID: id, Error: "UNCERTAIN"}
		}
	}
	return result
}
func (m *adminMonitor) kickHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 65536)
	defer r.Body.Close()
	var request adminKickRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	if decoder.Decode(&request) != nil {
		json.NewEncoder(w).Encode(adminKickResult{Error: "INVALID"})
		return
	}
	if r.URL.Path == "/kick-status" {
		json.NewEncoder(w).Encode(m.lookupKick(request.OperationID, mmoKickJournal))
		return
	}
	json.NewEncoder(w).Encode(m.executeKick(request, mmoKickJournal))
}
