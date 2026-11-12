// Package service — openspartan_post_import_service.go: recomputes the v6
// derived fields (sessions, performance scores, citations) after a bulk
// OpenSpartan import has finished writing the raw rows.
//
// The post-import service touches the per-player stats.duckdb (creating it
// via sync.OpenPlayerDB if missing — typical at onboarding time) and reads
// citation_mappings from metadata.duckdb. Each stage runs best-effort:
// per-stage failure goes into port.OpenSpartanPostImportResult.Errors without aborting the
// remaining stages.
package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/config"
	"levelup/go-api/internal/ctxkeys"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/platform/friendstore"
	settings_platform "levelup/go-api/internal/platform/settings"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/sync"
)

// OpenSpartanPostImportService runs the three recompute stages for a player
// after their OpenSpartan import succeeded.
//
// Pas de handle persistant : la base du joueur s'ouvre par port.OpenSpartanPostImportStore à chaque Run, et
// chaque étape prend elle-même la base partagée qu'elle écrit (via le Provider), ce qui évite le conflit
// "different configuration" au boot du serveur.
type OpenSpartanPostImportService struct {
	cfg   *config.AppConfig
	store port.OpenSpartanPostImportStore
	log   *slog.Logger
}

// NewOpenSpartanPostImportService construit le service ; cfg donne les chemins et la règle de session du
// titre, store ouvre la base du joueur et porte les étapes de recalcul.
func NewOpenSpartanPostImportService(cfg *config.AppConfig, store port.OpenSpartanPostImportStore) *OpenSpartanPostImportService {
	return &OpenSpartanPostImportService{
		cfg:   cfg,
		store: store,
		log:   slog.Default(),
	}
}

// Run executes the three recompute stages. Returns a non-nil error only on
// fatal initialisation failures (missing required inputs, player DB cannot
// be opened); per-stage failures are surfaced via port.OpenSpartanPostImportResult.Errors.
func (s *OpenSpartanPostImportService) Run(
	ctx context.Context,
	xuid, gamertag string,
	matchIDs []string,
	opts port.OpenSpartanPostImportOptions,
) (port.OpenSpartanPostImportResult, error) {
	if xuid == "" || gamertag == "" {
		return port.OpenSpartanPostImportResult{}, errors.New("post-import: xuid and gamertag are required")
	}
	opts = applyPostImportDefaults(opts)
	// LE TITRE DES BASES, STAMPÉ UNE FOIS, POUR TOUTES LES ÉTAPES. Toutes les bases
	// ouvertes ci-dessous sont celles de opts.TitleSlug ; le ctx entrant, lui, porte
	// le titre de la REQUÊTE (en-tête X-LevelUp-Title, session ouverte sur un autre
	// onglet). Les étapes qui lisent ctxkeys.TitleSlug — le replay LUSR
	// (GetLUSRChainForTitle) comme le recalcul des notes de performance
	// (GetPerformanceChain, title-aware depuis 5be99a2c3) — écriraient sinon dans la
	// base d'un titre la classification d'un AUTRE : c'est la corruption du
	// 2026-06-26, dont C.1
	// n'avait fermé que le volet LUSR. Le stamp est ici, À L'ENTRÉE, pour qu'une
	// étape AJOUTÉE plus bas en hérite sans que personne ait à y penser
	// (garde-rail : TestRunStampeLeTitreAvantLaPremiereEtape).
	ctx = postImportCtx(ctx, opts.TitleSlug)

	pr := s.cfg.PathResolver()
	paths := port.OpenSpartanPostImportPaths{
		Player:   config.PlayerDBPath(s.cfg, opts.TitleSlug, gamertag),
		Shared:   config.SharedDBPath(s.cfg, opts.TitleSlug),
		Metadata: pr.MetadataDBPath(opts.TitleSlug),
		PVE:      pr.SharedPVEDBPath(opts.TitleSlug),
	}

	session, err := s.store.Open(ctx, paths)
	if err != nil {
		return port.OpenSpartanPostImportResult{}, fmt.Errorf("OpenPlayerDB %s: %w", paths.Player, err)
	}
	defer session.Close()

	var result port.OpenSpartanPostImportResult
	s.ensureEnrichmentRows(ctx, session, opts.TitleSlug, matchIDs, &result)
	s.recomputeCSR(ctx, session, xuid, &result)
	s.recomputeLUSR(ctx, session, xuid, &result)
	s.recomputeSessions(ctx, paths.Player, paths.Shared, xuid, opts, &result)
	s.recomputePerfScores(ctx, session, xuid, opts.ForcePerfScores, &result)
	s.recomputeCitations(ctx, session, xuid, matchIDs, &result)
	return result, nil
}

// postImportCtx stampe sur le ctx le titre des BASES du post-import — jamais celui
// du ctx entrant. Fonction nommée, et pas un appel en ligne, pour que l'invariant
// « le titre stampé est celui de la base » soit testable seul.
func postImportCtx(ctx context.Context, titleSlug string) context.Context {
	return ctxkeys.WithTitleSlug(ctx, titleSlug)
}

// failStage range l'échec d'une étape dans le résultat et au journal. Le nom de l'étape est celui que la
// session porte dans son erreur (prise d'une base avant le calcul), sinon celui de l'étape appelée.
func (s *OpenSpartanPostImportService) failStage(result *port.OpenSpartanPostImportResult, stage, xuid string, err error) {
	var stageErr *port.OpenSpartanStageError
	if errors.As(err, &stageErr) {
		stage = stageErr.Stage
	}
	result.Errors = append(result.Errors, port.OpenSpartanPostImportError{Stage: stage, Err: err.Error()})
	s.log.Warn("post_import_"+stage+"_failed", "xuid", xuid, "err", err)
}

// recomputeCSR projette le CSR par-match du joueur depuis shared.match_csrs
// (écrit à l'import depuis RankRecap) vers player.match_skill_rank, que l'UI lit
// pour afficher le rang par match. Pur local, aucun appel API.
func (s *OpenSpartanPostImportService) recomputeCSR(
	ctx context.Context, session port.OpenSpartanPostImportSession, xuid string, result *port.OpenSpartanPostImportResult,
) {
	n, err := session.ProjectCSR(ctx, xuid)
	if err != nil {
		s.failStage(result, "csr", xuid, err)
		return
	}
	result.CSRProjected = n
}

// recomputeLUSR rejoue le LUSR (v2) du joueur sur tout son historique pour
// intégrer les matchs importés (anciens), que le chemin live incrémental
// sauterait via le watermark. Pur local, aucun appel API. Best-effort.
func (s *OpenSpartanPostImportService) recomputeLUSR(
	ctx context.Context, session port.OpenSpartanPostImportSession, xuid string, result *port.OpenSpartanPostImportResult,
) {
	n, err := session.RecomputeLUSR(ctx, xuid)
	if err != nil {
		s.failStage(result, stageLUSR, xuid, err)
		return
	}
	result.LUSRRecomputed = n
}

// stagePrimeEnrichment : libellé de l'étape « prime enrichment » dans les
// port.OpenSpartanPostImportError (constante pour éviter la répétition du littéral — goconst).
const stagePrimeEnrichment = "prime_enrichment"

// stageLUSR : libellé de l'étape du rejeu de la note LUSR dans les port.OpenSpartanPostImportError.
const stageLUSR = "lusr"

// counterPrimeEnrichmentFailed compte les lignes de base d'enrichissement qui n'ont pas pu être écrites.
const counterPrimeEnrichmentFailed = "openspartan_prime_enrichment_failed_total"

// ensureEnrichmentRows primes player_match_enrichment with one baseline row
// (stage='live') per imported match_id. Required because the recompute stages
// (sessions, performance_score) source their work-list from PME rows.
//
// L'écriture passe par persist.PostSyncEnrichmentPersister.PrimeBaseline (INSERT pur, idempotent par
// pré-filtre sur la vue _latest). Un match en échec n'arrête pas les suivants : chaque échec est
// journalisé, rangé dans le résultat et compté.
func (s *OpenSpartanPostImportService) ensureEnrichmentRows(
	ctx context.Context, session port.OpenSpartanPostImportSession, titleSlug string, matchIDs []string, result *port.OpenSpartanPostImportResult,
) {
	if len(matchIDs) == 0 {
		return
	}
	failures, err := session.PrimeEnrichment(ctx, matchIDs)
	if err != nil {
		result.Errors = append(result.Errors, port.OpenSpartanPostImportError{Stage: stagePrimeEnrichment, Err: err.Error()})
		s.log.Warn("post_import_prime_enrichment_load_failed", "err", err)
		return
	}
	for _, f := range failures {
		result.Errors = append(result.Errors, port.OpenSpartanPostImportError{Stage: stagePrimeEnrichment, MatchID: f.MatchID, Err: f.Err.Error()})
		s.log.Warn("post_import_prime_enrichment_exec_failed", "match_id", f.MatchID, "err", f.Err)
	}
	if len(failures) > 0 {
		observability.AddIntT(titleSlug, counterPrimeEnrichmentFailed, int64(len(failures)))
	}
}

// recomputeSessions runs sync.RecalculatePlayerSessions with the title's session rule
// (settings.SessionOptions, the one the sync and the admin recalc apply) and the player's
// friends (data/global/player_friends.json). The helper acquires its own write leases on
// player and shared DBs internally.
func (s *OpenSpartanPostImportService) recomputeSessions(
	ctx context.Context,
	playerDBPath, sharedDBPath, xuid string,
	opts port.OpenSpartanPostImportOptions,
	result *port.OpenSpartanPostImportResult,
) {
	sessionOpts := settings_platform.SessionOptionsForConfig(ctx, s.cfg, opts.TitleSlug)
	friends, ferr := friendstore.NewFriendStore(s.cfg.RuntimePaths().PlayerFriendsPath()).Get(xuid)
	if ferr != nil {
		s.log.WarnContext(ctx, "post_import_sessions_friends_unavailable", "xuid", xuid, "err", ferr)
	}
	n, err := sync.RecalculatePlayerSessions(ctx, s.cfg.SharedProvider, playerDBPath, sharedDBPath, xuid,
		sessionOpts, friends,
	)
	if err != nil {
		result.Errors = append(result.Errors, port.OpenSpartanPostImportError{Stage: scopeSessions, Err: err.Error()})
		s.log.Warn("post_import_sessions_failed", "xuid", xuid, "err", err)
		return
	}
	result.SessionsTouched = n
}

// recomputePerfScores runs sync.BatchComputePerformanceScores on the player.
// Skipped silently when the player has fewer than MinMatchesPerChainForRelative
// matches per chain — that's encoded in the helper itself.
func (s *OpenSpartanPostImportService) recomputePerfScores(
	ctx context.Context, session port.OpenSpartanPostImportSession, xuid string, force bool, result *port.OpenSpartanPostImportResult,
) {
	n, err := session.ComputePerfScores(ctx, xuid, force)
	if err != nil {
		s.failStage(result, "perf_scores", xuid, err)
		return
	}
	result.PerfScoresTouched = n
}

// recomputeCitations recalcule les citations des matchs donnés. Skipped if matchIDs is empty (nothing to do).
func (s *OpenSpartanPostImportService) recomputeCitations(
	ctx context.Context, session port.OpenSpartanPostImportSession, xuid string, matchIDs []string, result *port.OpenSpartanPostImportResult,
) {
	if len(matchIDs) == 0 {
		return
	}
	if err := session.BackfillCitations(ctx, xuid, matchIDs); err != nil {
		s.failStage(result, "citations", xuid, err)
		return
	}
	result.CitationsBackfilled = true
}

func applyPostImportDefaults(opts port.OpenSpartanPostImportOptions) port.OpenSpartanPostImportOptions {
	if opts.TitleSlug == "" {
		opts.TitleSlug = titlePkg.DefaultSlug
	}
	return opts
}
