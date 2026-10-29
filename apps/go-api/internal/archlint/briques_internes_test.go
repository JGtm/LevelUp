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
// # LA LIGNE `haloapi` AU LOT 1
//
// Elle désigne les dossiers ACTUELS des coupes, pas encore le dossier cible : `platform/halo`
// (arbre, `duration/` compris), `sync/haloclient`, et — pour `platform/auth`, qui n'est PAS une
// brique mais l'exploitation des jetons — la LISTE des fichiers que D3-L envoie dans `xblauth`,
// avec leurs `_test.go` jumeaux. Le dossier cible `internal/haloapi` (arbre) est déjà dans la
// ligne : tout fichier qui y atterrit est jugé dès son premier commit. Au lot 2 les fichiers
// d'authentification sortent de la liste en partant ; au lot 4 `platform/halo` et `sync/haloclient`
// sortent de la table avec leur suppression. À la clôture il ne reste que `internal/haloapi` et
// la table de tolérances est vide.
//
// Lot 2 fait (2026-10-29) : la liste `platform/auth` est retirée ; le mécanisme des dossiers à
// liste (`fichiers`) reste, exercé par ses auto-preuves, et part avec la table à la clôture.
//
// # CE QU'ELLE VÉRIFIE
//
// Pour chaque `.go` du périmètre, tests compris, chaque import (PARSÉ, jamais cherché au grep) est :
// de la bibliothèque standard ; un paquet de la brique elle-même (les dossiers du périmètre
// s'importent entre eux, un fichier de `platform/auth` importe son propre paquet : ce n'est pas un
// franchissement) ; sous `levelup/go-api/internal/haloapi` ; ou l'un des tiers nommés (`errgroup`,
// `rate`, décision D3-P). Tout autre import est un FRANCHISSEMENT, compté par import visé.
//
// # TOLÉRANCES PAR IMPORT, DATÉES, QUI NE FONT QUE DESCENDRE
//
// Comme la ligne `film` de D2 : une table (import visé -> nombre de fichiers, tests compris) au
// lieu d'une allowlist par fichier, parce que ce sont les COMPTES qui doivent tomber à zéro au fil
// des lots. Le compte mesuré par le parcours est confronté à la tolérance : plus de fichiers que la
// tolérance -> rouge ; moins -> rouge « baisse à N » (le ratchet se resserre dans le commit qui
// coupe, il ne garde pas de mou) ; tolérance 0 -> rouge « retirer l'entrée » ; import absent de la
// table -> rouge. À la clôture de D3 la table est vide ET le mécanisme est retiré (modèle D2).
//
// # MUTATIONS QUI DOIVENT LE FAIRE ROUGIR
//
// Abaisser une tolérance de 1 ; ajouter `"levelup/go-api/internal/domain"` à un fichier posé sous
// `internal/haloapi/` ; ajouter `"levelup/go-api/internal/observability/metrics"` à
// `platform/halo/endpoints.go`.
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
	// fichiers : dossier (relatif à la racine du dépôt) -> noms des seuls fichiers du périmètre.
	// Le dossier lui-même n'est PAS un paquet de la brique : seuls ces fichiers le sont.
	fichiers map[string][]string
	// permis : chemins d'import (paquet exact ou préfixe `x/`) que la brique peut importer.
	permis []string
	// tiers : paquets tiers (chemin exact) permis hors bibliothèque standard.
	tiers []string
	// tolerances : import visé (voir [cleDImport]) -> tolérance.
	tolerances map[string]toleranceImport
}

// cheminsHaloapiPermis : la brique cible et tout ce qui est dessous.
var cheminsHaloapiPermis = []string{prefixeInterneAPI + "haloapi"}

// tiersHaloapi : les seuls tiers de la brique (D3-P, mesurés au lot 0, découverte D3-L0-o).
var tiersHaloapi = []string{"golang.org/x/sync/errgroup", "golang.org/x/time/rate"}

const (
	dossierHaloapi    = "apps/go-api/internal/haloapi"
	dossierPlatHalo   = "apps/go-api/internal/platform/halo"
	dossierHaloclient = "apps/go-api/internal/sync/haloclient"
	dossierPlatAuth   = "apps/go-api/internal/platform/auth"
)

// briquesInternes : la table (règle 6 de CLAUDE.md : UNE table, pas un fichier par brique).
var briquesInternes = []briqueInterne{
	{
		nom:      "haloapi",
		dossiers: []string{dossierHaloapi, dossierPlatHalo, dossierHaloclient},
		permis:   cheminsHaloapiPermis,
		tiers:    tiersHaloapi,
		// Comptes MESURÉS le 2026-10-29 (HEAD b92916328) par ce parcours, tests compris. Le lot
		// qui vide chaque entrée est celui qui SUPPRIME ou DÉPLACE le fichier importeur : les
		// fichiers d'authentification quittent `platform/auth` au lot 2, `platform/halo` et
		// `sync/haloclient` sont supprimés au lot 4 (la coupe elle-même se fait au lot 3, dans la
		// copie que reçoit `internal/haloapi/infinite`). Lot 2 (2026-10-29) : les fichiers d'authentification
		// sont partis dans `internal/haloapi/xblauth`, la liste `platform/auth` a quitté la ligne et la part
		// « auth » de chaque tolérance est tombée à zéro (domain 22 -> 21, domain/title 9 -> 3,
		// platform/netguard 9 -> 6, observability/logging retirée).
		tolerances: map[string]toleranceImport{
			"analysis":                       {1, "2026-10-29", "lot 4 : haloclient/halo_skill.go (analysis.MMRKnown) -> haloadapter"},
			"assetnames":                     {1, "2026-10-29", "lot 4 : halo/asset_name_fetcher.go -> haloadapter"},
			"assets":                         {5, "2026-10-29", "lot 4 : halo/battlepass_details.go et tests (assets.Ref) -> haloadapter"},
			"ctxkeys":                        {13, "2026-10-29", "lot 4 : halo (11) et haloclient (3) : jetons, xuid, slug, langue sortis du ctx par l'adaptateur"},
			"domain":                         {17, "2026-10-29", "lot 4 : halo (18) et haloclient (3) -> haloadapter"},
			"domain/title":                   {3, "2026-10-29", "lot 4 : halo (1) et haloclient (2)"},
			"games":                          {16, "2026-10-29", "lot 4 : halo (7) et haloclient (9) : résolveur d'adresses -> infinite.Endpoints reçues"},
			"games/halo_infinite":            {1, "2026-10-29", "lot 4 : halo/catalog_fetcher.go (AssetFetcher) -> haloadapter"},
			"games/mappings":                 {1, "2026-10-29", "lot 4 : haloclient/halo_client_endpoints_test.go -> test de l'adaptateur"},
			"i18n":                           {3, "2026-10-29", "lot 4 : halo/challenges_details.go et leaderboard_scraper.go -> haloadapter"},
			"libs/film/decfilm":              {2, "2026-10-29", "lot 4 : haloclient (Inflate, Decompresser) -> sync/filmfetch"},
			"libs/film/filmcache":            {4, "2026-10-29", "lot 4 : haloclient/local_film_cache.go et tests -> sync/filmfetch"},
			"libs/film/finalise":             {3, "2026-10-29", "lot 4 : haloclient/halo_client_film.go et tests -> sync/filmfetch"},
			"mapcatalog":                     {1, "2026-10-29", "lot 4 : haloclient/halo_client_mvar.go -> haloadapter"},
			"metadata":                       {1, "2026-10-29", "lot 4 : halo/medal_provider.go (MedalEntry) -> haloadapter"},
			"observability":                  {6, "2026-10-29", "lot 4 : halo (2) et haloclient (4) : compteurs injectés comme crochets"},
			"observability/timing":           {2, "2026-10-29", "lot 4 : halo/privacy_provider.go et tests -> haloadapter"},
			"platform/netguard":              {6, "2026-10-29", "lot 4 : halo (3) et haloclient (3) -> Doer garde par surface (D3-K)"},
			"golang.org/x/sync/singleflight": {1, "2026-10-29", "lot 4 : halo/provider.go -> haloadapter"},
			// Transitoires du lot 4.3 (2026-10-29) : `duration` et le cache de jetons ont quitté l'arbre
			// `platform/halo` ; ses fichiers restants les importent jusqu'à sa suppression au lot 4.4.
			"duration":                 {2, "2026-10-29", "lot 4.4 : halo/compare_provider.go et son test, supprimés"},
			"platform/auth/tokencache": {1, "2026-10-29", "lot 4.4 : halo/auth_retry.go, supprimé"},
		},
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
// (un fichier d'un dossier à liste n'a que SON paquet), chemin permis.
func (b briqueInterne) importPermis(rel, imp string) bool {
	if estBibliothequeStandard(imp) {
		return true
	}
	for _, t := range b.tiers {
		if imp == t {
			return true
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
// si un dossier déclaré n'existe pas, ne porte aucun fichier Go, ou si un fichier nommé est absent :
// une table qui décrit une arborescence disparue ne garde rien.
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
	dossiers := make([]string, 0, len(b.fichiers))
	for d := range b.fichiers {
		dossiers = append(dossiers, d)
	}
	sort.Strings(dossiers)
	for _, d := range dossiers {
		fs, err := importsDuDossierErr(token.NewFileSet(), racine, d)
		if err != nil {
			return nil, fmt.Errorf("brique %s : %w", b.nom, err)
		}
		par := map[string]fichierImports{}
		for _, f := range fs {
			par[path.Base(f.rel)] = f
		}
		for _, nom := range b.fichiers[d] {
			f, ok := par[nom]
			if !ok {
				return nil, fmt.Errorf("brique %s : le fichier %s/%s n'existe plus : la liste décrit une arborescence disparue", b.nom, d, nom)
			}
			out = append(out, f)
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
