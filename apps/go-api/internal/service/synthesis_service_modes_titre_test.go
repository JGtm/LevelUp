package service

import (
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
)

// h5ModeRow : match d'un titre dont le mode est le type de jeu (copie de GameVariant, Kind game_variant,
// libellés déjà traduits par l'enrichissement). frName == "" : type de jeu sans nom.
func h5ModeRow(variantID, enName, frName string, outcome canonical.Outcome) canonical.PlayerMatchRow {
	ref := func() *canonical.AssetReference {
		r := &canonical.AssetReference{Kind: "game_variant", ID: variantID, DefaultLabel: enName}
		if enName != "" || frName != "" {
			r.Labels = map[string]string{"en": enName, "fr": frName}
		}
		return r
	}
	var row canonical.PlayerMatchRow
	row.Self.Outcome = outcome
	row.Summary.Map = &canonical.AssetReference{DefaultLabel: "Truth"}
	row.Summary.GameVariant = ref()
	row.Summary.PairMode = ref()
	return row
}

// La Synthèse d'un titre sans partie (Halo 5) classe ses matchs par mode : noms lisibles, jamais un identifiant.
func TestBuildBreakdownsFromCanonical_ModeFourniParLeTypeDeJeu(t *testing.T) {
	rows := []canonical.PlayerMatchRow{
		h5ModeRow("257a305e-4dd3-41f1-9824-dfe7e8bd59e1", "Slayer", "Assassin", canonical.OutcomeWin),
		h5ModeRow("257a305e-4dd3-41f1-9824-dfe7e8bd59e1", "Slayer", "Assassin", canonical.OutcomeLoss),
		h5ModeRow("a2949322-dc84-45ab-8454-cf94fb28c189", "Capture the Flag", "Capture du drapeau", canonical.OutcomeWin),
		// Type de jeu que le dictionnaire ne nomme pas : aucune ligne, surtout pas son identifiant.
		h5ModeRow("1571fdac-e0b4-4ebc-a73a-6e13001b71d3", "", "", canonical.OutcomeWin),
	}
	b := buildBreakdownsFromCanonical(rows, domain.GroupesDeCartes{})

	if len(b.TopModes) != 2 {
		t.Fatalf("TopModes = %+v, want 2 lignes (le type de jeu sans nom est omis)", b.TopModes)
	}
	if e := modeEntryByName(t, b.TopModes, "Assassin"); e.MatchCount != 2 || e.Wins != 1 || e.Losses != 1 {
		t.Errorf("Assassin = %+v, want 2 matchs (1 victoire, 1 défaite)", e)
	}
	if e := modeEntryByName(t, b.TopModes, "Capture du drapeau"); e.MatchCount != 1 {
		t.Errorf("Capture du drapeau = %+v, want 1 match", e)
	}
	for _, e := range b.TopModes {
		if len(e.ModeName) == 36 && e.ModeName[8] == '-' {
			t.Errorf("identifiant brut affiché comme nom de mode : %q", e.ModeName)
		}
	}
}
