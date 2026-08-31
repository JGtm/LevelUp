package filmdec

// lot1_attrib_arme_tir_research_test.go — ATTRIBUTION PAR LE TIR de la PRECISION et de la
// DISTANCE par arme, mesuree sur le film Theater.
//
// CADRAGE (correction d'une sonde precedente qui s'etait trompee de lien). L'arme est
// connue par le TIR : le record action_weapon_fire (0xD2, type 36, variante LONGUE) porte le
// WeaponID 64 bits a des offsets FIXES (fire_events.decodeFireEvent, prouve). On N'UTILISE PAS
// le tag source du damage_aftermath (projectile/effet, autre espace d'id, enfant de l'arme) :
// la sonde source-tag (lot1_sonde_precision_*) a montre qu'il ne joint PAS l'arme sans une
// table — c'est justement pourquoi on attribue TOUT par le tir.
//
// LE PONT TIR<->DEGAT est celui de lot1_modal_touche, reutilise tel quel : l'ATTAQUANT du tir
// (ref0, domaine 1, lu par lot1RefDom1 avant tout champ conteste) et le RESPONSABLE du degat
// (ref1, domaine 1, damage_aftermath, MEME encodage/espace) vivent dans le meme espace d'index
// brut. On apparie (meme index brut d'attaquant, |ts_degat - ts_tir| <= W), sans base. Le
// BLESSE d'un degat apparie est sa ref0 (lot1_degats_blesse), resolu en slot (base 512) pour
// la position -> distance tireur<->victime.
//
// SEUILS / TEMOINS ECRITS AVANT LA MESURE :
//	W    = 250 ms : fenetre d'appariement (meme valeur que lot1_modal_touche) ; un SWEEP
//	       250/500/1000/2000 ms accompagne la mesure (le degat est horodate a l'IMPACT, pas au tir).
//	OFF  = 3 s : decalage du TEMOIN. Le VERDICT porte sur le RATIO au temoin, pas sur le taux
//	       absolu (plafonne par la densite des degats dans le flux).
//	tol  = sondePosTolUS (120 ms) : ecart max evenement<->echantillon de position (distances).
//
// VERDICT VISE : via l'attribution PAR LE TIR, precision par arme (touches/tirs, cle WeaponID)
// et distance par arme sont-elles VIABLES ? Reserves : fiabilite du pont, fenetre d'impact,
// couverture par classe d'arme, echantillon film-seul a recaler sur le total API.
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
	attribW   = uint64(250_000)   // 250 ms : fenetre d'appariement
	attribOFF = uint64(3_000_000) // 3 s : temoin decale
)

// attribShot : un action_weapon_fire (0xD2 type 36) LONG horodate. att = attaquant (ref0 dom1
// brut, MEME espace que le responsable ref1 d'un damage_aftermath) ; wid = WeaponID 64 bits.
type attribShot struct {
	ts   uint64
	att  uint64
	wid  uint64
	fidx int
	has  bool // attaquant ET WeaponID lisibles
}

// dmgRef : renvoi vers un damage_aftermath (index dans la tranche + horodatage).
type dmgRef struct {
	ts uint64
	di int
}

// attribIndex : les index temporels pre-tries pour l'appariement et les temoins.
type attribIndex struct {
	dmgTsByResp  map[uint64][]uint64 // degats indexes par responsable (ref1) -> ts tries
	shotTsByAtt  map[uint64][]uint64 // tirs indexes par attaquant (ref0) -> ts tries
	dmgRefByResp map[uint64][]dmgRef // degats (ref + index) par responsable, pour le plus proche
	allShotTs    []uint64            // tous les ts de tir, tries (coincidence M4)
}

// attribWeaponName nomme une arme par son WeaponID via la table statique de analysis
// (metadata weapon_labels) ; a defaut, l'hexa 64 bits.
func attribWeaponName(wid uint64) string {
	if n, ok := analysis.WeaponIDToName[wid]; ok {
		return n
	}
	return fmt.Sprintf("wid#%016x", wid)
}

// attribCollectShots decode les tirs LONGS 0xD2 : attaquant (lot1RefDom1, comme modal_touche)
// et WeaponID/index joueur (decodeFireEvent, offsets fixes) — un seul decodeur par champ.
func attribCollectShots(t *testing.T, dir string, n int) []attribShot {
	t.Helper()
	var out []attribShot
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			t.Fatalf("chunk_%02d illisible : %v", c, err)
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeDelta || pk.Size < 4 {
				continue
			}
			pay := pk.Payload(data)
			if pay[0] != 0xD2 { // type 36 variante LONGUE (porte l'arme)
				continue
			}
			s := attribShot{ts: pk.TimestampUS}
			br := NewBitReader(pay)
			br.Skip(2)
			if br.ReadBits(7) != 36 {
				continue
			}
			att, okA := lot1RefDom1(br) // ref0 = attaquant (dom1)
			fe, okF := decodeFireEvent(pay)
			if okA && okF {
				s.att, s.wid, s.fidx, s.has = att, fe.WeaponID, fe.FilmIndex, true
			}
			out = append(out, s)
		}
	}
	return out
}

// attribBuildIndex construit les index temporels a partir des tirs et des degats.
func attribBuildIndex(shots []attribShot, dmg []sondeDmgEvt) attribIndex {
	var dmgTs, dmgResp []uint64
	byResp := map[uint64][]dmgRef{}
	for di, e := range dmg {
		if e.idx1 < 0 {
			continue
		}
		dmgTs = append(dmgTs, e.ts)
		dmgResp = append(dmgResp, uint64(e.idx1))
		byResp[uint64(e.idx1)] = append(byResp[uint64(e.idx1)], dmgRef{ts: e.ts, di: di})
	}
	for k := range byResp {
		sort.Slice(byResp[k], func(a, b int) bool { return byResp[k][a].ts < byResp[k][b].ts })
	}
	var shotTs, shotAtt, allTs []uint64
	for _, s := range shots {
		if !s.has {
			continue
		}
		shotTs = append(shotTs, s.ts)
		shotAtt = append(shotAtt, s.att)
		allTs = append(allTs, s.ts)
	}
	sort.Slice(allTs, func(a, b int) bool { return allTs[a] < allTs[b] })
	return attribIndex{
		dmgTsByResp:  lot1mtIndexByKey(dmgTs, dmgResp),
		shotTsByAtt:  lot1mtIndexByKey(shotTs, shotAtt),
		dmgRefByResp: byResp,
		allShotTs:    allTs,
	}
}

// attribNearestDmg rend l'index du damage_aftermath du responsable resp le plus proche de T
// dans [T-W, T+W].
func attribNearestDmg(refs []dmgRef, T, W uint64) (int, bool) {
	i := sort.Search(len(refs), func(i int) bool { return refs[i].ts >= T })
	best, ok := -1, false
	bd := ^uint64(0)
	consider := func(j int) {
		if j < 0 || j >= len(refs) {
			return
		}
		d := T - refs[j].ts
		if refs[j].ts > T {
			d = refs[j].ts - T
		}
		if d <= W && d < bd {
			best, ok, bd = refs[j].di, true, d
		}
	}
	consider(i - 1)
	consider(i)
	return best, ok
}

// attribM1 mesure la FIABILITE du lien tir<->degat (avant, arriere) contre le temoin decale.
func attribM1(t *testing.T, shots []attribShot, dmg []sondeDmgEvt, idx attribIndex) {
	t.Helper()
	fwdN, fwdSame, fwdShift := 0, 0, 0
	for _, s := range shots {
		if !s.has {
			continue
		}
		fwdN++
		if lot1mtNear(idx.dmgTsByResp[s.att], s.ts, attribW) {
			fwdSame++
		}
		if lot1mtNear(idx.dmgTsByResp[s.att], s.ts+attribOFF, attribW) {
			fwdShift++
		}
	}
	revN, revSame, revShift := 0, 0, 0
	for _, e := range dmg {
		if e.idx1 < 0 {
			continue
		}
		revN++
		key := uint64(e.idx1)
		if lot1mtNear(idx.shotTsByAtt[key], e.ts, attribW) {
			revSame++
		}
		if lot1mtNear(idx.shotTsByAtt[key], e.ts+attribOFF, attribW) {
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
	// SWEEP DE FENETRE : un projectile lent (roquette, empaleur) touche APRES un delai de vol.
	// Si l'elargissement de W recupere des tirs bien AU-DELA de la croissance du temoin, la
	// non-capture de certaines armes est un probleme de FENETRE (vol) ; si le temoin croit
	// autant, c'est de la non-emission de damage_aftermath pour ces armes.
	for _, w := range []uint64{250_000, 500_000, 1_000_000, 2_000_000} {
		same, shift := 0, 0
		for _, s := range shots {
			if !s.has {
				continue
			}
			if lot1mtNear(idx.dmgTsByResp[s.att], s.ts, w) {
				same++
			}
			if lot1mtNear(idx.dmgTsByResp[s.att], s.ts+attribOFF, w) {
				shift++
			}
		}
		t.Logf("   sweep W=%4dms : tir->degat meme attaquant %d/%d (%.1f %%) · temoin +3s %.1f %%",
			w/1000, same, fwdN, lot1Pct(same, fwdN), lot1Pct(shift, fwdN))
	}
}

// attribWTally : compteurs de precision d'une arme. hitsWide = touches a fenetre elargie (2 s)
// pour reveler les projectiles lents (roquettes) dont le degat arrive apres le vol.
type attribWTally struct {
	n, hits, hitsWide int
}

// attribWideW : fenetre elargie pour la colonne "vol" de M2.
const attribWideW = uint64(2_000_000)

// attribM2 mesure la PRECISION par arme : tirs qui touchent (>=1 degat apparie) / tirs, cle
// WeaponID. Rend le tally par arme (pour reference).
func attribM2(t *testing.T, shots []attribShot, idx attribIndex) map[uint64]*attribWTally {
	t.Helper()
	by := map[uint64]*attribWTally{}
	globN, globHit := 0, 0
	for _, s := range shots {
		if !s.has {
			continue
		}
		ta := by[s.wid]
		if ta == nil {
			ta = &attribWTally{}
			by[s.wid] = ta
		}
		ta.n++
		globN++
		if lot1mtNear(idx.dmgTsByResp[s.att], s.ts, attribW) {
			ta.hits++
			globHit++
		}
		if lot1mtNear(idx.dmgTsByResp[s.att], s.ts, attribWideW) {
			ta.hitsWide++
		}
	}
	type row struct {
		wid              uint64
		n, hits, hitWide int
	}
	var rows []row
	for w, ta := range by {
		rows = append(rows, row{w, ta.n, ta.hits, ta.hitsWide})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].n > rows[j].n })
	t.Logf("M2 PRECISION par arme (>=1 degat apparie = 1 touche · cle WeaponID du TIR) :")
	t.Logf("   GLOBAL : %d tirs · %d touchent = %.1f %% (proxy film, a recaler sur le total API)",
		globN, globHit, lot1Pct(globHit, globN))
	shown := 0
	for _, r := range rows {
		if r.n < 5 { // armes a effectif trop faible : bruit
			continue
		}
		t.Logf("   %-24s : %d tirs · %d touchent = %.1f %% (fenetre 2s : %.1f %%)", attribWeaponName(r.wid),
			r.n, r.hits, lot1Pct(r.hits, r.n), lot1Pct(r.hitWide, r.n))
		shown++
	}
	t.Logf("   (%d armes >=5 tirs affichees · %d WeaponID distincts au total)", shown, len(by))
	return by
}

// attribM3 mesure la DISTANCE tireur<->victime par arme, sur les tirs qui touchent (arme = celle
// du tir ; victime = ref0 du degat apparie ; positions au ts du degat).
func attribM3(t *testing.T, dir string, shots []attribShot, dmg []sondeDmgEvt, idx attribIndex, n int) {
	t.Helper()
	wr := sondeWorldRange(t, dir)
	if wr == nil {
		t.Logf("M3 DISTANCE : bornes monde absentes — distances non calculables, mesure sautee")
		return
	}
	tr := sondeBipedTracks(t, dir, wr, n)
	_, base, _ := sondeBaseSweep(dmg, tr)
	distByWid := map[uint64][]float64{}
	var distAll []float64
	bucket := make([]int, len(sondeDistEdges)+1)
	resolved, hitShots := 0, 0
	for _, s := range shots {
		if !s.has {
			continue
		}
		di, ok := attribNearestDmg(idx.dmgRefByResp[s.att], s.ts, attribW)
		if !ok {
			continue
		}
		hitShots++
		e := dmg[di]
		if e.idx0 < 0 || e.idx1 < 0 {
			continue
		}
		pv, okv := sondeLookup(tr[uint32(base+e.idx0)], e.ts, sondePosTolUS)
		pa, oka := sondeLookup(tr[uint32(base+e.idx1)], e.ts, sondePosTolUS)
		if !okv || !oka {
			continue
		}
		resolved++
		d := sondeDist(pa, pv)
		distAll = append(distAll, d)
		distByWid[s.wid] = append(distByWid[s.wid], d)
		bucket[sondeBucket(d)]++
	}
	t.Logf("M3 DISTANCE (m) tireur<->victime · base positions = %d :", base)
	t.Logf("   tirs touchant %d · resolus (2 positions) %d · mediane globale %.1f · %s",
		hitShots, resolved, sondeMedian(distAll), sonde5Hist(bucket))
	type row struct {
		wid    uint64
		ds     []float64
		median float64
	}
	var rows []row
	for w, ds := range distByWid {
		rows = append(rows, row{w, ds, sondeMedian(ds)})
	}
	sort.Slice(rows, func(i, j int) bool { return len(rows[i].ds) > len(rows[j].ds) })
	for _, r := range rows {
		if len(r.ds) < 4 {
			continue
		}
		t.Logf("   %-24s : n=%d · mediane %.1f m · min %.1f max %.1f", attribWeaponName(r.wid),
			len(r.ds), r.median, attribMin(r.ds), attribMax(r.ds))
	}
	t.Logf("   SENS PHYSIQUE : longue portee (sniper) doit mesurer LONG, courte (pompe/epee) COURT — lecture qualitative ci-dessus")
}

// attribM4 caracterise les degats SANS refs d'en-tete (ni attaquant ni victime resolubles).
func attribM4(t *testing.T, shots []attribShot, dmg []sondeDmgEvt) {
	t.Helper()
	var allTs []uint64
	for _, s := range shots {
		allTs = append(allTs, s.ts)
	}
	sort.Slice(allTs, func(a, b int) bool { return allTs[a] < allTs[b] })

	both, one, none, noneNeg := 0, 0, 0, 0
	srcNone := map[uint64]int{}
	var magNone []float64
	coincid := 0
	for _, e := range dmg {
		switch {
		case e.idx0 >= 0 && e.idx1 >= 0:
			both++
		case e.idx0 >= 0 || e.idx1 >= 0:
			one++
		default:
			none++
			if e.hasSrc {
				srcNone[e.src]++
			}
			magNone = append(magNone, e.magClear)
			if e.neg {
				noneNeg++
			}
			if lot1mtNear(allTs, e.ts, attribW) {
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
	t.Logf("   sans-refs coincidant avec un tir quelconque ±%dms : %d/%d (%.1f %%) — bas => degats NON-ARME (chute/environnement/DoT), exclus a JUSTE TITRE",
		attribW/1000, coincid, none, lot1Pct(coincid, none))
}

// attribM5 (secondaire) jette un oeil : le tag source de degat joint-il une arme (WeaponID) ?
func attribM5(t *testing.T, shots []attribShot, dmg []sondeDmgEvt) {
	t.Helper()
	widLo, widHi := map[uint64]bool{}, map[uint64]bool{}
	for _, s := range shots {
		if !s.has {
			continue
		}
		widLo[s.wid&0xFFFFFFFF] = true
		widHi[s.wid>>32] = true
	}
	src := map[uint64]int{}
	for _, e := range dmg {
		if e.hasSrc {
			src[e.src]++
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
	t.Logf("   joignable directement (> 50 %% des tags) : %s — sinon table requise ; NON BLOQUANT (chemin principal = le tir)",
		lot1Verdict(ns > 0 && best*2 > ns))
}

// attribMin/attribMax : bornes d'un echantillon non vide.
func attribMin(xs []float64) float64 {
	m := xs[0]
	for _, x := range xs[1:] {
		if x < m {
			m = x
		}
	}
	return m
}

func attribMax(xs []float64) float64 {
	m := xs[0]
	for _, x := range xs[1:] {
		if x > m {
			m = x
		}
	}
	return m
}

// TestLot1AttribArmeTir produit les 5 mesures d'attribution PAR LE TIR sur LOT1_TRAME_FILM.
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
	t.Logf("== film %s · %d chunks balayes (attribution PAR LE TIR) ==", filepath.Base(dir), n)

	shots := attribCollectShots(t, dir, n)
	dmg, _ := sondeScanDamage(t, dir, reg, n)
	t.Logf("collecte : %d tirs longs (0xD2 t36) · %d degats (0xC0 t0)", len(shots), len(dmg))

	idx := attribBuildIndex(shots, dmg)
	attribM1(t, shots, dmg, idx)
	attribM2(t, shots, idx)
	attribM3(t, dir, shots, dmg, idx, n)
	attribM4(t, shots, dmg)
	attribM5(t, shots, dmg)
}
