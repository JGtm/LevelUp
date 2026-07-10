//go:build cgo

// lusr_v2_canonical_backfill — applique LUSR v2 en CANONICAL sur l'historique
// complet des joueurs trackés : écrit match_skill_rank (rating_type='LUSR' lu
// par l'UI + 'LUSR_V2' audit) pour chaque match, via Stratégie C (ADR 0024).
//
// Différence avec cmd/lusr_v2_replay : ce backfill passe la DB JOUEUR (RW) à
// RunLUSRV2Shadow (playerDB ≠ nil) + pose LEVELUP_LUSR_CANONICAL=LUSR_V2, donc
// writeCanonicalLUSRRow écrit match_skill_rank. Le replay restait shadow-only.
//
// match_skill_rank est APPEND-ONLY : le backfill AJOUTE des lignes v2 (la vue
// _latest les fait gagner par written_at) ; les lignes v1 restent intactes →
// rollback possible en supprimant les lignes appendées.
//
// Usage :
//
//	go run -tags cgo ./cmd/lusr_v2_canonical_backfill                 # dry-run (compte, shadow-only)
//	go run -tags cgo ./cmd/lusr_v2_canonical_backfill --commit        # écrit match_skill_rank
//	go run -tags cgo ./cmd/lusr_v2_canonical_backfill --commit JGtm   # un joueur précis
//
// Reset PAR JOUEUR (DELETE WHERE xuid=?) + persist OWNER-ONLY : chaque joueur
// reprocesse tous ses matchs et écrit ses lignes SANS écraser l'état v2 des
// autres → couverture complète pour TOUS (corrige le couplage cross-joueur du
// backfill séquentiel, cf. .ai/thought_log 2026-06-07).
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/games/halo_infinite/skillchain"
	lusync "levelup/go-api/internal/sync"
)

var defaultPlayers = []string{"Madina97294", "Chocoboflor", "JGtm", "XxDaemonGamerxX"}

func sharedDBPath(root string) string {
	return root + "/data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb"
}

func playerDBPath(root, gamertag string) string {
	return fmt.Sprintf("%s/data/titles/halo_infinite/players/%s/stats.duckdb", root, gamertag)
}

func main() {
	// MT-15 : câble le classifier LUSR (fail-loud). CRITIQUE — ce binaire ÉCRIT
	// match_skill_rank.playlist_group via GetLUSRChain.
	lusync.SetLUSRChainClassifier(skillchain.ClassifyLUSRChain)

	commit := flag.Bool("commit", false, "écrit match_skill_rank (canonical). Défaut: dry-run shadow-only (compte).")
	dataRoot := flag.String("data-root", ".", "racine du repo (depuis apps/go-api : ../..)")
	flag.Parse()
	players := flag.Args()
	if len(players) == 0 {
		players = defaultPlayers
	}

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))

	if err := os.Setenv("LEVELUP_LUSR_V2_ENABLED", "1"); err != nil {
		slog.Error("setenv ENABLED", "err", err)
		os.Exit(1)
	}
	if *commit {
		if err := os.Setenv("LEVELUP_LUSR_CANONICAL", "LUSR_V2"); err != nil {
			slog.Error("setenv CANONICAL", "err", err)
			os.Exit(1)
		}
	}

	shared := openDB(sharedDBPath(*dataRoot))
	defer shared.Close()

	ctx := context.Background()
	mode := "DRY-RUN (shadow-only, aucune écriture match_skill_rank)"
	if *commit {
		mode = "COMMIT (écrit match_skill_rank canonical)"
	}
	fmt.Printf("=== Backfill LUSR v2 canonical — %s ===\n\n", mode)

	total := 0
	for _, gt := range players {
		xuid := resolveXUID(ctx, shared, gt)
		if xuid == "" {
			slog.Warn("xuid introuvable", "gamertag", gt)
			continue
		}

		// Reset PAR JOUEUR : on efface uniquement l'état v2 de CE joueur pour qu'il
		// reprocesse tous ses matchs, SANS toucher à l'état des autres (owner-only).
		// Évite le couplage cross-joueur du backfill séquentiel (un squad-mate traité
		// avant n'avance plus le watermark de celui-ci → plus de matchs sautés).
		if _, err := shared.ExecContext(ctx, `DELETE FROM player_skill_state_v2 WHERE xuid = ?`, xuid); err != nil {
			slog.Error("reset player_skill_state_v2 (par joueur)", "gamertag", gt, "err", err)
			continue
		}

		var playerDB *sql.DB
		if *commit {
			playerDB = openDB(playerDBPath(*dataRoot, gt))
		}

		processed, err := lusync.RunLUSRV2ShadowOwnerOnly(ctx, playerDB, lusync.NewPinnedSharedAccess(shared), xuid)
		if playerDB != nil {
			playerDB.Close()
		}
		if err != nil {
			slog.Warn("RunLUSRV2ShadowOwnerOnly", "gamertag", gt, "err", err)
			continue
		}
		total += processed
		fmt.Printf("  %-18s %5d matchs %s\n", gt, processed,
			map[bool]string{true: "écrits (canonical)", false: "seraient écrits"}[*commit])
	}

	fmt.Printf("\nTotal : %d matchs.\n", total)
	if !*commit {
		fmt.Println("[DRY-RUN] Relancer avec --commit pour écrire match_skill_rank.")
	} else {
		fmt.Println("[COMMIT] match_skill_rank mis à jour (rating_type='LUSR' = v2). UI = v2 sur tout l'historique.")
	}
}

func openDB(path string) *sql.DB {
	db, err := sql.Open("duckdb", path)
	if err != nil {
		slog.Error("open", "err", err, "path", path)
		os.Exit(1)
	}
	return db
}

// resolveXUID résout le xuid d'un gamertag via v_gamertag_lookup (shared).
func resolveXUID(ctx context.Context, shared *sql.DB, gamertag string) string {
	var xuid sql.NullString
	_ = shared.QueryRowContext(ctx,
		`SELECT xuid FROM v_gamertag_lookup WHERE LOWER(gamertag) = LOWER(?) LIMIT 1`,
		gamertag).Scan(&xuid)
	return strings.TrimSpace(xuid.String)
}
