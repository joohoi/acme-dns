package main

import (
	"github.com/satori/go.uuid"
	"golang.org/x/crypto/bcrypt"
	"unicode/utf8"
)

func GetValidUsername(u string) (uuid.UUID, error) {
	uname, err := uuid.FromString(u)
	if err != nil {
		return uuid.UUID{}, err
	}
	return uname, nil
}

func ValidKey(k string) bool {
	kn := SanitizeString(k)
	if utf8.RuneCountInString(k) == 40 && utf8.RuneCountInString(kn) == 40 {
		// Correct length and all chars valid
		return true
	}
	return false
}

func ValidSubdomain(s string) bool {
	_, err := uuid.FromString(s)
	if err == nil {
		return true
	}
	return false
}

func ValidTXT(s string) bool {
	sn := SanitizeString(s)
	if utf8.RuneCountInString(s) == 43 && utf8.RuneCountInString(sn) == 43 {
		// 43 chars is the current LE auth key size, but not limited / defined by ACME
		return true
	}
	return false
}

func CorrectPassword(pw string, hash string) bool {
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)); err == nil {
		return true
	}
	return false
}
