package filmdec

// playergameevent_0xe9_helpers_test.go — decodeur et collecteurs de l'instrument
// PlayerGameEventSmall (0xE9, type 82). Voir l'en-tete de
// playergameevent_0xe9_research_test.go pour le raisonnement, la grammaire sourcee de
// l'exe et les mesures.

import (
	"sort"
	"testing"
)

// Domaines des trois references d'en-tete du type 82, LUS DANS L'EXE (descripteur
// PlayerGameEventSmall = objet 0x143d0ec18, champ vtable+0x58 = fonction de domaine
// 0x142ef7f6c) : index0 -> domaine 0, index1 -> domaine 8, index2 -> domaine 7. Tous de
// largeur 13 (lot1RefDomWidths), sans sonde (la sonde est propre au domaine 1).
const (
	pgesDom0Width = 13
	pgesDom8Width = 13
	pgesDom7Width = 13
)

// pgesProp : une propriete nommee de la liste (nom hache R(32) + selecteur R(3) + valeur
// typee selon le selecteur).
type pgesProp struct {
	name   uint64
	sel    int
	val    uint64
	hasVal bool
}

// pgesPayload : ce que rend la charge PlayerGameEventSmall (lecteur FUN_14080add8 ->
// FUN_14080ae70 + finaliseur FUN_14080ae28).
type pgesPayload struct {
	fieldA       uint64 // R(32) — identifiant/type de l'evenement (out[0])
	fieldB       uint64 // R(8)  — champ court (out+8)
	props        []pgesProp
	hasText      bool
	textName     uint64
	participants []uint64 // sous-type "text" #1 : index de participant R(5) (espace killsource)
	textParts    int
	exact        bool // false si un selecteur a largeur runtime (7, ou "text" #2 quantifie) est atteint
}

// pgesRef consomme une reference gardee de largeur width : [R(1) porte ; si 1 : R(width)
// index + R(2) generation]. Rend (index, presente).
func pgesRef(br *BitReader, width uint) (uint64, bool) {
	if !br.ReadBit() {
		return 0, false
	}
	idx := br.ReadBits(width)
	br.Skip(2)
	return idx, true
}

// pgesDecodePayload consomme la charge du type 82 EXACTEMENT (grammaire de l'exe) a partir
// du bit courant (apres les 3 references d'en-tete). Ordre : R(32) A, R(8) B, liste de
// proprietes, bloc "text" optionnel, R(32) masque final.
//
// Largeurs de valeur par selecteur R(3) (FUN_14080eff0) :
//
//	0 -> 0 bit (drapeau)                    1 -> R(32) (int)          2 -> R(32) (string-id)
//	3 -> R(32)                              4 -> R(1) (bool)          6 -> R(32)
//	5 -> chaine (R(8) jusqu'a l'octet 0, max 16 octets ; FUN_1407cbc24)
//	7 -> palette quantifiee a largeur runtime (FUN_140f04f18) : NON reproduite -> exact=false
func pgesDecodePayload(br *BitReader) pgesPayload {
	p := pgesPayload{exact: true}
	p.fieldA = br.ReadBits(32)
	p.fieldB = br.ReadBits(8)
	cnt := int(br.ReadBits(3))
	for i := 0; i < cnt; i++ {
		pr := pgesProp{name: br.ReadBits(32), sel: int(br.ReadBits(3))}
		switch pr.sel {
		case 0:
			// drapeau : 0 bit de valeur
		case 1, 2, 3, 6:
			pr.val, pr.hasVal = br.ReadBits(32), true
		case 4:
			pr.val, pr.hasVal = br.ReadBits(1), true
		case 5:
			for k := 0; k < 16; k++ {
				if br.ReadBits(8) == 0 {
					break
				}
			}
		case 7:
			p.exact = false
		}
		p.props = append(p.props, pr)
	}
	// Bloc "text" optionnel (FUN_14080b034) : R(1) porte ; si 1 : R(32) nom, R(3) compte,
	// compte x element (FUN_1407f0ebc) : R(3) sous-type puis valeur.
	if br.ReadBit() {
		p.hasText = true
		p.textName = br.ReadBits(32)
		tc := int(br.ReadBits(3))
		p.textParts = tc
		for i := 0; i < tc; i++ {
			switch int(br.ReadBits(3)) {
			case 0:
				// 0 bit
			case 1:
				// FUN_1407f2058 : R(1) porte ; si 0 : R(5) index de participant absolu
				if !br.ReadBit() {
					p.participants = append(p.participants, br.ReadBits(5))
				}
			case 3:
				br.Skip(32) // string_id
			case 2:
				p.exact = false // FUN_142c70cd0 : quantifie a largeur runtime
			default:
				br.Skip(32) // 4..7 : R(32)
			}
		}
	}
	br.Skip(32) // masque final (FUN_14080ae28 : 32 x R(1))
	return p
}

// pgesRecord : un evenement 0xE9 type 82 horodate, refs d'en-tete resolues + charge.
type pgesRecord struct {
	ts               uint64
	ref0, ref1, ref2 int64 // index bruts ; -1 si absent
	payload          pgesPayload
	bitAfter         int // position bit apres la charge (pour l'oracle de trame)
}

// pgesDecodePacket decode un paquet 0xE9 : config, continuation, type (82 attendu), 3 refs
// (domaines 0/8/7), puis la charge. Rend (record, estType82).
func pgesDecodePacket(pay []byte, ts uint64) (pgesRecord, bool) {
	br := NewBitReader(pay)
	br.Skip(1) // bit de configuration
	if !br.ReadBit() {
		return pgesRecord{}, false // liste vide : impossible pour 0xE9 (bit 1 = 1)
	}
	if br.ReadBits(7) != 82 { // 83 = TeamGameEvent
		return pgesRecord{}, false
	}
	r := pgesRecord{ts: ts, ref0: -1, ref1: -1, ref2: -1}
	if v, ok := pgesRef(br, pgesDom0Width); ok {
		r.ref0 = int64(v)
	}
	if v, ok := pgesRef(br, pgesDom8Width); ok {
		r.ref1 = int64(v)
	}
	if v, ok := pgesRef(br, pgesDom7Width); ok {
		r.ref2 = int64(v)
	}
	r.payload = pgesDecodePayload(br)
	r.bitAfter = br.BitPos()
	return r, true
}

// pgesResolveBiped rend vrai si (base+idx) lie a un bipede dans le monde w.
func pgesResolveBiped(w *World, base int, idx int64) bool {
	if idx < 0 {
		return false
	}
	slot := base + int(idx)
	if slot < 0 || slot >= 8192 {
		return false
	}
	ti, ok := w.ArchetypeForSlot(uint32(slot))
	return ok && ti == BipedTypeIndex
}

// pgesCollectShotSets construit les ensembles de WeaponID (32 bits bas et haut) et les
// horodatages de tir tries, a partir des tirs 0xD2 t36 (reuse exploCollectShots).
func pgesCollectShotSets(t *testing.T, dir string, n int) (widLo, widHi map[uint64]bool, shotTs []uint64) {
	widLo, widHi = map[uint64]bool{}, map[uint64]bool{}
	for _, s := range exploCollectShots(t, dir, n) {
		if !s.has {
			continue
		}
		widLo[s.wid&0xFFFFFFFF] = true
		widHi[s.wid>>32] = true
		shotTs = append(shotTs, s.ts)
	}
	sort.Slice(shotTs, func(a, b int) bool { return shotTs[a] < shotTs[b] })
	return widLo, widHi, shotTs
}
