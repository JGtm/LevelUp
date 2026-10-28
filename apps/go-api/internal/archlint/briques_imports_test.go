// Package archlint — briques_imports_test.go : une BRIQUE n'importe rien du dépôt hors d'elle-même.
//
// # POURQUOI (2026-10-28, plan `.ai/V8/PLAN_DECOUPLAGE_BRIQUES_D1_2026-10-28.md`, lots 1 et 3)
//
// L'audit `.ai/AUDIT_DECOUPLAGE_BRIQUES_2026-10-27.md` (constat G) montre comment trois briques
// autonomes par nature (images de carte, décodeur de film, API Halo) se sont liées au reste de
// LevelUp : chaque import ajouté était local et raisonnable, et rien ne le refusait. Ce test est
// cette liste fermée. Il a été posé AVANT le premier déplacement de paquet, avec une allowlist
// datée égale aux coupes restantes, qui a fini VIDE (même méthode que `film_layers_deps_test.go`).
//
// Depuis les lots 3 et 4, une brique est un MODULE Go sous `libs/<brique>/` (ADR 0041) : le compilateur
// refuse déjà qu'elle importe `levelup/go-api/internal/...` ; ce test tient en plus la liste
// fermée des briques de rang inférieur qu'elle a le droit d'importer.
//
// # CE QU'IL VÉRIFIE
//
// Une brique = des paquets (répertoires relatifs à la RACINE DU DÉPÔT) + les paquets d'autres
// briques qu'elle a le droit d'importer (ses briques de rang inférieur). Pour chaque `.go` d'un
// paquet de la brique, TESTS COMPRIS (un module Go séparé ne peut pas importer
// `levelup/go-api/internal/...` même depuis un test : la frontière des tests est la même que
// celle de la production), tout import d'un paquet du module `levelup/go-api/...` ou d'une brique
// `github.com/JGtm/LevelUp/libs/...` doit viser un paquet de la brique ou un paquet permis. Les
// imports sont PARSÉS (go/parser, ImportsOnly) : un grep se laisserait prendre aux chemins cités
// en commentaire.
//
// Une entrée d'allowlist que plus aucun fichier n'honore fait rougir le test : une exemption qui
// survit à son site finit par en couvrir un autre.
//
// # MUTATION QUI DOIT LE FAIRE ROUGIR
//
// Ajouter `"levelup/go-api/internal/analysis"` aux imports de
// `libs/himap/himap/cuisson_forge.go`, ou `"github.com/JGtm/LevelUp/libs/himap/himap"`
// à ceux de `libs/gamefiles/mapvar/mapvar.go`.
package archlint

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

const (
	// prefixeModuleAPI : le préfixe d'import de tout paquet du module `apps/go-api`.
	prefixeModuleAPI = "levelup/go-api/"
	// prefixeBriques : le préfixe d'import de tout paquet d'un module sous `libs/` (D1-A).
	prefixeBriques = "github.com/JGtm/LevelUp/libs/"
	// racineAPI : le répertoire du module `go-api`, relatif à la racine du dépôt.
	racineAPI = "apps/go-api/"
)

// brique : un ensemble de paquets autonome, et ce qu'il a le droit d'importer hors de lui.
type brique struct {
	nom string
	// paquets : répertoires relatifs à la racine du dépôt, séparateurs "/" (`libs/gamefiles/ooz`,
	// `libs/himap/himap`). Chaque répertoire est un paquet Go (pas de descente
	// récursive : un sous-répertoire se déclare).
	paquets []string
	// permis : les paquets (même forme) d'AUTRES briques, de rang inférieur, que celle-ci peut
	// importer. Tout autre import `levelup/go-api/...` ou `.../libs/...` est une violation.
	permis []string
}

// cheminImport rend le chemin d'import Go d'un paquet donné par son répertoire (relatif à la
// racine du dépôt) : `apps/go-api/internal/x` -> `levelup/go-api/internal/x`, `libs/b/p` ->
// `github.com/JGtm/LevelUp/libs/b/p`.
func cheminImport(paquet string) string {
	if reste, ok := strings.CutPrefix(paquet, racineAPI); ok {
		return prefixeModuleAPI + reste
	}
	return "github.com/JGtm/LevelUp/" + paquet
}

// paquetsGamefiles : la brique « fichiers du jeu » (lecteurs de modules, décompresseur ooz,
// maillage de navigation, variantes de carte). Rang le plus bas : elle n'importe RIEN du dépôt.
var paquetsGamefiles = []string{
	"libs/gamefiles/himodule",
	"libs/gamefiles/ooz",
	"libs/gamefiles/hinavmesh",
	"libs/gamefiles/mapvar",
}

// briques : la table des briques (règle 6 de CLAUDE.md : UNE table, pas un fichier par brique).
// Les plans D2 (décodeur de film) et D3 (API Halo) y ajoutent leurs lignes.
var briques = []brique{
	{nom: "gamefiles", paquets: paquetsGamefiles},
	{nom: "himap", paquets: []string{"libs/himap/himap"}, permis: paquetsGamefiles},
}

// importsToleres : les fichiers d'une brique qui importent un paquet du dépôt hors de la brique,
// par chemin relatif à la racine du dépôt, avec la DATE d'inscription, l'import visé et le LOT du
// plan qui retire l'entrée. Ce n'est pas un blanc-seing : toute entrée est une coupe à faire.
//
// VIDÉE LE 2026-10-28 (lot 2 du plan D1) : la coupe de production (`himap/uslg.go` →
// `internal/i18n`, décision D1-C : `LexiqueLieux` rend des codes de langue du jeu, l'appelant
// `cmd/mapcallouts-build` les traduit) et les seize fichiers de test couplés à LevelUp (décision
// D1-D : ils déménagent sous `internal/gamefilescorpus/`, ou — quand ils exercent des internes
// de la brique — restent avec une fixture copiée dans le `testdata/` de la brique (carte de
// référence) et des oracles lus par un parseur local sans import ; `mapvar` porte ses `.mvar` et
// `hinavmesh` ses blobs dans leur `testdata/`). La table reste vide : le test échoue sur tout
// nouvel import hors brique, et sur toute entrée ajoutée qui n'est pas utilisée.
var importsToleres = map[string]string{}

// fichierImports : un fichier Go et ses imports, tel que le parcours le livre au contrôle pur.
type fichierImports struct {
	rel     string // relatif à la racine du dépôt, séparateurs "/"
	imports []string
}

func TestBriquesImports(t *testing.T) {
	racine := workspaceRoot(t)
	vus := map[string]bool{}
	var violations []string
	for _, b := range briques {
		fichiers := fichiersImportsDesPaquets(t, racine, b.paquets)
		violations = append(violations, violationsDeBrique(b, fichiers, importsToleres, vus)...)
	}
	for cle, motif := range importsToleres {
		if strings.TrimSpace(motif) == "" {
			t.Errorf("importsToleres : %q sans justification datée", cle)
		}
		if !vus[cle] {
			t.Errorf("importsToleres cite %q, qui n'importe plus rien hors de sa brique — retirer l'entrée", cle)
		}
	}
	if len(violations) > 0 {
		sort.Strings(violations)
		t.Errorf("une brique importe un paquet du dépôt hors d'elle-même (audit découplage, constat G) : "+
			"inverser la dépendance (la brique expose un type ou une interface, LevelUp la remplit) ou "+
			"déplacer le fichier côté LevelUp :\n  %s", strings.Join(violations, "\n  "))
	}
}

// fichiersImportsDesPaquets parse les imports de tous les `.go` (tests compris) des paquets
// donnés (répertoires relatifs à `racine`, la racine du dépôt). Échoue si un paquet n'existe pas
// ou ne contient aucun fichier Go : un garde-rail qui ne voit rien ne garde rien.
func fichiersImportsDesPaquets(t *testing.T, racine string, paquets []string) []fichierImports {
	t.Helper()
	fset := token.NewFileSet()
	var out []fichierImports
	for _, p := range paquets {
		dir := filepath.Join(racine, filepath.FromSlash(p))
		entrees, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("paquet %s introuvable : %v — la table des briques décrit une arborescence qui n'existe plus", p, err)
		}
		n := 0
		for _, e := range entrees {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
				continue
			}
			chemin := filepath.Join(dir, e.Name())
			f, perr := parser.ParseFile(fset, chemin, nil, parser.ImportsOnly)
			if perr != nil {
				t.Fatalf("%s : %v", chemin, perr)
			}
			fi := fichierImports{rel: p + "/" + e.Name()}
			for _, imp := range f.Imports {
				fi.imports = append(fi.imports, strings.Trim(imp.Path.Value, `"`))
			}
			out = append(out, fi)
			n++
		}
		if n == 0 {
			t.Fatalf("aucun fichier Go sous %s : le garde-rail ne couvre rien", p)
		}
	}
	return out
}

// violationsDeBrique rend, pour une brique, les imports de `levelup/go-api/...` ou de
// `github.com/JGtm/LevelUp/libs/...` qui ne visent ni un paquet de la brique ni un paquet permis.
// Les fichiers de `toleres` sont passés et marqués dans `vus` (un fichier toléré qui n'a plus de
// franchissement n'est PAS marqué : son entrée est morte). Fonction pure : c'est elle que
// l'auto-preuve exerce.
func violationsDeBrique(b brique, fichiers []fichierImports, toleres map[string]string, vus map[string]bool) []string {
	interne := map[string]bool{}
	for _, p := range b.paquets {
		interne[cheminImport(p)] = true
	}
	for _, p := range b.permis {
		interne[cheminImport(p)] = true
	}
	var violations []string
	for _, f := range fichiers {
		for _, imp := range f.imports {
			surveille := strings.HasPrefix(imp, prefixeModuleAPI) || strings.HasPrefix(imp, prefixeBriques)
			if !surveille || interne[imp] {
				continue
			}
			if _, ok := toleres[f.rel]; ok {
				vus[f.rel] = true
				continue
			}
			violations = append(violations, "["+b.nom+"] "+f.rel+"  -> "+imp)
		}
	}
	return violations
}

// TestBriquesImports_AutoPreuve : le contrôle pur rougit sur un import hors brique (module
// `go-api` OU autre brique), laisse passer la brique elle-même et ses briques permises, et signale
// une entrée morte.
func TestBriquesImports_AutoPreuve(t *testing.T) {
	b := brique{nom: "x", paquets: []string{"libs/x/p"}, permis: []string{"libs/y/q"}}
	const (
		moi   = "github.com/JGtm/LevelUp/libs/x/p"
		permi = "github.com/JGtm/LevelUp/libs/y/q"
		autre = "github.com/JGtm/LevelUp/libs/z/r"
	)
	fichiers := []fichierImports{
		{rel: "libs/x/p/a.go", imports: []string{"fmt", "golang.org/x/sys/unix", moi, permi}},
		{rel: "libs/x/p/b.go", imports: []string{prefixeModuleAPI + "internal/analysis"}},
		{rel: "libs/x/p/c_test.go", imports: []string{prefixeModuleAPI + "internal/testutil"}},
		{rel: "libs/x/p/d.go", imports: []string{autre}},
	}
	vus := map[string]bool{}
	got := violationsDeBrique(b, fichiers, map[string]string{"libs/x/p/c_test.go": "toléré"}, vus)
	if len(got) != 2 ||
		!strings.Contains(got[0], "libs/x/p/b.go  -> "+prefixeModuleAPI+"internal/analysis") ||
		!strings.Contains(got[1], "libs/x/p/d.go  -> "+autre) {
		t.Fatalf("violations attendues : b.go (module go-api) et d.go (autre brique) ; obtenu %v", got)
	}
	if !vus["libs/x/p/c_test.go"] {
		t.Fatal("le fichier toléré doit être marqué vu")
	}
	// Entrée morte : un fichier toléré sans franchissement n'est pas marqué, le test principal
	// le signale alors comme à retirer.
	vus = map[string]bool{}
	_ = violationsDeBrique(b, fichiers[:1], map[string]string{"libs/x/p/a.go": "toléré"}, vus)
	if vus["libs/x/p/a.go"] {
		t.Fatal("un fichier sans franchissement ne doit pas honorer son entrée d'allowlist")
	}
}

// TestCheminImport : la dérivation du chemin d'import suit la forme des deux sortes de modules.
func TestCheminImport(t *testing.T) {
	for paquet, attendu := range map[string]string{
		"apps/go-api/internal/i18n": "levelup/go-api/internal/i18n",
		"libs/himap/himap":          "github.com/JGtm/LevelUp/libs/himap/himap",
		"libs/gamefiles/ooz":        "github.com/JGtm/LevelUp/libs/gamefiles/ooz",
	} {
		if got := cheminImport(paquet); got != attendu {
			t.Errorf("cheminImport(%q) = %q, attendu %q", paquet, got, attendu)
		}
	}
}

// TestBriquesImports_TableCoherente : les briques permises existent dans la table et ne forment
// pas de cycle de rang (une brique ne permet que des briques déclarées AVANT elle).
func TestBriquesImports_TableCoherente(t *testing.T) {
	declares := map[string]bool{}
	for _, b := range briques {
		for _, p := range b.permis {
			if !declares[p] {
				t.Errorf("brique %s permet %q, qui n'appartient à aucune brique déclarée avant elle (rang)", b.nom, p)
			}
		}
		for _, p := range b.paquets {
			if declares[p] {
				t.Errorf("paquet %q déclaré dans deux briques", p)
			}
			declares[p] = true
		}
	}
}
