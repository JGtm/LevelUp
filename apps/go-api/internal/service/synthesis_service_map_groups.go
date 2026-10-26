package service

import "levelup/go-api/internal/port"

// WithMapGroups injecte le lecteur des groupes de cartes : le filtre de cartes et les « cartes les plus
// jouées » comptent une variante (« - Ranked », « Heavies ») sous sa carte de base. Sans injection,
// chaque carte reste seule.
func (s *SynthesisService) WithMapGroups(r port.MapGroupReader) *SynthesisService {
	s.mapGroups = r
	return s
}
