package filmdec

// deto_attribution_research_test.go — ATTRIBUER les TOUCHES EXPLOSIVES a leur TIREUR par la
// PISTE DE DETONATION (la bonne geometrie), la ou l'attribution victime->tireur echoue.
//
// PROBLEME (utilisateur). Le degat explosif est du SPLASH : la victime est HORS de l'axe de
// visee du tireur (geo_explosifs mesure ~57 deg, angle de coincidence au hasard). On NE PEUT
// DONC PAS attribuer victime->tireur par l'alignement. MAIS le POINT DE DETONATION, lui, est SUR
// l'axe : le tireur a vise LA ou ca explose. On decode donc le point de detonation, on l'attribue
// au tireur par alignement (qui redevient discriminant), et on rattache les touches splash
// PROCHES de ce point.
//
// SOURCE DU POINT DE DETONATION. Il n'existe pas d'evenement film portant une position d'impact
// exploitable (0xC2/0xC3 ne portent que ref0+variant-name, cf. lot1_projectiles : variant ~100 %
// distincte, pas de position deroulee). La source FIABLE est la DERNIERE POSITION REPLIQUEE de
// l'entite projectile (ti=41, ScanFilmProjectiles) juste avant la fin de sa vie. RESERVE ecrite
// AVANT (projectiles.go) : pour une GRENADE la replication cesse ~1,4 s avant la meche (~3 s),
// donc le dernier point n'est PAS l'explosion ; pour un projectile DIRECT (roquette/empaleur/
// mangler/choc/stalker/bulldog) qui detone a l'impact, le dernier point EST le point d'impact.
// On mesure donc separement les armes DIRECTES (geoIsDirect). at-rest (i18) certifie une fin de vol.
//
// METHODE (5 mesures).
//  M0 — SOURCE : combien de detonations ; part appariee a un tir lourd par la NAISSANCE (oracle
//       projectiles.go, 70/70) ; distance naissance<->tireur vs temoin ; part at-rest.
//  M1 — ALIGNEMENT (le coeur) : pour les detonations a tireur connu (naissance), l'angle
//       visee->DETONATION est-il PETIT (sur l'axe) quand l'angle visee->VICTIME splash est GRAND
//       (hors axe) ? Et un temoin (detonation d'un AUTRE tir) est-il ~57 deg (hasard) ?
//  M2 — VITESSE PAR ARME : sur les cas propres (naissance appariee, direct), vitesse =
//       dist(tireur, detonation) / (T_deto - T_tir). Mediane par WeaponID.
//  M3 — DETONATION -> TIREUR : score alignement + temps de vol ; gagnant == tireur-naissance
//       (verite) ? vs temporel-recent, vs temoin decale. Unicite/desambiguisation.
//  M4 — TOUCHES SPLASH : part des touches non-bipede a portee (rayon+instant) d'une detonation
//       (couverture) ; detonation unique (attribution non ambigue) ; temoin decale.
//
// Garde LOT1_TRAME_FILM (un film). Un film par process, verrou pris, lecture seule, borne a
// geoMaxChunks (16). Arene : 000d5950 / 01e1f945 / 00502e52. BTB : 4f77afc1 (carte Forge,
// LOT1_SONDE_MAP="flood gulch"). Collecteurs, scan projectile borne et geometrie :
// deto_attribution_helpers_test.go.

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"testing"
)

// TestDetoAttribution produit les 5 mesures de la piste de detonation sur LOT1_TRAME_FILM.
func TestDetoAttribution(t *testing.T) {
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
	// Borne RAM par defaut geoMaxChunks (16, adapte au BTB). Les films ARENE sont petits (~23 Mo,
	// 30 chunks) : LOT1_MAXCHUNKS releve la borne pour MOISSONNER des kills explosifs (rares) sur la
	// verite terrain M5, sans toucher au defaut BTB. Verification adverse : la preuve exige des kills.
	maxCh := geoMaxChunks
	if v := os.Getenv("LOT1_MAXCHUNKS"); v != "" {
		if k, err := strconv.Atoi(v); err == nil && k > 0 {
			maxCh = k
		}
	}
	if n > maxCh {
		n = maxCh
	}
	wr := sondeWorldRange(t, dir)
	if wr == nil {
		t.Skipf("bornes monde absentes : la piste de detonation exige des positions (renseigner %s)", sondeMapEnv)
	}

	shots := geoCollectShots(t, dir, n)
	raws, kills := geoCollectDamageKills(t, dir, reg, n)
	tracks := geoTracks(t, dir, wr, n)
	detons := detoScanProjectiles(t, dir, wr, n)

	geoActiveBase = geoDetectBase(raws)
	defer func() { geoActiveBase = geoBase }()
	touch := geoBuildTouches(raws, geoActiveBase)
	geoMatchFatal(touch, kills)

	var heavy []geoShot
	films := map[int]bool{}
	for _, s := range shots {
		films[s.film] = true
		if s.heavy {
			heavy = append(heavy, s)
		}
	}
	sort.Slice(heavy, func(i, j int) bool { return heavy[i].ts < heavy[j].ts })
	nVict := 0
	for _, tc := range touch {
		if tc.hasVict {
			nVict++
		}
	}
	t.Logf("== film %s · %d chunks · base detectee %d · %d FilmIndex distincts ==",
		filepath.Base(dir), n, geoActiveBase, len(films))
	t.Logf("collecte : %d tirs (%d lourds) · %d detonations (ti=41) · %d touches non-bipede (%d victime resolue) · %d slots suivis",
		len(shots), len(heavy), len(detons), len(touch), nVict, len(tracks))

	detoHeavyDiag(t, heavy, tracks)
	detoBirthProbe(t, detons, heavy, tracks)
	speedByWid := detoM2Speed(t, detons, heavy, tracks)
	detoM0Source(t, detons, heavy, tracks)
	detoM1Alignment(t, detons, heavy, touch, tracks)
	detoM3Attribution(t, detons, heavy, tracks, speedByWid)
	detoM4Splash(t, detons, touch, tracks)

	table, card, inj := geoBuildIdentity(shots, kills)
	nFatal := 0
	for _, tc := range touch {
		if tc.fatal {
			nFatal++
		}
	}
	t.Logf("VERITE TERRAIN : %d morts · %d touches fatales · identite roster<->FilmIndex %d mappes (injective %v)",
		len(kills), nFatal, card, inj)
	detoM5GroundTruth(t, detoGTCtx{detons: detons, touch: touch, kills: kills, heavy: heavy, tracks: tracks, speed: speedByWid, table: table})
	detoM6Confound(t, detons, heavy, tracks, speedByWid)
}

// detoM0Source : combien de detonations exploitables, part appariee a un tir lourd par la
// naissance, distance naissance<->tireur vs temoin (tireur d'un autre tir), part at-rest.
func detoM0Source(t *testing.T, detons []detoDeton, heavy []geoShot, tracks map[uint32][]geoAimSample) {
	t.Helper()
	var matched, atRest, directMatched int
	var dist, distWit []float64
	for i, d := range detons {
		if d.atRest {
			atRest++
		}
		s, sh, ok := detoBirthMatch(d, heavy, tracks)
		if !ok {
			continue
		}
		matched++
		if geoIsDirect(s.name) {
			directMatched++
		}
		dist = append(dist, geoDist(sh, d.birthPos))
		// temoin : la naissance d'une AUTRE detonation contre le meme tireur (decorrele).
		w := detons[(i+len(detons)/2)%len(detons)]
		distWit = append(distWit, geoDist(sh, w.birthPos))
	}
	t.Logf("M0 SOURCE du point de detonation (derniere position repliquee ti=41) :")
	t.Logf("   %d detonations · at-rest (fin de vol certifiee i18) %d (%.1f %%) · appariees a un tir lourd (naissance) %d (%.1f %%) · dont DIRECTES %d",
		len(detons), atRest, lot1Pct(atRest, len(detons)), matched, lot1Pct(matched, len(detons)), directMatched)
	t.Logf("   distance naissance<->tireur : mediane %.2f u (n=%d) · TEMOIN (autre detonation) mediane %.2f u",
		geoMedian(dist), len(dist), geoMedian(distWit))
	t.Logf("   LECTURE : petite distance naissance<->tireur = la detonation est bien celle du projectile de CE tir lourd")
	t.Logf("   (oracle projectiles.go). Le point de detonation est fiable pour les armes DIRECTES ; en cloche/grenade la")
	t.Logf("   replication cesse avant la meche (dernier point = derniere position connue, pas l'explosion).")
}

// detoM1Alignment : LE COEUR. Angle visee->DETONATION (sur axe, petit) vs visee->VICTIME splash
// (hors axe, grand) vs temoin (autre detonation, ~hasard), sur les detonations a tireur connu.
func detoM1Alignment(t *testing.T, detons []detoDeton, heavy []geoShot, touch []geoTouch, tracks map[uint32][]geoAimSample) {
	t.Helper()
	var aDeto, aVict, aWit []float64
	var hDeto, hWit [6]int
	perName := map[string][]float64{}
	for i, d := range detons {
		s, sh, ok := detoBirthMatch(d, heavy, tracks)
		if !ok || !geoIsDirect(s.name) {
			continue // le point de detonation n'est fiable que pour les armes directes
		}
		if a, okA := geoAngleToVictim(sh, d.pos); okA {
			aDeto = append(aDeto, a)
			perName[s.name] = append(perName[s.name], a)
			detoAngleBucket(a, &hDeto)
		}
		if _, vp, okv := detoNearestTouch(d, touch, tracks); okv {
			if a, okA := geoAngleToVictim(sh, vp); okA {
				aVict = append(aVict, a)
			}
		}
		w := detons[(i+len(detons)/2)%len(detons)]
		if a, okA := geoAngleToVictim(sh, w.pos); okA {
			aWit = append(aWit, a)
			detoAngleBucket(a, &hWit)
		}
	}
	t.Logf("M1 ALIGNEMENT visee du tireur (armes DIRECTES, tireur connu par la naissance) :")
	t.Logf("   -> DETONATION (sur axe attendu) : n=%d mediane %.1f deg · <5:%d <15:%d <30:%d <45:%d <60:%d >=60:%d",
		len(aDeto), geoMedian(aDeto), hDeto[0], hDeto[1], hDeto[2], hDeto[3], hDeto[4], hDeto[5])
	t.Logf("   -> VICTIME splash proche (hors axe): n=%d mediane %.1f deg (starve sur ce film : cf. M4 · geo_explosifs mesure ~57)",
		len(aVict), geoMedian(aVict))
	t.Logf("   -> TEMOIN (autre detonation)       : n=%d mediane %.1f deg · <5:%d <15:%d <30:%d <45:%d <60:%d >=60:%d",
		len(aWit), geoMedian(aWit), hWit[0], hWit[1], hWit[2], hWit[3], hWit[4], hWit[5])
	detoM1PerName(t, perName)
	confirmed := len(aDeto) >= 8 && geoMedian(aDeto) < 30 && geoMedian(aDeto) < 0.5*geoMedian(aWit)
	t.Logf("   VERDICT (detonation SUR l'axe : mediane < 30 deg ET < 0,5x le temoin) : %s", lot1Verdict(confirmed))
	t.Logf("   LECTURE : la detonation est nettement mieux alignee que le temoin ; la ventilation par arme separe les")
	t.Logf("   vrais lanceurs (SPNKr : sur axe) des armes HITSCAN faussement appariees (Stalker : pas de projectile ti=41).")
}

// detoM1PerName ventile l'alignement visee->detonation par arme (n>=5), pour separer les vrais
// lanceurs de projectile des armes hitscan faussement appariees a un projectile voisin.
func detoM1PerName(t *testing.T, perName map[string][]float64) {
	t.Helper()
	type row struct {
		name string
		vs   []float64
	}
	var rows []row
	for nm, vs := range perName {
		rows = append(rows, row{nm, vs})
	}
	sort.Slice(rows, func(i, j int) bool { return len(rows[i].vs) > len(rows[j].vs) })
	for _, r := range rows {
		if len(r.vs) < 5 {
			continue
		}
		var h [6]int
		for _, a := range r.vs {
			detoAngleBucket(a, &h)
		}
		t.Logf("      %-20s : n=%d mediane %.1f deg · <5:%d <15:%d <30:%d <45:%d <60:%d >=60:%d",
			r.name, len(r.vs), geoMedian(r.vs), h[0], h[1], h[2], h[3], h[4], h[5])
	}
}

// detoM2Speed calibre la vitesse par arme sur les detonations DIRECTES appariees a leur tir par
// la naissance (cas propre : tireur et instant connus). Rend la mediane par WeaponID (n>=3).
func detoM2Speed(t *testing.T, detons []detoDeton, heavy []geoShot, tracks map[uint32][]geoAimSample) map[uint64]float64 {
	t.Helper()
	byWid := map[uint64][]float64{}
	perName := map[string][]float64{}
	var all []float64
	for _, d := range detons {
		s, sh, ok := detoBirthMatch(d, heavy, tracks)
		if !ok || !geoIsDirect(s.name) || d.ts <= s.ts+detoMinDtUS {
			continue
		}
		v := geoDist(sh, d.pos) / (float64(d.ts-s.ts) / 1e6)
		if v <= 0 || v > detoSpeedMax {
			continue
		}
		byWid[s.wid] = append(byWid[s.wid], v)
		perName[s.name] = append(perName[s.name], v)
		all = append(all, v)
	}
	med := map[uint64]float64{}
	for w, vs := range byWid {
		if len(vs) >= 3 {
			med[w] = geoMedian(vs)
		}
	}
	t.Logf("M2 VITESSE par arme (unites monde/s ; detonations DIRECTES appariees a leur tir par la naissance) :")
	t.Logf("   GLOBAL n=%d · mediane %.1f · defaut applique aux armes n<3 = %.1f", len(all), geoMedian(all), geoDefSpeedU)
	type row struct {
		name string
		vs   []float64
	}
	var rows []row
	for nm, vs := range perName {
		rows = append(rows, row{nm, vs})
	}
	sort.Slice(rows, func(i, j int) bool { return len(rows[i].vs) > len(rows[j].vs) })
	for _, r := range rows {
		mark := " "
		if len(r.vs) >= 3 {
			mark = "*"
		}
		t.Logf("   %s %-20s : n=%d · mediane %.1f (min %.1f max %.1f)", mark, r.name, len(r.vs),
			geoMedian(r.vs), attribMin(r.vs), attribMax(r.vs))
	}
	return med
}

// detoM3Attribution : detonation -> tireur par score (alignement + temps de vol). Justesse
// contre le tireur-naissance (verite), vs temporel-recent, vs temoin decale ; unicite.
func detoM3Attribution(t *testing.T, detons []detoDeton, heavy []geoShot, tracks map[uint32][]geoAimSample, speedByWid map[uint64]float64) {
	t.Helper()
	var evalN, geoOK, tmpOK, witOK int
	var ambN, ambGeoOK, uniqueAlign int
	amb := [4]int{}
	for _, d := range detons {
		trueShot, _, okT := detoBirthMatch(d, heavy, tracks)
		cands := detoFindCandidates(d, heavy, tracks)
		if len(cands) == 0 {
			continue
		}
		ds := detoDistinctShooters(cands)
		if ds >= 3 {
			amb[3]++
		} else {
			amb[ds]++
		}
		if ba, okb := detoBestAligned(cands); okb && ba.angle < detoAlignConfirm {
			// un seul candidat aligne < seuil = attribution non ambigue par la seule geometrie
			nAligned := 0
			for _, c := range cands {
				if c.hasAngle && c.angle < detoAlignConfirm {
					nAligned++
				}
			}
			if nAligned == 1 {
				uniqueAlign++
			}
		}
		if !okT {
			continue // pas de verite terrain (naissance) pour ce point : hors justesse
		}
		evalN++
		ambiguous := ds >= 2
		if ambiguous {
			ambN++
		}
		if gw, _, ok := detoGeometricWinner(cands, speedByWid); ok && gw.shot.film == trueShot.film {
			geoOK++
			if ambiguous {
				ambGeoOK++
			}
		}
		if tw, ok := detoTemporalWinner(cands); ok && tw.shot.film == trueShot.film {
			tmpOK++
		}
		// temoin : gagnant geometrique d'une detonation DECALEE (memes candidats, mauvaise cible).
		if wgw, ok := detoWitnessWinner(d, heavy, tracks, speedByWid); ok && wgw == trueShot.film {
			witOK++
		}
	}
	t.Logf("M3 DETONATION -> TIREUR (gagnant score alignement+temps de vol == tireur-naissance) :")
	t.Logf("   detonations a candidat : ambiguite tireurs distincts 1=%d 2=%d >=3=%d", amb[1], amb[2], amb[3])
	t.Logf("   evaluables (verite naissance) %d · GEOMETRIE %d (%.1f %%) · TEMPOREL-recent %d (%.1f %%) · TEMOIN decale %d (%.1f %%)",
		evalN, geoOK, lot1Pct(geoOK, evalN), tmpOK, lot1Pct(tmpOK, evalN), witOK, lot1Pct(witOK, evalN))
	t.Logf("   sous-ensemble AMBIGU (>=2 tireurs) %d : GEOMETRIE juste %d (%.1f %%)", ambN, ambGeoOK, lot1Pct(ambGeoOK, ambN))
	t.Logf("   UNICITE : detonations a UN seul tir aligne < %.0f deg : %d", detoAlignConfirm, uniqueAlign)
	t.Logf("   LECTURE : l'ecart GEOMETRIE - TEMOIN mesure le pouvoir discriminant de l'alignement sur la detonation.")
}

// detoWitnessWinner rend le FilmIndex gagnant geometrique quand on vise une detonation DECALEE
// dans le temps (meme jeu de tirs, cible fausse) : temoin de non-specificite de l'alignement.
func detoWitnessWinner(d detoDeton, heavy []geoShot, tracks map[uint32][]geoAimSample, speedByWid map[uint64]float64) (int, bool) {
	shifted := d
	shifted.ts = d.ts + detoWitnessShift
	cands := detoFindCandidates(shifted, heavy, tracks)
	// on garde la cible spatiale d'origine mais les candidats de la fenetre decalee : recalcule l'angle vers d.pos.
	for i := range cands {
		cands[i].angle, cands[i].hasAngle = geoAngleToVictim(cands[i].shooter, d.pos)
		cands[i].dist = geoDist(cands[i].shooter, d.pos)
		cands[i].dtS = float64(shifted.ts-cands[i].shot.ts) / 1e6
	}
	gw, _, ok := detoGeometricWinner(cands, speedByWid)
	if !ok {
		return 0, false
	}
	return gw.shot.film, true
}

// detoM4Splash : part des touches non-bipede a portee spatiale d'une detonation, SWEEP du
// decalage temporel touche<->detonation. Le point de detonation etant la DERNIERE position
// repliquee (qui, pour une grenade, precede la meche), un pic de couverture a un decalage POSITIF
// quantifie ce trou ; un plat au niveau du hasard refute le rattachement spatial.
func detoM4Splash(t *testing.T, detons []detoDeton, touch []geoTouch, tracks map[uint32][]geoAimSample) {
	t.Helper()
	dts := make([]uint64, len(detons))
	for i, d := range detons {
		dts[i] = d.ts
	}
	var vps []geoAimSample
	var vts []uint64
	for _, tc := range touch {
		if vp, ok := detoTouchPos(tc, tracks); ok {
			vps = append(vps, vp)
			vts = append(vts, tc.ts)
		}
	}
	// shift NEGATIF = la detonation est APRES la touche ; POSITIF = detonation AVANT (fuse/gap).
	shifts := []int64{-3_000_000, -1_000_000, 0, 500_000, 1_000_000, 2_000_000, 3_000_000}
	t.Logf("M4 TOUCHES SPLASH <-> detonation (rayon %.0f u · fenetre +-%dms · %d touches a victime resolue) :",
		detoSplashRadius, detoSplashTimeW/1000, len(vps))
	bestShift, bestNear, bestUnique := int64(0), -1, 0
	for _, sh := range shifts {
		near, uniq := 0, 0
		for i := range vps {
			nd := detoCountNearDetons(vps[i], vts[i], detons, dts, sh)
			if nd > 0 {
				near++
				if nd == 1 {
					uniq++
				}
			}
		}
		mark := " "
		if near > bestNear {
			bestShift, bestNear, bestUnique, mark = sh, near, uniq, "*"
		}
		t.Logf("   %s decalage %+5dms : %d/%d touches a portee (%.1f %%) · dont detonation unique %d",
			mark, sh/1000, near, len(vps), lot1Pct(near, len(vps)), uniq)
	}
	t.Logf("   MEILLEUR decalage %+dms : %.1f %% de couverture · unique %d (%.1f %% des rattachees)",
		bestShift/1000, lot1Pct(bestNear, len(vps)), bestUnique, lot1Pct(bestUnique, bestNear))
	t.Logf("   LECTURE : un pic net au-dessus des decalages voisins = rattachement reel (le decalage donne le trou")
	t.Logf("   position repliquee<->impact) ; un plat = les touches splash ne se rattachent PAS au point de detonation.")
}

// detoCountNearDetons compte les detonations dans le rayon d'eclaboussure d'une position de
// touche a son instant, decale de shift (signe : negatif = detonation apres la touche).
func detoCountNearDetons(vp geoAimSample, ts uint64, detons []detoDeton, dts []uint64, shift int64) int {
	T := int64(ts) + shift
	lo := int64(0)
	if T > int64(detoSplashTimeW) {
		lo = T - int64(detoSplashTimeW)
	}
	hi := uint64(T + int64(detoSplashTimeW))
	i := sort.Search(len(dts), func(i int) bool { return dts[i] >= uint64(lo) })
	cnt := 0
	for ; i < len(detons) && detons[i].ts <= hi; i++ {
		if geoDist(vp, detons[i].pos) <= detoSplashRadius {
			cnt++
		}
	}
	return cnt
}
