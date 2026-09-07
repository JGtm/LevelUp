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
	maps     []domain.TacticalMapRow
	pos      domain.TacticalPositions
	ev       domain.TacticalKillEvents
	univ     domain.TacticalUnivers
	morts    domain.TacticalMortsContexte
	errMaps  error
	errPos   error
	errEv    error
	errUniv  error
	errMorts error

	vuMaps  domain.TacticalQuery
	vuPos   domain.TacticalQuery
	vuEv    domain.TacticalQuery
	vuUniv  domain.TacticalQuery
	vuMorts domain.TacticalQuery
}

// Univers : la lecture d'OCCUPATION (phase 6) n'a besoin que de l'univers — ses valeurs
// viennent des sidecars, pas de la base.
//
// IL HONORE LA LISTE BLANCHE, et ce n'est pas un detail de double : le vrai lecteur
// l'applique dans son SELECT, et un mock qui l'ignorait rendait INVISIBLE tout defaut de
// perimetre — le filtre de spawn a ainsi pu etre servi sur l'univers entier sans qu'aucun
// test ne rougisse (revue P1-1).
func (m *mockTacticalRepo) Univers(_ context.Context, q domain.TacticalQuery) (domain.TacticalUnivers, error) {
	m.vuUniv = q
	if m.errUniv != nil || !perimetreAFiltrer(q) {
		return m.univ, m.errUniv
	}
	return universFiltre(m.univ, gardeDuPerimetre(q)), nil
}

func (m *mockTacticalRepo) MapsPlayed(_ context.Context, q domain.TacticalQuery) ([]domain.TacticalMapRow, error) {
	m.vuMaps = q
	return m.maps, m.errMaps
}

func (m *mockTacticalRepo) KillPositions(_ context.Context, q domain.TacticalQuery) (domain.TacticalPositions, error) {
	m.vuPos = q
	if m.errPos != nil || !perimetreAFiltrer(q) {
		return m.pos, m.errPos
	}
	garde := gardeDuPerimetre(q)
	out := domain.TacticalPositions{Univers: universFiltre(m.pos.Univers, garde)}
	for _, p := range m.pos.Points {
		if garde[p.MatchID] {
			out.Points = append(out.Points, p)
		}
	}
	return out, nil
}

func (m *mockTacticalRepo) KillEvents(_ context.Context, q domain.TacticalQuery) (domain.TacticalKillEvents, error) {
	m.vuEv = q
	if m.errEv != nil || !perimetreAFiltrer(q) {
		return m.ev, m.errEv
	}
	garde := gardeDuPerimetre(q)
	out := domain.TacticalKillEvents{Univers: universFiltre(m.ev.Univers, garde)}
	for _, e := range m.ev.Events {
		if garde[e.MatchID] {
			out.Events = append(out.Events, e)
		}
	}
	return out, nil
}

// MortsAvecContexte : la lecture d'isolement (lot 7C). Comme les trois autres, ELLE HONORE LA
// LISTE BLANCHE — un double plus permissif que la production rend invisible tout defaut de
// perimetre, et c'est deja arrive deux fois sur ce meme fichier.
func (m *mockTacticalRepo) MortsAvecContexte(_ context.Context, q domain.TacticalQuery) (domain.TacticalMortsContexte, error) {
	m.vuMorts = q
	if m.errMorts != nil || !perimetreAFiltrer(q) {
		return m.morts, m.errMorts
	}
	garde := gardeDuPerimetre(q)
	out := domain.TacticalMortsContexte{Univers: universFiltre(m.morts.Univers, garde)}
	for _, d := range m.morts.Morts {
		if garde[d.MatchID] {
			out.Morts = append(out.Morts, d)
		}
	}
	return out, nil
}

// perimetreAFiltrer dit si le double doit appliquer la liste blanche.
//
// UNE LISTE VIDE MAIS POSEE VAUT « AUCUN MATCH », ici comme chez le vrai lecteur (corrige
// le 2026-09-07, revue ronde 2). La version precedente exemptait ce cas et rendait alors
// l'univers ENTIER : or `requeteDuScope` POSE TOUJOURS la liste, si bien que toute fixture
// sans `MatchIDs` lisait un univers que le vrai lecteur aurait vide. Un double plus
// permissif que la production est ce qui rend les defauts de perimetre invisibles — c'est
// deja ainsi que le filtre de spawn avait pu etre servi sur l'univers entier (P1-1).
//
// Le seul etat non filtre est donc l'ABSENCE de liste, que `ListeBlancheMatchs` distingue
// du vide par construction (`RestreindreAux` vs zero-value).
func perimetreAFiltrer(q domain.TacticalQuery) bool {
	return q.Matchs.Restreint()
}

// gardeDuPerimetre / universFiltre : LE DOUBLE APPLIQUE LA LISTE BLANCHE, comme le vrai
// lecteur le fait dans son SELECT. Un mock qui l'ignore rend INVISIBLE tout defaut de
// perimetre — c'est ainsi que le filtre de spawn a pu etre servi sur l'univers entier sans
// qu'aucun test ne rougisse (revue P1-1).
func gardeDuPerimetre(q domain.TacticalQuery) map[string]bool {
	out := make(map[string]bool, len(q.Matchs.IDs()))
	for _, id := range q.Matchs.IDs() {
		out[id] = true
	}
	return out
}

func universFiltre(u domain.TacticalUnivers, garde map[string]bool) domain.TacticalUnivers {
	out := domain.TacticalUnivers{Equipes: domain.EquipesParMatch{}}
	for _, m := range u.Matchs {
		if garde[m.MatchID] {
			out.Matchs = append(out.Matchs, m)
		}
	}
	// LES COMPOSITIONS SUIVENT LES MATCHS : les rendre toutes laissait le service voir les
	// equipes de matchs qu'il n'avait pas le droit de lire — la meme demi-verite que le
	// perimetre non applique, une couche plus bas.
	for id, eq := range u.Equipes {
		if garde[id] {
			out.Equipes[id] = eq
		}
	}
	return out
}
