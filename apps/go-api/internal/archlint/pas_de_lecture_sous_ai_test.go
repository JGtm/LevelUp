package archlint

// pas_de_lecture_sous_ai_test.go — aucun fichier Go du module ne construit un chemin sous `.ai`.
//
// `.ai/` est l'espace de travail des agents (plans, journaux, archives de recherche) : il
// tourne, se range, s'archive. Un test ou un outil qui y lit une fixture casse à chaque
// déménagement de document, et la CI le découvre seule (l'archivage du dossier de recherche
// vers `.ai/archive/` a ainsi rougi douze tests). Les fixtures vivent sous `testdata/` du
// paquet qui les lit, ou sous `internal/testfixtures/testdata/` quand plusieurs paquets les
// partagent (`testfixtures.Carte`).
//
// Le garde-rail lit l'AST de TOUS les fichiers `.go` du module (tests et outils compris) et
// compte les littéraux de chaîne qui construisent un chemin sous `.ai` :
//   - un littéral qui commence par `.ai/` ou contient `/.ai/` ;
//   - un littéral égal à `.ai` passé à un `Join` (`filepath.Join(racine, ".ai", ...)`).
//
// Les commentaires ne sont pas lus : la citation d'un document est couverte par
// `TestCheminsAiCitesDansLeCodeExistent`.
//
// RATCHET. `sitesAiAutorises` liste les sites légitimes au compte EXACT, justification d'une
// ligne chacun. Il ne peut que décroître : un site en plus échoue (le chemin est nouveau), un
// site en moins échoue aussi (retirer l'entrée ou baisser le compte dans le même commit).
// Aucune entrée ne se pose sans justification datée.
//
// Mutation qui doit le faire rougir : écrire `filepath.Join(racine, ".ai", "x.json")` dans un
// test d'un paquet hors de la liste (vérifié le 2026-10-27).

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
	"testing"
)

// dossierAi : le nom du dossier de travail, découpé pour que ce fichier ne porte pas lui-même
// le littéral qu'il interdit.
const dossierAi = "." + "ai"

// siteAi : un site autorisé (fichier du module) avec son compte exact de littéraux et la
// raison pour laquelle il lit ou écrit sous `.ai`.
type siteAi struct {
	nombre   int
	pourquoi string
}

// Raisons partagées (une ligne chacune, répétées par fichier dans la table).
const (
	raisonReDump = "dépôt local des dumps du jeu (.ai/re_dump, ignoré par git, hors clone) : le test se déclare absent sans lui"
	raisonSortie = "ÉCRIT sa sortie de recherche sous l'archive : une sortie n'est pas une fixture"
)

// sitesAiAutorises : fichiers relatifs à `apps/go-api`, séparateurs `/`. Entrées posées à la
// création du garde-rail (2026-10-27) ; la liste ne grossit pas.
//
// 2026-10-28 (plan D1, lot 2c, decision utilisateur) : les sondes de retro-ingenierie des briques
// `himap`, `hinavmesh` et `mapvar` sont des instruments `*_research_test.go` ; les chemins
// absolus perimes (`C:\Users\...\Projects\LevelUp\.ai\re_dump`) sont remplaces par UN helper de
// remontee par paquet (`depot_research_test.go`), qui porte seul le litteral du dump. Les
// treize entrees (une par fichier qui lisait le dump) sont retirees, deux helpers ajoutes
// (plafond 27 -> 16).
var sitesAiAutorises = map[string]siteAi{
	// Dumps locaux du jeu (hors clone, jamais versionnés).
	"cmd/mapfond-build/resolution_test.go":                                   {1, "fabrique une arborescence .ai/re_dump factice dans un TempDir, ne lit pas le dépôt"},
	"internal/himap/cartes_forge.go":                                         {2, "constantes DepotVariantesCarte et DepotNavmesh : " + raisonReDump},
	"internal/himap/sonde_forge_research_test.go":                            {4, raisonReDump},
	"internal/himap/sonde_forge_mvar_research_test.go":                       {1, raisonReDump},
	"internal/hinavmesh/depot_research_test.go":                              {1, raisonReDump + " (helper unique des instruments de la brique, lot 2c du plan D1, 2026-10-28)"},
	"internal/games/halo_infinite/film/replay/mapvar/depot_research_test.go": {1, raisonReDump + " (helper unique des instruments de la brique, lot 2c du plan D1, 2026-10-28)"},
	// Sorties de recherche (écriture, pas lecture de fixture).
	"internal/games/halo_infinite/film/internal/grammar/zone_census_test.go": {1, raisonSortie},
	"internal/games/halo_infinite/film/replay/colline_propriete_test.go":     {1, raisonSortie},
	"internal/games/halo_infinite/film/replay/lettres_corpus_test.go":        {1, raisonSortie},
	"internal/games/halo_infinite/film/replay/zone_state_ti13_scan_test.go":  {1, raisonSortie},
	"cmd/engagement-calibrate/main.go":                                       {1, "écrit son rapport de calibration à la racine de .ai (sortie de l'outil)"},
	"cmd/weapon-icons-build/page.go":                                         {1, "réécrit la page de nommage des icônes dans l'archive (sortie de l'outil, pas une entrée)"},
	// Contrôles de cohérence entre documents de .ai.
	"internal/archlint/doc_chemins_ai_test.go":           {4, "ratchet des chemins .ai cités en commentaire : son objet est .ai"},
	"internal/archlint/no_stale_fallback_target_test.go": {1, "contrôle de cohérence entre le registre de replis et le plan du décodeur (accepté, 2026-10-27)"},
	"internal/testfixtures/paths_test.go":                {1, "sentinelle : RepoRoot() doit retrouver la racine, reconnue à son dossier .ai"},
	// Dette connue : fixture versionnée lue sous .ai, à migrer sous testdata/ du paquet.
	"internal/games/halo_infinite/film/internal/facts/objectives/named_test.go": {1, "DETTE (2026-10-27) : lit .ai/refs/TABLE_STATS_STATBORG.tsv, versionnée et non archivée ; à migrer sous testdata/"},
}

// nombreSitesAiAutorisesPlafond : taille maximale de la liste (ratchet). Baisser avec elle.
const nombreSitesAiAutorisesPlafond = 16

// sitesSousAi compte, dans une source Go, les littéraux qui construisent un chemin sous `.ai`.
func sitesSousAi(t *testing.T, nom string, src []byte) int {
	t.Helper()
	f, err := parser.ParseFile(token.NewFileSet(), nom, src, parser.SkipObjectResolution)
	if err != nil {
		t.Errorf("%s : analyse impossible : %v", nom, err)
		return 0
	}
	arguments := map[*ast.BasicLit]bool{} // littéraux directement passés à un Join
	ast.Inspect(f, func(n ast.Node) bool {
		appel, ok := n.(*ast.CallExpr)
		if !ok || !estJoin(appel.Fun) {
			return true
		}
		for _, a := range appel.Args {
			if lit, ok := a.(*ast.BasicLit); ok {
				arguments[lit] = true
			}
		}
		return true
	})
	n := 0
	ast.Inspect(f, func(node ast.Node) bool {
		lit, ok := node.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return true
		}
		valeur, err := strconv.Unquote(lit.Value)
		if err != nil {
			return true
		}
		if strings.HasPrefix(valeur, dossierAi+"/") || strings.Contains(valeur, "/"+dossierAi+"/") ||
			(valeur == dossierAi && arguments[lit]) {
			n++
		}
		return true
	})
	return n
}

// estJoin : l'appelé est `Join` ou `pkg.Join` (filepath, path, ou un alias).
func estJoin(fun ast.Expr) bool {
	switch f := fun.(type) {
	case *ast.SelectorExpr:
		return f.Sel.Name == "Join"
	case *ast.Ident:
		return f.Name == "Join"
	}
	return false
}

// TestAucuneLectureDeFixtureSousAi — chaque fichier Go du module est propre, ou listé au compte
// exact dans `sitesAiAutorises`.
func TestAucuneLectureDeFixtureSousAi(t *testing.T) {
	trouves := map[string]int{}
	vus := 0
	balayerGoTexte(t, func(rel, texte string) {
		vus++
		if n := sitesSousAi(t, rel, []byte(texte)); n > 0 {
			trouves[rel] = n
		}
	})
	if vus < 1000 {
		t.Fatalf("%d fichier(s) Go balayé(s), plancher 1000 — le garde-rail ne voit plus le module", vus)
	}
	for rel, n := range trouves {
		site, ok := sitesAiAutorises[rel]
		switch {
		case !ok:
			t.Errorf("%s construit %d chemin(s) sous %s : une fixture se range sous testdata/ "+
				"(ou testfixtures.Carte), jamais dans l'espace de travail des agents", rel, n, dossierAi)
		case site.nombre != n:
			t.Errorf("%s : %d chemin(s) sous %s, %d autorisé(s) — aligner l'entrée (le ratchet ne "+
				"monte pas)", rel, n, dossierAi, site.nombre)
		}
	}
	for rel, site := range sitesAiAutorises {
		if trouves[rel] == 0 {
			t.Errorf("%s est listé (%s) mais ne construit plus de chemin sous %s — retirer l'entrée",
				rel, site.pourquoi, dossierAi)
		}
	}
	if len(sitesAiAutorises) > nombreSitesAiAutorisesPlafond {
		t.Errorf("%d site(s) autorisé(s), plafond %d — la liste ne grossit pas",
			len(sitesAiAutorises), nombreSitesAiAutorisesPlafond)
	}
}

// TestDetectionDesCheminsSousAi — auto-preuve du détecteur (@AI = le dossier de travail) : il voit les formes interdites et
// laisse passer les littéraux voisins.
func TestDetectionDesCheminsSousAi(t *testing.T) {
	cas := []struct {
		nom string
		src string
		n   int
	}{
		{"prefixe", `package p; var _ = "@AI/x/y.json"`, 1},
		{"milieu", `package p; var _ = "../../@AI/x"`, 1},
		{"join", `package p; import "path/filepath"; var _ = filepath.Join("r", "@AI", "x")`, 1},
		{"racine_seule_hors_join", `package p; var _ = "@AI"`, 0},
		{"domaine", `package p; var _ = "claude.ai/docs"`, 0},
		{"prose", "package p; var _ = \"voir `@AI/x`\"", 0},
		{"commentaire", "package p\n// @AI/x/y.md\nvar _ = 1", 0},
	}
	for _, c := range cas {
		if got := sitesSousAi(t, c.nom+".go", []byte(strings.ReplaceAll(c.src, "@AI", dossierAi))); got != c.n {
			t.Errorf("%s : %d site(s), %d attendu(s)", c.nom, got, c.n)
		}
	}
}
