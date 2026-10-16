package scheduler

// SetPoolShared applique tout de suite le consentement d'un joueur (par xuid) au pool de
// jetons : un retrait cesse de servir les autres joueurs à la requête suivante, sans
// attendre le prochain scan. Sans pool (mode dégradé) ou sans slot pour ce xuid, il n'y a
// rien à appliquer et la décision, déjà persistée, jouera au prochain scan.
func (s *AutoSyncScheduler) SetPoolShared(xuid string, shared bool) bool {
	if s == nil || s.pool == nil {
		return false
	}
	return s.pool.SetShared(xuid, shared)
}
