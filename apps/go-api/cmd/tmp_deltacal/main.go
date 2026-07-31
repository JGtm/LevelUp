// tmp_deltacal — SONDE (weapon-attribution-v3) : calibration du SPINE DELTA biped.
//
// But : rendre le décodeur de DELTAS bit-exact AVANT i11 (object-dead-state), le seul
// blocage pour lire la cause de mort enregistrée par le kill feed. Deux modes :
//
//	tmp_deltacal replay [maxChunk]
//	    Rejoue les FRAME deltas (World capturé) et rapporte, pour les bipeds :
//	    - nb deltas, nb clean (DesyncAt==-1), nb atteignant object-dead-state-component
//	    - histogramme du 1er composant qui DÉSYNC (où ça casse)
//	    - pour les premiers deltas biped : la largeur Go de CHAQUE composant (StartBit
//	      diffs) = à comparer à la vérité-terrain Cheat Engine.
//
//	tmp_deltacal ingest <capture.csv>
//	    Ingère le CSV produit par tools/ce/filmdec_delta_capture.lua, segmente en
//	    records (compIndex décroissant = nouveau record), et imprime par compIndex :
//	    les valeurs param_4 distinctes + l'histogramme de largeur (bitCursor diffs).
//	    => la largeur RÉELLE de i0/i10/i11... en mode delta, runtime.
//
// La comparaison replay(Go) vs ingest(CE) localise au bit près le composant à corriger.
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

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/filmdec"
)

// knownHigh32 : une arme du catalogue dont high-32 == v (discriminant famille).
func knownHigh32(v uint32) (string, bool) {
	for id, n := range analysis.WeaponIDToName {
		if uint32(id>>32) == v {
			return n, true
		}
	}
	return "", false
}

const cache = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`

var calCfg = filmdec.FrameConfig{HasExtraFields: false, IDLowBits: 11}

var bipedSlots = map[uint32]bool{512: true, 513: true, 514: true, 515: true, 516: true, 517: true, 518: true, 519: true}

const bipedTypeIndex = 35

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

type packet struct {
	payload []byte
}

func listFrames(d []byte) []packet {
	var out []packet
	off := 0
	for off+16 <= len(d) {
		typ := binary.LittleEndian.Uint16(d[off:])
		sz := int(binary.LittleEndian.Uint32(d[off+4:]))
		if sz < 0 || off+16+sz > len(d) {
			break
		}
		if typ == 0 {
			out = append(out, packet{d[off+16 : off+16+sz]})
		}
		off += 16 + sz
	}
	return out
}

func loadWorld(reg *filmdec.Registry) *filmdec.World {
	raw, _ := os.ReadFile(cache + "/world_dump.txt")
	w := filmdec.NewWorld(reg)
	for _, tok := range bytes.Fields(raw) {
		s := string(tok)
		if len(s) == 0 || s[0] == '#' {
			continue
		}
		var slot, ti uint32
		if _, e := fmt.Sscanf(s, "%d:%d", &slot, &ti); e == nil {
			w.BindFull(slot, ti)
		}
	}
	return w
}

// calib = largeurs CE constantes confirmées (composants à largeur runtime AVANT i11).
// Étendre au fil du diff : ajouter un composant SEULEMENT si sa largeur CE est stable
// (même valeur sur n>1 samples). Le dead-state i11 n'y est jamais.
var calib = map[string]int{
	// i0 N'EST PLUS calibré : le deser FUN_1406cfe44 (branche predFlag==1) est porté -> il
	// produit 47 (mouvement) ET 101 (ragdoll bUsePred==1) tout seul.
	"object-forward-and-up-component":         9,   // i2 (n=105, stable)
	"object-angular-velocity-component":       1,   // i3 (n=10, stable)
	"object-shield-vitality-component":        29,  // i5 (n=2841, stable)
	"object-region-state-component":           358, // i6 (n=77, stable)
	"object-multiplayer-properties-component": 334, // i9 (n=107, stable)
	"object-dissolver-component":              4,   // i14 (n=107, stable)
	"unit-desired-aiming-vector-component":    25,  // i21 (n=4604, stable)
	"unit-grenade-counts-component":           35,  // i22 (n=141, stable)
	"unit-malleable-property-component":       19,  // i23 (n=110, stable)
	"unit-command-tick-component":             10,  // i25 (n=1938, stable)
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: tmp_deltacal replay [maxChunk] | ingest <capture.csv>")
		return
	}
	// CALIBRATION itérative (largeurs CE constantes, filmdec_delta.csv). On ajoute un
	// composant ici dès que le diff confirme une largeur CE stable (même valeur sur n>1).
	// Le dead-state (i11) n'est JAMAIS calibré (il doit être décodé).
	filmdec.PositionCalibratedSkip = true // i0 : saut intelligent 47/101 (cohérent avec les autres sondes)
	for name, w := range calib {
		filmdec.SetCalibratedWidth(name, w)
	}
	switch os.Args[1] {
	case "replay":
		replay()
	case "ingest":
		if len(os.Args) < 3 {
			fmt.Println("usage: tmp_deltacal ingest <capture.csv>")
			return
		}
		ingest(os.Args[2])
	default:
		fmt.Println("mode inconnu:", os.Args[1])
	}
}

func replay() {
	maxChunk := 26
	if len(os.Args) >= 3 {
		fmt.Sscanf(os.Args[2], "%d", &maxChunk)
	}
	filmdec.SetRecordStateParam(2)
	reg, err := filmdec.ParseRegistryChunk(inflate(cache + "/chunk_00.bin"))
	if err != nil {
		panic(err)
	}
	arch, _ := reg.Archetype(bipedTypeIndex)
	fmt.Printf("=== Archétype biped #%d : %d composants ===\n", bipedTypeIndex, len(arch.Components))
	for i := 0; i <= 12 && i < len(arch.Components); i++ {
		fmt.Printf("  i%-2d %s\n", i, arch.Components[i])
	}

	bipedTotal, clean, reachDead := 0, 0, 0
	desyncHist := map[string]int{}
	var sampleDumped int
	frameClean, frameErr := 0, 0
	wstSeen, wstHit := 0, 0      // arme (i43+) sur deltas biped clean : match catalogue ?
	wstVocab := map[uint32]int{} // vocabulaire des variantes WST rencontrées
	i0PathSeen := map[string]int{}
	i0PathHit := map[string]int{}
	for idx := 2; idx <= maxChunk; idx++ {
		data := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, idx))
		for _, fr := range listFrames(data) {
			w := loadWorld(reg)
			br := filmdec.NewBitReader(fr.payload)
			recs, err := filmdec.DecodeFrameRecords(br, w, calCfg)
			// terminaison propre = loop finie sur type==0 (err==nil) ET curseur proche de la fin
			if err == nil && br.BitPos() >= len(fr.payload)*8-16 {
				frameClean++
			} else {
				frameErr++
			}
			for _, r := range recs {
				if !bipedSlots[r.Slot] {
					continue
				}
				bipedTotal++
				if r.DesyncAt == -1 {
					clean++
					// validation SPINE : sur un delta clean portant l'arme (i43+, APRES i11),
					// la variante WST doit matcher le catalogue si le spine est bit-exact.
					if r.Trace.HeldWeapon != 0xFFFFFFFF {
						wstSeen++
						wstVocab[r.Trace.HeldWeapon]++
						hit := false
						if _, ok := knownHigh32(r.Trace.HeldWeapon); ok {
							wstHit++
							hit = true
						}
						// corrèle le CHEMIN i0 (lu directement des 2 bits a son StartBit)
						// avec le match WST -> isole si i0-predicted est le bug.
						if path := i0PathOf(r, fr.payload); path != "" {
							i0PathSeen[path]++
							if hit {
								i0PathHit[path]++
							}
						}
					}
				} else if a, ok := reg.Archetype(int(r.TypeIndex)); ok && r.DesyncAt < len(a.Components) {
					desyncHist[fmt.Sprintf("i%d %s", r.DesyncAt, a.Components[r.DesyncAt])]++
				}
				for _, c := range r.Trace.Comps {
					if c.Name == "object-dead-state-component" {
						reachDead++
					}
				}
				// Dump la largeur Go par composant pour les premiers deltas biped "riches"
				// (mask avec i0..i11) : c'est ce qu'on compare à la capture CE.
				if sampleDumped < 8 && len(r.Trace.Comps) >= 3 && (r.Trace.Mask&((uint64(1)<<11)-1)) != 0 {
					dumpGoWidths(r, fr.payload)
					sampleDumped++
				}
			}
		}
	}
	fmt.Printf("\n=== REPLAY chunks 02..%d : bipeds=%d clean(DesyncAt==-1)=%d atteignant dead-state=%d ===\n",
		maxChunk, bipedTotal, clean, reachDead)
	fmt.Printf("    frames terminant proprement (loop->type0 + curseur~fin) : %d / %d\n", frameClean, frameClean+frameErr)
	fmt.Printf("    ARME (WST i43+) sur deltas clean : vue=%d, matche catalogue=%d (%.0f%%), variantes distinctes=%d\n",
		wstSeen, wstHit, 100*ratioF(wstHit, wstSeen), len(wstVocab))
	fmt.Println("    match WST par CHEMIN i0 (isole le bug i0-predicted) :")
	for _, p := range []string{"keep", "absolute", "predicted", "noI0"} {
		if i0PathSeen[p] > 0 {
			fmt.Printf("      i0=%-9s : vue=%-5d match=%-4d (%.0f%%)\n", p, i0PathSeen[p], i0PathHit[p], 100*ratioF(i0PathHit[p], i0PathSeen[p]))
		}
	}
	// top variantes WST (une variante = un type d'arme ; doit etre un petit vocabulaire)
	{
		type kv struct {
			k uint32
			v int
		}
		var a []kv
		for k, v := range wstVocab {
			a = append(a, kv{k, v})
		}
		sort.Slice(a, func(i, j int) bool { return a[i].v > a[j].v })
		fmt.Println("    top variantes WST :")
		for i := 0; i < len(a) && i < 12; i++ {
			tag := ""
			if n, ok := knownHigh32(a[i].k); ok {
				tag = " = " + n
			}
			fmt.Printf("      0x%08x : %d%s\n", a[i].k, a[i].v, tag)
		}
	}
	type kv struct {
		k string
		v int
	}
	var arr []kv
	for k, v := range desyncHist {
		arr = append(arr, kv{k, v})
	}
	sort.Slice(arr, func(a, b int) bool { return arr[a].v > arr[b].v })
	fmt.Println("1er composant qui désync sur un biped (top 20) :")
	for i := 0; i < len(arr) && i < 20; i++ {
		fmt.Printf("  %-50s : %d\n", arr[i].k, arr[i].v)
	}
}

// bitMSB lit le bit a la position pos (MSB-first, comme le BitReader du jeu).
func bitMSB(payload []byte, pos int) int {
	if pos < 0 || pos/8 >= len(payload) {
		return -1
	}
	return int((payload[pos/8] >> (7 - uint(pos%8))) & 1)
}

// i0PathOf classe le chemin pris par i0 (object-position) en lisant ses 2 bits d'entete
// (bUsePred, bDelta) directement a son StartBit. "" si i0 absent du record.
func i0PathOf(r filmdec.FrameRecord, payload []byte) string {
	start := -1
	for _, c := range r.Trace.Comps {
		if c.Index == 0 && c.Name == "object-position-dynamic-precision-component" {
			start = c.StartBit
			break
		}
	}
	if start < 0 {
		return "noI0"
	}
	b0 := bitMSB(payload, start)
	b1 := bitMSB(payload, start+1)
	switch {
	case b0 == 1:
		return "keep"
	case b0 == 0 && b1 == 0:
		return "absolute"
	case b0 == 0 && b1 == 1:
		return "predicted"
	}
	return ""
}

func ratioF(a, b int) float64 {
	if b == 0 {
		return 0
	}
	return float64(a) / float64(b)
}

// dumpGoWidths imprime, pour un record biped, la largeur Go de chaque composant
// présent (StartBit du suivant - StartBit), + param recordStateParam courant.
func dumpGoWidths(r filmdec.FrameRecord, payload []byte) {
	fmt.Printf("\n-- biped slot=%d typeIdx=%d mask=0x%x DesyncAt=%d EndBit=%d --\n",
		r.Slot, r.TypeIndex, r.Trace.Mask, r.DesyncAt, r.Trace.EndBit)
	cs := r.Trace.Comps
	for i, c := range cs {
		end := r.Trace.EndBit
		if i+1 < len(cs) {
			end = cs[i+1].StartBit
		}
		w := end - c.StartBit
		fmt.Printf("   i%-2d %-46s start=%-6d width=%-4d ported=%v\n", c.Index, c.Name, c.StartBit, w, c.Ported)
	}
}

// ---- ingest : CSV de capture Cheat Engine ----

type capRow struct {
	eid, typeIndex, compIndex, param4, bitCursor, skipCount int
}

func ingest(path string) {
	reg, err := filmdec.ParseRegistryChunk(inflate(cache + "/chunk_00.bin"))
	if err != nil {
		panic(err)
	}
	arch, _ := reg.Archetype(bipedTypeIndex)

	f, err := os.Open(path)
	if err != nil {
		fmt.Println("open:", err)
		return
	}
	defer f.Close()
	var rows []capRow
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	first := true
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		if first {
			first = false
			if strings.HasPrefix(line, "eid") {
				continue
			}
		}
		p := strings.Split(line, ",")
		if len(p) < 6 {
			continue
		}
		ai := func(s string) int { n, _ := strconv.Atoi(strings.TrimSpace(s)); return n }
		rows = append(rows, capRow{ai(p[0]), ai(p[1]), ai(p[2]), ai(p[3]), ai(p[4]), ai(p[5])})
	}
	fmt.Printf("=== CSV CE : %d lignes ===\n", len(rows))

	// Histogramme typeIndex : la capture ne filtre plus -> on voit toutes les archetypes.
	// Le biped = l'archetype du registre qui porte weapon-state-type-info (et qu'on attend
	// a typeIndex 35). On l'affiche pour confirmer la numerotation [rsi+0x30].
	tiHist := map[int]int{}
	for _, r := range rows {
		tiHist[r.typeIndex]++
	}
	{
		var tis []int
		for ti := range tiHist {
			tis = append(tis, ti)
		}
		sort.Slice(tis, func(i, j int) bool { return tiHist[tis[i]] > tiHist[tis[j]] })
		fmt.Println("=== histogramme typeIndex (top 15 ; *=archetype porte weapon-state-type-info) ===")
		for i, ti := range tis {
			if i >= 15 {
				break
			}
			mark, ncomp := "", 0
			if a, ok := reg.Archetype(ti); ok {
				ncomp = len(a.Components)
				for _, c := range a.Components {
					if c == "weapon-state-type-info" {
						mark = " *BIPED?"
						break
					}
				}
			}
			fmt.Printf("  typeIndex=%-4d lignes=%-7d (%d comps)%s\n", ti, tiHist[ti], ncomp, mark)
		}
	}
	// Filtre sur le biped (typeIndex 35) pour l'analyse de largeurs.
	{
		var keep []capRow
		for _, r := range rows {
			if r.typeIndex == bipedTypeIndex {
				keep = append(keep, r)
			}
		}
		if len(keep) == 0 {
			fmt.Printf("!!! aucune ligne typeIndex==%d (biped) — vois l'histogramme ci-dessus pour la vraie valeur.\n", bipedTypeIndex)
			return
		}
		fmt.Printf("=== %d lignes biped (typeIndex==%d) retenues ===\n", len(keep), bipedTypeIndex)
		rows = keep
	}

	// Segmente en records : nouveau record quand compIndex <= précédent (le compteur de
	// boucle redémarre au 1er composant présent du record suivant).
	type record struct {
		rows []capRow
	}
	var records []record
	var cur []capRow
	prevIdx := -1
	for _, r := range rows {
		if r.compIndex <= prevIdx && len(cur) > 0 {
			records = append(records, record{cur})
			cur = nil
		}
		cur = append(cur, r)
		prevIdx = r.compIndex
	}
	if len(cur) > 0 {
		records = append(records, record{cur})
	}
	fmt.Printf("=== %d records biped segmentés ===\n", len(records))

	// Par compIndex : param_4 distincts + histogramme de largeur (cursor diff au suivant
	// DANS le même record). On distingue keyframe (indices contigus depuis 0) vs delta.
	deltaStat := map[int]*compStat{}
	keyframeStat := map[int]*compStat{}
	get := func(m map[int]*compStat, i int) *compStat {
		if m[i] == nil {
			m[i] = &compStat{widths: map[int]int{}, params: map[int]int{}}
		}
		return m[i]
	}
	deltaRecs, kfRecs := 0, 0
	for _, rec := range records {
		isKF := isContiguousFrom0(rec.rows)
		if isKF {
			kfRecs++
		} else {
			deltaRecs++
		}
		for i := 0; i < len(rec.rows); i++ {
			ci := rec.rows[i].compIndex
			var width int = -1
			if i+1 < len(rec.rows) {
				width = rec.rows[i+1].bitCursor - rec.rows[i].bitCursor
			}
			tgt := deltaStat
			if isKF {
				tgt = keyframeStat
			}
			s := get(tgt, ci)
			s.count++
			s.params[rec.rows[i].param4]++
			if width >= 0 {
				s.widths[width]++
			}
		}
	}
	fmt.Printf("    records keyframe (contigus depuis i0)=%d ; delta (sparse)=%d\n", kfRecs, deltaRecs)

	fmt.Println("\n=== DELTA : largeur + param_4 par composant (i0..i20) ===")
	printStat(deltaStat, arch)
	fmt.Println("\n=== KEYFRAME (référence) : largeur + param_4 par composant (i0..i20) ===")
	printStat(keyframeStat, arch)

	// --- DIFF Go vs CE : le 1er composant où la largeur diffère = le bug du decodeur ---
	goW := collectGoDeltaWidths(reg)
	fmt.Println("\n=== DIFF par composant : Go(replay) vs CE(capture) — 1er écart = bug ===")
	fmt.Printf("  %-3s %-44s %-14s %-14s %s\n", "idx", "composant", "CE width(mode)", "Go width(mode)", "verdict")
	firstDiff := -1
	for i := 0; i <= 48; i++ {
		ce := deltaStat[i]
		gw := goW[i]
		if ce == nil && gw == nil {
			continue
		}
		name := ""
		if i < len(arch.Components) {
			name = arch.Components[i]
		}
		ceMode, ceN := modeOf(ceWidths(ce))
		goMode, goN := modeOf(gw)
		verdict := ""
		if ceN > 0 && goN > 0 {
			if ceMode == goMode {
				verdict = "ok"
			} else {
				verdict = "DIFF <<<"
				if firstDiff < 0 {
					firstDiff = i
				}
			}
		}
		fmt.Printf("  i%-2d %-44s %-14s %-14s %s\n", i, name,
			fmt.Sprintf("%d (n=%d)", ceMode, ceN), fmt.Sprintf("%d (n=%d)", goMode, goN), verdict)
	}
	if firstDiff >= 0 {
		nm := ""
		if firstDiff < len(arch.Components) {
			nm = arch.Components[firstDiff]
		}
		fmt.Printf("\n  >>> PREMIER ÉCART à i%d (%s) — c'est le deser delta à corriger.\n", firstDiff, nm)
	} else {
		fmt.Println("\n  >>> aucun écart de largeur détecté i0..i48 — spine delta bit-exact !")
	}
}

func ceWidths(s *compStat) map[int]int {
	if s == nil {
		return nil
	}
	return s.widths
}

func modeOf(m map[int]int) (mode, n int) {
	best := -1
	for k, v := range m {
		n += v
		if v > best {
			best = v
			mode = k
		}
	}
	return mode, n
}

// collectGoDeltaWidths rejoue les FRAME deltas et agrège, par index de composant, la
// largeur Go (StartBit diff) sur les records biped. Avant le 1er bug, ces largeurs sont
// fiables ; après, c'est du bruit — d'où le diff "1er écart".
func collectGoDeltaWidths(reg *filmdec.Registry) map[int]map[int]int {
	filmdec.SetRecordStateParam(2)
	out := map[int]map[int]int{}
	add := func(idx, w int) {
		if out[idx] == nil {
			out[idx] = map[int]int{}
		}
		out[idx][w]++
	}
	for idx := 2; idx <= 26; idx++ {
		data := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, idx))
		for _, fr := range listFrames(data) {
			w := loadWorld(reg)
			br := filmdec.NewBitReader(fr.payload)
			recs, _ := filmdec.DecodeFrameRecords(br, w, calCfg)
			for _, r := range recs {
				if !bipedSlots[r.Slot] {
					continue
				}
				cs := r.Trace.Comps
				for i := 0; i+1 < len(cs); i++ {
					add(cs[i].Index, cs[i+1].StartBit-cs[i].StartBit)
				}
			}
		}
	}
	return out
}

func isContiguousFrom0(rows []capRow) bool {
	if len(rows) == 0 || rows[0].compIndex != 0 {
		return false
	}
	// keyframe = NEW = tous présents : indices 0,1,2,... sans gros trou
	gaps := 0
	for i := 1; i < len(rows); i++ {
		if rows[i].compIndex != rows[i-1].compIndex+1 {
			gaps++
		}
	}
	return gaps <= 2 // tolère qq composants config-skippés
}

// compStat agrège, pour un index de composant, l'histogramme de largeur (bits) et les
// valeurs param_4 observées en capture CE.
type compStat struct {
	widths map[int]int
	params map[int]int
	count  int
}

func printStat(m map[int]*compStat, arch filmdec.Archetype) {
	var idxs []int
	for i := range m {
		idxs = append(idxs, i)
	}
	sort.Ints(idxs)
	for _, i := range idxs {
		if i > 20 {
			continue
		}
		s := m[i]
		name := ""
		if i < len(arch.Components) {
			name = arch.Components[i]
		}
		fmt.Printf("  i%-2d %-44s n=%-4d width=%s param4=%s\n",
			i, name, s.count, topHist(s.widths), topHist(s.params))
	}
}

// topHist formate un histogramme map[valeur]count en "v×c, v×c, ..." trié par count.
func topHist(m map[int]int) string {
	type kv struct{ k, v int }
	var a []kv
	for k, v := range m {
		a = append(a, kv{k, v})
	}
	sort.Slice(a, func(i, j int) bool { return a[i].v > a[j].v })
	var sb strings.Builder
	for i := 0; i < len(a) && i < 5; i++ {
		if i > 0 {
			sb.WriteString(", ")
		}
		fmt.Fprintf(&sb, "%d×%d", a[i].k, a[i].v)
	}
	return sb.String()
}
