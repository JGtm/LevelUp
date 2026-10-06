package domain

import (
	"strings"
	"testing"
)

func TestTrendsQueryRequest_Validate(t *testing.T) {
	cases := []struct {
		name    string
		req     TrendsQueryRequest
		wantErr string // vide = accepté
	}{
		{"vide", TrendsQueryRequest{}, ""},
		{"solo", TrendsQueryRequest{View: "solo"}, ""},
		{"squad sans gamertag", TrendsQueryRequest{View: "squad"}, "au moins un gamertag"},
		{"squad 1 gamertag", TrendsQueryRequest{View: "squad", SelectedGamertags: []string{"a"}}, ""},
		{"squad 3 gamertags", TrendsQueryRequest{View: "squad", SelectedGamertags: []string{"a", "b", "c"}}, ""},
		{"squad 4 gamertags", TrendsQueryRequest{View: "squad", SelectedGamertags: []string{"a", "b", "c", "d"}}, "gamertags"},
		{"squad gamertag vide", TrendsQueryRequest{View: "squad", SelectedGamertags: []string{"a", " "}}, "vide"},
		{"vue inconnue", TrendsQueryRequest{View: "duo"}, "vue inconnue"},
		{"solo 3 gamertags ignorés", TrendsQueryRequest{SelectedGamertags: []string{"a", "b", "c"}}, ""},
		{"solo 4 gamertags", TrendsQueryRequest{SelectedGamertags: []string{"a", "b", "c", "d"}}, "gamertags"},
		{"type valide", TrendsQueryRequest{GameType: "ranked_slayer_2"}, ""},
		{"type 64 caractères", TrendsQueryRequest{GameType: strings.Repeat("a", 64)}, ""},
		{"type 65 caractères", TrendsQueryRequest{GameType: strings.Repeat("a", 65)}, "trop long"},
		{"type majuscule", TrendsQueryRequest{GameType: "Ranked"}, "minuscules"},
		{"type avec tiret", TrendsQueryRequest{GameType: "ranked-slayer"}, "minuscules"},
		{"type avec espace", TrendsQueryRequest{GameType: "a b"}, "minuscules"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.req.Validate()
			if c.wantErr == "" {
				if err != nil {
					t.Fatalf("erreur inattendue : %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), c.wantErr) {
				t.Fatalf("err = %v, attendu un message contenant %q", err, c.wantErr)
			}
		})
	}
}
