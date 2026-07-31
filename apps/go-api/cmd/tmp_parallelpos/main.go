// tmp_parallelpos — DÉCODAGE PARALLÈLE par ancrage-id des positions i0 des bipeds dans
// les paquets type-0 (frame delta). THROWAWAY / harness de validation.
//
// Thèse testée (cadrage utilisateur) : les positions des N joueurs sont TOUTES dans le
// film, trouvables EN PARALLÈLE — chaque record biped est repérable INDÉPENDAMMENT par
// son id (pas de walk séquentiel). La position i0 est à un OFFSET RELATIF (id -> [tag]
// [mask] -> i0) court après l'id.
//
// MÉTHODE : pour chaque paquet type-0, on SCANNE chaque bit ; à chaque position on tente
// de décoder un record DELTA (TryDeltaAt : type-prefix + id + mask + composants). Si le
// slot ∈ cibles (bipeds frame-space, bornés ti=35) ET le décode est propre ET i0 a émis
// un sample -> on accepte le record, on note recStart (bit de l'id), i0cursor (bit de
// début d'i0), la position décodée et le KIND (abs/d8/dax). C'est l'ancrage-id parallèle :
// on ne dépend JAMAIS du décode des records précédents.
//
// JUGE : (1) couverture = combien de slots cibles trouvés densément par paquet ;
// (2) offset id->i0 constant ? ; (3) les (slot,i0cursor) décodés matchent-ils l'oracle
// ce_pos_oracle.csv (bitCursor + x/y/z) ? ; (4) distribution des deltas.
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_parallelpos [filmDir] [oracleCsv] [deltaCsv]
package main

import (
	"bufio"
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"

	"levelup/go-api/internal/analysis/filmdec"
)

const (
	defFilm   = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`
	defOracle = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/.claude/worktrees/filmdec-continuation/.ai/re_dump/ce_pos_oracle.csv`
	defDelta  = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/.claude/worktrees/filmdec-continuation/.ai/re_dump/ce_capture_delta.csv`
	scratch   = `c:/Users/GUILLA~1/AppData/Local/Temp/claude/c--Users-Guillaume-Downloads-Scripts-LevelUp-go-migration/d6f109c3-2822-4e19-8fd4-cf65c50fec3d/scratchpad`
)

func inflate(p string) []byte {
	raw, err := os.ReadFile(p)
	if err != nil {
		return nil
	}
	if len(raw) >= 2 && raw[0] == 0x78 {
		if zr, e := zlib.NewReader(bytes.NewReader(raw)); e == nil {
			if d, e2 := io.ReadAll(zr); (e2 == nil || e2 == io.ErrUnexpectedEOF) && len(d) > 0 {
				return d
			}
		}
	}
	return raw
}

// listFrames retourne tous les payloads de frames type `want` d'un chunk inflé.
func listFrames(d []byte, want uint16) [][]byte {
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

// ---- oracle ce_pos_oracle.csv ----

type oraRow struct {
	slot, cursor int
	x, y, z      float32
}

func loadOracle(p string) []oraRow {
	f, err := os.Open(p)
	if err != nil {
		return nil
	}
	defer f.Close()
	var rows []oraRow
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "eid") {
			continue
		}
		fld := strings.Split(line, ",")
		if len(fld) < 6 {
			continue
		}
		slot, _ := strconv.Atoi(fld[1])
		cur, _ := strconv.Atoi(fld[2])
		x, _ := strconv.ParseFloat(fld[3], 32)
		y, _ := strconv.ParseFloat(fld[4], 32)
		z, _ := strconv.ParseFloat(fld[5], 32)
		rows = append(rows, oraRow{slot, cur, float32(x), float32(y), float32(z)})
	}
	return rows
}

// modalTi lit ce_capture_delta.csv et renvoie, par slot frame-space, le typeIndex modal.
func modalTi(p string) map[uint32]uint32 {
	f, err := os.Open(p)
	if err != nil {
		return nil
	}
	defer f.Close()
	tally := map[uint32]map[uint32]int{}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		line := sc.Text()
		if line == "" || line[0] == '#' || strings.HasPrefix(line, "eid") {
			continue
		}
		fld := strings.Split(line, ",")
		if len(fld) < 2 {
			continue
		}
		eid, _ := strconv.Atoi(fld[0])
		ti, _ := strconv.Atoi(fld[1])
		slot := uint32(eid) & 0x3fffffff
		if tally[slot] == nil {
			tally[slot] = map[uint32]int{}
		}
		tally[slot][uint32(ti)]++
	}
	out := map[uint32]uint32{}
	for slot, m := range tally {
		best, bc := uint32(0), -1
		for ti, c := range m {
			if c > bc {
				best, bc = ti, c
			}
		}
		out[slot] = best
	}
	return out
}

func finite(v float32) bool { return !math.IsNaN(float64(v)) && !math.IsInf(float64(v), 0) }

func main() {
	dir, oraclePath, deltaPath := defFilm, defOracle, defDelta
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	if len(os.Args) > 2 {
		oraclePath = os.Args[2]
	}
	if len(os.Args) > 3 {
		deltaPath = os.Args[3]
	}
	reg, err := filmdec.ParseRegistryChunk(inflate(dir + "/chunk_00.bin"))
	if err != nil {
		panic(err)
	}
	oracle := loadOracle(oraclePath)
	binds := modalTi(deltaPath)

	// cibles = slots frame-space présents dans l'oracle de position (les entités dont i0
	// est répliqué). On les borne à leur ti modal (frame-space) pour résoudre l'archétype.
	oraSlots := map[uint32]uint32{} // slot -> ti (35 attendu pour bipeds)
	for _, r := range oracle {
		s := uint32(r.slot)
		if ti, ok := binds[s]; ok {
			oraSlots[s] = ti
		} else {
			oraSlots[s] = 35
		}
	}
	targets := map[uint32]bool{}
	tiHist := map[uint32]int{}
	for s, ti := range oraSlots {
		targets[s] = true
		tiHist[ti]++
	}
	fmt.Printf("registre=%d archétypes ; oracle=%d rows ; slots cibles=%d ; ti des cibles=%v\n",
		len(reg.Archetypes), len(oracle), len(targets), tiHist)

	// oracle: index (slot,cursor)->pos + trajectoire ordonnée par slot + comptes par slot
	oraByKey := map[[2]int][][3]float32{}
	oraSlotRows := map[int][]oraRow{}
	oraSlotCount := map[int]int{}
	for _, r := range oracle {
		k := [2]int{r.slot, r.cursor}
		oraByKey[k] = append(oraByKey[k], [3]float32{r.x, r.y, r.z})
		oraSlotRows[r.slot] = append(oraSlotRows[r.slot], r)
		oraSlotCount[r.slot]++
	}
	// oracle: distinct-slots par bloc-paquet (frontière = reset du cursor : chute > 400)
	oraPktSlots := oraPacketCoverage(oracle)

	// ===== TEST D'ALIGNEMENT DIRECT : 1re frame type-0 vs 1er bloc oracle =====
	alignProbe(dir, reg, binds, targets, oracle)
	if os.Getenv("FULL") == "" {
		fmt.Println("\n(FULL non défini : arrêt après la sonde d'alignement)")
		return
	}

	// IDLowBits=11 retenu (sweep antérieur : max cursorHits). lead calibré ci-dessous.
	cfg := filmdec.FrameConfig{HasExtraFields: false, IDLowBits: 11}
	recs := decodeAllParallel(dir, reg, binds, targets, cfg)
	bestLead, bestHits := calibrateLead(recs, oraByKey)
	fmt.Printf("\nIDLowBits=11 records=%d lead=%d cursorHits=%d (%.1f%%)\n",
		len(recs), bestLead, bestHits, 100*float64(bestHits)/float64(max1(len(recs))))

	// ---- couverture par paquet ----
	report(recs, oraByKey, bestLead, oraPktSlots, oraSlotCount)

	// ---- offset id->i0 ----
	offsetStats(recs)

	// ---- analyse RESTREINTE aux records "réels" (cursor exact dans l'oracle) ----
	realHitAnalysis(recs, oraByKey, bestLead)

	// ---- deltas vs oracle ----
	deltaStats(recs, oracle)

	// ---- trajectoires accumulées + PNG ----
	accumulateAndPlot(dir, reg, binds, targets, cfg, oraSlotRows)
}

// alignProbe teste l'alignement DIRECT : le 1er paquet type-0 du film (chunk_03 frame 0)
// doit correspondre au 1er bloc de l'oracle. On dump les records trouvés par ancrage-id
// parallèle (slot, recStart, i0cursor, pos, kind) et le 1er bloc oracle en regard. Test
// décisif : si le paquet contient bien les 7-8 slots aux cursors oracle -> thèse validée.
func alignProbe(dir string, reg *filmdec.Registry, binds map[uint32]uint32, targets map[uint32]bool, oracle []oraRow) {
	fmt.Println("\n=== SONDE D'ALIGNEMENT : 1er bloc oracle vs 1re frame type-0 ===")
	// 1er bloc oracle (jusqu'au reset du cursor)
	fmt.Println("  ORACLE bloc 0 (slot@cursor -> x,y,z) :")
	prev := -1
	nb := 0
	for _, r := range oracle {
		if prev >= 0 && r.cursor < prev-400 && nb > 0 {
			break
		}
		if nb < 16 {
			fmt.Printf("    %d@%-5d -> %.2f, %.2f, %.2f\n", r.slot, r.cursor, r.x, r.y, r.z)
		}
		prev = r.cursor
		nb++
	}
	// 1re frame type-0
	var first []byte
	for idx := 3; idx <= 26 && first == nil; idx++ {
		fs := listFrames(inflate(fmt.Sprintf("%s/chunk_%02d.bin", dir, idx)), 0)
		if len(fs) > 0 {
			first = fs[0]
			fmt.Printf("  1re frame type-0 = chunk_%02d frame 0, %d octets (%d bits)\n", idx, len(first), len(first)*8)
		}
	}
	if first == nil {
		fmt.Println("  aucune frame type-0 trouvée")
		return
	}
	// SÉQUENTIEL : décode la frame dans l'ordre (tous slots bornés) et dump les records
	// cibles avec leur i0cursor. Teste si l'ordre naturel reproduit les cursors oracle.
	seqProbe(reg, binds, targets, first)
	// DIRECT : décode i0 à l'EXACT cursor oracle dans chaque frame précoce, cherche la
	// frame qui reproduit les positions du bloc 0 (test d'alignement paquet, bypass scan).
	directAlign(dir, oracle)
	var last filmdec.PositionSample
	var have bool
	filmdec.SetPositionCaptureHook(func(s filmdec.PositionSample) { last, have = s, true })
	defer filmdec.SetPositionCaptureHook(nil)
	for _, low := range []int{11} {
		cfg := filmdec.FrameConfig{HasExtraFields: false, IDLowBits: low}
		w := filmdec.NewWorld(reg)
		for s := range targets {
			ti := binds[s]
			if ti == 0 {
				ti = 35
			}
			w.BindFull((1<<30)|s, ti)
		}
		fmt.Printf("  OFFLINE parallèle (IDLowBits=%d) records slot@i0cursor (recStart) kind pos :\n", low)
		n := 0
		frameLen := len(first) * 8
		for b := 0; b < frameLen-16 && n < 40; {
			have = false
			rec, end, ok := filmdec.TryDeltaAt(first, b, w, cfg)
			if ok && targets[rec.Slot] && have && finite(last.Vec[0]) {
				fmt.Printf("    %d@%-5d (rec%d) %-5s %.2f,%.2f,%.2f\n", rec.Slot, last.BitPos, b, last.Kind.String(), last.Vec[0], last.Vec[1], last.Vec[2])
				n++
				b = end
				continue
			}
			b++
		}
	}
}

// directAlign : prend le 1er bloc oracle (slot,cursor,pos) et, pour chaque frame type-0
// précoce, décode i0 à l'EXACT cursor et compte les positions reproduites (<0.5u). Trouve
// la frame alignée sur le bloc 0 (si le film et l'oracle partagent le même paquet).
func directAlign(dir string, oracle []oraRow) {
	// bloc 0 = jusqu'au 2e reset de cursor
	var block []oraRow
	prev, resets := -1, 0
	for _, r := range oracle {
		if prev >= 0 && r.cursor < prev-400 {
			resets++
			if resets >= 2 {
				break
			}
		}
		block = append(block, r)
		prev = r.cursor
	}
	// dédup (slot,cursor) en gardant la 1re position
	seen := map[[2]int]bool{}
	var uniq []oraRow
	for _, r := range block {
		k := [2]int{r.slot, r.cursor}
		if seen[k] {
			continue
		}
		seen[k] = true
		uniq = append(uniq, r)
	}
	var got filmdec.PositionSample
	filmdec.SetPositionCaptureHook(func(s filmdec.PositionSample) { got = s })
	defer filmdec.SetPositionCaptureHook(nil)
	fmt.Printf("  DIRECT : bloc0 = %d (slot,cursor) uniques ; balayage des 300 1res frames type-0 (i0 @cursor exact) :\n", len(uniq))
	bestFrame, bestMatch := -1, -1
	fidx := 0
	for idx := 3; idx <= 26; idx++ {
		for _, fr := range listFrames(inflate(fmt.Sprintf("%s/chunk_%02d.bin", dir, idx)), 0) {
			if fidx >= 300 {
				break
			}
			m := 0
			for _, r := range uniq {
				if r.cursor+16 > len(fr)*8 {
					continue
				}
				br := filmdec.NewBitReader(fr)
				br.Skip(r.cursor)
				filmdec.ProbePos(br, 1, 6, 6, 6)
				if abs32(got.Vec[0]-r.x) < 0.5 && abs32(got.Vec[1]-r.y) < 0.5 && abs32(got.Vec[2]-r.z) < 0.5 {
					m++
				}
			}
			if m > bestMatch {
				bestMatch, bestFrame = m, fidx
			}
			fidx++
		}
	}
	fmt.Printf("    meilleure frame=%d reproduit %d/%d positions du bloc0 (<0.5u)\n", bestFrame, bestMatch, len(uniq))
	if bestMatch <= 1 {
		fmt.Println("    => AUCUN alignement paquet film<->oracle (positions non reproduites au cursor exact) : soit l'oracle vient d'un autre playthrough, soit le deser i0 (range/largeur) ne reproduit pas ces coordonnées.")
	}
}

// seqProbe décode la frame EN ORDRE avec tous les slots bornés (binds), pour plusieurs
// skipLeadBits, et dump les records cibles + leur i0cursor. Compare l'ordre naturel aux
// cursors oracle. Utilise DecodeFrameViews (grammaire réelle du frame-processor).
func seqProbe(reg *filmdec.Registry, binds map[uint32]uint32, targets map[uint32]bool, fr []byte) {
	var samples []filmdec.PositionSample
	filmdec.SetPositionCaptureHook(func(s filmdec.PositionSample) { samples = append(samples, s) })
	defer filmdec.SetPositionCaptureHook(nil)
	for _, lead := range []int{0, 1} {
		w := filmdec.NewWorld(reg)
		for s, ti := range binds {
			if ti == 0 {
				ti = 35
			}
			w.BindFull((1<<30)|s, ti)
		}
		samples = samples[:0]
		cfg := filmdec.FrameConfig{HasExtraFields: false, IDLowBits: 11}
		recs, views := filmdec.DecodeFrameViews(fr, w, cfg, 4, lead)
		byBit := map[int]filmdec.PositionSample{}
		for _, s := range samples {
			byBit[s.BitPos] = s
		}
		nTarget := 0
		var buf strings.Builder
		for _, r := range recs {
			if !targets[r.Slot] {
				continue
			}
			for _, c := range r.Trace.Comps {
				if c.Name == "object-position-dynamic-precision-component" {
					if s, ok := byBit[c.StartBit]; ok && nTarget < 12 {
						fmt.Fprintf(&buf, "    %d@%-5d %-5s %.2f,%.2f,%.2f\n", r.Slot, c.StartBit, s.Kind.String(), s.Vec[0], s.Vec[1], s.Vec[2])
					}
					nTarget++
					break
				}
			}
		}
		fmt.Printf("  SÉQUENTIEL lead=%d : %d records, %d vues décodées, %d records cibles :\n%s", lead, len(recs), views, nTarget, buf.String())
	}
}

// pktRec = un record biped accepté par ancrage-id parallèle.
type pktRec struct {
	pkt      int
	slot     uint32
	recStart int // bit de l'id (début du record)
	i0cursor int // bit de début d'i0 (posCaptureStartBit)
	pos      [3]float32
	kind     filmdec.PosKind
}

// decodeAllParallel décode TOUS les paquets type-0 des chunks 3..26 par ancrage-id
// parallèle (scan bit-à-bit, TryDeltaAt sur slots cibles bornés). Pas d'accumulation.
func decodeAllParallel(dir string, reg *filmdec.Registry, binds map[uint32]uint32, targets map[uint32]bool, cfg filmdec.FrameConfig) []pktRec {
	var out []pktRec
	var lastSample filmdec.PositionSample
	var haveSample bool
	filmdec.SetPositionCaptureHook(func(s filmdec.PositionSample) { lastSample, haveSample = s, true })
	defer filmdec.SetPositionCaptureHook(nil)

	pkt := 0
	for idx := 3; idx <= 26; idx++ {
		for _, fr := range listFrames(inflate(fmt.Sprintf("%s/chunk_%02d.bin", dir, idx)), 0) {
			w := filmdec.NewWorld(reg)
			for s := range targets {
				ti := binds[s]
				if ti == 0 {
					ti = 35
				}
				w.BindFull((1<<30)|s, ti)
			}
			frameLen := len(fr) * 8
			b := 0
			for b < frameLen-16 {
				haveSample = false
				rec, end, ok := filmdec.TryDeltaAt(fr, b, w, cfg)
				if ok && targets[rec.Slot] && haveSample && finite(lastSample.Vec[0]) {
					out = append(out, pktRec{
						pkt: pkt, slot: rec.Slot, recStart: b,
						i0cursor: lastSample.BitPos, pos: lastSample.Vec, kind: lastSample.Kind,
					})
					b = end
					continue
				}
				b++
			}
			pkt++
		}
	}
	return out
}

// calibrateLead cherche l'offset de tête (bit ajouté au i0cursor décodé) qui MAXIMISE le
// nombre de (slot, cursor) présents dans l'oracle. Retourne (lead, hits).
func calibrateLead(recs []pktRec, oraByKey map[[2]int][][3]float32) (int, int) {
	bestLead, bestHits := 0, -1
	for lead := -4; lead <= 4; lead++ {
		hits := 0
		for _, r := range recs {
			if _, ok := oraByKey[[2]int{int(r.slot), r.i0cursor + lead}]; ok {
				hits++
			}
		}
		if hits > bestHits {
			bestLead, bestHits = lead, hits
		}
	}
	return bestLead, bestHits
}

// oraPacketCoverage segmente l'oracle en blocs-paquets (frontière = chute du cursor > 400)
// et retourne, par bloc, le nombre de slots distincts.
func oraPacketCoverage(oracle []oraRow) []int {
	var counts []int
	seen := map[int]bool{}
	prev := -1
	for _, r := range oracle {
		if prev >= 0 && r.cursor < prev-400 {
			counts = append(counts, len(seen))
			seen = map[int]bool{}
		}
		seen[r.slot] = true
		prev = r.cursor
	}
	if len(seen) > 0 {
		counts = append(counts, len(seen))
	}
	return counts
}

func report(recs []pktRec, oraByKey map[[2]int][][3]float32, lead int, oraPkt []int, oraSlotCount map[int]int) {
	// couverture par paquet (offline)
	perPkt := map[int]map[uint32]bool{}
	perSlot := map[uint32]int{}
	for _, r := range recs {
		if perPkt[r.pkt] == nil {
			perPkt[r.pkt] = map[uint32]bool{}
		}
		perPkt[r.pkt][r.slot] = true
		perSlot[r.slot]++
	}
	// histogramme couverture (nb slots distincts par paquet)
	covHist := map[int]int{}
	totPkt := 0
	for _, m := range perPkt {
		covHist[len(m)]++
		totPkt++
	}
	fmt.Printf("\n=== COUVERTURE PAR PAQUET (offline, %d paquets avec >=1 biped) ===\n", totPkt)
	keys := sortedInts(covHist)
	for _, k := range keys {
		fmt.Printf("  %2d slots/paquet : %d paquets\n", k, covHist[k])
	}
	oc := map[int]int{}
	for _, c := range oraPkt {
		oc[c]++
	}
	fmt.Printf("  (oracle: %d blocs-paquets ; distribution slots/bloc %v)\n", len(oraPkt), oc)

	// densité par slot (offline vs oracle)
	fmt.Println("\n=== DENSITÉ PAR SLOT (records offline vs rows oracle) ===")
	var slots []int
	for s := range perSlot {
		slots = append(slots, int(s))
	}
	for s := range oraSlotCount {
		if perSlot[uint32(s)] == 0 {
			slots = append(slots, s)
		}
	}
	sort.Ints(slots)
	uniqSlots := uniq(slots)
	for _, s := range uniqSlots {
		fmt.Printf("  slot=%-4d offline=%-6d oracle=%-6d\n", s, perSlot[uint32(s)], oraSlotCount[s])
	}

	// validation cursor+pos
	cursorHits, posHits, tot := 0, 0, len(recs)
	for _, r := range recs {
		poss, ok := oraByKey[[2]int{int(r.slot), r.i0cursor + lead}]
		if !ok {
			continue
		}
		cursorHits++
		for _, p := range poss {
			if abs32(p[0]-r.pos[0]) < 0.2 && abs32(p[1]-r.pos[1]) < 0.2 && abs32(p[2]-r.pos[2]) < 0.2 {
				posHits++
				break
			}
		}
	}
	fmt.Printf("\n=== VALIDATION vs ORACLE (lead=%d) ===\n", lead)
	fmt.Printf("  records offline=%d ; (slot,cursor) dans oracle=%d (%.1f%%) ; pos match <0.2u=%d (%.1f%%)\n",
		tot, cursorHits, 100*float64(cursorHits)/float64(max1(tot)), posHits, 100*float64(posHits)/float64(max1(tot)))
}

func offsetStats(recs []pktRec) {
	hist := map[int]int{}
	for _, r := range recs {
		hist[r.i0cursor-r.recStart]++
	}
	fmt.Println("\n=== OFFSET id->i0 (i0cursor - recStart) ===")
	keys := sortedInts(hist)
	tot := len(recs)
	for _, k := range keys {
		if hist[k]*50 < tot { // n'affiche que les offsets >=2%
			continue
		}
		fmt.Printf("  offset=%-3d : %d records (%.1f%%)\n", k, hist[k], 100*float64(hist[k])/float64(max1(tot)))
	}
	// consistance : % au mode
	mode, mc := 0, -1
	for k, c := range hist {
		if c > mc {
			mode, mc = k, c
		}
	}
	fmt.Printf("  mode=%d (%.1f%% des records)\n", mode, 100*float64(mc)/float64(max1(tot)))
}

// realHitAnalysis se restreint aux records dont (slot,i0cursor+lead) EXISTE dans l'oracle
// (= records réels, pas les faux-positifs du scan). Mesure : offset id->i0 sur ces records,
// distribution des kinds, et match de position pour les kinds absolus (comparaison directe).
func realHitAnalysis(recs []pktRec, oraByKey map[[2]int][][3]float32, lead int) {
	offHist := map[int]int{}
	kindHist := map[filmdec.PosKind]int{}
	absTot, absMatch := 0, 0
	real := 0
	for _, r := range recs {
		poss, ok := oraByKey[[2]int{int(r.slot), r.i0cursor + lead}]
		if !ok {
			continue
		}
		real++
		offHist[r.i0cursor-r.recStart]++
		kindHist[r.kind]++
		if r.kind == filmdec.PosKindAbsolute || r.kind == filmdec.PosKindAbsFallback {
			absTot++
			for _, p := range poss {
				if abs32(p[0]-r.pos[0]) < 0.3 && abs32(p[1]-r.pos[1]) < 0.3 && abs32(p[2]-r.pos[2]) < 0.3 {
					absMatch++
					break
				}
			}
		}
	}
	fmt.Printf("\n=== ANALYSE RESTREINTE AUX RECORDS RÉELS (cursor exact oracle) : n=%d ===\n", real)
	fmt.Println("  offset id->i0 (sur records réels) :")
	mode, mc := 0, -1
	for _, k := range sortedInts(offHist) {
		if offHist[k]*40 < real {
			continue
		}
		fmt.Printf("    offset=%-3d : %d (%.1f%%)\n", k, offHist[k], 100*float64(offHist[k])/float64(max1(real)))
		if offHist[k] > mc {
			mode, mc = k, offHist[k]
		}
	}
	fmt.Printf("    mode=%d (%.1f%% des records réels)\n", mode, 100*float64(mc)/float64(max1(real)))
	fmt.Println("  kinds des records réels :")
	for k, c := range kindHist {
		fmt.Printf("    %-6s : %d (%.1f%%)\n", k.String(), c, 100*float64(c)/float64(max1(real)))
	}
	fmt.Printf("  pos match (kinds absolus, <0.3u) : %d/%d (%.1f%%)\n", absMatch, absTot, 100*float64(absMatch)/float64(max1(absTot)))
}

func deltaStats(recs []pktRec, oracle []oraRow) {
	kindHist := map[filmdec.PosKind]int{}
	for _, r := range recs {
		kindHist[r.kind]++
	}
	fmt.Println("\n=== KIND des samples i0 (offline) ===")
	for k, c := range kindHist {
		fmt.Printf("  %-6s : %d\n", k.String(), c)
	}
	// distribution des pas oracle (diff consécutive par slot, en unités) — dominante attendue 1-4 quanta (~0.0138)
	q := float64(0.01383)
	stepHist := map[int]int{}
	var sumAbs float64
	var nStep int
	bySlot := map[int][]oraRow{}
	for _, r := range oracle {
		bySlot[r.slot] = append(bySlot[r.slot], r)
	}
	for _, rows := range bySlot {
		for i := 1; i < len(rows); i++ {
			for a := 0; a < 3; a++ {
				var d float64
				switch a {
				case 0:
					d = float64(rows[i].x - rows[i-1].x)
				case 1:
					d = float64(rows[i].y - rows[i-1].y)
				case 2:
					d = float64(rows[i].z - rows[i-1].z)
				}
				if d == 0 {
					continue
				}
				nq := int(math.Round(math.Abs(d) / q))
				stepHist[nq]++
				sumAbs += math.Abs(d)
				nStep++
			}
		}
	}
	fmt.Println("\n=== ORACLE : pas consécutifs (en quanta de 0.0138) ===")
	if nStep > 0 {
		fmt.Printf("  moyenne |pas| = %.4f u ; n=%d\n", sumAbs/float64(nStep), nStep)
	}
	for _, k := range sortedInts(stepHist) {
		if k > 8 || stepHist[k]*100 < nStep {
			continue
		}
		fmt.Printf("  %d quanta : %.1f%%\n", k, 100*float64(stepHist[k])/float64(max1(nStep)))
	}
}

// accumulateAndPlot : décode les paquets EN ORDRE avec un World accumulateur, seed sur
// abs, applique les deltas, stocke la trajectoire par slot, puis PNG offline vs oracle.
func accumulateAndPlot(dir string, reg *filmdec.Registry, binds map[uint32]uint32, targets map[uint32]bool, cfg filmdec.FrameConfig, oraSlotRows map[int][]oraRow) {
	w := filmdec.NewWorld(reg)
	for s := range targets {
		ti := binds[s]
		if ti == 0 {
			ti = 35
		}
		w.BindFull((1<<30)|s, ti)
	}
	filmdec.SetPositionAccumulator(w)
	defer filmdec.SetPositionAccumulator(nil)

	traj := map[uint32][][3]float32{}
	var cur filmdec.PositionSample
	var have bool
	filmdec.SetPositionCaptureHook(func(s filmdec.PositionSample) { cur, have = s, true })
	defer filmdec.SetPositionCaptureHook(nil)

	for idx := 3; idx <= 26; idx++ {
		for _, fr := range listFrames(inflate(fmt.Sprintf("%s/chunk_%02d.bin", dir, idx)), 0) {
			frameLen := len(fr) * 8
			b := 0
			for b < frameLen-16 {
				have = false
				// TryDeltaAt re-crée un reader ; l'accumulation se fait via accumWorld pendant decodeDelta.
				rec, end, ok := filmdec.TryDeltaAt(fr, b, w, cfg)
				if ok && targets[rec.Slot] && have && finite(cur.Vec[0]) {
					traj[rec.Slot] = append(traj[rec.Slot], cur.Vec)
					b = end
					continue
				}
				b++
			}
		}
	}

	// 8 slots les plus denses offline
	var scs []slotCount
	for s, t := range traj {
		scs = append(scs, slotCount{s, len(t)})
	}
	sort.Slice(scs, func(i, j int) bool { return scs[i].n > scs[j].n })
	fmt.Println("\n=== TRAJECTOIRES ACCUMULÉES (8 slots offline les plus denses) ===")
	top := scs
	if len(top) > 8 {
		top = top[:8]
	}
	for _, x := range top {
		pts := traj[x.s]
		xr := spanStr(pts, 0)
		yr := spanStr(pts, 1)
		fmt.Printf("  slot=%-4d n=%-6d X:%s Y:%s (oracle rows=%d)\n", x.s, x.n, xr, yr, len(oraSlotRows[int(x.s)]))
	}

	plotPNG(traj, oraSlotRows, top8Slots(top))
}

type slotCount struct {
	s uint32
	n int
}

func top8Slots(top []slotCount) []uint32 {
	var out []uint32
	for _, x := range top {
		out = append(out, x.s)
	}
	return out
}

// plotPNG dessine X/Y offline (couleur pleine) vs oracle (points clairs) pour les slots donnés.
func plotPNG(traj map[uint32][][3]float32, oraSlotRows map[int][]oraRow, slots []uint32) {
	const W, H = 1200, 600
	img := image.NewRGBA(image.Rect(0, 0, W, H))
	for i := range img.Pix {
		img.Pix[i] = 20
	}
	// bornes monde depuis oracle
	minX, maxX, minY, maxY := float32(1e9), float32(-1e9), float32(1e9), float32(-1e9)
	for _, rows := range oraSlotRows {
		for _, r := range rows {
			minX, maxX = minf(minX, r.x), maxf(maxX, r.x)
			minY, maxY = minf(minY, r.y), maxf(maxY, r.y)
		}
	}
	if maxX <= minX {
		maxX = minX + 1
	}
	if maxY <= minY {
		maxY = minY + 1
	}
	sx := func(x float32) int { return int(float32(W-40)*(x-minX)/(maxX-minX)) + 20 }
	sy := func(y float32) int { return H - (int(float32(H-40)*(y-minY)/(maxY-minY)) + 20) }
	pal := []color.RGBA{
		{255, 80, 80, 255}, {80, 200, 255, 255}, {120, 255, 120, 255}, {255, 220, 60, 255},
		{255, 120, 255, 255}, {120, 255, 240, 255}, {255, 160, 60, 255}, {180, 140, 255, 255},
	}
	// oracle en gris clair
	for _, s := range slots {
		for _, r := range oraSlotRows[int(s)] {
			px, py := sx(r.x), sy(r.y)
			set(img, px, py, color.RGBA{90, 90, 90, 255})
		}
	}
	// offline en couleur
	for i, s := range slots {
		c := pal[i%len(pal)]
		for _, p := range traj[s] {
			if !finite(p[0]) {
				continue
			}
			px, py := sx(p[0]), sy(p[1])
			set(img, px, py, c)
		}
	}
	out := scratch + "/parallel_pos.png"
	f, err := os.Create(out)
	if err != nil {
		fmt.Println("PNG err:", err)
		return
	}
	defer f.Close()
	png.Encode(f, img)
	fmt.Printf("\nPNG écrit : %s (offline=couleur, oracle=gris)\n", out)
}

func set(img *image.RGBA, x, y int, c color.RGBA) {
	for dx := -1; dx <= 1; dx++ {
		for dy := -1; dy <= 1; dy++ {
			img.SetRGBA(x+dx, y+dy, c)
		}
	}
}

// ---- utilitaires ----

func spanStr(pts [][3]float32, ax int) string {
	if len(pts) == 0 {
		return "[]"
	}
	mn, mx := pts[0][ax], pts[0][ax]
	for _, p := range pts {
		mn, mx = minf(mn, p[ax]), maxf(mx, p[ax])
	}
	return fmt.Sprintf("[%.1f..%.1f]", mn, mx)
}

func sortedInts(m map[int]int) []int {
	var k []int
	for x := range m {
		k = append(k, x)
	}
	sort.Ints(k)
	return k
}
func uniq(s []int) []int {
	var o []int
	for i, x := range s {
		if i == 0 || x != s[i-1] {
			o = append(o, x)
		}
	}
	return o
}
func abs32(x float32) float32 {
	if x < 0 {
		return -x
	}
	return x
}
func minf(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}
func maxf(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}
func max1(x int) int {
	if x < 1 {
		return 1
	}
	return x
}
