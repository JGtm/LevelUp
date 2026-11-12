package service

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/analysis/leadshare"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
)

// leadShareFake : un lecteur de Mainmise à réponse fixe, qui note ce qu'on lui demande.
type leadShareFake struct {
	rendu      map[string]leadshare.PlayerShare
	err        error
	vusIDs     []string
	vuXUID     string
	nbLectures int
}

func (f *leadShareFake) LoadLeadShare(_ context.Context, ids []string, xuid string) (map[string]leadshare.PlayerShare, error) {
	f.nbLectures++
	f.vusIDs, f.vuXUID = ids, xuid
	return f.rendu, f.err
}

func TestApplyHeaderLeadShare_MatchMesure(t *testing.T) {
	f := &leadShareFake{rendu: map[string]leadshare.PlayerShare{"m1": {Share: 0.62, Changes: 4}}}
	s := &MatchViewService{xuid: "xu1", leadShareRepo: f}
	h := domain.MatchViewHeader{MatchID: "m1"}
	s.applyHeaderLeadShare(context.Background(), &h)
	if h.LeadShare == nil || *h.LeadShare != 0.62 || h.LeadChanges == nil || *h.LeadChanges != 4 {
		t.Fatalf("en-tête = %+v, attendu 0,62 et 4 retournements", h)
	}
	if f.vuXUID != "xu1" || len(f.vusIDs) != 1 || f.vusIDs[0] != "m1" {
		t.Errorf("lecture pour %q sur %v, attendu xu1 et [m1]", f.vuXUID, f.vusIDs)
	}
}

func TestApplyHeaderLeadShare_NonMesureResteAbsent(t *testing.T) {
	// Absent du résultat = non mesuré : ni zéro, ni zéro retournement.
	s := &MatchViewService{xuid: "xu1", leadShareRepo: &leadShareFake{rendu: map[string]leadshare.PlayerShare{}}}
	h := domain.MatchViewHeader{MatchID: "m1"}
	s.applyHeaderLeadShare(context.Background(), &h)
	if h.LeadShare != nil || h.LeadChanges != nil {
		t.Errorf("en-tête = %+v, attendu aucun champ de Mainmise", h)
	}
}

func TestApplyHeaderLeadShare_Degradations(t *testing.T) {
	for nom, s := range map[string]*MatchViewService{
		"sans lecteur":       {xuid: "xu1"},
		"sans joueur":        {leadShareRepo: &leadShareFake{}},
		"capability absente": {xuid: "xu1", leadShareRepo: &leadShareFake{err: games.ErrCapabilityNotSupported}},
		"lecture en échec":   {xuid: "xu1", leadShareRepo: &leadShareFake{err: errors.New("boom")}},
	} {
		t.Run(nom, func(t *testing.T) {
			h := domain.MatchViewHeader{MatchID: "m1"}
			s.applyHeaderLeadShare(context.Background(), &h)
			if h.LeadShare != nil || h.LeadChanges != nil {
				t.Errorf("en-tête = %+v, attendu aucun champ de Mainmise", h)
			}
		})
	}
}
