package acmedns

import (
	"crypto/rand"
	"math/big"
	"regexp"

	"golang.org/x/crypto/bcrypt"
)

func sanitizeIPv6addr(s string) string {
	// Remove brackets from IPv6 addresses, net.ParseCIDR needs this
	re, _ := regexp.Compile(`[\[\]]+`)
	return re.ReplaceAllString(s, "")
}

func SanitizeString(s string) string {
	// URL safe base64 alphabet without padding as defined in ACME
	re, _ := regexp.Compile(`[^A-Za-z\-\_0-9]+`)
	return re.ReplaceAllString(s, "")
}

func generatePassword(length int) string {
	ret := make([]byte, length)
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz1234567890-_"
	alphalen := big.NewInt(int64(len(alphabet)))
	for i := 0; i < length; i++ {
		c, _ := rand.Int(rand.Reader, alphalen)
		r := int(c.Int64())
		ret[i] = alphabet[r]
	}
	return string(ret)
}

func CorrectPassword(pw string, hash string) bool {
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)); err == nil {
		return true
	}
	return false
}
