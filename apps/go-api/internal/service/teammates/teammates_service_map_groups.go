package teammates

import "levelup/go-api/internal/port"

// WithMapGroups injecte le lecteur des groupes de cartes : le filtre de cartes et les statistiques par
// carte comptent une variante (« - Ranked », « Heavies ») sous sa carte de base. Sans injection, chaque
// carte reste seule.
func (s *TeammatesService) WithMapGroups(r port.MapGroupReader) *TeammatesService {
	s.mapGroups = r
	return s
}
