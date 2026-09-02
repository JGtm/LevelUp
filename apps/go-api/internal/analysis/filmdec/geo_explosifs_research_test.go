package filmdec

// geo_explosifs_research_test.go — ATTRIBUER une TOUCHE EXPLOSIVE non fatale a son TIREUR par
// une jointure GEOMETRIQUE, robuste au BTB, la ou la jointure TEMPORELLE seule echoue.
//
// PROBLEME (utilisateur). Une touche explosive est un damage_aftermath (0xC0 t0) dont le
// responsable ref1 n'est PAS un bipede joueur (c'est le PROJECTILE) ; la victime ref0 resout en
// slot (base 512) donc en POSITION (cf. NOTE_TOUCHES_EXPLOSIVES). Le tir lourd est un
// action_weapon_fire (0xD2 t36) qui porte le tireur (ref0 dom1), le WeaponID et le FilmIndex du
// joueur. La jointure TEMPORELLE (« l'unique tir lourd de la fenetre de vol ») attribue a 100 %
// en arene mais AMBIGUE en BTB (plusieurs projectiles en vol, grande carte). On a mieux : la
// VISEE du tireur (i21, cap+elevation, validee <1 deg contre la geometrie des kills), les
// POSITIONS des deux bipedes, et une VITESSE par arme.
//
// METHODE.
//  M1 — VITESSE PAR ARME : sur les cas FACILES (un seul tireur lourd dans la fenetre),
//       vitesse = distance(tireur, victime) / (T_touche - T_tir). Mediane par WeaponID.
//  M2 — SCORE GEOMETRIQUE : pour chaque touche, chaque tir lourd candidat recoit
//       ALIGNEMENT (angle visee vs direction tireur->victime) + ECART DE TEMPS DE VOL
//       (|dist/vitesse - dt|). Le gagnant MINIMISE. Couverture + ambiguite.
//  M3 — VERITE TERRAIN : sur les touches FATALES (appariees a une mort), le tueur est connu
//       (dead-state i11 +0x08 = EnumB, roster, 97,6 %). Le gagnant geometrique == le tueur ?
//       Table d'identite roster<->FilmIndex apprise des morts (EnumA <-> FilmIndex de la
//       victime). Taux d'accord geometrie vs temporel-recent.
//  M4 — GAIN BTB : sur les touches AMBIGUES (>1 tireur lourd dans la fenetre), la geometrie
//       tranche-t-elle la ou le temporel abstient ? Comparer arene et BTB (deux films).
//
// RESERVE ecrite AVANT : l'alignement de visee est NET pour les projectiles DIRECTS
// (roquette/empaleur/mangler/fusil a choc) ; les TRAQUEURS (Hydra en verrouillage) et les tirs
// en CLOCHE (Fuel Rod, Ravageur) courbent -> alignement moins discriminant. geoIsDirect les
// separe et M2 mesure l'alignement par classe.
//
// Garde LOT1_TRAME_FILM (un film). Un film par process, verrou pris, lecture seule, borne a
// deltaWitnessChunks (12). Arene : 000d5950. BTB : 4f77afc1 (63 chunks, borne 16). Collecteurs,
// geometrie et types : geo_explosifs_helpers_test.go.

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// geoCandidate : un tir lourd candidat pour une touche, avec sa geometrie resolue a T_tir.
type geoCandidate struct {
	shot     geoShot
	dist     float64
	dtS      float64 // (T_touche - T_tir) en secondes
	angle    float64 // angle visee vs direction tireur->victime (deg)
	hasAngle bool
}

// geoFlightTolS : tolerance d'ecart de temps de vol (s) au-dela de laquelle le terme sature.
const geoFlightTolS = 0.5

// geoFindCandidates rend les tirs lourds dont T_tir tombe dans [T_touche - vol, T_touche], avec
// tireur ET victime resolus en position a T_tir (donc tireur bipede : les tracks n'ont que ti=35).
func geoFindCandidates(tc geoTouch, heavy []geoShot, tracks map[uint32][]geoAimSample) []geoCandidate {
	if !tc.hasVict {
		return nil
	}
	lo := uint64(0)
	if tc.ts > geoFlightW {
		lo = tc.ts - geoFlightW
	}
	i := sort.Search(len(heavy), func(i int) bool { return heavy[i].ts >= lo })
	var out []geoCandidate
	for ; i < len(heavy) && heavy[i].ts <= tc.ts; i++ {
		s := heavy[i]
		sh, oks := geoLookup(tracks[uint32(geoActiveBase+int(s.att))], s.ts, geoPosTolUS)
		vic, okv := geoLookup(tracks[tc.victSlot], s.ts, geoPosTolUS)
		if !oks || !okv {
			continue
		}
		cand := geoCandidate{shot: s, dist: geoDist(sh, vic), dtS: float64(tc.ts-s.ts) / 1e6}
		cand.angle, cand.hasAngle = geoAngleToVictim(sh, vic)
		out = append(out, cand)
	}
	return out
}

// geoDistinctShooters compte les index d'attaquant distincts d'un jeu de candidats.
func geoDistinctShooters(cands []geoCandidate) int {
	seen := map[uint64]bool{}
	for _, c := range cands {
		seen[c.shot.att] = true
	}
	return len(seen)
}

// geoScore : cout d'un candidat = alignement (deg) + penalite de temps de vol (0..90 deg).
// Sans visee, alignement neutre a 60 deg (le candidat reste juge par le temps de vol).
func geoScore(c geoCandidate, speed float64) float64 {
	align := 60.0
	if c.hasAngle {
		align = c.angle
	}
	pred := c.dist / speed
	err := pred - c.dtS
	if err < 0 {
		err = -err
	}
	ft := err / geoFlightTolS
	if ft > 1 {
		ft = 1
	}
	return align + 90*ft
}

// geoSpeedFor rend la vitesse calibree du WeaponID, sinon la vitesse par defaut.
func geoSpeedFor(speedByWid map[uint64]float64, wid uint64) float64 {
	if v, ok := speedByWid[wid]; ok && v > 0 {
		return v
	}
	return geoDefSpeedU
}

// geoTemporalWinner rend le candidat le plus RECENT (T_tir max <= T_touche) : la jointure
// temporelle naive « le dernier tir lourd avant l'impact ».
func geoTemporalWinner(cands []geoCandidate) (geoCandidate, bool) {
	best, ok := geoCandidate{}, false
	for _, c := range cands {
		if !ok || c.shot.ts > best.shot.ts {
			best, ok = c, true
		}
	}
	return best, ok
}

// geoGeometricWinner rend le candidat de cout minimal (alignement + temps de vol) et la MARGE
// sur le second (grande marge = choix confiant).
func geoGeometricWinner(cands []geoCandidate, speedByWid map[uint64]float64) (geoCandidate, float64, bool) {
	best, ok := geoCandidate{}, false
	bestS, secondS := 0.0, 1e18
	for _, c := range cands {
		s := geoScore(c, geoSpeedFor(speedByWid, c.shot.wid))
		if !ok || s < bestS {
			secondS = bestS
			best, bestS, ok = c, s, true
		} else if s < secondS {
			secondS = s
		}
	}
	return best, secondS - bestS, ok
}

// geoAlignConfirm : sous cet angle (deg), un tir lourd candidat VISE la victime — attribution
// geometriquement CONFIRMEE (par opposition a une coincidence temporelle, ~57 deg au hasard).
const geoAlignConfirm = 15.0

// geoBestAligned rend le candidat le mieux aligne (angle min parmi ceux a visee lisible).
func geoBestAligned(cands []geoCandidate) (geoCandidate, bool) {
	best, ok := geoCandidate{}, false
	for _, c := range cands {
		if c.hasAngle && (!ok || c.angle < best.angle) {
			best, ok = c, true
		}
	}
	return best, ok
}

// geoCalibrateSpeed calibre la vitesse par arme sur les touches CONFIRMEES par la geometrie (le
// tir candidat le mieux aligne vise la victime a < geoAlignConfirm) — pas sur les coincidences
// temporelles, qui polluaient l'estimation. Rend (vitesse mediane par WeaponID n>=3, globale).
func geoCalibrateSpeed(t *testing.T, touch []geoTouch, heavy []geoShot, tracks map[uint32][]geoAimSample) (map[uint64]float64, float64) {
	t.Helper()
	byWid := map[uint64][]float64{}
	var all []float64
	perName := map[string][]float64{}
	for _, tc := range touch {
		cands := geoFindCandidates(tc, heavy, tracks)
		rec, ok := geoBestAligned(cands)
		if !ok || rec.angle >= geoAlignConfirm || tc.ts <= rec.shot.ts+geoMinDtUS {
			continue
		}
		v := rec.dist / rec.dtS
		if v <= 0 || v > 400 {
			continue // aberration (position fausse) : hors calibration
		}
		byWid[rec.shot.wid] = append(byWid[rec.shot.wid], v)
		perName[rec.shot.name] = append(perName[rec.shot.name], v)
		all = append(all, v)
	}
	med := map[uint64]float64{}
	for w, vs := range byWid {
		if len(vs) >= 3 {
			med[w] = geoMedian(vs)
		}
	}
	t.Logf("M1 VITESSE par arme (unites monde/s ; touches CONFIRMEES : tir aligne < %.0f deg sur la victime) :", geoAlignConfirm)
	t.Logf("   GLOBAL : n=%d · mediane %.1f · defaut applique aux armes n<3 = %.1f", len(all), geoMedian(all), geoDefSpeedU)
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
	t.Logf("   (* = arme calibree n>=3, utilisee dans le terme de temps de vol ; sinon defaut)")
	return med, geoMedian(all)
}

// geoBuildIdentity apprend la table roster(EnumA/B) -> FilmIndex : chaque mort lie la victime
// (slot -> FilmIndex via ses tirs) a son roster EnumA. Rend (table, cardinalite, injective?).
func geoBuildIdentity(shots []geoShot, kills []geoKill) (map[int32]int, int, bool) {
	slotFilm := map[uint32]map[int]int{}
	for _, s := range shots {
		slot := uint32(geoActiveBase + int(s.att))
		if slotFilm[slot] == nil {
			slotFilm[slot] = map[int]int{}
		}
		slotFilm[slot][s.film]++
	}
	argmax := func(m map[int]int) (int, bool) {
		best, bn, ok := 0, -1, false
		for f, n := range m {
			if n > bn {
				best, bn, ok = f, n, true
			}
		}
		return best, ok
	}
	rosterVotes := map[int32]map[int]int{}
	for _, k := range kills {
		if fm, ok := slotFilm[k.victSlot]; ok {
			if f, ok2 := argmax(fm); ok2 {
				if rosterVotes[k.victRost] == nil {
					rosterVotes[k.victRost] = map[int]int{}
				}
				rosterVotes[k.victRost][f]++
			}
		}
	}
	table := map[int32]int{}
	filmUsed := map[int]int32{}
	injective := true
	for r, m := range rosterVotes {
		if f, ok := argmax(m); ok {
			table[r] = f
			if prev, seen := filmUsed[f]; seen && prev != r {
				injective = false
			}
			filmUsed[f] = r
		}
	}
	return table, len(table), injective
}

// TestGeoExplosifs produit les 4 mesures de la jointure geometrique sur LOT1_TRAME_FILM.
func TestGeoExplosifs(t *testing.T) {
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
	if n > geoMaxChunks {
		n = geoMaxChunks
	}
	wr := sondeWorldRange(t, dir)
	if wr == nil {
		t.Skipf("bornes monde absentes : la geometrie exige des positions (renseigner %s)", sondeMapEnv)
	}

	shots := geoCollectShots(t, dir, n)
	raws, kills := geoCollectDamageKills(t, dir, reg, n)
	tracks := geoTracks(t, dir, wr, n)

	geoActiveBase = geoDetectBase(raws)
	defer func() { geoActiveBase = geoBase }()
	touch := geoBuildTouches(raws, geoActiveBase)

	var heavy []geoShot
	films := map[int]bool{}
	for _, s := range shots {
		films[s.film] = true
		if s.heavy {
			heavy = append(heavy, s)
		}
	}
	sort.Slice(heavy, func(i, j int) bool { return heavy[i].ts < heavy[j].ts })
	geoMatchFatal(touch, kills)
	nFatal := 0
	for _, tc := range touch {
		if tc.fatal {
			nFatal++
		}
	}
	t.Logf("== film %s · %d chunks · %d tireurs (FilmIndex) distincts · base detectee %d ==",
		filepath.Base(dir), n, len(films), geoActiveBase)
	t.Logf("collecte : %d tirs (%d lourds) · %d touches non-bipede · %d morts · %d touches fatales · %d slots suivis",
		len(shots), len(heavy), len(touch), len(kills), nFatal, len(tracks))

	geoValidateAim(t, raws, geoActiveBase, tracks)
	speedByWid, _ := geoCalibrateSpeed(t, touch, heavy, tracks)
	geoM2Coverage(t, touch, heavy, tracks, speedByWid)
	table, card, inj := geoBuildIdentity(shots, kills)
	t.Logf("M3 identite roster<->FilmIndex apprise des morts : %d roster mappes · injective %v", card, inj)
	geoM3Truth(t, touch, heavy, tracks, speedByWid, table)
	geoM4Gain(t, touch, heavy, tracks, speedByWid, table)
}
