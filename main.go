package main

import (
	"fmt"
	"net/http"
	"sync/atomic"
)

func main() {
	mux := http.NewServeMux()
	server := http.Server{
		Addr:    ":8080",
		Handler: mux,
	}
	apicfg := apiconfig{}
	mux.Handle("/app/", http.StripPrefix("/app", apicfg.middlewareMetricsInc(http.FileServer(http.Dir(".")))))

	mux.HandleFunc("GET /api/healthz", readiness)
	mux.HandleFunc("GET /admin/metrics", apicfg.requestToScreen)
	mux.HandleFunc("POST /admin/reset", apicfg.resetHits)
	server.ListenAndServe()

}

func readiness(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte("OK"))
}

func (cfg *apiconfig) requestToScreen(w http.ResponseWriter, r *http.Request) {
	hits := fmt.Sprintf("<html>\n  <body>\n    <h1>Welcome, Chirpy Admin</h1>\n    <p>Chirpy has been visited %d times!</p>\n  </body>\n</html>", cfg.fileserverhits.Load())
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte(hits))
	fmt.Printf("Hits: %d", cfg.fileserverhits.Load()) //debug for correct counting
}

func (cfg *apiconfig) resetHits(w http.ResponseWriter, r *http.Request) {
	cfg.fileserverhits.Swap(0)
}

type apiconfig struct {
	fileserverhits atomic.Int32
}

func (cfg *apiconfig) middlewareMetricsInc(next http.Handler) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverhits.Add(1)
		next.ServeHTTP(w, r)
	})
}
