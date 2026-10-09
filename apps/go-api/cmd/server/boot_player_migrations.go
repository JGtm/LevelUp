package main

// boot_player_migrations.go — migrations des bases JOUEUR au boot, titre par titre.
//
// RW-EXCLUSIF, AVANT que le provider, le scheduler et le watcher n'ouvrent les bases joueur.
// Sans cet appel, une base joueur ne recevrait que EnsurePlayerSchema (CREATE TABLE IF NOT
// EXISTS), sans effet sur une table existante : la PRIMARY KEY n'était jamais ajoutée et les
// écritures ON CONFLICT échouaient en Binder Error. Idempotent (schema_migrations).
//
// CHAQUE PROFIL MIGRE LA BASE DE SON TITRE : db_profiles.json déclare un bloc par titre et
// `LoadPlayers()` rend un profil par couple (titre, gamertag), avec son `TitleSlug`. Le chemin
// vient du PathResolver pour CE titre et le jeu de migrations de `migration.RunForTitleDB` pour
// CE titre (son jeu propre s'il en enregistre un, sinon le jeu par défaut) — aucune comparaison
// de slug.

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/platform/duckdb"
)

// playerMigrationRunner applique les migrations player du titre `titleSlug` à une base.
type playerMigrationRunner func(playerDBPath, titleSlug string) error

// migratePlayerDBs migre la base joueur de chaque profil déclaré, sous le titre de son bloc.
// Profil de démo, sans gamertag ou sans base (compte token-only : la création appartient au
// chemin sync/onboarding) : ignoré. Un échec est journalisé et n'interrompt ni la boucle ni le
// boot. Rend le nombre de bases migrées sans erreur.
//
// `fallbackSlug` ne sert qu'à un profil sans titre (format v2.1, dont `LoadPlayers` pose
// déjà le titre par défaut) : le titre déclaré prime toujours.
func migratePlayerDBs(ctx context.Context, players []domain.PlayerSummary, pr *title.PathResolver,
	fallbackSlug string, run playerMigrationRunner) int {
	migrated := 0
	for _, p := range players {
		if p.Gamertag == "" || p.IsDemo {
			continue
		}
		slug := p.TitleSlug
		if slug == "" {
			slug = fallbackSlug
		}
		dbPath := pr.PlayerDBPath(slug, p.Gamertag)
		if _, statErr := os.Stat(dbPath); statErr != nil {
			slog.DebugContext(ctx, "migrations player ignorées — player DB absente (compte token-only ?)",
				"gamertag", p.Gamertag, "titleSlug", slug, "db", dbPath)
			continue
		}
		if err := run(dbPath, slug); err != nil {
			slog.WarnContext(ctx, "migrations player échouées (non-fatal)",
				"gamertag", p.Gamertag, "titleSlug", slug, "err", err)
			continue
		}
		migrated++
	}
	return migrated
}

// RunPlayerMigrations applique les migrations player du titre `titleSlug` à une base
// individuelle (ouverture RW exclusive, fermée au retour).
func RunPlayerMigrations(playerDBPath, titleSlug string) error {
	db, err := duckdb.OpenReadWrite(playerDBPath)
	if err != nil {
		return fmt.Errorf("open player rw: %w", err)
	}
	defer db.Close()
	return migration.RunForTitleDB(db.SQLDb(), titleSlug, migration.TargetPlayer)
}
