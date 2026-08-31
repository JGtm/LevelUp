package mapvar

// assaut_sites_mvar_test.go — LES SITES D'AMORCAGE SONT-ILS NOMMES DANS LE FICHIER DE CARTE ?
//
// Le script du mode designe ses objets par des NOMS (`defender_bombsite`, `attacker_bombsite`,
// table `armzoneArgs`). Une variante de carte porte une liste de chaines lisibles
// (`Variant.Names`, racine 10/1) — c'est par la que le mode retrouve ses objets. Si ces noms y
// sont, la position des sites d'amorcage se lit DIRECTEMENT dans le fichier, et la phase A3 (qui
// les cherchait par ancrage `ti=13`) devient sans objet.
//
//	$env:VEHI_MVAR_DIR="C:/.../scratchpad/mvar"
//	go test ./internal/analysis/replay/mapvar/ -run AssautSitesMvar -v

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// assautMotifs : ce qu'on cherche dans les noms d'une variante de carte.
var assautMotifs = []string{"bombsite", "bomb", "armzone", "arming", "assault", "goalplate"}

func TestAssautSitesMvar(t *testing.T) {
	dir := os.Getenv("VEHI_MVAR_DIR")
	if dir == "" {
		t.Skip("mesure non demandee : VEHI_MVAR_DIR requis (corpus mapobj-build --save-mvar)")
	}
	trouves := map[string]map[string]bool{}
	fichiers := 0
	err := filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(strings.ToLower(p), ".mvar") {
			return nil
		}
		buf, err := os.ReadFile(p)
		if err != nil {
			return nil
		}
		v, err := Parse(buf)
		if err != nil {
			return nil
		}
		fichiers++
		carte := filepath.Base(filepath.Dir(p))
		for _, n := range v.Names {
			bas := strings.ToLower(n)
			for _, m := range assautMotifs {
				if strings.Contains(bas, m) {
					if trouves[n] == nil {
						trouves[n] = map[string]bool{}
					}
					trouves[n][carte] = true
					break
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("balayage : %v", err)
	}
	noms := make([]string, 0, len(trouves))
	for n := range trouves {
		noms = append(noms, n)
	}
	sort.Strings(noms)
	t.Logf("%d fichiers .mvar balayes ; %d nom(s) d'objet correspondant a %v", fichiers, len(noms), assautMotifs)
	for _, n := range noms {
		t.Logf("  %-40s sur %d carte(s)", n, len(trouves[n]))
	}
}
