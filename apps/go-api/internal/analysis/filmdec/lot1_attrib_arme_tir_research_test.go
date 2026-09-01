package filmdec

// lot1_attrib_arme_tir_research_test.go — ATTRIBUTION PAR LE TIR de la PRECISION et de la
// DISTANCE par arme, mesuree sur le film Theater.
//
// CADRAGE. L'arme est connue par le TIR : action_weapon_fire (0xD2 type 36 LONG) porte le
// WeaponID 64 bits a offsets fixes (decodeFireEvent). LE PONT TIR<->DEGAT : l'ATTAQUANT du tir
// (ref0 dom1) et le RESPONSABLE du degat (ref1 dom1, damage_aftermath) vivent dans le meme espace
// d'index brut. On apparie (meme index brut, |ts_degat - ts_tir| <= W). Le BLESSE d'un degat
// apparie est sa ref0, resolu en slot (base 512) pour la position -> distance tireur<->victime.
//
// PRODUCTIONISE (Lot 2) : la collecte des tirs (ScanFilmWeaponShots), la collecte des degats
// (ScanFilmWeaponDamages) et le pairing/distance (PairWeaponHits) vivent desormais dans
// weapon_hits.go / weapon_hits_decode.go. CET INSTRUMENT LES APPELLE — il reproduit ses taux via
// le code de production, il ne re-decode plus rien lui-meme. Restent ici les MESURES de recherche
// (fiabilite au temoin, sens physique, degats sans refs, tag source) qui ne sont pas du ressort de
// la production.
//
// SEUILS / TEMOINS :
//	W    = 250 ms : fenetre principale (sweep 250/500/1000/2000 ms) ; le VERDICT porte sur le
//	       RATIO au temoin decale (+3 s), pas sur le taux absolu.
//	tol  = sondePosTolUS (120 ms) : ecart max evenement<->echantillon de position (distances).
//
// Garde LOT1_TRAME_FILM. Un film par process, verrou pris, lecture seule, borne a
// deltaWitnessChunks. Lancer une fois par film (000d5950, 01e1f945, 00502e52).

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis"
)

const (
	attribW     = uint64(250_000)   // 250 ms : fenetre d'appariement principale
	attribOFF   = uint64(3_000_000) // 3 s : temoin decale
	attribWideW = uint64(2_000_000) // 2 s : fenetre elargie (projectiles lents)
)

// attribWeaponName nomme une arme par son WeaponID (metadata weapon_labels) ; a defaut, l'hexa.
func attribWeaponName(wid uint64) string {
	if n, ok := analysis.WeaponIDToName[wid]; ok {
		return n
	}
	return fmt.Sprintf("wid#%016x", wid)
}

// attribDamagesToSonde reduit des WeaponDamage aux champs qu'utilise sondeBaseSweep (ts + refs).
func attribDamagesToSonde(ds []WeaponDamage) []sondeDmgEvt {
	out := make([]sondeDmgEvt, len(ds))
	for i, d := range ds {
		out[i] = sondeDmgEvt{ts: d.TimestampUS, idx0: d.VictimIdx, idx1: d.ResponsibleIdx}
	}
	return out
}

// attribM1 mesure la FIABILITE du lien tir<->degat (avant, arriere) contre le temoin decale, via
// les index temporels du meme pont attaquant que PairWeaponHits.
func attribM1(t *testing.T, shots []WeaponShot, dmg []WeaponDamage) {
	t.Helper()
	var dmgTs, dmgResp, shotTs, shotAtt []uint64
	for _, d := range dmg {
		if d.ResponsibleIdx < 0 {
			continue
		}
		dmgTs = append(dmgTs, d.TimestampUS)
		dmgResp = append(dmgResp, uint64(d.ResponsibleIdx))
	}
	for _, s := range shots {
		if !s.HasPair {
			continue
		}
		shotTs = append(shotTs, s.TimestampUS)
		shotAtt = append(shotAtt, s.Attacker)
	}
	dmgByResp := lot1mtIndexByKey(dmgTs, dmgResp)
	shotByAtt := lot1mtIndexByKey(shotTs, shotAtt)

	fwdN, fwdSame, fwdShift := 0, 0, 0
	for _, s := range shots {
		if !s.HasPair {
			continue
		}
		fwdN++
		if lot1mtNear(dmgByResp[s.Attacker], s.TimestampUS, attribW) {
			fwdSame++
		}
		if lot1mtNear(dmgByResp[s.Attacker], s.TimestampUS+attribOFF, attribW) {
			fwdShift++
		}
	}
	revN, revSame, revShift := 0, 0, 0
	for _, e := range dmg {
		if e.ResponsibleIdx < 0 {
			continue
		}
		revN++
		key := uint64(e.ResponsibleIdx)
		if lot1mtNear(shotByAtt[key], e.TimestampUS, attribW) {
			revSame++
		}
		if lot1mtNear(shotByAtt[key], e.TimestampUS+attribOFF, attribW) {
			revShift++
		}
	}
	fs, fsh := lot1Pct(fwdSame, fwdN), lot1Pct(fwdShift, fwdN)
	rs, rsh := lot1Pct(revSame, revN), lot1Pct(revShift, revN)
	floor := func(v float64) float64 {
		if v < 1 {
			return 1
		}
		return v
	}
	fwdRatio, revRatio := fs/floor(fsh), rs/floor(rsh)
	t.Logf("M1 FIABILITE tir<->degat (pont attaquant dom1) :")
	t.Logf("   AVANT  (tir -> degat meme attaquant ±%dms) : %d/%d (%.1f %%) · temoin +%ds %.1f %% -> %.1fx",
		attribW/1000, fwdSame, fwdN, fs, attribOFF/1_000_000, fsh, fwdRatio)
	t.Logf("   ARRIERE (degat -> tir meme attaquant ±%dms) : %d/%d (%.1f %%) · temoin +%ds %.1f %% -> %.1fx",
		attribW/1000, revSame, revN, rs, attribOFF/1_000_000, rsh, revRatio)
	ok := fwdN >= 20 && revN >= 10 && fwdRatio >= 1.5 && revRatio >= 2
	t.Logf("   VERDICT lien reel (avant >=1.5x ET arriere >=2x le temoin) : %s", lot1Verdict(ok))
	attribSweep(t, shots, dmgByResp)
}

// attribSweep balaie la fenetre d'appariement (le degat est horodate a l'IMPACT) et logue, pour
// chaque W, le taux tir->degat contre le temoin +3 s. Le ratio a W = 1 s est le verdict de Lot 2.
func attribSweep(t *testing.T, shots []WeaponShot, dmgByResp map[uint64][]uint64) {
	t.Helper()
	fwdN := 0
	for _, s := range shots {
		if s.HasPair {
			fwdN++
		}
	}
	for _, w := range []uint64{250_000, 500_000, WeaponHitPairWindowUS, 2_000_000} {
		same, shift := 0, 0
		for _, s := range shots {
			if !s.HasPair {
				continue
			}
			if lot1mtNear(dmgByResp[s.Attacker], s.TimestampUS, w) {
				same++
			}
			if lot1mtNear(dmgByResp[s.Attacker], s.TimestampUS+attribOFF, w) {
				shift++
			}
		}
		floor := func(v int) int {
			if v < 1 {
				return 1
			}
			return v
		}
		t.Logf("   sweep W=%4dms : tir->degat %d/%d (%.1f %%) · temoin +3s %.1f %% -> %.1fx",
			w/1000, same, fwdN, lot1Pct(same, fwdN), lot1Pct(shift, fwdN),
			lot1Pct(same, fwdN)/lot1Pct(floor(shift), fwdN))
	}
}

// attribM2 mesure la PRECISION par arme via PairWeaponHits (touches/tirs, cle WeaponID). Les stats
// de production sont par (index tireur, WeaponID) ; on les agrege par arme pour le rapport.
func attribM2(t *testing.T, shots []WeaponShot, dmg []WeaponDamage) {
	t.Helper()
	stats := PairWeaponHits(shots, dmg, attribW, nil)
	statsWide := PairWeaponHits(shots, dmg, attribWideW, nil)
	type acc struct{ n, hits, wide int }
	by := map[uint64]*acc{}
	globN, globHit := 0, 0
	for _, s := range stats {
		a := by[s.WeaponID]
		if a == nil {
			a = &acc{}
			by[s.WeaponID] = a
		}
		a.n += s.ShotsPaired
		a.hits += s.Hits
		globN += s.ShotsPaired
		globHit += s.Hits
	}
	for _, s := range statsWide {
		if a := by[s.WeaponID]; a != nil {
			a.wide += s.Hits
		}
	}
	type row struct {
		wid           uint64
		n, hits, wide int
	}
	var rows []row
	for w, a := range by {
		rows = append(rows, row{w, a.n, a.hits, a.wide})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].n > rows[j].n })
	t.Logf("M2 PRECISION par arme (PairWeaponHits · >=1 degat apparie = 1 touche · cle WeaponID) :")
	t.Logf("   GLOBAL : %d tirs · %d touchent = %.1f %% (proxy film, a recaler sur le total API)",
		globN, globHit, lot1Pct(globHit, globN))
	shown := 0
	for _, r := range rows {
		if r.n < 5 {
			continue
		}
		t.Logf("   %-24s : %d tirs · %d touchent = %.1f %% (fenetre 2s : %.1f %%)", attribWeaponName(r.wid),
			r.n, r.hits, lot1Pct(r.hits, r.n), lot1Pct(r.wide, r.n))
		shown++
	}
	t.Logf("   (%d armes >=5 tirs affichees · %d WeaponID distincts au total)", shown, len(by))
}

// attribM3 mesure la DISTANCE tireur<->victime par arme via PairWeaponHits + un resolveur de
// position sonde (base position-resolvante, positions au ts du degat).
func attribM3(t *testing.T, dir string, shots []WeaponShot, dmg []WeaponDamage, n int) {
	t.Helper()
	wr := sondeWorldRange(t, dir)
	if wr == nil {
		t.Logf("M3 DISTANCE : bornes monde absentes — distances non calculables, mesure sautee")
		return
	}
	tr := sondeBipedTracks(t, dir, wr, n)
	_, base, _ := sondeBaseSweep(attribDamagesToSonde(dmg), tr)
	var distAll []float64
	dist := func(d WeaponDamage) (float64, bool) {
		if d.VictimIdx < 0 || d.ResponsibleIdx < 0 {
			return 0, false
		}
		pv, okv := sondeLookup(tr[uint32(base+d.VictimIdx)], d.TimestampUS, sondePosTolUS)
		pa, oka := sondeLookup(tr[uint32(base+d.ResponsibleIdx)], d.TimestampUS, sondePosTolUS)
		if !okv || !oka {
			return 0, false
		}
		v := sondeDist(pa, pv)
		distAll = append(distAll, v)
		return v, true
	}
	stats := PairWeaponHits(shots, dmg, attribW, dist)
	glob := make([]int, WeaponHitBucketCount())
	hits, resolved := 0, 0
	for _, s := range stats {
		for i, c := range s.DistBuckets {
			glob[i] += c
			resolved += c
		}
		hits += s.Hits
	}
	t.Logf("M3 DISTANCE (m) tireur<->victime · base positions = %d :", base)
	t.Logf("   touches %d · resolues (2 positions) %d · mediane globale %.1f · %s",
		hits, resolved, sondeMedian(distAll), sonde5Hist(glob))
	type row struct {
		name string
		b    []int
		n    int
	}
	var rows []row
	for _, s := range stats {
		nres := 0
		for _, c := range s.DistBuckets {
			nres += c
		}
		if nres < 4 {
			continue
		}
		rows = append(rows, row{attribWeaponName(s.WeaponID), s.DistBuckets, nres})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].n > rows[j].n })
	for _, r := range rows {
		t.Logf("   %-24s : n=%d · %s", r.name, r.n, sonde5Hist(r.b))
	}
	t.Logf("   SENS PHYSIQUE : longue portee (sniper) buckets LOINTAINS, courte (pompe/epee) PROCHES")
}

// attribM4 caracterise les degats SANS refs d'en-tete (ni attaquant ni victime resolubles).
func attribM4(t *testing.T, shots []WeaponShot, dmg []WeaponDamage) {
	t.Helper()
	var allTs []uint64
	for _, s := range shots {
		allTs = append(allTs, s.TimestampUS)
	}
	sort.Slice(allTs, func(a, b int) bool { return allTs[a] < allTs[b] })

	both, one, none, noneNeg := 0, 0, 0, 0
	srcNone := map[uint64]int{}
	var magNone []float64
	coincid := 0
	for _, e := range dmg {
		switch {
		case e.VictimIdx >= 0 && e.ResponsibleIdx >= 0:
			both++
		case e.VictimIdx >= 0 || e.ResponsibleIdx >= 0:
			one++
		default:
			none++
			if e.HasSource {
				srcNone[e.Source]++
			}
			magNone = append(magNone, e.MagClear)
			if e.Negative {
				noneNeg++
			}
			if lot1mtNear(allTs, e.TimestampUS, attribW) {
				coincid++
			}
		}
	}
	tot := len(dmg)
	t.Logf("M4 DEGATS SANS REFS (ni attaquant ni victime en en-tete) :")
	t.Logf("   deux refs %d (%.1f %%) · une ref %d (%.1f %%) · aucune ref %d (%.1f %%) du total %d",
		both, lot1Pct(both, tot), one, lot1Pct(one, tot), none, lot1Pct(none, tot), tot)
	if none == 0 {
		t.Logf("   aucun degat sans refs sur ce film")
		return
	}
	t.Logf("   sans-refs : magnitude mediane %.2f · soins (negatifs) %d/%d · %d tags source distincts · top %s",
		sondeMedian(magNone), noneNeg, none, len(srcNone), lot1TopU64(srcNone, 6))
	t.Logf("   sans-refs coincidant avec un tir ±%dms : %d/%d (%.1f %%) — bas => degats NON-ARME (chute/environnement/DoT)",
		attribW/1000, coincid, none, lot1Pct(coincid, none))
}

// attribM5 (secondaire) : le tag source de degat joint-il une arme (WeaponID) ?
func attribM5(t *testing.T, shots []WeaponShot, dmg []WeaponDamage) {
	t.Helper()
	widLo, widHi := map[uint64]bool{}, map[uint64]bool{}
	for _, s := range shots {
		if !s.HasPair {
			continue
		}
		widLo[s.WeaponID&0xFFFFFFFF] = true
		widHi[s.WeaponID>>32] = true
	}
	src := map[uint64]int{}
	for _, e := range dmg {
		if e.HasSource {
			src[e.Source]++
		}
	}
	interLo, interHi := 0, 0
	for s := range src {
		if widLo[s] {
			interLo++
		}
		if widHi[s] {
			interHi++
		}
	}
	ns := len(src)
	best := interLo
	if interHi > best {
		best = interHi
	}
	t.Logf("M5 (secondaire) tag source -> arme : %d tags source · intersection WeaponID bas %d (%.1f %%) · haut %d (%.1f %%)",
		ns, interLo, lot1Pct(interLo, ns), interHi, lot1Pct(interHi, ns))
	t.Logf("   joignable directement (> 50 %% des tags) : %s — sinon table requise ; NON BLOQUANT",
		lot1Verdict(ns > 0 && best*2 > ns))
}

// TestLot1AttribArmeTir produit les mesures d'attribution PAR LE TIR sur LOT1_TRAME_FILM, via le
// code de production (ScanFilmWeaponShots / ScanFilmWeaponDamages / PairWeaponHits).
func TestLot1AttribArmeTir(t *testing.T) {
	dir := os.Getenv(lot1TrameFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : instrument saute", lot1TrameFilmEnv)
	}
	release := LockProcessDecode()
	defer release()
	raw, err := ReadFilmChunk(dir, 0)
	if err != nil {
		t.Fatalf("chunk_00 illisible : %v", err)
	}
	reg, err := ParseRegistryChunk(raw)
	if err != nil {
		t.Fatalf("registre illisible : %v", err)
	}
	n := CountFilmChunks(dir)
	if n > deltaWitnessChunks {
		n = deltaWitnessChunks
	}
	t.Logf("== film %s · %d chunks balayes (attribution PAR LE TIR, code productionise) ==", filepath.Base(dir), n)

	shots, err := ScanFilmWeaponShots(dir, n)
	if err != nil {
		t.Fatalf("collecte des tirs : %v", err)
	}
	dmg, base, err := ScanFilmWeaponDamages(dir, reg, n)
	if err != nil {
		t.Fatalf("collecte des degats : %v", err)
	}
	t.Logf("collecte : %d tirs longs (0xD2 t36) · %d degats (0xC0 t0) · base bipede %d", len(shots), len(dmg), base)

	attribM1(t, shots, dmg)
	attribM2(t, shots, dmg)
	attribM3(t, dir, shots, dmg, n)
	attribM4(t, shots, dmg)
	attribM5(t, shots, dmg)
}
