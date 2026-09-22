//go:build research

package grammar

// mouvement_5_16_baseline_research_test.go — LA BASELINE DU CORPS DE DELTA (lot 5.16.3, D2 du
// 5.15).
//
// # CE QUE L ECRIVAIN DIT, ET LA REPONSE EST NETTE : UNE VALEUR, PAS UNE LARGEUR
//
// `FUN_1406cbaa0` (branche vive, type 3) lit le SELECTEUR par `FUN_1406cdc04` — `R(1)`, et `R(7)`
// de plus si le premier bit est leve, sentinelle `0xff` — puis resout l entree d historique
// `iVar20 = (*ctx - selecteur) - 1` par `FUN_141fda280` et la passe a `FUN_1406caad8`, qui la
// place dans le bloc d arguments de l iterateur `FUN_14076cb60`. L ecrivain correspondant est
// `FUN_140769e08(writer, index)` : un bit, plus sept si l index n est pas `0xff`.
//
// ET DANS `FUN_14076cb60` LA BASELINE NE TOUCHE AUCUNE LARGEUR :
//
//	FUN_1406d7610(descripteur, lecteur, &masque)      le MASQUE, avant toute baseline
//	pour chaque composant i du descripteur :
//	   si (masque >> ((i - decales) & 0xff)) & 1 :
//	      niveau = vtable[0x00](deser)                 la PRECISION, du descripteur
//	      si args[4] == 0 : prediction = 0             <- PAS de baseline : reference NULLE
//	      sinon           : prediction = vtable[0x48](deser, tampon, &args[3])
//	      vtable[0x28](deser, lecteur, args, &prediction, niveau)   <- la LECTURE
//
// La baseline n entre donc que par `&prediction`, un vecteur de 16 octets que le deserialiseur
// APPLIQUE ; la largeur, elle, vient de `niveau` (le descripteur) et des bits. **La baseline
// change une VALEUR, pas une largeur** : `decodeDelta` qui consomme le selecteur et jette la
// baseline ne DERIVE donc jamais — mais les valeurs qu il publie sur un record a selecteur
// OUVERT sont relatives a une reference NULLE au lieu de l entree d historique designee.
//
// DECOUVERTE HORS PERIMETRE, CONSIGNEE : le bit de masque teste est `i - decales`, ou `decales`
// compte les composants que `FUN_1428e1dac(&DAT_144c23178, typeIndex, nom)` ECARTE quand
// `*(TLS + 0x238)` porte un nom de contexte non vide. Le depot teste le bit `i` BRUT
// (`traverseComponentLoop`). Voir le §4 du lot 5.16.
//
// Rejouable :
//
//	MOUV511_FILM=<dir> MOUV511_BORNES=<catalogue> MOUV511_CARTE=<carte> \
//	  go test -tags=research -count=1 -v -timeout 60m \
//	  -run '^TestBaseline516Selecteur$' \
//	  ./internal/games/halo_infinite/film/internal/grammar/

import "testing"

// TestBaseline516Selecteur MESURE le taux d OUVERTURE du selecteur de baseline sur le film
// courant — la mesure que D2 du 5.15 reclamait pour un film DENSE (sur `000d5950` il etait ferme
// 54 760 fois sur 54 760).
//
// La position du selecteur se retrouve depuis le premier composant du record : le corps est
// `[selecteur][masque][composants]`, et la largeur du masque se deduit de sa valeur
// (`consumeMask` : `1 + 64` en forme longue, `1 + 3 + 6 * n` sinon).
func TestBaseline516Selecteur(t *testing.T) {
	tc := t516Cadre(t)
	w := NewWorld(tc.reg)
	var records, mesurables, ferme, ouvert, indetermine int
	for _, c := range tc.fc.ChunkNumbers() {
		data, pks, ok := tc.fc.ChunkAt(c)
		if !ok {
			continue
		}
		m533bLierMonde(w, data, pks)
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta || pk.Size < 1 {
				continue
			}
			pay := pk.Payload(data)
			debut := DefaultPacketPreambleBits
			if _, present := PacketHeadEventType(pay); present {
				if debut = marchLocateStrict(pay, w, tc.cfg); debut < 0 {
					continue
				}
			}
			recs, _, _ := t515Records(pay, w, tc.cfg, debut)
			for _, r := range recs {
				if r.Type != recDelta {
					continue
				}
				records++
				if len(r.Trace.Comps) == 0 || r.DesyncAt != -1 {
					continue
				}
				debutCorps := r.Trace.Comps[0].StartBit - b516LargeurMasque(r.Trace.Mask)
				if debutCorps < 8 {
					continue
				}
				mesurables++
				switch {
				case t515LireBits(pay, debutCorps-1, 1) == 0:
					ferme++
				case t515LireBits(pay, debutCorps-8, 1) == 1:
					ouvert++
				default:
					indetermine++
				}
			}
		}
	}
	t.Logf("SELECTEUR DE BASELINE : %d records DELTA · %d mesurables · FERME %d · OUVERT %d · "+
		"indetermine %d", records, mesurables, ferme, ouvert, indetermine)
}

// b516LargeurMasque rend la largeur EN BITS du masque de presence qui porte `masque`, sous la
// grammaire de `consumeMask` : forme longue `1 + 64`, forme courte `1 + 3 + 6 * n`.
//
// La forme courte est celle que l ecrivain emprunte des que le nombre d indices tient sur trois
// bits (au plus sept) ; au-dela, seule la forme longue peut porter le masque.
func b516LargeurMasque(masque uint64) int {
	n := 0
	for m := masque; m != 0; m &= m - 1 {
		n++
	}
	if n > 7 {
		return 1 + 64
	}
	return 1 + 3 + 6*n
}
