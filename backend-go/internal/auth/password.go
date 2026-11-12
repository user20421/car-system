package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

type Argon2id struct {
	Time, Memory uint32
	Threads      uint8
	KeyLength    uint32
}

func DefaultArgon2id() Argon2id {
	return Argon2id{Time: 3, Memory: 64 * 1024, Threads: 2, KeyLength: 32}
}

func (a Argon2id) Hash(password string) (string, error) {
	if len(password) < 8 || len(password) > 256 {
		return "", fmt.Errorf("password length must be 8..256")
	}
	if a.Time == 0 || a.Memory < 8192 || a.Threads == 0 || a.KeyLength < 16 {
		return "", fmt.Errorf("unsafe Argon2id parameters")
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(password), salt, a.Time, a.Memory, a.Threads, a.KeyLength)
	enc := base64.RawStdEncoding
	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s", a.Memory, a.Time, a.Threads, enc.EncodeToString(salt), enc.EncodeToString(key)), nil
}

func Verify(password, encoded string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" || parts[2] != "v=19" {
		return false
	}
	var memory, time uint32
	var threads uint8
	params := strings.Split(parts[3], ",")
	if len(params) != 3 {
		return false
	}
	for _, param := range params {
		kv := strings.SplitN(param, "=", 2)
		if len(kv) != 2 {
			return false
		}
		n, err := strconv.ParseUint(kv[1], 10, 32)
		if err != nil {
			return false
		}
		switch kv[0] {
		case "m":
			memory = uint32(n)
		case "t":
			time = uint32(n)
		case "p":
			if n > 255 {
				return false
			}
			threads = uint8(n)
		default:
			return false
		}
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil || len(salt) < 8 {
		return false
	}
	expected, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil || len(expected) < 16 {
		return false
	}
	if time == 0 || memory < 8192 || threads == 0 {
		return false
	}
	actual := argon2.IDKey([]byte(password), salt, time, memory, threads, uint32(len(expected)))
	return subtle.ConstantTimeCompare(actual, expected) == 1
}
