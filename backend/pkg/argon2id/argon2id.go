// Package argon2id — hashing seguro de senhas com Argon2id (RFC 9106).
//
// Parâmetros calibrados para Raspberry Pi 4 (~250–400 ms por hash):
//
//	Time:    3
//	Memory:  64 MiB
//	Threads: 2
//	KeyLen:  32 bytes
//	SaltLen: 16 bytes
//
// Formato de saída no padrão PHC, compatível com qualquer impl. Argon2id:
//
//	$argon2id$v=19$m=65536,t=3,p=2$<salt b64>$<key b64>
package argon2id

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

type Params struct {
	Time    uint32
	Memory  uint32 // KiB
	Threads uint8
	KeyLen  uint32
	SaltLen uint32
}

// Default são os parâmetros calibrados para Raspberry Pi 4.
// Aumente se o hardware for mais potente.
var Default = Params{
	Time:    3,
	Memory:  64 * 1024,
	Threads: 2,
	KeyLen:  32,
	SaltLen: 16,
}

var (
	ErrInvalidHash         = errors.New("argon2id: hash inválido")
	ErrIncompatibleVersion = errors.New("argon2id: versão incompatível")
)

// Hash gera o hash codificado em formato PHC.
func Hash(password string) (string, error) {
	return HashWith(password, Default)
}

func HashWith(password string, p Params) (string, error) {
	salt := make([]byte, p.SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(password), salt, p.Time, p.Memory, p.Threads, p.KeyLen)

	b64salt := base64.RawStdEncoding.EncodeToString(salt)
	b64key := base64.RawStdEncoding.EncodeToString(key)
	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, p.Memory, p.Time, p.Threads, b64salt, b64key,
	), nil
}

// Verify checa em tempo constante se a senha bate com o hash.
func Verify(password, encoded string) (bool, error) {
	p, salt, key, err := decode(encoded)
	if err != nil {
		return false, err
	}
	other := argon2.IDKey([]byte(password), salt, p.Time, p.Memory, p.Threads, p.KeyLen)
	if subtle.ConstantTimeCompare(key, other) == 1 {
		return true, nil
	}
	return false, nil
}

func decode(encoded string) (Params, []byte, []byte, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return Params{}, nil, nil, ErrInvalidHash
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return Params{}, nil, nil, ErrInvalidHash
	}
	if version != argon2.Version {
		return Params{}, nil, nil, ErrIncompatibleVersion
	}

	var p Params
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &p.Memory, &p.Time, &p.Threads); err != nil {
		return Params{}, nil, nil, ErrInvalidHash
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return Params{}, nil, nil, ErrInvalidHash
	}
	p.SaltLen = uint32(len(salt))

	key, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return Params{}, nil, nil, ErrInvalidHash
	}
	p.KeyLen = uint32(len(key))

	return p, salt, key, nil
}
