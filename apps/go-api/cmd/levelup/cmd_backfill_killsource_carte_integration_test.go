//go:build integration

package main

// cmd_backfill_killsource_carte_integration_test.go — LE `--dry-run` ANNONCE LA SELECTION DE LA
// PASSE QUI DECODE (cmd_backfill_killsource_carte.go), sur un shared migre par les vraies
// migrations.
//
// LE CAS QUI REPRODUIT L ECART : quatre films en cache, deux seulement dont la carte se resout, et le
// premier dans l ordre de la passe sans carte. Le plan annoncait les quatre (et, sous `--limit 1`, le
// film sans carte) ; la passe n en decode que deux (sous `--limit 1`, le premier qui a une carte).
//
// MUTATIONS QUI DOIVENT LE FAIRE ROUGIR : garder la borne a la selection sous `--dry-run`
// (`selectionSansBorne`) ; ne plus retirer les matchs sans carte dans `candidatsDeLaPasse` ou
// `idsDeLaPasseEnLigne`.

import (
	"context"
	"slices"
	"testing"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/sync/killcollector"
	"levelup/go-api/internal/testutil"
)

// collecteurDeSelection : un collecteur dont seules les cartes des matchs 3 et 1 de
// `registreDeTest` se resolvent (Bazaar, au catalogue de bornes).
func collecteurDeSelection(t *testing.T) *killcollector.KillSourceCollector {
	t.Helper()
	racine, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine du depot : %v", err)
	}
	cartes := cartesDuTest{registreDeTest[3]: {"Bazaar"}, registreDeTest[1]: {"Bazaar"}}
	deps, err := killcollector.CaptureDepuisCatalogue(context.Background(), racine, title.DefaultSlug, cartes)
	if err != nil {
		t.Fatalf("capture : %v", err)
	}
	return killcollector.NewKillSourceCollector(nil, nil, nil,
		games.CapabilityMap{games.CapFilmKillSource: games.CapSupported}, 0).AvecCapture(deps)
}

func TestCandidatsDeLaPasse_LePlanEstLaSelectionDeLaPasse(t *testing.T) {
	db, cache := registreAvecFilms(t, registreDeTest...)
	col := collecteurDeSelection(t)
	ctx := context.Background()
	idsDe := func(o killsourceOptions) []string {
		t.Helper()
		got, _, err := candidatsDeLaPasse(ctx, db, cache, o, col)
		if err != nil {
			t.Fatalf("candidatsDeLaPasse(%+v) : %v", o, err)
		}
		ids := make([]string, 0, len(got))
		for _, c := range got {
			ids = append(ids, c.matchID)
		}
		return ids
	}
	// Ordre de la passe (un chunk chacun, puis l identifiant) : 0, 2, 3, 1 — le premier sans carte.
	for limite, attendu := range map[int][]string{
		0: {registreDeTest[3], registreDeTest[1]},
		1: {registreDeTest[3]},
	} {
		passe := idsDe(killsourceOptions{limit: limite})
		plan := idsDe(killsourceOptions{limit: limite, dryRun: true})
		if !slices.Equal(plan, passe) {
			t.Errorf("--limit %d : le plan annonce %v, la passe decode %v", limite, plan, passe)
		}
		if !slices.Equal(passe, attendu) {
			t.Errorf("--limit %d : la passe decode %v, attendu %v (les seuls matchs dont la carte se "+
				"resout, la borne apres leur retrait)", limite, passe, attendu)
		}
	}
}

func TestIdsDeLaPasseEnLigne_LePlanEstLaSelectionDeLaPasse(t *testing.T) {
	db, _ := registreAvecFilms(t, registreDeTest...)
	col := collecteurDeSelection(t)
	ctx := context.Background()
	for _, limite := range []int{0, 1} {
		passe, err := idsDeLaPasseEnLigne(ctx, db, killsourceOptions{limit: limite}, col)
		if err != nil {
			t.Fatalf("passe : %v", err)
		}
		plan, err := idsDeLaPasseEnLigne(ctx, db, killsourceOptions{limit: limite, dryRun: true}, col)
		if err != nil {
			t.Fatalf("plan : %v", err)
		}
		if !slices.Equal(plan, passe) {
			t.Errorf("--online --limit %d : le plan annonce %v, la passe traite %v", limite, plan, passe)
		}
		for _, id := range passe {
			if id != registreDeTest[3] && id != registreDeTest[1] {
				t.Errorf("--online --limit %d : %s retenu sans carte resolue (%v)", limite, id, passe)
			}
		}
		if n := len(passe); n == 0 || (limite == 1 && n != 1) || (limite == 0 && n != 2) {
			t.Errorf("--online --limit %d : %d matchs retenus %v", limite, n, passe)
		}
	}
}
