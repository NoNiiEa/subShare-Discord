package billver

import (
	"strings"
	"unicode"
)

func extractNumericCharacters(s string) string {
	var result strings.Builder
	for _, r := range s {
		if unicode.IsDigit(r) {
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