package port

import (
	"errors"

	"levelup/go-api/internal/domain"
)

// Erreurs d'un import depuis le site ; leur identité fixe le code d'API rendu.
var (
	// ErrSiteImportRunning : un import est déjà en cours.
	ErrSiteImportRunning = errors.New("port: a site import is already running")
	// ErrSiteImportNoPlayer : aucun joueur suivi localement.
	ErrSiteImportNoPlayer = errors.New("port: no locally tracked player")
	// ErrSiteImportAmbiguous : plusieurs joueurs suivis, le gamertag doit être nommé.
	ErrSiteImportAmbiguous = errors.New("port: several tracked players, name the gamertag")
	// ErrSiteImportPlayerNoXID : le joueur suivi n'a pas d'identifiant Xbox résolu.
	ErrSiteImportPlayerNoXID = errors.New("port: the tracked player has no resolved Xbox id")
)

// SiteImportRequest : la demande d'import. Gamertag n'est nécessaire que si plusieurs joueurs
// sont suivis.
type SiteImportRequest struct {
	URL      string
	Token    string
	Gamertag string
}

// SiteImportService importe les données d'un joueur depuis une instance hébergée ; implémenté par
// service.SiteImport.
type SiteImportService interface {
	// Start lance l'import en arrière-plan. Erreurs : ErrSiteImportRunning, ErrSiteImportNoPlayer,
	// ErrSiteImportAmbiguous, ErrSiteImportPlayerNoXID ; toute autre est une indisponibilité.
	Start(req SiteImportRequest) error
	// Status rend l'état de l'import courant ou du dernier.
	Status() domain.SiteImportStatus
}
