package replay

// f1_origine_mesure_research_test.go — LOT F.1 : LA MESURE AVANT / APRES du retrait de la
// clause de DISTANCE dans `equipmentOrigin`.
//
// # CE QU'ELLE COMPARE
//
// La MEME chaine de production, sur les MEMES poses, avec les deux regles :
//
//	AVANT  `gap <= originDropWindowUS` ET `dist < originDropMaxDist`  -> dropped
//	APRES  `gap <= originDropWindowUS`                                -> dropped
//
// La regle d'AVANT est recopiee ici — et c'est assume : c'est le seul endroit du depot ou elle
// doit survivre, comme TEMOIN de ce que le correctif change. Elle porte son nom
// (`f1OrigineAvant`) et ce commentaire pour qu'aucun lecteur ne la prenne pour la regle vivante.
//
// # LA CARTE DE CHAQUE FILM SE MESURE, ELLE NE SE DECLARE PAS
//
// La distance est en METRES : une carte fausse fausserait la mesure. Deux chaines INDEPENDANTES
// la ferment : le decoupage d'i0 se LIT dans le film (`DetectI0Layout`), ce qui reduit le
// catalogue aux entrees de memes largeurs d'axe ; et parmi celles-la, la seule qui reproduise
// les REPERES PUBLIES par l'artefact (la mediane des coordonnees de chaque piste, en metres
// vrais) est retenue. Un film dont aucune candidate ne reproduit ses reperes a moins d'un metre
// est EXCLU de la mesure, et le rapport le dit.
//
// LECTURE SEULE : aucune ecriture, aucune DuckDB, aucun artefact recuit. Skip par defaut.
//
//	CGO_ENABLED=0 \
//	  F1_ROOT=<depot>/data/cache/film_chunks F1_ARTS=<depot>/data/cache/replays/halo_infinite \
//	  F1_CAT=<worktree>/data/titles/halo_infinite/reference/map_quant_bounds.json \
//	  F1_IDS=a,b,c \
//	  go test ./internal/games/halo_infinite/film/replay/ -run '^TestF1OrigineAvantApres$' \
//	  -count=1 -timeout 90m -v

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
)

const (
	f1RootEnv = "F1_ROOT"
	f1ArtsEnv = "F1_ARTS"
	f1CatEnv  = "F1_CAT"
	f1IDsEnv  = "F1_IDS"
	// f1BornesToleranceM : ecart maximal admis, en metres, entre les reperes de piste
	// RECALCULES avec une carte candidate et ceux que l'artefact PUBLIE. Un metre : les
	// candidates concurrentes d'une meme classe de largeurs different de dizaines a centaines
	// de metres d'etendue, le seuil n'a donc rien a departager de serre.
	f1BornesToleranceM = 1.0
)

// f1OrigineParFenetre est LA REGLE DE LA FENETRE TEMPORELLE, retiree de la PRODUCTION le
// 2026-09-15 (decision utilisateur du lot 1.9.1 : une pose dont le film ne dit rien sort
// `unknown`, elle ne se classe plus par correlation). Elle survit ICI, et ici seulement, comme
// TEMOIN : c'est elle qui permet de mesurer ce que la conversion a change, pose par pose.
//
// ELLE N'EST PLUS APPELEE PAR AUCUN CODE DE PRODUCTION. Si un jour elle l'etait de nouveau, le
// registre des replis devrait reprendre ses deux entrees — sorties le 2026-09-15 avec elle.
func f1OrigineParFenetre(lives []equipLife, p grammar.EquipmentPlacement) string {
	if len(lives) == 0 {
		return OriginUnknown
	}
	best, bestGap := equipLife{}, ^uint64(0)
	for _, v := range lives {
		gap := uint64(0)
		switch {
		case p.T0US < v.from:
			gap = v.from - p.T0US
		case p.T0US > v.to:
			gap = p.T0US - v.to
		}
		if gap == 0 {
			best = v
			bestGap = 0
			break
		}
		if gap < bestGap {
			best, bestGap = v, gap
		}
	}
	if equipTimeGap(p.T0US, best.to) > originDropWindowUS {
		return OriginDeployed
	}
	return OriginDropped
}

// f1OrigineAvant est la regle D'AVANT le lot F.1 — le TEMOIN, jamais la regle vivante : la
// fenetre temporelle ET la clause de distance.
func f1OrigineAvant(lives []equipLife, p grammar.EquipmentPlacement) string {
	apres := f1OrigineParFenetre(lives, p)
	if apres != OriginDropped {
		return apres
	}
	best, ok := f1VieRetenue(lives, p.T0US)
	if !ok {
		return OriginUnknown
	}
	if dist3([3]float32{p.X, p.Y, p.Z}, [3]float32{best.x, best.y, best.z}) >= originDropMaxDist {
		return OriginDeployed
	}
	return OriginDropped
}

// f1VieRetenue rejoue le choix de vie d'`equipmentOrigin` : celle qui CONTIENT l'instant, a
// defaut la plus proche en temps.
func f1VieRetenue(lives []equipLife, atUS uint64) (equipLife, bool) {
	if len(lives) == 0 {
		return equipLife{}, false
	}
	best, bestGap := equipLife{}, ^uint64(0)
	for _, v := range lives {
		gap := uint64(0)
		switch {
		case atUS < v.from:
			gap = v.from - atUS
		case atUS > v.to:
			gap = atUS - v.to
		}
		if gap == 0 {
			return v, true
		}
		if gap < bestGap {
			best, bestGap = v, gap
		}
	}
	return best, true
}

// f1Repere est la MEDIANE des coordonnees publiees, par slot de piste. C'est la piece qui
// identifie la carte, et elle est ROBUSTE : la boite englobante ne l'est pas — la production
// ECARTE les positions aberrantes avant de publier ses bornes (`bounds_aberrants_test.go`),
// donc le min/max recalcule brut s'en ecarte de centaines de metres sur un film qui en porte,
// pendant que la mediane ne bouge pas. Mesure : le premier essai par bornes rendait 0,47 m et
// 0,01 m sur deux films et 240,58 m sur un troisieme, ou la CARTE etait pourtant la bonne.
type f1Repere map[uint32][3]float64

// f1LitRepere lit les pistes de l'artefact et rend la mediane par slot.
func f1LitRepere(id string) (f1Repere, bool) {
	dir := os.Getenv(f1ArtsEnv)
	if dir == "" {
		return nil, false
	}
	raw, err := os.ReadFile(filepath.Join(dir, id+".json"))
	if err != nil {
		return nil, false
	}
	var doc struct {
		Tracks []struct {
			Slot   uint32 `json:"slot"`
			Points []struct {
				X float64 `json:"x"`
				Y float64 `json:"y"`
				Z float64 `json:"z"`
			} `json:"points"`
		} `json:"tracks"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, false
	}
	acc := map[uint32][3][]float64{}
	for _, tr := range doc.Tracks {
		v := acc[tr.Slot]
		for _, pt := range tr.Points {
			v[0] = append(v[0], pt.X)
			v[1] = append(v[1], pt.Y)
			v[2] = append(v[2], pt.Z)
		}
		acc[tr.Slot] = v
	}
	out := f1Repere{}
	for slot, v := range acc {
		if len(v[0]) < f1ReperePointsMin {
			continue
		}
		out[slot] = [3]float64{f1Mediane(v[0]), f1Mediane(v[1]), f1Mediane(v[2])}
	}
	return out, len(out) > 0
}

// f1ReperePointsMin : nombre minimal de points pour qu'une mediane de slot compte. Dix — une
// piste de trois points n'est pas un repere.
const f1ReperePointsMin = 10

func f1Mediane(v []float64) float64 {
	s := append([]float64(nil), v...)
	sort.Float64s(s)
	return s[len(s)/2]
}

// f1Films rend la racine et les films demandes, ou skip l'instrument.
func f1Films(t *testing.T) (string, []string) {
	t.Helper()
	root, ids := os.Getenv(f1RootEnv), os.Getenv(f1IDsEnv)
	if root == "" || ids == "" {
		t.Skipf("instrument F.1 : definir %s et %s", f1RootEnv, f1IDsEnv)
	}
	var out []string
	for _, id := range strings.Split(ids, ",") {
		if id = strings.TrimSpace(id); id != "" {
			out = append(out, id)
		}
	}
	return root, out
}

// f1Catalogue charge le catalogue de bornes de production.
func f1Catalogue(t *testing.T) *profile.MapQuantCatalog {
	t.Helper()
	path := os.Getenv(f1CatEnv)
	if path == "" {
		t.Skipf("instrument F.1 : definir %s (map_quant_bounds.json)", f1CatEnv)
	}
	cat, err := profile.LoadMapQuantCatalog(path)
	if err != nil {
		t.Fatalf("catalogue de bornes : %v", err)
	}
	return cat
}

// f1Carte identifie la carte d'un film : largeurs LUES dans le film, puis la candidate qui
// reproduit les REPERES de piste PUBLIES par l'artefact.
func f1Carte(t *testing.T, dir, id string, cat *profile.MapQuantCatalog) (
	profile.MapQuantEntry, string, float64, bool) {
	t.Helper()
	lay, _, err := detecterI0Layout(dir)
	if err != nil || !lay.Valid() {
		t.Logf("film %s : decoupage i0 illisible (%v) — hors mesure", id, err)
		return profile.MapQuantEntry{}, "", 0, false
	}
	vues, ok := f1LitRepere(id)
	if !ok {
		t.Logf("film %s : aucun artefact exploitable — hors mesure (la carte ne se ferme pas)", id)
		return profile.MapQuantEntry{}, "", 0, false
	}
	// CANDIDATES DEDUPLIQUEES PAR BORNES, et ce n'est pas une optimisation gratuite : le
	// catalogue porte une quarantaine de canevas de Forge qui partagent le MEME AABB. Les
	// essayer un par un, c'est rebalayer le film quarante fois pour quarante fois le meme
	// resultat — mesure : la campagne restait bloquee sur un film BTB.
	vus := map[string]bool{}
	var cands []struct {
		nom string
		e   profile.MapQuantEntry
	}
	noms := make([]string, 0, len(cat.Maps))
	for n := range cat.Maps {
		noms = append(noms, n)
	}
	sort.Strings(noms)
	for _, n := range noms {
		e := cat.Maps[n]
		if e.AxisWidths != lay.AxisW {
			continue
		}
		cle := fmt.Sprintf("%v/%v", e.Min, e.Max)
		if vus[cle] {
			continue
		}
		vus[cle] = true
		cands = append(cands, struct {
			nom string
			e   profile.MapQuantEntry
		}{n, e})
	}
	bestNom, bestEcart := "", math.Inf(1)
	var best profile.MapQuantEntry
	for _, c := range cands {
		if ec := f1Ecart(t, dir, c.e, vues); ec < bestEcart {
			best, bestNom, bestEcart = c.e, c.nom, ec
		}
	}
	if bestNom == "" || bestEcart > f1BornesToleranceM {
		t.Logf("film %s : aucune carte de largeurs %v ne reproduit les reperes publies "+
			"(meilleur ecart %.2f m) — hors mesure", id, lay.AxisW, bestEcart)
		return profile.MapQuantEntry{}, "", bestEcart, false
	}
	return best, bestNom, bestEcart, true
}

// f1Ecart rend l'ecart MEDIAN, en metres, entre les reperes de slot recalcules avec `e` et ceux
// que l'artefact publie. Median sur les slots aussi : une piste dont la decimation a mange un
// long arret ne doit pas emporter le verdict a elle seule.
func f1Ecart(t *testing.T, dir string, e profile.MapQuantEntry, vues f1Repere) float64 {
	t.Helper()
	pos, ok := f1Positions(t, dir, e)
	if !ok || len(pos) == 0 {
		return math.Inf(1)
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

// f1Positions balaie les positions de bipede avec les bornes donnees, sous le verrou de
// decodage et la precision de la carte.
func f1Positions(t *testing.T, dir string, e profile.MapQuantEntry) ([]grammar.BipedPosition, bool) {
	t.Helper()
	scan := grammar.DefaultScanFilmOptions()
	wr := e.Range()
	scan.WorldRange = &wr
	pos, err := grammar.ScanFilmBipedPositions(dir, scan)
	if err != nil {
		return nil, false
	}
	sort.SliceStable(pos, func(i, j int) bool { return pos[i].TimestampUS < pos[j].TimestampUS })
	return pos, true
}
