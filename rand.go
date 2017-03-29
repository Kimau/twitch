package twitch

import (
	"crypto/rand"
	"encoding/base64"
	"log"
)

// GenerateRandomBytes gets random byte stream NOT CRYPTO QUALITY
func GenerateRandomBytes(n uint) []byte {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		// Should never really happen
		log.Fatalln(err)
	}

	return b
}

// GenerateRandomString returns a URL-safe, base64 encoded NOT CRYPTO QUALITY
func GenerateRandomString(s uint) string {
	return base64.URLEncoding.EncodeToString(GenerateRandomBytes(s))
}
