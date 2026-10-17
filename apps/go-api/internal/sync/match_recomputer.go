// Package sync — match_recomputer.go : recompute perf_score + LUSR après une
// exclusion/réactivation manuelle de match.
//
// Sémantique : lorsque l'utilisateur (dé)exclut un match via PATCH
// /matches/{id}/exclusion, le flag `is_excluded` est mis à jour dans
// player_match_enrichment. `batchComputePerformanceScores` et le rejeu LUSR v2
// (RecomputeLUSRCanonicalWithAccess) filtrent les matchs exclus en amont via
// `loadExcludedMatchIDs`. Il faut donc relancer ces deux calculs en mode complet pour
// propager le changement aux matchs ultérieurs de la même chaîne / playlist_group.
//
// Choix d'implémentation : recompute global plutôt que ciblé par chaîne+fromTime.
// Justification :
//   - La cascade LUSR doit de toute façon repartir du premier match du playlist_group
//     concerné pour reconstituer l'état mu/sigma (pas de reprise en milieu de chaîne) :
//     le rejeu v2 remet à zéro l'état du joueur (sentinelle is_reset) puis rejoue tout.
//   - Les chaînes/groupes non impactés produisent des scores identiques (idempotent par
//     construction) → coût marginal en lecture seulement, pas de divergence de valeurs.
//   - Implémentation drastiquement plus simple, sans dupliquer la logique des calculs.
//
// Volumétrie typique : 5000-10000 matchs total → ~2-10s en synchrone.
// Si la latence pose problème, ré-introduire un mode ciblé via paramètre.
//
// Base partagée — contrat d'accès (D-6, plan backlog 2026-10-16) : le recalcul LIT
// la base partagée sous des segments de lecture (le calcul des notes de performance),
// et le LUSR v2 y écrit son état (player_skill_state_v2) par RAFALES COURTES : une
// rafale pour la sentinelle de reset, puis au plus 50 matchs ou 2 s par rafale pour le
// rejeu. L'écrivain partagé n'est JAMAIS tenu en continu pendant le recalcul : une
// bascule en écriture longue ferait répondre 503 aux lecteurs de tout le site. Les notes
// LUSR et les composantes s'écrivent en base joueur (persisters INSERT-only). Des demandes
// répétées pour un même joueur sont absorbées (coalesce.Runs).
package sync

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/platform/coalesce"
	"levelup/go-api/internal/platform/dblease"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
	"levelup/go-api/internal/sync/skill"
)

// MatchRecomputer recalcule performance_score + LUSR pour un joueur après une
// (dé)exclusion de match. Détient les paths nécessaires à l'acquisition des
// leases d'écriture sur les deux DBs.
//
// sharedProvider : si non-nil, la base partagée se lit via Provider.Get (coordonné
// avec le pool joueur et les swaps du sync). Si nil, OpenReadForQuery.
type MatchRecomputer struct {
	playerDBPath   string
	sharedDBPath   string
	metadataDBPath string
	xuid           string
	gamertag       string
	sharedProvider sharedprovider.Provider
	// titleSlug : titre des bases du joueur (profil), posé sur le ctx des lectures de sync
	// qui excluent la Campagne. Vide : le ctx entrant décide (repli title-agnostique si absent).
	titleSlug string
}

// NewMatchRecomputer construit un MatchRecomputer pour un joueur donné.
// metadataDBPath peut être vide → recompute fonctionne sans bonus médailles.
// provider peut être nil → lecture par OpenReadForQuery.
func NewMatchRecomputer(
	playerDBPath, sharedDBPath, metadataDBPath, xuid, gamertag string,
	provider sharedprovider.Provider,
) *MatchRecomputer {
	return &MatchRecomputer{
		playerDBPath:   playerDBPath,
		sharedDBPath:   sharedDBPath,
		metadataDBPath: metadataDBPath,
		xuid:           xuid,
		gamertag:       gamertag,
		sharedProvider: provider,
	}
}

// WithTitle fixe le titre des bases du joueur (celui de son profil, jamais celui de la requête).
func (r *MatchRecomputer) WithTitle(slug string) *MatchRecomputer {
	r.titleSlug = slug
	return r
}

// titledCtx pose sur le ctx le titre des bases du joueur quand il est connu.
func (r *MatchRecomputer) titledCtx(ctx context.Context) context.Context {
	if r.titleSlug == "" {
		return ctx
	}
	return ctxkeys.WithTitleSlug(ctx, r.titleSlug)
}

// RecomputeAfterExclusion relance les batches perf_score + LUSR en force=true
// pour propager l'effet d'une (dé)exclusion. matchID est utilisé pour les logs
// uniquement (le filtrage se fait via le flag `is_excluded` déjà persisté). Les demandes
// simultanées pour le même joueur sont absorbées : jamais deux recalculs en parallèle. Le titre
// des lectures est celui du profil (titledCtx), pas celui de la requête.
func (r *MatchRecomputer) RecomputeAfterExclusion(ctx context.Context, matchID string) error {
	return exclusionRecomputes.Do(ctx, r.playerDBPath, func(ctx context.Context) error {
		return r.recompute(ctx, matchID)
	})
}

// RecomputePending dit si le dernier recalcul de ce joueur a échoué : une demande
// d'exclusion identique à l'état déjà persisté doit alors recalculer quand même.
func (r *MatchRecomputer) RecomputePending() bool {
	return exclusionRecomputes.FailedLast(r.playerDBPath)
}

// exclusionRecomputes absorbe les recalculs d'exclusion par base joueur, pour tout le process.
var exclusionRecomputes coalesce.Runs

// sharedAccess construit l'accès en rafales à la base partagée de ce recalcul : lecture par
// le fournisseur (ou OpenReadForQuery sans fournisseur), écriture par un burst
// AcquireSharedWriterStandalone relâché dès la rafale terminée.
func (r *MatchRecomputer) sharedAccess() *SharedAccess {
	return NewBurstSharedAccess(
		func(ctx context.Context) (*sql.DB, func(), error) {
			return AcquireSharedWriterStandalone(ctx, r.sharedProvider, r.sharedDBPath)
		},
		func(ctx context.Context) (*sql.DB, func(), error) {
			return acquireSharedReaderStandalone(ctx, r.sharedProvider, r.sharedDBPath)
		},
		"match_recompute")
}

func (r *MatchRecomputer) recompute(ctx context.Context, matchID string) error {
	ctx = r.titledCtx(ctx)
	start := time.Now()
	slog.InfoContext(ctx, "MatchRecomputer: démarrage",
		"match_id", matchID, "gamertag", r.gamertag, "xuid", r.xuid)

	writerPlayer, err := dblease.AcquireWriterCtx(ctx, nil, r.playerDBPath, dblease.KindPlayer)
	if err != nil {
		return fmt.Errorf("MatchRecomputer.RecomputeAfterExclusion lease player: %w", err)
	}
	defer writerPlayer.Release()

	playerHandle, err := OpenPlayerDB(r.playerDBPath)
	if err != nil {
		return fmt.Errorf("MatchRecomputer.RecomputeAfterExclusion OpenPlayerDB: %w", err)
	}
	defer playerHandle.Close()

	shared := r.sharedAccess()
	playerSQL := playerHandle.SQLDb()

	updatedPerf, err := r.recomputePerformance(ctx, playerSQL, shared)
	if err != nil {
		return err
	}

	updatedLUSR, err := RecomputeLUSRCanonicalWithAccess(ctx, playerSQL, shared, r.xuid,
		skill.WithMedalExploit(r.medalExploit))
	if err != nil {
		return fmt.Errorf("MatchRecomputer.RecomputeAfterExclusion lusr: %w", err)
	}

	slog.InfoContext(ctx, "MatchRecomputer: terminé",
		"match_id", matchID, "gamertag", r.gamertag,
		"updated_perf", updatedPerf, "updated_lusr", updatedLUSR,
		"duration_ms", time.Since(start).Milliseconds())
	return nil
}

// medalExploit charge les scores d'exploit médailles du joueur (metadata DB + médailles
// gagnées de la base partagée) pour les calculs de ce recalcul.
func (r *MatchRecomputer) medalExploit(ctx context.Context, sharedDB *sql.DB) map[string]float64 {
	return loadMedalExploitMap(ctx, r.metadataDBPath, sharedDB, r.xuid)
}

// recomputePerformance relance les notes de performance en force sous un segment de
// LECTURE shared, relâché avant le rejeu LUSR (garde anti-deadlock d'un SharedAccess :
// aucun Read en vol quand une rafale d'écriture est demandée).
func (r *MatchRecomputer) recomputePerformance(ctx context.Context, playerSQL *sql.DB, shared *SharedAccess) (int, error) {
	sharedSQL, release, err := shared.Read(ctx)
	if err != nil {
		return 0, fmt.Errorf("MatchRecomputer.RecomputeAfterExclusion: %w", err)
	}
	defer release()
	updated, err := batchComputePerformanceScores(ctx, playerSQL, sharedSQL, r.xuid, r.medalExploit(ctx, sharedSQL), true)
	if err != nil {
		return 0, fmt.Errorf("MatchRecomputer.RecomputeAfterExclusion perf: %w", err)
	}
	return updated, nil
}
