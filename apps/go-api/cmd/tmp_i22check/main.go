// tmp_i22check — mesure i22 (unit-grenade-counts) et i48 (biped-desired-ability-set)
// contre les BORNES DE JEU, AVANT et APRÈS l'amorce de paquet. THROWAWAY.
//
// CRITERE ABSOLU (source utilisateur, GRAMMAIRE_RECORD_FILM.md §6) : au plus DEUX types
// de grenade portés, DEUX unités de chacun. Le désassemblage de FUN_140f0de1c donne la
// grammaire : compteur R(3) (FUN_1424d0f48, brut) puis count × R(8). La vérité terrain
// CE donne 35 bits FIXES (100 %, 259 mesures) = 3 + 4×8, donc count == 4 TOUJOURS, et
// la borne porte sur les VALEURS : au plus 2 non nulles, chacune <= 2.
//
// Deux mesures, donc :
//   - count != 4                      -> impossible (structure)
//   - une valeur > 2, ou > 2 non nulles -> impossible (borne de jeu)
//
// Le World est bootstrappé OFFLINE depuis le keyframe type-2 (WorldFromKeyframe) : aucun
// Cheat Engine, donc n'importe quel film du cache est mesurable, pas seulement 000d5950.
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_i22check [filmdir] [idLowSweep]
package main

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"levelup/go-api/internal/analysis/filmdec"
)

const defFilm = `C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`

func inflate(p string) []byte {
	raw, _ := os.ReadFile(p)
	if len(raw) >= 2 && raw[0] == 0x78 {
		if zr, e := zlib.NewReader(bytes.NewReader(raw)); e == nil {
			if d, e2 := io.ReadAll(zr); e2 == nil || len(d) > 0 {
				return d
			}
		}
	}
	return raw
}

func packetsOfType(d []byte, want uint16) [][]byte {
	var out [][]byte
	off := 0
	for off+16 <= len(d) {
		typ := binary.LittleEndian.Uint16(d[off:])
		sz := int(binary.LittleEndian.Uint32(d[off+4:]))
		if sz < 0 || off+16+sz > len(d) {
			break
		}
		if typ == want {
			out = append(out, d[off+16:off+16+sz])
		}
		off += 16 + sz
	}
	return out
}

func countChunks(dir string) int {
	n := 0
	for i := 1; ; i++ {
		if _, err := os.Stat(fmt.Sprintf("%s/chunk_%02d.bin", dir, i)); err != nil {
			return n
		}
		n = i
	}
}

type stat struct {
	records   int
	hits      int // records portant i22
	countHist map[uint64]int
	valHist   map[uint64]int
	badCount  int // count != 4
	badVal    int // au moins une valeur > 2
	badTypes  int // plus de 2 valeurs non nulles
	// présence par nom de composant (sur les records ti=35 DELTA)
	bipedRecs int
	presence  map[string]int
	// i48
	hits48  int
	v48Hist map[uint64]int
	w48Hist map[int]int
}

func newStat() *stat {
	return &stat{presence: map[string]int{}, countHist: map[uint64]int{}, valHist: map[uint64]int{}, v48Hist: map[uint64]int{}, w48Hist: map[int]int{}}
}

func main() {
	dir := defFilm
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	reg, err := filmdec.ParseRegistryChunk(inflate(dir + "/chunk_00.bin"))
	if err != nil {
		panic(err)
	}
	// keyframe : le paquet type-2 vit dans chunk_02 sur les films observés ; on balaie
	// les premiers chunks jusqu'à en trouver un.
	var kf []byte
	for i := 1; i <= 4 && kf == nil; i++ {
		for _, p := range packetsOfType(inflate(fmt.Sprintf("%s/chunk_%02d.bin", dir, i)), 2) {
			if len(p) > len(kf) {
				kf = p
			}
		}
	}
	if kf == nil {
		fmt.Println("AUCUN keyframe type-2 : impossible de bootstrapper le World")
		os.Exit(1)
	}
	nc := countChunks(dir)
	var frames [][]byte
	for i := 3; i <= nc; i++ {
		frames = append(frames, packetsOfType(inflate(fmt.Sprintf("%s/chunk_%02d.bin", dir, i)), 0)...)
	}
	// Bindings : le dump CE (world_dump_full.txt) s'il existe — c'est l'oracle, il couvre
	// tous les slots ; sinon le bootstrap keyframe OFFLINE (moins couvrant, mais universel).
	dumpBind := loadDump(dir + "/world_dump_full.txt")
	mkWorld := func() *filmdec.World {
		if len(dumpBind) > 0 {
			w := filmdec.NewWorld(reg)
			for id, ti := range dumpBind {
				w.BindFull(id, ti)
			}
			return w
		}
		return filmdec.WorldFromKeyframe(reg, kf)
	}
	src := "keyframe OFFLINE"
	if len(dumpBind) > 0 {
		src = "world_dump CE"
	}
	fmt.Printf("film=%s chunks=%d frames(type0)=%d keyframe=%d o -> World (%s) : %d slots\n",
		dir, nc, len(frames), len(kf), src, mkWorld().Bound())

	idLows := []int{11}
	if len(os.Args) > 2 {
		idLows = nil
		for _, s := range []string{"9", "10", "11", "12", "13"} {
			v, _ := strconv.Atoi(s)
			idLows = append(idLows, v)
		}
	}

	run := func(pre, idlow int) *stat {
		st := newStat()
		filmdec.SetGrenadeCountsHook(func(count uint64, vals []uint64) {
			st.hits++
			st.countHist[count]++
			nz, bad := 0, false
			for _, v := range vals {
				st.valHist[v]++
				if v != 0 {
					nz++
				}
				if v > 2 {
					bad = true
				}
			}
			if count != 4 {
				st.badCount++
			}
			if bad {
				st.badVal++
			}
			if nz > 2 {
				st.badTypes++
			}
		})
		filmdec.SetAbilitySetHook(func(v uint64, width int) {
			st.hits48++
			st.v48Hist[v]++
			st.w48Hist[width]++
		})
		defer func() {
			filmdec.SetGrenadeCountsHook(nil)
			filmdec.SetAbilitySetHook(nil)
		}()
		// CALIBRATION DU FILM. Source : .ai/ETAT_DE_L_ART_KILLWEAPON.md tableau 5.1, ou
		// cv_autocalib mesure QUATRE couples (axisW, indexW) DISTINCTS sur quatre films.
		// Pour 000d5950 : axisW = 14, indexW = 1, recordStateParam = 3. Notre defaut 6/6/6
		// avec rsp = 0 n'est la calibration d'AUCUN film.
		// Reglable par I22_AXISW / I22_INDEXW / I22_RSP (os.Args[1] est deja le dossier du film).
		if w := envInt("I22_AXISW", 0); w > 0 {
			filmdec.TraversalPrecision = filmdec.PrecisionDescriptor{
				IndexW: uint(envInt("I22_INDEXW", 1)),
				AxisW:  [3]uint{uint(w), uint(w), uint(w)}}
		}
		// i0 EST LA PREMIERE FAUTE DE LECTURE (mesure cmd/tmp_widthdiff) : notre mode vaut
		// 99 bits a 24 % seulement, la verite 47 bits a 100 % sur 134 767 mesures. Cet
		// interrupteur saute exactement 47 (ou 101 sur le chemin keep-baseline).
		filmdec.PositionCalibratedSkip = envInt("I22_CALIBSKIP", 0) != 0
		if r := envInt("I22_RSP", -1); r >= 0 {
			filmdec.SetRecordStateParam(uint32(r))
		}
		cfg := filmdec.FrameConfig{IDLowBits: idlow, PacketPreambleBits: pre}
		for _, pay := range frames {
			w := mkWorld()
			recs, _ := filmdec.DecodeFrameRecords(filmdec.NewBitReader(pay), w, cfg)
			st.records += len(recs)
			for _, rc := range recs {
				if rc.Type != 3 || rc.TypeIndex != 35 {
					continue
				}
				st.bipedRecs++
				for _, c := range rc.Trace.Comps {
					st.presence[c.Name]++
				}
			}
		}
		return st
	}

	for _, idlow := range idLows {
		for _, pre := range []int{0, 2} {
			st := run(pre, idlow)
			lbl := "AVANT (amorce 0)"
			if pre == 2 {
				lbl = "APRES (amorce 2)"
			}
			fmt.Printf("\n===== %s — idLow=%d =====\n", lbl, idlow)
			fmt.Printf("records décodés : %d | i22 atteint : %d (%.2f %% des records)\n",
				st.records, st.hits, pct(st.hits, st.records))
			if st.hits > 0 {
				fmt.Printf("  count != 4 (impossible)         : %6d / %6d = %6.2f %%\n", st.badCount, st.hits, pct(st.badCount, st.hits))
				fmt.Printf("  une valeur > 2 (impossible)     : %6d / %6d = %6.2f %%\n", st.badVal, st.hits, pct(st.badVal, st.hits))
				fmt.Printf("  > 2 types portés (impossible)   : %6d / %6d = %6.2f %%\n", st.badTypes, st.hits, pct(st.badTypes, st.hits))
				fmt.Printf("  distribution du compteur : %s\n", hist(st.countHist, 8))
				fmt.Printf("  distribution des valeurs : %s\n", hist(st.valHist, 10))
			}
			fmt.Printf("records bipedes (ti=35 DELTA) : %d | presence i22=%.3f%% i47=%.3f%% i48=%.3f%% i56=%.3f%% (verite : 0.19 / 0.14 / 0.13 / 0.15)\n",
				st.bipedRecs,
				pct(st.presence["unit-grenade-counts-component"], st.bipedRecs),
				pct(st.presence["biped-desired-grenade-set-component"]+st.presence["biped-desired-grenade-set"], st.bipedRecs),
				pct(st.presence["biped-desired-ability-set-component"]+st.presence["biped-desired-ability-set"], st.bipedRecs),
				pct(st.presence["biped-spartan-ability-energy-component"]+st.presence["biped-spartan-ability-energy"], st.bipedRecs))
			fmt.Printf("i48 atteint : %d (%.2f %% des records) | valeurs R(3) : %s | largeurs : %s\n",
				st.hits48, pct(st.hits48, st.records), hist(st.v48Hist, 8), histI(st.w48Hist, 4))
		}
	}
}

// loadDump lit le dump de bindings CE (tokens « id:typeIndex »).
func loadDump(path string) map[uint32]uint32 {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	out := map[uint32]uint32{}
	sc := bufio.NewScanner(bytes.NewReader(raw))
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		for _, tok := range strings.Fields(line) {
			kv := strings.SplitN(tok, ":", 2)
			if len(kv) != 2 {
				continue
			}
			id, e1 := strconv.ParseUint(kv[0], 10, 64)
			ti, e2 := strconv.Atoi(kv[1])
			if e1 != nil || e2 != nil {
				continue
			}
			out[uint32(id)] = uint32(ti)
		}
	}
	return out
}

func pct(a, b int) float64 {
	if b == 0 {
		return 0
	}
	return 100 * float64(a) / float64(b)
}

func hist(h map[uint64]int, k int) string {
	type kv struct {
		k uint64
		v int
	}
	var a []kv
	tot := 0
	for x, y := range h {
		a = append(a, kv{x, y})
		tot += y
	}
	sort.Slice(a, func(i, j int) bool { return a[i].v > a[j].v })
	var b bytes.Buffer
	for i := 0; i < len(a) && i < k; i++ {
		fmt.Fprintf(&b, " %d:%.1f%%", a[i].k, 100*float64(a[i].v)/float64(tot))
	}
	if len(a) > k {
		fmt.Fprintf(&b, " (+%d autres valeurs)", len(a)-k)
	}
	return b.String()
}

func histI(h map[int]int, k int) string {
	m := map[uint64]int{}
	for x, y := range h {
		m[uint64(x)] = y
	}
	return hist(m, k)
}

// envInt lit un entier d'environnement ; def si absent ou illisible.
func envInt(k string, def int) int {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}
