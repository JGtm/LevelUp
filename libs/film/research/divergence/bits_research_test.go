//go:build research

package divergence

// bits_research_test.go : LOUPE SUR UN PAQUET (plan marche complete, A1.3). Rend, pour UN paquet,
// notre structure de lecture (records et composants avec leur etendue) et les bits du payload d une
// plage, pour relire a la main les positions du temoin.
//
//	BITS_FILM=<id8> BITS_CHUNK=<n> BITS_PAQUET=<rang> [BITS_DE=<bit> BITS_A=<bit>] [BITS_RECORDS=1]
//	DIVERGENCE_RACINE=<film_chunks> go test -tags=research ./research/divergence/ -run TestBits -v
//
// Les positions sont celles des TSV du temoin : bit 0 = bit de poids fort du premier octet de la
// charge utile du paquet.

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
	"github.com/JGtm/halo-infinite-film-decoder/internal/profile"
)

func envInt(nom string, defaut int) int {
	if v, err := strconv.Atoi(os.Getenv(nom)); err == nil {
		return v
	}
	return defaut
}

// formeBits rend les bits [de, a) de la charge utile, groupes par 8 a partir de `de`.
func formeBits(payload []byte, de, a int) string {
	var sb strings.Builder
	for i := de; i < a && i < 8*len(payload); i++ {
		if i > de && (i-de)%8 == 0 {
			sb.WriteByte(' ')
		}
		sb.WriteByte('0' + (payload[i/8]>>(7-uint(i%8)))&1)
	}
	return sb.String()
}

func TestBits(t *testing.T) {
	racine, id := os.Getenv("DIVERGENCE_RACINE"), os.Getenv("BITS_FILM")
	if racine == "" || id == "" {
		t.Skip("DIVERGENCE_RACINE et BITS_FILM requis")
	}
	chunk, paquet := envInt("BITS_CHUNK", -1), envInt("BITS_PAQUET", -1)
	fc, err := ouvrirFilm(racine, id, os.Getenv("BITS_NIVEAU"), catalogueDeLEnvironnement(t))
	if err != nil {
		t.Fatal(err)
	}
	for p, err := range fc.Trames(grammar.NouvelleObservation()) {
		if err != nil {
			t.Fatal(err)
		}
		if p.Chunk != chunk || p.Index != paquet {
			continue
		}
		fmt.Printf("paquet %d/%d (debut %d, etatA %d, presume %d) : %d octets, vue A [%d,+%d) genres %v, vue B [%d,+%d) sortie %d eid rejete %#x, vue C [%d,+%d)\n",
			p.Chunk, p.Index, p.Debut, p.VueA.Etat, p.VueA.PremierPresume, len(p.Payload), p.VueA.Debut, p.VueA.Bits, p.VueA.Genres, p.VueB.Debut, p.VueB.Bits, p.VueB.Sortie, p.VueB.EIDRejete, p.VueC.Debut, p.VueC.Bits)
		if os.Getenv("BITS_RECORDS") != "" {
			for i := range p.Records {
				r := &p.Records[i]
				fmt.Printf("  record %d : debut %d bits %d genre %d slot %d gen %d ti %d desync %d\n", i, r.Debut, r.Bits, r.Genre, r.Vie.Slot, r.Vie.Gen, r.TI, r.Desync)
				for _, c := range p.Comps[r.Comps[0]:r.Comps[1]] {
					fmt.Printf("      i%d etat %d debut %d bits %d prov %d arret %d\n", c.Index, c.Etat, c.Debut, c.Bits, c.Prov, c.Arret)
				}
			}
		}
		if de, a := envInt("BITS_DE", -1), envInt("BITS_A", -1); de >= 0 && a > de {
			fmt.Printf("bits [%d,%d) : %s\n", de, a, formeBits(p.Payload, de, a))
		}
		return
	}
	t.Fatalf("paquet %d/%d introuvable", chunk, paquet)
}

// TestDatums : la table de datums que NOTRE lecture tire de l image-cle d un chunk (A1.3, rejets hors
// datum). BITS_FILM, BITS_CHUNK, BITS_SLOT (liste separee par des virgules) ; rend la taille de la table
// et le type de chaque slot demande (-1 : absent).
func TestDatums(t *testing.T) {
	racine, id := os.Getenv("DIVERGENCE_RACINE"), os.Getenv("BITS_FILM")
	if racine == "" || id == "" {
		t.Skip("DIVERGENCE_RACINE et BITS_FILM requis")
	}
	chunk := envInt("BITS_CHUNK", -1)
	fc, err := ouvrirFilm(racine, id, os.Getenv("BITS_NIVEAU"), catalogueDeLEnvironnement(t))
	if err != nil {
		t.Fatal(err)
	}
	data, pks, ok := fc.ChunkAt(chunk)
	if !ok {
		t.Fatalf("chunk %d absent", chunk)
	}
	for _, pk := range pks {
		if pk.Type != grammar.PacketTypeKeyframe {
			continue
		}
		table, ambigus := grammar.TableDeDatums(pk.Payload(data))
		fmt.Printf("chunk %d image-cle paquet %d : %d octets, table de %d slots, %d ambigus\n", chunk, pk.Index, len(pk.Payload(data)), len(table), ambigus)
		for _, s := range strings.Split(os.Getenv("BITS_SLOT"), ",") {
			n, err := strconv.Atoi(s)
			if err != nil {
				continue
			}
			if ti, ok := table[uint32(n)]; ok { //nolint:gosec // slot de l operateur
				fmt.Printf("  slot %d : ti=%d\n", n, ti)
			} else {
				fmt.Printf("  slot %d : absent\n", n)
			}
		}
	}
}

// catalogueDeLEnvironnement : le catalogue de DIVERGENCE_CARTE, nil quand la variable est vide.
func catalogueDeLEnvironnement(t *testing.T) *profile.MapQuantCatalog {
	chemin := os.Getenv("DIVERGENCE_CARTE")
	if chemin == "" {
		return nil
	}
	cat, err := profile.LoadMapQuantCatalog(chemin)
	if err != nil {
		t.Fatal(err)
	}
	return cat
}

// TestRegistre : ou vivent, dans le registre d UN film, les composants d interet (A1.5, builds anciens).
// BITS_FILM ; BITS_NOMS : sous-chaines de noms de composants separees par des virgules.
func TestRegistre(t *testing.T) {
	racine, id := os.Getenv("DIVERGENCE_RACINE"), os.Getenv("BITS_FILM")
	if racine == "" || id == "" {
		t.Skip("DIVERGENCE_RACINE et BITS_FILM requis")
	}
	fc, err := ouvrirFilm(racine, id, os.Getenv("BITS_NIVEAU"), catalogueDeLEnvironnement(t))
	if err != nil {
		t.Fatal(err)
	}
	reg, err := fc.Registry()
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("%s : %d archetypes\n", id, len(reg.Archetypes))
	for _, motif := range strings.Split(os.Getenv("BITS_NOMS"), ",") {
		var trouves []string
		for _, a := range reg.Archetypes {
			for i, nom := range a.Components {
				if strings.Contains(nom, motif) {
					trouves = append(trouves, fmt.Sprintf("ti=%d i%d", a.Index, i))
				}
			}
		}
		fmt.Printf("  %-52s %s\n", motif, strings.Join(trouves, "; "))
	}
}

// TestArchetype : les composants d un archetype du registre (BITS_FILM, BITS_TI).
func TestArchetype(t *testing.T) {
	racine, id := os.Getenv("DIVERGENCE_RACINE"), os.Getenv("BITS_FILM")
	if racine == "" || id == "" {
		t.Skip("DIVERGENCE_RACINE et BITS_FILM requis")
	}
	fc, err := ouvrirFilm(racine, id, os.Getenv("BITS_NIVEAU"), catalogueDeLEnvironnement(t))
	if err != nil {
		t.Fatal(err)
	}
	reg, err := fc.Registry()
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range strings.Split(os.Getenv("BITS_TI"), ",") {
		n, err := strconv.Atoi(s)
		if err != nil {
			continue
		}
		if a, ok := reg.Archetype(n); ok {
			fmt.Printf("ti=%d : %d composants\n", n, len(a.Components))
			for i, nom := range a.Components {
				fmt.Printf("   i%d %s\n", i, nom)
			}
		}
	}
}
