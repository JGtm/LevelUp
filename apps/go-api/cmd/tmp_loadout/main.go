// tmp_loadout — THROWAWAY (TRACK A) : produit la table joueur -> {arme primaire,
// arme secondaire} pour les 8 joueurs du keyframe type-2.
//
// Modèle structurel (acquis) : les 8 joueurs sont 8 records biped #35 consécutifs.
// Chaque record porte ses armes dans ses slots WST i43..i46 (high-32 = famille,
// low-32 = suffixe, reconstruit en id64 catalogué analysis.WeaponIDToName) ET un
// composant i9 'object-multiplayer-properties' ('obje') dont le préfixe encode
// LocalID (handle local R32, gated) + VariantName (R32 inconditionnel).
//
// Méthode :
//
//	PHASE 1 (calibrate) : pour chaque ancre = 1er littéral de chaque paire d'armes,
//	  brute-force (start, defaultBits, rsp) du record biped #35 dont un WST tombe
//	  PILE sur l'ancre (id64 catalogué) avec DesyncAt le plus tardif (idéalement -1).
//	PHASE 2 (extract)   : dans ce record, lit le 'obje' i9 (LocalID + VariantName via
//	  re-décodage du préfixe) et TOUS les WST (armes). Regroupe par record => joueur.
//	PHASE 3 (map)       : compare LocalID/VariantName du 'obje' aux pi->xuid connus.
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

// pi->xuid bit-vérifiés (contexte TRACK A).
var piXUID = []uint64{
	2535467794760703, // pi0
	2535437947245250, // pi1
	2533274823110022, // pi2 = JGtm
	2533274980284321, // pi3
	2533274815845110, // pi4
	2535444178793711, // pi5
	2533274882097883, // pi6
	2533274826120416, // pi7
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

func knownHigh32(v uint32) bool {
	for id := range analysis.WeaponIDToName {
		if uint32(id>>32) == v {
			return true
		}
	}
	return false
}

// weaponLit holds one anchored weapon literal (high<<32|low catalogued).
type weaponLit struct {
	bit  int
	id64 uint64
	name string
}

// findWeaponLits scans the payload for all complete weapon literals.
func findWeaponLits(payload []byte) []weaponLit {
	var lits []weaponLit
	total := len(payload) * 8
	for bp := 0; bp+64 <= total; bp++ {
		v := uint32(bitsAt(payload, bp, 32))
		if !knownHigh32(v) {
			continue
		}
		low := uint32(bitsAt(payload, bp+32, 32))
		id64 := (uint64(v) << 32) | uint64(low)
		if nm, ok := analysis.WeaponIDToName[id64]; ok {
			lits = append(lits, weaponLit{bp, id64, nm})
		}
	}
	sort.Slice(lits, func(i, j int) bool { return lits[i].bit < lits[j].bit })
	return lits
}

// groupLits regroupe les littéraux en records (nouveau groupe si gap > 1000 bits).
func groupLits(lits []weaponLit) [][]weaponLit {
	var groups [][]weaponLit
	for _, l := range lits {
		if len(groups) == 0 || l.bit-groups[len(groups)-1][len(groups[len(groups)-1])-1].bit > 1000 {
			groups = append(groups, []weaponLit{l})
		} else {
			groups[len(groups)-1] = append(groups[len(groups)-1], l)
		}
	}
	return groups
}

// objePrefix re-decodes the 'obje' record prefix from objeStart and returns the
// (LocalID, localIDPresent, VariantName, variantBit). Mirrors DecodeEntityRecordQ
// preamble up to VariantName (RawFlag,ModeFlag,HeaderA,Field0C,ID5,LocalID,VariantName).
func objePrefix(payload []byte, objeStart int) (localID uint32, lidPresent bool, variant uint32, variantBit int) {
	p := objeStart
	rd := func(n int) uint64 { v := bitsAt(payload, p, n); p += n; return v }
	rd(1)           // RawFlag rec[0x00]
	rd(1)           // ModeFlag rec[0x1C]
	rd(7)           // lo7
	rd(1)           // hi1
	if rd(1) == 0 { // Field0C gate: 0 -> R(5)
		rd(5)
	}
	if rd(1) == 0 { // ID5 gate: 0 -> R(2)
		rd(2)
	}
	if rd(1) == 1 { // LocalID gate: 1 -> R(32)
		localID = uint32(rd(32))
		lidPresent = true
	}
	variantBit = p
	variant = uint32(rd(32))
	return
}

// bipedRecord is a calibrated biped #35 record.
type bipedRecord struct {
	anchorBit int
	start, d  int
	rsp       uint32
	desyncAt  int
	nComps    int
	objeStart int
	weapons   []weaponLit // WST literals reconstructed inside this record
}

// calibrate brute-forces the biped #35 record carrying the WST anchored at anchorID64.
// Returns the best record (latest DesyncAt) or nil.
func calibrate(reg *filmdec.Registry, payload []byte, anchorBit int, anchorID64 uint64) *bipedRecord {
	gateBit := anchorBit - 1 // WST.StartBit == gateBit (gate just before high32)
	var best *bipedRecord
	for start := gateBit - 2800; start <= gateBit-20; start++ {
		if uint32(bitsAt(payload, start, 6)) != 35 {
			continue
		}
		for d := 1; d <= 420; d++ {
			for r := uint32(0); r <= 3; r++ {
				filmdec.SetRecordStateParam(r)
				br := filmdec.NewBitReader(payload)
				br.Skip(start)
				t := filmdec.TraverseEntity(br, reg, d)
				if t.TypeIndex != 35 {
					continue
				}
				hitAnchor := false
				for _, c := range t.Comps {
					if c.Name == "weapon-state-type-info" && c.StartBit == gateBit {
						h := uint32(bitsAt(payload, gateBit+1, 32))
						v := uint32(bitsAt(payload, gateBit+33, 32))
						if (uint64(h)<<32)|uint64(v) == anchorID64 {
							hitAnchor = true
						}
					}
				}
				if !hitAnchor {
					continue
				}
				rec := buildRecord(payload, t, anchorBit, start, d, r)
				// Prefer latest DesyncAt (deepest sane traversal).
				if best == nil || rec.desyncAt < 0 && best.desyncAt >= 0 ||
					(sameSign(rec.desyncAt, best.desyncAt) && rec.desyncAt > best.desyncAt) {
					rc := rec
					best = &rc
				}
			}
		}
	}
	return best
}

func sameSign(a, b int) bool { return (a < 0) == (b < 0) }

// buildRecord extracts obje i9 + all WST weapons from a traced biped record.
func buildRecord(payload []byte, t filmdec.EntityTrace, anchorBit, start, d int, r uint32) bipedRecord {
	rec := bipedRecord{anchorBit: anchorBit, start: start, d: d, rsp: r, desyncAt: t.DesyncAt, nComps: len(t.Comps), objeStart: -1}
	for _, c := range t.Comps {
		if c.Name == "object-multiplayer-properties-component" && rec.objeStart < 0 {
			rec.objeStart = c.StartBit
		}
		if c.Name == "weapon-state-type-info" {
			h := uint32(bitsAt(payload, c.StartBit+1, 32))
			v := uint32(bitsAt(payload, c.StartBit+33, 32))
			id64 := (uint64(h) << 32) | uint64(v)
			if nm, ok := analysis.WeaponIDToName[id64]; ok {
				rec.weapons = append(rec.weapons, weaponLit{c.StartBit + 1, id64, nm})
			}
		}
	}
	return rec
}

func main() {
	reg, err := filmdec.ParseRegistryChunk(inflate(cache + "/chunk_00.bin"))
	if err != nil {
		panic(err)
	}
	payload := extractType2(inflate(cache + "/chunk_02.bin"))
	fmt.Printf("registre %d archétypes ; keyframe type-2 %d octets\n", len(reg.Archetypes), len(payload))

	// Mode dispatch (early, isolé du pipeline de calibration par défaut).
	mode := ""
	if len(os.Args) > 1 {
		mode = os.Args[1]
	}
	switch mode {
	case "trace":
		st, dd, rr := 194126, 380, uint32(2)
		if len(os.Args) > 4 {
			fmt.Sscanf(os.Args[2], "%d", &st)
			fmt.Sscanf(os.Args[3], "%d", &dd)
			fmt.Sscanf(os.Args[4], "%d", &rr)
		}
		traceOne(reg, payload, st, dd, rr)
		return
	case "xuid":
		scanXUIDs(payload, []int{194126, 197378, 200200, 202521, 205826, 208600, 211360, 214200})
		return
	case "chain":
		chainRecords(reg, payload, 194126, 12)
		return
	case "slots":
		probeSecondary(reg, payload, groupLits(findWeaponLits(payload)))
		return
	case "catalog":
		dumpCatalog()
		return
	case "verify2":
		verifySecondLiterals(payload, groupLits(findWeaponLits(payload)))
		return
	case "final":
		buildFinalTable(payload, groupLits(findWeaponLits(payload)))
		return
	case "anat":
		dumpPairAnatomy(payload, groupLits(findWeaponLits(payload)))
		return
	case "db":
		dbCrossCheck()
		return
	case "order":
		dbRosterOrder()
		return
	case "objefull":
		objeForEachRecord(reg, payload, groupLits(findWeaponLits(payload)))
		return
	case "medals":
		dbMedals()
		return
	case "rosters":
		fmt.Printf("\n================ RECHERCHE xuid dans chunks 00/01/26/27 ================\n")
		for _, cn := range []string{"chunk_00", "chunk_01", "chunk_26", "chunk_27"} {
			scanXUIDsInChunk(cn, inflate(cache+"/"+cn+".bin"))
		}
		return
	}

	lits := findWeaponLits(payload)
	fmt.Printf("\n%d littéraux d'armes complets :\n", len(lits))
	for i, l := range lits {
		fmt.Printf("  [%2d] @bit%-7d id64=0x%016x %s\n", i, l.bit, l.id64, l.name)
	}

	// Regroupe en paires : un nouveau groupe quand l'écart au littéral précédent > 1000 bits.
	fmt.Printf("\n--- Regroupement en records (gap>1000) ---\n")
	var groups [][]weaponLit
	for _, l := range lits {
		if len(groups) == 0 || l.bit-groups[len(groups)-1][len(groups[len(groups)-1])-1].bit > 1000 {
			groups = append(groups, []weaponLit{l})
		} else {
			groups[len(groups)-1] = append(groups[len(groups)-1], l)
		}
	}
	for i, g := range groups {
		names := ""
		for _, w := range g {
			names += fmt.Sprintf(" %s@%d", w.name, w.bit)
		}
		fmt.Printf("  groupe %d (%d armes):%s\n", i, len(g), names)
	}

	// PHASE 1+2 : calibre 1 record par groupe (ancré sur le 1er littéral du groupe).
	fmt.Printf("\n================ CALIBRATION 8 RECORDS BIPED ================\n")
	var records []*bipedRecord
	for gi, g := range groups {
		anchor := g[0]
		fmt.Printf("\n>>> groupe %d : ancre %s @bit%d\n", gi, anchor.name, anchor.bit)
		rec := calibrate(reg, payload, anchor.bit, anchor.id64)
		if rec == nil {
			fmt.Printf("    ÉCHEC calibration\n")
			continue
		}
		fmt.Printf("    start=%d d=%d rsp=%d desyncAt=i%d nComps=%d objeStart=%d\n",
			rec.start, rec.d, rec.rsp, rec.desyncAt, rec.nComps, rec.objeStart)
		if rec.objeStart >= 0 {
			lid, lidp, variant, vbit := objePrefix(payload, rec.objeStart)
			fmt.Printf("    obje@%d : LocalID present=%v =0x%08x (%d) | VariantName@%d=0x%08x (%d)\n",
				rec.objeStart, lidp, lid, lid, vbit, variant, variant)
		} else {
			fmt.Printf("    obje ABSENT de la trace (DesyncAt avant i9 ?)\n")
		}
		fmt.Printf("    armes WST dans ce record:")
		for _, w := range rec.weapons {
			fmt.Printf(" %s", w.name)
		}
		fmt.Printf("\n")
		records = append(records, rec)
	}

	// PHASE 3 : table finale + tentative de mapping joueur.
	fmt.Printf("\n================ TABLE FINALE joueur(record) -> armes ================\n")
	for gi, rec := range records {
		var prim, sec string
		if len(rec.weapons) >= 1 {
			prim = rec.weapons[0].name
		}
		if len(rec.weapons) >= 2 {
			sec = rec.weapons[1].name
		}
		objeInfo := "obje:absent"
		if rec.objeStart >= 0 {
			lid, lidp, variant, _ := objePrefix(payload, rec.objeStart)
			objeInfo = fmt.Sprintf("LocalID=0x%08x(present=%v) VariantName=0x%08x", lid, lidp, variant)
		}
		fmt.Printf("  record#%d start=%-7d : primaire=%-16s secondaire=%-16s | %s\n",
			gi, rec.start, prim, sec, objeInfo)
	}

	// Diagnostic : est-ce qu'un LocalID/VariantName du 'obje' matche un xuid (ou un dérivé) ?
	fmt.Printf("\n================ DIAG mapping obje -> xuid ================\n")
	fmt.Printf("pi->xuid de référence :\n")
	for i, x := range piXUID {
		fmt.Printf("  pi%d = %d (0x%016x  low32=0x%08x  high32=0x%08x)\n", i, x, x, uint32(x), uint32(x>>32))
	}
}
