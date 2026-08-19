package mapvar

// socles_research_test.go — INSTRUMENT DE MESURE : les socles d'armes sont-ils dans le
// fichier de carte ?
//
// CE QU'IL CHERCHE. La detection des socles en production (`replay/ground_weapon_pads.go`)
// exige une RECURRENCE dans le film : deux apparitions au meme point. Un socle servi une
// seule fois est donc invisible, et l'objet deja pose a t=0 n'est pas repliqué. Si le
// `.mvar` de la carte portait les spawners, on connaitrait TOUS les socles des le depart.
//
// LECTURE SEULE, aucune base, aucune ecriture. Plan (hypotheses et seuils ECRITS AVANT la
// mesure) : `.ai/V7.5/replay2d/PLAN_SOCLES_MVAR.md`.
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 MVAR_FILE=<depot>/.ai/re_dump/mapvar/catalyst_catalyst.mvar \
//	  go test ./internal/analysis/replay/mapvar/ -run '^TestSoclesInventaire$' -v
//
//	CGO_ENABLED=0 MVAR_FILE=<depot>/.ai/re_dump/mapvar/catalyst_catalyst.mvar \
//	  MVAR_ORACLE=<depot>/data/cache/replays/halo_infinite/01e1f945.json \
//	  MVAR_ORACLE_POINTS="0.257,-0.003,21.36" \
//	  go test ./internal/analysis/replay/mapvar/ -run '^TestSoclesAppariement$' -v
//
//	CGO_ENABLED=0 MVAR_DIR=<depot>/.ai/re_dump/mapvar \
//	  go test ./internal/analysis/replay/mapvar/ -run '^TestSoclesCorpus$' -v

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand/v2"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// LES GARDES. Sans elles le test SKIPPE : aucun fichier de jeu n'est en depot, et un test
// qui se declare vert sans avoir rien lu ne garde rien.
const (
	soclesFileEnv   = "MVAR_FILE"          // un .mvar a inventorier / apparier
	soclesDirEnv    = "MVAR_DIR"           // le depot de .mvar (balayage de corpus)
	soclesOracleEnv = "MVAR_ORACLE"        // artefact de rejeu cuit, porteur de weaponPads
	soclesPtsEnv    = "MVAR_ORACLE_POINTS" // points additionnels "x,y,z;x,y,z" (socle power-up)
)

// soclesSeuilM est le rayon d'appariement, ecrit au plan AVANT la mesure.
const soclesSeuilM = 1.0

// soclesTirages / soclesGraine : le temoin negatif, reproductible.
const (
	soclesTirages = 100
	soclesGraine  = uint64(20260819)
)

// -------------------------------------------------------------------------------------
// Chargement
// -------------------------------------------------------------------------------------

// soclesVariante lit le .mvar designe par la garde, ou saute le test.
func soclesVariante(t *testing.T) (*Variant, string) {
	t.Helper()
	path := strings.TrimSpace(os.Getenv(soclesFileEnv))
	if path == "" {
		t.Skipf("%s absent — instrument de mesure ignore", soclesFileEnv)
	}
	buf, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("lecture de %s: %v", path, err)
	}
	v, err := Parse(buf)
	if err != nil {
		t.Fatalf("parse de %s: %v", path, err)
	}
	return v, path
}

// -------------------------------------------------------------------------------------
// Item 0.1 / phase 1 — inventaire de TOUS les objets, pas seulement des roles connus
// -------------------------------------------------------------------------------------

// TestSoclesInventaire dresse l'etat des lieux d'une variante : types, categories,
// labels, formes, noms. C'est la reponse a la question 1 (le fichier porte-t-il des
// spawners d'armes ?), et elle se lit dans les histogrammes, pas dans une intuition.
func TestSoclesInventaire(t *testing.T) {
	v, path := soclesVariante(t)
	t.Logf("== %s : %d objets, level_id %d, %d noms ==",
		filepath.Base(path), len(v.Objects), v.LevelID, len(v.Names))

	types, cats, avecForme, avecEquipe := map[int32]int{}, map[int]int{}, 0, 0
	labels, inconnus := map[string]int{}, map[int32]int{}
	for _, o := range v.Objects {
		types[o.TypeID]++
		cats[o.Category]++
		if o.ShapeRaw != nil {
			avecForme++
		}
		if o.TeamIndex != TeamUnset {
			avecEquipe++
		}
		for _, h := range o.Labels {
			if n := LabelName(h); n != "" {
				labels[n]++
			} else {
				inconnus[h]++
			}
		}
	}
	t.Logf("type_id distincts : %d | categories distinctes : %d | avec forme : %d | avec equipe : %d",
		len(types), len(cats), avecForme, avecEquipe)
	soclesLogTypes(t, types, len(v.Objects))
	soclesLogCats(t, cats)
	soclesLogLabels(t, labels, inconnus)
	soclesLogNoms(t, v.Names)
	soclesLogRacine(t, path)
}

// soclesLogTypes affiche l'histogramme des type_id, les plus frequents d'abord.
func soclesLogTypes(t *testing.T, types map[int32]int, total int) {
	t.Helper()
	type paire struct {
		id int32
		n  int
	}
	liste := make([]paire, 0, len(types))
	for id, n := range types {
		liste = append(liste, paire{id, n})
	}
	sort.Slice(liste, func(i, j int) bool { return liste[i].n > liste[j].n })
	for i, p := range liste {
		if i >= 25 {
			t.Logf("  ... %d autres type_id", len(liste)-25)
			break
		}
		t.Logf("  type_id %12d : %5d objets (%.1f %%)", p.id, p.n, 100*float64(p.n)/float64(total))
	}
}

// soclesLogCats affiche l'histogramme des categories.
func soclesLogCats(t *testing.T, cats map[int]int) {
	t.Helper()
	cles := make([]int, 0, len(cats))
	for c := range cats {
		cles = append(cles, c)
	}
	sort.Ints(cles)
	for _, c := range cles {
		t.Logf("  categorie %4d : %5d objets", c, cats[c])
	}
}

// soclesLogLabels affiche les labels resolus et les hashs inconnus. UN HASH INCONNU EST
// LE SIGNAL LE PLUS INTERESSANT : c'est la que vivrait un label de spawner jamais nomme.
func soclesLogLabels(t *testing.T, labels map[string]int, inconnus map[int32]int) {
	t.Helper()
	noms := make([]string, 0, len(labels))
	for n := range labels {
		noms = append(noms, n)
	}
	sort.Strings(noms)
	for _, n := range noms {
		t.Logf("  label %-24s : %4d", n, labels[n])
	}
	if len(inconnus) == 0 {
		t.Log("  labels inconnus : AUCUN")
		return
	}
	for h, n := range inconnus {
		t.Logf("  label INCONNU %12d : %4d", h, n)
	}
}

// soclesLogNoms echantillonne la table de chaines.
func soclesLogNoms(t *testing.T, noms []string) {
	t.Helper()
	if len(noms) == 0 {
		t.Log("  table de chaines : VIDE")
		return
	}
	for i, n := range noms {
		if i >= 30 {
			t.Logf("  ... %d autres noms", len(noms)-30)
			break
		}
		t.Logf("  nom[%3d] %q", i, n)
	}
}

// -------------------------------------------------------------------------------------
// Item 0.2 — les champs racine JAMAIS decodes (root[6] regroupements, root[11] surcharges)
// -------------------------------------------------------------------------------------

// soclesLogRacine inventorie les champs de la racine Bond, y compris ceux que la grammaire
// declare « non exploites ». Si l'identite d'un spawner est ecrite quelque part, c'est un
// candidat de premier rang.
func soclesLogRacine(t *testing.T, path string) {
	t.Helper()
	buf, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("relecture: %v", err)
	}
	root, err := DecodeRoot(buf)
	if err != nil {
		t.Fatalf("DecodeRoot: %v", err)
	}
	cles := make([]int, 0, len(root.Fields))
	for id := range root.Fields {
		cles = append(cles, int(id))
	}
	sort.Ints(cles)
	t.Logf("== champs de la racine : %v ==", cles)
	for _, id := range cles {
		f := root.Fields[uint16(id)]
		t.Logf("  root[%2d] type %2d, %d items, %d champs, str=%q, int=%d",
			id, f.Type, len(f.Items), len(f.Fields), soclesTronque(f.Str), f.Int)
	}
	for _, id := range []uint16{6, 11} {
		f, ok := root.Field(id)
		if !ok {
			t.Logf("  root[%d] ABSENT de ce fichier", id)
			continue
		}
		soclesLogSousArbre(t, fmt.Sprintf("root[%d]", id), f, 0)
	}
}

// soclesLogSousArbre deroule un sous-arbre sur deux niveaux — assez pour dire ce qu'il
// porte, pas assez pour noyer la sortie.
func soclesLogSousArbre(t *testing.T, prefixe string, v Value, prof int) {
	t.Helper()
	if prof > 2 {
		return
	}
	t.Logf("  %s type %d : %d items, %d champs, int=%d, str=%q",
		prefixe, v.Type, len(v.Items), len(v.Fields), v.Int, soclesTronque(v.Str))
	for i, it := range v.Items {
		if i >= 3 {
			t.Logf("  %s ... %d items de plus", prefixe, len(v.Items)-3)
			break
		}
		soclesLogSousArbre(t, fmt.Sprintf("%s[%d]", prefixe, i), it, prof+1)
	}
	cles := make([]int, 0, len(v.Fields))
	for id := range v.Fields {
		cles = append(cles, int(id))
	}
	sort.Ints(cles)
	for _, id := range cles {
		soclesLogSousArbre(t, fmt.Sprintf("%s.%d", prefixe, id), v.Fields[uint16(id)], prof+1)
	}
}

func soclesTronque(s string) string {
	if len(s) > 40 {
		return s[:40] + "..."
	}
	return s
}

// -------------------------------------------------------------------------------------
// Phase 2 — l'oracle : les socles mesures dans un film, et leur appariement
// -------------------------------------------------------------------------------------

// soclesOracle est la partie utile d'un artefact de rejeu cuit.
type soclesOracle struct {
	MatchID string `json:"matchId"`
	Bounds  struct {
		MinX, MinY, MaxX, MaxY, MinZ, MaxZ float64
	} `json:"bounds"`
	WeaponPads []struct {
		X, Y, Z float64
		Weapon  string
	} `json:"weaponPads"`
}

// soclesPoint est une position de l'oracle, avec ce qui l'a produite.
type soclesPoint struct {
	Pos    Vec3
	Arme   string
	Source string
}

// soclesChargeOracle lit l'artefact designe et y ajoute les points fournis a la main
// (le socle de power-up, mesure par une autre voie que `weaponPads`).
func soclesChargeOracle(t *testing.T) ([]soclesPoint, soclesOracle) {
	t.Helper()
	path := strings.TrimSpace(os.Getenv(soclesOracleEnv))
	if path == "" {
		t.Skipf("%s absent — appariement ignore", soclesOracleEnv)
	}
	buf, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("lecture de l'oracle %s: %v", path, err)
	}
	var doc soclesOracle
	if err := json.Unmarshal(buf, &doc); err != nil {
		t.Fatalf("decodage de l'oracle %s: %v", path, err)
	}
	pts := make([]soclesPoint, 0, len(doc.WeaponPads)+2)
	for _, p := range doc.WeaponPads {
		pts = append(pts, soclesPoint{Pos: Vec3{X: p.X, Y: p.Y, Z: p.Z}, Arme: p.Weapon, Source: "weaponPads"})
	}
	pts = append(pts, soclesPointsManuels(t)...)
	if len(pts) == 0 {
		t.Fatalf("oracle %s : aucun socle (weaponPads absent)", filepath.Base(path))
	}
	return pts, doc
}

// soclesPointsManuels decode MVAR_ORACLE_POINTS ("x,y,z;x,y,z").
func soclesPointsManuels(t *testing.T) []soclesPoint {
	t.Helper()
	brut := strings.TrimSpace(os.Getenv(soclesPtsEnv))
	if brut == "" {
		return nil
	}
	var out []soclesPoint
	for _, bloc := range strings.Split(brut, ";") {
		champs := strings.Split(strings.TrimSpace(bloc), ",")
		if len(champs) != 3 {
			t.Fatalf("%s: %q n'est pas un triplet x,y,z", soclesPtsEnv, bloc)
		}
		var c [3]float64
		for i := range champs {
			f, err := strconv.ParseFloat(strings.TrimSpace(champs[i]), 64)
			if err != nil {
				t.Fatalf("%s: %q illisible: %v", soclesPtsEnv, champs[i], err)
			}
			c[i] = f
		}
		out = append(out, soclesPoint{Pos: Vec3{X: c[0], Y: c[1], Z: c[2]}, Source: "manuel"})
	}
	return out
}

// soclesPlusProche rend l'index de l'objet le plus proche, la distance 3D et la distance XY.
func soclesPlusProche(objs []Object, p Vec3) (int, float64, float64) {
	best, bestD, bestXY := -1, math.Inf(1), math.Inf(1)
	for i, o := range objs {
		dx, dy, dz := o.Pos.X-p.X, o.Pos.Y-p.Y, o.Pos.Z-p.Z
		d := math.Sqrt(dx*dx + dy*dy + dz*dz)
		if d < bestD {
			best, bestD, bestXY = i, d, math.Hypot(dx, dy)
		}
	}
	return best, bestD, bestXY
}

// TestSoclesAppariement — items 2.2 et 2.3 : chaque socle de l'oracle a-t-il un objet du
// .mvar a moins d'un metre, et un tirage au hasard fait-il aussi bien ?
func TestSoclesAppariement(t *testing.T) {
	v, path := soclesVariante(t)
	pts, doc := soclesChargeOracle(t)
	t.Logf("== %s (%d objets) contre l'oracle %s (%d socles) ==",
		filepath.Base(path), len(v.Objects), doc.MatchID, len(pts))

	apparies, appariesXY := 0, 0
	dists := make([]float64, 0, len(pts))
	for i, p := range pts {
		idx, d, dxy := soclesPlusProche(v.Objects, p.Pos)
		dists = append(dists, d)
		if d < soclesSeuilM {
			apparies++
		}
		if dxy < soclesSeuilM {
			appariesXY++
		}
		o := v.Objects[idx]
		t.Logf("  socle %2d (%8.3f %8.3f %8.3f, %s %s) -> objet %3d type_id %12d a %7.2f m (XY %7.2f m)",
			i+1, p.Pos.X, p.Pos.Y, p.Pos.Z, p.Source, p.Arme, idx, o.TypeID, d, dxy)
	}
	sort.Float64s(dists)
	t.Logf("APPARIEMENT 3D : %d / %d (%.1f %%) a moins de %.0f m — seuil du plan : 90 %%",
		apparies, len(pts), 100*float64(apparies)/float64(len(pts)), soclesSeuilM)
	t.Logf("DIAGNOSTIC XY  : %d / %d (%.1f %%)",
		appariesXY, len(pts), 100*float64(appariesXY)/float64(len(pts)))
	t.Logf("distances au plus proche objet : min %.2f m, mediane %.2f m, max %.2f m",
		dists[0], dists[len(dists)/2], dists[len(dists)-1])

	soclesTemoin(t, v.Objects, pts, doc)
}

// soclesTemoin tire des positions au hasard dans l'emprise de la carte, en conservant
// l'altitude des socles : sans lui, « a moins d'un metre » ne prouve rien sur une carte
// dense en objets.
func soclesTemoin(t *testing.T, objs []Object, pts []soclesPoint, doc soclesOracle) {
	t.Helper()
	if doc.Bounds.MaxX <= doc.Bounds.MinX || doc.Bounds.MaxY <= doc.Bounds.MinY {
		t.Log("TEMOIN : emprise absente de l'oracle, tirage impossible")
		return
	}
	r := rand.New(rand.NewPCG(soclesGraine, soclesGraine))
	total := 0
	for tirage := 0; tirage < soclesTirages; tirage++ {
		for _, p := range pts {
			q := Vec3{
				X: doc.Bounds.MinX + r.Float64()*(doc.Bounds.MaxX-doc.Bounds.MinX),
				Y: doc.Bounds.MinY + r.Float64()*(doc.Bounds.MaxY-doc.Bounds.MinY),
				Z: p.Pos.Z,
			}
			if _, d, _ := soclesPlusProche(objs, q); d < soclesSeuilM {
				total++
			}
		}
	}
	n := soclesTirages * len(pts)
	t.Logf("TEMOIN : %d / %d (%.1f %%) a moins de %.0f m (graine %d, emprise X[%.1f %.1f] Y[%.1f %.1f]) — seuil du plan : <= 20 %%",
		total, n, 100*float64(total)/float64(n), soclesSeuilM, soclesGraine,
		doc.Bounds.MinX, doc.Bounds.MaxX, doc.Bounds.MinY, doc.Bounds.MaxY)
}

// -------------------------------------------------------------------------------------
// Phase 4 — le corpus : combien de .mvar par carte, et portent-ils la meme chose ?
// -------------------------------------------------------------------------------------

// TestSoclesCorpus balaie le depot de .mvar : un fichier par ligne, avec de quoi voir
// d'un coup d'oeil si un fichier de MODE differe d'un fichier de carte autrement que par
// ses objectifs.
func TestSoclesCorpus(t *testing.T) {
	dir := strings.TrimSpace(os.Getenv(soclesDirEnv))
	if dir == "" {
		t.Skipf("%s absent — balayage de corpus ignore", soclesDirEnv)
	}
	entrees, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("lecture de %s: %v", dir, err)
	}
	lus, echecs := 0, 0
	for _, e := range entrees {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".mvar") {
			continue
		}
		buf, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Errorf("  %s : lecture impossible (%v)", e.Name(), err)
			echecs++
			continue
		}
		v, err := Parse(buf)
		if err != nil {
			t.Logf("  %-56s PARSE KO : %v", e.Name(), err)
			echecs++
			continue
		}
		lus++
		t.Logf("  %-56s level_id %12d  objets %5d  types %4d  objectifs %3d  noms %5d",
			e.Name(), v.LevelID, len(v.Objects), soclesNbTypes(v), len(v.Objectives()), len(v.Names))
	}
	t.Logf("CORPUS : %d fichiers lus, %d en echec", lus, echecs)
}

// soclesNbTypes compte les type_id distincts d'une variante — le seul chiffre qui
// distingue d'un coup une palette d'objets riche (carte Forge) d'un fichier qui ne porte
// que des volumes d'objectif.
func soclesNbTypes(v *Variant) int {
	vus := map[int32]struct{}{}
	for _, o := range v.Objects {
		vus[o.TypeID] = struct{}{}
	}
	return len(vus)
}
