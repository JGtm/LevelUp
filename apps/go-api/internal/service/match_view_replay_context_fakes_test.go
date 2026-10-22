package service

// Les doublures du test de parité du contexte du rejeu (match_view_replay_context_test.go) :
// repo qui compte ses appels, compteurs des lectures hors repo, et dépendances de l'en-tête
// (rejeu disponible, favori) qui donnent à chaque `apply*` un effet visible.

import (
	"context"
	"sync"
	"sync/atomic"

	"levelup/go-api/internal/analysis/squademprise"
	"levelup/go-api/internal/domain"
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

// countingKillDistances compte la lecture des distances de kill (bloc de l'onglet Combat).
type countingKillDistances struct{ calls atomic.Int32 }

func (c *countingKillDistances) LoadMatch(_ context.Context, _ string) ([]domain.MatchKillDistancePlayer, error) {
	c.calls.Add(1)
	return nil, nil
}

// replayDisponible : le rejeu 2D existe pour tout match (`replay_available` vrai).
type replayDisponible struct{ stubReplayService }

func (*replayDisponible) IsAvailable(context.Context, string) bool { return true }

// socialFavori : tout match est en favori pour le joueur de la page.
type socialFavori struct{ port.SocialRepository }

func (socialFavori) IsMatchFavorite(context.Context, string, string) (bool, error) { return true, nil }
