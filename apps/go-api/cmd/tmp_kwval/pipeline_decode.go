package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"levelup/go-api/internal/analysis"
)

// pipeKill = un kill décodé par le pipeline offline (index joueur 0..7, arme = famille du dégât).
type pipeKill struct {
	killer, victim int
	assist         int // index assistant (readOpt du kill-event 0xE6) ; -1 = absent
	weapon         string
	marker         byte
	ts             uint64
	cur, f2, nCand int
	sz             int
	wRel           bool // arme FIABLE : weaponAnchorLast a trouvé la famille fatale AVANT le curseur
	//                     (cas 0xd2/0xd3). Faux = repli weaponAnchor (1re arme, frères) -> arme incertaine.
}

// pipeMultiKE : quand true, decodePipeline émet TOUS les kill-events d'un paquet fatal (multi-kill
// même-horloge) au lieu du seul premier curseur. Active via l'arg "multi". Défaut false = comportement legacy.
var pipeMultiKE bool

// pipeDiag = compteurs de diagnostic du décodage (identiques à ceux imprimés par runPipeline).
type pipeDiag struct {
	nFatal, nLoc, nNoKE, nRoster int
	markerLoc                    map[byte]int
}

// decodePipeline exécute le décodage offline complet et renvoie le kill feed dédupliqué.
// Chaîne : détection fatal (marqueur damage-family + sz>=700) -> localisation kill-event
// (locateKillEventCursor) -> gate profondeur/arme -> décodage victime=field0/tueur=field1/arme=famille
// -> filtre roster (indices vus >=2x) -> dédup (tueur,victime,~3ms). Source UNIQUE partagée par
// runPipeline (affichage + validation CE) et runPairMatrix (validation iso vs chunk_27). Renvoie
// (dedup, rawFeedLen post-roster, diag).
func decodePipeline(m string, h32 map[uint32]string) ([]pipeKill, int, pipeDiag) {
	cache := root + "/" + m
	dmgMk := map[byte]bool{0xD2: true, 0xC0: true, 0xC2: true, 0xC3: true, 0xCA: true, 0xD3: true, 0xE9: true}
	var feed []pipeKill
	diag := pipeDiag{markerLoc: map[byte]int{}}
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
			if typ != 0 || len(pl) == 0 || !dmgMk[pl[0]] || sz < 700 {
				continue
			}
			diag.nFatal++
			// MULTI-KE : un paquet fatal peut sérialiser 2-3 records (2-3 kills même horloge). Le legacy ne
			// prenait que le PREMIER curseur (locateKillEventCursor=c[0]) -> perte des kills suivants. On itère
			// TOUS les candidats keCandidates (préfixe 0x2A8 spécifique), dédupliqués par paire décodée dans
			// le paquet (curseurs quasi-doublons du même kill = 1 seul), chacun avec SON arme co-localisée.
			cands := keCandidates(pl, keFloor, len(pl)*8)
			if len(cands) == 0 {
				diag.nNoKE++
				continue
			}
			if !pipeMultiKE {
				cands = cands[:1] // legacy : premier curseur seulement
			}
			waFallback := weaponAnchor(pl)
			seenPair := map[[2]int]bool{}
			emitted := false
			for _, cur := range cands {
				// Gate anti-FP : kill-event profond (cur>=1024) dans un paquet SANS arme = parasite.
				if waFallback < 0 && cur >= 1024 {
					continue
				}
				vic, kil := decodeKE(pl, cur)
				if seenPair[[2]int{vic, kil}] {
					continue // curseur quasi-doublon du même kill dans ce paquet
				}
				seenPair[[2]int{vic, kil}] = true
				// SOURCE DE DÉGÂT FATALE : le record de dégât juste avant le curseur. La CAUSE = sa FAMILLE
				// (S6, +0x10). Le variant (S7, +0x14) const 0x42C9679F = ANCRE. Doctrine : cause = source de
				// dégât, jamais held-weapon. weaponAnchorLast = dernière source AVANT ce curseur (co-localisée).
				weapon := "cause-?"
				fam := weaponAnchorLast(pl, cur)
				wRel := fam >= 0
				if fam < 0 {
					fam = waFallback // repli frères : 1re source du paquet (arme après le kill-event).
				}
				if fam >= 0 {
					weapon = weaponName(uint32(bitsAt(pl, fam, 32)), h32)
				}
				_, b2 := keReadOpt(pl, cur)
				_, b3 := keReadOpt(pl, b2)
				f2 := int(bitsAt(pl, b3, 32))
				assist, _ := keReadOpt(pl, b3+33) // [vic][kil][R32@b3][R1@b3+32][assist@b3+33][R32]
				feed = append(feed, pipeKill{kil, vic, assist, weapon, pl[0], ts, cur, f2, 0, sz, wRel})
				diag.markerLoc[pl[0]]++
				emitted = true
			}
			if emitted {
				diag.nLoc++
			} else {
				diag.nNoKE++
			}
		}
	}
	// ROSTER : indices vus >=2x (filet universel anti-bruit hors-roster).
	idxFreq := map[int]int{}
	for _, k := range feed {
		idxFreq[k.killer]++
		idxFreq[k.victim]++
	}
	roster := map[int]bool{}
	for i, n := range idxFreq {
		if n >= 2 {
			roster[i] = true
		}
	}
	var kept []pipeKill
	for _, k := range feed {
		if roster[k.killer] && roster[k.victim] {
			kept = append(kept, k)
		} else {
			diag.nRoster++
		}
	}
	feed = kept
	rawFeed := len(feed)
	// DÉDUP inter-marqueurs (tueur,victime,~3ms) : un même kill peut être sérialisé dans deux paquets.
	sort.SliceStable(feed, func(i, j int) bool { return feed[i].ts < feed[j].ts })
	var dedup []pipeKill
	for _, k := range feed {
		dup := false
		for j := len(dedup) - 1; j >= 0 && k.ts-dedup[j].ts < 3000000; j-- {
			if dedup[j].killer == k.killer && dedup[j].victim == k.victim {
				dup = true
				break
			}
		}
		if !dup {
			dedup = append(dedup, k)
		}
	}
	return dedup, rawFeed, diag
}

// chunk27KV reconstruit le kill feed offline chunk_27 via l'algorithme de production
// analysis.ComputeKillerVictimPairs (tolérance 5 ms, bijection kill->death la plus proche) — le MÊME
// appariement que la shared DB killer_victim_pairs. VÉRITÉ-TERRAIN indépendante du décodage frame
// (chunk différent, horloge temps-jeu) disponible sur TOUT match. Renvoie les paires (avec temps +
// gamertags) et le nombre TOTAL d'events KILL (dénominateur autoritatif du kill feed).
func chunk27KV(m string) (kv []analysis.KVPair, nKills int) {
	cache := root + "/" + m
	var best []analysis.HighlightEvent
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
		if nk > nKills {
			nKills, best = nk, evs
		}
	}
	var raw []analysis.RawEvent
	for _, e := range best {
		if e.EventType == analysis.EventTypeKill || e.EventType == analysis.EventTypeDeath {
			raw = append(raw, analysis.RawEvent{EventType: e.EventType, XUID: fmt.Sprintf("%d", e.XUID), TimeMS: int64(e.TimeMS)})
		}
	}
	return analysis.ComputeKillerVictimPairs(raw, 5), nKills
}

// chunk27Pairs = vue (tueur XUID, victime XUID) du kill feed chunk_27 (pour la validation iso).
// Second retour = kills chunk_27 non appariés (aucun death dans la fenêtre).
func chunk27Pairs(m string) (pairs [][2]uint64, unpaired int) {
	kv, nKills := chunk27KV(m)
	for _, p := range kv {
		kx, _ := strconv.ParseUint(p.KillerXUID, 10, 64)
		vx, _ := strconv.ParseUint(p.VictimXUID, 10, 64)
		pairs = append(pairs, [2]uint64{kx, vx})
	}
	return pairs, nKills - len(kv)
}

// solveRoster trouve la carte index-frame(0..7) -> XUID maximisant le recouvrement des paires
// (tueur,victime) décodées avec le kill feed chunk_27. Permutation-invariant : la carte tombe du
// calcul, sans corrélation d'horloge. Renvoie (carte, recouvrement max, multiset chunk_27).
func solveRoster(dedup []pipeKill, c27 [][2]uint64) (map[int]uint64, int, map[[2]uint64]int) {
	idxSet := map[int]bool{}
	for _, k := range dedup {
		idxSet[k.killer] = true
		idxSet[k.victim] = true
	}
	var di []int
	for i := range idxSet {
		di = append(di, i)
	}
	sort.Ints(di)
	xf := map[uint64]int{}
	for _, p := range c27 {
		xf[p[0]]++
		xf[p[1]]++
	}
	var cx []uint64
	for x := range xf {
		cx = append(cx, x)
	}
	// ordre TOTAL déterministe (fréquence desc, puis XUID asc) — sinon le tie-break dépend de l'ordre
	// d'itération de map Go (aléatoire) -> carte roster instable -> calibration warp non déterministe.
	sort.Slice(cx, func(i, j int) bool {
		if xf[cx[i]] != xf[cx[j]] {
			return xf[cx[i]] > xf[cx[j]]
		}
		return cx[i] < cx[j]
	})
	if len(cx) > 8 {
		cx = cx[:8]
	}
	c27cnt := map[[2]uint64]int{}
	for _, p := range c27 {
		c27cnt[p]++
	}
	best, bestMap := bestInjectionOverlap(dedup, di, cx, c27cnt)
	return bestMap, best, c27cnt
}

// runPairMatrix : VALIDATION GÉNÉRALISABLE (tout match, sans oracle CE). Compare le multiset des
// paires (tueur,victime) décodées — en index 0..7 — à celui du kill feed chunk_27 — en XUID — via
// une recherche de permutation roster (index -> XUID) maximisant le recouvrement. Permutation-
// invariant : pas besoin de connaître à l'avance la correspondance index/XUID (elle tombe du calcul).
// overlap/décodées = accuracy par-paire génuine ; overlap/chunk_27 = couverture. La permutation
// gagnante EST la carte roster dont la productionisation a besoin.
func runPairMatrix(m string, h32 map[uint32]string) {
	dedup, _, _ := decodePipeline(m, h32)
	c27, unpaired := chunk27Pairs(m)
	if len(dedup) == 0 || len(c27) == 0 {
		fmt.Printf("=== ISO %s : décodées=%d chunk_27=%d (rien à comparer) ===\n", m, len(dedup), len(c27))
		return
	}
	bestMap, best, c27cnt := solveRoster(dedup, c27)
	accPct := float64(best) * 100 / float64(len(dedup))
	covPct := float64(best) * 100 / float64(len(c27))
	fmt.Printf("=== ISO %s : %d paires décodées, %d paires chunk_27 (%d kills non appariés) ===\n", m, len(dedup), len(c27), unpaired)
	fmt.Printf(">>> ACCURACY par-paire vs chunk_27 (permutation roster) : %d/%d = %.1f%%\n", best, len(dedup), accPct)
	fmt.Printf(">>> COUVERTURE du kill feed chunk_27 : %d/%d = %.1f%%\n", best, len(c27), covPct)
	// carte roster gagnante (index frame -> XUID).
	var di []int
	for i := range bestMap {
		di = append(di, i)
	}
	sort.Ints(di)
	fmt.Printf("    carte roster (index->XUID) : ")
	for _, i := range di {
		fmt.Printf("%d->%d ", i, bestMap[i])
	}
	fmt.Println()
	// VENTILATION du non-recouvrement (len(dedup)-overlap) : distinguer les DOUBLONS (paire décodée en
	// excès mais existant dans chunk_27 = même kill sérialisé 2x, dédup insuffisante) des ERREURS (paire
	// décodée ABSENTE de chunk_27 = décodage faux). Le premier est un problème de couverture-gonflée, le
	// second un problème d'accuracy.
	decCnt := map[[2]uint64]int{}
	for _, k := range dedup {
		kx, ok1 := bestMap[k.killer]
		vx, ok2 := bestMap[k.victim]
		if ok1 && ok2 {
			decCnt[[2]uint64{kx, vx}]++
		}
	}
	dupExcess, wrongExcess := 0, 0
	for key, dc := range decCnt {
		cc := c27cnt[key]
		if dc > cc {
			if cc > 0 {
				dupExcess += dc - cc // paire réelle, mais comptée trop de fois = doublon
			} else {
				wrongExcess += dc // paire inexistante dans chunk_27 = erreur
			}
		}
	}
	fmt.Printf("    non-recouvert = %d : dont %d DOUBLONS (paire réelle sur-comptée) + %d ERREURS (paire absente de chunk_27)\n",
		len(dedup)-best, dupExcess, wrongExcess)
}

// runWeaponFeed : LIVRABLE production. chunk_27 EST le kill feed autoritatif (100% des killer/victim/
// temps) ; le décodage frame sert à ASSIGNER l'arme. Pour chaque kill chunk_27, on prend une arme
// non-consommée décodée pour la même paire (tueur,victime) mappée via la carte roster. Élimine le
// gonflage par doublons (le compte chunk_27 fait autorité) et produit le vrai livrable (arme par kill).
// La couverture-arme honnête = %kills chunk_27 avec une arme ; le reste = suicide/environnement (sans
// arme légitime) ou trou de capture (paquet fatal non décodé).
func runWeaponFeed(m string, h32 map[uint32]string) {
	dedup, _, _ := decodePipeline(m, h32)
	kv, nKills := chunk27KV(m)
	if len(kv) == 0 {
		fmt.Printf("=== WEAPON FEED %s : kill feed chunk_27 vide ===\n", m)
		return
	}
	c27 := make([][2]uint64, 0, len(kv))
	for _, p := range kv {
		kx, _ := strconv.ParseUint(p.KillerXUID, 10, 64)
		vx, _ := strconv.ParseUint(p.VictimXUID, 10, 64)
		c27 = append(c27, [2]uint64{kx, vx})
	}
	rmap, _, _ := solveRoster(dedup, c27)
	// armes décodées regroupées par paire XUID (mappée).
	byPair := map[[2]uint64][]string{}
	for _, k := range dedup {
		kx, ok1 := rmap[k.killer]
		vx, ok2 := rmap[k.victim]
		if ok1 && ok2 {
			byPair[[2]uint64{kx, vx}] = append(byPair[[2]uint64{kx, vx}], k.weapon)
		}
	}
	// assignation bijective : chaque kill chunk_27 consomme au plus une arme décodée de sa paire.
	used := map[[2]uint64]int{}
	nWeapon, nFire, nSelf := 0, 0, 0
	wd := map[string]int{}
	for _, p := range kv {
		kx, _ := strconv.ParseUint(p.KillerXUID, 10, 64)
		vx, _ := strconv.ParseUint(p.VictimXUID, 10, 64)
		if kx == vx {
			nSelf++
		}
		key := [2]uint64{kx, vx}
		i := used[key]
		if i < len(byPair[key]) {
			w := byPair[key][i]
			used[key]++
			nWeapon++
			wd[w]++
			if !strings.HasPrefix(w, "cause-") && !strings.HasPrefix(w, "fam-") {
				nFire++
			}
		}
	}
	covPct := float64(nWeapon) * 100 / float64(nKills)
	fmt.Printf("=== WEAPON FEED %s : %d kills chunk_27 (autoritatif) | %d appariés (tueur+victime) ===\n", m, nKills, len(kv))
	fmt.Printf(">>> ARME ASSIGNÉE : %d/%d = %.1f%% des kills (dont %d armes nommées) | non-assignés = suicide/environnement OU trou de capture\n",
		nWeapon, nKills, covPct, nFire)
	fmt.Printf("    self-kills chunk_27 : %d | armes : %s\n", nSelf, fmtHist(wd))
}

// theilSen ajuste y = a*x + b par l'estimateur de Theil-Sen (médiane des pentes par-paire) : ROBUSTE
// aux outliers (jusqu'à ~29%) et à la faible cardinalité, contrairement aux moindres carrés qui donnent
// des pentes aberrantes (voire négatives) sur 2-5 ancres bruitées. Les deux horloges (frame, jeu) étant
// monotones croissantes, une pente <=0 est rejetée (ok=false). Renvoie (a, b, ok).
func theilSen(xs, ys []float64) (a, b float64, ok bool) {
	if len(xs) < 2 {
		return 0, 0, false
	}
	var slopes []float64
	for i := 0; i < len(xs); i++ {
		for j := i + 1; j < len(xs); j++ {
			dx := xs[j] - xs[i]
			if dx == 0 {
				continue
			}
			slopes = append(slopes, (ys[j]-ys[i])/dx)
		}
	}
	if len(slopes) == 0 {
		return 0, 0, false
	}
	a = median(slopes)
	if a <= 0 {
		return 0, 0, false // horloges monotones croissantes : pente positive obligatoire.
	}
	inter := make([]float64, len(xs))
	for i := range xs {
		inter[i] = ys[i] - a*xs[i]
	}
	b = median(inter)
	return a, b, true
}

func median(v []float64) float64 {
	s := append([]float64(nil), v...)
	sort.Float64s(s)
	n := len(s)
	if n == 0 {
		return 0
	}
	if n%2 == 1 {
		return s[n/2]
	}
	return (s[n/2-1] + s[n/2]) / 2
}

// warpWeapon renvoie l'arme du coup fatal estimée par le flux de dégât : la DERNIÈRE arme-à-feu de
// l'attaquant atk à/avant l'instant cible (le coup fatal précède la mort), dans une fenêtre ; à défaut
// la plus proche. Le coup fatal étant le dernier tir avant la mort, « dernier avant » bat « le plus
// proche » quand l'attaquant tirait aussi sur une autre cible juste après. win = demi-fenêtre (unités
// frame) ; 0 = pas de contrainte de fenêtre.
func warpWeapon(dmgs []dmgEv, atk int, target, win float64) string {
	bestBefore := -math.MaxFloat64
	var wBefore string
	bestDT := math.MaxFloat64
	var wNear string
	for _, dg := range dmgs {
		if dg.atk != atk || !dg.firearm {
			continue
		}
		t := float64(dg.ts)
		dt := math.Abs(t - target)
		if win == 0 || dt <= win {
			if t <= target && t > bestBefore {
				bestBefore = t
				wBefore = dg.fam
			}
			if dt < bestDT {
				bestDT = dt
				wNear = dg.fam
			}
		}
	}
	if wBefore != "" {
		return wBefore
	}
	return wNear
}

// runPrecal : CALIBRATION PUR-FILM de la grammaire du préambule via l'oracle attacker==field1.
// Sur les paquets fataux (loadFatalPackets, curseur CE), pour les records ARME-À-FEU (variant=sfx trouvé
// par weaponAnchorLast avant le curseur), on connaît field1 (tueur, decodeKE). L'attaquant du record
// (readOpt5) est un offset FIXE avant la famille (famPos). On cherche la position g où keReadOpt(g)==field1,
// et on rapporte la distribution (famPos-g) — le mode = la structure du préambule. base = g - 10.
func runPrecal(m string, h32 map[uint32]string) {
	pkts := loadFatalPackets(m)
	attOff := map[int]int{}  // (famPos - gate attaquant) -> compte
	baseImp := map[int]int{} // base impliquée (= g - 10) -> compte
	n, matched := 0, 0
	for _, p := range pkts {
		famPos := weaponAnchorLast(p.pl, p.cursor)
		if famPos < 40 {
			continue
		}
		_, field1 := decodeKE(p.pl, p.cursor)
		if field1 < 0 {
			continue
		}
		n++
		// scan la porte de l'attaquant juste avant la famille : keReadOpt(g)==field1, g le plus proche.
		for g := famPos - 6; g >= famPos-18 && g >= 0; g-- {
			if v, _ := keReadOpt(p.pl, g); v == field1 {
				attOff[famPos-g]++
				baseImp[g-10]++
				matched++
				break
			}
		}
	}
	fmt.Printf("=== PRECAL %s : %d paquets firearm fataux, %d avec attaquant==field1 ===\n", m, n, matched)
	fmt.Printf("distribution (famPos - porte attaquant) : %s\n", sortedCounts2(attOff))
	fmt.Printf("base impliquée (porte-10) : %s\n", sortedCounts2(baseImp))
}

// sortedCounts2 : histogramme trié par compte décroissant (clé:compte).
func sortedCounts2(m map[int]int) string {
	type kv struct{ k, v int }
	var s []kv
	for k, v := range m {
		s = append(s, kv{k, v})
	}
	sort.Slice(s, func(i, j int) bool { return s[i].v > s[j].v })
	out := ""
	for i, e := range s {
		if i > 8 {
			break
		}
		out += fmt.Sprintf("%d:%d ", e.k, e.v)
	}
	return out
}

// runA0Probe : structure du stream ECS 0xA0 (container du dead-state). Distribution des tailles + de
// l'octet payload[1] (candidat = type de composant ; on cherche 0x23/0x28 = dead-state). Dump hex d'un
// échantillon. Prépare le walk ECS.
func runA0Probe(m string, h32 map[uint32]string) {
	cache := root + "/" + m
	szDist := map[int]int{}
	b1 := map[byte]int{}
	has2328 := 0
	shown := 0
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
			if typ != 0 || len(pl) == 0 || pl[0] != 0xA0 {
				continue
			}
			szDist[sz]++
			if len(pl) > 1 {
				b1[pl[1]]++
			}
			for _, x := range pl {
				if x == 0x23 || x == 0x28 {
					has2328++
					break
				}
			}
			if shown < 4 && sz < 60 {
				fmt.Printf("  sz=%d : % X\n", sz, pl[:min(sz, 40)])
				shown++
			}
		}
	}
	fmt.Printf("=== A0PROBE %s : paquets 0xA0 contenant octet 0x23/0x28 : %d ===\n", m, has2328)
	fmt.Printf("tailles les plus fréquentes : %s\n", sortedCounts(szDist))
	fmt.Printf("payload[1] (candidat type) : %s\n", sortedCountsB(b1))
}

func sortedCountsB(m map[byte]int) string {
	type kv struct {
		k byte
		v int
	}
	var s []kv
	for k, v := range m {
		s = append(s, kv{k, v})
	}
	sort.Slice(s, func(i, j int) bool { return s[i].v > s[j].v })
	out := ""
	for i, e := range s {
		if i > 10 {
			break
		}
		out += fmt.Sprintf("0x%02X:%d ", e.k, e.v)
	}
	return out
}

// runSrcProbe : le dead-state (grammaire FUN_140c1dd44) porte le TAG SOURCE (= la cause) dans le MÊME
// paquet fatal que le kill-event, en espace FAMILLE (pour firearm = la famille de l'arme). On calibre sa
// POSITION relative au curseur : pour chaque paquet fatal firearm (famille connue via weaponAnchorLast),
// on cherche l'offset d où bitsAt(cur+d,32) == famille. L'offset constant = la position du tag source.
func runSrcProbe(m string, h32 map[uint32]string) {
	pkts := loadFatalPackets(m)
	offAt := map[int]int{}
	n := 0
	for _, p := range pkts {
		fam := weaponAnchorLast(p.pl, p.cursor)
		if fam < 0 {
			continue
		}
		family := uint32(bitsAt(p.pl, fam, 32))
		n++
		for d := -120; d <= 260; d++ {
			pos := p.cursor + d
			if pos < 0 || pos+32 > len(p.pl)*8 {
				continue
			}
			if uint32(bitsAt(p.pl, pos, 32)) == family {
				offAt[d]++
			}
		}
	}
	fmt.Printf("=== SRCPROBE %s : %d paquets fataux firearm | offsets (cur+d) où un mot 32b == famille ===\n", m, n)
	fmt.Printf("%s\n", sortedCounts2(offAt))
}

// runC4Dead : parse les paquets 0xC4 (porteurs des dead-states) — pour chacun, trouve le 1er dead-state
// valide (P0=1 [tag32] flags8 G1=0 killer5 G2=0 victim5, killer/victim 0..7 distincts) et rapporte le TAG
// SOURCE (arme connue OU inconnu = candidat MÊLÉE/GRENADE). But : la cause de TOUTE mort, pur-film.
func runC4Dead(m string, h32 map[uint32]string) {
	cache := root + "/" + m
	nPkt, nHit := 0, 0
	tagN := map[string]int{}
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
			if typ != 0 || len(pl) == 0 || pl[0] != 0xC4 {
				continue
			}
			nPkt++
			for p := 0; p+53 <= len(pl)*8; p++ {
				if bitsAt(pl, p, 1) != 1 || bitsAt(pl, p+41, 1) != 0 || bitsAt(pl, p+47, 1) != 0 {
					continue
				}
				k := int(bitsAt(pl, p+42, 5))
				v := int(bitsAt(pl, p+48, 5))
				if k >= 8 || v >= 8 || k == v {
					continue
				}
				src := uint32(bitsAt(pl, p+1, 32))
				nm, ok := h32[src]
				if !ok {
					nm = fmt.Sprintf("?%08X", src)
				}
				tagN[nm]++
				nHit++
				break
			}
		}
	}
	fmt.Printf("=== C4DEAD %s : %d paquets 0xC4, %d avec dead-state valide ===\n", m, nPkt, nHit)
	fmt.Printf("tags source : %s\n", fmtHist(tagN))
}

// runE6Scan : les paquets 0xE6 (kill-events, ~= nb kills) contiennent-ils un TAG SOURCE (dead-state) ?
// Scan des tags famille CONNUS (source) dans les paquets 0xE6, à toute position. Si ~80 familles firearm
// apparaissent (1 par kill firearm), le tag source du dead-state est co-localisé avec le kill-event 0xE6.
func runE6Scan(m string, h32 map[uint32]string) {
	cache := root + "/" + m
	fam := map[uint32]int{}
	nPkt, withFam := 0, 0
	var sizes []int
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
			if typ != 0 || len(pl) == 0 || pl[0] != 0xE6 {
				continue
			}
			nPkt++
			if len(sizes) < 5 {
				sizes = append(sizes, sz)
			}
			found := false
			for b := 0; b+32 <= len(pl)*8; b++ {
				if nm, ok := h32[uint32(bitsAt(pl, b, 32))]; ok {
					fam[uint32(bitsAt(pl, b, 32))]++
					found = true
					_ = nm
				}
			}
			if found {
				withFam++
			}
		}
	}
	fmt.Printf("=== E6SCAN %s : %d paquets 0xE6, %d contiennent un tag famille connu | tailles ex %v ===\n", m, nPkt, withFam, sizes)
	for s, n := range fam {
		fmt.Printf("  %s:%d ", h32[s], n)
	}
	fmt.Println()
}

// runDeadScan : cherche le DEAD-STATE (déser FUN_140c1dd44) par sa signature sérialisée
// [présence 1b][tag source 32b][tueur R5][victime R5]. Le tag source = MÊME espace que la famille firearm
// (lecteur FUN_14080d69c). On valide un candidat quand source ∈ catalogue (firearm) OU quand (tueur,victime)
// sont des index 0..7 distincts plausibles. But : localiser le dead-state -> lire le tag source = la CAUSE
// de mêlée/grenade (tag hors catalogue). Scan sur tous les marqueurs.
// runMelHunt : le tag mêlée/grenade (Gravity Hammer 0x841ac5e5, Energy Sword 0x4ff3937e,
// Frag/Plasma/Dynamo grenade) apparaît-il QUELQUE PART dans le film (tous marqueurs, toutes
// positions bit), et sous quel marqueur ? But : fermer le trou "record de dégât mêlée sous un
// marqueur non-0xd2 que parsePreamble(24) skippe". Si le tag mêlée n'apparaît QUE sous 0xA0 (flux
// ECS delta = arme tenue rendue chaque frame = bruit held-weapon), il n'y a PAS de record de dégât
// mêlée. S'il apparaît sous un marqueur fatal (0xC3/0xC0/...) ~#morts-mêlée, c'est un candidat.
func runMelHunt(m string, h32 map[uint32]string) {
	cache := root + "/" + m
	// familles mêlée/grenade (high32) + id64 complets, dérivés du catalogue (DRY).
	melFamHi := map[uint32]string{}
	melID64 := map[uint64]string{}
	for id, n := range analysis.WeaponIDToName {
		if containsAny(n, []string{"Hammer", "Sword", "Grenade", "Bloodblade", "Diminisher"}) {
			melFamHi[uint32(id>>32)] = n
			melID64[id] = n
		}
	}
	fmt.Printf("=== MELHUNT %s : %d familles mêlée/grenade, %d id64 ===\n", m, len(melFamHi), len(melID64))
	type mk struct{ famHi, id64 int }
	byMk := map[byte]*mk{}
	famNameByMk := map[byte]map[string]int{}
	nFamTot, nID64Tot := 0, 0
	// hit id64 horodaté (même horloge frame que les kills 0xE6) pour corréler aux morts.
	type idHit struct {
		ts   uint64
		mk   byte
		name string
	}
	var idHits []idHit
	var kills []killEv
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
			if pl[0] == 0xE6 {
				kills = append(kills, killEv{int(bitsAt(pl, 83, 5)), int(bitsAt(pl, 88, 5)), ts})
			}
			e := byMk[pl[0]]
			if e == nil {
				e = &mk{}
				byMk[pl[0]] = e
				famNameByMk[pl[0]] = map[string]int{}
			}
			hitFam, hitID := false, false
			for b := 0; b+32 <= len(pl)*8; b++ {
				hi := uint32(bitsAt(pl, b, 32))
				if nm, ok := melFamHi[hi]; ok {
					hitFam = true
					famNameByMk[pl[0]][nm]++
				}
				if b+64 <= len(pl)*8 {
					id := (uint64(hi) << 32) | uint64(bitsAt(pl, b+32, 32))
					if nm, ok := melID64[id]; ok {
						if !hitID {
							idHits = append(idHits, idHit{ts, pl[0], nm})
						}
						hitID = true
					}
				}
			}
			if hitFam {
				e.famHi++
				nFamTot++
			}
			if hitID {
				e.id64++
				nID64Tot++
			}
		}
	}
	fmt.Printf("paquets contenant une FAMILLE mêlée/grenade (high32) : %d | un id64 complet : %d\n", nFamTot, nID64Tot)
	fmt.Println("par marqueur (famHi = #paquets avec famille ; id64 = #paquets avec id64 complet) :")
	for _, b := range []byte{0xD2, 0xC0, 0xC2, 0xC3, 0xCA, 0xD3, 0xE9, 0xA0, 0xC7, 0xE5, 0xE6} {
		if e := byMk[b]; e != nil && (e.famHi > 0 || e.id64 > 0) {
			var fams string
			for nm, n := range famNameByMk[b] {
				fams += fmt.Sprintf("%s:%d ", nm, n)
			}
			fmt.Printf("  0x%02X : famHi=%d id64=%d  {%s}\n", b, e.famHi, e.id64, fams)
		}
	}
	// CORRÉLATION : chaque hit id64 mêlée est-il proche (en ts frame) d'un kill 0xE6 ?
	// Un record de CAUSE mêlée doit tomber au clock de la mort. Un record d'entité arme
	// (spawn/pickup/tenue) est décorrélé des morts.
	sort.Slice(kills, func(i, j int) bool { return kills[i].ts < kills[j].ts })
	sort.Slice(idHits, func(i, j int) bool { return idHits[i].ts < idHits[j].ts })
	fmt.Printf("\n=== CORRÉLATION %d hits id64 mêlée vs %d kills 0xE6 (Δts frame, |Δ|<=3e6 ~ proche mort) ===\n", len(idHits), len(kills))
	nNear := 0
	for _, h := range idHits {
		best := int64(1 << 62)
		for _, k := range kills {
			dt := int64(h.ts) - int64(k.ts)
			if dt < 0 {
				dt = -dt
			}
			if dt < best {
				best = dt
			}
		}
		if best <= 3_000_000 {
			nNear++
		}
	}
	fmt.Printf("hits id64 proches d'un kill (|Δts|<=3e6) : %d/%d (%.0f%%)\n", nNear, len(idHits), 100*float64(nNear)/float64(max1(len(idHits))))
	// échantillon : 14 premiers hits, marqueur + Δ au kill le plus proche
	shown := 0
	for _, h := range idHits {
		if shown >= 14 {
			break
		}
		best := int64(1 << 62)
		for _, k := range kills {
			dt := int64(h.ts) - int64(k.ts)
			if dt < 0 {
				dt = -dt
			}
			if dt < best {
				best = dt
			}
		}
		fmt.Printf("  0x%02X %-18s ts=%d  Δkill=%.2fs\n", h.mk, h.name, h.ts, float64(best)/1e6)
		shown++
	}
}

func max1(n int) int {
	if n < 1 {
		return 1
	}
	return n
}

func containsAny(s string, subs []string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

func runDeadScan(m string, h32 map[uint32]string) {
	cache := root + "/" + m
	srcHits := map[uint32]int{}
	knownHits, mkHits := 0, map[byte]int{}
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
			// signature dead-state (grammaire FUN_140C1DD44) :
			// P0(1)=1 [tag32] flags(8) G1(1)=0 killer5 G2(1)=0 victim5
			for p := 0; p+53 <= len(pl)*8; p++ {
				if bitsAt(pl, p, 1) != 1 { // P0 : tag présent
					continue
				}
				src := uint32(bitsAt(pl, p+1, 32))
				if _, ok := h32[src]; !ok {
					continue // ancrer sur famille CONNUE (firearm) pour valider
				}
				if bitsAt(pl, p+41, 1) != 0 { // G1 : tueur présent (gate inverse : 0=présent)
					continue
				}
				k := int(bitsAt(pl, p+42, 5))
				if bitsAt(pl, p+47, 1) != 0 { // G2 : victime présente
					continue
				}
				v := int(bitsAt(pl, p+48, 5))
				if k < 8 && v < 8 && k != v {
					knownHits++
					srcHits[src]++
					mkHits[pl[0]]++
				}
			}
		}
	}
	fmt.Printf("=== DEADSCAN %s : %d signatures [présence|source connue|R5 k|R5 v] ===\n", m, knownHits)
	fmt.Printf("par marqueur : ")
	for mk, n := range mkHits {
		fmt.Printf("0x%02X:%d ", mk, n)
	}
	fmt.Println()
	fmt.Printf("sources (firearm) : ")
	for s, n := range srcHits {
		fmt.Printf("%s:%d ", h32[s], n)
	}
	fmt.Println()
}

// runKEAllScan : cherche un kill-event VALIDE dans TOUS les paquets type-0 (tous marqueurs, toutes
// tailles), pas seulement 0xd2/sz>=700. But : les kills MÊLÉE ont-ils un kill-event STANDALONE (petit
// paquet) que le détecteur (sz>=700) rate ? Compte par marqueur+tranche de taille.
func runKEAllScan(m string, h32 map[uint32]string) {
	cache := root + "/" + m
	byMk := map[byte]int{}
	small, big := 0, 0
	tot := 0
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
			if cur := locateKillEventCursor(pl); cur >= 0 {
				byMk[pl[0]]++
				tot++
				if sz < 700 {
					small++
				} else {
					big++
				}
			}
		}
	}
	fmt.Printf("=== KEALLSCAN %s : %d kill-events valides (sz>=700:%d sz<700:%d) ===\n", m, tot, big, small)
	fmt.Printf("par marqueur : ")
	for mk, n := range byMk {
		fmt.Printf("0x%02X:%d ", mk, n)
	}
	fmt.Println()
}

// runFamScan : distribution des FAMILLES (+0x10, S6 du préambule) des records 0xd2 via parsePreamble(24).
// La cause = la FAMILLE (le variant +0x14 est constant 0x42C9679F). Known firearm vs inconnu (candidat
// mêlée/grenade). Croise avec l'oracle align_dmg (9b191a7f = 9 armes-à-feu).
func runFamScan(m string, h32 map[uint32]string) {
	cache := root + "/" + m
	fam := map[uint32]int{}
	nRec, noFam := 0, 0
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
			if typ != 0 || len(pl) == 0 || pl[0] != 0xd2 {
				continue
			}
			nRec++
			rec, ok := parsePreamble(pl, 24)
			if !ok || rec.family == 0xffffffff {
				noFam++
				continue
			}
			fam[rec.family]++
		}
	}
	type kv struct {
		k uint32
		v int
	}
	var s []kv
	for k, v := range fam {
		s = append(s, kv{k, v})
	}
	sort.Slice(s, func(i, j int) bool { return s[i].v > s[j].v })
	fmt.Printf("=== FAMSCAN %s : %d records 0xd2, %d sans famille | %d familles distinctes ===\n", m, nRec, noFam, len(fam))
	for i, e := range s {
		if i > 20 {
			break
		}
		nm := h32[e.k]
		if nm == "" {
			nm = "??? INCONNU"
		}
		fmt.Printf("  %08X : %5d  %s\n", e.k, e.v, nm)
	}
}

// runKEFields : le kill-event embarqué porte-t-il l'ARME dans un de ses champs (field2/field5) ? Le kill
// feed du jeu affiche l'arme -> elle DOIT être dans le kill-event ou son consommateur. On dump field2/
// field5 et on compare à la famille d'arme du record de dégât fatal (weaponAnchorLast). Pur-film.
func runKEFields(m string, h32 map[uint32]string) {
	pkts := loadFatalPackets(m)
	nF2fam, nF5fam, nF2sfx, nF5sfx := 0, 0, 0, 0
	shown := 0
	for _, p := range pkts {
		cur := p.cursor
		_, b2 := keReadOpt(p.pl, cur) // field0 victime
		_, b3 := keReadOpt(p.pl, b2)  // field1 tueur
		f2 := uint32(bitsAt(p.pl, b3, 32))
		_, b5 := keReadOpt(p.pl, b3+33) // field3=R1 à b3+32, field4=assist à b3+33
		f5 := uint32(bitsAt(p.pl, b5, 32))
		famPos := weaponAnchorLast(p.pl, cur)
		var fam uint32
		if famPos >= 0 {
			fam = uint32(bitsAt(p.pl, famPos, 32))
		}
		if f2 == fam {
			nF2fam++
		}
		if f5 == fam {
			nF5fam++
		}
		if f2 == sfx {
			nF2sfx++
		}
		if f5 == sfx {
			nF5sfx++
		}
		// QUEUE : dump des R32 après field5, cherche la famille d'arme ou le suffixe sfx dans les ~12 champs
		// suivants (l'arme/cause est probablement dans la queue de la struct kill-event 0x28o).
		qFamHit, qSfxHit := -1, -1
		for i := 0; i < 20; i++ {
			pos := b5 + 32 + i*32
			if pos+32 > len(p.pl)*8 {
				break
			}
			v := uint32(bitsAt(p.pl, pos, 32))
			if fam != 0 && v == fam && qFamHit < 0 {
				qFamHit = i
			}
			if v == sfx && qSfxHit < 0 {
				qSfxHit = i
			}
		}
		if qFamHit >= 0 {
			nF2fam++ // réutilise le compteur : nb de paquets où la famille apparaît dans la queue
		}
		if shown < 8 {
			fmt.Printf("  cur=%d f2=%X f5=%X fam=%08X(%s) | queue: famAt=%d sfxAt=%d\n", cur, f2, f5, fam, h32[fam], qFamHit, qSfxHit)
			shown++
		}
		_ = nF5fam
		_ = nF2sfx
		_ = nF5sfx
	}
	fmt.Printf("=== KEFIELDS %s : %d paquets | famille trouvée dans la queue du kill-event : %d ===\n",
		m, len(pkts), nF2fam)
}

// runMarkerScan : pour CHAQUE marqueur de paquet type-0, tente parsePreamble(24) et compte les classes.
// But : les records mêlée/grenade sont-ils sous un AUTRE marqueur que 0xd2 (dégât mêlée = marqueur propre) ?
func runMarkerScan(m string, h32 map[uint32]string) {
	cache := root + "/" + m
	type cnt struct{ n, fire, mel, gren, skip int }
	byMk := map[byte]*cnt{}
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
			c := byMk[pl[0]]
			if c == nil {
				c = &cnt{}
				byMk[pl[0]] = c
			}
			c.n++
			rec, ok := parsePreamble(pl, 24)
			if !ok || rec.attacker < 0 {
				c.skip++
				continue
			}
			switch damageClass(rec.variant) {
			case dmgFirearm:
				c.fire++
			case dmgMelee:
				c.mel++
			case dmgGrenade:
				c.gren++
			}
		}
	}
	fmt.Printf("=== MARKERSCAN %s (parsePreamble base=24 par marqueur) ===\n", m)
	for _, mk := range []byte{0xD2, 0xC0, 0xC2, 0xC3, 0xCA, 0xD3, 0xE9, 0xA0, 0xC7, 0xE5, 0xE6} {
		if c := byMk[mk]; c != nil && c.n > 3 {
			fmt.Printf("  0x%02X : %d paquets | firearm=%d mêlée=%d grenade=%d skip=%d\n", mk, c.n, c.fire, c.mel, c.gren, c.skip)
		}
	}
}

// runSkipProbe : pour les paquets 0xd2 que parsePreamble(24) SKIPPE (attacker<0), examine leur structure :
// le variant à base+52 (position firearm-avec-famille) est-il mêlée/grenade ? La porte attaquant (bit 34)
// vaut-elle 1 (absent) ? Objectif : comprendre la grammaire du record mêlée/grenade.
func runSkipProbe(m string, h32 map[uint32]string) {
	cache := root + "/" + m
	nSkip := 0
	varAt := map[int]int{}      // classe du variant à base+52 pour les skippés
	attGate := map[uint32]int{} // valeur de la porte attaquant (bit 34) pour les skippés
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
			if typ != 0 || len(pl) == 0 || pl[0] != 0xd2 {
				continue
			}
			rec, ok := parsePreamble(pl, 24)
			if ok && rec.attacker >= 0 {
				continue // pas skippé
			}
			nSkip++
			attGate[uint32(bitsAt(pl, 34, 1))]++
			// variant à base+52 (=76) = position firearm-avec-famille
			if 76+32 <= len(pl)*8 {
				varAt[damageClass(uint32(bitsAt(pl, 76, 32)))]++
			}
		}
	}
	fmt.Printf("=== SKIPPROBE %s : %d paquets skippés ===\n", m, nSkip)
	fmt.Printf("porte attaquant (bit 34) : 0->%d 1->%d\n", attGate[0], attGate[1])
	fmt.Printf("classe du variant@bit76 : firearm=%d mêlée=%d grenade=%d autre=%d\n",
		varAt[dmgFirearm], varAt[dmgMelee], varAt[dmgGrenade], varAt[dmgOther])
}

// runVarScan : où (à quel bit) apparaissent les préfixes mêlée (0x592CF3) et grenade (0x164B3C) dans les
// paquets 0xd2 ? Scan brut de toutes les positions. Dit si les records mêlée/grenade EXISTENT dans ce flux
// et à quelle position structurelle (histogramme des positions).
func runVarScan(m string, h32 map[uint32]string) {
	cache := root + "/" + m
	melPos, grePos := map[int]int{}, map[int]int{}
	nPkt, pktMel, pktGre := 0, 0, 0
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
			if typ != 0 || len(pl) == 0 || pl[0] != 0xd2 {
				continue
			}
			nPkt++
			hasM, hasG := false, false
			for b := 0; b+32 <= len(pl)*8; b++ {
				v := uint32(bitsAt(pl, b, 32)) >> 8
				if v == meleeVar24 {
					melPos[b]++
					hasM = true
				} else if v == grenadeVar24 {
					grePos[b]++
					hasG = true
				}
			}
			if hasM {
				pktMel++
			}
			if hasG {
				pktGre++
			}
		}
	}
	fmt.Printf("=== VARSCAN %s : %d paquets 0xd2 | %d contiennent mêlée, %d grenade ===\n", m, nPkt, pktMel, pktGre)
	fmt.Printf("positions mêlée (0x592CF3) : %s\n", sortedCounts2(melPos))
	fmt.Printf("positions grenade (0x164B3C) : %s\n", sortedCounts2(grePos))
}

// runDmgClass : compte les records du flux de dégât par classe (arme-à-feu / mêlée / grenade / bruit) —
// diagnostic pour vérifier que collectRaw capte bien les records mêlée/grenade dans le petit 0xD2.
func runDmgClass(m string, h32 map[uint32]string) {
	_, dmgs := collectRaw(m, h32, 83)
	nMelee, nGren, nFire, nOther := 0, 0, 0, 0
	for _, d := range dmgs {
		switch {
		case d.fam == "Mêlée":
			nMelee++
		case d.fam == "Grenade":
			nGren++
		case d.firearm:
			nFire++
		default:
			nOther++
		}
	}
	fmt.Printf("=== DMGCLASS %s : %d records | firearm=%d mêlée=%d grenade=%d bruit=%d ===\n",
		m, len(dmgs), nFire, nMelee, nGren, nOther)
}

// runBaseScan : CALIBRATION du préambule. Pour chaque base candidate, parse tous les paquets 0xd2 et
// compte combien donnent un variant de classe CONNUE (firearm/mêlée/grenade). La base qui maximise =
// l'alignement correct du record. Sert à trouver la base film-indépendante.
func runBaseScan(m string, h32 map[uint32]string) {
	cache := root + "/" + m
	var pkts [][]byte
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
			if typ == 0 && len(pl) > 0 && pl[0] == 0xd2 {
				pkts = append(pkts, pl)
			}
		}
	}
	fmt.Printf("=== BASESCAN %s : %d paquets 0xd2 ===\n", m, len(pkts))
	for base := 0; base <= 48; base++ {
		fire, mel, gren := 0, 0, 0
		for _, pl := range pkts {
			if r, ok := parsePreamble(pl, base); ok {
				switch damageClass(r.variant) {
				case dmgFirearm:
					fire++
				case dmgMelee:
					mel++
				case dmgGrenade:
					gren++
				}
			}
		}
		if fire+mel+gren > len(pkts)/10 {
			fmt.Printf("  base=%2d : firearm=%d mêlée=%d grenade=%d (total connu=%d/%d)\n",
				base, fire, mel, gren, fire+mel+gren, len(pkts))
		}
	}
}

// feedKill = un kill du feed final : killer/victim (gamertags) + arme assignée + sa source.
type feedKill struct {
	TimeMS             int
	KillerGT, VictimGT string
	Weapon             string
	Source             string // same-clock | warp | fallback | none
}

// hybridResult = résultat complet de l'attribution hybride pour un match (métriques + feed par-kill).
type hybridResult struct {
	Match                           string
	NKills, Anchors                 int
	NSameClock, NWarp, NNone, NSelf int
	WarpAgree, WarpTot              int
	SlopeA                          float64
	Weapons                         map[string]int
	Feed                            []feedKill
}

// computeHybrid exécute toute l'attribution hybride (décodage same-clock + kill feed chunk_27 + carte
// roster + calibration frame<->jeu + warp calibré) et renvoie le feed par-kill + les métriques. Source
// UNIQUE pour runHybridFeed (affichage terminal) et runHybridDump (JSON pour la page HTML d'analyse).
func computeHybrid(m string, h32 map[uint32]string) hybridResult {
	res := hybridResult{Match: m, Weapons: map[string]int{}}
	dedup, _, _ := decodePipeline(m, h32)
	kv, nKills := chunk27KV(m)
	res.NKills = nKills
	if len(kv) == 0 {
		return res
	}
	c27 := make([][2]uint64, 0, len(kv))
	for _, p := range kv {
		kx, _ := strconv.ParseUint(p.KillerXUID, 10, 64)
		vx, _ := strconv.ParseUint(p.VictimXUID, 10, 64)
		c27 = append(c27, [2]uint64{kx, vx})
	}
	rmap, _, _ := solveRoster(dedup, c27)
	inv := map[uint64]int{} // XUID -> index frame
	for i, x := range rmap {
		inv[x] = i
	}
	_, dmgs := collectRaw(m, h32, 83) // flux de dégât per-hit (attaquant, arme, ts-frame)
	c27cnt := map[[2]uint64]int{}
	pairTimes := map[[2]uint64][]int{}
	for _, p := range kv {
		kx, _ := strconv.ParseUint(p.KillerXUID, 10, 64)
		vx, _ := strconv.ParseUint(p.VictimXUID, 10, 64)
		key := [2]uint64{kx, vx}
		c27cnt[key]++
		pairTimes[key] = append(pairTimes[key], int(p.TimeMS))
	}
	// source d'arme = kills FIABLES seulement (wRel) ; ancres = tous (temps fiable).
	byPair := map[[2]uint64][]string{}
	scFrames := map[[2]uint64][]float64{}
	for _, k := range dedup {
		kx, ok1 := rmap[k.killer]
		vx, ok2 := rmap[k.victim]
		if !ok1 || !ok2 {
			continue
		}
		key := [2]uint64{kx, vx}
		if k.wRel && !strings.HasPrefix(k.weapon, "cause-") {
			byPair[key] = append(byPair[key], k.weapon)
		}
		scFrames[key] = append(scFrames[key], float64(k.ts))
	}
	var ax, ay []float64
	for key, fts := range scFrames {
		gts := append([]int(nil), pairTimes[key]...)
		sort.Float64s(fts)
		sort.Ints(gts)
		n := min(len(fts), len(gts))
		for i := 0; i < n; i++ {
			ax = append(ax, fts[i])
			ay = append(ay, float64(gts[i]))
		}
	}
	a, b, calOK := theilSen(ax, ay)
	res.Anchors, res.SlopeA = len(ax), a
	win := 0.0
	if calOK {
		win = 3000 / a
	}
	usedSC := map[[2]uint64]int{}
	for _, p := range kv {
		kx, _ := strconv.ParseUint(p.KillerXUID, 10, 64)
		vx, _ := strconv.ParseUint(p.VictimXUID, 10, 64)
		if kx == vx {
			res.NSelf++
		}
		key := [2]uint64{kx, vx}
		fk := feedKill{TimeMS: int(p.TimeMS), KillerGT: p.KillerGT, VictimGT: p.VictimGT}
		if i := usedSC[key]; i < len(byPair[key]) {
			usedSC[key]++
			fk.Weapon, fk.Source = byPair[key][i], "same-clock"
			res.NSameClock++
		} else if ki, ok := inv[kx]; ok && calOK {
			// warp = source de dégât fatale du tueur (arme-à-feu/mêlée/grenade) la plus proche du kill.
			// PLUS de repli held-weapon (killerMainWeapon banni, doctrine) : non trouvé -> NON RÉSOLU.
			if w := warpWeapon(dmgs, ki, (float64(int(p.TimeMS))-b)/a, win); w != "" {
				fk.Weapon, fk.Source = w, "warp"
				res.NWarp++
			} else {
				fk.Source = "none"
				res.NNone++
			}
		} else {
			fk.Source = "none"
			res.NNone++
		}
		if fk.Weapon != "" {
			res.Weapons[fk.Weapon]++
		}
		res.Feed = append(res.Feed, fk)
	}
	// validation warp vs same-clock FIABLE (arme fatale sûre) : accuracy intrinsèque du warp.
	for _, k := range dedup {
		if !k.wRel || strings.HasPrefix(k.weapon, "cause-") || strings.HasPrefix(k.weapon, "fam-") {
			continue
		}
		w := warpWeapon(dmgs, k.killer, float64(k.ts), win)
		if w == "" {
			continue
		}
		res.WarpTot++
		if w == k.weapon {
			res.WarpAgree++
		}
	}
	return res
}

// runHybridFeed affiche le résumé de l'attribution hybride au terminal.
func runHybridFeed(m string, h32 map[uint32]string) {
	r := computeHybrid(m, h32)
	if r.NKills == 0 {
		fmt.Printf("=== HYBRID %s : kill feed chunk_27 vide ===\n", m)
		return
	}
	nAssigned := r.NSameClock + r.NWarp
	fmt.Printf("=== HYBRID %s : %d kills chunk_27 | calibration %d ancres (a=%.4g) ===\n", m, r.NKills, r.Anchors, r.SlopeA)
	fmt.Printf(">>> ARME ASSIGNÉE : %d/%d = %.1f%% (same-clock %d + warp %d) | non-assignés %d (dont self-kills %d)\n",
		nAssigned, r.NKills, float64(nAssigned)*100/float64(r.NKills), r.NSameClock, r.NWarp, r.NNone, r.NSelf)
	fmt.Printf("    armes : %s\n", fmtHist(r.Weapons))
	if r.WarpTot > 0 {
		fmt.Printf(">>> WARP vs same-clock FIABLE (accuracy warp sur kills à arme fatale sûre) : %d/%d = %.1f%%\n",
			r.WarpAgree, r.WarpTot, float64(r.WarpAgree)*100/float64(r.WarpTot))
	}
}

// runHybridDump émet le résultat hybride complet en JSON (pour construire la page HTML d'analyse).
func runHybridDump(m string, h32 map[uint32]string) {
	out, _ := json.MarshalIndent(computeHybrid(m, h32), "", "  ")
	fmt.Println(string(out))
}

// bestInjectionOverlap brute-force toutes les injections di->cx et renvoie le recouvrement multiset
// maximal + la carte gagnante. di,cx <= 8 -> <= 8! injections (instantané). Le recouvrement d'une
// injection = somme_type min(#paires_décodées_mappées[type], #paires_chunk27[type]).
func bestInjectionOverlap(dedup []pipeKill, di []int, cx []uint64, c27cnt map[[2]uint64]int) (int, map[int]uint64) {
	best := -1
	var bestMap map[int]uint64
	assign := make(map[int]uint64, len(di))
	used := make([]bool, len(cx))
	var rec func(pos int)
	rec = func(pos int) {
		if pos == len(di) {
			ov := overlapForMap(dedup, assign, c27cnt)
			if ov > best {
				best = ov
				bestMap = map[int]uint64{}
				for k, v := range assign {
					bestMap[k] = v
				}
			}
			return
		}
		for j := range cx {
			if used[j] {
				continue
			}
			used[j] = true
			assign[di[pos]] = cx[j]
			rec(pos + 1)
			delete(assign, di[pos])
			used[j] = false
		}
	}
	// si moins de XUID que d'indices, certaines injections sont impossibles : on autorise l'absence
	// (l'index non mappé ne contribue à aucun recouvrement).
	if len(cx) >= len(di) {
		rec(0)
	} else {
		best = greedyOverlap(dedup, di, cx, c27cnt, assign, &bestMap)
	}
	if best < 0 {
		best = 0
	}
	return best, bestMap
}

// overlapForMap calcule le recouvrement multiset pour une carte index->XUID donnée.
func overlapForMap(dedup []pipeKill, assign map[int]uint64, c27cnt map[[2]uint64]int) int {
	decCnt := map[[2]uint64]int{}
	for _, k := range dedup {
		kx, ok1 := assign[k.killer]
		vx, ok2 := assign[k.victim]
		if !ok1 || !ok2 {
			continue
		}
		decCnt[[2]uint64{kx, vx}]++
	}
	ov := 0
	for key, dc := range decCnt {
		if cc := c27cnt[key]; cc < dc {
			ov += cc
		} else {
			ov += dc
		}
	}
	return ov
}

// greedyOverlap : repli quand len(cx) < len(di) (plus d'indices frame que de XUID chunk_27, cas
// dégénéré rare). Assigne gloutonnement chaque index à son XUID le plus payant. Approximatif.
func greedyOverlap(dedup []pipeKill, di []int, cx []uint64, c27cnt map[[2]uint64]int, assign map[int]uint64, bestMap *map[int]uint64) int {
	used := make([]bool, len(cx))
	for _, i := range di {
		bestJ, bestGain := -1, -1
		for j := range cx {
			if used[j] {
				continue
			}
			assign[i] = cx[j]
			g := overlapForMap(dedup, assign, c27cnt)
			delete(assign, i)
			if g > bestGain {
				bestGain, bestJ = g, j
			}
		}
		if bestJ >= 0 {
			used[bestJ] = true
			assign[i] = cx[bestJ]
		}
	}
	*bestMap = map[int]uint64{}
	for k, v := range assign {
		(*bestMap)[k] = v
	}
	return overlapForMap(dedup, assign, c27cnt)
}
