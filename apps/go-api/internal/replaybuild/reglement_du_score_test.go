package replaybuild

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/replay"
)

// TestReglementDuScorePoseLaGardeDeMode — la barre de garde ne s'ouvre que sur un mode a colline,
// et la table n'y entre que comme entree de repli. Bases et Total Control restent fermes, meme
// s'ils portent le meme compteur `comp 23 A`.
func TestReglementDuScorePoseLaGardeDeMode(t *testing.T) {
	b, _ := zonesTestBuilder(t)
	cas := []struct {
		variant string
		colline bool
		table   int
	}{
		{"Ranked:King of the Hill", true, 40},
		{"Doubles:King of the Hill", true, 35},
		{"Squad:King of the Hill", true, 0}, // mode a colline sans entree : le film seul donne le seuil
		{"Strongholds:Arena", false, 0},
		{"BTB:Total Control", false, 0},
	}
	for _, c := range cas {
		in := &replay.ScoreInput{}
		b.poserLeReglementDuScore(in, c.variant)
		if in.HillScoring != c.colline || in.HoldTicksPerPointTable != c.table {
			t.Errorf("%s : colline=%v table=%d, attendu colline=%v table=%d",
				c.variant, in.HillScoring, in.HoldTicksPerPointTable, c.colline, c.table)
		}
	}
	b.poserLeReglementDuScore(nil, "Ranked:King of the Hill") // aucune entree : rien a poser, pas de panique
}

// TestReglementDuScoreUnSeulSite — la pose du reglement du score ne se recopie pas : la cuisson et
// les outils de recherche passent par `poserLeReglementDuScore`.
func TestReglementDuScoreUnSeulSite(t *testing.T) {
	fichiers, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range fichiers {
		if f == "reglement_du_score.go" || f == "reglement_du_score_test.go" {
			continue
		}
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		for _, motif := range []string{"regulation.ScoreTarget(", "regulation.HoldTicksPerPoint("} {
			if strings.Contains(string(src), motif) {
				t.Errorf("%s recopie %q : passer par poserLeReglementDuScore", f, motif)
			}
		}
	}
}
