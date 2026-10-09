//go:build research

package grammar

// mouvement_5_11_partiels_research_test.go — LES QUATRE COMPOSANTS PARTIELS DU BIPEDE, MESURES
// SUR LES RECORDS RETENUS (lot 5.11.0).
//
// # CE QU IL MESURE, ET POURQUOI IL RELIT AU `StartBit`
//
// La porte de publication ne dit rien de `i57`, `i59`, `i60` et `i63` : trois d entre eux n ont
// pas de hook, et le quatrieme (`i57`) n en publie qu une valeur. Cet instrument garde les
// records RENDUS par [DecodeFrameViews] et RELIT le composant a son `StartBit`, celui que la
// boucle de composants a consigne dans `Trace.Comps` — meme idiome que
// `mouvement_5_7_retenus_research_test.go` (lot 5.7), et pour la meme raison : la porte ne
// distingue pas un essai d alignement d un record retenu.
//
// POUR `i63` il rend LE MASQUE DE TETE et le compte du second tour qu il implique
// ([bipedActionLoop2Count]) : c est la mesure de la correction du 5.11.0, et elle dit combien de
// records le port SOUS-LISAIT.
//
// Rejouable :
//
//	MOUV511_FILM=<dir> MOUV511_CARTE=<carte> MOUV511_BORNES=<catalogue> \
//	  go test -tags=research -count=1 -v -timeout 60m \
//	  -run '^TestMouvement511Partiels$' ./internal/games/halo_infinite/film/internal/grammar/

import (
	"fmt"
	"sort"
	"strings"
	"testing"
)

// m511PartielsCibles : les quatre composants PARTIELS de l archetype bipede, par leur NOM de
// registre (l index n est pas un invariant — lecon du lot 5.7, piege 5).
var m511PartielsCibles = map[string]string{
	"biped-spartan-ability-component":           "i57",
	"biped-spartan-ability-non-predicted-state": "i59",
	"simulation-state-component":                "i60",
	"biped-action-component":                    "i63",
	"biped-action":                              "i63",
}

func TestMouvement511Partiels(t *testing.T) {
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
	var ti35, desync int
	etalon := map[int]int{}
	parComp := map[string]int{}
	nonPorte := map[string]int{}
	fautif := map[string]int{}
	comptes := map[int]int{}
	parType, nulParType := map[int]int{}, map[int]int{}
	var lignes []string
	var paquets, resteTotal, ferme8, ferme32 int
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
			recs, _ := DecodeFrameViews(pay, w, cfg, 3, debut)
			paquets++
			if len(recs) > 0 {
				reste := len(pay)*8 - recs[len(recs)-1].Trace.EndBit
				if reste < 0 {
					reste = -reste
				}
				resteTotal += reste
				if reste <= 8 {
					ferme8++
				}
				if reste <= 32 {
					ferme32++
				}
			}
			for _, r := range recs {
				if r.TypeIndex != BipedTypeIndex {
					continue
				}
				ti35++
				if r.DesyncAt >= 0 {
					desync++
				}
				for _, b := range []int{0, 1, 21, 25} {
					if r.Trace.Mask&(1<<uint(b)) != 0 {
						etalon[b]++
					}
				}
				for _, cp := range r.Trace.Comps {
					nom, vise := m511PartielsCibles[cp.Name]
					if !vise {
						continue
					}
					parComp[nom]++
					if !cp.Ported {
						nonPorte[nom]++
					}
					if r.DesyncAt == cp.Index {
						fautif[nom]++
					}
					if nom != "i63" {
						continue
					}
					mots, n := m511MasqueI63(pay, cp.StartBit)
					comptes[n]++
					parType[r.Type]++
					if n == 0 {
						nulParType[r.Type]++
					}
					if n > 0 && len(lignes) < 20 {
						lignes = append(lignes, fmt.Sprintf(
							"slot %5d @bit %7d masque %08x %08x %08x -> count2 %d",
							r.Slot, cp.StartBit, mots[0], mots[1], mots[2], n))
					}
				}
			}
		}
	}
	t.Logf("ORACLE DE CONTENU : %d records ti=35 (%d desynchronises) · i0 %.1f %% · i1 %.1f %% · "+
		"i21 %.1f %% · i25 %.1f %%", ti35, desync, m533bPart(etalon[0], ti35),
		m533bPart(etalon[1], ti35), m533bPart(etalon[21], ti35), m533bPart(etalon[25], ti35))
	t.Logf("  FERMETURE DE PAQUET : %d paquets · %d bits non lus au total (%.2f par paquet) · "+
		"%d fermes a 8 bits pres (%.2f %%) · %d a 32 bits pres (%.2f %%)",
		paquets, resteTotal, float64(resteTotal)/float64(max(paquets, 1)),
		ferme8, m533bPart(ferme8, paquets), ferme32, m533bPart(ferme32, paquets))
	for _, nom := range []string{"i57", "i59", "i60", "i63"} {
		t.Logf("  %-4s : %5d declarations · %5d rendues NON PORTEES · %5d fois composant fautif",
			nom, parComp[nom], nonPorte[nom], fautif[nom])
	}
	cles := make([]int, 0, len(comptes))
	for n := range comptes {
		cles = append(cles, n)
	}
	sort.Ints(cles)
	var parts []string
	for _, n := range cles {
		parts = append(parts, fmt.Sprintf("count2=%d : %d", n, comptes[n]))
	}
	t.Logf("  i63 — COMPTE DU SECOND TOUR, tire du masque de tete : %s",
		strings.Join(parts, " · "))
	tps := make([]int, 0, len(parType))
	for ty := range parType {
		tps = append(tps, ty)
	}
	sort.Ints(tps)
	for _, ty := range tps {
		t.Logf("      type de record %d : %d declarations d i63, dont %d a masque NUL (%.1f %%)",
			ty, parType[ty], nulParType[ty], m533bPart(nulParType[ty], parType[ty]))
	}
	for _, l := range lignes {
		t.Logf("      %s", l)
	}
}

// m511MasqueI63 relit le bloc de tete d `i63` a son `StartBit` et rend ses trois mots ainsi que
// le compte du second tour qu ils impliquent. AUCUN bit n est consomme du parcours principal :
// c est un lecteur neuf sur le meme payload.
func m511MasqueI63(pay []byte, depart int) ([3]uint64, int) {
	br := LecteurSur(pay)
	br.Skip(depart)
	mots := consumeBipedActionSubBlock(br)
	return mots, bipedActionLoop2Count(mots)
}
