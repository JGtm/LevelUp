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
	"path/filepath"
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

// TestViseeOctet0Corpus recense les PREMIERS OCTETS sur tout le corpus. Depuis que la grammaire
// d'en-tete est etablie (bit de continuation puis R(7) du type), l'octet vaut 0x80 | type : ce
// recensement EST donc le recensement des types reels, et il tranche deux questions ouvertes —
// l'octet 0xA4 (action_weapon_fire, lecteur FUN_14080C1F8) existe-t-il ? et l'octet 0x95
// (unit_zoom) ? Garde OCTET0_CORPUS.
func TestViseeOctet0Corpus(t *testing.T) {
	root := os.Getenv("OCTET0_CORPUS")
	if root == "" {
		t.Skipf("OCTET0_CORPUS absent : recensement saute")
	}
	ents, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("corpus illisible : %v", err)
	}
	var compte [256]int
	var films [256]int
	nFilms, total := 0, 0
	for _, e := range ents {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(root, e.Name())
		var loc [256]int
		n := filmdec.CountFilmChunks(dir)
		if n == 0 {
			continue
		}
		nFilms++
		for c := 1; c <= n; c++ {
			chunk, err := filmdec.ReadFilmChunk(dir, c)
			if err != nil {
				continue
			}
			for _, p := range filmdec.WalkPackets(chunk) {
				if p.Type != filmdec.PacketTypeDelta || p.Size < 1 {
					continue
				}
				loc[p.Payload(chunk)[0]]++
			}
		}
		for b, v := range loc {
			compte[b] += v
			if v > 0 {
				films[b]++
			}
		}
		total++
	}
	t.Logf("CORPUS — %d films lus", nFilms)
	sousSeuil := 0
	for b := 0; b < 0x80; b++ {
		sousSeuil += compte[b]
	}
	t.Logf("CONTROLE DE GRAMMAIRE — paquets dont le premier octet est < 0x80 : %d"+
		" (attendu 0 si le bit de continuation est toujours a 1)", sousSeuil)
	for b := 0x80; b < 0x100; b++ {
		if compte[b] > 0 {
			t.Logf("  0x%02X = type %3d : %10d paquets sur %4d films", b, b&0x7f, compte[b], films[b])
		}
	}
	for _, q := range []struct {
		o   byte
		nom string
	}{{0xA4, "action_weapon_fire (type 36)"}, {0x95, "unit_zoom (type 21)"}, {0xD2, "type 82"}} {
		t.Logf("REPONSE — 0x%02X %s : %d paquets sur %d films", q.o, q.nom, compte[q.o], films[q.o])
	}
}

// TestViseeTaillesPaquets mesure la distribution des TAILLES de payload par premier octet. C'est
// l'arbitre entre deux lectures incompatibles du debut d'un paquet delta : « en-tete d'evenement
// (1 bit de continuation + R(7) type + references) », qui exige au moins 11 bits, et « trame
// d'etat dont le premier octet n'est pas un type ». Un paquet d'UN SEUL octet ne peut pas porter
// un en-tete d'evenement. Garde TAILLES_FILM (un film) ou TAILLES_CORPUS (tout le corpus).
func TestViseeTaillesPaquets(t *testing.T) {
	root := os.Getenv("TAILLES_CORPUS")
	un := os.Getenv("TAILLES_FILM")
	if root == "" && un == "" {
		t.Skipf("TAILLES_CORPUS/TAILLES_FILM absents : mesure sautee")
	}
	var dirs []string
	if un != "" {
		dirs = []string{un}
	} else {
		ents, err := os.ReadDir(root)
		if err != nil {
			t.Fatalf("corpus illisible : %v", err)
		}
		for _, e := range ents {
			if e.IsDir() {
				dirs = append(dirs, filepath.Join(root, e.Name()))
			}
		}
	}
	courts := map[byte]int{}
	total, sous2o := 0, 0
	tailleMin := map[byte]int{}
	for _, dir := range dirs {
		n := filmdec.CountFilmChunks(dir)
		for c := 1; c <= n; c++ {
			chunk, err := filmdec.ReadFilmChunk(dir, c)
			if err != nil {
				continue
			}
			for _, p := range filmdec.WalkPackets(chunk) {
				if p.Type != filmdec.PacketTypeDelta || p.Size < 1 {
					continue
				}
				b := p.Payload(chunk)[0]
				total++
				if m, ok := tailleMin[b]; !ok || p.Size < m {
					tailleMin[b] = p.Size
				}
				if p.Size < 2 {
					sous2o++
					courts[b]++
				}
			}
		}
	}
	t.Logf("PAQUETS — %d au total ; %d font MOINS DE 2 OCTETS (%.4f %%)",
		total, sous2o, 100*float64(sous2o)/float64(total))
	for b, n := range courts {
		t.Logf("  premier octet 0x%02X : %d paquets d'un seul octet", b, n)
	}
	t.Log("TAILLE MINIMALE observee par premier octet (les octets frequents) :")
	for _, b := range []byte{0xA0, 0xD2, 0xD3, 0xC7, 0xC0, 0xE9, 0xE5, 0x80} {
		if m, ok := tailleMin[b]; ok {
			t.Logf("  0x%02X : %d octets", b, m)
		}
	}
	t.Log("ARBITRAGE : un paquet d'UN octet ne peut pas porter un en-tete d'evenement" +
		" (1 + 7 + au moins 3 portes = 11 bits) ; s'il en existe, le premier octet du payload" +
		" n'est pas, a lui seul, un numero de type d'evenement.")
}
