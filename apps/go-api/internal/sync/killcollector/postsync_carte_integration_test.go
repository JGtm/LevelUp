//go:build integration

package killcollector

// postsync_carte_integration_test.go — UN MATCH SANS CARTE NE PREND PAS DE PLACE DANS LE CYCLE
// (2026-09-27).
//
// Un match dont la carte n est pas resolue (Forge, carte hors catalogue, pas de nom en base) est
// mis de cote par le decodeur. Mais s il n est retire qu APRES le telechargement, il consomme une
// place de `perCycle` et un telechargement A CHAQUE CYCLE — et comme il ne quitte jamais le
// backlog (aucun marqueur terminal), des matchs Forge recents, tries en tete, affament le vrai
// backlog. La carte se resout donc AVANT le telechargement et AVANT la borne du cycle : le match
// sans carte est retire de la liste de travail, compte, et sa place va au match suivant.
//
// MUTATIONS QUI DOIVENT LE FAIRE ROUGIR : ne plus filtrer (les K premiers sont telecharges) ;
// filtrer APRES la borne `perCycle` (le cycle traite moins de `perCycle` matchs avec carte) ; ne
// lire qu une page de backlog (la pagination rend la place au-dela de l horizon).

import (
	"context"
	"testing"
	"time"

	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/port"
)

// cartesParMatch : le resolveur de carte PAR MATCH, sans base. Un match absent n a aucun nom.
type cartesParMatch map[string][]string

func (c cartesParMatch) MapKeysForMatch(_ context.Context, id string) (port.MatchMapKeys, error) {
	return port.MatchMapKeys{Names: c[id]}, nil
}

func (c cartesParMatch) MapKeysForMap(context.Context, string) (port.MatchMapKeys, error) {
	return port.MatchMapKeys{}, nil
}

// backlogAvecCarte inscrit m1..mN (mN le plus RECENT) et rend le cycle a jouer : les `sansCarte`
// plus recents n ont pas de carte resolue (le plus recent sans nom en base, les autres hors
// catalogue).
func backlogAvecCarte(t *testing.T, n, sansCarte int) func() []string {
	t.Helper()
	db := baseBacklog(t)
	t0 := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	cartes := cartesParMatch{}
	for i := 1; i <= n; i++ {
		id := "m" + string(rune('0'+i))
		inscrireMatch(t, db, id, t0.AddDate(0, i, 0), 0)
		switch {
		case i == n:
			// aucun nom en base
		case i > n-sansCarte:
			cartes[id] = []string{"Forge Personnalisee"}
		default:
			cartes[id] = []string{"Bazaar"}
		}
	}
	films := &filmsTraces{}
	d := depsDeTest(db, films, nil)
	d.MapNames = cartes
	return func() []string {
		RunPostSync(context.Background(), hookDuTest, d, nil)
		return films.demandes
	}
}

var hookDuTest *PostSyncHook

func verifierDemandes(t *testing.T, got, attendu []string) {
	t.Helper()
	if len(got) != len(attendu) {
		t.Fatalf("films demandes = %v, attendu %v — un match sans carte a pris une place du cycle "+
			"(ou declenche un telechargement)", got, attendu)
	}
	for i := range attendu {
		if got[i] != attendu[i] {
			t.Errorf("demande[%d] = %q, attendu %q (films demandes %v)", i, got[i], attendu[i], got)
		}
	}
}

func TestRunPostSync_MatchsSansCarteNeConsommentPasDePlace(t *testing.T) {
	hookDuTest = NewPostSyncHook(racineDepot(t), 3)
	cycle := backlogAvecCarte(t, 6, 2) // m6 sans nom, m5 hors catalogue
	avant := observability.LoadCounter(metricCarteAvantTelechargement)
	verifierDemandes(t, cycle(), []string{"m4", "m3", "m2"})
	if n := observability.LoadCounter(metricCarteAvantTelechargement) - avant; n != 2 {
		t.Errorf("%s : +%d, attendu +2 (m6, m5)", metricCarteAvantTelechargement, n)
	}
}

// TestRunPostSync_LaPlaceVaAuDelaDeLHorizon : plus de matchs sans carte qu une page de backlog n en
// lit — le cycle lit la page suivante au lieu de s arreter a vide.
func TestRunPostSync_LaPlaceVaAuDelaDeLHorizon(t *testing.T) {
	hookDuTest = NewPostSyncHook(racineDepot(t), 2)
	hookDuTest.horizon = 2
	cycle := backlogAvecCarte(t, 6, 3) // m6, m5, m4 sans carte : la premiere page (m6, m5) est vide
	verifierDemandes(t, cycle(), []string{"m3", "m2"})
}
