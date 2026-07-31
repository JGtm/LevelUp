// tmp_kwval — VALIDATION arme-par-kill same-clock vs vérité-terrain CE, MÊME MATCH.
//
// Le user a raison (2026-07-05) : la validation n'a de sens que si la capture live vient
// du MÊME match que les fichiers décodés. Le warp (tmp_offwarp) a un fallback silencieux
// (dmgcapture_run2.bin) quand {film}_dmg.bin manque → il comparait 000d5950 (Fiesta) à une
// capture d'un AUTRE match. Le SEUL film avec une paire dmg+kill propre = 9b191a7f (TS).
//
// Ce tool décode l'OFFLINE (0xE6 kills + 0xd2 dégâts, même horloge) ET la capture LIVE du
// MÊME match (pas de fallback : erreur dure si absente), attribue l'arme des DEUX côtés par
// la MÊME règle (dernier dégât du tueur avant le kill), puis compare PAR JOUEUR.
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_kwval [filmID]   # défaut 9b191a7f
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"levelup/go-api/internal/analysis"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
)

const root = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks`
const liveDir = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/.claude/worktrees/weapon-attribution-v3/tools/ce`
const kevPath = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/.claude/worktrees/filmdec-continuation/.ai/re_dump/killevents_sample.txt`
const sfx = uint32(0x42c9679f)

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

func bitsAt(d []byte, bp, n int) uint64 {
	var v uint64
	for i := 0; i < n; i++ {
		p := bp + i
		if p>>3 >= len(d) {
			v <<= 1
			continue
		}
		v = (v << 1) | uint64((d[p>>3]>>uint(7-(p&7)))&1)
	}
	return v
}

// idxD/idxK : IDs d'entité de la capture CE -> idx 0-7 (mêmes constantes que tmp_offwarp).
func idxD(h uint32) int {
	if h < 0xEC500000 || h > 0xEC600000 {
		return -1
	}
	return int((h - 0xEC500000) / 0x10002)
}
func idxK(h uint32) int {
	if h < 0xE1500000 || h > 0xE1600000 {
		return -1
	}
	return int((h - 0xE1500000) / 0x10002)
}

type killEv struct {
	killer, victim int
	ts             uint64
}
type dmgEv struct {
	atk     int
	fam     string
	ts      uint64
	firearm bool
}
type attr struct {
	killer, victim int
	w              string
}

// collectRaw : parse les chunks → kills (tueur=R5@killOff, victime=R5@killOff+5, grammaire
// Ghidra FUN_14104bd08 : struct fixe [R5 tueur][R5 victime][R32][R1][R5 assist][R32] ; le champ
// est un index LOCAL 5-bit, pas de bit de présence — l'absence se résout au registre, offline =
// index brut) et dégâts 0xd2 triés par ts. Même horloge frame pour les deux flux.
func collectRaw(m string, h32 map[uint32]string, killOff int) ([]killEv, []dmgEv) {
	cache := root + "/" + m
	var kills []killEv
	var dmgs []dmgEv
	for ch := 0; ch <= 41; ch++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, ch))
		off := 0
		for off+16 <= len(d) {
			typ := binary.LittleEndian.Uint16(d[off:])
			sz := int(binary.LittleEndian.Uint32(d[off+4:]))
			ts := binary.LittleEndian.Uint64(d[off+8:])
			if sz <= 0 || off+16+sz > len(d) {
				break
			}
			pl := d[off+16 : off+16+sz]
			off += 16 + sz
			if typ != 0 || len(pl) == 0 {
				continue
			}
			switch pl[0] {
			case 0xE6:
				kills = append(kills, killEv{int(bitsAt(pl, killOff, 5)), int(bitsAt(pl, killOff+5, 5)), ts})
			case 0xd2:
				// Lecture film-INDÉPENDANTE via le préambule déterministe, base=24 (calibré pur-film par
				// attacker==field1, confirmé Ghidra = 16b marqueur + 8b en-tête parent). La CAUSE = la
				// FAMILLE (S6, param_3+0x10) : le tag source de dégât (vérifié vs oracle align_dmg = 9
				// familles firearm exactes). Le "variant" (S7, +0x14) est CONSTANT 0x42C9679F -> ignoré.
				rec, ok := parsePreamble(pl, 24)
				if !ok || rec.attacker < 0 || rec.family == 0xffffffff {
					break
				}
				dmgs = append(dmgs, dmgEv{rec.attacker, weaponName(rec.family, h32), ts, true})
			}
		}
	}
	sort.Slice(dmgs, func(i, j int) bool { return dmgs[i].ts < dmgs[j].ts })
	return kills, dmgs
}

// decodeOffline : attribution same-clock. fatalOnly=true : ignore le tueur 0xE6 et prend
// l'attaquant du DERNIER 0xd2 avant l'instant du kill = coup fatal (0xE6 = horodatage seul).
func decodeOffline(m string, h32 map[uint32]string, killOff int, fatalOnly bool) []attr {
	kills, dmgs := collectRaw(m, h32, killOff)
	var out []attr
	for _, k := range kills {
		var bestW, anyW string
		anyAtk := -1
		bestDT, anyDT := uint64(1<<63), uint64(1<<63)
		for _, dg := range dmgs {
			if dg.ts > k.ts {
				continue
			}
			dt := k.ts - dg.ts
			if dt < anyDT {
				anyDT, anyW, anyAtk = dt, dg.fam, dg.atk
			}
			if dg.atk == k.killer && dt < bestDT {
				bestDT, bestW = dt, dg.fam
			}
		}
		killer, w := k.killer, bestW
		if fatalOnly {
			killer, w = anyAtk, anyW // coup fatal : 0xE6 = simple horodatage, tueur+arme = dernier 0xd2
		} else if w == "" && anyW != "" && anyDT < 50_000_000 {
			w = anyW // fallback coup-fatal collé (tueur 0xE6 mal décodé)
		}
		out = append(out, attr{killer, k.victim, w})
	}
	return out
}

// decodeLive : capture CE {m}_dmg.bin + {m}_kill.bin (MÊME match, pas de fallback).
func decodeLive(m string, h32 map[uint32]string) ([]attr, bool) {
	dd, e1 := os.ReadFile(liveDir + "/" + m + "_dmg.bin")
	kc, e2 := os.ReadFile(liveDir + "/" + m + "_kill.bin")
	if e1 != nil || e2 != nil {
		return nil, false
	}
	type ld struct {
		atk int
		w   string
		tsc uint64
	}
	var lds []ld
	for o := 0; o+32 <= len(dd); o += 32 {
		at := idxD(binary.LittleEndian.Uint32(dd[o:]))
		if at < 0 {
			continue
		}
		tsc := uint64(binary.LittleEndian.Uint32(dd[o+20:]))<<32 | uint64(binary.LittleEndian.Uint32(dd[o+16:]))
		lds = append(lds, ld{at, h32[binary.LittleEndian.Uint32(dd[o+8:])], tsc})
	}
	type lk struct {
		kil, vic int
		tsc      uint64
	}
	var lkAll []lk
	for o := 0; o+16 <= len(kc); o += 16 {
		vi := idxK(binary.LittleEndian.Uint32(kc[o:]))
		ki := idxK(binary.LittleEndian.Uint32(kc[o+4:]))
		if ki < 0 || vi < 0 {
			continue
		}
		tsc := uint64(binary.LittleEndian.Uint32(kc[o+12:]))<<32 | uint64(binary.LittleEndian.Uint32(kc[o+8:]))
		lkAll = append(lkAll, lk{ki, vi, tsc})
	}
	sort.Slice(lkAll, func(i, j int) bool { return lkAll[i].tsc < lkAll[j].tsc })
	var lks []lk
	for _, k := range lkAll {
		dup := false
		for j := len(lks) - 1; j >= 0 && k.tsc-lks[j].tsc < 3000000; j-- {
			if lks[j].kil == k.kil && lks[j].vic == k.vic {
				dup = true
				break
			}
		}
		if !dup {
			lks = append(lks, k)
		}
	}
	var out []attr
	for _, k := range lks {
		w := ""
		var bt uint64
		for _, d := range lds {
			if d.atk == k.kil && d.tsc <= k.tsc && d.tsc >= bt {
				bt, w = d.tsc, d.w
			}
		}
		out = append(out, attr{k.kil, k.vic, w})
	}
	return out, true
}

func hist(as []attr) map[int]map[string]int {
	h := map[int]map[string]int{}
	for _, a := range as {
		if a.w == "" {
			continue
		}
		if h[a.killer] == nil {
			h[a.killer] = map[string]int{}
		}
		h[a.killer][a.w]++
	}
	return h
}

func cos(a, b map[string]int) float64 {
	var dot, na, nb float64
	for w, x := range a {
		dot += float64(x) * float64(b[w])
		na += float64(x) * float64(x)
	}
	for _, y := range b {
		nb += float64(y) * float64(y)
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}

func inter(a, b map[string]int) int { // intersection d'histogrammes = kills où l'arme concorde
	s := 0
	for w, x := range a {
		if y := b[w]; y < x {
			s += y
		} else {
			s += x
		}
	}
	return s
}

func fmtHist(h map[string]int) string {
	type kv struct {
		k string
		v int
	}
	var a []kv
	for k, v := range h {
		a = append(a, kv{k, v})
	}
	sort.Slice(a, func(i, j int) bool { return a[i].v > a[j].v })
	s := ""
	for _, x := range a {
		s += fmt.Sprintf("%s:%d ", x.k, x.v)
	}
	return s
}

func main() {
	m := "9b191a7f"
	if len(os.Args) > 1 {
		m = os.Args[1]
	}
	h32 := map[uint32]string{}
	for id := range analysis.WeaponIDToName {
		h32[uint32(id>>32)] = analysis.WeaponIDToName[id]
	}
	// Alias 1-bit : le champ famille du paquet 0xd2 est parfois lu décalé d'1 bit à gauche
	// (family == high32<<1, le bit de poids fort tombe hors fenêtre 32-bit ; le bit de poids faible = MSB
	// du suffixe 0x42=0). On ajoute ces clés décalées, DÉRIVÉES du catalogue (DRY, suit tout ajout futur),
	// pour résoudre 13 familles réelles jusqu'ici affichées "fam-XXXX" (MA40/Needler/S7 Sniper/Sentinel
	// Beam/Skewer...). Priorité aux clés réelles : on n'ajoute l'alias que si la clé est libre.
	for id := range analysis.WeaponIDToName {
		k := uint32(id>>32) << 1
		if _, ok := h32[k]; !ok {
			h32[k] = analysis.WeaponIDToName[id]
		}
	}

	if len(os.Args) > 2 && os.Args[2] == "oracle" {
		runOracle(m, h32)
		return
	}
	if len(os.Args) > 2 && os.Args[2] == "align" {
		runAlign(m, h32)
		return
	}
	if len(os.Args) > 2 && os.Args[2] == "kev" {
		runKillEvents(m, h32)
		return
	}
	if len(os.Args) > 2 && os.Args[2] == "sizes" {
		runSizes(m)
		return
	}
	if len(os.Args) > 2 && os.Args[2] == "match" {
		runMatch(m)
		return
	}
	if len(os.Args) > 2 && os.Args[2] == "decode" {
		runDecode(m, h32)
		return
	}
	if len(os.Args) > 2 && os.Args[2] == "warp" {
		runWarp(m, h32)
		return
	}
	if len(os.Args) > 2 && os.Args[2] == "c27" {
		runChunk27(m)
		return
	}
	if len(os.Args) > 2 && os.Args[2] == "cever" {
		runCever(m, h32)
		return
	}
	if len(os.Args) > 2 && os.Args[2] == "fataldet" {
		runFatalDet(m)
		return
	}
	if len(os.Args) > 2 && os.Args[2] == "deserprobe" {
		runDeserProbe(m)
		return
	}
	if len(os.Args) > 2 && os.Args[2] == "deserlen" {
		runDeserLen(m)
		return
	}
	if len(os.Args) > 2 && os.Args[2] == "deseranchor" {
		runDeserAnchor(m)
		return
	}
	if len(os.Args) > 2 && os.Args[2] == "kedump" {
		runKEDump(m)
		return
	}
	if len(os.Args) > 2 && os.Args[2] == "locate" {
		runLocate(m)
		return
	}
	if len(os.Args) > 2 && os.Args[2] == "sfxcount" {
		runSfxCount(m)
		return
	}
	if len(os.Args) > 2 && os.Args[2] == "desersplit" {
		runDeserSplit(m)
		return
	}
	if len(os.Args) > 2 && os.Args[2] == "anchorscan" {
		runAnchorScan(m)
		return
	}
	if len(os.Args) > 2 && os.Args[2] == "deserprobe2" {
		runDeserProbe2(m)
		return
	}
	if len(os.Args) > 2 && os.Args[2] == "pipeline" {
		runPipeline(m, h32)
		return
	}
	if len(os.Args) > 2 && os.Args[2] == "pipedet" {
		runPipeDet(m)
		return
	}
	if len(os.Args) > 2 && os.Args[2] == "pairmatrix" {
		pipeMultiKE = len(os.Args) >= 4 && os.Args[3] == "multi"
		runPairMatrix(m, h32)
		return
	}
	if len(os.Args) > 2 && os.Args[2] == "weaponfeed" {
		runWeaponFeed(m, h32)
		return
	}
	if len(os.Args) > 2 && os.Args[2] == "hybridfeed" {
		runHybridFeed(m, h32)
		return
	}
	if len(os.Args) > 2 && os.Args[2] == "hybriddump" {
		runHybridDump(m, h32)
		return
	}
	if len(os.Args) > 2 && os.Args[2] == "dmgclass" {
		runDmgClass(m, h32)
		return
	}
	if len(os.Args) > 2 && os.Args[2] == "basescan" {
		runBaseScan(m, h32)
		return
	}
	if len(os.Args) > 2 && os.Args[2] == "precal" {
		runPrecal(m, h32)
		return
	}
	if len(os.Args) > 2 && os.Args[2] == "srcprobe" {
		runSrcProbe(m, h32)
		return
	}
	if len(os.Args) > 2 && os.Args[2] == "c4dead" {
		runC4Dead(m, h32)
		return
	}
	if len(os.Args) > 2 && os.Args[2] == "a0probe" {
		runA0Probe(m, h32)
		return
	}
	if len(os.Args) > 2 && os.Args[2] == "e6scan" {
		runE6Scan(m, h32)
		return
	}
	if len(os.Args) > 2 && os.Args[2] == "deadscan" {
		runDeadScan(m, h32)
		return
	}
	if len(os.Args) > 2 && os.Args[2] == "melhunt" {
		runMelHunt(m, h32)
		return
	}
	if len(os.Args) > 2 && os.Args[2] == "killfeed" {
		runKillFeed(m, h32)
		return
	}
	if len(os.Args) > 2 && os.Args[2] == "idxmap" {
		runIdxMap(m, h32)
		return
	}
	if len(os.Args) > 2 && os.Args[2] == "gap" {
		runGap(m, h32)
		return
	}
	if len(os.Args) > 2 && os.Args[2] == "augcov" {
		runAugCov(m, h32)
		return
	}
	if len(os.Args) > 2 && os.Args[2] == "augcov2" {
		runAugCov2(m, h32)
		return
	}
	if len(os.Args) > 2 && os.Args[2] == "augcov2live" {
		runAugCov2Live(m, h32)
		return
	}
	if len(os.Args) > 2 && os.Args[2] == "augcovsc" {
		runAugCovSC(m, h32)
		return
	}
	if len(os.Args) > 2 && os.Args[2] == "kfaug" {
		runKillFeedAug(m, h32)
		return
	}
	if len(os.Args) > 2 && os.Args[2] == "deserlen2" {
		runDeserLen2(m)
		return
	}
	if len(os.Args) > 2 && os.Args[2] == "varcls" {
		runVarCls(m, h32)
		return
	}
	if len(os.Args) > 2 && os.Args[2] == "basecheck" {
		runBaseCheck(m)
		return
	}
	if len(os.Args) > 2 && os.Args[2] == "fullcheck" {
		runFullCheck(m)
		return
	}
	if len(os.Args) > 2 && os.Args[2] == "bracketcheck" {
		runBracketCheck(m)
		return
	}
	if len(os.Args) > 2 && os.Args[2] == "walkdump" {
		runWalkDump(m)
		return
	}
	if len(os.Args) > 2 && os.Args[2] == "dispatchcheck" {
		runDispatchCheck(m)
		return
	}
	if len(os.Args) > 2 && os.Args[2] == "burst" {
		runBurst(m)
		return
	}
	if len(os.Args) > 2 && os.Args[2] == "walkval" {
		runWalkVal(m)
		return
	}
	if len(os.Args) > 2 && os.Args[2] == "modelcheck" {
		runModelCheck(m)
		return
	}
	if len(os.Args) > 2 && os.Args[2] == "killpre" {
		runKillPre(m)
		return
	}
	if len(os.Args) > 2 && os.Args[2] == "presweep" {
		runPresenceSweep(m)
		return
	}
	if len(os.Args) > 2 && os.Args[2] == "loopstart" {
		runLoopStart(m)
		return
	}
	if len(os.Args) > 2 && os.Args[2] == "killdecode" {
		runKillDecode(m)
		return
	}
	if len(os.Args) > 2 && os.Args[2] == "killscan" {
		runKillScan(m)
		return
	}
	if len(os.Args) > 2 && os.Args[2] == "code36check" {
		runCode36Check(m)
		return
	}
	if len(os.Args) > 2 && os.Args[2] == "keallscan" {
		runKEAllScan(m, h32)
		return
	}
	if len(os.Args) > 2 && os.Args[2] == "famscan" {
		runFamScan(m, h32)
		return
	}
	if len(os.Args) > 2 && os.Args[2] == "kefields" {
		runKEFields(m, h32)
		return
	}
	if len(os.Args) > 2 && os.Args[2] == "markerscan" {
		runMarkerScan(m, h32)
		return
	}
	if len(os.Args) > 2 && os.Args[2] == "skipprobe" {
		runSkipProbe(m, h32)
		return
	}
	if len(os.Args) > 2 && os.Args[2] == "varscan" {
		runVarScan(m, h32)
		return
	}
	// verifkills : accepte les DEUX ordres ("<film> verifkills" comme les autres modes, ET
	// "verifkills <film>" tel qu'invoqué dans la consigne).
	if len(os.Args) > 2 && os.Args[2] == "verifkills" {
		runVerifKills(m, h32)
		return
	}
	if os.Args[1] == "verifkills" {
		film := "9b191a7f"
		if len(os.Args) > 2 {
			film = os.Args[2]
		}
		runVerifKills(film, h32)
		return
	}
	fatalOnly := len(os.Args) > 2 && os.Args[2] == "fatal"
	if fatalOnly {
		fmt.Println("[MODE FATAL] tueur+arme = dernier 0xd2 avant l'instant du kill (0xE6 = horodatage seul)")
	}

	off := decodeOffline(m, h32, 83, fatalOnly) // S=83 : tueur R5@83 (Ghidra + oracle)
	live, ok := decodeLive(m, h32)
	if !ok {
		fmt.Printf("PAS de capture live MEME MATCH pour %s (%s_dmg.bin + %s_kill.bin absents).\n", m, m, m)
		fmt.Println("Refus du fallback : comparer a un autre match n'a aucun sens. Seul 9b191a7f a une paire propre.")
		return
	}
	offH, liveH := hist(off), hist(live)

	// total kills nommés par côté
	offN, liveN := 0, 0
	offG, liveG := map[string]int{}, map[string]int{}
	for _, a := range off {
		if a.w != "" {
			offN++
			offG[a.w]++
		}
	}
	for _, a := range live {
		if a.w != "" {
			liveN++
			liveG[a.w]++
		}
	}
	fmt.Printf("=== VALIDATION MEME MATCH %s ===\n", m)
	fmt.Printf("offline : %d kills (0xE6), %d avec arme nommee\n", len(off), offN)
	fmt.Printf("live CE : %d kills (dedup), %d avec arme nommee\n", len(live), liveN)

	fmt.Printf("\n--- DISTRIBUTION GLOBALE ---\n")
	fmt.Printf("offline: %s\n", fmtHist(offG))
	fmt.Printf("live   : %s\n", fmtHist(liveG))
	gi := inter(offG, liveG)
	fmt.Printf("intersection globale = %d  (accord distribution : %.0f%% du live)\n", gi, float64(gi)*100/float64(max(liveN, 1)))

	// mapping slot offline -> idx live : d'abord IDENTITE (slot==idx), puis greedy cosine.
	var slots, idxs []int
	for s := range offH {
		slots = append(slots, s)
	}
	for i := range liveH {
		idxs = append(idxs, i)
	}
	sort.Ints(slots)
	sort.Ints(idxs)

	fmt.Printf("\n--- PAR JOUEUR (mapping IDENTITE slot==idx) ---\n")
	idOK, idTot := 0, 0
	all := map[int]bool{}
	for _, s := range slots {
		all[s] = true
	}
	for _, i := range idxs {
		all[i] = true
	}
	var keys []int
	for k := range all {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	for _, k := range keys {
		o, l := offH[k], liveH[k]
		it := inter(o, l)
		ln := 0
		for _, v := range l {
			ln += v
		}
		on := 0
		for _, v := range o {
			on += v
		}
		idOK += it
		idTot += ln
		fmt.Printf(" joueur %2d | off(%2d): %-46s | live(%2d): %-46s | accord=%d\n", k, on, fmtHist(o), ln, fmtHist(l), it)
	}
	fmt.Printf("ACCORD par-joueur (identite) = %d/%d = %.0f%% des kills live\n", idOK, idTot, float64(idOK)*100/float64(max(idTot, 1)))

	// greedy cosine (au cas où slot != idx)
	used := map[int]bool{}
	mapSI := map[int]int{}
	for _, s := range slots {
		best, bi := -1.0, -1
		for _, i := range idxs {
			if used[i] {
				continue
			}
			if c := cos(offH[s], liveH[i]); c > best {
				best, bi = c, i
			}
		}
		if bi >= 0 {
			mapSI[s] = bi
			used[bi] = true
		}
	}
	fmt.Printf("\n--- PAR JOUEUR (mapping GREEDY cosine) ---\n")
	gOK, gTot := 0, 0
	for _, s := range slots {
		i := mapSI[s]
		o, l := offH[s], liveH[i]
		it := inter(o, l)
		ln := 0
		for _, v := range l {
			ln += v
		}
		gOK += it
		gTot += ln
		fmt.Printf(" slot %2d -> idx %2d | off: %-46s | live: %-46s | accord=%d\n", s, i, fmtHist(o), fmtHist(l), it)
	}
	fmt.Printf("ACCORD par-joueur (greedy) = %d/%d = %.0f%% des kills live apparies\n", gOK, gTot, float64(gOK)*100/float64(max(gTot, 1)))
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// collectE6 : payloads des frames 0xE6 (kill-event) en ordre chronologique.
func collectE6(m string) [][]byte {
	cache := root + "/" + m
	type fr struct {
		ts uint64
		pl []byte
	}
	var frs []fr
	for ch := 0; ch <= 41; ch++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, ch))
		off := 0
		for off+16 <= len(d) {
			typ := binary.LittleEndian.Uint16(d[off:])
			sz := int(binary.LittleEndian.Uint32(d[off+4:]))
			ts := binary.LittleEndian.Uint64(d[off+8:])
			if sz <= 0 || off+16+sz > len(d) {
				break
			}
			pl := d[off+16 : off+16+sz]
			off += 16 + sz
			if typ != 0 || len(pl) == 0 || pl[0] != 0xE6 {
				continue
			}
			cp := make([]byte, len(pl))
			copy(cp, pl)
			frs = append(frs, fr{ts, cp})
		}
	}
	sort.Slice(frs, func(i, j int) bool { return frs[i].ts < frs[j].ts })
	var out [][]byte
	for _, f := range frs {
		out = append(out, f.pl)
	}
	return out
}

// runOracle : cherche le décalage de bits 0xE6 (tueur/victime) qui reproduit le multiset
// tueurs de la vérité-terrain (8 joueurs propres). L'accuracy est plafonnée par la qualité
// du décodage 0xE6 : cet oracle trouve le bon offset au lieu de deviner (80 = mauvais).
func runOracle(m string, h32 map[uint32]string) {
	e6 := collectE6(m)
	live, ok := decodeLive(m, h32)
	if !ok {
		fmt.Printf("PAS de capture live MEME MATCH pour %s — oracle impossible.\n", m)
		return
	}
	lk := map[int]int{}
	for _, a := range live {
		lk[a.killer]++
	}
	var liveCounts []int
	for _, v := range lk {
		liveCounts = append(liveCounts, v)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(liveCounts)))
	fmt.Printf("=== ORACLE 0xE6 %s : %d frames 0xE6, %d kills live, %d tueurs live ===\n", m, len(e6), len(live), len(lk))
	fmt.Printf("multiset tueurs live (desc) = %v (somme %d)\n", liveCounts, len(live))

	// grammaire RÉELLE (Ghidra FUN_14104bd08) : tueur=R5@S, victime=R5@(S+5), fixe.
	decodeAt := func(pl []byte, bo int) (int, int) {
		return int(bitsAt(pl, bo, 5)), int(bitsAt(pl, bo+5, 5))
	}
	type sc struct {
		off, score, players, decoded int
	}
	var scs []sc
	for bo := 8; bo <= 512; bo++ {
		dist := map[int]int{}
		decoded := 0
		for _, pl := range e6 {
			k, v := decodeAt(pl, bo)
			if k != v && k < 24 && v < 24 { // un kill a tueur != victime
				dist[k]++
				decoded++
			}
		}
		var dc []int
		for _, c := range dist {
			dc = append(dc, c)
		}
		sort.Sort(sort.Reverse(sort.IntSlice(dc)))
		score := 0
		for i := 0; i < len(liveCounts) && i < len(dc); i++ {
			if dc[i] < liveCounts[i] {
				score += dc[i]
			} else {
				score += liveCounts[i]
			}
		}
		// pénalise l'écart au nombre de joueurs attendu (8) : les faux offsets éparpillent.
		score -= 2 * abs(len(dist)-len(lk))
		scs = append(scs, sc{bo, score, len(dist), decoded})
	}
	sort.Slice(scs, func(i, j int) bool { return scs[i].score > scs[j].score })
	fmt.Println("top offsets (score = accord multiset - pénalité joueurs) :")
	for i := 0; i < 16 && i < len(scs); i++ {
		s := scs[i]
		fmt.Printf("  bitoff=%3d (byte %d.%d) score=%d players=%d decoded=%d\n", s.off, s.off/8, s.off%8, s.score, s.players, s.decoded)
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// runMatch : localise le kill-event dans mes paquets 0xE6 offline via la SIGNATURE d'octets de la
// capture CE (killevents_sample.txt). Chaque ligne KILL porte la fenêtre de bits (hex24) du reader.
// Si cette signature apparaît dans un paquet 0xE6 à l'octet O, le kill-event y est à bit O*8+bit%8 →
// PROUVE que le buffer CE = mon paquet, et lit tueur=R5@(O*8+bit%8), victime=R5@(+5). Confirme le
// modèle "kill-event profond dans le paquet 0xE6" et la grammaire, sans décoder la record-loop.
func runMatch(m string) {
	raw, err := os.ReadFile(kevPath)
	if err != nil {
		fmt.Printf("killevents_sample.txt introuvable: %v\n", err)
		return
	}
	// concatène tous les chunks inflatés (buffer continu potentiel)
	cache := root + "/" + m
	var chunks [][]byte
	for ch := 0; ch <= 41; ch++ {
		chunks = append(chunks, inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, ch)))
	}
	rev8 := func(b []byte) []byte { // reverse chaque groupe de 8 octets (dé-byteswap du reader)
		out := make([]byte, len(b))
		for i := 0; i < len(b); i += 8 {
			for j := 0; j < 8 && i+j < len(b); j++ {
				k := i + 7 - j
				if k < len(b) {
					out[i+j] = b[k]
				}
			}
		}
		return out
	}
	revAll := func(b []byte) []byte { // reverse total
		out := make([]byte, len(b))
		for i := range b {
			out[i] = b[len(b)-1-i]
		}
		return out
	}
	var found, total int
	fmt.Printf("=== MATCH signature CE -> chunks inflatés (%s) ===\n", m)
	nDmg := 0
	for _, ln := range strings.Split(string(raw), "\n") {
		f := strings.Fields(ln)
		if len(f) < 4 || (f[0] != "KILL" && f[0] != "DMG") {
			continue
		}
		if f[0] == "DMG" {
			if nDmg >= 6 {
				continue // 6 DMG de contrôle suffisent
			}
			nDmg++
		}
		total++
		bitpos, _ := strconv.Atoi(f[1])
		win, e := hex.DecodeString(f[3])
		if e != nil || len(win) < 16 {
			continue
		}
		variants := map[string][]byte{"brut": win[:16], "rev8": rev8(win)[:16], "revAll": revAll(win)[:16]}
		hitDesc := ""
		for name, sig := range variants {
			for ch, d := range chunks {
				if o := indexBytes(d, sig); o >= 0 {
					hitDesc = fmt.Sprintf("[%s] chunk %d octet %d", name, ch, o)
					break
				}
			}
			if hitDesc != "" {
				break
			}
		}
		if hitDesc == "" {
			fmt.Printf("  %-4s bitpos=%-5d : INTROUVABLE (brut/rev8/revAll)\n", f[0], bitpos)
			continue
		}
		found++
		fmt.Printf("  %-4s bitpos=%-5d -> %s\n", f[0], bitpos, hitDesc)
	}
	fmt.Printf("=> %d/%d signatures (KILL+DMG contrôle) retrouvées.\n", found, total)
}

func indexBytes(hay, needle []byte) int {
	for i := 0; i+len(needle) <= len(hay); i++ {
		match := true
		for j := range needle {
			if hay[i+j] != needle[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

// runSizes : distribution des tailles de paquets par marqueur payload[0]. But : trancher si le
// kill-event (0xE6) est un GROS paquet (record-loop avant, bitpos CE 578-2756 y tient) ou PETIT
// (autonome → le bitpos CE vient d'un autre buffer). Compare aux bitpos CE observés (max 2756 = 344 o).
func runSizes(m string) {
	cache := root + "/" + m
	sizesByMarker := map[byte][]int{}
	for ch := 0; ch <= 41; ch++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, ch))
		off := 0
		for off+16 <= len(d) {
			typ := binary.LittleEndian.Uint16(d[off:])
			sz := int(binary.LittleEndian.Uint32(d[off+4:]))
			if sz <= 0 || off+16+sz > len(d) {
				break
			}
			pl := d[off+16 : off+16+sz]
			off += 16 + sz
			if typ != 0 || len(pl) == 0 {
				continue
			}
			sizesByMarker[pl[0]] = append(sizesByMarker[pl[0]], sz)
		}
	}
	report := func(mk byte) {
		s := sizesByMarker[mk]
		if len(s) == 0 {
			fmt.Printf("  marqueur 0x%02X : absent\n", mk)
			return
		}
		sort.Ints(s)
		sum := 0
		for _, x := range s {
			sum += x
		}
		fmt.Printf("  marqueur 0x%02X : %d paquets | taille min=%d méd=%d max=%d moy=%d octets (=%d bits max)\n",
			mk, len(s), s[0], s[len(s)/2], s[len(s)-1], sum/len(s), s[len(s)-1]*8)
	}
	fmt.Printf("=== TAILLES paquets type-0 %s (bitpos CE kill observés: 578..2756 = 72..344 o) ===\n", m)
	report(0xE6)
	report(0xd2)
	// top marqueurs par volume
	type mc struct {
		mk byte
		n  int
	}
	var all []mc
	for mk, s := range sizesByMarker {
		all = append(all, mc{mk, len(s)})
	}
	sort.Slice(all, func(i, j int) bool { return all[i].n > all[j].n })
	fmt.Print("marqueurs les plus fréquents: ")
	for i := 0; i < 10 && i < len(all); i++ {
		fmt.Printf("0x%02X:%d ", all[i].mk, all[i].n)
	}
	fmt.Println()
}

// runPipeline : PIPELINE OFFLINE COMPLET kill-feed. Détecte les paquets de dégât FATAUX (marqueur
// damage-family + taille > seuil), y lit l'arme (famille du dégât), et localise le kill-event
// embarqué (locateKillEventCursor) → victime=field0, tueur=field1 (index joueur DIRECT 0..7, readOpt).
// Décodage délégué à decodePipeline (source unique). Sortie : kill feed (tueur, victime, arme).
// Valide vs chunk_27 (killer/victime offline) + CE consumer si dispo. Universel, offline, sans CE.
func runPipeline(m string, h32 map[uint32]string) {
	cache := root + "/" + m
	dedup, feedLen, diag := decodePipeline(m, h32)
	nFatal, nLoc, nNoKE, nRoster := diag.nFatal, diag.nLoc, diag.nNoKE, diag.nRoster
	markerLoc := diag.markerLoc
	fmt.Printf("=== PIPELINE OFFLINE %s : %d fatals détectés, %d localisés (%d sans kill-event, %d hors-roster) ===\n", m, nFatal, nLoc, nNoKE, nRoster)
	fmt.Printf("localisés par marqueur : ")
	for _, mk := range []byte{0xD2, 0xD3, 0xC0, 0xC2, 0xC3, 0xCA, 0xE9} {
		if markerLoc[mk] > 0 {
			fmt.Printf("0x%02X:%d ", mk, markerLoc[mk])
		}
	}
	fmt.Println()
	wd := map[string]int{}
	kd, vd := map[int]int{}, map[int]int{}
	nSelf := 0
	for _, k := range dedup {
		wd[k.weapon]++
		kd[k.killer]++
		vd[k.victim]++
		if k.killer == k.victim {
			nSelf++
		}
	}
	fmt.Printf("kill feed : %d kills bruts -> %d après dédup (tueur,victime,~3ms) | self-kills=%d\n", feedLen, len(dedup), nSelf)
	fmt.Printf("armes : %s\n", fmtHist(wd))
	fmt.Printf("tueurs : %s\n", sortedCounts(kd))
	fmt.Printf("victimes : %s\n", sortedCounts(vd))

	// validation killer/victime vs chunk_27 (offline, tout match).
	var kills27, deaths27 []uint64
	for chi := 41; chi >= 18; chi-- {
		b := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, chi))
		if len(b) == 0 {
			continue
		}
		evs, _ := analysis.ParseHighlightEvents(b, 0)
		var kk, dd []uint64
		for _, e := range evs {
			if e.EventType == analysis.EventTypeKill {
				kk = append(kk, e.XUID)
			} else if e.EventType == analysis.EventTypeDeath {
				dd = append(dd, e.XUID)
			}
		}
		if len(kk) > len(kills27) {
			kills27, deaths27 = kk, dd
		}
	}
	kx := map[uint64]int{}
	for _, x := range kills27 {
		kx[x]++
	}
	vx := map[uint64]int{}
	for _, x := range deaths27 {
		vx[x]++
	}
	fmt.Printf("chunk_27 (vérité offline) : %d kills, %d tueurs distincts, %d morts\n", len(kills27), len(kx), len(deaths27))
	// accord de distribution (multiset trié, permutation-invariant) tueur & victime.
	msetInterU := func(a map[int]int, b map[uint64]int) int {
		var av, bv []int
		for _, v := range a {
			av = append(av, v)
		}
		for _, v := range b {
			bv = append(bv, v)
		}
		sort.Sort(sort.Reverse(sort.IntSlice(av)))
		sort.Sort(sort.Reverse(sort.IntSlice(bv)))
		s := 0
		for i := 0; i < len(av) && i < len(bv); i++ {
			if av[i] < bv[i] {
				s += av[i]
			} else {
				s += bv[i]
			}
		}
		return s
	}
	ik, iv := msetInterU(kd, kx), msetInterU(vd, vx)
	fmt.Printf(">>> couverture : %d/%d = %.0f%% des kills chunk_27\n", len(dedup), len(kills27), float64(len(dedup))*100/float64(max(len(kills27), 1)))
	fmt.Printf(">>> accord distribution TUEUR vs chunk_27 : %d/%d = %.0f%% | VICTIME : %d/%d = %.0f%%\n",
		ik, len(dedup), float64(ik)*100/float64(max(len(dedup), 1)), iv, len(dedup), float64(iv)*100/float64(max(len(dedup), 1)))
	// VALIDATION PAR-PAIRE vs vérité-terrain CE (si dispo) : chaque paire (tueur,victime) du pipeline
	// existe-t-elle réellement dans le match ? (test dur, pas juste distribution.) + ventilation par marqueur.
	if kcb, err := os.ReadFile(liveDir + "/" + m + "_align_kill.bin"); err == nil {
		truePairs := map[[2]int]bool{}
		for o := 0; o+16 <= len(kcb); o += 16 {
			vic := idxK(binary.LittleEndian.Uint32(kcb[o:]))
			kil := idxK(binary.LittleEndian.Uint32(kcb[o+4:]))
			if kil >= 0 && vic >= 0 {
				truePairs[[2]int{kil, vic}] = true
			}
		}
		inSet := 0
		perMk := map[byte][2]int{} // marqueur -> [dansSet, total]
		for _, k := range dedup {
			e := perMk[k.marker]
			e[1]++
			if truePairs[[2]int{k.killer, k.victim}] {
				inSet++
				e[0]++
			}
			perMk[k.marker] = e
		}
		fmt.Printf(">>> VALIDATION CE par-paire (après dédup) : %d/%d paires (tueur,victime) RÉELLES = %.1f%%\n",
			inSet, len(dedup), float64(inSet)*100/float64(max(len(dedup), 1)))
		fmt.Printf("    par marqueur : ")
		for _, mk := range []byte{0xD2, 0xD3, 0xC0, 0xC2, 0xC3, 0xCA, 0xE9} {
			if perMk[mk][1] > 0 {
				fmt.Printf("0x%02X:%d/%d ", mk, perMk[mk][0], perMk[mk][1])
			}
		}
		fmt.Println()
		fmt.Println("    PAIRES FAUSSES (non dans truePairs) :")
		for _, k := range dedup {
			if !truePairs[[2]int{k.killer, k.victim}] {
				fmt.Printf("      0x%02X kil=%d vic=%d cur=%d f2=%d nCand=%d sz=%d w=%s\n",
					k.marker, k.killer, k.victim, k.cur, k.f2, k.nCand, k.sz, k.weapon)
			}
		}
	}
}

// runFatalDet : cherche comment DÉTECTER offline les paquets de dégât FATAUX (ceux qui portent le
// kill-event embarqué) via l'oracle CE. Pour chaque killdeser matché → paquet contenant (marqueur,
// taille payload, curseur). Compare aux paquets NON-fataux de mêmes marqueurs. Hypothèses testées :
// H1 = les fataux sont plus GROS (event kill en plus) ; H3 = le curseur (offset kill-event) est ~fixe.
func runFatalDet(m string) {
	baseP := `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/.claude/worktrees/weapon-attribution-v3/tools/ce/` + m + "_align_"
	kd, e1 := os.ReadFile(baseP + "killdeser.bin")
	if e1 != nil {
		fmt.Printf("killdeser introuvable: %v\n", e1)
		return
	}
	cache := root + "/" + m
	var chunks [][]byte
	for ch := 0; ch <= 41; ch++ {
		chunks = append(chunks, inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, ch)))
	}
	type pkt struct {
		marker            byte
		plen, cursor, off int
		ch                int
	}
	var fatal []pkt
	fatalKey := map[[2]int]bool{} // (chunk, payloadStart) des fataux
	for o := 0; o+128 <= len(kd); o += 128 {
		cursor := int(binary.LittleEndian.Uint32(kd[o+8:]))
		win := kd[o+16 : o+16+16]
		hit, hitCh := -1, -1
		for ci, d := range chunks {
			if x := indexBytes(d, win); x >= 0 {
				hit, hitCh = x, ci
				break
			}
		}
		if hit < 0 {
			continue
		}
		d := chunks[hitCh]
		off := 0
		for off+16 <= len(d) {
			sz := int(binary.LittleEndian.Uint32(d[off+4:]))
			if sz <= 0 || off+16+sz > len(d) {
				break
			}
			if hit >= off && hit < off+16+sz {
				fatal = append(fatal, pkt{d[off+16], sz, cursor, off, hitCh})
				fatalKey[[2]int{hitCh, off}] = true
				break
			}
			off += 16 + sz
		}
	}
	fmt.Printf("=== FATALDET %s : %d paquets fatals identifiés ===\n", m, len(fatal))
	// PRÉCISION du locator R7=85 vs le curseur VRAI (CE) : le scan tombe-t-il pile sur le kill-event ?
	{
		readOptAt := func(pl []byte, bp int) (int, int) {
			if bp < 0 || bp>>3 >= len(pl) {
				return -2, bp
			}
			if bitsAt(pl, bp, 1) == 0 {
				return int(bitsAt(pl, bp+1, 5)), bp + 6
			}
			return -1, bp + 1
		}
		ke := func(pl []byte, b int) bool {
			v, b2 := readOptAt(pl, b)
			k, b3 := readOptAt(pl, b2)
			if v < 0 || k < 0 || v >= 16 || k >= 16 || v == k || v%2 != 0 || k%2 != 0 {
				return false
			}
			a, _ := readOptAt(pl, b3+33)
			return a == -1 || (a >= 0 && a < 16 && a%2 == 0)
		}
		// dérive l'ancre : le préambule de W bits juste avant field0 (fixe 134/134).
		for _, W := range []int{10, 12, 14, 16, 18} {
			amode := map[uint64]int{}
			for _, p := range fatal {
				pl := chunks[p.ch][p.off+16 : p.off+16+p.plen]
				if p.cursor-W >= 0 {
					amode[bitsAt(pl, p.cursor-W, W)]++
				}
			}
			anchor, an := uint64(0), 0
			for v, n := range amode {
				if n > an {
					anchor, an = v, n
				}
			}
			nExact := 0
			for _, p := range fatal {
				pl := chunks[p.ch][p.off+16 : p.off+16+p.plen]
				loc := -1
				for b := 100; b+W+17 <= p.plen*8; b++ {
					if bitsAt(pl, b, W) == anchor && ke(pl, b+W) {
						loc = b + W
						break
					}
				}
				if loc == p.cursor {
					nExact++
				}
			}
			fmt.Printf("ANCRE %d bits (0x%X, constante %d/%d) : locator EXACT %d/%d\n", W, anchor, an, len(fatal), nExact, len(fatal))
		}
	}
	// H3 : curseur (offset kill-event dans le buffer) par marqueur.
	curByMk := map[byte][]int{}
	for _, p := range fatal {
		curByMk[p.marker] = append(curByMk[p.marker], p.cursor)
	}
	// TYPE du kill-event : R7 juste avant field0. field0=cursor ; dispatcher = R7(7)+3loop(3) -> R7 à
	// cursor-10. On teste plusieurs offsets (le 3-loop pourrait varier) pour trouver le code constant.
	for _, back := range []int{7, 8, 9, 10, 11, 13} {
		h := map[uint64]int{}
		for _, p := range fatal {
			pl := chunks[p.ch][p.off+16 : p.off+16+p.plen]
			if p.cursor-back >= 0 {
				h[bitsAt(pl, p.cursor-back, 7)]++
			}
		}
		// top valeur
		bestv, bestn := uint64(0), 0
		for v, n := range h {
			if n > bestn {
				bestv, bestn = v, n
			}
		}
		fmt.Printf("R7 à cursor-%d : top=%d (=0x%02X) sur %d/%d\n", back, bestv, bestv, bestn, len(fatal))
	}
	fmt.Println("H3 — curseur (offset kill-event) par marqueur :")
	for mk, cs := range curByMk {
		sort.Ints(cs)
		fmt.Printf("  0x%02X (%d) : min=%d méd=%d max=%d\n", mk, len(cs), cs[0], cs[len(cs)/2], cs[len(cs)-1])
	}
	// LOCATOR : le kill-event (curseur) est-il à offset ~fixe après le suffixe d'arme 42c9679f ?
	// Pour chaque paquet fatal 0xD2, cherche le suffixe (32 bits, tout offset) le plus proche AVANT le
	// curseur, et reporte (curseur - fin_suffixe). Si groupé -> locator = suffixe + offset.
	{
		var deltas []int
		nNoSfx := 0
		for _, p := range fatal {
			if p.marker != 0xD2 {
				continue
			}
			pl := chunks[p.ch][p.off+16 : p.off+16+p.plen]
			lastSfx := -1
			for b := 0; b+32 <= p.plen*8 && b < p.cursor; b++ {
				if uint32(bitsAt(pl, b, 32)) == sfx {
					lastSfx = b + 32 // fin du suffixe
				}
			}
			if lastSfx < 0 {
				nNoSfx++
				continue
			}
			deltas = append(deltas, p.cursor-lastSfx)
		}
		sort.Ints(deltas)
		fmt.Printf("LOCATOR suffixe→curseur (0xD2, %d avec suffixe, %d sans) : ", len(deltas), nNoSfx)
		if len(deltas) > 0 {
			bk := map[int]int{}
			for _, d := range deltas {
				bk[d/10*10]++
			}
			var ks []int
			for k := range bk {
				ks = append(ks, k)
			}
			sort.Ints(ks)
			for _, k := range ks {
				if bk[k] >= 2 {
					fmt.Printf("[%d-%d]:%d ", k, k+10, bk[k])
				}
			}
			fmt.Printf("(min=%d méd=%d max=%d)", deltas[0], deltas[len(deltas)/2], deltas[len(deltas)-1])
		}
		fmt.Println()
	}
	// distribution fine des curseurs 0xD2 (le marqueur principal) : groupés ?
	if d2 := curByMk[0xD2]; len(d2) > 0 {
		sort.Ints(d2)
		buckets := map[int]int{}
		for _, c := range d2 {
			buckets[c/20*20]++ // buckets de 20 bits
		}
		var bk []int
		for b := range buckets {
			bk = append(bk, b)
		}
		sort.Ints(bk)
		fmt.Print("  0xD2 curseurs (buckets de 20 bits): ")
		for _, b := range bk {
			fmt.Printf("[%d-%d]:%d ", b, b+20, buckets[b])
		}
		fmt.Println()
	}
	// H1 : taille payload fatal vs NON-fatal, par marqueur.
	allByMk := map[byte][]int{}
	fatalByMk := map[byte][]int{}
	for _, p := range fatal {
		fatalByMk[p.marker] = append(fatalByMk[p.marker], p.plen)
	}
	for ci, d := range chunks {
		off := 0
		for off+16 <= len(d) {
			typ := binary.LittleEndian.Uint16(d[off:])
			sz := int(binary.LittleEndian.Uint32(d[off+4:]))
			if sz <= 0 || off+16+sz > len(d) {
				break
			}
			if typ == 0 && len(d) > off+16 {
				mk := d[off+16]
				if !fatalKey[[2]int{ci, off}] {
					allByMk[mk] = append(allByMk[mk], sz)
				}
			}
			off += 16 + sz
		}
	}
	fmt.Println("H1 — taille payload FATAL vs NON-fatal (même marqueur) — GAP = détecteur :")
	minv := func(s []int) int {
		if len(s) == 0 {
			return -1
		}
		sort.Ints(s)
		return s[0]
	}
	maxv := func(s []int) int {
		if len(s) == 0 {
			return -1
		}
		sort.Ints(s)
		return s[len(s)-1]
	}
	for mk, fs := range fatalByMk {
		fmt.Printf("  0x%02X : fatal min=%d (n=%d) | non-fatal max=%d (n=%d) %s\n",
			mk, minv(fs), len(fs), maxv(allByMk[mk]), len(allByMk[mk]),
			map[bool]string{true: "SÉPARÉ", false: "chevauche"}[minv(fs) > maxv(allByMk[mk])])
	}
	// rendement du détecteur : combien de paquets damage-family > seuil T (par film)
	dmgMk := map[byte]bool{0xD2: true, 0xC0: true, 0xC2: true, 0xC3: true, 0xCA: true, 0xD3: true, 0xE9: true, 0xC7: true, 0xC4: true, 0x89: true}
	for _, T := range []int{400, 500, 600, 700} {
		n := 0
		for ci, d := range chunks {
			off := 0
			for off+16 <= len(d) {
				typ := binary.LittleEndian.Uint16(d[off:])
				sz := int(binary.LittleEndian.Uint32(d[off+4:]))
				if sz <= 0 || off+16+sz > len(d) {
					break
				}
				if typ == 0 && len(d) > off+16 && dmgMk[d[off+16]] && sz > T {
					n++
				}
				_ = ci
				off += 16 + sz
			}
		}
		fmt.Printf("détecteur : %d paquets damage-family avec payload > %d octets\n", n, T)
	}
}

// runCever : analyse la capture CE d'alignement (filmdec_align_capture.lua). TEST A d'abord :
// les octets bruts du reader (killdeser window) matchent-ils mes chunks offline ? Si oui → alignement
// par contenu → décode field0 offline + lie au consumer par rdtsc → tranche field0 tueur/victime.
func runCever(m string, h32 map[uint32]string) {
	dir := liveDir[:len(liveDir)-len("/weapon-attribution-v3/tools/ce")] + "/weapon-attribution-v3/tools/ce"
	_ = dir
	base := `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/.claude/worktrees/weapon-attribution-v3/tools/ce/` + m + "_align_"
	kd, e1 := os.ReadFile(base + "killdeser.bin")
	kc, e2 := os.ReadFile(base + "kill.bin")
	if e1 != nil || e2 != nil {
		fmt.Printf("fichiers align introuvables: %v %v\n", e1, e2)
		return
	}
	// killdeser rec 128o : rdtsc@0(8) cursor@8(u32) bytePtr@12(u32) data@16(112)
	type kdr struct {
		rdtsc  uint64
		cursor uint32
		win    []byte
	}
	var kds []kdr
	for o := 0; o+128 <= len(kd); o += 128 {
		kds = append(kds, kdr{
			binary.LittleEndian.Uint64(kd[o:]),
			binary.LittleEndian.Uint32(kd[o+8:]),
			kd[o+16 : o+128],
		})
	}
	// consumer rec 16o : vic@0 kil@4 rdtsc@8(8)
	type kcr struct {
		vic, kil int
		rdtsc    uint64
	}
	var kcs []kcr
	for o := 0; o+16 <= len(kc); o += 16 {
		kcs = append(kcs, kcr{
			idxK(binary.LittleEndian.Uint32(kc[o:])),
			idxK(binary.LittleEndian.Uint32(kc[o+4:])),
			binary.LittleEndian.Uint64(kc[o+8:]),
		})
	}
	fmt.Printf("=== CEVER %s : %d killdeser, %d consumer ===\n", m, len(kds), len(kcs))

	// TEST A : match des fenêtres killdeser dans les chunks offline inflatés.
	cache := root + "/" + m
	var chunks [][]byte
	for ch := 0; ch <= 41; ch++ {
		chunks = append(chunks, inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, ch)))
	}
	// paquets 0xE6 offline avec plage d'octets + field0 (readOpt@10) + ts.
	type opkt struct {
		ch, start, end, f0 int
	}
	var opkts []opkt
	for ch := 0; ch <= 41; ch++ {
		d := chunks[ch]
		off := 0
		for off+16 <= len(d) {
			sz := int(binary.LittleEndian.Uint32(d[off+4:]))
			typ := binary.LittleEndian.Uint16(d[off:])
			if sz <= 0 || off+16+sz > len(d) {
				break
			}
			pl := d[off+16 : off+16+sz]
			ps, pe := off, off+16+sz
			off += 16 + sz
			if typ != 0 || len(pl) == 0 || pl[0] != 0xE6 {
				continue
			}
			f0 := -1
			if bitsAt(pl, 10, 1) == 0 {
				f0 = int(bitsAt(pl, 11, 5))
			}
			opkts = append(opkts, opkt{ch, ps, pe, f0})
		}
	}

	// DIAG : marqueur du paquet contenant chaque fenêtre + plages rdtsc.
	markerHist := map[byte]int{}
	for _, r := range kds {
		sig := r.win[:16]
		hit, hitCh := -1, -1
		for ci, d := range chunks {
			if o := indexBytes(d, sig); o >= 0 {
				hit, hitCh = o, ci
				break
			}
		}
		if hit < 0 {
			continue
		}
		d := chunks[hitCh]
		off := 0
		for off+16 <= len(d) {
			sz := int(binary.LittleEndian.Uint32(d[off+4:]))
			if sz <= 0 || off+16+sz > len(d) {
				break
			}
			if hit >= off && hit < off+16+sz {
				if len(d) > off+16 {
					markerHist[d[off+16]]++
				}
				break
			}
			off += 16 + sz
		}
	}
	fmt.Print("DIAG marqueur du paquet contenant les fenêtres killdeser: ")
	for mk, n := range markerHist {
		fmt.Printf("0x%02X:%d ", mk, n)
	}
	fmt.Println()
	var kdMin, kdMax, kcMin, kcMax uint64 = 1 << 63, 0, 1 << 63, 0
	for _, r := range kds {
		if r.rdtsc < kdMin {
			kdMin = r.rdtsc
		}
		if r.rdtsc > kdMax {
			kdMax = r.rdtsc
		}
	}
	for _, c := range kcs {
		if c.rdtsc < kcMin {
			kcMin = c.rdtsc
		}
		if c.rdtsc > kcMax {
			kcMax = c.rdtsc
		}
	}
	fmt.Printf("DIAG rdtsc killdeser=[%d..%d] consumer=[%d..%d]\n", kdMin, kdMax, kcMin, kcMax)

	_ = opkts
	readOptAt := func(pl []byte, bp int) (int, int) {
		if bp < 0 || bp>>3 >= len(pl) {
			return -2, bp
		}
		if bitsAt(pl, bp, 1) == 0 {
			return int(bitsAt(pl, bp+1, 5)), bp + 6
		}
		return -1, bp + 1
	}
	// SWEEP du delta autour du curseur (le buffer peut inclure/exclure l'en-tête). Pour chaque delta,
	// décode field0/field1 = readOpt au curseur dans le paquet contenant, lie au consumer par rdtsc,
	// et mesure la pureté des confusions. Le meilleur (delta, champ, rôle) tranche.
	type cf struct{ m map[int]map[int]int }
	pur := func(c map[int]map[int]int) int {
		s := 0
		for _, row := range c {
			b := 0
			for _, v := range row {
				if v > b {
					b = v
				}
			}
			s += b
		}
		return s
	}
	bestDelta, bestScore, bestLabel := 0, -1, ""
	var bestConf map[int]map[int]int
	confAtk := map[int]map[int]int{} // attaquant du dégât (bit36>>1) × tueur consumer
	for delta := -8; delta <= 8; delta++ {
		f0k, f0v, f1k, f1v := &cf{map[int]map[int]int{}}, &cf{map[int]map[int]int{}}, &cf{map[int]map[int]int{}}, &cf{map[int]map[int]int{}}
		n := 0
		for _, r := range kds {
			sig := r.win[:16]
			hit, hitCh := -1, -1
			for ci, d := range chunks {
				if o := indexBytes(d, sig); o >= 0 {
					hit, hitCh = o, ci
					break
				}
			}
			if hit < 0 {
				continue
			}
			d := chunks[hitCh]
			var payload []byte
			off := 0
			for off+16 <= len(d) {
				sz := int(binary.LittleEndian.Uint32(d[off+4:]))
				if sz <= 0 || off+16+sz > len(d) {
					break
				}
				if hit >= off && hit < off+16+sz {
					payload = d[off+16 : off+16+sz]
					break
				}
				off += 16 + sz
			}
			if payload == nil {
				continue
			}
			f0, bp := readOptAt(payload, int(r.cursor)+delta)
			f1, _ := readOptAt(payload, bp)
			var bestDT uint64 = 1 << 63
			bk, bv := -1, -1
			for _, c := range kcs {
				if c.kil < 0 || c.vic < 0 {
					continue
				}
				dt := c.rdtsc - r.rdtsc
				if c.rdtsc < r.rdtsc {
					dt = r.rdtsc - c.rdtsc
				}
				if dt < bestDT {
					bestDT, bk, bv = dt, c.kil, c.vic
				}
			}
			if bk < 0 || f0 < 0 {
				continue
			}
			n++
			add := func(c *cf, k, v int) {
				if c.m[k] == nil {
					c.m[k] = map[int]int{}
				}
				c.m[k][v]++
			}
			add(f0k, f0, bk)
			add(f0v, f0, bv)
			if f1 >= 0 {
				add(f1k, f1, bk)
				add(f1v, f1, bv)
			}
			if delta == 0 { // attaquant du dégât fatal (bit36>>1) vs tueur — testé une fois
				atk := int(bitsAt(payload, 36, 5)) >> 1
				if confAtk[atk] == nil {
					confAtk[atk] = map[int]int{}
				}
				confAtk[atk][bk]++
			}
		}
		for _, cand := range []struct {
			lbl string
			c   *cf
		}{{"field0×tueur", f0k}, {"field0×victime", f0v}, {"field1×tueur", f1k}, {"field1×victime", f1v}} {
			p := pur(cand.c.m)
			if p > bestScore {
				bestScore, bestDelta, bestLabel, bestConf = p, delta, cand.lbl, cand.c.m
			}
		}
		if delta == 0 {
			fmt.Printf("delta=0 (n=%d) : field0×tueur=%d field0×victime=%d field1×tueur=%d field1×victime=%d\n",
				n, pur(f0k.m), pur(f0v.m), pur(f1k.m), pur(f1v.m))
		}
	}
	fmt.Printf(">>> MEILLEUR : %s à delta=%d, pureté=%d\n", bestLabel, bestDelta, bestScore)
	printConf("meilleure confusion", bestConf)
	na := 0
	for _, row := range confAtk {
		for _, v := range row {
			na += v
		}
	}
	fmt.Printf(">>> attaquant dégât (bit36>>1) × TUEUR consumer : pureté %d/%d = %.0f%%\n",
		pur(confAtk), na, float64(pur(confAtk))*100/float64(max(na, 1)))
	printConf("attaquant dégât -> tueur", confAtk)
}

// runChunk27 : TEST DÉCISIF via chunk_27 (kill feed offline, temps-jeu — même horloge que le frame
// ts, contrairement à la capture live). Rang-aligne les kills 0xE6 (par ts) sur les events KILL
// (killer XUID) ET DEATH (victim XUID) de chunk_27 (par TimeMS), et compare la pureté des confusions
// field0×killerXUID vs field0×victimXUID. Pas de mapping requis (XUID = classe). La donnée décide.
func runChunk27(m string) {
	cache := root + "/" + m
	type ek struct {
		f0 int
		ts uint64
	}
	var okills []ek
	for ch := 0; ch <= 41; ch++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, ch))
		off := 0
		for off+16 <= len(d) {
			typ := binary.LittleEndian.Uint16(d[off:])
			sz := int(binary.LittleEndian.Uint32(d[off+4:]))
			ts := binary.LittleEndian.Uint64(d[off+8:])
			if sz <= 0 || off+16+sz > len(d) {
				break
			}
			pl := d[off+16 : off+16+sz]
			off += 16 + sz
			if typ != 0 || len(pl) == 0 || pl[0] != 0xE6 {
				continue
			}
			f0 := -1
			if bitsAt(pl, 10, 1) == 0 {
				f0 = int(bitsAt(pl, 11, 5))
			}
			okills = append(okills, ek{f0, ts})
		}
	}
	sort.Slice(okills, func(i, j int) bool { return okills[i].ts < okills[j].ts })

	type ev struct {
		x uint64
		t int
	}
	var kills, deaths []ev
	for ch := 41; ch >= 18; ch-- {
		b := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, ch))
		if len(b) == 0 {
			continue
		}
		evs, _ := analysis.ParseHighlightEvents(b, 0)
		var kk, dd []ev
		for _, e := range evs {
			switch e.EventType {
			case analysis.EventTypeKill:
				kk = append(kk, ev{e.XUID, e.TimeMS})
			case analysis.EventTypeDeath:
				dd = append(dd, ev{e.XUID, e.TimeMS})
			}
		}
		if len(kk) > len(kills) {
			kills, deaths = kk, dd
		}
	}
	sort.Slice(kills, func(i, j int) bool { return kills[i].t < kills[j].t })
	sort.Slice(deaths, func(i, j int) bool { return deaths[i].t < deaths[j].t })
	fmt.Printf("=== CHUNK27 %s : %d kills 0xE6 | %d KILL events | %d DEATH events ===\n", m, len(okills), len(kills), len(deaths))

	// rang-alignement + confusion field0 × killerXUID et field0 × victimXUID.
	confPurity := func(evts []ev) (int, int) {
		n := len(okills)
		if len(evts) < n {
			n = len(evts)
		}
		conf := map[int]map[uint64]int{}
		for i := 0; i < n; i++ {
			if okills[i].f0 < 0 {
				continue
			}
			if conf[okills[i].f0] == nil {
				conf[okills[i].f0] = map[uint64]int{}
			}
			conf[okills[i].f0][evts[i].x]++
		}
		s, tot := 0, 0
		for _, row := range conf {
			bestv := 0
			for _, v := range row {
				tot += v
				if v > bestv {
					bestv = v
				}
			}
			s += bestv
		}
		return s, tot
	}
	pk, tk := confPurity(kills)
	pd, td := confPurity(deaths)
	fmt.Printf(">>> field0 × KILLER (events KILL)  : pureté %d/%d = %.0f%%\n", pk, tk, float64(pk)*100/float64(max(tk, 1)))
	fmt.Printf(">>> field0 × VICTIM (events DEATH)  : pureté %d/%d = %.0f%%\n", pd, td, float64(pd)*100/float64(max(td, 1)))
	if pk > pd {
		fmt.Println("=> field0 = KILLER (peut battre le warp via arme par-tueur same-clock)")
	} else if pd > pk {
		fmt.Println("=> field0 = VICTIM (arme reste via 0xd2 fatal)")
	} else {
		fmt.Println("=> INCONCLUSIF (pureté égale)")
	}
}

// runWarp : TEST DÉCISIF tueur-vs-victime. Aligne les kills 0xE6 (frame ts) sur les kills live
// (capture tsc) via un fit ts→tsc dérivé des flux de dégâts (0xd2 offline vs dmg live, appariés
// par quantiles → robuste aux comptes différents). Puis, par-kill, construit la confusion field0 ×
// tueur-live ET field0 × victime-live. La pureté LA PLUS HAUTE tranche ce qu'est field0. On
// n'assume rien : la donnée décide.
func runWarp(m string, h32 map[uint32]string) {
	cache := root + "/" + m
	type ekill struct {
		f0 int
		ts uint64
	}
	var okills []ekill
	var odmg []uint64
	for ch := 0; ch <= 41; ch++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, ch))
		off := 0
		for off+16 <= len(d) {
			typ := binary.LittleEndian.Uint16(d[off:])
			sz := int(binary.LittleEndian.Uint32(d[off+4:]))
			ts := binary.LittleEndian.Uint64(d[off+8:])
			if sz <= 0 || off+16+sz > len(d) {
				break
			}
			pl := d[off+16 : off+16+sz]
			off += 16 + sz
			if typ != 0 || len(pl) == 0 {
				continue
			}
			switch pl[0] {
			case 0xE6:
				bp := 10
				present := bitsAt(pl, bp, 1) == 0
				bp++
				f0 := -1
				if present {
					f0 = int(bitsAt(pl, bp, 5))
				}
				okills = append(okills, ekill{f0, ts})
			case 0xd2:
				odmg = append(odmg, ts)
			}
		}
	}
	dd, e1 := os.ReadFile(liveDir + "/" + m + "_dmg.bin")
	kc, e2 := os.ReadFile(liveDir + "/" + m + "_kill.bin")
	if e1 != nil || e2 != nil {
		fmt.Printf("pas de capture live pour %s\n", m)
		return
	}
	var ldmg []uint64
	for o := 0; o+32 <= len(dd); o += 32 {
		if idxD(binary.LittleEndian.Uint32(dd[o:])) < 0 {
			continue
		}
		ldmg = append(ldmg, uint64(binary.LittleEndian.Uint32(dd[o+20:]))<<32|uint64(binary.LittleEndian.Uint32(dd[o+16:])))
	}
	type lkill struct {
		kil, vic int
		tsc      uint64
	}
	var lkAll []lkill
	for o := 0; o+16 <= len(kc); o += 16 {
		vi := idxK(binary.LittleEndian.Uint32(kc[o:]))
		ki := idxK(binary.LittleEndian.Uint32(kc[o+4:]))
		if ki < 0 || vi < 0 {
			continue
		}
		lkAll = append(lkAll, lkill{ki, vi, uint64(binary.LittleEndian.Uint32(kc[o+12:]))<<32 | uint64(binary.LittleEndian.Uint32(kc[o+8:]))})
	}
	sort.Slice(lkAll, func(i, j int) bool { return lkAll[i].tsc < lkAll[j].tsc })
	var lks []lkill
	for _, k := range lkAll {
		dup := false
		for j := len(lks) - 1; j >= 0 && k.tsc-lks[j].tsc < 3000000; j-- {
			if lks[j].kil == k.kil && lks[j].vic == k.vic {
				dup = true
				break
			}
		}
		if !dup {
			lks = append(lks, k)
		}
	}
	sort.Slice(okills, func(i, j int) bool { return okills[i].ts < okills[j].ts })
	sort.Slice(odmg, func(i, j int) bool { return odmg[i] < odmg[j] })
	sort.Slice(ldmg, func(i, j int) bool { return ldmg[i] < ldmg[j] })
	fmt.Printf("=== WARP %s : %d kills 0xE6, %d dmg offline | %d kills live, %d dmg live ===\n", m, len(okills), len(odmg), len(lks), len(ldmg))

	// fit tsc = a*ts + b par quantiles des flux de dégâts (least squares sur ~40 points).
	var sx, sy, sxx, sxy, n float64
	for q := 1; q < 40; q++ {
		ox := float64(odmg[q*len(odmg)/40])
		ly := float64(ldmg[q*len(ldmg)/40])
		sx += ox
		sy += ly
		sxx += ox * ox
		sxy += ox * ly
		n++
	}
	a := (n*sxy - sx*sy) / (n*sxx - sx*sx)
	b := (sy - a*sx) / n
	fmt.Printf("fit tsc = %.6g*ts + %.6g\n", a, b)
	// contrôle de qualité du fit : résidu médian des kills mappés vs le kill live le plus proche,
	// comparé à l'espacement médian des kills live. résidu << espacement = fit exploitable.
	var res []float64
	for _, k := range okills {
		pred := a*float64(k.ts) + b
		best := math.MaxFloat64
		for _, l := range lks {
			if d := math.Abs(float64(l.tsc) - pred); d < best {
				best = d
			}
		}
		res = append(res, best)
	}
	sort.Float64s(res)
	var gaps []float64
	for i := 1; i < len(lks); i++ {
		gaps = append(gaps, float64(lks[i].tsc-lks[i-1].tsc))
	}
	sort.Float64s(gaps)
	medRes, medGap := res[len(res)/2], gaps[len(gaps)/2]
	fmt.Printf("qualité fit : résidu médian=%.3g | espacement médian kills=%.3g | ratio=%.2f %s\n",
		medRes, medGap, medRes/medGap, map[bool]string{true: "(EXPLOITABLE)", false: "(FIT MAUVAIS — test invalide)"}[medRes < medGap])

	// aligne chaque kill 0xE6 sur le kill live le plus proche en tsc prédit ; confusion.
	confKil, confVic := map[int]map[int]int{}, map[int]map[int]int{}
	matched := 0
	for _, k := range okills {
		if k.f0 < 0 {
			continue
		}
		pred := a*float64(k.ts) + b
		best, bi := math.MaxFloat64, -1
		for i, l := range lks {
			d := math.Abs(float64(l.tsc) - pred)
			if d < best {
				best, bi = d, i
			}
		}
		if bi < 0 {
			continue
		}
		matched++
		if confKil[k.f0] == nil {
			confKil[k.f0] = map[int]int{}
			confVic[k.f0] = map[int]int{}
		}
		confKil[k.f0][lks[bi].kil]++
		confVic[k.f0][lks[bi].vic]++
	}
	pur := func(c map[int]map[int]int) int {
		s := 0
		for _, row := range c {
			bestv := 0
			for _, v := range row {
				if v > bestv {
					bestv = v
				}
			}
			s += bestv
		}
		return s
	}
	pk, pv := pur(confKil), pur(confVic)
	fmt.Printf("field0 aligné sur %d kills live\n", matched)
	fmt.Printf(">>> field0 × TUEUR live   : pureté %d/%d = %.0f%%\n", pk, matched, float64(pk)*100/float64(max(matched, 1)))
	fmt.Printf(">>> field0 × VICTIME live : pureté %d/%d = %.0f%%\n", pv, matched, float64(pv)*100/float64(max(matched, 1)))
	if pk > pv {
		fmt.Println("=> field0 = TUEUR (arme par-tueur same-clock possible -> peut battre le warp)")
		printConf("field0 -> tueur live", confKil)
	} else {
		fmt.Println("=> field0 = VICTIME (arme reste via 0xd2 fatal)")
		printConf("field0 -> victime live", confVic)
	}
}

// runDecode : décode le kill-event via le FRAMING RÉEL du dispatcher d'événements (Ghidra
// FUN_14080a9d4) : [R7 type][3× R1 gate (3-loop, vtable[0x58] ne lit rien)][déser]. Le déser
// (DeserKillEvent_ECS) : chaque champ entité = FUN_1407f2058 = R1 gate + optR5. Donc tueur =
// readOpt @ bit preSkip (défaut 10 = 7+3). Valide en same-clock (0xd2 fatal) + distribution live.
// Usage : tmp_kwval <film> decode [preSkip]
func runDecode(m string, h32 map[uint32]string) {
	preSkip := 10
	if len(os.Args) > 3 {
		preSkip, _ = strconv.Atoi(os.Args[3])
	}
	cache := root + "/" + m
	type kev struct {
		f0, f1, f4     int // les 3 références entité du déser (DeserKillEvent_ECS)
		killer, victim int
		ts             uint64
	}
	var kills []kev
	var dmgs []dmgEv
	for ch := 0; ch <= 41; ch++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, ch))
		off := 0
		for off+16 <= len(d) {
			typ := binary.LittleEndian.Uint16(d[off:])
			sz := int(binary.LittleEndian.Uint32(d[off+4:]))
			ts := binary.LittleEndian.Uint64(d[off+8:])
			if sz <= 0 || off+16+sz > len(d) {
				break
			}
			pl := d[off+16 : off+16+sz]
			off += 16 + sz
			if typ != 0 || len(pl) == 0 {
				continue
			}
			switch pl[0] {
			case 0xE6:
				bp := preSkip
				readOpt := func() int { // FUN_1407f2058 : R1 gate (présent si 0) + R5
					present := bitsAt(pl, bp, 1) == 0
					bp++
					if present {
						v := int(bitsAt(pl, bp, 5))
						bp += 5
						return v
					}
					return -1
				}
				// grammaire DeserKillEvent_ECS : [readOpt f0][readOpt f1][R32 f2][R1 f3][readOpt f4][R32 f5]
				f0 := readOpt()
				f1 := readOpt()
				bp += 32 // f2 R32
				bp++     // f3 R1
				f4 := readOpt()
				kills = append(kills, kev{f0, f1, f4, f0, f4, ts}) // hyp : tueur=f0, victime=f4
			case 0xd2:
				bp := 41
				if bitsAt(pl, bp, 1) == 1 {
					bp++
				} else {
					bp += 3
				}
				f := uint32(bitsAt(pl, bp, 32))
				suf := uint32(bitsAt(pl, bp+32, 32))
				atk := int(bitsAt(pl, 36, 5)) >> 1
				nm := fmt.Sprintf("cause-%08X", suf)
				fa := false
				if n, ok := h32[f]; ok && suf == sfx {
					nm, fa = n, true
				}
				dmgs = append(dmgs, dmgEv{atk, nm, ts, fa})
			}
		}
	}
	sort.Slice(dmgs, func(i, j int) bool { return dmgs[i].ts < dmgs[j].ts })
	sort.Slice(kills, func(i, j int) bool { return kills[i].ts < kills[j].ts }) // ordre chrono pour l'alignement
	fmt.Printf("=== DECODE framing réel %s (preSkip=%d) : %d kills, %d dégâts ===\n", m, preSkip, len(kills), len(dmgs))

	f0d, f1d, f4d := map[int]int{}, map[int]int{}, map[int]int{}
	confSC := map[int]map[int]int{}
	for _, k := range kills {
		f0d[k.f0]++
		f1d[k.f1]++
		f4d[k.f4]++
		fatal := -1
		for _, dg := range dmgs {
			if dg.ts <= k.ts {
				fatal = dg.atk
			} else {
				break
			}
		}
		if confSC[k.killer] == nil {
			confSC[k.killer] = map[int]int{}
		}
		confSC[k.killer][fatal]++
	}
	fmt.Printf("field0 offline: %s\n", sortedCounts(f0d))
	fmt.Printf("field1 offline: %s\n", sortedCounts(f1d))
	fmt.Printf("field4 offline: %s\n", sortedCounts(f4d))
	if live, ok := decodeLive(m, h32); ok {
		lk, lv := map[int]int{}, map[int]int{}
		for _, a := range live {
			lk[a.killer]++
			lv[a.victim]++
		}
		fmt.Printf("tueurs LIVE   : %s\n", sortedCounts(lk))
		fmt.Printf("victimes LIVE : %s\n", sortedCounts(lv))
		nl := len(live)
		fmt.Printf(">>> ACCORD DISTRIBUTION vs TUEUR live  : f0=%.0f%% f1=%.0f%% f4=%.0f%%\n",
			float64(msetInter(f0d, lk))*100/float64(max(nl, 1)),
			float64(msetInter(f1d, lk))*100/float64(max(nl, 1)),
			float64(msetInter(f4d, lk))*100/float64(max(nl, 1)))
		fmt.Printf(">>> ACCORD DISTRIBUTION vs VICTIME live : f0=%.0f%% f1=%.0f%% f4=%.0f%%\n",
			float64(msetInter(f0d, lv))*100/float64(max(nl, 1)),
			float64(msetInter(f1d, lv))*100/float64(max(nl, 1)),
			float64(msetInter(f4d, lv))*100/float64(max(nl, 1)))
		// confusion offline vs LIVE, alignée par rang chrono (le bon oracle). Bijection nette =
		// framing CONFIRMÉ + table index-local -> idx live.
		n := len(kills)
		if len(live) < n {
			n = len(live)
		}
		confK, confV := map[int]map[int]int{}, map[int]map[int]int{}
		for i := 0; i < n; i++ {
			if confK[kills[i].killer] == nil {
				confK[kills[i].killer] = map[int]int{}
			}
			confK[kills[i].killer][live[i].killer]++
			if confV[kills[i].victim] == nil {
				confV[kills[i].victim] = map[int]int{}
			}
			confV[kills[i].victim][live[i].victim]++
		}
		printConf("LIVE (rang) tueur-local -> tueur live", confK)
		printConf("LIVE (rang) victime-local -> victime live", confV)
	}
	printConf("SAME-CLOCK tueur(readOpt) -> attaquant 0xd2 fatal", confSC)
}

// msetInter : intersection de multisets de comptes (permutation-invariant) — trie les comptes
// desc des deux côtés et somme les min par rang. Exclut les index négatifs (absent/-1).
func msetInter(a, b map[int]int) int {
	var av, bv []int
	for k, v := range a {
		if k >= 0 {
			av = append(av, v)
		}
	}
	for k, v := range b {
		if k >= 0 {
			bv = append(bv, v)
		}
	}
	sort.Sort(sort.Reverse(sort.IntSlice(av)))
	sort.Sort(sort.Reverse(sort.IntSlice(bv)))
	s := 0
	for i := 0; i < len(av) && i < len(bv); i++ {
		if av[i] < bv[i] {
			s += av[i]
		} else {
			s += bv[i]
		}
	}
	return s
}

func sortedCounts(d map[int]int) string {
	type kv struct{ k, v int }
	var a []kv
	for k, v := range d {
		a = append(a, kv{k, v})
	}
	sort.Slice(a, func(i, j int) bool { return a[i].v > a[j].v })
	s := ""
	for _, x := range a {
		s += fmt.Sprintf("%d:%d ", x.k, x.v)
	}
	return s
}

// runKillEvents : valide la grammaire du kill-event sur l'ORACLE CE BRUT (killevents_sample.txt,
// capture du flux 9b191a7f — contient le suffixe cause 592cf3e9 propre à ce match). Chaque ligne
// "KILL bitpos bit%8 hex24" = la fenêtre de bits du reader À L'ENTRÉE du deser. Par la grammaire
// Ghidra : tueur=R5@(bit%8), victime=R5@(bit%8+5). On confronte la distribution des tueurs (index
// local) à la vérité-terrain 9b191a7f_kill.bin. Si les multisets triés coïncident, la grammaire est
// CONFIRMEE sur le flux réel — et le bitpos VARIABLE (578, 2756, 158…) confirme la position variable.
func runKillEvents(m string, h32 map[uint32]string) {
	raw, err := os.ReadFile(kevPath)
	if err != nil {
		fmt.Printf("killevents_sample.txt introuvable: %v\n", err)
		return
	}
	kdist, vdist := map[int]int{}, map[int]int{}
	var bitposEx []int
	nKill := 0
	for _, ln := range strings.Split(string(raw), "\n") {
		f := strings.Fields(ln)
		if len(f) < 4 || f[0] != "KILL" {
			continue
		}
		bitmod, _ := strconv.Atoi(f[2])
		win, e := hex.DecodeString(f[3])
		if e != nil || len(win) < 4 {
			continue
		}
		kdist[int(bitsAt(win, bitmod, 5))]++
		vdist[int(bitsAt(win, bitmod+5, 5))]++
		if bp, e2 := strconv.Atoi(f[1]); e2 == nil && len(bitposEx) < 12 {
			bitposEx = append(bitposEx, bp)
		}
		nKill++
	}
	fmt.Printf("=== ORACLE CE BRUT killevents_sample.txt : %d lignes KILL ===\n", nKill)
	fmt.Printf("bitpos (12 premiers, VARIABLES) : %v\n", bitposEx)
	fmt.Printf("tueurs  (index local R5@bit%%8) : %s\n", sortedCounts(kdist))
	fmt.Printf("victimes                        : %s\n", sortedCounts(vdist))
	if live, ok := decodeLive(m, h32); ok {
		lk, lv := map[int]int{}, map[int]int{}
		for _, a := range live {
			lk[a.killer]++
			lv[a.victim]++
		}
		fmt.Printf("tueurs  LIVE (%s_kill.bin, idx) : %s\n", m, sortedCounts(lk))
		fmt.Printf("victimes LIVE                   : %s\n", sortedCounts(lv))
		fmt.Println("=> multisets triés coincidents ⇒ grammaire R5@bit%8 CONFIRMEE sur le flux brut.")
	}
}

func printConf(title string, conf map[int]map[int]int) {
	var locs []int
	for l := range conf {
		locs = append(locs, l)
	}
	sort.Ints(locs)
	fmt.Printf("\n--- %s ---\n", title)
	pureTot, tot := 0, 0
	for _, l := range locs {
		best, bi, t := -1, -1, 0
		for iv, c := range conf[l] {
			t += c
			if c > best {
				best, bi = c, iv
			}
		}
		pureTot += best
		tot += t
		fmt.Printf("  local %2d -> %2d  (%d/%d = %.0f%%)  %v\n", l, bi, best, t, float64(best)*100/float64(max(t, 1)), conf[l])
	}
	fmt.Printf("  pureté globale = %d/%d = %.0f%%\n", pureTot, tot, float64(pureTot)*100/float64(max(tot, 1)))
}

// runAlign : valide le champ tueur 0xE6 (R5@S) en SAME-CLOCK, sans la fragile capture live 2-horloges.
// Le tueur 0xE6 (index local) doit matcher l'attaquant du 0xd2 fatal (dernier avant l'instant du kill,
// même horloge) == idx live. Confusion nette → S=83 est bien le tueur ET donne la table index-local ->
// slot 0xd2. On confronte aussi à la capture live (aligné par rang) pour référence.
func runAlign(m string, h32 map[uint32]string) {
	const S = 83
	kills, dmgs := collectRaw(m, h32, S)
	fmt.Printf("=== ALIGN %s : %d kills 0xE6, %d dégâts 0xd2 (S=%d) ===\n", m, len(kills), len(dmgs), S)

	// (a) SAME-CLOCK : confusion tueur-local 0xE6 × attaquant du 0xd2 fatal (fiable, 1 horloge).
	confSC := map[int]map[int]int{}
	for _, k := range kills {
		fatal := -1
		for _, dg := range dmgs { // dmgs triés asc → le dernier avec ts<=k.ts gagne
			if dg.ts <= k.ts {
				fatal = dg.atk
			} else {
				break
			}
		}
		if confSC[k.killer] == nil {
			confSC[k.killer] = map[int]int{}
		}
		confSC[k.killer][fatal]++
	}
	printConf("SAME-CLOCK  tueur-local 0xE6 -> attaquant 0xd2 fatal", confSC)

	// (b) LIVE : confusion tueur-local 0xE6 × tueur live (aligné par rang, fragile).
	if live, ok := decodeLive(m, h32); ok {
		n := len(kills)
		if len(live) < n {
			n = len(live)
		}
		confLV := map[int]map[int]int{}
		for i := 0; i < n; i++ {
			if confLV[kills[i].killer] == nil {
				confLV[kills[i].killer] = map[int]int{}
			}
			confLV[kills[i].killer][live[i].killer]++
		}
		printConf("LIVE (rang) tueur-local 0xE6 -> tueur live", confLV)
	}

	// (c) PRÉFIXE VARIABLE ? Par frame, cherche le S où R5@S == attaquant 0xd2 fatal (oracle
	// same-clock, valable pour les hitscan). Histogramme des S qui matchent : bande étroite =
	// préfixe court variable (crackable) ; éparpillé = pleinement variable (besoin du décodeur frame).
	e6 := collectE6(m)
	sHist := map[int]int{}
	matched := 0
	for i, pl := range e6 { // e6 et kills sont dans le même ordre de collecte
		if i >= len(kills) {
			break
		}
		fatal := -1
		for _, dg := range dmgs {
			if dg.ts <= kills[i].ts {
				fatal = dg.atk
			} else {
				break
			}
		}
		if fatal < 0 {
			continue
		}
		for s := 40; s <= 160; s++ {
			if int(bitsAt(pl, s, 5)) == fatal {
				sHist[s]++
				matched++
				break
			}
		}
	}
	fmt.Printf("\n--- PRÉFIXE : S où R5@S==attaquant 0xd2 fatal (par frame, %d/%d matchent) ---\n", matched, len(e6))
	var ss []int
	for s := range sHist {
		ss = append(ss, s)
	}
	sort.Ints(ss)
	for _, s := range ss {
		if sHist[s] >= 2 {
			fmt.Printf("  S=%d : %d frames\n", s, sHist[s])
		}
	}
}
