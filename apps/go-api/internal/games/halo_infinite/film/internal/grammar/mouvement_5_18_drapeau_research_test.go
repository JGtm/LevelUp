//go:build research

package grammar

// mouvement_5_18_drapeau_research_test.go — LA VALEUR DU DRAPEAU DE CONTROLE DE CORRUPTION,
// LUE DANS LE FILM (lot 5.18.1).
//
// Il ne decode rien : il lit le bit de `base+0x0CB45C` par le lecteur de PRODUCTION
// ([ReadFilmIdentity] -> `ControleDeCorruption`) et publie sa valeur, sur le film courant puis
// sur tout le cache. C est la VERIFICATION du maillon lu chez l ecrivain
// (`FUN_14299b198` @14299b25b / `FUN_14299ab50` @14299ac28 / `FUN_1428e219c` @1428e2239) —
// la preuve est l ecrivain, ce tableau n en est que le controle.
//
// Rejouable :
//
//	MOUV511_FILM=<dir> go test -tags=research -count=1 -v \
//	  -run '^TestDrapeau518Temoin$' ./internal/games/halo_infinite/film/internal/grammar/
//	CHUNK00_CORPUS=<racine> go test -tags=research -count=1 -v -timeout 30m \
//	  -run '^TestDrapeau518Corpus$' ./internal/games/halo_infinite/film/internal/grammar/

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// TestDrapeau518Temoin publie le drapeau du film courant, et les octets qui le portent.
func TestDrapeau518Temoin(t *testing.T) {
	film := m511Film(t)
	reg, ok := FilmRegistryChunk(film)
	if !ok {
		t.Fatalf("le film ne porte pas son chunk_00")
	}
	id, err := ReadFilmIdentity(reg)
	if err != nil {
		t.Fatalf("identite : %v", err)
	}
	off := id.BuildOffset + identBoolOff
	t.Logf("BUILD %q · format %d · blocs %d · buildOff 0x%X · octet du drapeau 0x%X",
		id.Build, id.FormatVersion, id.RegistryBlocks, id.BuildOffset, off)
	t.Logf("CONTROLE DE CORRUPTION = %v  (bit de poids fort de l octet 0x%X)",
		id.ControleDeCorruption, off)
	t.Logf("  buildID %d · changelist %d · corps au bit %d",
		id.BuildID, id.Changelist, id.BodyBit)
	t.Logf("  octets 0x%X..0x%X : % 02X", off-8, off+7, reg[off-8:off+8])
	t.Logf("  les huit bits de l octet 0x%X : %08b", off, reg[off])
}

// TestDrapeau518Corpus ventile le drapeau sur tout le cache, par build et par format.
func TestDrapeau518Corpus(t *testing.T) {
	dirs := corpusFilms(t)
	if len(dirs) == 0 {
		t.Skip("CHUNK00_CORPUS / CHUNK00_FILMS absents")
	}
	var lus, sansSection, leves int
	parCle := map[string][2]int{}
	for _, dir := range dirs {
		brut, err := os.ReadFile(filepath.Join(dir, "chunk_00.bin")) //nolint:gosec // corpus local
		if err != nil {
			continue
		}
		id, err := ReadFilmIdentity(source.Inflate(brut))
		if errors.Is(err, ErrNoFilmIdentity) {
			sansSection++
			continue
		}
		if err != nil {
			continue
		}
		lus++
		cle := fmt.Sprintf("%s / format %d", id.Build, id.FormatVersion)
		v := parCle[cle]
		if id.ControleDeCorruption {
			leves++
			v[1]++
		} else {
			v[0]++
		}
		parCle[cle] = v
		if id.ControleDeCorruption {
			t.Logf("  LEVE : %s", filepath.Base(dir))
		}
	}
	t.Logf("CORPUS : %d films lus · %d sans section · drapeau LEVE sur %d", lus, sansSection, leves)
	cles := make([]string, 0, len(parCle))
	for k := range parCle {
		cles = append(cles, k)
	}
	sort.Strings(cles)
	for _, k := range cles {
		t.Logf("  %-28s  faux %5d · vrai %5d", k, parCle[k][0], parCle[k][1])
	}
}
