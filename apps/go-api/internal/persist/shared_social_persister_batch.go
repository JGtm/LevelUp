package persist

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"
)

func (p *SharedSocialPersister) Persist(ctx context.Context, batch *SharedSocialBatch) error {
	if batch == nil || batch.IsEmpty() {
		return nil
	}

	start := time.Now()

	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("shared_social: begin tx: %w", err)
	}
	rollback := func() {
		if rbErr := tx.Rollback(); rbErr != nil && rbErr != sql.ErrTxDone {
			slog.WarnContext(ctx, "shared_social: rollback failed (non-fatal)",
				"batch_id", batch.BatchID, "err", rbErr)
		}
	}

	if err := p.persistMediaFiles(ctx, tx, batch.MediaFiles); err != nil {
		rollback()
		return fmt.Errorf("shared_social: persist media_files: %w", err)
	}
	if err := p.persistMediaThumbnails(ctx, tx, batch.MediaThumbnails); err != nil {
		rollback()
		return fmt.Errorf("shared_social: persist media_thumbnails: %w", err)
	}
	if err := p.persistMediaAssociations(ctx, tx, batch.MediaAssociations); err != nil {
		rollback()
		return fmt.Errorf("shared_social: persist media_associations: %w", err)
	}
	if err := p.persistLikes(ctx, tx, batch.Likes, batch.LikesToRemove); err != nil {
		rollback()
		return fmt.Errorf("shared_social: persist likes: %w", err)
	}
	if err := p.persistFavorites(ctx, tx, batch.Favorites, batch.FavoritesToRemove); err != nil {
		rollback()
		return fmt.Errorf("shared_social: persist favorites: %w", err)
	}
	if err := p.persistNotifications(ctx, tx, batch.Notifications, batch.NotificationReads); err != nil {
		rollback()
		return fmt.Errorf("shared_social: persist notifications: %w", err)
	}
	if err := p.persistPlayerRecords(ctx, tx, batch.PlayerRecordsAppend); err != nil {
		rollback()
		return fmt.Errorf("shared_social: persist player_records: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("shared_social: commit: %w", err)
	}
	// Pas de CHECKPOINT ici : la connexion shared_social est ouverte
	// MaxOpenConns(4) (OpenReadWriteShared) — le motif historique « un CHECKPOINT
	// prendrait la seule connexion » est donc faux. Le flush WAL est couvert par
	// le CHECKPOINT scheduler périodique (5 min, main.go) + le CHECKPOINT synchrone
	// au shutdown (workers arrêtés). Fenêtre d'exposition WAL bornée à 5 min,
	// intentionnelle ; les écritures critiques (likes/favoris/associations média)
	// utilisent CommitWithCheckpoint pour un flush immédiat.

	slog.InfoContext(ctx, "shared_social: batch persisted",
		"batch_id", batch.BatchID,
		"source", batch.Source,
		"media_files", len(batch.MediaFiles),
		"media_associations", len(batch.MediaAssociations),
		"media_thumbnails", len(batch.MediaThumbnails),
		"likes", len(batch.Likes),
		"likes_removed", len(batch.LikesToRemove),
		"favorites", len(batch.Favorites),
		"favorites_removed", len(batch.FavoritesToRemove),
		"notifications", len(batch.Notifications),
		"notification_reads", len(batch.NotificationReads),
		"player_records_appended", len(batch.PlayerRecordsAppend),
		"duration_ms", time.Since(start).Milliseconds())

	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Persistence helpers (1 par table, sans logique métier)
// ─────────────────────────────────────────────────────────────────────────────

func (p *SharedSocialPersister) persistMediaFiles(ctx context.Context, tx *sql.Tx, rows []MediaFileInsert) error {
	if len(rows) == 0 {
		return nil
	}
	// Dédup applicative file_path : l'ex-contrainte UNIQUE(file_path) a été retirée
	// pour éradiquer le bug ART DuckDB #23046 (cf. media_files_drop_filepath_unique_v1).
	// SELECT-then-INSERT — skip si le file_path est déjà indexé (re-upload même contenu).
	sel, err := tx.PrepareContext(ctx, `SELECT 1 FROM media_files WHERE file_path = ? LIMIT 1`)
	if err != nil {
		return err
	}
	defer sel.Close()
	ins, err := tx.PrepareContext(ctx, `
		INSERT INTO media_files (
			player_slug, file_path, file_name, file_stem, file_ext, file_hash, kind,
			capture_start_utc, capture_end_utc, duration_seconds
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer ins.Close()
	for _, r := range rows {
		var dup int
		switch err := sel.QueryRowContext(ctx, r.FilePath).Scan(&dup); err {
		case nil:
			continue // file_path déjà indexé → skip
		case sql.ErrNoRows:
			// pas de doublon → INSERT
		default:
			return fmt.Errorf("media_file dedup %s: %w", r.FilePath, err)
		}
		if _, err := ins.ExecContext(ctx,
			r.PlayerSlug, r.FilePath, r.FileName, r.FileStem, r.FileExt, r.FileHash, r.Kind,
			r.CaptureStartUTC, r.CaptureEndUTC, r.DurationSeconds,
		); err != nil {
			return fmt.Errorf("media_file %s: %w", r.FilePath, err)
		}
	}
	return nil
}

func (p *SharedSocialPersister) persistMediaThumbnails(ctx context.Context, tx *sql.Tx, rows []MediaThumbnailUpdate) error {
	if len(rows) == 0 {
		return nil
	}
	// UPDATE sur PK INTEGER (id auto-increment) — pas de pression ART
	// contrairement aux UPDATE sur PK VARCHAR.
	stmt, err := tx.PrepareContext(ctx, `
		UPDATE media_files SET thumbnail_path = ? WHERE id = ? AND thumbnail_path IS NULL
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, r := range rows {
		if _, err := stmt.ExecContext(ctx, r.ThumbnailPath, r.MediaFileID); err != nil {
			return fmt.Errorf("thumbnail media_id=%d: %w", r.MediaFileID, err)
		}
	}
	return nil
}

func (p *SharedSocialPersister) persistMediaAssociations(ctx context.Context, tx *sql.Tx, rows []MediaAssociationInsert) error {
	if len(rows) == 0 {
		return nil
	}
	// APPEND-ONLY : auto-association live sync = INSERT event (is_manual=FALSE) dans
	// _history. Plus d'INSERT OR IGNORE sur la table legacy. La dédup est assurée par
	// loadUnassociatedMedia (forward-only) + la vue _latest.
	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO media_match_associations_history
			(media_file_id, match_id, delta_seconds, is_manual, is_active, associated_at, written_at)
		VALUES (?, ?, ?, FALSE, TRUE, CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP), CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP))
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, r := range rows {
		if _, err := stmt.ExecContext(ctx, r.MediaFileID, r.MatchID, r.DeltaSeconds); err != nil {
			return fmt.Errorf("assoc media=%d match=%s: %w", r.MediaFileID, r.MatchID, err)
		}
	}
	return nil
}

// persistLikes — APPEND-ONLY : ajout/retrait = INSERT pur dans media_likes_history
// (is_liked TRUE/FALSE). Plus aucun DELETE ni ON CONFLICT (surface ART éliminée).
// État courant lu via media_likes_latest.
func (p *SharedSocialPersister) persistLikes(ctx context.Context, tx *sql.Tx, adds []LikeInsert, removes []LikeRemove) error {
	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO media_likes_history (media_path, liker_slug, liker_gamertag, is_liked, liked_at)
		VALUES (?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, r := range adds {
		if _, err := stmt.ExecContext(ctx, r.MediaPath, r.LikerSlug, r.LikerGamertag, true, r.LikedAt); err != nil {
			return fmt.Errorf("like add %s/%s: %w", r.MediaPath, r.LikerSlug, err)
		}
	}
	for _, r := range removes {
		// Event de retrait : is_liked=FALSE, gamertag/liked_at NULL.
		if _, err := stmt.ExecContext(ctx, r.MediaPath, r.LikerSlug, nil, false, nil); err != nil {
			return fmt.Errorf("like remove %s/%s: %w", r.MediaPath, r.LikerSlug, err)
		}
	}
	return nil
}

// persistFavorites — APPEND-ONLY : chaque ajout/retrait = un INSERT pur dans
// match_favorites_history (is_favorite TRUE/FALSE). Plus AUCUN DELETE (surface ART
// éliminée sur shared_social). L'état courant se lit via la vue match_favorites_latest.
func (p *SharedSocialPersister) persistFavorites(ctx context.Context, tx *sql.Tx, adds []FavoriteInsert, removes []FavoriteRemove) error {
	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO match_favorites_history (player_slug, match_id, is_favorite, favorited_at)
		VALUES (?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, r := range adds {
		if _, err := stmt.ExecContext(ctx, r.PlayerSlug, r.MatchID, true, r.FavoritedAt); err != nil {
			return fmt.Errorf("favorite add %s/%s: %w", r.PlayerSlug, r.MatchID, err)
		}
	}
	for _, r := range removes {
		// Event de retrait : is_favorite=FALSE, favorited_at NULL.
		if _, err := stmt.ExecContext(ctx, r.PlayerSlug, r.MatchID, false, nil); err != nil {
			return fmt.Errorf("favorite remove %s/%s: %w", r.PlayerSlug, r.MatchID, err)
		}
	}
	return nil
}

func (p *SharedSocialPersister) persistNotifications(ctx context.Context, tx *sql.Tx, adds []NotificationInsert, reads []NotificationReadUpdate) error {
	if len(adds) > 0 {
		// APPEND-ONLY : INSERT pur d'un event create (read_at NULL, is_deleted FALSE)
		// dans player_notifications_history. La vue _latest expose l'état courant.
		stmt, err := tx.PrepareContext(ctx, `
			INSERT INTO player_notifications_history (
				xuid, id, category, severity, title_key, body_key, params,
				target_route, target_search, actor_xuid, actor_name, source,
				created_at, read_at, is_deleted, written_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NULL, FALSE, CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP))
		`)
		if err != nil {
			return err
		}
		for _, r := range adds {
			if _, err := stmt.ExecContext(ctx,
				r.XUID, r.ID, r.Category, r.Severity, r.TitleKey, r.BodyKey, r.Params,
				r.TargetRoute, r.TargetSearch, r.ActorXUID, r.ActorName, r.Source, r.CreatedAt,
			); err != nil {
				stmt.Close()
				return fmt.Errorf("notification insert xuid=%s id=%d: %w", r.XUID, r.ID, err)
			}
		}
		stmt.Close()
	}
	if len(reads) > 0 {
		// APPEND-ONLY : event read = INSERT…SELECT carry-forward du payload depuis
		// _latest avec read_at positionné (plus d'UPDATE in-place).
		stmt, err := tx.PrepareContext(ctx, `
			INSERT INTO player_notifications_history (
				xuid, id, category, severity, title_key, body_key, params,
				target_route, target_search, actor_xuid, actor_name, source,
				created_at, read_at, is_deleted, written_at
			)
			SELECT xuid, id, category, severity, title_key, body_key, params,
			       target_route, target_search, actor_xuid, actor_name, source,
			       created_at, ?, FALSE, CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
			FROM player_notifications_latest
			WHERE xuid = ? AND id = ?
		`)
		if err != nil {
			return err
		}
		for _, r := range reads {
			if _, err := stmt.ExecContext(ctx, r.ReadAt, r.XUID, r.ID); err != nil {
				stmt.Close()
				return fmt.Errorf("notification read xuid=%s id=%d: %w", r.XUID, r.ID, err)
			}
		}
		stmt.Close()
	}
	return nil
}

func (p *SharedSocialPersister) persistPlayerRecords(ctx context.Context, tx *sql.Tx, rows []PlayerRecordAppend) error {
	if len(rows) == 0 {
		return nil
	}
	// Pattern append-only (Phase 2) : insert dans player_records_history.
	// La vue player_records_latest expose la valeur courante.
	// Si la migration Phase 2 n'est pas encore appliquée, on tombe sur
	// player_records (table d'origine) — comportement legacy compatible.
	//
	// Détection one-shot (memoïsée) de player_records_history via information_schema,
	// au lieu d'un échec de Prepare silencieux à chaque batch. Rend le fallback
	// legacy explicite (WARN-once). Phase 2 supprimera ce fallback.
	if !p.playerRecordsHistoryExists(ctx) {
		p.legacyRecordsWarnOnce.Do(func() {
			slog.WarnContext(ctx, "legacy_player_records_upsert_used",
				"reason", "player_records_history absent (migration Phase 2 non appliquée)",
				"fallback", "player_records ON CONFLICT DO UPDATE")
		})
		return p.persistPlayerRecordsLegacy(ctx, tx, rows)
	}
	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO player_records_history (xuid, metric, period, value, achieved_at, achieved_match_id, previous_value, previous_achieved_at, written_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		// Filet secondaire : la sonde a dit « présente » mais le Prepare échoue
		// (schéma incohérent / table droppée entre-temps). On retombe sur le
		// chemin legacy + WARN une fois, sans perdre l'écriture.
		p.legacyRecordsWarnOnce.Do(func() {
			slog.WarnContext(ctx, "legacy_player_records_upsert_used",
				"reason", "prepare player_records_history a échoué malgré détection positive",
				"err", err)
		})
		return p.persistPlayerRecordsLegacy(ctx, tx, rows)
	}
	defer stmt.Close()
	for _, r := range rows {
		period := r.Period
		if period == "" {
			period = "all_time"
		}
		writtenAt := r.WrittenAt
		if writtenAt.IsZero() {
			writtenAt = time.Now().UTC()
		}
		if _, err := stmt.ExecContext(ctx, r.XUID, r.Metric, period, r.Value, r.AchievedAt, r.AchievedMatchID, r.PreviousValue, r.PreviousAchievedAt, writtenAt); err != nil {
			return fmt.Errorf("player_record_append xuid=%s metric=%s: %w", r.XUID, r.Metric, err)
		}
	}
	return nil
}

// playerRecordsHistoryExists détecte UNE SEULE FOIS (sync.Once) la présence de la
// table append-only player_records_history via information_schema. La sonde tourne
// sur p.db (HORS transaction d'écriture) : information_schema est en lecture seule
// et une erreur de sonde ne doit jamais polluer la tx courante. En cas d'erreur,
// renvoie false (→ chemin legacy, conservateur).
func (p *SharedSocialPersister) playerRecordsHistoryExists(ctx context.Context) bool {
	p.recordsDetectOnce.Do(func() {
		// Sonde DÉCOUPLÉE du ctx appelant (Background + timeout court) : une
		// annulation/timeout de l'écriture en cours ne doit pas figer
		// définitivement (sync.Once) la détection sur "legacy". Read-only.
		probeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		var n int
		err := p.db.QueryRowContext(probeCtx,
			"SELECT COUNT(*) FROM information_schema.tables WHERE table_schema='main' AND table_name='player_records_history'",
		).Scan(&n)
		if err != nil {
			slog.WarnContext(ctx, "shared_social: sonde player_records_history échouée, fallback legacy", "err", err)
			p.recordsHistoryExists = false
			return
		}
		p.recordsHistoryExists = n > 0
	})
	return p.recordsHistoryExists
}

// persistPlayerRecordsLegacy : chemin de compatibilité tant que la migration
// Phase 2 (append-only) n'est pas appliquée. Sera supprimé après Phase 2.
func (p *SharedSocialPersister) persistPlayerRecordsLegacy(ctx context.Context, tx *sql.Tx, rows []PlayerRecordAppend) error {
	// ART-safe : SELECT-then-UPDATE-or-INSERT (plus d'ON CONFLICT DO UPDATE, qui
	// réécrit via l'index ART de la PK sur shared_social). Chemin de compat sur
	// player_records (legacy) — jamais atteint en prod (la migration _history tourne
	// au boot). player_records sans index secondaire → UPDATE non-indexé sûr.
	//
	// updated_at en UTC EXPLICITE (lot S6) : `NOW()` nu rend un TIMESTAMPTZ coercé par le
	// fuseau de SESSION, alors qu'`achieved_at` de la MEME ligne est une valeur Go déjà en
	// UTC — la ligne se serait contredite d'un offset. Le chemin append-only équivalent
	// (persistPlayerRecordsHistory) est déjà canonique depuis S5 : les deux chemins d'une
	// même table doivent dater sur une seule horloge, sans quoi une base migrée en cours de
	// route mélangerait les deux référentiels.
	selStmt, err := tx.PrepareContext(ctx, `SELECT 1 FROM player_records WHERE xuid = ? AND metric = ?`)
	if err != nil {
		return err
	}
	defer selStmt.Close()
	updStmt, err := tx.PrepareContext(ctx, `
		UPDATE player_records SET value = ?, achieved_at = ?, achieved_match_id = ?,
			updated_at = CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP)
		WHERE xuid = ? AND metric = ?`)
	if err != nil {
		return err
	}
	defer updStmt.Close()
	insStmt, err := tx.PrepareContext(ctx, `
		INSERT INTO player_records (xuid, metric, value, achieved_at, achieved_match_id, updated_at)
		VALUES (?, ?, ?, ?, ?, CAST(now() AT TIME ZONE 'UTC' AS TIMESTAMP))`)
	if err != nil {
		return err
	}
	defer insStmt.Close()
	for _, r := range rows {
		var dummy int
		serr := selStmt.QueryRowContext(ctx, r.XUID, r.Metric).Scan(&dummy)
		switch {
		case serr == nil:
			_, serr = updStmt.ExecContext(ctx, r.Value, r.AchievedAt, r.AchievedMatchID, r.XUID, r.Metric)
		case errors.Is(serr, sql.ErrNoRows):
			_, serr = insStmt.ExecContext(ctx, r.XUID, r.Metric, r.Value, r.AchievedAt, r.AchievedMatchID)
		}
		if serr != nil {
			return fmt.Errorf("player_record_legacy xuid=%s metric=%s: %w", r.XUID, r.Metric, serr)
		}
	}
	return nil
}
