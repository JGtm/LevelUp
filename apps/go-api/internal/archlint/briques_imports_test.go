// Package archlint — briques_imports_test.go : une BRIQUE n'importe rien du dépôt hors d'elle-même.
//
// # POURQUOI (2026-10-28, plan `.ai/V8/PLAN_DECOUPLAGE_BRIQUES_D1_2026-10-28.md`, lots 1 et 3)
//
// L'audit `.ai/V8/AUDIT_DECOUPLAGE_BRIQUES_2026-10-27.md` (constat G) montre comment trois briques
// autonomes par nature (images de carte, décodeur de film, API Halo) se sont liées au reste de
// LevelUp : chaque import ajouté était local et raisonnable, et rien ne le refusait. Ce test est
// cette liste fermée. Il a été posé AVANT le premier déplacement de paquet, avec une allowlist
// datée égale aux coupes restantes, qui a fini VIDE (même méthode que `film_layers_deps_test.go`).
//
// Depuis les lots 3 et 4, une brique est un MODULE Go sous `libs/<brique>/` (ADR 0041) : le compilateur
// refuse déjà qu'elle importe `levelup/go-api/internal/...` ; ce test tient en plus la liste
// fermée des briques de rang inférieur qu'elle a le droit d'importer. Les trois briques de l'audit
// sont des modules : `himap` (D1), `film` (D2) et, depuis le lot 3 du plan PUBLICATION
// (`.ai/V8/PLAN_PUBLICATION_BRIQUES_2026-10-30.md`, 2026-11-01), `haloapi` (l'API Halo), qui vivait
// jusque-là dans `apps/go-api/internal/haloapi`, tenue par une table des briques INTERNES retirée le
// même jour (le compilateur la remplace).
//
// # TIERS PAR SOUS-ARBRE (champ `tiers`, ligne `haloapi`)
//
// Une ligne qui porte `tiers` juge aussi les imports HORS DU DÉPÔT : hors bibliothèque standard, un
// fichier n'importe que les tiers nommés pour le sous-arbre qui le contient. Dans un espace de travail,
// le `go.mod` d'une brique ne suffit pas à les borner (la liste de construction est l'union des modules
// du `go.work`). Les autres lignes n'ont pas de `tiers` : leurs tiers ne sont pas jugés ici.
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
// # TOLÉRANCES PAR IMPORT : POSÉES LE 2026-10-29, RETIRÉES LE MÊME JOUR (plan D2, lots 1 et 2)
//
// La brique `film` (décodeur de film) a été posée AVANT son déplacement avec une table de comptes
// (import visé -> nombre de fichiers, tests compris) au lieu d'une allowlist par fichier : elle
// portait 189 franchissements, et c'est leur NOMBRE par import qui devait descendre à zéro au fil
// du lot 2. Elle a fini VIDE à l'item 2.11 et le mécanisme (table, champ `tolerances`, règles de
// confrontation et leurs auto-preuves) est retiré : la ligne `film` se juge comme `gamefiles` et
// `himap`, tout import hors brique est une violation. L'exhaustivité des dossiers de `libs/` est
// dans `briques_garde_test.go`.
//
// # MUTATION QUI DOIT LE FAIRE ROUGIR
//
// Ajouter `"levelup/go-api/internal/analysis"` aux imports de
// `libs/himap/himap/cuisson_forge.go`, ou `"github.com/JGtm/LevelUp/libs/himap/himap"`
// à ceux de `libs/gamefiles/mapvar/mapvar.go`, ou `"golang.org/x/sync/errgroup"` à ceux de
// `libs/haloapi/xblauth/doc.go` (tiers hors de son sous-arbre).
package archlint

import (
	"fmt"
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
	// `libs/himap/himap`). Sans `recursif`, chaque répertoire est un paquet Go exact (un
	// sous-répertoire se déclare).
	paquets []string
	// recursif : chaque entrée de `paquets` désigne un ARBRE (le répertoire et tout ce qui est
	// dessous, hors `testdata`, `vendor`, `.x`, `_x`) ; un import vers un paquet de l'arbre est
	// interne à la brique. Sert aux briques dont le module porte de nombreux paquets (`film`).
	recursif bool
	// permis : les paquets (même forme) d'AUTRES briques, de rang inférieur, que celle-ci peut
	// importer. Tout autre import `levelup/go-api/...` ou `.../libs/...` est une violation.
	permis []string
	// tiers : nil, ou les paquets tiers (chemin exact) permis PAR SOUS-ARBRE : clé = dossier relatif à
	// la racine du dépôt sous lequel ils sont permis. Non nil, tout autre import hors bibliothèque
	// standard et hors dépôt est une violation (voir l'en-tête).
	tiers map[string][]string
}

// franchit dit si l'import `imp` du fichier `rel` (relatif à la racine du dépôt) sort de la brique :
// un paquet du dépôt qui n'est ni de la brique ni permis ; ou, si la brique porte `tiers`, un paquet
// hors bibliothèque standard et hors dépôt qui n'est pas permis pour le sous-arbre du fichier.
func (b brique) franchit(rel, imp string) bool {
	if strings.HasPrefix(imp, prefixeModuleAPI) || strings.HasPrefix(imp, prefixeBriques) {
		return !b.interne(imp)
	}
	return b.tiers != nil && !estBibliothequeStandard(imp) && !b.tiersPermis(rel, imp)
}

// tiersPermis dit si le fichier `rel` (relatif à la racine du dépôt) peut importer le tiers `imp`.
func (b brique) tiersPermis(rel, imp string) bool {
	for dossier, liste := range b.tiers {
		if !strings.HasPrefix(rel, dossier+"/") {
			continue
		}
		for _, t := range liste {
			if imp == t {
				return true
			}
		}
	}
	return false
}

// estBibliothequeStandard : le premier élément du chemin n'a pas de point, et le chemin n'est pas
// un paquet du module (`levelup/...`).
func estBibliothequeStandard(imp string) bool {
	if strings.HasPrefix(imp, prefixeModuleAPI) {
		return false
	}
	premier, _, _ := strings.Cut(imp, "/")
	return !strings.Contains(premier, ".")
}

// sousArbre dit si `imp` est le paquet `base` ou un paquet dessous.
func sousArbre(imp, base string) bool {
	return imp == base || strings.HasPrefix(imp, base+"/")
}

// interne dit si le chemin d'import vise un paquet de la brique ou un paquet permis.
func (b brique) interne(imp string) bool {
	for _, p := range b.paquets {
		base := cheminImport(p)
		if imp == base || (b.recursif && strings.HasPrefix(imp, base+"/")) {
			return true
		}
	}
	for _, p := range b.permis {
		if imp == cheminImport(p) {
			return true
		}
	}
	return false
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

// repertoireDuDepot est l inverse de [cheminImport] : le repertoire, relatif a la racine du depot,
// d un paquet du depot donne par son chemin d import ; faux pour un paquet hors du depot.
func repertoireDuDepot(imp string) (string, bool) {
	if reste, ok := strings.CutPrefix(imp, prefixeModuleAPI); ok {
		return racineAPI + reste, true
	}
	return strings.CutPrefix(imp, "github.com/JGtm/LevelUp/")
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
// Le plan D2 (décodeur de film) et le plan PUBLICATION (API Halo) y ont ajouté leurs lignes.
var briques = []brique{
	{nom: "gamefiles", paquets: paquetsGamefiles},
	{nom: "himap", paquets: paquetsHimap, permis: paquetsGamefiles},
	{
		nom: "film",
		// Le module `libs/film` entier (ADR 0041) : ses paquets sont au premier niveau et sous
		// `internal/`, `research/` ; un arbre plutot qu une liste de paquets.
		paquets:  []string{racineBriqueFilm},
		recursif: true,
		permis:   append(append([]string{}, paquetsGamefiles...), paquetsHimap...),
	},
	{
		nom: "haloapi",
		// Le module `libs/haloapi` entier (ADR 0041 et 0042) : `xblauth`, `infinite`, `halo5`,
		// `xboxlive`, `haloapitest` et le paquet racine. Il n'importe AUCUNE autre brique.
		paquets:  []string{racineBriqueHaloapi},
		recursif: true,
		tiers:    tiersHaloapi,
	},
}

// tiersHaloapi : les seuls tiers de la brique « API Halo », par sous-paquet. `infinite` télécharge les
// fragments de film en parallèle (`errgroup`) et ses tests construisent un limiteur (`rate`) ;
// `xblauth`, `halo5`, `xboxlive`, `haloapitest` et le paquet racine n'ont droit qu'à la bibliothèque
// standard (contrats de leurs `doc.go`).
var tiersHaloapi = map[string][]string{
	racineBriqueHaloapi + "/infinite": {"golang.org/x/sync/errgroup", "golang.org/x/time/rate"},
}

// paquetsHimap : la brique « images de carte ». Elle importe `gamefiles`, rien d'autre.
var paquetsHimap = []string{"libs/himap/himap"}

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
		fichiers := fichiersImportsDeLaBrique(t, racine, b)
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

// fichiersImportsDeLaBrique parse les imports de tous les `.go` (tests compris, quels que soient
// leurs tags de build : `go/parser` ne les évalue pas) de la brique. Échoue si un paquet n'existe
// pas ou ne contient aucun fichier Go : un garde-rail qui ne voit rien ne garde rien.
func fichiersImportsDeLaBrique(t *testing.T, racine string, b brique) []fichierImports {
	t.Helper()
	fset := token.NewFileSet()
	var out []fichierImports
	for _, p := range b.paquets {
		dirs := []string{p}
		if b.recursif {
			dirs = sousDossiersGo(t, racine, p)
		}
		n := 0
		for _, d := range dirs {
			fichiers := importsDuDossier(t, fset, racine, d)
			n += len(fichiers)
			out = append(out, fichiers...)
		}
		if n == 0 {
			t.Fatalf("aucun fichier Go sous %s : le garde-rail ne couvre rien", p)
		}
	}
	return out
}

// sousDossiersGo rend le répertoire donné et tous ses sous-répertoires que l'outil Go ouvre
// (hors `testdata`, `vendor`, `.x`, `_x`), relatifs à `racine`, séparateurs "/".
func sousDossiersGo(t *testing.T, racine, dossier string) []string {
	t.Helper()
	out, err := sousDossiersGoErr(racine, dossier)
	if err != nil {
		t.Fatalf("%v — la table des briques décrit une arborescence qui n'existe plus", err)
	}
	return out
}

// sousDossiersGoErr est la forme à erreur de [sousDossiersGo] : erreur si la racine n'existe pas.
func sousDossiersGoErr(racine, dossier string) ([]string, error) {
	base := filepath.Join(racine, filepath.FromSlash(dossier))
	if st, err := os.Stat(base); err != nil || !st.IsDir() {
		return nil, fmt.Errorf("%s n'existe pas : la table décrit une arborescence disparue", dossier)
	}
	var out []string
	err := filepath.WalkDir(base, func(chemin string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		nom := d.Name()
		if chemin != base && (dossiersInvisiblesAuGo[nom] || strings.HasPrefix(nom, ".") || strings.HasPrefix(nom, "_")) {
			return filepath.SkipDir
		}
		rel, _ := filepath.Rel(racine, chemin)
		out = append(out, filepath.ToSlash(rel))
		return nil
	})
	return out, err
}

// importsDuDossier parse les imports des `.go` du seul répertoire `p` (relatif à `racine`).
func importsDuDossier(t *testing.T, fset *token.FileSet, racine, p string) []fichierImports {
	t.Helper()
	out, err := importsDuDossierErr(fset, racine, p)
	if err != nil {
		t.Fatalf("%v — la table des briques décrit une arborescence qui n'existe plus", err)
	}
	return out
}

// importsDuDossierErr est la forme à erreur de [importsDuDossier] : un dossier déclaré qui n'existe
// pas est refusé.
func importsDuDossierErr(fset *token.FileSet, racine, p string) ([]fichierImports, error) {
	dir := filepath.Join(racine, filepath.FromSlash(p))
	entrees, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("paquet %s introuvable : %w", p, err)
	}
	var out []fichierImports
	for _, e := range entrees {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		chemin := filepath.Join(dir, e.Name())
		f, perr := parser.ParseFile(fset, chemin, nil, parser.ImportsOnly)
		if perr != nil {
			return nil, fmt.Errorf("%s : %w", chemin, perr)
		}
		fi := fichierImports{rel: p + "/" + e.Name()}
		for _, imp := range f.Imports {
			fi.imports = append(fi.imports, strings.Trim(imp.Path.Value, `"`))
		}
		out = append(out, fi)
	}
	return out, nil
}

// violationsDeBrique rend, pour une brique, les imports de `levelup/go-api/...` ou de
// `github.com/JGtm/LevelUp/libs/...` qui ne visent ni un paquet de la brique ni un paquet permis,
// et, si la brique porte `tiers`, les imports tiers hors de la liste du sous-arbre du fichier.
// Les fichiers de `toleres` sont passés et marqués dans `vus` (un fichier toléré qui n'a plus de
// franchissement n'est PAS marqué : son entrée est morte). Fonction pure : c'est elle que
// l'auto-preuve exerce.
func violationsDeBrique(b brique, fichiers []fichierImports, toleres map[string]string, vus map[string]bool) []string {
	var violations []string
	for _, f := range fichiers {
		deja := map[string]bool{}
		for _, imp := range f.imports {
			if deja[imp] || !b.franchit(f.rel, imp) {
				continue
			}
			deja[imp] = true
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

// TestBriquesImports_AutoPreuve_Tiers : une brique qui porte `tiers` refuse un tiers hors de la liste
// du sous-arbre du fichier (le paquet parent et un sous-arbre voisin n'y ont pas droit, un tiers non
// nommé non plus) et laisse passer la bibliothèque standard et les tiers nommés sous leur sous-arbre ;
// une brique sans `tiers` ne juge pas les siens.
func TestBriquesImports_AutoPreuve_Tiers(t *testing.T) {
	const (
		errgroup    = "golang.org/x/sync/errgroup"
		rate        = "golang.org/x/time/rate"
		singleflite = "golang.org/x/sync/singleflight"
	)
	b := brique{nom: "x", paquets: []string{"libs/x"}, recursif: true,
		tiers: map[string][]string{"libs/x/infinite": {errgroup, rate}}}
	fichiers := []fichierImports{
		{rel: "libs/x/infinite/a.go", imports: []string{"fmt", "net/http", errgroup, rate, prefixeBriques + "x"}},
		{rel: "libs/x/b.go", imports: []string{"context", errgroup}},
		{rel: "libs/x/xblauth/c.go", imports: []string{rate}},
		{rel: "libs/x/infinite/d_test.go", imports: []string{singleflite}},
	}
	got := violationsDeBrique(b, fichiers, nil, map[string]bool{})
	sort.Strings(got)
	attendu := []string{
		"[x] libs/x/b.go  -> " + errgroup,
		"[x] libs/x/infinite/d_test.go  -> " + singleflite,
		"[x] libs/x/xblauth/c.go  -> " + rate,
	}
	if strings.Join(got, "\n") != strings.Join(attendu, "\n") {
		t.Fatalf("violations = %v, attendu %v", got, attendu)
	}
	sansTiers := brique{nom: "y", paquets: []string{"libs/x"}, recursif: true}
	if v := violationsDeBrique(sansTiers, fichiers, nil, map[string]bool{}); len(v) != 0 {
		t.Fatalf("une brique sans `tiers` ne juge pas ses tiers ; obtenu %v", v)
	}
	if estBibliothequeStandard(prefixeModuleAPI+"internal/x") || !estBibliothequeStandard("net/http") ||
		estBibliothequeStandard(rate) {
		t.Error("estBibliothequeStandard : stdlib = premier élément sans point et hors module")
	}
}

// TestCheminImport : la dérivation du chemin d'import suit la forme des deux sortes de modules.
func TestCheminImport(t *testing.T) {
	for paquet, attendu := range map[string]string{
		"apps/go-api/internal/i18n": "levelup/go-api/internal/i18n",
		"libs/himap/himap":          "github.com/JGtm/LevelUp/libs/himap/himap",
		"libs/gamefiles/ooz":        "github.com/JGtm/LevelUp/libs/gamefiles/ooz",
		"libs/haloapi/infinite":     "github.com/JGtm/LevelUp/libs/haloapi/infinite",
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
