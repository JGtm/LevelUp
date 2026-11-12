package service

import (
	"context"
	"math"
	"testing"
	"time"

	"levelup/go-api/internal/analysis/leadshare"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/i18n"
	"levelup/go-api/internal/legacymatch"
)

func ligneDeCarte(id, carte string, quand time.Time) legacymatch.StatsMatchRow {
	return legacymatch.StatsMatchRow{MatchID: id, StartTime: quand, MapName: i18n.FromColumns("", carte)}
}

// Trois matchs d'octobre (deux mesurés), un de novembre non mesuré, un de novembre mesuré sur une
// autre carte.
func populationMainmise() ([]legacymatch.StatsMatchRow, map[string]leadshare.PlayerShare) {
	oct := time.Date(2026, 10, 5, 20, 0, 0, 0, time.UTC)
	nov := time.Date(2026, 11, 3, 20, 0, 0, 0, time.UTC)
	rows := []legacymatch.StatsMatchRow{
		ligneDeCarte("m1", "Bazaar", oct), ligneDeCarte("m2", "Bazaar", oct.Add(time.Hour)),
		ligneDeCarte("m3", "Recharge", oct.Add(2*time.Hour)),
		ligneDeCarte("m4", "Bazaar", nov), ligneDeCarte("m5", "Streets", nov.Add(time.Hour)),
	}
	got := map[string]leadshare.PlayerShare{
		"m1": {Share: 0.2, Changes: 1}, "m2": {Share: 0.6, Changes: 2},
		"m5": {Share: 0.5, Changes: 0},
		// m3 et m4 : non mesurés.
	}
	return rows, got
}

func TestLeadShareByMonth_MoyenneDesMatchsMesures(t *testing.T) {
	rows, got := populationMainmise()
	mois := leadShareByMonth(rows, got)
	if len(mois) != 2 || mois[0].Month != "2026-10" || mois[1].Month != "2026-11" {
		t.Fatalf("mois = %+v, attendu octobre puis novembre", mois)
	}
	oct := mois[0]
	if oct.Matches != 3 || oct.MeasuredMatches != 2 || oct.LeadShare == nil || math.Abs(*oct.LeadShare-0.4) > 1e-9 {
		t.Errorf("octobre = %+v, attendu 3 matchs dont 2 mesurés, moyenne 0,4 (le non mesuré hors dénominateur)", oct)
	}
	if nov := mois[1]; nov.Matches != 2 || nov.MeasuredMatches != 1 || nov.LeadShare == nil || *nov.LeadShare != 0.5 {
		t.Errorf("novembre = %+v, attendu 2 matchs dont 1 mesuré, moyenne 0,5", nov)
	}
}

func TestLeadShareByMonth_UnMoisSansMesureNaPasDeMoyenne(t *testing.T) {
	rows, _ := populationMainmise()
	mois := leadShareByMonth(rows, map[string]leadshare.PlayerShare{})
	for _, m := range mois {
		if m.LeadShare != nil || m.MeasuredMatches != 0 || m.Matches == 0 {
			t.Errorf("%+v : attendu aucun chiffre (jamais 0 %%) mais les matchs comptés", m)
		}
	}
}

func TestLeadShareByMap_ParCleDeCarte(t *testing.T) {
	rows, got := populationMainmise()
	par := leadShareByMap(rows, got, domain.GroupesDeCartes{}, i18n.Default)
	if a := par["Bazaar"]; a.n != 2 || math.Abs(a.avg-0.4) > 1e-9 {
		t.Errorf("Bazaar = %+v, attendu 0,4 sur 2 matchs mesurés (m4 non mesuré hors dénominateur)", a)
	}
	if _, ok := par["Recharge"]; ok {
		t.Error("Recharge, sans match mesuré, porte une moyenne")
	}
	if a := par["Streets"]; a.n != 1 || a.avg != 0.5 {
		t.Errorf("Streets = %+v", a)
	}
}

func TestAttachLeadShare_LignesCartesEtMois(t *testing.T) {
	rows, got := populationMainmise()
	f := &leadShareFake{rendu: got}
	s := &TimeseriesService{leadShareRepo: f, playerXUID: "xu1"}
	filtered := rows[:3] // octobre seulement
	resp := &domain.TimeseriesPageResponse{
		MatchRows: []domain.TimeseriesMatchRow{{MatchID: "m1"}, {MatchID: "m3"}},
		MapBreakdown: []domain.MapBreakdownRow{
			{MapID: "Bazaar"}, {MapID: "Recharge"},
		},
	}
	s.attachLeadShare(context.Background(), resp, filtered, rows, domain.GroupesDeCartes{})

	if f.nbLectures != 1 || len(f.vusIDs) != 5 || f.vuXUID != "xu1" {
		t.Errorf("%d lecture(s) de %d matchs : attendu une seule, de 5 matchs sans doublon", f.nbLectures, len(f.vusIDs))
	}
	if r := resp.MatchRows[0]; r.LeadShare == nil || *r.LeadShare != 0.2 || r.LeadChanges == nil || *r.LeadChanges != 1 {
		t.Errorf("ligne m1 = %+v", r)
	}
	if r := resp.MatchRows[1]; r.LeadShare != nil || r.LeadChanges != nil {
		t.Errorf("ligne m3 non mesurée = %+v : jamais zéro", r)
	}
	baz := resp.MapBreakdown[0]
	if baz.LeadShare == nil || math.Abs(*baz.LeadShare-0.4) > 1e-9 || baz.LeadShareMatches != 2 {
		t.Errorf("Bazaar filtré = %v / %d, attendu 0,4 sur 2", baz.LeadShare, baz.LeadShareMatches)
	}
	// Historique de Bazaar : m1, m2, m4 -> m4 non mesuré, donc 0,4 sur 2 matchs aussi.
	if baz.HistoricalLeadShare == nil || baz.HistoricalLeadShareMatches != 2 {
		t.Errorf("Bazaar historique = %v / %d", baz.HistoricalLeadShare, baz.HistoricalLeadShareMatches)
	}
	if rch := resp.MapBreakdown[1]; rch.LeadShare != nil || rch.HistoricalLeadShare != nil {
		t.Errorf("Recharge = %+v : aucune mesure, aucun chiffre", rch)
	}
	if len(resp.LeadShareByMonth) != 2 {
		t.Errorf("%d mois, attendu 2", len(resp.LeadShareByMonth))
	}
}

func TestAttachLeadShare_SansLecteurNePoseRien(t *testing.T) {
	rows, _ := populationMainmise()
	s := &TimeseriesService{playerXUID: "xu1"}
	resp := &domain.TimeseriesPageResponse{MatchRows: []domain.TimeseriesMatchRow{{MatchID: "m1"}}}
	s.attachLeadShare(context.Background(), resp, rows, rows, domain.GroupesDeCartes{})
	if resp.MatchRows[0].LeadShare != nil || resp.LeadShareByMonth != nil {
		t.Error("sans lecteur, la page ne porte rien")
	}
}
