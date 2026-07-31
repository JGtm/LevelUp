// tmp_acurtis — TEST (hypothèse, à valider) de la méthode acurtis : la cause mêlée/
// grenade est dans des records d'ÉVÉNEMENT dédiés des frame packets, avec marqueurs propres :
//
//	Grenade : marqueur 0x4c0c00 (24b, bit-unaligned) + id32 arme + player index 5b.
//	Mêlée   : marqueur 0x532 (11b) ; anchor=+3 ; type uint8 @+76 in {0x42,0x47,0x60} ;
//	          weapon id @+offset(type) ; player index @+20 ; nibble anim avant l'arme.
//
// On NE fait PAS confiance aveugle : on valide contre chunk_27 (kill feed) + kills narrés.
// Diagnostics lourds pour vérifier/recaler les offsets.
//
// Usage : CGO_ENABLED=0 go run ./cmd/tmp_acurtis [film] [mode]
//
//	mode: melee (defaut) | gren | both
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
)

const root = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks`

// t0 (frame-ts origine) pour convertir en ms — repris de tmp_dmgsource.
const t0Us = uint64(4537898226)

// grenade ids (acurtis, "premiers 32 bits").
var grenadeIDs = map[uint32]string{
	0xB0171062: "Frag Grenade",
	0xC0E34C44: "Plasma Grenade",
	0x3B2567D4: "Shock Grenade",
	0x9212E428: "Spike Grenade",
}

// melee weapon offsets par type (acurtis), relatifs à l'anchor.
var meleeOffsets = map[uint32][]int{
	0x42: {88},       // arme non-mêlée miss / épée non-powered
	0x47: {86},       // gravity hammer powered
	0x60: {101, 103}, // épée powered hit / unpowered hit
}

var xuidGamertag = map[uint64]string{
	2535467794760703: "whiteknight2519", 2535437947245250: "JAVIERLOLITO540",
	2533274823110022: "JGtm", 2533274980284321: "LORD PEINX13",
	2533274815845110: "IKE ILYA", 2535444178793711: "Akatsuki fire17",
	2533274882097883: "aldusbroncus", 2533274826120416: "VitaminA1688",
}

// h32name : family high32 -> nom d'arme (catalogue).
var h32name = map[uint32]string{}

func buildCatalog() {
	for id, n := range analysis.WeaponIDToName {
		h32name[uint32(id>>32)] = n
	}
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

// loadPackets liste tous les frame packets type-0 (ts + marqueur + payload) des chunks 0..41.
func loadPackets(film string) []pkt {
	cache := root + "/" + film
	var out []pkt
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
			if typ == 0 && len(pl) > 0 {
				out = append(out, pkt{ts, pl[0], pl})
			}
		}
	}
	return out
}

// bitsAt lit n bits MSB-first à la position bp.
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

type pkt struct {
	ts     uint64
	marker byte
	pl     []byte
}

type meleeEvt struct {
	tms    int
	marker byte
	player int
	typ    uint32
	off    int
	fam    uint32
	weapon string
	anim   uint32
}

type grenEvt struct {
	tms    int
	player int
	weapon string
}

func main() {
	film := "000d5950"
	mode := "both"
	if len(os.Args) >= 2 {
		film = os.Args[1]
	}
	if len(os.Args) >= 3 {
		mode = os.Args[2]
	}
	buildCatalog()
	pkts := loadPackets(film)
	fmt.Printf("=== ACURTIS %s : %d frame packets type-0 ===\n", film, len(pkts))

	if mode == "sweep" {
		runMeleeSweep(pkts)
		return
	}
	if mode == "feed" {
		runFeed(pkts)
		return
	}
	if mode == "struct" {
		runStruct(pkts)
		return
	}
	if mode == "kecause" {
		runKeCause(pkts)
		return
	}
	if mode == "find" {
		// arg3 = id64 hex (defaut = Mk51 Sidekick de l'exemple acurtis)
		id := uint64(0xf408190f42c9679f)
		if len(os.Args) >= 4 {
			fmt.Sscanf(os.Args[3], "%x", &id)
		}
		runFind(pkts, id)
		return
	}
	if mode == "melee" || mode == "both" {
		runMelee(pkts)
	}
	if mode == "gren" || mode == "both" {
		runGren(pkts)
	}
}

// meleeE = événement mêlée horodaté (arme + player candidat @bit20).
type meleeE struct {
	tms    int
	weapon string
	player int
}

// grenE = lancer de grenade horodaté (arme + thrower @+79).
type grenE struct {
	tms    int
	weapon string
	player int
}

// id64name : id64 complet -> nom (distingue les variantes, ex Diminisher vs Rushdown vs Gravity).
var id64name = map[uint64]string{}

func collectMelee(pkts []pkt) []meleeE {
	if len(id64name) == 0 {
		for id, n := range analysis.WeaponIDToName {
			id64name[id] = n
		}
	}
	var out []meleeE
	for _, p := range pkts {
		if p.marker != 0xD3 {
			continue
		}
		tms := int((p.ts - t0Us) / 1000)
		nb := len(p.pl)*8 - 64
		for bp := 0; bp <= nb; bp++ {
			hi := uint32(bitsAt(p.pl, bp, 32))
			if _, ok := h32name[hi]; !ok {
				continue
			}
			id := (uint64(hi) << 32) | uint64(bitsAt(p.pl, bp+32, 32))
			nm, ok := id64name[id]
			if !ok {
				continue // low32 non catalogué -> pas un vrai id64 d'arme
			}
			out = append(out, meleeE{tms: tms, weapon: nm, player: int(bitsAt(p.pl, 20, 5))})
			bp += 63
		}
	}
	return out
}

func collectGren(pkts []pkt) []grenE {
	var out []grenE
	for _, p := range pkts {
		tms := int((p.ts - t0Us) / 1000)
		nb := len(p.pl)*8 - 120
		for bp := 0; bp <= nb; bp++ {
			if bitsAt(p.pl, bp, 24) != 0x4c0c00 {
				continue
			}
			id := uint32(bitsAt(p.pl, bp+24, 32))
			if nm, ok := grenadeIDs[id]; ok {
				// player index = marker+24 (fin marqueur) + 32 (id) + 47 = bp+103 (validé sweep 0-7).
				out = append(out, grenE{tms: tms, weapon: nm, player: int(bitsAt(p.pl, bp+24+32+47, 5))})
				bp += 24
			}
		}
	}
	return out
}

// kfKill = un kill apparié du kill feed chunk_27.
type kfKill struct {
	killer, victim uint64
	tms            int
}

func loadKills(film string) []kfKill {
	data, _ := os.ReadFile(root + "/" + film + "/chunk_27.bin")
	events, _ := analysis.ParseHighlightEvents(data, 0)
	type ev struct {
		xuid uint64
		t    int
	}
	var kills, deaths []ev
	for _, e := range events {
		switch e.EventType {
		case analysis.EventTypeKill:
			kills = append(kills, ev{e.XUID, e.TimeMS})
		case analysis.EventTypeDeath:
			deaths = append(deaths, ev{e.XUID, e.TimeMS})
		}
	}
	sort.Slice(kills, func(i, j int) bool { return kills[i].t < kills[j].t })
	sort.Slice(deaths, func(i, j int) bool { return deaths[i].t < deaths[j].t })
	var out []kfKill
	used := make([]bool, len(deaths))
	for _, k := range kills {
		best, bd := -1, 400
		for i, d := range deaths {
			if used[i] || d.xuid == k.xuid {
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
			used[best] = true
			out = append(out, kfKill{k.xuid, deaths[best].xuid, k.t})
		}
	}
	return out
}

func nameOf(x uint64) string {
	if g, ok := xuidGamertag[x]; ok {
		return g
	}
	return fmt.Sprintf("xuid:%d", x)
}

var narrated = map[[2]string]string{
	{"LORD PEINX13", "Akatsuki fire17"}: "MARTEAU Diminisher (147.6s)",
	{"Akatsuki fire17", "LORD PEINX13"}: "MARTEAU Rushdown (323.9s)",
	{"aldusbroncus", "LORD PEINX13"}:    "MARTEAU Rushdown (344.6s)",
}

// runFeed : pour chaque kill (chunk_27), cherche l'événement mêlée/grenade le plus proche en temps
// (±win ms). But : attribuer l'arme mêlée/grenade par-kill et VALIDER contre les kills narrés.
// runFeed : ATTRIBUTION PAR-KILL UNIFIÉE. Pour chaque kill (chunk_27 = tueur/victime/temps
// autoritatif) on classe MÊLÉE / GRENADE / FIREARM par corrélation temporelle aux flux d'events :
//
//	MÊLÉE   : event 0xD3 id64 à Δ<=meleeWin (mêlée = instantanée -> fenêtre serrée).
//	GRENADE : lancer 0x4c0c00 dans [kill-grenWin, kill+200] (le projectile vole puis explose).
//	FIREARM : défaut (arme = famille du 0xd2 même-horloge, résolue par le pipeline tmp_kwval).
//
// Priorité mêlée (serrée) > grenade > firearm. Valide contre les kills marteau narrés.
func runFeed(pkts []pkt) {
	meleeWin, grenWin := 300, 2500
	if len(os.Args) >= 4 {
		fmt.Sscanf(os.Args[3], "%d", &meleeWin)
	}
	kills := loadKills("000d5950")
	melee := collectMelee(pkts)
	gren := collectGren(pkts)
	fmt.Printf("\n=== ATTRIBUTION UNIFIÉE : %d kills | %d events mêlée, %d grenades (mêlée ±%dms, grenade [-%d,+200]ms) ===\n",
		len(kills), len(melee), len(gren), meleeWin, grenWin)
	nMel, nGren, nFire := 0, 0, 0
	nNarr, nNarrOK := 0, 0
	for _, k := range kills {
		kGT, vGT := nameOf(k.killer), nameOf(k.victim)
		// mêlée la plus proche (fenêtre serrée)
		var mBest *meleeE
		mdt := meleeWin + 1
		for i := range melee {
			if dt := abs(k.tms - melee[i].tms); dt <= mdt {
				mdt, mBest = dt, &melee[i]
			}
		}
		// grenade : lancer AVANT le kill (délai vol+explosion), fenêtre asymétrique
		var gBest *grenE
		gdt := 1 << 30
		for i := range gren {
			d := k.tms - gren[i].tms // >0 : lancer avant le kill
			if d >= -200 && d <= grenWin && d < gdt {
				gdt, gBest = d, &gren[i]
			}
		}
		cls, weapon, detail := "FIREARM", "(0xd2 famille via pipeline)", ""
		switch {
		case mBest != nil:
			cls, weapon, detail = "MÊLÉE", mBest.weapon, fmt.Sprintf("Δ%dms", mdt)
			nMel++
		case gBest != nil:
			cls, weapon, detail = "GRENADE", gBest.weapon, fmt.Sprintf("Δ%dms p%d", gdt, gBest.player)
			nGren++
		default:
			nFire++
		}
		narr := ""
		if w, ok := narrated[[2]string{kGT, vGT}]; ok {
			nNarr++
			narr = " <<< NARRÉ: " + w
			if cls == "MÊLÉE" {
				nNarrOK++
				narr += " [OK mêlée]"
			}
		}
		if cls != "FIREARM" || narr != "" {
			fmt.Printf("  %6.1fs %-16s -> %-16s | %-7s %s %s%s\n", float64(k.tms)/1000, kGT, vGT, cls, weapon, detail, narr)
		}
	}
	fmt.Printf("\n=== BILAN : MÊLÉE=%d GRENADE=%d FIREARM=%d (sur %d kills) ===\n", nMel, nGren, nFire, len(kills))
	fmt.Printf("kills marteau narrés classés MÊLÉE : %d/%d\n", nNarrOK, nNarr)
}

// runKeCause : le kill-event 0xE6 du film porte-t-il un CODE DE CAUSE par-kill (le modèle "1 code ->
// 1 algo") ? Grammaire FUN_14104bd08 : [R5 tueur@83][R5 victime@88][R32@93][R1@125][R5 assist@126][R32@131].
// On dump les 2 R32 + cherche un petit champ catégorie (0..7) + confronte aux tags de cause connus (grenade
// DamageEffect, familles d'arme). Si un champ distingue mêlée/grenade/firearm par-kill -> modèle propre.
func runKeCause(pkts []pkt) {
	// tags de cause connus (pour reconnaître un champ cause)
	causeTag := map[uint32]string{}
	for id, n := range grenadeIDs {
		causeTag[id] = "GREN:" + n
	}
	for id := range analysis.WeaponIDToName {
		causeTag[uint32(id>>32)] = "WFAM:" + analysis.WeaponIDToName[id]
	}
	_ = causeTag
	melee := collectMelee(pkts)
	gren := collectGren(pkts)
	// events 0xE6 horodatés
	type ke struct {
		tms    int
		f1, f2 uint32
	}
	var kes []ke
	f1hist := map[uint32]int{}
	for _, p := range pkts {
		if p.marker != 0xE6 {
			continue
		}
		tms := int((p.ts - t0Us) / 1000)
		f1 := uint32(bitsAt(p.pl, 93, 32))
		kes = append(kes, ke{tms, f1, uint32(bitsAt(p.pl, 131, 32))})
		f1hist[f1]++
	}
	fmt.Printf("\n=== KECAUSE : %d kill-events 0xE6 ===\n", len(kes))
	fmt.Printf("f1@93 distribution (candidat code cause) : %v\n", f1hist)
	// corrélation : f1 selon qu'un événement mêlée/grenade est proche (±400ms)
	near := func(tms int, ev func(int) bool) bool { return ev(tms) }
	hasMelee := func(t int) bool {
		for _, m := range melee {
			if abs(m.tms-t) <= 400 {
				return true
			}
		}
		return false
	}
	hasGren := func(t int) bool {
		for _, g := range gren {
			if abs(g.tms-t) <= 400 {
				return true
			}
		}
		return false
	}
	f1Melee, f1Gren, f1Other := map[uint32]int{}, map[uint32]int{}, map[uint32]int{}
	for _, k := range kes {
		switch {
		case near(k.tms, hasMelee):
			f1Melee[k.f1]++
		case near(k.tms, hasGren):
			f1Gren[k.f1]++
		default:
			f1Other[k.f1]++
		}
	}
	fmt.Printf("\nf1 chez les kills AVEC mêlée proche  : %v\n", f1Melee)
	fmt.Printf("f1 chez les kills AVEC grenade proche : %v\n", f1Gren)
	fmt.Printf("f1 chez les AUTRES kills (firearm ?)  : %v\n", f1Other)
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// runStruct : validation STRUCTURELLE (sans alignement temporel, donc universelle multi-film).
// Grenade : marqueur 0x4c0c00 -> id valide -> taux d'index 0-7 @+103. Mêlée : id64 dans 0xD3 -> distribution
// des armes + player@bit20. But : les marqueurs/offsets d'acurtis (corrigés) transfèrent-ils à d'autres films ?
func runStruct(pkts []pkt) {
	gren := collectGren(pkts)
	melee := collectMelee(pkts)
	// grenade : distribution index @+103
	gi := map[int]int{}
	g07 := 0
	for _, g := range gren {
		gi[g.player]++
		if g.player <= 7 {
			g07++
		}
	}
	gw := map[string]int{}
	for _, g := range gren {
		gw[g.weapon]++
	}
	fmt.Printf("\n=== STRUCT ===\n")
	fmt.Printf("GRENADE : %d événements | index 0-7 : %d/%d (%.0f%%) | armes : %v\n",
		len(gren), g07, len(gren), pctI(g07, len(gren)), gw)
	// mêlée : distribution armes + player@bit20
	mw := map[string]int{}
	mp := map[int]int{}
	m07 := 0
	for _, m := range melee {
		mw[m.weapon]++
		mp[m.player]++
		if m.player <= 7 {
			m07++
		}
	}
	fmt.Printf("MÊLÉE   : %d événements (0xD3 id64) | player@bit20 0-7 : %d/%d (%.0f%%)\n",
		len(melee), m07, len(melee), pctI(m07, len(melee)))
	fmt.Printf("  armes mêlée : %v\n", mw)
	fmt.Printf("  distribution player@bit20 : %v\n", mp)
	fmt.Printf("  distribution index grenade@+103 : %v\n", gi)
}

func pctI(a, b int) float64 {
	if b == 0 {
		return 0
	}
	return 100 * float64(a) / float64(b)
}

// meleeCand = un événement mêlée candidat : arme (family high32) trouvée à bit q dans un paquet 0xD3.
type meleeCand struct {
	pl  []byte
	q   int // position bit de la famille high32
	tms int
	fam uint32
}

// runMeleeSweep : dans les paquets 0xD3, trouve toute famille d'arme du catalogue (validée par
// low32 = 42c9679f OU une variante connue = vrai id64), puis SWEEP l'offset du player index (depuis
// bit0 ET depuis la famille) pour trouver celui qui donne des indices 0-7 pour le max d'événements.
func runMeleeSweep(pkts []pkt) {
	// low32 connus (session tag + variantes du catalogue) pour valider un id64.
	lowOK := map[uint32]bool{0x42c9679f: true}
	for id := range analysis.WeaponIDToName {
		lowOK[uint32(id)] = true
	}
	var cands []meleeCand
	for _, p := range pkts {
		if p.marker != 0xD3 {
			continue
		}
		tms := int((p.ts - t0Us) / 1000)
		nb := len(p.pl)*8 - 64
		for bp := 0; bp <= nb; bp++ {
			hi := uint32(bitsAt(p.pl, bp, 32))
			if _, ok := h32name[hi]; !ok {
				continue
			}
			lo := uint32(bitsAt(p.pl, bp+32, 32))
			if !lowOK[lo] {
				continue
			}
			cands = append(cands, meleeCand{pl: p.pl, q: bp, tms: tms, fam: hi})
			bp += 63
		}
	}
	fmt.Printf("\n--- MELEE SWEEP : %d candidats (0xD3, id64 arme valide) ---\n", len(cands))
	wHist := map[string]int{}
	for _, c := range cands {
		wHist[h32name[c.fam]]++
	}
	fmt.Printf("armes : ")
	for k, n := range wHist {
		fmt.Printf("%s:%d ", k, n)
	}
	fmt.Println()
	// SWEEP depuis bit0 : offset absolu δ (le player serait AVANT l'arme, à position fixe depuis le début).
	fmt.Println("\n[SWEEP depuis bit0] offset : #events avec index 0-7 (sur total) :")
	bestAbs, bestAbsN := -1, -1
	for d := 8; d <= 88; d++ {
		n07 := 0
		for _, c := range cands {
			if int(bitsAt(c.pl, d, 5)) <= 7 {
				n07++
			}
		}
		if n07 > bestAbsN {
			bestAbsN, bestAbs = n07, d
		}
	}
	fmt.Printf("  meilleur δ(bit0)=%d : %d/%d à 0-7\n", bestAbs, bestAbsN, len(cands))
	// SWEEP depuis la famille : offset relatif ε (player à q+ε).
	fmt.Println("[SWEEP depuis id64] offset ε : #events 0-7 :")
	bestRel, bestRelN := 0, -1
	for e := -88; e <= 40; e++ {
		n07 := 0
		for _, c := range cands {
			pos := c.q + e
			if pos >= 0 && pos+5 <= len(c.pl)*8 && int(bitsAt(c.pl, pos, 5)) <= 7 {
				n07++
			}
		}
		if n07 > bestRelN {
			bestRelN, bestRel = n07, e
		}
	}
	fmt.Printf("  meilleur ε(id64)=%d : %d/%d à 0-7\n", bestRel, bestRelN, len(cands))
	// échantillon : pour les 2 meilleurs offsets, dump (t, arme, playerAbs, playerRel)
	fmt.Println("\néchantillon (t, arme, player@δbit0, player@εid64) :")
	sort.Slice(cands, func(i, j int) bool { return cands[i].tms < cands[j].tms })
	for i, c := range cands {
		if i >= 30 {
			break
		}
		pa := int(bitsAt(c.pl, bestAbs, 5))
		pr := -1
		if c.q+bestRel >= 0 {
			pr = int(bitsAt(c.pl, c.q+bestRel, 5))
		}
		fmt.Printf("  %6.1fs %-18s pAbs=%d pRel=%d (id64@%d)\n", float64(c.tms)/1000, h32name[c.fam], pa, pr, c.q)
	}
}

// runFind localise un id64 arme dans le flux (tout bit-offset) et dump le contexte bit
// autour = reverse de la VRAIE structure de l'événement mêlée (exemple acurtis = vérité-terrain).
func runFind(pkts []pkt, id uint64) {
	fmt.Printf("\n--- FIND id64=%016x dans les frame packets ---\n", id)
	hits := 0
	for _, p := range pkts {
		tms := int((p.ts - t0Us) / 1000)
		nb := len(p.pl)*8 - 64
		for bp := 0; bp <= nb; bp++ {
			if bitsAt(p.pl, bp, 64) != id {
				continue
			}
			hits++
			if hits > 30 {
				continue
			}
			// dump 64 bits avant + l'id + 40 bits après, en hex, + champs candidats
			pre := ""
			for k := bp - 96; k < bp; k += 8 {
				if k >= 0 {
					pre += fmt.Sprintf("%02x", bitsAt(p.pl, k, 8))
				}
			}
			post := ""
			for k := bp + 64; k < bp+64+64 && k+8 <= len(p.pl)*8; k += 8 {
				post += fmt.Sprintf("%02x", bitsAt(p.pl, k, 8))
			}
			fmt.Printf("  %6.1fs mk=0x%02X bit=%d\n     pre[bit-96..]=%s | ID | post=%s\n",
				float64(tms)/1000, p.marker, bp, pre, post)
			if p.marker == 0xD3 && bp < 200 {
				// dump le paquet depuis bit 0 (structure complète de l'événement mêlée)
				full := ""
				for k := 0; k+8 <= len(p.pl)*8 && k < bp+96; k += 4 {
					full += fmt.Sprintf("%x", bitsAt(p.pl, k, 4))
				}
				fmt.Printf("     FULL[bit0..](nibbles)=%s  (id64@bit%d)\n", full, bp)
			}
		}
	}
	fmt.Printf("total hits=%d\n", hits)
}

func runMelee(pkts []pkt) {
	var evts []meleeEvt
	markerHits := 0
	typeHist := map[uint32]int{}
	byMarker := map[byte]int{}
	// CALIBRATION : l'offset arme d'acurtis (88/86/101/103) ne valide rien sur notre film.
	// Le type tombe (0x42/47/60 @+76) donc l'anchor est bon. On SCANNE une fenêtre autour de
	// l'anchor pour la famille reconnue et on histogramme l'offset trouvé PAR TYPE -> vrai offset.
	offHist := map[uint32]map[int]int{0x42: {}, 0x47: {}, 0x60: {}}
	// anchor est aussi une hypothèse : on scanne l'offset arme depuis le MARQUEUR (bp) pour être robuste.
	const winLo, winHi = 20, 220
	for _, p := range pkts {
		tms := int((p.ts - t0Us) / 1000)
		nb := len(p.pl)*8 - 240
		for bp := 0; bp <= nb; bp++ {
			mk := bitsAt(p.pl, bp, 11)
			if mk != 0x534 && mk != 0x535 { // 0x534=hit 0x535=miss (Ghidra-validé ; 0x532 = source communauté erronée)
				continue
			}
			markerHits++
			anchor := bp + 3
			typ := uint32(bitsAt(p.pl, anchor+76, 8))
			if _, ok := meleeOffsets[typ]; !ok {
				continue
			}
			typeHist[typ]++
			// offset arme FIXE par type (Ghidra-validé : 0x47->+86 marteau 43/43 ; 0x42->+88 ; 0x60->+101|103)
			for _, off := range meleeOffsets[typ] {
				fam := uint32(bitsAt(p.pl, anchor+off, 32))
				if nm, ok := h32name[fam]; ok {
					offHist[typ][off]++
					evts = append(evts, meleeEvt{tms: tms, marker: p.marker, typ: typ, off: off, fam: fam, weapon: nm})
					byMarker[p.marker]++
					break
				}
			}
			_, _ = winLo, winHi
		}
	}
	fmt.Println("\n[CALIBRATION] offset arme trouvé par type (offset:compte) :")
	for _, t := range []uint32{0x42, 0x47, 0x60} {
		fmt.Printf("  type 0x%02X : ", t)
		type oc struct{ o, c int }
		var ocs []oc
		for o, c := range offHist[t] {
			ocs = append(ocs, oc{o, c})
		}
		sort.Slice(ocs, func(i, j int) bool { return ocs[i].c > ocs[j].c })
		for i, e := range ocs {
			if i >= 8 {
				break
			}
			fmt.Printf("%d:%d ", e.o, e.c)
		}
		fmt.Println()
	}
	sort.Slice(evts, func(i, j int) bool { return evts[i].tms < evts[j].tms })
	fmt.Printf("\n--- MÊLÉE : %d marqueurs 0x534/0x535, %d événements validés (arme reconnue) ---\n", markerHits, len(evts))
	fmt.Printf("types validés : ")
	for t, n := range typeHist {
		fmt.Printf("0x%02X:%d ", t, n)
	}
	fmt.Printf("\nrépartition par marqueur de paquet : ")
	for mk, n := range byMarker {
		fmt.Printf("0x%02X:%d ", mk, n)
	}
	// distribution des armes
	wHist := map[string]int{}
	for _, e := range evts {
		wHist[e.weapon]++
	}
	fmt.Printf("\narmes mêlée détectées : ")
	type kv struct {
		k string
		n int
	}
	var ws []kv
	for k, n := range wHist {
		ws = append(ws, kv{k, n})
	}
	sort.Slice(ws, func(i, j int) bool { return ws[i].n > ws[j].n })
	for _, w := range ws {
		fmt.Printf("%s:%d ", w.k, w.n)
	}
	fmt.Println()
	// échantillon 25 premiers événements
	fmt.Println("\néchantillon (t, mk, player, type, off, arme, anim) :")
	for i, e := range evts {
		if i >= 25 {
			break
		}
		fmt.Printf("  %6.1fs mk=0x%02X p=%d type=0x%02X off=%d %s anim=%d\n",
			float64(e.tms)/1000, e.marker, e.player, e.typ, e.off, e.weapon, e.anim)
	}
}

func runGren(pkts []pkt) {
	type ev struct {
		tms, p47, p79 int
		weapon        string
	}
	var evts []ev
	markerHits := 0
	for _, p := range pkts {
		tms := int((p.ts - t0Us) / 1000)
		nb := len(p.pl)*8 - 120
		for bp := 0; bp <= nb; bp++ {
			if bitsAt(p.pl, bp, 24) != 0x4c0c00 {
				continue
			}
			markerHits++
			id := uint32(bitsAt(p.pl, bp+24, 32))
			if nm, ok := grenadeIDs[id]; ok {
				// deux interprétations du "skip 47 bits" : depuis fin marqueur (+47) et depuis fin id (+56+47=+103)
				evts = append(evts, ev{tms: tms, p47: int(bitsAt(p.pl, bp+24+47, 5)), p79: int(bitsAt(p.pl, bp+24+32+47, 5)), weapon: nm})
			}
		}
	}
	sort.Slice(evts, func(i, j int) bool { return evts[i].tms < evts[j].tms })
	fmt.Printf("\n--- GRENADE : %d marqueurs 0x4c0c00, %d événements validés ---\n", markerHits, len(evts))
	wHist := map[string]int{}
	for _, e := range evts {
		wHist[e.weapon]++
	}
	fmt.Printf("armes grenade : ")
	for k, n := range wHist {
		fmt.Printf("%s:%d ", k, n)
	}
	fmt.Println("\néchantillon (t, weapon, player@+47, player@+79) :")
	for i, e := range evts {
		if i >= 20 {
			break
		}
		fmt.Printf("  %6.1fs %s p47=%d p79=%d\n", float64(e.tms)/1000, e.weapon, e.p47, e.p79)
	}
}
