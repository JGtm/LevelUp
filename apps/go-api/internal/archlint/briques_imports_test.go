// Package archlint — briques_imports_test.go : une BRIQUE n'importe rien du dépôt hors d'elle-même.
//
// # POURQUOI (2026-10-28, plan `.ai/V8/PLAN_DECOUPLAGE_BRIQUES_D1_2026-10-28.md`, lot 1)
//
// L'audit `.ai/AUDIT_DECOUPLAGE_BRIQUES_2026-10-27.md` (constat G) montre comment trois briques
// autonomes par nature (images de carte, décodeur de film, API Halo) se sont liées au reste de
// LevelUp : chaque import ajouté était local et raisonnable, et rien ne le refusait. Ce test est
// cette liste fermée. Il est posé AVANT le premier déplacement de paquet, avec une allowlist
// datée égale aux coupes restantes, qui doit finir VIDE à la clôture du plan (même méthode que
// `film_layers_deps_test.go`).
//
// # CE QU'IL VÉRIFIE
//
// Une brique = des paquets (répertoires relatifs à `apps/go-api/`) + les paquets d'autres
// briques qu'elle a le droit d'importer (ses briques de rang inférieur). Pour chaque `.go` d'un
// paquet de la brique, TESTS COMPRIS (un module Go séparé ne peut pas importer
// `levelup/go-api/internal/...` même depuis un test : la frontière des tests est la même que
// celle de la production), tout import `levelup/go-api/...` doit viser un paquet de la brique ou
// un paquet permis. Les imports sont PARSÉS (go/parser, ImportsOnly) : un grep se laisserait
// prendre aux chemins cités en commentaire.
//
// Une entrée d'allowlist que plus aucun fichier n'honore fait rougir le test : une exemption qui
// survit à son site finit par en couvrir un autre.
//
// # MUTATION QUI DOIT LE FAIRE ROUGIR
//
// Ajouter `"levelup/go-api/internal/analysis"` aux imports de `internal/himap/cuisson_forge.go`.
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

// prefixeModuleAPI : le préfixe d'import de tout paquet du module `apps/go-api`.
const prefixeModuleAPI = "levelup/go-api/"

// brique : un ensemble de paquets autonome, et ce qu'il a le droit d'importer hors de lui.
type brique struct {
	nom string
	// paquets : répertoires relatifs à `apps/go-api/`, séparateurs "/". Chaque répertoire est
	// un paquet Go (pas de descente récursive : un sous-répertoire se déclare).
	paquets []string
	// permis : les paquets (même forme) d'AUTRES briques, de rang inférieur, que celle-ci peut
	// importer. Tout autre import `levelup/go-api/...` est une violation.
	permis []string
}

// paquetsGamefiles : la brique « fichiers du jeu » (lecteurs de modules, décompresseur ooz,
// maillage de navigation, variantes de carte). Rang le plus bas : elle n'importe RIEN du dépôt.
var paquetsGamefiles = []string{
	"internal/himodule",
	"internal/ooz",
	"internal/hinavmesh",
	"internal/games/halo_infinite/film/replay/mapvar",
}

// briques : la table des briques (règle 6 de CLAUDE.md : UNE table, pas un fichier par brique).
// Les plans D2 (décodeur de film) et D3 (API Halo) y ajoutent leurs lignes.
var briques = []brique{
	{nom: "gamefiles", paquets: paquetsGamefiles},
	{nom: "himap", paquets: []string{"internal/himap"}, permis: paquetsGamefiles},
}

// importsToleres : les fichiers d'une brique qui importent un paquet du dépôt hors de la brique,
// par chemin relatif à `apps/go-api/`, avec la DATE d'inscription, l'import visé et le LOT du
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
	rel     string // relatif à apps/go-api, séparateurs "/"
	imports []string
}

func TestBriquesImports(t *testing.T) {
	racine := apiModuleRoot(t)
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
// donnés. Échoue si un paquet n'existe pas ou ne contient aucun fichier Go : un garde-rail qui
// ne voit rien ne garde rien.
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

// violationsDeBrique rend, pour une brique, les imports `levelup/go-api/...` qui ne visent ni un
// paquet de la brique ni un paquet permis. Les fichiers de `toleres` sont passés et marqués
// dans `vus` (un fichier toléré qui n'a plus de franchissement n'est PAS marqué : son entrée
// est morte). Fonction pure : c'est elle que l'auto-preuve exerce.
func violationsDeBrique(b brique, fichiers []fichierImports, toleres map[string]string, vus map[string]bool) []string {
	interne := map[string]bool{}
	for _, p := range b.paquets {
		interne[prefixeModuleAPI+p] = true
	}
	for _, p := range b.permis {
		interne[prefixeModuleAPI+p] = true
	}
	var violations []string
	for _, f := range fichiers {
		for _, imp := range f.imports {
			if !strings.HasPrefix(imp, prefixeModuleAPI) || interne[imp] {
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

// TestBriquesImports_AutoPreuve : le contrôle pur rougit sur un import hors brique, laisse
// passer la brique elle-même et ses briques permises, et signale une entrée morte.
func TestBriquesImports_AutoPreuve(t *testing.T) {
	b := brique{nom: "x", paquets: []string{"internal/x"}, permis: []string{"internal/y"}}
	fichiers := []fichierImports{
		{rel: "internal/x/a.go", imports: []string{"fmt", prefixeModuleAPI + "internal/x", prefixeModuleAPI + "internal/y"}},
		{rel: "internal/x/b.go", imports: []string{prefixeModuleAPI + "internal/analysis"}},
		{rel: "internal/x/c_test.go", imports: []string{prefixeModuleAPI + "internal/testutil"}},
	}
	vus := map[string]bool{}
	got := violationsDeBrique(b, fichiers, map[string]string{"internal/x/c_test.go": "toléré"}, vus)
	if len(got) != 1 || !strings.Contains(got[0], "internal/x/b.go  -> "+prefixeModuleAPI+"internal/analysis") {
		t.Fatalf("violations attendues : b.go seul ; obtenu %v", got)
	}
	if !vus["internal/x/c_test.go"] {
		t.Fatal("le fichier toléré doit être marqué vu")
	}
	// Entrée morte : un fichier toléré sans franchissement n'est pas marqué, le test principal
	// le signale alors comme à retirer.
	vus = map[string]bool{}
	_ = violationsDeBrique(b, fichiers[:1], map[string]string{"internal/x/a.go": "toléré"}, vus)
	if vus["internal/x/a.go"] {
		t.Fatal("un fichier sans franchissement ne doit pas honorer son entrée d'allowlist")
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
