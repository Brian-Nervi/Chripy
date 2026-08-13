package apiconfig

import (
	"chirpy/internal/database"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

type Config struct {
	fileserverhits atomic.Int32
	Queries        *database.Queries
	Platform       string
}

type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
}

func (cfg *Config) RequestToScreen(w http.ResponseWriter, r *http.Request) {
	hits := fmt.Sprintf("<html>\n  <body>\n    <h1>Welcome, Chirpy Admin</h1>\n    <p>Chirpy has been visited %d times!</p>\n  </body>\n</html>", cfg.fileserverhits.Load())
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(200)
	w.Write([]byte(hits))
	fmt.Printf("Hits: %d", cfg.fileserverhits.Load()) //debug for correct counting
}

func (cfg *Config) ResetHits(w http.ResponseWriter, r *http.Request) {
	if cfg.Platform != "dev" {
		w.WriteHeader(403)
		w.Write([]byte("Forbiden"))
		return
	}
	cfg.fileserverhits.Swap(0)
	err := cfg.Queries.DeleteAllUsers(r.Context())
	if err != nil {
		log.Fatal(err)
	}
}

func (cfg *Config) MiddlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverhits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *Config) CreateUser(w http.ResponseWriter, r *http.Request) {
	type email struct {
		Email string `json:"email"`
	}
	decoder := json.NewDecoder(r.Body)
	params := email{}
	err := decoder.Decode(&params)
	if err != nil {
		log.Fatal(err)
		return
	}
	user, err := cfg.Queries.CreateUser(r.Context(), params.Email)
	if err != nil {
		log.Fatal(err)
		return
	}
	res := User{
		ID:        user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Email:     user.Email,
	}
	data, err := json.Marshal(res)
	if err != nil {
		log.Fatal(err)
		return
	}
	w.WriteHeader(201)
	w.Write(data)
}

func (cfg *Config) SendChirp(w http.ResponseWriter, r *http.Request) {

	decoder := json.NewDecoder(r.Body)
	params := chirpy{}
	err := decoder.Decode(&params)
	if err != nil {
		log.Fatal(err)
		return
	}
	filteredmsg, err := validationAndFiltering(params.Body)
	if err != nil {
		log.Fatal(err)
		return
	}
	params.Body = filteredmsg
	msg, err := cfg.Queries.CreateChirp(r.Context(), database.CreateChirpParams{Body: params.Body, UserID: params.UserID})
	if err != nil {
		log.Fatal(err)
		return
	}
	res := responseJSON{
		ID:        msg.ID,
		CreatedAt: msg.CreatedAt,
		UpdatedAt: msg.UpdatedAt,
		Body:      msg.Body,
		UserID:    msg.UserID,
	}
	data, err := json.Marshal(res)
	if err != nil {
		log.Fatal(err)
		return
	}
	w.WriteHeader(201)
	w.Write(data)

}

type responseJSON struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserID    uuid.UUID `json:"user_id"`
}

type chirpy struct {
	Body   string    `json:"body"`
	UserID uuid.UUID `json:"user_id"`
}

func validationAndFiltering(msg string) (string, error) {
	if len(msg) > 140 {
		return "", errors.New("Chirps cant exceed 140 characters")
	}
	filteredText := textFiltering(msg)
	return filteredText, nil
}

func textFiltering(body string) string {
	filteredWords := []string{"kerfuffle", "sharbert", "fornax"} //words to filter
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

func (cfg *Config) GetChirps(w http.ResponseWriter, r *http.Request) {
	Chirps, err := cfg.Queries.GetAllChirps(r.Context())
	if err != nil {
		w.WriteHeader(400)
		w.Write([]byte("Failed getting Chirp"))
	}
	var ret []responseJSON
	for _, c := range Chirps {
		res := responseJSON{
			ID:        c.ID,
			CreatedAt: c.CreatedAt,
			UpdatedAt: c.UpdatedAt,
			Body:      c.Body,
			UserID:    c.UserID,
		}
		ret = append(ret, res)
	}
	data, err := json.Marshal(ret)
	if err != nil {
		w.WriteHeader(400)
		log.Fatal(err)
		return
	}
	w.WriteHeader(200)
	w.Write(data)
}

func (cfg *Config) GetChirpById(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("chirpID")
	parsedid, err := uuid.Parse(id)
	if err != nil {
		w.WriteHeader(404)
		fmt.Printf("Error:%v", err)
		return
	}
	msg, err := cfg.Queries.GetChirpByID(r.Context(), parsedid)
	if err != nil {
		w.WriteHeader(404)
		fmt.Printf("Error:%v", err)
		return
	}
	res := responseJSON{
		ID:        msg.ID,
		CreatedAt: msg.CreatedAt,
		UpdatedAt: msg.UpdatedAt,
		Body:      msg.Body,
		UserID:    msg.UserID,
	}
	data, err := json.Marshal(res)
	if err != nil {
		fmt.Printf("Error:%v", err)
		return
	}
	w.WriteHeader(200)
	w.Write(data)
}
