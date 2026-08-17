package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/mobbyg/otterlink/server/internal/accounts"
	"github.com/mobbyg/otterlink/server/internal/api"
	"github.com/mobbyg/otterlink/server/internal/db"
	"github.com/mobbyg/otterlink/server/internal/protocol"
	_ "modernc.org/sqlite"
)

const (
	defaultAddr         = ":8080"
	defaultProtocolAddr = ":8023"
	defaultDB           = "data/otterlink.db"
)

type healthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}

func main() {
	addr := getenv("OTTERLINK_ADDR", defaultAddr)
	protocolAddr := getenv("OTTERLINK_PROTOCOL_ADDR", defaultProtocolAddr)
	dbPath := getenv("OTTERLINK_DB", defaultDB)

	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		log.Fatalf("create database directory: %v", err)
	}

	database, err := sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer database.Close()

	if err := database.Ping(); err != nil {
		log.Fatalf("connect to database: %v", err)
	}
	if err := db.Initialize(database); err != nil {
		log.Fatalf("initialize database: %v", err)
	}

	authAPI := api.AuthAPI{Accounts: accounts.Service{DB: database}}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", healthHandler)
	mux.HandleFunc("POST /api/auth/register", authAPI.Register)
	mux.HandleFunc("POST /api/auth/login", authAPI.Login)
	mux.HandleFunc("POST /api/auth/logout", authAPI.Logout)
	mux.HandleFunc("GET /api/me", authAPI.Me)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	protocolServer := &protocol.Server{
		Addr:    protocolAddr,
		Handler: protocol.DefaultHandler{},
		Logger:  log.Default(),
	}
	protocolErr := make(chan error, 1)
	go func() {
		protocolErr <- protocolServer.ListenAndServe(ctx)
	}()

	log.Printf("Otter Link HTTP server listening on %s", addr)
	log.Printf("Otter Link protocol listening on %s", protocolAddr)

	if err := http.ListenAndServe(addr, mux); err != nil {
		stop()
		log.Fatalf("HTTP server stopped: %v", err)
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
