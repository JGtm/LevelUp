// tmp_kfmatch — décode la CAPTURE LIVE des deltas biped de TOUS les joueurs
// (allbipeds_capture.txt : lignes "id bp%8 byteoff hex96", hook CE non-bloquant sur le
// processeur de delta COMMUN FUN_1406caad8 filtré ti=35) en trajectoires par joueur.
//
// Découverte 2026-07-04 : « seulement 2 joueurs » était un artefact de chemin de
// réplication (FUN_1406cbaa0 = 2 entités possédées POV+bot). Le processeur commun
// FUN_1406caad8 voit les 12 bipeds. Chaque delta = mask + composants (ti=35 lié dans le
// World, PAS de header) ; le i0 object-position (composant index 0) émet la position via
// le hook AVANT tout desync éventuel.
//
// RECONSTRUCTION : le i0 encode soit un ABSOLU (PosKindAbsolute/AbsFallback = coord monde,
// fixe la position), soit un DELTA (PosKindDelta8/DeltaAxis = offset relatif, à accumuler).
// PosKindRaw = keep-baseline (PAS une position, ignoré). On reconstruit la position monde
// par slot : set sur absolu, += sur delta (à partir du 1er absolu vu).
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_kfmatch <allbipeds_capture.txt> [filmDir]
package main

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"

	"levelup/go-api/internal/analysis/filmdec"
)

const defFilm = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`

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

type sample struct {
	kind filmdec.PosKind
	vec  [3]float32
	has  bool
}

func main() {
	capPath := os.Args[1]
	// AxisW=15 = largeur calibrée pour cette map (transition de phase du pas moyen : 179 à
	// W=14 -> 0.07 à W=15 = alignement des bits). Trajectoires lisses+physiques (span ~125u,
	// pas max 0.23u/frame ~14u/s, Z quasi-plat). Override en arg2 pour re-sweeper.
	axisW := uint(15)
	if len(os.Args) > 2 {
		if v, e := strconv.Atoi(os.Args[2]); e == nil {
			axisW = uint(v)
		}
	}
	film := defFilm
	if len(os.Args) > 3 {
		film = os.Args[3]
	}
	reg, err := filmdec.ParseRegistryChunk(inflate(film + "/chunk_00.bin"))
	if err != nil {
		panic(err)
	}
	filmdec.SetFilmComponentCorruptionCheck(true)
	filmdec.TraversalPrecision = filmdec.PrecisionDescriptor{IndexW: 1, AxisW: [3]uint{axisW, axisW, axisW}}
	// DSPRESKIP : sweep de calibration du désalignement amont du dead-state (kill feed).
	dsps := 0
	if v, e := strconv.Atoi(os.Getenv("DSPRESKIP")); e == nil {
		dsps = v
		filmdec.SetDeadStatePreSkip(v)
	}
	fmt.Printf("AxisW=%d DSPRESKIP=%d\n", axisW, dsps)

	var cur sample
	filmdec.SetPositionCaptureHook(func(s filmdec.PositionSample) {
		if !cur.has {
			cur = sample{kind: s.Kind, vec: s.Vec, has: true}
		}
	})

	w := filmdec.NewWorld(reg)
	f, err := os.Open(capPath)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	perSlot := map[uint32][]sample{}
	var order []uint32
	kinds := map[filmdec.PosKind]int{}
	var deaths []string
	nrec, ndesync, ndead := 0, 0, 0
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) < 4 {
			continue
		}
		id64, _ := strconv.ParseUint(fields[0], 16, 64)
		id := uint32(id64)
		bitoff, _ := strconv.Atoi(fields[1])
		data, e := hex.DecodeString(fields[3])
		if e != nil {
			continue
		}
		slot := id & 0x3fffffff
		w.BindFull(id, 35)
		cur = sample{}
		t, _ := filmdec.DecodeDeltaRecordAt(data, bitoff, w, slot)
		if _, seen := perSlot[slot]; !seen {
			order = append(order, slot)
		}
		perSlot[slot] = append(perSlot[slot], cur)
		if cur.has {
			kinds[cur.kind]++
		}
		nrec++
		if t.DesyncAt != -1 {
			ndesync++
		}
		// KILL FEED : le dead-state biped (composant object-dead-state) porte victime/tueur/arme.
		if t.Dead != nil {
			ndead++
			deaths = append(deaths, fmt.Sprintf(
				"slot=%X mort=%v victime(A)=%d tueur(B)=%d armeGID=0x%X hasRef=%v gid=%v",
				slot, t.Dead.Mort, t.Dead.EnumA, t.Dead.EnumB, t.Dead.GlobalID, t.Dead.HasRef, t.Dead.GIDPresent))
		}
	}
	sort.Slice(order, func(i, j int) bool { return order[i] < order[j] })

	fmt.Printf("=== %d records | %d desync | %d slots ===\n", nrec, ndesync, len(order))
	fmt.Print("kinds i0: ")
	for k := filmdec.PosKindRaw; k <= filmdec.PosKindDeltaAxis; k++ {
		if kinds[k] > 0 {
			fmt.Printf("%s=%d ", k, kinds[k])
		}
	}
	fmt.Println()

	fmt.Printf("=== KILL FEED : %d events dead-state ===\n", ndead)
	for i, d := range deaths {
		if i >= 40 {
			fmt.Printf("  ... (%d de plus)\n", len(deaths)-40)
			break
		}
		fmt.Println("  " + d)
	}

	isAbs := func(k filmdec.PosKind) bool {
		return k == filmdec.PosKindAbsolute || k == filmdec.PosKindAbsFallback
	}
	isDelta := func(k filmdec.PosKind) bool {
		return k == filmdec.PosKindDelta8 || k == filmdec.PosKindDeltaAxis
	}

	// reconstruction par slot : set sur absolu, += sur delta (dès qu'un absolu a ancré)
	fmt.Println("--- trajectoires reconstruites (absolu fixe, delta accumule) ---")
	var bestSlot uint32
	var bestTraj [][3]float32
	for _, slot := range order {
		var run [3]float32
		anchored := false
		var traj [][3]float32
		for _, s := range perSlot[slot] {
			if !s.has {
				continue
			}
			switch {
			case isAbs(s.kind):
				run, anchored = s.vec, true
			case isDelta(s.kind) && anchored:
				run[0] += s.vec[0]
				run[1] += s.vec[1]
				run[2] += s.vec[2]
			default:
				continue
			}
			traj = append(traj, run)
		}
		if len(traj) == 0 {
			fmt.Printf("slot=%-5X (aucun absolu ancre)\n", slot)
			continue
		}
		first, last := traj[0], traj[len(traj)-1]
		// amplitude du mouvement
		var span [3]float32
		mn, mx := first, first
		for _, p := range traj {
			for i := 0; i < 3; i++ {
				if p[i] < mn[i] {
					mn[i] = p[i]
				}
				if p[i] > mx[i] {
					mx[i] = p[i]
				}
			}
		}
		for i := 0; i < 3; i++ {
			span[i] = mx[i] - mn[i]
		}
		fmt.Printf("slot=%-5X %-4d pts | first=[%7.1f %7.1f %7.1f] last=[%7.1f %7.1f %7.1f] span=[%6.1f %6.1f %6.1f]\n",
			slot, len(traj), first[0], first[1], first[2], last[0], last[1], last[2], span[0], span[1], span[2])
		if len(traj) > len(bestTraj) {
			bestSlot, bestTraj = slot, traj
		}
	}

	// FLUIDITÉ : distance 3D moyenne/médiane entre positions consécutives. Un joueur bouge
	// de quelques unités par frame ; la bonne calibration MINIMISE le pas (sans collapse span→0).
	if len(bestTraj) > 1 {
		var steps []float64
		var sum float64
		for i := 1; i < len(bestTraj); i++ {
			dx := float64(bestTraj[i][0] - bestTraj[i-1][0])
			dy := float64(bestTraj[i][1] - bestTraj[i-1][1])
			dz := float64(bestTraj[i][2] - bestTraj[i-1][2])
			d := math.Sqrt(dx*dx + dy*dy + dz*dz)
			steps = append(steps, d)
			sum += d
		}
		sort.Float64s(steps)
		med := steps[len(steps)/2]
		p90 := steps[len(steps)*9/10]
		fmt.Printf("FLUIDITE slot 0x%X (n=%d): pas moyen=%.2f median=%.2f p90=%.2f max=%.2f\n",
			bestSlot, len(bestTraj), sum/float64(len(steps)), med, p90, steps[len(steps)-1])
	}
}
