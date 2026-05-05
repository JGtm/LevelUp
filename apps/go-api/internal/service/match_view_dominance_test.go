package service

import (
	"context"
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
			h := buildMatchHeader(context.Background(), "m1", &domain.MatchMetaRaw{}, nil, enrich, nil, nil, false)

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
	h := buildMatchHeader(context.Background(), "m1", &domain.MatchMetaRaw{}, nil, nil, nil, nil, false)
	if h.DominanceBadge != nil {
		t.Errorf("nil enrich: DominanceBadge want nil, got %+v", h.DominanceBadge)
	}
	if h.DominanceFlag {
		t.Error("nil enrich: DominanceFlag bool want false")
	}
}

// TestOutcomeColorToken_MapsAllCodes verifie le mapping outcome -> token
// (Phase 1 méta-plan § 6.1.3 — chunk MV3 cleanup hex codes).
func TestOutcomeColorToken_MapsAllCodes(t *testing.T) {
	t.Parallel()
	cases := map[int]string{
		1: "outcome-draw",
		2: "outcome-win",
		3: "outcome-loss",
		4: "outcome-dnf",
		0: "",
		9: "",
	}
	for code, want := range cases {
		got := outcomeColorToken(code)
		if got != want {
			t.Errorf("outcomeColorToken(%d) want %q, got %q", code, want, got)
		}
	}
}

// TestPerfColorToken_MapsScoreToTier verifie le mapping score -> perf-tier-N.
func TestPerfColorToken_MapsScoreToTier(t *testing.T) {
	t.Parallel()
	cases := map[float64]string{
		95.0: "perf-tier-1", // excellent
		75.0: "perf-tier-2", // bon
		55.0: "perf-tier-3", // moyen
		25.0: "perf-tier-4", // faible
		10.0: "perf-tier-5", // très faible
		0.0:  "perf-tier-5",
	}
	for score, want := range cases {
		got := perfColorToken(score)
		if got != want {
			t.Errorf("perfColorToken(%g) want %q, got %q", score, want, got)
		}
	}
}

// TestBuildMatchHeader_OutcomeColorToken verifie que le token sémantique
// est exposé dans le header pour chaque code outcome.
func TestBuildMatchHeader_OutcomeColorToken(t *testing.T) {
	t.Parallel()
	stats := &domain.PlayerMatchStatsRaw{OutcomeCode: 2} // win
	h := buildMatchHeader(context.Background(), "m1", &domain.MatchMetaRaw{}, stats, nil, nil, nil, false)
	if h.OutcomeColorToken != "outcome-win" {
		t.Errorf("OutcomeColorToken want outcome-win, got %q", h.OutcomeColorToken)
	}
	// Le hex legacy reste exposé pour rétrocompat
	if h.OutcomeColor == "" {
		t.Error("OutcomeColor (hex legacy) want non-empty")
	}
}
