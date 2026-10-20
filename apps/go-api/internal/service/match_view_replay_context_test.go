package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"levelup/go-api/internal/analysis/squademprise"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/highlightevent"
	"levelup/go-api/internal/games/canonical"
	"levelup/go-api/internal/port"
)

// countingMatchViewRepo : le repo factice de la vue match, enrichi des lectures que le mock de
// base rend vides (médias, médailles et armes de tous les joueurs, rangs) et qui COMPTE les
// appels. Chaque lecture rend une copie : la vue complète modifie le scoreboard chargé en
// place (attendus de la ligne is_me), une réponse ne doit pas déteindre sur l'autre.
type countingMatchViewRepo struct {
	*mockMatchViewRepo
	mu           sync.Mutex
	calls        map[string]int
	media        []domain.MediaAssocRaw
	bulkMedals   []domain.BulkMedalRaw
	bulkWeapons  []domain.BulkWeaponKillRaw
	skillRank    *domain.SkillRankRaw
	sharedCSRs   map[string]*domain.SkillRankRaw
	assistsModel *domain.PlayerAssistsModel
}

func (c *countingMatchViewRepo) count(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.calls[name]++
}

func (c *countingMatchViewRepo) callsOf(name string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.calls[name]
}

func (c *countingMatchViewRepo) GetMatchScoreboard(_ context.Context, _ string) ([]domain.ScoreboardRaw, error) {
	c.count("GetMatchScoreboard")
	return append([]domain.ScoreboardRaw(nil), c.board...), c.boardErr
}
func (c *countingMatchViewRepo) GetMatchMedia(_ context.Context, _ string) ([]domain.MediaAssocRaw, error) {
	c.count("GetMatchMedia")
	return append([]domain.MediaAssocRaw(nil), c.media...), nil
}
func (c *countingMatchViewRepo) GetMatchBulkMedals(_ context.Context, _ string) ([]domain.BulkMedalRaw, error) {
	c.count("GetMatchBulkMedals")
	return append([]domain.BulkMedalRaw(nil), c.bulkMedals...), nil
}
func (c *countingMatchViewRepo) GetMatchBulkWeaponKills(_ context.Context, _ string) ([]domain.BulkWeaponKillRaw, error) {
	c.count("GetMatchBulkWeaponKills")
	return append([]domain.BulkWeaponKillRaw(nil), c.bulkWeapons...), nil
}
func (c *countingMatchViewRepo) GetMatchSkillRank(_ context.Context, _ string) (*domain.SkillRankRaw, error) {
	c.count("GetMatchSkillRank")
	return c.skillRank, nil
}
func (c *countingMatchViewRepo) GetMatchSharedCSRs(_ context.Context, _ string) (map[string]*domain.SkillRankRaw, error) {
	c.count("GetMatchSharedCSRs")
	return c.sharedCSRs, nil
}
func (c *countingMatchViewRepo) GetPlayerAssistsModel(_ context.Context, _ string) (*domain.PlayerAssistsModel, error) {
	c.count("GetPlayerAssistsModel")
	return c.assistsModel, nil
}
func (c *countingMatchViewRepo) GetMatchMedals(ctx context.Context, x, m string) ([]domain.MedalRaw, error) {
	c.count("GetMatchMedals")
	return c.mockMatchViewRepo.GetMatchMedals(ctx, x, m)
}
func (c *countingMatchViewRepo) GetMatchObjectiveScore(ctx context.Context, x, m string) (int, error) {
	c.count("GetMatchObjectiveScore")
	return c.mockMatchViewRepo.GetMatchObjectiveScore(ctx, x, m)
}
func (c *countingMatchViewRepo) GetMatchAssistPairs(ctx context.Context, m string) ([]domain.MatchAssistPairRaw, domain.MatchAssistScopeRaw, error) {
	c.count("GetMatchAssistPairs")
	return c.mockMatchViewRepo.GetMatchAssistPairs(ctx, m)
}
func (c *countingMatchViewRepo) GetMatchEncounters(_ context.Context, _, _ string) ([]domain.EncounterRaw, error) {
	c.count("GetMatchEncounters")
	return nil, nil
}
func (c *countingMatchViewRepo) GetMatchEncounterStats(_ context.Context, _, _ string) ([]domain.EncounterStatsRaw, error) {
	c.count("GetMatchEncounterStats")
	return nil, nil
}
func (c *countingMatchViewRepo) GetMatchEncounterAssists(_ context.Context, _, _ string) (map[string]domain.RelationAssists, error) {
	c.count("GetMatchEncounterAssists")
	return nil, nil
}
func (c *countingMatchViewRepo) GetHistoryForAvg(_ context.Context, _ string) ([]domain.MatchHistAvgRow, error) {
	c.count("GetHistoryForAvg")
	return nil, nil
}
func (c *countingMatchViewRepo) GetHistoryForAvgBulk(_ context.Context, _ []string) (map[string][]domain.MatchHistAvgRow, error) {
	c.count("GetHistoryForAvgBulk")
	return nil, nil
}
func (c *countingMatchViewRepo) GetMatchExpectedStats(_ context.Context, _, _ string) (*domain.ExpectedStatsRaw, error) {
	c.count("GetMatchExpectedStats")
	return nil, nil
}

// countingCitations compte les deux lectures de citations de la vue match.
type countingCitations struct {
	port.CitationsRepository
	calls atomic.Int32
}

func (c *countingCitations) LoadMatchCitationsForView(_ context.Context, _ string) ([]domain.CitationMatchViewRow, error) {
	c.calls.Add(1)
	return nil, nil
}
func (c *countingCitations) LoadMatchCitationsRich(_ context.Context, _ string) ([]domain.HomeMatchCitationRaw, error) {
	c.calls.Add(1)
	return nil, nil
}

// countingEvents compte les lectures d'events canoniques (cadence, rôles).
type countingEvents struct{ calls atomic.Int32 }

func (c *countingEvents) Load(_ context.Context, _ port.HighlightEventFilters) ([]canonical.HighlightEvent, error) {
	c.calls.Add(1)
	return nil, nil
}

// countingEmprise compte les lectures de la feuille de match de l'Emprise.
type countingEmprise struct{ calls atomic.Int32 }

func (c *countingEmprise) LoadPowerWeaponKills(_ context.Context, _ []string) ([]squademprise.PowerKillRow, error) {
	c.calls.Add(1)
	return nil, nil
}
func (c *countingEmprise) LoadJournalWeaponKills(_ context.Context, _ []string) (squademprise.JournalRead, error) {
	c.calls.Add(1)
	return squademprise.JournalRead{}, nil
}

// countingLives compte la lecture des vies de l'équipe (Isolement).
type countingLives struct{ calls atomic.Int32 }

func (c *countingLives) LoadLivesNearTeammateForPlayers(_ context.Context, _, _ []string) (map[string]domain.ViesLues, error) {
	c.calls.Add(1)
	return nil, nil
}

// replayFixture : un match riche, sur l'horloge du FILM (T0 = 15 s) — kills avec arme et
// assistant, médaille, kill d'un bot, event de countdown, joueur entré et sorti en cours de
// match, bot, médias, rangs, médailles et armes de chaque joueur.
func replayFixture() *countingMatchViewRepo {
	start := time.Date(2026, 10, 1, 20, 0, 0, 0, time.UTC)
	joined, left := start.Add(90*time.Second), start.Add(400*time.Second)
	gv, pair, mapName, pl := "Slayer", "Slayer:Live Fire", "Live Fire", "Ranked Arena"
	dur, playable, t0 := 620.0, int64(600), int64(15_000)
	me, friend, adv, bot := "me", "fr", "adv", "bid(1.0)"
	ms := func(v int64) *int64 { return &v }
	str := func(v string) *string { return &v }
	f64 := func(v float64) *float64 { return &v }
	tru := true
	return &countingMatchViewRepo{
		calls: map[string]int{},
		mockMatchViewRepo: &mockMatchViewRepo{
			meta: &domain.MatchMetaRaw{
				MatchID: "m1", StartTime: &start, DurationSeconds: &dur, PlayableDurationSeconds: &playable,
				T0Ms: &t0, GameVariantName: &gv, PairName: &pair, MapName: &mapName, PlaylistName: &pl, IsRanked: true,
			},
			stats: &domain.PlayerMatchStatsRaw{OutcomeCode: 2, TeamID: ptrIk(0), Kills: 2, Deaths: 1, Assists: 1, DamageDealt: f64(900)},
			board: []domain.ScoreboardRaw{
				{XUID: me, Gamertag: "Moi", TeamID: ptrIk(0), OutcomeCode: 2, Kills: 2, Deaths: 1, Assists: 1, PersonalScore: f64(1200)},
				{XUID: friend, Gamertag: "Ami", TeamID: ptrIk(0), OutcomeCode: 2, Kills: 1, Deaths: 1, Assists: 2},
				{XUID: adv, Gamertag: "Adverse", TeamID: ptrIk(1), OutcomeCode: 3, Kills: 1, Deaths: 2,
					JoinedInProgress: &tru, LeftInProgress: &tru, FirstJoinedTime: &joined, LastLeaveTime: &left},
				{XUID: bot, Gamertag: "343 Bot", IsBot: true, TeamID: ptrIk(1), OutcomeCode: 3, Kills: 1, Deaths: 1},
			},
			events: []domain.EventRaw{
				{EventType: highlightevent.EventTypeKill, TimeMS: ms(10_000), XUID: str(adv)}, // countdown : écarté
				{EventType: highlightevent.EventTypeKill, TimeMS: ms(20_000), XUID: str(me), Gamertag: str("Moi")},
				{EventType: highlightevent.EventTypeMedal, TimeMS: ms(20_000), XUID: str(me), MedalName: str("Double Kill")},
				{EventType: highlightevent.EventTypeKill, TimeMS: ms(30_000), XUID: str(adv), Gamertag: str("Adverse")},
				{EventType: highlightevent.EventTypeKill, TimeMS: ms(40_000), XUID: str(bot), Gamertag: str("343 Bot")},
			},
			kvPairs: []domain.KVPairRaw{
				{MatchID: "m1", KillerXUID: me, KillerGT: "Moi", VictimXUID: adv, VictimGT: "Adverse", KillCount: 1, TimeMS: 20_000},
				{MatchID: "m1", KillerXUID: adv, KillerGT: "Adverse", VictimXUID: me, VictimGT: "Moi", KillCount: 1, TimeMS: 30_000},
				{MatchID: "m1", KillerXUID: "", KillerGT: "343 Bot", VictimXUID: friend, VictimGT: "Ami", KillCount: 1, TimeMS: 40_000},
			},
			killSources: []domain.KillSourceRaw{{XUID: me, TimeMS: 20_000, SourceTag: 0x11, Headshot: true}},
			killAssists: []domain.KillAssistRaw{{XUID: me, TimeMS: 20_000, AssistXUID: str(friend), AssistGamertag: str("Ami")}},
			medalMetasByName: map[string]domain.MedalNameMeta{
				"Double Kill": {MedalNameID: 7, Label: "Double frag", Description: "Deux frags rapprochés"},
			},
		},
		media: []domain.MediaAssocRaw{{FileID: "f1", FileName: "clip.mp4", FilePath: "captures/clip.mp4", Kind: "video",
			ThumbnailPath: str("captures/clip.webp"), CaptureStartTime: str("2026-10-01T20:02:00Z"), OwnerSlug: "ami"}},
		bulkMedals:  []domain.BulkMedalRaw{{XUID: me, MedalID: 7, Count: 1, Label: "Double frag"}, {XUID: adv, MedalID: 9, Count: 2}},
		bulkWeapons: []domain.BulkWeaponKillRaw{{XUID: me, WeaponID: 3, Kills: 2, WeaponLabel: "BR75"}},
		skillRank:   &domain.SkillRankRaw{RatingType: "csr", TierLabel: str("Diamant 2"), RatingValue: f64(1320)},
		sharedCSRs:  map[string]*domain.SkillRankRaw{adv: {RatingType: "csr", RatingValue: f64(1280)}},
	}
}

// replayFixtureService câble le service sur le repo factice, l'adapter d'assets (icône d'arme,
// visuel de médaille) et les compteurs des lectures que le contexte du rejeu ne doit pas faire.
func replayFixtureService(repo *countingMatchViewRepo) (*MatchViewService, *replayCounters) {
	c := &replayCounters{}
	assets := &stubAssetURL{medalImg: "/static/medal.png", killIcons: map[uint32]canonical.KillSourceIcon{
		0x11: {WeaponKey: "hinf_br75", Label: "BR75", ImageURL: "/static/killfeed-11.png", Tinted: true},
	}}
	svc := NewMatchViewService(repo, "me").WithTitleSlug("halo_infinite").WithAssetURL(assets).
		WithCitationsRepo(&c.citations).WithHighlightEventsRepo(&c.events).WithCampLives(&c.lives).
		WithEmpriseSheet(&c.emprise).
		WithFriendsExtras(func(_ context.Context, _, _ string, _ []string) (map[string]port.FriendMatchExtras, error) {
			c.friends.Add(1)
			return nil, nil
		})
	return svc, c
}

// replayCounters : les compteurs des lectures hors repo (citations, events canoniques, vies de
// l'équipe, feuille de l'Emprise, extras des amis).
type replayCounters struct {
	citations countingCitations
	events    countingEvents
	lives     countingLives
	emprise   countingEmprise
	friends   atomic.Int32
}

func (c *replayCounters) loads() map[string]int32 {
	return map[string]int32{"citations": c.citations.calls.Load(), "events canoniques": c.events.calls.Load(),
		"vies (Isolement)": c.lives.calls.Load(), "Emprise": c.emprise.calls.Load(), "amis": c.friends.Load()}
}

// sansChampsDesChargementsOmis retire des lignes les champs que seuls les chargements omis par
// le contexte du rejeu alimentent (liste fermée D4 : attendus, extras des amis suivis).
func sansChampsDesChargementsOmis(rows []domain.MatchScoreboardRow) []domain.MatchScoreboardRow {
	out := append([]domain.MatchScoreboardRow(nil), rows...)
	for i := range out {
		out[i].PerformanceScore, out[i].HadBotTeammate, out[i].SkillRank = nil, false, nil
		out[i].ExpectedAssists, out[i].ExpectedKills, out[i].ExpectedDeaths = nil, nil, nil
		out[i].LocallyEstimated = false
	}
	return out
}

func getBoth(t *testing.T, svc *MatchViewService) (domain.MatchReplayContext, domain.MatchViewResponse) {
	t.Helper()
	ctxResp, err := svc.GetReplayContext(context.Background(), "m1")
	if err != nil {
		t.Fatalf("GetReplayContext : %v", err)
	}
	full, err := svc.GetMatchView(context.Background(), "m1")
	if err != nil {
		t.Fatalf("GetMatchView : %v", err)
	}
	return ctxResp, full
}

func assertSamePiece(t *testing.T, name string, got, want any) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("%s : le contexte du rejeu diffère de la vue complète\n contexte : %+v\n vue      : %+v", name, got, want)
	}
}

// assertFixtureExercised : la fixture exerce bien ce qu'elle prétend (sans quoi une parité
// « vide = vide » passerait pour une preuve).
func assertFixtureExercised(t *testing.T, got domain.MatchReplayContext, killsFromKV bool) {
	t.Helper()
	if got.Header.T0Ms != 15_000 || got.Header.MatchID != "m1" {
		t.Errorf("en-tête : T0 = %d, match = %q", got.Header.T0Ms, got.Header.MatchID)
	}
	if len(got.Scoreboard) != 4 || got.Scoreboard[2].JoinedInProgress == nil || got.Scoreboard[2].LastLeaveTime == nil {
		t.Fatalf("lignes : %d, entrée/sortie en cours absente : %+v", len(got.Scoreboard), got.Scoreboard)
	}
	if !got.Scoreboard[3].IsBot || len(got.Scoreboard[0].Medals) == 0 || got.Scoreboard[0].SkillRank == nil {
		t.Errorf("lignes : bot / médailles / rang non exercés : %+v", got.Scoreboard)
	}
	if len(got.Media.MediaItems) != 1 {
		t.Errorf("médias : %d éléments", len(got.Media.MediaItems))
	}
	var withWeapon, withAssist, withMedal, shifted, kills bool
	for _, e := range got.HighlightEvents {
		if e.EventTimeMS != nil && *e.EventTimeMS < 0 {
			t.Errorf("event de countdown servi : %+v", e)
		}
		withWeapon = withWeapon || e.WeaponKey == "hinf_br75"
		withAssist = withAssist || e.AssistGamertag == "Ami"
		withMedal = withMedal || e.MedalLabel == "Double frag"
		shifted = shifted || (e.EventTimeMS != nil && *e.EventTimeMS == 5_000 && e.VictimXUID != nil)
		kills = kills || e.EventType == highlightevent.EventTypeKill
	}
	if !withMedal || !kills {
		t.Errorf("fil : médaille %v, kills %v", withMedal, kills)
	}
	// Les kills synthétisés depuis les paires n'ont ni arme ni assistant (aucune source appariée).
	if !killsFromKV && (!withWeapon || !withAssist || !shifted) {
		t.Errorf("fil : arme %v, assistant %v, kill recalé T0 avec victime %v — décorations non exercées",
			withWeapon, withAssist, shifted)
	}
}

// TestGetReplayContext_PariteAvecLaVueComplete : sur un même match, chacun des quatre morceaux
// du contexte du rejeu est ÉGAL (comparaison profonde) à son homologue de la vue complète —
// kills portés par les events, puis events sans kill (synthèse depuis les paires killer→victim).
func TestGetReplayContext_PariteAvecLaVueComplete(t *testing.T) {
	cases := map[string]struct {
		mutate      func(*countingMatchViewRepo)
		killsFromKV bool
	}{
		"kills portés par les events": {mutate: func(*countingMatchViewRepo) {}},
		"events sans kill : synthèse depuis les paires": {killsFromKV: true, mutate: func(r *countingMatchViewRepo) {
			r.events = []domain.EventRaw{r.events[2]} // la médaille seule
		}},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			repo := replayFixture()
			tc.mutate(repo)
			svc, _ := replayFixtureService(repo)
			got, full := getBoth(t, svc)
			assertSamePiece(t, "en-tête", got.Header, full.Header)
			assertSamePiece(t, "lignes du tableau de score", got.Scoreboard, full.TeamTab.Scoreboard)
			assertSamePiece(t, "événements", got.HighlightEvents, full.CombatTab.HighlightEvents)
			assertSamePiece(t, "médias", got.Media, full.MediaTab)
			assertFixtureExercised(t, got, tc.killsFromKV)
		})
	}
}

// TestGetReplayContext_ChampsDesChargementsOmis : avec un ami suivi et un modèle d'attendus, la
// vue complète pose sur les lignes des champs que le contexte du rejeu ne charge pas. Les deux
// diffèrent EXACTEMENT sur la liste fermée D4 : égales une fois ces champs retirés, et
// différentes avant (sinon la fixture n'exercerait pas l'écart).
func TestGetReplayContext_ChampsDesChargementsOmis(t *testing.T) {
	repo := replayFixture()
	repo.assistsModel = &domain.PlayerAssistsModel{GameVariantName: "Slayer", Intercept: 1, CoefKills: 0.5}
	perf, rating := 71.0, 1400.0
	svc, _ := replayFixtureService(repo)
	svc.WithFriendsExtras(func(_ context.Context, _, _ string, _ []string) (map[string]port.FriendMatchExtras, error) {
		return map[string]port.FriendMatchExtras{"fr": {
			PerformanceScore: &perf, HadBotTeammate: true,
			SkillRank:    &domain.MatchScoreboardSkillRank{RatingType: "csr", RatingValue: &rating},
			AssistsModel: &domain.PlayerAssistsModel{Intercept: 2},
		}}, nil
	})
	got, full := getBoth(t, svc)
	if reflect.DeepEqual(got.Scoreboard, full.TeamTab.Scoreboard) {
		t.Fatal("les lignes sont égales : la fixture n'exerce ni les attendus ni les extras des amis")
	}
	assertSamePiece(t, "lignes hors champs omis", sansChampsDesChargementsOmis(got.Scoreboard),
		sansChampsDesChargementsOmis(full.TeamTab.Scoreboard))
	assertSamePiece(t, "en-tête", got.Header, full.Header)
	assertSamePiece(t, "événements", got.HighlightEvents, full.CombatTab.HighlightEvents)
	assertSamePiece(t, "médias", got.Media, full.MediaTab)
}

// TestGetReplayContext_AucunChargementLourd : le contexte du rejeu ne lit ni les rencontres, ni
// l'historique, ni les attendus, ni les citations, ni les events canoniques, ni les vies de
// l'équipe, ni les extras des amis. Chaque compteur est d'abord prouvé par la vue complète
// (sans quoi un zéro ne prouverait rien).
func TestGetReplayContext_AucunChargementLourd(t *testing.T) {
	heavy := []string{"GetMatchEncounters", "GetMatchEncounterStats", "GetMatchEncounterAssists",
		"GetHistoryForAvg", "GetMatchExpectedStats", "GetPlayerAssistsModel", "GetMatchAssistPairs",
		"GetMatchObjectiveScore", "GetMatchMedals"}
	fullRepo := replayFixture()
	fullSvc, fullCounters := replayFixtureService(fullRepo)
	if _, err := fullSvc.GetMatchView(context.Background(), "m1"); err != nil {
		t.Fatalf("GetMatchView : %v", err)
	}
	for _, name := range heavy {
		if fullRepo.callsOf(name) == 0 {
			t.Fatalf("compteur %s jamais incrémenté par la vue complète : il ne prouverait rien", name)
		}
	}
	for name, n := range fullCounters.loads() {
		if n == 0 {
			t.Fatalf("compteur %q jamais incrémenté par la vue complète : il ne prouverait rien", name)
		}
	}

	repo := replayFixture()
	svc, counters := replayFixtureService(repo)
	if _, err := svc.GetReplayContext(context.Background(), "m1"); err != nil {
		t.Fatalf("GetReplayContext : %v", err)
	}
	for _, name := range heavy {
		if n := repo.callsOf(name); n != 0 {
			t.Errorf("le contexte du rejeu a appelé %s (%d fois)", name, n)
		}
	}
	for name, n := range counters.loads() {
		if n != 0 {
			t.Errorf("le contexte du rejeu a lu : %s (%d fois)", name, n)
		}
	}
	for _, name := range []string{"GetMatchScoreboard", "GetMatchMedia", "GetMatchBulkMedals", "GetMatchSkillRank"} {
		if repo.callsOf(name) != 1 {
			t.Errorf("%s : %d appels, attendu 1", name, repo.callsOf(name))
		}
	}
}

// TestGetReplayContext_ErreursCommeLaVueComplete : match absent → domain.ErrNotFound, joueur
// non participant → match_not_participant (avant tout chargement), panne de métadonnées → erreur.
func TestGetReplayContext_ErreursCommeLaVueComplete(t *testing.T) {
	absent := replayFixture()
	absent.meta, absent.metaErr = nil, fmt.Errorf("lecture : %w", sql.ErrNoRows)
	svc, _ := replayFixtureService(absent)
	var apiErr *domain.APIError
	if _, err := svc.GetReplayContext(context.Background(), "m1"); !errors.As(err, &apiErr) || apiErr.Code != "not_found" {
		t.Errorf("match absent : err = %v, attendu not_found", err)
	}

	outsider := replayFixture()
	outsider.notParticipant = true
	svc, _ = replayFixtureService(outsider)
	if _, err := svc.GetReplayContext(context.Background(), "m1"); !errors.As(err, &apiErr) || apiErr.Code != "match_not_participant" {
		t.Errorf("non participant : err = %v, attendu match_not_participant", err)
	}
	if outsider.callsOf("GetMatchScoreboard") != 0 {
		t.Error("non participant : le tableau de score a été chargé avant la garde")
	}

	broken := replayFixture()
	broken.meta, broken.metaErr = nil, errors.New("verrou")
	svc, _ = replayFixtureService(broken)
	if _, err := svc.GetReplayContext(context.Background(), "m1"); err == nil || errors.As(err, &apiErr) {
		t.Errorf("panne de métadonnées : err = %v, attendu une erreur technique", err)
	}
}

// TestGetReplayContext_SourceEnPanneRendLeContexteTransitoire : une lecture en échec ne casse
// pas la réponse, elle la marque transitoire (le handler en tire Cache-Control: no-store).
func TestGetReplayContext_SourceEnPanneRendLeContexteTransitoire(t *testing.T) {
	repo := replayFixture()
	svc, _ := replayFixtureService(repo)
	ok, err := svc.GetReplayContext(context.Background(), "m1")
	if err != nil || ok.SourceUnavailable {
		t.Fatalf("contexte nominal : err = %v, transitoire = %v", err, ok.SourceUnavailable)
	}
	repo.killSrcErr = errors.New("verrou")
	ko, err := svc.GetReplayContext(context.Background(), "m1")
	if err != nil || !ko.SourceUnavailable {
		t.Errorf("source en panne : err = %v, transitoire = %v", err, ko.SourceUnavailable)
	}
}
