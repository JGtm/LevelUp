package replaybuild

// filmfacts_sans_ecriture_test.go — `SansEcritureDesFaits` : la cuisson n'écrit RIEN sous la
// racine du dépôt (revue R1 du lot recos-d, P2-5 : la recuisson des rejeux de la démo écrivait
// data/cache/film_facts dans le vrai dépôt). Le témoin sans le réglage prouve que le test mord.

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
)

func artefactValidePourFaits(t *testing.T) []byte {
	t.Helper()
	raw, err := json.Marshal(replay.ReplayDocument{
		SchemaVersion: replay.SchemaVersion, MatchID: "m1", TitleSlug: title.DefaultSlug,
		Tracks: []replay.Track{{Slot: 1}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestSansEcritureDesFaits_RienSousLaRacine(t *testing.T) {
	for _, tc := range []struct {
		nom        string
		sansFaits  bool
		attenduEcr bool
	}{{"reglage_pose", true, false}, {"temoin_sans_reglage", false, true}} {
		t.Run(tc.nom, func(t *testing.T) {
			root := t.TempDir()
			b := &Builder{repoRoot: root, titleSlug: title.DefaultSlug}
			if tc.sansFaits {
				b = b.SansEcritureDesFaits()
			}
			b.rangerLesFaits(context.Background(), "m1", artefactValidePourFaits(t), &replay.FilmFactsFile{})
			_, err := os.Stat(title.NewPathResolver(root).FilmFactsPath(title.DefaultSlug, "m1"))
			if ecrit := err == nil; ecrit != tc.attenduEcr {
				t.Errorf("faits écrits = %v, attendu %v (err %v)", ecrit, tc.attenduEcr, err)
			}
		})
	}
}
