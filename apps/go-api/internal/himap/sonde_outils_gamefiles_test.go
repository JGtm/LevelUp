//go:build gamefiles

package himap

// Outils communs des sondes d'investigation (food, scnr, stse) et des temoins gamefiles.
//
// Ils vivaient dans la sonde sddt/pfnd du 2026-08-10 ; la lecture sddt promue en production
// (`sddt.go`) a emporte la navigation de struct-table (meilleurTagInfo, liensBlocs,
// liensVers, compteChamp) — ne restent ici que les utilitaires strictement de test.

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// moduleAny compose le chemin du module any/ d'une carte (memes noms de dossier que pc/ :
// ridgeline, catalyst).
func moduleAny(t *testing.T, carte string) string {
	t.Helper()
	dir, err := LevelsDir("any")
	if err != nil {
		t.Skipf("installation du jeu introuvable : %v", err)
	}
	p := filepath.Join(dir, carte, carte+"-rtx-new.module")
	if _, err := os.Stat(p); err != nil {
		t.Skipf("module absent (%s)", p)
	}
	return p
}

// dumpRecord publie un enregistrement en double lecture f32 / u32, pour lecture humaine.
func dumpRecord(t *testing.T, tag []byte, base, taille int, prefixe string) {
	t.Helper()
	ligne := prefixe + " f32:"
	for o := 0; o < taille; o += 4 {
		ligne += fmt.Sprintf(" %10.3f", f32(tag, base+o))
	}
	t.Log(ligne)
	ligne = prefixe + " u32:"
	for o := 0; o < taille; o += 4 {
		ligne += fmt.Sprintf(" %08x", uint32(u32(tag, base+o)))
	}
	t.Log(ligne)
}
