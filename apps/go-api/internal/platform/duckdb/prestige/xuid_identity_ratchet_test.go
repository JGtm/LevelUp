// Package prestige — xuid_identity_ratchet_test.go : GARDE-RAIL « une requête sur
// match_participants.xuid ne se nourrit JAMAIS d'une identité applicative ».
//
// **Le défaut qu'il ferme (2026-07-26).** Deux identités distinctes coexistent :
//
//	player_slug — identité APPLICATIVE du joueur ("JGtm", "Chocoboflor",
//	              "demo-player"). Clé des données Prestige : arc.user_id,
//	              challenge.user_id, prestige_events.user_id.
//	xuid        — identité XBOX ("2533274823110022"). Clé des données de MATCH :
//	              match_participants.xuid, medals_earned.xuid.
//
// Elles voyageaient toutes deux dans un `string` nu. `HaloBaselineProvider.
// RecentMatches` déclarait `userID string` et l'injectait dans
// `WHERE mp.xuid = ?` ; ses TROIS appelants (service.computeBaselineFor,
// service.computeCurrentValue, service_evaluate.evaluateOne) le remplissaient
// avec `Challenge.UserID`, donc un slug. Résultat : zéro ligne, toujours,
// partout — baseline vide, jauges à 0, objectifs jamais évalués, SANS le moindre
// message d'erreur, en production. Le correctif supprime le paramètre : le
// provider est lié au joueur à la construction (wire.serviceAndPlayerDB →
// PlayerDB.XUID). Ce ratchet interdit sa réintroduction.
//
// **Sans build tag `integration`** (comme bare_db_recovery_routing_test.go et
// shared_reader_routing_test.go) : scan AST PUR — aucune connexion DuckDB,
// aucun CGO. Il DOIT tourner dans le gate par défaut `go test ./...`, sinon il
// ne protège rien.
//
// **Périmètre : le sous-paquet `prestige/` UNIQUEMENT — décision assumée.** C'est
// la frontière Prestige ↔ données de match, celle où le défaut est né et la seule
// où les deux vocabulaires d'identité se croisent (les repos Prestige y prennent
// des `userID`, les providers de match y prennent des `xuid`/`rosterXUIDs`).
// Mesure sur pièces au moment de l'écriture : allowlist VIDE — forme la plus
// forte du garde-rail. Élargir le scan au paquet duckdb racine imposerait une
// allowlist volumineuse (beaucoup de repos y prennent un `gamertag`/`userID` pour
// des raisons sans rapport), c'est-à-dire un ratchet décoratif.
//
// **Ce qu'il NE couvre PAS.** L'interface `prestige.BaselineProvider` vit dans
// `internal/prestige/` (hors scan) : rien ici n'empêche d'y RE-déclarer un
// paramètre d'identité. Mais un tel paramètre ne cause de dégât qu'une fois
// branché sur une requête `xuid` — et cette jonction, elle, est dans le scan.
// C'est le chokepoint choisi.
//
// **Style** : aligné sur internal/prestige/no_cross_title_aggregation_test.go
// (scan AST + allowlist datée + preuve de morsure). Les helpers de parcours de
// fichiers sont RÉUTILISÉS depuis bare_db_recovery_routing_test.go
// (bareDBScanGoFiles / bareDBScanCovers / bareDBItoa) : même paquet de test, même
// sous-arbre — pas de 3e copie (CLAUDE.md n°6).
package prestige

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var (
	// reXUIDIdentityParam : noms de paramètres désignant une identité APPLICATIVE
	// (celle des tables Prestige). `titleSlug` n'est PAS dans la liste (comparaison
	// sur le nom ENTIER : "titleSlug" != "slug"), `xuid`/`rosterXUIDs` non plus —
	// ce sont les identités correctes pour une requête de match.
	reXUIDIdentityParam = regexp.MustCompile(
		`(?i)^(user|users|userid|userids|user_id|userslug|player|playerid|playerslug|slug|slugs|gamertag|gamertags)$`)

	// reXUIDColumn : référence à une colonne xuid dans un littéral SQL. Le
	// préfixe `_` est accepté pour attraper killer_xuid / victim_xuid.
	reXUIDColumn = regexp.MustCompile(`(?i)(?:\b|_)xuids?\b`)

	// reSQLShape : le littéral ressemble à du SQL (évite de compter une simple
	// chaîne de log contenant le mot « xuid »).
	reSQLShape = regexp.MustCompile(`(?i)\b(SELECT|WHERE|JOIN|FROM|INTO|UPDATE|GROUP\s+BY)\b`)

	// reCollapseWS : condense le SQL multi-lignes pour un message d'erreur lisible.
	reCollapseWS = regexp.MustCompile(`\s+`)
)

// allowlistXUIDIdentity : violations TOLÉRÉES, clé `chemin/relatif.go#NomDeFonction`,
// valeur = justification DATÉE. VIDE, et c'est l'objectif. Une entrée sans date ni
// justification technique transforme le ratchet en décoration.
// TestXUIDIdentityAllowlistNotStale supprime le risque d'entrée morte.
var allowlistXUIDIdentity = map[string]string{}

// xuidIdentityWatchedFiles : fichiers dont la présence dans le scan est
// OBLIGATOIRE. Sentinelle anti-faux-vert : si le walk casse ou si l'un de ces
// fichiers est renommé sans mettre à jour ce garde-rail, le test échoue au lieu
// de passer sur un ensemble vide. Ce sont les deux seuls fichiers du sous-paquet
// qui requêtent match_participants.
var xuidIdentityWatchedFiles = []string{
	"prestige_baseline_provider.go",
	"prestige_squad_match_provider.go",
}

// xuidIdentityViolation localise une fonction qui requête `xuid` tout en
// acceptant une identité applicative en paramètre.
type xuidIdentityViolation struct {
	File    string // relatif au sous-paquet, séparateurs slash
	Func    string
	Param   string
	Line    int
	Snippet string
}

func (v xuidIdentityViolation) key() string { return v.File + "#" + v.Func }

func (v xuidIdentityViolation) String() string {
	return v.File + ":" + bareDBItoa(v.Line) + " — " + v.Func +
		"(… " + v.Param + " …) requête `xuid` : " + v.Snippet
}

// scanXUIDIdentitySource analyse UNE source Go et retourne ses violations.
// Extrait en fonction pure (source → violations) précisément pour que la preuve
// de morsure puisse lui soumettre le code PRÉ-FIX réel : c'est le même code qui
// garde le dépôt et qui est prouvé mordant. Parsing en mode 0 → les commentaires
// ne sont pas dans l'AST, donc jamais matchés.
func scanXUIDIdentitySource(rel string, src []byte) ([]xuidIdentityViolation, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, rel, src, 0)
	if err != nil {
		return nil, err
	}
	var out []xuidIdentityViolation
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		param := appIdentityParamOf(fn.Type)
		if param == "" {
			continue
		}
		snippet, line := firstXUIDQueryLiteral(fset, fn.Body)
		if snippet == "" {
			continue
		}
		out = append(out, xuidIdentityViolation{
			File: rel, Func: fn.Name.Name, Param: param, Line: line, Snippet: snippet,
		})
	}
	return out, nil
}

// appIdentityParamOf retourne le premier paramètre nommé comme une identité
// applicative, "" si la signature n'en porte aucun.
func appIdentityParamOf(ft *ast.FuncType) string {
	if ft == nil || ft.Params == nil {
		return ""
	}
	for _, field := range ft.Params.List {
		for _, name := range field.Names {
			if reXUIDIdentityParam.MatchString(name.Name) {
				return name.Name
			}
		}
	}
	return ""
}

// firstXUIDQueryLiteral retourne le premier littéral SQL du corps qui référence
// une colonne xuid (condensé sur une ligne), et sa ligne.
func firstXUIDQueryLiteral(fset *token.FileSet, body *ast.BlockStmt) (string, int) {
	var (
		snippet string
		line    int
	)
	ast.Inspect(body, func(n ast.Node) bool {
		if snippet != "" {
			return false
		}
		lit, ok := n.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return true
		}
		if !reXUIDColumn.MatchString(lit.Value) || !reSQLShape.MatchString(lit.Value) {
			return true
		}
		snippet = condenseXUIDSnippet(lit.Value)
		line = fset.Position(lit.Pos()).Line
		return false
	})
	return snippet, line
}

// condenseXUIDSnippet met le SQL sur une ligne et le borne (message lisible).
func condenseXUIDSnippet(lit string) string {
	s := strings.TrimSpace(reCollapseWS.ReplaceAllString(lit, " "))
	if len(s) > 140 {
		s = s[:140] + "…"
	}
	return s
}

// scanXUIDIdentityPackage scanne les .go NON-test du sous-paquet.
func scanXUIDIdentityPackage(t *testing.T) []xuidIdentityViolation {
	t.Helper()
	files := bareDBScanGoFiles(t)
	if len(files) == 0 {
		t.Fatal("aucun fichier .go scanné dans internal/platform/duckdb/prestige — walk cassé (faux vert)")
	}
	for _, want := range xuidIdentityWatchedFiles {
		if !bareDBScanCovers(files, want) {
			t.Fatalf("fichier surveillé %q absent du scan — walk cassé ou fichier renommé (faux vert)", want)
		}
	}
	var all []xuidIdentityViolation
	for _, path := range files {
		src, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		vs, err := scanXUIDIdentitySource(filepath.ToSlash(path), src)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		all = append(all, vs...)
	}
	return all
}

// TestNoXUIDQueryFedByAppIdentity — RATCHET PRINCIPAL.
func TestNoXUIDQueryFedByAppIdentity(t *testing.T) {
	var unexpected []string
	for _, v := range scanXUIDIdentityPackage(t) {
		if _, allowed := allowlistXUIDIdentity[v.key()]; allowed {
			continue
		}
		unexpected = append(unexpected, v.String())
	}
	if len(unexpected) == 0 {
		return
	}
	t.Errorf("CONFUSION D'IDENTITÉ (%d) — une requête sur `xuid` (identité XBOX, clé des "+
		"données de match) est exposée à un paramètre d'identité APPLICATIVE (player_slug, "+
		"clé des données Prestige). Les deux sont des `string` : rien n'empêche de les "+
		"intervertir, et l'inversion ne produit AUCUNE erreur — seulement zéro ligne, "+
		"silencieusement :", len(unexpected))
	for _, v := range unexpected {
		t.Errorf("  %s", v)
	}
	t.Errorf("\nFix : lier l'identité à la CONSTRUCTION (le provider est déjà per-joueur) —")
	t.Errorf("  NewHaloBaselineProvider(pdb.SharedReadDB(), pdb.XUID) puis `p.xuid` dans la requête.")
	t.Errorf("Ne pas créer de résolution slug→xuid maison : la résolution canonique est")
	t.Errorf("  config.ResolvePlayer (db_profiles.json / DemoRoster), déjà appliquée par le PlayerDB.")
}

// TestXUIDIdentityAllowlistNotStale — anti-pourrissement : une entrée d'allowlist
// qui ne correspond plus à aucune violation protège du code disparu.
func TestXUIDIdentityAllowlistNotStale(t *testing.T) {
	seen := make(map[string]bool)
	for _, v := range scanXUIDIdentityPackage(t) {
		seen[v.key()] = true
	}
	for key, reason := range allowlistXUIDIdentity {
		if !seen[key] {
			t.Errorf("allowlistXUIDIdentity obsolète : %q ne correspond à aucune violation — "+
				"retirer l'entrée (raison historique : %q)", key, reason)
		}
	}
}

// ── Preuve de morsure ────────────────────────────────────────────────────────────

// readXUIDFixture lit une fixture testdata (code source NON compilé).
func readXUIDFixture(t *testing.T, name string) []byte {
	t.Helper()
	src, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("fixture %s illisible (%v) — sans elle la morsure n'est pas prouvée", name, err)
	}
	return src
}

// TestXUIDIdentityDetectorBites — PREUVE D'ANTI-NO-OP. Le détecteur est confronté
// au code PRÉ-FIX RÉEL (testdata/xuid_identity_prefix.go.txt, copie verbatim de
// `git show HEAD:…/prestige_baseline_provider.go`) et DOIT le signaler. Sans ce
// test, un ratchet vert ne prouverait rien : il pourrait ne rien détecter du tout.
func TestXUIDIdentityDetectorBites(t *testing.T) {
	got, err := scanXUIDIdentitySource("testdata/xuid_identity_prefix.go.txt",
		readXUIDFixture(t, "xuid_identity_prefix.go.txt"))
	if err != nil {
		t.Fatalf("parse fixture pré-fix: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("MORSURE ABSENTE — le code PRÉ-FIX réel (RecentMatches(… userID …) + "+
			"WHERE mp.xuid = ?) produit %d violation(s), want 1 : %v", len(got), got)
	}
	if got[0].Func != "RecentMatches" {
		t.Errorf("fonction signalée = %q, want \"RecentMatches\"", got[0].Func)
	}
	if !strings.EqualFold(got[0].Param, "userID") {
		t.Errorf("paramètre signalé = %q, want \"userID\"", got[0].Param)
	}
	if !strings.Contains(got[0].Snippet, "mp.xuid = ?") {
		t.Errorf("extrait SQL signalé = %q, doit contenir \"mp.xuid = ?\"", got[0].Snippet)
	}
}

// TestXUIDIdentityDetectorSparesConformCode — contrôle négatif : les formes
// CONFORMES (provider lié au joueur livré le 2026-07-26, agrégat population,
// provider d'escouade sur rosterXUIDs, repo Prestige sur user_id sans xuid) ne
// doivent produire AUCUNE violation, sans quoi le ratchet serait ingérable.
func TestXUIDIdentityDetectorSparesConformCode(t *testing.T) {
	got, err := scanXUIDIdentitySource("testdata/xuid_identity_conform.go.txt",
		readXUIDFixture(t, "xuid_identity_conform.go.txt"))
	if err != nil {
		t.Fatalf("parse fixture conforme: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("FAUX POSITIF : le code conforme déclenche %d violation(s) : %v", len(got), got)
	}
}

// TestXUIDIdentityDetectorIgnoresNonSQL — un littéral qui contient « xuid » sans
// être du SQL (message de log, clé structurée) n'est pas une requête : le
// détecteur ne doit pas s'y déclencher, sinon logger le xuid deviendrait interdit.
func TestXUIDIdentityDetectorIgnoresNonSQL(t *testing.T) {
	const src = `package prestige

func logFor(ctx context.Context, userID string) {
	slog.ErrorContext(ctx, "prestige_missing_xuid", "user_id", userID, "xuid", "")
}
`
	got, err := scanXUIDIdentitySource("fixture_log.go", []byte(src))
	if err != nil {
		t.Fatalf("parse fixture log: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("FAUX POSITIF sur un log structuré : %v", got)
	}
}
