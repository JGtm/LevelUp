//go:build research

package grammar

// mouvement_5_11_6_cadre_research_test.go — LE CADRAGE DES PAQUETS ET LA FIN DE LA MARCHE
// (lot 5.11.6).
//
// # CE QUE LE POINT 1 A TROUVE, ET QUI CHANGE LA QUESTION
//
// La « queue de 6,3 bits non lus » du lot 5.11 etait un artefact de l instrument : il ne sommait
// que les restes POSITIFS. La mesure juste dit le contraire — sur `dad793c7` la marche consomme
// **57 bits de PLUS** que le paquet n en porte, sur 95,7 % des paquets. Le decodeur lit donc au
// DELA de la fin du payload, et ce qu il lit la-bas n existe pas.
//
// Ce fichier repond a deux questions, dans l ordre :
//
//	LE CADRAGE EST-IL JUSTE ? Les paquets d un chunk sont-ils contigus, ou reste-t-il des octets
//	  entre la fin d un paquet et le debut du suivant — c est-a-dire une zone que
//	  [FilmPacket.Payload] exclut et que personne ne lit ?
//	QUEL RECORD DEBORDE ? Pour chaque paquet, la liste des records avec leur intervalle de bits,
//	  et lequel franchit la fin du payload. Un record qui deborde est un record FABRIQUE a partir
//	  de zeros : tout ce qu il declare est faux, et tout ce qui le suivait est perdu.

import (
	"fmt"
	"sort"
	"strings"
	"testing"
)

// TestMouvement5116Cadre — LE CADRAGE : contiguite des paquets dans le chunk.
func TestMouvement5116Cadre(t *testing.T) {
	film := m511Film(t)
	fc := NewFilmContext(film)
	for _, c := range fc.ChunkNumbers() {
		data, pks, ok := fc.ChunkAt(c)
		if !ok {
			continue
		}
		ecarts := map[int]int{}
		var fin int
		for i, pk := range pks {
			if i > 0 {
				ecarts[pk.Start-fin]++
			}
			fin = pk.Start + pk.Size
		}
		reste := len(data) - fin
		cles := make([]int, 0, len(ecarts))
		for k := range ecarts {
			cles = append(cles, k)
		}
		sort.Ints(cles)
		var parts []string
		for _, k := range cles {
			parts = append(parts, fmt.Sprintf("%d o : %d", k, ecarts[k]))
		}
		t.Logf("chunk %d : %d octets, %d paquets · ECART entre fin d un paquet et debut du "+
			"suivant : %s · reste en fin de chunk : %d o",
			c, len(data), len(pks), strings.Join(parts, " · "), reste)
	}
}

// TestMouvement5116Records — QUEL RECORD DEBORDE, et de combien.
func TestMouvement5116Records(t *testing.T) {
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
	a, b := m511Bornes()
	cfg := fc.CadreDeBalayage()
	w := NewWorld(reg)
	debordeParTI := map[uint32]int{}
	totalParTI := map[uint32]int{}
	var t0 uint64
	var montres int
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
			if t0 == 0 {
				t0 = pk.TimestampUS
			}
			rel := float64(pk.TimestampUS-t0) / 1e6
			pay := pk.Payload(data)
			debut := 2
			if _, present := PacketHeadEventType(pay); present {
				if debut = marchLocateStrict(pay, w, cfg); debut < 0 {
					continue
				}
			}
			recs, _ := DecodeFrameViews(pay, w, cfg, 3, debut)
			bits := len(pay) * 8
			dansFenetre := rel >= a && rel <= b
			if dansFenetre && montres < 6 {
				montres++
				t.Logf("t=%8.3f s · payload %d bits · %d records :", rel, bits, len(recs))
			}
			for i, r := range recs {
				totalParTI[r.TypeIndex]++
				deb := r.Trace.EndBit
				if len(r.Trace.Comps) > 0 {
					deb = r.Trace.Comps[0].StartBit
				}
				if r.Trace.EndBit > bits {
					debordeParTI[r.TypeIndex]++
				}
				if dansFenetre && montres <= 6 && montres > 0 && i < 8 {
					etat := "dans le paquet"
					if r.Trace.EndBit > bits {
						etat = fmt.Sprintf("DEBORDE de %d bits", r.Trace.EndBit-bits)
					}
					t.Logf("    ti=%2d slot %5d type %d · %2d comps · premier comp @%4d · "+
						"fin %4d · %s", r.TypeIndex, r.Slot, r.Type, len(r.Trace.Comps), deb,
						r.Trace.EndBit, etat)
				}
			}
		}
	}
	tis := make([]int, 0, len(totalParTI))
	for ti := range totalParTI {
		tis = append(tis, int(ti))
	}
	sort.Ints(tis)
	t.Logf("RECORDS QUI DEBORDENT LA FIN DU PAYLOAD, par archetype :")
	for _, ti := range tis {
		k := uint32(ti) //nolint:gosec // index de registre
		t.Logf("  ti=%2d : %6d records, dont %6d debordent (%.1f %%)", ti, totalParTI[k],
			debordeParTI[k], m533bPart(debordeParTI[k], totalParTI[k]))
	}
}
