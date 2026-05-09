// Package jwt — emissão e verificação de access tokens HS256.
//
// Refresh tokens NÃO são JWT: são bytes opacos guardados (hash) no banco,
// para permitir revogação imediata e rotação. Aqui só lidamos com access tokens.
package jwt

import (
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const KeyFileName = "jwt.key"

// LoadOrCreateKey carrega chave HS256 de 32 bytes do arquivo, ou cria uma
// nova se não existir. Permissões 0600.
func LoadOrCreateKey(secretsDir string) ([]byte, error) {
	if err := os.MkdirAll(secretsDir, 0o700); err != nil {
		return nil, err
	}
	path := filepath.Join(secretsDir, KeyFileName)

	data, err := os.ReadFile(path)
	if err == nil {
		if len(data) < 32 {
			return nil, fmt.Errorf("jwt.key tem %d bytes, esperado >=32", len(data))
		}
		return data, nil
	}
	if !os.IsNotExist(err) {
		return nil, err
	}

	// Não existe: cria.
	key := make([]byte, 64)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, key, 0o600); err != nil {
		return nil, err
	}
	return key, nil
}

// Claims são as claims customizadas do PiNAS.
type Claims struct {
	UserID   int64  `json:"uid"`
	Username string `json:"un"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// Issuer emite e verifica access tokens.
type Issuer struct {
	key []byte
	ttl time.Duration
}

func NewIssuer(key []byte, ttl time.Duration) *Issuer {
	return &Issuer{key: key, ttl: ttl}
}

func (i *Issuer) Issue(userID int64, username, role string) (string, time.Time, error) {
	now := time.Now()
	exp := now.Add(i.ttl)
	claims := &Claims{
		UserID:   userID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "pinas",
			Subject:   username,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString(i.key)
	if err != nil {
		return "", time.Time{}, err
	}
	return signed, exp, nil
}

func (i *Issuer) Parse(tokenString string) (*Claims, error) {
	tok, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		// Trava o algoritmo: defesa contra "alg=none" e key confusion.
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("alg inesperado: %v", t.Header["alg"])
		}
		return i.key, nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil {
		return nil, err
	}
	claims, ok := tok.Claims.(*Claims)
	if !ok || !tok.Valid {
		return nil, errors.New("token inválido")
	}
	return claims, nil
}
