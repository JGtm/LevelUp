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
	"sync"
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

	// Détection one-shot de la table append-only player_records_history (migration
	// Phase 2). recordsHistoryExists n'est valide qu'après recordsDetectOnce.
	recordsDetectOnce    sync.Once
	recordsHistoryExists bool
	// legacyRecordsWarnOnce : un seul WARN 'legacy_player_records_upsert_used' par
	// instance (le chemin legacy peut être emprunté à chaque batch).
	legacyRecordsWarnOnce sync.Once
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
