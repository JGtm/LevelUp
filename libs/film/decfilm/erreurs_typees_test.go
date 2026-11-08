package decfilm_test

// erreurs_typees_test.go — LES DEUX SENTINELLES DE LA TABLE DE PROFIL SE NOMMENT DEPUIS LA FACADE.
//
// Un consommateur du module ne peut pas importer `internal/profile` : sans ce re-export il recevait
// l erreur typee « build inconnu » (ADR 0034 D-4) sans pouvoir la reconnaitre. Le test n emploie que
// l API publique du module : `replay.CleDeChunk0` rend, pour un `chunk_00` donne, la cle du film et
// l erreur que la table de profil lui oppose.

import (
	"bytes"
	"encoding/binary"
	"errors"
	"os"
	"testing"

	"github.com/JGtm/LevelUp/libs/film/decfilm"
	"github.com/JGtm/LevelUp/libs/film/replay"
)

// Le temoin : le `chunk_00` de la fixture `bcb6d393` (build `HI_1_12_0`, format 27), de quoi
// fabriquer un build et un format hors table sans deplacer un octet.
const (
	chunk0Temoin     = "../replay/testdata/minifilm_bcb6d393/chunk_00.bin"
	buildDuTemoin    = "HI_1_12_0"
	buildInconnu     = "HI_9_99_9" // meme longueur : aucun octet de la section ne bouge
	formatInconnu    = 0x7f7f
	decalageDuFormat = 4 // `chunk_00 + 4` : la version de format
)

func chunk0Decompresse(t *testing.T) []byte {
	t.Helper()
	brut, err := os.ReadFile(chunk0Temoin)
	if err != nil {
		t.Fatalf("lecture de %s : %v", chunk0Temoin, err)
	}
	return decfilm.Inflate(brut)
}

func TestLesSentinellesDeLaTableDeProfilSontNommablesParUnConsommateurExterne(t *testing.T) {
	temoin := chunk0Decompresse(t)

	t.Run("cle connue : aucune erreur", func(t *testing.T) {
		if c := replay.CleDeChunk0(temoin); c.Err != nil || c.Build != buildDuTemoin {
			t.Fatalf("le temoin doit etre une cle connue : build %q, erreur %v", c.Build, c.Err)
		}
	})

	t.Run("build hors table", func(t *testing.T) {
		patche := bytes.Replace(temoin, []byte(buildDuTemoin), []byte(buildInconnu), 1)
		c := replay.CleDeChunk0(patche)
		if c.Build != buildInconnu {
			t.Fatalf("build lu %q, attendu %q : le temoin n a pas ete patche", c.Build, buildInconnu)
		}
		if !errors.Is(c.Err, decfilm.ErrUnknownBuild) {
			t.Fatalf("errors.Is(%v, decfilm.ErrUnknownBuild) est faux", c.Err)
		}
		if errors.Is(c.Err, decfilm.ErrUnknownFormat) {
			t.Fatalf("un build hors table ne doit pas se lire comme un format hors table : %v", c.Err)
		}
		if !bytes.Contains([]byte(c.Err.Error()), []byte(buildInconnu)) {
			t.Fatalf("l erreur ne nomme pas le build refuse : %v", c.Err)
		}
	})

	t.Run("format hors table", func(t *testing.T) {
		patche := append([]byte(nil), temoin...)
		binary.LittleEndian.PutUint32(patche[decalageDuFormat:], formatInconnu)
		c := replay.CleDeChunk0(patche)
		if c.Format != formatInconnu {
			t.Fatalf("format lu %#x, attendu %#x : le temoin n a pas ete patche", c.Format, formatInconnu)
		}
		if !errors.Is(c.Err, decfilm.ErrUnknownFormat) {
			t.Fatalf("errors.Is(%v, decfilm.ErrUnknownFormat) est faux", c.Err)
		}
		if errors.Is(c.Err, decfilm.ErrUnknownBuild) {
			t.Fatalf("un format hors table ne doit pas se lire comme un build hors table : %v", c.Err)
		}
	})
}
