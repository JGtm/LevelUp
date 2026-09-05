package filmdec

// r6_layout117_research_test.go — lot R6 du PLAN_LECTURE_FIABLE_EQUIPEMENT_2026-09-03,
// question A : la charge utile de l'evenement 117 (EquipmentTranslocatorTeleportEffects),
// layout SOURCE DE L'EXE (Ghidra, projet HI, lecteur FUN_140f04fb8) puis REJOUE sur les
// evenements reels des films temoins.
//
// LAYOUT SOURCE (decompilation FUN_140f04fb8 + FUN_14080d69c + FUN_14076e524 + FUN_140cc5128,
// ecrivain FUN_142eec354 symetrique) — charge apres l'en-tete commun (config + continuation +
// R(7) type + 3 refs gardees, ref0 domaine 2 = 8 bits base 512) :
//
//	[R(1) g0 ; si g0 : R(32) mot]            // chaine-id d'effet, defaut -1 — mesure constant
//	                                         // 0xA1344FC2 (le « 0x42689F84 aligne octet » de R1
//	                                         // etait un artefact de fenetre decale d'un bit)
//	position A : [R(1) gr ; si gr : R(wr) region] R(bx) R(by) R(bz)
//	position B : idem
//
// Dequantification (FUN_14076e524, demi-pas DAT_143cd84b0 = 0.5) :
//	pos[i] = min[i] + (q[i] + 0.5) * (max[i] - min[i]) / 2^b[i]
// Bits par axe (FUN_140be9b88, granularite k=16 : DAT_143cd9758 = 1/120 m) :
//	b[i] = min(26, ceil(log2(min(2^22, ceil(extent[i] * 60)))))
// — cette formule REPRODUIT les axisWidths de map_quant_bounds.json (verifie : aquarius
// 13/12/11). Si gr=0 : bornes par defaut +/-20000 (DAT_143b8c6b8), b = 22/22/22.
// wr = DAT_144632be0 (runtime, log2ceil du nombre de regions BSP ; 1 si une seule region) —
// balaye ici en hypothese 1..4.
//
// VERITE TERRAIN : la piste du slot designe par ref0 dans l'artefact du rejeu (metres vrais,
// piege R3 par.0 : les bornes de dequantification sont celles de map_quant_bounds.json, JAMAIS
// le champ bounds de l'artefact qui n'est qu'un cadrage d'affichage). La carte du film n'etant
// documentee nulle part (DuckDB interdit), elle est IDENTIFIEE par calibration : l'entree du
// catalogue qui valide le plus d'evenements gagne (methode R3 par.0.4).
//
// LECTURE SEULE, skip par defaut, CGO_ENABLED=0. USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 \
//	  R6_ROOT=<repo>/data/cache/film_chunks \
//	  R6_ARTS=<repo>/data/cache/replays/halo_infinite \
//	  R6_CAT=<repo>/data/titles/halo_infinite/reference/map_quant_bounds.json \
//	  R6_IDS=1b2d9e08,a0c36016,4577fcc4,f2966f08,faff9935 \
//	  go test ./internal/analysis/filmdec/ -run '^TestR6Layout117$' -timeout 20m -v

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

const (
	r6RootEnv = "R6_ROOT"
	r6ArtsEnv = "R6_ARTS"
	r6CatEnv  = "R6_CAT"
	r6IDsEnv  = "R6_IDS"
	// r6TolM : tolerance de validation (2D) entre position decodee et position de piste.
	// La piste est sur une grille de 100 ms : un bipede a 6 m/s bouge 0,6 m par frame.
	r6TolM = 1.5
)

// r6CatEntry est une entree de map_quant_bounds.json (bornes VRAIES de quantification).
type r6CatEntry struct {
	Module     string     `json:"module"`
	Min        [3]float64 `json:"min"`
	Max        [3]float64 `json:"max"`
	AxisWidths [3]uint    `json:"axisWidths"`
}

type r6Catalog struct {
	Maps map[string]r6CatEntry `json:"maps"`
}

// r6Artefact est le sous-ensemble utile de l'artefact du rejeu (les pistes sont decodees a
// part, en generique, car chaque point porte t/x/y/z sous des cles distinctes).
type r6Artefact struct {
	OriginMs        int64 `json:"originMs"`
	FrameIntervalMs int64 `json:"frameIntervalMs"`
}

// r6Point est un point de piste (t en frames, x/y/z en metres vrais).
type r6Point struct {
	t       int64
	x, y, z float64
}

// r6Occ est une tete de type 117 avec son payload copie.
type r6Occ struct {
	tsMS   int64
	chunk  int
	paquet int
	pay    []byte
}

// r6Decoded est le resultat d'un decodage sous une hypothese (entree, wr).
type r6Decoded struct {
	slot         int
	gen          uint64
	g0           bool
	mot          uint64
	grA, grB     bool
	ridxA, ridxB uint64
	ax, ay, az   float64
	bx, by, bz   float64
	cont         bool
	ok           bool
	motif        string
}

// TestR6Layout117 rejoue le layout exe sur les evenements 117 des films temoins.
func TestR6Layout117(t *testing.T) {
	root, arts, ids := os.Getenv(r6RootEnv), os.Getenv(r6ArtsEnv), os.Getenv(r6IDsEnv)
	catPath := os.Getenv(r6CatEnv)
	if root == "" || arts == "" || ids == "" || catPath == "" {
		t.Skipf("instrument R6 : definir %s, %s, %s et %s", r6RootEnv, r6ArtsEnv, r6CatEnv, r6IDsEnv)
	}
	cat := r6LireCatalogue(t, catPath)
	totalOK, totalEv := 0, 0
	for _, id := range strings.Split(ids, ",") {
		id = strings.TrimSpace(id)
		t.Logf("")
		t.Logf("############ FILM %s ############", id)
		occs := r6Recense117(t, filepath.Join(root, id))
		art, pistes := r6LireArtefact(t, filepath.Join(arts, id+".json"))
		ok, ev := r6ValideFilm(t, occs, art, pistes, cat)
		totalOK += ok
		totalEv += ev
	}
	t.Logf("")
	t.Logf("############ BILAN PARC : %d/%d evenements 117 valides (from ET to a <= %.1f m) ############",
		totalOK, totalEv, r6TolM)
}

func r6LireCatalogue(t *testing.T, path string) map[string]r6CatEntry {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("catalogue illisible %s : %v", path, err)
	}
	var c r6Catalog
	if err := json.Unmarshal(raw, &c); err != nil {
		t.Fatalf("catalogue indecodable : %v", err)
	}
	// pseudo-entree : bornes par defaut du moteur (+/-20000, bits 22) — le cas gr=0.
	c.Maps["__defaut_20k__"] = r6CatEntry{Module: "defaut moteur",
		Min: [3]float64{-20000, -20000, -20000}, Max: [3]float64{20000, 20000, 20000},
		AxisWidths: [3]uint{22, 22, 22}}
	t.Logf("== catalogue : %d entrees (pseudo-entree defaut incluse) ==", len(c.Maps))
	return c.Maps
}

// r6Recense117 balaie les paquets delta et copie le payload des tetes de type 117.
func r6Recense117(t *testing.T, dir string) []r6Occ {
	t.Helper()
	n := CountFilmChunks(dir)
	if n == 0 {
		t.Fatalf("aucun chunk film dans %s", dir)
	}
	var origine uint64
	raw, err := ReadFilmChunk(dir, 1)
	if err != nil {
		t.Fatalf("chunk 1 illisible : %v", err)
	}
	if pks := WalkPackets(raw); len(pks) > 0 {
		origine = pks[0].TimestampUS
	} else {
		t.Fatalf("aucun paquet dans le chunk 1")
	}
	var occs []r6Occ
	total := 0
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeDelta || pk.Size < 2 {
				continue
			}
			total++
			pay := pk.Payload(data)
			if pay[0]&0xC0 != 0xC0 {
				continue
			}
			if typ := int(pay[0]&0x3F)<<1 | int(pay[1]>>7); typ != 117 {
				continue
			}
			cp := make([]byte, 0, 48)
			if len(pay) > 48 {
				cp = append(cp, pay[:48]...)
			} else {
				cp = append(cp, pay...)
			}
			occs = append(occs, r6Occ{tsMS: (int64(pk.TimestampUS) - int64(origine)) / 1000,
				chunk: c, paquet: pk.Index, pay: cp})
		}
	}
	t.Logf("== %d paquets delta, %d tetes 117 ==", total, len(occs))
	return occs
}

func r6LireArtefact(t *testing.T, path string) (r6Artefact, map[int][]r6Point) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("artefact illisible %s : %v", path, err)
	}
	// decodage generique pour recuperer x, y, z separement (le struct tague ne suffit pas :
	// json ne mappe qu'un tag par champ).
	var g struct {
		OriginMs        int64 `json:"originMs"`
		FrameIntervalMs int64 `json:"frameIntervalMs"`
		Tracks          []struct {
			Slot   int              `json:"slot"`
			Points []map[string]any `json:"points"`
		} `json:"tracks"`
	}
	if err := json.Unmarshal(raw, &g); err != nil {
		t.Fatalf("artefact indecodable : %v", err)
	}
	if g.FrameIntervalMs == 0 {
		g.FrameIntervalMs = 100
	}
	pistes := map[int][]r6Point{}
	for _, tr := range g.Tracks {
		for _, p := range tr.Points {
			pt := r6Point{}
			if v, ok := p["t"].(float64); ok {
				pt.t = int64(v)
			}
			if v, ok := p["x"].(float64); ok {
				pt.x = v
			}
			if v, ok := p["y"].(float64); ok {
				pt.y = v
			}
			if v, ok := p["z"].(float64); ok {
				pt.z = v
			}
			pistes[tr.Slot] = append(pistes[tr.Slot], pt)
		}
	}
	for slot := range pistes {
		sort.Slice(pistes[slot], func(i, j int) bool { return pistes[slot][i].t < pistes[slot][j].t })
	}
	return r6Artefact{OriginMs: g.OriginMs, FrameIntervalMs: g.FrameIntervalMs}, pistes
}

// r6FromTo cherche la discontinuite de piste du slot autour de la frame de l'evenement :
// la paire de frames consecutives au deplacement 2D maximal dans [f-2, f+6].
func r6FromTo(pts []r6Point, frame int64) (r6Point, r6Point, float64, bool) {
	var from, to r6Point
	best := -1.0
	for i := 0; i+1 < len(pts); i++ {
		if pts[i].t < frame-2 || pts[i].t > frame+6 || pts[i+1].t != pts[i].t+1 {
			continue
		}
		d := math.Hypot(pts[i+1].x-pts[i].x, pts[i+1].y-pts[i].y)
		if d > best {
			best, from, to = d, pts[i], pts[i+1]
		}
	}
	return from, to, best, best >= 0
}

// r6Decode decode une tete 117 sous une hypothese (entree de catalogue, largeur wr).
func r6Decode(pay []byte, e r6CatEntry, wr uint) r6Decoded {
	d := r6Decoded{}
	br := NewBitReader(pay)
	br.Skip(9) // config + continuation + R(7) type
	if !br.ReadBit() {
		d.motif = "ref0 absente"
		return d
	}
	d.slot = int(br.ReadBits(8)) + 512
	d.gen = br.ReadBits(2)
	if br.ReadBit() || br.ReadBit() { // ref1 / ref2 : domaines non sources pour i != 0
		d.motif = "ref1/ref2 presente (domaine non source)"
		return d
	}
	d.g0 = br.ReadBit()
	if d.g0 {
		d.mot = br.ReadBits(32)
	}
	// PORTE INVERSEE (decompilation FUN_14076e524, verifiee sur pieces : le bloc "lire
	// l'index de region" s'execute quand le bit vaut 0) : bit 0 -> index de region R(wr) +
	// bornes de LA region ; bit 1 -> bornes par defaut du moteur (+/-20000, bits 22).
	lire := func() (float64, float64, float64, bool, uint64, bool) {
		gr := br.ReadBit()
		var ridx uint64
		min, max, bits := e.Min, e.Max, e.AxisWidths
		if !gr {
			ridx = br.ReadBits(wr)
		} else {
			min = [3]float64{-20000, -20000, -20000}
			max = [3]float64{20000, 20000, 20000}
			bits = [3]uint{22, 22, 22}
		}
		var out [3]float64
		for i := 0; i < 3; i++ {
			q := br.ReadBits(bits[i])
			out[i] = min[i] + (float64(q)+0.5)*(max[i]-min[i])/float64(uint64(1)<<bits[i])
		}
		return out[0], out[1], out[2], gr, ridx, br.Remaining() >= 0
	}
	var okA, okB bool
	d.ax, d.ay, d.az, d.grA, d.ridxA, okA = lire()
	d.bx, d.by, d.bz, d.grB, d.ridxB, okB = lire()
	if br.Remaining() < 1 {
		d.motif = "payload trop court pour l'hypothese"
		return d
	}
	d.cont = br.ReadBit()
	d.ok = okA && okB
	return d
}

// r6ValideFilm essaie chaque entree du catalogue x wr 1..4, retient la combinaison qui valide
// le plus d'evenements, et detaille chaque evenement sous cette combinaison.
func r6ValideFilm(t *testing.T, occs []r6Occ, art r6Artefact, pistes map[int][]r6Point,
	cat map[string]r6CatEntry) (int, int) {
	t.Helper()
	if len(occs) == 0 {
		t.Logf("== aucun evenement 117 dans ce film ==")
		return 0, 0
	}
	type combo struct {
		nom string
		wr  uint
	}
	scores := map[combo]int{}
	for nom, e := range cat {
		for wr := uint(1); wr <= 4; wr++ {
			c := combo{nom, wr}
			for _, o := range occs {
				if _, ok := r6Match(o, e, wr, art, pistes); ok {
					scores[c]++
				}
			}
		}
	}
	var best combo
	bestN := -1
	for c, n := range scores {
		if n > bestN || (n == bestN && (c.nom < best.nom || (c.nom == best.nom && c.wr < best.wr))) {
			best, bestN = c, n
		}
	}
	if bestN <= 0 {
		t.Logf("== AUCUNE combinaison (entree catalogue x wr) ne valide le moindre evenement ==")
		for _, o := range occs {
			d := r6Decode(o.pay, cat["__defaut_20k__"], 1)
			t.Logf("  @%d ms slot %d : g0=%v mot=0x%08X grA=%v grB=%v motif=%q tete % X",
				o.tsMS, d.slot, d.g0, d.mot, d.grA, d.grB, d.motif, o.pay[:16])
		}
		return 0, len(occs)
	}
	e := cat[best.nom]
	t.Logf("== MEILLEURE HYPOTHESE : entree %q (module %s, bits %v) x wr=%d : %d/%d valides ==",
		best.nom, e.Module, e.AxisWidths, best.wr, bestN, len(occs))
	nOK := 0
	for _, o := range occs {
		det, ok := r6Match(o, e, best.wr, art, pistes)
		if ok {
			nOK++
		}
		t.Logf("  %s", det)
	}
	return nOK, len(occs)
}

// r6Match decode une occurrence sous l'hypothese et la confronte a la discontinuite de piste.
// Rend le detail formate et le verdict (from ET to a <= r6TolM, dans un ordre ou l'autre).
func r6Match(o r6Occ, e r6CatEntry, wr uint, art r6Artefact, pistes map[int][]r6Point) (string, bool) {
	d := r6Decode(o.pay, e, wr)
	if d.motif != "" {
		return fmt.Sprintf("@%d ms : indecodable (%s)", o.tsMS, d.motif), false
	}
	frame := (o.tsMS - art.OriginMs) / art.FrameIntervalMs
	pts := pistes[d.slot]
	if len(pts) == 0 {
		return fmt.Sprintf("@%d ms slot %d : aucune piste artefact", o.tsMS, d.slot), false
	}
	from, to, saut, ok := r6FromTo(pts, frame)
	if !ok {
		return fmt.Sprintf("@%d ms slot %d : pas de paire de frames consecutives autour de f=%d",
			o.tsMS, d.slot, frame), false
	}
	dAfrom := math.Hypot(d.ax-from.x, d.ay-from.y)
	dBto := math.Hypot(d.bx-to.x, d.by-to.y)
	dAto := math.Hypot(d.ax-to.x, d.ay-to.y)
	dBfrom := math.Hypot(d.bx-from.x, d.by-from.y)
	ordre, e1, e2 := "A=from B=to", dAfrom, dBto
	if math.Max(dAto, dBfrom) < math.Max(dAfrom, dBto) {
		ordre, e1, e2 = "A=to B=from", dAto, dBfrom
	}
	valide := e1 <= r6TolM && e2 <= r6TolM
	verdict := "ECHEC"
	if valide {
		verdict = "VALIDE"
	}
	return fmt.Sprintf("@%d ms slot %d f=%d : %s · mot=0x%08X g0=%v grA=%v(%d) grB=%v(%d) cont=%v · "+
		"A=(%.2f,%.2f,%.2f) B=(%.2f,%.2f,%.2f) · saut piste %.2f m from=(%.2f,%.2f) to=(%.2f,%.2f) · "+
		"%s : dA=%.2f dB=%.2f m",
		o.tsMS, d.slot, frame, verdict, d.mot, d.g0, d.grA, d.ridxA, d.grB, d.ridxB, d.cont,
		d.ax, d.ay, d.az, d.bx, d.by, d.bz, saut, from.x, from.y, to.x, to.y,
		ordre, e1, e2), valide
}
