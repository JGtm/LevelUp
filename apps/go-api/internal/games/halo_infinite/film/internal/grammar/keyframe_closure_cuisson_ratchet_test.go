package grammar

// keyframe_closure_cuisson_ratchet_test.go — LA FERMETURE D IMAGE-CLE EN CONTEXTE DE CUISSON NE
// DESCEND JAMAIS.
//
// # CE QUE CE RATCHET GARDE, QUE `TestKeyframeClosureRatchet` NE VOIT PAS
//
// `TestKeyframeClosureRatchet` mesure les sept bobines dans le contexte d un film SANS carte
// (`NewFilmContext`, decoupage MPP resolu). Sans carte, `i60 simulation-state` n est pas porte
// (`SimStateComplet` suit la carte, [grammaireSousCarte]) et la marche d un bipede s y arrete toujours :
// une lecture des records `ti=35` y est invisible. Ce ratchet-ci mesure les memes bobines dans le
// contexte de la CUISSON : l entree de catalogue de la carte du match ([NewFilmContextForMap]), puis
// le contexte de carte que la cuisson pose ([FilmContext.PoserLaCarteEtLeDecoupage], le meme geste,
// pas une copie : garde-rail `archlint/pose_de_carte_unique_test.go`). Le profil calibre par killsource
// n y entre pas : sur ces bobines il donne les memes lignes (mesure du plan LK, R-7).
//
// La carte de chaque bobine est la premiere identite de carte de ses faits d equivalence versionnes
// (`replay/testdata/equivalence/<id>.facts.json`), resolue au catalogue de bornes versionne : toutes
// les donnees du test sont dans le depot, il joue en CI.
//
// Meme comparateur que le golden sans carte ([comparerFermeture]) : une BAISSE ou une ligne qui
// DISPARAIT rougit, une hausse se fige par la porte nommee.
//
// REGENERATION (jamais d edition a la main) :
//
//	go test ./internal/games/halo_infinite/film/internal/grammar/ -run KeyframeClosureCuissonRatchet -update-keyframe-closure-cuisson

import (
	"encoding/json"
	"flag"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// updateFermetureCuisson : LA PORTE DE REGENERATION DE CE GOLDEN, ET D AUCUN AUTRE.
var updateFermetureCuisson = flag.Bool("update-keyframe-closure-cuisson", false,
	"reecrire testdata/keyframe_closure_cuisson.golden — CE golden seulement")

// closureCuissonGoldenPath : le golden, a cote de celui du contexte sans carte.
const closureCuissonGoldenPath = "testdata/keyframe_closure_cuisson.golden"

// fermeturesDeCuisson : [KeyframeClosure] par bobine, en contexte de cuisson (decodage memorise).
var fermeturesDeCuisson memoDeBobine[map[uint32]KeyframeClosureStat]

// TestKeyframeClosureCuissonRatchet : la fermeture par archetype en contexte de cuisson ne descend
// jamais.
func TestKeyframeClosureCuissonRatchet(t *testing.T) {
	got := mesurerFermetureCuisson(t)
	if *updateFermetureCuisson {
		if err := os.WriteFile(closureCuissonGoldenPath, []byte(got), 0o600); err != nil {
			t.Fatalf("ecriture du golden : %v", err)
		}
		// Une porte de regeneration ne rend jamais `ok` (cf. TestKeyframeClosureRatchet).
		t.Fatalf("1 reference(s) reecrite(s) : %s (%d octets) ; relancer sans "+
			"-update-keyframe-closure-cuisson pour verifier", closureCuissonGoldenPath, len(got))
	}
	brut, err := os.ReadFile(closureCuissonGoldenPath) //nolint:gosec // chemin fige dans le code
	if err != nil {
		t.Fatalf("golden absent (%s) : %v — regenerer avec -update-keyframe-closure-cuisson",
			closureCuissonGoldenPath, err)
	}
	comparerFermeture(t, string(brut), got, "-update-keyframe-closure-cuisson")
}

// mesurerFermetureCuisson rend le rendu textuel de la mesure sur les sept bobines, historique en tete.
func mesurerFermetureCuisson(t *testing.T) string {
	t.Helper()
	var b strings.Builder
	b.WriteString("# FERMETURE DES RECORDS D'IMAGE-CLE PAR ARCHETYPE, EN CONTEXTE DE CUISSON.\n")
	b.WriteString("# Contexte : carte des faits d'equivalence au catalogue de bornes, puis le contexte de carte\n")
	b.WriteString("# de la cuisson (FilmContext.PoserLaCarteEtLeDecoupage). Colonnes : film, archetype, fermes,\n")
	b.WriteString("# total, bloquant le plus frequent. Le ratchet rougit sur une BAISSE de `fermes`. Regeneration :\n")
	b.WriteString("#   go test ./internal/games/halo_infinite/film/internal/grammar/ -run KeyframeClosureCuissonRatchet -update-keyframe-closure-cuisson\n")
	b.WriteString("#\n")
	b.WriteString("# HISTORIQUE DES REGENERATIONS — une ligne par lot, avec CE QUI MONTE ET POURQUOI.\n")
	b.WriteString("#\n")
	b.WriteString("#   2026-10-08 plan LK (LK.2.4) : creation, 7 bobines, 215 lignes ; ti=35 ferme 53 records\n")
	b.WriteString("#     sur 1 368 (la reference du plan, R-7 : lignes « cuisson base » de la mesure 2).\n")
	b.WriteString("#   2026-10-08 plan LK (LK.3) : la portee DAT_144e61ea0 posee par la marche d etat complet,\n")
	b.WriteString("#     branche absolue d i0 de l ecrivain sous elle. ti=35 ferme 53 -> 666 records sur 1 368.\n")
	b.WriteString("#     UNE BAISSE, ADJUGEE : 60ae07c4 ti=35 2 -> 1, le record du slot 539 au bit 200 424, qui\n")
	b.WriteString("#     fermait par hasard a la base (plan LK, §2 « Adjudication »).\n")
	b.WriteString("#   2026-10-08 plan LK (LK.5.2) : queue de FUN_14076e3e4 fidele sous la portee (handle par\n")
	b.WriteString("#     FUN_1408f0ac4(.., 0), 13 bits ; puis le mot de region). Aucune ligne ne bouge.\n")
	b.WriteString("#   2026-10-08 plan LK (LK.5.4.2) : ti=5 i12 player-desired-respawn-location sous la portee\n")
	b.WriteString("#     (porte, R(96), R(19)). Aucune ligne ne bouge.\n")
	b.WriteString("#   2026-10-08 plan LK (LK.5.4.3) : ti=14 i0 crew-order sous la portee (R(3), porte, R(96)).\n")
	b.WriteString("#     Aucune ligne ne bouge.\n")
	b.WriteString("#   2026-10-08 plan LK (LK.5.4.4) : ti=30 i0 tacmap-poiicon sous la portee (le vecteur R(96)).\n")
	b.WriteString("#     Aucune ligne ne bouge.\n")
	b.WriteString("#   2026-10-08 plan LK (LK.5.4.5) : ti=32 i0 tacmap-areaofinterest sous la portee (R(32), R(3),\n")
	b.WriteString("#     R(96), R(12)). Aucune ligne ne bouge.\n")
	b.WriteString("#   2026-10-08 plan LK (LK.5.4.6) : ti=33 i0 tacmap-displayasset sous la portee (la position\n")
	b.WriteString("#     R(96)). Aucune ligne ne bouge.\n")
	for _, court := range closureMiniFilms() {
		stats := fermetureDeCuisson(t, court)
		tis := make([]int, 0, len(stats))
		for ti := range stats {
			tis = append(tis, int(ti))
		}
		sort.Ints(tis)
		for _, ti := range tis {
			s := stats[uint32(ti)] //nolint:gosec // ti vient d'une cle uint32
			fmt.Fprintf(&b, "%s\tti=%d\t%d\t%d\t%s\n", court, ti, s.Closed, s.Total, s.Blocking)
		}
	}
	return b.String()
}

// fermetureDeCuisson charge une bobine, ouvre son contexte de cuisson et rend sa fermeture par
// archetype, calculee une fois par processus de test ; chaque appelant recoit sa copie.
func fermetureDeCuisson(t *testing.T, court string) map[uint32]KeyframeClosureStat {
	t.Helper()
	carte := carteDesFaitsDEquivalence(t, court)
	stats, err := fermeturesDeCuisson.valeur(court, func() (map[uint32]KeyframeClosureStat, error) {
		dir := filepath.Join("..", "..", "replay", "testdata", "minifilm_"+court)
		film, err := source.LoadDir(dir, nil)
		if err != nil {
			return nil, fmt.Errorf("LoadDir %s : %w", dir, err)
		}
		cat, err := profile.LoadMapQuantCatalog(e191bCatalogue())
		if err != nil {
			return nil, fmt.Errorf("catalogue de bornes : %w", err)
		}
		entree, err := cat.Lookup(carte)
		if err != nil {
			return nil, fmt.Errorf("carte %q de %s : %w", carte, court, err)
		}
		fc := NewFilmContextForMap(film, &entree, nil)
		fc.PoserLaCarteEtLeDecoupage()
		s, err := KeyframeClosure(fc)
		if err != nil {
			return nil, fmt.Errorf("KeyframeClosure %s : %w", court, err)
		}
		return s, nil
	})
	if err != nil {
		t.Fatalf("%v", err)
	}
	return maps.Clone(stats)
}

// carteDesFaitsDEquivalence rend la premiere identite de carte des faits d equivalence versionnes
// d un film.
func carteDesFaitsDEquivalence(t *testing.T, court string) string {
	t.Helper()
	chemin := filepath.Join("..", "..", "replay", "testdata", "equivalence", court+".facts.json")
	brut, err := os.ReadFile(chemin) //nolint:gosec // chemin construit dans le test
	if err != nil {
		t.Fatalf("faits de %s : %v", court, err)
	}
	var f struct {
		MapNames []string `json:"mapNames"`
	}
	if err := json.Unmarshal(brut, &f); err != nil || len(f.MapNames) == 0 {
		t.Fatalf("carte de %s introuvable dans %s (%v)", court, chemin, err)
	}
	return f.MapNames[0]
}
