package api_test

import (
	"encoding/json"
	"regexp"
	"testing"

	"github.com/hsuanlee/watch-compare/backend/internal/model"
)

// The frontend spreads and maps every array in the API payload, and it deploys independently of
// the backend. A JSON null where an array is expected crashes the page, so the contract is:
// every array field is always present and never null, even when empty.
func TestArrayFieldsAreNeverNull(t *testing.T) {
	payloads := map[string]any{
		"search":  &model.SearchResult{Items: []model.Listing{}, Facets: model.NewFacets()},
		"facets":  model.NewFacets(),
		"listing": &model.Listing{ImageURLs: []string{}},
	}
	nullArray := regexp.MustCompile(`"(items|brands|sources|conditions|movements|countries|dialColors|years|imageUrls)":null`)
	for name, v := range payloads {
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		if m := nullArray.FindString(string(b)); m != "" {
			t.Errorf("%s payload contains a null array: %s\nfull: %s", name, m, b)
		}
	}
}
