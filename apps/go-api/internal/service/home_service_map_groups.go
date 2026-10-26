package service

import "levelup/go-api/internal/port"

// WithMapGroups injecte le lecteur des groupes de cartes : la tuile « carte favorite » de l'Accueil compte
// une variante (« - Ranked », « Heavies ») sous sa carte de base. Sans injection, chaque carte reste seule.
func (s *HomeService) WithMapGroups(r port.MapGroupReader) *HomeService {
	s.mapGroups = r
	return s
}
