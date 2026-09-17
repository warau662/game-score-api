package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"

	_ "github.com/lib/pq"
)

type PlayerInfo struct {
	Name  string  `json:"name"`
	Score float32 `json:"score"`
}

var db *sql.DB

const apikey = "my-secret-api-key-12345"

func isValidAPIKey(r *http.Request) bool {
	clientKey := r.Header.Get("X-API-KEY")
	return clientKey == apikey
}

func saveScore(w http.ResponseWriter, r *http.Request) {

	if !isValidAPIKey(r) {
		http.Error(w, "Forrbin: Invalid API Key", http.StatusForbidden)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var playerInfo PlayerInfo

	err := json.NewDecoder(r.Body).Decode(&playerInfo)
	if err != nil {
		log.Println("Json decode failed:", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	insertSQL := `INSERT INTO ranking (name,score) VALUES($1,$2);`
	_, err = db.Exec(insertSQL, playerInfo.Name, playerInfo.Score)
	if err != nil {
		log.Println("Insert failed:", err)
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]string{"message": "Score saved successfully"}
	json.NewEncoder(w).Encode(response)
}

func sendScore(w http.ResponseWriter, r *http.Request) {

	if !isValidAPIKey(r) {
		http.Error(w, "Forbidden: Invalid API Key", http.StatusForbidden)
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	scoreSQL := `SELECT name,score FROM ranking ORDER BY score DESC LIMIT 10;`
	rows, err := db.Query(scoreSQL)
	if err != nil {
		http.Error(w, "Get score faild", http.StatusBadRequest)
		return
	}
	defer rows.Close()

	var ranking []PlayerInfo

	for rows.Next() {
		var p PlayerInfo
		if err := rows.Scan(&p.Name, &p.Score); err != nil {
			log.Println("Scan err:", err)
			continue
		}
		ranking = append(ranking, p)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(ranking)

}

func main() {
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		log.Fatal("DATABASE_URL environment variable is not set")
	}

	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	createTable := `CREATE TABLE IF NOT EXISTS ranking(id SERIAL PRIMARY KEY, name TEXT, score REAL);`
	_, err = db.Exec(createTable)
	if err != nil {
		log.Fatal("Failed to create table:", err)
	}

	http.HandleFunc("/save", saveScore)
	http.HandleFunc("/ranking", sendScore)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server running on :%s...", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
