package billver

import (
	"strings"
	"unicode"
)

func extractNumericCharacters(s string) string {
	var result strings.Builder // Use strings.Builder for efficient string building
	for _, r := range s {
		if unicode.IsDigit(r) { // Check if the rune is a digit
			result.WriteRune(r)
		}
	}
	return result.String()
}

func last4(s string) string {
	if len(s) <= 4 {
		return s
	}
	return s[len(s)-4:]
}