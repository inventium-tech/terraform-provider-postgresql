package helpers

import (
	"crypto/rand"
	"math/big"
)

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// RandomString generates a random string of the specified length using characters from the charset.
// If the length is less than or equal to 0, an empty string is returned.
func RandomString(length int) string {
	if length <= 0 {
		return ""
	}

	result := make([]byte, length)
	charsetLength := big.NewInt(int64(len(charset)))

	for i := 0; i < length; i++ {
		randomIndex, err := rand.Int(rand.Reader, charsetLength)
		if err != nil {
			// In case of error, use a fallback character
			result[i] = 'x'
			continue
		}
		result[i] = charset[randomIndex.Int64()]
	}

	return string(result)
}
