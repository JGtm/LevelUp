package filmdec

// deto_attribution_groundtruth_test.go — VERIFICATION ADVERSE de l'attribution
// detonation -> tireur -> splash contre la SEULE verite terrain interne : le dead-state (i11
// +0x08 = EnumB) de la victime d'un KILL explosif. L'instrument principal
// (deto_attribution_research_test.go) valide le gagnant geometrique contre l'oracle NAISSANCE
// (le tir lourd dont le tireur est le plus proche du point de naissance du projectile). Cet
// oracle et le gagnant geometrique derivent tous deux des memes `tracks` (positions/visee) : leur
// accord (98 %) mesure une COHERENCE INTERNE, PAS la verite. La vraie preuve est ailleurs : sur un
// KILL explosif, le tueur est ecrit dans le film (dead-state), independamment de toute geometrie.
//
// M5 — VERITE TERRAIN. Pour chaque touche explosive FATALE (appariee a une mort, tueur EnumB
//   connu, mappe en FilmIndex par geoBuildIdentity), on relie la detonation SOURCE de
//   l'eclaboussure (la plus proche de la victime, dans le rayon+instant), puis on compare a
//   trueFilm quatre attributions : (a) DETONATION->tireur geometrique (la these), (b) VICTIME->
//   tireur geometrique (la voie REFUTEE, splash hors axe), (c) TEMPOREL-recent, (d) NAISSANCE
//   (l'oracle interne). Si (a) >> (b), la these « viser la detonation, pas la victime » survit a la
//   verite terrain. Un temoin SPATIAL (victime decalee) mesure la part de coincidence du lien.
//
// M6 — CONFOND (nettete du pic). Pour les detonations DIRECTES a tireur connu (naissance), angle
//   visee->detonation a T_tir (la these) vs le MEME tireur vise a T_tir+3s (visee decalee, meme
//   geometrie mauvais instant) vs un tireur ALEATOIRE. Un pic net a ~0 pour la these et un plat
//   diffus pour les temoins = l'alignement est DISCRIMINANT, pas un artefact de densite d'angles.
//
// Garde LOT1_TRAME_FILM. Un film par process, verrou pris par TestDetoAttribution (appelant),
// lecture seule, borne a geoMaxChunks. Reutilise les collecteurs/geometrie de
// geo_explosifs_helpers_test.go (verite terrain) et deto_attribution_helpers_test.go (detonation).

import (
	"math/rand"
	"testing"
)

const (
	// detoGTRadius : rayon d'eclaboussure (unites monde) pour relier une mort a sa detonation source.
	detoGTRadius = 8.0
	// detoGTPreW : la detonation (derniere position repliquee) PRECEDE la meche ; on la cherche
	// jusqu'a 2,5 s avant la touche fatale (M4 situe le pic a -1 s).
	detoGTPreW = uint64(2_500_000)
	// detoGTPostW : petite marge apres la touche.
	detoGTPostW = uint64(300_000)
	// detoGTWitnessOffset : decalage spatial (unites monde) du temoin (victime deplacee).
	detoGTWitnessOffset = 30.0
	// detoGTAimShift : decalage de visee du temoin de confond (3 s).
	detoGTAimShift = uint64(3_000_000)
	// detoM6MinSpeed : sous cette vitesse calibree (unites monde/s) l'arme classee DIRECT est en
	// fait HITSCAN (Stalker ~2,7) : son projectile apparie est fortuit, on l'ecarte du confond.
	detoM6MinSpeed = 8.0
)

// detoGTCtx regroupe les entrees de M5 (param count <= 5).
type detoGTCtx struct {
	detons []detoDeton
	touch  []geoTouch
	kills  []geoKill
	heavy  []geoShot
	tracks map[uint32][]geoAimSample
	speed  map[uint64]float64
	table  map[int32]int
}

// detoLinkFatalDeton relie une position de victime (a l'instant de la touche fatale) a la
// detonation SOURCE : la plus proche dans le rayon, d'instant dans [ts-pre, ts+post].
func detoLinkFatalDeton(vp geoAimSample, ts uint64, detons []detoDeton, radius float64) (detoDeton, bool) {
	lo := uint64(0)
	if ts > detoGTPreW {
		lo = ts - detoGTPreW
	}
	hi := ts + detoGTPostW
	best, ok := detoDeton{}, false
	bestD := radius
	for _, d := range detons {
		if d.ts < lo || d.ts > hi {
			continue
		}
		if dd := geoDist(vp, d.pos); dd < bestD {
			best, ok, bestD = d, true, dd
		}
	}
	return best, ok
}

// detoGTWinners : les 4 FilmIndex attribues pour une detonation source d'un kill (et leur validite).
type detoGTWinners struct {
	deto, vict, temp, birth             int
	hasDeto, hasVict, hasTemp, hasBirth bool
}

// detoGTAttribute calcule les 4 attributions concurrentes pour un kill (victime en victSlot a
// l'instant ts) et sa detonation source.
func detoGTAttribute(c detoGTCtx, victSlot uint32, ts uint64, d detoDeton) detoGTWinners {
	var w detoGTWinners
	dc := detoFindCandidates(d, c.heavy, c.tracks)
	if gw, _, ok := detoGeometricWinner(dc, c.speed); ok {
		w.deto, w.hasDeto = gw.shot.film, true
	}
	if tw, ok := detoTemporalWinner(dc); ok {
		w.temp, w.hasTemp = tw.shot.film, true
	}
	// voie VICTIME (refutee) : viser la victime, pas la detonation. geoFindCandidates prend une
	// touche synthetique (meme victime, meme instant que la mort).
	vc := geoFindCandidates(geoTouch{ts: ts, victSlot: victSlot, hasVict: true}, c.heavy, c.tracks)
	if gv, _, ok := geoGeometricWinner(vc, c.speed); ok {
		w.vict, w.hasVict = gv.shot.film, true
	}
	if bs, _, ok := detoBirthMatch(d, c.heavy, c.tracks); ok {
		w.birth, w.hasBirth = bs.film, true
	}
	return w
}

// detoGTTally cumule les accords a trueFilm des 4 voies.
type detoGTTally struct {
	linked, evalDeto, okDeto, evalVict, okVict, evalTemp, okTemp, evalBirth, okBirth int
}

func (a *detoGTTally) add(w detoGTWinners, trueFilm int) {
	a.linked++
	if w.hasDeto {
		a.evalDeto++
		if w.deto == trueFilm {
			a.okDeto++
		}
	}
	if w.hasVict {
		a.evalVict++
		if w.vict == trueFilm {
			a.okVict++
		}
	}
	if w.hasTemp {
		a.evalTemp++
		if w.temp == trueFilm {
			a.okTemp++
		}
	}
	if w.hasBirth {
		a.evalBirth++
		if w.birth == trueFilm {
			a.okBirth++
		}
	}
}

// detoM5GroundTruth : LA preuve. On lie chaque MORT (dead-state) a sa detonation source (la plus
// proche de la victime dans le rayon+instant) — sans dependre de la resolution des touches
// non-bipede (qui affame). Accord des 4 voies au tueur EnumB->FilmIndex, plus un temoin spatial.
// La MORT porte l'oracle (EnumB) ; on cherche l'eclaboussure qui l'a causee.
func detoM5GroundTruth(t *testing.T, c detoGTCtx) {
	t.Helper()
	var killMapped, killLinked, explosiveDeaths int
	var hit, wit detoGTTally
	for _, k := range c.kills {
		trueFilm, ok := c.table[k.killer]
		if !ok {
			continue
		}
		vp, okv := geoLookup(c.tracks[k.victSlot], k.ts, detoPosTol)
		if !okv {
			continue
		}
		killMapped++
		if d, ok := detoLinkFatalDeton(vp, k.ts, c.detons, detoGTRadius); ok {
			explosiveDeaths++
			hit.add(detoGTAttribute(c, k.victSlot, k.ts, d), trueFilm)
			killLinked++
		}
		wp := vp
		wp.x += detoGTWitnessOffset
		if d, ok := detoLinkFatalDeton(wp, k.ts, c.detons, detoGTRadius); ok {
			wit.add(detoGTAttribute(c, k.victSlot, k.ts, d), trueFilm)
		}
	}
	nFatalTouch := 0
	for _, tc := range c.touch {
		if tc.fatal {
			nFatalTouch++
		}
	}
	t.Logf("M5 VERITE TERRAIN (kill explosif · gagnant == tueur dead-state EnumB, PAS l'oracle naissance) :")
	t.Logf("   PLAFOND du harvest : %d morts (rejeu par chunk) · %d tueur mappe · touches fatales non-bipede %d",
		len(c.kills), killMapped, nFatalTouch)
	t.Logf("   morts a detonation source proche (rayon %.0f u = kill EXPLOSIF appariable) : %d (%.1f %% des morts mappees)",
		detoGTRadius, explosiveDeaths, lot1Pct(killLinked, killMapped))
	t.Logf("   DETONATION->tireur (these) : %d/%d (%.1f %%)  <- comparer a VICTIME->tireur (voie refutee) : %d/%d (%.1f %%)",
		hit.okDeto, hit.evalDeto, lot1Pct(hit.okDeto, hit.evalDeto),
		hit.okVict, hit.evalVict, lot1Pct(hit.okVict, hit.evalVict))
	t.Logf("   TEMPOREL-recent : %d/%d (%.1f %%) · NAISSANCE (oracle interne) : %d/%d (%.1f %%)",
		hit.okTemp, hit.evalTemp, lot1Pct(hit.okTemp, hit.evalTemp),
		hit.okBirth, hit.evalBirth, lot1Pct(hit.okBirth, hit.evalBirth))
	t.Logf("   TEMOIN SPATIAL (victime decalee de %.0f u) : reliee %d · DETONATION juste %d/%d (%.1f %%)",
		detoGTWitnessOffset, wit.linked, wit.okDeto, wit.evalDeto, lot1Pct(wit.okDeto, wit.evalDeto))
	confirmed := hit.evalDeto >= 3 && hit.okDeto*2 >= hit.evalDeto && hit.okDeto >= hit.okVict
	t.Logf("   VERDICT (>=3 kills relies · these >= 50 %% ET these >= voie victime) : %s", lot1Verdict(confirmed))
	t.Logf("   LECTURE : la these SURVIT si DETONATION bat VICTIME contre le dead-state. PLAFOND BAS = le harvest de morts")
	t.Logf("   par rejeu-par-chunk (pas de localisateur d'events) rate la plupart des kills : preuve terrain data-starved.")
}

// detoM6Confound : nettete du pic d'alignement visee->detonation (VRAIS lanceurs de projectile,
// tireur connu par la naissance) vs deux temoins — le MEME tireur vise 3 s plus tard, et un tireur
// ALEATOIRE. On ECARTE les armes HITSCAN faussement classees DIRECT (Stalker/Bulldog) par leur
// vitesse calibree absurde (< detoM6MinSpeed) : sans ce filtre, leur alignement ~110 deg (projectile
// voisin fortuit) pollue le pic. Le filtre vitesse est le meme discriminant que la note (M2).
func detoM6Confound(t *testing.T, detons []detoDeton, heavy []geoShot, tracks map[uint32][]geoAimSample, speed map[uint64]float64) {
	t.Helper()
	var hReal, hShift, hRand [6]int
	var nReal, nShift, nRand int
	var slots []uint32
	for s := range tracks {
		slots = append(slots, s)
	}
	rng := rand.New(rand.NewSource(1))
	for _, d := range detons {
		s, sh, ok := detoBirthMatch(d, heavy, tracks)
		if !ok || !geoIsDirect(s.name) {
			continue
		}
		// Filtre HITSCAN : une arme dont la vitesse calibree est trop faible n'est pas un projectile
		// (le "projectile" apparie est une grenade voisine fortuite) — hors confond.
		if v, okv := speed[s.wid]; okv && v < detoM6MinSpeed {
			continue
		}
		if a, okA := geoAngleToVictim(sh, d.pos); okA {
			detoAngleBucket(a, &hReal)
			nReal++
		}
		// temoin 1 : meme tireur, visee 3 s plus tard (meme trajectoire de jeu, mauvais instant).
		if sh2, ok2 := geoLookup(tracks[uint32(geoActiveBase+int(s.att))], s.ts+detoGTAimShift, detoPosTol); ok2 {
			if a, okA := geoAngleToVictim(sh2, d.pos); okA {
				detoAngleBucket(a, &hShift)
				nShift++
			}
		}
		// temoin 2 : tireur aleatoire (autre slot) a l'instant du tir.
		if len(slots) > 0 {
			rs := slots[rng.Intn(len(slots))]
			if sh3, ok3 := geoLookup(tracks[rs], s.ts, detoPosTol); ok3 && sh3.hasAim {
				if a, okA := geoAngleToVictim(sh3, d.pos); okA {
					detoAngleBucket(a, &hRand)
					nRand++
				}
			}
		}
	}
	t.Logf("M6 CONFOND — nettete du pic visee->detonation (armes DIRECTES, tireur connu) :")
	t.Logf("   THESE  (tireur vrai, T_tir)      : n=%d · <5:%d <15:%d <30:%d <45:%d <60:%d >=60:%d",
		nReal, hReal[0], hReal[1], hReal[2], hReal[3], hReal[4], hReal[5])
	t.Logf("   TEMOIN (meme tireur, T_tir+3s)   : n=%d · <5:%d <15:%d <30:%d <45:%d <60:%d >=60:%d",
		nShift, hShift[0], hShift[1], hShift[2], hShift[3], hShift[4], hShift[5])
	t.Logf("   TEMOIN (tireur aleatoire, T_tir) : n=%d · <5:%d <15:%d <30:%d <45:%d <60:%d >=60:%d",
		nRand, hRand[0], hRand[1], hRand[2], hRand[3], hRand[4], hRand[5])
	near := hReal[0] + hReal[1]
	sharp := nReal >= 8 && near*2 > nReal && hShift[0]+hShift[1] <= near/2
	t.Logf("   VERDICT (pic these <15 deg majoritaire ET temoin decale <= moitie) : %s", lot1Verdict(sharp))
	t.Logf("   LECTURE : pic net a ~0 pour la these, plat diffus pour les temoins = l'alignement est DISCRIMINANT.")
}
