package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestTokenValidationSadPath(t *testing.T) {
	_, err := ValidateJWT("sfdasfaedfadf", "sfdasfaedfadf")
	if err == nil {
		t.Errorf("FAIL at token validation on sad path")
	}
}

func TestTokenValidationHappyPath(t *testing.T) {
	userID := uuid.New()
	secret := "1234"
	token, err := MakeJWT(userID, secret, time.Hour)
	if err != nil {
		t.Fatalf("FAIL at token creation on happy path. err: %v", err)
	}
	validatedID, err := ValidateJWT(token, secret)
	if err != nil {
		t.Fatalf("FAIL at token validation on happy path. err: %v", err)
	}
	if validatedID != userID {
		t.Errorf("FAIL validated Token and initial token didnt match on happy path. Got:%v, used:%s", validatedID, userID)
	}
}

func TestExpiredTokenRejected(t *testing.T) {
	userID := uuid.New()
	secret := "1234"
	expiredToken := -10 * time.Second
	token, err := MakeJWT(userID, secret, expiredToken)
	if err != nil {
		t.Fatalf("FAIL at token creation at ExpiredTokenRejected. err: %v", err)
	}
	_, err = ValidateJWT(token, secret)
	if err == nil {
		t.Errorf("FAIL accepted expired token at ExpiredTokenRejected.")
	}
}

func TestWrongSecretRejected(t *testing.T) {
	userID := uuid.New()
	secret := "1234"
	incorrectSecret := "4321"
	token, err := MakeJWT(userID, secret, time.Hour)
	if err != nil {
		t.Fatalf("FAIL at token creation on WrongSecretRejected. err: %v", err)
	}
	_, err = ValidateJWT(token, incorrectSecret)
	if err == nil {
		t.Errorf("FAIL validated wrong secret at WrongSecretRejected.")
	}
}
