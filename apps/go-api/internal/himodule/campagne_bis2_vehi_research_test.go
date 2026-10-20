//go:build research

// campagne_bis2_vehi_research_test.go — MESURES BIS 2 DE LA CAMPAGNE DE GRAMMAIRE (2026-10-01),
// T6-C2 : LE TYPE DE PHYSIQUE DES CHASSIS `vehi`, lu dans les `.module` installes. Lecture SEULE,
// aucun film ouvert, rien d ecrit.
//
// CE QUE LE JEU LIT (note T6 §2.2, Ghidra lecture seule) : la porte `+0x818` de l etat vehicule
// vaut `FUN_1408b44fc(vehi) == 6`, et `FUN_1408b44fc` rend l index k (0..11) du PREMIER des douze
// blocs `vehi + 0xde0 + 0x14*k` dont le compte (`+0x10` du champ de 20 octets) est non nul, 13
// sinon. La table des noms (`0x1448021a0`) dit k = 6 « vtol ». L instrument relit ces douze
// comptes dans la MainStruct du tag (la disposition d un champ de tableau est la meme dans le
// fichier : 20 octets, compte a +0x10, cf. `m4bBarillets`), pour chaque chassis demande.
//
//	BIS2_DEPLOY=<bibliotheque>/<jeux>/Halo Infinite/deploy BIS2_VEHI=0000254b,77ef810a \
//	  go test -tags=research -count=1 -v -run '^TestCampagneBis2TypeDePhysique$' ./internal/himodule/

package himodule_test

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// bis2NomsDeType : la table `0x1448021a0` (note T6 §2.3).
var bis2NomsDeType = []string{"human_tank", "human_jeep", "human_plane", "alien_scout", "alien_fighter",
	"turret", "vtol", "chopper", "guardian", "jackal_glider", "space_fighter", "revenant"}

const (
	bis2PhysiqueBase = 0xde0
	bis2PhysiquePas  = 0x14
	bis2Compte       = 0x10
)

// TestCampagneBis2TypeDePhysique publie, pour chaque `vehi` demande, le module ou il est trouve,
// les douze comptes et le type que `FUN_1408b44fc` en deduirait.
func TestCampagneBis2TypeDePhysique(t *testing.T) {
	racine := strings.TrimSpace(os.Getenv("BIS2_DEPLOY"))
	if racine == "" {
		t.Skip("BIS2_DEPLOY non pose (racine deploy de l installation)")
	}
	restants := map[uint32]bool{}
	for _, g := range m4bGids(os.Getenv("BIS2_VEHI")) {
		restants[g] = true
	}
	var modules []string
	for _, d := range []string{"any/globals", "pc/globals", "ds/globals", "any/levels/multi/*", "ds/levels/multi/*"} {
		m, _ := filepath.Glob(filepath.Join(racine, d, "*.module"))
		modules = append(modules, m...)
	}
	for _, chemin := range modules {
		if len(restants) == 0 {
			break
		}
		mc, ic := ca9Index(t, chemin)
		for g := range restants {
			f, ok := ic[g]
			if !ok {
				continue
			}
			if f.Group != "vehi" {
				t.Logf("BIS2_VEHI\t%08x\t%s\tgroupe %s (pas un vehi)", g, filepath.Base(chemin), f.Group)
				delete(restants, g)
				continue
			}
			data, err := mc.Extract(f)
			if err != nil {
				t.Logf("BIS2_VEHI\t%08x\t%s\textraction : %v", g, filepath.Base(chemin), err)
				continue
			}
			tag, ok := m4bOuvrir(data)
			if !ok {
				t.Logf("BIS2_VEHI\t%08x\t%s\ttag illisible", g, filepath.Base(chemin))
				continue
			}
			abs, taille := tag.bloc(tag.racine())
			fin := bis2PhysiqueBase + 12*bis2PhysiquePas
			if abs < 0 || taille < fin || abs+fin > len(data) {
				t.Logf("BIS2_VEHI\t%08x\t%s\tMainStruct de %d octets, trop courte", g, filepath.Base(chemin), taille)
				continue
			}
			comptes := make([]uint32, 12)
			premier := 13
			for k := 0; k < 12; k++ {
				comptes[k] = binary.LittleEndian.Uint32(data[abs+bis2PhysiqueBase+k*bis2PhysiquePas+bis2Compte:])
				if comptes[k] != 0 && premier == 13 {
					premier = k
				}
			}
			nom := "aucun"
			if premier < len(bis2NomsDeType) {
				nom = bis2NomsDeType[premier]
			}
			t.Logf("BIS2_VEHI\t%08x\t%s\tmain=%d\tcomptes=%v\ttype=%d\t%s\tporte_818=%t", g, filepath.Base(chemin),
				taille, comptes, premier, nom, premier == 6)
			delete(restants, g)
		}
	}
	for g := range restants {
		t.Logf("BIS2_VEHI\t%08x\tabsent des modules globals installes", g)
	}
}
