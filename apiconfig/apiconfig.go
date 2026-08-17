package apiconfig

import (
	"chirpy/internal/auth"
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
	Secret         string
}

type User struct {
	ID             uuid.UUID `json:"id"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	Email          string    `json:"email"`
	HashedPassword string    `json:"-"`
	Token          string    `json:"token"`
	RefreshToken   string    `json:"refresh_token"`
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
	type register struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	decoder := json.NewDecoder(r.Body)
	params := register{}
	err := decoder.Decode(&params)
	if err != nil {
		w.WriteHeader(400)
		fmt.Printf("Error:%v", err)
		return
	}
	hashedPassword, err := auth.HashPassword(params.Password)
	if err != nil {
		w.WriteHeader(500)
		fmt.Printf("Error:%v", err)
		return
	}
	user, err := cfg.Queries.CreateUser(r.Context(), database.CreateUserParams{Email: params.Email, HashedPassword: hashedPassword})
	if err != nil {
		w.WriteHeader(400)
		fmt.Printf("Error:%v", err)
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
		w.WriteHeader(500)
		fmt.Printf("Error:%v", err)
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
		w.WriteHeader(http.StatusBadRequest)
		fmt.Printf("Error:%v", err)
		return
	}
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Printf("Error:%v", err)
		return
	}
	validatedUser, err := auth.ValidateJWT(token, cfg.Secret)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Printf("Error:%v", err)
		return
	}
	filteredmsg, err := validationAndFiltering(params.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Printf("Error:%v", err)
		return
	}
	params.Body = filteredmsg
	msg, err := cfg.Queries.CreateChirp(r.Context(), database.CreateChirpParams{Body: params.Body, UserID: validatedUser})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
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
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Printf("Error:%v", err)
		return
	}
	w.WriteHeader(http.StatusCreated)
	w.Write(data)

}

type responseJSON struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserID    uuid.UUID `json:"user_id"`
	Token     string    `json:"token"`
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
		w.WriteHeader(500)
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
		w.WriteHeader(500)
		fmt.Printf("Error:%v", err)
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
		w.WriteHeader(500)
		fmt.Printf("Error:%v", err)
		return
	}
	w.WriteHeader(200)
	w.Write(data)
}

func (cfg *Config) Login(w http.ResponseWriter, r *http.Request) {
	type register struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	decoder := json.NewDecoder(r.Body)
	params := register{}
	err := decoder.Decode(&params)
	if err != nil {
		w.WriteHeader(400)
		fmt.Printf("Error:%v", err)
		return
	}
	user, err := cfg.Queries.GetUserByEmail(r.Context(), params.Email)
	if err != nil {
		w.WriteHeader(401)
		w.Write([]byte("Unauthorized"))
		fmt.Printf("Error:%v", err)
		return
	}
	check, err := auth.CheckPasswordHash(params.Password, user.HashedPassword)
	if err != nil {
		w.WriteHeader(500)
		fmt.Printf("Error:%v", err)
		return
	}
	if !check {
		w.WriteHeader(401)
		w.Write([]byte("Unauthorized"))
		return
	}
	refreshToken, err := auth.MakeRefreshToken()
	if err != nil {
		w.WriteHeader(500)
		fmt.Printf("Error:%v", err)
		return
	}
	sixtydays := time.Now().Add(60 * 24 * time.Hour)
	_, err = cfg.Queries.CreateRefreshToken(r.Context(), database.CreateRefreshTokenParams{Token: refreshToken, UserID: user.ID, ExpiresAt: sixtydays})
	if err != nil {
		w.WriteHeader(500)
		fmt.Printf("Error:%v", err)
		return
	}
	token, err := auth.MakeJWT(user.ID, cfg.Secret, time.Hour)
	if err != nil {
		w.WriteHeader(500)
		fmt.Printf("Error:%v", err)
		return
	}

	res := User{
		ID:           user.ID,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
		Email:        user.Email,
		Token:        token,
		RefreshToken: refreshToken,
	}
	data, err := json.Marshal(res)
	if err != nil {
		w.WriteHeader(500)
		fmt.Printf("Error:%v", err)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(200)
	w.Write(data)
}

func (cfg *Config) RefreshTokenToAccessToken(w http.ResponseWriter, r *http.Request) {
	token, err := getTokenFromHeader(r)
	if err != nil {
		w.WriteHeader(401)
		return
	}
	userID, err := cfg.Queries.GetUserFromRefreshToken(r.Context(), token)
	if err != nil {
		w.WriteHeader(401)
		return
	}
	accessToken, err := auth.MakeJWT(userID, cfg.Secret, time.Hour)
	if err != nil {
		w.WriteHeader(500)
		return
	}
	res := responseJSON{
		Token: accessToken,
	}

	data, err := json.Marshal(res)
	if err != nil {
		w.WriteHeader(500)
		fmt.Printf("Error:%v", err)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(200)
	w.Write(data)

}

func (cfg *Config) Revoke(w http.ResponseWriter, r *http.Request) {
	token, err := getTokenFromHeader(r)
	if err != nil {
		w.WriteHeader(401)
		return
	}
	err = cfg.Queries.RevokeRefreshToken(r.Context(), token)
	if err != nil {
		w.WriteHeader(500)
		fmt.Printf("Error:%v", err)
		return
	}
	w.WriteHeader(204)
}

func getTokenFromHeader(r *http.Request) (string, error) {
	token := r.Header.Get("Authorization")
	if token == "" {
		return "", fmt.Errorf("No token on header")
	}
	cutToken, foundBearer := strings.CutPrefix(token, "Bearer ")
	if !foundBearer {
		return "", fmt.Errorf("No prefix on header")
	}
	return cutToken, nil
}

func (cfg *Config) UpdateEmailAndPassword(w http.ResponseWriter, r *http.Request) {
	token, err := getTokenFromHeader(r)
	if err != nil {
		w.WriteHeader(401)
		return
	}
	validatedUserId, err := auth.ValidateJWT(token, cfg.Secret)
	if err != nil {
		w.WriteHeader(401)
		return
	}

	type toUpdate struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	decoder := json.NewDecoder(r.Body)
	params := toUpdate{}
	err = decoder.Decode(&params)
	if err != nil {
		w.WriteHeader(400)
		fmt.Printf("Error:%v", err)
		return
	}
	hashedPassword, err := auth.HashPassword(params.Password)
	if err != nil {
		w.WriteHeader(400)
		fmt.Printf("Error:%v", err)
		return
	}
	err = cfg.Queries.UpdateEmailAndPassword(r.Context(), database.UpdateEmailAndPasswordParams{Email: params.Email, HashedPassword: hashedPassword, ID: validatedUserId})
	if err != nil {
		w.WriteHeader(500)
		fmt.Printf("Error:%v", err)
		return
	}
	user, err := cfg.Queries.GetUserByEmail(r.Context(), params.Email)
	if err != nil {
		w.WriteHeader(400)
		fmt.Printf("Error:%v", err)
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
		w.WriteHeader(500)
		fmt.Printf("Error:%v", err)
		return
	}
	w.WriteHeader(200)
	w.Write(data)
}

func (cfg *Config) DeleteChirp(w http.ResponseWriter, r *http.Request) {
	token, err := getTokenFromHeader(r)
	if err != nil {
		w.WriteHeader(401)
		return
	}
	validatedUserId, err := auth.ValidateJWT(token, cfg.Secret)
	if err != nil {
		w.WriteHeader(403)
		return
	}
	id := r.PathValue("chirpID")
	parsedid, err := uuid.Parse(id)
	if err != nil {
		w.WriteHeader(404)
		fmt.Printf("Error:%v", err)
		return
	}
	chirp, err := cfg.Queries.GetChirpByID(r.Context(), parsedid) //maybe this is wrong and validation occurs con validatedjWT
	if err != nil {
		w.WriteHeader(404)
		fmt.Printf("Error:%v", err)
		return
	}
	if chirp.UserID != validatedUserId {
		w.WriteHeader(403)
		return
	}
	err = cfg.Queries.DeleteChirpByID(r.Context(), parsedid)
	if err != nil {
		w.WriteHeader(404)
		fmt.Printf("Error:%v", err)
		return
	}
	w.WriteHeader(204)

}
