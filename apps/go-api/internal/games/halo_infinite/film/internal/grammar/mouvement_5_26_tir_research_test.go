//go:build research

package grammar

// mouvement_5_26_tir_research_test.go — LE TIR D ARME COMME ACTE DE NAISSANCE ? (lot 5.26.2,
// mesure seule.)
//
// CE QUE L ECRIVAIN DIT DU RECORD 36 (`action_weapon_fire`), EN ENTIER (Ghidra, lecture seule,
// `.ai/V7.5/film_re/NOTE_5_26_*`) :
//
//	repartiteur `FUN_14080a9d4` : R(7) type ; puis TROIS references gardees — R(1) garde, et si
//	    posee `FUN_1406d3140(lecteur, domaine, &ref)` — dont le domaine est rendu par la vtable du
//	    descripteur (`0x143d0aca0`, entree +0x58 -> `0x14080a048`) : ref0 = domaine 1, ref1 =
//	    domaine 8, ref2 = domaine 7. Le domaine 7 est celui de la boucle de records
//	    (`DAT_1451f9908` = `DAT_1451f98d0 + 7*8`) : c est la SEULE reference du record dans
//	    l espace d identifiants des trames delta. Puis le corps, `FUN_14080c1f8`, appele avec
//	    son cinquieme argument a 1 (`MOV byte ptr [RSP+0x20],0x1` en `0x14080aad1`).
//	corps `FUN_14080c1f8` : R(1) variante courte, R(1) bloc, `FUN_141fcf670` (attaquant),
//	    `FUN_1407f2034`, `FUN_1406d00ec`, `FUN_14080d69c` (arme, famille), `FUN_14080dec4`
//	    (arme, variante R(32)), R(1), R(1), [bloc : R(1), R(1), horodatage] ; si court : fin ;
//	    `FUN_14080cc68` (comptes cibles / composantes), boucle de composantes (identifiant de
//	    DOMAINE 1, `0x14080c580`), boucle de cibles, `FUN_140c9e4d8` -> `FUN_140c9e990` (reference
//	    typee : genre R(2), domaine = genre, 1 ou 2), `FUN_1408eff64` (reference typee, idem),
//	    R(30) visee, puis des champs sans identifiant.
//
// AUCUN champ du corps n est dans le domaine 7 ; aucun ne porte un archetype. Si le record nomme
// l entite rejetee, c est par ref2 (ou, a la rigueur, ref0 / ref1 si les bases de domaine
// coincident). La mesure le verifie sur les eid dont le PREMIER rejet tombe dans un paquet dont
// l evenement de tete est 36 (liste de `TestTicks525`, UN decodage) :
//
//	(a) les trois references, decodees a leur place, contre l eid rejete : a l identique
//	    ((index + base) | tete<<30) et en (index, tete) bruts ;
//	(b) un balayage bit a bit de l etendue du record que la grammaire localise (jusqu a la fin
//	    de la visee pour un record modal, jusqu a la fin des references sinon), contre un TEMOIN
//	    (l eid rejete d un AUTRE paquet cible) qui mesure le hasard ;
//	(c) sur TOUS les paquets de tete 36 du film : ce que ref2 designe (declare par une
//	    image-cle, rejete, ou ni l un ni l autre).
//
// Rejouable (un film a la fois) :
//
//	MOUV511_FILM=<dir> MOUV511_BORNES=<catalogue> MOUV511_CARTE=<carte> \
//	  go test -tags=research -count=1 -v -timeout 60m -run '^TestTir526$' \
//	  ./internal/games/halo_infinite/film/internal/grammar/

import (
	"sort"
	"testing"
)

// t526TeteTir est le type d evenement de tete du record de tir (numerotation TRAME), le meme que
// [TypeTirArme].
const t526TeteTir = TypeTirArme

// t526Refs sont les trois references gardees du repartiteur, decodees a leur place.
type t526Refs struct {
	r0, r1, r2 guardedRef
}

// t526LireRefs decode les trois references du record de tete, dans les domaines que
// l ecrivain donne (1, 8, 7), a partir du bit qui suit le preambule de 9 bits.
func t526LireRefs(pay []byte) t526Refs {
	var r t526Refs
	r.r0 = readDom1Ref(pay, eventPayloadStartBit)
	r.r1 = readPlainRef(pay, r.r0.EndBit, int(refDomWidth(8)))
	r.r2 = readPlainRef(pay, r.r1.EndBit, int(refDomWidth(7)))
	return r
}

// t526Eid compose l eid d une reference sur la base du domaine des trames (`cfg.IDBase`).
func t526Eid(g guardedRef, base uint32) uint32 {
	return (g.Index+base)&0x3fffffff | g.Gen<<30
}

// t526Egal dit si une reference designe l eid : a l identique (base des trames) ou en (index,
// tete) bruts.
func t526Egal(g guardedRef, eid, base uint32) (exact, brut bool) {
	if !g.Present {
		return false, false
	}
	exact = t526Eid(g, base) == eid
	brut = g.Index == eid&0x3fffffff && g.Gen == eid>>30
	return exact, brut
}

// TestTir526 joue la passe de `TestTicks525` et confronte le record 36 aux eid rejetes.
func TestTir526(t *testing.T) {
	tc := t516Cadre(t)
	p := t525Marcher(tc) // LE decodage de la mesure
	payloads := t526Payloads(tc)
	premiers, rejetes := t526Premiers(p)
	cibles := make([]int, 0)
	for _, i := range premiers {
		if p.tr[i].tete == t526TeteTir {
			cibles = append(cibles, i)
		}
	}
	sort.Ints(cibles) // l ordre des trames : le temoin de (b) est reproductible
	t.Logf("(0) %d eid rejetes · %d dont le premier rejet tombe dans un paquet de tete 36 · "+
		"IDBase %d · IDLowBits %d", len(premiers), len(cibles), tc.cfg.IDBase, tc.cfg.IDLowBits)
	t526TableauA(t, p, payloads, cibles, tc.cfg)
	t526TableauB(t, p, payloads, cibles, tc.cfg)
	t526TableauC(t, p, payloads, rejetes, tc.cfg)
}

// t526Payloads indexe les payloads delta par (chunk, rang) — une lecture d octets, aucun
// decodage.
func t526Payloads(tc t516Temoin) map[h526Cle][]byte {
	out := map[h526Cle][]byte{}
	for _, c := range tc.fc.ChunkNumbers() {
		data, pks, ok := tc.fc.ChunkAt(c)
		if !ok {
			continue
		}
		for _, pk := range pks {
			if pk.Type == PacketTypeDelta && pk.Size >= 1 {
				out[h526Cle{c, pk.Index}] = pk.Payload(data)
			}
		}
	}
	return out
}

// t526Premiers rend, par eid rejete, l index de trame de son PREMIER rejet (dans l ordre des
// trames), et l ensemble des eid rejetes.
func t526Premiers(p *t525Passe) (map[uint32]int, map[uint32]bool) {
	premiers, rejetes := map[uint32]int{}, map[uint32]bool{}
	for i, tr := range p.tr {
		if !tr.rejet {
			continue
		}
		rejetes[tr.rejetID] = true
		if _, vu := premiers[tr.rejetID]; !vu {
			premiers[tr.rejetID] = i
		}
	}
	return premiers, rejetes
}

// t526TableauA confronte les trois references a l eid rejete.
func t526TableauA(t *testing.T, p *t525Passe, pls map[h526Cle][]byte, cibles []int,
	cfg FrameConfig) {
	t.Helper()
	var present [3]int
	var exact, brut [3]int
	for _, i := range cibles {
		tr := p.tr[i]
		pay := pls[h526Cle{tr.chunk, tr.paquet}]
		refs := t526LireRefs(pay)
		for k, g := range []guardedRef{refs.r0, refs.r1, refs.r2} {
			if g.Present {
				present[k]++
			}
			e, b := t526Egal(g, tr.rejetID, cfg.IDBase)
			if e {
				exact[k]++
			}
			if b {
				brut[k]++
			}
		}
	}
	t.Logf("(a) LES TROIS REFERENCES DU RECORD 36, contre l eid rejete du MEME paquet (%d cibles) :",
		len(cibles))
	for k, dom := range []int{1, 8, 7} {
		t.Logf("      ref%d (domaine %d) : presente %3d · = eid a l identique %3d · = (index, tete) "+
			"bruts %3d", k, dom, present[k], exact[k], brut[k])
	}
}

// t526TableauB balaye bit a bit l etendue localisee du record, pour l eid rejete et pour un
// temoin (l eid rejete de la cible suivante).
func t526TableauB(t *testing.T, p *t525Passe, pls map[h526Cle][]byte, cibles []int,
	cfg FrameConfig) {
	t.Helper()
	w := cfg.IDLowBits
	var modaux, trouve, trouveTemoin, trouve32, bitsBalayes int
	for n, i := range cibles {
		tr := p.tr[i]
		pay := pls[h526Cle{tr.chunk, tr.paquet}]
		fin := t526LireRefs(pay).r2.EndBit
		if pc, ok := m526PostComptes(pay); ok {
			modaux++
			fin = pc + modalAimGap + int(FireAimBits)
		}
		if fin > len(pay)*8 {
			fin = len(pay) * 8
		}
		bitsBalayes += fin - eventPayloadStartBit
		temoin := p.tr[cibles[(n+1)%len(cibles)]].rejetID
		if t526Balayer(pay, eventPayloadStartBit, fin, tr.rejetID-cfg.IDBase, w) {
			trouve++
		}
		if temoin != tr.rejetID && t526Balayer(pay, eventPayloadStartBit, fin, temoin-cfg.IDBase, w) {
			trouveTemoin++
		}
		if t526Balayer32(pay, eventPayloadStartBit, fin, tr.rejetID) {
			trouve32++
		}
	}
	t.Logf("(b) BALAYAGE BIT A BIT de l etendue localisee (%d cibles, dont %d modales, %d bits) :",
		len(cibles), modaux, bitsBalayes)
	t.Logf("      (index %d bits, tete 2) de l eid rejete trouve %d · TEMOIN (eid d une autre "+
		"cible) %d · mot de 32 bits exact %d", w, trouve, trouveTemoin, trouve32)
}

// t526Balayer cherche (index sur w bits, tete sur 2) a toute position de [de, a).
func t526Balayer(pay []byte, de, a int, eid uint32, w int) bool {
	motif := (eid&0x3fffffff)<<2 | eid>>30
	masque := uint32(1)<<(w+2) - 1
	for pos := de; pos+w+2 <= a; pos++ {
		if readBitsAt(pay, pos, w+2) == motif&masque {
			return true
		}
	}
	return false
}

// t526Balayer32 cherche le mot de 32 bits de l eid a toute position de [de, a).
func t526Balayer32(pay []byte, de, a int, eid uint32) bool {
	for pos := de; pos+32 <= a; pos++ {
		if readBitsAt(pay, pos, 32) == eid {
			return true
		}
	}
	return false
}

// t526TableauC dit ce que ref2 (domaine 7) designe sur TOUS les paquets de tete 36 du film.
func t526TableauC(t *testing.T, p *t525Passe, pls map[h526Cle][]byte, rejetes map[uint32]bool,
	cfg FrameConfig) {
	t.Helper()
	var paquets, presente, declaree, rejetee, ailleurs, r0, r0Sonde, r1 int
	parTI := map[uint32]int{}
	for _, tr := range p.tr {
		if tr.tete != t526TeteTir {
			continue
		}
		paquets++
		refs := t526LireRefs(pls[h526Cle{tr.chunk, tr.paquet}])
		r0, r0Sonde, r1 = r0+t526Un(refs.r0.Present), r0Sonde+refs.r0.Sonde, r1+t526Un(refs.r1.Present)
		g := refs.r2
		if !g.Present {
			continue
		}
		presente++
		eid := t526Eid(g, cfg.IDBase)
		switch ti, _, ok := p.tab.ArchetypeApres(eid, -1); {
		case ok:
			declaree++
			parTI[ti]++
		case rejetes[eid]:
			rejetee++
		default:
			ailleurs++
		}
	}
	t.Logf("(c) ref2 (domaine 7) sur TOUS les paquets de tete 36 : %d paquets · presente %d · "+
		"declaree par une image-cle %d · eid REJETE %d · ni l un ni l autre %d",
		paquets, presente, declaree, rejetee, ailleurs)
	t.Logf("      ref0 (domaine 1) presente %d, dont sonde=1 (index sur 9 bits) %d · ref1 (domaine 8) "+
		"presente %d", r0, r0Sonde, r1)
	tis := make([]int, 0, len(parTI))
	for ti := range parTI {
		tis = append(tis, int(ti))
	}
	sort.Slice(tis, func(a, b int) bool { return parTI[uint32(tis[a])] > parTI[uint32(tis[b])] }) //nolint:gosec // archetype
	for _, ti := range tis {
		t.Logf("      archetype declare ti=%-3d %6d", ti, parTI[uint32(ti)]) //nolint:gosec // archetype
	}
}

// t526Un rend 1 pour vrai, 0 pour faux.
func t526Un(b bool) int {
	if b {
		return 1
	}
	return 0
}

// m526PostComptes rend la position post-comptes d un record de tir modal, par la grammaire de
// production (tete [lireEnteteTir36], lot M4b).
func m526PostComptes(pay []byte) (int, bool) {
	h, ok := lireEnteteTir36(pay)
	if !ok {
		return 0, false
	}
	return modalPostCountsBitFrom(pay, h)
}
