package helper

import "testing"

func TestMaskedDigitsMatch(t *testing.T) {
	cases := []struct {
		name   string
		masked string
		stored string
		want   bool
	}{
		{"promptpay match", "086xxx7894", "0861237894", true},
		{"promptpay wrong suffix", "086xxx7894", "0861239999", false},
		{"promptpay wrong prefix", "086xxx7894", "0991237894", false},
		{"bank suffix match", "xxx-x-x3109-x", "1234563109x", true},
		{"bank suffix match plain", "xxxxx3109x", "9876543109", true},
		{"bank wrong middle", "xxx-x-x3109-x", "1234569999", false},
		{"doc example", "XXX-X-XX123-4", "9999991234", true},
		{"fully visible contains", "7894", "0861237894", true},
		{"fully visible not contained", "7894", "0861230000", false},
		{"empty stored", "086xxx7894", "", false},
		{"empty masked", "", "0861237894", false},
		{"all mask", "xxxx", "0861237894", false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := MaskedDigitsMatch(c.masked, c.stored); got != c.want {
				t.Errorf("MaskedDigitsMatch(%q, %q) = %v, want %v", c.masked, c.stored, got, c.want)
			}
		})
	}
}
