package main

import (
	"chirpy/apiconfig"
	"chirpy/internal/database"
	"database/sql"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		log.Fatal("DB_URL must be set")
	}
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Error opening database: %s", err)
	}
	platform := os.Getenv("PLATFORM")
	if platform == "" {
		log.Fatal("No platform provided")
	}
	dbQueries := database.New(db)
	mux := http.NewServeMux()
	server := http.Server{
		Addr:    ":8080",
		Handler: mux,
	}
	apicfg := apiconfig.Config{Queries: dbQueries, Platform: platform}
	mux.Handle("/app/", http.StripPrefix("/app", apicfg.MiddlewareMetricsInc(http.FileServer(http.Dir(".")))))

	mux.HandleFunc("GET /api/healthz", readiness)
	mux.HandleFunc("POST /api/users", apicfg.CreateUser)
	mux.HandleFunc("POST /api/chirps", apicfg.SendChirp)
	mux.HandleFunc("GET /admin/metrics", apicfg.RequestToScreen)
	mux.HandleFunc("POST /admin/reset", apicfg.ResetHits)
	mux.HandleFunc("GET /api/chirps", apicfg.GetChirps)
	mux.HandleFunc("GET /api/chirps/{chirpID}", apicfg.GetChirpById)

	log.Fatal(server.ListenAndServe())
}

func readiness(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}
