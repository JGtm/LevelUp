//go:build research

package grammar

// vues_5_13_espace_de_noms_research_test.go — L ESPACE DE NOMS DE LA VUE, CONFRONTE ENTRE
// L IMAGE-CLE ET LE FLUX DELTA (lot 5.13.1).
//
// # LA QUESTION, ET POURQUOI ELLE SE POSE EN DEUX ENDROITS
//
// L ecrivain de la liste de REFERENCE d une vue (`FUN_142f2e174`, vtable de vue `0x1436a87e0`
// slot `+0x10`) met les DEUX BITS DE TETE de chaque identifiant a `vue + 8`, et `vue + 8` est le
// RANG de la vue — le registraire `FUN_1409c9860` l y ecrit (`*(int *)(param_3 + 1) = param_2`).
// `TestImageCle513Vues` le lit dans les films : UN SEUL rang par image-cle, et il vaut 1 sur tout
// `dad793c7` et tout `bfecd02b`.
//
// Le flux DELTA, lui, ecrit l identifiant que la table de la vue porte (`FUN_142f30610` :
// `uVar3 = *(uint *)(slot * 0xa0 + 8 + vue[0x38])`, passe a `FUN_142f2c754`), et `readRecordID`
// le relit en `[idLow][tag 2]`. Cet identifiant est pose par `FUN_1408f1730` a
// `*(byte *)(datum + 1) << 0x1e | slot` : un champ du DATUM, par entite — PAS le rang de la vue.
// Cet instrument mesure l ecart entre les deux, par RANG DE VUE de la marche.
//
// # LES TROIS VUES SONT TROIS CLASSES, ET LEURS BOUCLES DE RECORDS SONT TROIS GRAMMAIRES
//
// `FUN_142987460` appelle `vtable[0x40]` sur ses trois vues. Les trois vtables lues (`/read_memory`)
// n ont PAS la meme fonction a ce slot :
//
//	0x1436a8700 +0x40 = FUN_14076a1c4   si `vue[0x11]` -> ZERO bit ; sinon boucle
//	                                    `R(1)` (0 = fin) puis UN corps (`FUN_14080a9d4`) ;
//	                                    et elle rend TOUJOURS zero record (`*param_6 = 0`).
//	0x1436a87e0 +0x40 = FUN_1406cd128   la boucle du GESTIONNAIRE D ENTITES : `[R(32) film]`,
//	                                    `prefixe R(1)`, `si 0 -> type R(2)`, `idLow + tag 2`,
//	                                    corps selon le type. LA SEULE que la marche porte.
//	0x1436a8770 +0x40 = FUN_1406cf548   `[prologue FUN_142f2539c si drapeau]` puis boucle
//	                                    `R(1)` (0 = fin), `kind R(2)`, trois handlers
//	                                    (`FUN_1406d0388` / `FUN_142f29b38` / `FUN_142f29e54`),
//	                                    `kind == 3` = zero bit.
//
// LA CONSEQUENCE, ET C EST LA REPONSE A « NOMMER L ENTITE 7140 GENERATION 2 » DU LOT 5.11.7 :
// les deux « en-tetes de record rejetes » du pied de trame ne sont pas des en-tetes du
// gestionnaire d entites — ce sont les flux des DEUX AUTRES classes de vue, lus avec la grammaire
// du gestionnaire. `slot 7136 tag 0` et `slot 7140 tag 2` ne designent donc AUCUNE entite : ce
// sont `R(1)` + `kind R(2)` + du corps d une autre grammaire, decoupes en `[prefixe][idLow][tag]`.
// Les bits 27 et 30 qui basculaient au decollage sont des bits DE CE CORPS.
//
// Controle sur le film : sur `dad793c7` les vues de rang 1 et 2 ne rendent que 13 records en tout,
// TOUS de type DEL (slots 0, 260, 261, 262) — jamais un NEW, donc jamais un archetype ; et le slot
// 7140 n apparait dans aucun record d image-cle (les images-cles de ce film s arretent au slot
// 1 345). Le film ne porte pas de table pour ces deux vues : leur grammaire, elle, est nommee.
//
// # CE QUE CET INSTRUMENT REND
//
// Par RANG DE VUE (0, 1, 2 — l ordre de la marche), l histogramme du tag des records RETENUS, le
// nombre de vues qui n emettent aucun record, et la description de tout record des rangs 1 et 2.
// Aucune inference : des histogrammes et leurs denominateurs.
//
// Rejouable :
//
//	MOUV511_FILM=<dir> MOUV511_CARTE=<carte> MOUV511_BORNES=<catalogue> \
//	  go test -tags=research -count=1 -v -run '^TestVues513EspaceDeNoms$' \
//	  ./internal/games/halo_infinite/film/internal/grammar/

import (
	"fmt"
	"sort"
	"strings"
	"testing"
)

func TestVues513EspaceDeNoms(t *testing.T) {
	film := m511Film(t)
	entry := m511Entree(t)
	fc := m511Contexte(film, entry)
	reg, err := fc.Registry()
	if err != nil {
		t.Fatalf("registre : %v", err)
	}
	if restore, errMPP := InstallFilmFormatMPP(fc); errMPP == nil {
		defer restore()
	}
	cfg := fc.CadreDeBalayage()
	w := NewWorld(reg)
	// tags[rang de vue][tag] = nombre de records ; vides[rang] = vues sans record.
	tags := [3]map[uint32]int{{}, {}, {}}
	vides := [3]int{}
	marchees := [3]int{}
	var paquets int
	// pistes : tout record emis hors du rang 0, decrit — c est la SEULE fenetre du film sur les
	// entites des vues de rang 1 et 2, celles que l image-cle n enumere pas.
	pistes := map[string]int{}
	for _, c := range fc.ChunkNumbers() {
		data, pks, ok := fc.ChunkAt(c)
		if !ok {
			continue
		}
		m533bLierMonde(w, data, pks)
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 1 {
				continue
			}
			pay := pk.Payload(data)
			debut := 2
			if _, present := PacketHeadEventType(pay); present {
				if debut = marchLocateStrict(pay, w, cfg); debut < 0 {
					continue
				}
			}
			paquets++
			br := LecteurSur(pay)
			br.poserCadre(cfg)
			br.Skip(debut)
			frameLen := len(pay) * 8
			for v := 0; v < 3 && br.BitPos() < frameLen-3; v++ {
				w.PoserVueCourante(v)
				start := br.BitPos()
				recs, _, hitEnd := decodeInferLoop(br, pay, w, cfg)
				marchees[v]++
				if len(recs) == 0 {
					vides[v]++
				}
				for _, r := range recs {
					tags[v][r.ID>>30]++
					if v > 0 {
						pistes[fmt.Sprintf("rang %d · type %d · slot %d · tag %d · i%d · desync %d",
							v, r.Type, r.Slot, r.ID>>30, r.TypeIndex, r.DesyncAt)]++
					}
				}
				if !hitEnd || br.BitPos() == start {
					break
				}
			}
		}
	}
	t.Logf("%d paquets delta marches", paquets)
	for v := 0; v < 3; v++ {
		t.Logf("  vue de rang %d : %d marches, %d sans record, tags %s",
			v, marchees[v], vides[v], v513Tags(tags[v]))
	}
	cles := make([]string, 0, len(pistes))
	for k := range pistes {
		cles = append(cles, k)
	}
	sort.Slice(cles, func(i, j int) bool {
		if pistes[cles[i]] != pistes[cles[j]] {
			return pistes[cles[i]] > pistes[cles[j]]
		}
		return cles[i] < cles[j]
	})
	t.Logf("RECORDS DES VUES DE RANG 1 ET 2 (%d formes distinctes) :", len(cles))
	for i, k := range cles {
		if i >= 40 {
			t.Logf("   ... %d formes de plus", len(cles)-i)
			break
		}
		t.Logf("   %s x%d", k, pistes[k])
	}
}

// v513Tags rend l histogramme des tags, tag croissant.
func v513Tags(m map[uint32]int) string {
	cles := make([]int, 0, len(m))
	for k := range m {
		cles = append(cles, int(k))
	}
	sort.Ints(cles)
	var b strings.Builder
	for _, k := range cles {
		fmt.Fprintf(&b, "tag %d : %d · ", k, m[uint32(k)]) //nolint:gosec // k vient d une cle uint32
	}
	return strings.TrimSuffix(b.String(), " · ")
}
