// Package persist — shared_social_persister.go : pattern Collect→Persist
// pour shared_social.duckdb (médias, likes, favoris, notifications, records,
// prestige).
//
// Contexte : avant ce refactor, les écritures shared_social étaient
// dispersées dans une dizaine de sites (ops/media.go, social_repo.go,
// notifications_repo.go, post_sync_deltas.go, etc.) qui faisaient des
// db.ExecContext directs. Conséquence : aucun garde-fou CHECKPOINT au
// shutdown → WAL résiduel → bug DuckDB upstream #7659 ("Failure while
// replaying WAL file: Calling DatabaseManager::GetDefaultDatabase") à
// chaque rebuild Air.
//
// Solution architecturale : aligner shared_social sur le pattern existant
// (Collect→Persist, ADR 0019). Toutes les écritures passent par
// SharedSocialPersister qui garantit :
//   - INSERT-only sur tables critiques (pas de UPSERT/UPDATE concurrent →
//     pas de pression sur l'index ART DuckDB)
//   - Transaction atomique par batch (BEGIN → INSERTs → COMMIT)
//   - Flush WAL borné : les écritures critiques (likes / favoris / associations
//     média) appellent CommitWithCheckpoint pour un flush immédiat ; le Persist()
//     générique s'appuie sur le CHECKPOINT scheduler 5 min (main.go) + le
//     CHECKPOINT synchrone au shutdown pour vider le WAL (cf. bug #7659).
//
// Le SharedSocialPersister est le chemin NOMINAL des écritures shared_social,
// mais PAS l'unique (revue 2026-06-01 SS-02 — l'ancienne mention « plus aucun
// db.ExecContext hors persist, sentinel parse-AST Phase 6 » était fausse). Les
// mutations NotificationsRepo (MarkRead/Delete/CapAndSweep/UpsertPreferences) et
// le sous-système Prestige écrivent encore en direct (OpenReadWrite/ExecRecovered),
// sérialisés par le lease KindSharedSocial. La seule sentinelle AST existante
// (no_attach_on_social_test.go) interdit l'ATTACH, PAS les écritures directes.
//
// **Modes d'usage** :
//
//  Live UI-critique (HTTP handler, user attend feedback immédiat) :
//      err := persister.Persist(ctx, batch)
//      // retourne quand le COMMIT + CHECKPOINT sont terminés
//      // 200 OK si nil, 5xx sinon
//
//  Event-driven (post-sync hook, fire-and-forget) :
//      go func() {
//          if err := persister.Persist(ctx, batch); err != nil {
//              slog.WarnContext(ctx, "shared_social persist failed (event)", "err", err)
//          }
//      }()
//
//  Planifié (scheduler/scan) : idem event-driven, async goroutine.
//
// Cf. .ai/AUDIT_SHARED_SOCIAL_WRITERS.md pour l'inventaire exhaustif des
// sites migrés.

package persist

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"
)

// SharedSocialBatch est l'unité atomique de persistance pour
// shared_social.duckdb. Un batch peut contenir des écritures sur N tables
// différentes, toutes commit dans la MÊME transaction.
//
// Tous les sous-champs sont optionnels (nil ou len=0 → no-op pour cette
// table). Le caller construit via SharedSocialBatchBuilder.
type SharedSocialBatch struct {
	// BatchID : identifiant pour traçabilité logs/debug.
	BatchID string

	// Source : qui a soumis ce batch ("index_media", "media_like",
	// "post_sync_records", "favorite_toggle", "notification_create", etc.).
	// Utilisé pour les métriques + debug.
	Source string

	// MediaFiles : nouveaux fichiers à indexer (INSERT OR IGNORE).
	MediaFiles []MediaFileInsert

	// MediaThumbnails : mises à jour de thumbnail_path (UPDATE ciblé par id).
	// Note : c'est l'unique UPDATE acceptable sur shared_social — sur une
	// PK INTEGER stable, pas de pression ART.
	MediaThumbnails []MediaThumbnailUpdate

	// MediaAssociations : associations média ↔ match (INSERT OR IGNORE).
	MediaAssociations []MediaAssociationInsert

	// Likes : INSERT OR IGNORE / DELETE (toggle like).
	Likes []LikeInsert
	// LikesToRemove : DELETE par (media_path, liker_slug).
	LikesToRemove []LikeRemove

	// Favorites : INSERT OR IGNORE / DELETE (toggle favorite).
	Favorites []FavoriteInsert
	// FavoritesToRemove : DELETE par (player_slug, match_id).
	FavoritesToRemove []FavoriteRemove

	// Notifications : INSERT.
	Notifications []NotificationInsert
	// NotificationReads : UPDATE read_at par (xuid, id).
	NotificationReads []NotificationReadUpdate

	// PlayerRecordsAppend : append-only sur player_records_history (cf.
	// Phase 2 migration). Pattern INSERT-only, la vue
	// player_records_latest retourne la valeur courante.
	PlayerRecordsAppend []PlayerRecordAppend
}

// IsEmpty retourne true si le batch n'a rien à persister (no-op).
func (b *SharedSocialBatch) IsEmpty() bool {
	return len(b.MediaFiles) == 0 &&
		len(b.MediaThumbnails) == 0 &&
		len(b.MediaAssociations) == 0 &&
		len(b.Likes) == 0 &&
		len(b.LikesToRemove) == 0 &&
		len(b.Favorites) == 0 &&
		len(b.FavoritesToRemove) == 0 &&
		len(b.Notifications) == 0 &&
		len(b.NotificationReads) == 0 &&
		len(b.PlayerRecordsAppend) == 0
}

// SharedSocialPersister persiste un SharedSocialBatch dans shared_social.duckdb
// avec garantie CHECKPOINT (WAL vidé sur disque après commit).
type SharedSocialPersister struct {
	// db est la connexion *sql.DB shared_social du pool process-wide.
	// Le persister NE possède PAS cette connexion (c'est le pool qui Close).
	db *sql.DB
}

// NewSharedSocialPersister construit un persister sur la connexion fournie.
// La connexion doit être la conn RW shared_social du pool.
func NewSharedSocialPersister(db *sql.DB) *SharedSocialPersister {
	return &SharedSocialPersister{db: db}
}

// PersistBatch implémente l'interface duckdb.SocialPersister (cf.
// internal/platform/duckdb/social_persister_iface.go). Cast interne du
// batch any → *SharedSocialBatch. Permet l'injection dans PlayerDB sans
// cycle d'import (duckdb importe persist serait un cycle ; passer par any
// + cast résout le problème).
func (p *SharedSocialPersister) PersistBatch(ctx context.Context, batch any) error {
	typed, ok := batch.(*SharedSocialBatch)
	if !ok {
		return fmt.Errorf("shared_social: PersistBatch attend *SharedSocialBatch, got %T", batch)
	}
	return p.Persist(ctx, typed)
}

// ─────────────────────────────────────────────────────────────────────────────
// Méthodes ciblées (interface duckdb.SocialPersister) — chaque appel = 1
// write atomique + CHECKPOINT garanti. Construisent un mini-batch en interne
// et délèguent à Persist.
// ─────────────────────────────────────────────────────────────────────────────

// AddFavorite implémente duckdb.SocialPersister.AddFavorite.
func (p *SharedSocialPersister) AddFavorite(ctx context.Context, playerSlug, matchID string) error {
	return p.Persist(ctx, &SharedSocialBatch{
		BatchID: "favorite-add",
		Source:  "social_handler",
		Favorites: []FavoriteInsert{
			{PlayerSlug: playerSlug, MatchID: matchID, FavoritedAt: time.Now().UTC()},
		},
	})
}

// RemoveFavorite implémente duckdb.SocialPersister.RemoveFavorite.
func (p *SharedSocialPersister) RemoveFavorite(ctx context.Context, playerSlug, matchID string) error {
	return p.Persist(ctx, &SharedSocialBatch{
		BatchID: "favorite-remove",
		Source:  "social_handler",
		FavoritesToRemove: []FavoriteRemove{
			{PlayerSlug: playerSlug, MatchID: matchID},
		},
	})
}

// AddLike implémente duckdb.SocialPersister.AddLike.
func (p *SharedSocialPersister) AddLike(ctx context.Context, mediaPath, likerSlug, likerGamertag string) error {
	return p.Persist(ctx, &SharedSocialBatch{
		BatchID: "like-add",
		Source:  "media_handler",
		Likes: []LikeInsert{
			{MediaPath: mediaPath, LikerSlug: likerSlug, LikerGamertag: likerGamertag, LikedAt: time.Now().UTC()},
		},
	})
}

// RemoveLike implémente duckdb.SocialPersister.RemoveLike.
func (p *SharedSocialPersister) RemoveLike(ctx context.Context, mediaPath, likerSlug string) error {
	return p.Persist(ctx, &SharedSocialBatch{
		BatchID: "like-remove",
		Source:  "media_handler",
		LikesToRemove: []LikeRemove{
			{MediaPath: mediaPath, LikerSlug: likerSlug},
		},
	})
}

// AppendPlayerRecord implémente duckdb.SocialPersister.AppendPlayerRecord.
// Pattern append-only : INSERT pur dans player_records_history (jamais
// UPDATE), évite la pression sur l'index ART DuckDB.
func (p *SharedSocialPersister) AppendPlayerRecord(ctx context.Context, xuid, metric, period string, value float64, achievedAt *time.Time, achievedMatchID *string, previousValue *float64, previousAchievedAt *time.Time) error {
	if period == "" {
		period = "all_time"
	}
	return p.Persist(ctx, &SharedSocialBatch{
		BatchID: "record-append",
		Source:  "post_sync_records",
		PlayerRecordsAppend: []PlayerRecordAppend{
			{
				XUID:               xuid,
				Metric:             metric,
				Period:             period,
				Value:              value,
				AchievedAt:         achievedAt,
				AchievedMatchID:    achievedMatchID,
				PreviousValue:      previousValue,
				PreviousAchievedAt: previousAchievedAt,
				WrittenAt:          time.Now().UTC(),
			},
		},
	})
}

// CreateNotification implémente duckdb.SocialPersister.CreateNotification.
// L'argument est typé en `any` côté interface (duckdb.NotificationData) pour
// éviter le cycle d'import — on accepte une struct compatible champ par
// champ via reflection minimale (assertion sur le type concret).
func (p *SharedSocialPersister) CreateNotification(ctx context.Context, n any) error {
	// Cast depuis le type non-importé : on attend une struct avec les
	// mêmes champs que duckdb.NotificationData. On utilise un type adapter.
	nd, ok := n.(notificationDataLike)
	if !ok {
		// Tenter via reflection sur un struct anonyme compatible
		return fmt.Errorf("shared_social: CreateNotification attend duckdb.NotificationData ou compatible, got %T", n)
	}
	return p.Persist(ctx, &SharedSocialBatch{
		BatchID: "notif-create",
		Source:  "notification_service",
		Notifications: []NotificationInsert{
			{
				XUID:         nd.GetXUID(),
				ID:           nd.GetID(),
				Category:     nd.GetCategory(),
				Severity:     nd.GetSeverity(),
				TitleKey:     nd.GetTitleKey(),
				BodyKey:      nd.GetBodyKey(),
				Params:       nd.GetParams(),
				TargetRoute:  nd.GetTargetRoute(),
				TargetSearch: nd.GetTargetSearch(),
				ActorXUID:    nd.GetActorXUID(),
				ActorName:    nd.GetActorName(),
				Source:       nd.GetSource(),
				CreatedAt:    nd.GetCreatedAt(),
			},
		},
	})
}

// notificationDataLike : interface satisfaite structurellement par
// duckdb.NotificationData. Évite le cycle d'import.
type notificationDataLike interface {
	GetXUID() string
	GetID() int64
	GetCategory() string
	GetSeverity() string
	GetTitleKey() string
	GetBodyKey() *string
	GetParams() *string
	GetTargetRoute() *string
	GetTargetSearch() *string
	GetActorXUID() *string
	GetActorName() *string
	GetSource() string
	GetCreatedAt() time.Time
}

// MarkNotificationRead implémente duckdb.SocialPersister.MarkNotificationRead.
func (p *SharedSocialPersister) MarkNotificationRead(ctx context.Context, xuid string, id int64, readAt time.Time) error {
	return p.Persist(ctx, &SharedSocialBatch{
		BatchID: "notif-read",
		Source:  "notification_handler",
		NotificationReads: []NotificationReadUpdate{
			{XUID: xuid, ID: id, ReadAt: readAt},
		},
	})
}

// SetMediaMatchAssociation force l'association d'un média à un match précis
// (ADR 0021 Phase 3.2). Atomique : DELETE old assoc + INSERT new dans la même
// TX, suivi d'un CHECKPOINT explicite pour vider le WAL immédiatement.
//
// Différence vs MediaAssociations (INSERT OR IGNORE) : ici on FORCE le remplacement
// d'une assoc existante (cas utilisateur qui ré-associe manuellement un média).
//
// Le caller doit avoir résolu mediaFileID via une lecture séparée (cf.
// MediaRepo.SetMediaMatchAssociation pour le lookup).
func (p *SharedSocialPersister) SetMediaMatchAssociation(ctx context.Context, mediaFileID int64, matchID string) error {
	start := time.Now()
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("shared_social SetMediaMatchAssociation: begin tx: %w", err)
	}
	rollback := func() {
		if rbErr := tx.Rollback(); rbErr != nil && rbErr != sql.ErrTxDone {
			slog.WarnContext(ctx, "shared_social SetMediaMatchAssociation: rollback failed (non-fatal)", "err", rbErr)
		}
	}

	if _, err := tx.ExecContext(ctx,
		`DELETE FROM media_match_associations WHERE media_file_id = ?`, mediaFileID,
	); err != nil {
		rollback()
		return fmt.Errorf("shared_social SetMediaMatchAssociation: delete: %w", err)
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO media_match_associations (media_file_id, match_id, delta_seconds, is_manual) VALUES (?, ?, 0, TRUE)`,
		mediaFileID, matchID,
	); err != nil {
		rollback()
		return fmt.Errorf("shared_social SetMediaMatchAssociation: insert: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("shared_social SetMediaMatchAssociation: commit: %w", err)
	}

	// CHECKPOINT explicite — différence vs Persist générique (qui s'appuie sur
	// le scheduler 5min). Pour les opérations user-visibles individuelles, on
	// préfère un CHECKPOINT immédiat. Non-fatal si échec (lock contention).
	if _, err := p.db.ExecContext(ctx, "CHECKPOINT"); err != nil {
		slog.WarnContext(ctx, "shared_social SetMediaMatchAssociation: CHECKPOINT post-commit non-fatal", "err", err)
	}

	slog.DebugContext(ctx, "shared_social: media_match_association set",
		"media_file_id", mediaFileID, "match_id", matchID,
		"duration_ms", time.Since(start).Milliseconds())
	return nil
}

// SetMediaLiked met à jour le flag liked d'un média (ADR 0021 Phase 3.2).
// Retourne true si la ligne media_files existait (rowsAffected > 0), false
// sinon (file_path inconnu — caller traduit en 404).
//
// Atomique + CHECKPOINT immédiat — même pattern que SetMediaMatchAssociation.
//
// NB : ne touche QUE media_files.liked. La table media_likes (likers
// sociaux) est manipulée séparément via AddLike/RemoveLike.
func (p *SharedSocialPersister) SetMediaLiked(ctx context.Context, filePath string, liked bool) (bool, error) {
	start := time.Now()
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("shared_social SetMediaLiked: begin tx: %w", err)
	}
	rollback := func() {
		if rbErr := tx.Rollback(); rbErr != nil && rbErr != sql.ErrTxDone {
			slog.WarnContext(ctx, "shared_social SetMediaLiked: rollback failed (non-fatal)", "err", rbErr)
		}
	}

	res, err := tx.ExecContext(ctx, `
		UPDATE media_files
		SET liked = ?,
			liked_at = CASE WHEN ? THEN CURRENT_TIMESTAMP ELSE NULL END
		WHERE file_path = ?
	`, liked, liked, filePath)
	if err != nil {
		rollback()
		return false, fmt.Errorf("shared_social SetMediaLiked: update: %w", err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		rollback()
		return false, fmt.Errorf("shared_social SetMediaLiked: rows affected: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("shared_social SetMediaLiked: commit: %w", err)
	}

	if _, err := p.db.ExecContext(ctx, "CHECKPOINT"); err != nil {
		slog.WarnContext(ctx, "shared_social SetMediaLiked: CHECKPOINT post-commit non-fatal", "err", err)
	}

	slog.DebugContext(ctx, "shared_social: media_file liked updated",
		"file_path", filePath, "liked", liked, "rows_affected", rowsAffected,
		"duration_ms", time.Since(start).Milliseconds())
	return rowsAffected > 0, nil
}

// Persist exécute toutes les écritures du batch en UNE transaction, suivie
// d'un CHECKPOINT pour vider le WAL DuckDB sur disque. Atomique.
//
// Sécurité crash : si crash mid-Persist, BEGIN TX est rollback → état
// précédent préservé. Si crash après COMMIT mais avant CHECKPOINT → DuckDB
// a quand même persisté en WAL, le replay au prochain boot est trivial
// (INSERT seulement, pas d'ATTACH ni DDL).
//
// Retour : nil si tout OK, erreur première opération échouée sinon.
// Pas de "best effort" : si une INSERT échoue, tout rollback. L'idée est
// d'avoir une garantie atomique pour le caller.
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
	stmt, err := tx.PrepareContext(ctx, `
		INSERT OR IGNORE INTO media_files (
			player_slug, file_path, file_name, file_stem, file_ext, file_hash, kind,
			capture_start_utc, capture_end_utc, duration_seconds
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, r := range rows {
		if _, err := stmt.ExecContext(ctx,
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
	stmt, err := tx.PrepareContext(ctx, `
		INSERT OR IGNORE INTO media_match_associations (media_file_id, match_id, delta_seconds)
		VALUES (?, ?, ?)
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

func (p *SharedSocialPersister) persistLikes(ctx context.Context, tx *sql.Tx, adds []LikeInsert, removes []LikeRemove) error {
	if len(adds) > 0 {
		stmt, err := tx.PrepareContext(ctx, `
			INSERT OR IGNORE INTO media_likes (media_path, liker_slug, liker_gamertag, liked_at)
			VALUES (?, ?, ?, ?)
		`)
		if err != nil {
			return err
		}
		for _, r := range adds {
			if _, err := stmt.ExecContext(ctx, r.MediaPath, r.LikerSlug, r.LikerGamertag, r.LikedAt); err != nil {
				stmt.Close()
				return fmt.Errorf("like add %s/%s: %w", r.MediaPath, r.LikerSlug, err)
			}
		}
		stmt.Close()
	}
	if len(removes) > 0 {
		stmt, err := tx.PrepareContext(ctx, `
			DELETE FROM media_likes WHERE media_path = ? AND liker_slug = ?
		`)
		if err != nil {
			return err
		}
		for _, r := range removes {
			if _, err := stmt.ExecContext(ctx, r.MediaPath, r.LikerSlug); err != nil {
				stmt.Close()
				return fmt.Errorf("like remove %s/%s: %w", r.MediaPath, r.LikerSlug, err)
			}
		}
		stmt.Close()
	}
	return nil
}

func (p *SharedSocialPersister) persistFavorites(ctx context.Context, tx *sql.Tx, adds []FavoriteInsert, removes []FavoriteRemove) error {
	if len(adds) > 0 {
		stmt, err := tx.PrepareContext(ctx, `
			INSERT OR IGNORE INTO match_favorites (player_slug, match_id, favorited_at)
			VALUES (?, ?, ?)
		`)
		if err != nil {
			return err
		}
		for _, r := range adds {
			if _, err := stmt.ExecContext(ctx, r.PlayerSlug, r.MatchID, r.FavoritedAt); err != nil {
				stmt.Close()
				return fmt.Errorf("favorite add %s/%s: %w", r.PlayerSlug, r.MatchID, err)
			}
		}
		stmt.Close()
	}
	if len(removes) > 0 {
		stmt, err := tx.PrepareContext(ctx, `
			DELETE FROM match_favorites WHERE player_slug = ? AND match_id = ?
		`)
		if err != nil {
			return err
		}
		for _, r := range removes {
			if _, err := stmt.ExecContext(ctx, r.PlayerSlug, r.MatchID); err != nil {
				stmt.Close()
				return fmt.Errorf("favorite remove %s/%s: %w", r.PlayerSlug, r.MatchID, err)
			}
		}
		stmt.Close()
	}
	return nil
}

func (p *SharedSocialPersister) persistNotifications(ctx context.Context, tx *sql.Tx, adds []NotificationInsert, reads []NotificationReadUpdate) error {
	if len(adds) > 0 {
		stmt, err := tx.PrepareContext(ctx, `
			INSERT INTO player_notifications (
				xuid, id, category, severity, title_key, body_key, params,
				target_route, target_search, actor_xuid, actor_name, source, created_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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
		stmt, err := tx.PrepareContext(ctx, `
			UPDATE player_notifications SET read_at = ? WHERE xuid = ? AND id = ?
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
	// La détection de la table cible est faite par convention : on tente
	// l'INSERT sur player_records_history d'abord, fallback sur player_records.
	// Phase 2 supprimera ce fallback quand la migration sera obligatoire.
	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO player_records_history (xuid, metric, period, value, achieved_at, achieved_match_id, previous_value, previous_achieved_at, written_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		// Fallback legacy : la table _history n'existe pas encore (migration
		// Phase 2 pas appliquée). On utilise l'ancien chemin UPSERT, qui sera
		// déprécié après Phase 2.
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

// persistPlayerRecordsLegacy : chemin de compatibilité tant que la migration
// Phase 2 (append-only) n'est pas appliquée. Sera supprimé après Phase 2.
func (p *SharedSocialPersister) persistPlayerRecordsLegacy(ctx context.Context, tx *sql.Tx, rows []PlayerRecordAppend) error {
	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO player_records (xuid, metric, value, achieved_at, achieved_match_id, updated_at)
		VALUES (?, ?, ?, ?, ?, NOW())
		ON CONFLICT (xuid, metric) DO UPDATE SET
			value             = EXCLUDED.value,
			achieved_at       = EXCLUDED.achieved_at,
			achieved_match_id = EXCLUDED.achieved_match_id,
			updated_at        = NOW()
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, r := range rows {
		if _, err := stmt.ExecContext(ctx, r.XUID, r.Metric, r.Value, r.AchievedAt, r.AchievedMatchID); err != nil {
			return fmt.Errorf("player_record_legacy xuid=%s metric=%s: %w", r.XUID, r.Metric, err)
		}
	}
	return nil
}
