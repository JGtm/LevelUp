package teammates

import (
	"context"
	"errors"
	"testing"
	"time"

	"levelup/go-api/internal/analysis/leadshare"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
)

type leadShareFake struct {
	rendu  map[string]leadshare.PlayerShare
	err    error
	vusIDs []string
	vuXUID string
	n      int
}

func (f *leadShareFake) LoadLeadShare(_ context.Context, ids []string, xuid string) (map[string]leadshare.PlayerShare, error) {
	f.n++
	f.vusIDs, f.vuXUID = ids, xuid
	return f.rendu, f.err
}

func TestBuildSquadLeadShare_SerieChronologiqueEtNonMesuresSansChiffre(t *testing.T) {
	base := time.Date(2026, 11, 3, 20, 0, 0, 0, time.UTC)
	soiree := "2026-11-03"
	rows := []domain.SquadMatchRow{
		{MatchID: "m2", StartTime: base.Add(time.Hour), Outcome: 3, MapUI: "Recharge", SessionLabel: &soiree},
		{MatchID: "m1", StartTime: base, Outcome: 2, MapUI: "Bazaar", SessionLabel: &soiree},
		{MatchID: "m3", StartTime: base.Add(2 * time.Hour), Outcome: 2},
	}
	f := &leadShareFake{rendu: map[string]leadshare.PlayerShare{"m1": {Share: 0.7, Changes: 2}, "m2": {Share: 0.3, Changes: 0}}}
	s := &TeammatesService{leadShareRepo: f}
	got := s.buildSquadLeadShare(context.Background(), rows, "xu1")

	if f.n != 1 || f.vuXUID != "xu1" || len(f.vusIDs) != 3 {
		t.Errorf("%d lecture(s) pour %q de %d matchs : attendu une seule, pour xu1, de 3 matchs", f.n, f.vuXUID, len(f.vusIDs))
	}
	if len(got) != 3 || got[0].MatchID != "m1" || got[1].MatchID != "m2" || got[2].MatchID != "m3" {
		t.Fatalf("série = %+v, attendu m1, m2, m3 du plus ancien au plus récent", got)
	}
	if got[0].LeadShare == nil || *got[0].LeadShare != 0.7 || got[0].LeadChanges == nil || *got[0].LeadChanges != 2 {
		t.Errorf("m1 = %+v", got[0])
	}
	if got[1].LeadShare == nil || *got[1].LeadShare != 0.3 || got[1].LeadChanges == nil || *got[1].LeadChanges != 0 {
		t.Errorf("m2 = %+v : zéro retournement MESURÉ reste un zéro", got[1])
	}
	if got[2].LeadShare != nil || got[2].LeadChanges != nil {
		t.Errorf("m3 non mesuré = %+v : jamais zéro", got[2])
	}
	if got[0].SessionLabel == nil || got[0].MapName != "Bazaar" {
		t.Errorf("m1 sans sa soirée ou sa carte : %+v", got[0])
	}
}

func TestBuildSquadLeadShare_Degradations(t *testing.T) {
	rows := []domain.SquadMatchRow{{MatchID: "m1"}}
	for nom, s := range map[string]*TeammatesService{
		"sans lecteur":       {},
		"capability absente": {leadShareRepo: &leadShareFake{err: games.ErrCapabilityNotSupported}},
		"lecture en échec":   {leadShareRepo: &leadShareFake{err: errors.New("boom")}},
	} {
		if got := s.buildSquadLeadShare(context.Background(), rows, "xu1"); got != nil {
			t.Errorf("%s : série %+v, attendu aucune série", nom, got)
		}
	}
	if got := (&TeammatesService{leadShareRepo: &leadShareFake{}}).buildSquadLeadShare(context.Background(), nil, "xu1"); got != nil {
		t.Errorf("sans match : %+v", got)
	}
}
