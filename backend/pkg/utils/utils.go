// Package utils — helpers genéricos.
package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"strings"
)

// RandomToken gera n bytes aleatórios e devolve em base64 url-safe sem padding.
func RandomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// SHA256Hex devolve o hash hex de uma string. Usado para fingerprintar refresh tokens.
func SHA256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// SafeFilename remove caracteres problemáticos de um nome de arquivo.
// Não substitui validação de path traversal — é só sanitização de exibição/storage.
func SafeFilename(name string) string {
	name = strings.TrimSpace(name)
	// Remove separadores de path + caracteres perigosos.
	bad := "/\\:*?\"<>|\x00"
	out := make([]rune, 0, len(name))
	for _, r := range name {
		if strings.ContainsRune(bad, r) || r < 0x20 {
			out = append(out, '_')
			continue
		}
		out = append(out, r)
	}
	res := strings.TrimSpace(string(out))
	if res == "" || res == "." || res == ".." {
		return "_"
	}
	if len(res) > 255 {
		res = res[:255]
	}
	return res
}
