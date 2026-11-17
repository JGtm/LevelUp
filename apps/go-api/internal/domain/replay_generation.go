package domain

// replay_generation.go — la génération à la demande du rejeu d'UN match qui n'en a pas (placement local :
// application de bureau et poste de développement). Le serveur ne sert que des codes (état, raison, phase) :
// le web écrit les textes. Les erreurs vivent ici pour que `replayregen`, `service` et `api/handlers` les
// partagent par `errors.Is` sans s'importer.

import (
	"errors"
	"time"
)

// États servis d'une génération de rejeu.
const (
	// ReplayGenerationGeneratable : rien n'empêche de générer ; le bouton « générer » est offert.
	ReplayGenerationGeneratable = "generatable"
	// ReplayGenerationQueued : la demande attend son tour dans la file.
	ReplayGenerationQueued = "queued"
	// ReplayGenerationRunning : la demande est en cours (phase : téléchargement puis construction).
	ReplayGenerationRunning = "running"
	// ReplayGenerationFailed : le dernier essai de ce processus a échoué.
	ReplayGenerationFailed = "failed"
	// ReplayGenerationFilmUnavailable : le film du match n'existe plus chez l'éditeur.
	ReplayGenerationFilmUnavailable = "film_unavailable"
	// ReplayGenerationNotSupported : ce match ne se génère pas sur ce poste.
	ReplayGenerationNotSupported = "not_supported"
	// ReplayGenerationReady : le rejeu est rangé.
	ReplayGenerationReady = "ready"
)

// Raisons d'un état (`reason`).
const (
	// Échecs réessayables.
	ReplayGenReasonFilmNotReady     = "film_not_ready"
	ReplayGenReasonFetchFailed      = "fetch_failed"
	ReplayGenReasonTokenUnavailable = "token_unavailable"
	ReplayGenReasonBuildFailed      = "build_failed"
	// Échecs définitifs.
	ReplayGenReasonMapNotInCatalog = "map_not_in_catalog"
	ReplayGenReasonUnknownFilmKey  = "unknown_film_key"
	// Raisons de `not_supported`.
	ReplayGenReasonModeUnmanaged = "mode_unmanaged"
	ReplayGenReasonMatchUnknown  = "match_unknown"
	ReplayGenReasonNoToken       = "no_token"
)

// Phases d'une demande en cours (`phase`).
const (
	ReplayGenPhaseDownloading = "downloading"
	ReplayGenPhaseBuilding    = "building"
)

// ReplayGenerationStatus : l'état de la génération du rejeu d'un match. GET et POST le rendent.
type ReplayGenerationStatus struct {
	// State : voir les constantes ReplayGeneration*.
	State string `json:"state" enum:"generatable,queued,running,failed,film_unavailable,not_supported,ready" doc:"État de la génération du rejeu de ce match."`
	// Reason précise `failed` et `not_supported` ; vide sinon.
	Reason string `json:"reason,omitempty" doc:"Raison de l'état failed ou not_supported ; absente pour les autres états."`
	// Phase : étape d'une demande `running` ; vide sinon.
	Phase string `json:"phase,omitempty" enum:"downloading,building" doc:"Étape d'une demande en cours : téléchargement du film puis construction du rejeu."`
	// Position : rang dans la file pour `queued` (1 = le suivant) ; 0 sinon.
	Position int `json:"position" doc:"Rang de la demande dans la file (1 = la suivante) ; 0 hors de la file."`
	// WaitingForGame : la file attend la fin de la partie en cours.
	WaitingForGame bool `json:"waiting_for_game" doc:"La file attend la fin de la partie en cours avant de lancer ce film."`
	// Paused : la file de régénération du lot est en pause à la demande de l'utilisateur.
	Paused bool `json:"paused" doc:"La file est en pause à la demande de l'utilisateur."`
	// Retryable : un nouvel essai a une chance d'aboutir (état `failed` seulement).
	Retryable bool `json:"retryable" doc:"Un nouvel essai peut aboutir (état failed seulement)."`
}

// ReplayGenerationDemand : la demande de génération d'un match. Elle porte l'identité du joueur de la page
// qui la formule (xuid : la clé de jointure, ADR 0035) ; aucun secret.
type ReplayGenerationDemand struct {
	TitleSlug string `json:"title_slug"`
	// MatchID : l'identifiant complet du registre.
	MatchID     string    `json:"match_id"`
	PlayerSlug  string    `json:"player_slug"`
	XUID        string    `json:"xuid"`
	Gamertag    string    `json:"gamertag"`
	RequestedAt time.Time `json:"requested_at"`
}

// Erreurs de la génération à la demande.
var (
	// ErrReplayGenerationQueueFull : le plafond de demandes en vol est atteint.
	ErrReplayGenerationQueueFull = errors.New("replay generation: file pleine")
	// ErrReplayFilmUnavailable : le film du match n'existe plus (bit « film absent » posé).
	ErrReplayFilmUnavailable = errors.New("replay generation: film indisponible")
	// ErrReplayGenerationNotSupported : ce match ne se génère pas ici (la raison est portée par ReplayGenerationNotSupportedError).
	ErrReplayGenerationNotSupported = errors.New("replay generation: non pris en charge")
	// ErrReplayGenerationUnbound : la file n'est pas reliée au serveur.
	ErrReplayGenerationUnbound = errors.New("replay generation: file non reliée")
	// ErrReplayFilmGone : le téléchargement a constaté que le film n'existe plus (404, 410 ou manifeste vide).
	ErrReplayFilmGone = errors.New("replay generation: film retiré chez l'éditeur")
	// ErrReplayNoToken : aucun jeton du joueur de la demande n'est utilisable.
	ErrReplayNoToken = errors.New("replay generation: aucun jeton du joueur")
)

// ReplayGenerationNotSupportedError : un refus `not_supported` avec sa raison (ReplayGenReason*). Il se
// reconnaît par errors.Is(err, ErrReplayGenerationNotSupported).
type ReplayGenerationNotSupportedError struct{ Reason string }

func (e *ReplayGenerationNotSupportedError) Error() string {
	return ErrReplayGenerationNotSupported.Error() + ": " + e.Reason
}

func (e *ReplayGenerationNotSupportedError) Is(target error) bool {
	return target == ErrReplayGenerationNotSupported
}

// ReplayGenerationRequestResponse : la réponse du POST de génération : l'état et si la demande vient d'être créée
// (faux : elle était déjà en file ou en cours).
type ReplayGenerationRequestResponse struct {
	Created bool                   `json:"created" doc:"Vrai si la demande vient d'être mise en file ; faux si elle était déjà en file, en cours, ou si le rejeu est déjà rangé."`
	Status  ReplayGenerationStatus `json:"status" doc:"État de la génération après la demande."`
}
