package repository

import "testing"

func TestPrefixTSQuery(t *testing.T) {
	cases := map[string]string{
		// references keep their punctuation and gain a prefix marker
		"116500":              `'116500':*`,
		"116500LN":            `'116500LN':*`, // case folding is Postgres' job, not ours
		"5711/1A-010":         `'5711/1A-010':*`,
		"311.30.42.30.01.005": `'311.30.42.30.01.005':*`,
		// multiple terms are AND-ed, matching websearch_to_tsquery's default
		"rolex 116500": `'rolex':* & '116500':*`,
		// tsquery operators typed by a user become literal terms, never syntax
		"a & !b":  `'a' & '&' & '!b'`,
		"foo|bar": `'foo|bar':*`,
		// the two characters that are special inside a quoted lexeme
		`\`:    `'\\'`,
		"it's": `'it''s':*`,
		// terms shorter than three characters match exactly, never by prefix
		"a":  `'a'`,
		"ap": `'ap'`,
		// nothing usable
		"":    "",
		"   ": "",
	}
	for in, want := range cases {
		if got := prefixTSQuery(in); got != want {
			t.Errorf("prefixTSQuery(%q) = %q, want %q", in, got, want)
		}
	}
}
