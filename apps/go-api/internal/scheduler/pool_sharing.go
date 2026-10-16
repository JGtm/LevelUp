package scheduler

// SetPoolShared applique tout de suite le consentement d'un joueur au pool de jetons :
// un retrait cesse de servir les autres joueurs à la requête suivante, sans attendre le
// prochain scan. Sans pool (mode dégradé) ou sans slot pour ce gamertag, il n'y a rien
// à appliquer et la décision, déjà persistée, jouera au prochain scan.
func (s *AutoSyncScheduler) SetPoolShared(gamertag string, shared bool) bool {
	if s == nil || s.pool == nil {
		return false
	}
	return s.pool.SetShared(gamertag, shared)
}
