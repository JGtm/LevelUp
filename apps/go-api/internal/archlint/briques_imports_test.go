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
// # TOLÉRANCES PAR IMPORT (2026-10-29, plan D2 lot 1)
//
// La brique `film` (décodeur de film) est posée AVANT son déplacement, avec une table de comptes
// (`tolerancesFilm`, import visé -> nombre de fichiers, tests compris) au lieu d'une allowlist par
// fichier : elle porte des centaines de franchissements, et c'est leur NOMBRE par import qui doit
// descendre à zéro au fil du lot 2. Un compte qui monte, un import absent de la table, un compte
// qui baisse sans que la table le suive, une entrée à zéro : rouge. La table est retirée à la
// clôture du lot 2. Les auto-preuves et l'exhaustivité des dossiers de `libs/` sont dans
// `briques_garde_test.go`.
//
// Mutation : baisser une valeur de `tolerancesFilm` de 1 (rouge, fichiers et import listés).
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
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
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
	// interne à la brique. Sert aux briques encore logées dans `apps/go-api/internal/` avant leur
	// déplacement sous `libs/` (D2 : le décodeur de film).
	recursif bool
	// permis : les paquets (même forme) d'AUTRES briques, de rang inférieur, que celle-ci peut
	// importer. Tout autre import `levelup/go-api/...` ou `.../libs/...` est une violation.
	permis []string
	// tolerances : import visé (hors brique, hors permis) -> nombre de FICHIERS de la brique,
	// production et tests confondus, qui l'importent. Nil = aucune tolérance : tout import hors
	// brique est une violation. Non nil : un import absent de la table, un compte supérieur à la
	// tolérance (la dette croît) ou inférieur (la dette a baissé et la table ne le dit pas), et
	// toute entrée à zéro sont rouges. Le ratchet ne peut que descendre.
	tolerances map[string]int
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
	{nom: "himap", paquets: paquetsHimap, permis: paquetsGamefiles},
	{
		nom: "film",
		// Lot 1 du plan D2 : le décodeur de film est encore dans le module `apps/go-api`. Sa racine
		// est l'arbre que le lot 3 déplace sous `libs/film`, où cette ligne est réécrite. Depuis
		// l'item 2.5 (2026-10-29) `filmshell` et `equipenergy` vivent dessous : une seule racine.
		paquets:    []string{"apps/go-api/internal/games/halo_infinite/film"},
		recursif:   true,
		permis:     append(append([]string{}, paquetsGamefiles...), paquetsHimap...),
		tolerances: tolerancesFilm,
	},
}

// paquetsHimap : la brique « images de carte ». Elle importe `gamefiles`, rien d'autre.
var paquetsHimap = []string{"libs/himap/himap"}

// tolerancesFilm : par import visé hors de la brique `film`, le nombre de FICHIERS de la brique
// (production et tests confondus, tests `research` compris) qui l'importent. Dette RENDUE VISIBLE
// le 2026-10-29 (plan D2, lot 1), mesurée sur `a5bae50b4` : elle ne peut que descendre (voir
// `brique.tolerances`) et chaque entrée nomme le lot du plan qui la vide. Table retirée à la
// clôture du lot 2 (critère de succès 2 du plan : allowlist VIDE).
var tolerancesFilm = map[string]int{
	// Production (voir le relevé de départ du plan) et tests mêlés : le lot indiqué vide l'entrée.
	"levelup/go-api/internal/testutil":         21, // lot 2.11 : helpers locaux R2 / filmcorpus R1 (tests seuls)
	"levelup/go-api/internal/domain/title":     14, // lots 2.10 (prod en commentaire) et 2.11 (tests)
	"levelup/go-api/internal/games/mappings":   7,  // lots 2.10 et 2.11 (tests)
	"levelup/go-api/internal/games/weapons":    3,  // lot 2.11 : tests R1 -> filmcorpus
	"levelup/go-api/internal/testfixtures":     2,  // lot 2.11 : tests R1 -> filmcorpus
	"levelup/go-api/internal/domain/replaydoc": 2,  // lot 2.11 : tests R1 -> filmcorpus
	"levelup/go-api/internal/domain":           1,  // lot 2.11 : test research
	"levelup/go-api/internal/analysis/digest":  1,  // lot 2.11 : test R1 -> filmcorpus
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
	base := filepath.Join(racine, filepath.FromSlash(dossier))
	var out []string
	err := filepath.WalkDir(base, func(chemin string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		nom := d.Name()
		if chemin != base && (dossiersInvisiblesAuGo[nom] || strings.HasPrefix(nom, ".") || strings.HasPrefix(nom, "_")) {
			return fs.SkipDir
		}
		rel, _ := filepath.Rel(racine, chemin)
		out = append(out, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		t.Fatalf("paquet %s introuvable : %v — la table des briques décrit une arborescence qui n'existe plus", dossier, err)
	}
	return out
}

// importsDuDossier parse les imports des `.go` du seul répertoire `p` (relatif à `racine`).
func importsDuDossier(t *testing.T, fset *token.FileSet, racine, p string) []fichierImports {
	t.Helper()
	dir := filepath.Join(racine, filepath.FromSlash(p))
	entrees, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("paquet %s introuvable : %v — la table des briques décrit une arborescence qui n'existe plus", p, err)
	}
	var out []fichierImports
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
	}
	return out
}

// violationsDeBrique rend, pour une brique, les imports de `levelup/go-api/...` ou de
// `github.com/JGtm/LevelUp/libs/...` qui ne visent ni un paquet de la brique ni un paquet permis.
// Les fichiers de `toleres` sont passés et marqués dans `vus` (un fichier toléré qui n'a plus de
// franchissement n'est PAS marqué : son entrée est morte). Quand la brique porte une table de
// `tolerances`, les franchissements sont comptés par import visé (un fichier compte une fois par
// import) et confrontés à la table par `violationsDeTolerances`. Fonction pure : c'est elle que
// l'auto-preuve exerce.
func violationsDeBrique(b brique, fichiers []fichierImports, toleres map[string]string, vus map[string]bool) []string {
	var violations []string
	hors := map[string][]string{} // import visé -> fichiers qui l'importent
	for _, f := range fichiers {
		deja := map[string]bool{}
		for _, imp := range f.imports {
			surveille := strings.HasPrefix(imp, prefixeModuleAPI) || strings.HasPrefix(imp, prefixeBriques)
			if !surveille || b.interne(imp) || deja[imp] {
				continue
			}
			deja[imp] = true
			if _, ok := toleres[f.rel]; ok {
				vus[f.rel] = true
				continue
			}
			if b.tolerances != nil {
				hors[imp] = append(hors[imp], f.rel)
				continue
			}
			violations = append(violations, "["+b.nom+"] "+f.rel+"  -> "+imp)
		}
	}
	if b.tolerances != nil {
		violations = append(violations, violationsDeTolerances(b, hors)...)
	}
	return violations
}

// violationsDeTolerances confronte les comptes mesurés (import visé -> fichiers) à la table de
// tolérances de la brique : import absent de la table, compte supérieur (liste fichier -> import),
// compte inférieur (la table doit descendre), entrée à zéro (à retirer), entrée sans aucun fichier.
func violationsDeTolerances(b brique, hors map[string][]string) []string {
	var violations []string
	for imp, fichiers := range hors {
		sort.Strings(fichiers)
		tol, ok := b.tolerances[imp]
		switch {
		case !ok:
			for _, f := range fichiers {
				violations = append(violations, "["+b.nom+"] "+f+"  -> "+imp+"  (import absent de la table des tolérances)")
			}
		case len(fichiers) > tol:
			for _, f := range fichiers {
				violations = append(violations, "["+b.nom+"] "+f+"  -> "+imp+
					"  (compte "+strconv.Itoa(len(fichiers))+" > tolérance "+strconv.Itoa(tol)+")")
			}
		case len(fichiers) < tol:
			violations = append(violations, "["+b.nom+"] "+imp+" : compte mesuré "+strconv.Itoa(len(fichiers))+
				" < tolérance "+strconv.Itoa(tol)+" — baisse la tolérance à "+strconv.Itoa(len(fichiers)))
		}
	}
	for imp, tol := range b.tolerances {
		if tol == 0 {
			violations = append(violations, "["+b.nom+"] tolérance à 0 pour "+imp+" — retirer l'entrée de la table")
		} else if _, mesure := hors[imp]; !mesure {
			violations = append(violations, "["+b.nom+"] "+imp+" : compte mesuré 0 < tolérance "+
				strconv.Itoa(tol)+" — retirer l'entrée de la table")
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
