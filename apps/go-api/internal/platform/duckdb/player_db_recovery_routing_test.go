// Package duckdb — player_db_recovery_routing_test.go : garde-rail anti-régression
// du sweep recovery des lectures player-DB (Lot D — .ai/PLAN_PLAYER_DB_RECOVERY_SWEEP_2026-07.md).
//
// **Contexte.** `pdb.Player` est l'UNIQUE handle RW du `stats.duckdb` joueur (renvoyé
// aussi par `pdb.ReadDB()`), partagé par tous les repos player. Un writer concurrent
// qui déclenche `db.Reopen()` (invalidation ART / B-swap → `old.Close()` transitoire)
// fait voir « sql: database is closed » à un accès EN VOL sur les méthodes PLATES
// `Query`/`QueryRow`/`Exec`. Les lots A/B/C ont routé les lectures vers les variantes
// `*Recovered` (`QueryRecovered` / `QueryRowRecovered`) ; l'extension E5 (revue 2026-07)
// y ajoute les écritures (`ExecRecovered`). Toutes font Reopen + retry une fois. Ce test
// empêche la régression : il échoue si un accès PLAT (lecture OU écriture) réapparaît sur
// un handle player-DB accédé sous forme EXPLICITE.
//
// **Sans build tag `integration`** (comme shared_reader_routing_test.go / le trio
// no_art_patterns_test.go) : scan statique PUR — aucune connexion DuckDB, aucun CGO,
// aucune fixture. Il DOIT tourner dans le gate par défaut `go test ./...`, sinon il ne
// protège rien.
//
// **Portée et limites (à connaître).**
//   - Couvre les FORMES EXPLICITES `X.Player.Query(` / `X.Player.QueryRow(` et
//     `X.ReadDB().Query(` / `X.ReadDB().QueryRow(` — le vecteur de régression DOMINANT
//     (la grande majorité des repos accèdent au handle ainsi).
//   - Le champ NU `db *DB` (repos qui lisent via `r.db.Query(...)`, invisible au grep)
//     est désormais fermé PAR CONSTRUCTION (F13 / décision D6, 2026-07-17) : les repos
//     player-DB de cette couche stockent un `PlayerReadHandle` (player_read_handle.go)
//     qui n'expose QUE les variantes recovery-safe. Une lecture/écriture plate sur ces
//     repos ne compile plus — la garantie passe du grep au TYPE. Migrés :
//     coach_proposal_repo, streaks_repo, record_history_repo, milestones_earned_repo,
//     campaign_repo, prestige/prestige_player_repo (5 structs). Ce garde-rail grep reste
//     en filet pour les FORMES EXPLICITES (`pdb.Player.Query(` / `pdb.ReadDB().Query(`).
//   - Scope = arbre `internal/platform/duckdb/` RÉCURSIF (root + halo5/ + prestige/ +
//     sharedprovider/). Les lectures player-DB HORS de cette couche (ex. les snapshots
//     post-sync `internal/api/wire/post_sync_*.go`, converties dans le MÊME lot D) ne
//     sont pas gardées ici : le plan a scopé le garde-rail à la couche duckdb.
package duckdb

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// rePlainPlayerHandleReads : accès PLATS (lectures Query/QueryRow ET écritures Exec)
// sur un handle player-DB accédé explicitement. Motifs du plan (Lot D + extension E5
// aux écritures : revue 2026-07). Les variantes `*Recovered` NE matchent PAS : après
// `Query`/`QueryRow`/`Exec` vient `Recovered` (pas `(`), donc `(Query|QueryRow|Exec)\(`
// échoue sur `QueryRecovered(` / `QueryRowRecovered(` / `ExecRecovered(`. De même
// `QueryContext(` / `QueryRowContext(` / `ExecContext(` (SharedReader) ne matchent pas.
var rePlainPlayerHandleReads = []*regexp.Regexp{
	regexp.MustCompile(`\.Player\.(Query|QueryRow|Exec)\(`),
	regexp.MustCompile(`\.ReadDB\(\)\.(Query|QueryRow|Exec)\(`),
}

// plainPlayerReadAllowEntry : exception DATÉE et justifiée. Une lecture plate est
// tolérée si `fileSuffix` correspond au fichier ET `needle` apparaît dans la fenêtre
// (ligne du match + 2 suivantes) — ainsi seule la lecture ciblée est exemptée, pas
// tout le fichier (une NOUVELLE lecture plate non-`needle` y serait quand même prise).
type plainPlayerReadAllowEntry struct {
	fileSuffix string
	needle     string
	reason     string
}

// plainPlayerReadAllowlist : exceptions au sweep recovery. Idéalement vide.
//
// Les 2 seules entrées ciblent la table clé/valeur `sync_meta`, EXPLICITEMENT exclue
// du périmètre par la décision DC-3 du plan (ce n'est pas une table de stats /
// enrichment append-only ; son couple read+write ART-safe est traité à part).
var plainPlayerReadAllowlist = []plainPlayerReadAllowEntry{
	// allowlist(2026-07-16): ReadSyncMeta lit la table sync_meta sur le handle player.
	// DC-3 exclut sync_meta du sweep recovery. Lecture plate cohérente avec l'écriture
	// ART-safe (SELECT-then-UPDATE-or-INSERT) du même fichier.
	{fileSuffix: "sync_meta_repo.go", needle: "sync_meta", reason: "DC-3: table sync_meta hors perimetre"},
	// allowlist(2026-07-16): GetProgressionDiag lit sync_meta (last_post_sync_at) sur le
	// handle player pour un diag best-effort. Même exclusion DC-3 ; consigné comme
	// ambigu dès le Lot A (af6ccdd6) et laissé tel quel.
	{fileSuffix: "progression_diag_repo.go", needle: "sync_meta", reason: "DC-3: table sync_meta hors perimetre (consigne Lot A)"},
}

// TestNoPlainPlayerHandleReads — garde-rail principal. Scanne les .go non-test de
// l'arbre duckdb ; échoue si une lecture PLATE sur handle player-DB explicite
// apparaît hors allowlist.
func TestNoPlainPlayerHandleReads(t *testing.T) {
	files := recoveryScanGoFiles(t)
	if len(files) == 0 {
		t.Fatal("aucun fichier .go scanné dans internal/platform/duckdb — walk cassé (faux vert)")
	}

	var violations []string
	for _, path := range files {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		lines := strings.Split(string(raw), "\n")
		for i, line := range lines {
			if !recoveryLineHasPlainRead(recoveryStripLineComment(line)) {
				continue
			}
			if recoveryReadAllowed(path, recoveryReadWindow(lines, i)) {
				continue
			}
			violations = append(violations,
				filepath.ToSlash(path)+":"+recoveryItoa(i+1)+" — "+strings.TrimSpace(line))
		}
	}

	if len(violations) == 0 {
		return
	}
	t.Errorf("régression sweep recovery : %d lecture(s) PLATE(s) sur handle player-DB "+
		"(pdb.Player / pdb.ReadDB()) — une lecture en vol voit « database is closed » si un "+
		"writer concurrent Reopen() le handle :", len(violations))
	for _, v := range violations {
		t.Errorf("  %s", v)
	}
	t.Errorf("\nFix : router vers la variante *Recovered (Reopen+retry) —")
	t.Errorf("  .Query(ctx, ...)            -> .QueryRecovered(ctx, ...)")
	t.Errorf("  .QueryRow(ctx, ...).Scan()  -> rows, err := .QueryRowRecovered(ctx, ...) ; defer rows.Close() ; rows.Scan(...)")
	t.Errorf("  .Exec(ctx, ...)             -> .ExecRecovered(ctx, ...)")
	t.Errorf("Cf. .ai/PLAN_PLAYER_DB_RECOVERY_SWEEP_2026-07.md (forme de référence : prestige/prestige_player_repo.go).")
	t.Errorf("Si l'accès est HORS périmètre (table sync_meta / metadata / shared / shared_social — DC-3), " +
		"ajouter une entrée DATÉE à plainPlayerReadAllowlist avec justification.")
}

// TestPlainPlayerReadAllowlistNotStale — anti-pourrissement : chaque entrée
// d'allowlist doit correspondre à une lecture plate réelle (fichier présent + match +
// needle dans la fenêtre). Sinon l'entrée protège du code disparu → la retirer. Double
// aussi comme sentinelle : si le walk casse (0 fichier), aucune entrée ne matche → fail.
func TestPlainPlayerReadAllowlistNotStale(t *testing.T) {
	files := recoveryScanGoFiles(t)
	for _, entry := range plainPlayerReadAllowlist {
		if !recoveryAllowlistEntryMatches(t, files, entry) {
			t.Errorf("allowlist obsolète : {file=%q needle=%q} ne correspond à aucune lecture "+
				"plate player-DB (raison: %s) — retirer l'entrée.",
				entry.fileSuffix, entry.needle, entry.reason)
		}
	}
}

// TestPlainPlayerReadPatternSanity — un garde-rail qui ne mord jamais est inutile :
// on prouve que les motifs attrapent les formes plates ET laissent passer les
// `*Recovered` / les accessors shared.
func TestPlainPlayerReadPatternSanity(t *testing.T) {
	shouldMatch := []string{
		"rows, err := r.pdb.Player.Query(ctx, q)",
		"x := r.pdb.Player.QueryRow(ctx, q).Scan(&v)",
		"y := pdb.ReadDB().Query(ctx, q)",
		"err := pdb.ReadDB().QueryRow(ctx, q).Scan(&v)",
	}
	for _, s := range shouldMatch {
		if !recoveryLineHasPlainRead(s) {
			t.Errorf("devrait matcher (lecture PLATE) : %q", s)
		}
	}
	shouldMatchExec := []string{
		"_, err := r.pdb.Player.Exec(ctx, q, args...)",
		"if _, err := pdb.ReadDB().Exec(ctx, q); err != nil {",
	}
	for _, s := range shouldMatchExec {
		if !recoveryLineHasPlainRead(s) {
			t.Errorf("devrait matcher (écriture PLATE Exec, extension E5) : %q", s)
		}
	}
	shouldNotMatch := []string{
		"rows, err := r.pdb.Player.QueryRecovered(ctx, q)",
		"rows, err := pdb.Player.QueryRowRecovered(ctx, q)",
		"rows, err := r.pdb.ReadDB().QueryRecovered(ctx, q)",
		"rows, err := r.pdb.ReadDB().QueryRowRecovered(ctx, q)",
		"sharedDB.QueryContext(ctx, q)",
		"sharedDB.QueryRowContext(ctx, q).Scan(&v)",
		"db, release, err := r.pdb.SharedReadDB().Get(ctx)",
		"_, err := r.pdb.Player.ExecRecovered(ctx, q)",
		"_, err := r.pdb.ReadDB().ExecRecovered(ctx, q)",
		"_, err := r.pdb.Player.ExecContext(ctx, q)",
	}
	for _, s := range shouldNotMatch {
		if recoveryLineHasPlainRead(s) {
			t.Errorf("ne devrait PAS matcher (déjà recovered / hors périmètre) : %q", s)
		}
	}
}

// recoveryLineHasPlainRead retourne true si la ligne (commentaire de fin déjà retiré)
// contient l'une des lectures plates ciblées.
func recoveryLineHasPlainRead(code string) bool {
	for _, re := range rePlainPlayerHandleReads {
		if re.MatchString(code) {
			return true
		}
	}
	return false
}

// recoveryReadAllowed retourne true si la fenêtre du match est couverte par une
// entrée d'allowlist (suffixe de fichier + needle présent dans la fenêtre).
func recoveryReadAllowed(path, window string) bool {
	slash := filepath.ToSlash(path)
	for _, entry := range plainPlayerReadAllowlist {
		if strings.HasSuffix(slash, entry.fileSuffix) && strings.Contains(window, entry.needle) {
			return true
		}
	}
	return false
}

// recoveryAllowlistEntryMatches vérifie qu'une entrée d'allowlist correspond bien à
// une lecture plate réelle dans les fichiers scannés.
func recoveryAllowlistEntryMatches(t *testing.T, files []string, entry plainPlayerReadAllowEntry) bool {
	t.Helper()
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
			if !recoveryLineHasPlainRead(recoveryStripLineComment(line)) {
				continue
			}
			if strings.Contains(recoveryReadWindow(lines, i), entry.needle) {
				return true
			}
		}
	}
	return false
}

// recoveryReadWindow retourne la ligne d'index i et les 2 suivantes jointes — assez
// pour capturer le SQL inline placé sur la ligne suivant l'appel (ex.
// progression_diag_repo.go : appel .ReadDB().QueryRow(ctx, sur une ligne, le SQL
// contenant sync_meta sur la ligne d'après).
func recoveryReadWindow(lines []string, i int) string {
	end := i + 3
	if end > len(lines) {
		end = len(lines)
	}
	return strings.Join(lines[i:end], "\n")
}

// recoveryStripLineComment retire un commentaire `//` de fin de ligne (approximation
// suffisante : un `//` DANS une string sur la même ligne qu'une lecture plate est un
// cas inexistant ici — les appels de méthode sont sur leur propre ligne). Neutralise
// le faux positif du commentaire de shared_query_helpers.go.
func recoveryStripLineComment(line string) string {
	if idx := strings.Index(line, "//"); idx >= 0 {
		return line[:idx]
	}
	return line
}

// recoveryScanGoFiles liste récursivement les .go NON-test de l'arbre duckdb (cwd du
// package au moment du test), en sautant testdata/.
func recoveryScanGoFiles(t *testing.T) []string {
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

// recoveryItoa : conversion int->string minimale (évite un import strconv, aligné sur
// le style de shared_reader_routing_test.go).
func recoveryItoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
