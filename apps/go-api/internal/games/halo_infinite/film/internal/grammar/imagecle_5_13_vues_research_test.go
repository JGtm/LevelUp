//go:build research

package grammar

// imagecle_5_13_vues_research_test.go — LES DEUX BITS DE TETE D UN IDENTIFIANT D IMAGE-CLE SONT
// LA VUE, ET L ECRIVAIN LE DIT (lot 5.13.1).
//
// # CE QUE L ECRIVAIN DIT, ET CE QUE LE DEPOT EN AVAIT FAIT
//
// L encodeur de la liste de REFERENCE d une vue de replication est `FUN_142f2e174`
// (`replication_entity_manager_view.cpp`), slot `+0x10` de la vtable de vue `0x1436a87e0`. Il
// parcourt la table d entites DE SA VUE (`vue+0x38` .. `vue+0x40`, pas de 0xa0, bitmap de
// presence `vue+0x58`) et ecrit, par entite retenue, UN mot de 32 bits :
//
//	*mot = *(int *)(vue + 8) << 0x1e | *mot & <masque> | slot & 0x1fff | <genre>
//	FUN_140bbd808(mot, priorite)   // *mot = *mot & 0xff801fff | (priorite & 0x3ff) << 0xd
//
// soit `[rang:2 @30][genre:2 @23][priorite:10 @13][slot:13 @0]`.
//
// LE MAILLON DU LOT : les deux bits de tete viennent de `vue + 8`, PAS de l entite. Verifie sur
// l instruction, pas sur le pseudo-code : `142f2e2ec MOV ECX, dword ptr [RDI + 0x8]` puis
// `142f2e304 SHL ECX, 0x1e` — `RDI` est `param_1`, c est-a-dire LA VUE (meme registre que
// `vue+0x38`, `vue+0x58`, `vue+0x14`). Les trois sites de genre (`142f2e304`, `142f2e38a`,
// `142f2e440`) lisent tous le meme champ.
//
// ET `vue + 8` EST LE RANG DE LA VUE : le registraire `FUN_1409c9860(conteneur, rang, vue)`
// l ecrit, `*(int *)(param_3 + 1) = param_2`, au moment ou il range la vue dans le tableau que
// `FUN_142987460` parcourt (`FUN_141f855b4` l appelle pour les rangs 0, 1 et 2). Le champ est
// donc constant pour toutes les entites d une vue, et le journal RE du lot G (2026-08-27) le
// nommait `gen` sans avoir decompile l instruction.
//
// CE QUE CELA CHANGE POUR LE DEPOT. `keyframe_world.go` lit ce champ (`gen = id >> 30`) et
// rejette `gen == 0` ; les deux binders d image-cle le JETAIENT (`BindWildcard`, qui posait
// `Vue = 0` d office). [World.BindImageCle] le lit.
//
// # CE QUE CET INSTRUMENT MESURE
//
// Il ne cherche RIEN : il rend, par paquet d image-cle, la ventilation des records par `ns`, et
// pour chaque `ns` la plage de slots et les archetypes. Trois lectures possibles du resultat, et
// elles se distinguent sur pieces :
//
//	(a) un seul rang par paquet d image-cle -> le film n enregistre QU UNE vue a l image-cle ;
//	(b) plusieurs rangs, en SECTIONS contigues -> les trois listes sont concatenees ;
//	(c) plusieurs rangs melanges slot par slot -> le champ n est pas un rang de vue.
//
// MESURE DU LOT : (a), sur les deux films temoins — un seul rang, et il vaut 1.
//
// Rejouable : `MOUV511_FILM=<dir du film> go test -tags=research -run TestImageCle513Vues`.

import (
	"fmt"
	"sort"
	"strings"
	"testing"
)

// ic513Ventilation : ce qu un `ns` porte dans UN paquet d image-cle.
type ic513Ventilation struct {
	ns       int
	n        int
	slotMin  int
	slotMax  int
	premier  int // rang du premier record de ce ns dans l ordre de marche
	dernier  int // rang du dernier
	archetyp map[int]int
}

// TestImageCle513Vues ventile les records d image-cle par les deux bits de tete de leur
// identifiant — le champ que l ecrivain prend dans `vue + 8`.
func TestImageCle513Vues(t *testing.T) {
	film := m511Film(t)
	var paquets int
	for c := 0; c < film.NumChunks(); c++ {
		for _, pk := range film.Packets(c) {
			if pk.Type != int(PacketTypeKeyframe) {
				continue
			}
			paquets++
			recs := WalkKeyframeWorld(pk.Payload)
			par := map[int]*ic513Ventilation{}
			for i, r := range recs {
				v, ok := par[r.Gen]
				if !ok {
					v = &ic513Ventilation{ns: r.Gen, slotMin: 1 << 30, slotMax: -1, premier: i, archetyp: map[int]int{}}
					par[r.Gen] = v
				}
				v.n++
				v.dernier = i
				if r.Slot < v.slotMin {
					v.slotMin = r.Slot
				}
				if r.Slot > v.slotMax {
					v.slotMax = r.Slot
				}
				v.archetyp[r.TI]++
			}
			ns := make([]int, 0, len(par))
			for k := range par {
				ns = append(ns, k)
			}
			sort.Ints(ns)
			t.Logf("chunk %d paquet %d : %d records, ns %v", c, pk.Index, len(recs), ns)
			for _, k := range ns {
				v := par[k]
				t.Logf("   ns=%d : %d records, slots %d..%d, rangs %d..%d, archetypes %s",
					v.ns, v.n, v.slotMin, v.slotMax, v.premier, v.dernier, ic513Archetypes(v.archetyp))
			}
		}
	}
	if paquets == 0 {
		t.Fatalf("aucun paquet d image-cle dans le film")
	}
	t.Logf("%d paquets d image-cle", paquets)
}

// ic513Archetypes rend les archetypes d un ns, les plus nombreux d abord.
func ic513Archetypes(m map[int]int) string {
	type kv struct{ ti, n int }
	l := make([]kv, 0, len(m))
	for ti, n := range m {
		l = append(l, kv{ti, n})
	}
	sort.Slice(l, func(i, j int) bool {
		if l[i].n != l[j].n {
			return l[i].n > l[j].n
		}
		return l[i].ti < l[j].ti
	})
	var b strings.Builder
	for i, e := range l {
		if i >= 8 {
			fmt.Fprintf(&b, " +%d autres", len(l)-i)
			break
		}
		fmt.Fprintf(&b, "i%d:%d ", e.ti, e.n)
	}
	return strings.TrimSpace(b.String())
}
