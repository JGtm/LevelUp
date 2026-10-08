// Package archlint — no_bare_duckdb_rw_open_test.go : ratchet des ouvertures DuckDB en
// écriture hors du point unique.
//
// Toute ouverture d'un fichier DuckDB en écriture passe par le cache de
// internal/platform/duckdb (OpenReadWrite / OpenReadWriteShared, puis openPhysicalSQLDB) :
// c'est là que le modèle mono-process est tenu (ADR 0013) et que l'ouverture physique d'une
// base joueur aligne ses séquences avant toute écriture (physical_open.go). Un
// `sql.Open("duckdb", <fichier>)` ou un `duckdb.NewConnector(<fichier>, …)` ouvert en écriture
// ailleurs contourne les deux : une base joueur ainsi ouverte peut écrire avec une séquence en
// retard (« Duplicate key » sur chaque insertion, cas du LUSR de Chocoboflor, 2026-10-08).
//
// Le test recense ces appels (arbre syntaxique, fichiers de test exclus) sous internal/, cmd/,
// pkg/ et scripts/. Une ouverture est en LECTURE quand le texte de son DSN contient
// `read_only` (toute casse) ; une base en mémoire (DSN "" ou ":memory:") n'est pas un fichier.
//
// Allowlist DÉCROISSANTE, datée 2026-10-08 : le compte par fichier des ouvertures en écriture
// présentes ce jour-là — outils CLI ponctuels (serveur arrêté), seeds, restauration et archive de
// internal/ops, import de sauvegarde, scripts. Un compte ne fait que baisser : un site migré vers
// duckdb.OpenReadWrite (ou passé en lecture) impose d'abaisser l'entrée dans le même commit ;
// un nouveau fichier ou un site de plus échoue. Critère de sortie : allowlist vide.
package archlint

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// bareDuckDBRWOpenAllowlist — fichier (relatif à apps/go-api) -> nombre d'ouvertures en
// écriture hors du point unique au 2026-10-08. Ne fait que décroître.
var bareDuckDBRWOpenAllowlist = map[string]int{
	// Outils CLI ponctuels (un binaire par dossier), lancés serveur arrêté sur une base dont ils
	// sont le seul processus : backfills, diagnostics, réparations, migrations historiques.
	"cmd/apply_tz_migration/main.go":          1,
	"cmd/backfill-csr-history/main.go":        1,
	"cmd/backfill-world-player-stats/main.go": 1,
	"cmd/backfill_first_joined_tz/main.go":    1,
	"cmd/backfill_milestone_dates/main.go":    3,
	"cmd/backfill_participation_info/main.go": 1,
	"cmd/backfill_quit_timestamps/main.go":    1,
	"cmd/backfill_t0/main.go":                 1,
	"cmd/backfill_time_played/main.go":        1,
	"cmd/check_playlists/main.go":             1,
	"cmd/cleanup_media_index/main.go":         2,
	"cmd/cleanup_orphan_match/main.go":        1,
	"cmd/cleanup_post_art/main.go":            2,
	"cmd/diag_bot_resolution/main.go":         1,
	"cmd/diag_citation_counters/main.go":      1,
	"cmd/diag_composite/main.go":              1,
	"cmd/diag_exec/main.go":                   1,
	"cmd/duckdb_7659_repro/main.go":           2,
	"cmd/h5-enrich/main.go":                   1,
	"cmd/h5-events-backfill/main.go":          1,
	"cmd/h5-kill-kind-backfill/main.go":       1,
	"cmd/h5-lusr-backfill/main.go":            1,
	"cmd/h5-lusr-smoke/main.go":               1,
	"cmd/h5-metadata-fetch/main.go":           1,
	"cmd/h5-roster-refetch/main.go":           1,
	"cmd/h5-roster-topup/main.go":             1,
	"cmd/h5-teamscore-backfill/main.go":       1,
	"cmd/lusr_v2_canonical_backfill/main.go":  1,
	"cmd/lusr_v2_replay/main.go":              1,
	"cmd/lusr_v2_squad_estimate/main.go":      1,
	"cmd/lusr_v2_ttt_batch/main.go":           1,
	"cmd/migrate-media-paths/main.go":         1,
	"cmd/migrate-roman-ranks/main.go":         1,
	"cmd/migrate-to-shared-social/main.go":    1,
	"cmd/migrate-xuid-aliases-global/main.go": 2,
	"cmd/populate-playlists-catalog/main.go":  1,
	"cmd/prestige-seed/main.go":               4,
	"cmd/purge_corrupt_records/main.go":       1,
	"cmd/purge_foreign_lusr_chain/main.go":    1,
	"cmd/purge_player_media/main.go":          1,
	"cmd/rebuild_mp/main.go":                  1,
	"cmd/rebuild_pme_art/main.go":             1,
	"cmd/rebuild_shared_social/main.go":       2,
	"cmd/recompute_perfnote/main.go":          1,
	"cmd/regen-thumbnails/main.go":            1,
	"cmd/reindex-media-thumbs/main.go":        1,
	"cmd/repair-metadata/main.go":             1,
	"cmd/seed-assists-model/main.go":          1,
	"cmd/seed-medal/main.go":                  1,
	"cmd/seed-ranked-playlists/main.go":       1,
	"cmd/seed-weapon-labels/main.go":          1,
	"cmd/snapshot-world-leaderboard/main.go":  1,
	"cmd/wal_forensic_compare/main.go":        5,
	"cmd/world-aliases-persist/main.go":       1,
	// CLI principale : sous-commandes ponctuelles (restauration CSR, reconstruction PME,
	// consolidation des alias), même régime que les outils ci-dessus.
	"cmd/levelup/cmd_consolidate_aliases.go": 1,
	"cmd/levelup/cmd_rebuild_pme.go":         1,
	"cmd/levelup/cmd_restore_csr.go":         1,
	// internal/ops : seeds (metadata, démo, synthétiques : bases fabriquées par la commande),
	// restauration et archive (CLI `levelup data`), repli d'indexation média sur shared_social.
	"internal/ops/archive.go":                    1,
	"internal/ops/media_hls.go":                  1,
	"internal/ops/restore.go":                    1,
	"internal/ops/seed.go":                       4,
	"internal/ops/seed_demo.go":                  5,
	"internal/ops/seed_demo_corpus.go":           1,
	"internal/ops/seed_demo_media.go":            1,
	"internal/ops/seed_demo_media_h5.go":         1,
	"internal/ops/seed_demo_prestige.go":         1,
	"internal/ops/seed_demo_prestige_social.go":  1,
	"internal/ops/seed_demo_synthetic_meta.go":   2,
	"internal/ops/seed_demo_synthetic_player.go": 1,
	"internal/ops/seed_demo_synthetic_shared.go": 2,
	// Import de sauvegarde (base reconstruite depuis les Parquet exportés).
	"pkg/duckdbbackup/importer.go": 1,
	// scripts/ : utilitaires ponctuels Battle Pass et sync_meta.
	"scripts/check_syncmeta/main.go":      1,
	"scripts/import_bp_items/main.go":     1,
	"scripts/import_bp_tracks/main.go":    1,
	"scripts/repair_tracks_table/main.go": 1,
	"scripts/warm_bp_assets/main.go":      2,
}

// bareDuckDBRWRoots — racines balayées, relatives à apps/go-api.
var bareDuckDBRWRoots = []string{"internal", "cmd", "pkg", "scripts"}

// bareDuckDBRWExemptDir — le point unique lui-même.
const bareDuckDBRWExemptDir = "internal/platform/duckdb/"

func TestAucuneOuvertureDuckDBEnEcritureHorsDuPointUnique(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	goAPIRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))

	found := map[string][]string{} // fichier -> sites "ligne  texte"
	for _, root := range bareDuckDBRWRoots {
		dir := filepath.Join(goAPIRoot, root)
		if _, err := os.Stat(dir); err != nil {
			continue
		}
		err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			rel, _ := filepath.Rel(goAPIRoot, path)
			rel = filepath.ToSlash(rel)
			if strings.HasPrefix(rel, bareDuckDBRWExemptDir) && !strings.Contains(rel[len(bareDuckDBRWExemptDir):], "/") {
				return nil
			}
			sites, perr := bareDuckDBRWOpens(path)
			if perr != nil {
				return perr
			}
			if len(sites) > 0 {
				found[rel] = sites
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", root, err)
		}
	}

	var problems []string
	for rel, sites := range found {
		allowed := bareDuckDBRWOpenAllowlist[rel]
		if len(sites) > allowed {
			problems = append(problems, rel+" : "+strconv.Itoa(len(sites))+" ouverture(s) en écriture, allowlist "+
				strconv.Itoa(allowed)+"\n      "+strings.Join(sites, "\n      "))
		}
		if len(sites) < allowed {
			problems = append(problems, rel+" : "+strconv.Itoa(len(sites))+" ouverture(s) < allowlist "+
				strconv.Itoa(allowed)+" — ABAISSER l'entrée dans le même commit (le ratchet ne fait que décroître)")
		}
	}
	for rel := range bareDuckDBRWOpenAllowlist {
		if _, ok := found[rel]; !ok {
			problems = append(problems, rel+" : plus aucune ouverture en écriture — RETIRER l'entrée de l'allowlist")
		}
	}
	if len(problems) > 0 {
		sort.Strings(problems)
		t.Errorf("ouverture DuckDB en écriture hors du point unique (internal/platform/duckdb) : passer par "+
			"duckdb.OpenReadWrite / OpenReadWriteShared (une base joueur y est alignée à l'ouverture), ou "+
			"ouvrir en lecture (`?access_mode=read_only`, OpenReadOnly, OpenReadForQuery) :\n  %s",
			strings.Join(problems, "\n  "))
	}
}

// bareDuckDBRWOpens rend les ouvertures DuckDB en écriture d'un fichier source :
// `sql.Open("duckdb", dsn)` et `<driver duckdb>.NewConnector(dsn, …)`, DSN ni en lecture ni en
// mémoire.
func bareDuckDBRWOpens(path string) ([]string, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, src, 0)
	if err != nil {
		return nil, err
	}
	sqlName, driverName := "", ""
	for _, imp := range f.Imports {
		p, _ := strconv.Unquote(imp.Path.Value)
		name := ""
		if imp.Name != nil {
			name = imp.Name.Name
		}
		switch p {
		case "database/sql":
			sqlName = orDefault(name, "sql")
		case "github.com/duckdb/duckdb-go/v2":
			driverName = orDefault(name, "duckdb")
		}
	}
	var sites []string
	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		x, ok := sel.X.(*ast.Ident)
		if !ok {
			return true
		}
		var dsn ast.Expr
		switch {
		case sqlName != "" && x.Name == sqlName && sel.Sel.Name == "Open" && len(call.Args) == 2 && isDuckDBDriverLit(call.Args[0]):
			dsn = call.Args[1]
		case driverName != "" && x.Name == driverName && sel.Sel.Name == "NewConnector" && len(call.Args) >= 1:
			dsn = call.Args[0]
		default:
			return true
		}
		text := string(src[fset.Position(dsn.Pos()).Offset:fset.Position(dsn.End()).Offset])
		if strings.Contains(strings.ToLower(text), "read_only") || text == `""` || text == `":memory:"` {
			return true
		}
		line := fset.Position(call.Pos()).Line
		callText := string(src[fset.Position(call.Pos()).Offset:fset.Position(call.End()).Offset])
		sites = append(sites, strconv.Itoa(line)+"  "+callText)
		return true
	})
	return sites, nil
}

func isDuckDBDriverLit(e ast.Expr) bool {
	lit, ok := e.(*ast.BasicLit)
	return ok && lit.Kind == token.STRING && lit.Value == `"duckdb"`
}

func orDefault(name, def string) string {
	if name == "" {
		return def
	}
	return name
}
