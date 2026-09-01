package filmdec

// lot1_sonde_precision_research_test.go — SONDE DE FIABILITE pour une future feature
// « precision par arme et par distance » calculee depuis le film Theater.
//
// CONTEXTE (decide avec l'utilisateur) : l'API du match donne la precision GLOBALE (sans
// distinction). Le film doit apporter la VENTILATION (par arme, par distance). Le film
// SOUS-REPLIQUE les degats : on n'utilisera donc le film que pour la FORME (distributions
// relatives) recalee sur le total API. LA QUESTION CENTRALE : cette forme est-elle
// REPRESENTATIVE, c.-a-d. la sous-replication est-elle NON biaisee par arme / par distance ?
//
// Cet instrument NE REIMPLEMENTE RIEN : il compose les decodeurs eprouves du paquet —
//   - tirs : ScanFilmFireEvents/decodeFireEvent (WeaponID 64 bits, offsets fixes) ET la
//     grammaire de reference du type 36 (variant_name R(32), l'arme vettee de lot1_tirs) ;
//   - degats : lot1DecodeDamageAftermath (source R(32), victime), refs d'en-tete dom-1
//     ref0=blesse / ref1=responsable (lot1_degats_blesse), base par lot1chIsBiped ;
//   - positions : ScanFilmBipedPositions (positions monde + horodatage).
//
// SEUILS / TEMOINS ECRITS AVANT LA MESURE :
//
//	tol     = 120 ms (= replay/shots.go shotPosToleranceUS) : ecart max evenement <-> echantillon
//	          de position. Une position est retenue si |ts - T| <= tol.
//	M1 plausibilite : nb d'index tireurs distincts <= ~16 (arene 8 joueurs, respawns compris) ;
//	          arme (variant_name) CATEGORIELLE = peu de distinctes (<< nb d'evenements).
//	M3 join : deux cles joignent si l'INTERSECTION des ensembles d'id couvre une part
//	          notable (> 50 %) des tags de degat ; sinon il faut une table.
//	M6 biais arme : la resolvabilite (les deux positions presentes) est jugee UNIFORME si
//	          l'ecart max-min de taux entre les tags de degat frequents reste < 20 points.
//	M6 biais distance : les buckets de distance des degats resolus sont SENSES si les armes a
//	          courte portee mesurent court et les longue portee mesurent long (qualitatif).
//
// Garde LOT1_TRAME_FILM. Un film par process, verrou pris, lecture seule, borne a
// deltaWitnessChunks. Lancer une fois par film (000d5950, 01e1f945, 00502e52).

import (
	"math"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// sondePosTolUS : tolerance temporelle evenement<->position. MEME valeur que
// replay/shots.go shotPosToleranceUS (120 ms) ; recopiee ici pour rester DANS filmdec
// (pas d'import de internal/analysis/replay depuis un instrument de filmdec).
const sondePosTolUS = uint64(120_000)

// sondeMapEnv force la carte quand la signature de largeurs est ambigue.
const sondeMapEnv = "LOT1_SONDE_MAP"

// sondeDistEdges : bornes (metres) des buckets de distance attaquant<->victime. PRODUCTIONISE :
// alias des bornes de weapon_hits.go — une seule source pour l'instrument et la table.
var sondeDistEdges = WeaponHitDistanceEdges

// sondeDmgEvt : un evenement damage_aftermath horodate, refs d'en-tete non resolues, source.
// magClear/magRaw sont additifs (peuples par sondeScanDamage, lus par l'instrument
// d'attribution par le tir) — l'ancienne sonde source-tag les ignore.
type sondeDmgEvt struct {
	ts         uint64
	idx0, idx1 int // ref0=blesse(victime), ref1=responsable(attaquant) ; -1 si absente
	src        uint64
	hasSrc     bool
	neg        bool
	magClear   float64 // magnitude en clair (signee ; soin si negative)
	magRaw     uint64  // code magnitude R(5)
}

// sondeSample : une position monde horodatee.
type sondeSample struct {
	ts      uint64
	x, y, z float32
}

// TestLot1SondePrecisionDistance produit les 7 mesures de la sonde sur le film LOT1_TRAME_FILM.
// Decodeurs et utilitaires : lot1_sonde_precision_helpers_test.go.
func TestLot1SondePrecisionDistance(t *testing.T) {
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
	t.Logf("== film %s · %d chunks balayes ==", filepath.Base(dir), n)

	// M1 — TIRS PAR ARME.
	armes, court, long, attaquants := sondeScanFireArme(t, dir, n)
	fires, ferr := ScanFilmFireEvents(dir)
	sonde1Fire(t, armes, court, long, attaquants, fires, ferr)

	// M2 — DEGATS PAR ARME (source).
	dmg, base := sondeScanDamage(t, dir, reg, n)
	srcCount := map[uint64]int{}
	for _, e := range dmg {
		if e.hasSrc {
			srcCount[e.src]++
		}
	}
	t.Logf("M2 DEGATS : %d evenements damage_aftermath · source presente %d · %d tags source distincts",
		len(dmg), len(dmg)-sonde0(dmg), len(srcCount))
	t.Logf("   top sources : %s", lot1TopU64(srcCount, 8))

	// M3 — JOIN des cles d'arme.
	sonde3Join(t, srcCount, armes, fires)

	// M4/M5/M6 — RESOLVABILITE, DISTANCE, BIAIS.
	wr := sondeWorldRange(t, dir)
	sonde456(t, dir, dmg, srcCount, base, wr, n)

	// M7 — PROXY DE PRECISION.
	sonde7Proxy(t, long, len(dmg))
}

// sonde0 compte les evenements sans source.
func sonde0(dmg []sondeDmgEvt) int {
	n := 0
	for _, e := range dmg {
		if !e.hasSrc {
			n++
		}
	}
	return n
}

// sonde1Fire publie la mesure 1 (tirs par arme + plausibilite + cross-check WeaponID).
func sonde1Fire(t *testing.T, armes map[uint64]int, court, long int, attaquants map[uint64]int, fires []FireEvent, ferr error) {
	t.Helper()
	t.Logf("M1 TIRS (type 36, grammaire de reference) : %d records longs · %d courts", long, court)
	t.Logf("   arme (variant_name) : %d distinctes / %d records longs (%.1f %%) — CATEGORIEL si << 100 %% ; top %s",
		len(armes), long, lot1Pct(len(armes), long), lot1TopU64(armes, 8))
	// ref0 du type 36 est un HANDLE d'entite (idx 9/13 bits), pas un index de joueur 0..7 :
	// son nombre de distinctes (biped x respawns) n'est pas la plausibilite « 8 joueurs ».
	t.Logf("   handles d'entite tireur (ref0, informational) : %d distincts", len(attaquants))
	if ferr != nil {
		t.Logf("   cross-check WeaponID (offsets fixes) indisponible : %v", ferr)
		return
	}
	wid := map[uint64]int{}
	fidx := map[int]int{}
	for _, f := range fires {
		wid[f.WeaponID]++
		fidx[f.FilmIndex]++
	}
	t.Logf("   cross-check ScanFilmFireEvents (WeaponID 64 bits, offsets FIXES) : %d events · %d WeaponID distincts (%.1f %%) · %d index joueur (FilmIndex)",
		len(fires), len(wid), lot1Pct(len(wid), len(fires)), len(fidx))
	// LA plausibilite « 8 joueurs » : l'index de JOUEUR (FilmIndex, 0..7 en arene), pas le handle.
	t.Logf("   plausibilite tireurs (index joueur <= 16 pour 8 joueurs + respawns) : %s", lot1Verdict(len(fidx) > 0 && len(fidx) <= 16))
}

// sonde3Join teste si le tag source (32 bits) joint le variant_name / WeaponID (halves).
func sonde3Join(t *testing.T, srcCount, armes map[uint64]int, fires []FireEvent) {
	t.Helper()
	widLo, widHi := map[uint64]bool{}, map[uint64]bool{}
	for _, f := range fires {
		widLo[f.WeaponID&0xFFFFFFFF] = true
		widHi[f.WeaponID>>32] = true
	}
	interVariant, interLo, interHi := 0, 0, 0
	for s := range srcCount {
		if armes[s] > 0 {
			interVariant++
		}
		if widLo[s] {
			interLo++
		}
		if widHi[s] {
			interHi++
		}
	}
	ns := len(srcCount)
	t.Logf("M3 JOIN — tags source de degat : %d · intersection avec variant_name (type36) : %d (%.1f %%)",
		ns, interVariant, lot1Pct(interVariant, ns))
	t.Logf("   intersection avec WeaponID moitie basse : %d (%.1f %%) · moitie haute : %d (%.1f %%)",
		interLo, lot1Pct(interLo, ns), interHi, lot1Pct(interHi, ns))
	best := interVariant
	if interLo > best {
		best = interLo
	}
	if interHi > best {
		best = interHi
	}
	t.Logf("   VERDICT joignable directement (intersection > 50 %% des tags source) : %s — sinon une TABLE (tag de degat -> arme) est requise",
		lot1Verdict(ns > 0 && best*2 > ns))
}

// sonde456 publie resolvabilite (M4), distance par arme (M5) et le test de biais (M6).
func sonde456(t *testing.T, dir string, dmg []sondeDmgEvt, srcCount map[uint64]int, base int, wr *Vec3Range, n int) {
	t.Helper()
	if wr == nil {
		t.Logf("M4/M5/M6 : bornes monde absentes — distances non calculables, mesures sautees")
		return
	}
	tr := sondeBipedTracks(t, dir, wr, n)
	nSamp := 0
	for _, ss := range tr {
		nSamp += len(ss)
	}
	t.Logf("   couverture positions : %d slots suivis · %d echantillons", len(tr), nSamp)
	// CALIBRATION DE LA BASE : la bande de slots bipede est decalee d'un film a l'autre
	// (idLow runtime). L'argmax structurel lands-on-biped (lot1chIsBiped) est INSTABLE sur les
	// films a degats rares (il a choisi 510 au lieu de 512 sur 01e1f945). On retient plutot le
	// PIC du balayage de resolvabilite : il est franchement unimodal et vaut 512 sur les trois
	// films — un vrai parametre de decalage, pas un surajustement (le pic est aigu). La
	// resolvabilite ne depend PAS des bornes monde (juste presence de position), donc ce
	// choix ne fabrique aucune distance.
	sweep, peakBase, peakRes := sondeBaseSweep(dmg, tr)
	t.Logf("   base : argmax structurel lands-on-biped = %d · PIC de resolvabilite = %d (%d resolus) — retenu : %d",
		base, peakBase, peakRes, peakBase)
	t.Logf("   resolus par base candidate : %s", sweep)
	base = peakBase
	var (
		resolvable, withRefs int
		distAll              []float64
		distBySrc            = map[uint64][]float64{}
		resBySrc             = map[uint64][2]int{} // [resolus, total] par tag source
		bucketCount          = make([]int, len(sondeDistEdges)+1)
	)
	for _, e := range dmg {
		if e.idx0 < 0 || e.idx1 < 0 {
			continue
		}
		withRefs++
		rc := resBySrc[e.src]
		rc[1]++
		pv, okv := sondeLookup(tr[uint32(base+e.idx0)], e.ts, sondePosTolUS)
		pa, oka := sondeLookup(tr[uint32(base+e.idx1)], e.ts, sondePosTolUS)
		if okv && oka {
			resolvable++
			rc[0]++
			d := sondeDist(pa, pv)
			distAll = append(distAll, d)
			distBySrc[e.src] = append(distBySrc[e.src], d)
			bucketCount[sondeBucket(d)]++
		}
		resBySrc[e.src] = rc
	}
	t.Logf("M4 RESOLVABILITE : %d degats a DEUX refs · %d resolus (les 2 positions <= %d ms) = %.1f %%",
		withRefs, resolvable, sondePosTolUS/1000, lot1Pct(resolvable, withRefs))
	t.Logf("M5 DISTANCE (m) attaquant<->victime : mediane globale %.1f · %s", sondeMedian(distAll), sonde5Hist(bucketCount))
	sonde6Bias(t, srcCount, distBySrc, resBySrc)
}

// sonde6Bias juge l'uniformite de la CAPTURE GEOMETRIQUE entre tags source frequents. Pour
// chaque tag : total (tous les degats de cette source) -> avecRefs (les deux handles d'en-tete
// presents) -> resolus (les deux positions trouvees). Le taux qui compte pour la
// representativite est resolus/total : un tag frequent a 0 % (refs absentes) est une arme dont
// la FORME distance serait totalement absente du film.
func sonde6Bias(t *testing.T, srcCount map[uint64]int, distBySrc map[uint64][]float64, resBySrc map[uint64][2]int) {
	t.Helper()
	type row struct {
		src                      uint64
		total, withRefs, res     int
		captRate, posRate, medic float64
	}
	var rows []row
	for s, c := range srcCount {
		if c < 8 {
			continue // tags rares : bruit, ecartes du test de biais
		}
		rc := resBySrc[s] // [resolus, avecRefs]
		r := row{src: s, total: c, withRefs: rc[1], res: rc[0], medic: sondeMedian(distBySrc[s])}
		r.captRate = lot1Pct(rc[0], c)
		if rc[1] > 0 {
			r.posRate = lot1Pct(rc[0], rc[1])
		}
		rows = append(rows, r)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].total > rows[j].total })
	t.Logf("M6 BIAIS — capture geometrique par tag source frequent (>=8 evts) :")
	minR, maxR := math.Inf(1), math.Inf(-1)
	for _, r := range rows {
		t.Logf("   src %#010x : total %d · avecRefs %d · resolus %d · capture %.1f %% (pos %.1f %%) · mediane %.1f m",
			r.src, r.total, r.withRefs, r.res, r.captRate, r.posRate, r.medic)
		if r.captRate < minR {
			minR = r.captRate
		}
		if r.captRate > maxR {
			maxR = r.captRate
		}
	}
	if len(rows) == 0 {
		t.Logf("   aucun tag source frequent : test de biais non concluant sur ce film")
		return
	}
	spread := maxR - minR
	t.Logf("   ECART de capture (resolus/total) entre tags frequents : %.1f points (min %.1f %% / max %.1f %%)", spread, minR, maxR)
	t.Logf("   VERDICT biais ARME (capture uniforme si ecart < 20 pts) : %s", lot1Verdict(spread < 20))
	t.Logf("   NOTE biais DISTANCE : la distance n'existe QUE sur les degats resolus ; la completude")
	t.Logf("   absolue par tranche exige le total API (hors ligne). Cote positions la capture est quasi")
	t.Logf("   totale (pos %% ci-dessus) : le biais, quand il existe, est au niveau des REFS (une source")
	t.Logf("   sans handles d'en-tete disparait entierement de la forme distance).")
}

// sonde7Proxy publie le proxy brut degats/tirs (ordre de grandeur, PAS la vraie precision).
func sonde7Proxy(t *testing.T, longFires, nDamage int) {
	t.Helper()
	ratio := 0.0
	if longFires > 0 {
		ratio = float64(nDamage) / float64(longFires)
	}
	t.Logf("M7 PROXY BRUT : %d degats / %d tirs (records longs) = %.2f degats par tir (AGREGAT).",
		nDamage, longFires, ratio)
	t.Logf("   Ce n'est PAS la precision (1 tir -> plusieurs composantes de degat, et vice-versa ;")
	t.Logf("   la ventilation par arme exige le join M3). A recaler sur le total API pour toute feature.")
}
