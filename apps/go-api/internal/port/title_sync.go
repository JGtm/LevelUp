package port

// TitleSyncProfiles active, met en pause ou purge un titre d'un joueur dans ses profils
// (db_profiles.json) ; implémenté par service.ProfileService.
//
// Les deux opérations propagent les erreurs sentinelles du magasin de profils
// (dbprofiles.ErrEntryNotFound : couple inexistant ; dbprofiles.ErrLastActiveTitle : refus de
// retirer ou de mettre en pause le dernier titre actif du joueur).
type TitleSyncProfiles interface {
	// SetTitleSyncEnabled bascule l'état de synchronisation (actif ou pause) du couple
	// (titleSlug, gamertag).
	SetTitleSyncEnabled(titleSlug, gamertag string, enabled bool) error
	// PurgeTitleData retire le couple du profil puis supprime les fichiers du joueur pour ce titre.
	// dataRemoved=false (sans erreur) : le profil est retiré mais les fichiers n'ont pas pu l'être
	// (verrou disque résiduel).
	PurgeTitleData(titleSlug, gamertag string) (dataRemoved bool, err error)
}
