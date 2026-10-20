package replaybuild

// kills_carte_test.go — LA SOURCE DE DEGAT SE DECODE SOUS LA CARTE QUE LA CUISSON A RESOLUE, ET
// SOUS AUCUNE AUTRE (2026-09-27, « le flux du film est la seule source fiable. Pas de repli. »).
//
// `decodeKillSource` recevait les NOMS de carte et les re-resolvait ; un echec y faisait decoder la
// source de degat aux largeurs PAR DEFAUT (celles de Cliffhanger). Il recoit desormais l ENTREE
// deja resolue par `BuildBytes`, qui refuse en amont une carte hors catalogue.
//
// MUTATION QUI DOIT LE FAIRE ROUGIR : ne plus poser `opts.Carte` (le decodage est alors refuse par
// `killsource`, le resultat est nil) ; poser une autre carte que l entree recue (la calibration ne
// porte plus ses largeurs).

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/film/decfilm"
	"levelup/go-api/internal/testutil"
)

// bobineKillsource000d5950 : la mini-bobine versionnee du paquet `killsource` (film 000d5950, Cliffhanger).
func bobineKillsource000d5950(t *testing.T) *decfilm.Film {
	t.Helper()
	racine, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine du depot : %v", err)
	}
	dir := filepath.Join(racine, "apps", "go-api", "internal", "games", "halo_infinite", "film",
		"internal", "facts", "killsource", "testdata", "minibobine_000d5950")
	film, err := decfilm.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("mini-bobine illisible sous %s : %v", dir, err)
	}
	return film
}

func entreeDuCatalogue(t *testing.T, nom string) decfilm.MapQuantEntry {
	t.Helper()
	racine, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine du depot : %v", err)
	}
	cat, err := decfilm.LoadMapQuantCatalog(title.NewPathResolver(racine).MapQuantBoundsPath(title.DefaultSlug))
	if err != nil {
		t.Fatalf("catalogue de bornes : %v", err)
	}
	e, err := cat.Lookup(nom)
	if err != nil {
		t.Fatalf("carte %q : %v", nom, err)
	}
	return e
}

func TestDecodeKillSource_DecodeSousLaCarteResolue(t *testing.T) {
	b := &Builder{}
	res := b.decodeKillSource(context.Background(), "000d5950", entreeDuCatalogue(t, "Cliffhanger"), bobineKillsource000d5950(t))
	if res == nil {
		t.Fatal("source de degat non decodee sous la carte resolue par la cuisson")
	}
	if !strings.Contains(res.Calibration, "[CARTE]") {
		t.Errorf("calibration %q : la carte recue n est pas celle qui decide des largeurs", res.Calibration)
	}
}

// TestDecodeKillSource_EntreeSansLargeursNEstPasDecodee : une entree qui ne decrit pas la carte
// (sans largeurs) n ouvre pas les largeurs par defaut : la source de degat n est pas decodee.
func TestDecodeKillSource_EntreeSansLargeursNEstPasDecodee(t *testing.T) {
	b := &Builder{}
	if res := b.decodeKillSource(context.Background(), "000d5950", decfilm.MapQuantEntry{Module: "vide"}, bobineKillsource000d5950(t)); res != nil {
		t.Errorf("source de degat decodee sans carte (%d ligne(s)) : repli aux largeurs par defaut", len(res.Kills))
	}
}
