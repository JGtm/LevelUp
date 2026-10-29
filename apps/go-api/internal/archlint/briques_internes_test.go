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
// Au lot 1 elle désignait les dossiers des coupes (`platform/halo`, `sync/haloclient`, et la LISTE
// des fichiers d'authentification de `platform/auth`), avec des tolérances datées par import visé.
// Lot 2 : la liste `platform/auth` est partie dans `xblauth`. Lot 4 (2026-10-29) : `platform/halo` et
// `sync/haloclient` sont SUPPRIMÉS ; il ne reste que `internal/haloapi` (arbre), toutes les tolérances
// sont tombées à zéro et la table de tolérances est vide : tout import hors de la liste fermée est
// rouge. Le mécanisme des dossiers à liste est parti avec le dernier qui s'en servait.
//
// # CE QU'ELLE VÉRIFIE
//
// Pour chaque `.go` du périmètre, tests compris, chaque import (PARSÉ, jamais cherché au grep) est :
// de la bibliothèque standard ; un paquet de la brique elle-même (les dossiers du périmètre
// s'importent entre eux, un fichier importe son propre paquet : ce n'est pas un franchissement) ; sous `levelup/go-api/internal/haloapi` ; ou l'un des tiers nommés (`errgroup`,
// `rate`, décision D3-P). Tout autre import est un FRANCHISSEMENT, compté par import visé.
//
// # TOLÉRANCES PAR IMPORT, DATÉES, QUI NE FONT QUE DESCENDRE
//
// Comme la ligne `film` de D2 : une table (import visé -> nombre de fichiers, tests compris) au
// lieu d'une allowlist par fichier, parce que ce sont les COMPTES qui doivent tomber à zéro au fil
// des lots. Le compte mesuré par le parcours est confronté à la tolérance : plus de fichiers que la
// tolérance -> rouge ; moins -> rouge « baisse à N » (le ratchet se resserre dans le commit qui
// coupe, il ne garde pas de mou) ; tolérance 0 -> rouge « retirer l'entrée » ; import absent de la
// table -> rouge. Table vide depuis le lot 4 ; le mécanisme part à la clôture de D3 (modèle D2).
//
// # MUTATIONS QUI DOIVENT LE FAIRE ROUGIR
//
// Abaisser une tolérance de 1 ; ajouter `"levelup/go-api/internal/domain"` à un fichier posé sous
// `internal/haloapi/` ; ajouter `"levelup/go-api/internal/observability/metrics"` à
// `internal/haloapi/infinite/endpoints.go`.
package archlint

import (
	"fmt"
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
	lot  string // lot(s) du plan D3 qui vident l'entrée, avec le motif de la coupe
}

// briqueInterne : une brique dont les paquets vivent dans le module `go-api`.
type briqueInterne struct {
	nom string
	// dossiers : arbres (le répertoire et tout ce qui est dessous, hors `testdata`, `vendor`,
	// `.x`, `_x`), relatifs à la racine du dépôt. Les imports de l'un à l'autre sont internes.
	dossiers []string
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
		permis:   cheminsHaloapiPermis,
		tiers:    tiersHaloapi,
		// Aucune tolérance : les comptes mesurés le 2026-10-29 au lot 1 (20 imports visés, 105
		// franchissements) sont tombés à zéro avec la suppression de `platform/halo` et
		// `sync/haloclient` au lot 4 (2026-10-29).
	},
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
	return out, nil
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
				"(audit découplage, constat G ; plan D3). Couper l'import (la brique reçoit la valeur, "+
				"LevelUp la remplit) ou, si le compte BAISSE, abaisser la tolérance dans le même commit :\n  %s",
				b.nom, strings.Join(ecarts, "\n  "))
		}
	}
}
