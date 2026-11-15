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
//
// # LE MEME GARDE TIENT LA DEFINITION DE « TOUTES LES VIES »
//
// Le parametre `vies=toutes` ajoute une seconde definition — toutes les apparitions du joueur :
// premiere vie, reapparition, debut de manche — qui vit dans `service.reapparitionsDuJoueur`,
// a cote de la premiere et dans le meme fichier. Les trois drapeaux qui la composent sont
// gardes comme le premier : les lire hors du proprietaire, c'est reecrire une definition
// d'apparition que les grappes, le filtre et la route ne partageraient plus.
package archlint

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// motifsDApparition : le test direct des drapeaux qui disent « cette piste est une apparition
// du joueur » — la premiere vie, la reapparition apres mort, le debut de manche. Les deux
// derniers se reconnaissent par leur acces de champ (`sp.Reapparition`), ce qui laisse en paix
// les constantes voisines (`TacticalFaceReapparition`) et le compte par source
// (`DebutsDeManche`).
var motifsDApparition = []*regexp.Regexp{
	regexp.MustCompile(`PremiereVie`),
	regexp.MustCompile(`\.Reapparition\b`),
	regexp.MustCompile(`\.DebutDeManche\b`),
}

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
// SELF-CHECK POSITIF : le proprietaire doit porter CHACUN des motifs, sinon le garde ne verifie
// plus rien (un helper a-t-il ete renomme ou vide ?).
func TestUneSeuleDefinitionDuSpawnDeDepart(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a echoue")
	}
	apiRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))

	var violations []string
	vusChezLeProprietaire := make([]bool, len(motifsDApparition))
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
			for k, motif := range motifsDApparition {
				if !motif.MatchString(line) {
					continue
				}
				if rel == proprietaireDuSpawnDepart {
					vusChezLeProprietaire[k] = true
					continue
				}
				violations = append(violations, rel+":"+strconv.Itoa(i+1)+"  "+strings.TrimSpace(line))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("parcours apps/go-api: %v", err)
	}
	for k, vu := range vusChezLeProprietaire {
		if !vu {
			t.Fatalf("le motif %q est introuvable dans %s : un helper a-t-il ete renomme ou "+
				"deplace ? Le garde ne verifie plus rien", motifsDApparition[k], proprietaireDuSpawnDepart)
		}
	}
	if len(violations) > 0 {
		t.Errorf("une apparition du joueur est relue hors de %s (%d) — appeler spawnsDeDepart, "+
			"reapparitionsDuJoueur ou spawnsSelonVies : le predicat a deja existe en trois "+
			"exemplaires, et la definition aurait fini par diverger de celle du filtre :\n  %s",
			proprietaireDuSpawnDepart, len(violations), strings.Join(violations, "\n  "))
	}
}
