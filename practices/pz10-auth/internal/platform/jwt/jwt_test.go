package jwtutil

import (
	"strings"
	"testing"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
)

func TestValidation(t *testing.T) {
	secret := []byte(strings.Repeat("a", 32))
	manager := New(secret, time.Minute, time.Hour)
	access, err := manager.Sign(2, "user@example.com", "user", Access)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Parse(access, Access); err != nil {
		t.Fatalf("valid access: %v", err)
	}
	if _, err := manager.Parse(access, Refresh); err == nil {
		t.Fatal("access accepted as refresh")
	}
	other := New([]byte(strings.Repeat("b", 32)), time.Minute, time.Hour)
	if _, err := other.Parse(access, Access); err == nil {
		t.Fatal("wrong signature accepted")
	}
	expired := New(secret, -time.Second, time.Hour)
	old, err := expired.Sign(2, "user@example.com", "user", Access)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Parse(old, Access); err == nil {
		t.Fatal("expired token accepted")
	}
	claims := Claims{Email: "user@example.com", Role: "user", Kind: Access, RegisteredClaims: jwt.RegisteredClaims{Subject: "2", Issuer: issuer, Audience: jwt.ClaimStrings{audience}, ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)), ID: "test"}}
	wrongAlg, err := jwt.NewWithClaims(jwt.SigningMethodHS384, claims).SignedString(secret)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Parse(wrongAlg, Access); err == nil {
		t.Fatal("HS384 accepted")
	}
}
