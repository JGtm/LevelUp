package settings

// replay_regen.go — l'état persisté de la file de régénération des rejeux (application de bureau).
//
// `replay_regen_pending` n'est PAS un réglage de l'utilisateur : le serveur le pose quand la file démarre ou
// reprend, et l'efface quand elle finit ou qu'elle est mise en pause. Il survit à un redémarrage pour que la
// file reprenne à l'ouverture suivante de l'application. Il n'entre dans aucune réponse de l'API
// (`ToResponse` ne le recopie pas) ni dans aucune écriture partielle (`Apply`).

// ReplayRegenPending dit si la file de régénération était active à la dernière écriture.
func (s *Store) ReplayRegenPending() (bool, error) {
	cfg, err := s.Load()
	if err != nil {
		return false, err
	}
	return cfg.ReplayRegenPending, nil
}

// SetReplayRegenPending écrit le drapeau, sans toucher aux autres clés. Une valeur inchangée n'écrit rien.
func (s *Store) SetReplayRegenPending(pending bool) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	cfg, err := s.Load()
	if err != nil {
		return err
	}
	if cfg.ReplayRegenPending == pending {
		return nil
	}
	cfg.ReplayRegenPending = pending
	return s.Save(cfg)
}
