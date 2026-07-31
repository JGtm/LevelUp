// tmp_objsol2 — POSITIONS des objets au sol (ti=42 armes, ti=37 equipement) lues dans les
// paquets DELTA avec le detecteur d en-tete deja valide (ScanBipedRecords), pilote par le
// jeu de slots de l archetype voulu au lieu du jeu de slots bipede. THROWAWAY.
//
// POURQUOI CE CHEMIN. Le detecteur d en-tete de offline_biped.go n a RIEN de bipede : il
// teste la grammaire generique [prefixe][slot 13b][tag][gate][maskCount][index...] puis lit
// i0. Or i0 = object-position-component est le composant d INDEX 0 de ti=37, 38, 41, 42 et
// 43 exactement comme de ti=35 (verifie sur le registre du film, cmd/tmp_objcomp). Le seul
// parametre archetype-dependant est donc le JEU DE SLOTS, et il se lit dans les keyframes.
//
// CONTROLE NEGATIF INTEGRE : le meme balayage tourne sur un jeu de slots FANTOME de meme
// cardinalite, tire dans les slots que AUCUN keyframe n a jamais declares. Ce qu il rend
// est le bruit du detecteur, mesure et non calcule.
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_objsol2 [filmDir]
package main

import (
	"fmt"
	"math/rand"
	"os"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
)

const defFilm = `C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`

var worldRange = filmdec.Vec3Range{
	{Min: -41.102932, Max: 72.109375},
	{Min: -56.606728, Max: 57.211975},
	{Min: -84.37054, Max: 53.179653},
}

type sample struct {
	slot       uint32
	tSec       float64
	x, y, z    float32
	q          [3]uint32
	chunk, pkt int
}

func main() {
	dir := defFilm
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	n := filmdec.CountFilmChunks(dir)

	lay, _, err := filmdec.DetectI0Layout(dir)
	if err != nil {
		fmt.Println("decoupage i0:", err)
		return
	}
	fmt.Printf("film %s : %d chunks · decoupage i0 %s\n", dir, n, lay)

	// --- jeux de slots par archetype, lus dans TOUS les keyframes ---
	slotsByTI := map[int]map[uint32]bool{}
	allDeclared := map[uint32]bool{}
	var t0 uint64
	first := true
	for c := 1; c <= n; c++ {
		data, err := filmdec.ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, pk := range filmdec.WalkPackets(data) {
			if first {
				t0, first = pk.TimestampUS, false
			}
			if pk.Type != filmdec.PacketTypeKeyframe {
				continue
			}
			for _, r := range filmdec.WalkKeyframeWorld(pk.Payload(data)) {
				if slotsByTI[r.TI] == nil {
					slotsByTI[r.TI] = map[uint32]bool{}
				}
				slotsByTI[r.TI][uint32(r.Slot)] = true
				allDeclared[uint32(r.Slot)] = true
			}
		}
	}
	for _, ti := range []int{35, 37, 42} {
		fmt.Printf("  ti=%d : %d slots declares dans les keyframes\n", ti, len(slotsByTI[ti]))
	}

	// --- controle negatif : meme cardinalite que ti=42, slots JAMAIS declares ---
	ghost := ghostSlots(allDeclared, len(slotsByTI[42]), 7)
	fmt.Printf("  controle negatif : %d slots fantomes (aucun jamais declare)\n\n", len(ghost))

	opt := filmdec.ScanFilmOptions{RequireTag1: true, DropSaturated: true, WorldRange: &worldRange}
	optNoTag := opt
	optNoTag.RequireTag1 = false

	for _, cas := range []struct {
		nom   string
		slots map[uint32]bool
		o     filmdec.ScanFilmOptions
	}{
		{"ti=42 armes au sol (tag1)", slotsByTI[42], opt},
		{"ti=42 armes au sol (sans tag1)", slotsByTI[42], optNoTag},
		{"ti=37 equipement (tag1)", slotsByTI[37], opt},
		{"ti=35 bipedes (tag1) [temoin positif]", slotsByTI[35], opt},
		{"FANTOME meme cardinalite que ti=42 (tag1) [controle negatif]", ghost, opt},
	} {
		s := scan(dir, n, cas.slots, lay, cas.o, t0)
		perSlot := map[uint32]int{}
		for _, x := range s {
			perSlot[x.slot]++
		}
		fmt.Printf("%-62s : %6d echantillons · %4d slots touches (sur %d declares)\n",
			cas.nom, len(s), len(perSlot), len(cas.slots))
	}

	// --- detail ti=42 : premier echantillon de chaque slot (= position d apparition) ---
	s42 := scan(dir, n, slotsByTI[42], lay, opt, t0)
	sort.Slice(s42, func(i, j int) bool { return s42[i].tSec < s42[j].tSec })
	firstOf := map[uint32]sample{}
	var order []uint32
	for _, x := range s42 {
		if _, ok := firstOf[x.slot]; !ok {
			firstOf[x.slot] = x
			order = append(order, x.slot)
		}
	}
	sort.Slice(order, func(i, j int) bool { return firstOf[order[i]].tSec < firstOf[order[j]].tSec })
	fmt.Printf("\n=== ti=42 : premiere position observee par slot (%d slots) ===\n", len(order))
	for i, sl := range order {
		f := firstOf[sl]
		fmt.Printf("  %3d. slot %5d t=%7.1f s  (%7.2f, %7.2f, %7.2f)  n=%d\n",
			i+1, sl, f.tSec, f.x, f.y, f.z, countSlot(s42, sl))
		if i >= 120 {
			fmt.Printf("  ... (%d slots supplementaires)\n", len(order)-i-1)
			break
		}
	}
}

func countSlot(s []sample, sl uint32) int {
	n := 0
	for _, x := range s {
		if x.slot == sl {
			n++
		}
	}
	return n
}

func scan(dir string, n int, slots map[uint32]bool, lay filmdec.I0Layout,
	opt filmdec.ScanFilmOptions, t0 uint64) []sample {
	if len(slots) == 0 {
		return nil
	}
	var out []sample
	for c := 1; c <= n; c++ {
		data, err := filmdec.ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, pk := range filmdec.WalkPackets(data) {
			if pk.Type != filmdec.PacketTypeDelta {
				continue
			}
			for _, r := range filmdec.ScanBipedRecords(pk.Payload(data), slots, lay, opt) {
				out = append(out, sample{
					slot: r.Slot, tSec: float64(pk.TimestampUS-t0) / 1e6,
					x: r.X, y: r.Y, z: r.Z, q: r.Q, chunk: c, pkt: pk.Index,
				})
			}
		}
	}
	return out
}

// ghostSlots tire `want` slots jamais declares par aucun keyframe, dans la meme plage que
// les slots reels (1..3000) — un controle negatif hors plage ne mesurerait rien.
func ghostSlots(declared map[uint32]bool, want int, seed int64) map[uint32]bool {
	rng := rand.New(rand.NewSource(seed))
	out := map[uint32]bool{}
	for len(out) < want {
		s := uint32(rng.Intn(3000)) + 1
		if declared[s] || out[s] {
			continue
		}
		out[s] = true
	}
	return out
}
