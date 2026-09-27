package teammates

// teammates_service_objective_history_test.go — l'historique d'objectif de la composition (lot
// L3, D6 / D7) publié par GetPage : soirée affichée = périmètre D2, soirée précédente tirée de
// l'historique de la composition, camp = équipe alliée du joueur principal, mode écarté par le
// prédicat du titre.

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"levelup/go-api/internal/analysis/narrative"
	"levelup/go-api/internal/analysis/sessionusage"
	"levelup/go-api/internal/analysis/squadformes"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/legacymatch"
)

// objectifsFactices — port.SquadFormesObjectiveRepository : les lignes rendues, les ids lus.
type objectifsFactices struct {
	rows []squadformes.ObjectiveColumnRow
	err  error
	lus  []string
}

func (o *objectifsFactices) LoadObjectiveColumnRows(_ context.Context, ids []string) ([]squadformes.ObjectiveColumnRow, error) {
	o.lus = ids
	return o.rows, o.err
}
func (o *objectifsFactices) LoadFlagGrabsNet(context.Context, []string) ([]sessionusage.FlagGrabsNetRow, error) {
	return nil, nil
}

// historiqueFixture — soirée S1 (3 Drapeau, parts 50 %) puis soirée S2 (4 matchs dont un drapeau
// neutre, parts 25 %). Le filtre de la page ne garde que S2.
func historiqueFixture() (*mockSquadRepo, *objectifsFactices) {
	t0 := time.Date(2026, 9, 1, 20, 0, 0, 0, time.UTC)
	s1, s2 := "S1", "S2"
	repo := &mockSquadRepo{topRows: []domain.TopTeammateRow{{XUID: "x1", Gamertag: "Ally1", GamesTogether: 7}}}
	obj := &objectifsFactices{}
	ajoute := func(id, label, pair string, at time.Time, filtre bool, nous float64) {
		sess := label
		repo.squadRows = append(repo.squadRows, domain.SquadMatchRow{
			MatchID: id, StartTime: at, SessionLabel: &sess, PairName: pair, Outcome: domain.OutcomeWin,
		})
		if filtre {
			repo.synthRows = append(repo.synthRows, legacymatch.SynthesisMatchRow{MatchID: id, StartTime: at, Outcome: domain.OutcomeWin})
		}
		repo.allyRows = append(repo.allyRows,
			domain.AllyParticipant{MatchID: id, XUID: "player-xuid"}, domain.AllyParticipant{MatchID: id, XUID: "x1"})
		ligne := func(xuid string, v float64) squadformes.ObjectiveColumnRow {
			return squadformes.ObjectiveColumnRow{MatchID: id, XUID: xuid, Family: narrative.FamilyCTF, Values: map[string]float64{
				"flag_captures": v, "flag_returns": v, "time_as_flag_carrier_seconds": v * 10,
			}}
		}
		obj.rows = append(obj.rows, ligne("player-xuid", nous), ligne("adv", 4-nous))
	}
	for i := 0; i < 3; i++ {
		ajoute(fmt.Sprintf("a%d", i), s1, "Arena:CTF on Aquarius", t0.Add(time.Duration(i)*time.Minute), false, 2)
	}
	for i := 0; i < 3; i++ {
		ajoute(fmt.Sprintf("b%d", i), s2, "Arena:CTF on Aquarius", t0.AddDate(0, 0, 1).Add(time.Duration(i)*time.Minute), true, 1)
	}
	ajoute("bn", s2, "NEUTRE", t0.AddDate(0, 0, 1).Add(10*time.Minute), true, 4)
	return repo, obj
}

func estNeutre(pair string) bool { return pair == "NEUTRE" }

// TestTeammatesService_GetPage_HistoriqueObjectif — la soirée affichée compte 3 matchs (le drapeau
// neutre écarté) à 25 %, la soirée précédente 3 matchs à 50 % ; toute l'histoire est lue en une
// lecture.
func TestTeammatesService_GetPage_HistoriqueObjectif(t *testing.T) {
	t.Parallel()
	repo, obj := historiqueFixture()
	svc := NewTeammatesService(repo, nil).
		WithPlayerMatchesRepo(newSynthMockFromRows(repo.synthRows, nil), "halo_infinite", "Main").
		WithSquadFormes(nil, obj, "").
		WithObjectiveHistory(estNeutre)
	resp, err := svc.GetPage(context.Background(), "player-xuid", domain.TeammatesQueryRequest{SelectedGamertags: []string{"Ally1"}})
	if err != nil {
		t.Fatalf("GetPage : %v", err)
	}
	h := resp.SquadObjectiveHistory
	if h == nil {
		t.Fatal("historique d'objectif absent")
	}
	if h.Current.ObjectiveMatches != 3 || h.Current.Take == nil || *h.Current.Take != 0.25 || h.Current.SessionLabel != "S2" {
		t.Errorf("soirée affichée = %+v, attendu 3 matchs à 25 %% (drapeau neutre écarté), session S2", h.Current)
	}
	if len(h.Previous) != 1 || h.Previous[0].SessionLabel != "S1" || *h.Previous[0].Take != 0.5 || h.Previous[0].Wins != 3 {
		t.Errorf("soirées précédentes = %+v, attendu S1 à 50 %%, 3 victoires", h.Previous)
	}
	if len(obj.lus) != 7 {
		t.Errorf("%d matchs lus, attendu les 7 de la composition en une lecture", len(obj.lus))
	}
}

// TestTeammatesService_GetPage_HistoriqueObjectifDegrade — lecture en échec ou titre sans stats
// d'objectif : pas d'historique, la page se sert quand même.
func TestTeammatesService_GetPage_HistoriqueObjectifDegrade(t *testing.T) {
	t.Parallel()
	for _, cas := range []struct {
		nom string
		obj *objectifsFactices
	}{{"lecture en échec", &objectifsFactices{err: errors.New("vue absente")}}, {"titre sans stats", nil}} {
		repo, _ := historiqueFixture()
		svc := NewTeammatesService(repo, nil).
			WithPlayerMatchesRepo(newSynthMockFromRows(repo.synthRows, nil), "halo_infinite", "Main")
		if cas.obj != nil {
			svc = svc.WithSquadFormes(nil, cas.obj, "")
		}
		resp, err := svc.GetPage(context.Background(), "player-xuid", domain.TeammatesQueryRequest{SelectedGamertags: []string{"Ally1"}})
		if err != nil {
			t.Fatalf("%s : GetPage : %v", cas.nom, err)
		}
		if resp.SquadObjectiveHistory != nil {
			t.Errorf("%s : historique publié %+v, attendu nil", cas.nom, resp.SquadObjectiveHistory)
		}
	}
}
