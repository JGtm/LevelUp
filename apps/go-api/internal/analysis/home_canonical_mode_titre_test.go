package analysis

import (
	"testing"
	"time"

	"levelup/go-api/internal/games/canonical"
)

// Accueil d'un titre dont le mode est le type de jeu : le mode lu (modeText) et le mode dominant d'une session
// sont ceux que l'ancien repli « PairMode sinon GameVariant » rendait pour le même match (mêmes libellés).
func TestAccueil_ModeFourniParLeTypeDeJeu_IdentiqueALAncienRepli(t *testing.T) {
	sessionLabel := "session-h5"
	start := time.Date(2026, 5, 20, 14, 0, 0, 0, time.UTC)
	mkRow := func(id string, offset time.Duration, en, fr string) canonical.PlayerMatchRow {
		variant := func() *canonical.AssetReference {
			return &canonical.AssetReference{Kind: "game_variant", ID: "gv-" + en, DefaultLabel: en,
				Labels: map[string]string{"en": en, "fr": fr}}
		}
		return canonical.PlayerMatchRow{
			Summary: canonical.MatchSummary{
				MatchID: id, StartedAtUTC: start.Add(offset), Outcome: canonical.OutcomeWin,
				Map:         &canonical.AssetReference{ID: "truth", DefaultLabel: "Truth"},
				GameVariant: variant(),
				PairMode:    variant(),
			},
			Self:       canonical.MatchParticipant{Outcome: canonical.OutcomeWin},
			Enrichment: canonical.PlayerMatchEnrichment{SessionLabel: &sessionLabel},
		}
	}
	rows := []canonical.PlayerMatchRow{
		mkRow("m1", 0, "Slayer", "Assassin"),
		mkRow("m2", time.Minute, "Slayer", "Assassin"),
		mkRow("m3", 2*time.Minute, "Capture the Flag", "Capture du drapeau"),
	}

	for _, r := range rows {
		en, fr := enFr(modeText(r))
		wantEN, wantFR := enFr(assetText(r.Summary.GameVariant)) // ce que rendait l'ancien repli
		if en != wantEN || fr != wantFR {
			t.Errorf("%s : modeText = (%q, %q), l'ancien repli rendait (%q, %q)", r.Summary.MatchID, en, fr, wantEN, wantFR)
		}
	}

	got := BuildSessionSummariesFromCanonical(rows, false, 5, "fr", 225)
	if len(got) != 1 || got[0].DominantMode == nil {
		t.Fatalf("sessions = %+v, want 1 session avec un mode dominant", got)
	}
	if *got[0].DominantMode != "Assassin" {
		t.Errorf("DominantMode = %q, want \"Assassin\"", *got[0].DominantMode)
	}
}
