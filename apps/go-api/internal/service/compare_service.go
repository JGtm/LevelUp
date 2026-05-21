// Package service — compare_service.go : comparaison joueur vs joueur.
//
// Sprint 54 C : CompareService avec chargement parallèle via errgroup.
// Joueur A : données DuckDB (CompareRepository) + CSR depuis Waypoint skill endpoint.
// Joueur B : Waypoint (PlayerStatsProvider) + fallback DuckDB si connu localement.
package service

import (
	"context"
	"fmt"
	"log/slog"
	"math"

	"levelup/go-api/internal/analysis/narrative"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/port"

	"golang.org/x/sync/errgroup"
)

// CompareService orchestre la comparaison joueur A vs joueur B.
type CompareService struct {
	repo      port.CompareRepository
	provider  port.PlayerStatsProvider
	xuidA     string
	titleSlug string
}

// NewCompareService crée un CompareService.
func NewCompareService(repo port.CompareRepository, provider port.PlayerStatsProvider, xuidA, titleSlug string) *CompareService {
	return &CompareService{repo: repo, provider: provider, xuidA: xuidA, titleSlug: titleSlug}
}

// GetPage construit la réponse de comparaison en chargeant les stats en parallèle.
func (s *CompareService) GetPage(ctx context.Context, req domain.CompareRequest) (domain.CompareResponse, error) {
	if err := req.Validate(); err != nil {
		return domain.CompareResponse{}, fmt.Errorf("CompareService: %w", err)
	}

	var statsA, statsB *domain.NormalizedPlayerStats
	var ath *domain.PlayerATH
	var xuidBResolved string
	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error
		statsA, ath, err = s.loadPlayerA(gctx)
		return err
	})

	g.Go(func() error {
		var err error
		statsB, xuidBResolved, err = s.loadPlayerB(gctx, req.TargetGamertag)
		return err
	})

	if err := g.Wait(); err != nil {
		return domain.CompareResponse{}, err
	}

	// Merge ATH dans statsA.
	if ath != nil {
		statsA.CareerRank = ath.CareerRank
		statsA.PerfATH = ath.PerfATH
		statsA.LusrATH = ath.LusrATH
	}

	metrics := buildMetrics(*statsA, *statsB)
	resp := domain.CompareResponse{
		PlayerA:   *statsA,
		PlayerB:   *statsB,
		Metrics:   metrics,
		TitleSlug: s.titleSlug,
	}

	s.attachEncounterBadges(ctx, &resp, xuidBResolved, statsB.Gamertag)
	return resp, nil
}

// loadPlayerA charge stats du joueur courant + ATH + arme favorite. ATH/arme
// sont best-effort (loggées en debug si absentes).
func (s *CompareService) loadPlayerA(ctx context.Context) (*domain.NormalizedPlayerStats, *domain.PlayerATH, error) {
	statsA, err := s.repo.GetLocalStats(ctx, s.xuidA, s.titleSlug)
	if err != nil {
		return nil, nil, err
	}
	statsA.IsLocal = true
	ath, athErr := s.repo.GetPlayerATH(ctx)
	if athErr != nil {
		slog.DebugContext(ctx, "CompareService: ATH non disponible (best-effort)", "xuid", s.xuidA, "err", athErr)
	}
	var wErr error
	statsA.FavoriteWeapon, wErr = s.repo.GetFavoriteWeapon(ctx, s.xuidA)
	if wErr != nil {
		slog.DebugContext(ctx, "CompareService: arme favorite non disponible (best-effort)", "xuid", s.xuidA, "err", wErr)
	}
	return statsA, ath, nil
}

// loadPlayerB tente d'abord la résolution locale (xuid → stats locales), sinon
// fallback Waypoint. Renvoie aussi le xuid résolu (vide si pas de match local).
func (s *CompareService) loadPlayerB(ctx context.Context, targetGamertag string) (*domain.NormalizedPlayerStats, string, error) {
	xuidB, _ := s.repo.ResolveXUID(ctx, targetGamertag)
	if xuidB != "" {
		if local, err := s.repo.GetLocalStats(ctx, xuidB, s.titleSlug); err == nil && local != nil {
			s.enrichLocalPlayerB(ctx, local, xuidB)
			slog.DebugContext(ctx, "CompareService: joueur B résolu localement", "gamertag", targetGamertag, "xuid", xuidB)
			return local, xuidB, nil
		}
	}
	// Fallback : Waypoint.
	slog.DebugContext(ctx, "CompareService: joueur B non local, fallback Waypoint", "gamertag", targetGamertag)
	remote, err := s.provider.FetchRemoteStats(ctx, targetGamertag, s.titleSlug)
	if err != nil {
		return nil, xuidB, fmt.Errorf("CompareService.GetPage: stats joueur B introuvables: %w", err)
	}
	if xuidB != "" {
		s.enrichRemotePlayerBWithCrossSample(ctx, remote, xuidB, targetGamertag)
	}
	return remote, xuidB, nil
}

// enrichLocalPlayerB enrichit un joueur B local avec FavoriteWeapon + ATH propre.
func (s *CompareService) enrichLocalPlayerB(ctx context.Context, local *domain.NormalizedPlayerStats, xuidB string) {
	local.IsLocal = true
	local.FavoriteWeapon, _ = s.repo.GetFavoriteWeapon(ctx, xuidB)

	athB, athErr := s.repo.GetPlayerATHFor(ctx, local.Gamertag, s.titleSlug)
	if athErr != nil {
		slog.DebugContext(ctx, "CompareService: ATH joueur B non disponible (best-effort)", "gamertag", local.Gamertag, "err", athErr)
		return
	}
	if athB != nil {
		local.CareerRank = athB.CareerRank
		local.PerfATH = athB.PerfATH
		local.LusrATH = athB.LusrATH
	}
}

// enrichRemotePlayerBWithCrossSample calcule les 4 métriques locale-only sur
// l'échantillon croisé (matchs en commun avec le joueur A).
func (s *CompareService) enrichRemotePlayerBWithCrossSample(
	ctx context.Context, remote *domain.NormalizedPlayerStats, xuidB, targetGamertag string,
) {
	sample, sErr := s.repo.GetCrossMatchSample(ctx, s.xuidA, xuidB)
	if sErr != nil {
		slog.DebugContext(ctx, "CompareService: cross-match sample non disponible (best-effort)", "err", sErr)
		return
	}
	if sample == nil || sample.MatchesCount == 0 {
		return
	}
	remote.IsLocalSample = true
	remote.MaxKillingSpree = sample.MaxKillingSpree
	remote.AvgLifeSecs = sample.AvgLifeSecs
	remote.PerfectKillsPerGame = sample.PerfectKillsPerGame
	remote.HeadshotKillsPerGame = sample.HeadshotKillsPerGame
	remote.Matches = sample.MatchesCount
	slog.DebugContext(ctx, "CompareService: stats B enrichies par échantillon croisé",
		"gamertag_b", targetGamertag, "matches", sample.MatchesCount)
}

// attachEncounterBadges calcule les badges historiques de rencontre. Best-effort.
func (s *CompareService) attachEncounterBadges(
	ctx context.Context, resp *domain.CompareResponse, xuidB, gamertagB string,
) {
	if xuidB == "" {
		return
	}
	enc, err := s.repo.GetEncounterStats(ctx, s.xuidA, xuidB)
	if err != nil || enc == nil || enc.TotalEncounters == 0 {
		return
	}
	stats := narrative.EncounterStats{
		XUID:            xuidB,
		Gamertag:        gamertagB,
		TotalEncounters: enc.TotalEncounters,
		AllyCount:       enc.AllyCount,
		EnemyCount:      enc.EnemyCount,
		WinrateAsAlly:   enc.WinrateAsAlly,
		WinrateVsEnemy:  enc.WinrateVsEnemy,
		KillsDealt:      enc.KillsDealt,
		DeathsSuffered:  enc.DeathsSuffered,
	}
	resp.EncounterBadges = convertNarrativeBadgesCompare(narrative.ComputeEncounterBadges(stats, enc.TotalEncounters))
	slog.DebugContext(ctx, "CompareService: encounter badges calculés",
		"gamertag_b", gamertagB, "total", enc.TotalEncounters, "badges", len(resp.EncounterBadges))
}

func convertNarrativeBadgesCompare(badges []narrative.EncounterBadge) []domain.MatchEncounterBadge {
	if len(badges) == 0 {
		return nil
	}
	result := make([]domain.MatchEncounterBadge, 0, len(badges))
	for _, b := range badges {
		result = append(result, domain.MatchEncounterBadge{
			Kind:       string(b.Kind),
			LabelKey:   b.LabelKey,
			ColorToken: b.ColorToken,
			Detail:     b.Detail,
		})
	}
	return result
}

// Clés canoniques des métriques de comparaison (utilisées comme keys de map ET
// comme key des MetricRows envoyés au front).
const (
	compareMetricMaxKillingSpree      = "max_killing_spree"
	compareMetricAvgLifeSecs          = "avg_life_secs"
	compareMetricPerfectKillsPerGame  = "perfect_kills_per_game"
	compareMetricHeadshotKillsPerGame = "headshot_kills_per_game"
	compareMetricPerfATH              = "perf_ath"
	compareMetricLusrATH              = "lusr_ath"
	compareMetricCareerRank           = "career_rank"
	compareMetricAccuracy             = "accuracy"
)

// Métriques calculées uniquement à partir de la DB locale d'un joueur (stats.duckdb
// + agrégats shared.match_participants par xuid). Quand le joueur n'est pas local,
// l'API Waypoint career-stats ne les fournit pas → valeur 0 = donnée absente.
var localOnlyMetrics = map[string]bool{
	compareMetricMaxKillingSpree:      true,
	compareMetricAvgLifeSecs:          true,
	compareMetricPerfectKillsPerGame:  true,
	compareMetricHeadshotKillsPerGame: true,
}

// Métriques issues de stats.duckdb (ATH + rang carrière). Indisponibles si le
// joueur n'est pas local, ET considérées non renseignées si la valeur est 0
// (cas d'un joueur local dont l'ATH n'a pas encore été calculé).
var athMetrics = map[string]bool{
	compareMetricPerfATH:    true,
	compareMetricLusrATH:    true,
	compareMetricCareerRank: true,
}

// metricAvailability détermine si la valeur d'une métrique est exploitable pour un joueur.
//
//   - athMetrics (perf_ath/lusr_ath/career_rank) : exigent IsLocal=true ET valeur>0.
//     L'échantillon croisé ne donne pas l'ATH (stats de carrière globales).
//   - localOnlyMetrics (spree/life/perfect/headshots) : IsLocal OU IsLocalSample
//     (le service alimente IsLocalSample pour un joueur B remote ayant un échantillon
//     de matchs croisés avec A — métriques alors calculées sur cet échantillon).
//   - Autres : toujours disponibles (alimentées par Waypoint ou les agrégats locaux).
func metricAvailability(key string, value float64, isLocal, isLocalSample bool) bool {
	if athMetrics[key] {
		return isLocal && value > 0
	}
	if localOnlyMetrics[key] {
		return isLocal || isLocalSample
	}
	return true
}

// buildMetrics construit les CompareMetricRows à partir des deux stats normalisées.
func buildMetrics(a, b domain.NormalizedPlayerStats) []domain.CompareMetricRow {
	// Rendement = dégâts / frag / 225 — moins = plus efficace.
	rendementA, rendementB := computeRendement(a), computeRendement(b)
	// Résistance = dégâts subis / mort / 225 — plus = plus résistant.
	resistanceA, resistanceB := computeResistance(a), computeResistance(b)

	type metricDef struct {
		key          string
		label        string
		va           float64
		vb           float64
		lessIsBetter bool
	}
	defs := []metricDef{
		// win_rate et accuracy envoyés en fraction 0..1 — le frontend multiplie par 100 à l'affichage.
		{"win_rate", "Taux de victoire", a.WinRate, b.WinRate, false},
		{"kda", "KDA", a.KDA, b.KDA, false},
		{"kdr", "K/D", a.KDR, b.KDR, false},
		{"kills_per_game", "Frags / partie", a.KillsPerGame, b.KillsPerGame, false},
		{"deaths_per_game", "Morts / partie", a.DeathsPerGame, b.DeathsPerGame, true},
		{"assists_per_game", "Assistances / partie", a.AssistsPerGame, b.AssistsPerGame, false},
		{compareMetricAccuracy, "Précision", a.Accuracy, b.Accuracy, false},
		{"damage_per_game", "Dégâts / partie", a.DamagePerGame, b.DamagePerGame, false},
		{"rendement", "Rendement", rendementA, rendementB, true},
		{"damage_taken_per_game", "Dégâts subis / partie", a.DamageTakenPerGame, b.DamageTakenPerGame, true},
		{"resistance", "Résistance", resistanceA, resistanceB, false},
		{compareMetricPerfectKillsPerGame, "Tirs parfaits / partie", a.PerfectKillsPerGame, b.PerfectKillsPerGame, false},
		{compareMetricMaxKillingSpree, "Folie meurtrière max", float64(a.MaxKillingSpree), float64(b.MaxKillingSpree), false},
		{compareMetricAvgLifeSecs, "Survie moy. / partie", a.AvgLifeSecs, b.AvgLifeSecs, false},
		{compareMetricHeadshotKillsPerGame, "Headshots / partie", a.HeadshotKillsPerGame, b.HeadshotKillsPerGame, false},
		{"matches", "Parties", float64(a.Matches), float64(b.Matches), false},
		{compareMetricCareerRank, "Rang Carrière", float64(a.CareerRank), float64(b.CareerRank), false},
		{compareMetricPerfATH, "Perf. record", a.PerfATH, b.PerfATH, false},
		{compareMetricLusrATH, "LUSR record", a.LusrATH, b.LusrATH, false},
	}

	// SampleSizeB non nul uniquement si B est un joueur local croisé.
	sampleSizeB := 0
	if b.IsLocal {
		sampleSizeB = b.Matches
	}

	rows := make([]domain.CompareMetricRow, 0, len(defs))
	for _, d := range defs {
		aAvail := metricAvailability(d.key, d.va, a.IsLocal, a.IsLocalSample)
		bAvail := metricAvailability(d.key, d.vb, b.IsLocal, b.IsLocalSample)
		// Si la métrique est indisponible des deux côtés, on masque la ligne :
		// pas de valeur comparable et rien d'informatif à afficher.
		if !aAvail && !bAvail {
			continue
		}
		// Si la métrique est disponible des deux côtés mais vaut 0 partout,
		// on masque aussi (pas d'info utile à afficher).
		if aAvail && bAvail && d.va == 0 && d.vb == 0 {
			continue
		}
		var winner string
		var delta float64
		if aAvail && bAvail {
			winner = computeWinner(d.va, d.vb, d.lessIsBetter)
			delta = d.vb - d.va
		}
		rows = append(rows, domain.CompareMetricRow{
			Metric:          d.key,
			LabelFR:         d.label,
			ValueA:          d.va,
			ValueB:          d.vb,
			ValueAAvailable: aAvail,
			ValueBAvailable: bAvail,
			Delta:           delta,
			Winner:          winner,
			LessIsBetter:    d.lessIsBetter,
			SampleSizeB:     sampleSizeB,
		})
	}
	return rows
}

func computeWinner(va, vb float64, lessIsBetter bool) string {
	const eps = 0.001
	if lessIsBetter {
		if va < vb-eps {
			return "a"
		}
		if vb < va-eps {
			return "b"
		}
		return "tie"
	}
	if math.Abs(va-vb) <= eps {
		return "tie"
	}
	if va > vb {
		return "a"
	}
	return "b"
}

func computeRendement(s domain.NormalizedPlayerStats) float64 {
	if s.KillsPerGame <= 0 {
		return 0
	}
	return s.DamagePerGame / s.KillsPerGame / 225.0
}

func computeResistance(s domain.NormalizedPlayerStats) float64 {
	if s.DeathsPerGame <= 0 {
		return 0
	}
	return s.DamageTakenPerGame / s.DeathsPerGame / 225.0
}
