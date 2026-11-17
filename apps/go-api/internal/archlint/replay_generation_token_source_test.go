package archlint

// replay_generation_token_source_test.go — le jeton de la génération à la demande d'un rejeu est celui du joueur de la
// demande, et d'aucun autre.
//
// La génération télécharge le film d'un match avec le jeton du profil de la page (admis par la garde de propriété). Les
// fichiers `registry_replay_generation*.go` de `api/wire` obtiennent ce jeton par UN SEUL appel,
// `tokencache.ResolveFreshPlayerTokens(ctx, xuid)`, paramétré par le xuid porté par la demande. Ils ne parcourent jamais
// les profils (`LoadPlayers`, `IteratePool`), n'empruntent le jeton d'aucun autre joueur (`AnyPlayerTokens`, le tirage
// du drain `haloTokensForDrain` / `jetonsCatalogue`, la mise en commun `publicReadTokenSrc` / `ResolveAny`) et ne
// contournent pas le chemin unique par joueur (cache lu à la main, rafraîchissement du magasin appelé directement).
// Les noms se lisent dans l'AST : une mention en commentaire n'est pas un appel.

import (
	"go/ast"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// replayGenerationTokenScope : les fichiers que la règle garde (chemin de garde-rail, préfixe).
const replayGenerationTokenScope = "internal/api/wire/registry_replay_generation"

// replayGenerationTokenFunc : l'unique appel de jeton admis, et le nombre d'occurrences attendu dans le périmètre.
const replayGenerationTokenFunc = "ResolveFreshPlayerTokens"

// replayGenerationForbiddenTokenPaths : ce que le périmètre ne nomme jamais.
var replayGenerationForbiddenTokenPaths = map[string]string{
	"LoadPlayers":                    "parcours des profils suivis",
	"IteratePool":                    "parcours des profils ouverts",
	"AnyPlayerTokens":                "jeton du premier joueur disponible",
	"haloTokensForDrain":             "jeton du premier joueur valide (drain du catalogue)",
	"jetonsCatalogue":                "jeton tiré pour le catalogue",
	"publicReadTokenSrc":             "mise en commun des jetons",
	"ResolveAny":                     "jeton de la mise en commun",
	"RefreshHaloTokensViaStoreFirst": "chemin de rafraîchissement hors du chemin unique par joueur",
	"ResolveMSAccessTokenStoreFirst": "chemin de rafraîchissement hors du chemin unique par joueur",
	"GetCachedPlayerTokens":          "cache lu à la main",
	"SetCachedPlayerTokens":          "cache écrit à la main",
	"RefreshTokensForXUID":           "rafraîchissement hors du chemin unique par joueur",
	"mintFreshTokensForXUID":         "rafraîchissement hors du chemin unique par joueur",
	"InvalidateCachedPlayerTokens":   "cache manipulé à la main",
	"RefreshFromStoreEntry":          "rafraîchissement hors du chemin unique par joueur",
	"NewMultiUserTokenStore":         "second magasin de jetons",
}

// replayGenerationTokenProblems rend les violations de la règle dans les sources du périmètre.
func replayGenerationTokenProblems(sources []goSource) []string {
	var out []string
	calls, scoped := 0, 0
	for _, src := range sources {
		if !strings.HasPrefix(src.Rel, replayGenerationTokenScope) {
			continue
		}
		scoped++
		ast.Inspect(src.File, func(n ast.Node) bool {
			// Un identifiant isolé comme le nom d'un sélecteur (`x.Nom`) est un *ast.Ident : une seule lecture suffit.
			if id, ok := n.(*ast.Ident); ok {
				if why, bad := replayGenerationForbiddenTokenPaths[id.Name]; bad {
					out = append(out, src.Rel+" : "+id.Name+" ("+why+") — le jeton est celui du joueur de la demande, par un seul appel de "+replayGenerationTokenFunc)
				}
			}
			if sel, ok := n.(*ast.SelectorExpr); ok && sel.Sel.Name == replayGenerationTokenFunc {
				calls++
			}
			return true
		})
	}
	switch {
	case scoped == 0:
		out = append(out, "aucun fichier "+replayGenerationTokenScope+"*.go : la garde ne garde rien")
	case calls != 1:
		out = append(out, replayGenerationTokenFunc+" doit être appelée exactement une fois dans le périmètre, trouvée "+
			strconv.Itoa(calls)+" fois")
	}
	sort.Strings(out)
	return out
}

func TestReplayGenerationTokenSource(t *testing.T) {
	sources := goSourcesUnder(t, "internal/api/wire")
	if problems := replayGenerationTokenProblems(sources); len(problems) > 0 {
		t.Errorf("le jeton de la génération à la demande n'est pas celui du seul joueur de la demande :\n  %s",
			strings.Join(problems, "\n  "))
	}
}

// Preuve de détection : un extrait fautif par nom interdit rougit la garde, deux appels ou aucun aussi ; l'extrait
// conforme et un fichier hors périmètre ne rougissent pas.
func TestReplayGenerationTokenSourceDetectsDefects(t *testing.T) {
	const scoped = replayGenerationTokenScope + "_film.go"
	for name := range replayGenerationForbiddenTokenPaths {
		snippet := parseGoSnippet(t, scoped, `package wire
func f() { _ = tokencache.ResolveFreshPlayerTokens; x.`+name+`() }`)
		if got := replayGenerationTokenProblems([]goSource{snippet}); len(got) != 1 || !strings.Contains(got[0], name) {
			t.Errorf("%s non signalé : %v", name, got)
		}
	}
	twice := parseGoSnippet(t, scoped, `package wire
func f() { _ = tokencache.ResolveFreshPlayerTokens; _ = other.ResolveFreshPlayerTokens }`)
	if got := replayGenerationTokenProblems([]goSource{twice}); len(got) != 1 {
		t.Errorf("deux appels non signalés : %v", got)
	}
	none := parseGoSnippet(t, scoped, `package wire
func f() {}`)
	if got := replayGenerationTokenProblems([]goSource{none}); len(got) != 1 {
		t.Errorf("aucun appel non signalé : %v", got)
	}
	clean := parseGoSnippet(t, scoped, `package wire
// LoadPlayers et AnyPlayerTokens cités en commentaire ne comptent pas.
func f() { _ = tokencache.ResolveFreshPlayerTokens }`)
	if got := replayGenerationTokenProblems([]goSource{clean}); len(got) != 0 {
		t.Errorf("faux positif : %v", got)
	}
	outside := parseGoSnippet(t, "internal/api/wire/registry_auth.go", `package wire
func f() { x.AnyPlayerTokens(); x.LoadPlayers() }`)
	if got := replayGenerationTokenProblems([]goSource{clean, outside}); len(got) != 0 {
		t.Errorf("un fichier hors périmètre ne doit pas être gardé : %v", got)
	}
}
