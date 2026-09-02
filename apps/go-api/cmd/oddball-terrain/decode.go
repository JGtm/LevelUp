package main

// decode.go — L'ENFANT : decode UN film sous la sentinelle memoire et emet le dump tague.
//
// AUCUNE base : le pont slot->xuid vient du fil des morts du film (SlotIdentityByDeaths) et le
// gamertag vient du MEME fil (replay.Death.Gamertag, lu dans le film). Tout tient hors ligne.
//
// La sentinelle n'a droit de cite que dans un processus sans ecriture : c'est le cas, ce binaire
// n'ouvre aucune base et n'ecrit que sur stdout (le parent capte).

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"levelup/go-api/internal/analysis/objectiveevents"
	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/filmproc"
	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
)

// sweepMaxComp : dernier index de composant balaye (archetype 6 = 28 emplacements, 0..27).
const sweepMaxComp = 27

// emitCap borne le nombre d'instants imprimes pour une serie (au-dela, seul le total sort).
const emitCap = 600

// runChild decode le film -id sous la sentinelle memoire et rend le code de protocole filmproc.
func runChild(cache, id string) int {
	if id == "" {
		fmt.Fprintln(os.Stderr, "enfant : -match obligatoire")
		return filmproc.CodePreparation
	}
	g := filmproc.Arm("oddball-terrain", filmproc.MeasureLimitGiB, func(peak uint64) {
		filmproc.EmitPeak(peak)
		fmt.Fprintf(os.Stderr, "enfant : plafond memoire depasse (%d octets) — film abandonne\n", peak)
		os.Exit(filmproc.CodeMemory)
	})
	defer func() {
		g.Disarm()
		filmproc.EmitPeak(g.Peak())
	}()
	if err := decodeFilm(cache, id); err != nil {
		fmt.Fprintf(os.Stderr, "enfant %s : %v\n", id, err)
		return filmproc.CodeFailed
	}
	return filmproc.CodeOK
}

// decodeFilm decode le statborg et le fil des morts, resout l'identite, et emet le dump.
//
// UN SEUL CHARGEMENT DU FILM pour les deux lectures : jamais un `*Film` d'un cote et une
// enveloppe `dir` de l'autre, ce serait deux decompressions du meme film (item 1.6 de
// PLAN_CUISSON_PERF).
func decodeFilm(cache, id string) error {
	film, ok, err := filmcache.LoadFilm(cache, id)
	if err != nil {
		return fmt.Errorf("chargement du film : %w", err)
	}
	if !ok {
		return fmt.Errorf("film absent du cache (%s)", cache)
	}
	recs, truncated := objectiveevents.StatRecordsCtx(context.Background(), film, id)
	deaths, err := replay.ScanDeaths(film)
	if err != nil {
		return fmt.Errorf("fil des morts : %w", err)
	}
	fmt.Printf("NREC\t%s\t%d\tTRUNC=%v\n", id, len(recs), truncated)
	emitGamertags(id, deaths)
	emitDeaths(id, deaths)
	// Le pont slot -> xuid est calcule par le PARENT (identity.go), par-manche : le pont
	// monotone de production est casse par le multi-manche (compteur de morts remis a zero).
	emitRounds(id, recs)
	emitSeries(id, recs)
	return nil
}

// emitGamertags emet le gamertag lu DANS le film pour chaque xuid (deduplique).
func emitGamertags(id string, deaths []replay.Death) {
	seen := map[uint64]string{}
	for _, d := range deaths {
		if d.Gamertag != "" {
			seen[d.XUID] = d.Gamertag
		}
	}
	for _, x := range sortedU64(seen) {
		fmt.Printf("GT\t%s\t%d\t%s\n", id, x, seen[x])
	}
}

// emitDeaths emet les instants de mort par xuid (horloge du match, ms).
func emitDeaths(id string, deaths []replay.Death) {
	byX := map[uint64][]int{}
	for _, d := range deaths {
		byX[d.XUID] = append(byX[d.XUID], int(d.TimeMS))
	}
	for _, x := range sortedKeysInt(byX) {
		fmt.Printf("DEATH\t%s\t%d\t%s\n", id, x, joinInts(byX[x]))
	}
}

// emitRounds emet les manches REELLES du film.
func emitRounds(id string, recs []objectiveevents.StatRecord) {
	real := objectiveevents.RealRounds(recs)
	var rs []int
	for r, ok := range real {
		if ok {
			rs = append(rs, r)
		}
	}
	sort.Ints(rs)
	fmt.Printf("ROUNDS\t%s\t%s\n", id, joinInts(rs))
}

// emitSeries emet, par slot de joueur, par manche, par emplacement (comp,cote), la suite
// filtree des points (t:valeur). Non vide seulement — les emplacements muets ne sortent pas.
func emitSeries(id string, recs []objectiveevents.StatRecord) {
	for comp := 0; comp <= sweepMaxComp; comp++ {
		for _, sideB := range []bool{false, true} {
			c := objectiveevents.StatComponent{Comp: comp, SideB: sideB}
			byslot := objectiveevents.SeriesByRound(recs, c, false)
			for _, slot := range sortedSeriesSlots(byslot) {
				for _, round := range sortedKeysInt2(byslot[slot]) {
					pts := byslot[slot][round]
					if len(pts) == 0 {
						continue
					}
					fmt.Printf("SERIES\t%s\t%d\t%d\t%d\t%s\t%s\n",
						id, slot, round, comp, sideLabel(sideB), joinPoints(pts))
				}
			}
		}
	}
}

func sideLabel(sideB bool) string {
	if sideB {
		return "B"
	}
	return "A"
}

// joinPoints serialise une suite de points en "t:v;t:v;...", bornee par emitCap.
func joinPoints(pts []objectiveevents.ScorePoint) string {
	var b strings.Builder
	n := len(pts)
	if n > emitCap {
		n = emitCap
	}
	for i := 0; i < n; i++ {
		if i > 0 {
			b.WriteByte(';')
		}
		fmt.Fprintf(&b, "%d:%d", pts[i].TimeMS, pts[i].Value)
	}
	if len(pts) > emitCap {
		fmt.Fprintf(&b, ";+%d", len(pts)-emitCap)
	}
	return b.String()
}

func joinInts(v []int) string {
	s := make([]string, len(v))
	for i, x := range v {
		s[i] = strconv.Itoa(x)
	}
	return strings.Join(s, ",")
}

func sortedKeys(m map[int]string) []int {
	out := make([]int, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Ints(out)
	return out
}

func sortedKeysInt(m map[uint64][]int) []uint64 {
	out := make([]uint64, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func sortedKeysInt2(m map[int][]objectiveevents.ScorePoint) []int {
	out := make([]int, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Ints(out)
	return out
}

func sortedU64(m map[uint64]string) []uint64 {
	out := make([]uint64, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func sortedSeriesSlots(m map[int]map[int][]objectiveevents.ScorePoint) []int {
	out := make([]int, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Ints(out)
	return out
}
