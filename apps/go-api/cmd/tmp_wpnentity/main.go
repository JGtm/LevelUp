// tmp_wpnentity — THROWAWAY : identifie l'ARCHÉTYPE ENTITÉ-ARME (la cible de
// résolution du handle WST du biped).
//
// 3 parties :
//
//	PART 1 : dump du registre chunk_00, liste de TOUS les archétypes qui contiennent
//	         weapon-state-type-info, et discrimination biped (#35, a des biped-*/unit-*)
//	         vs entité-arme (weapon-* mais PAS biped-*/unit-*).
//	PART 2 : ancrage empirique — positions bit des 16 littéraux d'armes connues dans
//	         le keyframe type-2 de chunk_02, + pour ≥1 le typeIndex du record porteur
//	         (brute-force start + sweep defaultStateBits + sweep recordStateParam).
//	PART 3 : décomposition du handle 0x01002b4a (index|génération) + lien à l'id record.
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/filmdec"
)

const cache = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache/film_chunks/000d5950`

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

func extractType2(d []byte) []byte {
	off := 0
	for off+16 <= len(d) {
		typ := binary.LittleEndian.Uint16(d[off:])
		sz := int(binary.LittleEndian.Uint32(d[off+4:]))
		if sz <= 0 || off+16+sz > len(d) {
			break
		}
		if typ == 2 {
			return d[off+16 : off+16+sz]
		}
		off += 16 + sz
	}
	return nil
}

func bitsAt(d []byte, bp, n int) uint64 {
	var v uint64
	for i := 0; i < n; i++ {
		p := bp + i
		if p>>3 >= len(d) {
			v = v << 1
			continue
		}
		v = (v << 1) | uint64((d[p>>3]>>uint(7-(p&7)))&1)
	}
	return v
}

func main() {
	reg, err := filmdec.ParseRegistryChunk(inflate(cache + "/chunk_00.bin"))
	if err != nil {
		panic(err)
	}
	payload := extractType2(inflate(cache + "/chunk_02.bin"))
	fmt.Printf("registre : %d archétypes ; keyframe type-2 chunk_02 : %d octets\n",
		len(reg.Archetypes), len(payload))

	mode := "all"
	if len(os.Args) > 1 {
		mode = os.Args[1]
	}
	if mode == "all" || mode == "1" {
		part1Registry(reg)
	}
	if mode == "all" || mode == "2" {
		part2Anchor(reg, payload)
	}
	if mode == "all" || mode == "3" {
		part3Handle()
	}
	if mode == "probe" {
		probeHydra(reg, payload)
	}
	if mode == "obje" {
		probeObjePrefix(payload)
	}
	if mode == "exact" {
		probeExactWST(reg, payload)
	}
	if mode == "carriers" {
		probeAllCarriers(reg, payload)
	}
}

// probeAllCarriers : pour chacun des 16 littéraux d'arme, trouve le record WST
// porteur (gate@litBit-1, handle=high32, variant=low32) en sweepant start/d/rsp.
// Reporte le typeIndex + index de composant + desyncAt.
func probeAllCarriers(reg *filmdec.Registry, payload []byte) {
	fmt.Printf("\n================ CARRIERS : record WST porteur des 16 littéraux ================\n")
	// Récupère les littéraux d'arme complets.
	type lit struct {
		bit  int
		id64 uint64
		name string
	}
	var lits []lit
	total := len(payload) * 8
	for bp := 0; bp+64 <= total; bp++ {
		v := uint32(bitsAt(payload, bp, 32))
		if _, ok := knownHigh32(v); ok {
			low := uint32(bitsAt(payload, bp+32, 32))
			id64 := (uint64(v) << 32) | uint64(low)
			if nm, ok2 := analysis.WeaponIDToName[id64]; ok2 {
				lits = append(lits, lit{bp, id64, nm})
			}
		}
	}
	sort.Slice(lits, func(i, j int) bool { return lits[i].bit < lits[j].bit })

	for _, l := range lits {
		gateBit := l.bit - 1
		type hit struct {
			start, d int
			rsp      uint32
			ti       uint32
			ci       int
			desync   int
			ncomp    int
		}
		var hits []hit
		seen := map[[2]int]bool{}
		for start := gateBit - 2500; start <= gateBit-20; start++ {
			for d := 1; d <= 400; d++ {
				for r := uint32(0); r <= 3; r++ {
					filmdec.SetRecordStateParam(r)
					br := filmdec.NewBitReader(payload)
					br.Skip(start)
					t := filmdec.TraverseEntity(br, reg, d)
					for _, c := range t.Comps {
						if c.Name != "weapon-state-type-info" || c.StartBit != gateBit {
							continue
						}
						h := uint32(bitsAt(payload, gateBit+1, 32))
						v := uint32(bitsAt(payload, gateBit+33, 32))
						if (uint64(h)<<32)|uint64(v) != l.id64 {
							continue
						}
						key := [2]int{int(t.TypeIndex), start}
						if seen[key] {
							continue
						}
						seen[key] = true
						hits = append(hits, hit{start, d, r, t.TypeIndex, c.Index, t.DesyncAt, len(t.Comps)})
					}
				}
			}
		}
		sort.Slice(hits, func(i, j int) bool { return hits[i].desync > hits[j].desync })
		if len(hits) == 0 {
			fmt.Printf("  %-16s @%-7d : aucun record WST exact trouvé\n", l.name, l.bit)
			continue
		}
		b := hits[0]
		mark := ""
		if b.ti != 35 {
			mark = "  <<< NON-35 !"
		}
		fmt.Printf("  %-16s @%-7d : ti=%-3d compIdx=i%-2d desyncAt=i%-2d ncomp=%-2d (start=%d d=%d rsp=%d, %d hits)%s\n",
			l.name, l.bit, b.ti, b.ci, b.desync, b.ncomp, b.start, b.d, b.rsp, len(hits), mark)
	}
}

// probeExactWST cherche le record (TOUT typeIndex) dont un weapon-state-type-info
// a StartBit == 195322 EXACTEMENT (le gate du littéral Hydra), avec gate=1 et
// reconstruction id64 = Hydra. Sweep d élargi [1..400]. Trace le record complet du
// meilleur hit.
func probeExactWST(reg *filmdec.Registry, payload []byte) {
	const gateBit = 195322
	const id64 = uint64(0x767db96d42c9679f)
	fmt.Printf("\n================ EXACT : WST.StartBit==%d => Hydra ================\n", gateBit)
	type hit struct {
		start, d int
		rsp      uint32
		ti       uint32
		ci       int
		desync   int
		ncomp    int
	}
	var hits []hit
	seen := map[[2]int]bool{}
	for start := gateBit - 5000; start <= gateBit-20; start++ {
		for d := 1; d <= 400; d++ {
			for r := uint32(0); r <= 3; r++ {
				filmdec.SetRecordStateParam(r)
				br := filmdec.NewBitReader(payload)
				br.Skip(start)
				t := filmdec.TraverseEntity(br, reg, d)
				for _, c := range t.Comps {
					if c.Name != "weapon-state-type-info" || c.StartBit != gateBit {
						continue
					}
					handle := uint32(bitsAt(payload, gateBit+1, 32))
					variant := uint32(bitsAt(payload, gateBit+33, 32))
					if (uint64(handle)<<32)|uint64(variant) != id64 {
						continue
					}
					key := [2]int{int(t.TypeIndex), start}
					if seen[key] {
						continue
					}
					seen[key] = true
					hits = append(hits, hit{start, d, r, t.TypeIndex, c.Index, t.DesyncAt, len(t.Comps)})
				}
			}
		}
	}
	fmt.Printf("%d hits WST.StartBit==%d reconstruisant Hydra :\n", len(hits), gateBit)
	sort.Slice(hits, func(i, j int) bool {
		if (hits[i].ti == 35) != (hits[j].ti == 35) {
			return hits[i].ti != 35
		}
		return hits[i].desync > hits[j].desync // desync le plus profond = traversée la plus saine
	})
	for i, h := range hits {
		mark := "  <<< NON-35"
		if h.ti == 35 {
			mark = "  (biped 35)"
		}
		fmt.Printf("  ti=%-3d i%-2d start=%-7d d=%-3d rsp=%d desyncAt=i%-2d ncomp=%d%s\n",
			h.ti, h.ci, h.start, h.d, h.rsp, h.desync, h.ncomp, mark)
		if i >= 25 {
			fmt.Printf("  ...\n")
			break
		}
	}
	// Trace le meilleur hit (desync le plus profond, ti==35 accepté).
	if len(hits) > 0 {
		// préfère le hit ti!=35 le plus sain, sinon ti==35.
		best := hits[0]
		fmt.Printf("\n--- TRACE meilleur hit : ti=%d start=%d d=%d rsp=%d ---\n", best.ti, best.start, best.d, best.rsp)
		filmdec.SetRecordStateParam(best.rsp)
		br := filmdec.NewBitReader(payload)
		br.Skip(best.start)
		t := filmdec.TraverseEntity(br, reg, best.d)
		fmt.Printf("typeIndex=%d desyncAt=i%d nComps=%d endBit=%d\n", t.TypeIndex, t.DesyncAt, len(t.Comps), t.EndBit)
		for _, c := range t.Comps {
			extra := ""
			if c.Name == "weapon-state-type-info" {
				g := bitsAt(payload, c.StartBit, 1)
				h := uint32(bitsAt(payload, c.StartBit+1, 32))
				v := uint32(bitsAt(payload, c.StartBit+33, 32))
				id := (uint64(h) << 32) | uint64(v)
				nm := analysis.WeaponIDToName[id]
				extra = fmt.Sprintf("  gate=%d handle=0x%08x variant=0x%08x id64=0x%016x arme=%q", g, h, v, id, nm)
			}
			fmt.Printf("  i%-2d %-44s @bit%d%s\n", c.Index, c.Name, c.StartBit, extra)
		}
	}
}

// probeObjePrefix décode le préfixe d'un record 'obje' (DecodeEntityRecordQ) à
// partir de plusieurs StartBit candidats et imprime la position de chaque champ
// (LocalID, VariantName). Objectif : confirmer que pour le 'obje' d'une entité-arme,
// LocalID=high32 Hydra @195323 et VariantName=low32 @195355.
func probeObjePrefix(payload []byte) {
	fmt.Printf("\n================ PROBE préfixe 'obje' (LocalID/VariantName positions) ================\n")
	// Candidats : starts plausibles du 'obje' juste avant 195323.
	for _, objeStart := range []int{195305, 195306, 195307, 195308, 195309, 195310, 195311, 195312, 195290} {
		p := objeStart
		trace := func(label string, n int) uint64 {
			v := bitsAt(payload, p, n)
			fmt.Printf("    @%-7d %-14s R(%-2d) = 0x%x (%d)\n", p, label, n, v, v)
			p += n
			return v
		}
		fmt.Printf("\n  --- objeStart=%d ---\n", objeStart)
		trace("RawFlag", 1)
		trace("ModeFlag", 1)
		trace("lo7", 7)
		trace("hi1", 1)
		if trace("Field0C-gate", 1) == 0 {
			trace("Field0C", 5)
		}
		if trace("ID5-gate", 1) == 0 {
			trace("ID5", 2)
		}
		g := trace("LocalID-gate", 1)
		if g == 1 {
			lid := uint32(trace("LocalID(R32)", 32))
			vn := uint32(trace("VariantName(R32)", 32))
			id64 := (uint64(lid) << 32) | uint64(vn)
			nm, ok := analysis.WeaponIDToName[id64]
			fmt.Printf("    => id64=(LocalID<<32)|VariantName = 0x%016x  arme=%v (%s)\n", id64, ok, nm)
		} else {
			fmt.Printf("    => LocalID absent (gate=0) ; VariantName(R32)@%d = 0x%08x\n", p, uint32(bitsAt(payload, p, 32)))
		}
	}
}

// probeHydra : pour le littéral Hydra @195323, balaie les records (tout typeIndex)
// dont la traversée atteint un composant (obje OU WST) dont un StartBit tombe dans
// [195290..195360], et trace ce record en entier. Objectif : voir si le littéral
// est dans i9 (obje) ou i43..46 (WST), et quel typeIndex le porte.
func probeHydra(reg *filmdec.Registry, payload []byte) {
	const litBit = 195323
	fmt.Printf("\n================ PROBE Hydra @%d ================\n", litBit)
	type cand struct {
		start, d int
		rsp      uint32
		ti       uint32
		comp     filmdec.CompResult
	}
	var cands []cand
	seen := map[[3]int]bool{}
	for start := litBit - 4000; start <= litBit-50; start++ {
		for d := 1; d <= 200; d++ {
			for r := uint32(0); r <= 3; r++ {
				filmdec.SetRecordStateParam(r)
				br := filmdec.NewBitReader(payload)
				br.Skip(start)
				t := filmdec.TraverseEntity(br, reg, d)
				for _, c := range t.Comps {
					if c.StartBit < litBit-40 || c.StartBit > litBit+5 {
						continue
					}
					if c.Name != "object-multiplayer-properties-component" && c.Name != "weapon-state-type-info" {
						continue
					}
					key := [3]int{int(t.TypeIndex), c.Index, c.StartBit}
					if seen[key] {
						continue
					}
					seen[key] = true
					cands = append(cands, cand{start, d, r, t.TypeIndex, c})
				}
			}
		}
	}
	fmt.Printf("%d candidats (obje/WST) avec StartBit dans [%d..%d] :\n", len(cands), litBit-40, litBit+5)
	sort.Slice(cands, func(i, j int) bool { return cands[i].comp.StartBit < cands[j].comp.StartBit })
	for _, c := range cands {
		gate := bitsAt(payload, c.comp.StartBit, 1)
		r1 := uint32(bitsAt(payload, c.comp.StartBit, 32))
		fmt.Printf("  ti=%-3d %s i%-2d StartBit=%d gate=%d R32@start=0x%08x variant=0x%08x  start=%d d=%d rsp=%d\n",
			c.ti, c.comp.Name, c.comp.Index, c.comp.StartBit, gate, r1, c.comp.Variant, c.start, c.d, c.rsp)
	}
}

// ===========================================================================
// PART 1 : dump du registre, classification des archétypes
// ===========================================================================

func hasComp(a filmdec.Archetype, name string) bool {
	for _, c := range a.Components {
		if c == name {
			return true
		}
	}
	return false
}

func hasPrefix(a filmdec.Archetype, prefix string) bool {
	for _, c := range a.Components {
		if strings.HasPrefix(c, prefix) {
			return true
		}
	}
	return false
}

func countPrefix(a filmdec.Archetype, prefix string) int {
	n := 0
	for _, c := range a.Components {
		if strings.HasPrefix(c, prefix) {
			n++
		}
	}
	return n
}

func part1Registry(reg *filmdec.Registry) {
	fmt.Printf("\n================ PART 1 : ARCHÉTYPES AVEC weapon-state-type-info ================\n")

	var withWST []filmdec.Archetype
	for _, a := range reg.Archetypes {
		if hasComp(a, "weapon-state-type-info") {
			withWST = append(withWST, a)
		}
	}
	fmt.Printf("%d archétypes contiennent weapon-state-type-info :\n\n", len(withWST))

	for _, a := range withWST {
		isBiped := hasPrefix(a, "biped-")
		hasUnit := hasPrefix(a, "unit-")
		nWeapon := countPrefix(a, "weapon-")
		nWST := countOf(a, "weapon-state-type-info")
		kind := "ENTITÉ-ARME?"
		if isBiped || hasUnit {
			kind = "BIPED/UNIT (arme tenue)"
		}
		fmt.Printf("  typeIndex=%-3d  nComps=%-3d  weapon-*=%-2d  WST×%d  biped-*=%v unit-*=%v  => %s\n",
			a.Index, len(a.Components), nWeapon, nWST, isBiped, hasUnit, kind)
	}

	// Dump COMPLET des candidats entité-arme (pas biped, pas unit) + du biped #35 pour comparaison.
	fmt.Printf("\n---- DUMP COMPLET : candidats ENTITÉ-ARME (weapon-* SANS biped-*/unit-*) ----\n")
	for _, a := range withWST {
		if hasPrefix(a, "biped-") || hasPrefix(a, "unit-") {
			continue
		}
		dumpArch(a)
	}

	fmt.Printf("\n---- DUMP COMPLET : BIPED #35 (référence, arme TENUE) ----\n")
	if a, ok := reg.Archetype(35); ok {
		dumpArch(a)
	}

	// Pour exhaustivité : tous les archétypes "object-*" only (pas weapon, utile pour repérer
	// l'archétype d'arme qui aurait weapon-* mais qu'on liste déjà). On liste aussi les
	// archétypes qui ont weapon-* sans WST (pour ne rien rater).
	fmt.Printf("\n---- Archétypes avec weapon-* MAIS SANS weapon-state-type-info ----\n")
	for _, a := range reg.Archetypes {
		if hasPrefix(a, "weapon-") && !hasComp(a, "weapon-state-type-info") {
			fmt.Printf("  typeIndex=%-3d nComps=%-3d weapon-*=%d biped-*=%v unit-*=%v head=%s\n",
				a.Index, len(a.Components), countPrefix(a, "weapon-"),
				hasPrefix(a, "biped-"), hasPrefix(a, "unit-"), a.Components[0])
			dumpArch(a)
		}
	}

	// Catalogue COMPLET de tous les archétypes (tête + nb comps) pour repérer celui
	// qui porte l'arme (object-multiplayer-properties = 'obje' avec variant-name).
	fmt.Printf("\n---- CATALOGUE COMPLET des %d archétypes (typeIndex : head [nComps]) ----\n", len(reg.Archetypes))
	for _, a := range reg.Archetypes {
		head := "(vide)"
		if len(a.Components) > 0 {
			head = a.Components[0]
		}
		obje := ""
		if hasComp(a, "object-multiplayer-properties-component") {
			obje = "  [obje]"
		}
		wpn := ""
		if hasPrefix(a, "weapon-") {
			wpn = fmt.Sprintf("  weapon-*=%d", countPrefix(a, "weapon-"))
		}
		fmt.Printf("  ti=%-3d [%2d] %s%s%s\n", a.Index, len(a.Components), head, obje, wpn)
	}

	// Identifie l'archétype "object" pur le plus probable pour une entité-arme :
	// a object-multiplayer-properties (le 'obje' qui porte variant-name) + des weapon-*
	// mais PAS biped-*/unit-*. Dump de tous ceux qui ont 'obje' et sont "petits".
	fmt.Printf("\n---- Archétypes avec 'obje' (object-multiplayer-properties) SANS biped/unit ----\n")
	for _, a := range reg.Archetypes {
		if !hasComp(a, "object-multiplayer-properties-component") {
			continue
		}
		if hasPrefix(a, "biped-") || hasPrefix(a, "unit-") {
			continue
		}
		fmt.Printf("  ti=%-3d nComps=%-3d weapon-*=%d head=%s\n",
			a.Index, len(a.Components), countPrefix(a, "weapon-"), a.Components[0])
	}
}

func countOf(a filmdec.Archetype, name string) int {
	n := 0
	for _, c := range a.Components {
		if c == name {
			n++
		}
	}
	return n
}

func dumpArch(a filmdec.Archetype) {
	fmt.Printf("\n  === typeIndex=%d  (%d composants) ===\n", a.Index, len(a.Components))
	for i, c := range a.Components {
		fmt.Printf("    i%-2d %s\n", i, c)
	}
}

// ===========================================================================
// PART 2 : ancrage empirique
// ===========================================================================

func knownHigh32(v uint32) (string, bool) {
	for id, n := range analysis.WeaponIDToName {
		if uint32(id>>32) == v {
			return n, true
		}
	}
	return "", false
}

func part2Anchor(reg *filmdec.Registry, payload []byte) {
	fmt.Printf("\n================ PART 2 : 16 LITTÉRAUX D'ARMES + RECORD PORTEUR ================\n")

	// 2a. Scan high-32 sur tout le payload, repérage des littéraux d'armes connues.
	type lit struct {
		bit   int
		high  uint32
		name  string
		low   uint32 // R(32) suivant (low-32 candidat = suffixe)
		id64  uint64
		isVar bool // low == suffixe d'une arme connue
	}
	var lits []lit
	total := len(payload) * 8
	for bp := 0; bp+64 <= total; bp++ {
		v := uint32(bitsAt(payload, bp, 32))
		if name, ok := knownHigh32(v); ok {
			low := uint32(bitsAt(payload, bp+32, 32))
			id64 := (uint64(v) << 32) | uint64(low)
			_, isVar := analysis.WeaponIDToName[id64]
			lits = append(lits, lit{bp, v, name, low, id64, isVar})
		}
	}
	sort.Slice(lits, func(i, j int) bool { return lits[i].bit < lits[j].bit })
	fmt.Printf("%d littéraux high-32 d'armes connues dans le payload type-2 :\n", len(lits))
	for i, l := range lits {
		tag := ""
		if l.isVar {
			tag = fmt.Sprintf("  id64=0x%016x (low=0x%08x = SUFFIXE arme connue -> littéral arme complet)", l.id64, l.low)
		}
		fmt.Printf("  [%2d] bit=%-7d high=0x%08x %-16s%s\n", i, l.bit, l.high, l.name, tag)
	}

	// On garde la liste des "vrais littéraux arme" (high+low forment un id64 catalogué).
	var weaponLits []lit
	for _, l := range lits {
		if l.isVar {
			weaponLits = append(weaponLits, l)
		}
	}
	fmt.Printf("\n=> %d littéraux d'ARME COMPLETS (high<<32|low présent au catalogue) :\n", len(weaponLits))
	for _, l := range weaponLits {
		fmt.Printf("   bit=%-7d id64=0x%016x %s\n", l.bit, l.id64, l.name)
	}

	// 2b. Pour CHAQUE littéral d'arme complet : trouver le record porteur.
	// Structure ancrée (cf. tmp_bipedcal SECTION 6) : gate@S=1, handle=R(32)@S+1, variant=R(32)@S+33.
	// Donc pour un littéral à bit B (high32), B == S+1 (handle) si le high32 est le handle,
	// et B+32 == variant. On cherche un record dont un weapon-state-type-info a StartBit == B-1
	// (gate juste avant le high32) ET dont le typeIndex != 35.
	fmt.Printf("\n---- 2b. RECORD PORTEUR de chaque littéral d'arme (sweep start/defaultBits/rsp) ----\n")
	maxCarriers := len(weaponLits)
	if len(os.Args) > 2 {
		if os.Args[2] == "first" {
			maxCarriers = 1
		}
	}
	for i, l := range weaponLits {
		if i >= maxCarriers {
			fmt.Printf("  (... %d littéraux restants non traités en mode 'first')\n", len(weaponLits)-maxCarriers)
			break
		}
		findCarrier(reg, payload, l.bit, l.id64, l.name)
	}
}

// findCarrier brute-force le record porteur d'un littéral d'arme à bit litBit.
//
// MODÈLE STRUCTUREL (confirmé par entity_quant.go DecodeEntityRecordQ) :
// le littéral id64 d'arme = DEUX R(32) consécutifs du record 'obje' (object-multiplayer-
// properties-component) : LocalID (rec[0x10], R32) = high-32 (famille) @litBit,
// puis VariantName (rec[0x14], R32) = low-32 (suffixe) @litBit+32. Le gate
// FUN_14080d69c qui précède LocalID doit valoir 1 (sinon LocalID absent).
//
// On cherche donc le record dont un composant 'obje' a CompResult.Variant == low32
// (la VariantName retournée par DecodeEntityRecordQ) ET dont la traversée place
// ce 'obje' de sorte que LocalID@litBit. On reporte le typeIndex porteur (≠ 35).
func findCarrier(reg *filmdec.Registry, payload []byte, litBit int, id64 uint64, name string) {
	high32 := uint32(id64 >> 32)
	low32 := uint32(id64)
	gateBit := litBit - 1 // FUN_14080d69c gate, doit valoir 1
	fmt.Printf("\n  >>> littéral %s @bit%d  id64=0x%016x (high=0x%08x low=0x%08x)\n",
		name, litBit, id64, high32, low32)
	fmt.Printf("      gate@%d=%d  R32@%d=0x%08x (=LocalID?)  R32@%d=0x%08x (=VariantName?)\n",
		gateBit, bitsAt(payload, gateBit, 1), litBit, uint32(bitsAt(payload, litBit, 32)),
		litBit+32, uint32(bitsAt(payload, litBit+32, 32)))

	type result struct {
		start, d  int
		rsp       uint32
		typeIdx   uint32
		compIdx   int
		objeStart int
		desyncAt  int
		endBit    int
		nComps    int
	}
	var hits []result
	seen := map[[2]uint32]bool{} // dédup (typeIndex, objeStart)

	// Le record 'obje' commence peu avant son composant 'obje'. La fenêtre de
	// recherche du start est modérée (le préfixe object-* avant 'obje' fait
	// quelques centaines de bits). On balaie start dans [litBit-4000, litBit-1].
	lo := litBit - 4000
	if lo < 0 {
		lo = 0
	}
	hi := litBit - 1

	for start := lo; start <= hi; start++ {
		for d := 1; d <= 200; d++ {
			for r := uint32(0); r <= 3; r++ {
				filmdec.SetRecordStateParam(r)
				br := filmdec.NewBitReader(payload)
				br.Skip(start)
				t := filmdec.TraverseEntity(br, reg, d)
				for _, c := range t.Comps {
					if c.Name != "object-multiplayer-properties-component" {
						continue
					}
					// VariantName (low-32) retournée par DecodeEntityRecordQ.
					if c.Variant != low32 {
						continue
					}
					// Vérifie que dans CE record, LocalID tombe sur high32 @litBit :
					// on re-décode le préfixe du 'obje' pour localiser LocalID.
					if !objeLocalIDMatches(payload, c.StartBit, high32, litBit) {
						continue
					}
					key := [2]uint32{t.TypeIndex, uint32(c.StartBit)}
					if seen[key] {
						continue
					}
					seen[key] = true
					hits = append(hits, result{start, d, r, t.TypeIndex, c.Index, c.StartBit, t.DesyncAt, t.EndBit, len(t.Comps)})
				}
			}
		}
	}

	if len(hits) == 0 {
		fmt.Printf("      AUCUN record 'obje' dont VariantName==low32 ET LocalID==high32@litBit (start[%d..%d] d[1..200] rsp[0..3])\n", lo, hi)
		return
	}
	sort.Slice(hits, func(i, j int) bool {
		a35 := hits[i].typeIdx == 35
		b35 := hits[j].typeIdx == 35
		if a35 != b35 {
			return !a35
		}
		return hits[i].start < hits[j].start
	})
	for _, h := range hits {
		mark := "  <<< ENTITÉ-ARME (typeIndex != 35)"
		if h.typeIdx == 35 {
			mark = "  (biped #35)"
		}
		arch, _ := reg.Archetype(int(h.typeIdx))
		archHead := ""
		if len(arch.Components) > 0 {
			archHead = arch.Components[0]
		}
		fmt.Printf("      typeIndex=%-3d [head=%s] start=%-7d d=%-3d rsp=%d objeIdx=i%-2d objeStart=%d desyncAt=i%-2d nComps=%d%s\n",
			h.typeIdx, archHead, h.start, h.d, h.rsp, h.compIdx, h.objeStart, h.desyncAt, h.nComps, mark)
	}
}

// objeLocalIDMatches re-décode le préfixe du record 'obje' à partir de objeStart
// pour localiser le bit où LocalID (R32) est lu, et vérifie qu'il vaut high32 et
// se trouve exactement à litBit. Reproduit le préfixe de DecodeEntityRecordQ
// jusqu'à LocalID (gate FUN_14080d69c puis R(32)).
func objeLocalIDMatches(payload []byte, objeStart int, high32 uint32, litBit int) bool {
	p := objeStart
	rd := func(n int) uint64 { v := bitsAt(payload, p, n); p += n; return v }
	rd(1)           // RawFlag rec[0x00]
	rd(1)           // ModeFlag rec[0x1C]
	rd(7)           // lo7
	rd(1)           // hi1
	if rd(1) == 0 { // Field0C gate : bit==0 -> R(5)
		rd(5)
	}
	if rd(1) == 0 { // ID5 gate : bit==0 -> R(2)
		rd(2)
	}
	// LocalID gate FUN_14080d69c : bit==1 -> R(32) @ p
	if rd(1) != 1 {
		return false // LocalID absent
	}
	return p == litBit && uint32(bitsAt(payload, p, 32)) == high32
}

// ===========================================================================
// PART 3 : décomposition du handle
// ===========================================================================

func part3Handle() {
	fmt.Printf("\n================ PART 3 : DÉCOMPOSITION DU HANDLE 0x01002b4a ================\n")
	handle := uint32(0x01002b4a)
	fmt.Printf("  handle brut = 0x%08x = %d (binaire %032b)\n", handle, handle, handle)

	// SCHÉMA CONFIRMÉ PAR GHIDRA (FUN_14049a384 / FUN_1405839d0) :
	//   index de slot   = handle >> 13   (DAT_1452f2ed0[(handle>>13)*8])
	//   génération/tag   = handle & 0x1FFF (13 bits de poids faible)
	// Le moteur indexe la table d'entités globale par (handle>>13) puis appelle
	// FUN_140749248(handle, 0x6f626a65='obje') pour résoudre le composant 'obje'.
	idx := handle >> 13
	gen := handle & 0x1FFF
	fmt.Printf("\n  -- SCHÉMA CONFIRMÉ GHIDRA : index = handle>>13, génération = handle & 0x1FFF --\n")
	fmt.Printf("     index/slot (handle>>13)      = 0x%05x = %d\n", idx, idx)
	fmt.Printf("     génération (handle & 0x1FFF) = 0x%04x = %d\n", gen, gen)
	fmt.Printf("     -> le moteur lit DAT_1452f2ed0[index*8] = pointeur entité, puis\n")
	fmt.Printf("        FUN_140749248(handle, 'obje'=0x6f626a65) résout le composant 'obje'.\n")

	fmt.Printf("\n  -- Provenance du handle dans le WST (FUN_1407f06bc) --\n")
	fmt.Printf("     gate FUN_14080d69c=1 -> FUN_14080dec4 lit le variant-name R(32) (global-id) dans [comp+4]\n")
	fmt.Printf("     puis FUN_14080d61c = GetLocalHandleFromGlobalId (0 bit) écrit le LOCAL HANDLE dans [comp+8]\n")
	fmt.Printf("     => 0x01002b4a est le handle RÉSOLU EN MÉMOIRE, PAS un champ du bitstream.\n")
}
