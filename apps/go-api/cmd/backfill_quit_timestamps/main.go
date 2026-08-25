//go:build cgo

// cmd/backfill_quit_timestamps — backfille first_joined_time + last_leave_time
// pour les lignes match_participants où first_joined_time IS NULL.
//
// Modèle identique à cmd/backfill_participation_info, mais cible les 2
// timestamps absolus ajoutés en 2026-05-27 pour ordonner précisément les
// quitters (cf. .ai/LUSR_V2_HANDOFF.md "Priority quitter").
//
// Stratégie :
//  1. Applique la migration ADD COLUMN IF NOT EXISTS (idempotent).
//  2. Sélectionne les match_ids distincts où first_joined_time IS NULL.
//  3. Pour chaque match : GetMatchStats API → ExtractParticipants → UPDATE.
//
// L'UPDATE est limité aux 2 colonnes NULL : aucun risque de régression sur les
// autres colonnes. La clause WHERE ... IS NULL rend l'opération idempotente.
//
// Ne pas lancer pendant que le serveur est en cours d'écriture sur shared.
//
// Usage (depuis apps/go-api/) :
//
//	go run -tags cgo ./cmd/backfill_quit_timestamps/
//	go run -tags cgo ./cmd/backfill_quit_timestamps/ -dry-run -limit 10
//	go run -tags cgo ./cmd/backfill_quit_timestamps/ -gamertag Chocoboflor
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"

	"levelup/go-api/internal/platform/auth"
	gosync "levelup/go-api/internal/sync"
)

func main() {
	dataDir := flag.String("data", "data", "Racine du dossier data/")
	titleSlug := flag.String("title", "halo_infinite", "Title slug")
	envFile := flag.String("env-file", ".env.local", "Chemin .env.local")
	authFile := flag.String("auth-file", "data/auth/watcher_tokens.json", "Chemin watcher_tokens.json")
	gamertag := flag.String("gamertag", "Chocoboflor", "Gamertag pour charger les tokens")
	limit := flag.Int("limit", 0, "Nombre max de matchs à traiter (0 = tous)")
	dryRun := flag.Bool("dry-run", false, "Afficher les matchs sans modifier la DB")
	rps := flag.Int("rps", 3, "Requêtes API par seconde")
	flag.Parse()

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))
	loadEnvLocal(*envFile)

	sharedDBPath := filepath.Join(*dataDir, "titles", *titleSlug, "warehouse", "shared_matches_v2.duckdb")

	ctx := context.Background()

	dbDSN := sharedDBPath
	if *dryRun {
		dbDSN = sharedDBPath + "?access_mode=read_only"
	}
	db, err := sql.Open("duckdb", dbDSN)
	if err != nil {
		fatalf("open shared DB: %v", err)
	}
	defer db.Close()

	if !*dryRun {
		if err := applyMigration(ctx, db); err != nil {
			fatalf("migration: %v", err)
		}
	}

	matchIDs, err := queryNullMatchIDs(ctx, db, *limit)
	if err != nil {
		fatalf("query nulls: %v", err)
	}
	fmt.Printf("Matchs à backfiller (first_joined_time IS NULL) : %d\n", len(matchIDs))

	if *dryRun || len(matchIDs) == 0 {
		if *dryRun && len(matchIDs) > 0 {
			n := 20
			if n > len(matchIDs) {
				n = len(matchIDs)
			}
			fmt.Println("--- dry-run : premiers matchs ---")
			for _, id := range matchIDs[:n] {
				fmt.Printf("  %s\n", id)
			}
		}
		return
	}

	tokens, err := loadTokens(ctx, *authFile, *gamertag)
	if err != nil {
		fatalf("tokens: %v", err)
	}
	client := gosync.NewHaloAPIClient(tokens.SpartanToken, tokens.ClearanceToken, *rps)

	var (
		updated int
		skipped int
		apiGone int
		apiErr  int
	)
	start := time.Now()

	for i, matchID := range matchIDs {
		matchJSON, err := client.GetMatchStats(ctx, matchID)
		if err != nil {
			if isGone(err) {
				apiGone++
				if (i+1)%50 == 0 || apiGone <= 5 {
					fmt.Printf("  [%d/%d] EXPIRÉ %s\n", i+1, len(matchIDs), matchID)
				}
				continue
			}
			slog.WarnContext(ctx, "GetMatchStats échoué", "match_id", matchID, "err", err)
			apiErr++
			continue
		}

		rows := gosync.ExtractParticipants(matchJSON)
		if len(rows) == 0 {
			skipped++
			continue
		}

		n, err := updateQuitTimestamps(ctx, db, rows)
		if err != nil {
			slog.WarnContext(ctx, "UPDATE échoué", "match_id", matchID, "err", err)
			apiErr++
			continue
		}
		updated += n

		if (i+1)%100 == 0 || i == len(matchIDs)-1 {
			fmt.Printf("  [%d/%d] +%d lignes (total %d) | expirés=%d erreurs=%d\n",
				i+1, len(matchIDs), n, updated, apiGone, apiErr)
		}
	}

	fmt.Printf("\n=== Résultat ===\n")
	fmt.Printf("Lignes mises à jour : %d\n", updated)
	fmt.Printf("Matchs expirés API  : %d\n", apiGone)
	fmt.Printf("Matchs sans joueurs : %d\n", skipped)
	fmt.Printf("Erreurs             : %d\n", apiErr)
	fmt.Printf("Durée               : %s\n", time.Since(start).Round(time.Second))
}

// applyMigration ajoute les 2 colonnes si elles n'existent pas encore.
func applyMigration(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `
		ALTER TABLE match_participants ADD COLUMN IF NOT EXISTS first_joined_time TIMESTAMPTZ;
		ALTER TABLE match_participants ADD COLUMN IF NOT EXISTS last_leave_time   TIMESTAMPTZ;
	`)
	return err
}

// queryNullMatchIDs retourne les match_ids distincts où first_joined_time IS NULL,
// ordonnés du plus récent au plus ancien (les récents ont plus de chances d'être
// encore accessibles via l'API).
func queryNullMatchIDs(ctx context.Context, db *sql.DB, limit int) ([]string, error) {
	q := `
		SELECT DISTINCT mp.match_id
		FROM match_participants mp
		JOIN match_registry mr ON mr.match_id = mp.match_id
		WHERE mp.first_joined_time IS NULL
		ORDER BY mr.start_time DESC`
	if limit > 0 {
		q += fmt.Sprintf(" LIMIT %d", limit)
	}
	rows, err := db.QueryContext(ctx, q)
	if err != nil && strings.Contains(err.Error(), "first_joined_time") {
		// Colonne absente : migration pas encore appliquée → tous les matchs.
		q2 := `SELECT DISTINCT mp.match_id FROM match_participants mp
			JOIN match_registry mr ON mr.match_id = mp.match_id
			ORDER BY mr.start_time DESC`
		if limit > 0 {
			q2 += fmt.Sprintf(" LIMIT %d", limit)
		}
		rows, err = db.QueryContext(ctx, q2)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// updateQuitTimestamps écrit les 2 timestamps pour les lignes dont
// first_joined_time est encore NULL. Retourne le nombre de lignes modifiées.
//
// Note : FirstJoinedTime n'est jamais nil dans la réponse API. LastLeaveTime
// peut être nil (joueur encore présent à la fin). On accepte donc une row
// avec FirstJoinedTime != nil et LastLeaveTime nil.
func updateQuitTimestamps(ctx context.Context, db *sql.DB, rows []gosync.ParticipantRow) (int, error) {
	total := 0
	for _, row := range rows {
		if row.FirstJoinedTime == nil {
			continue // ParticipationInfo absent dans le JSON de ce match
		}
		res, err := db.ExecContext(ctx, `
			UPDATE match_participants
			SET
				first_joined_time = ?,
				last_leave_time   = ?
			WHERE match_id = ?
			  AND xuid = ?
			  AND first_joined_time IS NULL`,
			row.FirstJoinedTime, row.LastLeaveTime,
			row.MatchID, row.XUID,
		)
		if err != nil {
			return total, fmt.Errorf("UPDATE %s/%s: %w", row.MatchID, row.XUID, err)
		}
		n, _ := res.RowsAffected()
		total += int(n)
	}
	return total, nil
}

func isGone(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "HTTP 404") || strings.Contains(s, "HTTP 410") ||
		strings.Contains(s, "ressource absente")
}

func loadEnvLocal(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.IndexByte(line, '=')
		if idx <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])
		if len(val) >= 2 && ((val[0] == '"' && val[len(val)-1] == '"') ||
			(val[0] == '\'' && val[len(val)-1] == '\'')) {
			val = val[1 : len(val)-1]
		}
		if os.Getenv(key) == "" {
			_ = os.Setenv(key, val)
		}
	}
}

func loadTokens(ctx context.Context, authFile, gamertag string) (*struct {
	SpartanToken   string
	ClearanceToken string
}, error) {
	store := auth.NewTokenStore(authFile)
	stored, _ := store.Load()
	if stored != nil && stored.IsXSTSValid(0) {
		result, err := auth.ExchangeXSTSForHaloTokens(ctx, stored.XSTSToken)
		if err == nil {
			return &struct {
				SpartanToken   string
				ClearanceToken string
			}{result.SpartanToken, result.ClearanceToken}, nil
		}
	}

	// ADR 0023 Phase 5 : le refresh token vient du MultiUserTokenStore, seule
	// source (plus d'env var SPNKR_OAUTH_REFRESH_TOKEN_*).
	// data/auth/watcher_tokens.json → data/auth/watcher_tokens (répertoire du store).
	tokenStore := auth.NewMultiUserTokenStore(strings.TrimSuffix(authFile, ".json"))
	if user, lerr := tokenStore.LoadByGamertag(gamertag); lerr == nil && user != nil {
		res, rerr := auth.RefreshHaloTokensViaStoreFirst(ctx, tokenStore, auth.NewSISUProvider(), user.XUID, gamertag)
		if rerr == nil {
			if tokens := auth.HaloTokensFromExchange(res); tokens != nil {
				return &struct {
					SpartanToken   string
					ClearanceToken string
				}{tokens.SpartanToken, tokens.ClearanceToken}, nil
			}
		}
	}

	return nil, fmt.Errorf("tokens introuvables pour %s (vérifier %s et data/auth/watcher_tokens)", gamertag, authFile)
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "FATAL: "+format+"\n", args...)
	os.Exit(1)
}
