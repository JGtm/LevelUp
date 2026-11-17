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
// La regle vit desormais dans `tactical.SpawnsDeDepart` (analysis/tactical/spawns_joueurs.go) ; le
// service ne fait que TRADUIRE le sidecar vers les types de la regle (`service.joueurLu`). Ce ratchet
// interdit d'en reecrire une seconde ailleurs dans le service, sous la forme d'un test direct du
// drapeau.
//
// # LE MEME GARDE TIENT LA DEFINITION DE « TOUTES LES VIES »
//
// Le parametre `vies=toutes` ajoute une seconde definition — toutes les apparitions du joueur :
// premiere vie, reapparition, debut de manche — qui vit dans `tactical.ApparitionsDuJoueur`, a cote
// de la premiere et dans le meme fichier. Les trois drapeaux qui la composent sont gardes comme le
// premier : les lire hors de la traduction, c'est reecrire une definition d'apparition que les
// grappes, le filtre et la route ne partageraient plus. La regle de la reapparition adverse
// (`tactical.ObservationsAdversesDuMatch`) vit dans le meme fichier d'analysis.
//
// DEUX PROPRIETAIRES, CHACUN VERIFIE : la traduction cote service (seul fichier de service qui nomme
// les drapeaux) et la regle cote analysis (le fichier qui porte les predicats). Un garde qui ne voit
// plus l'un des deux ne verifie plus rien et echoue.
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

// proprietaireDuSpawnDepart : le seul fichier de service qui a le droit de nommer les drapeaux : la
// traduction du sidecar vers les types de la regle.
//
// Le paquet `sync/replayartifacts` le POSE a la cuisson et `analysis/tactical` le PRODUIT :
// ce sont des producteurs, pas des lecteurs, et ils sont donc hors perimetre. Ce que ce
// ratchet garde, c'est la LECTURE cote service.
const proprietaireDuSpawnDepart = "internal/service/tactical_service_lectures.go"

// proprietaireDeLaRegle : le fichier d'analysis qui porte les predicats « depart », « toutes les vies » et
// « reapparition adverse ». Le garde exige qu'il porte chacun des trois drapeaux et chacune des regles.
const proprietaireDeLaRegle = "internal/analysis/tactical/spawns_joueurs.go"

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
		t.Errorf("une apparition du joueur est relue hors de %s (%d) — appeler tactical.SpawnsDeDepart, "+
			"tactical.ApparitionsDuJoueur ou tactical.SpawnsSelonVies : le predicat a deja existe en trois "+
			"exemplaires, et la definition aurait fini par diverger de celle du filtre :\n  %s",
			proprietaireDuSpawnDepart, len(violations), strings.Join(violations, "\n  "))
	}
}

// TestLaRegleDuSpawnDeDepartVitDansAnalysis — SELF-CHECK POSITIF de l'autre proprietaire : le fichier
// d'analysis lit chacun des trois drapeaux et porte les cinq regles que le service appelle.
func TestLaRegleDuSpawnDeDepartVitDansAnalysis(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a echoue")
	}
	apiRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))
	data, err := os.ReadFile(filepath.Join(apiRoot, filepath.FromSlash(proprietaireDeLaRegle)))
	if err != nil {
		t.Fatalf("la regle des spawns est introuvable (%s) : %v", proprietaireDeLaRegle, err)
	}
	var code []string
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.HasPrefix(strings.TrimSpace(line), "//") {
			code = append(code, line)
		}
	}
	src := strings.Join(code, "\n")
	for _, motif := range motifsDApparition {
		if !motif.MatchString(src) {
			t.Errorf("le motif %q est introuvable dans %s : le garde ne verifie plus la regle", motif, proprietaireDeLaRegle)
		}
	}
	for _, regle := range []string{"func SpawnsDeDepart(", "func ApparitionsDuJoueur(", "func SpawnsSelonVies(",
		"func RoutesAuxVies(", "func ObservationsAdversesDuMatch("} {
		if !strings.Contains(src, regle) {
			t.Errorf("%s ne porte plus %q", proprietaireDeLaRegle, regle)
		}
	}
}
