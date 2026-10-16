package service

import "levelup/go-api/internal/domain"

// WithPoolSharingResolver injecte la fonction qui dit où en est un xuid sur la mise en
// commun de son jeton (domain.PoolSharing*). Sans elle, pool_sharing vaut « unavailable »
// et le front ne propose aucun choix.
func (s *BootstrapService) WithPoolSharingResolver(fn func(xuid string) string) *BootstrapService {
	s.poolSharing = fn
	return s
}

// resolvePoolSharing donne l'état de la décision du user CONNECTÉ, scopé à son propre
// compte (ownXUID) comme reauth_required : un admin qui consulte un autre joueur ne
// décide jamais à la place du propriétaire du jeton.
func (s *BootstrapService) resolvePoolSharing(sess *domain.SessionData) string {
	if s.poolSharing == nil {
		return domain.PoolSharingUnavailable
	}
	xuid := s.ownXUID(sess)
	if xuid == "" {
		return domain.PoolSharingUnavailable
	}
	return s.poolSharing(xuid)
}
