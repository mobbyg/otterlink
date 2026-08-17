package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

const (
	defaultAddr = ":8080"
	defaultDB   = "data/otterlink.db"
)

type healthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}

func main() {
	addr := getenv("OTTERLINK_ADDR", defaultAddr)
	dbPath := getenv("OTTERLINK_DB", defaultDB)

	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		log.Fatalf("create database directory: %v", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("connect to database: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", healthHandler)

	log.Printf("Otter Link server listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(healthResponse{
		Status:  "ok",
		Service: "otter-link",
	})
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
