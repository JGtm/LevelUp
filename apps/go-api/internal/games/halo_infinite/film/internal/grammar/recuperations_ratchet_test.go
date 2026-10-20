package grammar

// recuperations_ratchet_test.go — LES RELEVES DE LA COUCHE DE RECUPERATION N ONT QUE DES APPELANTS
// NOMMES (ADR 0037 IR-6 ; lots 2.4 et 2.5 du plan de l etape 2).
//
// # CE QUE CE RATCHET TIENT
//
// L ancrage des records bipedes, les pistes des objets du monde et leurs creations se relevent une
// fois par contexte de film (`recuperations.go`) : chaque lecteur de la cuisson passe par la memoire
// du contexte (`parcourirLesAncresBipedes`, `pistesDeLaBande`, `creationsRelevees`). Un balayage
// neuf qui appellerait la primitive lui-meme referait un parcours bit a bit des trames delta que la
// memoire evite — le nombre de parcours d une cuisson remonterait sans qu aucun test de donnees ne
// le voie, puisque le resultat serait le meme.
//
// Le test releve, par l AST des sources de PRODUCTION du paquet, chaque reference a une primitive
// de releve et sa fonction englobante, et exige que l ensemble soit EXACTEMENT la liste ci-dessous
// — un appelant neuf rougit, un appelant retire aussi (la liste se met a jour et ne peut que
// diminuer). Les sources de test sont hors portee : les instruments et les oracles appellent les
// primitives directement, c est leur role.
//
// MUTATIONS — `var _ = walkDeltaBipedRecords` ajoute a `camo_state.go`, puis
// `var _ = equipCreationWalk.creationA` ajoute a `ground_weapon_creation.go` : ROUGES.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strings"
	"testing"
)

// appelantsDesReleves : LA LISTE NOMMEE, DATEE (2026-10-04). Par primitive de releve, ses
// appelants de production — cle : fichier du paquet + « : » + fonction englobante (« paquet » pour
// une reference hors fonction) — et la raison de chacun.
var appelantsDesReleves = map[string]map[string]string{
	"walkDeltaBipedRecords": {
		"ancres_bipedes.go:ancresBipedes": "le releve de l ancrage bipede du contexte, memorise",
	},
	"walkDeltaBipedPayload": {
		"delta_biped_walk.go:walkDeltaBipedRecords": "le marcheur du film, payload par payload",
		"offline_biped.go:ScanBipedRecords": "l ancrage d un payload sous des parametres autres que ceux du " +
			"contexte (bande des vehicules, options forcees) : un autre releve",
	},
	"releverLesPistes": {
		"pistes_du_monde.go:pistesDeLaBande": "la memoire des pistes, au premier lecteur de la bande",
	},
	"echantillonsDesBandes": {
		"pistes_du_monde.go:releverLesPistes": "la passe sur l union des bandes",
	},
	"scanProjectileRecords": {
		"keyframe_ground_weapons.go:WorldObjectPositionsForBand": "enveloppe hors production, son seul " +
			"consommateur est un instrument de mesure",
	},
	"releverLesCreations": {
		"creations_du_monde.go:releverLesCreationsDe": "la memoire des creations, au premier lecteur sous sa cle",
	},
	"nouvellePasseDesCreations": {
		"creations_du_monde.go:releverLesCreations": "la passe des creations sur le film",
	},
	"creationA": {
		"creations_du_monde.go:payload": "la passe des creations, un curseur par archetype",
		"objets_du_monde_lus.go:creationsDerriereLaMarche": "les records NEW que la marche des trames a lus, " +
			"lus a leur en-tete (2.7.d3) : la marche designe, aucun parcours bit a bit",
	},
	"matchWorldObjectNewHeader": {
		"creations_du_monde.go:payload": "la passe des creations",
		"equipment_creation.go:matchEquipmentNewHeader": "la calibration MPP des poses " +
			"(equipment_creation_width.go), preliminaire joue avant la passe",
	},
}

func TestRelevesDeRecuperation_AppelantsNommes(t *testing.T) {
	vus := map[string]map[string]bool{}
	for _, nom := range sourcesDeProductionDuPaquet(t) {
		af, err := parser.ParseFile(token.NewFileSet(), nom, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("%s : %v", nom, err)
		}
		for _, d := range af.Decls {
			englobante, corps := "paquet", ast.Node(d)
			if fn, ok := d.(*ast.FuncDecl); ok {
				if fn.Body == nil {
					continue
				}
				englobante, corps = fn.Name.Name, fn.Body
			}
			ast.Inspect(corps, func(n ast.Node) bool {
				id, ok := n.(*ast.Ident)
				if !ok {
					return true
				}
				if _, primitive := appelantsDesReleves[id.Name]; primitive {
					if vus[id.Name] == nil {
						vus[id.Name] = map[string]bool{}
					}
					vus[id.Name][nom+":"+englobante] = true
				}
				return true
			})
		}
	}
	for primitive, attendus := range appelantsDesReleves {
		if got, want := clesTriees(vus[primitive]), clesTriees(attendus); got != want {
			t.Errorf("appelants de production de %s :\n  %s\nattendus :\n  %s\nun lecteur de la cuisson passe "+
				"par la memoire du contexte (recuperations.go) ; un appelant retire sort de la liste",
				primitive, strings.ReplaceAll(got, "\n", "\n  "), strings.ReplaceAll(want, "\n", "\n  "))
		}
	}
}

// clesTriees rend les cles de `m`, triees, une par ligne.
func clesTriees[V any](m map[string]V) string {
	cles := make([]string, 0, len(m))
	for k := range m {
		cles = append(cles, k)
	}
	sort.Strings(cles)
	return strings.Join(cles, "\n")
}
