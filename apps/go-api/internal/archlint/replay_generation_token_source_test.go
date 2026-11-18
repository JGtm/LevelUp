package archlint

// replay_generation_token_source_test.go — le jeton de la génération à la demande d'un rejeu est celui du joueur de la
// demande, et d'aucun autre.
//
// La génération télécharge le film d'un match avec le jeton du profil de la page (admis par la garde de propriété). Les
// fichiers `registry_replay_generation*.go` de `api/wire` obtiennent ce jeton par UN SEUL appel,
// `tokencache.ResolveFreshPlayerTokens(ctx, xuid)`, paramétré par le xuid porté par la demande ; le filet d'un refus
// d'authentification, `tokencache.RetryWithFreshTokens`, re-dérive ce même jeton par le même chemin.
//
// La règle est une LISTE BLANCHE, lue dans l'AST (une mention en commentaire n'est pas un appel) :
//   - de `platform/auth` et de ses sous-paquets, le périmètre n'importe que `platform/auth` (pour la seule sentinelle
//     `ErrUserTokensNotFound`, aucun appel de fonction) et `platform/auth/tokencache` (pour les deux fonctions
//     ci-dessus) ; tout autre sous-paquet, dont la mise en commun `platform/auth/pool`, est refusé à l'import ;
//   - du magasin de jetons du registre (`authStore`), seule la lecture d'un joueur par son xuid (`Load`) est admise ;
//   - la source de la mise en commun du registre (`publicReadTokenSrc`) et les parcours de profils ou de jetons
//     d'autres joueurs ne se nomment pas.
//
// Une source de jeton qui n'est pas dans cette liste est donc refusée par défaut, sans qu'il faille la nommer.

import (
	"go/ast"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// replayGenerationTokenScope : les fichiers que la règle garde (chemin de garde-rail, préfixe).
const replayGenerationTokenScope = "internal/api/wire/registry_replay_generation"

// replayGenerationAuthPrefix : le préfixe d'import de `platform/auth` et de ses sous-paquets.
const replayGenerationAuthPrefix = "levelup/go-api/internal/platform/auth"

// replayGenerationResolveFunc : l'unique fonction qui rend le jeton d'un joueur.
const replayGenerationResolveFunc = "ResolveFreshPlayerTokens"

// replayGenerationRetryFunc : le filet d'un refus d'authentification, qui re-dérive le jeton du même joueur.
const replayGenerationRetryFunc = "RetryWithFreshTokens"

// replayGenerationAllowedAuthSelectors : par paquet d'authentification importable, les seuls noms qu'on y lit.
var replayGenerationAllowedAuthSelectors = map[string][]string{
	replayGenerationAuthPrefix:                 {"ErrUserTokensNotFound"},
	replayGenerationAuthPrefix + "/tokencache": {replayGenerationResolveFunc, replayGenerationRetryFunc},
}

// replayGenerationRegistryTokenFields : par champ du registre qui porte des jetons ou des profils, les seuls noms qu'on
// y lit (liste vide : le champ ne se nomme pas).
var replayGenerationRegistryTokenFields = map[string][]string{
	"authStore":          {"Load"},
	"publicReadTokenSrc": nil,
}

// replayGenerationForeignSources : ce que le registre sait faire d'AUTRE que lire le jeton d'un joueur donné, et que le
// périmètre ne nomme jamais (parcours des profils, jeton d'un autre joueur, tirage du drain).
var replayGenerationForeignSources = map[string]string{
	"LoadPlayers":        "parcours des profils suivis",
	"IteratePool":        "parcours des profils ouverts",
	"AnyPlayerTokens":    "jeton du premier joueur disponible",
	"haloTokensForDrain": "jeton du premier joueur valide (drain du catalogue)",
	"jetonsCatalogue":    "jeton tiré pour le catalogue",
	"ResolveAny":         "jeton de la mise en commun",
}

// replayGenerationTokenProblems rend les violations de la règle dans les sources du périmètre.
func replayGenerationTokenProblems(sources []goSource) []string {
	var out []string
	resolves, retries, scoped := 0, 0, 0
	for _, src := range sources {
		if !strings.HasPrefix(src.Rel, replayGenerationTokenScope) {
			continue
		}
		scoped++
		aliases, bad := replayGenerationAuthAliases(src)
		out = append(out, bad...)
		ast.Inspect(src.File, func(n ast.Node) bool {
			sel, ok := n.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			out = append(out, replayGenerationSelectorProblems(src.Rel, sel, aliases)...)
			if id, isIdent := sel.X.(*ast.Ident); isIdent && aliases[id.Name] == replayGenerationAuthPrefix+"/tokencache" {
				switch sel.Sel.Name {
				case replayGenerationResolveFunc:
					resolves++
				case replayGenerationRetryFunc:
					retries++
				}
			}
			return true
		})
	}
	switch {
	case scoped == 0:
		out = append(out, "aucun fichier "+replayGenerationTokenScope+"*.go : la garde ne garde rien")
	case resolves != 1:
		out = append(out, "tokencache."+replayGenerationResolveFunc+" doit être appelée exactement une fois dans le périmètre, trouvée "+
			strconv.Itoa(resolves)+" fois")
	case retries != 1:
		out = append(out, "tokencache."+replayGenerationRetryFunc+" doit être référencée exactement une fois dans le périmètre, trouvée "+
			strconv.Itoa(retries)+" fois")
	}
	sort.Strings(out)
	return out
}

// replayGenerationAuthAliases lit les imports d'un fichier : rend, par nom local, le chemin des paquets
// d'authentification importés et les imports refusés (sous-paquet hors liste, import « point »).
func replayGenerationAuthAliases(src goSource) (map[string]string, []string) {
	aliases := map[string]string{}
	var bad []string
	for _, imp := range src.File.Imports {
		path, err := strconv.Unquote(imp.Path.Value)
		if err != nil || !strings.HasPrefix(path, replayGenerationAuthPrefix) {
			continue
		}
		if _, allowed := replayGenerationAllowedAuthSelectors[path]; !allowed {
			bad = append(bad, src.Rel+" : import de "+path+" — hors de la liste blanche des sources de jeton ("+
				"seuls platform/auth pour sa sentinelle et platform/auth/tokencache sont admis)")
			continue
		}
		name := path[strings.LastIndex(path, "/")+1:]
		if imp.Name != nil {
			name = imp.Name.Name
		}
		if name == "." {
			bad = append(bad, src.Rel+" : import point de "+path+" — les noms utilisés ne seraient plus contrôlables")
			continue
		}
		aliases[name] = path
	}
	return aliases, bad
}

// replayGenerationSelectorProblems rend les violations d'un sélecteur `x.Nom` : nom hors liste d'un paquet
// d'authentification, nom hors liste d'un champ du registre porteur de jetons, source étrangère nommée.
func replayGenerationSelectorProblems(rel string, sel *ast.SelectorExpr, aliases map[string]string) []string {
	var out []string
	name := sel.Sel.Name
	if why, foreign := replayGenerationForeignSources[name]; foreign {
		out = append(out, rel+" : "+name+" ("+why+") — le jeton est celui du joueur de la demande, par un seul appel de "+
			replayGenerationResolveFunc)
	}
	if id, ok := sel.X.(*ast.Ident); ok {
		if path, isAuth := aliases[id.Name]; isAuth && !containsName(replayGenerationAllowedAuthSelectors[path], name) {
			out = append(out, rel+" : "+id.Name+"."+name+" — hors de la liste blanche de "+path)
		}
	}
	if inner, ok := sel.X.(*ast.SelectorExpr); ok {
		if allowed, guarded := replayGenerationRegistryTokenFields[inner.Sel.Name]; guarded && !containsName(allowed, name) {
			out = append(out, rel+" : "+inner.Sel.Name+"."+name+" — hors de la liste blanche du champ du registre")
		}
	}
	if _, guarded := replayGenerationRegistryTokenFields[name]; guarded && len(replayGenerationRegistryTokenFields[name]) == 0 {
		out = append(out, rel+" : "+name+" — la source de la mise en commun ne sert jamais la génération à la demande")
	}
	return out
}

// containsName dit si names porte name.
func containsName(names []string, name string) bool {
	for _, n := range names {
		if n == name {
			return true
		}
	}
	return false
}

func TestReplayGenerationTokenSource(t *testing.T) {
	sources := goSourcesUnder(t, "internal/api/wire")
	if problems := replayGenerationTokenProblems(sources); len(problems) > 0 {
		t.Errorf("le jeton de la génération à la demande n'est pas celui du seul joueur de la demande :\n  %s",
			strings.Join(problems, "\n  "))
	}
}

// Preuve de détection : un extrait fautif rougit la garde pour chaque source hors liste blanche (import du pool, appel
// d'une fonction de `platform/auth`, nom hors liste de `tokencache`, champ du registre, source étrangère), deux appels
// ou aucun aussi (pour chacune des deux fonctions) ; l'extrait conforme et un fichier hors périmètre ne rougissent pas.
func TestReplayGenerationTokenSourceDetectsDefects(t *testing.T) {
	const scoped = replayGenerationTokenScope + "_film.go"
	const head = `package wire
import (
	"levelup/go-api/internal/platform/auth"
	"levelup/go-api/internal/platform/auth/tokencache"
`
	const okCall = `_ = tokencache.ResolveFreshPlayerTokens; _ = tokencache.RetryWithFreshTokens; _ = auth.ErrUserTokensNotFound`
	defects := map[string]string{
		"import du pool": head + `"levelup/go-api/internal/platform/auth/pool"
)
func f() { ` + okCall + `; _ = pool.X }`,
		"appel de platform/auth": head + `)
func f() { ` + okCall + `; auth.RefreshHaloTokensViaStoreFirst() }`,
		"cache lu à la main": head + `)
func f() { ` + okCall + `; tokencache.GetCachedPlayerTokens("x") }`,
		"invalidation à la main": head + `)
func f() { ` + okCall + `; tokencache.InvalidateCachedPlayerTokens("x") }`,
		"import point": head + `. "levelup/go-api/internal/platform/auth/pool"
)
func f() { ` + okCall + ` }`,
		"magasin hors Load": head + `)
func f() { ` + okCall + `; r.authStore.Save(nil) }`,
		"source de la mise en commun": head + `)
func f() { ` + okCall + `; _ = r.publicReadTokenSrc }`,
		"parcours des profils": head + `)
func f() { ` + okCall + `; r.profiles.LoadPlayers() }`,
		"tirage du drain": head + `)
func f() { ` + okCall + `; r.haloTokensForDrain() }`,
		"jeton d'un autre joueur": head + `)
func f() { ` + okCall + `; x.AnyPlayerTokens() }`,
		"deux appels": head + `)
func f() { ` + okCall + `; _ = tokencache.ResolveFreshPlayerTokens }`,
		"aucun appel": head + `)
func f() { _ = auth.ErrUserTokensNotFound; _ = tokencache.RetryWithFreshTokens }`,
		"aucun filet": head + `)
func f() { _ = auth.ErrUserTokensNotFound; _ = tokencache.ResolveFreshPlayerTokens }`,
		"deux filets": head + `)
func f() { ` + okCall + `; _ = tokencache.RetryWithFreshTokens }`,
	}
	for name, src := range defects {
		snippet := parseGoSnippet(t, scoped, src)
		if got := replayGenerationTokenProblems([]goSource{snippet}); len(got) == 0 {
			t.Errorf("%s non signalé", name)
		}
	}
	none := parseGoSnippet(t, scoped, `package wire
func f() {}`)
	if got := replayGenerationTokenProblems([]goSource{none}); len(got) != 1 {
		t.Errorf("fichier sans appel non signalé : %v", got)
	}
	clean := parseGoSnippet(t, scoped, head+`)
// LoadPlayers et AnyPlayerTokens cités en commentaire ne comptent pas.
func f() { `+okCall+`; _, _ = r.authStore.Load("x") }`)
	if got := replayGenerationTokenProblems([]goSource{clean}); len(got) != 0 {
		t.Errorf("faux positif : %v", got)
	}
	outside := parseGoSnippet(t, "internal/api/wire/registry_auth.go", `package wire
func f() { x.AnyPlayerTokens(); x.LoadPlayers() }`)
	if got := replayGenerationTokenProblems([]goSource{clean, outside}); len(got) != 0 {
		t.Errorf("un fichier hors périmètre ne doit pas être gardé : %v", got)
	}
}
