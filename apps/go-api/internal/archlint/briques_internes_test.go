// Package archlint — briques_internes_test.go : la table des briques INTERNES de `apps/go-api`
// (plan D3, `.ai/V8/PLAN_DECOUPLAGE_BRIQUES_D3_2026-10-29.md`, lot 1, items 1.1 et 1.2).
//
// # POURQUOI UNE SECONDE TABLE
//
// `briques_imports_test.go` juge les briques qui sont des MODULES Go sous `libs/` : le compilateur
// leur refuse déjà `levelup/go-api/internal/...`, la table ajoute la liste fermée des briques de
// rang inférieur. La brique « API Halo » (`internal/haloapi/xblauth`, `internal/haloapi/infinite`,
// décision D3-A) reste DANS le module `go-api` : rien dans le compilateur ne l'empêche d'importer
// `domain`, `ctxkeys` ou `observability`. Cette table est ce refus, posé AVANT le premier
// déplacement de fichier (même méthode que D1 et D2).
//
// # LA LIGNE `haloapi`
//
// Elle désigne l'arbre `internal/haloapi` (champ `dossiers`) ET quatre fichiers clients encore
// hors de l'arbre (champ `fichiers`, plan PUBLICATION `.ai/V8/PLAN_PUBLICATION_BRIQUES_2026-10-30.md`,
// lot 1.1) : `games/halo_5/client.go`, `mapcatalog/ugc.go`, `assets/fetcher_gamecms.go`,
// `sync/xbox_client.go`. Un fichier nommé est jugé comme un fichier du périmètre : ses imports sont
// parsés, son propre paquet et les paquets des `dossiers` lui sont permis, le reste est un
// franchissement. Seuls les fichiers nommés sont jugés, pas les autres fichiers de leur paquet.
// L'arbre `internal/haloapi` ne porte aucune tolérance : tout import hors de la liste fermée y est
// rouge. Les tolérances de la table sont celles des quatre fichiers nommés.
//
// # CE QU'ELLE VÉRIFIE
//
// Pour chaque `.go` du périmètre (les dossiers, tests compris, et les fichiers nommés), chaque import
// (PARSÉ, jamais cherché au grep) est :
// de la bibliothèque standard ; un paquet de la brique elle-même (les dossiers du périmètre
// s'importent entre eux, un fichier importe son propre paquet : ce n'est pas un franchissement) ; sous `levelup/go-api/internal/haloapi` ; ou l'un des tiers nommés (`errgroup`,
// `rate`, décision D3-P), seulement sous le sous-arbre qui le porte (`infinite`) : pour un fichier
// nommé, un tiers est toujours un franchissement, de clé son chemin complet. Tout autre import est
// un FRANCHISSEMENT, compté par import visé.
//
// # TOLÉRANCES PAR IMPORT, DATÉES, QUI NE FONT QUE DESCENDRE
//
// Comme la ligne `film` de D2 : une table (import visé -> nombre de fichiers) au
// lieu d'une allowlist par fichier, parce que ce sont les COMPTES qui doivent tomber à zéro au fil
// des lots. Le compte mesuré par le parcours est confronté à la tolérance : plus de fichiers que la
// tolérance -> rouge ; moins -> rouge « baisse à N » (le ratchet se resserre dans le commit qui
// coupe, il ne garde pas de mou) ; tolérance 0 -> rouge « retirer l'entrée » ; import absent de la
// table -> rouge. La table est à zéro à la fin du lot 2 du plan PUBLICATION ; le mécanisme part à la
// clôture de ce plan (modèle D2).
//
// # MUTATIONS QUI DOIVENT LE FAIRE ROUGIR
//
// Abaisser une tolérance de 1 ; ajouter `"levelup/go-api/internal/domain"` à un fichier posé sous
// `internal/haloapi/` ; ajouter `"levelup/go-api/internal/observability/metrics"` à
// `internal/haloapi/infinite/endpoints.go` ; ajouter un import de LevelUp hors table à l'un des
// quatre fichiers nommés.
package archlint

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// prefixeInterneAPI : le préfixe des paquets de `apps/go-api/internal/`, retiré pour écrire la clé
// d'un import visé (`domain`, `platform/netguard`, ...). Les paquets de `libs/` s'écrivent
// `libs/film/decfilm`, les tiers par leur chemin complet.
const prefixeInterneAPI = prefixeModuleAPI + "internal/"

// toleranceImport : le nombre de fichiers (production et tests) du périmètre qui importent encore
// un paquet donné, la date d'inscription et le lot du plan D3 qui le ramène à zéro.
type toleranceImport struct {
	n    int
	date string // AAAA-MM-JJ
	lot  string // lot(s) du plan qui vident l'entrée, avec le motif de la coupe
}

// briqueInterne : une brique dont les paquets vivent dans le module `go-api`.
type briqueInterne struct {
	nom string
	// dossiers : arbres (le répertoire et tout ce qui est dessous, hors `testdata`, `vendor`,
	// `.x`, `_x`), relatifs à la racine du dépôt. Les imports de l'un à l'autre sont internes.
	dossiers []string
	// fichiers : fichiers nommés (relatifs à la racine du dépôt) jugés comme des fichiers du
	// périmètre alors que leur dossier n'en fait pas partie. Seuls ces fichiers sont parsés.
	fichiers []string
	// permis : chemins d'import (paquet exact ou préfixe `x/`) que la brique peut importer.
	permis []string
	// tiers : paquets tiers (chemin exact) permis hors bibliothèque standard, PAR SOUS-ARBRE : clé =
	// dossier (relatif à la racine du dépôt) sous lequel ces tiers sont permis. Un fichier hors de
	// toute clé n'a droit à aucun tiers.
	tiers map[string][]string
	// tolerances : import visé (voir [cleDImport]) -> tolérance.
	tolerances map[string]toleranceImport
}

// cheminsHaloapiPermis : la brique cible et tout ce qui est dessous.
var cheminsHaloapiPermis = []string{prefixeInterneAPI + "haloapi"}

const dossierHaloapi = "apps/go-api/internal/haloapi"

// tiersHaloapi : les seuls tiers de la brique, par sous-paquet. `infinite` télécharge les fragments
// de film en parallèle (`errgroup`) et reçoit un limiteur (`rate`) ; `xblauth` et le paquet parent
// n'ont droit qu'à la bibliothèque standard (contrat de `xblauth/doc.go`).
var tiersHaloapi = map[string][]string{
	dossierHaloapi + "/infinite": {"golang.org/x/sync/errgroup", "golang.org/x/time/rate"},
}

// briquesInternes : la table (règle 6 de CLAUDE.md : UNE table, pas un fichier par brique).
var briquesInternes = []briqueInterne{
	{
		nom:      "haloapi",
		dossiers: []string{dossierHaloapi},
		fichiers: fichiersClientsHaloapi,
		permis:   cheminsHaloapiPermis,
		tiers:    tiersHaloapi,
		// L'arbre `internal/haloapi` n'a aucune tolérance : les entrées ci-dessous sont celles des
		// quatre fichiers clients nommés, comptes mesurés le 2026-11-01.
		tolerances: map[string]toleranceImport{
			"platform/netguard":      {4, dateReleveClients, "lots 2.1 à 2.4 du plan PUBLICATION : la garde de réseau sort du client (Doer gardé fourni par l'adaptateur de LevelUp)"},
			"domain":                 {2, dateReleveClients, "lots 2.2 (ugc.go) et 2.3 (fetcher_gamecms.go) du plan PUBLICATION : les jetons passent en paramètre de la brique"},
			"ctxkeys":                {1, dateReleveClients, "lot 2.3 du plan PUBLICATION : le titre du contexte reste dans la résolution d'hôte de `assets`"},
			"games":                  {1, dateReleveClients, "lot 2.3 du plan PUBLICATION : le singleton d'adresses devient des Endpoints passés à la brique"},
			"platform/auth":          {1, dateReleveClients, "lot 2.4 du plan PUBLICATION : l'en-tête d'autorisation XSTS passe en chaîne, plus d'alias de type"},
			"golang.org/x/time/rate": {1, dateReleveClients, "lot 2.1 du plan PUBLICATION : le limiteur concret devient l'interface Limiter"},
		},
	},
}

// dateReleveClients : date de la mesure des comptes de tolérance des quatre fichiers clients.
const dateReleveClients = "2026-11-01"

// fichiersClientsHaloapi : les quatre clients HTTP Halo/Xbox restés hors de l'arbre `haloapi`, que
// le lot 2 du plan PUBLICATION coupe en protocole (brique) et exploitation (LevelUp).
var fichiersClientsHaloapi = []string{
	"apps/go-api/internal/games/halo_5/client.go",
	"apps/go-api/internal/mapcatalog/ugc.go",
	"apps/go-api/internal/assets/fetcher_gamecms.go",
	"apps/go-api/internal/sync/xbox_client.go",
}

// cleDImport rend la clé d'un import visé dans une table de tolérances : le chemin sans le préfixe
// `levelup/go-api/internal/` pour un paquet du module, `libs/<brique>/<paquet>` pour une brique,
// le chemin complet pour un tiers.
func cleDImport(imp string) string {
	if reste, ok := strings.CutPrefix(imp, prefixeInterneAPI); ok {
		return reste
	}
	if reste, ok := strings.CutPrefix(imp, prefixeModuleAPI); ok {
		return reste
	}
	if reste, ok := strings.CutPrefix(imp, "github.com/JGtm/LevelUp/"); ok {
		return reste
	}
	return imp
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

// importPermis dit si le fichier `rel` (relatif à la racine du dépôt) peut importer `imp` sans que
// ce soit un franchissement : bibliothèque standard, tiers nommé, paquet des dossiers de la brique
// (et son propre paquet), chemin permis.
func (b briqueInterne) importPermis(rel, imp string) bool {
	if estBibliothequeStandard(imp) {
		return true
	}
	for dossier, tiers := range b.tiers {
		if !strings.HasPrefix(rel, dossier+"/") {
			continue
		}
		for _, t := range tiers {
			if imp == t {
				return true
			}
		}
	}
	if !strings.HasPrefix(imp, prefixeModuleAPI) {
		return false
	}
	if imp == cheminImport(path.Dir(rel)) {
		return true // son propre paquet (y compris `x_test` externe)
	}
	for _, d := range b.dossiers {
		if sousArbre(imp, cheminImport(d)) {
			return true
		}
	}
	for _, p := range b.permis {
		if sousArbre(imp, p) {
			return true
		}
	}
	return false
}

// comptesDeFranchissements rend, par import visé (clé de [cleDImport]), le nombre de fichiers qui
// l'importent sans y être autorisés. Fonction pure : l'auto-preuve l'exerce sur des extraits.
func comptesDeFranchissements(b briqueInterne, fichiers []fichierImports) map[string]int {
	comptes := map[string]int{}
	for _, f := range fichiers {
		deja := map[string]bool{}
		for _, imp := range f.imports {
			if deja[imp] || b.importPermis(f.rel, imp) {
				continue
			}
			deja[imp] = true
			comptes[cleDImport(imp)]++
		}
	}
	return comptes
}

// confronterTolerances rend, triés, les écarts entre les comptes mesurés et les tolérances :
// l'import absent de la table, le compte au-dessus de la tolérance, le compte en dessous (la
// tolérance doit descendre avec lui), la tolérance 0 (entrée à retirer).
func confronterTolerances(comptes map[string]int, toleres map[string]toleranceImport) []string {
	var ecarts []string
	for imp, n := range comptes {
		t, ok := toleres[imp]
		if !ok {
			ecarts = append(ecarts, fmt.Sprintf("%s : %d fichier(s), import absent de la table des tolérances", imp, n))
			continue
		}
		if n > t.n {
			ecarts = append(ecarts, fmt.Sprintf("%s : %d fichier(s) > tolérance %d (%s)", imp, n, t.n, t.lot))
		}
	}
	for imp, t := range toleres {
		n := comptes[imp]
		switch {
		case t.n == 0:
			ecarts = append(ecarts, fmt.Sprintf("%s : tolérance 0, retirer l'entrée", imp))
		case n < t.n:
			ecarts = append(ecarts, fmt.Sprintf("%s : baisse à %d (tolérance %d) : abaisser la tolérance, le ratchet ne fait que descendre", imp, n, t.n))
		}
	}
	sort.Strings(ecarts)
	return ecarts
}

var formeDate = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// anomaliesDeTable rend les défauts de forme d'une table de tolérances : date, lot, compte négatif.
func anomaliesDeTable(toleres map[string]toleranceImport) []string {
	var out []string
	for imp, t := range toleres {
		if !formeDate.MatchString(t.date) {
			out = append(out, fmt.Sprintf("%s : date %q, AAAA-MM-JJ attendu", imp, t.date))
		}
		if strings.TrimSpace(t.lot) == "" {
			out = append(out, imp+" : aucun lot du plan ne vide cette entrée")
		}
		if t.n < 0 {
			out = append(out, fmt.Sprintf("%s : tolérance négative (%d)", imp, t.n))
		}
	}
	sort.Strings(out)
	return out
}

// fichiersDeBriqueInterne parse les imports de tous les `.go` du périmètre de `b`. Rend une erreur
// si un dossier déclaré n'existe pas ou ne porte aucun fichier Go : une table qui décrit une
// arborescence disparue ne garde rien.
func fichiersDeBriqueInterne(racine string, b briqueInterne) ([]fichierImports, error) {
	fset := token.NewFileSet()
	var out []fichierImports
	for _, d := range b.dossiers {
		sous, err := sousDossiersGoErr(racine, d)
		if err != nil {
			return nil, fmt.Errorf("brique %s : dossier %s : %w", b.nom, d, err)
		}
		n := 0
		for _, s := range sous {
			fs, err := importsDuDossierErr(fset, racine, s)
			if err != nil {
				return nil, fmt.Errorf("brique %s : %w", b.nom, err)
			}
			n += len(fs)
			out = append(out, fs...)
		}
		if n == 0 {
			return nil, fmt.Errorf("brique %s : aucun fichier Go sous %s", b.nom, d)
		}
	}
	for _, f := range b.fichiers {
		fi, err := importsDuFichierErr(fset, racine, f)
		if err != nil {
			return nil, fmt.Errorf("brique %s : %w", b.nom, err)
		}
		out = append(out, fi)
	}
	return out, nil
}

// importsDuFichierErr parse les imports du fichier `rel` (relatif à la racine du dépôt). Erreur si
// le fichier n'existe pas : un fichier nommé qui a disparu ou changé de nom ne garde plus rien.
func importsDuFichierErr(fset *token.FileSet, racine, rel string) (fichierImports, error) {
	f, err := parser.ParseFile(fset, filepath.Join(racine, filepath.FromSlash(rel)), nil, parser.ImportsOnly)
	if err != nil {
		return fichierImports{}, fmt.Errorf("fichier nommé %s : %w", rel, err)
	}
	fi := fichierImports{rel: rel}
	for _, imp := range f.Imports {
		fi.imports = append(fi.imports, strings.Trim(imp.Path.Value, `"`))
	}
	return fi, nil
}

// sousDossiersGoErr : le répertoire et ses sous-répertoires que l'outil Go ouvre ; erreur si la
// racine n'existe pas (la version à `testing.T` du parcours échoue en fatal, pas en erreur).
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

func TestBriquesInternes_Imports(t *testing.T) {
	racine := workspaceRoot(t)
	for _, b := range briquesInternes {
		fichiers, err := fichiersDeBriqueInterne(racine, b)
		if err != nil {
			t.Fatalf("%v", err)
		}
		if anomalies := anomaliesDeTable(b.tolerances); len(anomalies) > 0 {
			t.Errorf("brique %s : table de tolérances mal formée :\n  %s", b.nom, strings.Join(anomalies, "\n  "))
		}
		if ecarts := confronterTolerances(comptesDeFranchissements(b, fichiers), b.tolerances); len(ecarts) > 0 {
			t.Errorf("brique %s : les imports hors brique ne correspondent pas à la table des tolérances "+
				"(audit découplage, constat G ; plans D3 et PUBLICATION). Couper l'import (la brique reçoit la valeur, "+
				"LevelUp la remplit) ou, si le compte BAISSE, abaisser la tolérance dans le même commit :\n  %s",
				b.nom, strings.Join(ecarts, "\n  "))
		}
	}
}
