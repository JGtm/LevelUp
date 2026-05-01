// filters_match_context_test.go — tests Phase C plan catalogue : champ MatchContext.
package domain

import "testing"

func TestIsValidMatchContext(t *testing.T) {
	t.Parallel()
	tests := []struct {
		val  string
		want bool
	}{
		{"", true}, // vide accepté = "all" implicite
		{"solo", true},
		{"squad", true},
		{"all", true},
		{"SOLO", false}, // case-sensitive
		{"Solo", false},
		{"squd", false},
		{"any", false},
		{"  solo  ", false}, // pas de trim
	}
	for _, tc := range tests {
		t.Run(tc.val, func(t *testing.T) {
			if got := IsValidMatchContext(tc.val); got != tc.want {
				t.Errorf("IsValidMatchContext(%q) = %v, want %v", tc.val, got, tc.want)
			}
		})
	}
}

func TestFilterContextInput_Validate_MatchContext(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		ctx     string
		wantErr bool
	}{
		{"vide accepté", "", false},
		{"solo accepté", "solo", false},
		{"squad accepté", "squad", false},
		{"all accepté", "all", false},
		{"valeur inconnue rejetée", "duo", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			input := FilterContextInput{MatchContext: tc.ctx}
			err := input.Validate()
			if tc.wantErr && err == nil {
				t.Errorf("Validate(%q) attendait une erreur", tc.ctx)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("Validate(%q) erreur inattendue : %v", tc.ctx, err)
			}
		})
	}
}
