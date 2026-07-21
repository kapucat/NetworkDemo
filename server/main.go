package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	_ "github.com/lib/pq"
)

type ScoreRequest struct {
	PlayerName string  `json:"player_name"`
	StageID    int     `json:"stage_id"`
	ClearTime  float64 `json:"clear_time"`
	DeathCount int     `json:"death_count"`
}

type RankingResponse struct {
	Rank       int     `json:"rank"`
	PlayerName string  `json:"player_name"`
	ClearTime  float64 `json:"clear_time"`
	DeathCount int     `json:"death_count"`
}

type BestResponse struct {
	PlayerName string  `json:"player_name"`
	StageID    int     `json:"stage_id"`
	BestTime   float64 `json:"best_time"`
	DeathCount int     `json:"death_count"`
}

type StatsResponse struct {
	ServerStatus     string  `json:"server_status"`
	DBStatus         string  `json:"db_status"`
	StageID          int     `json:"stage_id"`
	TotalScores      int     `json:"total_scores"`
	PlayerCount      int     `json:"player_count"`
	HasScores        bool    `json:"has_scores"`
	BestTime         float64 `json:"best_time"`
	LatestPlayer     string  `json:"latest_player"`
	LatestTime       float64 `json:"latest_time"`
	LatestDeathCount int     `json:"latest_death_count"`
}

var db *sql.DB

func main() {
	initDB()
	initTables()
	defer db.Close()

	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/api/scores", scoresHandler)
	http.HandleFunc("/api/rankings", rankingsHandler)
	http.HandleFunc("/api/best", bestHandler)
	http.HandleFunc("/api/stats", statsHandler)

	log.Println("Go server started: http://localhost:8080")

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal(err)
	}
}

func initDB() {
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "hook_user")
	password := getEnv("DB_PASSWORD", "hook_password")
	dbName := getEnv("DB_NAME", "hook_runner_db")

	connStr := "host=" + host +
		" port=" + port +
		" user=" + user +
		" password=" + password +
		" dbname=" + dbName +
		" sslmode=disable"

	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("DB open failed:", err)
	}

	// PostgreSQLへ同時に接続しすぎないように制限する
	db.SetMaxOpenConns(20)

	// 再利用できる待機中の接続数
	db.SetMaxIdleConns(10)

	// 1つの接続を使い続ける最大時間
	db.SetConnMaxLifetime(30 * time.Minute)

	// 使用されていない接続を保持する最大時間
	db.SetConnMaxIdleTime(5 * time.Minute)

	for i := 1; i <= 10; i++ {
		err = db.Ping()
		if err == nil {
			log.Println("DB connected")
			log.Println("DB connection pool ready")
			return
		}

		log.Printf("DB ping failed. retrying... (%d/10): %v", i, err)
		time.Sleep(2 * time.Second)
	}

	log.Fatal("DB ping failed:", err)
}

func initTables() {
	createScoresTableSQL := `
CREATE TABLE IF NOT EXISTS scores (
    id SERIAL PRIMARY KEY,
    player_name VARCHAR(50) NOT NULL,
    stage_id INT NOT NULL,
    clear_time DOUBLE PRECISION NOT NULL,
    death_count INT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
`

	_, err := db.Exec(createScoresTableSQL)
	if err != nil {
		log.Fatalf("failed to create scores table: %v", err)
	}

	log.Println("scores table ready")

	createRankingIndexSQL := `
CREATE INDEX IF NOT EXISTS idx_scores_ranking
ON scores (
    stage_id,
    player_name,
    clear_time,
    created_at
)
INCLUDE (death_count);
`

	_, err = db.Exec(createRankingIndexSQL)
	if err != nil {
		log.Fatalf("failed to create ranking index: %v", err)
	}

	log.Println("scores ranking index ready")
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	response := map[string]string{
		"status": "ok",
	}

	json.NewEncoder(w).Encode(response)
}

func scoresHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var score ScoreRequest

	err := json.NewDecoder(r.Body).Decode(&score)
	if err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	score.PlayerName = strings.TrimSpace(score.PlayerName)

	if score.PlayerName == "" {
		http.Error(w, "player_name is required", http.StatusBadRequest)
		return
	}

	if utf8.RuneCountInString(score.PlayerName) > 20 {
		http.Error(w, "player_name is too long", http.StatusBadRequest)
		return
	}

	if score.StageID != 1 {
		http.Error(w, "stage_id is invalid", http.StatusBadRequest)
		return
	}

	if score.ClearTime <= 0 {
		http.Error(w, "clear_time is invalid", http.StatusBadRequest)
		return
	}

	if score.ClearTime > 600 {
		http.Error(w, "clear_time is too large", http.StatusBadRequest)
		return
	}

	if score.DeathCount < 0 {
		http.Error(w, "death_count is invalid", http.StatusBadRequest)
		return
	}

	if score.DeathCount > 999 {
		http.Error(w, "death_count is too large", http.StatusBadRequest)
		return
	}

	_, err = db.Exec(
		`
		INSERT INTO scores (player_name, stage_id, clear_time, death_count)
		VALUES ($1, $2, $3, $4)
		`,
		score.PlayerName,
		score.StageID,
		score.ClearTime,
		score.DeathCount,
	)

	if err != nil {
		log.Println("score insert failed:", err)
		http.Error(w, "score insert failed", http.StatusInternalServerError)
		return
	}

	log.Printf(
		"score saved: player_name=%s stage_id=%d clear_time=%.2f death_count=%d",
		score.PlayerName,
		score.StageID,
		score.ClearTime,
		score.DeathCount,
	)

	w.Header().Set("Content-Type", "application/json")

	response := map[string]string{
		"message": "score saved",
	}

	json.NewEncoder(w).Encode(response)
}

func rankingsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stageID := 1

	stageIDText := r.URL.Query().Get("stage_id")
	if stageIDText != "" {
		parsedStageID, err := strconv.Atoi(stageIDText)
		if err == nil {
			stageID = parsedStageID
		}
	}

	rows, err := db.Query(
		`
		SELECT player_name, clear_time, death_count
		FROM (
			SELECT DISTINCT ON (player_name)
				player_name,
				clear_time,
				death_count,
				created_at
			FROM scores
			WHERE stage_id = $1
			ORDER BY player_name, clear_time ASC, created_at ASC
		) AS best_scores
		ORDER BY clear_time ASC
		LIMIT 10
		`,
		stageID,
	)

	if err != nil {
		log.Println("ranking query failed:", err)
		http.Error(w, "ranking query failed", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	rankings := []RankingResponse{}
	rank := 1

	for rows.Next() {
		var ranking RankingResponse

		err := rows.Scan(
			&ranking.PlayerName,
			&ranking.ClearTime,
			&ranking.DeathCount,
		)

		if err != nil {
			log.Println("ranking scan failed:", err)
			http.Error(w, "ranking scan failed", http.StatusInternalServerError)
			return
		}

		ranking.Rank = rank
		rankings = append(rankings, ranking)
		rank++
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rankings)
}

func bestHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	playerName := r.URL.Query().Get("player_name")
	if playerName == "" {
		http.Error(w, "player_name is required", http.StatusBadRequest)
		return
	}

	stageID := 1

	stageIDText := r.URL.Query().Get("stage_id")
	if stageIDText != "" {
		parsedStageID, err := strconv.Atoi(stageIDText)
		if err == nil {
			stageID = parsedStageID
		}
	}

	var best BestResponse
	best.PlayerName = playerName
	best.StageID = stageID

	err := db.QueryRow(
		`
		SELECT player_name, stage_id, clear_time, death_count
		FROM scores
		WHERE player_name = $1
		  AND stage_id = $2
		ORDER BY clear_time ASC, created_at ASC
		LIMIT 1
		`,
		playerName,
		stageID,
	).Scan(
		&best.PlayerName,
		&best.StageID,
		&best.BestTime,
		&best.DeathCount,
	)

	if err == sql.ErrNoRows {
		http.Error(w, "best score not found", http.StatusNotFound)
		return
	}

	if err != nil {
		log.Println("best query failed:", err)
		http.Error(w, "best query failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(best)
}

func statsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stageID := 1

	stageIDText := r.URL.Query().Get("stage_id")
	if stageIDText != "" {
		parsedStageID, err := strconv.Atoi(stageIDText)
		if err == nil {
			stageID = parsedStageID
		}
	}

	stats := StatsResponse{
		ServerStatus: "ok",
		DBStatus:     "ok",
		StageID:      stageID,
	}

	err := db.Ping()
	if err != nil {
		log.Println("stats db ping failed:", err)
		stats.DBStatus = "error"
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(stats)
		return
	}

	err = db.QueryRow(
		`
		SELECT
			COUNT(*),
			COUNT(DISTINCT player_name)
		FROM scores
		WHERE stage_id = $1
		`,
		stageID,
	).Scan(
		&stats.TotalScores,
		&stats.PlayerCount,
	)

	if err != nil {
		log.Println("stats count query failed:", err)
		http.Error(w, "stats count query failed", http.StatusInternalServerError)
		return
	}

	err = db.QueryRow(
		`
		SELECT clear_time
		FROM scores
		WHERE stage_id = $1
		ORDER BY clear_time ASC, created_at ASC
		LIMIT 1
		`,
		stageID,
	).Scan(&stats.BestTime)

	if err == sql.ErrNoRows {
		stats.HasScores = false
	} else if err != nil {
		log.Println("stats best time query failed:", err)
		http.Error(w, "stats best time query failed", http.StatusInternalServerError)
		return
	} else {
		stats.HasScores = true
	}

	err = db.QueryRow(
		`
		SELECT player_name, clear_time, death_count
		FROM scores
		WHERE stage_id = $1
		ORDER BY created_at DESC
		LIMIT 1
		`,
		stageID,
	).Scan(
		&stats.LatestPlayer,
		&stats.LatestTime,
		&stats.LatestDeathCount,
	)

	if err == sql.ErrNoRows {
		stats.LatestPlayer = ""
		stats.LatestTime = 0
		stats.LatestDeathCount = 0
	} else if err != nil {
		log.Println("stats latest score query failed:", err)
		http.Error(w, "stats latest score query failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func getEnv(key string, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	return value
}
