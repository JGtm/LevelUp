package filmdec

// zoom_events_test.go — QUELLE PART DES EVENEMENTS DE LUNETTE LE SCANNER RATE-T-IL ?
//
// Le scanner de production (`ScanFilmZoomEvents`) ne lit que l'evenement de TETE de chaque
// paquet. Une liste peut en porter plusieurs : un dezoom PROVOQUE (degat recu, changement
// d'arme) voyagerait dans le paquet de sa cause, donc en 2e position, et echapperait a la
// lecture. C'est l'explication designee pour les deux tiers des entrees orphelines.
//
// CE QUE CET INSTRUMENT MESURE, ET IL N'A PAS BESOIN DE LA GRAMMAIRE DE TOUS LES TYPES : le bit
// de CONTINUATION qui suit chaque evenement est lisible sans savoir decoder les charges des
// autres types, DES LORS que l'evenement de tete est un `unit_zoom` (charge R(2), longueur
// connue). On mesure donc, sur la seule famille qu'on sait traverser, la frequence des listes
// MULTIPLES — et le type du deuxieme evenement quand il y en a un.
//
// C'est un DIMENSIONNEMENT, pas un verdict : il dit si le chantier « marcher la liste entiere »
// vaut la peine, et ce qu'il faudra savoir decoder pour y arriver.
//
// Garde ZOOM_EVT_FILM (un film ; ne jamais balayer le corpus — bombe RAM connue du depot).

import (
	"os"
	"sort"
	"testing"
)

func TestZoomListesMultiples(t *testing.T) {
	dir := os.Getenv("ZOOM_EVT_FILM")
	if dir == "" {
		t.Skipf("ZOOM_EVT_FILM absent : mesure sautee")
	}
	paquets, avecSuite := 0, 0
	seconds := map[int]int{}
	for c := 1; c <= CountFilmChunks(dir); c++ {
		chunk, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(chunk) {
			if pk.Type != PacketTypeDelta || pk.Size < 2 {
				continue
			}
			pay := pk.Payload(chunk)
			if pay[0] != zoomFamilyByte {
				continue
			}
			br := NewBitReader(pay)
			br.Skip(1)
			if !br.ReadBit() {
				continue
			}
			if int(br.ReadBits(7)) != zoomEventType {
				continue
			}
			paquets++
			readZoomRef(br, zoomRefDomains[0])
			readZoomRef(br, zoomRefDomains[1])
			readZoomRef(br, zoomRefDomains[2])
			br.Skip(2) // la charge : R(2)
			if !br.ReadBit() {
				continue // fin de liste : l'evenement de tete etait seul
			}
			avecSuite++
			seconds[int(br.ReadBits(7))]++
		}
	}
	if paquets == 0 {
		t.Fatalf("aucun paquet de lunette dans %s", dir)
	}
	t.Logf("LISTES — %d paquets a evenement de lunette en tete ; %d portent un SECOND evenement"+
		" (%.1f %%)", paquets, avecSuite, 100*float64(avecSuite)/float64(paquets))
	type tc struct{ t, n int }
	var l []tc
	for ty, n := range seconds {
		l = append(l, tc{ty, n})
	}
	sort.Slice(l, func(i, j int) bool { return l[i].n > l[j].n })
	for i, e := range l {
		if i == 10 {
			break
		}
		t.Logf("  2e evenement de type %3d : %d fois", e.t, e.n)
	}
	t.Log("LECTURE : une part elevee de listes multiples dit que le scanner de tete est aveugle" +
		" a une fraction du flux, et le palmares des seconds types dit CE QU'IL FAUDRA savoir" +
		" decoder pour marcher la liste entiere.")
}
