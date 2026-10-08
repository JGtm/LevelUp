package teammates

// teammates_squad_impact_test.go — la matrice d'impact et les points d'impact par soirée
// (teammates_squad_impact.go) : mêmes rôles, même barème, une lecture des événements et du
// journal des morts pour les deux.

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

// soireesFixture — deux soirées de la composition {Main, Ally} : S1 (3 matchs, la veille) puis
// S2 (3 matchs). Chaque match porte des frags et des morts horodatés, l'équipe alliée complète
// (un allié hors escouade compris) et un vol au premier match de chaque soirée.
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

// TestBuildSquadImpact_PariteMatriceHistorique : sur une même soirée, le net de chaque joueur
// dans les points par soirée égale son score dans la matrice calculée sur les matchs de cette
// soirée, et ses comptes de rôles sont les mêmes. Vérifié pour la soirée affichée ET pour une
// soirée précédente (lue depuis l'historique de la composition).
func TestBuildSquadImpact_PariteMatriceHistorique(t *testing.T) {
	repo, all := soireesFixture()
	svc := &TeammatesService{repo: repo, titleSlug: "halo_infinite", gamertag: "Main"}
	tm := []domain.TeammateRow{{Gamertag: "Ally", XUID: strPtr("x_ally")}}
	s2 := lignesDeSession(all, "S2")
	in := impactEscouade{
		rows: s2, timeline: all, evenings: soireesDImpact(s2, all, true),
		mainXUID: "x_main", selected: []string{"Ally"}, teammates: tm, allies: repo.allyRows,
	}
	_, history := svc.buildSquadImpact(context.Background(), in)
	if history == nil || len(history.Evenings) != 2 {
		t.Fatalf("deux soirées attendues (S1 précédente, S2 affichée), obtenu %+v", history)
	}
	for i, label := range []string{"S1", "S2"} {
		ev := history.Evenings[i]
		if ev.SessionLabel != label || ev.Matches != 3 || ev.Wins != 2 {
			t.Fatalf("soirée %d = %s (%d matchs, %d victoires), attendu %s, 3 matchs, 2 victoires",
				i, ev.SessionLabel, ev.Matches, ev.Wins, label)
		}
		matrix, _ := svc.buildSquadImpact(context.Background(), impactEscouade{
			rows: lignesDeSession(all, label), mainXUID: "x_main", selected: []string{"Ally"},
			teammates: tm, allies: repo.allyRows,
		})
		if matrix == nil {
			t.Fatalf("%s : matrice nulle", label)
		}
		comparerSoireeEtMatrice(t, label, ev, matrix)
	}
}

// comparerSoireeEtMatrice : net = score, comptes de rôles identiques, joueur par joueur.
func comparerSoireeEtMatrice(t *testing.T, label string, ev domain.SquadImpactEvening, m *domain.SquadImpactMatrix) {
	t.Helper()
	parJoueur := map[string]domain.SquadImpactPlayerSummary{}
	for _, p := range m.Players {
		parJoueur[p.Player] = p
	}
	vol := false
	for _, p := range ev.Players {
		ligne, ok := parJoueur[p.Player]
		if !ok {
			t.Fatalf("%s : %s absent de la matrice", label, p.Player)
		}
		if ligne.Score != p.Points {
			t.Errorf("%s / %s : net %v, score de la matrice %v", label, p.Player, p.Points, ligne.Score)
		}
		comptes := map[string]int{}
		for _, r := range p.Roles {
			comptes[r.Role] = r.Count
			vol = vol || r.Role == "thief"
		}
		for _, c := range ligne.Counts {
			if comptes[c.BadgeKey] != c.Count {
				t.Errorf("%s / %s / %s : %d dans la soirée, %d dans la matrice",
					label, p.Player, c.BadgeKey, comptes[c.BadgeKey], c.Count)
			}
		}
	}
	if !vol {
		t.Errorf("%s : le Voleur du premier match manque à la soirée", label)
	}
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

// TestGetPage_PointsParSoiree_UneLectureDesEvenementsEtDuJournal : la page affichée sur S2 publie
// S1 et S2 ; les événements (Q32) et le journal des morts sont lus UNE fois chacun, sur les six
// matchs ; la matrice reste sur les trois matchs de S2 et les autres blocs reçoivent leur part.
func TestGetPage_PointsParSoiree_UneLectureDesEvenementsEtDuJournal(t *testing.T) {
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
	h := resp.SquadImpactHistory
	if h == nil || len(h.Evenings) != 2 || h.Evenings[0].SessionLabel != "S1" || h.Evenings[1].SessionLabel != "S2" {
		t.Fatalf("points par soirée : attendu S1 puis S2, obtenu %+v", h)
	}
	if fmt.Sprint(h.Players) != "[Main Ally]" {
		t.Errorf("joueurs %v, attendu [Main Ally] (principal puis sélection)", h.Players)
	}
	if resp.ImpactMatrix == nil || len(resp.ImpactMatrix.Matches) != 3 {
		t.Fatalf("matrice : attendu les 3 matchs de S2, obtenu %+v", resp.ImpactMatrix)
	}
	for _, m := range resp.ImpactMatrix.Matches {
		if m.MatchID[:2] != "S2" {
			t.Errorf("match %s de la matrice hors de la soirée affichée", m.MatchID)
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

// fixtureS2SansFragNatif — la composition des soirées S1 / S2, mais les matchs de S2 n'ont dans
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

// TestGetPage_PointsParSoiree_FragsReconstituesParGroupe : la population affichée (S2) n'a aucun
// frag natif, la soirée précédente (S1) en a. Lue avec S1 en une seule lecture, la population
// reçoit quand même ses frags reconstitués : sa matrice est celle d'une lecture dédiée (Premier
// sang, Finisseur… présents), le premier frag est servi, et le net de S1 égale sa matrice
// dédiée.
func TestGetPage_PointsParSoiree_FragsReconstituesParGroupe(t *testing.T) {
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
	dedie := &TeammatesService{repo: mock, titleSlug: "halo_infinite", gamertag: "Main"}
	tm := []domain.TeammateRow{{Gamertag: "Ally", XUID: strPtr("x_ally")}}
	matrice := func(label string) *domain.SquadImpactMatrix {
		m, _ := dedie.buildSquadImpact(context.Background(), impactEscouade{
			rows: lignesDeSession(mock.squadRows, label), mainXUID: "x_main", selected: []string{"Ally"},
			teammates: tm, allies: mock.allyRows,
		})
		if m == nil {
			t.Fatalf("%s : matrice dédiée nulle", label)
		}
		return m
	}
	s2 := matrice("S2")
	premierSang := false
	for _, c := range s2.Cells {
		premierSang = premierSang || slices.Contains(c.BadgeKeys, "first_blood")
	}
	if !premierSang {
		t.Fatal("la lecture dédiée de S2 doit porter Premier sang (frags reconstitués)")
	}
	if !reflect.DeepEqual(resp.ImpactMatrix, s2) {
		t.Errorf("matrice de la page ≠ matrice d'une lecture dédiée de la population :\n page   %+v\n dédiée %+v", resp.ImpactMatrix, s2)
	}
	if len(resp.FirstBlood) == 0 {
		t.Error("premier frag / première mort : la population doit recevoir ses frags reconstitués")
	}
	h := resp.SquadImpactHistory
	if h == nil || len(h.Evenings) != 2 {
		t.Fatalf("points par soirée : attendu S1 puis S2, obtenu %+v", h)
	}
	comparerSoireeEtMatrice(t, "S1", h.Evenings[0], matrice("S1"))
	comparerSoireeEtMatrice(t, "S2", h.Evenings[1], s2)
}
