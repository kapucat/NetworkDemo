package main

import (
	"encoding/json"
	"log"
	"net/http"
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

func main() {
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/api/scores", scoresHandler)
	http.HandleFunc("/api/rankings", rankingsHandler)

	log.Println("Go server started: http://localhost:8080")

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal(err)
	}
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

	log.Printf(
		"score received: player_name=%s stage_id=%d clear_time=%.2f death_count=%d",
		score.PlayerName,
		score.StageID,
		score.ClearTime,
		score.DeathCount,
	)

	w.Header().Set("Content-Type", "application/json")

	response := map[string]string{
		"message": "score received",
	}

	json.NewEncoder(w).Encode(response)
}

func rankingsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stageID := r.URL.Query().Get("stage_id")
	log.Println("ranking requested: stage_id=", stageID)

	rankings := []RankingResponse{
		{
			Rank:       1,
			PlayerName: "Player01",
			ClearTime:  38.20,
			DeathCount: 0,
		},
		{
			Rank:       2,
			PlayerName: "Player02",
			ClearTime:  42.35,
			DeathCount: 1,
		},
		{
			Rank:       3,
			PlayerName: "Player03",
			ClearTime:  49.80,
			DeathCount: 3,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rankings)
}