// Package sync — achievements.go : synchronisation des achievements Xbox.
//
// Flow :
//  1. Un appel API par langue de config/locales.toml (étiquette Halo principale de la langue)
//  2. Fusion par achievement_id → un texte par langue (la langue de repli fait foi pour les
//     champs partagés)
//  3. Upsert dans metadata.xbox_achievement_definitions (champs non localisés), dans
//     metadata.asset_translations (noms et descriptions, une ligne par langue) et dans
//     player_achievements
//  4. Pré-warming optionnel des images (fire-and-forget si resolver non nil)
package sync

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/assets"
	"levelup/go-api/internal/i18n"
	"levelup/go-api/internal/migration"
	ddb "levelup/go-api/internal/platform/duckdb"
)

// PlayerAchievement contient les données fusionnées (une valeur par langue) d'un achievement.
type PlayerAchievement struct {
	AchievementID   string
	Name            i18n.Text
	Description     i18n.Text
	LockedDesc      i18n.Text
	Gamerscore      int
	ImageURL        string
	IsSecret        bool
	RarityCategory  string
	RarityPercent   float64
	Unlocked        bool
	UnlockedAt      time.Time
	CurrentProgress int
	TargetProgress  int
	XboxTitleID     string
	ServiceConfigID string
}

// SyncAchievements récupère les achievements du joueur dans chaque langue de
// config/locales.toml, fusionne les textes par langue, puis les écrit dans les deux DBs.
// titleID est le slug LevelUp du titre (ex: "halo_infinite") — stocké dans
// xbox_achievement_definitions pour permettre le filtrage multi-titres.
// resolver peut être nil (le pré-warming des images est alors ignoré).
func SyncAchievements(
	ctx context.Context,
	client XboxAchievementsClient,
	resolver assets.Resolver,
	metadataDB *sql.DB,
	playerDB *sql.DB,
	xuid string,
	titleID string,
) error {
	// Étape 1 : un appel API par langue (une erreur, quelle que soit la langue, abandonne la synchro).
	raw, err := fetchAchievementsByLocale(ctx, client, xuid)
	if err != nil {
		return err
	}

	// Étape 2 : fusion par achievement_id.
	merged := mergeAchievements(raw)
	slog.InfoContext(ctx, "achievements: fusion terminée", "xuid", xuid, "count", len(merged), "locales", len(raw))

	// Étape 3 : upserts (UPDATE-then-INSERT, ART-safe). Pas de purge des périmés :
	// le filtre SCID empêche toute nouvelle contamination cross-titre, et l'ancien
	// purgeStaleAchievementDefinitions (DELETE per-row sur l'index PK ART) était un
	// nettoyage historique vestigial (0 ligne en régime permanent) — retiré pour
	// éliminer cette surface ART du hot path (campagne append-only 2026-06-20).
	if err := upsertAchievementDefinitions(ctx, metadataDB, merged, titleID); err != nil {
		return fmt.Errorf("achievements: upsert definitions: %w", err)
	}
	if err := upsertPlayerAchievements(ctx, playerDB, merged); err != nil {
		return fmt.Errorf("achievements: upsert player progress: %w", err)
	}

	// Étape 4 : pré-warming des images (fire-and-forget).
	if resolver != nil {
		warmAchievementImages(ctx, resolver, merged, titleID)
	}

	return nil
}

// fetchAchievementsByLocale appelle l'API Xbox une fois par langue de config/locales.toml, avec
// l'étiquette Halo principale de la langue.
func fetchAchievementsByLocale(ctx context.Context, client XboxAchievementsClient, xuid string) (map[i18n.Locale][]PlayerAchievementRaw, error) {
	out := make(map[i18n.Locale][]PlayerAchievementRaw, len(i18n.Locales()))
	for _, l := range i18n.Locales() {
		tags := i18n.HaloTags(l)
		if len(tags) == 0 {
			return nil, fmt.Errorf("achievements: langue %s sans étiquette Halo", l)
		}
		slog.DebugContext(ctx, "achievements: récupération", "xuid", xuid, "locale", l, "tag", tags[0])
		list, err := client.GetPlayerAchievements(ctx, xuid, tags[0])
		if err != nil {
			return nil, fmt.Errorf("achievements: fetch %s: %w", l, err)
		}
		out[l] = list
	}
	return out, nil
}

// mergeAchievements fusionne les listes par langue en un slice portant un texte par langue.
// La liste de la langue de repli (i18n.Fallback) fait foi pour l'ensemble des identifiants et les
// champs partagés (dont XboxTitleID) ; une langue qui n'a pas l'identifiant, ou dont le texte est
// vide, ne renseigne rien (la lecture se replie alors sur la langue de repli).
func mergeAchievements(byLocale map[i18n.Locale][]PlayerAchievementRaw) []PlayerAchievement {
	index := make(map[i18n.Locale]map[string]PlayerAchievementRaw, len(byLocale))
	for l, list := range byLocale {
		m := make(map[string]PlayerAchievementRaw, len(list))
		for _, a := range list {
			m[a.ID] = a
		}
		index[l] = m
	}

	base := byLocale[i18n.Fallback]
	result := make([]PlayerAchievement, 0, len(base))
	for _, a := range base {
		pa := PlayerAchievement{
			AchievementID:   a.ID,
			Name:            i18n.Text{},
			Description:     i18n.Text{},
			LockedDesc:      i18n.Text{},
			Gamerscore:      a.Gamerscore,
			ImageURL:        a.ImageURL,
			IsSecret:        a.IsSecret,
			RarityCategory:  a.RarityCategory,
			RarityPercent:   a.RarityPercent,
			Unlocked:        a.Unlocked,
			UnlockedAt:      a.UnlockedAt,
			CurrentProgress: a.CurrentProgress,
			TargetProgress:  a.TargetProgress,
			XboxTitleID:     a.XboxTitleID,
			ServiceConfigID: a.ServiceConfigID,
		}
		for _, l := range i18n.Locales() {
			loc, ok := index[l][a.ID]
			if !ok {
				continue
			}
			setIfNotEmpty(pa.Name, l, loc.Name)
			setIfNotEmpty(pa.Description, l, loc.Description)
			setIfNotEmpty(pa.LockedDesc, l, loc.LockedDesc)
		}
		result = append(result, pa)
	}
	return result
}

// setIfNotEmpty inscrit v pour la langue l, sauf s'il est vide.
func setIfNotEmpty(t i18n.Text, l i18n.Locale, v string) {
	if v != "" {
		t[l] = v
	}
}

// upsertAchievementDefinitions écrit les définitions dans metadata.xbox_achievement_definitions
// (champs non localisés) et leurs textes dans metadata.asset_translations (types `achievement` et
// `achievement_locked`), dans une seule transaction.
// titleID (ex: "halo_infinite") est stocké pour permettre le filtrage multi-titres côté query.
func upsertAchievementDefinitions(ctx context.Context, db *sql.DB, achievements []PlayerAchievement, titleID string) error {
	if len(achievements) == 0 {
		return nil
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	// Pattern ART-safe UPDATE-then-INSERT (CLAUDE.md) : PAS d'ON CONFLICT —
	// celui-ci déclenche le bug ART DuckDB "Failed to delete all rows from index"
	// qui INVALIDE toute la metadata.duckdb (cascade : milestone_catalog devient
	// illisible). Colonnes UPDATE = non-clé. La concurrence est sérialisée par le
	// lease KindMetadata (cf. achievements_race_test.go) → pas de TOCTOU.
	const updQ = `
		UPDATE xbox_achievement_definitions SET
			gamerscore = ?, image_url = ?,
			is_secret = ?, rarity_category = ?, rarity_percent = ?, title_id = ?,
			xbox_title_id = ?, service_config_id = ?, fetched_at = CURRENT_TIMESTAMP
		WHERE achievement_id = ?`
	const insQ = `
		INSERT INTO xbox_achievement_definitions
			(achievement_id, gamerscore, image_url, is_secret,
			 rarity_category, rarity_percent, title_id, xbox_title_id, service_config_id, fetched_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`

	updStmt, err := tx.PrepareContext(ctx, updQ)
	if err != nil {
		return fmt.Errorf("prepare update definitions: %w", err)
	}
	defer updStmt.Close() //nolint:errcheck
	insStmt, err := tx.PrepareContext(ctx, insQ)
	if err != nil {
		return fmt.Errorf("prepare insert definitions: %w", err)
	}
	defer insStmt.Close() //nolint:errcheck

	for _, a := range achievements {
		var rarityPercent *float64
		if a.RarityPercent > 0 {
			v := a.RarityPercent
			rarityPercent = &v
		}
		res, err := updStmt.ExecContext(ctx,
			a.Gamerscore, a.ImageURL,
			a.IsSecret, a.RarityCategory, rarityPercent, titleID,
			a.XboxTitleID, a.ServiceConfigID, a.AchievementID,
		)
		if err != nil {
			return fmt.Errorf("update definition %s: %w", a.AchievementID, err)
		}
		if n, _ := res.RowsAffected(); n == 0 {
			if _, err := insStmt.ExecContext(ctx,
				a.AchievementID, a.Gamerscore, a.ImageURL, a.IsSecret,
				a.RarityCategory, rarityPercent, titleID, a.XboxTitleID, a.ServiceConfigID,
			); err != nil {
				return fmt.Errorf("insert definition %s: %w", a.AchievementID, err)
			}
		}
		if err := upsertAchievementTexts(ctx, tx, a); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// upsertAchievementTexts écrit les textes d'un succès : nom et description (type `achievement`),
// texte de succès verrouillé (type `achievement_locked`, rangé dans `name`).
func upsertAchievementTexts(ctx context.Context, tx *sql.Tx, a PlayerAchievement) error {
	if err := migration.UpsertAssetTexts(ctx, tx, ddb.AssetTypeAchievement, a.AchievementID, a.Name, a.Description); err != nil {
		return fmt.Errorf("texts of achievement %s: %w", a.AchievementID, err)
	}
	if err := migration.UpsertAssetTexts(ctx, tx, ddb.AssetTypeAchievementLocked, a.AchievementID, a.LockedDesc, nil); err != nil {
		return fmt.Errorf("locked texts of achievement %s: %w", a.AchievementID, err)
	}
	return nil
}

// upsertPlayerAchievements écrit la progression joueur dans player_achievements.
func upsertPlayerAchievements(ctx context.Context, db *sql.DB, achievements []PlayerAchievement) error {
	if len(achievements) == 0 {
		return nil
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	// Pattern ART-safe UPDATE-then-INSERT (cf. upsertAchievementDefinitions) :
	// pas d'ON CONFLICT (bug ART → invalidation metadata.duckdb). Sérialisé par
	// le lease KindMetadata.
	const updQ = `
		UPDATE player_achievements SET
			unlocked = ?, unlocked_at = ?, current_progress = ?, target_progress = ?,
			fetched_at = CURRENT_TIMESTAMP
		WHERE achievement_id = ?`
	const insQ = `
		INSERT INTO player_achievements
			(achievement_id, unlocked, unlocked_at, current_progress, target_progress, fetched_at)
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`

	updStmt, err := tx.PrepareContext(ctx, updQ)
	if err != nil {
		return fmt.Errorf("prepare update player achievements: %w", err)
	}
	defer updStmt.Close() //nolint:errcheck
	insStmt, err := tx.PrepareContext(ctx, insQ)
	if err != nil {
		return fmt.Errorf("prepare insert player achievements: %w", err)
	}
	defer insStmt.Close() //nolint:errcheck

	for _, a := range achievements {
		var unlockedAt *time.Time
		if !a.UnlockedAt.IsZero() {
			t := a.UnlockedAt
			unlockedAt = &t
		}
		var currentProgress, targetProgress *int
		if a.TargetProgress > 0 {
			currentProgress = &a.CurrentProgress
			targetProgress = &a.TargetProgress
		}
		res, err := updStmt.ExecContext(ctx,
			a.Unlocked, unlockedAt, currentProgress, targetProgress, a.AchievementID,
		)
		if err != nil {
			return fmt.Errorf("update player achievement %s: %w", a.AchievementID, err)
		}
		if n, _ := res.RowsAffected(); n == 0 {
			if _, err := insStmt.ExecContext(ctx,
				a.AchievementID, a.Unlocked, unlockedAt, currentProgress, targetProgress,
			); err != nil {
				return fmt.Errorf("insert player achievement %s: %w", a.AchievementID, err)
			}
		}
	}

	return tx.Commit()
}

// warmAchievementImages pré-chauffe le cache local des icônes d'achievements.
// Exécuté en goroutine (fire-and-forget) pour ne pas bloquer la sync. titleID
// (slug LevelUp du titre, threadé depuis SyncAchievements) scope le cache d'assets
// par titre — title-agnostic (C7), plus de "halo_infinite" figé qui mélangeait les
// icônes Halo 5 sous le namespace Infinite.
func warmAchievementImages(_ context.Context, resolver assets.Resolver, achievements []PlayerAchievement, titleID string) {
	refs := make([]assets.Ref, 0, len(achievements))
	for _, a := range achievements {
		if a.ImageURL != "" {
			refs = append(refs, assets.Ref{
				Kind:    assets.KindAchievementImage,
				TitleID: titleID,
				ID:      a.AchievementID,
			})
		}
	}
	if len(refs) == 0 {
		return
	}
	go func() {
		resolver.Warm(context.Background(), refs...)
	}()
}
