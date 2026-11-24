//go:build integration

package killcollector

// postsync_archive_coupee_integration_test.go — « archiver les films » coupé : la source des frags, de bout en bout
// (RunPostSync sur une base migrée), n'écrit RIEN à l'archive, ni film ni manifeste, quelle que soit l'issue du
// décodage. Cas du relecteur : un match de l'arriéré dont l'artefact de rejeu existe déjà (l'arriéré des frags ne
// regarde pas les artefacts) et dont le décodage échoue ; le disque ne garde aucun film. Témoin : réglage allumé,
// le même film est écrit.

import (
	"context"
	"io/fs"
	"path/filepath"
	"testing"
	"time"

	"levelup/go-api/internal/sync/filmfetch"
)

// filmsCompteSansDecodage : une source qui sert un film (octets inexploitables : le décodage échoue) pour tout match demandé.
type filmsCompteSansDecodage struct{ appels int }

func (f *filmsCompteSansDecodage) GetFilmChunks(_ context.Context, _ string) ([]filmfetch.FilmChunk, bool, error) {
	f.appels++
	return chunksTemoins(), true, nil
}

// fichiersSousRacine compte les fichiers (hors dossiers) sous racine.
func fichiersSousRacine(t *testing.T, racine string) []string {
	t.Helper()
	var out []string
	_ = filepath.WalkDir(racine, func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			out = append(out, p)
		}
		return nil
	})
	return out
}

func TestRunPostSync_ArchiveCoupee_RienNEstEcrit(t *testing.T) {
	for _, tc := range []struct {
		nom      string
		archiver bool
		ecrit    bool
	}{
		{"réglage coupé : aucun film, aucun manifeste", false, false},
		{"réglage allumé (témoin) : le film est écrit", true, true},
	} {
		t.Run(tc.nom, func(t *testing.T) {
			db := baseBacklog(t)
			inscrireMatch(t, db, "aabbccdd-1111-2222-3333-444455556666", time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC), 0)
			films := &filmsCompteSansDecodage{}
			h := NewPostSyncHook(racineDepot(t), 4).WithArchive(func(context.Context) bool { return tc.archiver })
			// Une archive à part : jamais le dossier de données du dépôt de travail.
			archive := t.TempDir()
			h.archiveRoot = func() string { return archive }
			deps := depsDeTest(db, &filmsTraces{}, nil)
			deps.Fetcher = films
			RunPostSync(context.Background(), h, deps, nil)
			if films.appels == 0 {
				t.Fatal("la source n'a pas été sollicitée : le test ne prouve rien")
			}
			fichiers := fichiersSousRacine(t, h.cacheRacine)
			if tc.ecrit && len(fichiers) == 0 {
				t.Errorf("réglage allumé : aucun film écrit sous %s", h.cacheRacine)
			}
			if !tc.ecrit && len(fichiers) != 0 {
				t.Errorf("réglage coupé : %d fichier(s) écrit(s) : %v", len(fichiers), fichiers)
			}
		})
	}
}
