// Package service — compare_service.go : comparaison joueur vs joueur.
//
// Sprint 54 C : CompareService avec chargement parallèle via errgroup.
// Joueur A : données DuckDB (CompareRepository) + CSR depuis Waypoint skill endpoint.
// Joueur B : Waypoint (PlayerStatsProvider) + fallback DuckDB si connu localement.
package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"math"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/narrative"
	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/mappings"
	"levelup/go-api/internal/port"

	"golang.org/x/sync/errgroup"
)

// CompareService orchestre la comparaison joueur A vs joueur B.
type CompareService struct {
	repo            port.CompareRepository
	provider        port.PlayerStatsProvider
	liveIdentity    ExplorerTargetIdentityProvider // optionnel : rang carrière live d'un B non-local
	csr             ExplorerTargetCSRProvider      // optionnel : CSR saison courante live (A et B)
	currentSeasonID string                         // saison CSR courante (pour le fetch CSR)
	ranks           *mappings.RankCatalog          // optionnel : titres de rang carrière (même catalogue que l'Explorer)
	liveResolver    GamertagXUIDResolver           // optionnel : résout gamertag→xuid live pour un B jamais croisé (enrichit rang/CSR)
	xuidA           string
	titleSlug       string
}

// NewCompareService crée un CompareService.
func NewCompareService(repo port.CompareRepository, provider port.PlayerStatsProvider, xuidA, titleSlug string) *CompareService {
	return &CompareService{repo: repo, provider: provider, xuidA: xuidA, titleSlug: titleSlug}
}

// WithLiveIdentity injecte le provider d'identité live (career rank/emblem/...)
// — même provider que l'Explorer — pour récupérer le rang carrière d'un joueur B
// NON-local (le service record Waypoint ne le fournit pas). nil → rang carrière
// reste "N/A" pour les non-locaux (comportement historique).
func (s *CompareService) WithLiveIdentity(p ExplorerTargetIdentityProvider) *CompareService {
	s.liveIdentity = p
	return s
}

// WithLiveGamertagResolver injecte le résolveur live gamertag→xuid : permet, pour
// un joueur B JAMAIS croisé (absent localement), de récupérer son xuid et donc
// d'enrichir rang carrière + CSR (sinon B reste sans rang/CSR). nil → no-op.
func (s *CompareService) WithLiveGamertagResolver(r GamertagXUIDResolver) *CompareService {
	s.liveResolver = r
	return s
}

// WithCSR injecte le provider CSR live (même que l'Explorer) + la saison courante,
// pour comparer le meilleur CSR de A et B. nil → pas de ligne CSR.
func (s *CompareService) WithCSR(p ExplorerTargetCSRProvider, currentSeasonID string) *CompareService {
	s.csr = p
	s.currentSeasonID = currentSeasonID
	return s
}

// WithRanks injecte le catalogue de rangs carrière (même source que l'Explorer)
// pour afficher le rang en titre ("Général Platine VI") plutôt qu'en numéro.
func (s *CompareService) WithRanks(ranks *mappings.RankCatalog) *CompareService {
	s.ranks = ranks
	return s
}

// csrSummary regroupe le meilleur CSR courant et all-time d'un joueur (valeur +
// libellé tier prêt à afficher), dérivés du même provider que le profil de combat.
type csrSummary struct {
	currentValue float64
	currentLabel string
	allTimeValue float64
	allTimeLabel string
}

// csrUnrankedLabel : libellé quand le CSR a bien été RÉCUPÉRÉ mais que le joueur
// n'est pas classé — à distinguer du libellé vide (= non récupéré → N/A côté front).
const csrUnrankedLabel = "Non classé"

// fetchCSRSummary récupère les CSR du joueur (live, tout xuid) et en extrait le
// meilleur courant + le meilleur all-time avec leurs libellés tier ("Platine IV",
// "Onyx"). Tri-état porté par le libellé :
//   - "" (label vide)        → données NON récupérées (pas d'auth/erreur) → N/A.
//   - "Non classé"           → récupéré mais joueur non classé.
//   - "Or III" / "Onyx" etc. → classé.
func (s *CompareService) fetchCSRSummary(ctx context.Context, xuid string) csrSummary {
	if s.csr == nil || xuid == "" || s.currentSeasonID == "" {
		return csrSummary{} // non configuré → non récupéré
	}
	csrs, err := s.csr.SeasonCSRs(ctx, xuid, s.currentSeasonID)
	if err != nil {
		logBestEffortErr(ctx, "CompareService: CSR saison non disponible", err, "xuid", xuid)
		return csrSummary{} // échec fetch → non récupéré
	}
	// Récupéré : on part de "Non classé" et on remplace par le tier si classé.
	out := csrSummary{currentLabel: csrUnrankedLabel, allTimeLabel: csrUnrankedLabel}
	for _, c := range csrs {
		if c.Current.Value > out.currentValue {
			out.currentValue = c.Current.Value
			if lbl := csrRankLabel(c.Current); lbl != "" {
				out.currentLabel = lbl
			}
		}
		if c.AllTime.Value > out.allTimeValue {
			out.allTimeValue = c.AllTime.Value
			if lbl := csrRankLabel(c.AllTime); lbl != "" {
				out.allTimeLabel = lbl
			}
		}
	}
	return out
}

// csrRankLabel formate un rang CSR en libellé FR "tier + sous-palier romain"
// ("Platine IV", "Onyx", "Diamant II") via skillTierLabel (EN→FR) partagé.
// "" si tier vide (non classé).
func csrRankLabel(r domain.CareerCSRRank) string {
	if r.Tier == "" {
		return ""
	}
	tier := skillTierLabel(r.Tier)
	if tier == csrTierOnyx || r.SubTier < 1 || r.SubTier > 6 {
		return tier // Onyx (ou sous-palier hors plage) : pas de sous-palier romain
	}
	return tier + " " + romanSubTierCSR(r.SubTier)
}

var romanSubTierCSR = func() func(int) string {
	romans := map[int]string{1: "I", 2: "II", 3: "III", 4: "IV", 5: "V", 6: "VI"}
	return func(n int) string { return romans[n] }
}()

// careerRankTitle résout le titre localisé d'un numéro de rang carrière via le
// RankCatalog (réutilise analysis.BuildSpartanIdentity, comme le profil de combat).
// "" si rang ≤ 0 / non résolu.
func (s *CompareService) careerRankTitle(ctx context.Context, rankNumber int) string {
	if rankNumber <= 0 {
		return ""
	}
	id := analysis.BuildSpartanIdentity(
		&domain.HomeSpartanIdentityRow{RankNumber: rankNumber}, ctxkeys.Locale(ctx), s.ranks,
	)
	if id == nil || id.CareerRank == nil {
		return ""
	}
	return id.CareerRank.RankTitle
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
	// Titre du rang carrière A (résolu après le merge ATH, comme le profil de combat).
	statsA.CareerRankLabel = s.careerRankTitle(ctx, statsA.CareerRank)

	metrics := buildMetrics(*statsA, *statsB, games.EffectiveHpToKill(s.titleSlug))
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
		logBestEffortErr(ctx, "CompareService: ATH non disponible", athErr, "xuid", s.xuidA)
	}
	applyCSRSummary(statsA, s.fetchCSRSummary(ctx, s.xuidA))
	return statsA, ath, nil
}

// applyCSRSummary recopie un csrSummary sur les champs CSR des stats normalisées.
func applyCSRSummary(s *domain.NormalizedPlayerStats, sum csrSummary) {
	s.HighestCSR = sum.currentValue
	s.HighestCSRLabel = sum.currentLabel
	s.HighestCSRAllTime = sum.allTimeValue
	s.HighestCSRAllTimeLabel = sum.allTimeLabel
}

// logBestEffortErr distingue les erreurs "pas de data" (sql.ErrNoRows, attendu)
// des anomalies SQL/conn (anormal). Pattern motivé par le bug 2026-05-26 où
// un Catalog Error a vécu ~1 semaine sous Debug silencieux. Cf. thought_log
// "Fix shared.X via SharedReader — 10 sites cassés en silence".
func logBestEffortErr(ctx context.Context, msg string, err error, kv ...any) {
	if errors.Is(err, sql.ErrNoRows) {
		slog.DebugContext(ctx, msg+" (no rows)", kv...)
		return
	}
	attrs := append(kv, "err", err)
	slog.WarnContext(ctx, msg+" (best-effort, fallback)", attrs...)
}

// loadPlayerB tente d'abord la résolution locale (xuid → stats locales), sinon
// fallback Waypoint. Renvoie aussi le xuid résolu (vide si pas de match local).
func (s *CompareService) loadPlayerB(ctx context.Context, targetGamertag string) (*domain.NormalizedPlayerStats, string, error) {
	xuidB, _ := s.repo.ResolveXUID(ctx, targetGamertag)
	if xuidB == "" && s.liveResolver != nil {
		// Joueur B JAMAIS croisé : résolution live (PeopleHub→profil Xbox) pour
		// disposer de son xuid et enrichir rang carrière + CSR (le service record
		// Waypoint seul ne les fournit pas). Best-effort : sur échec, on garde le
		// fallback Waypoint gamertag-only (B s'affiche sans rang/CSR).
		if live, lerr := s.liveResolver.ResolveXUID(ctx, targetGamertag); lerr == nil && live != "" {
			xuidB = live
			slog.DebugContext(ctx, "compare_player_b_resolved_live", "gamertag", targetGamertag, "xuid", live)
		} else if lerr != nil {
			logBestEffortErr(ctx, "CompareService: résolution live joueur B", lerr, "gamertag", targetGamertag)
		}
	}
	if xuidB != "" {
		if local, err := s.repo.GetLocalStats(ctx, xuidB, s.titleSlug); err == nil && local != nil {
			s.enrichLocalPlayerB(ctx, local)
			// Fallback rang carrière live (même source que l'Explorer) si l'ATH local
			// ne le fournit pas — idempotent : skip si déjà > 0.
			s.fillCareerRankLive(ctx, local, xuidB)
			local.CareerRankLabel = s.careerRankTitle(ctx, local.CareerRank)
			applyCSRSummary(local, s.fetchCSRSummary(ctx, xuidB))
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
		s.fillCareerRankLive(ctx, remote, xuidB)
		remote.CareerRankLabel = s.careerRankTitle(ctx, remote.CareerRank)
		applyCSRSummary(remote, s.fetchCSRSummary(ctx, xuidB))
	}
	return remote, xuidB, nil
}

// fillCareerRankLive complète le rang carrière d'un joueur B via le fetch live
// identity (même source que le profil de combat Explorer) quand il manque — utile
// pour un B non-local (le service record Waypoint ne fournit pas le rang) comme
// pour un B local dont l'ATH ne porte pas encore le rang. xuidB doit être résolu.
// Best-effort : skip si pas de provider, pas de xuid, ou rang déjà présent.
func (s *CompareService) fillCareerRankLive(ctx context.Context, stats *domain.NormalizedPlayerStats, xuidB string) {
	if s.liveIdentity == nil || xuidB == "" || stats.CareerRank > 0 {
		return
	}
	id, err := s.liveIdentity.FetchLiveIdentity(ctx, xuidB)
	if err != nil {
		logBestEffortErr(ctx, "CompareService: rang carrière live joueur B non disponible", err, "xuid", xuidB)
		return
	}
	if id != nil && id.RankNumber > 0 {
		stats.CareerRank = id.RankNumber
		slog.DebugContext(ctx, "CompareService: rang carrière live joueur B", "xuid", xuidB, "rank", id.RankNumber)
	}
}

// enrichLocalPlayerB enrichit un joueur B local avec son ATH propre (rang carrière,
// records perf/LUSR).
func (s *CompareService) enrichLocalPlayerB(ctx context.Context, local *domain.NormalizedPlayerStats) {
	local.IsLocal = true

	athB, athErr := s.repo.GetPlayerATHFor(ctx, local.Gamertag, s.titleSlug)
	if athErr != nil {
		logBestEffortErr(ctx, "CompareService: ATH joueur B non disponible", athErr, "gamertag", local.Gamertag)
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
		logBestEffortErr(ctx, "CompareService: cross-match sample non disponible", sErr, "xuidA", s.xuidA, "xuidB", xuidB)
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
	compareMetricCSR                  = "csr"
	compareMetricCSRAllTime           = "csr_alltime"
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

// Métriques issues de stats.duckdb (records ATH). Indisponibles si le joueur
// n'est pas local, ET considérées non renseignées si la valeur est 0 (cas d'un
// joueur local dont l'ATH n'a pas encore été calculé). Le rang carrière en est
// EXCLU : il a une source live (FetchLiveIdentity) pour un B non-local, donc il
// est traité à part dans metricAvailability (cf. compareMetricCareerRank).
var athMetrics = map[string]bool{
	compareMetricPerfATH: true,
	compareMetricLusrATH: true,
}

// metricAvailability détermine si la valeur d'une métrique est exploitable pour un joueur.
//
//   - career_rank : disponible dès que valeur>0 (ATH local côté A OU rang récupéré
//     en live côté B non-local via FetchLiveIdentity).
//   - athMetrics (perf_ath/lusr_ath) : exigent IsLocal=true ET valeur>0.
//     L'échantillon croisé ne donne pas l'ATH (stats de carrière globales).
//   - localOnlyMetrics (spree/life/perfect/headshots) : IsLocal OU IsLocalSample
//     (le service alimente IsLocalSample pour un joueur B remote ayant un échantillon
//     de matchs croisés avec A — métriques alors calculées sur cet échantillon).
//   - Autres : toujours disponibles (alimentées par Waypoint ou les agrégats locaux).
func metricAvailability(key string, value float64, isLocal, isLocalSample bool) bool {
	if key == compareMetricCareerRank {
		// Disponible dès value>0 : rang connu côté A (local/live) comme côté B
		// non-local (fetch live). (Le CSR est traité à part dans buildMetrics, via
		// son libellé, pour distinguer "Non classé" de "non récupéré".)
		return value > 0
	}
	if athMetrics[key] {
		return isLocal && value > 0
	}
	if localOnlyMetrics[key] {
		return isLocal || isLocalSample
	}
	return true
}

// buildMetrics construit les CompareMetricRows à partir des deux stats normalisées.
func buildMetrics(a, b domain.NormalizedPlayerStats, effectiveHpToKill float64) []domain.CompareMetricRow {
	// Rendement (OC) / Résistance (DR) : MÊMES formules que la KPI bar home
	// (analysis.ComputeCombatYield) pour que le Face à face affiche les mêmes
	// chiffres. OC = 225*(kills+assists/3)/dégâts (plus haut = mieux) ;
	// DR = dégâts_subis/(225*morts), baseline 1.0 (plus haut = mieux).
	cyA, cyB := combatYieldOf(a, effectiveHpToKill), combatYieldOf(b, effectiveHpToKill)
	rendementA, rendementB := cyA.OffensiveConversion, cyB.OffensiveConversion
	resistanceA, resistanceB := cyA.DefensiveResistance, cyB.DefensiveResistance

	type metricDef struct {
		key          string
		label        string
		va           float64
		vb           float64
		lessIsBetter bool
		dispA        string // libellé d'affichage optionnel (rang titre, CSR tier)
		dispB        string
	}
	defs := []metricDef{
		// win_rate et accuracy envoyés en fraction 0..1 — le frontend multiplie par 100 à l'affichage.
		{"win_rate", "Taux de victoire", a.WinRate, b.WinRate, false, "", ""},
		{"kda", "KDA", a.KDA, b.KDA, false, "", ""},
		{"kdr", "K/D", a.KDR, b.KDR, false, "", ""},
		{"kills_per_game", "Frags / partie", a.KillsPerGame, b.KillsPerGame, false, "", ""},
		{"deaths_per_game", "Morts / partie", a.DeathsPerGame, b.DeathsPerGame, true, "", ""},
		{"assists_per_game", "Assistances / partie", a.AssistsPerGame, b.AssistsPerGame, false, "", ""},
		{compareMetricAccuracy, "Précision", a.Accuracy, b.Accuracy, false, "", ""},
		{"damage_per_game", "Dégâts / partie", a.DamagePerGame, b.DamagePerGame, false, "", ""},
		{"rendement", "Rendement", rendementA, rendementB, false, "", ""},
		{"damage_taken_per_game", "Dégâts subis / partie", a.DamageTakenPerGame, b.DamageTakenPerGame, true, "", ""},
		{"resistance", "Résistance", resistanceA, resistanceB, false, "", ""},
		{compareMetricPerfectKillsPerGame, "Tirs parfaits / partie", a.PerfectKillsPerGame, b.PerfectKillsPerGame, false, "", ""},
		{compareMetricMaxKillingSpree, "Folie meurtrière max", float64(a.MaxKillingSpree), float64(b.MaxKillingSpree), false, "", ""},
		{compareMetricAvgLifeSecs, "Survie moy. / partie", a.AvgLifeSecs, b.AvgLifeSecs, false, "", ""},
		{compareMetricHeadshotKillsPerGame, "Headshots / partie", a.HeadshotKillsPerGame, b.HeadshotKillsPerGame, false, "", ""},
		{"matches", "Parties", float64(a.Matches), float64(b.Matches), false, "", ""},
		{compareMetricCareerRank, "Rang Carrière", float64(a.CareerRank), float64(b.CareerRank), false, a.CareerRankLabel, b.CareerRankLabel},
		{compareMetricCSR, "CSR", a.HighestCSR, b.HighestCSR, false, a.HighestCSRLabel, b.HighestCSRLabel},
		{compareMetricCSRAllTime, "CSR record", a.HighestCSRAllTime, b.HighestCSRAllTime, false, a.HighestCSRAllTimeLabel, b.HighestCSRAllTimeLabel},
		{compareMetricPerfATH, "Perf. record", a.PerfATH, b.PerfATH, false, "", ""},
		{compareMetricLusrATH, "LUSR record", a.LusrATH, b.LusrATH, false, "", ""},
	}

	// SampleSizeB non nul uniquement si B est un joueur local croisé.
	sampleSizeB := 0
	if b.IsLocal {
		sampleSizeB = b.Matches
	}

	rows := make([]domain.CompareMetricRow, 0, len(defs))
	for _, d := range defs {
		// CSR : la disponibilité est portée par le LIBELLÉ (tri-état) pour distinguer
		// "Non classé" (récupéré) de N/A (non récupéré). dispX == "" → non récupéré.
		isCSR := d.key == compareMetricCSR || d.key == compareMetricCSRAllTime
		aAvail := metricAvailability(d.key, d.va, a.IsLocal, a.IsLocalSample)
		bAvail := metricAvailability(d.key, d.vb, b.IsLocal, b.IsLocalSample)
		if isCSR {
			aAvail = d.dispA != ""
			bAvail = d.dispB != ""
		}
		// Si la métrique est indisponible des deux côtés, on masque la ligne :
		// pas de valeur comparable et rien d'informatif à afficher.
		if !aAvail && !bAvail {
			continue
		}
		// Si la métrique est disponible des deux côtés mais vaut 0 partout, on masque
		// (pas d'info utile) — SAUF le CSR, où "Non classé" des deux côtés reste informatif.
		if !isCSR && aAvail && bAvail && d.va == 0 && d.vb == 0 {
			continue
		}
		var winner string
		var delta float64
		if aAvail && bAvail {
			winner = computeWinner(d.va, d.vb, d.lessIsBetter)
			delta = d.vb - d.va
		}
		row := domain.CompareMetricRow{
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
		}
		// Libellés d'affichage (rang titre, CSR tier) — uniquement du côté disponible.
		if aAvail {
			row.DisplayA = d.dispA
		}
		if bAvail {
			row.DisplayB = d.dispB
		}
		rows = append(rows, row)
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

// combatYieldOf calcule l'OC/DR d'un joueur depuis ses moyennes par partie, via
// la formule canonique partagée avec la home (analysis.ComputeCombatYieldFloat).
// Les moyennes par partie suffisent : le facteur 1/matches se simplifie.
func combatYieldOf(s domain.NormalizedPlayerStats, effectiveHpToKill float64) analysis.CombatYield {
	return analysis.ComputeCombatYieldFloat(
		s.KillsPerGame, s.AssistsPerGame, s.DamagePerGame, s.DamageTakenPerGame, s.DeathsPerGame, effectiveHpToKill,
	)
}
