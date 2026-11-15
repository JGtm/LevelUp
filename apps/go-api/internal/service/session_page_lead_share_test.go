package service

import (
	"context"
	"math"
	"testing"

	"levelup/go-api/internal/analysis/leadshare"
	"levelup/go-api/internal/domain"
)

func sessionAvecMainmise(rendu map[string]leadshare.PlayerShare) (*SessionPageService, *leadShareFake) {
	f := &leadShareFake{rendu: rendu}
	s := &SessionPageService{}
	s.sessionXUID = "xu1"
	s.leadShareRepo = f
	return s, f
}

func reponseDeTest() *domain.SessionPageResponse {
	return &domain.SessionPageResponse{
		CurrentSession: &domain.SessionCompareEntry{SessionLabel: "A"},
		Matches:        []domain.SessionDetailMatchRow{{MatchID: "m1"}, {MatchID: "m2"}, {MatchID: "m3"}},
		CompareSession: &domain.SessionCompareEntry{SessionLabel: "B"},
		CompareMatches: []domain.SessionDetailMatchRow{{MatchID: "c1"}, {MatchID: "c2"}},
	}
}

func TestAttachSessionLeadShare_LignesEtMoyenneDesMatchsMesures(t *testing.T) {
	s, f := sessionAvecMainmise(map[string]leadshare.PlayerShare{
		"m1": {Share: 0.2, Changes: 1},
		"m3": {Share: 0.6, Changes: 5},
		// m2 : non mesuré. c1 mesuré, c2 non.
		"c1": {Share: 0.9, Changes: 0},
	})
	resp := reponseDeTest()
	s.attachSessionLeadShare(context.Background(), resp)

	if f.nbLectures != 1 || len(f.vusIDs) != 5 || f.vuXUID != "xu1" {
		t.Errorf("%d lecture(s) de %d matchs pour %q : attendu une seule, de 5 matchs, pour xu1", f.nbLectures, len(f.vusIDs), f.vuXUID)
	}
	if resp.Matches[1].LeadShare != nil || resp.Matches[1].LeadChanges != nil {
		t.Error("m2 non mesuré porte une mesure : jamais zéro")
	}
	if v := resp.Matches[2]; v.LeadShare == nil || *v.LeadShare != 0.6 || v.LeadChanges == nil || *v.LeadChanges != 5 {
		t.Errorf("m3 = %+v, attendu 0,6 et 5 retournements", v)
	}
	// Moyenne des deux matchs mesurés (0,2 et 0,6), le non mesuré n'entrant pas au dénominateur.
	cur := resp.CurrentSession
	if cur.LeadShare == nil || math.Abs(*cur.LeadShare-0.4) > 1e-9 || cur.LeadShareMatches != 2 {
		t.Errorf("session courante : %+v / %d, attendu 0,4 sur 2 matchs mesurés", cur.LeadShare, cur.LeadShareMatches)
	}
	cmp := resp.CompareSession
	if cmp.LeadShare == nil || math.Abs(*cmp.LeadShare-0.9) > 1e-9 || cmp.LeadShareMatches != 1 {
		t.Errorf("session comparée : %+v / %d, attendu 0,9 sur 1 match mesuré", cmp.LeadShare, cmp.LeadShareMatches)
	}
}

func TestAttachSessionLeadShare_AucunMatchMesureLaSessionNePorteRien(t *testing.T) {
	s, _ := sessionAvecMainmise(map[string]leadshare.PlayerShare{})
	resp := reponseDeTest()
	s.attachSessionLeadShare(context.Background(), resp)
	if resp.CurrentSession.LeadShare != nil || resp.CurrentSession.LeadShareMatches != 0 ||
		resp.CompareSession.LeadShare != nil {
		t.Errorf("sessions = %+v / %+v, attendu aucune mesure (jamais 0 %%)", resp.CurrentSession, resp.CompareSession)
	}
}

func TestAttachSessionLeadShare_SansLecteurOuSansMatch(t *testing.T) {
	s := &SessionPageService{}
	s.sessionXUID = "xu1"
	resp := reponseDeTest()
	s.attachSessionLeadShare(context.Background(), resp) // pas de lecteur : no-op
	if resp.CurrentSession.LeadShare != nil {
		t.Error("sans lecteur, la session ne porte rien")
	}
	vide, f := sessionAvecMainmise(nil)
	vide.attachSessionLeadShare(context.Background(), &domain.SessionPageResponse{})
	if f.nbLectures != 0 {
		t.Errorf("%d lecture(s) pour une page sans match, attendu aucune", f.nbLectures)
	}
}
