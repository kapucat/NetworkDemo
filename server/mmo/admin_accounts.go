package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

// ログイン・WSの受け入れと、オフラインアカウントの削除を排他制御する。
// DB取引だけでは、認証直後で監視一覧へ登録される前の接続を防げない。
var accountAdmission sync.RWMutex

type adminAccount struct {
	ID                   int64
	Name, Status         string
	CreatedAt, UpdatedAt time.Time
	LastLoginAt          *time.Time
}
type accountAdminRequest struct {
	Version             int
	Action, OperationID string
	AccountID           int64
	Name, Reason        string
}
type accountAdminResponse struct {
	Version             int
	OperationID         string `json:",omitempty"`
	Status              string
	Count, SessionCount int
	Account             *adminAccount `json:",omitempty"`
	Accounts            []adminAccount
	At, ExpiresAt       time.Time
	Error               string `json:",omitempty"`
}

const accountColumns = `id,login_name,status,created_at,updated_at,last_login_at`

func scanAdminAccount(row interface{ Scan(...any) error }) (adminAccount, error) {
	var a adminAccount
	err := row.Scan(&a.ID, &a.Name, &a.Status, &a.CreatedAt, &a.UpdatedAt, &a.LastLoginAt)
	return a, err
}
func validAccountRequest(r *accountAdminRequest) bool {
	r.OperationID = strings.ToLower(r.OperationID)
	r.Reason = strings.TrimSpace(r.Reason)
	if r.Version != 1 {
		return false
	}
	if r.Action == "list" {
		return r.OperationID == "" && r.AccountID == 0 && r.Name == "" && r.Reason == ""
	}
	if !operationPattern.MatchString(r.OperationID) {
		return false
	}
	if r.Action == "lookup" {
		return r.AccountID == 0 && r.Name == "" && r.Reason == ""
	}
	if r.Action != "prepare" && r.Action != "execute" && r.Action != "cancel" {
		return false
	}
	if r.AccountID <= 0 || len(r.Name) < 1 || len(r.Name) > 32 {
		return false
	}
	if r.Action == "execute" {
		return utf8.RuneCountInString(r.Reason) >= 1 && utf8.RuneCountInString(r.Reason) <= 500
	}
	return r.Reason == ""
}
func (m *adminMonitor) accountConnected(id int64) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, c := range m.clients {
		if c.AccountID == id {
			return true
		}
	}
	return false
}
func accountList(ctx context.Context) (accountAdminResponse, error) {
	result := accountAdminResponse{Version: 1, Status: "Listed", At: time.Now().UTC(), Accounts: []adminAccount{}}
	rows, err := db.QueryContext(ctx, `SELECT `+accountColumns+` FROM accounts ORDER BY id LIMIT 5001`)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	for rows.Next() {
		a, e := scanAdminAccount(rows)
		if e != nil {
			return result, e
		}
		result.Accounts = append(result.Accounts, a)
	}
	if err = rows.Err(); err != nil {
		return result, err
	}
	if len(result.Accounts) > 5000 {
		return result, errors.New("account limit exceeded")
	}
	result.Count = len(result.Accounts)
	return result, nil
}

// privateAccountBackupはこの取引とDB監査表の中だけで扱う。管理端末へ認証ハッシュを返さない。
func privateAccountBackup(ctx context.Context, tx *sql.Tx, id int64) (account, sessions []byte, err error) {
	err = tx.QueryRowContext(ctx, `SELECT to_jsonb(a),COALESCE((SELECT jsonb_agg(to_jsonb(s) ORDER BY s.id) FROM account_sessions s WHERE s.account_id=a.id),'[]'::jsonb) FROM accounts a WHERE a.id=$1`, id).Scan(&account, &sessions)
	return
}
func publicAccount(raw []byte) (*adminAccount, error) {
	var saved struct {
		ID          int64
		Name        string `json:"login_name"`
		Status      string
		CreatedAt   time.Time  `json:"created_at"`
		UpdatedAt   time.Time  `json:"updated_at"`
		LastLoginAt *time.Time `json:"last_login_at"`
	}
	if err := json.Unmarshal(raw, &saved); err != nil {
		return nil, err
	}
	return &adminAccount{ID: saved.ID, Name: saved.Name, Status: saved.Status, CreatedAt: saved.CreatedAt, UpdatedAt: saved.UpdatedAt, LastLoginAt: saved.LastLoginAt}, nil
}
func (m *adminMonitor) accountOperation(ctx context.Context, r accountAdminRequest) (result accountAdminResponse, err error) {
	result = accountAdminResponse{Version: 1, OperationID: r.OperationID, At: time.Now().UTC()}
	// 書き込みはログイン・接続の完了を待ち、それ以降の新規接続を取引終了まで止める。
	accountAdmission.Lock()
	defer accountAdmission.Unlock()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return result, err
	}
	defer tx.Rollback()
	// DBを直接使う別プロセスのセッション追加も、確認と削除の間へ入れない。
	_, err = tx.ExecContext(ctx, `LOCK TABLE accounts,account_sessions IN SHARE ROW EXCLUSIVE MODE`)
	if err != nil {
		return result, err
	}
	var backup, sessions []byte
	var accountID int64
	var reason string
	err = tx.QueryRowContext(ctx, `SELECT account_id,account_backup,sessions_backup,status,reason,expires_at FROM mmo_admin.account_operations WHERE operation_id=$1 FOR UPDATE`, r.OperationID).Scan(&accountID, &backup, &sessions, &result.Status, &reason, &result.ExpiresAt)
	if errors.Is(err, sql.ErrNoRows) {
		if r.Action != "prepare" {
			result.Status = "Unknown"
			return result, nil
		}
		if m.accountConnected(r.AccountID) {
			result.Status = "Connected"
			return result, nil
		}
		backup, sessions, err = privateAccountBackup(ctx, tx, r.AccountID)
		if errors.Is(err, sql.ErrNoRows) {
			result.Status = "Changed"
			return result, nil
		}
		if err != nil {
			return result, err
		}
		a, e := publicAccount(backup)
		if e != nil {
			return result, e
		}
		if a.Name != r.Name {
			result.Status = "Changed"
			return result, nil
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO mmo_admin.account_operations(operation_id,account_id,account_backup,sessions_backup,status) VALUES($1,$2,$3::jsonb,$4::jsonb,'Prepared')`, r.OperationID, r.AccountID, string(backup), string(sessions))
		if err != nil {
			return result, err
		}
		err = tx.QueryRowContext(ctx, `SELECT expires_at FROM mmo_admin.account_operations WHERE operation_id=$1`, r.OperationID).Scan(&result.ExpiresAt)
		if err != nil {
			return result, err
		}
		accountID = r.AccountID
		result.Status = "Prepared"
	} else if err != nil {
		return result, err
	}
	result.Account, err = publicAccount(backup)
	if err != nil {
		return result, err
	}
	var savedSessions []json.RawMessage
	if err = json.Unmarshal(sessions, &savedSessions); err != nil {
		return result, err
	}
	result.SessionCount = len(savedSessions)
	if r.Action != "lookup" && (accountID != r.AccountID || result.Account.Name != r.Name) {
		return result, errors.New("operation target mismatch")
	}
	if r.Action == "execute" && reason != "" && reason != r.Reason {
		return result, errors.New("operation reason mismatch")
	}
	if result.Status == "Prepared" {
		var expired bool
		err = tx.QueryRowContext(ctx, `SELECT expires_at<=clock_timestamp() FROM mmo_admin.account_operations WHERE operation_id=$1`, r.OperationID).Scan(&expired)
		if err != nil {
			return result, err
		}
		switch {
		case expired:
			result.Status = "Expired"
		case r.Action == "cancel":
			result.Status = "Cancelled"
		case r.Action == "execute":
			if m.accountConnected(accountID) {
				result.Status = "Connected"
				break
			}
			var same bool
			err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM accounts a WHERE a.id=$1 AND to_jsonb(a)=$2::jsonb AND COALESCE((SELECT jsonb_agg(to_jsonb(s) ORDER BY s.id) FROM account_sessions s WHERE s.account_id=a.id),'[]'::jsonb)=$3::jsonb)`, accountID, string(backup), string(sessions)).Scan(&same)
			if err != nil {
				return result, err
			}
			if !same {
				result.Status = "Changed"
				break
			}
			deleted, e := tx.ExecContext(ctx, `DELETE FROM accounts WHERE id=$1`, accountID)
			if e != nil {
				return result, e
			}
			n, e := deleted.RowsAffected()
			if e != nil || n != 1 {
				return result, errors.New("unexpected account delete count")
			}
			result.Status = "Completed"
		}
		_, err = tx.ExecContext(ctx, `UPDATE mmo_admin.account_operations SET status=$2,reason=CASE WHEN $3='' THEN reason ELSE $3 END,updated_at=now() WHERE operation_id=$1`, r.OperationID, result.Status, r.Reason)
		if err != nil {
			return result, err
		}
	}
	if result.Status == "Completed" {
		result.Count = 1
	}
	err = tx.Commit()
	return
}
func (m *adminMonitor) accountsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	r.Body = http.MaxBytesReader(w, r.Body, 65536)
	defer r.Body.Close()
	var request accountAdminRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if dec.Decode(&request) != nil || !validAccountRequest(&request) {
		json.NewEncoder(w).Encode(accountAdminResponse{Error: "INVALID"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	var result accountAdminResponse
	var err error
	if request.Action == "list" {
		result, err = accountList(ctx)
	} else {
		result, err = m.accountOperation(ctx, request)
	}
	if err != nil {
		result = accountAdminResponse{Version: 1, OperationID: request.OperationID, Error: "UNCERTAIN"}
	}
	json.NewEncoder(w).Encode(result)
}
