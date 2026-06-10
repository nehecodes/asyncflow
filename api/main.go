package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

var startTime = time.Now()

type Config struct {
	RedisHost string
	RedisPort int
	RedisPass string
}

type App struct {
	redis *redis.Client
}

func loadConfig() (Config, error) {
	redisHost := getEnv("REDIS_HOST", "127.0.0.1")
	port := getEnv("REDIS_PORT", "6379")

	redisPort, err := strconv.Atoi(port)
	if err != nil {
		return Config{}, fmt.Errorf("invalid REDIS_PORT: %w", err)
	}

	secretPath := getEnv("REDIS_PASSWORD_FILE", "/opt/asyncflow/secrets/redis_password")
	passBytes, err := os.ReadFile(secretPath)
	if err != nil {
		return Config{}, fmt.Errorf("could not read redis password secret: %w", err)
	}

	return Config{
		RedisHost: redisHost,
		RedisPort: redisPort,
		RedisPass: strings.TrimSpace(string(passBytes)),
	}, nil
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return fallback
}

func newApp(cfg Config) (*App, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.RedisHost, cfg.RedisPort),
		Password: cfg.RedisPass,
		DB:       0,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	return &App{redis: client}, nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (a *App) rootHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"message": "API is up and running",
	})
}

func (a *App) healthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"app":    "ok",
		"uptime": time.Since(startTime).String(),
	})
}

func (a *App) jobCreateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	jobID, err := uuid.NewV7()
	if err != nil {
		log.Printf("failed to generate UUIDv7: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	jobIDStr := jobID.String()
	jobKey := fmt.Sprintf("job:%s", jobIDStr)
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	// Queue commands — nothing is sent to Redis yet
	pipe := a.redis.TxPipeline()
	pipe.HSet(ctx, jobKey, "status", "queued")
	pipe.LPush(ctx, "jobs", jobIDStr)

	// Exec sends everything atomically 
	if _, err = pipe.Exec(ctx); err != nil {
		log.Printf("pipeline failed for job %s: %v", jobIDStr, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]any{
		"job_id": jobIDStr,
	})
}

func (a *App) jobGetHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// r.PathValue extracts the {id} segment from the URL
	jobID := r.PathValue("id")
	if jobID == "" {
		http.Error(w, "missing job id", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()
	status, err := a.redis.HGet(ctx, fmt.Sprintf("job:%s", jobID), "status").Result()
	if errors.Is(err, redis.Nil) {
		http.Error(w, "job not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("failed to HGet job %s: %v", jobID, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"job_id": jobID,
		"status": status,
	})
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println(".env not found, using environment variables")
	}

	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("config load failed: %v", err)
	}

	app, err := newApp(cfg)
	if err != nil {
		log.Fatalf("startup failed: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", app.rootHandler)
	mux.HandleFunc("/health", app.healthHandler)
	mux.HandleFunc("/jobs", app.jobCreateHandler)
	mux.HandleFunc("/jobs/{id}", app.jobGetHandler)

	srv := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	fmt.Println("Server is running on port 8080")
	log.Fatal(srv.ListenAndServe())
}
