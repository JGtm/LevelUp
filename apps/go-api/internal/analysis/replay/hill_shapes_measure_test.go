package replay

// hill_shapes_measure_test.go — LOT C-ter VOLET 2 : LES FORMES DE COLLINE CONTRE LES PERIODES DU FILM.
//
// LA QUESTION. Le catalogue de formes n'a jamais porte de role de colline : les temoins KOTH du
// lot C-bis ont ete apparies aux formes de Bastion et d'Extraction, faute de mieux. L'inventaire
// des variantes de carte (CT.2.1) a fait ressortir des volumes candidats — des boites et des
// cylindres neutres, porteurs de deux labels non resolus, en nombre egal sur les quatre cartes
// des films KOTH. Ce fichier confronte ces volumes a l'ORACLE FORT du plan : la grappe des
// positions pendant chaque periode de garde de la jauge. Une vraie colline CONTIENT la grappe.
//
// DEFINITIONS ET SEUILS, ECRITS AVANT LA MESURE (jamais abaisses ensuite) :
//
//	(D1) PERIODE     une rampe BRUTE de la jauge (tag 3 de ti=13) : t0..tPeak, telle que
//	                 `zoneRampsOf` la decoupe en production — sans fusion, sans dependance au
//	                 catalogue. C'est le denominateur.
//	(D2) APPARIEMENT la forme candidate qui compte le PLUS de positions TENUES (toutes vies) a
//	                 distance <= hillTolM pendant la periode ; egalite ou zero = non appariee.
//	(D3) DEDANS      la periode « tombe dans » sa forme appariee si au moins hillOccupancyMin
//	                 (50 %) de ses frames ont >= 1 position tenue a distance <= hillTolM de la
//	                 forme : une colline en cours de capture est OCCUPEE. hillTolM = 0,5 m
//	                 absorbe le quantum vertical (le bas des volumes est a down_z = 0 : un pied
//	                 pose au ras du sol oscille autour de la base) sans elargir la forme de facon
//	                 sensible (demi-cotes de 1,25 a 5,25 m ; la production tolere 5 m). Le taux
//	                 STRICT (tolerance nulle) est publie a cote, a titre de diagnostic.
//
//	                 POSITION TENUE (correction R1, apres la premiere passe sur `01e1f945`) : le
//	                 film ne replique une position QU'AU CHANGEMENT (`decimateTracks` pose un
//	                 point par frame ou un record existe) ; une vie qui tient la colline SANS
//	                 BOUGER n'emet rien, et l'occupation lue sur les seuls points publies
//	                 tombait a 30-45 % pendant des rampes pourtant nettes (une vie, un point tous
//	                 les 2-3 frames). La position d'une vie est donc TENUE entre deux de ses
//	                 points (jusqu'a hillHoldMaxFrames), jamais au-dela de son dernier point (la
//	                 vie s'arrete a la mort). L'unite change (position tenue au lieu de point
//	                 publie), le seuil ne change pas ; le taux sur points bruts reste publie.
//
//	                 EMISSIONS CROISSANTES (correction R3, apres la deuxieme passe sur `01e1f945`) :
//	                 la tenue des positions n'a rien change (les points sont DENSES : ecart p99 = 2
//	                 frames), et un balayage de decalage temporel a exclu un defaut d'horloge
//	                 (pic a 0). La cause est dans la definition de la rampe : `findZoneRamps`
//	                 retient une suite NON DECROISSANTE, donc une « rampe » englobe les PLATEAUX
//	                 (jauge tenue quand la colline se vide, aucune emission), et l'occupation par
//	                 frame de toute la rampe est bornee par construction (mesure : 62 % en moyenne
//	                 a l'alignement, 45 % au hasard). La clause D3 se juge donc aux INSTANTS OU LA
//	                 JAUGE MONTE : la periode « tombe dans » sa forme si >= hillOccupancyMin de ses
//	                 emissions croissantes ont >= 1 position tenue a distance <= hillTolM de la
//	                 forme dans les hillStepWindow frames autour de l'emission. Meme seuil ; les
//	                 taux par frame (D3 premiere forme) restent publies.
//	(D4) GATE        periodes appariees ET dedans / toutes les periodes >= hillGateRate (90 %)
//	                 sur chacun des 4 films.
//	(D5) TEMOINS     (i) formes PERMUTEES : la forme i posee au centre de la forme i+1 — memes
//	                 tailles, mauvais endroits ; (ii) periodes DECALEES de +20 s sur les vraies
//	                 formes. Les deux doivent tomber nettement sous le taux reel, et le NIVEAU
//	                 DU HASARD (part des frames du match ou une colline quelconque est occupee)
//	                 est publie.
//
// LE MEME INSTRUMENT SERT LES DEUX SOURCES DE FORMES : un `.mvar` et deux hashs de label
// (HILL_MVAR + HILL_LABEL le role, HILL_INCLUDE le filtre de mode que la forme doit AUSSI
// porter — l'inventaire de CT.2.1 a montre que le label de role est partage par deux objets
// de minigame par carte de developpeur, que le filtre ecarte) ou le catalogue servi sous le
// role `hill` (par defaut, la mesure de CT.2.3). Comparer les deux est un controle : le catalogue doit
// porter exactement ce que le fichier de carte contient.
//
// SOUS GARDE D'ENVIRONNEMENT (`ZONE_FILM`), UN film par processus, avant-plan (D17) :
//
//	$env:CGO_ENABLED=0
//	$env:ZONE_FILM="C:/Users/Guillaume/Projects/LevelUp/data/cache/film_chunks/01e1f945"
//	$env:HILL_MVAR="C:/.../catalyst_map.mvar"   # optionnel ; sinon le catalogue, role hill
//	$env:HILL_LABEL="-767961569"                 # optionnel avec HILL_MVAR (role)
//	$env:HILL_INCLUDE="2133978317"               # optionnel avec HILL_MVAR (filtre de mode)
//	go test -count=1 -run TestHillShapesMeasure -v -timeout 30m ./internal/analysis/replay/

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"testing"

	"levelup/go-api/internal/analysis/replay/mapvar"
)

const (
	hillMvarEnv    = "HILL_MVAR"
	hillLabelEnv   = "HILL_LABEL"
	hillIncludeEnv = "HILL_INCLUDE"

	hillTolM         = 0.5
	hillOccupancyMin = 0.5
	hillGateRate     = 0.90
	hillShiftMS      = 20000
	// hillHoldMaxFrames borne la tenue d'une position entre deux points d'une meme vie : 60 s a
	// 100 ms. Un trou plus long dans une vie vivante ne se produit pas (les images-cle du film
	// re-emettent les positions), et s'il se produisait il vaut mieux perdre l'occupation que
	// l'inventer.
	hillHoldMaxFrames = 600
	// hillStepWindow : demi-fenetre, en frames, autour d'une emission croissante de la jauge (R3).
	hillStepWindow = 5
)

// hillFormes rend les volumes de colline a mesurer, avec leur origine (pour le rapport).
func hillFormes(t *testing.T, mapID string) ([]Zone, string) {
	t.Helper()
	path := os.Getenv(hillMvarEnv)
	if path == "" {
		return p2aZones(t, mapID, mapvar.RoleHill), "catalogue (role hill)"
	}
	buf, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("lecture de %s : %v", path, err)
	}
	v, err := mapvar.Parse(buf)
	if err != nil {
		t.Fatalf("parse de %s : %v", path, err)
	}
	label := hillEnvHash(t, hillLabelEnv, -767961569)
	include := hillEnvHash(t, hillIncludeEnv, 0)
	var out []Zone
	for _, o := range v.Objects {
		porte, filtre := false, include == 0
		for _, h := range o.Labels {
			if h == label {
				porte = true
			}
			if include != 0 && h == include {
				filtre = true
			}
		}
		if !porte || !filtre || o.ShapeRaw == nil {
			continue
		}
		sh := o.Shape()
		vol, err := mapvar.NewVolume(o.Pos, sh)
		if err != nil {
			t.Logf("objet %d : forme inutilisable (%v) — ecarte", o.Index, err)
			continue
		}
		out = append(out, Zone{Role: "candidat", InstanceID: o.InstanceID, ObjectIdx: o.Index,
			TeamIndex: o.TeamIndex, Center: o.Pos, Volume: vol, Shape: sh})
	}
	sortZonesSpatially(out)
	return out, fmt.Sprintf("%s (role %d, filtre %d)", path, label, include)
}

// hillEnvHash lit un hash de label int32 dans l'environnement (defaut si absent).
func hillEnvHash(t *testing.T, env string, def int32) int32 {
	t.Helper()
	s := os.Getenv(env)
	if s == "" {
		return def
	}
	v, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		t.Fatalf("%s=%q illisible : %v", env, s, err)
	}
	return int32(v)
}

// hillHeldPoints indexe par frame les positions TENUES : chaque vie garde sa derniere position
// entre deux points (au plus hillHoldMaxFrames), et rien apres son dernier point (R1). Publie
// aussi la densite des points bruts, pour que le rapport dise sur quoi la tenue s'appuie.
func hillHeldPoints(t *testing.T, tracks []Track, frames int) map[int][]Point {
	t.Helper()
	out := map[int][]Point{}
	raw, held, gaps := 0, 0, []int{}
	for _, tr := range tracks {
		for k, p := range tr.Points {
			raw++
			end := p.T
			if k+1 < len(tr.Points) {
				next := tr.Points[k+1].T
				gaps = append(gaps, next-p.T)
				end = min(next-1, p.T+hillHoldMaxFrames)
			}
			for f := p.T; f <= end && f < frames; f++ {
				out[f] = append(out[f], p)
				held++
			}
		}
	}
	sort.Ints(gaps)
	q := func(r float64) int {
		if len(gaps) == 0 {
			return 0
		}
		return gaps[min(len(gaps)-1, int(r*float64(len(gaps))))]
	}
	t.Logf("  POSITIONS : %d points bruts, %d positions tenues ; ecart entre deux points d'une vie"+
		" p50 %d, p90 %d, p99 %d, max %d frames", raw, held, q(0.5), q(0.9), q(0.99), q(0.999999))
	return out
}

// hillPermute pose la forme i au centre de la forme i+1 : le temoin des formes permutees.
func hillPermute(zones []Zone) []Zone {
	n := len(zones)
	out := make([]Zone, n)
	for i := range zones {
		j := (i + 1) % n
		z := zones[i]
		d := mapvar.Vec3{X: zones[j].Center.X - z.Center.X, Y: zones[j].Center.Y - z.Center.Y,
			Z: zones[j].Center.Z - z.Center.Z}
		z.Volume = z.Volume.Translate(d)
		z.Center = zones[j].Center
		out[i] = z
	}
	return out
}

// hillNear dit si une position tenue de la frame est a distance <= hillTolM du volume.
func hillNear(z Zone, pts map[int][]Point, f int) (near, strict bool) {
	for _, p := range pts[f] {
		d := z.Volume.DistanceTo(mapvar.Vec3{X: float64(p.X), Y: float64(p.Y), Z: float64(p.Z)})
		if d <= hillTolM {
			near = true
		}
		if d == 0 {
			strict = true
		}
	}
	return near, strict
}

// hillVerdict est le resultat d'une periode contre un jeu de formes.
type hillVerdict struct {
	ref       int     // forme appariee (rang dans le jeu), -1 = aucune
	votes     int     // positions tenues a distance <= tol de la forme appariee
	occupancy float64 // part des FRAMES de la periode avec >= 1 position dedans (tolerance)
	strict    float64 // idem, tolerance nulle
	steps     int     // emissions croissantes de la jauge dans la periode (R3)
	stepOcc   float64 // part des emissions croissantes ou la forme est occupee (R3)
	inside    bool    // D3 tenue (emissions croissantes, R3)
	insideFrm bool    // D3 premiere forme (frames)
}

// hillPair applique D2 : la forme qui compte le plus de positions tenues a distance <= tol.
func hillPair(zones []Zone, pts map[int][]Point, t0, t1 int) (int, int) {
	votes := make([]int, len(zones))
	for f := t0; f <= t1; f++ {
		for _, p := range pts[f] {
			w := mapvar.Vec3{X: float64(p.X), Y: float64(p.Y), Z: float64(p.Z)}
			for i, z := range zones {
				if z.Volume.DistanceTo(w) <= hillTolM {
					votes[i]++
				}
			}
		}
	}
	best, bestN, tie := -1, 0, false
	for i, n := range votes {
		switch {
		case n > bestN:
			best, bestN, tie = i, n, false
		case n == bestN && n > 0:
			tie = true
		}
	}
	if tie {
		return -1, 0
	}
	return best, bestN
}

// hillJuge applique D2 et D3 (frames ET emissions croissantes) a une rampe, decalee de `shift`.
func hillJuge(zones []Zone, pts map[int][]Point, ser zoneSeries, r zoneRamp, shift, frames int) hillVerdict {
	t0, t1 := r.t0+shift, r.tPeak+shift
	if t1 >= frames {
		t1 = frames - 1
	}
	if t0 > t1 {
		return hillVerdict{ref: -1}
	}
	best, votes := hillPair(zones, pts, t0, t1)
	if best < 0 {
		return hillVerdict{ref: -1}
	}
	occ, strict := 0, 0
	for f := t0; f <= t1; f++ {
		n, s := hillNear(zones[best], pts, f)
		if n {
			occ++
		}
		if s {
			strict++
		}
	}
	nf := float64(t1 - t0 + 1)
	v := hillVerdict{ref: best, votes: votes, occupancy: float64(occ) / nf, strict: float64(strict) / nf}
	v.insideFrm = v.occupancy >= hillOccupancyMin
	steps := hillSteps(ser, r.slot, r.t0, r.tPeak)
	v.steps = len(steps)
	sOcc := 0
	for _, s := range steps {
		in := false
		for f := s + shift - hillStepWindow; f <= s+shift+hillStepWindow && !in; f++ {
			in, _ = hillNear(zones[best], pts, f)
		}
		if in {
			sOcc++
		}
	}
	if v.steps > 0 {
		v.stepOcc = float64(sOcc) / float64(v.steps)
	}
	v.inside = v.steps > 0 && v.stepOcc >= hillOccupancyMin
	return v
}

// hillSteps rend les frames des emissions CROISSANTES de la jauge d'un slot dans [t0, t1] (R3).
func hillSteps(ser zoneSeries, slot uint32, t0, t1 int) []int {
	ss := ser.gauge[slot]
	var out []int
	for k := 1; k < len(ss); k++ {
		if ss[k].t < t0 || ss[k].t > t1 {
			continue
		}
		if ss[k].v > ss[k-1].v {
			out = append(out, ss[k].t)
		}
	}
	return out
}

// hillBilan est le compte de D4 sur toutes les rampes contre un jeu de formes.
type hillBilan struct {
	paired, insideSteps, insideFrames, total int
}

func (b hillBilan) String() string {
	return fmt.Sprintf("appariees %d/%d, DEDANS (emissions croissantes) %d/%d = %.1f %%, dedans (frames) %d/%d = %.1f %%",
		b.paired, b.total, b.insideSteps, b.total, 100*p2aRate(b.insideSteps, b.total),
		b.insideFrames, b.total, 100*p2aRate(b.insideFrames, b.total))
}

// hillMesure applique D1..D4 a toutes les rampes contre un jeu de formes ; `shift` decale les
// periodes (temoin temporel).
func hillMesure(zones []Zone, ramps []zoneRamp, pts map[int][]Point, ser zoneSeries, frames, shift int) hillBilan {
	b := hillBilan{total: len(ramps)}
	for _, r := range ramps {
		v := hillJuge(zones, pts, ser, r, shift, frames)
		if v.ref >= 0 {
			b.paired++
		}
		if v.inside {
			b.insideSteps++
		}
		if v.insideFrm {
			b.insideFrames++
		}
	}
	return b
}

// hillHasard rend la part des frames du match ou AU MOINS UNE forme est occupee (tolerance).
func hillHasard(zones []Zone, pts map[int][]Point, frames int) float64 {
	occ := 0
	for f := 0; f < frames; f++ {
		for _, z := range zones {
			if n, _ := hillNear(z, pts, f); n {
				occ++
				break
			}
		}
	}
	return float64(occ) / float64(frames)
}

// TestHillShapesMeasure — voir l'en-tete.
func TestHillShapesMeasure(t *testing.T) {
	dir := p2aRequireFilm(t)
	short, film := p2aFilmOf(t, dir)
	if film.Mode != "KOTH" {
		t.Skipf("film %s hors corpus KOTH (%s)", short, film.Mode)
	}
	hills, source := hillFormes(t, film.MapID)
	if len(hills) < 2 {
		t.Fatalf("%d forme(s) de colline depuis %s — rien a mesurer", len(hills), source)
	}
	quant := p2aQuant(t, film.Carte)
	src := p2aSource(t, dir)
	sc := p2bScan(t, dir)
	doc, origin := p2bBuild(t, dir, short, quant, ZoneInput{
		Scanned: true, Reads: sc.Reads, Zones: hills, Roles: "hill", Hill: true,
	}, p2aCaptures(src, film))
	t.Logf("FILM %s (%s, %s) — %d formes de colline depuis %s ; %d trajectoires, %d frames de %d ms",
		short, film.Mode, film.Carte, len(hills), source, len(doc.Tracks), doc.FrameCount,
		doc.FrameIntervalMS)
	for i, z := range hills {
		t.Logf("  forme %d : %s centre (%.2f ; %.2f ; %.2f) idx %d inst %d team %d",
			i, hillShapeDesc(z.Shape), z.Center.X, z.Center.Y, z.Center.Z, z.ObjectIdx,
			z.InstanceID, z.TeamIndex)
	}

	c := zoneCtx{origin: origin, step: uint64(doc.FrameIntervalMS) * 1000,
		frames: doc.FrameCount, intervalMS: doc.FrameIntervalMS, tracks: doc.Tracks}
	ser := zoneSeriesOf(sc.Reads, c)
	ramps := zoneRampsOf(ser)
	sort.SliceStable(ramps, func(i, j int) bool { return ramps[i].t0 < ramps[j].t0 })
	pts := hillHeldPoints(t, doc.Tracks, doc.FrameCount)
	if len(ramps) == 0 {
		t.Fatalf("aucune rampe de jauge sur ce film — la mesure n'a pas de denominateur")
	}
	t.Logf("  RAMPES : %d (premiere a la frame %d = %.1f s ; %d slots de jauge)",
		len(ramps), ramps[0].t0, float64(ramps[0].t0*doc.FrameIntervalMS)/1000, len(ser.gauge))

	hillTableau(t, hills, ramps, pts, ser, doc)

	shiftFrames := hillShiftMS / max(doc.FrameIntervalMS, 1)
	reel := hillMesure(hills, ramps, pts, ser, doc.FrameCount, 0)
	perm := hillMesure(hillPermute(hills), ramps, pts, ser, doc.FrameCount, 0)
	shift := hillMesure(hills, ramps, pts, ser, doc.FrameCount, shiftFrames)
	t.Logf("  MESURE (D4, seuil %.0f %%) : %s", 100*hillGateRate, reel)
	t.Logf("  TEMOIN formes permutees   : %s", perm)
	t.Logf("  TEMOIN periodes +20 s     : %s", shift)
	t.Logf("  HASARD : une colline occupee sur %.1f %% des frames du match (permutees : %.1f %%)",
		100*hillHasard(hills, pts, doc.FrameCount), 100*hillHasard(hillPermute(hills), pts, doc.FrameCount))

	hillProduction(t, doc, hills)
	hillAvantApres(t, film, ser, c, hills, pts, doc.FrameCount)
	verdict := "TENU"
	if p2aRate(reel.insideSteps, reel.total) < hillGateRate {
		verdict = "NON TENU"
	}
	t.Logf("  GATE 2 (ce film) : %s", verdict)
}

// hillTableau publie chaque rampe : sa forme appariee, ses votes, son occupation.
func hillTableau(t *testing.T, hills []Zone, ramps []zoneRamp, pts map[int][]Point, ser zoneSeries, doc ReplayDocument) {
	t.Helper()
	perHill := map[int]int{}
	ms := doc.FrameIntervalMS
	for i, r := range ramps {
		v := hillJuge(hills, pts, ser, r, 0, doc.FrameCount)
		perHill[v.ref]++
		t.Logf("    rampe %2d slot %d [%6.1f s ; %6.1f s] -> forme %2d votes %4d · emissions %3d occupees %5.1f %% ·"+
			" frames %5.1f %% (stricte %5.1f %%) %s",
			i, r.slot, float64(r.t0*ms)/1000, float64(r.tPeak*ms)/1000, v.ref, v.votes,
			v.steps, 100*v.stepOcc, 100*v.occupancy, 100*v.strict, hillMark(v))
	}
	refs := make([]int, 0, len(perHill))
	for r := range perHill {
		refs = append(refs, r)
	}
	sort.Ints(refs)
	for _, r := range refs {
		t.Logf("    forme %2d : %d rampe(s)", r, perHill[r])
	}
}

func hillMark(v hillVerdict) string {
	switch {
	case v.ref < 0:
		return "NON APPARIEE"
	case !v.inside:
		return "HORS"
	}
	return "dedans"
}

// hillProduction publie ce que le chemin de PRODUCTION (buildHillStates) rend avec ces formes.
func hillProduction(t *testing.T, doc ReplayDocument, hills []Zone) {
	t.Helper()
	if doc.Coverage == nil || doc.Coverage.Zones == nil {
		t.Logf("  PRODUCTION : aucune couverture de zone publiee")
		return
	}
	cv := doc.Coverage.Zones
	actives := 0
	for _, st := range doc.ZoneStates {
		for _, sp := range st.Spans {
			if sp.Active {
				actives += sp.T1 - sp.T0 + 1
			}
		}
	}
	t.Logf("  PRODUCTION (buildHillStates sur %d formes) : methode %s · periodes %d · zones %d ·"+
		" non appariees %d · frames actives %d/%d = %.1f %%",
		len(hills), cv.Method, cv.HillPeriods, len(doc.ZoneStates), cv.Unpaired, actives,
		doc.FrameCount, 100*p2aRate(actives, doc.FrameCount))
}

// hillAvantApres — sur les cartes du catalogue, compare les periodes de PRODUCTION obtenues avec
// les formes de Bastion/Extraction (AVANT, ce que l'artefact publie aujourd'hui) et avec les
// collines (APRES) : pour chaque periode AVANT, la colline qui la contient et si elle est
// distincte de la forme AVANT (centre de la colline hors de la forme AVANT).
func hillAvantApres(t *testing.T, film p2aFilm, ser zoneSeries, c zoneCtx, hills []Zone, pts map[int][]Point, frames int) {
	t.Helper()
	avant := p2aZonesOrNil(t, film.MapID, mapvar.RoleStrongholdZone, mapvar.RoleExtractionZone)
	if len(avant) == 0 {
		t.Logf("  AVANT/APRES : sans objet (aucune forme de Bastion/Extraction au catalogue)")
		return
	}
	avant = zoneCatalogOf(avant)
	covA := &ZonesCoverage{}
	statesA := buildHillStates(avant, ser, c, covA)
	covB := &ZonesCoverage{}
	statesB := buildHillStates(zoneCatalogOf(hills), ser, c, covB)
	t.Logf("  AVANT (Bastion+Extraction, %d formes) : %d periodes, %d zones, %d rampes non appariees",
		len(avant), covA.HillPeriods, len(statesA), covA.Unpaired)
	t.Logf("  APRES (collines, %d formes)            : %d periodes, %d zones, %d rampes non appariees",
		len(hills), covB.HillPeriods, len(statesB), covB.Unpaired)
	type per struct {
		t0, t1, ref int
	}
	var pa []per
	for _, st := range statesA {
		for _, sp := range st.Spans {
			pa = append(pa, per{sp.T0, sp.T1, st.ZoneRef})
		}
	}
	sort.Slice(pa, func(i, j int) bool { return pa[i].t0 < pa[j].t0 })
	differ := 0
	for i, p := range pa {
		ref, _ := hillPair(hills, pts, p.t0, min(p.t1, frames-1))
		za := avant[p.ref]
		desc := "colline NON appariee"
		if ref >= 0 {
			h := hills[ref]
			d := za.Volume.DistanceTo(h.Center)
			distinct := d > 0
			if distinct {
				differ++
			}
			desc = fmt.Sprintf("colline %d (%.1f ; %.1f) a %.1f m de la forme AVANT — %s",
				ref, h.Center.X, h.Center.Y, d, map[bool]string{true: "DISTINCTE", false: "confondue"}[distinct])
		}
		t.Logf("    periode %2d [%5d ; %5d] AVANT zone %d %s (%.1f ; %.1f) -> %s",
			i, p.t0, p.t1, p.ref, za.Role, za.Center.X, za.Center.Y, desc)
	}
	t.Logf("  AVANT/APRES : %d periodes AVANT, %d dont la colline est DISTINCTE de la forme AVANT",
		len(pa), differ)
}

// p2aZonesOrNil : comme p2aZones, mais rend nil (au lieu de sauter) quand la carte manque.
func p2aZonesOrNil(t *testing.T, mapID string, roles ...mapvar.Role) []Zone {
	t.Helper()
	cat, err := LoadMapObjectives(p2aRefDir(t) + "/map_objectives.json")
	if err != nil {
		t.Fatalf("catalogue d'objectifs illisible : %v", err)
	}
	e, err := cat.Lookup(mapID)
	if err != nil {
		return nil
	}
	var out []Zone
	for _, r := range roles {
		out = append(out, e.ZonesOfRole(r).Zones...)
	}
	return out
}

func hillShapeDesc(s *mapvar.Shape) string {
	if s == nil {
		return "sans forme"
	}
	switch s.Family {
	case mapvar.ShapeBox:
		return fmt.Sprintf("boite %.2f x %.2f (h +%.2f/-%.2f)", *s.HalfX, *s.HalfY, s.UpZ, s.DownZ)
	case mapvar.ShapeCylinder:
		return fmt.Sprintf("cylindre r=%.2f (h +%.2f/-%.2f)", *s.Radius, s.UpZ, s.DownZ)
	}
	return string(s.Family)
}
