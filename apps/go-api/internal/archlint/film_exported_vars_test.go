package archlint

// film_exported_vars_test.go — PLUS DE VARIABLE DE PAQUET EXPORTEE ET MODIFIABLE DANS LE DECODEUR
// (J12.4 du PLAN_SUITE_AUDIT_DECODEUR_FILM_2026-09-25, faiblesse 8). Famille de
// `filmdec_package_vars_test.go`, qui borne l etat de `grammar` : celui-ci borne la SURFACE
// EXPORTEE de tout le perimetre, `replay`, `replaybuild` et `killcollector` compris.
//
// # LA REGLE
//
// Une variable de paquet exportee peut etre ecrasee par n importe quel importeur : une table de
// grammaire, une liste d etapes ou un composant de score ainsi exposes sont modifiables a
// distance, et deux decodages du meme processus se la partageraient. Dans le perimetre, une
// valeur exportee est donc une CONSTANTE, une FONCTION (accesseur rendant une copie ou une
// valeur) ou un CHAMP ; la variable, elle, est non exportee.
//
// EXCLUES DE LA MESURE, par nature et non par allowlist : les erreurs sentinelles (`ErrXxx` =
// `errors.New(...)` / `fmt.Errorf(...)`, ou renvoi `pkg.ErrXxx` de la facade : Go n a pas de `const` d erreur, et le test
// `errors.Is` est l usage prevu).
//
// # PERIMETRE
//
// `libs/film/` (hors `research/`), `internal/replaybuild/`,
// `internal/sync/killcollector/`, `cmd/replay-*`, et depuis le 2026-11-08 (ADR 0041 D7, la regle
// est generale : une variable exportee est modifiable par tout importeur, dans n importe quelle
// brique) `libs/gamefiles`, `libs/haloapi`, `libs/himap`. Fichiers non-test seulement.
//
// # ALLOWLIST
//
// Le decodeur n en a AUCUNE. Les quatre entrees posees le 2026-09-30 (J12.4) sont retirees le meme
// jour : leurs lecteurs `research` sont migres (accesseurs `ModeScoreComponent()`,
// `KnownWeaponHigh32Lookup` / `KnownWeaponHigh32Copie`, `GrenadeTypeIDsByRank()` +
// `GrenadeRankCount`, `WeaponHitDistanceEdges()`).
//
// [varsExporteesAdmises] porte les NEUF variables de `libs/himap` que l extension du 2026-11-08 a
// trouvees : une baseline gelee, datee, par nom. Elle ne s ouvre pas a une variable neuve, et une
// entree dont la variable a disparu se retire (`TestVarsExporteesAdmisesNeSontPasPerimees`).
//
// # LA MUTATION QUI DOIT ROUGIR
//
// Ajouter `var Table = []int{1}` dans un fichier non-test de `replay`, `replaybuild` ou
// `killcollector` : le test nomme la variable.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// varsExporteesAdmises : les variables de paquet exportees que le ratchet tolere, cle
// `dossier.Nom` (dossier relatif a la racine du depot, en slash), valeur la raison. BASELINE DU
// 2026-11-08 (extension aux briques voisines du decodeur) : elle ne recoit aucune entree neuve.
//
// Deux familles, toutes dans `libs/himap/himap`. Les TABLES (`CartesForge`, `TypesVolumesDeMort`,
// `ModulesDesEquipements`, `StylesFond`) sont lues sans etre ecrites : des accesseurs rendant une
// copie les remplaceraient. Les REGLAGES (`LumiereRendu`, `NiveauHautNavmesh`,
// `MargeSolVuDuDessousCarte`, `SeuilSubstitution`, `SeuilCouvertureCarte`) sont poses puis rendus
// par la cuisson d une carte (`CuitCarteForge`), ce qui n est pas sur entre deux cuissons
// concurrentes d un meme processus. Retrait : le jour ou la cuisson passe ses reglages en
// parametre (les cinq reglages) et ou les tables deviennent des accesseurs.
var varsExporteesAdmises = map[string]string{
	"libs/himap/himap.CartesForge":              "table de cartes, lue seule (2026-11-08)",
	"libs/himap/himap.TypesVolumesDeMort":       "table de types de volumes, lue seule (2026-11-08)",
	"libs/himap/himap.ModulesDesEquipements":    "liste de modules, lue seule (2026-11-08)",
	"libs/himap/himap.StylesFond":               "liste de styles, lue seule (2026-11-08)",
	"libs/himap/himap.LumiereRendu":             "reglage de rendu, pose par la cuisson (2026-11-08)",
	"libs/himap/himap.NiveauHautNavmesh":        "reglage de rendu, pose par la cuisson (2026-11-08)",
	"libs/himap/himap.MargeSolVuDuDessousCarte": "reglage de rendu, pose par la cuisson (2026-11-08)",
	"libs/himap/himap.SeuilSubstitution":        "reglage de rendu, pose par la cuisson (2026-11-08)",
	"libs/himap/himap.SeuilCouvertureCarte":     "reglage de rendu, pose par la cuisson (2026-11-08)",
}

func TestAucuneVarExporteeModifiableDansLeDecodeur(t *testing.T) {
	trouvees, fichiers := mesurerVarsExportees(t)
	if fichiers == 0 {
		t.Fatal("aucun fichier balaye : le ratchet ne mesure plus rien")
	}
	var fautes []string
	for cle := range trouvees {
		if _, admise := varsExporteesAdmises[cle]; !admise {
			fautes = append(fautes, cle)
		}
	}
	if len(fautes) > 0 {
		sort.Strings(fautes)
		t.Fatalf("variable(s) de paquet exportee(s) modifiable(s) dans le perimetre du decodeur et des briques :\n  %s\n\n"+
			"Une valeur exportee est une constante, une fonction (accesseur rendant une copie) ou un "+
			"champ ; la variable est non exportee (cf. l en-tete). Mettre la variable dans "+
			"varsExporteesAdmises N EST PAS une reponse : cette baseline est datee et fermee au "+
			"2026-11-08.", strings.Join(fautes, "\n  "))
	}
}

// TestVarsExporteesAdmisesNeSontPasPerimees : une entree de la baseline dont la variable n existe
// plus (supprimee, devenue non exportee, devenue constante) se RETIRE.
func TestVarsExporteesAdmisesNeSontPasPerimees(t *testing.T) {
	trouvees, _ := mesurerVarsExportees(t)
	entrees := make([]string, 0, len(varsExporteesAdmises))
	for cle := range varsExporteesAdmises {
		entrees = append(entrees, cle)
	}
	sort.Strings(entrees)
	for _, cle := range entrees {
		if !trouvees[cle] {
			t.Errorf("%s est dans varsExporteesAdmises mais n est plus une variable exportee "+
				"modifiable : retirer l entree (la variable est rentree dans la regle).", cle)
		}
	}
}

// mesurerVarsExportees rend les variables de paquet exportees (hors erreurs sentinelles) des
// racines du perimetre, cle `dossier.Nom`, et le nombre de fichiers balayes.
func mesurerVarsExportees(t *testing.T) (map[string]bool, int) {
	t.Helper()
	// Chemins relatifs a la racine du depot : la brique, ses lecteurs de LevelUp, les autres briques.
	racine := workspaceRoot(t)
	racines := []string{
		racineBriqueFilm,
		"libs/gamefiles",
		"libs/haloapi",
		"libs/himap",
		"apps/go-api/internal/replaybuild",
		"apps/go-api/internal/sync/killcollector",
	}
	cmds, _ := filepath.Glob(filepath.Join(racine, "apps", "go-api", "cmd", "replay-*"))
	if len(cmds) == 0 {
		t.Fatal("aucun cmd/replay-* trouve : le perimetre du ratchet a bouge")
	}
	for _, c := range cmds {
		rel, _ := filepath.Rel(racine, c)
		racines = append(racines, filepath.ToSlash(rel))
	}
	trouvees := map[string]bool{}
	fichiers := 0
	for _, r := range racines {
		err := filepath.WalkDir(cheminDuDepotOuFatal(t, racine, r), func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			rel := relDuDepotOuFatal(t, racine, p)
			if d.IsDir() {
				if rel == racineBriqueFilm+"/research" {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(d.Name(), ".go") || strings.HasSuffix(d.Name(), "_test.go") {
				return nil
			}
			fichiers++
			f, err := parser.ParseFile(token.NewFileSet(), p, nil, parser.SkipObjectResolution)
			if err != nil {
				return err
			}
			dossier := filepath.ToSlash(filepath.Dir(rel))
			for _, dc := range f.Decls {
				gd, ok := dc.(*ast.GenDecl)
				if !ok || gd.Tok != token.VAR {
					continue
				}
				for _, sp := range gd.Specs {
					vs := sp.(*ast.ValueSpec)
					for i, id := range vs.Names {
						if !id.IsExported() || estSentinelleDErreur(id.Name, vs, i) {
							continue
						}
						trouvees[dossier+"."+id.Name] = true
					}
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("balayage de %s : %v", r, err)
		}
	}
	return trouvees, fichiers
}

// estSentinelleDErreur : `ErrXxx = errors.New(...)` ou `fmt.Errorf(...)`.
func estSentinelleDErreur(nom string, vs *ast.ValueSpec, i int) bool {
	if !strings.HasPrefix(nom, "Err") || i >= len(vs.Values) {
		return false
	}
	if sel, ok := vs.Values[i].(*ast.SelectorExpr); ok { // renvoi d une sentinelle d un paquet interne
		return strings.HasPrefix(sel.Sel.Name, "Err")
	}
	call, ok := vs.Values[i].(*ast.CallExpr)
	if !ok {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	return ok && ((pkg.Name == "errors" && sel.Sel.Name == "New") || (pkg.Name == "fmt" && sel.Sel.Name == "Errorf"))
}
