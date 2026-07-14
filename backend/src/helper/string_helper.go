package helper

import (
	"strings"
	"unicode"
)

func ExtractNumericCharacters(s string) string {
	var result strings.Builder
	for _, r := range s {
		if unicode.IsDigit(r) {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// MaskedDigitsMatch reports whether a masked account/proxy value from a slip
// (e.g. PromptPay "086xxx7894" or bank "xxx-x-x3109-x") is consistent with a
// stored account number.
//
// SlipOK always masks the value with 'x'/'X', and the mask may sit in the
// middle, so a plain substring check fails for split masks like PromptPay. We
// keep only digits and 'x', then require the leading visible run to be a prefix
// of the stored digits, the trailing visible run to be a suffix, and any
// middle runs to appear somewhere in between.
func MaskedDigitsMatch(maskedValue, storedAccount string) bool {
	stored := ExtractNumericCharacters(storedAccount)
	if stored == "" {
		return false
	}

	// Normalize the masked value: keep digits and mask markers only.
	var m strings.Builder
	for _, r := range maskedValue {
		if unicode.IsDigit(r) {
			m.WriteRune(r)
		} else if r == 'x' || r == 'X' {
			m.WriteRune('x')
		}
	}
	masked := m.String()
	if masked == "" {
		return false
	}

	// Fully visible (no mask): the visible digits must appear in the account.
	if !strings.ContainsRune(masked, 'x') {
		return strings.Contains(stored, masked)
	}

	// Split into maximal digit runs, tracking leading/trailing anchoring.
	runs := strings.FieldsFunc(masked, func(r rune) bool { return r == 'x' })
	if len(runs) == 0 {
		return false // all mask, nothing to compare
	}

	prefix := ""
	if masked[0] != 'x' {
		prefix = runs[0]
		runs = runs[1:]
	}
	suffix := ""
	if masked[len(masked)-1] != 'x' && len(runs) > 0 {
		suffix = runs[len(runs)-1]
		runs = runs[:len(runs)-1]
	}

	if len(stored) < len(prefix)+len(suffix) {
		return false
	}
	if prefix != "" && !strings.HasPrefix(stored, prefix) {
		return false
	}
	if suffix != "" && !strings.HasSuffix(stored, suffix) {
		return false
	}
	for _, run := range runs { // remaining middle runs
		if !strings.Contains(stored, run) {
			return false
		}
	}
	return true
}