// Package port — progression.go : la progression d'un joueur (séries, records, jalons) et les campagnes
// d'amélioration, lues par la couche HTTP.
//
// Un handler décode la requête, appelle un service de ce fichier et encode la réponse. Le dépôt
// (`ProgressionStore`, `CampaignScope`) est la seule chose qui connaît le stockage ; le service
// (`service.ProgressionService`, `service.CampaignService`) porte les règles et les jointures.
package port

import (
	"context"
	"fmt"
	"time"

	"levelup/go-api/internal/campaign"
	"levelup/go-api/internal/progression/milestones"
	"levelup/go-api/internal/progression/records"
	"levelup/go-api/internal/progression/streaks"
)

// ProgressionErrorKind classe un échec de lecture de la progression. La couche HTTP en tire le statut et
// le code ; le service ne porte aucun texte destiné à l'écran.
type ProgressionErrorKind string

// Échecs de lecture de la progression.
const (
	// ProgErrPlayerNotFound : le slug ne se résout pas en joueur suivi.
	ProgErrPlayerNotFound ProgressionErrorKind = "player_not_found"
	// ProgErrListStreaks : lecture des séries.
	ProgErrListStreaks ProgressionErrorKind = "list_streaks"
	// ProgErrListPersonalBests : lecture des records personnels courants.
	ProgErrListPersonalBests ProgressionErrorKind = "list_personal_bests"
	// ProgErrListRecordHistory : lecture de l'historique des records battus.
	ProgErrListRecordHistory ProgressionErrorKind = "list_record_history"
	// ProgErrListCatalog : lecture du catalogue des jalons du titre.
	ProgErrListCatalog ProgressionErrorKind = "list_catalog"
	// ProgErrListEarned : lecture des jalons franchis par le joueur.
	ProgErrListEarned ProgressionErrorKind = "list_earned"
)

// ProgressionError est l'échec typé d'un service de progression, de campagne ou de profil. Err porte la
// cause (journal) ; son texte ne part pas au client.
type ProgressionError struct {
	Kind ProgressionErrorKind
	Err  error
}

// Error rend le texte de la cause.
func (e *ProgressionError) Error() string {
	if e.Err == nil {
		return fmt.Sprintf("progression: %s", e.Kind)
	}
	return e.Err.Error()
}

// Unwrap rend la cause, pour errors.Is / errors.As.
func (e *ProgressionError) Unwrap() error { return e.Err }

// ProgressionStore lit la progression d'UN joueur : il est lié à lui à la résolution, les méthodes ne
// prennent donc ni identifiant de joueur ni gamertag. Les tables append-only se lisent par leurs vues
// `_latest`.
type ProgressionStore interface {
	// ListStreaks rend les séries du joueur pour le titre (active et historique).
	ListStreaks(ctx context.Context, titleSlug string) ([]streaks.Streak, error)
	// ListPersonalBests rend les records personnels courants du joueur.
	ListPersonalBests(ctx context.Context) ([]records.PersonalRecord, error)
	// ListRecordHistory rend les `limit` records battus les plus récents du joueur pour le titre.
	ListRecordHistory(ctx context.Context, titleSlug string, limit int) ([]records.RecordHistory, error)
	// ListMilestoneCatalog rend le catalogue des jalons du titre.
	ListMilestoneCatalog(ctx context.Context, titleSlug string) ([]milestones.CatalogEntry, error)
	// ListMilestonesEarned rend les jalons franchis par le joueur pour le titre.
	ListMilestonesEarned(ctx context.Context, titleSlug string) ([]milestones.Earned, error)
}

// ProgressionStoreResolver résout un slug de joueur en son dépôt de progression ; une erreur = joueur
// introuvable.
type ProgressionStoreResolver func(ctx context.Context, playerSlug string) (ProgressionStore, error)

// PlayerRecords : les records d'un joueur, filtrés sur le catalogue des métriques connues.
type PlayerRecords struct {
	PersonalBests []records.PersonalRecord
	History       []records.RecordHistory
}

// MilestoneStatus : un jalon du catalogue et son statut pour le joueur.
type MilestoneStatus struct {
	Entry  milestones.CatalogEntry
	Earned bool
	// EarnedAt : date de franchissement ; nil quand le jalon est franchi sans date connue.
	EarnedAt *time.Time
}

// ProgressionService : ce que consomme le handler des routes /streaks, /records et /milestones.
type ProgressionService interface {
	// Streaks rend les séries du joueur pour le titre.
	Streaks(ctx context.Context, playerSlug, titleSlug string) ([]streaks.Streak, error)
	// Records rend les records personnels courants et les `historyLimit` derniers records battus ; une
	// métrique hors catalogue n'est pas servie (journalisée).
	Records(ctx context.Context, playerSlug, titleSlug string, historyLimit int) (PlayerRecords, error)
	// Milestones joint le catalogue du titre et les jalons franchis, dans l'ordre du catalogue.
	Milestones(ctx context.Context, playerSlug, titleSlug string) ([]MilestoneStatus, error)
}

// CampaignScope : les dépôts de campagne d'UN joueur et son identifiant.
type CampaignScope struct {
	// XUID : identifiant du joueur, clé de toutes les lectures de campagne.
	XUID    string
	Repo    campaign.Repo
	Samples campaign.SampleProvider
}

// CampaignScopeResolver résout un slug de joueur en ses dépôts de campagne ; une erreur = joueur
// introuvable.
type CampaignScopeResolver func(ctx context.Context, playerSlug string) (CampaignScope, error)

// CampaignService : ce que consomme le handler des routes /campaigns. Les refus métier sont ceux du
// paquet `campaign` (ErrNotFound, ErrAlreadyActive, ErrInvalidAxis, ErrInvalidStatus) ; un joueur
// introuvable est un *ProgressionError de nature ProgErrPlayerNotFound.
type CampaignService interface {
	// Start crée une campagne sur l'axe demandé ; UserID de `p` est ignoré (c'est le joueur résolu).
	Start(ctx context.Context, playerSlug string, p campaign.StartParams, now time.Time) (campaign.ImprovementCampaign, error)
	// Active rend la campagne active du joueur pour le titre (campaign.ErrNotFound si aucune).
	Active(ctx context.Context, playerSlug, titleSlug string) (campaign.ImprovementCampaign, error)
	// Ended rend les campagnes closes du joueur pour le titre, les plus récentes d'abord.
	Ended(ctx context.Context, playerSlug, titleSlug string) ([]campaign.ImprovementCampaign, error)
	// ByID rend une campagne et ses défis liés.
	ByID(ctx context.Context, playerSlug, id string) (campaign.ImprovementCampaign, error)
	// Pause met une campagne active en pause.
	Pause(ctx context.Context, playerSlug, id string) error
	// Resume reprend une campagne en pause.
	Resume(ctx context.Context, playerSlug, id string) error
	// Close clôture une campagne (terminée).
	Close(ctx context.Context, playerSlug, id string, now time.Time) error
	// Abandon abandonne une campagne, sans pénalité.
	Abandon(ctx context.Context, playerSlug, id string, now time.Time) error
}
