package main

// cmd_backfill_squad_creators.go — sous-commande one-shot
// `levelup backfill-squad-creators`.
//
// Répare les escouades LEGACY dont le créateur n'a jamais été inscrit comme
// membre (les escouades créées AVANT que buildSquadMembers injecte le créateur).
// Symptôme corrigé : une escouade « Big Bsses » créée par JGtm mais dont le
// roster persisté ne contient PAS JGtm → sélection player-agnostic cassée
// (le viewer croit voir la compo d'un autre, ou lui-même en double).
//
// Pour chaque escouade : résout created_by (player_slug) → xuid via db_profiles ;
// si ce xuid manque de squad_member_latest, INSÈRE un événement append-only
// is_member=TRUE (via PrestigeSquadRepo.AddMember → execCheckpointed = INSERT pur
// + CHECKPOINT, ADR 0022 ; jamais d'UPSERT). Idempotent : un créateur déjà membre
// est ignoré → relançable sans effet de bord.
//
// Usage (SERVEUR ARRÊTÉ — DuckDB mono-writer sur shared_social) :
//
//	levelup backfill-squad-creators             # écrit
//	levelup backfill-squad-creators --dry-run   # liste sans écrire
//	levelup backfill-squad-creators --title halo_5

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/duckdb"
	prestigedb "levelup/go-api/internal/platform/duckdb/prestige"
	"levelup/go-api/internal/prestige"
)

type squadCreatorRow struct {
	id        string
	createdBy string
}

func runBackfillSquadCreators(cfg *config.AppConfig, args []string) error {
	fs := flag.NewFlagSet("backfill-squad-creators", flag.ExitOnError)
	titleSlug := fs.String("title", titlePkg.DefaultSlug, "slug du titre (shared_social ciblé)")
	dryRun := fs.Bool("dry-run", false, "affiche les insertions sans écrire")
	if err := fs.Parse(args); err != nil {
		return err
	}
	ctx := context.Background()

	resolver := titlePkg.NewPathResolver(cfg.RepoRoot)
	socialPath := resolver.SharedSocialDBPath(*titleSlug)
	if _, err := os.Stat(socialPath); err != nil {
		return fmt.Errorf("shared_social introuvable (%s): %w", socialPath, err)
	}

	// db_profiles : slug -> xuid, xuid -> (slug canonique, gamertag).
	xuidBySlug, slugByXUID, gtByXUID, err := backfillPlayerMaps(cfg)
	if err != nil {
		return err
	}

	socialDB, err := duckdb.OpenReadWrite(socialPath)
	if err != nil {
		return fmt.Errorf("open shared_social RW (%s): %w (serveur arrêté ?)", socialPath, err)
	}
	defer socialDB.Close()

	squads, err := loadAllSquads(ctx, socialDB)
	if err != nil {
		return err
	}

	repo := prestigedb.NewPrestigeSquadRepo(socialDB)
	now := time.Now()
	var inserted, skipped, unresolved int
	for _, sq := range squads {
		creatorXUID := xuidBySlug[strings.ToLower(sq.createdBy)]
		if creatorXUID == "" {
			unresolved++
			slog.WarnContext(ctx, "backfill-squad-creators: created_by non résolu en xuid (skip)",
				"squad_id", sq.id, "created_by", sq.createdBy)
			continue
		}
		members, mErr := repo.ListMembers(ctx, sq.id)
		if mErr != nil {
			return fmt.Errorf("list members %s: %w", sq.id, mErr)
		}
		if squadHasMember(members, creatorXUID) {
			skipped++
			continue
		}
		userID := slugByXUID[creatorXUID]
		if userID == "" {
			userID = sq.createdBy
		}
		if *dryRun {
			fmt.Printf("[dry-run] squad %s : insérerait le créateur xuid=%s user_id=%s\n",
				sq.id, creatorXUID, userID)
			inserted++
			continue
		}
		member := prestige.SquadMember{
			SquadID:  sq.id,
			Xuid:     creatorXUID,
			UserID:   userID,
			Gamertag: gtByXUID[creatorXUID],
			JoinedAt: now,
		}
		if aErr := repo.AddMember(ctx, member); aErr != nil {
			return fmt.Errorf("add creator to squad %s: %w", sq.id, aErr)
		}
		slog.InfoContext(ctx, "backfill-squad-creators: créateur réinséré (append-only)",
			"squad_id", sq.id, "xuid", creatorXUID, "user_id", userID)
		inserted++
	}

	fmt.Printf("backfill-squad-creators OK (title=%s dry_run=%v): squads=%d inserted=%d skipped=%d unresolved=%d\n",
		*titleSlug, *dryRun, len(squads), inserted, skipped, unresolved)
	return nil
}

// backfillPlayerMaps construit les annuaires db_profiles (slug<->xuid, xuid->gamertag).
func backfillPlayerMaps(cfg *config.AppConfig) (xuidBySlug, slugByXUID, gtByXUID map[string]string, err error) {
	players, err := cfg.LoadPlayers()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("load players: %w", err)
	}
	xuidBySlug = map[string]string{}
	slugByXUID = map[string]string{}
	gtByXUID = map[string]string{}
	for _, p := range players {
		if p.XUID == "" {
			continue
		}
		if p.PlayerSlug != "" {
			xuidBySlug[strings.ToLower(p.PlayerSlug)] = p.XUID
			slugByXUID[p.XUID] = p.PlayerSlug
		}
		if p.Gamertag != "" {
			gtByXUID[p.XUID] = p.Gamertag
		}
	}
	return xuidBySlug, slugByXUID, gtByXUID, nil
}

// loadAllSquads énumère toutes les escouades (lecture sur le handle RW du CLI —
// writer unique, serveur arrêté).
func loadAllSquads(ctx context.Context, db *duckdb.DB) ([]squadCreatorRow, error) {
	rows, err := db.QueryRecovered(ctx, "SELECT id, created_by FROM squad")
	if err != nil {
		return nil, fmt.Errorf("list squads (table absente ou DB non initialisée ?): %w", err)
	}
	defer rows.Close()
	var out []squadCreatorRow
	for rows.Next() {
		var s squadCreatorRow
		if scanErr := rows.Scan(&s.id, &s.createdBy); scanErr != nil {
			return nil, fmt.Errorf("scan squad: %w", scanErr)
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// squadHasMember indique si un xuid figure déjà dans le roster courant.
func squadHasMember(members []prestige.SquadMember, xuid string) bool {
	for _, m := range members {
		if m.Xuid == xuid {
			return true
		}
	}
	return false
}
