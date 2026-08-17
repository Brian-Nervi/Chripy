package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func HashPassword(password string) (string, error) {
	hash, err := argon2id.CreateHash(password, argon2id.DefaultParams)
	if err != nil {
		return "", err
	}
	return hash, nil
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

func CheckPasswordHash(password string, hash string) (bool, error) {
	check, err := argon2id.ComparePasswordAndHash(password, hash)
	if err != nil {
		return false, err
	}
	return check, nil

}

func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
	now := time.Now().UTC()
	claims := jwt.RegisteredClaims{
		Issuer:    "chirpy-access",
		Subject:   userID.String(),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(expiresIn)),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(tokenSecret))
	if err != nil {
		return "", err
	}

	return signedToken, nil
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	claims := jwt.RegisteredClaims{}
	keyFunc := func(t *jwt.Token) (interface{}, error) {
		return []byte(tokenSecret), nil
	}
	token, err := jwt.ParseWithClaims(tokenString, &claims, keyFunc)
	if err != nil {
		return uuid.Nil, err
	}
	user, err := token.Claims.GetSubject()
	if err != nil {
		return uuid.Nil, err
	}
	id, err := uuid.Parse(user)
	if err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

func GetBearerToken(headers http.Header) (string, error) { //gets users token
	token := headers.Get("Authorization")
	if token == "" {
		return "", fmt.Errorf("header sended an empty token")
	}
	cutToken, foundBearer := strings.CutPrefix(token, "Bearer ")
	if !foundBearer {
		return "", fmt.Errorf("Bearer  wasnt found")
	}
	return cutToken, nil
}

func MakeRefreshToken() (string, error) {
	randomnum := make([]byte, 32)
	_, err := rand.Read(randomnum)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(randomnum), nil
}

func GetAPIKey(headers http.Header) (string, error) {
	apikey := headers.Get("Authorization")
	if apikey == "" {
		fmt.Println(apikey)
		return "", fmt.Errorf("No apikey on header")
	}
	cutApiKey, foundBearer := strings.CutPrefix(apikey, "ApiKey ")
	if !foundBearer {
		fmt.Println(apikey)
		return "", fmt.Errorf("No prefix on header")
	}
	return cutApiKey, nil
}
