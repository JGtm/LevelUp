//go:build research

package reapparition

// grammaire.go — LES LARGEURS LITTERALES D'UN ECRIVAIN, ET SES APPELS.
//
// `NOTE_3_6_METHODE_DESCRIPTEURS_2026-09-16.md` § 4 : dans l'objet de flux de bits, `+0x2c` est
// le COMPTEUR DE BITS ; chaque `ADD dword ptr [<reg> + 0x2c], N` du desassemblage est un champ de
// N bits, et « la liste ordonnee de ces constantes EST la suite des largeurs ».
//
// DEUX PRECAUTIONS QUE LA NOTE ECRIT ELLE-MEME, ET QUE CE FICHIER APPLIQUE :
//
//	1. UN MEME CHAMP APPARAIT DEUX FOIS — chemin rapide (« la reserve de bits suffit ») et chemin
//	   lent (« il faut recharger »). Le releve publie donc les OCCURRENCES avec leur adresse, et
//	   compte a part les valeurs DISTINCTES : c'est au lecteur de compter les champs, pas les
//	   sites.
//	2. LA LISTE N'EST PAS LA GRAMMAIRE. Les conditions, les boucles et les appels dequantifies
//	   (dont la largeur est un ARGUMENT, pas un litteral) n'y sont pas. C'est pourquoi les cibles
//	   des `CALL rel32` sortent a cote : un ecrivain qui n'a AUCUN `ADD [reg+0x2c], N` et un seul
//	   appel delegue toute sa grammaire, et c'est un resultat en soi.
//
// LE BALAYAGE EST BORNE PAR `.pdata`, jamais par un `RET` devine : hors des bornes exactes de la
// fonction, un motif d'octets n'est pas une instruction, c'est une coincidence.

import (
	"encoding/binary"
	"fmt"
	"sort"
)

// Largeur est UNE occurrence d'avance du compteur de bits, avec l'adresse de l'instruction.
type Largeur struct {
	VA   uint64
	Bits uint32
}

// Appel est une cible de `CALL rel32` relevee dans les bornes de la fonction.
type Appel struct {
	VA, Cible uint64
}

// Releve est ce que l'instrument sait dire d'un ecrivain sans desassembler.
type Releve struct {
	Bornes     FuncRange
	Octets     int
	Largeurs   []Largeur
	Distinctes []uint32
	Appels     []Appel
	// CiblesUniques compte les cibles d'appel distinctes, dans l'ordre d'apparition.
	CiblesUniques []uint64
}

// offsetCompteurBits est le champ `+0x2c` de l'objet de flux : le compteur de bits.
const offsetCompteurBits = 0x2c

// Relever balaye les octets d'une fonction et en tire les largeurs litterales et les appels.
func (e *Executable) Relever(f FuncRange) Releve {
	r := Releve{Bornes: f}
	n := int(f.Fin - f.Debut)
	if n <= 0 {
		return r
	}
	code := e.Lire(f.Debut, n)
	r.Octets = len(code)
	vues := map[uint32]bool{}
	ciblesVues := map[uint64]bool{}

	for i := 0; i < len(code); i++ {
		if bits, ok := lireAvanceCompteur(code, i); ok {
			va := f.Debut + uint64(i)
			r.Largeurs = append(r.Largeurs, Largeur{VA: va, Bits: bits})
			vues[bits] = true
			continue
		}
		if code[i] == 0xe8 && i+5 <= len(code) {
			rel := int32(binary.LittleEndian.Uint32(code[i+1:]))
			suivante := f.Debut + uint64(i) + 5
			cible := uint64(int64(suivante) + int64(rel))
			r.Appels = append(r.Appels, Appel{VA: f.Debut + uint64(i), Cible: cible})
			if !ciblesVues[cible] {
				ciblesVues[cible] = true
				r.CiblesUniques = append(r.CiblesUniques, cible)
			}
		}
	}
	for b := range vues {
		r.Distinctes = append(r.Distinctes, b)
	}
	sort.Slice(r.Distinctes, func(i, j int) bool { return r.Distinctes[i] < r.Distinctes[j] })
	return r
}

// lireAvanceCompteur reconnait `ADD dword ptr [reg + 0x2c], imm` a la position donnee, avec ou
// sans prefixe REX, en immediat court (`0x83 /0`) ou long (`0x81 /0`), avec ou sans SIB.
//
// L'EXTENSION D'OPCODE DOIT VALOIR ZERO : `/0` est ADD ; `/1` est OR, `/5` est SUB. Confondre
// les trois ferait compter comme une largeur ce qui est un recul ou un masque.
func lireAvanceCompteur(code []byte, i int) (uint32, bool) {
	j := i
	if j < len(code) && code[j]&0xf0 == 0x40 { // prefixe REX
		j++
	}
	if j+1 >= len(code) {
		return 0, false
	}
	op := code[j]
	if op != 0x83 && op != 0x81 {
		return 0, false
	}
	modrm := code[j+1]
	if modrm>>6 != 0b01 || (modrm>>3)&0b111 != 0 { // mod=01 (disp8) et /0 (ADD)
		return 0, false
	}
	k := j + 2
	if modrm&0b111 == 0b100 { // rm=100 : un octet SIB precede le deplacement
		k++
	}
	if k >= len(code) || code[k] != offsetCompteurBits {
		return 0, false
	}
	k++
	if op == 0x83 {
		if k >= len(code) {
			return 0, false
		}
		return uint32(code[k]), true
	}
	if k+4 > len(code) {
		return 0, false
	}
	return binary.LittleEndian.Uint32(code[k:]), true
}

// Resume rend le releve en une ligne lisible : largeurs distinctes et nombre d'appels.
func (r Releve) Resume() string {
	if len(r.Largeurs) == 0 {
		return fmt.Sprintf("%d octets · AUCUNE largeur litterale · %d appel(s), %d cible(s)",
			r.Octets, len(r.Appels), len(r.CiblesUniques))
	}
	return fmt.Sprintf("%d octets · %d occurrence(s) de largeur, %v distinctes · %d appel(s), %d cible(s)",
		r.Octets, len(r.Largeurs), r.Distinctes, len(r.Appels), len(r.CiblesUniques))
}
