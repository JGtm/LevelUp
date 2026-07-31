package main

import (
	"encoding/binary"
	"fmt"
	"os"
)

// walker.go — WALKER DÉTERMINISTE de la boucle d'events du paquet film (reframe 2026-07-11, RE_LOG §7ter).
// Un paquet = boucle : R(1)continue ; R(7)type ; 3×R(1)présence ; déser(type) ; [trailer=0]. Le kill-event =
// type R7=85 (0x14104bd08). But : walker depuis le début du paquet en sautant chaque event par la longueur de
// son déser, jusqu'à R7=85 -> curseur EXACT (déterministe, remplace le scan locator 87%).
// Longueurs d'event (workflow filmdec-walker-events) : stub(13 codes)=0 ; code15 = [gate15?15]+13+10+N ;
// code36 = deserDamageV2 (déser dégât) ; code82 = u32+u8+props+opt+32b. Encadrement = 11 bits/event.

var stubCodes = map[int]bool{3: true, 23: true, 24: true, 25: true, 26: true, 33: true, 49: true, 54: true, 57: true, 59: true, 92: true, 103: true}

// fixedLen : events à longueur de déser FIXE (min==max en vérité-terrain dispatcher). Skip déterministe exact.
// Sources : table empirique (gap-11) + confirmation modelcheck (delta==0 sur toutes les occurrences).
var fixedLen = map[int]int{4: 15, 5: 121, 6: 103, 9: 46, 21: 13, 38: 20, 75: 66, 76: 104}

func atoiSafe(s string) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return -1
		}
		n = n*10 + int(c-'0')
	}
	if s == "" {
		return -1
	}
	return n
}

// presenceField : FUN_1406d3140 — un champ de présence config-quantifié. R(1) flag ; si set qbits(0x200)=9,
// sinon qbits(0x1FFF)=13 ; puis R(2). Validé 100% code 1 / 99% code 0 en vérité-terrain dispatcher.
func presenceField(r *br) {
	if r.g1() == 1 {
		r.R(9)
	} else {
		r.R(13)
	}
	r.R(2)
}

// presenceLoop : boucle de présence du dispatcher (EventDispatch_Generic_ReadType7). 3 slots ; chaque slot
// présent (R(1)==1) lit un presenceField. C'EST le vrai encadrement variable (remplace le faux "3 bits").
func presenceLoop(r *br) {
	for i := 0; i < 3; i++ {
		if r.g1() == 1 {
			presenceField(r)
		}
	}
}

// has15 : état runtime du gate config de code 15 (FUN_1404f25f4()==2). N'est PAS un bit du flux -> déterminé
// empiriquement par film (presweep : 000d5950 -> true, 28/28 exact). Field A (R(15)) présent si true, 0 bit sinon.
var has15 = true

// skipEvent15 : déser code 15 (FUN_14080bb4c). [si has15: R(15)] + R(13) + R(10)count + R(count). Le count est
// une LONGUEUR EN BITS (pas un nb de records). Confirmé au désassemblage (agent Ghidra).
func skipEvent15(r *br, gate15 bool) {
	if gate15 {
		r.R(15)
	}
	r.R(13)
	n := int(r.R(10))
	r.R(n)
}

// variantEf08 : FUN_14080ef08 lit un tag 3 bits puis FUN_14080eff0 dispatche. Résolu au désassemblage :
// tag0=0b ; tag1/2/3=R(32) ; tag4=R(1) ; tag5=R(128) ; tag6=R(32) ; tag7=transform config-quantifié NON résolu.
func variantEf08(r *br) bool {
	switch int(r.R(3)) {
	case 0: // écrit un octet de sortie, 0 bit lu
	case 1:
		r.R(32) // FUN_140f2734c
	case 2:
		r.R(32) // FUN_14080dec4 (string-id, 32 bits — PAS 0)
	case 3:
		r.R(32) // FUN_14080f0c4
	case 4:
		r.R(1) // FUN_1406cf008
	case 5:
		r.R(128) // FUN_1407cbc24(0x10)
	case 6:
		r.R(32) // FUN_140f04e28 (+ lookup ECS 0 bit)
	default:
		return false // tag7 : FUN_140f04f18, vecteur monde quantifié config-runtime — non modélisable statiquement
	}
	return true
}

// variant7f0ebc : FUN_1407f0ebc lit un tag 3 bits inline puis dispatche. Résolu au désassemblage :
// tag0=0b ; tag1=E5(1|6b) ; tag2=R(1)+{0:R(32)|1:R(24)} ; tag3=R(32) ; tag>=4=R(32).
func variant7f0ebc(r *br) bool {
	switch int(r.R(3)) {
	case 0: // écrit un octet, 0 bit
	case 1: // ECS_ReadEntityRefIndex5 : R(1) gate, +R(5) si gate==0
		if r.g1() == 0 {
			r.R(5)
		}
	case 2: // FUN_142c70cd0 : R(1) flag ; 0 -> R(32) ; 1 -> R(24) quantifié (FUN_1406d84b4 w=0x18)
		if r.g1() == 0 {
			r.R(32)
		} else {
			r.R(24)
		}
	case 3:
		r.R(32) // FUN_14080dec4 (string_id)
	default:
		r.R(32) // tag>=4 : FUN_142c70be0
	}
	return true
}

func skipEvent82(r *br) bool {
	r.R(32)
	r.R(8)
	n1 := int(r.R(3))
	for i := 0; i < n1; i++ {
		r.R(32) // name
		if !variantEf08(r) {
			return false
		}
	}
	if r.g1() == 1 {
		r.R(32)
		n2 := int(r.R(3))
		for i := 0; i < n2; i++ {
			if !variant7f0ebc(r) {
				return false
			}
		}
	}
	r.R(32) // masque 32 bits
	return true
}

// walkEvent0 : déser event CODE 0 (FUN_1407f15a4, param5=1). Modèle bit-exact (workflow, high ;
// plancher 84 / max 241 = borne empirique). La branche rare @0x68 lit qbits(config) — w=13 (DAT_1451f98d4=7679).
func walkEvent0(r *br) {
	if r.g1() != 0 {
		r.R(32)
	}
	if r.g1() == 0 {
		r.R(5)
	}
	r.R(19)
	if r.g1() != 0 {
		r.R(19)
		r.R(12)
	}
	r.R(5)
	r.R(5)
	r.R(6)
	if r.g1() != 0 {
		r.R(5)
	}
	r.R(14)
	if r.g1() != 0 {
		r.R(32)
	}
	r.R(1)
	r.R(3)
	r.R(5)
	r.R(5)
	r.R(1)
	r.R(4)
	if r.g1() == 0 {
		r.R(10)
	}
	v58 := int(r.R(4))
	if r.g1() != 0 {
		r.R(32)
	}
	if v58 == 1 {
		r.R(8)
	}
	r.R(4)
	if r.g1() != 0 {
		r.R(13) // qbits(config) ; puis
		r.R(2)
	}
}

// skipEvent1 : déser event CODE 1 (FUN_140968368, param5=1). Modèle bit-exact (workflow, high ; 10-33 bits).
// skipEvent1 : déser SEUL de l'event code 1 (FUN_140968368, param5=1). Confirmé au désassemblage :
// R(5) + [R(1)+opt R(4)] (FUN_1409684dc) + R(3) (FUN_1424d0f48) + R(1) présence + [si présent: R(19)] (MOV R9D,0x13).
// 10-33 bits. Le +16 vu en vérité-terrain N'EST PAS ici : c'est la BOUCLE DE PRÉSENCE du dispatcher (framing).
func skipEvent1(r *br) {
	r.R(5)
	if r.g1() == 0 {
		r.R(4)
	}
	r.R(3)
	if r.g1() != 0 {
		r.R(19)
	}
}

// skipEvent7 : déser event code 7 (FUN_142f1c6cc, vtable+0x68). Modèle bit-exact (désassemblage, high).
// 118 bits fixes + bloc conditionnel (present==0 -> R(1)+optR(32)). Min 118 / max 151. Ne renvoie jamais false.
func skipEvent7(r *br) bool {
	if r.g1() == 0 { // [+0x3d] gate présent/inline
		if r.g1() != 0 {
			r.R(32) // variant-id (FUN_14080d69c/d6f0)
		}
	}
	r.R(7)  // float quantifié 7b
	r.R(7)  // float quantifié 7b
	r.R(19) // FUN_14076dc04
	r.R(2)  // sélecteur 2b
	r.R(12) // vec[0]
	r.R(12) // vec[1]
	r.R(12) // vec[2]
	r.R(19) // FUN_14076dc04
	r.R(9)  // inline 9b
	r.R(16) // inline 16b
	r.g1()  // flag
	r.g1()  // flag
	return true
}

// skipEventDeser : avance r par la longueur du déser de l'event `code` (le corps, hors encadrement). Renvoie
// false si le code n'est pas modélisé (walk impossible au-delà).
func skipEventDeser(r *br, code int, gate15 bool) bool {
	switch {
	case stubCodes[code]:
		return true // 0 bit
	case fixedLen[code] != 0:
		r.bp += fixedLen[code]
		return true
	case code == 0:
		walkEvent0(r)
		return true
	case code == 1:
		skipEvent1(r)
		return true
	case code == 7:
		return skipEvent7(r)
	case code == 15:
		skipEvent15(r, gate15)
		return true
	case code == 36:
		L := deserDamageV2(r.pl, dwV2{base: r.bp, e494: 67, tail84: 4, guard: 0})
		r.bp += L
		return true
	case code == 82:
		return skipEvent82(r)
	default:
		return false
	}
}

// code36Variant : lit le variant_name (catégorie d'arme, R(32)) d'un event dégât code 36 (FUN_14080c1f8).
// Le variant flotte entre bit 13 et 52 : préfixe 10 bits + 3 champs présence-gatés (ECS 1|6, opt2 1|3,
// opt32 1|33). Confirmé au désassemblage (agent Ghidra, HIGH). r doit être au DÉBUT du corps du déser 36.
func code36Variant(r *br) uint32 {
	r.g1()           // fA (compact/full)
	r.g1()           // f1c (extended actor)
	r.R(7)           // FUN_141fcf670
	r.g1()           //   + 1 bit
	if r.g1() == 0 { // ECS_ReadEntityRefIndex5
		r.R(5)
	}
	if r.g1() == 0 { // FUN_1406d00ec
		r.R(2)
	}
	if r.g1() == 1 { // FUN_14080d69c
		r.R(32)
	}
	return uint32(r.R(32)) // variant_name (FUN_14080dec4)
}

// readE5 : ECS_ReadEntityRefIndex5 (FUN_1407f2058). R(1) gate ; si 0 -> R(5) index local (0-31) ; sinon -1.
func readE5(r *br) int {
	if r.g1() == 0 {
		return int(r.R(5))
	}
	return -1
}

// killEvent : sortie du déser kill (code 85, DeserKillEvent_ECS FUN_14104bd08).
type killEvent struct{ killer, victim, assist int }

// readKillEvent : déser du kill-event à partir du curseur (après R7 + boucle de présence). Confirmé au
// désassemblage : E5 killer + E5 victim + R(32) + R(1) + E5 assist + R(32) [+ tail runtime-gated ignoré].
func readKillEvent(r *br) killEvent {
	k := readE5(r)
	v := readE5(r)
	r.R(32)
	r.R(1)
	a := readE5(r)
	r.R(32)
	return killEvent{k, v, a}
}

// walkToKill : walke la boucle d'events depuis `start`. Renvoie (position R7 du kill-event, code d'arrêt).
// code=85 -> kill trouvé (posR7 = bit du R7 du kill, comparable à la vérité dispatcher) ; sinon code = l'event
// non modélisé ou -1 fin/-2 end. Le curseur du déser kill = posR7 + 7 + boucle de présence.
func walkToKill(pl []byte, start int, gate15 bool) (int, int) {
	r := &br{pl: pl, bp: start}
	for r.bp+11 <= len(pl)*8 {
		if r.g1() == 0 {
			return -1, -2 // fin de paquet (continue=0)
		}
		posR7 := r.bp
		t := int(r.R(7))
		if t >= 123 {
			return -1, -3
		}
		presenceLoop(r) // encadrement variable (boucle de présence du dispatcher)
		if t == 85 {
			return posR7, 85 // position R7 du kill (vérité dispatcher = pos du R7)
		}
		if !skipEventDeser(r, t, gate15) {
			return -1, t // event non modélisé
		}
	}
	return -1, -1
}

// runWalkDump : pour les premiers paquets fatals, walke depuis bit 0 et log la SÉQUENCE d'events (code@bp)
// jusqu'au kill (85) / event non modélisé / fin. Montre empiriquement la structure du paquet.
func runWalkDump(m string) {
	start := 0
	gate15 := false
	for _, a := range os.Args[3:] {
		if a == "g15" {
			gate15 = true
		} else if n := atoiSafe(a); n >= 0 {
			start = n
		}
	}
	cache := root + "/" + m
	dmgMk := map[byte]bool{0xD2: true, 0xC0: true, 0xC2: true, 0xC3: true, 0xCA: true, 0xD3: true, 0xE9: true}
	fmt.Printf("=== WALKDUMP %s (start=%d gate15=%v) ===\n", m, start, gate15)
	shown := 0
	for ch := 0; ch <= 41 && shown < 25; ch++ {
		d := inflate(fmt.Sprintf("%s/chunk_%02d.bin", cache, ch))
		off := 0
		for off+16 <= len(d) && shown < 25 {
			typ := binary.LittleEndian.Uint16(d[off:])
			sz := int(binary.LittleEndian.Uint32(d[off+4:]))
			if sz <= 0 || off+16+sz > len(d) {
				break
			}
			pl := d[off+16 : off+16+sz]
			off += 16 + sz
			if typ != 0 || len(pl) == 0 || !dmgMk[pl[0]] || sz < 700 {
				continue
			}
			// walke en loggant chaque event
			r := &br{pl: pl, bp: start}
			seq := ""
			steps := 0
			for r.bp+11 <= len(pl)*8 && steps < 40 {
				startBp := r.bp
				if r.g1() == 0 {
					seq += " END"
					break
				}
				t := int(r.R(7))
				r.R(3)
				seq += fmt.Sprintf(" [%d@%d]", t, startBp)
				if t == 85 {
					seq += "=KILL"
					break
				}
				if t >= 123 {
					seq += "!bad"
					break
				}
				if !skipEventDeser(r, t, gate15) {
					seq += "!unk"
					break
				}
				steps++
			}
			fmt.Printf("mk=0x%02X sz=%d :%s\n", pl[0], sz, seq)
			shown++
		}
	}
}
