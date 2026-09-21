package service

// coordination_block_test.go — LE PRODUCTEUR PARTAGÉ (lot N1, 2026-09-21).
//
// Ce que ces tests cadenassent :
//   - les deux mailles sortent du MÊME appel : cases par match (Sessions) OU points par
//     soirée (Timeseries), jamais les deux ;
//   - une capability de journal des morts fermée rend un bloc INDISPONIBLE avec sa raison
//     machine, jamais un bloc de zéros ;
//   - une lecture d'appuis en échec dégrade LE VERSANT APPUI, pas le bloc entier ;
//   - l'effectif de camp reçu de l'appelant alimente la parité (réserve R1).

import (
	"context"
	"errors"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/port"
)

// tacticalRepoStub — mock du port du journal des morts.
//
// L'INTERFACE EST EMBARQUÉE, NON RECOPIÉE : le bloc n'appelle que `KillEvents`, et
// réécrire les huit autres méthodes aurait fait du test un miroir du port — miroir qu'il
// aurait fallu suivre à chaque ajout, sans qu'aucune assertion n'en dépende. Une méthode
// non surchargée appelée par erreur panique sur le nil embarqué : c'est le comportement
// voulu, pas un silence.
type tacticalRepoStub struct {
	port.TacticalRepository
	lecture domain.TacticalKillEvents
	err     error
	vus     []string
}

func (s *tacticalRepoStub) KillEvents(_ context.Context, q domain.TacticalQuery) (domain.TacticalKillEvents, error) {
	s.vus = q.Matchs.IDs()
	return s.lecture, s.err
}

// appuisRepoStub — mock du port des appuis.
type appuisRepoStub struct {
	rows []domain.CoordinationAppuiRow
	err  error
}

func (s *appuisRepoStub) LoadAppuis(context.Context, []string) ([]domain.CoordinationAppuiRow, error) {
	return s.rows, s.err
}

// lectureDeTest — deux matchs mesurés, deux soirées, un camp à quatre.
//
//	m1  E1 tue A a 1 s, MOI tue E1 a 3 s  -> une mort de camp, ripostee par moi ;
//	m2  E1 tue MOI a 1 s, A tue E1 a 2 s  -> ma mort, ripostee par un coequipier.
func lectureDeTest() domain.TacticalKillEvents {
	equipe := map[string]int{"P": 0, "A": 0, "E1": 1}
	return domain.TacticalKillEvents{
		Univers: domain.TacticalUnivers{
			Matchs: []domain.TacticalMatch{
				{MatchID: "m1", Mesure: true}, {MatchID: "m2", Mesure: true},
			},
			Equipes: domain.EquipesParMatch{"m1": equipe, "m2": equipe},
		},
		Events: []domain.KillEvent{
			{MatchID: "m1", KillerXUID: "E1", VictimXUID: "A", TimeMs: 1000},
			{MatchID: "m1", KillerXUID: "P", VictimXUID: "E1", TimeMs: 3000},
			{MatchID: "m2", KillerXUID: "E1", VictimXUID: "P", TimeMs: 1000},
			{MatchID: "m2", KillerXUID: "A", VictimXUID: "E1", TimeMs: 2000},
		},
	}
}

func requeteDeTest() coordinationQuery {
	return coordinationQuery{
		Tactical: &tacticalRepoStub{lecture: lectureDeTest()},
		Appuis: &appuisRepoStub{rows: []domain.CoordinationAppuiRow{
			{MatchID: "m1", AssistXUID: "A", KillerXUID: "P", Nombre: 1},
			{MatchID: "m1", AssistXUID: "", KillerXUID: "P", Nombre: 1},
		}},
		Caps:       games.CapabilityMap{games.CapFilmKillSource: games.CapSupported},
		PlayerXUID: "P",
		MatchIDs:   []string{"m1", "m2"},
		TeamSize:   map[string]int{"m1": 4, "m2": 4},
	}
}

func TestBuildCoordinationBlock_MailleMatch(t *testing.T) {
	got := buildCoordinationBlock(context.Background(), requeteDeTest())

	if got == nil || !got.Available {
		t.Fatalf("bloc = %+v, attendu disponible", got)
	}
	if got.MatchesMeasured != 2 || got.MatchesTotal != 2 {
		t.Fatalf("couverture = %d/%d, attendu 2/2", got.MatchesMeasured, got.MatchesTotal)
	}
	if len(got.PerMatch) != 2 || len(got.Sessions) != 0 {
		t.Fatalf("%d cases et %d soirées, attendu 2 et 0 : la maille MATCH ne publie pas de soirées",
			len(got.PerMatch), len(got.Sessions))
	}
	if got.Riposte.TeamDeaths != 2 || got.Riposte.TeamDeathsAvenged != 2 {
		t.Errorf("morts de camp = %d/%d, attendu 2 ripostées sur 2",
			got.Riposte.TeamDeathsAvenged, got.Riposte.TeamDeaths)
	}
	if got.Riposte.ParityPct == nil || *got.Riposte.ParityPct != 25 {
		t.Errorf("parité = %v, attendu 25 — l'effectif vient de l'appelant (réserve R1)",
			got.Riposte.ParityPct)
	}
	if got.Appui.OnMePrepare.Brut != 1 || got.Appui.OnMePrepare.N != 2 {
		t.Errorf("on me prépare = %d/%d, attendu 1/2",
			got.Appui.OnMePrepare.Brut, got.Appui.OnMePrepare.N)
	}
	if got.FenetreMs != 5000 {
		t.Errorf("fenêtre = %d ms, attendu 5000", got.FenetreMs)
	}
}

// TestBuildCoordinationBlock_LePerimetreEstUneListeBlanche — le lecteur ne balaie pas
// l'historique : il reçoit les matchs du scope, comme le reste de la page.
func TestBuildCoordinationBlock_LePerimetreEstUneListeBlanche(t *testing.T) {
	q := requeteDeTest()
	stub := q.Tactical.(*tacticalRepoStub)

	buildCoordinationBlock(context.Background(), q)

	if len(stub.vus) != 2 {
		t.Fatalf("liste blanche = %v, attendu les 2 matchs du scope", stub.vus)
	}
}

func TestBuildCoordinationBlock_MailleSoiree(t *testing.T) {
	q := requeteDeTest()
	q.Soirees = []coordinationSoiree{
		{Label: "2026-09-20", MatchIDs: []string{"m1"}},
		{Label: "2026-09-21", MatchIDs: []string{"m2"}},
	}

	got := buildCoordinationBlock(context.Background(), q)

	if len(got.Sessions) != 2 || len(got.PerMatch) != 0 {
		t.Fatalf("%d soirées et %d cases, attendu 2 et 0 : la frise ne peint pas de cases par match",
			len(got.Sessions), len(got.PerMatch))
	}
	if got.Sessions[0].SessionLabel != "2026-09-20" {
		t.Errorf("première soirée = %q, attendu l'ordre reçu", got.Sessions[0].SessionLabel)
	}
	// Soirée 1 : la mort de A, vengée par moi. Soirée 2 : ma mort, vengée par A.
	if got.Sessions[0].Riposte.JeRiposte.Brut != 1 || got.Sessions[0].Riposte.JeSuisCouvert.N != 0 {
		t.Errorf("soirée 1 = %+v, attendu ma riposte et aucune de mes morts",
			got.Sessions[0].Riposte)
	}
	if got.Sessions[1].Riposte.JeSuisCouvert.Brut != 1 {
		t.Errorf("soirée 2 : je suis couvert = %+v, attendu 1 mort vengée",
			got.Sessions[1].Riposte.JeSuisCouvert)
	}
}

// TestBuildCoordinationBlock_CapabilityFermee — un titre qui ne nomme pas le tueur de
// chaque mort n'a pas un taux de riposte nul : il n'en a pas.
func TestBuildCoordinationBlock_CapabilityFermee(t *testing.T) {
	q := requeteDeTest()
	q.Caps = games.CapabilityMap{}

	got := buildCoordinationBlock(context.Background(), q)

	if got == nil || got.Available {
		t.Fatalf("bloc = %+v, attendu indisponible", got)
	}
	if got.UnavailableReason != domain.CoordinationUnsupported {
		t.Errorf("raison = %q, attendu %q", got.UnavailableReason, domain.CoordinationUnsupported)
	}
	if got.MatchesTotal != 2 {
		t.Errorf("matchs total = %d, attendu 2 : le scope reste publié", got.MatchesTotal)
	}
}

// TestBuildCoordinationBlock_AppuisEnEchec_DegradeUnSujet — dégrader UN sujet vaut mieux
// que retirer le bloc entier : la riposte reste servie, l'appui a des dénominateurs vides
// et la couverture le dit.
func TestBuildCoordinationBlock_AppuisEnEchec_DegradeUnSujet(t *testing.T) {
	q := requeteDeTest()
	q.Appuis = &appuisRepoStub{err: errors.New("lecteur indisponible")}

	got := buildCoordinationBlock(context.Background(), q)

	if !got.Available || got.Riposte.TeamDeaths != 2 {
		t.Fatalf("bloc = %+v, attendu la riposte servie", got)
	}
	if got.Appui.OnMePrepare.N != 0 || !got.Appui.OnMePrepare.EchantillonFaible {
		t.Errorf("appui = %+v, attendu un dénominateur vide et l'échantillon faible posé",
			got.Appui.OnMePrepare)
	}
}

// TestBuildCoordinationBlock_JournalEnEchec — lecture en échec = raison machine, jamais
// l'échec de la page.
func TestBuildCoordinationBlock_JournalEnEchec(t *testing.T) {
	q := requeteDeTest()
	q.Tactical = &tacticalRepoStub{err: errors.New("shared reader")}

	got := buildCoordinationBlock(context.Background(), q)

	if got.Available || got.UnavailableReason != domain.CoordinationLoadFailed {
		t.Fatalf("bloc = %+v, attendu indisponible pour cause de lecture", got)
	}
}

// TestBuildCoordinationBlock_ScopeVide — rien à dire n'est pas une panne : le champ est
// omis, pas rempli d'un message d'indisponibilité.
func TestBuildCoordinationBlock_ScopeVide(t *testing.T) {
	q := requeteDeTest()
	q.MatchIDs = nil

	if got := buildCoordinationBlock(context.Background(), q); got != nil {
		t.Fatalf("bloc = %+v, attendu nil sur un scope vide", got)
	}
}
