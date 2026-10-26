package service

import "levelup/go-api/internal/port"

// WithMapGroups injecte le lecteur des groupes de cartes : le découpage par carte et l'Emprise par carte
// comptent une variante (« - Ranked », « Heavies ») sous sa carte de base. Sans injection, chaque carte
// reste seule.
func (s *TimeseriesService) WithMapGroups(r port.MapGroupReader) *TimeseriesService {
	s.mapGroups = r
	return s
}
