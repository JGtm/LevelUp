package teammates

// teammates_squad_impact_test.go — le tableau d'impact par session (teammates_squad_impact.go) :
// une lecture des événements et du journal des morts pour toutes les sessions.

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"sort"
	"strings"
	"testing"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/legacymatch"
)

// soireesFixture — deux sessions de la composition {Main, Ally} : S1 (3 matchs, la veille) puis
// S2 (3 matchs). Chaque match porte des frags et des morts horodatés, l'équipe alliée complète
// (un allié hors escouade compris) et un vol au premier match de chaque session.
func soireesFixture() (*mockSquadRepo, []domain.SquadMatchRow) {
	t0 := time.Date(2026, 9, 1, 19, 0, 0, 0, time.UTC)
	repo := &mockSquadRepo{topRows: []domain.TopTeammateRow{{XUID: "x_ally", Gamertag: "Ally", GamesTogether: 6}}}
	pct := 5
	for s, label := range []string{"S1", "S2"} {
		for i := 0; i < 3; i++ {
			id := fmt.Sprintf("%s-m%d", label, i)
			sess := label
			start := t0.AddDate(0, 0, s).Add(time.Duration(i) * 20 * time.Minute)
			outcome := domain.OutcomeWin
			if i == 1 {
				outcome = domain.OutcomeLoss
			}
			repo.squadRows = append(repo.squadRows, domain.SquadMatchRow{
				MatchID: id, StartTime: start, SessionLabel: &sess, Outcome: outcome,
				Kills: 10, Deaths: 5, Assists: 3, TimePlayedSecs: 600, IsWithFriends: true, MapUI: "Aquarius",
			})
			repo.synthRows = append(repo.synthRows, legacymatch.SynthesisMatchRow{
				MatchID: id, StartTime: start, Outcome: outcome, Kills: 10, Deaths: 5,
				IsWithFriends: true, SessionLabel: &sess,
			})
			repo.impactRows = append(repo.impactRows,
				domain.ImpactEventRow{MatchID: id, XUID: "x_main", EventType: "kill", TimeMS: 10_000},
				domain.ImpactEventRow{MatchID: id, XUID: "x_ally", EventType: "death", TimeMS: int64(20_000 + i*1000)},
				domain.ImpactEventRow{MatchID: id, XUID: "x_ally", EventType: "kill", TimeMS: 300_000},
				domain.ImpactEventRow{MatchID: id, XUID: "x_main", EventType: "kill", TimeMS: 350_000},
				domain.ImpactEventRow{MatchID: id, XUID: "x_main", EventType: "death", TimeMS: 351_000},
			)
			repo.allyRows = append(repo.allyRows,
				domain.AllyParticipant{MatchID: id, XUID: "x_main", Gamertag: "Main", Kills: 10, Deaths: 5, Assists: 3, Outcome: outcome},
				domain.AllyParticipant{MatchID: id, XUID: "x_ally", Gamertag: "Ally", Kills: 4 + i, Deaths: 7 - i, Assists: 1 + 2*i, Outcome: outcome},
				domain.AllyParticipant{MatchID: id, XUID: "x_ns", Gamertag: "NS", Kills: 2, Deaths: 3, Assists: 0, Outcome: outcome},
			)
			if i == 0 {
				repo.killLog = append(repo.killLog, domain.SquadKillLogRow{
					MatchID: id, TimeMS: 120_000, KillerXUID: "x_ally", VictimXUID: "x_enemy",
					AssistXUID: "x_main", KillerDamagePct: &pct,
				})
			}
		}
	}
	return repo, repo.squadRows
}

func lignesDeSession(rows []domain.SquadMatchRow, label string) []domain.SquadMatchRow {
	var out []domain.SquadMatchRow
	for _, r := range rows {
		if r.SessionLabel != nil && *r.SessionLabel == label {
			out = append(out, r)
		}
	}
	return out
}

// dedieSessions : une lecture dédiée du tableau sur les seules lignes données (sans la mémoire de la
// page ni les autres sessions de la composition).
func dedieSessions(t *testing.T, repo *mockSquadRepo, rows []domain.SquadMatchRow) *domain.SquadImpactSessions {
	t.Helper()
	svc := &TeammatesService{repo: repo, titleSlug: "halo_infinite", gamertag: "Main"}
	return svc.buildSquadImpact(context.Background(), impactEscouade{
		rows: rows, timeline: rows, evenings: soireesDImpact(rows, rows, true),
		mainXUID: "x_main", selected: []string{"Ally"},
		teammates: []domain.TeammateRow{{Gamertag: "Ally", XUID: strPtr("x_ally")}}, allies: repo.allyRows,
	})
}

// rolesDe rend les rôles d'un joueur sur tous les matchs d'une session, à plat, dans l'ordre des
// matchs.
func rolesDe(p domain.SquadImpactSessionsPlayer) []string {
	var out []string
	for _, m := range p.PerMatch {
		out = append(out, m.Roles...)
	}
	return out
}

// joueurDe rend l'entrée d'un joueur d'une session, ou le zéro si elle n'y figure pas.
func joueurDe(s domain.SquadImpactSessionsSession, joueur string) domain.SquadImpactSessionsPlayer {
	for _, p := range s.Players {
		if p.Player == joueur {
			return p
		}
	}
	return domain.SquadImpactSessionsPlayer{}
}

// etiquetees rend les lignes avec un libellé de session (celui donné quand elles n'en ont pas) : une
// ligne sans libellé ne forme aucune ligne du tableau.
func etiquetees(rows []domain.SquadMatchRow, label string) []domain.SquadMatchRow {
	out := slices.Clone(rows)
	for i := range out {
		if out[i].SessionLabel == nil {
			out[i].SessionLabel = strPtr(label)
		}
	}
	return out
}

// impactKillLogCompte compte, en plus des lectures Q32, les lectures du journal des morts.
type impactKillLogCompte struct {
	*countingSquadRepo
	killLogCalls int
	killLogIDs   []string
}

func (c *impactKillLogCompte) LoadSquadKillLog(ctx context.Context, ids, xuids []string) ([]domain.SquadKillLogRow, error) {
	c.mu.Lock()
	c.killLogCalls++
	c.killLogIDs = append([]string(nil), ids...)
	c.mu.Unlock()
	return c.countingSquadRepo.LoadSquadKillLog(ctx, ids, xuids)
}

// TestGetPage_ImpactParSession_UneLectureDesEvenementsEtDuJournal : la page affichée sur S2 publie
// S2 puis S1 ; les événements (Q32) et le journal des morts sont lus UNE fois chacun, sur les six
// matchs ; les autres blocs reçoivent leur part.
func TestGetPage_ImpactParSession_UneLectureDesEvenementsEtDuJournal(t *testing.T) {
	mock, _ := soireesFixture()
	repo := &impactKillLogCompte{countingSquadRepo: &countingSquadRepo{mockSquadRepo: mock}}
	var mainRows, allyRows []canonical.PlayerMatchRow
	for _, r := range mock.squadRows {
		mainRows = append(mainRows, rowWithStatsXUID("x_main", r.MatchID, r.StartTime, canonical.OutcomeWin, 10, 5, 3, 600, 45, 60))
		allyRows = append(allyRows, rowWithStatsXUID("x_ally", r.MatchID, r.StartTime, canonical.OutcomeWin, 4, 7, 1, 600, 40, 50))
	}
	svc := NewTeammatesService(repo, nil).
		WithPlayerMatchesRepo(newSynthMockFromRows(mock.synthRows, nil), "halo_infinite", "Main").
		WithSquadLoader(&fakeSquadLoader{rowsByGT: map[string][]canonical.PlayerMatchRow{"Main": mainRows, "Ally": allyRows}})
	resp, err := svc.GetPage(context.Background(), "x_main", domain.TeammatesQueryRequest{
		SelectedGamertags: []string{"Ally"}, PickedSquadSessions: []string{"S2"},
	})
	if err != nil {
		t.Fatalf("GetPage : %v", err)
	}
	h := resp.SquadImpactSessions
	if h == nil || len(h.Sessions) != 2 || h.Sessions[0].SessionLabel != "S2" || h.Sessions[1].SessionLabel != "S1" {
		t.Fatalf("impact par session : attendu S2 puis S1, obtenu %+v", h)
	}
	if fmt.Sprint(h.Players) != "[Main Ally]" {
		t.Errorf("joueurs %v, attendu [Main Ally] (principal puis sélection)", h.Players)
	}
	for _, s := range h.Sessions {
		if len(s.Matches) != 3 || !strings.HasPrefix(s.Matches[0].MatchID, s.SessionLabel) {
			t.Errorf("session %s : matchs %+v, attendu ses trois matchs", s.SessionLabel, s.Matches)
		}
	}
	if repo.impactCalls != 1 || len(repo.impactMatchs[0]) != 6 {
		t.Errorf("LoadImpactEvents : %d lecture(s) %v, attendu 1 sur les 6 matchs", repo.impactCalls, repo.impactMatchs)
	}
	sort.Strings(repo.killLogIDs)
	if repo.killLogCalls != 1 || len(repo.killLogIDs) != 6 {
		t.Errorf("LoadSquadKillLog : %d lecture(s) %v, attendu 1 sur les 6 matchs", repo.killLogCalls, repo.killLogIDs)
	}
	if resp.IntensityProfile == nil || len(resp.FirstBlood) == 0 {
		t.Errorf("les blocs qui lisent la population reçoivent leur part : intensité %v, premier frag %d",
			resp.IntensityProfile != nil, len(resp.FirstBlood))
	}
}

// TestImpactsDe_PartDUneLectureLarge : un groupe d'une lecture déjà faite en reçoit les lignes
// sans relire ; une partie ou une réunion de groupes est relue (sa décision de repli des frags
// reconstitués peut différer de celle des groupes lus).
func TestImpactsDe_PartDUneLectureLarge(t *testing.T) {
	repo := &countingSquadRepo{mockSquadRepo: &mockSquadRepo{impactRows: []domain.ImpactEventRow{
		{MatchID: "m1", XUID: "a", EventType: "kill", TimeMS: 1},
		{MatchID: "m2", XUID: "b", EventType: "kill", TimeMS: 2},
		{MatchID: "m3", XUID: "c", EventType: "kill", TimeMS: 3},
	}}}
	_, l := (&TeammatesService{repo: repo}).pourLaRequete()
	ctx := context.Background()
	if _, err := l.impactsDe(ctx, [][]string{{"m1", "m2"}, {"m3"}}); err != nil {
		t.Fatal(err)
	}
	part, _ := l.impactsDe(ctx, [][]string{{"m2", "m1"}})
	if repo.impactCalls != 1 || len(part) != 2 || part[0].MatchID != "m1" || part[1].MatchID != "m2" {
		t.Errorf("groupe lu : %d lecture(s), lignes %+v ; attendu 1 lecture, m1 puis m2", repo.impactCalls, part)
	}
	if _, err := l.impactsDe(ctx, [][]string{{"m1"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := l.impactsDe(ctx, [][]string{{"m1", "m2", "m3"}}); err != nil {
		t.Fatal(err)
	}
	if repo.impactCalls != 3 {
		t.Errorf("une partie puis une réunion de groupes doivent relire : %d lecture(s), attendu 3", repo.impactCalls)
	}
}

// fixtureS2SansFragNatif — la composition des sessions S1 / S2, mais les matchs de S2 n'ont dans
// highlight_events que des médailles : leurs frags et morts sont ceux que le dépôt reconstitue
// (impactSynth). S1 garde ses frags natifs.
func fixtureS2SansFragNatif() *mockSquadRepo {
	repo, _ := soireesFixture()
	var natifs, synth []domain.ImpactEventRow
	for _, r := range repo.impactRows {
		if strings.HasPrefix(r.MatchID, "S2") {
			synth = append(synth, r)
			continue
		}
		natifs = append(natifs, r)
	}
	for _, r := range repo.squadRows {
		if strings.HasPrefix(r.MatchID, "S2") {
			natifs = append(natifs, domain.ImpactEventRow{MatchID: r.MatchID, XUID: "x_main", EventType: "medal", TimeMS: 5_000})
		}
	}
	repo.impactRows, repo.impactSynth = natifs, synth
	return repo
}

// TestGetPage_ImpactParSession_FragsReconstituesParGroupe : la population affichée (S2) n'a aucun
// frag natif, la session précédente (S1) en a. Lue avec S1 en une seule lecture, la population
// reçoit quand même ses frags reconstitués : sa session est celle d'une lecture dédiée (Premier
// sang, Finisseur… présents), le premier frag est servi, et S1 égale sa lecture dédiée.
func TestGetPage_ImpactParSession_FragsReconstituesParGroupe(t *testing.T) {
	mock := fixtureS2SansFragNatif()
	repo := &countingSquadRepo{mockSquadRepo: mock}
	var mainRows, allyRows []canonical.PlayerMatchRow
	for _, r := range mock.squadRows {
		mainRows = append(mainRows, rowWithStatsXUID("x_main", r.MatchID, r.StartTime, canonical.OutcomeWin, 10, 5, 3, 600, 45, 60))
		allyRows = append(allyRows, rowWithStatsXUID("x_ally", r.MatchID, r.StartTime, canonical.OutcomeWin, 4, 7, 1, 600, 40, 50))
	}
	svc := NewTeammatesService(repo, nil).
		WithPlayerMatchesRepo(newSynthMockFromRows(mock.synthRows, nil), "halo_infinite", "Main").
		WithSquadLoader(&fakeSquadLoader{rowsByGT: map[string][]canonical.PlayerMatchRow{"Main": mainRows, "Ally": allyRows}})
	resp, err := svc.GetPage(context.Background(), "x_main", domain.TeammatesQueryRequest{
		SelectedGamertags: []string{"Ally"}, PickedSquadSessions: []string{"S2"},
	})
	if err != nil {
		t.Fatalf("GetPage : %v", err)
	}
	if repo.impactCalls != 1 {
		t.Errorf("LoadImpactEvents : %d lectures %v, attendu 1", repo.impactCalls, repo.impactMatchs)
	}

	// Lectures dédiées, sans la mémoire de la page : la population seule, S1 seule.
	s2 := dedieSessions(t, mock, lignesDeSession(mock.squadRows, "S2"))
	s1 := dedieSessions(t, mock, lignesDeSession(mock.squadRows, "S1"))
	if s2 == nil || s1 == nil || len(s2.Sessions) != 1 || len(s1.Sessions) != 1 {
		t.Fatalf("lectures dédiées nulles : S2 %+v, S1 %+v", s2, s1)
	}
	if !slices.Contains(rolesDe(joueurDe(s2.Sessions[0], "Main")), "first_blood") {
		t.Fatal("la lecture dédiée de S2 doit porter Premier sang (frags reconstitués)")
	}
	h := resp.SquadImpactSessions
	if h == nil || len(h.Sessions) != 2 {
		t.Fatalf("impact par session : attendu S2 puis S1, obtenu %+v", h)
	}
	if !reflect.DeepEqual(h.Sessions[0], s2.Sessions[0]) {
		t.Errorf("S2 de la page ≠ S2 d'une lecture dédiée :\n page   %+v\n dédiée %+v", h.Sessions[0], s2.Sessions[0])
	}
	if !reflect.DeepEqual(h.Sessions[1], s1.Sessions[0]) {
		t.Errorf("S1 de la page ≠ S1 d'une lecture dédiée :\n page   %+v\n dédiée %+v", h.Sessions[1], s1.Sessions[0])
	}
	if len(resp.FirstBlood) == 0 {
		t.Error("premier frag / première mort : la population doit recevoir ses frags reconstitués")
	}
}

// TestBuildSquadImpact_SessionsAbsentesJournalise : une escouade sélectionnée dont aucun rôle ne
// tombe (ni événement ni équipe alliée) rend un bloc nul ET le journalise avec sa raison ; sans
// session, la raison est « no_session ».
func TestBuildSquadImpact_SessionsAbsentesJournalise(t *testing.T) {
	repo, all := soireesFixture()
	repo.impactRows, repo.allyRows = nil, nil
	svc := &TeammatesService{repo: repo, titleSlug: "halo_infinite", gamertag: "Main"}
	tm := []domain.TeammateRow{{Gamertag: "Ally", XUID: strPtr("x_ally")}}
	cas := []struct {
		nom, raison string
		sansSession bool
	}{
		{"aucun rôle attribué", "no_role_attributed", false},
		{"aucune session", "no_session", true},
	}
	for _, c := range cas {
		t.Run(c.nom, func(t *testing.T) {
			evenings := soireesDImpact(all, all, true)
			if c.sansSession {
				evenings = nil
			}
			var sessions *domain.SquadImpactSessions
			logs := withCapturedLogs(t, func() {
				sessions = svc.buildSquadImpact(context.Background(), impactEscouade{
					rows: all, timeline: all, evenings: evenings,
					mainXUID: "x_main", selected: []string{"Ally"}, teammates: tm,
				})
			})
			if sessions != nil {
				t.Fatalf("bloc nul attendu, obtenu %+v", sessions)
			}
			if !strings.Contains(logs, "teammates_squad_impact_sessions_absent") || !strings.Contains(logs, `"reason":"`+c.raison+`"`) {
				t.Errorf("journal attendu avec la raison %q, obtenu :\n%s", c.raison, logs)
			}
		})
	}
}
