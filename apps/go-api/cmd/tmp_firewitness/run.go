package main

import (
	"encoding/csv"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"levelup/go-api/internal/analysis/filmdec"
)

// killWindowUS : fenêtre avant la mort dans laquelle chercher le tir mortel.
const killWindowUS = 1_200_000

// eyeHeight : hauteur de l'origine du tir au-dessus de la position d'entité du biped, en
// unités monde. MESURÉE (balayage) et rapportée telle quelle — cf. rapport.
const eyeHeight = 0.0

// run enchaîne les témoins.
func run(outDir string, lives []Life, deaths []Death, events []FireEvent,
	parts []Participant, t0, off int64) {
	longs := make([]FireEvent, 0, len(events))
	for _, e := range events {
		if e.Variant == 0 {
			longs = append(longs, e)
		}
	}
	sort.Slice(longs, func(i, j int) bool { return longs[i].TS < longs[j].TS })

	nameToLives := map[string][]int{}
	for i, l := range lives {
		if l.Player != "" {
			nameToLives[l.Player] = append(nameToLives[l.Player], i)
		}
	}
	teams := map[string]int{}
	for _, p := range parts {
		teams[p.Gamertag] = p.Team
	}

	var players []string
	for _, p := range parts {
		players = append(players, p.Gamertag)
	}
	sort.Strings(players)
	byIdx := map[int][]FireEvent{}
	for _, e := range longs {
		byIdx[e.PlayerIdx] = append(byIdx[e.PlayerIdx], e)
	}
	var idxs []int
	for i := range byIdx {
		idxs = append(idxs, i)
	}
	sort.Ints(idxs)

	// ATTRIBUTION DE RÉFÉRENCE : celle issue de la corrélation des morts (les vies sont
	// nommées par appariement fin-de-vie <-> killer_victim_pairs). C'est elle qui sert aux
	// témoins 1 et 3 ; les deux autres méthodes ne servent qu'à la CONTRÔLER.
	idxToName, aliveDiag := witness2Alive(lives, nameToLives, longs, parts)
	deadMap, violOf := witness2Dead(deaths, longs, parts, t0, off)
	// ARBITRE DISJOINT : la géométrie. Le rayon de visée doit partir d'une position d'où il
	// passe près d'un autre joueur (le record est une application de dégât). Aucune date de
	// mort n'y intervient.
	geo := geometryMatrix(lives, nameToLives, longs, idxs, players)
	printMatrix("\nmatrice géométrique (médiane de l'angle au joueur le plus proche, deg — plus bas = meilleur)",
		idxs, players, geo, "%8.1f ")
	geoMap, gBest, gSecond := bestAssignment(idxs, players, geo)
	fmt.Printf("affectation géométrique optimale (coût %.1f ; 2e meilleure %.1f) :\n", gBest, gSecond)
	agGeo, agDead, totalViol := 0, 0, 0
	for _, i := range idxs {
		mark := ""
		if geoMap[i] == idxToName[i] {
			agGeo++
		} else {
			mark += "  geo:" + geoMap[i]
		}
		if deadMap[i] == idxToName[i] {
			agDead++
		} else {
			mark += "  mort:" + deadMap[i]
		}
		totalViol += violOf[i][idxToName[i]]
		fmt.Printf("  index %d -> %-16s (%.0f%% vivant, %d tirs pendant sa mort)%s\n",
			i, idxToName[i], 100*aliveDiag[i], violOf[i][idxToName[i]], mark)
	}
	fmt.Printf("ACCORD géométrie vs morts : %d/%d index ; accord fenêtre-de-mort : %d/%d ;"+
		" violations de l'attribution retenue : %d/%d events (%.1f%%)\n",
		agGeo, len(idxs), agDead, len(idxs), totalViol, len(longs),
		100*float64(totalViol)/float64(len(longs)))

	witness2Kills(deaths, longs, idxToName, t0, off)
	witness3(longs, idxToName, parts)
	witness1(outDir, lives, nameToLives, deaths, longs, idxToName, t0, off)
	witness1Broad(lives, nameToLives, longs, idxToName, teams)
	selfAimLink(lives, longs, idxToName)
	voteLifeIndex(lives, longs, idxToName)
	dumpEvents(outDir, longs, idxToName)
}

// aliveAt : le joueur `name` a-t-il une vie (attribuée par corrélation des morts) couvrant
// l'instant tUS ?
func aliveAt(lives []Life, nameToLives map[string][]int, name string, tUS int64) bool {
	for _, li := range nameToLives[name] {
		if tUS >= lives[li].StartUS && tUS <= lives[li].EndUS {
			return true
		}
	}
	return false
}

// witness2Alive — TÉMOIN 2 (indépendant) : un joueur mort ne tire pas. Pour chaque index
// d'event de tir, on mesure la part de ses events pendant lesquels chaque joueur est VIVANT
// selon l'attribution issue de la corrélation des morts (source totalement disjointe :
// fins de vie x killer_victim_pairs). L'attribution correcte doit saturer.
func witness2Alive(lives []Life, nameToLives map[string][]int, longs []FireEvent, parts []Participant) (map[int]string, map[int]float64) {
	fmt.Printf("\n=== TEMOIN 2 : index de tir x joueur VIVANT (attribution par les morts) ===\n")
	var players []string
	for _, p := range parts {
		players = append(players, p.Gamertag)
	}
	sort.Strings(players)
	byIdx := map[int][]FireEvent{}
	for _, e := range longs {
		byIdx[e.PlayerIdx] = append(byIdx[e.PlayerIdx], e)
	}
	var idxs []int
	for i := range byIdx {
		idxs = append(idxs, i)
	}
	sort.Ints(idxs)

	score := map[int]map[string]float64{}
	fmt.Printf("%-6s", "index")
	for _, p := range players {
		fmt.Printf(" %-9.9s", p)
	}
	fmt.Println("   n")
	for _, i := range idxs {
		score[i] = map[string]float64{}
		fmt.Printf("%-6d", i)
		for _, p := range players {
			n := 0
			for _, e := range byIdx[i] {
				if aliveAt(lives, nameToLives, p, int64(e.TS)) {
					n++
				}
			}
			r := float64(n) / float64(len(byIdx[i]))
			score[i][p] = r
			fmt.Printf(" %8.0f%%", 100*r)
		}
		fmt.Printf("  %4d\n", len(byIdx[i]))
	}
	// affectation OPTIMALE EXACTE : maximiser le temps vivant = minimiser son complément
	neg := map[int]map[string]float64{}
	for i, m := range score {
		neg[i] = map[string]float64{}
		for p, v := range m {
			neg[i][p] = 1 - v
		}
	}
	idxToName, aBest, aSecond := bestAssignment(idxs, players, neg)
	fmt.Printf("affectation optimale exacte (coût %.3f ; 2e meilleure %.3f) :\n", aBest, aSecond)
	var diag, alt []float64
	for _, i := range idxs {
		fmt.Printf("  index %d -> %-16s (vivant %.0f%% de ses tirs)\n", i, idxToName[i], 100*score[i][idxToName[i]])
		diag = append(diag, score[i][idxToName[i]])
		for _, p := range players {
			if p != idxToName[i] {
				alt = append(alt, score[i][p])
			}
		}
	}
	sort.Float64s(diag)
	sort.Float64s(alt)
	fmt.Printf("DIAGONALE : médiane %.1f%% (min %.1f%%)   TEMOIN hors-diagonale : médiane %.1f%% (max %.1f%%)\n",
		100*median(diag), 100*diag[0], 100*median(alt), 100*alt[len(alt)-1])
	out := map[int]float64{}
	for _, i := range idxs {
		out[i] = score[i][idxToName[i]]
	}
	return idxToName, out
}

// deadWindowUS : durée pendant laquelle un joueur est CERTAINEMENT mort après sa mort
// (respawn Halo Infinite : jamais instantané). Fenêtre volontairement courte et sûre.
const deadWindowUS = 2_500_000

// witness2Dead — TÉMOIN 2 PRINCIPAL, indépendant du décodage des positions : un joueur mort
// ne tire pas. Pour chaque index d'event de tir et chaque joueur, on compte les events qui
// tombent dans une fenêtre où CE joueur est certainement mort (dates de killer_victim_pairs
// uniquement). L'index du joueur doit donner ZÉRO ; un mauvais appariement doit donner la
// valeur du hasard (part du temps mort du joueur).
func witness2Dead(deaths []Death, longs []FireEvent, parts []Participant, t0, off int64) (map[int]string, map[int]map[string]int) {
	fmt.Printf("\n=== TEMOIN 2 : « un joueur mort ne tire pas » (dates de mort de la base seules) ===\n")
	deadUS := map[string][][2]int64{}
	for _, d := range deaths {
		t := t0 + (d.TimeMS+off)*1000
		deadUS[d.Victim] = append(deadUS[d.Victim], [2]int64{t, t + deadWindowUS})
	}
	var players []string
	for _, p := range parts {
		players = append(players, p.Gamertag)
	}
	sort.Strings(players)
	byIdx := map[int][]FireEvent{}
	for _, e := range longs {
		byIdx[e.PlayerIdx] = append(byIdx[e.PlayerIdx], e)
	}
	var idxs []int
	for i := range byIdx {
		idxs = append(idxs, i)
	}
	sort.Ints(idxs)
	inDead := func(p string, t int64) bool {
		for _, w := range deadUS[p] {
			if t >= w[0] && t <= w[1] {
				return true
			}
		}
		return false
	}
	fmt.Printf("%-6s", "index")
	for _, p := range players {
		fmt.Printf(" %-9.9s", p)
	}
	fmt.Println("     n")
	viol := map[int]map[string]int{}
	for _, i := range idxs {
		viol[i] = map[string]int{}
		fmt.Printf("%-6d", i)
		for _, p := range players {
			n := 0
			for _, e := range byIdx[i] {
				if inDead(p, int64(e.TS)) {
					n++
				}
			}
			viol[i][p] = n
			fmt.Printf(" %9d", n)
		}
		fmt.Printf("  %4d\n", len(byIdx[i]))
	}
	cost := map[int]map[string]float64{}
	for i, m := range viol {
		cost[i] = map[string]float64{}
		for p, v := range m {
			cost[i][p] = float64(v)
		}
	}
	idxToName, cBest, cSecond := bestAssignment(idxs, players, cost)
	sumDiag, sumOff, nOff := 0, 0, 0
	fmt.Printf("affectation optimale exacte (violations totales %.0f ; 2e meilleure %.0f) :\n", cBest, cSecond)
	for _, i := range idxs {
		fmt.Printf("  index %d -> %-16s (%d tirs pendant sa propre mort, sur %d)\n",
			i, idxToName[i], viol[i][idxToName[i]], len(byIdx[i]))
		sumDiag += viol[i][idxToName[i]]
		for _, p := range players {
			if p != idxToName[i] {
				sumOff += viol[i][p]
				nOff++
			}
		}
	}
	fmt.Printf("VIOLATIONS diagonale = %d (sur %d events)   TEMOIN hors-diagonale = %.1f par case (total %d sur %d cases)\n",
		sumDiag, len(longs), float64(sumOff)/float64(maxInt(nOff, 1)), sumOff, nOff)
	return idxToName, viol
}

// witness2Kills — recoupement secondaire : l'index du dernier event de tir précédant une
// mort doit être celui du tueur.
func witness2Kills(deaths []Death, longs []FireEvent, idxToName map[int]string, t0, off int64) {
	hit, tot := 0, 0
	var dts []float64
	for _, d := range deaths {
		if d.LifeIndex < 0 {
			continue
		}
		tUS := t0 + (d.TimeMS+off)*1000
		e, ok := lastEventBefore(longs, tUS, killWindowUS)
		if !ok {
			continue
		}
		tot++
		dts = append(dts, float64(tUS-int64(e.TS))/1000)
		if idxToName[e.PlayerIdx] == d.Killer {
			hit++
		}
	}
	sort.Float64s(dts)
	fmt.Printf("recoupement secondaire (dernier tir avant la mort = le tueur) : %d/%d = %.0f%% "+
		"(hasard 12,5%%) ; écart médian tir->mort %.0f ms\n",
		hit, tot, 100*float64(hit)/float64(maxInt(tot, 1)), median(dts))
}

func lastEventBefore(longs []FireEvent, tUS int64, window int64) (FireEvent, bool) {
	i := sort.Search(len(longs), func(i int) bool { return int64(longs[i].TS) > tUS })
	if i == 0 {
		return FireEvent{}, false
	}
	e := longs[i-1]
	if tUS-int64(e.TS) > window {
		return FireEvent{}, false
	}
	return e, true
}

// witness3 — volume d'events par joueur contre shots_fired / shots_hit de la base.
func witness3(longs []FireEvent, idxToName map[int]string, parts []Participant) {
	fmt.Printf("\n=== TEMOIN 3 : volume d'events par joueur vs base ===\n")
	cnt := map[int]int{}
	wpn := map[int]map[uint64]int{}
	for _, e := range longs {
		cnt[e.PlayerIdx]++
		if wpn[e.PlayerIdx] == nil {
			wpn[e.PlayerIdx] = map[uint64]int{}
		}
		wpn[e.PlayerIdx][e.Weapon64]++
	}
	byName := map[string]Participant{}
	for _, p := range parts {
		byName[p.Gamertag] = p
	}
	var idxs []int
	for i := range cnt {
		idxs = append(idxs, i)
	}
	sort.Ints(idxs)
	fmt.Printf("%-6s %-17s %7s %10s %9s %8s %8s\n", "index", "joueur", "events", "shots_fired", "shots_hit", "ev/fired", "ev/hit")
	var rf, rh []float64
	for _, i := range idxs {
		p := byName[idxToName[i]]
		f1 := float64(cnt[i]) / float64(maxInt(p.ShotsFired, 1))
		f2 := float64(cnt[i]) / float64(maxInt(p.ShotsHit, 1))
		rf = append(rf, f1)
		rh = append(rh, f2)
		fmt.Printf("%-6d %-17s %7d %10d %9d %8.2f %8.2f\n", i, idxToName[i], cnt[i], p.ShotsFired, p.ShotsHit, f1, f2)
	}
	fmt.Printf("corrélation de Pearson events vs shots_fired = %.3f ; vs shots_hit = %.3f\n",
		pearsonEvents(cnt, idxToName, byName, true), pearsonEvents(cnt, idxToName, byName, false))
	sort.Float64s(rf)
	sort.Float64s(rh)
	fmt.Printf("ratio ev/shots_fired : médiane %.2f (min %.2f max %.2f) ; ev/shots_hit : médiane %.2f (min %.2f max %.2f)\n",
		median(rf), rf[0], rf[len(rf)-1], median(rh), rh[0], rh[len(rh)-1])
}

func pearsonEvents(cnt map[int]int, idxToName map[int]string, byName map[string]Participant, fired bool) float64 {
	var xs, ys []float64
	for i, n := range cnt {
		p := byName[idxToName[i]]
		xs = append(xs, float64(n))
		if fired {
			ys = append(ys, float64(p.ShotsFired))
		} else {
			ys = append(ys, float64(p.ShotsHit))
		}
	}
	return pearson(xs, ys)
}

func pearson(x, y []float64) float64 {
	n := float64(len(x))
	var sx, sy float64
	for i := range x {
		sx += x[i]
		sy += y[i]
	}
	mx, my := sx/n, sy/n
	var num, dx, dy float64
	for i := range x {
		num += (x[i] - mx) * (y[i] - my)
		dx += (x[i] - mx) * (x[i] - mx)
		dy += (y[i] - my) * (y[i] - my)
	}
	if dx == 0 || dy == 0 {
		return math.NaN()
	}
	return num / math.Sqrt(dx*dy)
}

// witness1 — TÉMOIN PRINCIPAL : lors d'un tir mortel, la visée décodée pointe-t-elle vers
// la victime ? On ne retient que les events de tir DE L'INDEX DU TUEUR, proches de la mort.
func witness1(outDir string, lives []Life, nameToLives map[string][]int, deaths []Death,
	longs []FireEvent, idxToName map[int]string, t0, off int64) {
	fmt.Printf("\n=== TEMOIN 1 : visée du tir mortel vs direction tireur->victime ===\n")
	nameToIdx := map[string]int{}
	for i, n := range idxToName {
		nameToIdx[n] = i
	}
	var real, real2D, ctrl []float64
	rows := [][]string{{"death_ms", "killer", "victim", "dt_ms", "weapon", "angle_deg", "angle2d_deg",
		"ctrl_deg", "dist", "sx", "sy", "sz", "vx", "vy", "vz", "ax", "ay", "az"}}
	r := rand.New(rand.NewSource(11))
	for _, d := range deaths {
		if d.LifeIndex < 0 {
			continue
		}
		ki, ok := nameToIdx[d.Killer]
		if !ok {
			continue
		}
		tUS := t0 + (d.TimeMS+off)*1000
		e, ok := nearestAimEventOf(longs, ki, tUS)
		if !ok {
			continue
		}
		sp, okS := posOfPlayerAt(lives, nameToLives, d.Killer, int64(e.TS))
		if !okS {
			continue
		}
		vp, okV := posOfPlayerAt(lives, nameToLives, d.Victim, int64(e.TS))
		if !okV {
			continue
		}
		dir := [3]float64{float64(vp.X - sp.X), float64(vp.Y - sp.Y), float64(vp.Z-sp.Z) - eyeHeight}
		aim := [3]float64{float64(e.Aim[0]), float64(e.Aim[1]), float64(e.Aim[2])}
		a := angleDeg(aim, dir)
		a2 := angleDeg([3]float64{aim[0], aim[1], 0}, [3]float64{dir[0], dir[1], 0})
		cd := -1.0
		others := aliveOthers(lives, nameToLives, d.Killer, d.Victim, int64(e.TS))
		if len(others) > 0 {
			o := others[r.Intn(len(others))]
			cd = angleDeg(aim, [3]float64{float64(o.X - sp.X), float64(o.Y - sp.Y), float64(o.Z-sp.Z) - eyeHeight})
			ctrl = append(ctrl, cd)
		}
		real = append(real, a)
		real2D = append(real2D, a2)
		rows = append(rows, []string{
			strconv.FormatInt(d.TimeMS, 10), d.Killer, d.Victim,
			strconv.FormatInt((int64(e.TS)-tUS)/1000, 10),
			fmt.Sprintf("0x%016X", e.Weapon64),
			fmt.Sprintf("%.2f", a), fmt.Sprintf("%.2f", a2), fmt.Sprintf("%.2f", cd),
			fmt.Sprintf("%.2f", norm3(dir)),
			f(sp.X), f(sp.Y), f(sp.Z), f(vp.X), f(vp.Y), f(vp.Z),
			f(e.Aim[0]), f(e.Aim[1]), f(e.Aim[2]),
		})
	}
	report("tirs mortels", real, real2D, ctrl)
	writeCSV(filepath.Join(outDir, "witness1_kills.csv"), rows)
}

// witness1Broad — même test sur TOUS les events porteurs d'une visée : l'angle minimal vers
// un ENNEMI vivant, contre le témoin « angle minimal vers un COÉQUIPIER vivant ».
func witness1Broad(lives []Life, nameToLives map[string][]int, longs []FireEvent,
	idxToName map[int]string, teams map[string]int) {
	fmt.Printf("\n=== TEMOIN 1 bis : visée vs ennemi le plus proche angulairement (n large) ===\n")
	var enemy, mate []float64
	for _, e := range longs {
		if !e.HasAim {
			continue
		}
		shooter := idxToName[e.PlayerIdx]
		sp, ok := posOfPlayerAt(lives, nameToLives, shooter, int64(e.TS))
		if !ok {
			continue
		}
		aim := [3]float64{float64(e.Aim[0]), float64(e.Aim[1]), float64(e.Aim[2])}
		bestE, bestM := math.Inf(1), math.Inf(1)
		for name := range nameToLives {
			if name == shooter {
				continue
			}
			p, ok := posOfPlayerAt(lives, nameToLives, name, int64(e.TS))
			if !ok {
				continue
			}
			a := angleDeg(aim, [3]float64{float64(p.X - sp.X), float64(p.Y - sp.Y), float64(p.Z-sp.Z) - eyeHeight})
			if teams[name] != teams[shooter] {
				bestE = math.Min(bestE, a)
			} else {
				bestM = math.Min(bestM, a)
			}
		}
		if !math.IsInf(bestE, 1) {
			enemy = append(enemy, bestE)
		}
		if !math.IsInf(bestM, 1) {
			mate = append(mate, bestM)
		}
	}
	sort.Float64s(enemy)
	sort.Float64s(mate)
	fmt.Printf("  ennemi le plus aligné   : n=%d médiane %.1f deg, <15deg %.0f%%\n", len(enemy), median(enemy), 100*frac(enemy, 15))
	fmt.Printf("  TEMOIN coéquipier       : n=%d médiane %.1f deg, <15deg %.0f%%\n", len(mate), median(mate), 100*frac(mate, 15))
	rnd := randomAimControl(lives, nameToLives, longs, idxToName, teams)
	fmt.Printf("  TEMOIN visée ALÉATOIRE  : n=%d médiane %.1f deg, <15deg %.0f%%\n", len(rnd), median(rnd), 100*frac(rnd, 15))
	fmt.Printf("  balayage de la hauteur d'oeil (médiane ennemi) :")
	for _, h := range []float64{0, 0.3, 0.6, 0.9, 1.2} {
		fmt.Printf("  %.1f u -> %.1f deg", h, medianEnemyAt(lives, nameToLives, longs, idxToName, teams, h))
	}
	fmt.Println()
}

func report(label string, real, real2D, ctrl []float64) {
	sort.Float64s(real)
	sort.Float64s(real2D)
	sort.Float64s(ctrl)
	fmt.Printf("%s avec visée décodée ET tireur localisé : %d\n", label, len(real))
	if len(real) > 0 {
		fmt.Printf("  angle 3D : médiane %.1f deg  p25 %.1f  p75 %.1f  <15deg %.0f%%  <30deg %.0f%%\n",
			median(real), quantile(real, 0.25), quantile(real, 0.75), 100*frac(real, 15), 100*frac(real, 30))
		fmt.Printf("  angle 2D : médiane %.1f deg  <15deg %.0f%%\n", median(real2D), 100*frac(real2D, 15))
	}
	if len(ctrl) > 0 {
		fmt.Printf("  TEMOIN (autre joueur vivant au hasard) : médiane %.1f deg  <15deg %.0f%%  <30deg %.0f%%\n",
			median(ctrl), 100*frac(ctrl, 15), 100*frac(ctrl, 30))
	}
}

func f(v float32) string { return fmt.Sprintf("%.2f", v) }

func norm3(v [3]float64) float64 { return math.Sqrt(v[0]*v[0] + v[1]*v[1] + v[2]*v[2]) }

func frac(sorted []float64, thr float64) float64 {
	n := 0
	for _, v := range sorted {
		if v < thr {
			n++
		}
	}
	return float64(n) / float64(maxInt(len(sorted), 1))
}

// nearestAimEventOf : event de tir de l'index `idx`, porteur d'une visée, le plus proche de
// tUS dans la fenêtre killWindowUS.
func nearestAimEventOf(longs []FireEvent, idx int, tUS int64) (FireEvent, bool) {
	best, bestD, ok := FireEvent{}, int64(killWindowUS), false
	for _, e := range longs {
		if !e.HasAim || e.PlayerIdx != idx {
			continue
		}
		d := int64(e.TS) - tUS
		if d < 0 {
			d = -d
		}
		if d < bestD {
			bestD, best, ok = d, e, true
		}
	}
	return best, ok
}

// posOfPlayerAt renvoie la position du joueur `name` à l'instant tUS.
func posOfPlayerAt(lives []Life, nameToLives map[string][]int, name string, tUS int64) (filmdec.BipedPosition, bool) {
	for _, li := range nameToLives[name] {
		l := lives[li]
		if tUS >= l.StartUS-200_000 && tUS <= l.EndUS+200_000 {
			p, d := l.PosAt(tUS)
			if d <= 200_000 {
				return p, true
			}
		}
	}
	return filmdec.BipedPosition{}, false
}

func aliveOthers(lives []Life, nameToLives map[string][]int, shooter, victim string, tUS int64) []filmdec.BipedPosition {
	var out []filmdec.BipedPosition
	for name := range nameToLives {
		if name == shooter || name == victim {
			continue
		}
		if p, ok := posOfPlayerAt(lives, nameToLives, name, tUS); ok {
			out = append(out, p)
		}
	}
	return out
}

func dumpEvents(outDir string, longs []FireEvent, idxToName map[int]string) {
	rows := [][]string{{"ts_us", "player_idx", "name", "weapon64", "has_aim", "ax", "ay", "az"}}
	for _, e := range longs {
		rows = append(rows, []string{
			strconv.FormatUint(e.TS, 10), strconv.Itoa(e.PlayerIdx), idxToName[e.PlayerIdx],
			fmt.Sprintf("0x%016X", e.Weapon64), strconv.FormatBool(e.HasAim),
			f(e.Aim[0]), f(e.Aim[1]), f(e.Aim[2]),
		})
	}
	writeCSV(filepath.Join(outDir, "fire_events.csv"), rows)
}

func writeCSV(path string, rows [][]string) {
	f, err := os.Create(path)
	if err != nil {
		fmt.Println("csv:", err)
		return
	}
	defer f.Close()
	w := csv.NewWriter(f)
	defer w.Flush()
	for _, r := range rows {
		_ = w.Write(r)
	}
	fmt.Println("ecrit:", path)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
