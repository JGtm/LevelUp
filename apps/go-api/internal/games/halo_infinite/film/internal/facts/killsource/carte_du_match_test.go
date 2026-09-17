package killsource

// carte_du_match_test.go — LA CARTE DU MATCH ATTEINT LA MARCHE DES MORTS (lot 3.4.1).
//
// # LE DEFAUT QUE CES TEMOINS FERMENT
//
// `killsource.Decode` etait le SEUL chemin de decodage du depot a ne recevoir aucune entree de
// catalogue : il INFERAIT les largeurs d axe du chemin absolu de position par balayage, faute de
// pouvoir les lire. Le lot 3.4.1 demote cette inference en ORACLE (V17, M3-Q8 : « la valeur LUE
// prime sur la valeur mesuree ») — et une inference demotee sans que la valeur lue arrive serait
// une REGRESSION sur toute carte dont les largeurs ne sont pas l invariant, c est-a-dire sur
// toutes sauf `cliffhanger` (13/13/14). Les deux gestes sont donc le meme, et ces temoins le
// tiennent : la carte arrive, elle DECIDE, et son absence se dit.
//
// # DEUX TEMOINS, DEUX PORTEES
//
//	[TestCarteDuMatchDecideLesLargeursDeLaMarche]  INCONDITIONNEL, sur la bobine versionnee : la
//	                                               route et sa morsure, partout ou `go test`
//	                                               tourne, CI comprise.
//	[TestCarteReelleLueParLaMarche]                garde par KS_CARTE_FILM / KS_CARTE_NOM : la
//	                                               MESURE sur un film du cache dont la carte
//	                                               n est PAS l invariant (Fragmentation,
//	                                               17/17/15), avec le verdict de l oracle.

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
	"levelup/go-api/internal/testutil"
)

const (
	ksCarteFilmEnv = "KS_CARTE_FILM"
	ksCarteNomEnv  = "KS_CARTE_NOM"
)

// carteDuCatalogue : l entree commise d une carte, par son nom affiche.
func carteDuCatalogue(t *testing.T, nom string) profile.MapQuantEntry {
	t.Helper()
	racine, err := testutil.RepoRoot()
	if err != nil {
		t.Fatalf("racine du depot : %v", err)
	}
	cat, err := profile.LoadMapQuantCatalog(filepath.Join(racine, "data", "titles",
		"halo_infinite", "reference", "map_quant_bounds.json"))
	if err != nil {
		t.Fatalf("catalogue de bornes : %v", err)
	}
	e, err := cat.Lookup(nom)
	if err != nil {
		t.Fatalf("carte %q au catalogue : %v", nom, err)
	}
	return e
}

// TestCarteDuMatchDecideLesLargeursDeLaMarche — LA ROUTE, ET SA MORSURE.
//
// Mutation qui doit le faire rougir : retirer `opts.Carte` du chemin de `Decode` (les deux
// passes rendent alors les memes largeurs), ou reintroduire une decision de l inference (les
// largeurs cessent de suivre la carte).
func TestCarteDuMatchDecideLesLargeursDeLaMarche(t *testing.T) {
	src, err := source.LoadDir(miniBobineDir, nil)
	if err != nil {
		t.Fatalf("mini-bobine illisible : %v", err)
	}
	// UNE CARTE QUI N EST PAS L INVARIANT : Fragmentation, 17/17/15 contre 13/13/14. Ce n est
	// PAS la carte de cette bobine — la question posee ici est la ROUTE, pas la justesse du
	// decodage, et une carte differente de l invariant est justement ce qui la rend visible.
	frag := carteDuCatalogue(t, "Fragmentation")
	if frag.AxisWidths == profile.MouvementParDefaut().WorldObject.AxisW {
		t.Fatalf("temoin sans valeur : les largeurs de Fragmentation %v EGALENT l invariant",
			frag.AxisWidths)
	}

	sans := calibrationDeLaBobine(t, src, nil)
	avec := calibrationDeLaBobine(t, src, &frag)

	if sans.CarteLue {
		t.Errorf("sans carte, `CarteLue` vaut vrai — le repli ne se declare pas")
	}
	if sans.LueAxisW != profile.MouvementParDefaut().WorldObject.AxisW {
		t.Errorf("sans carte, largeurs %v — l invariant %v etait attendu",
			sans.LueAxisW, profile.MouvementParDefaut().WorldObject.AxisW)
	}
	if !avec.CarteLue {
		t.Fatalf("avec carte, `CarteLue` vaut faux — l entree de catalogue n atteint pas la marche")
	}
	if avec.LueAxisW != frag.AxisWidths {
		t.Fatalf("avec carte, la marche lit %v, le catalogue dit %v — la carte n est pas la source",
			avec.LueAxisW, frag.AxisWidths)
	}
	if avec.LueIndexW != frag.EffectiveRegionIndexBits() {
		t.Errorf("largeur d index de plage %d, catalogue %d",
			avec.LueIndexW, frag.EffectiveRegionIndexBits())
	}
	t.Logf("sans carte : LU %v indexW=%d (repli)", sans.LueAxisW, sans.LueIndexW)
	t.Logf("avec carte : LU %v indexW=%d (Fragmentation)", avec.LueAxisW, avec.LueIndexW)
}

// calibrationDeLaBobine rejoue la calibration de la bobine sous une entree de catalogue.
func calibrationDeLaBobine(t *testing.T, src *source.Film, carte *profile.MapQuantEntry) calibration {
	t.Helper()
	f, err := loadFilm(src)
	if err != nil {
		t.Fatalf("film : %v", err)
	}
	tl, err := newTimeline(f)
	if err != nil {
		t.Fatalf("timeline : %v", err)
	}
	tl.rewind()
	return calibrate(f, tl, DefaultOptions().Views, carte)
}

// TestCarteReelleLueParLaMarche — LA MESURE, SUR UN FILM DONT LA CARTE N EST PAS L INVARIANT.
//
// LECTURE SEULE d UN film, garde par deux variables — saute partout ailleurs, CI comprise :
//
//	KS_CARTE_FILM=<repo>/data/cache/film_chunks/e5adf7b2 KS_CARTE_NOM=Fragmentation \
//	  go test ./internal/games/halo_infinite/film/internal/facts/killsource/ \
//	  -run '^TestCarteReelleLueParLaMarche$' -v
func TestCarteReelleLueParLaMarche(t *testing.T) {
	dir, nom := os.Getenv(ksCarteFilmEnv), os.Getenv(ksCarteNomEnv)
	if dir == "" || nom == "" {
		t.Skipf("%s / %s absents : mesure sautee", ksCarteFilmEnv, ksCarteNomEnv)
	}
	entree := carteDuCatalogue(t, nom)
	src, err := source.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("film %s : %v", dir, err)
	}

	sans := calibrationDeLaBobine(t, src, nil)
	avec := calibrationDeLaBobine(t, src, &entree)
	t.Logf("film %s, carte %q (module %s)", filepath.Base(dir), nom, entree.Module)
	t.Logf("  SANS carte (avant le lot 3.4.1) : %s", sans)
	t.Logf("  AVEC carte (apres)              : %s", avec)

	if avec.LueAxisW != entree.AxisWidths {
		t.Fatalf("la marche lit %v, le catalogue dit %v", avec.LueAxisW, entree.AxisWidths)
	}
	if !avec.CarteLue {
		t.Fatalf("`CarteLue` faux avec une entree de catalogue valide")
	}
	// LA SORTIE PUBLIEE, des deux cotes : la mesure ne vaut que si elle porte sur ce que le
	// paquet PUBLIE, pas seulement sur ce qu il calibre.
	for _, cas := range []struct {
		nom   string
		carte *profile.MapQuantEntry
	}{{"SANS carte", nil}, {"AVEC carte", &entree}} {
		opts := DefaultOptions()
		opts.Carte = cas.carte
		res, errD := Decode(context.Background(), filepath.Base(dir), src, &opts)
		if errD != nil {
			t.Fatalf("%s : decodage : %v", cas.nom, errD)
		}
		c, s := res.Coverage, res.Stats
		t.Logf("  %s : %d morts publiees / %d couples REELS · marche %d appariees / %d "+
			"· scan %d / %d · desaccords oracle %d", cas.nom, c.Covered, c.RealPairs,
			s.Walk.Matched, s.Walk.Population, s.Scan.Matched, s.Scan.Population,
			avec.Desaccords)
	}
}
