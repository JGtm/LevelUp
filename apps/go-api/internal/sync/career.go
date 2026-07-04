// Package sync — career.go : synchronisation du rang carrière.
//
// Portage de src/data/sync/_career.py.
// Récupère la progression de rang via l'API economy player-gated
// et insère un snapshot dans career_progression.
package sync

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
	"unicode"

	"levelup/go-api/internal/games/halo_infinite/rankedplaylists"
	duckdbpkg "levelup/go-api/internal/platform/duckdb"
)

// CareerRankData contient les données d'un snapshot de rang.
// Portage de src/data/sync/models.py CareerRankData.
type CareerRankData struct {
	XUID             string
	CurrentRank      int
	CurrentRankName  string
	CurrentRankTier  string
	CurrentXP        int
	XPForNextRank    int
	XPTotal          int
	IsMaxRank        bool
	AdornmentPath    string
	SpartanID        string
	BannerImageURL   string
	EmblemImageURL   string
	BackdropImageURL string
}

// syncCareerRank récupère la progression du rang carrière via le client Halo.
// Si le token joueur est absent, la sync est sautée proprement (nil, nil).
// Utilisée uniquement par career_integration_test.go (-tags=integration) ;
// le code prod passe par syncEngine.fetchCareerRank — preserve pour les
// tests d'intégration sans modification d'API.
//
//nolint:unused // utilisée par career_integration_test.go (tags=integration)
func syncCareerRank(
	ctx context.Context,
	client HaloClient,
	xuid string,
) (*CareerRankData, error) {
	// B4 : validation xuid — non vide, numérique, longueur typique 16 chiffres.
	if strings.TrimSpace(xuid) == "" {
		return nil, fmt.Errorf("syncCareerRank: xuid vide")
	}
	for _, r := range xuid {
		if !unicode.IsDigit(r) {
			return nil, fmt.Errorf("syncCareerRank: xuid doit être numérique (reçu %q)", xuid)
		}
	}
	if len(xuid) < 12 || len(xuid) > 20 {
		return nil, fmt.Errorf("syncCareerRank: longueur xuid inattendue %d (attendu 12-20 chiffres)", len(xuid))
	}
	return client.GetCareerRank(ctx, xuid)
}

// parseCareerRank extrait les données de rang depuis la réponse API.
// Utilisée par career_test.go et career_integration_test.go (tag integration).
//
//nolint:unused // référencée uniquement depuis les tests sous tag integration
func parseCareerRank(body map[string]interface{}, xuid string) *CareerRankData {
	data := &CareerRankData{XUID: xuid}

	if rank, ok := body["Rank"]; ok {
		if r, ok := rank.(map[string]interface{}); ok {
			if v, ok := r["Value"].(float64); ok {
				data.CurrentRank = int(v)
			}
			if v, ok := r["Name"].(string); ok {
				data.CurrentRankName = v
			}
			if v, ok := r["Tier"].(string); ok {
				data.CurrentRankTier = v
			}
		}
	}

	if xp, ok := body["RewardTrack"]; ok {
		if r, ok := xp.(map[string]interface{}); ok {
			if v, ok := r["CurrentProgress"].(float64); ok {
				data.CurrentXP = int(v)
			}
			if v, ok := r["NextLevelRequirement"].(float64); ok {
				data.XPForNextRank = int(v)
			}
			if v, ok := r["TotalEarned"].(float64); ok {
				data.XPTotal = int(v)
			}
			if v, ok := r["IsMaxRank"].(bool); ok {
				data.IsMaxRank = v
			}
		}
	}

	if v, ok := body["AdornmentPath"].(string); ok {
		data.AdornmentPath = v
	}
	if v, ok := body["SpartanId"].(string); ok {
		data.SpartanID = v
	}

	return data
}

// saveCareerRank insère un snapshot de progression dans la player DB.
//
//nolint:unused // utilisée par career_test.go + career_integration_test.go
func saveCareerRank(ctx context.Context, db *sql.DB, data *CareerRankData) error {
	now := time.Now().UTC()
	_, err := db.ExecContext(ctx, `
		INSERT INTO career_progression (
			xuid, rank, rank_name, rank_tier,
			current_xp, xp_for_next_rank, xp_total,
			is_max_rank, adornment_path, spartan_id,
			banner_image_url, emblem_image_url, backdrop_image_url, recorded_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		data.XUID, data.CurrentRank, data.CurrentRankName, data.CurrentRankTier,
		data.CurrentXP, data.XPForNextRank, data.XPTotal,
		data.IsMaxRank, data.AdornmentPath, data.SpartanID,
		data.BannerImageURL, data.EmblemImageURL, data.BackdropImageURL, now,
	)
	if err != nil {
		return fmt.Errorf("saveCareerRank: %w", err)
	}

	// Mettre à jour sync_meta
	_ = SetSyncMeta(ctx, db, "last_career_sync_at", now.Format(time.RFC3339))
	_ = SetSyncMeta(ctx, db, "current_rank", fmt.Sprintf("%d", data.CurrentRank))

	return nil
}

func enrichCareerRankFromMetadata(ctx context.Context, db *sql.DB, data *CareerRankData) error {
	if db == nil || data == nil {
		return nil
	}

	var (
		titleEN       string
		tierType      sql.NullString
		grade         sql.NullInt64
		xpRequired    int
		adornmentPath sql.NullString
	)
	err := db.QueryRowContext(ctx,
		`SELECT title_en, tier_type, grade, xp_required, adornment_icon_path
		 FROM career_ranks
		 WHERE rank_id = ?`,
		data.CurrentRank,
	).Scan(&titleEN, &tierType, &grade, &xpRequired, &adornmentPath)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("enrichCareerRankFromMetadata row: %w", err)
	}

	data.XPForNextRank = xpRequired
	if tierType.Valid {
		data.CurrentRankTier = strings.TrimSpace(tierType.String)
	}
	data.CurrentRankName = buildCareerRankName(titleEN, data.CurrentRankTier, grade)
	if adornmentPath.Valid {
		data.AdornmentPath = strings.TrimSpace(adornmentPath.String)
	}

	var completedXP int
	if err := db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(xp_required), 0)
		 FROM career_ranks
		 WHERE rank_id < ?`,
		data.CurrentRank,
	).Scan(&completedXP); err != nil {
		return fmt.Errorf("enrichCareerRankFromMetadata sum: %w", err)
	}
	data.XPTotal = completedXP + data.CurrentXP
	return nil
}

// syncPlayerCSRs récupère les classements CSR du joueur via l'API et les persiste.
// Retourne (nil, nil) si la saison est vide ou si l'API ne renvoie rien.
// Retourne la slice CSR pour permettre au caller d'alimenter playlists_catalog
// en parallèle d'autres opérations (cf. runCSRSnapshotSync → errgroup step 4).
func syncPlayerCSRs(
	ctx context.Context,
	client HaloClient,
	db *sql.DB,
	xuid, seasonID string,
	activePlaylists []rankedplaylists.Playlist,
) ([]PlayerPlaylistCSR, error) {
	if strings.TrimSpace(seasonID) == "" {
		return nil, nil
	}
	// 1. Player-level : playlists classées ENGAGÉES (comportement historique).
	csrs, err := client.GetPlayerCSRs(ctx, xuid, seasonID)
	if err != nil {
		// Best-effort : on bascule sur le fetch par-playlist (mécanisme Grunt) qui
		// couvre de toute façon les playlists classées actives.
		slog.WarnContext(ctx, "syncPlayerCSRs: GetPlayerCSRs échoué, fallback per-playlist", "err", err)
		csrs = nil
	}
	// 2. Compléter avec les playlists classées ACTIVES manquantes via l'endpoint
	//    par-playlist (/hi/playlist/{id}/csrs) — garantit la couverture de toutes
	//    les playlists classées de la saison sans dériver de l'historique.
	//    `activePlaylists` = actives découvertes par le cron (dynamique) ; vide →
	//    fallback rankedplaylists.Active() dans l'augment.
	csrs = augmentWithActiveRankedCSRs(ctx, client, xuid, seasonID, csrs, activePlaylists)
	if len(csrs) == 0 {
		return nil, nil
	}
	if _, err := SaveCSRSnapshots(ctx, db, csrs, seasonID); err != nil {
		return nil, err
	}
	return csrs, nil
}

// augmentWithActiveRankedCSRs ajoute à csrs les playlists classées ACTIVES
// (référence rankedplaylists) absentes du player-level, en interrogeant
// l'endpoint par-playlist (Grunt Skill.GetPlaylistCsr). Nom/queue/input viennent
// de la référence. Best-effort par playlist : une erreur n'interrompt pas.
//
// Les playlists pour lesquelles l'API ne renvoie aucune entrée (jamais jouées)
// sont volontairement ignorées : la lecture catalogue-first (GetCSRSnapshots)
// synthétise alors une ligne "Non classé" cohérente avec le seuil de la saison.
func augmentWithActiveRankedCSRs(
	ctx context.Context,
	client HaloClient,
	xuid, seasonID string,
	csrs []PlayerPlaylistCSR,
	activePlaylists []rankedplaylists.Playlist,
) []PlayerPlaylistCSR {
	// activePlaylists vide → fallback sur la référence statique (comportement
	// historique + titres sans cron classement). Sinon = actives dynamiques.
	if len(activePlaylists) == 0 {
		activePlaylists = rankedplaylists.Active()
	}
	seen := make(map[string]struct{}, len(csrs))
	for _, c := range csrs {
		seen[strings.ToLower(strings.TrimSpace(c.PlaylistID))] = struct{}{}
	}
	for _, pl := range activePlaylists {
		if _, ok := seen[strings.ToLower(pl.AssetID)]; ok {
			continue
		}
		res, err := client.GetPlaylistCsr(ctx, pl.AssetID, xuid, seasonID)
		if err != nil {
			slog.WarnContext(ctx, "augmentWithActiveRankedCSRs: GetPlaylistCsr échoué",
				"playlist", pl.AssetID, "err", err)
			continue
		}
		if res == nil {
			continue // pas d'entrée → catalogue-first affichera "Non classé"
		}
		res.PlaylistName = pl.NameEN
		res.Queue = pl.Queue
		res.Input = pl.Input
		csrs = append(csrs, *res)
	}
	return csrs
}

// seedPlaylistsCatalog insère les playlists ranked découvertes via l'API CSR
// dans playlists_catalog si elles n'y sont pas encore. Best-effort : les erreurs
// sont loggées mais n'interrompent pas le sync.
func seedPlaylistsCatalog(ctx context.Context, metaDB *sql.DB, csrs []PlayerPlaylistCSR, titleSlug string) {
	// playlists_catalog est un référentiel OPTIONNEL par titre : présent pour
	// halo_infinite, volontairement ABSENT pour halo_5 dont la metadata.duckdb est
	// isolée des référentiels HINF (is_ranked y dérive de la présence CSR, pas du
	// catalogue — cf. TestHalo5Metadata_IsolatedFromInfinite). Sans la table, no-op
	// silencieux plutôt qu'un WARN par playlist à chaque cycle de sync (le post-sync
	// CSR tourne pour tout titre fournissant des CSR, h5 inclus).
	var hasCatalog int
	if err := metaDB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'playlists_catalog'`,
	).Scan(&hasCatalog); err != nil || hasCatalog == 0 {
		return
	}
	now := time.Now().UTC()
	var seeded int
	for _, c := range csrs {
		id := strings.TrimSpace(c.PlaylistID)
		if id == "" {
			continue
		}
		name := strings.TrimSpace(c.PlaylistName)
		if name == "" || isUUIDLike(name) {
			name = id
		}
		// DO UPDATE (et non DO NOTHING) : ces playlists viennent de l'API CSR
		// Waypoint, elles sont donc classées par définition. Forcer is_ranked=TRUE
		// + experience='ranked' corrige toute ligne précédemment marquée FALSE
		// (bug récurrent). is_active/name ne sont pas écrasés ici — la référence
		// rankedplaylists (seed migration) en est la source de vérité.
		// Pattern ART-safe UPDATE-then-INSERT (CLAUDE.md) : pas d'ON CONFLICT —
		// celui-ci déclenche le bug ART DuckDB "Failed to delete all rows from
		// index" qui invalide toute la metadata.duckdb (cascade : milestone_catalog
		// devient illisible, fix 2026-05-30). UPDATE = colonnes non-clé ; INSERT si
		// absente. is_active/name_canonical non écrasés (réf rankedplaylists).
		res, err := metaDB.ExecContext(ctx, `
			UPDATE playlists_catalog
			SET experience = 'ranked', is_ranked = TRUE, last_seen_at = ?
			WHERE title_slug = ? AND playlist_asset_id = ?`,
			now, titleSlug, id,
		)
		if err != nil {
			slog.WarnContext(ctx, "seedPlaylistsCatalog: update échoué",
				"playlist_id", id, "titleSlug", titleSlug, "err", err)
			continue
		}
		if n, _ := res.RowsAffected(); n == 0 {
			if _, err := metaDB.ExecContext(ctx, `
				INSERT INTO playlists_catalog
				  (title_slug, playlist_asset_id, current_version_id, name_canonical,
				   experience, is_ranked, is_active, first_seen_at, last_seen_at)
				VALUES (?, ?, '', ?, 'ranked', TRUE, TRUE, ?, ?)`,
				titleSlug, id, name, now, now,
			); err != nil {
				slog.WarnContext(ctx, "seedPlaylistsCatalog: insert échoué",
					"playlist_id", id, "titleSlug", titleSlug, "err", err)
				continue
			}
		}
		seeded++
	}
	if seeded > 0 {
		slog.InfoContext(ctx, "sync: playlists catalog enrichi depuis CSR", "new", seeded, "titleSlug", titleSlug)
	}
}

// isUUIDLike retourne true si s ressemble à un UUID v4 (36 chars, 4 tirets).
func isUUIDLike(s string) bool {
	return len(s) == 36 && strings.Count(s, "-") == 4
}

// SaveCSRSnapshots insère des snapshots CSR dans player_csr_snapshots
// (INSERT pur, append-only — Phase 2.G du refactor ART). La lecture
// courante passe par la vue player_csr_snapshots_latest. Exporté : réutilisé par
// le hook CSR Halo 5 (livesync) qui persiste les classements arena par playlist.
func SaveCSRSnapshots(ctx context.Context, db *sql.DB, csrs []PlayerPlaylistCSR, seasonID string) (int, error) {
	now := time.Now().UTC()
	var inserted int
	for _, c := range csrs {
		if strings.TrimSpace(c.PlaylistID) == "" {
			continue
		}
		_, err := db.ExecContext(ctx, `
			INSERT INTO player_csr_snapshots (
				playlist_id, playlist_name, queue, input, season_id,
				current_value, current_tier, current_sub_tier, current_measurement_remaining,
				season_value, season_tier, season_sub_tier,
				alltime_value, alltime_tier, alltime_sub_tier,
				fetched_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			c.PlaylistID, c.PlaylistName, c.Queue, c.Input, seasonID,
			c.Current.Value, c.Current.Tier, c.Current.SubTier, c.Current.MeasurementMatchesRemaining,
			c.Season.Value, c.Season.Tier, c.Season.SubTier,
			c.AllTime.Value, c.AllTime.Tier, c.AllTime.SubTier,
			now,
		)
		if err != nil {
			return inserted, fmt.Errorf("SaveCSRSnapshots insert %s: %w", c.PlaylistID, err)
		}
		inserted++
	}
	return inserted, nil
}

//nolint:unused // utilisée par career_integration_test.go (tags=integration)
func openCareerMetadataDB(path string) (*duckdbpkg.DB, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("openCareerMetadataDB: path vide")
	}
	db, err := duckdbpkg.OpenReadOnly(path)
	if err != nil {
		return nil, fmt.Errorf("openCareerMetadataDB: %w", err)
	}
	return db, nil
}

func buildCareerRankName(titleEN, tierType string, grade sql.NullInt64) string {
	titleEN = strings.TrimSpace(titleEN)
	tierType = strings.TrimSpace(tierType)
	if titleEN == "" {
		return ""
	}
	parts := []string{titleEN}
	if tierType != "" {
		parts = append(parts, tierType)
	}
	if grade.Valid && grade.Int64 > 0 {
		parts = append(parts, fmt.Sprintf("%d", grade.Int64))
	}
	return strings.Join(parts, " ")
}
