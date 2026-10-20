package domain

import (
	"strings"
	"testing"
)

func TestTrendsQueryRequest_ValidateDoublons(t *testing.T) {
	cases := []struct {
		name string
		req  TrendsQueryRequest
		dup  bool
	}{
		{"squad identiques", TrendsQueryRequest{View: "squad", SelectedGamertags: []string{"Alice", "Alice"}}, true},
		{"squad casse et espaces", TrendsQueryRequest{View: "squad", SelectedGamertags: []string{"Alice", " alice "}}, true},
		{"squad distincts", TrendsQueryRequest{View: "squad", SelectedGamertags: []string{"Alice", "Bob"}}, false},
		{"solo identiques", TrendsQueryRequest{SelectedGamertags: []string{"Alice", "ALICE"}}, true},
		{"solo distincts", TrendsQueryRequest{SelectedGamertags: []string{"Alice", "Bob"}}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.req.Validate()
			if !c.dup {
				if err != nil {
					t.Fatalf("erreur inattendue : %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), "en double") {
				t.Fatalf("erreur = %v, attendu un doublon refusé", err)
			}
		})
	}
}
