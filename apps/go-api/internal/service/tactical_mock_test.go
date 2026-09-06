// Package service — tactical_mock_test.go : LE DOUBLE DU PORT TACTIQUE, partage par les
// trois fichiers de test de l'onglet (lecture de placement, echange, occupation).
//
// Extrait de tactical_service_test.go le 2026-09-06 (phase 6), quand l'ajout de la
// quatrieme lecture l'a pousse au-dela du seuil de 500 lignes. La coupure suit la seule
// frontiere naturelle du fichier : ICI ce que le port RETOURNE, LA-BAS ce que le service
// en FAIT — meme discipline que la scission de tactical_repo_test.go en phase 2.
package service

import (
	"context"

	"levelup/go-api/internal/domain"
)

// mockTacticalRepo double port.TacticalRepository et retient ce qu'on lui demande.
type mockTacticalRepo struct {
	maps    []domain.TacticalMapRow
	pos     domain.TacticalPositions
	ev      domain.TacticalKillEvents
	univ    domain.TacticalUnivers
	errMaps error
	errPos  error
	errEv   error
	errUniv error

	vuMaps domain.TacticalQuery
	vuPos  domain.TacticalQuery
	vuEv   domain.TacticalQuery
	vuUniv domain.TacticalQuery
}

// Univers : la lecture d'OCCUPATION (phase 6) n'a besoin que de l'univers — ses valeurs
// viennent des sidecars, pas de la base.
func (m *mockTacticalRepo) Univers(_ context.Context, q domain.TacticalQuery) (domain.TacticalUnivers, error) {
	m.vuUniv = q
	return m.univ, m.errUniv
}

func (m *mockTacticalRepo) MapsPlayed(_ context.Context, q domain.TacticalQuery) ([]domain.TacticalMapRow, error) {
	m.vuMaps = q
	return m.maps, m.errMaps
}

func (m *mockTacticalRepo) KillPositions(_ context.Context, q domain.TacticalQuery) (domain.TacticalPositions, error) {
	m.vuPos = q
	return m.pos, m.errPos
}

func (m *mockTacticalRepo) KillEvents(_ context.Context, q domain.TacticalQuery) (domain.TacticalKillEvents, error) {
	m.vuEv = q
	return m.ev, m.errEv
}
