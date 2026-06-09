package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"

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

var db *sql.DB

func main() {
	initDB()
	defer db.Close()

	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/api/scores", scoresHandler)
	http.HandleFunc("/api/rankings", rankingsHandler)

	log.Println("Go server started: http://localhost:8080")

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal(err)
	}
}

func initDB() {
	connStr := "host=localhost port=5432 user=hook_user password=hook_password dbname=hook_runner_db sslmode=disable"

	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("DB open failed:", err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatal("DB ping failed:", err)
	}

	log.Println("DB connected")
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

	if score.PlayerName == "" {
		http.Error(w, "player_name is required", http.StatusBadRequest)
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
		FROM scores
		WHERE stage_id = $1
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