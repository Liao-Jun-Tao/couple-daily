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

const (
	argonMemory  uint32 = 64 * 1024
	argonTime    uint32 = 1
	argonThreads uint8  = 1
	argonKeyLen  uint32 = 32
)

// HashPassword returns a PHC-style argon2id hash.
func HashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	sum := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		argonMemory, argonTime, argonThreads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(sum),
	), nil
}

// VerifyPassword reports whether password matches an argon2id hash.
func VerifyPassword(encoded, password string) bool {
	mem, tim, threads, salt, want, ok := parseArgon(encoded)
	if !ok {
		return false
	}
	got := argon2.IDKey([]byte(password), salt, tim, mem, threads, uint32(len(want)))
	return subtle.ConstantTimeCompare(got, want) == 1
}

func parseArgon(encoded string) (mem, tim uint32, threads uint8, salt, sum []byte, ok bool) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" || parts[2] != "v=19" {
		return 0, 0, 0, nil, nil, false
	}
	fields := strings.Split(parts[3], ",")
	if len(fields) != 3 {
		return 0, 0, 0, nil, nil, false
	}
	mv, okM := strings.CutPrefix(fields[0], "m=")
	tv, okT := strings.CutPrefix(fields[1], "t=")
	pv, okP := strings.CutPrefix(fields[2], "p=")
	if !okM || !okT || !okP {
		return 0, 0, 0, nil, nil, false
	}
	m64, errM := strconv.ParseUint(mv, 10, 32)
	t64, errT := strconv.ParseUint(tv, 10, 32)
	p64, errP := strconv.ParseUint(pv, 10, 8)
	if errM != nil || errT != nil || errP != nil {
		return 0, 0, 0, nil, nil, false
	}
	if m64 == 0 || m64 > 256*1024 || t64 == 0 || t64 > 5 || p64 == 0 || p64 > 4 {
		return 0, 0, 0, nil, nil, false
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil || len(salt) < 8 {
		return 0, 0, 0, nil, nil, false
	}
	sum, err = base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil || len(sum) < 16 {
		return 0, 0, 0, nil, nil, false
	}
	return uint32(m64), uint32(t64), uint8(p64), salt, sum, true
}

// dummyHash keeps missing-user logins on the argon2 path.
var dummyHash string

func init() {
	h, err := HashPassword("dummy-password-not-used")
	if err != nil {
		panic(err)
	}
	dummyHash = h
}

// BurnMissingUser spends a password check when the email is unknown.
func BurnMissingUser(password string) {
	_ = VerifyPassword(dummyHash, password)
}
