// tmp_dmgsource — MISSION E1 : PREUVE sur le film (Go) de la source de dégât par kill.
//
// Objectif : pour le match 000d5950, sortir PAR KILL :
//   - tueur, victime, temps (kill feed chunk_27, 93/93)
//   - kill medalType (b[59] du kill event chunk_27 = proxy de méthode/source côté jeu)
//   - SOURCE-ID BRUT : le record de dégât sérialisé FUN_14080c1f8 (variant_name@+0x14 /
//     'weap'@+0x18) n'est PAS localisé dans le flux (gap unique R1/R2). Faute de ce record,
//     le MEILLEUR PROXY ENREGISTRÉ et déjà prouvé OFFLINE = les littéraux d'arme id64
//     (high32|low32 ∈ analysis.WeaponIDToName) décodés du flux type-0, horodatés. On prend
//     le(s) littéral(aux) le(s) plus proche(s) en temps du kill (fenêtre AVANT le kill).
//   - MÉTHODE dead-state : le dead-state CLEAN (DesyncAt==-1, Mort==true) le plus proche en
//     temps. On expose EnumA/EnumB (victim/killer datum-handle, R3), Val0c (catégorie
//     mêlée 0x40000 / lancé 0x10001 selon RE_FINDINGS §1) et GlobalID.
//
// VERDICT attendu : quantifie la couverture par type (arme à feu / grenade / mêlée / terrain)
// et dit honnêtement OUI/PARTIEL/NON + ce qui manque.
//
// Usage : tmp_dmgsource [maxChunk] [proxyWindowMs]
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sort"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/filmdec"
)

const cache = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`
const t0Us = uint64(4537898226)

var calCfg = filmdec.FrameConfig{HasExtraFields: false, IDLowBits: 11}
var bipedSlots = map[uint32]bool{512: true, 513: true, 514: true, 515: true, 516: true, 517: true, 518: true, 519: true}

// map xuid->gamertag (film 000d5950, bit-vérifiée, cf HANDOFF).
var xuidGamertag = map[uint64]string{
	2535467794760703: "whiteknight2519",
	2535437947245250: "JAVIERLOLITO540",
	2533274823110022: "JGtm",
	2533274980284321: "LORD PEINX13",
	2533274815845110: "IKE ILYA",
	2535444178793711: "Akatsuki fire17",
	2533274882097883: "aldusbroncus",
	2533274826120416: "VitaminA1688",
}

// kills narrés connus (vérité-terrain partielle). clé = (tueurGT, victimeGT) -> arme/methode attendue.
type narrKey struct{ killer, victim string }

var narrated = map[narrKey]string{
	{"IKE ILYA", "JGtm"}:             "MARTEAU (gravity hammer)",
	{"JGtm", "Akatsuki fire17"}:      "BR75",
	{"aldusbroncus", "LORD PEINX13"}: "PLASMA (5:43-44)",
	{"JGtm", "VitaminA1688"}:         "(JGtm narré)",
}

var h32set = map[uint32]bool{}
var id64name = map[uint64]string{}
var h32name = map[uint32]string{}  // family high32 -> weapon/melee/grenade name (same tag space)
var lo32name = map[uint32]string{} // low32 variant -> name (for the odd non-session-tag variants)

func buildWeaponSets() {
	for id, n := range analysis.WeaponIDToName {
		h32set[uint32(id>>32)] = true
		id64name[id] = n
		h32name[uint32(id>>32)] = n
		if lo := uint32(id); lo != 0x42c9679f { // 0x42c9679f = session tag (shared by all base weapons)
			lo32name[lo] = n
		}
	}
}

// tagName resolves a dead-state srcTag against the family tag space (the SAME space
// as the firearm family read from the 0xd2 damage deser). Melee (hammer/sword) and
// grenades ARE family tags here, so a resolved srcTag names a melee/grenade cause
// purely from film. Falls back to the low32 variant space, then a 1-bit-shift alias
// (the taxonomy stores some ids shifted), else the raw hex.
func tagName(t uint32) string {
	if t == 0xFFFFFFFF {
		return "-"
	}
	if n, ok := h32name[t]; ok {
		return n
	}
	if n, ok := lo32name[t]; ok {
		return n + "~lo"
	}
	if n, ok := h32name[t>>1]; ok {
		return n + "~sh"
	}
	return fmt.Sprintf("?%08x", t)
}

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
	ts      uint64
	payload []byte
}

func listFrames(d []byte) []packet {
	var out []packet
	off := 0
	for off+16 <= len(d) {
		typ := binary.LittleEndian.Uint16(d[off:])
		sz := int(binary.LittleEndian.Uint32(d[off+4:]))
		ts := binary.LittleEndian.Uint64(d[off+8:])
		if sz < 0 || off+16+sz > len(d) {
			break
		}
		if typ == 0 {
			out = append(out, packet{ts, d[off+16 : off+16+sz]})
		}
		off += 16 + sz
	}
	return out
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

// litEvt = un littéral d'arme id64 décodé du flux, horodaté.
type litEvt struct {
	timeMs int
	fam    string
	id64   uint64
}

// deadObs = un dead-state CLEAN Mort==true capturé.
type deadObs struct {
	timeMs int
	slot   uint32
	ds     filmdec.DeadState
}

// kfRow = un kill apparié (kill feed chunk_27).
type kfRow struct {
	killer, victim uint64
	t              int
	medalType      int
}

func main() {
	maxChunk := 26
	proxyWin := 3000 // ms : fenêtre AVANT le kill où on cherche un littéral d'arme proxy
	if len(os.Args) >= 2 {
		fmt.Sscanf(os.Args[1], "%d", &maxChunk)
	}
	if len(os.Args) >= 3 {
		fmt.Sscanf(os.Args[2], "%d", &proxyWin)
	}
	filmdec.SetRecordStateParam(2)
	buildWeaponSets()
	reg, err := filmdec.ParseRegistryChunk(inflate(cache + "/chunk_00.bin"))
	if err != nil {
		panic(err)
	}

	// ── 1) KILL FEED chunk_27 : tueur, victime, temps, medalType ──
	gt := map[uint64]string{} // xuidGamertag (hardcodé) couvre les 8 joueurs du film 000d5950
	events, _ := analysis.ParseHighlightEvents(mustRead(cache+"/chunk_27.bin"), 0)
	type ev struct {
		xuid      uint64
		t         int
		medalType int
	}
	var kills, deaths []ev
	for _, e := range events {
		switch e.EventType {
		case analysis.EventTypeKill:
			kills = append(kills, ev{e.XUID, e.TimeMS, e.MedalType})
		case analysis.EventTypeDeath:
			deaths = append(deaths, ev{e.XUID, e.TimeMS, e.MedalType})
		}
	}
	sort.Slice(kills, func(i, j int) bool { return kills[i].t < kills[j].t })
	sort.Slice(deaths, func(i, j int) bool { return deaths[i].t < deaths[j].t })
	var feed []kfRow
	usedD := make([]bool, len(deaths))
	for _, k := range kills {
		best, bd := -1, 400
		for i, d := range deaths {
			if usedD[i] || d.xuid == k.xuid {
				continue
			}
			dt := k.t - d.t
			if dt < 0 {
				dt = -dt
			}
			if dt < bd {
				bd, best = dt, i
			}
		}
		if best >= 0 {
			usedD[best] = true
			feed = append(feed, kfRow{k.xuid, deaths[best].xuid, k.t, k.medalType})
		}
	}
	fmt.Printf("=== KILL FEED chunk_27 : %d kills appaires (sur %d kills, %d deaths) ===\n", len(feed), len(kills), len(deaths))

	// distribution des medalType sur les kills (proxy méthode côté jeu)
	mt := map[int]int{}
	for _, f := range feed {
		mt[f.medalType]++
	}
	fmt.Printf("    distribution medalType des kills : ")
	type kvI struct{ k, v int }
	var mtArr []kvI
	for k, v := range mt {
		mtArr = append(mtArr, kvI{k, v})
	}
	sort.Slice(mtArr, func(a, b int) bool { return mtArr[a].v > mtArr[b].v })
	for _, e := range mtArr {
		fmt.Printf("mt=%d:%d ", e.k, e.v)
	}
	fmt.Println()

	// ── 2) SCAN BRUT type-0 : littéraux d'arme id64 horodatés (proxy source enregistré) ──
	var lits []litEvt
	for idx := 2; idx <= maxChunk; idx++ {
		for _, fr := range listFrames(inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, idx))) {
			tms := int((fr.ts - t0Us) / 1000)
			mx := len(fr.payload)*8 - 64
			for bp := 0; bp <= mx; bp++ {
				h := uint32(bitsAt(fr.payload, bp, 32))
				if !h32set[h] {
					continue
				}
				low := uint32(bitsAt(fr.payload, bp+32, 32))
				id := (uint64(h) << 32) | uint64(low)
				if n, ok := id64name[id]; ok {
					lits = append(lits, litEvt{tms, n, id})
					bp += 63
				}
			}
		}
	}
	sort.Slice(lits, func(i, j int) bool { return lits[i].timeMs < lits[j].timeMs })
	fmt.Printf("=== SCAN BRUT : %d littéraux d'arme id64 décodés du film (type-0) ===\n", len(lits))

	// ── 3) DEAD-STATES CLEAN Mort==true (méthode dead-state) ──
	var dos []deadObs
	for idx := 2; idx <= maxChunk; idx++ {
		for _, fr := range listFrames(inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, idx))) {
			w := freshWorld(reg)
			br := filmdec.NewBitReader(fr.payload)
			recs, _ := filmdec.DecodeFrameRecords(br, w, calCfg)
			tms := int((fr.ts - t0Us) / 1000)
			for _, r := range recs {
				if !bipedSlots[r.Slot] || r.DesyncAt != -1 || r.Trace.Dead == nil || !r.Trace.Dead.Mort {
					continue
				}
				dos = append(dos, deadObs{tms, r.Slot, *r.Trace.Dead})
			}
		}
	}
	sort.Slice(dos, func(i, j int) bool { return dos[i].timeMs < dos[j].timeMs })
	fmt.Printf("=== DEAD-STATES CLEAN Mort==true : %d obs ===\n", len(dos))

	// distribution des srcTags résolus sur TOUS les dead-states clean : si +0x00 ou +0x4c
	// est le tag de cause, sa distribution doit tomber majoritairement sur des noms d'armes
	// (mêlée/grenade/arme) et non sur du bruit hex.
	st0, st4c := map[string]int{}, map[string]int{}
	var nRes0, nRes4c int
	for _, d := range dos {
		n0, n4c := tagName(d.ds.SrcTag0), tagName(d.ds.SrcTag4c)
		st0[n0]++
		st4c[n4c]++
		if n0 != "-" && n0[0] != '?' {
			nRes0++
		}
		if n4c != "-" && n4c[0] != '?' {
			nRes4c++
		}
	}
	fmt.Printf("  SrcTag0  résolus en arme : %d/%d ; SrcTag4c résolus en arme : %d/%d\n", nRes0, len(dos), nRes4c, len(dos))
	printTagDist("SrcTag0 (+0x00)", st0)
	printTagDist("SrcTag4c (+0x4c)", st4c)
	fmt.Println()

	// ── 4) TABLEAU PAR KILL ──
	fmt.Println("=== TABLEAU PAR KILL : tueur -> victime | t | medalType | source-id BRUT (proxy littéral) | méthode dead-state ===")
	var (
		nProxy       int // kills avec au moins 1 littéral d'arme dans la fenêtre
		nDead        int // kills avec dead-state apparié (±400ms)
		nProxyUnique int // kills où 1 SEULE famille proxy dans la fenêtre (non ambigu)
		nNarrCorrect int
		nNarrChecked int
	)
	for _, r := range feed {
		killerGT := nameOf(gt, r.killer)
		victimGT := nameOf(gt, r.victim)

		// 4a) proxy : littéraux d'arme dans [t-proxyWin, t+200], les plus proches du kill.
		famWin := map[string]int{}
		var bestLit *litEvt
		bestDt := 1 << 30
		for i := range lits {
			l := &lits[i]
			if l.timeMs < r.t-proxyWin || l.timeMs > r.t+200 {
				continue
			}
			famWin[l.fam]++
			dt := r.t - l.timeMs
			if dt < 0 {
				dt = -dt
			}
			if dt < bestDt {
				bestDt = dt
				bestLit = l
			}
		}
		proxyStr := "(aucun littéral dans la fenêtre)"
		if bestLit != nil {
			nProxy++
			if len(famWin) == 1 {
				nProxyUnique++
			}
			proxyStr = fmt.Sprintf("id64=0x%016x %s (Δ%dms, %d fam distinctes en fenêtre)", bestLit.id64, bestLit.fam, bestDt, len(famWin))
		}

		// 4b) méthode dead-state : dead-state CLEAN Mort le plus proche en temps (±400ms).
		deadStr := "(pas de dead-state clean apparié)"
		var bestDs *deadObs
		dsDt := 1 << 30
		for i := range dos {
			d := &dos[i]
			dt := r.t - d.timeMs
			if dt < 0 {
				dt = -dt
			}
			if dt < dsDt {
				dsDt = dt
				bestDs = d
			}
		}
		if bestDs != nil && dsDt <= 400 {
			nDead++
			deadStr = fmt.Sprintf("slot%d Δ%dms E=%d/%d GID=%08x | SrcTag0=%08x[%s] SrcTag4c=%08x[%s]",
				bestDs.slot, dsDt, bestDs.ds.EnumA, bestDs.ds.EnumB, bestDs.ds.GlobalID,
				bestDs.ds.SrcTag0, tagName(bestDs.ds.SrcTag0), bestDs.ds.SrcTag4c, tagName(bestDs.ds.SrcTag4c))
		}

		// 4c) marquage narré
		narr := ""
		if want, ok := narrated[narrKey{killerGT, victimGT}]; ok {
			nNarrChecked++
			narr = " <<< NARRÉ: " + want
			if bestLit != nil && famMatchesNarr(bestLit.fam, want) {
				nNarrCorrect++
				narr += " [PROXY MATCH]"
			}
		}

		fmt.Printf("  %6.1fs %-16s -> %-16s | mt=%-3d | %s | %s%s\n",
			float64(r.t)/1000, killerGT, victimGT, r.medalType, proxyStr, deadStr, narr)
	}

	// ── 4bis) FOCUS kills narrés : tous les littéraux ±1.5s (montre pourquoi le proxy échoue) ──
	fmt.Println("\n=== FOCUS kills narrés : littéraux d'arme dans ±1500ms autour du kill ===")
	for _, r := range feed {
		kGT, vGT := nameOf(gt, r.killer), nameOf(gt, r.victim)
		want, ok := narrated[narrKey{kGT, vGT}]
		if !ok {
			continue
		}
		fmt.Printf("  %6.1fs %s -> %s  [attendu: %s]\n", float64(r.t)/1000, kGT, vGT, want)
		found := map[string]bool{}
		for i := range lits {
			l := &lits[i]
			if l.timeMs < r.t-1500 || l.timeMs > r.t+1500 {
				continue
			}
			if found[fmt.Sprintf("%d/%s", l.timeMs, l.fam)] {
				continue
			}
			found[fmt.Sprintf("%d/%s", l.timeMs, l.fam)] = true
			fmt.Printf("       Δ%+5dms  %s\n", l.timeMs-r.t, l.fam)
		}
	}

	// ── 5) BILAN ──
	fmt.Printf("\n=== BILAN COUVERTURE (sur %d kills appariés) ===\n", len(feed))
	fmt.Printf("  proxy littéral d'arme présent dans la fenêtre [-%dms,+200ms] : %d/%d (%.0f%%)\n",
		proxyWin, nProxy, len(feed), pct(nProxy, len(feed)))
	fmt.Printf("    dont fenêtre NON ambiguë (1 seule famille) : %d/%d (%.0f%%)\n", nProxyUnique, len(feed), pct(nProxyUnique, len(feed)))
	fmt.Printf("  dead-state clean apparié (±400ms)            : %d/%d (%.0f%%)\n", nDead, len(feed), pct(nDead, len(feed)))
	fmt.Printf("  kills narrés vérifiés                        : %d ; proxy match : %d\n", nNarrChecked, nNarrCorrect)
}

// dsCategory mappe Val0c (R(4) inline) vers la catégorie RE (mêlée 0x40000 / lancé 0x10001).
// NB : Val0c = 4 bits seulement (high de la struct RAM), donc on ne distingue que des petites
// valeurs ; affiché brut pour inspection.
func dsCategory(v uint8) string {
	switch v {
	case 0:
		return "v0c=0"
	default:
		return fmt.Sprintf("v0c=%d", v)
	}
}

// printTagDist affiche la distribution des noms de tags (top 16, tri décroissant).
func printTagDist(label string, m map[string]int) {
	type kv struct {
		k string
		n int
	}
	var kvs []kv
	for k, n := range m {
		kvs = append(kvs, kv{k, n})
	}
	sort.Slice(kvs, func(i, j int) bool { return kvs[i].n > kvs[j].n })
	fmt.Printf("  %s : ", label)
	for i, k := range kvs {
		if i >= 16 {
			fmt.Printf("... (+%d)", len(kvs)-16)
			break
		}
		fmt.Printf("%s=%d  ", k.k, k.n)
	}
	fmt.Println()
}

func famMatchesNarr(fam, narr string) bool {
	// match grossier sur mots-clés
	switch {
	case contains(narr, "MARTEAU"):
		return contains(fam, "Hammer")
	case contains(narr, "BR75"):
		return contains(fam, "BR75")
	case contains(narr, "PLASMA"):
		return contains(fam, "Plasma") || contains(fam, "Pulse") || contains(fam, "Disruptor")
	}
	return false
}

func contains(s, sub string) bool { return bytes.Contains([]byte(s), []byte(sub)) }

func pct(a, b int) float64 {
	if b == 0 {
		return 0
	}
	return 100 * float64(a) / float64(b)
}

func freshWorld(reg *filmdec.Registry) *filmdec.World {
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

func nameOf(gt map[uint64]string, x uint64) string {
	if g, ok := xuidGamertag[x]; ok {
		return g
	}
	if g, ok := gt[x]; ok && g != "" {
		return g
	}
	return fmt.Sprintf("xuid:%d", x)
}

func mustRead(p string) []byte { b, _ := os.ReadFile(p); return b }
