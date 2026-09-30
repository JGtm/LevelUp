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
// `errors.Is` est l usage prevu) et les instruments de recherche (`film/research/`), migres au
// jalon J12.7 avec leur tag de build.
//
// # PERIMETRE
//
// `internal/games/halo_infinite/film/` (hors `research/`), `internal/replaybuild/`,
// `internal/sync/killcollector/`, `cmd/replay-*`. Fichiers non-test seulement.
//
// # L ALLOWLIST ET SA DATE
//
// QUATRE entrees, posees le 2026-09-30 (J12.4) : chacune est lue par du CODE d un fichier
// `*_research_test.go` ou de `film/research/`, que le jalon J12.4 n a pas le droit de toucher
// (ils changent de tag au J12.7). Le critere de retrait est mesurable : des que ces lecteurs
// sont migres, l entree devient un accesseur, et ce test REFUSE une entree qui ne correspond plus
// a une variable (l allowlist ne peut que DECROITRE).
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

// varsExporteesEnAttenteJ127 : « dossier.Nom », avec le lecteur qui les retient.
var varsExporteesEnAttenteJ127 = map[string]string{
	"internal/games/halo_infinite/film/internal/facts/objectives.ModeScoreComponent": "e1911_manches_mesure_research_test.go",
	"internal/games/halo_infinite/film/internal/grammar/weaponv3.KnownWeaponHigh32":  "held_weapon_control_research_test.go",
	"internal/games/halo_infinite/film/internal/grammar.GrenadeTypeIDsByRank":        "grenade_production_research_test.go, film/research/grenadeids",
	"internal/games/halo_infinite/film/internal/grammar.WeaponHitDistanceEdges":      "lot1_sonde_precision_research_test.go",
}

func TestAucuneVarExporteeModifiableDansLeDecodeur(t *testing.T) {
	racine := apiRootDepuisIci(t)
	racines := []string{
		"internal/games/halo_infinite/film",
		"internal/replaybuild",
		"internal/sync/killcollector",
	}
	cmds, _ := filepath.Glob(filepath.Join(racine, "cmd", "replay-*"))
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
		err := filepath.WalkDir(filepath.Join(racine, filepath.FromSlash(r)), func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			rel, _ := filepath.Rel(racine, p)
			rel = filepath.ToSlash(rel)
			if d.IsDir() {
				if rel == "internal/games/halo_infinite/film/research" {
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
	if fichiers == 0 {
		t.Fatal("aucun fichier balaye : le ratchet ne mesure plus rien")
	}
	var fautes []string
	for cle := range trouvees {
		if _, permis := varsExporteesEnAttenteJ127[cle]; !permis {
			fautes = append(fautes, cle)
		}
	}
	for cle := range varsExporteesEnAttenteJ127 {
		if !trouvees[cle] {
			fautes = append(fautes, cle+" : entree d allowlist PERIMEE (la variable n existe plus) — la retirer")
		}
	}
	if len(fautes) > 0 {
		sort.Strings(fautes)
		t.Fatalf("variable(s) de paquet exportee(s) modifiable(s) dans le perimetre du decodeur :\n  %s\n\n"+
			"Une valeur exportee est une constante, une fonction (accesseur rendant une copie) ou un "+
			"champ ; la variable est non exportee (cf. l en-tete).", strings.Join(fautes, "\n  "))
	}
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
