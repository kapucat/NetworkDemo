package main

import (
	"crypto/rand"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/lib/pq"
	"golang.org/x/crypto/argon2"
)

const (
	argonTime       uint32 = 2
	argonMemory     uint32 = 19 * 1024 // KiB単位：19 MiB
	argonThreads    uint8  = 1
	argonKeyLength  uint32 = 32
	argonSaltLength        = 16
)

var loginNamePattern = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

type AccountRegisterRequest struct {
	LoginName string `json:"login_name"`
	Password  string `json:"password"`
}

type AccountRegisterResponse struct {
	Message   string `json:"message"`
	AccountID int64  `json:"account_id,omitempty"`
}

type AccountLoginRequest struct {
	LoginName string `json:"login_name"`
	Password  string `json:"password"`
}

type AccountLoginResponse struct {
	Message   string `json:"message"`
	AccountID int64  `json:"account_id,omitempty"`
	LoginName string `json:"login_name,omitempty"`
}

// MMOアカウント用テーブルを準備する。
func initAccountsTable() {
	const createAccountsTableSQL = `
CREATE TABLE IF NOT EXISTS accounts (
    id BIGSERIAL PRIMARY KEY,
    login_name VARCHAR(32) NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    status VARCHAR(16) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_login_at TIMESTAMPTZ
);
`

	_, err := db.Exec(createAccountsTableSQL)
	if err != nil {
		log.Fatalf("failed to create accounts table: %v", err)
	}

	log.Println("accounts table ready")
}

// POST /api/accounts/register
func accountRegisterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeAccountJSON(
			w,
			http.StatusMethodNotAllowed,
			AccountRegisterResponse{Message: "method not allowed"},
		)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 4*1024)
	defer r.Body.Close()

	var request AccountRegisterRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&request); err != nil {
		writeAccountJSON(
			w,
			http.StatusBadRequest,
			AccountRegisterResponse{Message: "invalid json"},
		)
		return
	}

	request.LoginName =
		strings.ToLower(strings.TrimSpace(request.LoginName))

	loginNameLength := utf8.RuneCountInString(request.LoginName)

	if loginNameLength < 3 || loginNameLength > 20 {
		writeAccountJSON(
			w,
			http.StatusBadRequest,
			AccountRegisterResponse{
				Message: "login_name must be between 3 and 20 characters",
			},
		)
		return
	}

	if !loginNamePattern.MatchString(request.LoginName) {
		writeAccountJSON(
			w,
			http.StatusBadRequest,
			AccountRegisterResponse{
				Message: "login_name may contain only letters, numbers, and underscores",
			},
		)
		return
	}

	passwordLength := utf8.RuneCountInString(request.Password)

	if passwordLength < 8 || passwordLength > 64 {
		writeAccountJSON(
			w,
			http.StatusBadRequest,
			AccountRegisterResponse{
				Message: "password must be between 8 and 64 characters",
			},
		)
		return
	}

	passwordHash, err := hashPassword(request.Password)
	if err != nil {
		log.Println("password hashing failed:", err)

		writeAccountJSON(
			w,
			http.StatusInternalServerError,
			AccountRegisterResponse{
				Message: "account creation failed",
			},
		)
		return
	}

	var accountID int64

	err = db.QueryRow(
		`
INSERT INTO accounts (login_name, password_hash)
VALUES ($1, $2)
RETURNING id
`,
		request.LoginName,
		passwordHash,
	).Scan(&accountID)

	if err != nil {
		var pqError *pq.Error

		if errors.As(err, &pqError) &&
			string(pqError.Code) == "23505" {
			writeAccountJSON(
				w,
				http.StatusConflict,
				AccountRegisterResponse{
					Message: "login_name already exists",
				},
			)
			return
		}

		log.Println("account insert failed:", err)

		writeAccountJSON(
			w,
			http.StatusInternalServerError,
			AccountRegisterResponse{
				Message: "account creation failed",
			},
		)
		return
	}

	log.Printf(
		"account created: id=%d login_name=%s",
		accountID,
		request.LoginName,
	)

	writeAccountJSON(
		w,
		http.StatusCreated,
		AccountRegisterResponse{
			Message:   "account created",
			AccountID: accountID,
		},
	)
}

// POST /api/accounts/login
func accountLoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeAccountJSON(
			w,
			http.StatusMethodNotAllowed,
			AccountLoginResponse{Message: "method not allowed"},
		)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 4*1024)
	defer r.Body.Close()

	var request AccountLoginRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&request); err != nil {
		writeAccountJSON(
			w,
			http.StatusBadRequest,
			AccountLoginResponse{Message: "invalid json"},
		)
		return
	}

	request.LoginName =
		strings.ToLower(strings.TrimSpace(request.LoginName))

	if request.LoginName == "" || request.Password == "" {
		writeAccountJSON(
			w,
			http.StatusBadRequest,
			AccountLoginResponse{
				Message: "login_name and password are required",
			},
		)
		return
	}

	var accountID int64
	var passwordHash string
	var status string

	err := db.QueryRow(
		`
SELECT id, password_hash, status
FROM accounts
WHERE login_name = $1
`,
		request.LoginName,
	).Scan(
		&accountID,
		&passwordHash,
		&status,
	)

	if errors.Is(err, sql.ErrNoRows) {
		writeInvalidLoginResponse(w)
		return
	}

	if err != nil {
		log.Println("account login query failed:", err)

		writeAccountJSON(
			w,
			http.StatusInternalServerError,
			AccountLoginResponse{Message: "login failed"},
		)
		return
	}

	if status != "active" {
		writeAccountJSON(
			w,
			http.StatusForbidden,
			AccountLoginResponse{Message: "account is unavailable"},
		)
		return
	}

	passwordMatches, err :=
		verifyPassword(request.Password, passwordHash)

	if err != nil {
		log.Println("password verification failed:", err)

		writeAccountJSON(
			w,
			http.StatusInternalServerError,
			AccountLoginResponse{Message: "login failed"},
		)
		return
	}

	if !passwordMatches {
		writeInvalidLoginResponse(w)
		return
	}

	_, err = db.Exec(
		`
UPDATE accounts
SET last_login_at = CURRENT_TIMESTAMP,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
`,
		accountID,
	)

	if err != nil {
		log.Println("last login update failed:", err)

		writeAccountJSON(
			w,
			http.StatusInternalServerError,
			AccountLoginResponse{Message: "login failed"},
		)
		return
	}

	log.Printf(
		"account login succeeded: id=%d login_name=%s",
		accountID,
		request.LoginName,
	)

	writeAccountJSON(
		w,
		http.StatusOK,
		AccountLoginResponse{
			Message:   "login successful",
			AccountID: accountID,
			LoginName: request.LoginName,
		},
	)
}

func writeInvalidLoginResponse(w http.ResponseWriter) {
	writeAccountJSON(
		w,
		http.StatusUnauthorized,
		AccountLoginResponse{
			Message: "invalid login_name or password",
		},
	)
}

func hashPassword(password string) (string, error) {
	salt := make([]byte, argonSaltLength)

	_, err := rand.Read(salt)
	if err != nil {
		return "", fmt.Errorf("salt generation failed: %w", err)
	}

	hash := argon2.IDKey(
		[]byte(password),
		salt,
		argonTime,
		argonMemory,
		argonThreads,
		argonKeyLength,
	)

	encodedSalt := base64.RawStdEncoding.EncodeToString(salt)
	encodedHash := base64.RawStdEncoding.EncodeToString(hash)

	encodedPasswordHash := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		argonMemory,
		argonTime,
		argonThreads,
		encodedSalt,
		encodedHash,
	)

	return encodedPasswordHash, nil
}

func verifyPassword(
	password string,
	encodedPasswordHash string,
) (bool, error) {
	parts := strings.Split(encodedPasswordHash, "$")

	if len(parts) != 6 {
		return false, errors.New("invalid password hash format")
	}

	if parts[1] != "argon2id" {
		return false, errors.New("unsupported password hash")
	}

	var version int

	_, err := fmt.Sscanf(parts[2], "v=%d", &version)
	if err != nil {
		return false, fmt.Errorf(
			"invalid argon2 version: %w",
			err,
		)
	}

	if version != argon2.Version {
		return false, errors.New("unsupported argon2 version")
	}

	var memory uint32
	var timeCost uint32
	var threads uint8

	_, err = fmt.Sscanf(
		parts[3],
		"m=%d,t=%d,p=%d",
		&memory,
		&timeCost,
		&threads,
	)

	if err != nil {
		return false, fmt.Errorf(
			"invalid argon2 parameters: %w",
			err,
		)
	}

	salt, err :=
		base64.RawStdEncoding.DecodeString(parts[4])

	if err != nil {
		return false, fmt.Errorf(
			"invalid password salt: %w",
			err,
		)
	}

	expectedHash, err :=
		base64.RawStdEncoding.DecodeString(parts[5])

	if err != nil {
		return false, fmt.Errorf(
			"invalid password hash: %w",
			err,
		)
	}

	actualHash := argon2.IDKey(
		[]byte(password),
		salt,
		timeCost,
		memory,
		threads,
		uint32(len(expectedHash)),
	)

	passwordMatches :=
		subtle.ConstantTimeCompare(
			actualHash,
			expectedHash,
		) == 1

	return passwordMatches, nil
}

func writeAccountJSON(w http.ResponseWriter, statusCode int, response any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Println("account response encode failed:", err)
	}
}
