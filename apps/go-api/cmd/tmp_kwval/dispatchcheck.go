package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"sort"
)

// dispatchcheck.go — exploite la capture CE filmdec_dispatch (hook du dispatcher FUN_14080a9d4, 1 rec/event).
// rec 128 : [00]rdtsc(8) [08]pos[R8+0x2c] [0C]bitsAvail [10]acc(8) [18]bytePtr [20..80]fenêtre 96o.
// Localise chaque event dans son paquet (fenêtre) ; R7 = bitsAt(payload, pos, 7) (pos packet-local). Groupe
// par paquet, trie par pos -> SÉQUENCE (pos, R7) des events = structure exacte du paquet -> crack du walker.

type dispRec struct {
	rdtsc                   uint64
	pos, bitsAvail, bytePtr uint32
	acc                     uint64
	window                  []byte
}

func parseDisp(path string) []dispRec {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var out []dispRec
	for o := 0; o+0x80 <= len(b); o += 0x80 {
		out = append(out, dispRec{
			rdtsc:     uint64(binary.LittleEndian.Uint32(b[o:])) | uint64(binary.LittleEndian.Uint32(b[o+4:]))<<32,
			pos:       binary.LittleEndian.Uint32(b[o+8:]),
			bitsAvail: binary.LittleEndian.Uint32(b[o+0xc:]),
			acc:       binary.LittleEndian.Uint64(b[o+0x10:]),
			bytePtr:   binary.LittleEndian.Uint32(b[o+0x18:]),
			window:    b[o+0x20 : o+0x80],
		})
	}
	return out
}

// accR7 : R7 = 7 bits de tête de l'accumulateur (MSB-first). shift calibré vs window (essais 57/50/…).
func accR7(acc uint64, shift uint) int { return int((acc >> shift) & 0x7f) }

// runBurst : groupe les events par RAFALE rdtsc (paquet), et pour chaque rafale montre le 1er event (pos+R7
// via accumulateur, MÊME non localisé) -> révèle le START de boucle du paquet.
func runBurst(m string) {
	ceDir := `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/.claude/worktrees/weapon-attribution-v3/tools/ce/`
	recs := parseDisp(ceDir + m + "_disp_dispatch.bin")
	if len(recs) == 0 {
		fmt.Printf("=== BURST %s : vide ===\n", m)
		return
	}
	sort.Slice(recs, func(i, j int) bool { return recs[i].rdtsc < recs[j].rdtsc })
	// calibre le shift accR7 vs window R7 sur les events localisés
	cache := root + "/" + m
	var chunks [][]byte
	for ch := 0; ch <= 41; ch++ {
		chunks = append(chunks, inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, ch)))
	}
	nCal := len(recs)
	if nCal > 400 {
		nCal = 400
	}
	bestShift, bestHit := uint(57), -1
	for _, sh := range []uint{57, 56, 58, 50, 49, 25, 33} {
		hit := 0
		for _, r := range recs[:nCal] {
			if pl, _, _, ok := locatePkt(chunks, r.window); ok && int(r.pos)+7 <= len(pl)*8 {
				if accR7(r.acc, sh) == int(bitsAt(pl, int(r.pos), 7)) {
					hit++
				}
			}
		}
		if hit > bestHit {
			bestHit, bestShift = hit, sh
		}
	}
	fmt.Printf("=== BURST %s : %d events | shift accR7=%d (hit %d) ===\n", m, len(recs), bestShift, bestHit)
	// rafales : nouvelle rafale si gap rdtsc > seuil
	var thr uint64 = 50000
	firstPos := map[int]int{}
	firstR7 := map[int]int{}
	shown := 0
	i := 0
	for i < len(recs) {
		j := i
		minPos, minR7, minPosVal := -1, -1, 1<<30
		for j < len(recs) && (j == i || recs[j].rdtsc-recs[j-1].rdtsc < thr) {
			p := int(recs[j].pos)
			if p < minPosVal {
				minPosVal, minPos, minR7 = p, p, accR7(recs[j].acc, bestShift)
			}
			j++
		}
		if j-i >= 3 { // rafale = paquet avec plusieurs events
			firstPos[minPos]++
			firstR7[minR7]++
			if shown < 12 {
				fmt.Printf("  rafale %d events : 1er pos=%d R7=%d\n", j-i, minPos, minR7)
				shown++
			}
		}
		i = j
	}
	// distribution du 1er pos + 1er R7
	top := func(h map[int]int, label string) {
		type kv struct{ k, v int }
		var a []kv
		for k, v := range h {
			a = append(a, kv{k, v})
		}
		sort.Slice(a, func(i, j int) bool { return a[i].v > a[j].v })
		fmt.Printf("%s :", label)
		for i := 0; i < 12 && i < len(a); i++ {
			fmt.Printf(" %d×%d", a[i].k, a[i].v)
		}
		fmt.Println()
	}
	top(firstPos, ">>> 1er pos de rafale")
	top(firstR7, ">>> 1er R7 de rafale")
}

// runWalkVal : valide le WALKER offline contre l'oracle dispatcher. Construit la table code->longueur
// (médiane empirique) depuis la capture, puis walke chaque paquet depuis bit 1 et compare la séquence
// (pos,R7) à celle capturée. Montre : combien de paquets walkent jusqu'au kill (85) au bon pos, et sur
// quel event la désync arrive (= l'event variable à modéliser exactement).
func runWalkVal(m string) {
	ceDir := `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/.claude/worktrees/weapon-attribution-v3/tools/ce/`
	recs := parseDisp(ceDir + m + "_disp_dispatch.bin")
	if len(recs) == 0 {
		fmt.Printf("=== WALKVAL %s : vide ===\n", m)
		return
	}
	cache := root + "/" + m
	var chunks [][]byte
	for ch := 0; ch <= 41; ch++ {
		chunks = append(chunks, inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, ch)))
	}
	type ev struct{ pos, r7 int }
	type pk struct{ ch, off int }
	byPkt := map[pk][]ev{}
	pktPl := map[pk][]byte{}
	for _, r := range recs {
		pl, ci, po, ok := locatePkt(chunks, r.window)
		if !ok || int(r.pos)+7 > len(pl)*8 {
			continue
		}
		p := pk{ci, po}
		byPkt[p] = append(byPkt[p], ev{int(r.pos), int(bitsAt(pl, int(r.pos), 7))})
		pktPl[p] = pl
	}
	// table code -> longueur médiane (gap-11)
	gaps := map[int][]int{}
	for _, evs := range byPkt {
		sort.Slice(evs, func(i, j int) bool { return evs[i].pos < evs[j].pos })
		for i := 0; i+1 < len(evs); i++ {
			if g := evs[i+1].pos - evs[i].pos; g > 11 && g < 8000 {
				gaps[evs[i].r7] = append(gaps[evs[i].r7], g-11)
			}
		}
	}
	length := map[int]int{}
	for c, gs := range gaps {
		sort.Ints(gs)
		length[c] = gs[len(gs)/2]
	}
	// walk chaque paquet ayant un kill, depuis bit 1
	nKill, walkReach, desync := 0, 0, 0
	desyncAt := map[int]int{}
	for p, evs := range byPkt {
		sort.Slice(evs, func(i, j int) bool { return evs[i].pos < evs[j].pos })
		capKillPos := -1
		for _, e := range evs {
			if e.r7 == 85 {
				capKillPos = e.pos
				break
			}
		}
		if capKillPos < 0 {
			continue
		}
		nKill++
		pl := pktPl[p]
		r := &br{pl: pl, bp: evs[0].pos - 1} // démarre au cont du 1er event capturé (isole le modèle du header paquet)
		reached := false
		for r.bp+11 <= len(pl)*8 {
			if r.g1() == 0 {
				break
			}
			posR7 := r.bp
			t := int(r.R(7))
			presenceLoop(r) // encadrement variable réel (boucle de présence, FUN_1406d3140 config-quantifié)
			if t == 85 {
				if posR7 == capKillPos {
					reached = true
				}
				break
			}
			if !skipEventDeser(r, t, false) { // déser bit-exact ; sinon désync
				desyncAt[t]++
				break
			}
		}
		if reached {
			walkReach++
		} else {
			desync++
		}
	}
	fmt.Printf("=== WALKVAL %s : %d paquets-kill | walk atteint le kill@bon pos : %d | désync : %d ===\n",
		m, nKill, walkReach, desync)
	_ = length
	fmt.Printf("désync par code (event qui casse le walk) : ")
	type kv struct{ k, v int }
	var a []kv
	for k, v := range desyncAt {
		a = append(a, kv{k, v})
	}
	sort.Slice(a, func(i, j int) bool { return a[i].v > a[j].v })
	for _, x := range a {
		fmt.Printf("%d×%d ", x.k, x.v)
	}
	fmt.Println()
}

// runModelCheck : valide CHAQUE modèle d'event en ISOLATION contre la vérité-terrain du dispatcher.
// Pour chaque paire consécutive (e_i, e_{i+1}), lance le modèle de e_i depuis pos+10 (après R7+3présence)
// et compare le curseur final à e_{i+1}.pos-1 (le bit cont suivant). delta=0 -> modèle exact.
func runModelCheck(m string) {
	ceDir := `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/.claude/worktrees/weapon-attribution-v3/tools/ce/`
	recs := parseDisp(ceDir + m + "_disp_dispatch.bin")
	if len(recs) == 0 {
		fmt.Printf("=== MODELCHECK %s : vide ===\n", m)
		return
	}
	cache := root + "/" + m
	var chunks [][]byte
	for ch := 0; ch <= 41; ch++ {
		chunks = append(chunks, inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, ch)))
	}
	type ev struct{ pos, r7 int }
	type pk struct{ ch, off int }
	byPkt := map[pk][]ev{}
	pktPl := map[pk][]byte{}
	for _, r := range recs {
		pl, ci, po, ok := locatePkt(chunks, r.window)
		if !ok || int(r.pos)+7 > len(pl)*8 {
			continue
		}
		p := pk{ci, po}
		byPkt[p] = append(byPkt[p], ev{int(r.pos), int(bitsAt(pl, int(r.pos), 7))})
		pktPl[p] = pl
	}
	// par code : distribution des deltas (modèle - réel) et des longueurs réelles
	type stat struct {
		n, exact int
		deltas   map[int]int // delta -> count
		realLens []int
	}
	st := map[int]*stat{}
	modeled := map[int]bool{0: true, 1: true, 7: true, 15: true, 36: true, 82: true}
	for c := range stubCodes {
		modeled[c] = true
	}
	for c := range fixedLen {
		modeled[c] = true
	}
	for p, evs := range byPkt {
		sort.Slice(evs, func(i, j int) bool { return evs[i].pos < evs[j].pos })
		pl := pktPl[p]
		for i := 0; i+1 < len(evs); i++ {
			code := evs[i].r7
			if !modeled[code] {
				continue
			}
			realLen := evs[i+1].pos - evs[i].pos - 11 // deser réel (gap - encadrement 11)
			if realLen < 0 || realLen > 8000 {
				continue
			}
			r := &br{pl: pl, bp: evs[i].pos + 10} // début du déser
			skipEventDeser(r, code, false)
			modelLen := r.bp - (evs[i].pos + 10)
			delta := modelLen - realLen
			s := st[code]
			if s == nil {
				s = &stat{deltas: map[int]int{}}
				st[code] = s
			}
			s.n++
			if delta == 0 {
				s.exact++
			}
			s.deltas[delta]++
			s.realLens = append(s.realLens, realLen)
		}
	}
	fmt.Printf("=== MODELCHECK %s : validation par event (delta = modèle - réel, 0 = exact) ===\n", m)
	var codes []int
	for c := range st {
		codes = append(codes, c)
	}
	sort.Ints(codes)
	for _, c := range codes {
		s := st[c]
		sort.Ints(s.realLens)
		rmin, rmax, rmed := s.realLens[0], s.realLens[len(s.realLens)-1], s.realLens[len(s.realLens)/2]
		// top 3 deltas
		type dk struct{ d, n int }
		var ds []dk
		for d, n := range s.deltas {
			ds = append(ds, dk{d, n})
		}
		sort.Slice(ds, func(i, j int) bool { return ds[i].n > ds[j].n })
		top := ""
		for i := 0; i < len(ds) && i < 4; i++ {
			top += fmt.Sprintf(" Δ%+d×%d", ds[i].d, ds[i].n)
		}
		fmt.Printf("code %3d : n=%4d exact=%4d (%3.0f%%) | réel[min=%d méd=%d max=%d] | top:%s\n",
			c, s.n, s.exact, 100*float64(s.exact)/float64(s.n), rmin, rmed, rmax, top)
	}
}

// runLoopStart : caractérise le DÉBUT de la boucle d'events dans chaque paquet-kill (header paquet).
// evs[0].pos = R7 du 1er event capturé ; le loop-start = evs[0].pos - 1 (cont). Cherche un header fixe/trouvable
// et teste la reconstruction par BRUTE-FORCE (essayer chaque bit de départ, retenir celui qui walke jusqu'au kill).
func runLoopStart(m string) {
	ceDir := `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/.claude/worktrees/weapon-attribution-v3/tools/ce/`
	recs := parseDisp(ceDir + m + "_disp_dispatch.bin")
	cache := root + "/" + m
	var chunks [][]byte
	for ch := 0; ch <= 41; ch++ {
		chunks = append(chunks, inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, ch)))
	}
	type ev struct{ pos, r7 int }
	type pk struct{ ch, off int }
	byPkt := map[pk][]ev{}
	pktPl := map[pk][]byte{}
	for _, r := range recs {
		pl, ci, po, ok := locatePkt(chunks, r.window)
		if !ok || int(r.pos)+7 > len(pl)*8 {
			continue
		}
		p := pk{ci, po}
		byPkt[p] = append(byPkt[p], ev{int(r.pos), int(bitsAt(pl, int(r.pos), 7))})
		pktPl[p] = pl
	}
	startHist := map[int]int{}
	bruteOK, bruteTot, ambTot := 0, 0, 0
	for p, evs := range byPkt {
		sort.Slice(evs, func(i, j int) bool { return evs[i].pos < evs[j].pos })
		hasKill := false
		capKillPos := -1
		for _, e := range evs {
			if e.r7 == 85 {
				hasKill = true
				capKillPos = e.pos
				break
			}
		}
		if !hasKill {
			continue
		}
		trueStart := evs[0].pos - 1
		startHist[trueStart]++
		bruteTot++
		// brute-force : essayer chaque bit de départ 0..capKillPos, walker ; un départ est "valide" si le walk
		// atteint un code 85 exactement à capKillPos (vérité). Compte l'ambiguïté (nb de départs valides).
		pl := pktPl[p]
		validStarts := 0
		trueReaches := false
		for s := 0; s < capKillPos && s+11 < len(pl)*8; s++ {
			posR7, code := walkToKill(pl, s, false)
			if code == 85 && posR7 == capKillPos {
				validStarts++
				if s == trueStart {
					trueReaches = true
				}
			}
		}
		if trueReaches {
			bruteOK++
		}
		ambTot += validStarts
	}
	fmt.Printf("=== LOOPSTART %s : %d paquets-kill ===\n", m, bruteTot)
	fmt.Printf("brute-force : le vrai loop-start walke jusqu'au kill (posR7==capKillPos) : %d/%d\n", bruteOK, bruteTot)
	fmt.Printf("ambiguïté moyenne (nb de départs atteignant capKillPos) : %.2f\n", float64(ambTot)/float64(bruteTot))
	type kv struct{ k, v int }
	var a []kv
	for k, v := range startHist {
		a = append(a, kv{k, v})
	}
	sort.Slice(a, func(i, j int) bool { return a[i].v > a[j].v })
	fmt.Printf("distribution loop-start (bit) : ")
	for i, x := range a {
		if i > 15 {
			break
		}
		fmt.Printf("%d×%d ", x.k, x.v)
	}
	fmt.Println()
}

// runCode36Check : pour chaque event code 36 (vérité dispatcher), lit variant_name via code36Variant
// (après R7 + présence) et rapporte la distribution des catégories d'arme. Valide l'extraction déterministe.
func runCode36Check(m string) {
	ceDir := `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/.claude/worktrees/weapon-attribution-v3/tools/ce/`
	recs := parseDisp(ceDir + m + "_disp_dispatch.bin")
	cache := root + "/" + m
	var chunks [][]byte
	for ch := 0; ch <= 41; ch++ {
		chunks = append(chunks, inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, ch)))
	}
	variantHist := map[uint32]int{}
	n := 0
	for _, r := range recs {
		pl, _, _, ok := locatePkt(chunks, r.window)
		if !ok || int(r.pos)+7 > len(pl)*8 {
			continue
		}
		if int(bitsAt(pl, int(r.pos), 7)) != 36 {
			continue
		}
		br0 := &br{pl: pl, bp: int(r.pos) + 7}
		presenceLoop(br0)
		v := code36Variant(br0)
		variantHist[v]++
		n++
	}
	fmt.Printf("=== CODE36CHECK %s : %d events code 36 ===\n", m, n)
	type kv struct {
		v uint32
		n int
	}
	var a []kv
	for v, c := range variantHist {
		a = append(a, kv{v, c})
	}
	sort.Slice(a, func(i, j int) bool { return a[i].n > a[j].n })
	known := map[uint32]string{0x42C9679F: "FIREARM", 0x592CF3: "MELEE?", 0x164B3C: "GRENADE?"}
	for i, x := range a {
		if i > 15 {
			break
		}
		lbl := known[x.v]
		fmt.Printf("  variant 0x%08X : %d %s\n", x.v, x.n, lbl)
	}
}

// runKillScan : teste le LOCATOR par signature. Scanne chaque bit d'un paquet-kill ; à chaque position lit
// R7 ; si 85 -> présence + kill deser ; garde les candidats à killer/victim valides (in [0,16), killer!=victim,
// assist in [-1,16)). Mesure : le vrai capKillPos est-il trouvé ? combien de faux positifs (précision) ?
func runKillScan(m string) {
	ceDir := `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/.claude/worktrees/weapon-attribution-v3/tools/ce/`
	recs := parseDisp(ceDir + m + "_disp_dispatch.bin")
	cache := root + "/" + m
	var chunks [][]byte
	for ch := 0; ch <= 41; ch++ {
		chunks = append(chunks, inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, ch)))
	}
	type ev struct{ pos, r7 int }
	type pk struct{ ch, off int }
	byPkt := map[pk][]ev{}
	pktPl := map[pk][]byte{}
	for _, r := range recs {
		pl, ci, po, ok := locatePkt(chunks, r.window)
		if !ok || int(r.pos)+7 > len(pl)*8 {
			continue
		}
		p := pk{ci, po}
		byPkt[p] = append(byPkt[p], ev{int(r.pos), int(bitsAt(pl, int(r.pos), 7))})
		pktPl[p] = pl
	}
	validKE := func(pl []byte, p int) (killEvent, bool) {
		if p+7 > len(pl)*8 {
			return killEvent{}, false
		}
		if int(bitsAt(pl, p, 7)) != 85 {
			return killEvent{}, false
		}
		r := &br{pl: pl, bp: p + 7}
		presenceLoop(r)
		ke := readKillEvent(r)
		ok := ke.killer >= 0 && ke.killer < 16 && ke.victim >= 0 && ke.victim < 16 &&
			ke.killer != ke.victim && ke.assist >= -1 && ke.assist < 16
		return ke, ok
	}
	nKill, foundTrue, candTot := 0, 0, 0
	for p, evs := range byPkt {
		capKillPos := -1
		for _, e := range evs {
			if e.r7 == 85 {
				capKillPos = e.pos
				break
			}
		}
		if capKillPos < 0 {
			continue
		}
		nKill++
		pl := pktPl[p]
		cands := 0
		for b := 0; b+7 <= len(pl)*8; b++ {
			if _, ok := validKE(pl, b); ok {
				cands++
				if b == capKillPos {
					foundTrue++
				}
			}
		}
		candTot += cands
	}
	fmt.Printf("=== KILLSCAN %s : %d kills ===\n", m, nKill)
	fmt.Printf("signature-scan trouve le vrai capKillPos : %d/%d (rappel)\n", foundTrue, nKill)
	fmt.Printf("candidats moyens par paquet (dont 1 vrai) : %.1f -> précision brute ~%.0f%% avant matching chunk_27\n",
		float64(candTot)/float64(nKill), 100*float64(nKill)/float64(candTot))
}

// runKillDecode : walke chaque paquet-kill jusqu'au code 85 (départ = 1er event capturé) et décode
// killer/victim/assist (E5). Valide que les index sont des slots roster plausibles (0-15).
func runKillDecode(m string) {
	ceDir := `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/.claude/worktrees/weapon-attribution-v3/tools/ce/`
	recs := parseDisp(ceDir + m + "_disp_dispatch.bin")
	cache := root + "/" + m
	var chunks [][]byte
	for ch := 0; ch <= 41; ch++ {
		chunks = append(chunks, inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, ch)))
	}
	type ev struct{ pos, r7 int }
	type pk struct{ ch, off int }
	byPkt := map[pk][]ev{}
	pktPl := map[pk][]byte{}
	for _, r := range recs {
		pl, ci, po, ok := locatePkt(chunks, r.window)
		if !ok || int(r.pos)+7 > len(pl)*8 {
			continue
		}
		p := pk{ci, po}
		byPkt[p] = append(byPkt[p], ev{int(r.pos), int(bitsAt(pl, int(r.pos), 7))})
		pktPl[p] = pl
	}
	nKill, decoded, validKV, directValid := 0, 0, 0, 0
	for p, evs := range byPkt {
		sort.Slice(evs, func(i, j int) bool { return evs[i].pos < evs[j].pos })
		capKillPos := -1
		for _, e := range evs {
			if e.r7 == 85 {
				capKillPos = e.pos
				break
			}
		}
		if capKillPos < 0 {
			continue
		}
		nKill++
		// décode DIRECT à capKillPos (indépendant du walk) : R7(7) + présence + kill deser
		{
			pl := pktPl[p]
			rd := &br{pl: pl, bp: capKillPos}
			rd.R(7) // R7=85
			presenceLoop(rd)
			ke := readKillEvent(rd)
			if ke.killer >= 0 && ke.killer < 16 && ke.victim >= 0 && ke.victim < 16 {
				directValid++
			}
		}
		// walke depuis le 1er event, jusqu'au kill, puis décode
		pl := pktPl[p]
		r := &br{pl: pl, bp: evs[0].pos - 1}
		for r.bp+11 <= len(pl)*8 {
			if r.g1() == 0 {
				break
			}
			posR7 := r.bp
			t := int(r.R(7))
			presenceLoop(r)
			if t == 85 {
				if posR7 == capKillPos {
					ke := readKillEvent(r)
					decoded++
					ok := ke.killer >= 0 && ke.killer < 16 && ke.victim >= 0 && ke.victim < 16
					if ok {
						validKV++
					}
					if decoded <= 12 {
						fmt.Printf("kill@%d : killer=%d victim=%d assist=%d %s\n", posR7, ke.killer, ke.victim, ke.assist,
							map[bool]string{true: "OK", false: "<-- index hors roster"}[ok])
					}
				}
				break
			}
			if !skipEventDeser(r, t, false) {
				break
			}
		}
	}
	fmt.Printf("=== KILLDECODE %s : %d kills | walk+décode atteint : %d | killer&victim in [0,16) : %d ===\n",
		m, nKill, decoded, validKV)
	fmt.Printf("décode DIRECT à capKillPos (indépendant du walk) : %d/%d killer&victim valides\n", directValid, nKill)
}

// runKillPre : pour chaque paquet-kill, séquence des codes (vérité dispatcher) AVANT le premier 85.
// Donne la priorité de modélisation : quels events faut-il absolument savoir sauter pour atteindre le kill.
func runKillPre(m string) {
	ceDir := `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/.claude/worktrees/weapon-attribution-v3/tools/ce/`
	recs := parseDisp(ceDir + m + "_disp_dispatch.bin")
	cache := root + "/" + m
	var chunks [][]byte
	for ch := 0; ch <= 41; ch++ {
		chunks = append(chunks, inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, ch)))
	}
	type ev struct{ pos, r7 int }
	type pk struct{ ch, off int }
	byPkt := map[pk][]ev{}
	for _, r := range recs {
		pl, ci, po, ok := locatePkt(chunks, r.window)
		if !ok || int(r.pos)+7 > len(pl)*8 {
			continue
		}
		p := pk{ci, po}
		byPkt[p] = append(byPkt[p], ev{int(r.pos), int(bitsAt(pl, int(r.pos), 7))})
	}
	preHist := map[int]int{}   // code -> nb d'apparitions avant un kill
	firstHist := map[int]int{} // code du 1er event du paquet-kill
	nKill := 0
	seqCount := map[string]int{}
	for _, evs := range byPkt {
		sort.Slice(evs, func(i, j int) bool { return evs[i].pos < evs[j].pos })
		ki := -1
		for i, e := range evs {
			if e.r7 == 85 {
				ki = i
				break
			}
		}
		if ki < 0 {
			continue
		}
		nKill++
		if ki > 0 {
			firstHist[evs[0].r7]++
		}
		seq := ""
		for i := 0; i < ki; i++ {
			preHist[evs[i].r7]++
			seq += fmt.Sprintf("%d,", evs[i].r7)
		}
		seqCount[seq]++
	}
	fmt.Printf("=== KILLPRE %s : %d paquets-kill ===\n", m, nKill)
	fmt.Printf("codes AVANT le kill (histogramme) : ")
	var cs []int
	for c := range preHist {
		cs = append(cs, c)
	}
	sort.Slice(cs, func(i, j int) bool { return preHist[cs[i]] > preHist[cs[j]] })
	for _, c := range cs {
		fmt.Printf("%d×%d ", c, preHist[c])
	}
	fmt.Println()
	type sc struct {
		s string
		n int
	}
	var seqs []sc
	for s, n := range seqCount {
		seqs = append(seqs, sc{s, n})
	}
	sort.Slice(seqs, func(i, j int) bool { return seqs[i].n > seqs[j].n })
	fmt.Println("séquences pré-kill les plus fréquentes :")
	for i := 0; i < len(seqs) && i < 12; i++ {
		fmt.Printf("  %2d× [%s]\n", seqs[i].n, seqs[i].s)
	}
}

// runPresenceSweep : teste l'hypothèse que le +16 des events non-exacts vient de la BOUCLE DE PRÉSENCE
// du dispatcher (3× R(1) ; si set: R(K) champ). Modèle plein event = presenceLoop(K) + deser, démarré à
// pos+7 (après R7), doit atterrir à next_pos-1. Sweep K -> trouve la largeur qui rend code 1 exact.
func runPresenceSweep(m string) {
	ceDir := `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/.claude/worktrees/weapon-attribution-v3/tools/ce/`
	recs := parseDisp(ceDir + m + "_disp_dispatch.bin")
	cache := root + "/" + m
	var chunks [][]byte
	for ch := 0; ch <= 41; ch++ {
		chunks = append(chunks, inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, ch)))
	}
	type ev struct{ pos, r7 int }
	type pk struct{ ch, off int }
	byPkt := map[pk][]ev{}
	pktPl := map[pk][]byte{}
	for _, r := range recs {
		pl, ci, po, ok := locatePkt(chunks, r.window)
		if !ok || int(r.pos)+7 > len(pl)*8 {
			continue
		}
		p := pk{ci, po}
		byPkt[p] = append(byPkt[p], ev{int(r.pos), int(bitsAt(pl, int(r.pos), 7))})
		pktPl[p] = pl
	}
	// collecte, par code, la liste (pos, next_pos, pl) des events consécutifs modélisables
	type pair struct {
		pl        []byte
		pos, next int
	}
	byCode := map[int][]pair{}
	for p, evs := range byPkt {
		sort.Slice(evs, func(i, j int) bool { return evs[i].pos < evs[j].pos })
		pl := pktPl[p]
		for i := 0; i+1 < len(evs); i++ {
			byCode[evs[i].r7] = append(byCode[evs[i].r7], pair{pl, evs[i].pos, evs[i+1].pos})
		}
	}
	fmt.Printf("=== PRESENCESWEEP %s : full event = presenceLoop(K) + deser, start pos+7 -> next_pos-1 ===\n", m)
	for _, code := range []int{1, 0, 7, 82} {
		prs := byCode[code]
		if len(prs) == 0 {
			continue
		}
		fmt.Printf("code %d (n=%d) : ", code, len(prs))
		best, bestK := -1, -1
		for _, K := range []int{0, 2, 9, 11, 13, 15, 16, 17, 21} {
			exact := 0
			for _, pr := range prs {
				r := &br{pl: pr.pl, bp: pr.pos + 7} // après R7
				for i := 0; i < 3; i++ {            // boucle de présence
					if r.g1() == 1 {
						r.R(K)
					}
				}
				if !skipModeled(r, code) {
					continue
				}
				if r.bp == pr.next-1 { // atterri sur le cont suivant
					exact++
				}
			}
			pct := 100 * exact / len(prs)
			fmt.Printf("K%d=%d%% ", K, pct)
			if exact > best {
				best, bestK = exact, K
			}
		}
		fmt.Printf("| best K=%d (%d/%d)\n", bestK, best, len(prs))
	}
	// Modèle FUN_1406d3140 exact : présenceField = R(1)flag ; si set R(9)[0x200] sinon R(13)[0x1FFF] ; R(2).
	fmt.Println("--- modèle FUN_1406d3140 (flag+9/13+2) par slot de présence ---")
	pf := func(r *br) {
		if r.g1() == 1 {
			r.R(9)
		} else {
			r.R(13)
		}
		r.R(2)
	}
	for _, code := range []int{1, 0, 7, 82} {
		prs := byCode[code]
		if len(prs) == 0 {
			continue
		}
		exact := 0
		for _, pr := range prs {
			r := &br{pl: pr.pl, bp: pr.pos + 7}
			for i := 0; i < 3; i++ {
				if r.g1() == 1 {
					pf(r)
				}
			}
			if !skipModeled(r, code) {
				continue
			}
			if r.bp == pr.next-1 {
				exact++
			}
		}
		fmt.Printf("code %d : %d/%d (%d%%)\n", code, exact, len(prs), 100*exact/len(prs))
	}
	// code 15 : teste les deux états du gate config has15 (pas un bit du flux) pour déterminer le film
	if prs := byCode[15]; len(prs) > 0 {
		for _, g := range []bool{false, true} {
			exact := 0
			for _, pr := range prs {
				r := &br{pl: pr.pl, bp: pr.pos + 7}
				for i := 0; i < 3; i++ {
					if r.g1() == 1 {
						pf(r)
					}
				}
				skipEvent15(r, g)
				if r.bp == pr.next-1 {
					exact++
				}
			}
			fmt.Printf("code 15 (has15=%v) : %d/%d (%d%%)\n", g, exact, len(prs), 100*exact/len(prs))
		}
	}
}

// skipModeled : avance r du déser de `code` ; renvoie false si non modélisé/échec variant.
func skipModeled(r *br, code int) bool {
	switch {
	case stubCodes[code]:
	case fixedLen[code] != 0:
		r.bp += fixedLen[code]
	case code == 0:
		walkEvent0(r)
	case code == 1:
		skipEvent1(r)
	case code == 7:
		return skipEvent7(r)
	case code == 15:
		skipEvent15(r, has15)
	case code == 82:
		return skipEvent82(r)
	default:
		return false
	}
	return true
}

func runDispatchCheck(m string) {
	ceDir := `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/.claude/worktrees/weapon-attribution-v3/tools/ce/`
	recs := parseDisp(ceDir + m + "_disp_dispatch.bin")
	if len(recs) == 0 {
		fmt.Printf("=== DISPATCHCHECK %s : dispatch vide ===\n", m)
		return
	}
	cache := root + "/" + m
	var chunks [][]byte
	for ch := 0; ch <= 41; ch++ {
		chunks = append(chunks, inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, ch)))
	}
	fmt.Printf("=== DISPATCHCHECK %s : %d events dispatchés ===\n", m, len(recs))

	// groupe par paquet : (chunk, pktOff) -> liste (pos, R7)
	type ev struct{ pos, r7 int }
	type pk struct{ ch, off int }
	byPkt := map[pk][]ev{}
	pktMk := map[pk]byte{}
	loc := 0
	for _, r := range recs {
		pl, ci, po, ok := locatePkt(chunks, r.window)
		if !ok {
			continue
		}
		loc++
		p := pk{ci, po}
		r7 := -1
		if int(r.pos)+7 <= len(pl)*8 {
			r7 = int(bitsAt(pl, int(r.pos), 7))
		}
		byPkt[p] = append(byPkt[p], ev{int(r.pos), r7})
		pktMk[p] = pl[0]
	}
	fmt.Printf("events localisés : %d/%d | paquets distincts : %d\n", loc, len(recs), len(byPkt))

	// distribution du 1er R7 par marker + séquences ayant un kill (R7=85)
	firstByMk := map[byte]map[int]int{}
	var withKill []string
	shownK := 0
	for p, evs := range byPkt {
		sort.Slice(evs, func(i, j int) bool { return evs[i].pos < evs[j].pos })
		mk := pktMk[p]
		if firstByMk[mk] == nil {
			firstByMk[mk] = map[int]int{}
		}
		firstByMk[mk][evs[0].r7]++
		hasKill := false
		for _, e := range evs {
			if e.r7 == 85 {
				hasKill = true
			}
		}
		if hasKill && shownK < 14 {
			s := fmt.Sprintf("mk=0x%02X:", mk)
			for _, e := range evs {
				kk := ""
				if e.r7 == 85 {
					kk = "=KILL"
				}
				s += fmt.Sprintf(" [%d@%d%s]", e.r7, e.pos, kk)
			}
			withKill = append(withKill, s)
			shownK++
		}
	}
	fmt.Println(">>> 1er event (R7) par marker de paquet :")
	for mk, h := range firstByMk {
		s := fmt.Sprintf("  0x%02X :", mk)
		type kv struct{ k, v int }
		var a []kv
		for k, v := range h {
			a = append(a, kv{k, v})
		}
		sort.Slice(a, func(i, j int) bool { return a[i].v > a[j].v })
		for i := 0; i < 6 && i < len(a); i++ {
			s += fmt.Sprintf(" code%d×%d", a[i].k, a[i].v)
		}
		fmt.Println(s)
	}
	fmt.Println(">>> séquences de paquets contenant un KILL (R7=85) :")
	for _, s := range withKill {
		fmt.Println(s)
	}

	// TABLE code -> longueur empirique (gap au prochain event − 11 encadrement) : la table de skip du walker.
	gaps := map[int][]int{}
	for _, evs := range byPkt {
		sort.Slice(evs, func(i, j int) bool { return evs[i].pos < evs[j].pos })
		for i := 0; i+1 < len(evs); i++ {
			g := evs[i+1].pos - evs[i].pos
			if g > 0 && g < 8000 {
				gaps[evs[i].r7] = append(gaps[evs[i].r7], g-11) // longueur du déser
			}
		}
	}
	type cl struct {
		code, n, med, min, max int
	}
	var cls []cl
	for c, gs := range gaps {
		sort.Ints(gs)
		cls = append(cls, cl{c, len(gs), gs[len(gs)/2], gs[0], gs[len(gs)-1]})
	}
	sort.Slice(cls, func(i, j int) bool { return cls[i].n > cls[j].n })
	fmt.Println(">>> longueur déser par code (empirique, n=échantillons) : code n méd[min..max]")
	for i, c := range cls {
		if i >= 20 {
			break
		}
		fmt.Printf("  code %-3d n=%-4d méd=%-5d [%d..%d]%s\n", c.code, c.n, c.med, c.min, c.max,
			func() string {
				if c.min == c.max {
					return "  FIXE"
				}
				return ""
			}())
	}
}
