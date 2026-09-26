package service

// session_page_flag_grabs_net_test.go — LE CHEMIN DE LECTURE DES PRISES NETTES sur la page
// Sessions, de bout en bout du service.
//
// CE FICHIER EXISTE PARCE QUE LA REVUE DU 2026-09-13 A RENDU `attachFlagGrabsNet` NO-OP ET QUE
// RIEN N'A ROUGI (mutation M7, constat C1). Le premier test ci-dessous rougit sous cette
// mutation ; les suivants tiennent les deux dégradations et le dénominateur de couverture.

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/analysis/narrative"
	"levelup/go-api/internal/analysis/sessionusage"
	"levelup/go-api/internal/domain"
)

// mockObjectiveIndexAvecPrises implémente port.ObjectiveIndexRepository, la capability
// `objectiveRoleRowsLoader` ET la capability `flagGrabsNetLoader` — comme
// duckdb.ObjectiveStatsRepo, qui les porte toutes les trois.
type mockObjectiveIndexAvecPrises struct {
	mockObjectiveIndexWithRoles
	prises    []sessionusage.FlagGrabsNetRow
	prisesErr error
}

func (m *mockObjectiveIndexAvecPrises) LoadFlagGrabsNet(_ context.Context, _ []string) ([]sessionusage.FlagGrabsNetRow, error) {
	return m.prises, m.prisesErr
}

// serviceAvecPrises câble le service avec un scope de deux matchs de CTF, dont un seul mesuré.
func serviceAvecPrises(index *mockObjectiveIndexAvecPrises) *SessionPageService {
	svc := NewSessionPageService(nil).WithSessionUsage(usageTestRepoMock(), "P", nil, "")
	svc.objectiveIndex = index
	return svc
}

// rolesCTFDeuxMatchs : deux matchs de la famille DRAPEAU, les deux camps.
func rolesCTFDeuxMatchs() []sessionusage.ObjectiveRow {
	return []sessionusage.ObjectiveRow{
		{MatchID: "m1", XUID: "P", Family: narrative.FamilyCTF, Take: 2},
		{MatchID: "m1", XUID: "A", Family: narrative.FamilyCTF, Take: 1},
		{MatchID: "m1", XUID: "E1", Family: narrative.FamilyCTF, Take: 3},
		{MatchID: "m2", XUID: "P", Family: narrative.FamilyCTF, Take: 1},
	}
}

func usageAvecPrises(t *testing.T, index *mockObjectiveIndexAvecPrises) *domain.SessionUsageBlock {
	t.Helper()
	var resp domain.SessionPageResponse
	serviceAvecPrises(index).attachSessionUsage(
		context.Background(), &resp, sessionBlocksScope{Matches: usageTestMatches(), MatchContext: domain.MatchContextSolo, Locale: "fr"})
	if resp.Usage == nil || resp.Usage.Objectives == nil {
		t.Fatalf("bloc objectifs absent : %+v", resp.Usage)
	}
	return resp.Usage
}

// TestAttachFlagGrabsNet_LeBlocEstServi — LE test qui rougit sous la mutation M7 (le crochet
// rendu no-op) : sans lui, débrancher la lecture ne cassait rien.
func TestAttachFlagGrabsNet_LeBlocEstServi(t *testing.T) {
	index := &mockObjectiveIndexAvecPrises{
		mockObjectiveIndexWithRoles: mockObjectiveIndexWithRoles{roleRows: rolesCTFDeuxMatchs()},
		prises: []sessionusage.FlagGrabsNetRow{
			{MatchID: "m1", XUID: "P", Raw: 9, Net: 3, Openings: 25, WindowMS: 1500},
			{MatchID: "m1", XUID: "A", Raw: 4, Net: 4, Openings: 25, WindowMS: 1500},
			{MatchID: "m1", XUID: "E1", Raw: 7, Net: 2, Openings: 25, WindowMS: 1500},
		},
	}
	fgn := usageAvecPrises(t, index).Objectives.FlagGrabsNet
	if fgn == nil {
		t.Fatal("flag_grabs_net absent du bloc objectifs — le crochet de lecture est débranché")
	}
	if fgn.PlayerTotal != 3 || fgn.PlayerRawTotal != 9 {
		t.Errorf("joueur = (%d net, %d brut), attendu (3, 9)", fgn.PlayerTotal, fgn.PlayerRawTotal)
	}
	if fgn.TeamTotal != 7 {
		t.Errorf("camp = %d, attendu 7 (P et A, pas l'adversaire)", fgn.TeamTotal)
	}
	if fgn.OpeningsTotal != 25 {
		t.Errorf("ouvertures = %d, attendu 25 (une fois par match)", fgn.OpeningsTotal)
	}
	if fgn.WindowSeconds != 1.5 {
		t.Errorf("fenêtre = %v, attendu 1.5", fgn.WindowSeconds)
	}
}

// TestAttachFlagGrabsNet_DenominateurEstLaFamilleDrapeau — CORRECTION DE REVUE (constat 6).
// La session porte DEUX matchs à objectif, dont UN SEUL de la famille drapeau : la couverture
// doit se lire sur ce seul match, sinon l'écran attribue au film l'absence d'un match de
// zones — un match qui n'a pas de drapeau.
func TestAttachFlagGrabsNet_DenominateurEstLaFamilleDrapeau(t *testing.T) {
	index := &mockObjectiveIndexAvecPrises{
		mockObjectiveIndexWithRoles: mockObjectiveIndexWithRoles{roleRows: []sessionusage.ObjectiveRow{
			{MatchID: "m1", XUID: "P", Family: narrative.FamilyCTF, Take: 2},
			{MatchID: "m2", XUID: "P", Family: narrative.FamilyZonesKOTH, Take: 5},
		}},
		prises: []sessionusage.FlagGrabsNetRow{
			{MatchID: "m1", XUID: "P", Raw: 4, Net: 2, Openings: 10, WindowMS: 1500},
		},
	}
	u := usageAvecPrises(t, index)
	if u.Objectives.MatchesWithObjectives != 2 {
		t.Fatalf("matchs à objectif = %d, attendu 2 (le scope global n'a pas changé)",
			u.Objectives.MatchesWithObjectives)
	}
	fgn := u.Objectives.FlagGrabsNet
	if fgn == nil {
		t.Fatal("flag_grabs_net absent")
	}
	if fgn.MatchesWithFlagFamily != 1 {
		t.Errorf("dénominateur = %d, attendu 1 — le match de zones ne doit pas y entrer",
			fgn.MatchesWithFlagFamily)
	}
	if fgn.MatchesMeasured != 1 {
		t.Errorf("mesurés = %d, attendu 1 (couverture pleine : 1 sur 1)", fgn.MatchesMeasured)
	}
}

// TestAttachFlagGrabsNet_LectureEnEchecDegradeSeule — le reste du bloc objectifs est servi.
func TestAttachFlagGrabsNet_LectureEnEchecDegradeSeule(t *testing.T) {
	index := &mockObjectiveIndexAvecPrises{
		mockObjectiveIndexWithRoles: mockObjectiveIndexWithRoles{roleRows: rolesCTFDeuxMatchs()},
		prisesErr:                   errors.New("vue absente"),
	}
	u := usageAvecPrises(t, index)
	if u.Objectives.FlagGrabsNet != nil {
		t.Errorf("bloc publié malgré l'échec de lecture : %+v", u.Objectives.FlagGrabsNet)
	}
	if len(u.Objectives.Roles) == 0 {
		t.Error("les rôles ont disparu avec les prises nettes — la dégradation doit être isolée")
	}
}

// TestAttachFlagGrabsNet_MontageSansLoaderResteSilencieux — un repo qui ne porte pas la
// capability optionnelle (titre qui ne produit pas la grandeur) ne casse rien.
func TestAttachFlagGrabsNet_MontageSansLoaderResteSilencieux(t *testing.T) {
	svc := NewSessionPageService(nil).WithSessionUsage(usageTestRepoMock(), "P", nil, "")
	// mockObjectiveIndexWithRoles n'implémente PAS flagGrabsNetLoader.
	svc.objectiveIndex = &mockObjectiveIndexWithRoles{roleRows: rolesCTFDeuxMatchs()}
	var resp domain.SessionPageResponse
	svc.attachSessionUsage(context.Background(), &resp, sessionBlocksScope{Matches: usageTestMatches(), MatchContext: domain.MatchContextSolo, Locale: "fr"})
	if resp.Usage == nil || resp.Usage.Objectives == nil {
		t.Fatal("bloc objectifs absent")
	}
	if resp.Usage.Objectives.FlagGrabsNet != nil {
		t.Errorf("bloc publié sans loader câblé : %+v", resp.Usage.Objectives.FlagGrabsNet)
	}
}
