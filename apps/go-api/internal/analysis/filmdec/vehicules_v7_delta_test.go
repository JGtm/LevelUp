package filmdec

// vehicules_v7_delta_test.go — INSTRUMENT (lot V7) : L'ECART ENTRE UN EVENEMENT QUI VISE UN
// VEHICULE ET LA FIN SERREE DE CE VEHICULE. C'est la mesure la plus fine du lot, et celle qui
// tranche.
//
// ELLE COMBINE LES DEUX ACQUIS. `TestV7Dom1` a etabli que la reference 0 d'un evenement de
// domaine 1 designe une UNITE, bipede OU vehicule (partition 100,0 % pour les types 1 et 7,
// 99,96 % pour le type 0, 99,8 % pour le type 36, sur 12 films) : on sait donc QUEL vehicule un
// evenement vise. `TestV7Temps` a etabli la FIN SERREE d'une vie de vehicule : son dernier
// echantillon de position, connu a ~0,5 s — cent fois mieux que la fenetre de recensement.
//
// L'ORACLE, ECRIT AVANT LA MESURE. Si un type d'evenement DATE la destruction, alors les
// instances qui visent un vehicule tombent SUR sa fin serree : l'ecart median est ~0 et la part
// a moins d'une seconde approche 100 %. Un type de DEGAT, lui, se repartit sur toute la vie —
// c'est deja ce que la position relative disait du type 0 (mediane 0,73, quartiles 0,46-0,87),
// mais avec une resolution de 20 s ; ici la resolution est la demi-seconde.
//
// LE TEMOIN : le meme ecart calcule avec l'instant DECALE de +60 s. Il conserve l'evenement, la
// cible et la vie, et ne change que l'instant.
//
// Garde d'environnement V7_ROOT / V7_FILMS (+ V7_TYPES) : sans elle, tout SKIP.

import (
	"os"
	"sort"
	"strconv"
	"testing"
)

// v7DeltaAcc accumule les ecarts d'un type.
type v7DeltaAcc struct {
	deltas       []float64 // t_evenement - fin serree, en secondes (signe)
	near1, near3 int
	shift1       int
	cibles       map[uint32]bool
}

// v7DeltaScan balaie un film.
func v7DeltaScan(t *testing.T, dir string, want map[int]bool, acc map[int]*v7DeltaAcc) (int, int) {
	k := ScanFilmWorldObjectKeyframes(dir, v0VehiculeTI)
	if len(k.Band) == 0 {
		return 0, 0
	}
	opt := DefaultScanFilmOptions()
	opt.RequireTag1, opt.MaxSpeedMPS, opt.IsolationGapMS, opt.QuantaOnly = false, 0, 0, true
	pos, err := ScanFilmBipedPositionsForBand(dir, NewSlotBand(k.Band), opt)
	if err != nil {
		t.Logf("positions illisibles dans %s : %v", dir, err)
		return 0, 0
	}
	ends := v7TightEnds(dir, k, pos)
	bySlot := map[uint32][]v7EndLife{}
	for _, e := range ends {
		bySlot[e.slot] = append(bySlot[e.slot], e)
	}
	n := CountFilmChunks(dir)
	chunks := make([]int, 0, n)
	for i := 1; i <= n; i++ {
		chunks = append(chunks, i)
	}
	bip := bipedSlotBandMapDir(dir, chunks)
	base := ^uint32(0)
	for s := range bip {
		if s < base {
			base = s
		}
	}
	if len(bip) == 0 {
		base = 0
	}
	hits := 0
	for _, c := range chunks {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, p := range WalkPackets(data) {
			if p.Type != PacketTypeDelta {
				continue
			}
			pay := p.Payload(data)
			ty, present := PacketHeadEventType(pay)
			if !present || !want[ty] {
				continue
			}
			r := readDom1Ref(pay, eventPayloadStartBit)
			if !r.Present || r.Sonde != 1 {
				continue
			}
			slot := base + r.Index
			if !k.Band[slot] {
				continue
			}
			if v7DeltaOne(acc, ty, slot, p.TimestampUS, bySlot[slot]) {
				hits++
			}
		}
	}
	return len(ends), hits
}

// v7DeltaOne range UNE instance : elle cherche la vie du slot qui couvre l'instant, et mesure
// l'ecart a sa fin serree.
func v7DeltaOne(acc map[int]*v7DeltaAcc, ty int, slot uint32, at uint64, lives []v7EndLife) bool {
	for _, l := range lives {
		if at < l.lo || at > l.hi || l.endUS == 0 {
			continue
		}
		a := acc[ty]
		if a == nil {
			a = &v7DeltaAcc{cibles: map[uint32]bool{}}
			acc[ty] = a
		}
		d := (float64(at) - float64(l.endUS)) / 1e6
		a.deltas = append(a.deltas, d)
		a.cibles[slot] = true
		if d >= -1 && d <= 1 {
			a.near1++
		}
		if d >= -3 && d <= 3 {
			a.near3++
		}
		if s := (float64(at+v7ShiftUS) - float64(l.endUS)) / 1e6; s >= -1 && s <= 1 {
			a.shift1++
		}
		return true
	}
	return false
}

// TestV7Delta — LA TABLE DES ECARTS.
func TestV7Delta(t *testing.T) {
	dirs := v7FilmDirs(t)
	release := LockProcessDecode()
	defer release()
	list := v7LetalTypes
	if s := os.Getenv("V7_TYPES"); s != "" {
		list = nil
		for _, x := range splitComma(s) {
			v, err := strconv.Atoi(x)
			if err != nil {
				t.Fatalf("V7_TYPES : %q n'est pas un entier", x)
			}
			list = append(list, v)
		}
	}
	want := map[int]bool{}
	for _, v := range list {
		want[v] = true
	}
	acc := map[int]*v7DeltaAcc{}
	totalEnds := 0
	for _, d := range dirs {
		ends, hits := v7DeltaScan(t, d, want, acc)
		totalEnds += ends
		t.Logf("film %-40s %3d vies a fin serree · %5d instances appariees", d, ends, hits)
	}
	var tys []int
	for ty := range acc {
		tys = append(tys, ty)
	}
	sort.Ints(tys)
	t.Logf("== V7 DELTA — ecart evenement -> FIN SERREE du vehicule vise · %d vies ==", totalEnds)
	t.Logf("%-5s %-7s %-7s %-9s %-9s %-10s | %-8s %-8s %-8s %-8s %-8s",
		"type", "n", "cibles", "<=1 s", "<=3 s", "tem+60 1s", "min", "q1", "MEDIANE", "q3", "max")
	for _, ty := range tys {
		a := acc[ty]
		if len(a.deltas) == 0 {
			continue
		}
		lo, q1, med, q3, hi := v7Quartiles(a.deltas)
		p := func(x int) float64 { return 100 * float64(x) / float64(len(a.deltas)) }
		t.Logf("%-5d %-7d %-7d %8.1f%% %8.1f%% %9.1f%% | %8.1f %8.1f %8.1f %8.1f %8.1f",
			ty, len(a.deltas), len(a.cibles), p(a.near1), p(a.near3), p(a.shift1),
			lo, q1, med, q3, hi)
	}
}
