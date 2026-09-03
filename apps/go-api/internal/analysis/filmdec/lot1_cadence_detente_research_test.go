package filmdec

// lot1_cadence_detente_research_test.go — DETENTE vs BALLE : le film emet-il un
// action_weapon_fire (0xD2 long) par DETENTE (une rafale = 1 event) ou par BALLE (une rafale
// BR75 = 3 events) ?
//
// POURQUOI CETTE MESURE. La precision par arme = touches / TIRS. Le denominateur « tirs film »
// n'est comparable a l'ancre API (match_participants.shots_fired) que si les deux comptent la
// MEME chose. Le moteur compte par BALLE (Ghidra : automatic_weapon_shot_count, compteur
// ShotsFired += 1 par round). Si le film emettait un event par DETENTE, le denominateur film
// serait ~3x trop bas sur le BR75 (rafale de 3) et la precision film surestimee d'autant.
//
// METHODE. On groupe les tirs 0xD2 longs par (TIREUR = FilmIndex, ARME = WeaponID) et on mesure
// l'ESPACEMENT temporel entre tirs CONSECUTIFS du meme tireur/arme.
//   - PAR BALLE, arme AUTOMATIQUE (MA40) : distribution UNIMODALE au temps de cycle (~83 ms
//     = 720 RPM nominal).
//   - PAR BALLE, arme a RAFALE (BR75, 3 rounds) : distribution BIMODALE — petits ecarts
//     INTRA-rafale (2 par cycle) + grand ecart INTER-rafale (1 par cycle, ~3x le petit).
//   - PAR DETENTE, BR75 : distribution UNIMODALE au seul rythme INTER-rafale, AUCUN petit ecart.
//
// Le discriminant est donc la PRESENCE d'un mode « intra-rafale » court sur le BR75. Controle
// positif integre : le MA40 (automatique pur) DOIT sortir unimodal — si l'instrument ne voit
// pas sa cadence connue, il ne lit rien.
//
// Garde LOT1_TRAME_FILM. Un film par process, verrou pris, borne a deltaWitnessChunks (12).
// Lancer une fois par film (000d5950, 01e1f945, 00502e52).

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// cadenceEdges : bornes (us) des tranches d'ecart inter-tir, du plus court au plus long.
var cadenceEdges = []uint64{
	20_000, 40_000, 60_000, 90_000, 130_000, 200_000, 300_000, 500_000, 1_000_000,
}

// cadenceLabels : etiquettes lisibles alignees sur cadenceEdges (une de plus, la queue).
var cadenceLabels = []string{
	"<20ms", "20-40", "40-60", "60-90", "90-130", "130-200", "200-300", "300-500", "500-1000", ">1s",
}

// cadenceBucket range un ecart (us) dans sa tranche.
func cadenceBucket(gap uint64) int {
	for i, e := range cadenceEdges {
		if gap < e {
			return i
		}
	}
	return len(cadenceEdges)
}

// cadenceHist rend l'histogramme textuel d'une liste d'ecarts.
func cadenceHist(gaps []uint64) string {
	buckets := make([]int, len(cadenceEdges)+1)
	for _, g := range gaps {
		buckets[cadenceBucket(g)]++
	}
	s := ""
	for i, b := range buckets {
		if b == 0 {
			continue
		}
		if s != "" {
			s += " "
		}
		s += fmt.Sprintf("%s:%d", cadenceLabels[i], b)
	}
	return s
}

// cadenceMedianU64 rend la mediane (us) d'une liste d'ecarts.
func cadenceMedianU64(gaps []uint64) uint64 {
	if len(gaps) == 0 {
		return 0
	}
	cp := append([]uint64(nil), gaps...)
	sort.Slice(cp, func(a, b int) bool { return cp[a] < cp[b] })
	return cp[len(cp)/2]
}

// TestLot1CadenceDetente mesure l'espacement inter-tir par arme pour trancher detente vs balle.
func TestLot1CadenceDetente(t *testing.T) {
	dir := os.Getenv(lot1TrameFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument saute", lot1TrameFilmEnv)
	}
	release := LockProcessDecode()
	defer release()

	n := CountFilmChunks(dir)
	if n > deltaWitnessChunks {
		n = deltaWitnessChunks
	}
	t.Logf("== film %s · %d chunks (detente vs balle) ==", filepath.Base(dir), n)

	shots := attribCollectShots(t, dir, n)

	// Groupe (tireur FilmIndex, arme WeaponID) -> ts tries.
	type key struct {
		fidx int
		wid  uint64
	}
	byKey := map[key][]uint64{}
	for _, s := range shots {
		if !s.has {
			continue
		}
		k := key{s.fidx, s.wid}
		byKey[k] = append(byKey[k], s.ts)
	}

	// Ecarts consecutifs, agreges par arme.
	gapsByWid := map[uint64][]uint64{}
	for k, ts := range byKey {
		if len(ts) < 2 {
			continue
		}
		sort.Slice(ts, func(a, b int) bool { return ts[a] < ts[b] })
		for i := 1; i < len(ts); i++ {
			g := ts[i] - ts[i-1]
			if g == 0 || g > 3_000_000 { // 0 = doublon meme paquet ; >3s = changement de contexte
				continue
			}
			gapsByWid[k.wid] = append(gapsByWid[k.wid], g)
		}
	}

	type row struct {
		wid  uint64
		gaps []uint64
	}
	var rows []row
	for w, g := range gapsByWid {
		rows = append(rows, row{w, g})
	}
	sort.Slice(rows, func(i, j int) bool { return len(rows[i].gaps) > len(rows[j].gaps) })

	t.Logf("ESPACEMENT inter-tir consecutif (meme tireur+arme) — cle du verdict = mode intra-rafale court sur le BR75 :")
	for _, r := range rows {
		if len(r.gaps) < 8 { // effectif trop faible : bruit
			continue
		}
		// Fraction d'ecarts « courts » (< 40 ms) : signature intra-rafale / cadence auto rapide.
		short := 0
		for _, g := range r.gaps {
			if g < 40_000 {
				short++
			}
		}
		t.Logf("   %-24s : n=%d ecarts · mediane %d ms · <40ms %.0f %% · %s",
			attribWeaponName(r.wid), len(r.gaps), cadenceMedianU64(r.gaps)/1000,
			100*float64(short)/float64(len(r.gaps)), cadenceHist(r.gaps))
	}
	t.Logf("LECTURE : MA40 (auto) unimodal ~80ms = controle positif. BR75 bimodal (petit intra-rafale + grand inter) = PAR BALLE ; BR75 unimodal sans petit mode = PAR DETENTE.")
}
