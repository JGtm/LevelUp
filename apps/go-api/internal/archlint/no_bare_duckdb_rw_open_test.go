// Package archlint — no_bare_duckdb_rw_open_test.go : ratchet des ouvertures DuckDB en
// écriture hors du point unique.
//
// Toute ouverture d'un fichier DuckDB en écriture passe par le cache de
// internal/platform/duckdb (OpenReadWrite / OpenReadWriteShared, puis openPhysicalSQLDB) :
// c'est là que le modèle mono-process est tenu (ADR 0013) et que l'ouverture physique d'une
// base joueur aligne ses séquences avant toute écriture (physical_open.go). Un
// `sql.Open("duckdb", <fichier>)` ou un `duckdb.NewConnector(<fichier>, …)` ouvert en écriture
// ailleurs contourne les deux : une base joueur ainsi ouverte peut écrire avec une séquence en
// retard (« Duplicate key » sur chaque insertion, cas du LUSR de Joueur90001, 2026-10-08).
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
	"regexp"
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
	"cmd/backfill-csr-history/main.go":        1,
	"cmd/backfill_milestone_dates/main.go":    3,
	"cmd/backfill_t0/main.go":                 1,
	"cmd/backfill_time_played/main.go":        1,
	"cmd/h5-events-backfill/main.go":          1,
	"cmd/h5-kill-kind-backfill/main.go":       1,
	"cmd/h5-metadata-fetch/main.go":           1,
	"cmd/h5-roster-refetch/main.go":           1,
	"cmd/lusr_v2_replay/main.go":              1,
	"cmd/lusr_v2_squad_estimate/main.go":      1,
	"cmd/lusr_v2_ttt_batch/main.go":           1,
	"cmd/migrate-media-paths/main.go":         1,
	"cmd/migrate-to-shared-social/main.go":    1,
	"cmd/migrate-xuid-aliases-global/main.go": 2,
	"cmd/populate-playlists-catalog/main.go":  1,
	"cmd/purge_corrupt_records/main.go":       1,
	"cmd/rebuild_shared_social/main.go":       2,
	"cmd/regen-thumbnails/main.go":            1,
	"cmd/seed-assists-model/main.go":          1,
	"cmd/seed-weapon-labels/main.go":          1,
	"cmd/wal_forensic_compare/main.go":        5,
	// CLI principale : sous-commandes ponctuelles (reconstruction PME,
	// consolidation des alias), même régime que les outils ci-dessus.
	"cmd/levelup/cmd_consolidate_aliases.go": 1,
	"cmd/levelup/cmd_rebuild_pme.go":         1,
	// internal/ops : seeds (metadata, démo, synthétiques : bases fabriquées par la commande),
	// restauration et archive (CLI `levelup data`), repli d'indexation média sur shared_social.
	"internal/ops/archive.go":                    1,
	"internal/ops/media_hls.go":                  1,
	"internal/ops/restore.go":                    1,
	"internal/ops/seed.go":                       4,
	"internal/ops/seed_demo.go":                  5,
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

// ouverturesParFichier balaye bareDuckDBRWRoots et rend, par fichier source (relatif à apps/go-api),
// les sites que repère finder. Fichiers de test et point unique (internal/platform/duckdb/, sans ses
// sous-dossiers) exclus.
func ouverturesParFichier(t *testing.T, finder func(path string) ([]string, error)) map[string][]string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a échoué")
	}
	goAPIRoot := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))

	found := map[string][]string{} // fichier -> sites "ligne  texte"
	for _, root := range bareDuckDBRWRoots {
		dir := racineDeclaree(t, root)
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
			sites, perr := finder(path)
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
	return found
}

func TestAucuneOuvertureDuckDBEnEcritureHorsDuPointUnique(t *testing.T) {
	found := ouverturesParFichier(t, bareDuckDBRWOpens)

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

// ─── duckdb.OpenReadWrite : surface gelée par une allowlist datée (ADR 0030 D-3) ─────────────────
//
// OpenReadWrite est le point d'entrée en écriture de internal/platform/duckdb : un appel de plus est
// un nouveau porteur de handle RW sur une base, donc un nouveau candidat au conflit de process
// (ADR 0013) et aux écritures hors persist (ADR 0019). Les appels légitimes ne sont pas à retirer :
// la surface est GELÉE. Un fichier hors de l'allowlist, ou un site de plus, échoue ; un compte ne
// fait que baisser. Chaque entrée porte la date de sa dernière justification sur pièces et sa
// raison ; une entrée sans date au format AAAA-MM-JJ échoue (TestOpenReadWriteAllowlistDatee).
//
// OpenReadWriteShared (même clé de cache, pool de lecture) n'est pas couvert par ce garde.

// openReadWriteSite : une entrée de l'allowlist.
type openReadWriteSite struct {
	count  int    // nombre d'appels dans le fichier
	depuis string // AAAA-MM-JJ : dernière justification sur pièces
	raison string
}

const (
	raisonOutilCLI = "outil ou sous-commande CLI ponctuelle, lancée serveur arrêté : seul processus de la base, le verrou du fichier refuse un second écrivain"
	raisonBoot     = "migrations ou provisioning d'une base, hors cycle de sync : au démarrage du serveur, avant les lecteurs, ou par une commande CLI serveur arrêté"
	dateRevue      = "2026-11-12"
)

func cli(n int) openReadWriteSite  { return openReadWriteSite{n, dateRevue, raisonOutilCLI} }
func boot(n int) openReadWriteSite { return openReadWriteSite{n, dateRevue, raisonBoot} }

// openReadWriteAllowlist : fichier (relatif à apps/go-api) -> appels de duckdb.OpenReadWrite.
// Seed de l'ADR 0030 D-3 : l'ensemble des sites au 2026-11-12.
var openReadWriteAllowlist = map[string]openReadWriteSite{
	"cmd/apply_shared_migrations/main.go":           cli(1),
	"cmd/backfill-team-rounds/main.go":              cli(1),
	"cmd/backfill_objective_stats/main.go":          cli(1),
	"cmd/backfill_t0_film/registre.go":              cli(1),
	"cmd/diag_weapons_v3/db.go":                     cli(1),
	"cmd/h5-backfill/main.go":                       cli(1),
	"cmd/h5-events-backfill/main.go":                cli(1),
	"cmd/h5-kill-kind-backfill/main.go":             cli(1),
	"cmd/levelup/cmd_backfill_bomb_stats.go":        cli(1),
	"cmd/levelup/cmd_backfill_flag_grabs_net.go":    cli(1),
	"cmd/levelup/cmd_backfill_h5_kill_mechanics.go": cli(1),
	"cmd/levelup/cmd_backfill_is_ranked.go":         cli(1),
	"cmd/levelup/cmd_backfill_killsource.go":        cli(1),
	"cmd/levelup/cmd_backfill_lead_share.go":        cli(1),
	"cmd/levelup/cmd_backfill_match_summaries.go":   cli(1),
	"cmd/levelup/cmd_backfill_medailles_feed.go":    cli(1),
	"cmd/levelup/cmd_backfill_pad_tiers.go":         cli(1),
	"cmd/levelup/cmd_backfill_registry_names.go":    cli(1),
	"cmd/levelup/cmd_backfill_squad_creators.go":    cli(1),
	"cmd/levelup/cmd_backfill_usage_summary.go":     cli(1),
	"cmd/levelup/cmd_backfill_vehicle_takes.go":     cli(1),
	"cmd/levelup/cmd_compact_passes.go":             cli(2),
	"cmd/levelup/cmd_compact_passes_rewrite.go":     cli(1),
	"cmd/levelup/cmd_populate_assets.go":            cli(1),
	"cmd/levelup/cmd_repair_highlight_phantoms.go":  cli(1),
	"cmd/levelup/cmd_replay_events.go":              cli(1),
	"cmd/levelup/cmd_reset_bitmasks.go":             cli(1),
	"cmd/levelup/cmd_restore_csr.go":                cli(1),
	"cmd/levelup/cmd_sync_achievements.go":          boot(1),
	"cmd/levelup/cmd_title.go":                      boot(1),
	"cmd/levelup/migrations_cli.go":                 boot(2),
	"cmd/migrate-static-maps/main.go":               cli(1),
	"cmd/refresh-career-ranks/main.go":              cli(1),
	"cmd/refresh-metadata/main.go":                  cli(1),
	"cmd/seed-rank-translations/main.go":            cli(1),
	"cmd/server/boot_migrations.go":                 boot(4),
	"cmd/server/boot_player_migrations.go":          boot(1),
	"cmd/server/provision_title.go":                 boot(1),
	"cmd/server/sync_v2_wiring.go": {1, dateRevue,
		"ouvreur de base joueur du post-sync V2 : ouverture sous le bail d'écriture dblease (AcquireWriterCtx) pris juste avant"},
	"internal/assets/store_duckdb.go": {1, dateRevue,
		"index d'assets : table asset_index de metadata.duckdb, écrite par SELECT-then-write (UpsertRowNoConflict) depuis la file d'écriture unique du magasin, jamais d'ON CONFLICT"},
	"internal/ops/monitoring_store.go": {1, dateRevue,
		"base de monitoring globale, propre au serveur : aucune table de match, aucune table append-only ; en mémoire en mode démo"},
	"internal/persist/combined_persister.go": {1, dateRevue,
		"persister du lot combiné : base joueur ouverte sous le bail dblease.AcquireLeaseCtx, écrite par PlayerPersister (INSERT-only)"},
	"internal/platform/duckdb/sharedprovider/provider_writer.go": {1, dateRevue,
		"fournisseur de la base partagée : seul porteur du handle RW de shared, ouvert après la fermeture du handle RO (ADR 0016)"},
	"internal/sync/schema.go": {2, dateRevue,
		"OpenPlayerDB et OpenSharedDB : points d'entrée communs des ouvertures RW de la couche sync (cache de processus, schéma appliqué à l'ouverture) ; le bail d'écriture relève de dblease chez l'appelant, non vérifié ici"},
}

// formeDateISO : une date AAAA-MM-JJ, rien d'autre.
var formeDateISO = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

func TestOpenReadWriteAllowlistDatee(t *testing.T) {
	for rel, e := range openReadWriteAllowlist {
		if !formeDateISO.MatchString(e.depuis) {
			t.Errorf("%s : date %q hors format AAAA-MM-JJ — toute entrée est datée (ADR 0030 D-3)", rel, e.depuis)
		}
		if strings.TrimSpace(e.raison) == "" {
			t.Errorf("%s : raison vide — toute entrée est justifiée (ADR 0030 D-3)", rel)
		}
		if e.count < 1 {
			t.Errorf("%s : compte %d — une entrée sans appel se RETIRE", rel, e.count)
		}
	}
}

func TestOpenReadWriteHorsDuPointUniqueSurAllowlist(t *testing.T) {
	found := ouverturesParFichier(t, openReadWriteCalls)

	var problems []string
	for rel, sites := range found {
		e, listed := openReadWriteAllowlist[rel]
		if len(sites) > e.count {
			problems = append(problems, rel+" : "+strconv.Itoa(len(sites))+" appel(s) de duckdb.OpenReadWrite, allowlist "+
				strconv.Itoa(e.count)+" — une entrée neuve exige une date et une raison (ADR 0030 D-3)\n      "+strings.Join(sites, "\n      "))
		}
		if listed && len(sites) < e.count {
			problems = append(problems, rel+" : "+strconv.Itoa(len(sites))+" appel(s) < allowlist "+strconv.Itoa(e.count)+
				" — ABAISSER l'entrée dans le même commit (la surface ne fait que décroître)")
		}
	}
	for rel := range openReadWriteAllowlist {
		if _, ok := found[rel]; !ok {
			problems = append(problems, rel+" : plus aucun appel de duckdb.OpenReadWrite — RETIRER l'entrée de l'allowlist")
		}
	}
	if len(problems) > 0 {
		sort.Strings(problems)
		t.Errorf("surface duckdb.OpenReadWrite (ADR 0030 D-3) :\n  %s", strings.Join(problems, "\n  "))
	}
}

// openReadWriteCalls rend les appels `<alias>.OpenReadWrite(...)` d'un fichier source, où alias est
// le nom local du paquet levelup/go-api/internal/platform/duckdb.
func openReadWriteCalls(path string) ([]string, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, src, 0)
	if err != nil {
		return nil, err
	}
	alias := ""
	for _, imp := range f.Imports {
		if p, _ := strconv.Unquote(imp.Path.Value); p == "levelup/go-api/internal/platform/duckdb" {
			alias = "duckdb"
			if imp.Name != nil {
				alias = imp.Name.Name
			}
		}
	}
	if alias == "" {
		return nil, nil
	}
	var sites []string
	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "OpenReadWrite" {
			return true
		}
		if x, ok := sel.X.(*ast.Ident); ok && x.Name == alias {
			callText := string(src[fset.Position(call.Pos()).Offset:fset.Position(call.End()).Offset])
			sites = append(sites, strconv.Itoa(fset.Position(call.Pos()).Line)+"  "+callText)
		}
		return true
	})
	return sites, nil
}
