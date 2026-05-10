// Package sync — achievements.go : synchronisation des achievements Xbox.
//
// Flow :
//  1. Deux appels API : GetPlayerAchievements("en-US") + GetPlayerAchievements("fr-FR")
//  2. Fusion par achievement_id → champs bilingues
//  3. Upsert dans metadata.xbox_achievement_definitions et player_achievements
//  4. Pré-warming optionnel des images (fire-and-forget si resolver non nil)
package sync

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"levelup/go-api/internal/assets"
)

// PlayerAchievement contient les données fusionnées (EN + FR) d'un achievement.
type PlayerAchievement struct {
	AchievementID   string
	NameEN          string
	NameFR          string
	DescriptionEN   string
	DescriptionFR   string
	LockedDescEN    string
	LockedDescFR    string
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

// SyncAchievements récupère les achievements du joueur en EN et FR,
// fusionne les données bilingues, puis les écrit dans les deux DBs.
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
	// Étape 1 : deux appels API (EN + FR).
	slog.DebugContext(ctx, "achievements: récupération EN", "xuid", xuid)
	enRaw, err := client.GetPlayerAchievements(ctx, xuid, "en-US")
	if err != nil {
		return fmt.Errorf("achievements: fetch EN: %w", err)
	}
	slog.DebugContext(ctx, "achievements: récupération FR", "xuid", xuid)
	frRaw, err := client.GetPlayerAchievements(ctx, xuid, "fr-FR")
	if err != nil {
		return fmt.Errorf("achievements: fetch FR: %w", err)
	}

	// Étape 2 : fusion par achievement_id.

	// Étape 3 : fusion par achievement_id.
	merged := mergeAchievements(enRaw, frRaw)
	slog.InfoContext(ctx, "achievements: fusion terminée", "xuid", xuid, "count", len(merged))

	// Étape 4 : upserts + purge des périmés.
	if err := upsertAchievementDefinitions(ctx, metadataDB, merged, titleID); err != nil {
		return fmt.Errorf("achievements: upsert definitions: %w", err)
	}
	if err := purgeStaleAchievementDefinitions(ctx, metadataDB, merged, titleID); err != nil {
		return fmt.Errorf("achievements: purge stale definitions: %w", err)
	}
	if err := upsertPlayerAchievements(ctx, playerDB, merged); err != nil {
		return fmt.Errorf("achievements: upsert player progress: %w", err)
	}

	// Étape 5 : pré-warming des images (fire-and-forget).
	if resolver != nil {
		warmAchievementImages(ctx, resolver, merged)
	}

	return nil
}

// mergeAchievements fusionne deux slices (EN + FR) en un slice bilingue.
// Les doublons sont ignorés (EN fait foi pour les champs partagés, dont XboxTitleID).
func mergeAchievements(en, fr []PlayerAchievementRaw) []PlayerAchievement {
	frMap := make(map[string]PlayerAchievementRaw, len(fr))
	for _, a := range fr {
		frMap[a.ID] = a
	}

	result := make([]PlayerAchievement, 0, len(en))
	for _, a := range en {
		pa := PlayerAchievement{
			AchievementID:   a.ID,
			NameEN:          a.Name,
			DescriptionEN:   a.Description,
			LockedDescEN:    a.LockedDesc,
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
		if f, ok := frMap[a.ID]; ok {
			pa.NameFR = f.Name
			pa.DescriptionFR = f.Description
			pa.LockedDescFR = f.LockedDesc
		}
		result = append(result, pa)
	}
	return result
}

// filterBySCID retient uniquement les achievements dont le ServiceConfigID correspond.
func filterBySCID(achievements []PlayerAchievementRaw, scid string) []PlayerAchievementRaw {
	out := make([]PlayerAchievementRaw, 0, len(achievements))
	for _, a := range achievements {
		if a.ServiceConfigID == scid {
			out = append(out, a)
		}
	}
	return out
}

// purgeStaleAchievementDefinitions supprime de metadata les achievements du titre
// qui ne font plus partie de la liste synchronisée (ex: achievements d'autres jeux
// Halo qui auraient été insérés avant l'introduction du filtre SCID).
func purgeStaleAchievementDefinitions(ctx context.Context, db *sql.DB, achievements []PlayerAchievement, titleID string) error {
	if len(achievements) == 0 {
		return nil
	}
	placeholders := make([]string, len(achievements))
	args := make([]any, len(achievements)+1)
	args[0] = titleID
	for i, a := range achievements {
		placeholders[i] = "?"
		args[i+1] = a.AchievementID
	}
	q := fmt.Sprintf(
		"DELETE FROM xbox_achievement_definitions WHERE title_id = ? AND achievement_id NOT IN (%s)",
		strings.Join(placeholders, ","),
	)
	res, err := db.ExecContext(ctx, q, args...)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n > 0 {
		slog.InfoContext(ctx, "achievements: périmés supprimés", "title_id", titleID, "deleted", n)
	}
	return nil
}

// upsertAchievementDefinitions écrit les définitions dans metadata.xbox_achievement_definitions.
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

	const q = `
		INSERT INTO xbox_achievement_definitions
			(achievement_id, name_en, name_fr, description_en, description_fr,
			 locked_desc_en, locked_desc_fr, gamerscore, image_url, is_secret,
			 rarity_category, rarity_percent, title_id, xbox_title_id, service_config_id, fetched_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT (achievement_id) DO UPDATE SET
			name_en           = excluded.name_en,
			name_fr           = excluded.name_fr,
			description_en    = excluded.description_en,
			description_fr    = excluded.description_fr,
			locked_desc_en    = excluded.locked_desc_en,
			locked_desc_fr    = excluded.locked_desc_fr,
			gamerscore        = excluded.gamerscore,
			image_url         = excluded.image_url,
			is_secret         = excluded.is_secret,  -- pragma: allowlist secret
			rarity_category   = excluded.rarity_category,
			rarity_percent    = excluded.rarity_percent,
			title_id          = excluded.title_id,
			xbox_title_id     = excluded.xbox_title_id,
			service_config_id = excluded.service_config_id,
			fetched_at        = excluded.fetched_at
	`

	stmt, err := tx.PrepareContext(ctx, q)
	if err != nil {
		return fmt.Errorf("prepare upsert definitions: %w", err)
	}
	defer stmt.Close() //nolint:errcheck

	for _, a := range achievements {
		var rarityPercent *float64
		if a.RarityPercent > 0 {
			v := a.RarityPercent
			rarityPercent = &v
		}
		if _, err := stmt.ExecContext(ctx,
			a.AchievementID, a.NameEN, a.NameFR,
			a.DescriptionEN, a.DescriptionFR,
			a.LockedDescEN, a.LockedDescFR,
			a.Gamerscore, a.ImageURL, a.IsSecret,
			a.RarityCategory, rarityPercent, titleID, a.XboxTitleID, a.ServiceConfigID,
		); err != nil {
			return fmt.Errorf("upsert definition %s: %w", a.AchievementID, err)
		}
	}

	return tx.Commit()
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

	const q = `
		INSERT INTO player_achievements
			(achievement_id, unlocked, unlocked_at, current_progress, target_progress, fetched_at)
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT (achievement_id) DO UPDATE SET
			unlocked         = excluded.unlocked,
			unlocked_at      = excluded.unlocked_at,
			current_progress = excluded.current_progress,
			target_progress  = excluded.target_progress,
			fetched_at       = excluded.fetched_at
	`

	stmt, err := tx.PrepareContext(ctx, q)
	if err != nil {
		return fmt.Errorf("prepare upsert player achievements: %w", err)
	}
	defer stmt.Close() //nolint:errcheck

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
		if _, err := stmt.ExecContext(ctx,
			a.AchievementID, a.Unlocked, unlockedAt,
			currentProgress, targetProgress,
		); err != nil {
			return fmt.Errorf("upsert player achievement %s: %w", a.AchievementID, err)
		}
	}

	return tx.Commit()
}

// warmAchievementImages pré-chauffe le cache local des icônes d'achievements.
// Exécuté en goroutine (fire-and-forget) pour ne pas bloquer la sync.
func warmAchievementImages(_ context.Context, resolver assets.Resolver, achievements []PlayerAchievement) {
	refs := make([]assets.Ref, 0, len(achievements))
	for _, a := range achievements {
		if a.ImageURL != "" {
			refs = append(refs, assets.Ref{
				Kind:    assets.KindAchievementImage,
				TitleID: "halo_infinite",
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
