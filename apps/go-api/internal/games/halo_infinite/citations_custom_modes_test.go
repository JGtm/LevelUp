package halo_infinite

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/highlightevent"
)

func TestComputeWinsAssault(t *testing.T) {
	cas := []struct {
		nom     string
		ctx     domain.CitationContext
		attendu int
	}{
		{"victoire bombe neutre", domain.CitationContext{Outcome: domain.OutcomeWin, GameVariant: "assault:neutral bomb"}, 1},
		{"victoire une bombe", domain.CitationContext{Outcome: domain.OutcomeWin, GameVariant: "assault:one bomb"}, 1},
		{"defaite", domain.CitationContext{Outcome: domain.OutcomeLoss, GameVariant: "assault:neutral bomb"}, 0},
		// « Assault » est aussi un conteneur de playlist : seule la variante decide.
		{"playlist assault, variante slayer", domain.CitationContext{Outcome: domain.OutcomeWin, Playlist: "assault", GameVariant: "slayer"}, 0},
	}
	for _, c := range cas {
		if got := computeWinsAssault(c.ctx); got != c.attendu {
			t.Errorf("%s : attendu %d, obtenu %d", c.nom, c.attendu, got)
		}
	}
}

func TestComputeFirstStrike(t *testing.T) {
	ev := func(typ string, ms int64, xuid string) domain.CitationEventRow {
		return domain.CitationEventRow{EventType: typ, TimeMS: ms, XUID: xuid}
	}
	cas := []struct {
		nom     string
		events  []domain.CitationEventRow
		attendu int
	}{
		{"premier frag du joueur", []domain.CitationEventRow{
			ev(highlightevent.EventTypeDeath, 900, "b"), ev(highlightevent.EventTypeKill, 1000, "moi"), ev(highlightevent.EventTypeKill, 2000, "b"),
		}, 1},
		{"premier frag d un autre", []domain.CitationEventRow{
			ev(highlightevent.EventTypeKill, 1000, "b"), ev(highlightevent.EventTypeKill, 2000, "moi"),
		}, 0},
		{"premier frag d un bot", []domain.CitationEventRow{
			ev(highlightevent.EventTypeKill, 1000, ""), ev(highlightevent.EventTypeKill, 2000, "moi"),
		}, 0},
		{"aucun frag", []domain.CitationEventRow{ev(highlightevent.EventTypeMedal, 1000, "moi")}, 0},
	}
	for _, c := range cas {
		ctx := domain.CitationContext{PlayerXUID: "moi", Events: c.events}
		if got := computeFirstStrike(ctx); got != c.attendu {
			t.Errorf("%s : attendu %d, obtenu %d", c.nom, c.attendu, got)
		}
	}
}

func TestDispatchCustom_NouvellesCitationsDeMode(t *testing.T) {
	ctx := domain.CitationContext{Outcome: domain.OutcomeWin, GameVariant: "assault:one bomb", PlayerXUID: "moi",
		Events: []domain.CitationEventRow{{EventType: highlightevent.EventTypeKill, TimeMS: 1, XUID: "moi"}}}
	for _, fn := range []string{"compute_wins_assault", "compute_first_strike"} {
		if got := DispatchCustom(fn, ctx); got != 1 {
			t.Errorf("%s non route : obtenu %d", fn, got)
		}
	}
}

// TestCitationsCustom_ModesParPredicat — garde-rail : la playlist et la variante ne se lisent que
// dans citations_custom_modes.go (un prédicat par mode). Quatre victoires de mode recopiaient
// le même test de sous-chaîne ; une nouvelle citation de mode passe par un prédicat.
func TestCitationsCustom_ModesParPredicat(t *testing.T) {
	fichiers, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range fichiers {
		if f == "citations_custom_modes.go" || strings.HasSuffix(f, "_test.go") {
			continue
		}
		blob, err := os.ReadFile(f) //nolint:gosec // fichiers du paquet
		if err != nil {
			t.Fatal(err)
		}
		for _, interdit := range []string{"ctx.Playlist", "ctx.GameVariant"} {
			if strings.Contains(string(blob), interdit) {
				t.Errorf("%s lit %s : passer par un prédicat de citations_custom_modes.go", f, interdit)
			}
		}
	}
}
