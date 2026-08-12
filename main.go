package main

import (
	"chirpy/apiconfig"
	"chirpy/internal/database"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/joho/godotenv"
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
	dbQueries := database.New(db)
	mux := http.NewServeMux()
	server := http.Server{
		Addr:    ":8080",
		Handler: mux,
	}
	apicfg := apiconfig.Config{Queries: dbQueries}
	mux.Handle("/app/", http.StripPrefix("/app", apicfg.MiddlewareMetricsInc(http.FileServer(http.Dir(".")))))

	mux.HandleFunc("GET /api/healthz", readiness)
	mux.HandleFunc("POST /api/validate_chirp", validation)
	mux.HandleFunc("GET /admin/metrics", apicfg.RequestToScreen)
	mux.HandleFunc("POST /admin/reset", apicfg.ResetHits)

	log.Fatal(server.ListenAndServe())
}

func readiness(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

func validation(w http.ResponseWriter, r *http.Request) {
	type chirpy struct {
		Body string `json:"body"`
	}
	type Invalid struct {
		Error string `json:"error"`
	}
	type cleanedb struct {
		CleanedBody string `json:"cleaned_body"`
	}

	decoder := json.NewDecoder(r.Body)
	params := chirpy{}
	err := decoder.Decode(&params)
	if err != nil {
		InvalidChirpy := Invalid{Error: "Error at decoding chirp"}
		data, err := json.Marshal(InvalidChirpy)
		if err != nil {
			log.Printf("Error marshaling JSON: %s", err)
			w.WriteHeader(500)
			return
		}
		w.WriteHeader(400)
		w.Write(data)
		return
	}
	if len(params.Body) > 140 {
		InvalidChirpy := Invalid{Error: "Chirp is longer than 140 characters"}
		data, err := json.Marshal(InvalidChirpy)
		if err != nil {
			log.Printf("Error marshaling JSON at lenght Check: %s", err)
			w.WriteHeader(500)
			return
		}
		w.WriteHeader(400)
		w.Write(data)
		return
	}

	filteredText := textFiltering(params.Body)
	res := cleanedb{CleanedBody: filteredText}
	data, err := json.Marshal(res)
	if err != nil {
		log.Printf("Error marshaling JSON at validated Check: %s", err)
		w.WriteHeader(500)
		return
	}
	w.WriteHeader(200)
	w.Write(data)

}

func textFiltering(body string) string {
	filteredWords := []string{"kerfuffle", "sharbert", "fornax"}
	splited := strings.Split(body, " ")
	var res []string
	for _, w := range splited {
		for _, p := range filteredWords {
			if strings.ToLower(w) == p {
				w = "****"
			}
		}
		res = append(res, w)
	}
	ret := strings.Join(res, " ")
	return ret
}
