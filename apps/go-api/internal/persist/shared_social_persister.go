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
	"strings"
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
//
// NB : délègue à Persist() qui NE CHECKPOINT PAS (flush différé scheduler 5min).
// Conservée pour rétrocompat ; les call sites user-facing doivent préférer
// MarkNotificationsRead (CHECKPOINT immédiat).
func (p *SharedSocialPersister) MarkNotificationRead(ctx context.Context, xuid string, id int64, readAt time.Time) error {
	return p.Persist(ctx, &SharedSocialBatch{
		BatchID: "notif-read",
		Source:  "notification_handler",
		NotificationReads: []NotificationReadUpdate{
			{XUID: xuid, ID: id, ReadAt: readAt},
		},
	})
}

// execNotifWriteCheckpointed exécute un write notif mono-instruction (UPDATE/
// DELETE) en TX atomique. Si checkpoint=true, fait un CHECKPOINT immédiat
// post-commit (non-fatal si échec — données déjà commit, scheduler 5min
// fera fallback). Renvoie le nombre de lignes affectées.
//
// Garantie de durabilité identique à dblease.LeasedWriter.CommitWithCheckpoint
// (ADR 0021 Phase 3.2 bis). Appelé exclusivement sous le lease KindSharedSocial
// tenu par notifications.Service.withWriter — pas d'acquisition de lease ici.
func (p *SharedSocialPersister) execNotifWriteCheckpointed(ctx context.Context, label string, checkpoint bool, query string, args ...any) (int64, error) {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("shared_social %s: begin tx: %w", label, err)
	}
	res, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil && rbErr != sql.ErrTxDone {
			slog.WarnContext(ctx, "shared_social "+label+": rollback failed (non-fatal)", "err", rbErr)
		}
		return 0, fmt.Errorf("shared_social %s: exec: %w", label, err)
	}
	n, _ := res.RowsAffected()
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("shared_social %s: commit: %w", label, err)
	}
	if checkpoint {
		if _, err := p.db.ExecContext(ctx, "CHECKPOINT"); err != nil {
			slog.WarnContext(ctx, "shared_social "+label+": CHECKPOINT post-commit non-fatal", "err", err)
		}
	}
	return n, nil
}

// MarkNotificationsRead marque N notifications comme lues en UNE transaction +
// CHECKPOINT immédiat. Ne touche que les non-lues (read_at IS NULL) pour rester
// idempotent et renvoyer un count signifiant. Implémente
// duckdb.SocialPersister.MarkNotificationsRead.
func (p *SharedSocialPersister) MarkNotificationsRead(ctx context.Context, xuid string, ids []int64, readAt time.Time) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	placeholders := make([]string, len(ids))
	args := make([]any, 0, len(ids)+2)
	args = append(args, readAt, xuid)
	for i, id := range ids {
		placeholders[i] = "?"
		args = append(args, id)
	}
	// APPEND-ONLY : INSERT…SELECT carry-forward du payload depuis _latest avec
	// read_at positionné (plus d'UPDATE). Ne cible que les non-lues (read_at IS
	// NULL) → RowsAffected = nb réellement marqué (idempotent).
	q := fmt.Sprintf(`
		INSERT INTO player_notifications_history (
			xuid, id, category, severity, title_key, body_key, params,
			target_route, target_search, actor_xuid, actor_name, source,
			created_at, read_at, is_deleted, written_at
		)
		SELECT xuid, id, category, severity, title_key, body_key, params,
		       target_route, target_search, actor_xuid, actor_name, source,
		       created_at, ?, FALSE, CURRENT_TIMESTAMP
		FROM player_notifications_latest
		WHERE xuid = ? AND read_at IS NULL AND id IN (%s)`,
		strings.Join(placeholders, ","),
	)
	return p.execNotifWriteCheckpointed(ctx, "notif-mark-read", true, q, args...)
}

// MarkNotificationUnread remet read_at = NULL (+ CHECKPOINT). Renvoie le nb de
// lignes affectées (0 = id inconnu, le caller traduit en ErrNotFound).
func (p *SharedSocialPersister) MarkNotificationUnread(ctx context.Context, xuid string, id int64) (int64, error) {
	// APPEND-ONLY : INSERT d'un event read_at=NULL (carry-forward payload depuis
	// _latest). RowsAffected = 1 si la notif existe (visible), 0 sinon → ErrNotFound.
	return p.execNotifWriteCheckpointed(ctx, "notif-mark-unread", true,
		`INSERT INTO player_notifications_history (
			xuid, id, category, severity, title_key, body_key, params,
			target_route, target_search, actor_xuid, actor_name, source,
			created_at, read_at, is_deleted, written_at
		)
		SELECT xuid, id, category, severity, title_key, body_key, params,
		       target_route, target_search, actor_xuid, actor_name, source,
		       created_at, NULL, FALSE, CURRENT_TIMESTAMP
		FROM player_notifications_latest
		WHERE xuid = ? AND id = ?`,
		xuid, id)
}

// MarkAllNotificationsRead applique read_at à toutes les non-lues du joueur
// (filtré par category si non vide) + CHECKPOINT immédiat.
func (p *SharedSocialPersister) MarkAllNotificationsRead(ctx context.Context, xuid, category string, readAt time.Time) (int64, error) {
	// APPEND-ONLY : un event read par notif non-lue (INSERT…SELECT depuis _latest).
	base := `
		INSERT INTO player_notifications_history (
			xuid, id, category, severity, title_key, body_key, params,
			target_route, target_search, actor_xuid, actor_name, source,
			created_at, read_at, is_deleted, written_at
		)
		SELECT xuid, id, category, severity, title_key, body_key, params,
		       target_route, target_search, actor_xuid, actor_name, source,
		       created_at, ?, FALSE, CURRENT_TIMESTAMP
		FROM player_notifications_latest
		WHERE xuid = ? AND read_at IS NULL`
	if category == "" {
		return p.execNotifWriteCheckpointed(ctx, "notif-mark-all-read", true, base, readAt, xuid)
	}
	return p.execNotifWriteCheckpointed(ctx, "notif-mark-all-read", true,
		base+` AND category = ?`, readAt, xuid, category)
}

// SweepStaleInfoNotificationsRead marque lues les notifs severity='info' non
// lues et plus anciennes que cutoff (expiry douce DP8). SANS CHECKPOINT immédiat :
// balayage idempotent, re-joué au prochain emit si perdu (même contrat que
// CapAndSweepNotifications). Renvoie le nb de notifs marquées.
func (p *SharedSocialPersister) SweepStaleInfoNotificationsRead(ctx context.Context, xuid string, cutoff time.Time) (int64, error) {
	// APPEND-ONLY : un event read par notif info périmée (INSERT…SELECT depuis _latest).
	return p.execNotifWriteCheckpointed(ctx, "notif-sweep-stale-info", false, `
		INSERT INTO player_notifications_history (
			xuid, id, category, severity, title_key, body_key, params,
			target_route, target_search, actor_xuid, actor_name, source,
			created_at, read_at, is_deleted, written_at
		)
		SELECT xuid, id, category, severity, title_key, body_key, params,
		       target_route, target_search, actor_xuid, actor_name, source,
		       created_at, ?, FALSE, CURRENT_TIMESTAMP
		FROM player_notifications_latest
		WHERE xuid = ? AND read_at IS NULL AND severity = 'info' AND created_at < ?`,
		time.Now().UTC(), xuid, cutoff)
}

// DeleteNotification supprime une notif (+ CHECKPOINT). Renvoie le nb de lignes
// affectées (0 = id inconnu, le caller traduit en ErrNotFound).
func (p *SharedSocialPersister) DeleteNotification(ctx context.Context, xuid string, id int64) (int64, error) {
	// APPEND-ONLY : INSERT d'un event tombstone (is_deleted=TRUE) au lieu d'un DELETE.
	// La vue _latest masque ensuite la notif. RowsAffected = 1 si visible, 0 sinon.
	return p.execNotifWriteCheckpointed(ctx, "notif-delete", true,
		`INSERT INTO player_notifications_history (
			xuid, id, category, severity, title_key, body_key, params,
			target_route, target_search, actor_xuid, actor_name, source,
			created_at, read_at, is_deleted, written_at
		)
		SELECT xuid, id, category, severity, title_key, body_key, params,
		       target_route, target_search, actor_xuid, actor_name, source,
		       created_at, read_at, TRUE, CURRENT_TIMESTAMP
		FROM player_notifications_latest
		WHERE xuid = ? AND id = ?`,
		xuid, id)
}

// CapAndSweepNotifications purge les notifs au-delà du cap de rétention.
// SANS CHECKPOINT immédiat : purge idempotente (re-tournée au prochain emit si
// perdue), on s'appuie sur le scheduler 5min pour éviter un CHECKPOINT à chaque
// émission de notification.
func (p *SharedSocialPersister) CapAndSweepNotifications(ctx context.Context, xuid string, max int) error {
	if max <= 0 {
		return nil
	}
	// APPEND-ONLY : tombstone (is_deleted=TRUE) les notifs au-delà du cap, au lieu
	// d'un DELETE. Masquées ensuite par _latest → jamais re-balayées (déjà hors vue).
	_, err := p.execNotifWriteCheckpointed(ctx, "notif-cap-sweep", false, `
		INSERT INTO player_notifications_history (
			xuid, id, category, severity, title_key, body_key, params,
			target_route, target_search, actor_xuid, actor_name, source,
			created_at, read_at, is_deleted, written_at
		)
		SELECT xuid, id, category, severity, title_key, body_key, params,
		       target_route, target_search, actor_xuid, actor_name, source,
		       created_at, read_at, TRUE, CURRENT_TIMESTAMP
		FROM player_notifications_latest
		WHERE xuid = ? AND id NOT IN (
			SELECT id FROM player_notifications_latest
			WHERE xuid = ?
			ORDER BY created_at DESC
			LIMIT ?
		)
	`, xuid, xuid, max)
	return err
}

// UpsertNotificationPreferences applique N préférences (INSERT ON CONFLICT par
// (xuid, category)) en UNE transaction + CHECKPOINT immédiat. Signatures plates
// (slices parallèles) pour éviter d'importer le package métier notifications
// (cycle d'import).
func (p *SharedSocialPersister) UpsertNotificationPreferences(ctx context.Context, xuid string, categories []string, enabled []bool, delivery []string, updatedAt time.Time) error {
	if len(categories) == 0 {
		return nil
	}
	if len(categories) != len(enabled) || len(categories) != len(delivery) {
		return fmt.Errorf("shared_social notif-prefs: slices parallèles de longueurs différentes (cat=%d en=%d del=%d)",
			len(categories), len(enabled), len(delivery))
	}
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("shared_social notif-prefs: begin tx: %w", err)
	}
	rollback := func() {
		if rbErr := tx.Rollback(); rbErr != nil && rbErr != sql.ErrTxDone {
			slog.WarnContext(ctx, "shared_social notif-prefs: rollback failed (non-fatal)", "err", rbErr)
		}
	}
	// APPEND-ONLY : chaque préférence = INSERT d'une nouvelle version dans
	// notification_preferences_history (plus d'ON CONFLICT DO UPDATE). Latest wins via vue.
	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO notification_preferences_history (xuid, category, enabled, delivery, updated_at)
		VALUES (?, ?, ?, ?, ?)
	`)
	if err != nil {
		rollback()
		return fmt.Errorf("shared_social notif-prefs: prepare: %w", err)
	}
	defer stmt.Close()
	for i := range categories {
		if _, err := stmt.ExecContext(ctx, xuid, categories[i], enabled[i], delivery[i], updatedAt); err != nil {
			rollback()
			return fmt.Errorf("shared_social notif-prefs (%s): %w", categories[i], err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("shared_social notif-prefs: commit: %w", err)
	}
	if _, err := p.db.ExecContext(ctx, "CHECKPOINT"); err != nil {
		slog.WarnContext(ctx, "shared_social notif-prefs: CHECKPOINT post-commit non-fatal", "err", err)
	}
	return nil
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

	// APPEND-ONLY : le replace manuel = UN SEUL INSERT d'event (is_manual=TRUE).
	// La vue media_match_associations_latest masque l'ancienne assoc du même
	// media_file_id (priorité is_manual DESC, written_at DESC). Plus de DELETE.
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO media_match_associations_history
			(media_file_id, match_id, delta_seconds, is_manual, is_active, associated_at, written_at)
		 VALUES (?, ?, 0, TRUE, TRUE, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`,
		mediaFileID, matchID,
	); err != nil {
		rollback()
		return fmt.Errorf("shared_social SetMediaMatchAssociation: insert event: %w", err)
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

// NB (2026-08-04) : SetMediaLiked — UPDATE de media_files.liked + liked_at — a
// été SUPPRIMÉ avec le passage du like au par-viewer. Cette colonne était un
// booléen GLOBAL : un like de n'importe quel joueur allumait le cœur de tous.
// L'état d'un like vit maintenant exclusivement dans media_likes_history
// (append-only, une ligne par liker) — cf. AddLike / RemoveLike.

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
