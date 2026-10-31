// Package auth is the email + password scaffold.
// Phone OTP is not part of v1: there is no phone column and no SMS sender.
package auth

import (
	"errors"
	"strings"
	"unicode/utf8"
)

const (
	MinPasswordLength = 8
	MaxPasswordLength = 128
	MaxDisplayNameLen = 64
	// PasswordHashAlgo is the algorithm handlers will use when persistence lands.
	PasswordHashAlgo = "argon2id"
)

var (
	ErrInvalidEmail       = errors.New("invalid email")
	ErrInvalidPassword    = errors.New("password must be 8 to 128 characters")
	ErrInvalidDisplayName = errors.New("display_name must be 1 to 64 characters")
)

// ValidateRegister checks a register request and returns the canonical email
// plus the trimmed display name.
func ValidateRegister(email, password, displayName string) (string, string, error) {
	canonical, err := NormalizeEmail(email)
	if err != nil {
		return "", "", err
	}
	if n := utf8.RuneCountInString(password); n < MinPasswordLength || n > MaxPasswordLength {
		return "", "", ErrInvalidPassword
	}
	name := strings.TrimSpace(displayName)
	if n := utf8.RuneCountInString(name); n < 1 || n > MaxDisplayNameLen {
		return "", "", ErrInvalidDisplayName
	}
	return canonical, name, nil
}

// NormalizeEmail applies a small canonical form. It is not a full RFC parser.
func NormalizeEmail(email string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	at := strings.LastIndex(email, "@")
	if at <= 0 || at == len(email)-1 || strings.Contains(email[:at], " ") {
		return "", ErrInvalidEmail
	}
	domain := email[at+1:]
	if !strings.Contains(domain, ".") || strings.HasPrefix(domain, ".") || strings.HasSuffix(domain, ".") {
		return "", ErrInvalidEmail
	}
	return email, nil
}
