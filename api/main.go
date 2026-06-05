package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)


var startTime = time.Now()

type Config struct {
	RedisAddr string
	RedisPass string
	ServerPort string
}

type App struct {
	redis *redis.Client
}

func loadConfig() Config {
	return Config{
		RedisAddr: getEnv("REDISADDR", "127.0.0.1:6379"),
		RedisPass: getEnv("REDISPASS", ""),
		ServerPort: getEnv("SERVERPORT", "8080"),

	}
}
func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return fallback
}

func initRedis(cfg Config) (*App, error) {
	client := redis.NewClient(&redis.Options{
		Addr: cfg.RedisAddr,
		Password: cfg.RedisPass,
		DB: 0,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}
	return &App{redis: client}, nil
}


func writeJSON(w http.ResponseWriter, status int, v any){
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func (a *App) rootHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any {
		"message": "Hurray!!! API is up and running",
	})

}
func (a *App) healthHandler(w http.ResponseWriter, r *http.Request){
	writeJSON(w, http.StatusOK, map[string]any{
		"app": "ok",
		"uptime": time.Since(startTime).Seconds(),
	})
}
func (a *App) jobCreateHandler(w http.ResponseWriter, r *http.Request){
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	jobId, err := uuid.NewV7()
	
	if err != nil {
		log.Printf("Failed to generate UUIDv7: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	jobIDStr := jobId.String()

	ctx := r.Context()

	if err = a.redis.LPush(ctx, "jobs", jobIDStr).Err(); err != nil {
		log.Printf("failed to LPush job %s: %v", jobIDStr, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	jobKey := fmt.Sprintf("job:%s", jobIDStr)
	if err = a.redis.HSet(ctx, jobKey, "status", "queued").Err(); err != nil {
		log.Printf("failed to HSet job %s: %v", jobKey, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	
	writeJSON(w, http.StatusAccepted, map[string]any{
		"job": jobIDStr,
	})
}
func (a *App) jobGetHandler( w http.ResponseWriter, r *http.Request) {
	jobID := r.URL.Query().Get("id")
	if jobID == "" {
		http.Error(w, "missing job id", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	status, err := a.redis.HGet(ctx, fmt.Sprintf("job:%s", jobID), "status").Result()
	if err == redis.Nil {
		http.Error(w, "no job found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("failed to HGet job %s: %v", jobID, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"job": jobID,
		"status": status,
	} )
}

func main() {
	godotenv.Load()
	cfg := loadConfig()
	app, err := initRedis(cfg)
	if err != nil {
		log.Fatalf("start up failed: %v", err)
	}
	mux := http.NewServeMux()

	mux.HandleFunc("/", app.rootHandler)
	mux.HandleFunc("/health", app.healthHandler)
	mux.HandleFunc("/jobs/create", app.jobCreateHandler)
	mux.HandleFunc("/jobs/get", app.jobGetHandler)

	srv := &http.Server{
		Addr: ":8080",
		Handler: mux,
		ReadTimeout: 5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout: 60 * time.Second,
	}
	fmt.Println("Server is running on port 8080")
	log.Fatal(srv.ListenAndServe())
}
