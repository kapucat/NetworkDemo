package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestAccountRequestValidation(t *testing.T) {
	empty, _ := json.Marshal(accountAdminResponse{Version: 1, Status: "Listed", Accounts: []adminAccount{}})
	if !strings.Contains(string(empty), `"Accounts":[]`) || strings.Contains(string(empty), `"OperationID"`) {
		t.Fatal("empty account list wire format")
	}
	for _, action := range []string{"prepare", "execute", "cancel", "lookup", "list"} {
		r := accountAdminRequest{Version: 1, Action: action, OperationID: kickTestID, AccountID: 42, Name: "test_user"}
		if action == "execute" {
			r.Reason = "日本語の削除確認😀"
		}
		if action == "lookup" {
			r.AccountID = 0
			r.Name = ""
		}
		if action == "list" {
			r.OperationID = ""
			r.AccountID = 0
			r.Name = ""
		}
		if !validAccountRequest(&r) {
			t.Fatal(action)
		}
	}
	for _, r := range []accountAdminRequest{
		{Version: 2, Action: "list"},
		{Version: 1, Action: "drop-all", OperationID: kickTestID, AccountID: 42, Name: "user"},
		{Version: 1, Action: "execute", OperationID: kickTestID, AccountID: 42, Name: "user", Reason: " "},
		{Version: 1, Action: "execute", OperationID: kickTestID, AccountID: 42, Name: "user", Reason: strings.Repeat("a", 501)},
		{Version: 1, Action: "prepare", OperationID: "not-uuid", AccountID: 42, Name: "user"},
		{Version: 1, Action: "prepare", OperationID: kickTestID, AccountID: -1, Name: "user"},
	} {
		if validAccountRequest(&r) {
			t.Fatal("invalid account request accepted")
		}
	}
	rr := httptest.NewRecorder()
	newAdminMonitor().accountsHandler(rr, httptest.NewRequest("GET", "/accounts", nil))
	if rr.Code != 405 {
		t.Fatal("write method guard")
	}
	rr = httptest.NewRecorder()
	newAdminMonitor().accountsHandler(rr, httptest.NewRequest("POST", "/accounts", strings.NewReader(`{"Version":1,"Action":"list","Password":"must-not-be-accepted"}`)))
	if !strings.Contains(rr.Body.String(), "INVALID") {
		t.Fatal("unknown field guard")
	}
}

// 専用の使い捨てDB以外では実行しない。本番のaccountsへテストデータを入れない。
func TestAccountDatabaseIntegration(t *testing.T) {
	dsn := os.Getenv("MMO_ACCOUNT_TEST_DSN")
	if dsn == "" {
		t.Skip("set MMO_ACCOUNT_TEST_DSN for a disposable PostgreSQL database")
	}
	testDB, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer testDB.Close()
	var name string
	if err = testDB.QueryRow(`SELECT current_database()`).Scan(&name); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(name, "networkdemo_account_test_") {
		t.Fatal("refusing to modify a non-test database")
	}
	old := db
	db = testDB
	defer func() { db = old }()
	must := func(query string, args ...any) {
		t.Helper()
		if _, e := db.Exec(query, args...); e != nil {
			t.Fatal(e)
		}
	}
	must(`CREATE TABLE accounts(id BIGSERIAL PRIMARY KEY,login_name varchar(32) UNIQUE NOT NULL,password_hash text NOT NULL,status varchar(16) NOT NULL DEFAULT 'active',created_at timestamptz NOT NULL DEFAULT now(),updated_at timestamptz NOT NULL DEFAULT now(),last_login_at timestamptz);
 CREATE TABLE account_sessions(id BIGSERIAL PRIMARY KEY,account_id bigint NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,token_hash char(64) UNIQUE NOT NULL,expires_at timestamptz NOT NULL,created_at timestamptz NOT NULL DEFAULT now());
 CREATE TABLE scores(id bigint PRIMARY KEY,player_name text NOT NULL);INSERT INTO scores VALUES(99,'delete_target');`)
	for _, file := range []string{"002_admin_kicks.sql", "003_admin_accounts.sql"} {
		raw, e := os.ReadFile(file)
		if e != nil {
			t.Fatal(e)
		}
		must(string(raw))
	}
	m := newAdminMonitor()
	ctx := context.Background()
	seed := func(n string) int64 {
		t.Helper()
		var id int64
		if e := db.QueryRow(`INSERT INTO accounts(login_name,password_hash) VALUES($1,'PRIVATE_HASH_BACKUP') RETURNING id`, n).Scan(&id); e != nil {
			t.Fatal(e)
		}
		return id
	}
	request := func(id int64, n string) accountAdminRequest {
		return accountAdminRequest{Version: 1, Action: "prepare", OperationID: newMonitorInstance()[0:8] + "-2222-3333-4444-555555555555", AccountID: id, Name: n}
	}
	call := func(r accountAdminRequest) accountAdminResponse {
		t.Helper()
		out, e := m.accountOperation(ctx, r)
		if e != nil {
			t.Fatal(e)
		}
		return out
	}
	exists := func(id int64) bool {
		t.Helper()
		var found bool
		if e := db.QueryRow(`SELECT EXISTS(SELECT 1 FROM accounts WHERE id=$1)`, id).Scan(&found); e != nil {
			t.Fatal(e)
		}
		return found
	}
	target := seed("delete_target")
	other := seed("keep_target")
	token, e := createSessionToken(target)
	if e != nil {
		t.Fatal(e)
	}
	listing, e := accountList(ctx)
	if e != nil || len(listing.Accounts) != 2 {
		t.Fatal("list", e)
	}
	raw, e := json.Marshal(listing)
	if e != nil || strings.Contains(string(raw), "PRIVATE_HASH") || strings.Contains(string(raw), "token_hash") {
		t.Fatal("credentials leaked in list")
	}
	r := request(target, "delete_target")
	p := call(r)
	if p.Status != "Prepared" || p.SessionCount != 1 || !exists(target) {
		t.Fatal("prepare changed account")
	}
	raw, _ = json.Marshal(p)
	if strings.Contains(string(raw), "PRIVATE_HASH") || strings.Contains(string(raw), "token_hash") {
		t.Fatal("credentials leaked in plan")
	}
	r.Action = "execute"
	r.Reason = "削除テスト😀"
	// 応答切断後の重複送信を同時に実行しても同じ完了結果となる。
	var wg sync.WaitGroup
	results := make(chan accountAdminResponse, 2)
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); out, e := m.accountOperation(ctx, r); results <- out; errs <- e }()
	}
	wg.Wait()
	close(results)
	close(errs)
	for e := range errs {
		if e != nil {
			t.Fatal(e)
		}
	}
	for out := range results {
		if out.Status != "Completed" || out.Count != 1 {
			t.Fatal("idempotent delete")
		}
	}
	if exists(target) || !exists(other) {
		t.Fatal("wrong account deleted")
	}
	var sessions, scores int
	db.QueryRow(`SELECT count(*) FROM account_sessions WHERE account_id=$1`, target).Scan(&sessions)
	db.QueryRow(`SELECT count(*) FROM scores WHERE id=99`).Scan(&scores)
	if sessions != 0 || scores != 1 {
		t.Fatal("session cascade or ranking isolation")
	}
	var backup string
	if e = db.QueryRow(`SELECT account_backup->>'password_hash' FROM mmo_admin.account_operations WHERE operation_id=$1`, r.OperationID).Scan(&backup); e != nil || backup != "PRIVATE_HASH_BACKUP" {
		t.Fatal("private backup missing", e)
	}
	if _, _, e = validateSessionToken(token); e == nil {
		t.Fatal("deleted account token accepted")
	}
	httpResult := httptest.NewRecorder()
	websocketHandler(httpResult, httptest.NewRequest("GET", "/ws?token="+url.QueryEscape(token), nil))
	if httpResult.Code != 401 {
		t.Fatal("deleted token admitted to websocket")
	}
	replacement := seed("delete_target")
	if call(r).Status != "Completed" || !exists(replacement) {
		t.Fatal("replay deleted a newly registered account")
	}
	wrong := r
	wrong.Reason = "different"
	if _, e = m.accountOperation(ctx, wrong); e == nil {
		t.Fatal("different reason replay")
	}
	wrong = r
	wrong.AccountID = other
	wrong.Name = "keep_target"
	if _, e = m.accountOperation(ctx, wrong); e == nil {
		t.Fatal("different target replay")
	}
	// 確認後のログインでセッションが増えたら削除しない。
	changed := request(other, "keep_target")
	call(changed)
	if _, e = createSessionToken(other); e != nil {
		t.Fatal(e)
	}
	changed.Action = "execute"
	changed.Reason = "changed"
	if call(changed).Status != "Changed" || !exists(other) {
		t.Fatal("changed session accepted")
	}
	// 確認中に接続したプレイヤーも削除しない。
	connected := request(other, "keep_target")
	call(connected)
	c := &Client{accountID: other}
	m.add(c, time.Now())
	connected.Action = "execute"
	connected.Reason = "connected"
	if call(connected).Status != "Connected" || !exists(other) {
		t.Fatal("connected account deleted")
	}
	m.remove(c)
	cancel := request(other, "keep_target")
	call(cancel)
	cancel.Action = "cancel"
	if call(cancel).Status != "Cancelled" {
		t.Fatal("cancel")
	}
	cancel.Action = "execute"
	cancel.Reason = "cancelled"
	if call(cancel).Status != "Cancelled" || !exists(other) {
		t.Fatal("cancelled request executed")
	}
	expired := request(other, "keep_target")
	call(expired)
	must(`UPDATE mmo_admin.account_operations SET expires_at=now()-interval '1 second' WHERE operation_id=$1`, expired.OperationID)
	expired.Action = "execute"
	expired.Reason = "expired"
	if call(expired).Status != "Expired" || !exists(other) {
		t.Fatal("expired request executed")
	}
	missing := request(9223372036854775807, "missing")
	if call(missing).Status != "Changed" {
		t.Fatal("missing account")
	}
	// DB削除失敗時は監査完了だけを記録せず、アカウントもセッションも維持する。
	rollback := request(other, "keep_target")
	call(rollback)
	must(`CREATE FUNCTION reject_test_delete() RETURNS trigger LANGUAGE plpgsql AS $$BEGIN RAISE EXCEPTION 'test rollback';END$$;CREATE TRIGGER test_delete_failure BEFORE DELETE ON accounts FOR EACH ROW EXECUTE FUNCTION reject_test_delete()`)
	rollback.Action = "execute"
	rollback.Reason = "rollback"
	if _, e = m.accountOperation(ctx, rollback); e == nil || !exists(other) {
		t.Fatal("failed delete did not roll back")
	}
	lookup := accountAdminRequest{Version: 1, Action: "lookup", OperationID: rollback.OperationID}
	if call(lookup).Status != "Prepared" {
		t.Fatal("failed transaction marked completed")
	}
	must(`DROP TRIGGER test_delete_failure ON accounts`)
	// 認証処理と削除が重なる場合、認証側の保護区間が終わるまで削除を待つ。
	accountAdmission.RLock()
	done := make(chan accountAdminResponse, 1)
	errDone := make(chan error, 1)
	go func() { out, e := m.accountOperation(ctx, rollback); done <- out; errDone <- e }()
	select {
	case <-done:
		accountAdmission.RUnlock()
		t.Fatal("admission gate bypassed")
	case <-time.After(40 * time.Millisecond):
	}
	accountAdmission.RUnlock()
	if out := <-done; out.Status != "Completed" {
		t.Fatal("delete after admission", out.Status)
	}
	if e := <-errDone; e != nil {
		t.Fatal(e)
	}
	lookup.OperationID = rollback.OperationID
	if call(lookup).Status != "Completed" {
		t.Fatal("durable lookup")
	}
	emptyList, e := accountList(ctx)
	if e != nil || len(emptyList.Accounts) != 1 || emptyList.Accounts[0].ID != replacement {
		t.Fatal("only replacement account should remain", e)
	}
}
