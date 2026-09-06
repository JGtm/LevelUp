// Package archlint — no_local_spawn_depart_test.go : ratchet « UNE SEULE DEFINITION DU
// SPAWN DE DEPART » (revue P2 de la phase 7A).
//
// # CE QU'IL EMPECHE
//
// Le predicat « ce spawn est celui de la premiere vie du joueur » a existe en TROIS
// exemplaires — le calcul des grappes, la resolution d'un amas par son identifiant, et le
// filtre qui restreint l'univers. Les trois parcouraient les memes sidecars et cherchaient
// le meme drapeau. A la troisieme copie, la regle du depot impose un helper et un
// garde-rail (CLAUDE.md n 6) : en trois exemplaires, la definition du depart aurait diverge
// de celle du filtre, et un lien `?spawn=` aurait designe des matchs que la grappe ne
// contient pas.
//
// La regle vit desormais dans `service.spawnsDeDepart`. Ce ratchet interdit d'en reecrire
// une seconde ailleurs, sous la forme d'un test direct du drapeau.
package archlint

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// motifSpawnDepart : le test direct du drapeau de premiere vie.
const motifSpawnDepart = "PremiereVie"

// proprietaireDuSpawnDepart : le seul fichier de production qui a le droit de le lire.
//
// Le paquet `sync/replayartifacts` le POSE a la cuisson et `analysis/tactical` le PRODUIT :
// ce sont des producteurs, pas des lecteurs, et ils sont donc hors perimetre. Ce que ce
// ratchet garde, c'est la LECTURE cote service.
const proprietaireDuSpawnDepart = "internal/service/tactical_service_lectures.go"

// perimetreDuRatchetSpawn : on ne balaye que le service — le producteur a ses propres
// raisons de nommer le champ.
const perimetreDuRatchetSpawn = "internal/service/"

// TestUneSeuleDefinitionDuSpawnDeDepart — le ratchet.
//
// SELF-CHECK POSITIF : le proprietaire doit porter le motif, sinon le garde ne verifie plus
// rien (le helper a-t-il ete renomme ou vide ?).
func TestUneSeuleDefinitionDuSpawnDeDepart(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a echoue")
	}
	apiRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))

	var violations []string
	vuChezLeProprietaire := false
	err := filepath.WalkDir(apiRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			switch d.Name() {
			case "vendor", ".git", "node_modules", "tmp":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel := filepath.ToSlash(mustRel(apiRoot, path))
		if !strings.HasPrefix(rel, perimetreDuRatchetSpawn) {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		for i, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "//") {
				continue
			}
			if !strings.Contains(line, motifSpawnDepart) {
				continue
			}
			if rel == proprietaireDuSpawnDepart {
				vuChezLeProprietaire = true
				continue
			}
			violations = append(violations, rel+":"+strconv.Itoa(i+1)+"  "+strings.TrimSpace(line))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("parcours apps/go-api: %v", err)
	}
	if !vuChezLeProprietaire {
		t.Fatalf("le motif %q est introuvable dans %s : le helper a-t-il ete renomme ou "+
			"deplace ? Le garde ne verifie plus rien", motifSpawnDepart, proprietaireDuSpawnDepart)
	}
	if len(violations) > 0 {
		t.Errorf("le spawn de DEPART est relu hors de %s (%d) — appeler spawnsDeDepart : le "+
			"predicat a deja existe en trois exemplaires, et la definition du depart aurait "+
			"fini par diverger de celle du filtre :\n  %s",
			proprietaireDuSpawnDepart, len(violations), strings.Join(violations, "\n  "))
	}
}
