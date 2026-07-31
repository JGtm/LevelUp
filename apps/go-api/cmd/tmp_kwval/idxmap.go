package main

// idxmap.go — DIAGNOSTIC (mode "idxmap") : caractérise EMPIRIQUEMENT le mapping
// index-local-du-kill-event -> joueur pour un film à capture live (9b191a7f).
//
// Trois flux comparés :
//   A) 0xE6 R5@83  (decodeOffline/collectRaw) : killer=R5@83, victim=R5@88 — lecture directe 5 bits.
//   B) allRawKE    (keReadOpt) : R1-gate + R5, cap <16 par validKE — le déser "readOpt".
//   L) live kill.bin : vérité terrain, slots 0..7 via idxK=(entityID-0xE1500000)/0x10002.
//
// But : (a) ENSEMBLE des valeurs d'index produites, (b) relation aux 8 joueurs / entityID live,
// (c) FORMULE fixe index-local -> slot si elle existe. Résout la permutation par recouvrement de
// paires (tueur,victime) contre la vérité live (bestInjectionOverlap), sans corrélation d'horloge.

import (
	"encoding/binary"
	"fmt"
	"os"
	"sort"
)

// liveKillPairs : paires (killer_slot, victim_slot) de la capture live, dédupliquées (~3ms), + set de slots.
func liveKillPairs(m string) (pairs [][2]int, slots map[int]int, ok bool) {
	kc, err := os.ReadFile(liveDir + "/" + m + "_kill.bin")
	if err != nil {
		return nil, nil, false
	}
	type lk struct {
		kil, vic int
		tsc      uint64
	}
	var all []lk
	for o := 0; o+16 <= len(kc); o += 16 {
		vi := idxK(binary.LittleEndian.Uint32(kc[o:]))
		ki := idxK(binary.LittleEndian.Uint32(kc[o+4:]))
		if ki < 0 || vi < 0 {
			continue
		}
		tsc := uint64(binary.LittleEndian.Uint32(kc[o+12:]))<<32 | uint64(binary.LittleEndian.Uint32(kc[o+8:]))
		all = append(all, lk{ki, vi, tsc})
	}
	sort.Slice(all, func(i, j int) bool { return all[i].tsc < all[j].tsc })
	slots = map[int]int{}
	var dd []lk
	for _, k := range all {
		dup := false
		for j := len(dd) - 1; j >= 0 && k.tsc-dd[j].tsc < 3000000; j-- {
			if dd[j].kil == k.kil && dd[j].vic == k.vic {
				dup = true
				break
			}
		}
		if !dup {
			dd = append(dd, k)
		}
	}
	for _, k := range dd {
		pairs = append(pairs, [2]int{k.kil, k.vic})
		slots[k.kil]++
		slots[k.vic]++
	}
	return pairs, slots, true
}

// idxHist : distribution (valeur d'index -> nombre d'occurrences) sur killer ET victime.
func idxHistFromPairs(kk, vv []int) (map[int]int, map[int]int) {
	hk, hv := map[int]int{}, map[int]int{}
	for _, x := range kk {
		hk[x]++
	}
	for _, x := range vv {
		hv[x]++
	}
	return hk, hv
}

func sortedKeys(h map[int]int) []int {
	var ks []int
	for k := range h {
		ks = append(ks, k)
	}
	sort.Ints(ks)
	return ks
}

func fmtIntHist(h map[int]int) string {
	s := ""
	for _, k := range sortedKeys(h) {
		s += fmt.Sprintf("%d:%d ", k, h[k])
	}
	return s
}

// bestInjectionToSlots : réutilise bestInjectionOverlap pour trouver la carte index->slot (0..7)
// maximisant le recouvrement multiset des paires décodées avec les paires live. Renvoie (overlap,
// carte, total paires décodées mappables). di = indices offline retenus (<=8 pour permutation exacte).
func bestInjectionToSlots(pairs [][2]int, di []int, liveCnt map[[2]uint64]int) (int, map[int]uint64) {
	// adapte les paires offline au type pipeKill (seuls killer/victim comptent) pour overlapForMap.
	dedup := make([]pipeKill, 0, len(pairs))
	for _, p := range pairs {
		dedup = append(dedup, pipeKill{killer: p[0], victim: p[1]})
	}
	cx := []uint64{0, 1, 2, 3, 4, 5, 6, 7}
	return bestInjectionOverlap(dedup, di, cx, liveCnt)
}

// runIdxMap : point d'entrée du mode "idxmap".
func runIdxMap(m string, h32 map[uint32]string) {
	// --- Flux L : vérité live ---
	livePairs, liveSlots, ok := liveKillPairs(m)
	if !ok {
		fmt.Printf("PAS de capture live pour %s (kill.bin absent) — mapping non vérifiable.\n", m)
		return
	}
	liveCnt := map[[2]uint64]int{}
	for _, p := range livePairs {
		liveCnt[[2]uint64{uint64(p[0]), uint64(p[1])}]++
	}
	fmt.Printf("=== IDXMAP %s ===\n", m)
	fmt.Printf("[LIVE] %d kills dédup | slots présents (idxK, occurrences k+v) : %s\n", len(livePairs), fmtIntHist(liveSlots))

	// --- Flux A : 0xE6 R5@83 (decodeOffline/collectRaw) ---
	killsA, _ := collectRaw(m, h32, 83)
	var aK, aV []int
	var pairsA [][2]int
	for _, k := range killsA {
		aK = append(aK, k.killer)
		aV = append(aV, k.victim)
		pairsA = append(pairsA, [2]int{k.killer, k.victim})
	}
	hkA, hvA := idxHistFromPairs(aK, aV)
	unionA := map[int]int{}
	for k, v := range hkA {
		unionA[k] += v
	}
	for k, v := range hvA {
		unionA[k] += v
	}
	fmt.Printf("\n[A: 0xE6 R5@83] %d kills\n", len(killsA))
	fmt.Printf("  killer idx : %s\n", fmtIntHist(hkA))
	fmt.Printf("  victim idx : %s\n", fmtIntHist(hvA))
	fmt.Printf("  UNION set  : %s\n", fmtIntHist(unionA))

	// --- Flux B : allRawKE (keReadOpt) ---
	rawB := allRawKE(m, h32)
	var bK, bV []int
	for _, e := range rawB {
		bK = append(bK, e.killer)
		bV = append(bV, e.victim)
	}
	hkB, hvB := idxHistFromPairs(bK, bV)
	unionB := map[int]int{}
	for k, v := range hkB {
		unionB[k] += v
	}
	for k, v := range hvB {
		unionB[k] += v
	}
	fmt.Printf("\n[B: allRawKE keReadOpt] %d kill-events bruts (candidats, non dédup)\n", len(rawB))
	fmt.Printf("  killer idx : %s\n", fmtIntHist(hkB))
	fmt.Printf("  victim idx : %s\n", fmtIntHist(hvB))
	fmt.Printf("  UNION set  : %s\n", fmtIntHist(unionB))
	// allRawKE full-scan
	kfFullScan = true
	rawBF := allRawKE(m, h32)
	kfFullScan = false
	var bfK, bfV []int
	for _, e := range rawBF {
		bfK = append(bfK, e.killer)
		bfV = append(bfV, e.victim)
	}
	hkBF, hvBF := idxHistFromPairs(bfK, bfV)
	unionBF := map[int]int{}
	for k, v := range hkBF {
		unionBF[k] += v
	}
	for k, v := range hvBF {
		unionBF[k] += v
	}
	fmt.Printf("[B-full: allRawKE full-scan] %d kill-events\n", len(rawBF))
	fmt.Printf("  UNION set  : %s\n", fmtIntHist(unionBF))

	// --- MAPPING A -> slots live (permutation exacte sur les 8 indices dominants) ---
	fmt.Printf("\n--- MAPPING A (0xE6 R5@83) -> slots live (recouvrement de paires) ---\n")
	solveMap("A", pairsA, unionA, liveCnt, len(livePairs))

	// --- MAPPING B (allRawKE dédup par paire+ts) -> slots live ---
	// déduplique allRawKE en paires (fenêtre 3ms) pour retirer le bruit multi-candidat.
	pairsBdd := dedupPairsTs(rawB)
	unionBdd := map[int]int{}
	for _, p := range pairsBdd {
		unionBdd[p[0]]++
		unionBdd[p[1]]++
	}
	fmt.Printf("\n--- MAPPING B (allRawKE dédup paire/3ms, %d paires) -> slots live ---\n", len(pairsBdd))
	solveMap("B", pairsBdd, unionBdd, liveCnt, len(livePairs))
}

// dedupPairsTs : déduplique les pipeKill par (killer,victim) dans une fenêtre 3ms, retourne les paires.
func dedupPairsTs(raw []pipeKill) [][2]int {
	byPair := map[[2]int][]uint64{}
	for _, e := range raw {
		byPair[[2]int{e.killer, e.victim}] = append(byPair[[2]int{e.killer, e.victim}], e.ts)
	}
	var out [][2]int
	for p, tss := range byPair {
		sort.Slice(tss, func(i, j int) bool { return tss[i] < tss[j] })
		var last uint64
		for i, t := range tss {
			if i == 0 || t-last >= 3000000 {
				out = append(out, p)
				last = t
			}
		}
	}
	return out
}

// solveMap : pour un flux de paires (index) donné, résout la carte index->slot maximisant le
// recouvrement avec les paires live, en prenant les 8 indices les plus fréquents (permutation exacte).
func solveMap(tag string, pairs [][2]int, union map[int]int, liveCnt map[[2]uint64]int, nLive int) {
	// 8 indices les plus fréquents.
	type iv struct {
		idx, cnt int
	}
	var ivs []iv
	for k, v := range union {
		ivs = append(ivs, iv{k, v})
	}
	sort.Slice(ivs, func(i, j int) bool {
		if ivs[i].cnt != ivs[j].cnt {
			return ivs[i].cnt > ivs[j].cnt
		}
		return ivs[i].idx < ivs[j].idx
	})
	var di []int
	for i := 0; i < len(ivs) && i < 8; i++ {
		di = append(di, ivs[i].idx)
	}
	sort.Ints(di)
	ov, mp := bestInjectionToSlots(pairs, di, liveCnt)
	// total paires décodées mappables (les deux indices dans di)
	inDi := map[int]bool{}
	for _, x := range di {
		inDi[x] = true
	}
	nMappable := 0
	for _, p := range pairs {
		if inDi[p[0]] && inDi[p[1]] {
			nMappable++
		}
	}
	fmt.Printf("  8 indices dominants retenus : %v\n", di)
	fmt.Printf("  carte index->slot (max-recouvrement) : ")
	var dk []int
	for k := range mp {
		dk = append(dk, k)
	}
	sort.Ints(dk)
	for _, k := range dk {
		fmt.Printf("%d->%d ", k, mp[k])
	}
	fmt.Printf("\n  recouvrement paires : %d (sur %d décodées-mappables, %d live) = %.0f%% du live\n",
		ov, nMappable, nLive, 100*float64(ov)/float64(max1(nLive)))
	// marge : overlap de l'IDENTITÉ (index==slot) vs meilleur NON-identité + baseline moyen aléatoire.
	dedup := make([]pipeKill, 0, len(pairs))
	for _, p := range pairs {
		dedup = append(dedup, pipeKill{killer: p[0], victim: p[1]})
	}
	idMap := map[int]uint64{}
	for _, k := range di {
		idMap[k] = uint64(k)
	}
	idOv := overlapForMap(dedup, idMap, liveCnt)
	bestNon, sumOv, nPerm := identityMargin(dedup, di, liveCnt)
	fmt.Printf("  overlap IDENTITÉ direct : %d | meilleur NON-identité : %d | moyen sur %d permutations : %.1f\n",
		idOv, bestNon, nPerm, float64(sumOv)/float64(max1(nPerm)))
	// test de formules fixes sur la carte trouvée.
	testFormulas(tag, mp)
}

// testFormulas : la carte index->slot suit-elle une formule fixe simple ?
func testFormulas(tag string, mp map[int]uint64) {
	var di []int
	for k := range mp {
		di = append(di, k)
	}
	sort.Ints(di)
	// candidat 1 : identité (slot == index)
	idOK := 0
	for _, k := range di {
		if uint64(k) == mp[k] {
			idOK++
		}
	}
	// candidat 2 : index >> 1 (2*slot)
	shrOK := 0
	for _, k := range di {
		if uint64(k>>1) == mp[k] {
			shrOK++
		}
	}
	// candidat 3 : rang dans l'ordre croissant des indices (index trié -> 0..7)
	rankOK := 0
	for r, k := range di {
		if uint64(r) == mp[k] {
			rankOK++
		}
	}
	// candidat 4 : nombre de bits à 1 (popcount) — test d'un encodage one-hot-ish
	popOK := 0
	for _, k := range di {
		if uint64(popcount(k)) == mp[k] {
			popOK++
		}
	}
	// candidat 5 : log2 pour puissances de 2, sinon -1
	log2OK, log2N := 0, 0
	for _, k := range di {
		if l := ilog2(k); l >= 0 {
			log2N++
			if uint64(l) == mp[k] {
				log2OK++
			}
		}
	}
	n := len(di)
	fmt.Printf("  [formules %s] identité:%d/%d | index>>1:%d/%d | rang-tri:%d/%d | popcount:%d/%d | log2(sur %d puiss2):%d\n",
		tag, idOK, n, shrOK, n, rankOK, n, popOK, n, log2N, log2OK)
}

// identityMargin : parcourt toutes les permutations di->slots(0..7), renvoie (meilleur overlap parmi
// les cartes DIFFÉRENTES de l'identité, somme des overlaps, nombre de permutations). Sert à mesurer
// à quel point l'identité domine (ou non). N'énumère que si di == exactement {0..7} (8! = 40320).
func identityMargin(dedup []pipeKill, di []int, liveCnt map[[2]uint64]int) (bestNon, sumOv, nPerm int) {
	// vérifie di == 0..7
	if len(di) != 8 {
		return -1, 0, 0
	}
	for i, k := range di {
		if k != i {
			return -1, 0, 0 // pas 0..7 contigu -> identité non définie sur cet ensemble
		}
	}
	perm := []int{0, 1, 2, 3, 4, 5, 6, 7}
	assign := map[int]uint64{}
	var rec func(pos int, used int)
	rec = func(pos, used int) {
		if pos == 8 {
			ov := overlapForMap(dedup, assign, liveCnt)
			sumOv += ov
			nPerm++
			isID := true
			for k := 0; k < 8; k++ {
				if assign[k] != uint64(k) {
					isID = false
					break
				}
			}
			if !isID && ov > bestNon {
				bestNon = ov
			}
			return
		}
		for s := 0; s < 8; s++ {
			if used&(1<<s) != 0 {
				continue
			}
			assign[perm[pos]] = uint64(s)
			rec(pos+1, used|(1<<s))
		}
	}
	rec(0, 0)
	return bestNon, sumOv, nPerm
}

func popcount(x int) int {
	c := 0
	for x > 0 {
		c += x & 1
		x >>= 1
	}
	return c
}

func ilog2(x int) int {
	if x <= 0 || (x&(x-1)) != 0 {
		return -1
	}
	l := 0
	for x > 1 {
		x >>= 1
		l++
	}
	return l
}
