package service

import (
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

// TestBuildMatchHeader_DominanceBadge verifie le branchement Phase 1 méta-plan
// entre l'enrichissement DB (dominance_flag) et le badge narratif typé exposé
// dans le DTO header.
func TestBuildMatchHeader_DominanceBadge(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name           string
		flag           int
		wantBadge      bool
		wantLabelKey   string
		wantColorToken string
	}{
		{
			name:           "domination -> badge typé",
			flag:           int(canonical.DominanceDomination),
			wantBadge:      true,
			wantLabelKey:   "narrative.dominance.domination",
			wantColorToken: "narrative.dominance.win.strong",
		},
		{
			name:           "humiliation -> badge inversé",
			flag:           int(canonical.DominanceHumiliation),
			wantBadge:      true,
			wantLabelKey:   "narrative.dominance.humiliation",
			wantColorToken: "narrative.dominance.loss.strong",
		},
		{
			name:           "remontada -> badge comeback",
			flag:           int(canonical.DominanceRemontada),
			wantBadge:      true,
			wantLabelKey:   "narrative.dominance.remontada",
			wantColorToken: "narrative.dominance.win.comeback",
		},
		{
			name:      "none -> pas de badge",
			flag:      int(canonical.DominanceNone),
			wantBadge: false,
		},
		{
			name:      "flag inconnu -> pas de badge (dégradation gracieuse)",
			flag:      99,
			wantBadge: false,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			enrich := &domain.MatchEnrichmentRaw{DominanceFlag: tc.flag}
			h := buildMatchHeader("m1", &domain.MatchMetaRaw{}, nil, enrich, nil)

			if tc.wantBadge {
				if h.DominanceBadge == nil {
					t.Fatalf("DominanceBadge want non-nil for flag %d", tc.flag)
				}
				if h.DominanceBadge.LabelKey != tc.wantLabelKey {
					t.Errorf("LabelKey want %s, got %s", tc.wantLabelKey, h.DominanceBadge.LabelKey)
				}
				if h.DominanceBadge.ColorToken != tc.wantColorToken {
					t.Errorf("ColorToken want %s, got %s", tc.wantColorToken, h.DominanceBadge.ColorToken)
				}
				if h.DominanceBadge.Flag != tc.flag {
					t.Errorf("Flag want %d, got %d", tc.flag, h.DominanceBadge.Flag)
				}
				if !h.DominanceFlag {
					t.Error("DominanceFlag bool legacy want true (badge présent)")
				}
			} else {
				if h.DominanceBadge != nil {
					t.Errorf("DominanceBadge want nil for flag %d, got %+v", tc.flag, h.DominanceBadge)
				}
				if h.DominanceFlag {
					t.Error("DominanceFlag bool legacy want false (pas de badge)")
				}
			}
		})
	}
}

// TestBuildMatchHeader_NilEnrichment verifie qu'une enrichissement absent ne
// casse pas le header (cas legacy ou capability gap).
func TestBuildMatchHeader_NilEnrichment(t *testing.T) {
	t.Parallel()
	h := buildMatchHeader("m1", &domain.MatchMetaRaw{}, nil, nil, nil)
	if h.DominanceBadge != nil {
		t.Errorf("nil enrich: DominanceBadge want nil, got %+v", h.DominanceBadge)
	}
	if h.DominanceFlag {
		t.Error("nil enrich: DominanceFlag bool want false")
	}
}
