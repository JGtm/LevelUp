package main

// cmd_backfill_killsource_carte_test.go — `backfill-killsource` : UN MATCH SANS CARTE NE PREND PAS
// DE PLACE DE `--limit` ET N EST NI TELECHARGE NI DECODE (2026-09-27).
//
// MUTATIONS QUI DOIVENT LE FAIRE ROUGIR : garder la borne a la selection (`selectionSansBorne`
// rend `o` tel quel) ; ne plus filtrer dans `candidatsAvecCarte` ; borner AVANT le filtre.

import (
	"context"
	"testing"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/sync/killcollector"
	"levelup/go-api/internal/testutil"
)

// cartesDuTest : le resolveur de carte par match, sans base. Un match absent n a aucun nom.
type cartesDuTest map[string][]string

func (c cartesDuTest) MapKeysForMatch(_ context.Context, id string) (port.MatchMapKeys, error) {
	return port.MatchMapKeys{Names: c[id]}, nil
}

func (c cartesDuTest) MapKeysForMap(context.Context, string) (port.MatchMapKeys, error) {
	return port.MatchMapKeys{}, nil
}

func TestCandidatsAvecCarte_LaPlaceVaAuSuivant(t *testing.T) {
	racine, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine du depot : %v", err)
	}
	cartes := cartesDuTest{"b": {"Bazaar"}, "c": {"Forge Personnalisee"}, "d": {"Bazaar"}, "e": {"Bazaar"}}
	deps, err := killcollector.CaptureDepuisCatalogue(racine, title.DefaultSlug, cartes)
	if err != nil {
		t.Fatalf("capture : %v", err)
	}
	col := killcollector.NewKillSourceCollector(nil, nil, nil,
		games.CapabilityMap{games.CapFilmKillSource: games.CapSupported}, 0).AvecCapture(deps)
	candidats := []filmCandidat{{matchID: "a"}, {matchID: "b"}, {matchID: "c"}, {matchID: "d"}, {matchID: "e"}}

	got := candidatsAvecCarte(context.Background(), col, candidats, 2)
	if len(got) != 2 || got[0].matchID != "b" || got[1].matchID != "d" {
		t.Fatalf("candidats = %+v, attendu b puis d — `a` (sans nom) et `c` (hors catalogue) ont pris "+
			"une place de --limit", got)
	}
	if o := selectionSansBorne(killsourceOptions{limit: 2}); o.limit != 0 {
		t.Errorf("selection bornee a %d : la borne s applique AVANT le retrait des matchs sans carte", o.limit)
	}
	if o := selectionSansBorne(killsourceOptions{limit: 2, dryRun: true}); o.limit != 2 {
		t.Errorf("--dry-run : borne %d, attendu 2 (le plan ne resout pas les cartes)", o.limit)
	}
}
