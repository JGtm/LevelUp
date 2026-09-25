package replay

// deaths_source_tempsforts_test.go — LE MORCEAU DES TEMPS FORTS SE CHOISIT PAR SON TYPE (lot L3,
// 2026-09-23).
//
// `ScanDeaths` prenait « le DERNIER numero » du film. Sur un film archive avant sa finalisation
// (`ab526724`, 34 morceaux sur 37), le dernier numero est un morceau de REPLICATION : la lecture
// echouait sur un message generique, le fil des morts tombait, et toute la chaine d identite
// avec lui — sans que rien ne dise pourquoi. Quand le manifeste est la, son TYPE designe le
// morceau ; et un film dont le manifeste n en porte aucun rend une erreur TYPEE, distincte d un
// morceau illisible.
//
// Les octets sont ceux de la bobine v40 commise au depot (`miniBobineV40`) : `chunk_00` est le
// registre, `chunk_01` de la replication, `chunk_02` les temps forts.

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/filmcache"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// octetsBobineV40 rend les trois morceaux BRUTS de la bobine v40, dans l ordre des numeros.
func octetsBobineV40(t *testing.T) [][]byte {
	t.Helper()
	out := make([][]byte, 0, 3)
	for _, nom := range []string{"chunk_00.bin", "chunk_01.bin", "chunk_02.bin"} {
		b, err := os.ReadFile(filepath.Join(miniBobineV40, nom))
		if err != nil {
			t.Fatalf("bobine v40 : %v", err)
		}
		out = append(out, b)
	}
	return out
}

// TestScanDeaths_TempsFortsChoisiParSonType : les temps forts ne sont PAS le dernier numero —
// le manifeste les designe au numero 1, suivis d un morceau de replication au numero 2.
// L ancienne regle (« le dernier numero ») lisait la replication.
func TestScanDeaths_TempsFortsChoisiParSonType(t *testing.T) {
	o := octetsBobineV40(t)
	film, err := source.Load(source.MemoryChunks{o[0], o[2], o[1]}, []types.ChunkMeta{
		{Index: 0, ChunkType: 1},
		{Index: 1, ChunkType: filmcache.ChunkTypeTempsForts},
		{Index: 2, ChunkType: 2},
	})
	if err != nil {
		t.Fatalf("chargement : %v", err)
	}
	deaths, err := ScanDeaths(film)
	if err != nil {
		t.Fatalf("ScanDeaths : %v — le morceau des temps forts n est pas choisi par son type", err)
	}
	if len(deaths) == 0 {
		t.Fatal("aucune mort lue dans le morceau des temps forts")
	}
}

// TestScanDeaths_DernierMorceauDeReplication_ErreurTypee : LA SIGNATURE D UN FILM NON FINALISE.
// Le manifeste ne porte aucun morceau des temps forts ; son dernier morceau est de la replication.
// L erreur est TYPEE (et de la famille « non finalise ») — meme si, ici, les octets du dernier
// numero SONT des temps forts : c est le manifeste qui fait foi, pas une lecture qui reussirait
// par hasard.
func TestScanDeaths_DernierMorceauDeReplication_ErreurTypee(t *testing.T) {
	o := octetsBobineV40(t)
	film, err := source.Load(source.MemoryChunks(o), []types.ChunkMeta{
		{Index: 0, ChunkType: 1}, {Index: 1, ChunkType: 2}, {Index: 2, ChunkType: 2},
	})
	if err != nil {
		t.Fatalf("chargement : %v", err)
	}
	_, err = ScanDeaths(film)
	if !errors.Is(err, ErrFilSansTempsForts) {
		t.Fatalf("err = %v, attendu ErrFilSansTempsForts", err)
	}
	if !errors.Is(err, filmcache.ErrFilmNonFinalise) {
		t.Errorf("err = %v : hors de la famille filmcache.ErrFilmNonFinalise", err)
	}
}
