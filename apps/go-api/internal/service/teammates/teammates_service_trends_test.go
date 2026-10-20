package teammates

// teammates_service_trends_test.go — la vue Escouade de la page Tendances (GetSquadTrends) :
// composition non stricte et stricte, une seule lecture de l'équipe alliée par requête,
// parité des matchs retenus avec CompositionSessions, erreurs rendues.

import (
	"context"
	"errors"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/legacymatch"
)

var tendancesNow = time.Date(2026, 9, 27, 2, 0, 0, 0, time.UTC)

// repoTendances : le scénario de la composition exacte (m1 propre, m2 avec AllyC) avec des
// frags sur l'équipe alliée.
func repoTendances() *mockSquadRepo {
	repo := newExtraTeammateRepo()
	kills := func(m, x string, k, d, a int) domain.AllyParticipant {
		return domain.AllyParticipant{MatchID: m, XUID: x, Kills: k, Deaths: d, Assists: a}
	}
	repo.allyRows = []domain.AllyParticipant{
		kills("m1", "px", 10, 5, 3), kills("m1", "xa", 6, 4, 0), kills("m1", "xb", 4, 2, 0),
		kills("m2", "px", 8, 6, 0), kills("m2", "xa", 5, 3, 0), kills("m2", "xb", 2, 9, 0), kills("m2", "xc", 5, 5, 0),
	}
	return repo
}

// historiqueTendances : m1 et m2 en escouade, m3 en solo.
func historiqueTendances() []legacymatch.SynthesisMatchRow {
	row := func(id string, at time.Time, avecAmis bool) legacymatch.SynthesisMatchRow {
		return legacymatch.SynthesisMatchRow{
			MatchID: id, StartTime: at, Outcome: domain.OutcomeWin, Kills: 10, Deaths: 5,
			IsWithFriends: avecAmis, PlaylistName: "Arene",
		}
	}
	return []legacymatch.SynthesisMatchRow{
		row("m1", time.Date(2026, 6, 1, 20, 0, 0, 0, time.UTC), true),
		row("m2", time.Date(2026, 6, 8, 20, 0, 0, 0, time.UTC), true),
		row("m3", time.Date(2026, 6, 9, 20, 0, 0, 0, time.UTC), false),
	}
}

func nouveauServiceTendances(wrapped *lecturesComptees) *TeammatesService {
	return avecConnus(NewTeammatesService(wrapped, nil), wrapped.mockSquadRepo).
		WithPlayerMatchesRepo(newSynthMockFromRows(historiqueTendances(), nil), "halo_infinite", "Test").
		WithTrends(TrendsDeps{Now: func() time.Time { return tendancesNow }})
}

func valeur365(t *testing.T, resp domain.TrendsPageResponse, key, variant string) *float64 {
	t.Helper()
	for _, ind := range resp.Indicators {
		if ind.Key == key && ind.Variant == variant {
			for _, h := range ind.Horizons {
				if h.Days == 365 {
					return h.Value
				}
			}
		}
	}
	return nil
}

func lecturesQ32b(l *lecturesComptees) int {
	n := 0
	for _, r := range l.lues {
		if r == "Q32b" {
			n++
		}
	}
	return n
}

func TestGetSquadTrends_NonStricte(t *testing.T) {
	l := &lecturesComptees{mockSquadRepo: repoTendances()}
	resp, err := nouveauServiceTendances(l).GetSquadTrends(context.Background(), "px",
		domain.TrendsQueryRequest{View: "squad", SelectedGamertags: []string{"AllyA", "AllyB"}})
	if err != nil {
		t.Fatal(err)
	}
	if resp.View != domain.TrendsViewSquad {
		t.Errorf("vue = %q", resp.View)
	}
	if v := valeur365(t, resp, "match_count", ""); v == nil || *v != 2 {
		t.Fatalf("match_count = %v, attendu 2", v)
	}
	if v := valeur365(t, resp, "win_rate_alone", ""); v == nil || *v != 1 {
		t.Errorf("win_rate_alone = %v (un match solo gagné)", v)
	}
	// m1 : 20 frags d'équipe, m2 : 20 ; membres Test(px)+AllyA+AllyB : 20 + 15 = 35 sur 40.
	if v := valeur365(t, resp, "squad_share_of_team_kills", ""); v == nil || *v != 0.875 {
		t.Errorf("part de l'équipe = %v, attendu 0,875", v)
	}
	if v := valeur365(t, resp, "member_share_of_squad_kills", "AllyA"); v == nil || *v != 11.0/35.0 {
		t.Errorf("part d'AllyA = %v", v)
	}
	if v := valeur365(t, resp, "kda", "Test"); v == nil {
		t.Error("ligne du joueur principal absente")
	}
	if n := lecturesQ32b(l); n != 1 {
		t.Errorf("%d lectures de l'équipe alliée, attendu 1", n)
	}
}

func TestGetSquadTrends_Stricte_MemesMatchsQueLesSessions(t *testing.T) {
	l := &lecturesComptees{mockSquadRepo: repoTendances()}
	resp, err := nouveauServiceTendances(l).GetSquadTrends(context.Background(), "px",
		domain.TrendsQueryRequest{View: "squad", SelectedGamertags: []string{"AllyA", "AllyB"}, ExactComposition: true})
	if err != nil {
		t.Fatal(err)
	}
	if n := lecturesQ32b(l); n != 1 {
		t.Errorf("%d lectures de l'équipe alliée, attendu 1", n)
	}
	sessions, _, err := serviceDe(repoTendances(), nil).CompositionSessions(
		context.Background(), "px", []string{"AllyA", "AllyB"}, true)
	if err != nil {
		t.Fatal(err)
	}
	retenus := 0
	for _, s := range sessions {
		retenus += s.MatchCount
	}
	v := valeur365(t, resp, "match_count", "")
	if v == nil || int(*v) != retenus || retenus != 1 {
		t.Fatalf("match_count %v, matchs des sessions %d : attendu 1 des deux côtés", v, retenus)
	}
}

func TestGetSquadTrends_EquipeAllieeIllisible(t *testing.T) {
	for _, exact := range []bool{false, true} {
		repo := repoTendances()
		repo.allyErr = errors.New("q32b en echec")
		_, err := nouveauServiceTendances(&lecturesComptees{mockSquadRepo: repo}).GetSquadTrends(
			context.Background(), "px",
			domain.TrendsQueryRequest{View: "squad", SelectedGamertags: []string{"AllyA", "AllyB"}, ExactComposition: exact})
		if err == nil {
			t.Errorf("exact=%v : erreur attendue", exact)
		}
	}
}

func TestGetSquadTrends_CoequipierIntrouvable(t *testing.T) {
	l := &lecturesComptees{mockSquadRepo: repoTendances()}
	resp, err := nouveauServiceTendances(l).GetSquadTrends(context.Background(), "px",
		domain.TrendsQueryRequest{View: "squad", SelectedGamertags: []string{"AllyA", "Inconnu"}})
	if err != nil {
		t.Fatal(err)
	}
	if valeur365(t, resp, "kda", "AllyA") == nil || valeur365(t, resp, "kda", "Inconnu") != nil {
		t.Error("seul AllyA doit figurer parmi les membres")
	}
}

func TestGetSquadTrends_AucunCoequipierResolu(t *testing.T) {
	l := &lecturesComptees{mockSquadRepo: repoTendances()}
	resp, err := nouveauServiceTendances(l).GetSquadTrends(context.Background(), "px",
		domain.TrendsQueryRequest{View: "squad", SelectedGamertags: []string{"Inconnu"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Indicators) != 0 || resp.Indicators == nil || resp.View != domain.TrendsViewSquad {
		t.Errorf("attendu une réponse valide sans indicateur : %+v", resp)
	}
}

func TestGetSquadTrends_RequeteAnnulee(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := nouveauServiceTendances(&lecturesComptees{mockSquadRepo: repoTendances()}).GetSquadTrends(
		ctx, "px", domain.TrendsQueryRequest{View: "squad", SelectedGamertags: []string{"AllyA", "AllyB"}})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("erreur = %v, attendu context.Canceled", err)
	}
}
