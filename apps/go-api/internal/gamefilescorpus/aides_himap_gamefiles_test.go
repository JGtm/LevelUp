//go:build gamefiles

package gamefilescorpus

// aides_himap_gamefiles_test.go — les petites aides de test que ce corpus partage avec les tests
// internes de `internal/himap`.
//
// POURQUOI ELLES SONT COPIEES ET NON PARTAGEES. Un test d'une brique ne peut pas importer un
// paquet de LevelUp, et le corpus ne peut pas importer les `_test.go` d'un autre paquet : chaque
// côté porte donc sa copie (deux copies au plus, règle 6 de CLAUDE.md ; aucun paquet d'aides
// commun entre la brique et le module). Les originaux vivent dans `internal/himap/` :
// `calage_banc_test.go` (calage de la carte validée), `instances_gamefiles_test.go`
// (moduleDuJeu), `sonde_outils_gamefiles_test.go` (moduleAny), `geometry_gamefiles_test.go`
// (mediane), `carte_globaux_gamefiles_test.go` (part), `origine_*_gamefiles_test.go`
// (origFokiDe, origIndexForge, itoaSimple), `sonde_levelid_gamefiles_test.go`
// (indexLevlInstalle, dossiersDistincts).

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/himap"

	"github.com/JGtm/LevelUp/libs/gamefiles/himodule"
)

// Calage de `carte_validee_v1.png`, mesuré sur les trajectoires de joueur
// (0,0920 m/px, X0 -43,5, Y1 61,0).
const (
	gateEchelle = 0.0920 // mètres par pixel
	gateX0      = -43.5
	gateY1      = 61.0
)

func part(v []float64, seuil float64) float64 {
	n := 0
	for _, x := range v {
		if x > seuil {
			n++
		}
	}
	return float64(n) / float64(len(v))
}

func mediane(v []float64) float64 {
	c := append([]float64(nil), v...)
	sort.Float64s(c)
	return c[len(c)/2]
}

// moduleDuJeu compose le chemin d'un module, ou declare le test absent.
//
// L'installation n'est plus ecrite en dur ici : elle passe par himap.DeployRoot()
// (deploy_root.go). Motif — le chemin cable pointait sur un disque qui n'existe plus, et
// ces tests se declaraient « module absent » en passant au vert.
func moduleDuJeu(t *testing.T, variante, carte string) string {
	t.Helper()
	dir, err := himap.LevelsDir(variante)
	if err != nil {
		t.Skipf("installation du jeu introuvable : %v", err)
	}
	p := filepath.Join(dir, carte, carte+"-rtx-new.module")
	if _, err := os.Stat(p); err != nil {
		t.Skipf("module absent (%s)", p)
	}
	return p
}

// origFokiDe rend les GlobalID des `foki` référencés par un type.
func origFokiDe(idxForge, idxRef *himap.ModuleIndex, typeID uint32) []uint32 {
	tag, err := idxForge.Extract(typeID)
	if err != nil {
		return nil
	}
	var out []uint32
	vus := map[uint32]bool{}
	himap.RefsInline(tag, func(h uint32) bool {
		if g, _, ok := idxRef.Lookup(h); ok && g == himap.GroupeRamassable && !vus[h] {
			vus[h] = true
			out = append(out, h)
		}
		return false
	})
	return out
}

func origIndexForge(t *testing.T, racine string) *himap.ModuleIndex {
	t.Helper()
	chemins := []string{filepath.Join(racine, "any", "globals", "forge", "forge_objects-rtx-new.module")}
	if p := filepath.Join(racine, "pc", "globals", "forge", "forge_objects-rtx-new.module"); estFichier(p) {
		chemins = append(chemins, p)
	}
	idx, err := himap.NewModuleIndex(chemins...)
	if err != nil {
		t.Skipf("index Forge indisponible : %v", err)
	}
	return idx
}

func itoaSimple(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

// indexLevlInstalle balaye tous les .module de `any/levels` + `ds/levels` et rend
// GlobalID des tags `levl` -> chemins des modules porteurs, plus le compte de modules.
//
// Un module illisible est une ERREUR, jamais un saut silencieux : un balayage partiel
// affaiblirait la pretention d'unicite sans que rien ne le dise.
func indexLevlInstalle(t *testing.T, deployRoot string) (map[uint32][]string, int) {
	t.Helper()
	index := map[uint32][]string{}
	n := 0
	for _, variante := range []string{"any", "ds"} {
		racine := filepath.Join(deployRoot, variante, "levels")
		err := filepath.WalkDir(racine, func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(p, ".module") {
				return err
			}
			m, openErr := himodule.Open(p)
			if openErr != nil {
				t.Fatalf("module illisible, balayage invalide : %s : %v", p, openErr)
			}
			n++
			for _, f := range m.Files("levl") {
				index[f.GlobalID] = append(index[f.GlobalID], p)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("balayage %s : %v", racine, err)
		}
	}
	if n == 0 {
		t.Fatal("aucun module balaye — l'unicite ne se mesurerait sur rien")
	}
	return index, n
}

// dossiersDistincts reduit des chemins de modules a leurs dossiers de carte, dedupliques
// (le meme dossier existe sous any/ et ds/ : c'est la MEME carte, pas deux occurrences).
func dossiersDistincts(chemins []string) []string {
	var out []string
	vus := map[string]bool{}
	for _, p := range chemins {
		d := filepath.Base(filepath.Dir(p))
		if !vus[d] {
			vus[d] = true
			out = append(out, d)
		}
	}
	return out
}

// moduleAny compose le chemin du module any/ d'une carte (memes noms de dossier que pc/ :
// ridgeline, catalyst).
func moduleAny(t *testing.T, carte string) string {
	t.Helper()
	dir, err := himap.LevelsDir("any")
	if err != nil {
		t.Skipf("installation du jeu introuvable : %v", err)
	}
	p := filepath.Join(dir, carte, carte+"-rtx-new.module")
	if _, err := os.Stat(p); err != nil {
		t.Skipf("module absent (%s)", p)
	}
	return p
}

// estFichier dit si le chemin existe.
func estFichier(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
