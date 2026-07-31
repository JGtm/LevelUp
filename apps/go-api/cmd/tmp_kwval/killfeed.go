package main

// runKillFeed — ATTRIBUTION PAR-KILL UNIFIÉE (livrable). Combine, sur l'horloge frame commune :
//   - firearm : decodePipeline (kill-event 0xE6 localisé + famille du 0xd2 fatal same-clock) ;
//   - grenade : event de lancer 0x4c0c00 (thrower index @+103) contraint thrower==tueur via roster ;
//   - mêlée   : event 0xD3 id64 arme, corrélé au kill en temps serré (l'attaquant mêlée = handle biped,
//               pas un slot -> le tueur vient du kill-event/chunk_27, pas de l'event mêlée).
// + ASSIST : champ assist du kill-event (readOpt), mappé via roster.
// Verdict architecture (workflow film-cause-code) : le film ne porte PAS de code cause par-kill ; la
// catégorie s'infère de QUEL flux tombe à l'horloge du kill. C'est la jointure same-clock du moteur.

import (
	"encoding/binary"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"levelup/go-api/internal/analysis"
)

// kfFullScan : quand true, allRawKE balaie CHAQUE position bit (au lieu de keCandidates) pour valider
// un kill-event. Diagnostic : mesure le plafond de couverture si le locator n'était pas le facteur limitant.
var kfFullScan bool

func containsAnyKF(s string, subs []string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

// grenade ids (acurtis, premiers 32 bits) — espace de tags distinct du catalogue filmshell.
var kfGrenIDs = map[uint32]string{
	0xB0171062: "Frag Grenade", 0xC0E34C44: "Plasma Grenade",
	0x3B2567D4: "Shock Grenade", 0x9212E428: "Spike Grenade",
}

// chunk27Gamertags : carte XUID -> gamertag depuis le kill-feed chunk_27 (champ Gamertag des
// HighlightEvents) — universel, tout film, sans DB. Utilise le meilleur chunk (le plus de kills).
func chunk27Gamertags(m string) map[uint64]string {
	cache := root + "/" + m
	gt := map[uint64]string{}
	bestN := 0
	for ch := 41; ch >= 18; ch-- {
		b := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, ch))
		if len(b) == 0 {
			continue
		}
		evs, _ := analysis.ParseHighlightEvents(b, 0)
		nk := 0
		for _, e := range evs {
			if e.EventType == analysis.EventTypeKill {
				nk++
			}
		}
		if nk > bestN {
			bestN = nk
			gt = map[uint64]string{}
			for _, e := range evs {
				if e.Gamertag != "" {
					gt[e.XUID] = e.Gamertag
				}
			}
		}
	}
	return gt
}

type grenEv struct {
	thrower int
	weapon  string
	ts      uint64
}
type meleeEv struct {
	weapon string
	ts     uint64
}

// scanEvents parse tous les frame packets type-0 pour les events grenade (0x4c0c00) et mêlée (0xD3 id64).
func scanEvents(m string, h32 map[uint32]string) ([]grenEv, []meleeEv) {
	cache := root + "/" + m
	id64name := map[uint64]string{}
	// meleeFam32 : familles d'armes de MÊLÉE PURE (marteau/épée/mutilator). Un id64 d'ARME À FEU dans un
	// paquet 0xD3 = état d'arme tenue (held-state), PAS un kill mêlée -> on ne garde que les familles mêlée.
	meleeFam32 := map[uint32]bool{}
	for id, n := range analysis.WeaponIDToName {
		id64name[id] = n
		if containsAnyKF(n, []string{"Hammer", "Sword", "Blade", "Mutilator", "Diminisher"}) {
			meleeFam32[uint32(id>>32)] = true
		}
	}
	fam32 := meleeFam32
	var grens []grenEv
	var melees []meleeEv
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
			// grenade : marqueur 0x4c0c00 (24b) + id32 + thrower @+103
			for bp := 0; bp+120 <= len(pl)*8; bp++ {
				if bitsAt(pl, bp, 24) != 0x4c0c00 {
					continue
				}
				if nm, ok := kfGrenIDs[uint32(bitsAt(pl, bp+24, 32))]; ok {
					grens = append(grens, grenEv{int(bitsAt(pl, bp+24+32+47, 5)), nm, ts})
					bp += 24
				}
			}
			// mêlée : 0xD3 id64 arme complet (family high32 + low32 catalogué)
			if pl[0] == 0xD3 {
				for bp := 0; bp+64 <= len(pl)*8; bp++ {
					hi := uint32(bitsAt(pl, bp, 32))
					if !fam32[hi] {
						continue
					}
					id := (uint64(hi) << 32) | uint64(bitsAt(pl, bp+32, 32))
					if nm, ok := id64name[id]; ok {
						melees = append(melees, meleeEv{nm, ts})
						bp += 63
					}
				}
			}
		}
	}
	return grens, melees
}

// tsWin : fenêtre en unités ts-frame. La ré-sérialisation dedup utilise 3e6 pour ~3ms -> ~1e6 ts/ms.
const tsPerMs = 1_000_000

// decodeBroadKE : scan LARGE — décode les kill-events dans TOUS les paquets type-0 (tous marqueurs,
// toutes tailles), pas seulement dmgMk+sz>=700. But : la couverture des KILLS (tueur/victime) ne doit
// PAS dépendre de la présence d'un record de dégât co-localisé — le kill-event existe pour chaque kill.
// allRawKE : TOUS les kill-events validKE de TOUS les paquets type-0 (aucune porte dmgMk/sz),
// bruts (avant filtre). Chaque pipeKill porte arme (source de dégât au curseur) + marker+sz+cur+assist.
func allRawKE(m string, h32 map[uint32]string) []pipeKill {
	cache := root + "/" + m
	var out []pipeKill
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
			if typ != 0 || len(pl) < 20 {
				continue
			}
			cands := keCandidates(pl, keFloor, len(pl)*8)
			if kfFullScan {
				cands = cands[:0]
				for c := keFloor; c+17 <= len(pl)*8; c++ {
					cands = append(cands, c)
				}
			}
			for _, cur := range cands {
				if !validKE(pl, cur) {
					continue
				}
				vic, kil := decodeKE(pl, cur)
				if vic < 0 || kil < 0 || vic == kil {
					continue
				}
				weapon := "cause-?"
				fam := weaponAnchorLast(pl, cur)
				wRel := fam >= 0
				if fam < 0 {
					fam = weaponAnchor(pl)
				}
				if fam >= 0 {
					weapon = weaponName(uint32(bitsAt(pl, fam, 32)), h32)
				}
				_, b2 := keReadOpt(pl, cur)
				_, b3 := keReadOpt(pl, b2)
				assist, _ := keReadOpt(pl, b3+33)
				out = append(out, pipeKill{killer: kil, victim: vic, assist: assist, weapon: weapon, marker: pl[0], ts: ts, cur: cur, sz: sz, wRel: wRel})
			}
		}
	}
	return out
}

// runGap : DIAGNOSTIC HONNÊTE de la couverture. Pour chaque kill de chunk_27, dit s'il est décodé,
// et sinon POURQUOI : self-kill (suicide, killer==victim), ou kill « normal » non décodé (= vrai trou).
// But : distinguer les kills légitimement sans arme des échecs de décodeur.
func runGap(m string, h32 map[uint32]string) {
	kfFullScan = len(os.Args) >= 4 && os.Args[3] == "full"
	dedup, _, _ := decodePipeline(m, h32)
	kv, nKills := chunk27KV(m)
	var c27 [][2]uint64
	for _, p := range kv {
		kx, _ := strconv.ParseUint(p.KillerXUID, 10, 64)
		vx, _ := strconv.ParseUint(p.VictimXUID, 10, 64)
		c27 = append(c27, [2]uint64{kx, vx})
	}
	rmap, _, _ := solveRoster(dedup, c27)
	// multiset des paires décodées (mappées XUID)
	dec := map[[2]uint64]int{}
	for _, k := range dedup {
		if kx, ok := rmap[k.killer]; ok {
			if vx, ok2 := rmap[k.victim]; ok2 {
				dec[[2]uint64{kx, vx}]++
			}
		}
	}
	gt := chunk27Gamertags(m)
	name := func(x uint64) string {
		if g, ok := gt[x]; ok {
			return g
		}
		return fmt.Sprintf("%d", x)
	}
	_ = name
	inv := map[uint64]int{}
	for i, x := range rmap {
		inv[x] = i
	}
	// index de TOUS les kill-events bruts (tous marqueurs/tailles) par (killer,victim) index
	rawByPair := map[[2]int][]pipeKill{}
	for _, e := range allRawKE(m, h32) {
		k := [2]int{e.killer, e.victim}
		rawByPair[k] = append(rawByPair[k], e)
	}
	nSelf, nCovered, nGap := 0, 0, 0
	used := map[[2]uint64]int{}
	type gapInfo struct {
		ki, vi int
	}
	var gaps []gapInfo
	for _, p := range kv {
		kx, _ := strconv.ParseUint(p.KillerXUID, 10, 64)
		vx, _ := strconv.ParseUint(p.VictimXUID, 10, 64)
		if kx == vx {
			nSelf++
			continue
		}
		key := [2]uint64{kx, vx}
		if used[key] < dec[key] {
			used[key]++
			nCovered++
		} else {
			nGap++
			if ki, ok1 := inv[kx]; ok1 {
				if vi, ok2 := inv[vx]; ok2 {
					gaps = append(gaps, gapInfo{ki, vi})
				}
			}
		}
	}
	nonSelf := nKills - nSelf
	fmt.Printf("=== GAP %s : %d kills | self=%d | couverts(pipeline)=%d (%.1f%%) | TROU=%d ===\n",
		m, nKills, nSelf, nCovered, 100*float64(nCovered)/float64(max1(nonSelf)), nGap)
	// TARGETED : pour chaque trou mappable au roster, son kill-event existe-t-il dans le film brut ?
	dmgMk := map[byte]bool{0xD2: true, 0xC0: true, 0xC2: true, 0xC3: true, 0xCA: true, 0xD3: true, 0xE9: true}
	foundKE, absentKE, small, notInDmg := 0, 0, 0, 0
	mkHist := map[byte]int{}
	for _, g := range gaps {
		evs := rawByPair[[2]int{g.ki, g.vi}]
		if len(evs) == 0 {
			absentKE++
			continue
		}
		foundKE++
		e := evs[0]
		mkHist[e.marker]++
		if e.sz < 700 {
			small++
		}
		if !dmgMk[e.marker] {
			notInDmg++
		}
	}
	fmt.Printf(">>> Sur %d trous mappables au roster : kill-event PRÉSENT dans le film brut %d, ABSENT %d\n", len(gaps), foundKE, absentKE)
	fmt.Printf("    marqueurs des présents : %v | dont sz<700 : %d | dont hors dmgMk : %d\n", mkHist, small, notInDmg)

	// COUVERTURE ATTEIGNABLE (roster-ancré) : pour chaque paire chunk_27, compter les KE bruts dédupliqués
	// (via roster) plafonnés au compte chunk_27. = décoder multi-KE SANS FP (le roster + chunk_27 filtrent).
	c27cnt := map[[2]uint64]int{}
	for _, p := range c27 {
		c27cnt[p]++
	}
	achievable := 0
	for pair, cnt := range c27cnt {
		ki, ok1 := inv[pair[0]]
		vi, ok2 := inv[pair[1]]
		if !ok1 || !ok2 {
			continue
		}
		raw := rawByPair[[2]int{ki, vi}]
		sort.SliceStable(raw, func(i, j int) bool { return raw[i].ts < raw[j].ts })
		dd := 0
		var lastTs uint64
		for i, e := range raw {
			if i == 0 || e.ts-lastTs >= 3000000 {
				dd++
				lastTs = e.ts
			}
		}
		achievable += min(dd, cnt)
	}
	fmt.Printf(">>> COUVERTURE ATTEIGNABLE (roster-ancré, multi-KE dédup, plafond chunk_27) : %d/%d = %.1f%%\n",
		achievable, nKills, 100*float64(achievable)/float64(max1(nKills)))
}

// dedupTs : déduplique une liste de pipeKill triée par ts (fenêtre 3ms).
func dedupTs(raw []pipeKill) []pipeKill {
	sort.SliceStable(raw, func(i, j int) bool { return raw[i].ts < raw[j].ts })
	var dd []pipeKill
	var last uint64
	for i, e := range raw {
		if i == 0 || e.ts-last >= 3000000 {
			dd = append(dd, e)
			last = e.ts
		}
	}
	return dd
}

// augmentedDecode : décodage AUGMENTÉ roster-ancré. Le pipeline propre ne prend qu'UN kill-event par
// paquet ; ici on décode TOUS les kill-events bruts (allRawKE) puis on ne garde que ceux dont la paire
// (via roster stable) existe dans chunk_27, plafonnés au compte chunk_27. Décode le multi-KE SANS FP
// (une paire inconnue de chunk_27 = rejetée). Renvoie kills sélectionnés (index) + roster + couverture.
func augmentedDecode(m string, h32 map[uint32]string) ([]pipeKill, map[int]uint64, int, int) {
	dedup, _, _ := decodePipeline(m, h32)
	kv, nKills := chunk27KV(m)
	var c27 [][2]uint64
	for _, p := range kv {
		kx, _ := strconv.ParseUint(p.KillerXUID, 10, 64)
		vx, _ := strconv.ParseUint(p.VictimXUID, 10, 64)
		c27 = append(c27, [2]uint64{kx, vx})
	}
	rmap, _, _ := solveRoster(dedup, c27)
	inv := map[uint64]int{}
	for i, x := range rmap {
		inv[x] = i
	}
	c27cnt := map[[2]uint64]int{}
	for _, p := range c27 {
		c27cnt[p]++
	}
	rawByPair := map[[2]int][]pipeKill{}
	for _, e := range allRawKE(m, h32) {
		rawByPair[[2]int{e.killer, e.victim}] = append(rawByPair[[2]int{e.killer, e.victim}], e)
	}
	var out []pipeKill
	covered := 0
	for pair, cnt := range c27cnt {
		ki, ok1 := inv[pair[0]]
		vi, ok2 := inv[pair[1]]
		if !ok1 || !ok2 {
			continue
		}
		dd := dedupTs(rawByPair[[2]int{ki, vi}])
		take := cnt
		if len(dd) < take {
			take = len(dd)
		}
		out = append(out, dd[:take]...)
		covered += take
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].ts < out[j].ts })
	return out, rmap, covered, nKills
}

// augmentedDecodeV2 : full-scan (100% rappel kill) + sélection PRÉFÉRANT L'ARME (clusterBestWeapon au lieu
// de dedupTs). But : garder le rappel-kill du full-scan SANS perdre l'arme co-localisée. Roster solvé sur le
// pipeline propre (dmgMk-gaté), appliqué au full-scan.
// augmentedDecodeV2Kills : comme augmentedDecodeV2 mais renvoie les kills sélectionnés (pour comparer la
// distribution d'armes à la vérité live).
func augmentedDecodeV2Kills(m string, h32 map[uint32]string) ([]pipeKill, int) {
	dedup, _, _ := decodePipeline(m, h32)
	kv, nk := chunk27KV(m)
	var c27 [][2]uint64
	for _, p := range kv {
		kx, _ := strconv.ParseUint(p.KillerXUID, 10, 64)
		vx, _ := strconv.ParseUint(p.VictimXUID, 10, 64)
		c27 = append(c27, [2]uint64{kx, vx})
	}
	rmap, _, _ := solveRoster(dedup, c27)
	inv := map[uint64]int{}
	for i, x := range rmap {
		inv[x] = i
	}
	c27cnt := map[[2]uint64]int{}
	for _, p := range c27 {
		c27cnt[p]++
	}
	kfFullScan = true
	rawByPair := map[[2]int][]pipeKill{}
	for _, e := range allRawKE(m, h32) {
		rawByPair[[2]int{e.killer, e.victim}] = append(rawByPair[[2]int{e.killer, e.victim}], e)
	}
	kfFullScan = false
	var out []pipeKill
	for pair, cnt := range c27cnt {
		ki, ok1 := inv[pair[0]]
		vi, ok2 := inv[pair[1]]
		if !ok1 || !ok2 {
			continue
		}
		dd := clusterBestWeapon(rawByPair[[2]int{ki, vi}])
		sort.SliceStable(dd, func(i, j int) bool {
			if weaponScore(dd[i]) != weaponScore(dd[j]) {
				return weaponScore(dd[i]) > weaponScore(dd[j])
			}
			return dd[i].ts < dd[j].ts
		})
		take := cnt
		if len(dd) < take {
			take = len(dd)
		}
		out = append(out, dd[:take]...)
	}
	return out, nk
}

// runAugCov2Live : ACCURACY index-free — compare la DISTRIBUTION d'armes d'augcov2 à la vérité live
// (decodeLive) pour un film à capture live (9b191a7f). Distribution d'accord = mesure d'accuracy sans
// dépendre du mapping index<->joueur (le point faible du roster).
func runAugCov2Live(m string, h32 map[uint32]string) {
	sel, nKills := augmentedDecodeV2Kills(m, h32)
	live, ok := decodeLive(m, h32)
	if !ok {
		fmt.Printf("PAS de capture live pour %s — accuracy arme non vérifiable ici.\n", m)
		return
	}
	selG, liveG := map[string]int{}, map[string]int{}
	selN, liveN := 0, 0
	for _, k := range sel {
		if k.weapon != "" && !strings.HasPrefix(k.weapon, "cause-") && !strings.HasPrefix(k.weapon, "fam-") {
			selG[k.weapon]++
			selN++
		}
	}
	for _, a := range live {
		if a.w != "" {
			liveG[a.w]++
			liveN++
		}
	}
	gi := inter(selG, liveG)
	fmt.Printf("=== AUGCOV2-LIVE %s : chunk_27=%d ===\n", m, nKills)
	fmt.Printf("augcov2 : %d kills armés | live : %d kills armés\n", selN, liveN)
	fmt.Printf("augcov2: %s\n", fmtHist(selG))
	fmt.Printf("live   : %s\n", fmtHist(liveG))
	fmt.Printf(">>> ACCORD DE DISTRIBUTION (intersection/live) : %d/%d = %.0f%%\n", gi, liveN, 100*float64(gi)/float64(max1(liveN)))
}

func augmentedDecodeV2(m string, h32 map[uint32]string) (covered, named, reliable, nKills int) {
	dedup, _, _ := decodePipeline(m, h32) // roster sur données propres
	kv, nk := chunk27KV(m)
	nKills = nk
	var c27 [][2]uint64
	for _, p := range kv {
		kx, _ := strconv.ParseUint(p.KillerXUID, 10, 64)
		vx, _ := strconv.ParseUint(p.VictimXUID, 10, 64)
		c27 = append(c27, [2]uint64{kx, vx})
	}
	rmap, _, _ := solveRoster(dedup, c27)
	inv := map[uint64]int{}
	for i, x := range rmap {
		inv[x] = i
	}
	c27cnt := map[[2]uint64]int{}
	for _, p := range c27 {
		c27cnt[p]++
	}
	kfFullScan = true // full scan pour le rappel-kill
	rawByPair := map[[2]int][]pipeKill{}
	for _, e := range allRawKE(m, h32) {
		rawByPair[[2]int{e.killer, e.victim}] = append(rawByPair[[2]int{e.killer, e.victim}], e)
	}
	kfFullScan = false
	for pair, cnt := range c27cnt {
		ki, ok1 := inv[pair[0]]
		vi, ok2 := inv[pair[1]]
		if !ok1 || !ok2 {
			continue
		}
		dd := clusterBestWeapon(rawByPair[[2]int{ki, vi}]) // 1 candidat par cluster-temps, meilleure arme
		// sélection : préférer les clusters PORTEURS D'ARME (score desc), puis par temps ; prendre cnt.
		sort.SliceStable(dd, func(i, j int) bool {
			if weaponScore(dd[i]) != weaponScore(dd[j]) {
				return weaponScore(dd[i]) > weaponScore(dd[j])
			}
			return dd[i].ts < dd[j].ts
		})
		take := cnt
		if len(dd) < take {
			take = len(dd)
		}
		for _, k := range dd[:take] {
			covered++
			if weaponScore(k) >= 1 {
				named++
			}
			if weaponScore(k) >= 2 { // reliableFirearm : record de dégât réel (dmgMk + sz>=700)
				reliable++
			}
		}
	}
	return
}

// runAugCov2 : couverture avec sélection préférant l'arme (full-scan + clusterBestWeapon).
// ATTENTION : le full-scan GONFLE la couverture-KILL par faux positifs (décode n'importe quelle paire
// (ki,vi) à une position aléatoire). couverture-ARME-FIABLE (record de dégât réel) est le seul chiffre
// non-gonflable. Accuracy des positions NON vérifiée ici -> voir la comparaison à l'oracle dispatcher.
func runAugCov2(m string, h32 map[uint32]string) {
	covered, named, reliable, nKills := augmentedDecodeV2(m, h32)
	fmt.Printf("=== AUGCOV2 %s : chunk_27=%d (full-scan + clusterBestWeapon) ===\n", m, nKills)
	fmt.Printf("  couverture-KILL (GONFLÉE par FP full-scan) : %d/%d = %.1f%%\n", covered, nKills, 100*float64(covered)/float64(max1(nKills)))
	fmt.Printf("  couverture-ARME nommée (suspecte)          : %d/%d = %.1f%%\n", named, nKills, 100*float64(named)/float64(max1(nKills)))
	fmt.Printf("  couverture-ARME FIABLE (record dégât réel) : %d/%d = %.1f%%\n", reliable, nKills, 100*float64(reliable)/float64(max1(nKills)))
}

// runAugCovSC : arme SAME-CLOCK cross-paquet. Kills corrects (augmentedDecode, ancré chunk_27, index=slot)
// + arme = dernier record de dégât 0xd2 (attaquant==tueur) avant l'instant du kill, TOUS paquets confondus
// (pas la co-localisation même-paquet weaponAnchorLast). Precision-safe (vrais records). Validé vs live.
func runAugCovSC(m string, h32 map[uint32]string) {
	out, _, covered, nKills := augmentedDecode(m, h32)
	_, dmgs := collectRaw(m, h32, 83) // flux de dégât (killOff sans effet sur les 0xd2)
	sameClockW := func(killer int, ts uint64) string {
		best := uint64(1) << 63
		w := ""
		for _, dg := range dmgs {
			if dg.ts > ts || dg.atk != killer {
				continue
			}
			if ts-dg.ts < best {
				best, w = ts-dg.ts, dg.fam
			}
		}
		return w
	}
	named := 0
	selG := map[string]int{}
	for _, k := range out {
		if w := sameClockW(k.killer, k.ts); w != "" {
			named++
			selG[w]++
		}
	}
	fmt.Printf("=== AUGCOV-SC %s : chunk_27=%d ===\n", m, nKills)
	fmt.Printf("  couverture-KILL (ancré chunk_27) : %d/%d = %.1f%%\n", covered, nKills, 100*float64(covered)/float64(max1(nKills)))
	fmt.Printf("  couverture-ARME same-clock       : %d/%d = %.1f%%\n", named, nKills, 100*float64(named)/float64(max1(nKills)))
	if live, ok := decodeLive(m, h32); ok {
		liveG := map[string]int{}
		liveN := 0
		for _, a := range live {
			if a.w != "" {
				liveG[a.w]++
				liveN++
			}
		}
		gi := inter(selG, liveG)
		fmt.Printf("  --- vs LIVE ---\n  SC   : %s\n  live : %s\n", fmtHist(selG), fmtHist(liveG))
		fmt.Printf("  >>> ACCORD DE DISTRIBUTION (intersection/live) : %d/%d = %.0f%%\n", gi, liveN, 100*float64(gi)/float64(max1(liveN)))
		// ACCURACY PER-PAIRE (index=slot=identité prouvé) : l'arme SC d'un kill (tueur,victime) correspond-elle
		// à une arme live du MÊME couple ? Métrique d'accuracy réelle (pas juste distribution).
		liveByPair := map[[2]int][]string{}
		for _, a := range live {
			if a.w != "" {
				liveByPair[[2]int{a.killer, a.victim}] = append(liveByPair[[2]int{a.killer, a.victim}], a.w)
			}
		}
		match, tot := 0, 0
		for _, k := range out {
			w := sameClockW(k.killer, k.ts)
			if w == "" {
				continue
			}
			tot++
			lw := liveByPair[[2]int{k.killer, k.victim}]
			for i, x := range lw {
				if x == w {
					match++
					liveByPair[[2]int{k.killer, k.victim}] = append(lw[:i], lw[i+1:]...)
					break
				}
			}
		}
		fmt.Printf("  >>> ACCURACY PER-PAIRE (arme SC == arme live du même couple) : %d/%d = %.0f%%\n", match, tot, 100*float64(match)/float64(max1(tot)))
	}
}

// runAugCov : mesure la couverture AUGMENTÉE, en distinguant couverture-KILL (tueur/victime décodés)
// et couverture-ARME (arme NOMMÉE, pas cause-?/fam-). Ancré chunk_27, donc accuracy ~100% par construction.
func runAugCov(m string, h32 map[uint32]string) {
	kfFullScan = len(os.Args) >= 4 && os.Args[3] == "full"
	out, _, covered, nKills := augmentedDecode(m, h32)
	named := 0
	for _, k := range out {
		if k.weapon != "" && !strings.HasPrefix(k.weapon, "cause-") && !strings.HasPrefix(k.weapon, "fam-") {
			named++
		}
	}
	fmt.Printf("=== AUGCOV %s : chunk_27=%d ===\n", m, nKills)
	fmt.Printf("  couverture-KILL (tueur/victime, roster-ancré) : %d/%d = %.1f%%\n",
		covered, nKills, 100*float64(covered)/float64(max1(nKills)))
	fmt.Printf("  couverture-ARME (arme nommée)                 : %d/%d = %.1f%%\n",
		named, nKills, 100*float64(named)/float64(max1(nKills)))
}

// firearmDmgMk : marqueurs de RECORD DE DÉGÂT arme-à-feu (0xd2 + frères). EXCLUT 0xD3 (état d'arme
// TENUE = held-state, source du piège held-weapon) et 0xE6 (kill-event nu, sans arme). Un firearm n'est
// FIABLE que si son record tombe sous l'un de ces marqueurs dans un gros paquet fatal (sz>=700).
var firearmDmgMk = map[byte]bool{0xD2: true, 0xC0: true, 0xC2: true, 0xC3: true, 0xCA: true, 0xE9: true}

// reliableFirearm : le candidat porte une arme NOMMÉE issue d'un vrai record de dégât (pas held/parasite).
func reliableFirearm(k pipeKill) bool {
	named := !strings.HasPrefix(k.weapon, "cause-") && !strings.HasPrefix(k.weapon, "fam-")
	return named && firearmDmgMk[k.marker] && k.sz >= 700
}

// weaponScore : qualité de l'arme d'un candidat (2=firearm fiable, 1=nommée mais suspecte held/parasite, 0=cause-?).
func weaponScore(k pipeKill) int {
	switch {
	case reliableFirearm(k):
		return 2
	case !strings.HasPrefix(k.weapon, "cause-") && !strings.HasPrefix(k.weapon, "fam-"):
		return 1
	default:
		return 0
	}
}

// clusterBestWeapon : regroupe les candidats par cluster-temps (3ms) et garde, par cluster, celui qui
// porte la meilleure arme. Sépare les vrais kill-events (arme co-localisée) des positions parasites.
func clusterBestWeapon(cands []pipeKill) []pipeKill {
	sort.SliceStable(cands, func(i, j int) bool { return cands[i].ts < cands[j].ts })
	var out []pipeKill
	for i := 0; i < len(cands); {
		best, bs, j := cands[i], weaponScore(cands[i]), i+1
		for j < len(cands) && cands[j].ts-cands[i].ts < 3000000 {
			if s := weaponScore(cands[j]); s > bs {
				best, bs = cands[j], s
			}
			j++
		}
		out = append(out, best)
		i = j
	}
	return out
}

// classifyAug : catégorise un cluster kill-event (firearm nommé > mêlée same-clock > grenade lancée par le
// tueur same-clock > inconnu). Renvoie (arme, catégorie, score de priorité pour le plafond chunk_27).
// classifyAug : priorité DOCTRINE-CORRECTE (identique au pipeline propre) : firearm FIABLE (vrai record
// de dégât) > mêlée same-clock > grenade lancée par le tueur same-clock > firearm SUSPECT (held/parasite,
// dernier recours honnête = FIREARM?) > inconnu. Ne JAMAIS laisser une arme tenue voler un kill mêlée/grenade.
func classifyAug(k pipeKill, kx uint64, melees []meleeEv, grens []grenEv, rmap map[int]uint64) (string, string, int) {
	if reliableFirearm(k) {
		return k.weapon, "FIREARM", 4
	}
	md := int64(400) * tsPerMs
	for i := range melees {
		d := int64(k.ts) - int64(melees[i].ts)
		if d < 0 {
			d = -d
		}
		if d <= md {
			return melees[i].weapon, "MÊLÉE", 3
		}
	}
	gd := int64(3000) * tsPerMs
	for i := range grens {
		if gx, ok := rmap[grens[i].thrower]; ok && gx == kx {
			if d := int64(k.ts) - int64(grens[i].ts); d >= -200*tsPerMs && d <= gd {
				return grens[i].weapon, "GRENADE", 2
			}
		}
	}
	if !strings.HasPrefix(k.weapon, "cause-") && !strings.HasPrefix(k.weapon, "fam-") {
		return k.weapon, "FIREARM?", 1
	}
	return "?", "INCONNU", 0
}

type augRow struct {
	kx, vx uint64
	cat    string
	weapon string
	assist int
	ts     uint64
	score  int
}

// runKillFeedAug — FEED AUGMENTÉ (livrable cible). Balaie TOUS les kill-events (full-scan), les ancre au
// roster+chunk_27 (aucun FP de paire), et pour chaque paire sélectionne les meilleurs clusters : arme
// nommée d'abord, puis mêlée/grenade same-clock, plafonné au compte chunk_27. Couverture-KILL ~100%,
// catégorie décodée pour le maximum de kills. Aucune corrélation par temps absolu (pas de warp).
func runKillFeedAug(m string, h32 map[uint32]string) {
	csv := len(os.Args) >= 4 && os.Args[3] == "csv"
	kfFullScan = true
	dedup, _, _ := decodePipeline(m, h32)
	kv, nKills := chunk27KV(m)
	if len(kv) == 0 {
		fmt.Printf("=== KFAUG %s : chunk_27 vide ===\n", m)
		return
	}
	var c27 [][2]uint64
	for _, p := range kv {
		kx, _ := strconv.ParseUint(p.KillerXUID, 10, 64)
		vx, _ := strconv.ParseUint(p.VictimXUID, 10, 64)
		c27 = append(c27, [2]uint64{kx, vx})
	}
	rmap, overlap, _ := solveRoster(dedup, c27)
	inv := map[uint64]int{}
	for i, x := range rmap {
		inv[x] = i
	}
	c27cnt := map[[2]uint64]int{}
	for _, p := range c27 {
		c27cnt[p]++
	}
	grens, melees := scanEvents(m, h32)
	gtMap := chunk27Gamertags(m)
	name := func(x uint64) string {
		if g, ok := gtMap[x]; ok {
			return g
		}
		return fmt.Sprintf("xuid:%d", x)
	}
	rawByPair := map[[2]int][]pipeKill{}
	for _, e := range allRawKE(m, h32) {
		rawByPair[[2]int{e.killer, e.victim}] = append(rawByPair[[2]int{e.killer, e.victim}], e)
	}
	var rows []augRow
	for pair, cnt := range c27cnt {
		ki, ok1 := inv[pair[0]]
		vi, ok2 := inv[pair[1]]
		if !ok1 || !ok2 {
			continue
		}
		clusters := clusterBestWeapon(rawByPair[[2]int{ki, vi}])
		var scored []augRow
		for _, c := range clusters {
			w, cat, sc := classifyAug(c, pair[0], melees, grens, rmap)
			assist := -1
			if c.assist >= 0 {
				if ax, ok := rmap[c.assist]; ok && ax != pair[0] {
					assist = c.assist
				}
			}
			scored = append(scored, augRow{pair[0], pair[1], cat, w, assist, c.ts, sc})
		}
		sort.SliceStable(scored, func(i, j int) bool {
			if scored[i].score != scored[j].score {
				return scored[i].score > scored[j].score
			}
			return scored[i].ts < scored[j].ts
		})
		take := cnt
		if len(scored) < take {
			take = len(scored)
		}
		rows = append(rows, scored[:take]...)
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].ts < rows[j].ts })
	nFire, nMel, nGren, nSusp, nUnk := 0, 0, 0, 0, 0
	for _, r := range rows {
		switch r.cat {
		case "FIREARM":
			nFire++
		case "MÊLÉE":
			nMel++
		case "GRENADE":
			nGren++
		case "FIREARM?":
			nSusp++
		default:
			nUnk++
		}
		if csv {
			as := ""
			if r.assist >= 0 {
				if ax, ok := rmap[r.assist]; ok {
					as = name(ax)
				}
			}
			fmt.Printf("ROW\t%s\t%s\t%s\t%s\t%s\n", name(r.kx), name(r.vx), r.cat, r.weapon, as)
		}
	}
	classified := nFire + nMel + nGren
	fmt.Printf("=== KFAUG %s : chunk_27=%d roster=%d/%d ===\n", m, nKills, overlap, nKills)
	fmt.Printf("  couverture-KILL       : %d/%d = %.1f%%\n", len(rows), nKills, 100*float64(len(rows))/float64(max1(nKills)))
	fmt.Printf("  catégorie FIABLE      : %d/%d = %.1f%% (FIREARM=%d MÊLÉE=%d GRENADE=%d)\n",
		classified, nKills, 100*float64(classified)/float64(max1(nKills)), nFire, nMel, nGren)
	fmt.Printf("  incertain             : FIREARM?(held/parasite)=%d INCONNU=%d\n", nSusp, nUnk)
}

func runKillFeed(m string, h32 map[uint32]string) {
	csv := len(os.Args) >= 4 && os.Args[3] == "csv"
	meleeWinMs, grenWinMs := 400, 3000
	dedup, _, _ := decodePipeline(m, h32)
	kv, nKills := chunk27KV(m)
	if len(dedup) == 0 || len(kv) == 0 {
		fmt.Printf("=== KILLFEED %s : décodées=%d chunk_27=%d (rien) ===\n", m, len(dedup), len(kv))
		return
	}
	var c27 [][2]uint64
	for _, p := range kv {
		kx, _ := strconv.ParseUint(p.KillerXUID, 10, 64)
		vx, _ := strconv.ParseUint(p.VictimXUID, 10, 64)
		c27 = append(c27, [2]uint64{kx, vx})
	}
	rmap, overlap, _ := solveRoster(dedup, c27)
	grens, melees := scanEvents(m, h32)
	gtMap := chunk27Gamertags(m)

	name := func(x uint64) string {
		if g, ok := gtMap[x]; ok {
			return g
		}
		return fmt.Sprintf("xuid:%d", x)
	}
	sort.SliceStable(dedup, func(i, j int) bool { return dedup[i].ts < dedup[j].ts })

	nFire, nMel, nGren, nUnk := 0, 0, 0, 0
	fmt.Printf("=== KILLFEED UNIFIÉ %s : %d kills décodés (roster overlap %d/%d) | %d grenades, %d mêlées ===\n",
		m, len(dedup), overlap, nKills, len(grens), len(melees))
	for _, k := range dedup {
		kx, kok := rmap[k.killer]
		vx, vok := rmap[k.victim]
		if !kok || !vok {
			continue
		}
		weapon, cat := k.weapon, "FIREARM"
		// firearmNamed = arme trouvée (pas "cause-?"/"fam-") ; firearmReliable = en plus wRel (source fatale
		// AVANT le curseur = 0xd2/0xd3, sûre). Priorité : firearm FIABLE > grenade(thrower==tueur) >
		// mêlée(marteau/épée proche) > firearm incertain > inconnu. Le mécanisme même-horloge du moteur.
		firearmNamed := !strings.HasPrefix(weapon, "cause-") && !strings.HasPrefix(weapon, "fam-")
		// CONSERVATEUR (verdict a : pas de code cause -> grenade non séparable du firearm quand une arme est
		// décodée). Une arme NOMMÉE = FIREARM. Seuls les "cause-?" (aucune arme trouvée dans le paquet fatal)
		// basculent : mêlée (marteau/épée en temps serré = kill mêlée) puis grenade (lancer par le tueur).
		if !firearmNamed {
			var mBest *meleeEv
			md := int64(meleeWinMs) * tsPerMs
			for i := range melees {
				d := int64(k.ts) - int64(melees[i].ts)
				if d < 0 {
					d = -d
				}
				if d <= md {
					md, mBest = d, &melees[i]
				}
			}
			var gBest *grenEv
			gd := int64(grenWinMs) * tsPerMs
			for i := range grens {
				if gx, ok := rmap[grens[i].thrower]; ok && gx == kx {
					if d := int64(k.ts) - int64(grens[i].ts); d >= -200*tsPerMs && d <= gd {
						gd, gBest = d, &grens[i]
					}
				}
			}
			switch {
			case mBest != nil:
				weapon, cat = mBest.weapon, "MÊLÉE"
			case gBest != nil:
				weapon, cat = gBest.weapon, "GRENADE"
			default:
				weapon, cat = "?", "INCONNU"
			}
		}
		switch {
		case strings.HasPrefix(cat, "FIREARM"):
			nFire++
		case cat == "MÊLÉE":
			nMel++
		case cat == "GRENADE":
			nGren++
		default:
			nUnk++
		}
		assistN := ""
		if k.assist >= 0 {
			if ax, ok := rmap[k.assist]; ok && ax != kx {
				assistN = name(ax)
			}
		}
		if csv {
			fmt.Printf("ROW\t%s\t%s\t%s\t%s\t%s\n", name(kx), name(vx), cat, weapon, assistN)
			continue
		}
		as := ""
		if assistN != "" {
			as = " assist=" + assistN
		}
		fmt.Printf("  %-16s -> %-16s | %-8s %-22s%s\n", name(kx), name(vx), cat, weapon, as)
	}
	fmt.Printf("\n=== BILAN %s : FIREARM=%d MÊLÉE=%d GRENADE=%d INCONNU=%d (sur %d décodés / %d kills chunk_27) ===\n",
		m, nFire, nMel, nGren, nUnk, len(dedup), nKills)
}
