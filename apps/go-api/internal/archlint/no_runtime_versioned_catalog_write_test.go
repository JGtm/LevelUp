// Package archlint — no_runtime_versioned_catalog_write_test.go : LE RUNTIME N'ECRIT PAS UN
// FICHIER SUIVI PAR GIT (constat A0 du registre v2, 2026-09-05).
//
// # Le défaut que ce ratchet ferme
//
// `sync/replayartifacts/mvar_rattrapage.go` complétait le catalogue des socles A L'EXECUTION,
// en écrivant `data/titles/{slug}/reference/map_weapon_pads.json` — un fichier VERSIONNÉ. Deux
// dégâts, un par environnement :
//
//	en LOCAL         le commit 5426e256b (« journal(wip)… sans relecture ») a fait entrer
//	                 +332 lignes de données de référence que personne n'a relues ;
//	en PRODUCTION    `scripts/deploy.sh` fait `git reset --hard origin/main` : chaque
//	                 déploiement aurait effacé, en silence, tout ce que le runtime avait
//	                 rattrapé depuis le précédent.
//
// La correction sépare une ENTRÉE d'une SORTIE : le fichier versionné est produit à la main par
// `cmd/mapopads-build` et relu en revue ; le runtime écrit un OVERLAY non versionné
// (`PathResolver.MapWeaponPadsOverlayPath`, `reference/generated/`, ignoré par git), et
// `replay.LoadMapWeaponPadsMerged` recolle les deux à la lecture, le versionné primant.
//
// # Ce que le ratchet vérifie, et comment
//
// Il ne se contente pas de compter des occurrences : il suit le CHEMIN. Pour chaque fichier Go
// de production, il repère les appels à `…MapWeaponPadsPath(…)`, suit la variable qui en reçoit
// le résultat DANS LA MEME FONCTION, et exige que cette valeur n'atteigne jamais un verbe
// d'écriture (`WriteAtomic`, `os.WriteFile`, `os.Rename`, `AddOverlayEntry`…). Une valeur qui
// part dans un contexte que l'analyse ne sait pas suivre (littéral composite, `return`, champ de
// structure) est refusée elle aussi : un ratchet qui « ne sait pas » doit dire non, sinon il
// donne une assurance qu'il n'a pas.
//
// `cmd/mapopads-build/` est la SEULE exception, et c'est sa raison d'être : cette chaîne de
// fabrication PRODUIT le fichier versionné, à la main, hors serveur.
package archlint

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
)

// chemVersionne : la méthode du PathResolver qui rend le chemin du fichier SUIVI PAR GIT.
const chemVersionne = "MapWeaponPadsPath"

// lecteursAutorises : les seules fonctions qui ont le droit de recevoir ce chemin.
//
// Ce sont des LECTEURS, et la liste est délibérément courte : l'élargir, c'est agrandir la
// surface par laquelle un fichier versionné peut se faire écrire par le runtime. Y ajouter une
// entrée demande de prouver que la fonction ne peut pas écrire.
var lecteursAutorises = map[string]bool{
	"LoadMapWeaponPads":       true, // lecture du seul fichier versionné (CLI, recherche)
	"LoadMapWeaponPadsMerged": true, // lecture fusionnée versionné + overlay (production)
}

// verbesDEcriture : préfixes de noms de fonction qui écrivent, effacent ou déplacent.
//
// Un PREFIXE et pas une liste close : le jour où quelqu'un écrit `SauverCatalogue` ou
// `PersistPads`, le ratchet doit mordre sans qu'on ait eu à le prévoir. Les faux positifs
// possibles (une fonction nommée `Add…` qui ne ferait que lire) coûtent une ligne
// d'explication ; un faux négatif coûte un fichier de référence écrasé en production.
//
// LES VERBES FRANÇAIS SONT DANS LA LISTE, et ce n'est pas décoratif : les fonctions internes de
// ce dépôt s'appellent `ajouterCarteAuCatalogue`, `deposerMvar`, `ecrireCatalogue`. Une liste
// anglaise seule aurait laissé passer exactement le chemin de code que ce ratchet doit garder.
var verbesDEcriture = []string{
	"Write", "Save", "Store", "Persist", "Add", "Append", "Create", "Remove", "Delete",
	"Rename", "Mkdir", "Chmod", "Truncate", "Update", "Put", "Flush", "Commit",
	"ecrire", "Ecrire", "ajouter", "Ajouter", "deposer", "Deposer", "enregistrer",
	"Enregistrer", "sauver", "Sauver", "supprimer", "Supprimer", "remplacer", "Remplacer",
	"publier", "Publier", "renommer", "Renommer",
}

// exceptionsChaineDeFabrication : les dossiers dont le MÉTIER est de produire le fichier
// versionné. Chemin relatif à `apps/go-api`, avec sa justification datée.
var exceptionsChaineDeFabrication = map[string]string{
	"cmd/mapopads-build": "2026-09-05 — LA chaîne de fabrication du catalogue versionné : " +
		"elle le produit à la main, hors serveur, et son résultat passe en revue",
}

func estVerbeDEcriture(nom string) bool {
	for _, v := range verbesDEcriture {
		if strings.HasPrefix(nom, v) {
			return true
		}
	}
	return false
}

// estAppelCheminVersionne dit si l'expression EST l'appel `…MapWeaponPadsPath(…)`.
//
// Le nom de l'appelé est lu par `nomAppele` (no_film_reread_test.go) — le paquet en a déjà un,
// en écrire un second serait la 3e copie que la règle des deux copies interdit.
func estAppelCheminVersionne(n ast.Node) bool {
	appel, ok := n.(*ast.CallExpr)
	return ok && nomAppele(appel.Fun) == chemVersionne
}

// analyserFichier rend les violations d'UN fichier déjà parsé.
//
// Extraite pour que le test de morsure (TestRatchetCatalogueVersionneMord) exerce EXACTEMENT le
// même code que le ratchet, sur des sources synthétiques — un garde-rail dont on ne prouve pas
// la morsure ne prouve rien.
func analyserFichier(fset *token.FileSet, f *ast.File) []string {
	var violations []string
	pos := func(n ast.Node) string { return fmt.Sprintf("%d", fset.Position(n.Pos()).Line) }

	ast.Inspect(f, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			return true
		}
		// 1. Les variables qui reçoivent le chemin versionné dans CETTE fonction, et les
		//    positions de leurs identifiants de DÉCLARATION (à ne pas compter comme usages).
		porteuses := map[string]bool{}
		declarations := map[token.Pos]bool{}
		ast.Inspect(fn.Body, func(m ast.Node) bool {
			aff, ok := m.(*ast.AssignStmt)
			if !ok {
				return true
			}
			for i, rhs := range aff.Rhs {
				if !estAppelCheminVersionne(rhs) || i >= len(aff.Lhs) {
					continue
				}
				id, ok := aff.Lhs[i].(*ast.Ident)
				if !ok {
					violations = append(violations, "ligne "+pos(aff)+
						" : chemin versionné affecté à autre chose qu'une variable simple")
					continue
				}
				porteuses[id.Name] = true
				declarations[id.Pos()] = true
			}
			return true
		})

		// 2. Tout appel qui reçoit le chemin versionné — directement ou par une porteuse.
		vus := map[token.Pos]bool{}
		ast.Inspect(fn.Body, func(m ast.Node) bool {
			appel, ok := m.(*ast.CallExpr)
			if !ok {
				return true
			}
			for _, arg := range appel.Args {
				var quoi string
				switch a := arg.(type) {
				case *ast.CallExpr:
					if estAppelCheminVersionne(a) {
						quoi = chemVersionne + "()"
						vus[a.Pos()] = true
					}
				case *ast.Ident:
					if porteuses[a.Name] && !declarations[a.Pos()] {
						quoi = a.Name
						vus[a.Pos()] = true
					}
				}
				if quoi == "" {
					continue
				}
				appele := nomAppele(appel.Fun)
				if lecteursAutorises[appele] || !estVerbeDEcriture(appele) {
					continue
				}
				violations = append(violations, "ligne "+pos(appel)+" : le chemin VERSIONNÉ ("+
					quoi+") est passé à "+appele+"() — le runtime doit écrire l'overlay "+
					"(PathResolver.MapWeaponPadsOverlayPath), jamais le fichier suivi par git")
			}
			return true
		})

		// 3. Les fuites : un chemin versionné (ou une porteuse) qui part ailleurs que dans un
		//    appel analysé ci-dessus. On ne sait pas où il va — donc on refuse.
		ast.Inspect(fn.Body, func(m ast.Node) bool {
			switch x := m.(type) {
			case *ast.CallExpr:
				if estAppelCheminVersionne(x) && !vus[x.Pos()] {
					// Seul cas licite : `v := …MapWeaponPadsPath(…)`, déjà couvert en 1.
					if !estAffectationSimple(fn.Body, x) {
						violations = append(violations, "ligne "+pos(x)+" : résultat de "+
							chemVersionne+"() utilisé dans un contexte non analysable "+
							"(littéral, retour, champ) — l'affecter à une variable locale")
					}
				}
			case *ast.Ident:
				if porteuses[x.Name] && !declarations[x.Pos()] && !vus[x.Pos()] {
					violations = append(violations, "ligne "+pos(x)+" : la variable "+x.Name+
						" (chemin versionné) sort du périmètre analysable — un ratchet qui ne "+
						"sait pas suivre une valeur doit dire non")
				}
			}
			return true
		})
		return false // une FuncDecl est analysée d'un bloc, pas de descente supplémentaire
	})
	sort.Strings(violations)
	return violations
}

// estAffectationSimple dit si l'appel est le membre droit d'une affectation à un identifiant.
func estAffectationSimple(corps *ast.BlockStmt, appel *ast.CallExpr) bool {
	trouve := false
	ast.Inspect(corps, func(n ast.Node) bool {
		aff, ok := n.(*ast.AssignStmt)
		if !ok {
			return true
		}
		for i, rhs := range aff.Rhs {
			if rhs == ast.Expr(appel) && i < len(aff.Lhs) {
				if _, ok := aff.Lhs[i].(*ast.Ident); ok {
					trouve = true
				}
			}
		}
		return true
	})
	return trouve
}

// racineGoAPI rend `apps/go-api` depuis ce fichier de test.
func racineGoAPI(t *testing.T) string {
	t.Helper()
	_, ici, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	return filepath.Dir(filepath.Dir(filepath.Dir(ici))) // internal/archlint -> internal -> apps/go-api
}

// parcourirSourcesProduction applique `visite` à chaque fichier Go non-test d'`internal/` et de
// `cmd/`, hors exceptions déclarées.
func parcourirSourcesProduction(t *testing.T, racine string, visite func(rel, chemin string)) {
	t.Helper()
	for _, sous := range []string{"internal", "cmd"} {
		err := filepath.WalkDir(filepath.Join(racine, sous), func(chemin string, d os.DirEntry, errMarche error) error {
			if errMarche != nil {
				return errMarche
			}
			if d.IsDir() {
				switch d.Name() {
				case "vendor", ".git", "node_modules", "tmp", "testdata":
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(chemin, ".go") || strings.HasSuffix(chemin, "_test.go") {
				return nil
			}
			rel, _ := filepath.Rel(racine, chemin)
			rel = filepath.ToSlash(rel)
			for prefixe := range exceptionsChaineDeFabrication {
				if strings.HasPrefix(rel, prefixe+"/") {
					return nil
				}
			}
			visite(rel, chemin)
			return nil
		})
		if err != nil {
			t.Fatalf("parcours de %s : %v", sous, err)
		}
	}
}

// TestRuntimeNEcritPasLeCatalogueVersionne — LE RATCHET.
func TestRuntimeNEcritPasLeCatalogueVersionne(t *testing.T) {
	racine := racineGoAPI(t)
	fset := token.NewFileSet()
	var violations []string
	parcourirSourcesProduction(t, racine, func(rel, chemin string) {
		blob, err := os.ReadFile(chemin) //nolint:gosec // chemin issu du parcours du module
		if err != nil {
			t.Fatalf("lecture %s : %v", rel, err)
		}
		if !strings.Contains(string(blob), chemVersionne) {
			return
		}
		f, err := parser.ParseFile(fset, chemin, blob, 0)
		if err != nil {
			t.Fatalf("parse %s : %v", rel, err)
		}
		for _, v := range analyserFichier(fset, f) {
			violations = append(violations, rel+" "+v)
		}
	})
	if len(violations) > 0 {
		t.Errorf("ECRITURE RUNTIME D'UN CATALOGUE VERSIONNE : %d violation(s).\n  - %s\n\n"+
			"Le fichier `data/titles/{slug}/reference/map_weapon_pads.json` est SUIVI PAR GIT : "+
			"un déploiement (`git reset --hard origin/main`) efface ce que le runtime y écrit, "+
			"et un commit local l'avale sans relecture. Le runtime écrit l'OVERLAY "+
			"(PathResolver.MapWeaponPadsOverlayPath, ignoré par git) via "+
			"mapcatalog.AddOverlayEntry ; la fusion se fait à la LECTURE "+
			"(replay.LoadMapWeaponPadsMerged).",
			len(violations), strings.Join(violations, "\n  - "))
	}
}

// TestCatalogueVersionneNommeParLeResolverSeul — LE CONTOURNEMENT PAR LE LITTÉRAL.
//
// Le ratchet ci-dessus suit `MapWeaponPadsPath`. Il serait aveugle à un
// `filepath.Join(dir, "map_weapon_pads.json")` écrit à la main. Le nom du fichier n'a donc le
// droit d'apparaître QUE là où le resolver le définit.
func TestCatalogueVersionneNommeParLeResolverSeul(t *testing.T) {
	racine := racineGoAPI(t)
	const literal = `"map_weapon_pads.json"`
	autorise := map[string]bool{
		"internal/domain/title/registry.go": true, // LA définition des deux chemins
	}
	var violations []string
	parcourirSourcesProduction(t, racine, func(rel, chemin string) {
		if autorise[rel] {
			return
		}
		blob, err := os.ReadFile(chemin) //nolint:gosec // chemin issu du parcours du module
		if err != nil {
			t.Fatalf("lecture %s : %v", rel, err)
		}
		for i, ligne := range strings.Split(string(blob), "\n") {
			nu := strings.TrimSpace(ligne)
			if strings.HasPrefix(nu, "//") || strings.HasPrefix(nu, "*") {
				continue
			}
			if strings.Contains(ligne, literal) {
				violations = append(violations, fmt.Sprintf("%s:%d  %s", rel, i+1, nu))
			}
		}
	})
	if len(violations) > 0 {
		t.Errorf("le nom du catalogue des socles est écrit en dur hors du PathResolver : %d.\n"+
			"  - %s\n\nUtiliser PathResolver.MapWeaponPadsPath (lecture) ou "+
			"MapWeaponPadsOverlayPath (écriture runtime) — un chemin construit à la main "+
			"contourne le ratchet ci-dessus.",
			len(violations), strings.Join(violations, "\n  - "))
	}
}

// TestRatchetCatalogueVersionneMord — LA PREUVE QUE LE RATCHET MORD.
//
// Même doctrine que TestBareBulkUpdateDetection_Sanity (internal/sync) : un garde-rail qui ne
// détecte jamais rien est inutile. Les cas REFUSÉS sont les trois formes que prenait le défaut ;
// les cas ACCEPTÉS sont exactement ce que la production fait aujourd'hui — sans eux, le ratchet
// pourrait être « vert par excès de sévérité », c'est-à-dire faux.
func TestRatchetCatalogueVersionneMord(t *testing.T) {
	refuses := map[string]string{
		"écriture directe du chemin versionné": `package p
func f(res R) { mapcatalog.WriteAtomic(cat, res.MapWeaponPadsPath(slug)) }`,
		"écriture via une variable porteuse": `package p
func f(res R) {
	chemin := res.MapWeaponPadsPath(slug)
	_ = mapcatalog.AddOverlayEntry(chemin, slug, id, entry)
}`,
		"os.WriteFile sur le chemin versionné": `package p
func f(res R) {
	chemin := res.MapWeaponPadsPath(slug)
	_ = os.WriteFile(chemin, blob, 0o600)
}`,
		"chemin versionné confié à un littéral composite": `package p
func f(res R) { _ = Config{Sortie: res.MapWeaponPadsPath(slug)} }`,
		// LE DEFAUT EXACT D'AVANT LA CORRECTION : le chemin versionné passé à la fonction
		// interne qui ajoute la carte. Verbe FRANÇAIS — sans les verbes français dans la liste,
		// le ratchet aurait été vert sur le code même qui a causé le constat A0.
		"chemin versionné passé à une fonction interne d'ajout": `package p
func f(res R, d Deps) {
	catPath := res.MapWeaponPadsPath(d.TitleSlug)
	_, _ = replay.LoadMapWeaponPadsMerged(catPath, res.MapWeaponPadsOverlayPath(d.TitleSlug))
	ajouterCarteAuCatalogue(ctx, d, fetcher, catPath, mapID, e)
}`,
	}
	for nom, src := range refuses {
		t.Run("refusé/"+nom, func(t *testing.T) {
			if v := analyserSource(t, src); len(v) == 0 {
				t.Errorf("le ratchet N'A RIEN VU — il est aveugle sur : %s", nom)
			}
		})
	}

	acceptes := map[string]string{
		"lecture fusionnée, écriture de l'overlay (la production d'aujourd'hui)": `package p
func f(res R, d Deps) {
	catPath := res.MapWeaponPadsPath(d.TitleSlug)
	overlayPath := res.MapWeaponPadsOverlayPath(d.TitleSlug)
	cat, err := replay.LoadMapWeaponPadsMerged(catPath, overlayPath)
	if err != nil {
		slog.WarnContext(ctx, "illisible", "err", err, "path", catPath)
		return
	}
	_ = mapcatalog.AddOverlayEntry(overlayPath, d.TitleSlug, id, entry)
	_ = cat
}`,
		"lecture simple du fichier versionné": `package p
func f(res R) { cat, _ := replay.LoadMapWeaponPads(res.MapWeaponPadsPath(slug)); _ = cat }`,
		"écriture d'un AUTRE fichier dans la même fonction": `package p
func f(res R, d Deps) {
	catPath := res.MapWeaponPadsPath(d.TitleSlug)
	_, _ = replay.LoadMapWeaponPadsMerged(catPath, res.MapWeaponPadsOverlayPath(d.TitleSlug))
	_ = os.WriteFile(filepath.Join(d.CacheRoot, "mvar", base), blob, 0o600)
}`,
	}
	for nom, src := range acceptes {
		t.Run("accepté/"+nom, func(t *testing.T) {
			if v := analyserSource(t, src); len(v) > 0 {
				t.Errorf("FAUX POSITIF sur %s :\n  - %s", nom, strings.Join(v, "\n  - "))
			}
		})
	}
}

func analyserSource(t *testing.T, src string) []string {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "synthetique.go", src, 0)
	if err != nil {
		t.Fatalf("parse de la source synthétique : %v", err)
	}
	return analyserFichier(fset, f)
}
