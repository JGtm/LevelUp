// Package service — sync_jobs_profile.go : recherche du profil d'un joueur pour un job de sync.
package service

import (
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"
)

// findPlayerProfile cherche, parmi les profils suivis, le premier dont le slug est playerSlug ; titles
// restreint la recherche aux profils de ces titres (aucun = tous). Un profil trouvé sans gamertag
// compte comme absent. Une base de profils illisible est rendue en port.SyncErrProfilesLoad, cause
// jointe (la couche HTTP la journalise).
func (s *SyncJobService) findPlayerProfile(playerSlug string, titles ...string) (domain.PlayerSummary, bool, error) {
	players, err := s.profiles.LoadPlayers(titles...)
	if err != nil {
		return domain.PlayerSummary{}, false, &port.SyncJobError{Kind: port.SyncErrProfilesLoad, Err: err}
	}
	for _, p := range players {
		if p.PlayerSlug == playerSlug {
			return p, p.Gamertag != "", nil
		}
	}
	return domain.PlayerSummary{}, false, nil
}
