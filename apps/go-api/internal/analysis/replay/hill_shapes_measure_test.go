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
//	                 formes ; (iii) ajoute apres la premiere passe (R4) : formes DEPLACEES de
//	                 (+6 m ; +6 m), memes tailles, HORS des collines — parce que (i) pose les
//	                 formes sur d'autres collines reelles, ou les joueurs vont aussi, et ne
//	                 discrimine pas le LIEU. Les temoins doivent tomber nettement sous le taux
//	                 reel, et le NIVEAU DU HASARD (part des frames du match ou une colline
//	                 quelconque est occupee) est publie.
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
//
// LES RAPPORTS que ce test imprime (tableau des rampes, chemin de production, AVANT/APRES,
// bilan des manques) vivent dans `hill_shapes_report_test.go` — scission de la revue ronde 1
// (R1-2, seuil de 500 lignes). L'instrument, lui, est ici en entier.

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
	// hillShiftM : deplacement du temoin HORS colline (R4), en metres sur x et y.
	hillShiftM = 6.0
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

// hillDeplace pose chaque forme a (+hillShiftM ; +hillShiftM) de son centre : le temoin HORS colline (R4).
func hillDeplace(zones []Zone) []Zone {
	out := make([]Zone, len(zones))
	for i, z := range zones {
		d := mapvar.Vec3{X: hillShiftM, Y: hillShiftM}
		z.Volume = z.Volume.Translate(d)
		z.Center = mapvar.Vec3{X: z.Center.X + d.X, Y: z.Center.Y + d.Y, Z: z.Center.Z}
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
	depl := hillMesure(hillDeplace(hills), ramps, pts, ser, doc.FrameCount, 0)
	t.Logf("  TEMOIN formes deplacees   : %s (deplacement +%.0f m ; +%.0f m)", depl, hillShiftM, hillShiftM)
	t.Logf("  HASARD : une colline occupee sur %.1f %% des frames du match (permutees : %.1f %%, deplacees : %.1f %%)",
		100*hillHasard(hills, pts, doc.FrameCount), 100*hillHasard(hillPermute(hills), pts, doc.FrameCount),
		100*hillHasard(hillDeplace(hills), pts, doc.FrameCount))
	hillManques(t, hills, ramps, pts, ser, doc)

	hillProduction(t, doc, hills)
	hillAvantApres(t, film, ser, c, hills, pts, doc.FrameCount)
	verdict := "TENU"
	if p2aRate(reel.insideSteps, reel.total) < hillGateRate {
		verdict = "NON TENU"
	}
	t.Logf("  GATE 2 (ce film) : %s", verdict)
}
