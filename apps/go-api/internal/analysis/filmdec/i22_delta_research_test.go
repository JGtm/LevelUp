package filmdec

// i22_delta_research_test.go — SONDE JETABLE (etude de faisabilite du 2026-08-24).
//
// QUESTION POSEE : le curseur moteur est-il reconstructible HORS capture Cheat Engine au
// point d'atteindre i22 (comptes de grenades) dans les paquets delta ? Le precedent qui
// marche est ScanFilmAbilityRanks (i48) : ancre matchBipedHeader + walkRecordTo avec les
// desers de PRODUCTION. Cette sonde applique exactement le meme chemin, cible 22, et publie
// le taux de lectures PLAUSIBLES (compteur == 4 et valeurs bornees a 2), qui est le test
// refutable etabli par unit_weaponstate.go (« count != 4 = signature d'un curseur mal place »).
//
// LECTURE SEULE, gate par I22_FILM, sautee partout ailleurs (CI comprise). Un seul decodage
// filmdec par process : le verrou est pris.
//
// USAGE (depuis apps/go-api) :
//
//	CGO_ENABLED=0 I22_FILM=<repo>/data/cache/film_chunks/000d5950 \
//	  go test ./internal/analysis/filmdec/ -run '^TestI22DeltaResearch$' -timeout 30m -v

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"testing"
)

const i22FilmEnv = "I22_FILM"

// i22Index est l'index d'iterateur d'`unit-grenade-counts` dans l'archetype biped.
const i22Index = 22

// i22Read est une lecture d'i22 localisee.
type i22Read struct {
	Slot        uint32   `json:"slot"`
	Chunk       int      `json:"chunk"`
	TimestampUS uint64   `json:"ts_us"`
	Count       uint64   `json:"count"`
	Values      []uint64 `json:"values"`
}

func TestI22DeltaResearch(t *testing.T) {
	dir := os.Getenv(i22FilmEnv)
	if dir == "" {
		t.Skipf("%s non defini : sonde de recherche sautee", i22FilmEnv)
	}
	release := LockProcessDecode()
	defer release()

	n := CountFilmChunks(dir)
	if n == 0 {
		t.Fatalf("aucun chunk film dans %s", dir)
	}
	chunks := make([]int, 0, n)
	for i := 1; i <= n; i++ {
		chunks = append(chunks, i)
	}
	slots := bipedSlotBandDir(dir, chunks)
	if slots.Count() == 0 {
		t.Fatalf("aucun slot biped dans les keyframes de %s", dir)
	}
	lay, _, err := DetectI0Layout(dir)
	if err != nil {
		t.Fatalf("decoupage i0 illisible : %v", err)
	}
	arch, err := bipedArchetypeDir(dir)
	if err != nil {
		t.Fatalf("archetype biped : %v", err)
	}

	var last struct {
		count uint64
		vals  []uint64
		got   bool
	}
	prev := grenadeCountsHook
	SetGrenadeCountsHook(func(c uint64, v []uint64) {
		last.count, last.vals, last.got = c, append([]uint64(nil), v...), true
	})
	defer SetGrenadeCountsHook(prev)

	var records, withI22, read, unread, plausible int
	countHist := map[uint64]int{}
	valHist := map[uint64]int{}
	var reads []i22Read

	minRecord := bipedHeaderBits + bipedIndexBits*bipedMinMaskCnt + lay.TotalBits()
	for _, c := range chunks {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeDelta {
				continue
			}
			pay := pk.Payload(data)
			total := len(pay) * 8
			for p := 0; p+minRecord <= total; {
				i0, slot, idx, ok := matchBipedHeader(pay, p, total, slots, true, lay)
				if !ok {
					p++
					continue
				}
				records++
				if maskHas(idx, i22Index) {
					withI22++
					last.got = false
					if walkRecordTo(pay, i0, total, idx, lay, arch, i22Index) && last.got {
						read++
						countHist[last.count]++
						ok := last.count == 4
						for _, v := range last.vals {
							valHist[v]++
							if v > 2 {
								ok = false
							}
						}
						if ok {
							plausible++
							reads = append(reads, i22Read{
								Slot: slot, Chunk: c, TimestampUS: pk.TimestampUS,
								Count: last.count, Values: last.vals,
							})
						}
					} else {
						unread++
					}
				}
				p = i0 + lay.TotalBits()
			}
		}
	}

	t.Logf("records biped delta = %d | masque annonce i22 = %d | lus = %d | non lus = %d",
		records, withI22, read, unread)
	if read > 0 {
		t.Logf("PLAUSIBLES (count==4 et valeurs<=2) = %d / %d = %.2f %%",
			plausible, read, 100*float64(plausible)/float64(read))
	}
	t.Logf("histogramme du compteur R(3) : %s", i22SortMap(countHist))
	t.Logf("histogramme des valeurs R(8) (20 premieres) : %s", i22TopMap(valHist, 20))

	// Series temporelles par slot, pour la confrontation aux images-cles.
	if out := os.Getenv("I22_OUT"); out != "" {
		sort.Slice(reads, func(i, j int) bool {
			if reads[i].Slot != reads[j].Slot {
				return reads[i].Slot < reads[j].Slot
			}
			return reads[i].TimestampUS < reads[j].TimestampUS
		})
		b, _ := json.MarshalIndent(reads, "", " ")
		if err := os.WriteFile(out, b, 0o644); err != nil {
			t.Logf("ecriture %s : %v", out, err)
		} else {
			t.Logf("%d lectures plausibles ecrites dans %s", len(reads), out)
		}
	}
}

// TestInventoryComponentsDeltaCensus — LE RECENSEMENT : pour chaque composant d'inventaire,
// combien de records delta l'annoncent au masque, et sur combien la marche des desers de
// PRODUCTION l'atteint. C'est ce qui dit, composant par composant, si le curseur moteur est
// reconstructible hors capture Cheat Engine.
func TestInventoryComponentsDeltaCensus(t *testing.T) {
	dir := os.Getenv(i22FilmEnv)
	if dir == "" {
		t.Skipf("%s non defini : sonde de recherche sautee", i22FilmEnv)
	}
	release := LockProcessDecode()
	defer release()

	n := CountFilmChunks(dir)
	chunks := make([]int, 0, n)
	for i := 1; i <= n; i++ {
		chunks = append(chunks, i)
	}
	slots := bipedSlotBandDir(dir, chunks)
	lay, _, err := DetectI0Layout(dir)
	if err != nil {
		t.Fatalf("i0 : %v", err)
	}
	arch, err := bipedArchetypeDir(dir)
	if err != nil {
		t.Fatalf("archetype : %v", err)
	}
	targets := []int{22, 25, 28, 30, 31, 32, 33, 34, 42, 43, 44, 47, 48, 54, 57}
	announced := map[int]int{}
	walked := map[int]int{}
	var records int

	minRecord := bipedHeaderBits + bipedIndexBits*bipedMinMaskCnt + lay.TotalBits()
	for _, c := range chunks {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeDelta {
				continue
			}
			pay := pk.Payload(data)
			total := len(pay) * 8
			for p := 0; p+minRecord <= total; {
				i0, _, idx, ok := matchBipedHeader(pay, p, total, slots, true, lay)
				if !ok {
					p++
					continue
				}
				records++
				for _, tg := range targets {
					if !maskHas(idx, tg) {
						continue
					}
					announced[tg]++
					if walkRecordTo(pay, i0, total, idx, lay, arch, tg) {
						walked[tg]++
					}
				}
				p = i0 + lay.TotalBits()
			}
		}
	}
	t.Logf("records biped delta = %d", records)
	for _, tg := range targets {
		a, w := announced[tg], walked[tg]
		r := 0.0
		if a > 0 {
			r = 100 * float64(w) / float64(a)
		}
		t.Logf("i%-3d  %-45s  masque=%-7d  atteint=%-7d  (%.1f %%)",
			tg, arch.component(tg), a, w, r)
	}
}

// walkCursorTo — variante JETABLE de walkRecordTo qui rend la POSITION BIT a laquelle le
// composant cible commence. Les desers de production ne publient rien pour i30/i33/i42/i47 ;
// cette sonde relit donc les bits a la position que la marche etablit, sans toucher au deser.
func walkCursorTo(pay []byte, i0, total int, idx []int, lay I0Layout, arch Archetype, target int) (int, bool) {
	at := i0 + lay.TotalBits() + i0TailBits
	for _, id := range idx[1:] {
		name := arch.component(id)
		if name == "" {
			return 0, false
		}
		if id == target {
			return at, true
		}
		br := NewBitReader(pay)
		br.SetBitPos(at)
		_, _, ported := consumeByName(br, name, uint32(BipedTypeIndex), arch.Level(id))
		if !ported || br.BitPos() > total {
			return 0, false
		}
		at = br.BitPos()
	}
	return 0, false
}

// TestInventoryValuesDeltaProbe relit les VALEURS des composants dont le deser de production
// ne publie rien : i30/i33 (chargeur R(8) apres une porte active-bas), i31/i34 (reserve R(11)),
// i42 (selecteur), i47 ([6 masque][3 selection]).
func TestInventoryValuesDeltaProbe(t *testing.T) {
	dir := os.Getenv(i22FilmEnv)
	if dir == "" {
		t.Skipf("%s non defini", i22FilmEnv)
	}
	release := LockProcessDecode()
	defer release()

	n := CountFilmChunks(dir)
	chunks := make([]int, 0, n)
	for i := 1; i <= n; i++ {
		chunks = append(chunks, i)
	}
	slots := bipedSlotBandDir(dir, chunks)
	lay, _, err := DetectI0Layout(dir)
	if err != nil {
		t.Fatalf("i0 : %v", err)
	}
	arch, err := bipedArchetypeDir(dir)
	if err != nil {
		t.Fatalf("archetype : %v", err)
	}

	mag := map[int][]uint32{30: nil, 33: nil}
	res := map[int][]uint32{31: nil, 34: nil}
	var selOK, selTot int
	i47Sel := map[uint32]int{}
	i42Vals := map[uint32]int{}

	minRecord := bipedHeaderBits + bipedIndexBits*bipedMinMaskCnt + lay.TotalBits()
	for _, c := range chunks {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(data) {
			if pk.Type != PacketTypeDelta {
				continue
			}
			pay := pk.Payload(data)
			total := len(pay) * 8
			for p := 0; p+minRecord <= total; {
				i0, _, idx, ok := matchBipedHeader(pay, p, total, slots, true, lay)
				if !ok {
					p++
					continue
				}
				for _, tg := range []int{30, 33} {
					if !maskHas(idx, tg) {
						continue
					}
					if at, ok := walkCursorTo(pay, i0, total, idx, lay, arch, tg); ok {
						if readBitsAt(pay, at, 1) == 0 { // porte active-bas : chargeur present
							mag[tg] = append(mag[tg], readBitsAt(pay, at+1, 8))
						}
					}
				}
				for _, tg := range []int{31, 34} {
					if !maskHas(idx, tg) {
						continue
					}
					if at, ok := walkCursorTo(pay, i0, total, idx, lay, arch, tg); ok {
						res[tg] = append(res[tg], readBitsAt(pay, at, 11))
					}
				}
				if maskHas(idx, 47) {
					if at, ok := walkCursorTo(pay, i0, total, idx, lay, arch, 47); ok {
						m := readBitsAt(pay, at, 6)
						s := readBitsAt(pay, at+6, 3)
						selTot++
						i47Sel[s]++
						// LE TEST REFUTABLE du handoff : la selection appartient au masque.
						if s >= 1 && s <= 4 && m&(1<<(s-1)) != 0 {
							selOK++
						}
					}
				}
				if maskHas(idx, 42) {
					if at, ok := walkCursorTo(pay, i0, total, idx, lay, arch, 42); ok {
						i42Vals[readBitsAt(pay, at, 7)]++
					}
				}
				p = i0 + lay.TotalBits()
			}
		}
	}
	for _, tg := range []int{30, 33} {
		t.Logf("i%d chargeur R(8) : n=%d  %s", tg, len(mag[tg]), i22Quantiles(mag[tg]))
	}
	for _, tg := range []int{31, 34} {
		t.Logf("i%d reserve  R(11): n=%d  %s", tg, len(res[tg]), i22Quantiles(res[tg]))
	}
	t.Logf("i47 : n=%d  selection dans le masque = %d (%.1f %%)  histogramme selection = %v",
		selTot, selOK, 100*float64(selOK)/float64(max(selTot, 1)), i47Sel)
	t.Logf("i42 : histogramme des 7 premiers bits = %v", i42Vals)
}

func i22Quantiles(v []uint32) string {
	if len(v) == 0 {
		return "(vide)"
	}
	c := append([]uint32(nil), v...)
	sort.Slice(c, func(i, j int) bool { return c[i] < c[j] })
	q := func(f float64) uint32 { return c[int(f*float64(len(c)-1))] }
	distinct := map[uint32]bool{}
	for _, x := range c {
		distinct[x] = true
	}
	return fmt.Sprintf("min=%d p10=%d p50=%d p90=%d max=%d distinctes=%d",
		c[0], q(0.1), q(0.5), q(0.9), c[len(c)-1], len(distinct))
}

func i22SortMap(m map[uint64]int) string {
	keys := make([]uint64, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	s := ""
	for _, k := range keys {
		s += fmt.Sprintf("%d:%d ", k, m[k])
	}
	return s
}

func i22TopMap(m map[uint64]int, n int) string {
	type kv struct {
		k uint64
		v int
	}
	all := make([]kv, 0, len(m))
	for k, v := range m {
		all = append(all, kv{k, v})
	}
	sort.Slice(all, func(i, j int) bool { return all[i].v > all[j].v })
	if len(all) > n {
		all = all[:n]
	}
	s := fmt.Sprintf("(%d valeurs distinctes) ", len(m))
	for _, e := range all {
		s += fmt.Sprintf("%d:%d ", e.k, e.v)
	}
	return s
}
