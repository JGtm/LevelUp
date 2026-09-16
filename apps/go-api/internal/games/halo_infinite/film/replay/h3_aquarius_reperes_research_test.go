package replay

// h3_aquarius_reperes_research_test.go — LOT H.3 : POURQUOI `0797ce72` ET `c88ec007` NE SE
// RACCROCHENT A AUCUNE CARTE (decouverte D-F5).
//
// CE QUE LE LOT F.1 A MESURE, ET CE QU'IL A LAISSE OUVERT. Ces deux films portent le
// decoupage d'i0 `[13 12 11]`, celui d'`aquarius` et de lui seul au catalogue ; et aucune
// entree de cette classe ne reproduit les REPERES que leurs propres artefacts publient (ecart
// median 29,2 et 29,9 m, quand les 21 autres films tiennent sous 0,20 m). Deux causes etaient
// dites possibles : des BORNES qui auraient change depuis la cuisson, ou des ARTEFACTS
// anterieurs a une correction de bornes.
//
// # LE VERDICT, MESURE LE 2026-09-13 : AUCUNE DES DEUX. CES DEUX FILMS SONT LIVE FIRE.
//
// En imposant a chaque entree SON PROPRE decoupage — ce que fait la PRODUCTION
// (`build_from_film.go` : `scan.Layout = fc.ImposedLayout()`) et ce que ce test ne faisait
// pas —, `live fire` (`sgh_interlock`) reproduit les reperes publies a **0,066 m** et
// **0,082 m**, tres en dessous du seuil de 0,20 m. Les artefacts sont JUSTES, le catalogue
// est JUSTE, et il n'y a rien a recuire.
//
// LA CAUSE EST DANS L'ORACLE DE MESURE, ET ELLE A UN NOM : `DetectI0Layout` lit `[13 12 11]`
// sur ces films parce que l'en-tete d'i0 de Live Fire porte un index de region de DEUX bits
// (sa region jouee est la 1 : catalogue du 2026-08-27), et le profil de bascule impute ce bit
// d'en-tete supplementaire a l'axe X. L'oracle de F.0 §0.3 / F.1 filtrant les candidates sur
// l'EGALITE EXACTE des largeurs, Live Fire n'est jamais proposee — et `aquarius`, seule entree
// en `[13 12 11]`, gagne par defaut a 29 m. La regression du temps 2 le confirme par un autre
// chemin : elle retrouve les bornes de Live Fire sur Y et Z (min -10,15 / -9,38 contre
// -10,10 / -9,33 au catalogue) et son max X (46,74 contre 46,51), l'axe X seul etant decale
// d'une etendue entiere — la signature exacte d'UN BIT DE TROP lu sur cet axe.
//
// LE DEPOT LE DISAIT DEJA, AILLEURS : `config/replay_corpus.toml` porte `0797ce72` comme
// temoin `region_index_2_bits`, `carte = "Live Fire"`, « seule carte du catalogue a index de
// region sur 2 bits » — ecrit le 2026-09-12 pour le correctif de la porte de position des
// objets du monde (schema 53). Corroboration entierement independante de cette mesure.
//
// CE TEST NE CORRIGE PAS L'ORACLE (perimetre H.3 ferme) : il l'etablit, et la decouverte est
// consignee au plan.
//
// CE QUE CE TEST AJOUTE, EN TROIS TEMPS :
//
//	1. IL RELACHE LE FILTRE DE CARTE et essaie TOUTES les bornes du catalogue (dedupliquees
//	   par AABB) — mais TOUJOURS aux largeurs d'axe LUES DANS LE FILM, jamais a celles de
//	   l'entree essayee. C'est la difference qui rend la liste lisible : decoder aux largeurs
//	   d'une autre carte ne deplace pas les positions, il les DETRUIT, et les ecarts qui en
//	   sortent ne classent rien.
//	1 bis. IL REJOUE LE CHEMIN DE LA PRODUCTION : chaque entree essayee AVEC SON PROPRE
//	   decoupage impose (largeurs, region, largeur d'index de region). C'est ce temps-la qui
//	   tranche, et les deux premiers temps sont ce qu'il fallait pour le rendre lisible.
//	2. IL RECUPERE LES BORNES REELLEMENT EMPLOYEES A LA CUISSON, par regression. La
//	   dequantification est affine par axe — `p = min + (q+0,5)·etendue/2^w` — donc les
//	   coordonnees PUBLIEES et celles DECODEES avec une entree de reference sont liees par
//	   `p = a·d + b`, d'ou `etendue = a·etendue_ref` et `min = b + a·min_ref`. Comparer les
//	   bornes ainsi retrouvees a celles du catalogue dit LAQUELLE des deux a bouge — ou
//	   qu'aucune des deux ne repond.
//
// LECTURE SEULE : aucune ecriture, aucune DuckDB, aucun artefact recuit. Skip par defaut — il
// reutilise les gardes de la mesure F.1 (`F1_ROOT`, `F1_ARTS`, `F1_CAT`).
//
//	CGO_ENABLED=0 \
//	  F1_ROOT=<depot>/data/cache/film_chunks F1_ARTS=<depot>/data/cache/replays/halo_infinite \
//	  F1_CAT=<worktree>/data/titles/halo_infinite/reference/map_quant_bounds.json \
//	  H3_IDS=0797ce72,c88ec007 H3_REF=aquarius \
//	  go test ./internal/games/halo_infinite/film/replay/ -run '^TestH3AquariusReperes$' \
//	  -count=1 -timeout 90m -v

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/filmdec"
)

const (
	h3IDsEnv = "H3_IDS"
	h3RefEnv = "H3_REF"
	// h3Candidats : nombre d'entrees les mieux classees a journaliser par film.
	h3Candidats = 6
)

// TestH3AquariusReperes essaie toutes les bornes du catalogue, puis retrouve celles de la
// cuisson par regression.
func TestH3AquariusReperes(t *testing.T) {
	root := os.Getenv(f1RootEnv)
	ids := strings.Split(os.Getenv(h3IDsEnv), ",")
	if root == "" || os.Getenv(f1ArtsEnv) == "" || os.Getenv(h3IDsEnv) == "" {
		t.Skipf("mesure H.3 : definir %s, %s et %s", f1RootEnv, f1ArtsEnv, h3IDsEnv)
	}
	ref := os.Getenv(h3RefEnv)
	if ref == "" {
		ref = "aquarius"
	}
	cat := f1Catalogue(t)
	refEntry, err := cat.Lookup(ref)
	if err != nil {
		t.Fatalf("entree de reference %q absente du catalogue : %v", ref, err)
	}
	t.Logf("reference de regression : %q min=%v max=%v widths=%v",
		ref, refEntry.Min, refEntry.Max, refEntry.AxisWidths)

	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		h3UnFilm(t, cat, refEntry, filepath.Join(root, id), id)
	}
}

// h3UnFilm joue les deux temps sur un film.
func h3UnFilm(t *testing.T, cat *filmdec.MapQuantCatalog, refEntry filmdec.MapQuantEntry,
	dir, id string) {
	t.Helper()
	lay, _, err := detecterI0Layout(dir)
	if err != nil || !lay.Valid() {
		t.Errorf("film %s : decoupage i0 illisible (%v)", id, err)
		return
	}
	vues, ok := f1LitRepere(id)
	if !ok {
		t.Errorf("film %s : artefact inexploitable", id)
		return
	}
	t.Logf("######## FILM %s — decoupage i0 LU %v, %d reperes de piste ########",
		id, lay.AxisW, len(vues))

	// TEMPS 1 — toutes les bornes du catalogue, aux largeurs DU FILM.
	type score struct {
		nom   string
		e     filmdec.MapQuantEntry
		ecart float64
	}
	var scores []score
	for _, c := range h3BornesDistinctes(cat, lay.AxisW) {
		ec := f1Ecart(t, dir, c.e, vues)
		if math.IsInf(ec, 1) {
			continue
		}
		scores = append(scores, score{c.nom, c.e, ec})
	}
	sort.Slice(scores, func(i, j int) bool { return scores[i].ecart < scores[j].ecart })
	for i, s := range scores {
		if i >= h3Candidats {
			break
		}
		t.Logf("  %-2d bornes de %-22s ecart median %.3f m", i+1, s.nom, s.ecart)
	}
	if len(scores) == 0 {
		t.Errorf("film %s : aucune borne n'a pu etre essayee", id)
		return
	}
	t.Logf("  MEILLEURE BORNE DU CATALOGUE : %q a %.3f m", scores[0].nom, scores[0].ecart)

	// TEMPS 1 bis — LE DECOUPAGE IMPOSE PAR L'ENTREE, comme la PRODUCTION le fait
	// (`build_from_film.go` : `scan.Layout = fc.ImposedLayout()`). C'est la difference qui
	// compte : le decoupage LU dans le film est un CONTROLE, l'entree de catalogue est la
	// source d'autorite — et sur une carte dont l'index de region fait 2 bits au lieu d'un,
	// les deux ne coincident pas.
	var imposes []score
	for _, c := range h3EntreesDistinctes(cat) {
		ec := h3EcartImpose(t, dir, c.e, vues)
		if math.IsInf(ec, 1) {
			continue
		}
		imposes = append(imposes, score{c.nom, c.e, ec})
	}
	sort.Slice(imposes, func(i, j int) bool { return imposes[i].ecart < imposes[j].ecart })
	for i, s := range imposes {
		if i >= h3Candidats {
			break
		}
		t.Logf("  [decoupage impose] %-2d %-22s widths=%v region=%d/%d bits · ecart median %.3f m",
			i+1, s.nom, s.e.AxisWidths, s.e.Region, s.e.EffectiveRegionIndexBits(), s.ecart)
	}
	if len(imposes) > 0 {
		t.Logf("  MEILLEURE ENTREE A DECOUPAGE IMPOSE : %q a %.3f m",
			imposes[0].nom, imposes[0].ecart)
	}

	// TEMPS 2 — les bornes de la cuisson, retrouvees.
	h3RetrouveBornes(t, dir, id, refEntry, lay.AxisW, vues)
}

// h3Borne est une AABB du catalogue, ramenee aux largeurs d'axe du film.
type h3Borne struct {
	nom string
	e   filmdec.MapQuantEntry
}

// h3BornesDistinctes rend les AABB distinctes du catalogue, chacune montee aux largeurs
// d'axe LUES DANS LE FILM. Le catalogue porte une quarantaine de canevas de Forge qui
// partagent les memes bornes : les essayer un par un rebalaierait le film pour rien.
func h3BornesDistinctes(cat *filmdec.MapQuantCatalog, widths [3]uint) []h3Borne {
	noms := make([]string, 0, len(cat.Maps))
	for n := range cat.Maps {
		noms = append(noms, n)
	}
	sort.Strings(noms)
	vus, out := map[string]bool{}, []h3Borne{}
	for _, n := range noms {
		e := cat.Maps[n]
		cle := fmt.Sprintf("%v/%v", e.Min, e.Max)
		if vus[cle] {
			continue
		}
		vus[cle] = true
		e.AxisWidths = widths
		out = append(out, h3Borne{n, e})
	}
	return out
}

// h3RetrouveBornes retrouve, par regression affine sur les reperes de piste, les bornes qui
// ont produit les coordonnees PUBLIEES.
func h3RetrouveBornes(t *testing.T, dir, id string, refEntry filmdec.MapQuantEntry,
	widths [3]uint, vues f1Repere) {
	t.Helper()
	e := refEntry
	e.AxisWidths = widths
	pos, ok := f1Positions(t, dir, e)
	if !ok || len(pos) == 0 {
		t.Errorf("film %s : balayage de reference vide", id)
		return
	}
	acc := map[uint32][3][]float64{}
	for _, p := range pos {
		if !p.HasWorld {
			continue
		}
		v := acc[p.Slot]
		v[0] = append(v[0], float64(p.X))
		v[1] = append(v[1], float64(p.Y))
		v[2] = append(v[2], float64(p.Z))
		acc[p.Slot] = v
	}
	var dec, pub [3][]float64
	for slot, att := range vues {
		v, vu := acc[slot]
		if !vu || len(v[0]) < f1ReperePointsMin {
			continue
		}
		for ax := 0; ax < 3; ax++ {
			dec[ax] = append(dec[ax], f1Mediane(v[ax]))
			pub[ax] = append(pub[ax], att[ax])
		}
	}
	if len(dec[0]) < 3 {
		t.Errorf("film %s : %d reperes appaires, trop peu pour une regression", id, len(dec[0]))
		return
	}
	t.Logf("  bornes RETROUVEES depuis %d reperes appaires (reference %q) :",
		len(dec[0]), refEntry.Module)
	for ax, nom := range []string{"X", "Y", "Z"} {
		a, b, res := h3Regression(dec[ax], pub[ax])
		minRef := float64(refEntry.Min[ax])
		etRef := float64(refEntry.Max[ax]) - minRef
		minP := b + a*minRef
		etP := a * etRef
		t.Logf("    %s : pente %.6f · min %.4f (catalogue %.4f) · max %.4f (catalogue %.4f) "+
			"· residu max %.4f m",
			nom, a, minP, minRef, minP+etP, float64(refEntry.Max[ax]), res)
	}
}

// h3Regression rend la pente, l'ordonnee a l'origine et le residu MAXIMAL d'un ajustement
// affine par moindres carres. Le residu dit si la relation est bien affine : au-dela de
// quelques centimetres, les deux jeux de coordonnees ne viennent pas du meme quantum.
func h3Regression(x, y []float64) (pente, origine, residuMax float64) {
	n := float64(len(x))
	var sx, sy, sxx, sxy float64
	for i := range x {
		sx += x[i]
		sy += y[i]
		sxx += x[i] * x[i]
		sxy += x[i] * y[i]
	}
	den := n*sxx - sx*sx
	if den == 0 {
		return math.NaN(), math.NaN(), math.Inf(1)
	}
	pente = (n*sxy - sx*sy) / den
	origine = (sy - pente*sx) / n
	for i := range x {
		residuMax = math.Max(residuMax, math.Abs(y[i]-(pente*x[i]+origine)))
	}
	return pente, origine, residuMax
}

// h3EntreesDistinctes rend les entrees distinctes du catalogue par (bornes, largeurs,
// region) — a l'identique, sans rien forcer.
func h3EntreesDistinctes(cat *filmdec.MapQuantCatalog) []h3Borne {
	noms := make([]string, 0, len(cat.Maps))
	for n := range cat.Maps {
		noms = append(noms, n)
	}
	sort.Strings(noms)
	vus, out := map[string]bool{}, []h3Borne{}
	for _, n := range noms {
		e := cat.Maps[n]
		cle := fmt.Sprintf("%v/%v/%v/%d/%d", e.Min, e.Max, e.AxisWidths, e.Region,
			e.EffectiveRegionIndexBits())
		if vus[cle] {
			continue
		}
		vus[cle] = true
		out = append(out, h3Borne{n, e})
	}
	return out
}

// h3EcartImpose mesure l'ecart aux reperes publies en IMPOSANT le decoupage de l'entree —
// le chemin de la production, et non celui du decoupage lu dans le film.
func h3EcartImpose(t *testing.T, dir string, e filmdec.MapQuantEntry, vues f1Repere) float64 {
	t.Helper()
	pos, ok := h3PositionsImposees(t, dir, e)
	if !ok || len(pos) == 0 {
		return math.Inf(1)
	}
	return h3EcartMedian(pos, vues)
}

// h3PositionsImposees balaie les positions de bipede en imposant le decoupage de l'entree.
func h3PositionsImposees(t *testing.T, dir string, e filmdec.MapQuantEntry) (
	[]filmdec.BipedPosition, bool) {
	t.Helper()
	scan := filmdec.DefaultScanFilmOptions()
	wr := e.Range()
	scan.WorldRange = &wr
	lay := e.Layout()
	scan.Layout = &lay
	pos, err := filmdec.ScanFilmBipedPositions(dir, scan)
	if err != nil {
		return nil, false
	}
	return pos, true
}

// h3EcartMedian est l'ecart MEDIAN, par slot, entre les medianes decodees et celles que
// l'artefact publie — la meme mesure que `f1Ecart`, sur des positions deja balayees.
func h3EcartMedian(pos []filmdec.BipedPosition, vues f1Repere) float64 {
	acc := map[uint32][3][]float64{}
	for _, p := range pos {
		if !p.HasWorld {
			continue
		}
		v := acc[p.Slot]
		v[0] = append(v[0], float64(p.X))
		v[1] = append(v[1], float64(p.Y))
		v[2] = append(v[2], float64(p.Z))
		acc[p.Slot] = v
	}
	var ecarts []float64
	for slot, att := range vues {
		v, vu := acc[slot]
		if !vu || len(v[0]) < f1ReperePointsMin {
			continue
		}
		got := [3]float64{f1Mediane(v[0]), f1Mediane(v[1]), f1Mediane(v[2])}
		ec := 0.0
		for i := range att {
			ec = math.Max(ec, math.Abs(att[i]-got[i]))
		}
		ecarts = append(ecarts, ec)
	}
	if len(ecarts) == 0 {
		return math.Inf(1)
	}
	return f1Mediane(ecarts)
}
