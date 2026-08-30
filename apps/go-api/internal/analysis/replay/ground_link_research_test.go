package replay

// ground_link_research_test.go — INSTRUMENT DE MESURE (pas de production). Trois questions
// posees par l'utilisateur le 2026-08-30, mesurees sur pieces :
//
//  A. LE LIEN PICKUP <-> OBJET AU SOL. « L'arme a un ID et des coordonnees, pourquoi ne pas
//     relier ? » Les trois refutations passees portaient sur d'AUTRES voies (suppression
//     d'entite dans le meme paquet, attachement, appariement par inventaire a 20 s). La voie
//     directe — le ramasseur est PHYSIQUEMENT SUR l'arme a l'instant du ramassage — n'a jamais
//     ete mesuree, parce que l'instant exact du ramassage n'existait pas avant le canal delta
//     du schema 24. Il existe maintenant : c'est la condition de reprise ecrite au
//     REGISTRE_REPORTS (« un oracle plus rapproche que 20 s ») qui est LEVEE.
//     Mesure : distance du ramasseur a l'objet ti=42 le plus proche vivant a cet instant,
//     contre un temoin (un AUTRE bipede vivant au meme instant — le temoin du poseur
//     d'equipement, qui rendait 11-36 m contre 0,5 m).
//
//  B. LA DISPARITION OBSERVEE. « Je veux voir quand elle est au sol et quand elle disparait »
//     — pas une minuterie. Le recensement des images-cles (une toutes les ~20 s) donne, par
//     vie d'objet (slot, generation), la derniere image-cle ou l'objet EST ENCORE LA et la
//     premiere ou il N'Y EST PLUS : une disparition observee, a 20 s pres. Et quand un pickup
//     (A) tombe dans cette fenetre, la disparition est datee A LA MILLISECONDE.
//
//  C. L'EQUIPEMENT TOMBE-T-IL AU SOL A LA MORT ? Le canal i48 du schema 25 ne peut pas le
//     dire (le mourant n'emet rien) — mais les POSES ti=37 le peuvent : si l'equipement tombe
//     a la mort, des objets ti=37 naissent la ou des bipedes MEURENT. Mesure : part des poses
//     dont le bipede le plus proche a l'instant de la pose finit sa vie dans la meme seconde.
//
// GARDE : PICKUP_FILM / PICKUP_MAP, comme les autres instruments du lot.

import (
	"math"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// glSetup porte les entrees communes aux trois mesures, decodees une fois par test.
type glSetup struct {
	dir string
	wr  filmdec.Vec3Range
	pos map[uint32][]filmdec.BipedPosition
}

func glResolve(t *testing.T) glSetup {
	t.Helper()
	dir, mapName := os.Getenv("PICKUP_FILM"), os.Getenv("PICKUP_MAP")
	if dir == "" || mapName == "" {
		t.Skip("PICKUP_FILM / PICKUP_MAP absents : instrument de mesure saute")
	}
	path := filepath.Join("..", "..", "..", "..", "..", "data", "titles", "halo_infinite",
		"reference", "map_quant_bounds.json")
	cat, err := filmdec.LoadMapQuantCatalog(path)
	if err != nil {
		t.Fatalf("catalogue de bornes : %v", err)
	}
	entry, err := cat.Lookup(mapName)
	if err != nil {
		t.Fatalf("carte %q : %v", mapName, err)
	}
	// MEME PREAMBULE QUE BuildFromFilm, et il n'est pas optionnel : le verrou de process, puis
	// les largeurs d'axe DE LA CARTE pour le chemin world-object. La troisieme version de cette
	// mesure tournait aux largeurs par defaut (13/13/14) : toutes les positions ti=42 etaient
	// dequantifiees faux — mediane 42 m, temoin egal, zero verdict.
	t.Cleanup(filmdec.LockProcessDecode())
	t.Cleanup(installWorldObjectPrecision(entry, dir))
	wr := entry.Range()
	raw, err := filmdec.ScanFilmBipedPositions(dir, filmdec.ScanFilmOptions{WorldRange: &wr})
	if err != nil {
		t.Fatalf("positions : %v", err)
	}
	pos := map[uint32][]filmdec.BipedPosition{}
	for _, p := range raw {
		pos[p.Slot] = append(pos[p.Slot], p)
	}
	for s := range pos {
		sort.Slice(pos[s], func(i, j int) bool { return pos[s][i].TimestampUS < pos[s][j].TimestampUS })
	}
	return glSetup{dir: dir, wr: wr, pos: pos}
}

// glAt rend la position d'un slot a l'instant demande (echantillon le plus proche, <= 300 ms).
func glAt(pos map[uint32][]filmdec.BipedPosition, slot uint32, at uint64) (filmdec.BipedPosition, bool) {
	list := pos[slot]
	best, ok := filmdec.BipedPosition{}, false
	var bestGap uint64 = 300_001
	for _, p := range list {
		gap := at - p.TimestampUS
		if p.TimestampUS > at {
			gap = p.TimestampUS - at
		}
		if gap < bestGap {
			best, bestGap, ok = p, gap, true
		}
	}
	return best, ok && bestGap <= 300_000
}

func glDist(ax, ay, az, bx, by, bz float32) float64 {
	dx, dy, dz := float64(ax-bx), float64(ay-by), float64(az-bz)
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

// glLife est la fenetre de presence d'une vie d'objet au sol, vue des images-cles.
type glLife struct{ firstSeen, lastSeen uint64 }

// glCensus recense les vies ti=42 aux images-cles, par paire (slot, generation).
func glCensus(t *testing.T, dir string) (map[filmdec.EquipmentLifeKey]*glLife, []uint64) {
	t.Helper()
	kf, err := filmdec.ScanFilmKeyframeGroundWeapons(dir, loadoutFamilies())
	if err != nil {
		t.Fatalf("recensement images-cles : %v", err)
	}
	lives := map[filmdec.EquipmentLifeKey]*glLife{}
	seen := map[uint64]bool{}
	for _, g := range kf {
		k := filmdec.EquipmentLifeKey{Slot: g.Slot, Gen: g.Gen}
		l := lives[k]
		if l == nil {
			l = &glLife{firstSeen: g.TimestampUS, lastSeen: g.TimestampUS}
			lives[k] = l
		}
		if g.TimestampUS < l.firstSeen {
			l.firstSeen = g.TimestampUS
		}
		if g.TimestampUS > l.lastSeen {
			l.lastSeen = g.TimestampUS
		}
		seen[g.TimestampUS] = true
	}
	times := make([]uint64, 0, len(seen))
	for ts := range seen {
		times = append(times, ts)
	}
	sort.Slice(times, func(i, j int) bool { return times[i] < times[j] })
	return lives, times
}

// glRestLife est une vie d'objet ti=42 telle que la CHAINE DE PRODUCTION la decode : ses
// bornes, et sa position de repos (le dernier echantillon de sa piste delta).
type glRestLife struct {
	t0, tEnd uint64
	x, y, z  float32
}

// glRestLives rend les vies ti=42 par la chaine de production ELLE-MEME (decodeFilmPlacements
// pour la calibration MPP, puis decodeFilmPadScans) — celle qui alimente les socles affiches
// juste sur le rejeu. Les deux premieres versions de cette mesure refaisaient la chaine a la
// main et rendaient un nuage de positions fantomes (mediane 24 m puis 117 m, EGALE au temoin) :
// la lecon est gardee ici pour que personne ne la refasse.
//
// La FIN de vie est bornee par le recensement des images-cles de la MEME paire, contenu a la
// fenetre de la vie (la paire (slot, generation) reboucle : le recensement d'une paire melange
// plusieurs vies, on ne garde que les instants anterieurs au debut de la vie suivante).
func glRestLives(t *testing.T, s glSetup) []glRestLife {
	t.Helper()
	_, pst := decodeFilmPlacements(s.dir, &s.wr)
	gw := decodeFilmPadScans(s.dir, &s.wr, pst.Calibration.Widths).Weapons
	if !gw.Scanned || len(gw.Tracks) == 0 {
		t.Fatalf("chaine des socles muette : scanned=%v pistes=%d", gw.Scanned, len(gw.Tracks))
	}
	byPair := map[filmdec.EquipmentLifeKey][]filmdec.ProjectileTrack{}
	for _, tr := range gw.Tracks {
		if len(tr.Pts) == 0 {
			continue
		}
		k := filmdec.EquipmentLifeKey{Slot: tr.Slot, Gen: tr.Gen}
		byPair[k] = append(byPair[k], tr)
	}
	var out []glRestLife
	for k, list := range byPair {
		sort.Slice(list, func(i, j int) bool {
			return list[i].Pts[0].TimestampUS < list[j].Pts[0].TimestampUS
		})
		for i, tr := range list {
			first, last := tr.Pts[0], tr.Pts[len(tr.Pts)-1]
			next := uint64(math.MaxUint64)
			if i+1 < len(list) {
				next = list[i+1].Pts[0].TimestampUS
			}
			end := last.TimestampUS
			for _, seen := range gw.Keyframes.SeenUS[k] {
				if seen >= first.TimestampUS && seen < next && seen > end {
					end = seen
				}
			}
			out = append(out, glRestLife{
				t0: first.TimestampUS, tEnd: end + 21_000_000,
				x: last.X, y: last.Y, z: last.Z,
			})
		}
	}
	t.Logf("VIES ti=42 (chaine de production) = %d, sur %d pistes delta", len(out), len(gw.Tracks))
	return out
}

// TestLienPickupObjetAuSol — mesure A.
func TestLienPickupObjetAuSol(t *testing.T) {
	s := glResolve(t)

	t.Log("CRITERE (enonce avant lecture) : si le lien existe, la distance ramasseur -> objet " +
		"ti=42 le plus proche vivant a l'instant de la prise est SUB-METRIQUE en mediane, et " +
		"ecrase son temoin (un autre bipede vivant au meme instant). Reference : le poseur " +
		"d'equipement, 0,52-0,60 m contre 11-36 m.")

	loadouts, err := filmdec.ScanFilmKeyframeLoadouts(s.dir, loadoutFamilies())
	if err != nil {
		t.Fatalf("loadouts : %v", err)
	}
	changes, _, err := filmdec.ScanFilmHeldWeaponChanges(s.dir, spawnSetFrom(loadouts))
	if err != nil {
		t.Fatalf("changements d arme : %v", err)
	}
	objects := glRestLives(t, s)
	nearest := func(x, y, z float32, at uint64) (float64, bool) {
		best, ok := math.MaxFloat64, false
		for _, o := range objects {
			if at < o.t0 || at > o.tEnd {
				continue
			}
			if d := glDist(x, y, z, o.x, o.y, o.z); d < best {
				best, ok = d, true
			}
		}
		return best, ok
	}

	var dists, witness []float64
	var noPos, noCand int
	for _, ch := range changes {
		if ch.Kind != filmdec.HeldWeaponTaken && ch.Kind != filmdec.HeldWeaponSwapped {
			continue
		}
		p, ok := glAt(s.pos, ch.Slot, ch.TimestampUS)
		if !ok {
			noPos++
			continue
		}
		d, ok := nearest(p.X, p.Y, p.Z, ch.TimestampUS)
		if !ok {
			noCand++
			continue
		}
		dists = append(dists, d)
		// Temoin : le premier AUTRE bipede vivant au meme instant.
		for slot := range s.pos {
			if slot == ch.Slot {
				continue
			}
			if w, ok := glAt(s.pos, slot, ch.TimestampUS); ok {
				if wd, ok := nearest(w.X, w.Y, w.Z, ch.TimestampUS); ok {
					witness = append(witness, wd)
				}
				break
			}
		}
	}
	if len(dists) == 0 {
		t.Log("VERDICT : aucune prise mesurable.")
		return
	}
	sort.Float64s(dists)
	sort.Float64s(witness)
	q := func(v []float64, f float64) float64 { return v[int(f*float64(len(v)-1))] }
	under := func(v []float64, lim float64) int {
		n := 0
		for _, d := range v {
			if d <= lim {
				n++
			}
		}
		return n
	}
	t.Logf("PRISES MESUREES = %d (sansPosition=%d sansCandidat=%d)", len(dists), noPos, noCand)
	t.Logf("DISTANCE ramasseur -> objet le plus proche : mediane=%.2f m  p75=%.2f  p90=%.2f",
		q(dists, 0.5), q(dists, 0.75), q(dists, 0.9))
	t.Logf("SOUS 1 m = %d (%.1f %%) ; sous 2 m = %d (%.1f %%)",
		under(dists, 1), 100*float64(under(dists, 1))/float64(len(dists)),
		under(dists, 2), 100*float64(under(dists, 2))/float64(len(dists)))
	if len(witness) > 0 {
		t.Logf("TEMOIN (autre bipede, meme instant) : mediane=%.2f m  sous 2 m = %.1f %%",
			q(witness, 0.5), 100*float64(under(witness, 2))/float64(len(witness)))
	}
}

// TestDisparitionObserveeDesArmesAuSol — mesure B.
func TestDisparitionObserveeDesArmesAuSol(t *testing.T) {
	s := glResolve(t)

	lives, times := glCensus(t, s.dir)
	if len(times) < 2 {
		t.Log("VERDICT : moins de deux images-cles, rien a recenser.")
		return
	}
	lastKF := times[len(times)-1]
	var gone, persist int
	for _, l := range lives {
		if l.lastSeen < lastKF {
			gone++
		} else {
			persist++
		}
	}
	t.Logf("IMAGES-CLES = %d (intervalle median ~%d s) ; VIES ti=42 recensees = %d",
		len(times), (times[len(times)/2]-times[len(times)/2-1])/1_000_000, len(lives))
	t.Logf("DISPARUES EN COURS DE MATCH (vues puis absentes) = %d ; encore la a la fin = %d",
		gone, persist)
	t.Log("LECTURE : chaque vie a une DERNIERE image-cle ou elle est vue et une PREMIERE ou " +
		"elle manque — une fenetre de disparition OBSERVEE de ~20 s, sans minuterie. Un pickup " +
		"(mesure A) qui tombe dans cette fenetre la date a la milliseconde.")
}

// TestEquipementTombeALaMort — mesure C.
func TestEquipementTombeALaMort(t *testing.T) {
	s := glResolve(t)

	t.Log("CRITERE (enonce avant lecture) : si l'equipement tombe a la mort, une part " +
		"importante des poses ti=37 a pour plus-proche-bipede (<= 3 m, +-300 ms) un bipede " +
		"dont la vie SE TERMINE dans la meme seconde et demie.")

	poses, st, err := filmdec.ScanFilmEquipmentPlacements(s.dir, &s.wr)
	if err != nil {
		t.Fatalf("poses ti=37 : %v", err)
	}
	if !st.Scanned || len(poses) == 0 {
		t.Logf("VERDICT : pas de pose exploitable (scanned=%v poses=%d).", st.Scanned, len(poses))
		return
	}
	endOfLife := map[uint32]uint64{}
	for slot, list := range s.pos {
		endOfLife[slot] = list[len(list)-1].TimestampUS
	}

	var atDeath, aliveAfter, noOwner int
	for _, p := range poses {
		var bestSlot uint32
		best, found := 3.0, false
		for slot := range s.pos {
			b, ok := glAt(s.pos, slot, p.T0US)
			if !ok {
				continue
			}
			if d := glDist(p.X, p.Y, p.Z, b.X, b.Y, b.Z); d <= best {
				best, bestSlot, found = d, slot, true
			}
		}
		if !found {
			noOwner++
			continue
		}
		end := endOfLife[bestSlot]
		if end <= p.T0US+1_500_000 {
			atDeath++
		} else {
			aliveAfter++
		}
	}
	total := len(poses)
	t.Logf("POSES ti=37 = %d ; avec un bipede a <= 3 m = %d", total, atDeath+aliveAfter)
	t.Logf("LE PLUS PROCHE MEURT dans les 1,5 s = %d (%.1f %% des poses attribuees)",
		atDeath, 100*float64(atDeath)/float64(max(1, atDeath+aliveAfter)))
	t.Logf("LE PLUS PROCHE SURVIT = %d ; poses sans bipede proche = %d", aliveAfter, noOwner)
	t.Log("LECTURE : une majorite de morts = l'equipement TOMBE au sol a la mort, et les poses " +
		"publiees (equipmentPlacements) les montrent deja. Une majorite de survivants = ces " +
		"poses sont des deploiements volontaires, pas des lachers de mort.")
}
