//go:build research

package grammar

// mouvement_5_11_6_gate_research_test.go — LE GATE DU LOT 5.11.6 : BITS NON LUS PAR PAQUET.
//
// # CE QUE CE GATE MESURE, ET POURQUOI IL N EXISTAIT PAS
//
// Une trame est LUE quand le curseur du lecteur atteint la fin du payload, au bourrage d octet
// pres. Le depot ne mesurait que des RECORDS — or un decodeur peut rendre le bon nombre de
// records ET lire 57 bits au-dela de la fin du paquet, ce qui etait le cas avant la garde d eid
// de la vue (`frame_infer.go`, lot 5.11.6). Aucun compteur du depot ne le voyait.
//
// La mesure se prend au CURSEUR ([DecodeFrameViewsCurseur]) :
//
//	reste = bits du payload - curseur final
//	reste dans [0 ; 7]  -> la trame est FERMEE (bourrage d octet)
//	reste < 0           -> DEBORDEMENT : le decodeur a lu ce qui n existe pas
//	reste >= 8          -> grammaire MANQUANTE : le jeu a ecrit des bits que personne ne lit
//
// Rejouable :
//
//	MOUV511_FILM=<dir> MOUV511_CARTE=<carte> MOUV511_BORNES=<catalogue> \
//	  go test -tags=research -count=1 -v -timeout 60m \
//	  -run '^TestMouvement5116Gate$' ./internal/games/halo_infinite/film/internal/grammar/

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"
)

// m5116GateOctet est le plus grand reste acceptable : un paquet se termine sur une frontiere
// d octet, donc jusqu a sept bits de bourrage.
const m5116GateOctet = 7

func TestMouvement5116Gate(t *testing.T) {
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
	if os.Getenv("MOUV511_VUES") != "" {
		bal := fc.ProfilDeBalayage()
		bal.Grammaire.TablesParVue = true
		fc.PoserProfilDeBalayage(bal)
		cfg = fc.CadreDeBalayage()
	}
	if os.Getenv("MOUV511_CLASSES") != "" {
		// LE GATE DU LOT 5.14 : chaque rang de vue sous la grammaire de SA classe.
		bal := fc.ProfilDeBalayage()
		bal.Grammaire.ClassesDeVue = true
		fc.PoserProfilDeBalayage(bal)
		cfg = fc.CadreDeBalayage()
	}
	if os.Getenv("MOUV511_GENSTRICTE") != "" {
		// L A/B QUE LA BASCULE APPELLE DEPUIS LE LOT 2.3 : l ecrivain compare TOUJOURS l eid
		// complet (FUN_1406caad8, FUN_1406cd128), et le defaut a false etait une prudence de
		// reconstruction hors ligne, pas une lecture de l image.
		bal := fc.ProfilDeBalayage()
		bal.Grammaire.GenerationStricte = true
		fc.PoserProfilDeBalayage(bal)
		cfg = fc.CadreDeBalayage()
	}
	w := NewWorld(reg)
	var paquets, ferme, deborde, manque, nonLocalise int
	var sommeManque, sommeDeborde int
	restes := map[int]int{}
	ti35, desync := 0, 0
	etalon := map[int]int{}
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
					nonLocalise++
					continue
				}
			}
			paquets++
			recs, _, curseur := DecodeFrameViewsCurseur(pay, w, cfg, 3, debut)
			reste := len(pay)*8 - curseur
			restes[reste]++
			switch {
			case reste < 0:
				deborde++
				sommeDeborde += -reste
			case reste <= m5116GateOctet:
				ferme++
			default:
				manque++
				sommeManque += reste
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
			}
		}
	}
	t.Logf("ORACLE DE CONTENU : %d records ti=35 (%d desynchronises) · i0 %.1f %% · i1 %.1f %% · "+
		"i21 %.1f %% · i25 %.1f %%", ti35, desync, m533bPart(etalon[0], ti35),
		m533bPart(etalon[1], ti35), m533bPart(etalon[21], ti35), m533bPart(etalon[25], ti35))
	t.Logf("GATE — BITS NON LUS PAR PAQUET (curseur du lecteur) sur %d paquets marches "+
		"(%d non localises) :", paquets, nonLocalise)
	t.Logf("  FERMES (reste 0..%d, bourrage d octet) : %d (%.2f %%)", m5116GateOctet, ferme,
		m533bPart(ferme, paquets))
	t.Logf("  DEBORDEMENTS (reste < 0)               : %d (%.2f %%), %d bits lus en trop au "+
		"total", deborde, m533bPart(deborde, paquets), sommeDeborde)
	t.Logf("  GRAMMAIRE MANQUANTE (reste >= 8)       : %d (%.2f %%), %d bits ecrits et non lus "+
		"au total", manque, m533bPart(manque, paquets), sommeManque)
	cles := make([]int, 0, len(restes))
	for k := range restes {
		cles = append(cles, k)
	}
	sort.Ints(cles)
	var parts []string
	for _, k := range cles {
		if len(parts) >= 20 {
			parts = append(parts, "...")
			break
		}
		parts = append(parts, fmt.Sprintf("%d : %d", k, restes[k]))
	}
	t.Logf("  DISTRIBUTION DU RESTE : %s", strings.Join(parts, " · "))
}
