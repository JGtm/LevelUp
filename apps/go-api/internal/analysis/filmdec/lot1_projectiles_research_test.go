package filmdec

// lot1_projectiles_research_test.go — LOT 1 : les EVENEMENTS PROJECTILE du film, voie de
// recuperation des armes A PROJECTILE (M41 SPNKr, Hydra, Skewer, Ravager, Shock Rifle,
// Mangler, Stalker, Bulldog...) qui n'emettent PAS de damage_aftermath (0 % de touches en
// type 0) et dont le type 1 (damage_section_response) a ete refute (pas d'attaquant).
//
// DEUX EVENEMENTS, CONFIRMES PAR LEURS NOMS DANS L'EXE (chaines a 0x143c97858
// "projectile_detonate" et 0x143c97838 "projectile_impact_effect", descripteurs
// 0x143d0bae8 et 0x143d0bb80) :
//   - projectile_detonate       : octet 0xC2, type 5 (Skip(2)+R(7)==5)
//   - projectile_impact_effect  : octet 0xC3, types 6 ET 7 (R(7) in {6,7})
//
// GRAMMAIRE LUE DANS L'EXE (dispatcher generique FUN_14080a9d4 : R(7) type, PUIS boucle de 3
// slots de reference gardes chacun par 1 bit, PUIS lecteur de charge vtable+0x68) :
//
//   EN-TETE (commun aux deux) : le commutateur de domaine vtable+0x58 (0x1408096ec) rend
//   DOMAINE 5 pour le slot 0 et ASSERTE (INT3) pour les slots 1 et 2 -> il n'y a qu'UNE
//   reference d'entite (ref0, domaine 5, largeur 8), les deux slots suivants ont toujours leur
//   bit de presence a 0. C'est la difference STRUCTURELLE avec damage_aftermath (dom1/dom1/dom7,
//   DEUX entites blesse+responsable) : un evenement projectile ne porte qu'UNE entite en en-tete.
//
//   CHARGE projectile_detonate (FUN_1408096f8) : R(6) ; R(1) porte ; si porte==0 :
//     [R(1) g ; si g : R(32)] puis "variant-name" R(32) ; ... (direction R(19), dequant R(5),
//     R(9), drapeaux) — le reste n'est pas necessaire a l'attribution et n'est pas decode ici.
//   CHARGE projectile_impact_effect (FUN_1410f03b4) : R(1) porte ; si porte==0 :
//     [R(1) g ; si g : R(32)] puis "variant-name" R(32) ; ...
//
//   variant-name (FUN_14080dec4, R(32)) est le tag de VARIANTE du projectile — le MEME espace
//   que le variant_name du tir type 36 (lot1_tirs / sondeScanFireArme). C'est la cle qui
//   identifie l'arme SANS remonter au tireur.
//
// L'ORACLE de cadrage n'est PAS la trame de records (charge trop longue/conditionnelle pour un
// decodage integral fiable), c'est le TAG connu : au bon offset, variant-name tombe sur un tag
// de variante DEJA VU dans les tirs ; a +3 bits (temoin), il tombe dans le vide. On mesure les
// deux. L'attribution, elle, se teste par COINCIDENCE temporelle evenement<->tir lourd, temoin
// decale — un lien qui ne depend d'aucun bit de charge.
//
// Garde LOT1_TRAME_FILM. Un film par process, verrou pris, lecture seule, borne 12 chunks.
// Lancer une fois par film (000d5950, 01e1f945, 00502e52).

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis"
)

// TestLot1Projectiles : grammaire (oracle de tag), en-tete (ref0), et attribution des armes
// lourdes par les evenements projectile.
func TestLot1Projectiles(t *testing.T) {
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
	t.Logf("== film %s · %d chunks (evenements projectile) ==", filepath.Base(dir), n)

	width := projCalibrateWidth(t, dir, n)
	evs, base := projScan(t, dir, reg, n, width)
	fires := projCollectFireVariants(t, dir, n)
	projReportCounts(t, evs)
	fireVar := projReportGrammar(t, evs, fires)
	projReportHeader(t, evs, base)
	projReportAttribution(t, evs, fires, fireVar)
}

// projReportCounts : M1 — comptages et sanity de l'en-tete (slots 1/2 doivent etre ~absents).
func projReportCounts(t *testing.T, evs []projEvt) {
	t.Helper()
	var det, imp, has1, has2, refPres, varPres int
	for _, e := range evs {
		if e.impact {
			imp++
		} else {
			det++
		}
		if e.has1 {
			has1++
		}
		if e.has2 {
			has2++
		}
		if e.ref0 >= 0 {
			refPres++
		}
		if e.hasVar {
			varPres++
		}
	}
	tot := len(evs)
	t.Logf("M1 COMPTAGES : detonate %d · impact_effect %d · total %d", det, imp, tot)
	t.Logf("   en-tete : ref0 (dom5) presente %d/%d (%.1f %%) · variant-name presente %d/%d (%.1f %%)",
		refPres, tot, lot1Pct(refPres, tot), varPres, tot, lot1Pct(varPres, tot))
	t.Logf("   SANITY en-tete a UNE ref : slot1 present %d · slot2 present %d (doivent etre ~0 ; sinon la grammaire d'en-tete est fausse)",
		has1, has2)
}

// projReportGrammar : M2 — ORACLE DE TAG. variant-name doit tomber sur un tag de variante deja
// vu dans les tirs ; a +3 bits (temoin), non. Rend l'ensemble des variantes de tir (pour M4).
func projReportGrammar(t *testing.T, evs []projEvt, fires []projFireVariant) map[uint64]uint64 {
	t.Helper()
	fireVar := map[uint64]uint64{} // variant -> WeaponID (pour nommer)
	for _, f := range fires {
		if f.has {
			fireVar[f.variant] = f.wid
		}
	}
	var nVar, known, knownWitness int
	dist := map[uint64]int{}
	for _, e := range evs {
		if !e.hasVar {
			continue
		}
		nVar++
		dist[e.variant]++
		if _, ok := fireVar[e.variant]; ok {
			known++
		}
		if _, ok := fireVar[e.variant3]; ok {
			knownWitness++
		}
	}
	t.Logf("M2 variant-name (charge, presente %d fois) : %d valeurs distinctes (%.1f %% de distinctes) · %s",
		nVar, len(dist), lot1Pct(len(dist), nVar), lot1TopU64(dist, 8))
	t.Logf("   ∈ espace des variantes de TIR (arme) : %d/%d (%.1f %%) · TEMOIN +3 bits : %d/%d — 0 %% attendu si l'espace differe",
		known, nVar, lot1Pct(known, nVar), knownWitness, nVar)
	t.Logf("   LECTURE : quasi-100 %% de distinctes => PAS un tag categoriel d'arme exploitable (soit espace")
	t.Logf("   projectile sans catalogue, soit champ mal cadre) ; l'evenement ne nomme PAS l'arme utilement.")
	return fireVar
}

// projReportHeader : M3 — la ref0 (domaine 5) atterrit-elle sur un bipede (owner/victime) ou
// dans un autre espace (entite projectile) ?
func projReportHeader(t *testing.T, evs []projEvt, base int) {
	t.Helper()
	mn, mx := uint64(1<<20), uint64(0)
	dist := map[uint64]int{}
	present := 0
	for _, e := range evs {
		if e.ref0 < 0 {
			continue
		}
		present++
		v := uint64(e.ref0)
		dist[v]++
		if v < mn {
			mn = v
		}
		if v > mx {
			mx = v
		}
	}
	if present == 0 {
		t.Logf("M3 EN-TETE ref0 : aucune reference presente sur ce film")
		return
	}
	t.Logf("M3 EN-TETE ref0 (domaine 5, largeur 8) : %d presentes · %d distinctes · min=%d max=%d",
		present, len(dist), mn, mx)
	t.Logf("   base lands-on-biped (argmax structurel) = %d · top valeurs : %s", base, lot1TopU64(dist, 8))
	t.Logf("   LECTURE : valeurs petites/peu nombreuses (~<=16) => index de joueur/owner ; nombreuses et")
	t.Logf("   etalees => pool d'entites projectile (non resoluble en tireur hors table).")
}

// projReportAttribution : M4 — les armes LOURDES touchent-elles via les evenements projectile ?
// (a) coincidence temporelle evenement<->tir lourd, temoin decale ; (b) attribution de l'arme
// par la variant-name de l'evenement.
func projReportAttribution(t *testing.T, evs []projEvt, fires []projFireVariant, fireVar map[uint64]uint64) {
	t.Helper()
	var allTs []uint64
	for _, e := range evs {
		allTs = append(allTs, e.ts)
	}
	sort.Slice(allTs, func(a, b int) bool { return allTs[a] < allTs[b] })
	W, OFF, Wide := attribW, attribOFF, attribWideW

	// (a) coincidence par arme, focalisee sur les lourdes.
	by := map[uint64]*projTally{}
	globN, globNear, globShift := 0, 0, 0
	for _, f := range fires {
		if !f.has {
			continue
		}
		ta := by[f.wid]
		if ta == nil {
			ta = &projTally{}
			by[f.wid] = ta
		}
		ta.n++
		globN++
		if lot1mtNear(allTs, f.ts, W) {
			ta.near++
			globNear++
		}
		if lot1mtNear(allTs, f.ts, Wide) {
			ta.nearWide++
		}
		if lot1mtNear(allTs, f.ts+OFF, W) {
			ta.shift++
			globShift++
		}
	}
	t.Logf("M4 COINCIDENCE tir -> evenement projectile (±%dms · temoin +%ds · fenetre large %ds) :",
		W/1000, OFF/1_000_000, Wide/1_000_000)
	t.Logf("   GLOBAL (tous tirs, n=%d) : proche %.1f %% · temoin %.1f %%", globN,
		lot1Pct(globNear, globN), lot1Pct(globShift, globN))
	projAttribRows(t, by)

	// (b) attribution de l'arme par la variant-name des evenements.
	projVariantWeapons(t, evs, fireVar)
}

// projTally : compteurs de coincidence d'une arme (fenetre serree, large, temoin decale).
type projTally struct{ n, near, nearWide, shift int }

// projAttribRows publie la coincidence par arme, lourdes d'abord.
func projAttribRows(t *testing.T, by map[uint64]*projTally) {
	t.Helper()
	type row struct {
		wid                     uint64
		n, near, nearWide, shft int
		heavy                   bool
	}
	var rows []row
	hN, hNear, hShift := 0, 0, 0
	for w, ta := range by {
		heavy := lot1IsHeavy(attribWeaponName(w))
		rows = append(rows, row{w, ta.n, ta.near, ta.nearWide, ta.shift, heavy})
		if heavy {
			hN += ta.n
			hNear += ta.near
			hShift += ta.shift
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].heavy != rows[j].heavy {
			return rows[i].heavy
		}
		return rows[i].n > rows[j].n
	})
	for _, r := range rows {
		if r.n < 5 {
			continue
		}
		mark := "     "
		if r.heavy {
			mark = "LOURD"
		}
		t.Logf("   [%s] %-24s : %d tirs · proche %.1f %% · large %.1f %% · temoin %.1f %%", mark,
			attribWeaponName(r.wid), r.n, lot1Pct(r.near, r.n), lot1Pct(r.nearWide, r.n), lot1Pct(r.shft, r.n))
	}
	t.Logf("M4 BILAN LOURDES : %d tirs · proche %.1f %% · temoin %.1f %%", hN,
		lot1Pct(hNear, hN), lot1Pct(hShift, hN))
	t.Logf("   VERDICT lien reel armes lourdes (proche >= 1.5x temoin ET >= 30 %%) : %s",
		lot1Verdict(hN >= 20 && lot1Pct(hNear, hN) >= 30 &&
			float64(hNear) >= 1.5*float64(hShift+1)))
}

// projVariantWeapons : l'arme portee par la variant-name de l'evenement (attribution SANS
// tireur), ventilee, lourdes signalees.
func projVariantWeapons(t *testing.T, evs []projEvt, fireVar map[uint64]uint64) {
	t.Helper()
	byWeap := map[uint64]int{}
	named, unnamed := 0, 0
	for _, e := range evs {
		if !e.hasVar {
			continue
		}
		if wid, ok := fireVar[e.variant]; ok {
			byWeap[wid]++
			named++
		} else {
			unnamed++
		}
	}
	t.Logf("M5 ARME PAR variant-name (attribution directe, sans tireur) : %d evenements nommes · %d non nommes",
		named, unnamed)
	type row struct {
		wid   uint64
		n     int
		heavy bool
	}
	var rows []row
	for w, c := range byWeap {
		rows = append(rows, row{w, c, lot1IsHeavy(attribWeaponName(w))})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].heavy != rows[j].heavy {
			return rows[i].heavy
		}
		return rows[i].n > rows[j].n
	})
	for _, r := range rows {
		mark := "     "
		if r.heavy {
			mark = "LOURD"
		}
		t.Logf("   [%s] %-24s : %d evenements projectile", mark, projWeaponLabel(r.wid), r.n)
	}
}

// projWeaponLabel nomme une arme par WeaponID, ou l'hexa a defaut.
func projWeaponLabel(wid uint64) string {
	if n, ok := analysis.WeaponIDToName[wid]; ok {
		return n
	}
	return attribWeaponName(wid)
}
