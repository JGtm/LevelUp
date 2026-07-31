// cmd/tmp_viesorphelines — POURQUOI 15 VIES SUR 105 NE SONT PAS NOMMEES.
//
// L'HYPOTHESE DE L'UTILISATEUR, a tester : « 15 vies ca me parait beaucoup, y a sans doute des
// suicides dans le tas ou une autre raison. La fin d'une partie devrait etre forcee comme fin
// de vie non ? »
//
// CE QUE CE PROGRAMME MESURE, sans rien corriger :
//  1. QUAND se terminent les vies non nommees. Si elles finissent toutes a la meme seconde,
//     c'est la fin du match, et la reponse est structurelle.
//  2. Le fil des morts contient-il des morts NON APPARIEES ? Une mort sans vie en face est le
//     symptome inverse — un suicide ou une mort dont la vie n'a pas ete decoupee.
//  3. Le film porte-t-il un horodatage de FIN DE MATCH exploitable ? L'utilisateur signale
//     qu'il existe ; s'il existe, il nomme les vies qui s'y arretent sans mort.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/domain/title"
)

func main() {
	repo := flag.String("repo", `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration`, "racine des films")
	bounds := flag.String("bounds", `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/.claude/worktrees/filmdec-continuation`, "racine du catalogue de bornes")
	match := flag.String("match", "000d5950", "match")
	mapName := flag.String("map", "Cliffhanger", "carte")
	flag.Parse()
	filmDir := filepath.Join(*repo, "data", "cache", "film_chunks", *match)

	cat, err := filmdec.LoadMapQuantCatalog(title.NewPathResolver(*bounds).MapQuantBoundsPath(title.DefaultSlug))
	if err != nil {
		fmt.Fprintln(os.Stderr, "catalogue:", err)
		os.Exit(1)
	}
	e, err := cat.Lookup(*mapName)
	if err != nil {
		fmt.Fprintln(os.Stderr, "carte:", err)
		os.Exit(1)
	}
	rng := e.Range()
	opt := filmdec.DefaultScanFilmOptions()
	opt.WorldRange = &rng
	opt.CaptureDirs = true
	pos, err := filmdec.ScanFilmBipedPositions(filmDir, opt)
	if err != nil {
		fmt.Fprintln(os.Stderr, "positions:", err)
		os.Exit(1)
	}
	deaths, err := replay.ScanFilmDeaths(filmDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "morts:", err)
		os.Exit(1)
	}
	report(pos, deaths, filmDir)
}

type life struct {
	slot     uint32
	from, to int64
	named    bool
	xuid     uint64
}

func report(pos []filmdec.BipedPosition, deaths []replay.Death, filmDir string) {
	lives := cut(pos)
	off, named := match(lives, deaths)
	fmt.Printf("=== VIES ET MORTS — %d vies, %d morts au fil\n", len(lives), len(deaths))
	fmt.Printf("vies nommees : %d ; NON nommees : %d\n\n", named, len(lives)-named)

	// 1. QUAND finissent les vies non nommees ?
	var orphanEnds []int64
	var namedEnds []int64
	for _, l := range lives {
		if l.named {
			namedEnds = append(namedEnds, l.to/1000)
		} else {
			orphanEnds = append(orphanEnds, l.to/1000)
		}
	}
	sort.Slice(orphanEnds, func(i, j int) bool { return orphanEnds[i] < orphanEnds[j] })
	sort.Slice(namedEnds, func(i, j int) bool { return namedEnds[i] < namedEnds[j] })
	last := namedEnds[len(namedEnds)-1]
	fmt.Printf("--- 1. QUAND se terminent les vies NON NOMMEES (ms, horloge film) ---\n")
	fmt.Printf("derniere fin de vie NOMMEE : %d\n", last)
	atEnd := 0
	for _, t := range orphanEnds {
		mark := ""
		if t >= last-2000 {
			mark = "   <-- a la fin du match"
			atEnd++
		}
		fmt.Printf("  %d%s\n", t, mark)
	}
	fmt.Printf("\n%d des %d vies non nommees finissent DANS LES 2 DERNIERES SECONDES du match.\n",
		atEnd, len(orphanEnds))

	// 2. Des morts non appariees ?
	fmt.Printf("\n--- 2. MORTS DU FIL NON APPARIEES A UNE VIE ---\n")
	unmatched := unpaired(lives, deaths, off)
	fmt.Printf("%d morts sur %d ne trouvent aucune fin de vie dans la fenetre.\n", len(unmatched), len(deaths))
	for _, d := range unmatched {
		fmt.Printf("  xuid %d a %d ms\n", d.XUID, d.TimeMS)
	}

	// 3. Le film porte-t-il une fin de match ?
	fmt.Printf("\n--- 3. FIN DE MATCH DANS LE FIL DES EVENEMENTS ---\n")
	scanEndOfMatch(filmDir, off)
	widenWindow(lives, deaths, off)
	chainRespawns(lives)
}

func cut(pos []filmdec.BipedPosition) []life {
	by := map[uint32][]filmdec.BipedPosition{}
	for _, p := range pos {
		by[p.Slot] = append(by[p.Slot], p)
	}
	var slots []uint32
	for s := range by {
		slots = append(slots, s)
	}
	sort.Slice(slots, func(i, j int) bool { return slots[i] < slots[j] })
	var out []life
	for _, s := range slots {
		ps := by[s]
		sort.Slice(ps, func(i, j int) bool { return ps[i].TimestampUS < ps[j].TimestampUS })
		start, lastT := int64(ps[0].TimestampUS), int64(ps[0].TimestampUS)
		for _, p := range ps[1:] {
			t := int64(p.TimestampUS)
			if t-lastT > 5_000_000 {
				out = append(out, life{slot: s, from: start, to: lastT})
				start = t
			}
			lastT = t
		}
		out = append(out, life{slot: s, from: start, to: lastT})
	}
	return out
}

// match reproduit l'appariement du rejeu et marque les vies nommees.
func match(lives []life, deaths []replay.Death) (int64, int) {
	ends := make([]int64, len(lives))
	for i, l := range lives {
		ends[i] = l.to / 1000
	}
	lo, hi := ends[0], ends[0]
	for _, e := range ends {
		if e < lo {
			lo = e
		}
		if e > hi {
			hi = e
		}
	}
	bestOff, bestN := int64(0), -1
	var plateau []int64
	for off := lo - 60_000; off <= hi; off += 10 {
		n := count(ends, deaths, off)
		if n > bestN {
			bestN, plateau = n, []int64{off}
		} else if n == bestN {
			plateau = append(plateau, off)
		}
	}
	bestOff = plateau[len(plateau)/2]
	// appariement glouton, comme le rejeu
	type pair struct {
		di, li int
		d      int64
	}
	var ps []pair
	for di, d := range deaths {
		for li, e := range ends {
			if x := abs(e - (d.TimeMS + bestOff)); x <= 150 {
				ps = append(ps, pair{di, li, x})
			}
		}
	}
	sort.Slice(ps, func(i, j int) bool {
		if ps[i].d != ps[j].d {
			return ps[i].d < ps[j].d
		}
		return ps[i].li < ps[j].li
	})
	ud, ul := make([]bool, len(deaths)), make([]bool, len(lives))
	n := 0
	for _, p := range ps {
		if ud[p.di] || ul[p.li] {
			continue
		}
		ud[p.di], ul[p.li] = true, true
		lives[p.li].named, lives[p.li].xuid = true, deaths[p.di].XUID
		n++
	}
	return bestOff, n
}

func count(ends []int64, deaths []replay.Death, off int64) int {
	used := make([]bool, len(ends))
	n := 0
	for _, d := range deaths {
		bi, bd := -1, int64(151)
		for i, e := range ends {
			if used[i] {
				continue
			}
			if x := abs(e - (d.TimeMS + off)); x < bd {
				bd, bi = x, i
			}
		}
		if bi >= 0 {
			used[bi] = true
			n++
		}
	}
	return n
}

func unpaired(lives []life, deaths []replay.Death, off int64) []replay.Death {
	ends := make([]int64, len(lives))
	for i, l := range lives {
		ends[i] = l.to / 1000
	}
	var out []replay.Death
	for _, d := range deaths {
		hit := false
		for _, e := range ends {
			if abs(e-(d.TimeMS+off)) <= 150 {
				hit = true
				break
			}
		}
		if !hit {
			out = append(out, d)
		}
	}
	return out
}

// scanEndOfMatch cherche, dans le chunk des highlights, les evenements de MODE — le film y
// marque le deroulement de la partie, donc potentiellement sa fin.
func scanEndOfMatch(filmDir string, off int64) {
	n := filmdec.CountFilmChunks(filmDir)
	raw, err := os.ReadFile(filepath.Join(filmDir, fmt.Sprintf("chunk_%02d.bin", n)))
	if err != nil {
		fmt.Println("  chunk highlight illisible:", err)
		return
	}
	evs, err := analysis.ParseHighlightEvents(raw, 0)
	if err != nil {
		fmt.Println("  parse:", err)
		return
	}
	byType := map[string]int{}
	var modes []analysis.HighlightEvent
	maxT := 0
	for _, e := range evs {
		byType[e.EventType]++
		if e.TimeMS > maxT {
			maxT = e.TimeMS
		}
		if e.EventType == analysis.EventTypeMode {
			modes = append(modes, e)
		}
	}
	fmt.Printf("  evenements du fil par type : %v\n", byType)
	fmt.Printf("  dernier horodatage du fil : %d ms (soit %d ms en horloge film)\n", maxT, int64(maxT)+off)
	sort.Slice(modes, func(i, j int) bool { return modes[i].TimeMS < modes[j].TimeMS })
	fmt.Printf("  evenements de MODE : %d\n", len(modes))
	for i, m := range modes {
		if i >= 12 {
			fmt.Printf("  ... (%d autres)\n", len(modes)-i)
			break
		}
		fmt.Printf("    t=%-8d typeHint=%-4d xuid=%d\n", m.TimeMS, m.TypeHint, m.XUID)
	}
}

func abs(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

// ---------------------------------------------------------------- pistes de recuperation

// widenWindow mesure ce qu'une fenetre d'appariement plus large recupere, ET ce qu'elle
// recupere PAR HASARD. Un elargissement qui gagne autant sur le temoin ne gagne rien.
func widenWindow(lives []life, deaths []replay.Death, off int64) {
	fmt.Printf("\n--- 4. ELARGIR LA FENETRE D'APPARIEMENT ---\n")
	fmt.Printf("%-10s %-12s %-12s %s\n", "fenetre", "apparies", "temoin", "gain net")
	ends := make([]int64, len(lives))
	for i, l := range lives {
		ends[i] = l.to / 1000
	}
	// TEMOIN : les memes morts, replacees au hasard sur la plage. Si la fenetre large gagne
	// autant sur le temoin, elle n'apparie plus des faits mais du bruit.
	lo, hi := ends[0], ends[0]
	for _, e := range ends {
		if e < lo {
			lo = e
		}
		if e > hi {
			hi = e
		}
	}
	fake := make([]replay.Death, len(deaths))
	seed := int64(12345)
	for i := range fake {
		seed = (seed*1103515245 + 12345) & 0x7fffffff
		fake[i].TimeMS = lo + seed%(hi-lo+1) - off
	}
	for _, w := range []int64{150, 500, 1000, 2000, 4000, 7000} {
		real := countW(ends, deaths, off, w)
		ctrl := countW(ends, fake, off, w)
		fmt.Printf("%-10d %-12d %-12d %+d\n", w, real, ctrl, real-ctrl)
	}
}

func countW(ends []int64, deaths []replay.Death, off, w int64) int {
	used := make([]bool, len(ends))
	n := 0
	for _, d := range deaths {
		bi, bd := -1, w+1
		for i, e := range ends {
			if used[i] {
				continue
			}
			if x := abs(e - (d.TimeMS + off)); x < bd {
				bd, bi = x, i
			}
		}
		if bi >= 0 {
			used[bi] = true
			n++
		}
	}
	return n
}

// chainRespawns teste la piste du CHAINAGE : apres la mort d'un joueur, sa vie suivante est
// la premiere vie NON NOMMEE qui commence apres. Si l'appariement est univoque, c'est
// deterministe ; s'il ne l'est pas, il ne faut pas s'en servir.
func chainRespawns(lives []life) {
	fmt.Printf("\n--- 5. CHAINAGE DES REAPPARITIONS (piste pour les survivants) ---\n")
	var orphans []int
	for i, l := range lives {
		if !l.named {
			orphans = append(orphans, i)
		}
	}
	sort.Slice(orphans, func(a, b int) bool { return lives[orphans[a]].from < lives[orphans[b]].from })
	amb, uniq := 0, 0
	for _, oi := range orphans {
		o := lives[oi]
		// candidats : les vies NOMMEES qui se terminent AVANT le debut de l'orpheline, et dont
		// aucune autre vie du meme joueur ne couvre l'intervalle.
		var cands []uint64
		for _, l := range lives {
			if !l.named || l.to >= o.from {
				continue
			}
			if o.from-l.to > 15_000_000 { // au-dela de 15 s, ce n'est plus une reapparition
				continue
			}
			dup := false
			for _, c := range cands {
				if c == l.xuid {
					dup = true
				}
			}
			if !dup {
				cands = append(cands, l.xuid)
			}
		}
		if len(cands) == 1 {
			uniq++
		} else {
			amb++
		}
		fmt.Printf("  vie orpheline slot=%-5d [%d..%d] : %d joueur(s) candidat(s)\n",
			o.slot, o.from/1000, o.to/1000, len(cands))
	}
	fmt.Printf("\n%d orphelines ont UN SEUL candidat, %d en ont plusieurs (ou aucun).\n", uniq, amb)
	fmt.Printf("Le chainage n'est utilisable que sur les premieres ; ailleurs il faudrait choisir.\n")
}
