package jwtutil

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strconv"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
)

const issuer = "pz10-auth"
const audience = "pz10-clients"

type Kind string

const (
	Access  Kind = "access"
	Refresh Kind = "refresh"
)

type Claims struct {
	Email string `json:"email"`
	Role  string `json:"role"`
	Kind  Kind   `json:"token_type"`
	jwt.RegisteredClaims
}
type Manager struct {
	secret                []byte
	AccessTTL, RefreshTTL time.Duration
}

func New(secret []byte, access, refresh time.Duration) *Manager {
	return &Manager{secret: append([]byte(nil), secret...), AccessTTL: access, RefreshTTL: refresh}
}
func (m *Manager) Sign(id int64, email, role string, kind Kind) (string, error) {
	ttl := m.AccessTTL
	if kind == Refresh {
		ttl = m.RefreshTTL
	} else if kind != Access {
		return "", errors.New("invalid token kind")
	}
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	now := time.Now().UTC()
	claims := Claims{Email: email, Role: role, Kind: kind, RegisteredClaims: jwt.RegisteredClaims{
		Subject: strconv.FormatInt(id, 10), Issuer: issuer, Audience: jwt.ClaimStrings{audience},
		IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(ttl)), ID: hex.EncodeToString(bytes),
	}}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
}
func (m *Manager) Parse(raw string, kind Kind) (*Claims, error) {
	claims := new(Claims)
	token, err := jwt.NewParser(jwt.WithValidMethods([]string{"HS256"}), jwt.WithIssuer(issuer), jwt.WithAudience(audience), jwt.WithExpirationRequired(), jwt.WithIssuedAt()).ParseWithClaims(raw, claims, func(t *jwt.Token) (any, error) { return m.secret, nil })
	if err != nil || token == nil || !token.Valid {
		return nil, errors.New("invalid token")
	}
	if claims.Kind != kind || claims.Subject == "" || claims.Email == "" || claims.Role == "" || claims.ID == "" {
		return nil, errors.New("invalid claims")
	}
	if _, err := strconv.ParseInt(claims.Subject, 10, 64); err != nil {
		return nil, errors.New("invalid subject")
	}
	return claims, nil
}
