package apiconfig

import (
	"fmt"
	"net/http"
	"sync/atomic"
)

type Config struct {
	fileserverhits atomic.Int32
}

func (cfg *Config) RequestToScreen(w http.ResponseWriter, r *http.Request) {
	hits := fmt.Sprintf("<html>\n  <body>\n    <h1>Welcome, Chirpy Admin</h1>\n    <p>Chirpy has been visited %d times!</p>\n  </body>\n</html>", cfg.fileserverhits.Load())
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte(hits))
	fmt.Printf("Hits: %d", cfg.fileserverhits.Load()) //debug for correct counting
}

func (cfg *Config) ResetHits(w http.ResponseWriter, r *http.Request) {
	cfg.fileserverhits.Swap(0)
}

func (cfg *Config) MiddlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverhits.Add(1)
		next.ServeHTTP(w, r)
	})
}
