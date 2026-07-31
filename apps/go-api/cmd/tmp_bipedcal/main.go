// tmp_bipedcal — THROWAWAY : valide la traversée biped corrigée. Brute-force le début
// d'un record biped (R6==35) dans le keyframe chunk_02, ancré sur R0 Hydra@195323,
// en sweepant default-state + recordStateParam. Si TraverseEntity atteint i43 et
// HeldWeapon == une arme connue (Hydra), les desers + la calibration sont bons.
package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"

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
		v = (v << 1) | uint64((d[p>>3]>>uint(7-(p&7)))&1)
	}
	return v
}

func knownHigh32(v uint32) (string, bool) {
	for id, n := range analysis.WeaponIDToName {
		if uint32(id>>32) == v {
			return n, true
		}
	}
	return "", false
}

// knownLow32 vérifie si v correspond au low-32 d'une arme connue.
func knownLow32(v uint32) (string, bool) {
	for id, n := range analysis.WeaponIDToName {
		if uint32(id) == v {
			return n, true
		}
	}
	return "", false
}

// knownAny32 vérifie high-32 ET low-32.
func knownAny32(v uint32) (string, string, bool) {
	hn, hok := knownHigh32(v)
	if hok {
		return hn, "high", true
	}
	ln, lok := knownLow32(v)
	if lok {
		return ln, "low", true
	}
	return "", "", false
}

func main() {
	reg, err := filmdec.ParseRegistryChunk(inflate(cache + "/chunk_00.bin"))
	if err != nil {
		panic(err)
	}
	payload := extractType2(inflate(cache + "/chunk_02.bin"))
	fmt.Printf("registre %d archétypes ; keyframe %d octets\n", len(reg.Archetypes), len(payload))

	// ===== SECTION 1 : vérification directe de ce qui se trouve @ bit 195323 =====
	const hydraBit = 195323
	hydraHigh32 := uint32(0x767db96d)
	hydraLow32 := uint32(0x42c9679f)
	fmt.Printf("\n--- Vérification directe bit 195323 ---\n")
	v32 := uint32(bitsAt(payload, hydraBit, 32))
	fmt.Printf("  bits[195323..195354] = 0x%08x  (Hydra high-32=0x%08x low-32=0x%08x)\n", v32, hydraHigh32, hydraLow32)
	fmt.Printf("  bits[195322..195353] = 0x%08x  (décalage -1)\n", uint32(bitsAt(payload, hydraBit-1, 32)))
	fmt.Printf("  bits[195324..195355] = 0x%08x  (décalage +1)\n", uint32(bitsAt(payload, hydraBit+1, 32)))
	// Chercher si hydraBit est gate=1 d'un WST (handle à +1, variant à +33)
	gate := bitsAt(payload, hydraBit, 1)
	fmt.Printf("  bit 195323 (gate WST?) = %d\n", gate)
	if gate == 1 {
		handle32 := uint32(bitsAt(payload, hydraBit+1, 32))
		variant32 := uint32(bitsAt(payload, hydraBit+33, 32))
		fmt.Printf("  si WST gate@195323 : handle=0x%08x variant=0x%08x\n", handle32, variant32)
	}
	// Est-ce que le high-32 Hydra apparaît quelques bits avant/après 195323 ?
	fmt.Printf("\n--- Scan ±32 bits autour de 195323 pour 0x767db96d ---\n")
	for delta := -32; delta <= 32; delta++ {
		v := uint32(bitsAt(payload, hydraBit+delta, 32))
		if v == hydraHigh32 || v == hydraLow32 {
			fmt.Printf("  MATCH 0x%08x @ bit %d (delta=%+d) %s\n", v,
				hydraBit+delta, delta,
				func() string {
					if v == hydraHigh32 {
						return "== Hydra high-32"
					}
					return "== Hydra low-32"
				}())
		}
	}

	// ===== SECTION 2 : scan large de tous les littéraux R(32) d'armes connues dans la fenêtre =====
	fmt.Printf("\n--- Scan armes connues dans payload[192000..200000] ---\n")
	nFound := 0
	for bp := 192000; bp <= 200000; bp++ {
		v := uint32(bitsAt(payload, bp, 32))
		if n, which, ok := knownAny32(v); ok {
			fmt.Printf("  0x%08x (%s-%s) @ bit %d\n", v, n, which, bp)
			nFound++
			if nFound >= 40 {
				fmt.Printf("  ... (limité à 40)\n")
				break
			}
		}
	}

	// ===== SECTION 3 : depuis les littéraux d'armes trouvés, chercher un record biped =====
	// Idée : si l'arme est la variant (gate@S+0=1, handle@S+1..32, variant@S+33..64),
	// alors S+33 = bit de l'arme. On remonte S = bitArme - 33 et on cherche gate==1 à S.
	// Puis on remonte davantage pour trouver le début du record (typeIndex==35 à ~2500 bits avant).
	fmt.Printf("\n--- Reconstruction inverse depuis variant=Hydra ---\n")
	// On sait que la Hydra high-32 est à bit 195323. Si c'est un champ variant dans WST :
	// StartBit_WST = 195323 - 33 = 195290 (gate + handle précèdent)
	// Mais gate doit être 1 à bit 195290.
	for variantBit := hydraBit - 5; variantBit <= hydraBit+5; variantBit++ {
		if uint32(bitsAt(payload, variantBit, 32)) != hydraHigh32 &&
			uint32(bitsAt(payload, variantBit, 32)) != hydraLow32 {
			continue
		}
		wstStart := variantBit - 33 // gate@wstStart, handle@wstStart+1, variant@wstStart+33
		gateBit := bitsAt(payload, wstStart, 1)
		which := "high32"
		if uint32(bitsAt(payload, variantBit, 32)) == hydraLow32 {
			which = "low32"
		}
		fmt.Printf("  Hydra %s @ bit %d => WST.gate @ bit %d = %d\n", which, variantBit, wstStart, gateBit)
		if gateBit == 1 {
			handle32 := uint32(bitsAt(payload, wstStart+1, 32))
			fmt.Printf("    gate=1 OK : handle=0x%08x\n", handle32)
		}
	}

	// ===== SECTION 4 : sweep brute-force élargi avec low-32 aussi =====
	// Le sweep original ne cherchait que high-32. On l'élargit au low-32 et à une
	// fenêtre de départ plus large (jusqu'à -4500 bits).
	fmt.Printf("\n--- Sweep biped élargi (low32 + fenêtre -4500..-800) ---\n")
	lo2 := hydraBit - 4500
	hi2 := hydraBit - 800
	nStarts2, nHit2 := 0, 0
	for start := lo2; start <= hi2; start++ {
		if uint32(bitsAt(payload, start, 6)) != 35 {
			continue
		}
		nStarts2++
		for d := 30; d <= 160; d += 5 {
			for r := uint32(0); r <= 3; r++ {
				filmdec.SetRecordStateParam(r)
				br := filmdec.NewBitReader(payload)
				br.Skip(start)
				t := filmdec.TraverseEntity(br, reg, d)
				if t.TypeIndex != 35 {
					continue
				}
				for _, c := range t.Comps {
					if c.Name != "weapon-state-type-info" {
						continue
					}
					handle := uint32(bitsAt(payload, c.StartBit+1, 32))
					variant := uint32(bitsAt(payload, c.StartBit+33, 32))
					// vérifier high-32 ET low-32 pour handle et variant
					for _, pair := range [][2]interface{}{
						{handle, "HANDLE"},
						{variant, "VARIANT"},
					} {
						v := pair[0].(uint32)
						field := pair[1].(string)
						if name, which, ok := knownAny32(v); ok {
							nHit2++
							fmt.Printf("HIT2 start=%d d=%d rsp=%d i%d %s(%s)=%s 0x%08x @bit%d desyncAt=i%d\n",
								start, d, r, c.Index, field, which, name, v,
								func() int {
									if field == "HANDLE" {
										return c.StartBit + 1
									}
									return c.StartBit + 33
								}(),
								t.DesyncAt)
						}
					}
				}
				if nHit2 >= 20 {
					goto doneHit2
				}
			}
		}
	}
doneHit2:
	fmt.Printf("%d starts R6==35, %d hits\n", nStarts2, nHit2)

	// ===== SECTION 5 : trace fallback avec diagnostic i43 =====
	fmt.Printf("\n--- Trace fallback (start=193552, d=88, rsp=0) ---\n")
	filmdec.SetRecordStateParam(0)
	br := filmdec.NewBitReader(payload)
	br.Skip(193552)
	t := filmdec.TraverseEntity(br, reg, 88)
	fmt.Printf("typeIndex=%d  %d composants  desyncAt=i%d  endBit=%d\n",
		t.TypeIndex, len(t.Comps), t.DesyncAt, t.EndBit)
	for _, c := range t.Comps {
		extra := ""
		if c.Name == "weapon-state-type-info" {
			gate0 := bitsAt(payload, c.StartBit, 1)
			handle := uint32(bitsAt(payload, c.StartBit+1, 32))
			variant := uint32(bitsAt(payload, c.StartBit+33, 32))
			hn, _, hok := knownAny32(handle)
			vn, _, vok := knownAny32(variant)
			extra = fmt.Sprintf("  gate=%d handle=0x%08x(%s) variant=0x%08x(%s)",
				gate0, handle, pick(hok, hn), variant, pick(vok, vn))
		}
		mark := ""
		if !c.Ported {
			mark = "  <<< DESYNC"
		}
		fmt.Printf("  i%-2d %-44s @bit%d%s%s\n", c.Index, c.Name, c.StartBit, extra, mark)
	}

	// ===== SECTION 5b : analyse détaillée des near-hits trouvés ====
	// Candidate 1 : start=194126, d=153, rsp=0, i43, varBit=195325
	// Candidate 2 : start=194741, d=103, rsp=0, i45, varBit=195318
	fmt.Printf("\n--- Trace détaillée candidat 1 (start=194126, d=153, rsp=0) ---\n")
	filmdec.SetRecordStateParam(0)
	{
		brC := filmdec.NewBitReader(payload)
		brC.Skip(194126)
		tC := filmdec.TraverseEntity(brC, reg, 153)
		fmt.Printf("typeIndex=%d  %d composants  desyncAt=i%d  endBit=%d\n",
			tC.TypeIndex, len(tC.Comps), tC.DesyncAt, tC.EndBit)
		for _, c := range tC.Comps {
			extra := ""
			if c.Name == "weapon-state-type-info" {
				gate0 := bitsAt(payload, c.StartBit, 1)
				handle := uint32(bitsAt(payload, c.StartBit+1, 32))
				variant := uint32(bitsAt(payload, c.StartBit+33, 32))
				n, _, ok := knownAny32(variant)
				extra = fmt.Sprintf("  gate=%d handle=0x%08x variant=0x%08x(%s)",
					gate0, handle, variant, func() string {
						if ok {
							return n
						}
						return "?"
					}())
			}
			mark := ""
			if !c.Ported {
				mark = "  <<< DESYNC"
			}
			fmt.Printf("  i%-2d %-44s @bit%d%s%s\n", c.Index, c.Name, c.StartBit, extra, mark)
		}
	}

	fmt.Printf("\n--- Analyse bits autour de 195323 (±5) ---\n")
	for bp := hydraBit - 5; bp <= hydraBit+5; bp++ {
		b32 := uint32(bitsAt(payload, bp, 32))
		note := ""
		if n, which, ok := knownAny32(b32); ok {
			note = fmt.Sprintf("  <== %s (%s)", n, which)
		}
		fmt.Printf("  bit%d: 0x%08x%s\n", bp, b32, note)
	}

	fmt.Printf("\n--- Structure autour de WST.gate=195290 (hypothèse variant@195323) ---\n")
	// Si variant=195323, gate=195290, bit195290=0 => slot absent => pas une arme
	// Si variant=195323 et c'est le HANDLE (pas variant), alors gate=195290, handle+1..32, puis variant@195323+32
	fmt.Printf("  bit 195290 (hypothèse gate) = %d\n", bitsAt(payload, 195290, 1))
	fmt.Printf("  bit 195290+1..32 (hypothèse handle) = 0x%08x\n", uint32(bitsAt(payload, 195291, 32)))
	fmt.Printf("  bit 195290+33..64 (hypothèse variant) = 0x%08x\n", uint32(bitsAt(payload, 195323, 32)))
	// Est-ce que bit195323 est le handle (gate @ 195322, handle @ 195323, variant @ 195355) ?
	fmt.Printf("  bit 195322 (gate si handle@195323) = %d\n", bitsAt(payload, 195322, 1))
	fmt.Printf("  bit 195323+32 (variant si handle@195323) = 0x%08x\n", uint32(bitsAt(payload, 195355, 32)))
	n355, _, ok355 := knownAny32(uint32(bitsAt(payload, 195355, 32)))
	fmt.Printf("    => 0x%08x est arme connue: %v (%s)\n", uint32(bitsAt(payload, 195355, 32)), ok355, n355)

	// ===== SECTION 6 : HYPOTHÈSE CLÉE =====
	// Le littéral 0x767db96d est à bit 195323. Avant lui, gate=1 @ bit 195322.
	// Structure: gate@195322=1, puis R(32)@195323=0x767db96d (Hydra high), puis R(32)@195355=0x42c9679f (low).
	// => L'ID 64-bit de l'arme est encodé en DEUX R(32) consécutifs: high puis low.
	// => Dans WST: gate=R(1), PUIS le "local-handle" = high-32 et "variant-name" = low-32.
	// => Pour lookup: reconstituer uint64 = (uint64(handle)<<32) | uint64(variant).
	// Cherchons un biped dont WST.gate atterrit à 195322 (= StartBit du WST == 195322).
	fmt.Printf("\n--- Recherche biped dont WST.StartBit == 195322 (gate@195322) ---\n")
	// WST.StartBit = 195322 => gate @ 195322
	// Cherchons aussi WST.StartBit == 195290 (gate@195290=0, slot absent, c'est autre chose)
	// et WST.StartBit dans [195290..195330] de façon générale
	targetGateLo, targetGateHi := 195310, 195330
	nS4, nH4 := 0, 0
	for start := hydraBit - 5000; start <= hydraBit-100; start++ {
		if uint32(bitsAt(payload, start, 6)) != 35 {
			continue
		}
		nS4++
		for d := 20; d <= 200; d++ {
			for r := uint32(0); r <= 3; r++ {
				filmdec.SetRecordStateParam(r)
				brx := filmdec.NewBitReader(payload)
				brx.Skip(start)
				tx := filmdec.TraverseEntity(brx, reg, d)
				if tx.TypeIndex != 35 {
					continue
				}
				for _, c := range tx.Comps {
					if c.Name != "weapon-state-type-info" {
						continue
					}
					if c.StartBit >= targetGateLo && c.StartBit <= targetGateHi {
						nH4++
						gate0 := bitsAt(payload, c.StartBit, 1)
						handle := uint32(bitsAt(payload, c.StartBit+1, 32))
						variant := uint32(bitsAt(payload, c.StartBit+33, 32))
						// reconstruire l'ID 64-bit
						id64 := (uint64(handle) << 32) | uint64(variant)
						wname, wok := analysis.WeaponIDToName[id64]
						fmt.Printf("  HIT6 start=%d d=%d rsp=%d i%d WST.start=%d gate=%d handle=0x%08x variant=0x%08x id64=0x%016x arme=%s desyncAt=%d\n",
							start, d, r, c.Index, c.StartBit, gate0, handle, variant, id64,
							func() string {
								if wok {
									return wname
								}
								return "?"
							}(), tx.DesyncAt)
					}
				}
				if nH4 >= 30 {
					goto doneH4
				}
			}
		}
	}
doneH4:
	fmt.Printf("%d starts, %d WST-gate hits in [%d,%d]\n", nS4, nH4, targetGateLo, targetGateHi)

	// ===== SECTION 7 : vérification directe de l'hypothèse gate@195322 =====
	fmt.Printf("\n--- Vérification directe gate@195322 : id64 = (handle<<32)|variant ---\n")
	for gateAt := 195318; gateAt <= 195326; gateAt++ {
		gate0 := bitsAt(payload, gateAt, 1)
		handle := uint32(bitsAt(payload, gateAt+1, 32))
		variant := uint32(bitsAt(payload, gateAt+33, 32))
		id64 := (uint64(handle) << 32) | uint64(variant)
		wname, wok := analysis.WeaponIDToName[id64]
		fmt.Printf("  gate@%d=%d handle=0x%08x variant=0x%08x id64=0x%016x arme=%s\n",
			gateAt, gate0, handle, variant, id64,
			func() string {
				if wok {
					return wname
				}
				return "?"
			}())
	}

	// ===== SECTION 8 : recherche record biped exact dont WST.start = 195322 =====
	// Les near-hits en Section 6 montrent que start=194741,d=103,rsp=0 place i45 à 195318.
	// On cherche start+d combos qui placent WST.start EXACTEMENT à 195322.
	fmt.Printf("\n--- Sweep fin : WST.StartBit == 195322 (exact) ---\n")
	nS5, nH5 := 0, 0
	for start := 192000; start <= 195200; start++ {
		if uint32(bitsAt(payload, start, 6)) != 35 {
			continue
		}
		nS5++
		for d := 1; d <= 250; d++ {
			for r := uint32(0); r <= 3; r++ {
				filmdec.SetRecordStateParam(r)
				brx := filmdec.NewBitReader(payload)
				brx.Skip(start)
				tx := filmdec.TraverseEntity(brx, reg, d)
				if tx.TypeIndex != 35 {
					continue
				}
				for _, c := range tx.Comps {
					if c.Name != "weapon-state-type-info" || c.StartBit != 195322 {
						continue
					}
					nH5++
					gate0 := bitsAt(payload, c.StartBit, 1)
					handle := uint32(bitsAt(payload, c.StartBit+1, 32))
					variant := uint32(bitsAt(payload, c.StartBit+33, 32))
					id64 := (uint64(handle) << 32) | uint64(variant)
					wname, wok := analysis.WeaponIDToName[id64]
					fmt.Printf("  EXACT start=%d d=%d rsp=%d i%d gate=%d handle=0x%08x variant=0x%08x id64=0x%016x arme=%s desyncAt=%d\n",
						start, d, r, c.Index, gate0, handle, variant, id64,
						func() string {
							if wok {
								return wname
							}
							return "?"
						}(), tx.DesyncAt)
				}
				if nH5 >= 20 {
					goto doneH5
				}
			}
		}
	}
doneH5:
	fmt.Printf("%d starts, %d EXACT hits\n", nS5, nH5)

	// ===== SECTION 9 : typeIndex scan autour de 195322 =====
	// Cherchons quels typeIndex R(6) sont présents dans [194500..195500].
	// Un record d'arme (weapon entity) a un typeIndex différent de 35 (biped).
	fmt.Printf("\n--- typeIndex R(6) proches de 195322 ---\n")
	// Scan : chercher tous les starts où R(6) donne un typeIndex d'une arche connue
	fmt.Printf("  archétype 0..63 présents dans [194000,195500] :\n")
	counts := make(map[uint32]int)
	for bp := 194000; bp <= 195500; bp++ {
		ti := uint32(bitsAt(payload, bp, 6))
		counts[ti]++
	}
	for ti := uint32(0); ti < 64; ti++ {
		if c2 := counts[ti]; c2 > 0 {
			arch, ok := reg.Archetype(int(ti))
			archName := "?"
			if ok && len(arch.Components) > 0 {
				archName = arch.Components[0]
			}
			fmt.Printf("    typeIndex=%d (%s) : %d occurrences\n", ti, archName, c2)
		}
	}

	// Cherchons spécifiquement les records dont la traversée placerait un composant
	// "weapon-state-type-info" avec id64=Hydra, toutes typeIndex confondus.
	fmt.Printf("\n--- Sweep TOUS typeIndex, WST.gate@195322 => Hydra ---\n")
	nS6, nH6 := 0, 0
	for start := 192000; start <= 195300; start++ {
		nS6++
		for d := 1; d <= 250; d++ {
			for r := uint32(0); r <= 3; r++ {
				filmdec.SetRecordStateParam(r)
				brx := filmdec.NewBitReader(payload)
				brx.Skip(start)
				tx := filmdec.TraverseEntity(brx, reg, d)
				for _, c := range tx.Comps {
					if c.Name != "weapon-state-type-info" || c.StartBit != 195322 {
						continue
					}
					gate0 := bitsAt(payload, c.StartBit, 1)
					if gate0 != 1 {
						continue
					}
					handle := uint32(bitsAt(payload, c.StartBit+1, 32))
					variant := uint32(bitsAt(payload, c.StartBit+33, 32))
					id64 := (uint64(handle) << 32) | uint64(variant)
					wname, wok := analysis.WeaponIDToName[id64]
					if !wok {
						continue
					}
					nH6++
					fmt.Printf("  FOUND start=%d d=%d rsp=%d typeIndex=%d i%d gate=%d id64=0x%016x arme=%s desyncAt=%d\n",
						start, d, r, tx.TypeIndex, c.Index, gate0, id64, wname, tx.DesyncAt)
				}
				if nH6 >= 10 {
					goto doneH6
				}
			}
		}
	}
doneH6:
	fmt.Printf("%d starts scanned, %d hits\n", nS6, nH6)
}

func pick(ok bool, s string) string {
	if ok {
		return s
	}
	return "?"
}
