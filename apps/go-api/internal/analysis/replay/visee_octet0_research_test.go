package replay

// visee_octet0_research_test.go — LE BIT DE POIDS FAIBLE DU PREMIER OCTET : « variante courte »
// ou « un autre evenement suit » ?
//
// ENJEU. `fire_events.go` lit le type d'event en `payload[0] >> 1` et traite `payload[0] & 1`
// comme une VARIANTE (« record court, sans arme »), qu'il ECARTE. Le lot B2 a lu chez l'ecrivain
// un BIT DE CONTINUATION accole au type : si c'est ce bit-la, alors `payload[0] & 1 == 1` ne
// signifie pas « record court » mais « UN AUTRE EVENEMENT SUIT DANS CE PAQUET » — et tous nos
// recensements, qui ne lisent que le PREMIER evenement de chaque paquet, sont borgnes.
//
// LE TEST, DECISIF ET SANS SEUIL A DECLARER : si le record est le MEME dans les deux cas, le
// champ arme (bits 44..107, offsets figes de fire_events.go) doit avoir la MEME forme pour
// payload[0]=0xD2 et 0xD3 — memes identifiants, meme moitie haute. Si 0xD3 rend du bruit, la
// lecture « variante » est la bonne.
//
// Garde OCTET0_FILM. Aucun code de production touche.

import (
	"os"
	"sort"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

func TestViseeOctet0(t *testing.T) {
	dir := os.Getenv("OCTET0_FILM")
	if dir == "" {
		t.Skipf("OCTET0_FILM absent : instrument saute")
	}
	n := filmdec.CountFilmChunks(dir)
	parOctet := map[byte]int{}
	armes := map[byte]map[uint64]int{0xD2: {}, 0xD3: {}}
	for c := 1; c <= n; c++ {
		chunk, err := filmdec.ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, p := range filmdec.WalkPackets(chunk) {
			if p.Type != filmdec.PacketTypeDelta || p.Size < 16 {
				continue
			}
			pay := p.Payload(chunk)
			parOctet[pay[0]]++
			if pay[0] != 0xD2 && pay[0] != 0xD3 {
				continue
			}
			w := uint64(filmdec.ReadBitsAtForDiag(pay, 44, 32))<<32 |
				uint64(filmdec.ReadBitsAtForDiag(pay, 76, 32))
			armes[pay[0]][w]++
		}
	}
	var octets []byte
	for o := range parOctet {
		octets = append(octets, o)
	}
	sort.Slice(octets, func(i, j int) bool { return parOctet[octets[i]] > parOctet[octets[j]] })
	t.Log("PREMIERS OCTETS (type = o>>1, bit bas = variante ou continuation) :")
	for i, o := range octets {
		if i == 14 {
			break
		}
		t.Logf("  0x%02X (type %3d, bas %d) : %d paquets", o, o>>1, o&1, parOctet[o])
	}
	for _, o := range []byte{0xD2, 0xD3} {
		m := armes[o]
		type wc struct {
			w uint64
			n int
		}
		var l []wc
		for w, k := range m {
			l = append(l, wc{w, k})
		}
		sort.Slice(l, func(i, j int) bool { return l[i].n > l[j].n })
		t.Logf("0x%02X — %d paquets, %d identifiants d'arme distincts ; top 5 :", o, parOctet[o], len(l))
		for i, e := range l {
			if i == 5 {
				break
			}
			t.Logf("    %016x : %d", e.w, e.n)
		}
	}
	t.Log("LECTURE : memes identifiants des deux cotes = le record est IDENTIQUE, donc le bit bas" +
		" n'est pas une variante de record -> il porte la CONTINUATION, et chaque paquet peut" +
		" contenir une CHAINE d'evenements que nos recensements n'ont jamais lue.")
}
