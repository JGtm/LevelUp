// Package prestige — bare_db_recovery_routing_test.go : garde-rail anti-régression
// du routage recovery des repos Prestige qui portent un champ NU `db *duckdb.DB`.
//
// **Le trou qu'il ferme.** `internal/platform/duckdb/player_db_recovery_routing_test.go`
// ne détecte que les FORMES EXPLICITES `pdb.Player.Query(` / `pdb.ReadDB().Query(`.
// Les repos de ce sous-package rangent la handle dans un champ NU (`r.db`) et
// lisent via `r.db.Query(...)` : ils lui sont INVISIBLES. C'est exactement par ce
// trou que sont passées DEUX régressions successives —
// prestige_metadata_repo.go (V721-08.1, 6 accès plats → 500 permanent sur le
// catalogue Prestige) puis prestige_social_repo.go (V721-08b, 5 accès plats →
// 500 permanent sur les lectures d'escouade). Un accès PLAT (`Query` / `QueryRow`
// / `Exec`) ne survit pas à un Reopen concurrent (invalidation ART, handle
// périmée « sql: database is closed ») : seules les variantes `*Recovered`
// (duckdb/db_query.go → WithReopenOnInvalidated) rouvrent et rejouent.
//
// **Sans build tag `cgo`** (comme player_db_recovery_routing_test.go et le trio
// no_art_patterns_test.go) : scan statique PUR — aucune connexion DuckDB, aucune
// fixture. Il DOIT tourner dans le gate par défaut `go test ./...`.
//
// **Périmètre : le sous-paquet `prestige/` UNIQUEMENT — décision assumée.**
// Mesure sur pièces au moment de l'écriture (rg `\bdb\.(Query|QueryRow|Exec)\(`
// sur les .go non-test de internal/platform/duckdb/) : ~25 accès plats subsistent
// dans le paquet RACINE (persist_sink*.go, social_repo.go, queries_auth.go,
// pool.go, milestones_catalog_repo.go, notifications_repo_helpers.go,
// match_view_repo_weapons.go, home_repo_medals_citations.go). Étendre le scan à
// tout l'arbre imposerait une allowlist de cette taille — un ratchet décoratif,
// pire que rien. Le sous-paquet `prestige/` est au contraire à ZÉRO accès plat
// après V721-08b : allowlist VIDE, forme la plus forte du garde-rail. C'est aussi
// le périmètre où le défaut coûte le plus cher — ces repos servent la surface
// HTTP Prestige/Escouade (500 visible utilisateur) et couvrent les 3 familles de
// bases (player, metadata, shared_social). Quand un repo racine sera converti,
// élargir le scan ici (ou dupliquer ce fichier côté racine) et documenter.
//
// **Style** : aligné sur internal/sync/no_art_patterns_test.go (scan + allowlist
// datée) et player_db_recovery_routing_test.go. Les helpers sont ré-écrits ici
// plutôt qu'importés : paquets de test disjoints depuis l'extraction K3a du
// sous-paquet prestige (même raison que walSize dans prestige_writes_checkpoint_test.go).
package prestige

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// reBareDBPlainAccess : accès PLAT sur une handle `*duckdb.DB` nommée `db` —
// champ de repo (`r.db.Query(`) ou paramètre de helper (`db.Exec(`). Les
// variantes `*Recovered` NE matchent PAS : après `Query`/`QueryRow`/`Exec` vient
// `Recovered` (pas `(`). Idem `QueryContext(` / `QueryRowContext(` / `ExecContext(`,
// utilisés légitimement sur le `*sql.DB` du SharedReadDB (prestige_baseline_provider.go,
// prestige_squad_match_provider.go). `duckdb.NullableStr(` ne matche pas non plus :
// le `\b` exige une frontière de mot avant `db.` (le `k` de `duckdb` n'en est pas une).
var reBareDBPlainAccess = regexp.MustCompile(`\bdb\.(Query|QueryRow|Exec)\(`)

// bareDBAllowEntry : exception DATÉE et justifiée. Tolérée si `fileSuffix`
// correspond au fichier ET `needle` apparaît dans la fenêtre (ligne du match + 2
// suivantes) — ainsi seul l'accès ciblé est exempté, pas tout le fichier.
type bareDBAllowEntry struct {
	fileSuffix string
	needle     string
	reason     string
}

// bareDBAllowlist : VIDE, et c'est l'objectif. Toute entrée ajoutée ici doit
// porter une date `allowlist(AAAA-MM-JJ)` et une justification technique (pas
// « pas eu le temps »). Une écriture n'est PAS une justification : ExecRecovered
// existe et est déjà utilisé partout dans ce sous-paquet.
var bareDBAllowlist []bareDBAllowEntry

// bareDBWatchedFiles : fichiers dont la présence dans le scan est OBLIGATOIRE.
// Sentinelle anti-faux-vert : si le walk casse, si le sous-paquet est déplacé ou
// si l'un de ces fichiers est renommé sans mettre à jour ce garde-rail, le test
// échoue au lieu de passer sur un ensemble vide. Ce sont les deux fichiers qui
// ont réellement régressé (V721-08.1 et V721-08b).
var bareDBWatchedFiles = []string{
	"prestige_social_repo.go",
	"prestige_metadata_repo.go",
}

// TestNoBareDBPlainAccess — garde-rail principal.
func TestNoBareDBPlainAccess(t *testing.T) {
	files := bareDBScanGoFiles(t)
	if len(files) == 0 {
		t.Fatal("aucun fichier .go scanné dans internal/platform/duckdb/prestige — walk cassé (faux vert)")
	}
	for _, want := range bareDBWatchedFiles {
		if !bareDBScanCovers(files, want) {
			t.Fatalf("fichier surveillé %q absent du scan — walk cassé ou fichier renommé (faux vert)", want)
		}
	}

	var violations []string
	for _, path := range files {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		lines := strings.Split(string(raw), "\n")
		for i, line := range lines {
			if !bareDBLineHasPlainAccess(bareDBStripLineComment(line)) {
				continue
			}
			if bareDBAccessAllowed(path, bareDBWindow(lines, i)) {
				continue
			}
			violations = append(violations,
				filepath.ToSlash(path)+":"+bareDBItoa(i+1)+" — "+strings.TrimSpace(line))
		}
	}

	if len(violations) == 0 {
		return
	}
	t.Errorf("régression routage recovery : %d accès PLAT(S) sur une handle `db *duckdb.DB` — "+
		"un Reopen concurrent (invalidation ART / handle périmée) les fait échouer "+
		"définitivement jusqu'au restart, soit un 500 permanent côté HTTP :", len(violations))
	for _, v := range violations {
		t.Errorf("  %s", v)
	}
	t.Errorf("\nFix : router vers la variante *Recovered (Reopen + retry) —")
	t.Errorf("  .Query(ctx, ...)            -> .QueryRecovered(ctx, ...)")
	t.Errorf("  .QueryRow(ctx, ...).Scan()  -> rows, err := .QueryRowRecovered(ctx, ...) ; defer rows.Close() ; rows.Scan(...)")
	t.Errorf("  .Exec(ctx, ...)             -> .ExecRecovered(ctx, ...) (ou execCheckpointed sur shared_social)")
	t.Errorf("Et traduire la contention fichier en dblease.ErrDBLocked (503) : cf. translateSocialLockErr " +
		"(prestige_social_recovery.go) / translateLockErr (prestige_metadata_repo.go).")
}

// TestBareDBAllowlistNotStale — anti-pourrissement : chaque entrée d'allowlist
// doit correspondre à un accès plat réel, sinon elle protège du code disparu.
func TestBareDBAllowlistNotStale(t *testing.T) {
	files := bareDBScanGoFiles(t)
	for _, entry := range bareDBAllowlist {
		if !bareDBAllowlistEntryMatches(files, entry) {
			t.Errorf("allowlist obsolète : {file=%q needle=%q} ne correspond à aucun accès plat "+
				"(raison: %s) — retirer l'entrée.", entry.fileSuffix, entry.needle, entry.reason)
		}
	}
}

// TestBareDBPatternSanity — un garde-rail qui ne mord jamais est pire que rien.
// On confronte le prédicat aux LIGNES RÉELLES du code PRÉ-FIX (récupérables via
// `git show HEAD~:...` au moment de l'écriture) : les 5 lectures plates de
// prestige_social_repo.go (V721-08b) et 2 de prestige_metadata_repo.go
// (V721-08.1). Elles DOIVENT matcher — sinon le scan ci-dessus est décoratif.
// Et on vérifie que les formes légitimes (variantes *Recovered, *Context sur le
// *sql.DB du SharedReadDB, helpers duckdb.*) ne matchent PAS.
func TestBareDBPatternSanity(t *testing.T) {
	preFix := []string{
		"\terr := r.db.QueryRow(ctx, `",
		"\terr := r.db.QueryRow(ctx,",
		"\trows, err := r.db.Query(ctx, templateSelectColumns+\" WHERE title_slug = ? ORDER BY cadence, id\", titleSlug)",
		"\trow := r.db.QueryRow(ctx, templateSelectColumns+\" WHERE id = ?\", id)",
		"\trows, err := r.db.Query(ctx, q, args...)",
		"\tif _, err := db.Exec(ctx, `INSERT INTO squad ...`); err != nil {",
	}
	for _, s := range preFix {
		if !bareDBLineHasPlainAccess(bareDBStripLineComment(s)) {
			t.Errorf("devrait matcher (accès PLAT, forme réelle du code pré-fix) : %q", s)
		}
	}
	legit := []string{
		"\trows, err := r.db.QueryRecovered(ctx, q, args...)",
		"\trows, err := r.db.QueryRowRecovered(ctx, `",
		"\t_, err := r.db.ExecRecovered(ctx, `UPDATE challenge SET status = ? WHERE id = ?`, string(status), id)",
		"\tif _, err := db.ExecRecovered(ctx, query, args...); err != nil {",
		"\trows, err := db.QueryContext(ctx, q, args...)",
		"\tif err := db.QueryRowContext(ctx, q, target).Scan(&below, &total); err != nil {",
		"\tm.SquadID, m.Xuid, m.UserID, duckdb.NullableStr(m.Gamertag), m.JoinedAt)",
		"\t_ = duckdb.CheckpointSharedSocial(ctx, db)",
		"\ttx, txErr := r.db.SQLDb().BeginTx(ctx, nil)",
	}
	for _, s := range legit {
		if bareDBLineHasPlainAccess(bareDBStripLineComment(s)) {
			t.Errorf("ne devrait PAS matcher (déjà recovered / hors périmètre) : %q", s)
		}
	}
	// Un accès plat commenté (doc, exemple) n'est pas une violation.
	if bareDBLineHasPlainAccess(bareDBStripLineComment("// rendait, avec db.Query(ctx) plat, toutes les lectures KO")) {
		t.Error("un accès plat en COMMENTAIRE ne doit pas être compté comme violation")
	}
}

// bareDBLineHasPlainAccess : la ligne (commentaire de fin déjà retiré) contient
// un accès plat ciblé.
func bareDBLineHasPlainAccess(code string) bool {
	return reBareDBPlainAccess.MatchString(code)
}

// bareDBAccessAllowed : la fenêtre du match est couverte par une entrée d'allowlist.
func bareDBAccessAllowed(path, window string) bool {
	slash := filepath.ToSlash(path)
	for _, entry := range bareDBAllowlist {
		if strings.HasSuffix(slash, entry.fileSuffix) && strings.Contains(window, entry.needle) {
			return true
		}
	}
	return false
}

// bareDBAllowlistEntryMatches : l'entrée correspond à un accès plat réel.
func bareDBAllowlistEntryMatches(files []string, entry bareDBAllowEntry) bool {
	for _, path := range files {
		if !strings.HasSuffix(filepath.ToSlash(path), entry.fileSuffix) {
			continue
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		lines := strings.Split(string(raw), "\n")
		for i, line := range lines {
			if !bareDBLineHasPlainAccess(bareDBStripLineComment(line)) {
				continue
			}
			if strings.Contains(bareDBWindow(lines, i), entry.needle) {
				return true
			}
		}
	}
	return false
}

// bareDBWindow : ligne d'index i + les 2 suivantes (le SQL inline vit souvent sur
// la ligne suivant l'appel).
func bareDBWindow(lines []string, i int) string {
	end := i + 3
	if end > len(lines) {
		end = len(lines)
	}
	return strings.Join(lines[i:end], "\n")
}

// bareDBStripLineComment retire un commentaire `//` de fin de ligne : la doc de
// prestige_metadata_repo.go / prestige_social_recovery.go cite `db.Query` plat
// comme contre-exemple, ce n'est pas une violation.
func bareDBStripLineComment(line string) string {
	if idx := strings.Index(line, "//"); idx >= 0 {
		return line[:idx]
	}
	return line
}

// bareDBScanCovers : le scan contient un fichier de ce nom de base.
func bareDBScanCovers(files []string, base string) bool {
	for _, path := range files {
		if filepath.Base(path) == base {
			return true
		}
	}
	return false
}

// bareDBScanGoFiles liste les .go NON-test du sous-paquet (cwd du test), en
// sautant testdata/.
func bareDBScanGoFiles(t *testing.T) []string {
	t.Helper()
	var out []string
	err := filepath.Walk(".", func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			if info.Name() == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		out = append(out, path)
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	return out
}

// bareDBItoa : conversion int->string minimale (aligné sur le style de
// player_db_recovery_routing_test.go, qui évite un import strconv).
func bareDBItoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
