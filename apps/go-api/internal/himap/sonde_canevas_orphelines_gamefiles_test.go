package himap

// SONDE DES CARTES ORPHELINES : celles dont le depot porte le `.mvar` par defaut mais
// AUCUN fichier-lien de canevas, d'ou le classement « canevas inconnu » du registre des
// cartes (Argyle, Detachment).
//
// CE QUE LA SONDE ETABLIT. Le fichier-lien n'est qu'une CORROBORATION : la preuve
// canevas repose sur le level_id (`root[1][0][0]` du `.mvar`) et sur l'unicite du module
// installe qui porte ce level_id en tag `levl` — exactement la methode de
// `TestPreuveLevelIDCartes`, moins l'etape de corroboration qui n'a pas de fichier.
//
// Ce que la sonde mesure AUSSI, parce que c'est la condition qui manquait au lot du
// 2026-08-26 : le nombre d'objectifs du `.mvar`. Sans ancre, pas de cadre, donc pas de
// fond — un canevas prouve ne suffit pas.
import (
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/analysis/replay/mapvar"
)

// cartesOrphelines : les cartes du depot sans fichier-lien de canevas. Une entree sort de
// cette table le jour ou elle entre dans `CartesForge` — pas avant.
var cartesOrphelines = []string{"argyle", "detachment"}

func TestSondeCanevasCartesOrphelines(t *testing.T) {
	root, err := DeployRoot()
	if err != nil {
		t.Skip(err)
	}
	index, nModules := indexLevlInstalle(t, root)
	t.Logf("index levl : %d level_id distincts sur %d modules", len(index), nModules)

	for _, base := range cartesOrphelines {
		t.Run(base, func(t *testing.T) {
			chemin, cerr := cheminDepuisDepot(filepath.Join(DepotVariantesCarte, base+"_map.mvar"))
			if cerr != nil {
				t.Skip(cerr)
			}
			brut, rerr := os.ReadFile(chemin) //nolint:gosec // chemin de test, lecture seule
			if rerr != nil {
				t.Fatal(rerr)
			}
			v, perr := mapvar.Parse(brut)
			if perr != nil {
				t.Fatalf("%s : %v", base, perr)
			}
			if v.LevelID == 0 {
				t.Fatalf("%s : level_id nul — la sonde ne prouverait rien", base)
			}
			dossiers := dossiersDistincts(index[uint32(v.LevelID)])
			t.Logf("%s : level_id %d (0x%08X) -> %v | objets %d | objectifs %d",
				base, v.LevelID, uint32(v.LevelID), dossiers, len(v.Objects), len(v.Objectives()))
			if len(dossiers) > 1 {
				t.Fatalf("%s : le level_id %d designe %d dossiers (%v) — l'unicite est la preuve",
					base, v.LevelID, len(dossiers), dossiers)
			}
		})
	}
}
