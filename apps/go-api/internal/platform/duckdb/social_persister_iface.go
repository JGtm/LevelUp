// Package duckdb — social_persister_iface.go : interface SocialPersister
// utilisée par PlayerDB pour router les écritures shared_social vers le
// pattern Collect→Persist (ADR 0020).
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
// Les repos qui écrivent sur shared_social DOIVENT passer par cette
// interface. Si nil (initialisation pas faite ou SharedSocial nil), le repo
// peut retomber sur l'ancien chemin db.Exec — mais cette dégradation sera
// supprimée en Phase 6 (sentinel parse-AST).

package duckdb

import (
	"context"
	"database/sql"
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

	// AddFavorite : INSERT OR IGNORE dans match_favorites.
	AddFavorite(ctx context.Context, playerSlug, matchID string) error
	// RemoveFavorite : DELETE FROM match_favorites.
	RemoveFavorite(ctx context.Context, playerSlug, matchID string) error

	// AddLike : INSERT OR IGNORE dans media_likes.
	AddLike(ctx context.Context, mediaPath, likerSlug, likerGamertag string) error
	// RemoveLike : DELETE FROM media_likes.
	RemoveLike(ctx context.Context, mediaPath, likerSlug string) error

	// AppendPlayerRecord : INSERT pur dans player_records_history (pattern
	// append-only, anti-bug ART). achievedAt et achievedMatchID peuvent être
	// nil. period = "all_time" si vide.
	AppendPlayerRecord(ctx context.Context, xuid, metric, period string, value float64, achievedAt *time.Time, achievedMatchID *string) error

	// CreateNotification : INSERT dans player_notifications. Le paramètre
	// est typé en `any` pour éviter le cycle d'import — l'impl attend une
	// duckdb.NotificationData (interface satisfaite structurellement par
	// ses getters Get*).
	CreateNotification(ctx context.Context, n any) error
	// MarkNotificationRead : UPDATE read_at dans player_notifications.
	MarkNotificationRead(ctx context.Context, xuid string, id int64, readAt time.Time) error
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
