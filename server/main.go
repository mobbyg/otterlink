package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/mobbyg/otterlink/server/internal/accounts"
	"github.com/mobbyg/otterlink/server/internal/api"
	"github.com/mobbyg/otterlink/server/internal/db"
	"github.com/mobbyg/otterlink/server/internal/presence"
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

	accountService := accounts.Service{DB: database}
	presenceService := presence.NewService()
	authAPI := api.AuthAPI{Accounts: accountService}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", healthHandler)
	mux.HandleFunc("POST /api/auth/register", authAPI.Register)
	mux.HandleFunc("POST /api/auth/login", authAPI.Login)
	mux.HandleFunc("POST /api/auth/logout", authAPI.Logout)
	mux.HandleFunc("GET /api/me", authAPI.Me)

	httpServer := &http.Server{Addr: addr, Handler: mux}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	protocolServer := &protocol.Server{
		Addr:    protocolAddr,
		Handler: protocol.DefaultHandler{Accounts: accountService, Presence: presenceService},
		Logger:  log.Default(),
	}
	protocolErr := make(chan error, 1)
	go func() {
		protocolErr <- protocolServer.ListenAndServe(ctx)
	}()

	log.Printf("Otter Link HTTP server listening on %s", addr)
	log.Printf("Otter Link protocol listening on %s", protocolAddr)

	httpErr := make(chan error, 1)
	go func() {
		httpErr <- httpServer.ListenAndServe()
	}()

	select {
	case err := <-httpErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP server stopped: %v", err)
		}
	case err := <-protocolErr:
		if err != nil {
			log.Fatalf("protocol server stopped: %v", err)
		}
	case <-ctx.Done():
		log.Printf("Shutdown signal received")
	}

	stop()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP shutdown: %v", err)
	}

	select {
	case err := <-protocolErr:
		if err != nil {
			log.Printf("protocol shutdown: %v", err)
		}
	case <-shutdownCtx.Done():
		log.Printf("shutdown timeout reached")
	}

	log.Printf("Otter Link stopped")
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
