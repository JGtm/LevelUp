package filmdec

// r8_i54_oracle_research_test.go — LE JUGE de la piste B, canal i54, EN TEMPS FILM.
//
// TOUT RESTE DANS L'HORLOGE DU FILM : positions (`ScanFilmBipedPositions`), grappin
// (`ScanFilmGrappleReads`), rang de capacite (`ScanFilmAbilityRanks`) et i54 partagent le
// meme horodatage de paquet. Aucun recalage sur l'axe du document n'est necessaire, donc
// aucune hypothese d'origine d'horloge n'entre dans la mesure.
//
// LES TROIS QUESTIONS, ET LEURS SEUILS ECRITS AVANT LA MESURE :
//
//  1. i54 EST-IL LE CANAL DE MOBILITE ? Les episodes `flag1==1` doivent montrer une bouffee
//     de vitesse. Reference POSITIVE : les lectures de grappin, dont l'instant est certain.
//     Reference NEGATIVE : des instants tires au hasard dans les memes vies.
//  2. LE CORPS DISCRIMINE-T-IL L'ACTION ? Le champ terminal `B7` (R(7)) est le candidat.
//     Une valeur qui date le propulseur doit (a) porter une bouffee de vitesse de l'ordre
//     de celle du grappin et (b) se concentrer sur les porteurs du rang « propulseur ».
//  3. LE GRAPPIN PASSE-T-IL PAR i54 ? Coincidence des episodes avec les lectures de
//     grappin a +/- 500 ms. S'il y passe, une valeur de `B7` doit lui correspondre — et
//     c'est la CLE DE LECTURE de la table, obtenue sans aucune supposition.
//
// GARDES : `R8_FILMS`, `R8_BOUNDS`, `R8_IDS`.
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 R8_FILMS=<repo>/data/cache/film_chunks \
//	  R8_BOUNDS=<worktree>/data/titles/halo_infinite/reference/map_quant_bounds.json \
//	  R8_IDS=00ba2e1c go test ./internal/analysis/filmdec/ -run '^TestR8I54Oracle$' \
//	  -timeout 120m -v

import (
	"math/rand"
	"path/filepath"
	"sort"
	"strconv"
	"testing"
)

const (
	// r8MobEpisodeGapUS : deux emissions du meme slot a moins de 1 s portent LA MEME action
	// (l'etat est repete tant qu'il dure), pas deux usages. Meme seuil que l'instrument i54
	// de 2026-08, repris tel quel pour rester comparable.
	r8MobEpisodeGapUS = 1_000_000
	// r8PeakWindowUS : demi-fenetre du pic de vitesse autour de l'instant juge (500 ms).
	r8PeakWindowUS = 500_000
	// r8SpeedMaxDtUS : ecart maximal entre deux echantillons pour que leur rapport soit une
	// vitesse. 250 ms — au-dela, le trou de replication domine le deplacement.
	r8SpeedMaxDtUS = 250_000
	// r8CoincUS : fenetre de coincidence entre deux canaux (500 ms).
	r8CoincUS = 500_000
)

// r8Speed est un segment de vitesse d'un slot : [t0, t1] et la vitesse horizontale.
type r8Speed struct {
	t0, t1 uint64
	v      float64
}

// r8SpeedIndex porte les segments de vitesse par slot, tries par instant.
type r8SpeedIndex map[uint32][]r8Speed

// r8BuildSpeeds construit l'index depuis les positions decodees du film.
func r8BuildSpeeds(pos []BipedPosition) r8SpeedIndex {
	bySlot := map[uint32][]BipedPosition{}
	for _, p := range pos {
		if p.HasWorld {
			bySlot[p.Slot] = append(bySlot[p.Slot], p)
		}
	}
	out := r8SpeedIndex{}
	for slot, list := range bySlot {
		sort.Slice(list, func(i, j int) bool { return list[i].TimestampUS < list[j].TimestampUS })
		for i := 1; i < len(list); i++ {
			a, b := list[i-1], list[i]
			dt := b.TimestampUS - a.TimestampUS
			if dt == 0 || dt > r8SpeedMaxDtUS {
				continue
			}
			d := r8Dist2(float64(a.X), float64(a.Y), float64(b.X), float64(b.Y))
			out[slot] = append(out[slot], r8Speed{t0: a.TimestampUS, t1: b.TimestampUS,
				v: d / (float64(dt) / 1e6)})
		}
	}
	return out
}

// peak rend la vitesse maximale du slot dans [at-w, at+w] et le nombre de segments lus.
func (ix r8SpeedIndex) peak(slot uint32, at uint64, w uint64) (float64, int) {
	list := ix[slot]
	lo, hi := uint64(0), at+w
	if at > w {
		lo = at - w
	}
	best, n := 0.0, 0
	for _, s := range list {
		if s.t1 < lo {
			continue
		}
		if s.t0 > hi {
			break
		}
		n++
		if s.v > best {
			best = s.v
		}
	}
	return best, n
}

// r8Episodes replie les emissions `flag1==1` en episodes : le PREMIER evenement de chaque
// rafale du meme slot.
func r8Episodes(evs []r8MobEvent) []r8MobEvent {
	last := map[uint32]uint64{}
	var out []r8MobEvent
	for _, e := range evs {
		if !e.Flag1 {
			continue
		}
		if t, ok := last[e.Slot]; !ok || e.TSUS-t > r8MobEpisodeGapUS {
			out = append(out, e)
		}
		last[e.Slot] = e.TSUS
	}
	return out
}

// r8RankAt rend le dernier rang de capacite lu pour ce slot a `at` ou avant, ou -1.
func r8RankAt(ranks []AbilityRank, slot uint32, at uint64) int {
	best, bestT := -1, uint64(0)
	for _, r := range ranks {
		if r.Slot != slot || r.TimestampUS > at {
			continue
		}
		if best < 0 || r.TimestampUS >= bestT {
			best, bestT = int(r.Rank), r.TimestampUS
		}
	}
	return best
}

// r8Bucket agrege les pics d'une population.
type r8Bucket struct {
	peaks []float64
	ranks map[int]int
}

func (b *r8Bucket) add(p float64, rank int) {
	if b.ranks == nil {
		b.ranks = map[int]int{}
	}
	b.peaks = append(b.peaks, p)
	b.ranks[rank]++
}

func TestR8I54Oracle(t *testing.T) {
	for _, dir := range r8FilmDirs(t) {
		r8I54OneFilm(t, dir)
	}
}

func r8I54OneFilm(t *testing.T, dir string) {
	t.Helper()
	entry := r8MapEntry(t, dir)
	wr := entry.Range()
	release := LockProcessDecode()
	defer release()
	saved := WorldObjectPrecision
	SetWorldObjectPrecisionFromLayout(entry.Layout())
	defer func() { WorldObjectPrecision = saved }()

	s := r8MobResolve(t, dir)
	opt := DefaultScanFilmOptions()
	opt.WorldRange = &wr
	pos, err := ScanFilmBipedPositions(dir, opt)
	if err != nil {
		t.Fatalf("positions illisibles dans %s : %v", dir, err)
	}
	speeds := r8BuildSpeeds(pos)
	grapples, _, err := ScanFilmGrappleReads(dir)
	if err != nil {
		t.Logf("lectures de grappin illisibles dans %s : %v", dir, err)
	}
	ranks, _, err := ScanFilmAbilityRanks(dir)
	if err != nil {
		t.Logf("rangs de capacite illisibles dans %s : %v", dir, err)
	}
	evs, records, with54 := r8ScanMobility(t, s)
	eps := r8Episodes(evs)
	t.Logf("%s : records biped=%d masque∋i54=%d emissions=%d episodes flag1=%d"+
		" | positions=%d slots=%d grappin=%d rangs=%d",
		filepath.Base(dir), records, with54, len(evs), len(eps), len(pos), len(speeds),
		len(grapples), len(ranks))
	r8LogBodyCensus(t, evs)
	r8LogOracle(t, eps, grapples, ranks, speeds)
}

// r8LogBodyCensus recense les champs terminaux du corps — sans quoi une table de pics par
// valeur ne se juge pas (une valeur vue 3 fois ne dit rien).
func r8LogBodyCensus(t *testing.T, evs []r8MobEvent) {
	t.Helper()
	var body, bits int
	b7, b2, blast := map[uint32]int{}, map[uint32]int{}, map[uint32]int{}
	for _, e := range evs {
		if !e.Body {
			continue
		}
		body++
		bits = e.Bits
		b7[e.B7]++
		b2[e.B2]++
		blast[e.BLast]++
	}
	t.Logf("  corps lus=%d (derniere largeur=%d bits) | B7 %v | B2 %v | Blast %v",
		body, bits, r8TopValues(b7, 12), r8TopValues(b2, 4), r8TopValues(blast, 2))
}

// r8TopValues rend les `n` valeurs les plus frequentes, triees par frequence decroissante.
func r8TopValues(m map[uint32]int, n int) []string {
	type kv struct {
		k uint32
		v int
	}
	all := make([]kv, 0, len(m))
	for k, v := range m {
		all = append(all, kv{k, v})
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].v != all[j].v {
			return all[i].v > all[j].v
		}
		return all[i].k < all[j].k
	})
	out := make([]string, 0, n)
	for i := 0; i < len(all) && i < n; i++ {
		out = append(out, r8Pair(all[i].k, all[i].v))
	}
	return out
}

func r8Pair(k uint32, v int) string {
	return strconv.Itoa(int(k)) + ":" + strconv.Itoa(v)
}

// r8LogOracle confronte les episodes i54 a l'oracle de vitesse, avec le grappin en positif
// et des instants tires au hasard en negatif.
func r8LogOracle(
	t *testing.T, eps []r8MobEvent, grapples []GrappleRead, ranks []AbilityRank,
	speeds r8SpeedIndex,
) {
	t.Helper()
	pops := map[string]*r8Bucket{}
	get := func(k string) *r8Bucket {
		if pops[k] == nil {
			pops[k] = &r8Bucket{}
		}
		return pops[k]
	}
	for _, e := range eps {
		p, n := speeds.peak(e.Slot, e.TSUS, r8PeakWindowUS)
		if n == 0 {
			continue
		}
		rank := r8RankAt(ranks, e.Slot, e.TSUS)
		get("i54 episodes (tous)").add(p, rank)
		if e.Body {
			get("i54 B7="+strconv.Itoa(int(e.B7))).add(p, rank)
		}
	}
	for _, g := range grapples {
		if p, n := speeds.peak(g.Slot, g.TimestampUS, r8PeakWindowUS); n > 0 {
			get("POSITIF grappin").add(p, r8RankAt(ranks, g.Slot, g.TimestampUS))
		}
	}
	r8RandomFilmWitness(speeds, get("TEMOIN aleatoire"))
	r8LogBuckets(t, pops)
}

// r8RandomFilmWitness tire 40 instants par slot dans ses propres segments de vitesse.
func r8RandomFilmWitness(speeds r8SpeedIndex, b *r8Bucket) {
	rng := rand.New(rand.NewSource(r8Seed)) //nolint:gosec // temoin reproductible
	slots := make([]uint32, 0, len(speeds))
	for s := range speeds {
		slots = append(slots, s)
	}
	sort.Slice(slots, func(i, j int) bool { return slots[i] < slots[j] })
	for _, s := range slots {
		list := speeds[s]
		if len(list) < 20 {
			continue
		}
		for n := 0; n < 40; n++ {
			at := list[rng.Intn(len(list))].t0
			if p, k := speeds.peak(s, at, r8PeakWindowUS); k > 0 {
				b.add(p, -1)
			}
		}
	}
}

func r8LogBuckets(t *testing.T, pops map[string]*r8Bucket) {
	t.Helper()
	keys := make([]string, 0, len(pops))
	for k := range pops {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	t.Logf("  %-26s %6s %8s %8s %8s   %s", "population", "n", "medPic", "p90Pic", "maxPic", "rangs i48")
	for _, k := range keys {
		b := pops[k]
		if len(b.peaks) < 3 {
			continue
		}
		t.Logf("  %-26s %6d %8.2f %8.2f %8.2f   %v", k, len(b.peaks),
			r8Quantile(b.peaks, 0.5), r8Quantile(b.peaks, 0.9), r8Quantile(b.peaks, 1.0),
			r8RankSummary(b.ranks))
	}
}

func r8RankSummary(m map[int]int) []string {
	keys := make([]int, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return m[keys[i]] > m[keys[j]] })
	out := make([]string, 0, 6)
	for i := 0; i < len(keys) && i < 6; i++ {
		out = append(out, strconv.Itoa(keys[i])+"x"+strconv.Itoa(m[keys[i]]))
	}
	return out
}
