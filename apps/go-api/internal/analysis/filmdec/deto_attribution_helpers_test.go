package filmdec

// deto_attribution_helpers_test.go — collecteurs, scan projectile BORNE et geometrie de
// l'instrument deto_attribution_research_test.go (scinde pour le seuil de 500 lignes). Voir
// l'en-tete de ce fichier-la pour le raisonnement, les sources et les mesures.

import (
	"sort"
	"testing"
)

const (
	// detoFlightW : fenetre de vol tir lourd -> detonation (2 s), comme geoFlightW.
	detoFlightW = uint64(2_000_000)
	// detoPosTol : ecart max evenement <-> echantillon position (120 ms), comme geoPosTolUS.
	detoPosTol = geoPosTolUS
	// detoBirthPreW / detoBirthPostW : le projectile NAIT juste apres le tir. On cherche le tir
	// lourd dont l'instant precede (<=400 ms) ou suit de peu (<=150 ms) la naissance.
	detoBirthPreW  = uint64(400_000)
	detoBirthPostW = uint64(150_000)
	// detoBirthDistMax : distance max naissance <-> tireur pour valider l'appariement (unites
	// monde). projectiles.go mesure 0,77 u median, 1,37 max sur 70 lancers : 3 u est large.
	detoBirthDistMax = 3.0
	// detoMinDtUS : temps de vol minimal (30 ms) pour calibrer une vitesse.
	detoMinDtUS = uint64(30_000)
	// detoSplashRadius : rayon spatial (unites monde) sous lequel une touche non-bipede est une
	// eclaboussure de la detonation.
	detoSplashRadius = 6.0
	// detoSplashTimeW : ecart temporel max touche <-> detonation (300 ms).
	detoSplashTimeW = uint64(300_000)
	// detoAlignConfirm : sous cet angle (deg) la visee du tireur pointe la detonation -> lien
	// geometriquement CONFIRME (l'axe de visee passe par le point d'explosion).
	detoAlignConfirm = 15.0
	// detoWitnessShift : decalage du temoin temporel (3 s).
	detoWitnessShift = uint64(3_000_000)
	// detoSpeedMax : borne haute d'une vitesse projectile plausible (unites monde/s).
	detoSpeedMax = 400.0
)

// detoDeton : un point de detonation = derniere position repliquee d'une piste projectile
// (ti=41), avec son instant, son drapeau at-rest (fin de vol certifiee) et la naissance.
type detoDeton struct {
	ts       uint64
	pos      geoAimSample // x,y,z du dernier point (hasAim=false : c'est un point, pas une visee)
	atRest   bool
	birthTs  uint64
	birthPos geoAimSample
	nPts     int
}

// detoProjBand releve la bande de slots ti=41 sur les n premiers chunks (BORNE, pour la RAM :
// worldObjectSlotBand balaie tout le film). Meme regle que slotBandExcluding : combler la plage
// de l'archetype, retirer tout slot vu porter un autre archetype.
func detoProjBand(dir string, n int) map[uint32]bool {
	seen, others := map[uint32]bool{}, map[uint32]bool{}
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			for _, r := range WalkKeyframeWorld(pk.Payload(data)) {
				if r.Slot < 0 {
					continue
				}
				if r.TI == ProjectileTypeIndex {
					seen[uint32(r.Slot)] = true
				} else {
					others[uint32(r.Slot)] = true
				}
			}
		}
	}
	return slotBandExcluding(seen, others)
}

// detoScanProjectiles decode les pistes projectile (ti=41) des n premiers chunks et rend UN
// point de detonation par vie (derniere position). Reutilise le decodeur pur scanProjectileRecords
// + le decoupage en vies splitLives (memes que ScanFilmProjectiles), borne aux chunks pour la RAM.
func detoScanProjectiles(t *testing.T, dir string, wr *Vec3Range, n int) []detoDeton {
	t.Helper()
	band := detoProjBand(dir, n)
	if len(band) == 0 || wr == nil {
		return nil
	}
	// Les largeurs d'axe des objets du monde (ti=41) sont celles de la CARTE, pas le defaut
	// 13/13/14 (arene Cliffhanger). Sans cette installation, un projectile de carte a signature
	// differente (Forge [15,15,17]) se decode a la mauvaise echelle. Verrou tenu par l'appelant.
	if lay, _, err := DetectI0Layout(dir); err == nil {
		saved := WorldObjectPrecision
		SetWorldObjectPrecisionFromLayout(lay)
		defer func() { WorldObjectPrecision = saved }()
	}
	type key struct{ slot, gen uint32 }
	lives := map[key][]ProjectileSample{}
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			t.Fatalf("chunk_%02d illisible : %v", c, err)
		}
		for _, p := range WalkPackets(data) {
			if p.Type != PacketTypeDelta {
				continue
			}
			pay := p.Payload(data)
			for _, s := range scanProjectileRecords(pay, band, wr) {
				s.TimestampUS, s.Chunk = p.TimestampUS, c
				k := key{s.slot, s.gen}
				lives[k] = append(lives[k], s.ProjectileSample)
			}
		}
	}
	var out []detoDeton
	for _, pts := range lives {
		sort.Slice(pts, func(i, j int) bool { return lessSample(pts[i], pts[j]) })
		for _, seg := range splitLives(pts) {
			out = append(out, detoFromLife(seg))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ts < out[j].ts })
	return out
}

// detoFromLife construit un point de detonation depuis une vie de projectile (>=3 points).
func detoFromLife(seg []ProjectileSample) detoDeton {
	last := seg[len(seg)-1]
	first := seg[0]
	return detoDeton{
		ts:       last.TimestampUS,
		pos:      geoAimSample{ts: last.TimestampUS, x: float64(last.X), y: float64(last.Y), z: float64(last.Z)},
		atRest:   last.AtRest,
		birthTs:  first.TimestampUS,
		birthPos: geoAimSample{ts: first.TimestampUS, x: float64(first.X), y: float64(first.Y), z: float64(first.Z)},
		nPts:     len(seg),
	}
}

// detoShooterAt rend la position+visee du tireur d'un tir a l'instant du tir.
func detoShooterAt(s geoShot, tracks map[uint32][]geoAimSample) (geoAimSample, bool) {
	return geoLookup(tracks[uint32(geoActiveBase+int(s.att))], s.ts, detoPosTol)
}

// detoBirthMatch apparie une detonation a son tir lourd par la NAISSANCE : le tir lourd dont
// l'instant encadre la naissance et dont le tireur est le plus proche du point de naissance
// (< detoBirthDistMax). C'est le lien VALIDE de projectiles.go (naissance 70/70), qui sert
// d'oracle a l'attribution geometrique. Rend le tir, le tireur a T_tir, et la validite.
func detoBirthMatch(d detoDeton, heavy []geoShot, tracks map[uint32][]geoAimSample) (geoShot, geoAimSample, bool) {
	lo := uint64(0)
	if d.birthTs > detoBirthPreW {
		lo = d.birthTs - detoBirthPreW
	}
	hi := d.birthTs + detoBirthPostW
	best, bestSh, ok := geoShot{}, geoAimSample{}, false
	bestDist := detoBirthDistMax
	i := sort.Search(len(heavy), func(i int) bool { return heavy[i].ts >= lo })
	for ; i < len(heavy) && heavy[i].ts <= hi; i++ {
		sh, oks := detoShooterAt(heavy[i], tracks)
		if !oks {
			continue
		}
		dd := geoDist(sh, d.birthPos)
		if dd < bestDist {
			best, bestSh, ok, bestDist = heavy[i], sh, true, dd
		}
	}
	return best, bestSh, ok
}

// detoCandidate : un tir lourd candidat d'une detonation, geometrie resolue a T_tir.
type detoCandidate struct {
	shot     geoShot
	shooter  geoAimSample
	dist     float64 // tireur -> point de detonation
	dtS      float64 // (T_deto - T_tir) en secondes
	angle    float64 // visee du tireur vs direction tireur -> DETONATION (deg)
	hasAngle bool
}

// detoFindCandidates rend les tirs lourds dont T_tir tombe dans [T_deto - vol, T_deto], tireur
// resolu, avec l'angle visee->detonation (l'axe de visee, contrairement a la victime hors-axe).
func detoFindCandidates(d detoDeton, heavy []geoShot, tracks map[uint32][]geoAimSample) []detoCandidate {
	lo := uint64(0)
	if d.ts > detoFlightW {
		lo = d.ts - detoFlightW
	}
	i := sort.Search(len(heavy), func(i int) bool { return heavy[i].ts >= lo })
	var out []detoCandidate
	for ; i < len(heavy) && heavy[i].ts <= d.ts; i++ {
		sh, oks := detoShooterAt(heavy[i], tracks)
		if !oks {
			continue
		}
		c := detoCandidate{shot: heavy[i], shooter: sh, dist: geoDist(sh, d.pos), dtS: float64(d.ts-heavy[i].ts) / 1e6}
		c.angle, c.hasAngle = geoAngleToVictim(sh, d.pos) // meme geometrie, cible = detonation
		out = append(out, c)
	}
	return out
}

// detoDistinctShooters compte les tireurs (FilmIndex) distincts d'un jeu de candidats.
func detoDistinctShooters(cands []detoCandidate) int {
	seen := map[int]bool{}
	for _, c := range cands {
		seen[c.shot.film] = true
	}
	return len(seen)
}

// detoScore : cout d'un candidat = alignement (deg) + penalite de temps de vol (0..90).
func detoScore(c detoCandidate, speed float64) float64 {
	align := 60.0
	if c.hasAngle {
		align = c.angle
	}
	if speed <= 0 {
		speed = geoDefSpeedU
	}
	err := c.dist/speed - c.dtS
	if err < 0 {
		err = -err
	}
	ft := err / geoFlightTolS
	if ft > 1 {
		ft = 1
	}
	return align + 90*ft
}

// detoGeometricWinner rend le candidat de cout minimal et la marge sur le second.
func detoGeometricWinner(cands []detoCandidate, speedByWid map[uint64]float64) (detoCandidate, float64, bool) {
	best, ok := detoCandidate{}, false
	bestS, secondS := 0.0, 1e18
	for _, c := range cands {
		s := detoScore(c, geoSpeedFor(speedByWid, c.shot.wid))
		switch {
		case !ok || s < bestS:
			secondS, bestS, best, ok = bestS, s, c, true
		case s < secondS:
			secondS = s
		}
	}
	return best, secondS - bestS, ok
}

// detoTemporalWinner rend le tir lourd le plus RECENT (jointure naive « dernier tir avant »).
func detoTemporalWinner(cands []detoCandidate) (detoCandidate, bool) {
	best, ok := detoCandidate{}, false
	for _, c := range cands {
		if !ok || c.shot.ts > best.shot.ts {
			best, ok = c, true
		}
	}
	return best, ok
}

// detoBestAligned rend le candidat le mieux aligne sur la detonation (angle min).
func detoBestAligned(cands []detoCandidate) (detoCandidate, bool) {
	best, ok := detoCandidate{}, false
	for _, c := range cands {
		if c.hasAngle && (!ok || c.angle < best.angle) {
			best, ok = c, true
		}
	}
	return best, ok
}

// detoTouchPos rend la position monde de la victime d'une touche non-bipede a l'instant de la
// touche, et sa validite.
func detoTouchPos(tc geoTouch, tracks map[uint32][]geoAimSample) (geoAimSample, bool) {
	if !tc.hasVict {
		return geoAimSample{}, false
	}
	return geoLookup(tracks[tc.victSlot], tc.ts, detoPosTol)
}

// detoNearestTouch rend la touche non-bipede la plus proche (espace+temps) d'une detonation dans
// le rayon d'eclaboussure, et sa position victime.
func detoNearestTouch(d detoDeton, touch []geoTouch, tracks map[uint32][]geoAimSample) (geoTouch, geoAimSample, bool) {
	best, bestPos, ok := geoTouch{}, geoAimSample{}, false
	bestD := detoSplashRadius
	for _, tc := range touch {
		if !tc.hasVict {
			continue
		}
		dt := d.ts - tc.ts
		if tc.ts > d.ts {
			dt = tc.ts - d.ts
		}
		if dt > detoSplashTimeW {
			continue
		}
		vp, okv := detoTouchPos(tc, tracks)
		if !okv {
			continue
		}
		dd := geoDist(vp, d.pos)
		if dd < bestD {
			best, bestPos, ok, bestD = tc, vp, true, dd
		}
	}
	return best, bestPos, ok
}

// detoAngleBucket incremente l'histogramme <5,<15,<30,<45,<60,>=60 degres.
func detoAngleBucket(a float64, h *[6]int) {
	switch {
	case a < 5:
		h[0]++
	case a < 15:
		h[1]++
	case a < 30:
		h[2]++
	case a < 45:
		h[3]++
	case a < 60:
		h[4]++
	default:
		h[5]++
	}
}

// detoHeavyDiag ventile les tirs lourds par arme et mesure le taux de resolution du tireur a
// l'instant du tir (diagnostic : distingue « pas de lanceur direct tire » de « base bipede mal
// resolue »). Un DIRECT non resolu signale une base fausse ; aucun DIRECT signale l'absence.
func detoHeavyDiag(t *testing.T, heavy []geoShot, tracks map[uint32][]geoAimSample) {
	t.Helper()
	type acc struct{ n, resolved int }
	by := map[string]*acc{}
	var direct, directResolved int
	for _, s := range heavy {
		a := by[s.name]
		if a == nil {
			a = &acc{}
			by[s.name] = a
		}
		a.n++
		_, ok := detoShooterAt(s, tracks)
		if ok {
			a.resolved++
		}
		if s.direct {
			direct++
			if ok {
				directResolved++
			}
		}
	}
	type row struct {
		name string
		a    *acc
	}
	var rows []row
	for nm, a := range by {
		rows = append(rows, row{nm, a})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].a.n > rows[j].a.n })
	t.Logf("DIAG tirs lourds par arme (tireur resolu a T_tir sous base %d) :", geoActiveBase)
	for _, r := range rows {
		t.Logf("   %-20s : %d tirs · tireur resolu %d (%.1f %%)%s", r.name, r.a.n, r.a.resolved,
			lot1Pct(r.a.resolved, r.a.n), map[bool]string{true: " [DIRECT]", false: ""}[geoIsDirect(r.name)])
	}
	t.Logf("   tirs DIRECTS %d · tireur resolu %d (%.1f %%) — 0 direct = pas de lanceur ; direct non resolu = base suspecte",
		direct, directResolved, lot1Pct(directResolved, direct))
}

// detoBirthProbe : pour chaque tir DIRECT, distance a la naissance de projectile la plus proche
// dans une fenetre temporelle LARGE (+-500 ms), SANS seuil de distance. Median petit = les
// naissances sont la (l'appariement echoue sur un reglage) ; median grand = pas de projectile de
// ce tir dans la fenetre bornee (echelle carte, ou lanceur hitscan/arc sans ti=41 exploitable).
func detoBirthProbe(t *testing.T, detons []detoDeton, heavy []geoShot, tracks map[uint32][]geoAimSample) {
	t.Helper()
	bts := make([]uint64, len(detons))
	for i, d := range detons {
		bts[i] = d.birthTs
	}
	order := make([]int, len(detons))
	for i := range order {
		order[i] = i
	}
	sort.Slice(order, func(a, b int) bool { return bts[order[a]] < bts[order[b]] })
	sortedBts := make([]uint64, len(order))
	for i, o := range order {
		sortedBts[i] = detons[o].birthTs
	}
	const probeW = uint64(500_000)
	var dists []float64
	n := 0
	for _, s := range heavy {
		if !s.direct {
			continue
		}
		sh, ok := detoShooterAt(s, tracks)
		if !ok {
			continue
		}
		n++
		lo := uint64(0)
		if s.ts > probeW {
			lo = s.ts - probeW
		}
		hi := s.ts + probeW
		i := sort.Search(len(sortedBts), func(i int) bool { return sortedBts[i] >= lo })
		best := 1e18
		for ; i < len(order) && sortedBts[i] <= hi; i++ {
			if d := geoDist(sh, detons[order[i]].birthPos); d < best {
				best = d
			}
		}
		if best < 1e17 {
			dists = append(dists, best)
		}
	}
	t.Logf("DIAG naissance la plus proche d'un tir DIRECT (fenetre +-500ms, sans seuil) : %d tirs directs · %d avec une naissance dans la fenetre · distance mediane %.2f u",
		n, len(dists), geoMedian(dists))
}
