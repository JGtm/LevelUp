package grammar

// livefire_positions_test.go — D1 (3.4.1) : LE CHEMIN DU REJEU EST-IL JUSTE SUR UNE CARTE A
// REGION ? LA MESURE, SUR LES DEUX FILMS LIVE FIRE DU CORPUS.
//
// # LA QUESTION
//
// Le lot 3.4.1 a corrige le deserialiseur d i0 (`consumeAbsolutePayload`) : « index 1 / no-index
// = +/-20000 » etait faux, et le filtre `if idx != 0 { return }` jetait les positions valides
// des cartes dont la plage jouee n est pas la 0. Les fixtures de contrat re-cuites apres ce
// correctif n ont PAS bouge d un octet — revisions exceptees — y compris sur `60ae07c4`, un des
// deux films Live Fire. Deux lectures possibles, et une seule est vraie :
//
//	(a) le rejeu porte le MEME defaut ailleurs, et personne ne le voit ;
//	(b) le rejeu ne passe pas par ce deserialiseur, et son chemin a lui est DEJA juste.
//
// # LA REPONSE, ET C EST (b)
//
// Le rejeu decode ses positions de bipede par `ScanBipedPositions` -> `walkDeltaBipedPayload` ->
// [matchBipedHeader], qui compare l index de plage lu a `lay.Region` — la valeur du CATALOGUE,
// posee par `NewFilmContextForMap` depuis le lot C catalogues (2026-08-27) et etendue aux six
// canaux delta au lot 3 (2026-09-03). Ce chemin lit donc la plage JOUEE de Live Fire (la 1) et
// ecarte les autres depuis un an de lots ; le deserialiseur d i0, lui, servait le crochet de
// capture de position, que la cuisson n installe pas.
//
// CE TEST LE MESURE AU LIEU DE L AFFIRMER. Il balaie les deux films sous DEUX decoupages qui ne
// different QUE par la region attendue : celui du CATALOGUE (`region = 1`) et celui d avant le
// lot C (`region = 0`, c est-a-dire la regle « seule la plage 0 est une position »). Le premier
// est ce que la production lit AUJOURD HUI ; le second est ce qu elle lisait avant.
//
// LECTURE SEULE, borne aux premiers chunks, garde par LIVEFIRE_POS_FILMS — saute partout
// ailleurs, CI comprise. Aucune ecriture, aucun artefact cuit.
//
//	LIVEFIRE_POS_FILMS='<repo>/data/cache/film_chunks/60ae07c4;<repo>/data/cache/film_chunks/0797ce72' \
//	  go test ./internal/games/halo_infinite/film/internal/grammar/ -run '^TestLiveFirePositionsParPlage$' -v

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
	"levelup/go-api/internal/testutil"
)

const liveFirePosEnv = "LIVEFIRE_POS_FILMS"

// liveFirePosChunks : le nombre de chunks balayes par film. BORNE ASSUMEE — la question est
// « le chemin ecarte-t-il les positions de la plage jouee ? », et elle se tranche sur un
// echantillon : un film entier ne changerait pas le verdict, il changerait le compte.
const liveFirePosChunks = 12

// TestLiveFirePositionsParPlage — LA MESURE.
func TestLiveFirePositionsParPlage(t *testing.T) {
	brut := os.Getenv(liveFirePosEnv)
	if brut == "" {
		t.Skipf("%s absent : mesure sautee", liveFirePosEnv)
	}
	entree := entreeLiveFire(t)
	t.Logf("catalogue `live fire` : module %s, axes %v, region %d sur %d bits",
		entree.Module, entree.AxisWidths, entree.Region, entree.EffectiveRegionIndexBits())

	for _, dir := range strings.Split(brut, ";") {
		dir = strings.TrimSpace(dir)
		if dir == "" {
			continue
		}
		t.Run(filepath.Base(dir), func(t *testing.T) {
			catalogue := entree.Layout()
			avant := catalogue
			avant.Region = 0 // la regle d avant le lot C : seule la plage 0 est une position
			nCat, slotsCat := positionsSousDecoupage(t, dir, catalogue)
			nZero, slotsZero := positionsSousDecoupage(t, dir, avant)
			t.Logf("%s (%d premiers chunks)", filepath.Base(dir), liveFirePosChunks)
			t.Logf("  plage JOUEE (region=%d, ce que la production lit AUJOURD HUI) : "+
				"%d positions, %d slots", catalogue.Region, nCat, slotsCat)
			t.Logf("  plage 0 (la regle d avant le lot C catalogues)                : "+
				"%d positions, %d slots", nZero, slotsZero)
			if nCat == 0 {
				t.Fatalf("AUCUNE position sous la plage jouee : le chemin du rejeu porterait le "+
					"meme defaut que `consumeAbsolutePayload` avant le lot 3.4.1, et D1 (3.4.1) "+
					"serait a CORRIGER et non a fermer (%s)", dir)
			}
			if nCat <= nZero {
				t.Errorf("la plage jouee ne rend pas PLUS de positions que la plage 0 "+
					"(%d contre %d) : la porte de region ne mord pas", nCat, nZero)
			}
		})
	}
}

// entreeLiveFire : l entree de catalogue commise de Live Fire.
func entreeLiveFire(t *testing.T) profile.MapQuantEntry {
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
	e, err := cat.Lookup("Live Fire")
	if err != nil {
		t.Fatalf("live fire au catalogue : %v", err)
	}
	return e
}

// positionsSousDecoupage rend le nombre de positions lues et le nombre de slots distincts, sous
// un decoupage IMPOSE — c est le seul parametre qui change d un appel a l autre.
func positionsSousDecoupage(t *testing.T, dir string, lay profile.I0Layout) (int, int) {
	t.Helper()
	film, err := source.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("film %s : %v", dir, err)
	}
	chunks := make([]int, 0, liveFirePosChunks)
	for c := 1; c <= liveFirePosChunks && c <= film.NumChunks(); c++ {
		chunks = append(chunks, c)
	}
	opt := DefaultScanFilmOptions()
	opt.Chunks = chunks
	opt.Layout = &lay
	opt.QuantaOnly = true // la question est le NOMBRE de positions lues, pas leur coordonnee
	pos, err := ScanBipedPositions(NewFilmContextForMap(film, nil, &lay), opt)
	if err != nil {
		t.Fatalf("balayage de %s : %v", dir, err)
	}
	slots := map[uint32]bool{}
	for _, p := range pos {
		slots[p.Slot] = true
	}
	return len(pos), len(slots)
}
