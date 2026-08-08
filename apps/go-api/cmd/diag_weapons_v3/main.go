// diag_weapons_v3 — CLI diagnostic (shadow) de la pipeline objective-events v3.
//
// Pour chaque match disposant de chunks film cachés + d'un manifest, décode les
// events objectif (objectiveevents.Extract) à partir du cache disque + d'un
// roster (xuid->team_id) résolu depuis shared.match_participants, affiche un
// résumé par match (events par objective_type/event_type, split par équipe), et
// pour le CTF compare le COUNT/split de captures décodées au score final DB
// (match_registry.team_0_score / team_1_score).
//
// SÉCURITÉ ÉCRITURE : DRY-RUN par défaut. Aucune écriture sur la DB tant que
// -write n'est pas passé. -write applique d'abord la migration des tables v3
// (shared_objective_events_v1, idempotente CREATE IF NOT EXISTS) puis persiste
// via ObjectiveEventsRepo.WriteMatch (DELETE-then-INSERT par match). La DB visée
// par -write est celle de -db (par défaut le shared du MAIN tree).
//
// Usage (depuis apps/go-api/, build CGO requis) :
//
//	CGO_ENABLED=1 go run ./cmd/diag_weapons_v3 -match 53ce4390 -dry-run
//	CGO_ENABLED=1 go run ./cmd/diag_weapons_v3 -all
//	CGO_ENABLED=1 go run ./cmd/diag_weapons_v3 -match 53ce4390 -write
//
// IMPORTANT : avec -write, stopper le serveur Go (lock écriture exclusif DuckDB
// sur shared_matches_v2). En dry-run / lectures seules, l'accès est read_only.
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"os"

	_ "github.com/duckdb/duckdb-go/v2"
)

const (
	// defaultCacheDir = racine du cache film (film_chunks/<id>/chunk_NN.bin +
	// film_manifests/<id>.json).
	defaultCacheDir = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/cache`
	// defaultSharedDB = shared_matches_v2.duckdb du MAIN tree (lecture par
	// défaut ; écriture seulement si -write).
	defaultSharedDB = `c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb`
)

func main() {
	var (
		matchID  = flag.String("match", "", "Match ID (prefixe 8-char du cache ou UUID complet)")
		all      = flag.Bool("all", false, "Traiter tous les matchs avec films cachés + manifest")
		cacheDir = flag.String("cache", defaultCacheDir, "Racine du cache film")
		dbPath   = flag.String("db", defaultSharedDB, "Chemin shared_matches_v2.duckdb")
		dryRun   = flag.Bool("dry-run", false, "Force le mode lecture seule (équivalent à ne pas passer -write)")
		write    = flag.Bool("write", false, "Persiste sur -db (sinon shadow / lecture seule)")
		weapons  = flag.Bool("weapons", false, "Mode ARMES : attribution v3 + rapport de comparaison §4 vs v2")
		posmode  = flag.Bool("positions", false, "Mode POSITIONS : décodage des positions keyframe (match-level, §N)")
		// firePi / relax3 — overrides de MESURE (isolation des leviers §8/§9). Vide =
		// défaut orchestrateur (auto-layout + relax3 par défaut). firePi ∈ {auto,4high,
		// 5span,5highb5,5lowb5} ; relax3 ∈ {default,on,off}. N'affecte QUE le shadow.
		firePi = flag.String("firepi", "", "MESURE: override layout fire-pi (auto|4high|5span|5highb5|5lowb5)")
		relax3 = flag.String("relax3", "", "MESURE: override recall relâché fire (default|on|off)")
	)
	flag.Parse()

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn})))

	if *matchID == "" && !*all {
		fmt.Fprintln(os.Stderr, "usage: diag_weapons_v3 (-match <id> | -all) [-cache <dir>] [-db <shared.duckdb>] [-write]")
		os.Exit(2)
	}
	// -dry-run prime sur -write : garde-fou explicite.
	doWrite := *write && !*dryRun

	if err := run(context.Background(), runConfig{
		matchArg:  *matchID,
		all:       *all,
		cacheDir:  *cacheDir,
		dbPath:    *dbPath,
		write:     doWrite,
		weapons:   *weapons,
		positions: *posmode,
		firePi:    *firePi,
		relax3:    *relax3,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "diag_weapons_v3: %v\n", err)
		os.Exit(1)
	}
}

// runConfig regroupe les paramètres résolus du run (évite >5 args, cf. règle).
type runConfig struct {
	matchArg  string
	all       bool
	cacheDir  string
	dbPath    string
	write     bool
	weapons   bool   // mode ARMES (attribution v3 + rapport §4) au lieu des events objectif
	positions bool   // mode POSITIONS (décodage keyframe match-level, §N)
	firePi    string // MESURE: override layout fire-pi (vide = défaut auto)
	relax3    string // MESURE: override recall relâché (vide = défaut)
}

// run ouvre la DB (RW si write, sinon RO), résout la liste de matchs à traiter,
// puis délègue à processMatch pour chacun.
func run(ctx context.Context, cfg runConfig) error {
	conn, toTemp, err := openRunConn(cfg)
	if err != nil {
		return err
	}
	defer conn.close()

	if cfg.write {
		if err := ensureWriteTables(conn.sqlDB, cfg); err != nil {
			return fmt.Errorf("ensure tables: %w", err)
		}
		if toTemp {
			fmt.Println("[NOTE] DB verrouillée (serveur up) -> écriture sur une COPIE temp jetable (vraie DB intacte).")
		}
	}

	ids, err := resolveMatchIDs(ctx, conn.sqlDB, cfg)
	if err != nil {
		return err
	}
	if len(ids) == 0 {
		fmt.Println("Aucun match à traiter (films cachés + manifest + présent en match_registry).")
		return nil
	}

	mode := "DRY-RUN (shadow, aucune écriture)"
	if cfg.write {
		mode = "WRITE -> " + cfg.dbPath
	}
	kind := "objective-events"
	switch {
	case cfg.weapons:
		kind = "weapons-v3"
	case cfg.positions:
		kind = "positions"
	}
	fmt.Printf("=== diag_weapons_v3 [%s] — %d match(s) — %s ===\n\n", kind, len(ids), mode)

	if cfg.weapons {
		return runWeapons(ctx, conn, cfg, ids)
	}
	if cfg.positions {
		return runPositions(ctx, conn, cfg, ids)
	}
	for _, m := range ids {
		if err := processMatch(ctx, conn, cfg, m); err != nil {
			fmt.Printf("[%s] ERREUR: %v\n\n", m.short, err)
		}
	}
	return nil
}

// openRunConn ouvre la connexion adaptée au run. Lecture seule par défaut
// (OpenReadForQuery, safe serveur up). Le -write armes passe par
// openWeaponsWriteConn (RW exclusif, ou copie temp si la DB est verrouillée) ; le
// -write objective-events garde le RW direct de openConn. toTemp signale une copie
// temp jetable.
func openRunConn(cfg runConfig) (c *conn, toTemp bool, err error) {
	// Le -write POSITIONS écrit sur une table shadow/additive : même garde-fou
	// que les armes (copie temp si la DB est verrouillée par le serveur).
	if (cfg.weapons || cfg.positions) && cfg.write {
		return openWeaponsWriteConn(cfg.dbPath)
	}
	c, err = openConn(cfg.dbPath, cfg.write)
	return c, false, err
}

// runWeapons traite le panel en mode ARMES : attribution v3 + rapport §4 par
// match, puis imprime la synthèse panel + verdict §0.
func runWeapons(ctx context.Context, conn *conn, cfg runConfig, ids []matchRef) error {
	agg := newPanelAgg()
	for _, m := range ids {
		if err := processMatchWeapons(ctx, conn, cfg, m, agg); err != nil {
			fmt.Printf("[%s] ERREUR: %v\n\n", m.short, err)
		}
	}
	agg.print()
	return nil
}

// ensureWriteTables applique la migration des tables shadow nécessaires au -write :
// weapon_kills_v3 (ARMES), positions, sinon shared_objective_events_v1.
func ensureWriteTables(db *sql.DB, cfg runConfig) error {
	switch {
	case cfg.weapons:
		return ensureMigration(db, "shared_weapon_kills_v3")
	case cfg.positions:
		return ensureMigration(db, "shared_match_player_positions_v1")
	default:
		return ensureObjectiveEventsTables(db)
	}
}
