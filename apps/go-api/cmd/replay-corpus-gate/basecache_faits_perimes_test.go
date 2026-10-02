package main

// basecache_faits_perimes_test.go — LA CUISSON DE LA BASE NE RELIT NI NE RANGE DES FAITS QU ELLE N A
// PAS ECRITS (revue finale P1-c, 2026-10-02).
//
// Avec `--work-root` explicite reutilise entre deux `--base`, la racine de travail de la base garde
// les faits persistes du film ecrits par le binaire de la base PRECEDENTE. A revisions constantes,
// `replay-build` les juge frais et republie depuis eux : l artefact de la base B sortait des faits
// de A, et il etait range sous la cle de B — verdict du banc faux, et persistant.
//
// MUTATIONS VUES ROUGES : ne plus vider les faits du temoin avant la cuisson de la base ; ranger un
// fichier de faits sans verifier qu il a ete ecrit par cette cuisson.

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/replaybuild"
)

func TestCuissonDeLaBase_PartSansLesFaitsDUnePrecedente(t *testing.T) {
	tc := temoinContexte{WorkRootBase: t.TempDir(), TitleSlug: title.DefaultSlug}
	facts := replaybuild.FactsFile{}
	facts.MatchID = "abcd1234-0000-0000-0000-000000000000"
	perimes := title.NewPathResolver(tc.WorkRootBase).FilmFactsPath(tc.TitleSlug, facts.MatchID)
	if err := os.MkdirAll(filepath.Dir(perimes), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(perimes, []byte("faits de la base precedente"), 0o600); err != nil {
		t.Fatal(err)
	}
	vus := false
	bake := func(_ context.Context, p cuissonParams, f replaybuild.FactsFile) (resultatCuisson, error) {
		vus = fichierExiste(title.NewPathResolver(p.WorkRoot).FilmFactsPath(p.TitleSlug, f.MatchID))
		return resultatCuisson{ArtifactPath: "a", FaitsPath: perimes}, nil
	}
	if _, _, err := tc.cuissonDeLaBase(context.Background(), facts, bake)(); err != nil {
		t.Fatal(err)
	}
	if vus {
		t.Fatalf("la cuisson de la base a demarre avec les faits d une cuisson precedente (%s) : "+
			"replay-build les republierait au lieu de decoder", perimes)
	}
}

func TestResoudreAvecCache_FaitsAnterieursALaCuissonNonRanges(t *testing.T) {
	capturerLogs(t)
	bc := baseCache{Racine: t.TempDir()}
	c := cleDeTest()
	vieux := faitsCuits(t)
	ilYA := time.Now().Add(-time.Hour)
	if err := os.Chtimes(vieux, ilYA, ilYA); err != nil {
		t.Fatal(err)
	}
	r, err := resoudreAvecCache(context.Background(), bc, c, false, func() (string, string, error) {
		return artefactCuit(t, artefactTest), vieux, nil // la cuisson n a ecrit aucun fait
	})
	if err != nil || r.ArtefactEnCache || r.FaitsEnCache {
		t.Fatalf("resolution %+v err=%v : des faits anterieurs a la cuisson ont ete ranges", r, err)
	}
	if _, ok := bc.chercher(context.Background(), c); ok {
		t.Fatal("une entree portant des faits que cette cuisson n a pas ecrits a ete rangee")
	}
}
