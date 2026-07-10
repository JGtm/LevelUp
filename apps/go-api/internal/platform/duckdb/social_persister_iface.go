// Package duckdb — social_persister_iface.go : interface SocialPersister
// utilisée par PlayerDB pour router les écritures shared_social vers le
// pattern Collect→Persist (ADR 0022).
//
// L'interface vit ici (pas dans internal/persist) pour éviter le cycle
// d'import : internal/persist importe déjà internal/platform/duckdb via
// combined_persister.go (pour le pool batch sync engine).
//
// L'implémentation concrète persist.SharedSocialPersister satisfait cette
// interface STRUCTURELLEMENT (Go duck typing) — pas de déclaration
// d'implémentation requise. Injection au boot dans main.go :
//
//	pdb.SocialPersister = persist.NewSharedSocialPersister(pdb.SharedSocial.SQLDb())
//
// Les repos qui écrivent sur shared_social DEVRAIENT passer par cette interface
// (chemin nominal). Si nil (init pas faite ou SharedSocial nil), le repo retombe
// sur l'ancien chemin db.Exec. NB (revue 2026-06-01 SS-02) : il n'existe PAS de
// sentinel AST interdisant ces écritures directes — seul l'ATTACH est gardé
// (no_attach_on_social_test.go). Les mutations notifications et Prestige restent
// hors persist aujourd'hui.

package duckdb

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// SocialPersister est l'API d'écriture sur shared_social.duckdb.
//
// Le type *Batch est une interface vide (any) ici : on délègue au caller la
// construction du batch concret (persist.SharedSocialBatch) et la garantie
// que ce batch est compatible avec l'implémentation injectée. C'est moche
// mais c'est le seul moyen sans cycle d'import.
//
// Alternative envisagée : déplacer SharedSocialBatch dans un sous-package
// shared (sans dépendance duckdb). Non fait dans cette session pour limiter
// le scope du refactor.
type SocialPersister interface {
	// PersistBatch persiste un *persist.SharedSocialBatch (typé en any pour
	// éviter le cycle d'import). L'implémentation concrète fait le cast en
	// interne. À utiliser quand on a plusieurs writes à grouper en 1 TX
	// (ex : IndexMedia batch).
	PersistBatch(ctx context.Context, batch any) error

	// Méthodes ciblées pour les sites légers — chaque appel = 1 INSERT/
	// UPDATE/DELETE en TX atomique + CHECKPOINT garanti. Évite au caller
	// d'avoir à connaître le type SharedSocialBatch (et donc d'importer
	// internal/persist, ce qui causerait un cycle).
	//
	// Tous garantissent : BEGIN TX → write → COMMIT → CHECKPOINT.

	// AddFavorite : INSERT event is_favorite=TRUE dans match_favorites_history (append-only).
	AddFavorite(ctx context.Context, playerSlug, matchID string) error
	// RemoveFavorite : INSERT event is_favorite=FALSE dans match_favorites_history
	// (append-only, plus de DELETE — surface ART éliminée).
	RemoveFavorite(ctx context.Context, playerSlug, matchID string) error

	// AddLike : INSERT event is_liked=TRUE dans media_likes_history (append-only).
	AddLike(ctx context.Context, mediaPath, likerSlug, likerGamertag string) error
	// RemoveLike : INSERT event is_liked=FALSE dans media_likes_history (append-only, plus de DELETE).
	RemoveLike(ctx context.Context, mediaPath, likerSlug string) error

	// SetMediaMatchAssociation : force l'association média→match (DELETE old +
	// INSERT new dans la même TX, CHECKPOINT garanti). ADR 0021 Phase 3.2.
	SetMediaMatchAssociation(ctx context.Context, mediaFileID int64, matchID string) error

	// SetMediaLiked : UPDATE media_files.liked (+ liked_at) atomique avec
	// CHECKPOINT garanti. Retourne true si la ligne existait. ADR 0021 Phase 3.2.
	SetMediaLiked(ctx context.Context, filePath string, liked bool) (bool, error)

	// AppendPlayerRecord : INSERT pur dans player_records_history (pattern
	// append-only, anti-bug ART). achievedAt, achievedMatchID, previousValue et
	// previousAchievedAt peuvent être nil. period = "all_time" si vide.
	AppendPlayerRecord(ctx context.Context, xuid, metric, period string, value float64, achievedAt *time.Time, achievedMatchID *string, previousValue *float64, previousAchievedAt *time.Time) error

	// CreateNotification : INSERT dans player_notifications. Le paramètre
	// est typé en `any` pour éviter le cycle d'import — l'impl attend une
	// duckdb.NotificationData (interface satisfaite structurellement par
	// ses getters Get*).
	CreateNotification(ctx context.Context, n any) error
	// MarkNotificationRead : UPDATE read_at dans player_notifications.
	// NB : pas de CHECKPOINT immédiat (délègue à Persist). Préférer
	// MarkNotificationsRead pour les actions user-facing.
	MarkNotificationRead(ctx context.Context, xuid string, id int64, readAt time.Time) error

	// Mutations notifications user-facing — chaque appel = write atomique +
	// CHECKPOINT immédiat (durabilité au restart, ADR 0022). Appelées sous le
	// lease KindSharedSocial tenu par notifications.Service.

	// MarkNotificationsRead : UPDATE read_at sur N ids (non-lus) en 1 TX +
	// CHECKPOINT. Renvoie le nb de lignes affectées.
	MarkNotificationsRead(ctx context.Context, xuid string, ids []int64, readAt time.Time) (int64, error)
	// MarkNotificationUnread : UPDATE read_at = NULL + CHECKPOINT. Renvoie le
	// nb de lignes affectées (0 = id inconnu).
	MarkNotificationUnread(ctx context.Context, xuid string, id int64) (int64, error)
	// MarkAllNotificationsRead : UPDATE read_at sur toutes les non-lues
	// (filtré par category si non vide) + CHECKPOINT. Renvoie le nb affecté.
	MarkAllNotificationsRead(ctx context.Context, xuid, category string, readAt time.Time) (int64, error)
	// DeleteNotification : DELETE + CHECKPOINT. Renvoie le nb de lignes
	// affectées (0 = id inconnu).
	DeleteNotification(ctx context.Context, xuid string, id int64) (int64, error)
	// CapAndSweepNotifications : purge de rétention (DELETE), SANS CHECKPOINT
	// immédiat (idempotent, flush via scheduler 5min).
	CapAndSweepNotifications(ctx context.Context, xuid string, max int) error
	// SweepStaleInfoNotificationsRead : marque lues les notifs severity='info'
	// non lues plus vieilles que cutoff (expiry douce DP8), SANS CHECKPOINT
	// immédiat (idempotent). Renvoie le nb marqué.
	SweepStaleInfoNotificationsRead(ctx context.Context, xuid string, cutoff time.Time) (int64, error)
	// UpsertNotificationPreferences : INSERT ON CONFLICT par (xuid, category)
	// en 1 TX + CHECKPOINT. Slices parallèles pour éviter le cycle d'import.
	UpsertNotificationPreferences(ctx context.Context, xuid string, categories []string, enabled []bool, delivery []string, updatedAt time.Time) error
}

// NotificationData : pure data type passé à CreateNotification. Évite une
// signature à 13 paramètres. Défini ici (pas dans internal/persist) pour
// éviter le cycle d'import.
//
// Les getters (Get*) sont fournis pour satisfaire l'interface structurelle
// notificationDataLike côté persist — sans eux, on devrait passer par
// reflection (lent + moche).
type NotificationData struct {
	XUID         string
	ID           int64
	Category     string
	Severity     string
	TitleKey     string
	BodyKey      *string
	Params       *string
	TargetRoute  *string
	TargetSearch *string
	ActorXUID    *string
	ActorName    *string
	Source       string
	CreatedAt    time.Time
}

// Getters pour satisfaire l'interface structurelle notificationDataLike
// (cf. internal/persist/shared_social_persister.go). Aucune logique, juste
// du plumbing trivial dû au cycle d'import qu'on doit contourner.

func (n NotificationData) GetXUID() string          { return n.XUID }
func (n NotificationData) GetID() int64             { return n.ID }
func (n NotificationData) GetCategory() string      { return n.Category }
func (n NotificationData) GetSeverity() string      { return n.Severity }
func (n NotificationData) GetTitleKey() string      { return n.TitleKey }
func (n NotificationData) GetBodyKey() *string      { return n.BodyKey }
func (n NotificationData) GetParams() *string       { return n.Params }
func (n NotificationData) GetTargetRoute() *string  { return n.TargetRoute }
func (n NotificationData) GetTargetSearch() *string { return n.TargetSearch }
func (n NotificationData) GetActorXUID() *string    { return n.ActorXUID }
func (n NotificationData) GetActorName() *string    { return n.ActorName }
func (n NotificationData) GetSource() string        { return n.Source }
func (n NotificationData) GetCreatedAt() time.Time  { return n.CreatedAt }

// SocialPersisterFactory est un hook configuré par main.go au boot pour
// permettre à openPlayerDB d'instancier un SocialPersister sans importer
// internal/persist (qui causerait un cycle).
//
// Wiring attendu dans main.go :
//
//	duckdb.SocialPersisterFactory = func(db *sql.DB) duckdb.SocialPersister {
//	    return persist.NewSharedSocialPersister(db)
//	}
//
// Si nil (cas tests, bootstrap CLI), pdb.SocialPersister reste nil et les
// repos retombent sur leur chemin legacy db.Exec.
var SocialPersisterFactory func(db *sql.DB) SocialPersister

// RequireSocialPersister, quand true, force les call sites à exiger un
// SocialPersister wired et retourner une erreur si nil au lieu de fallback
// silencieusement vers le path legacy db.Exec.
//
// Set à true par main.go au boot (cmd/server/main.go) APRÈS le wiring de
// SocialPersisterFactory. Default false pour préserver les tests qui
// n'instancient pas le Persister (cf. post_sync_deltas_test.go et al.).
//
// ADR 0021 Gap 1 : empêche silencieusement un site d'écriture de contourner
// le CHECKPOINT systématique du Persister en prod.
var RequireSocialPersister bool

// ErrSocialPersisterNotWired est retourné par les call sites quand
// RequireSocialPersister == true et pdb.SocialPersister == nil. Le caller
// HTTP doit mapper en 5xx (erreur serveur — wiring boot bugué).
var ErrSocialPersisterNotWired = errors.New("shared_social: SocialPersister non wired — refus d'écrire sans CHECKPOINT (ADR 0021 Gap 1)")
