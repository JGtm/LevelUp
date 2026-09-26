// Package v2 — post_sync.go : Phase 6 du pipeline V2 (ADR 0027).
//
// Phase 6 = post-sync parallèle par joueur. Pour chaque joueur, on
// exécute la pipeline de heals + films + citations + dominance + LUSR
// en goroutine indépendante. Plus de contention sur le shared writer
// lease : Phase 5 a déjà terminé l'écriture, le worker batch est libre,
// les heals lisent shared concurremment et écrivent dans des DBs
// player isolées.
//
// C'est la phase où le gain perf maximal va se voir : en V1, Madina97294
// attendait 6 min 30 sur 8 min 30 totales JUSTE pour avoir le shared
// lease pour son post-sync. En V2, cette attente disparaît entièrement.
//
// L'adapter V1-bridge (D6) wrappe la fonction runPostSyncPipeline de
// engine_postsync.go en fournissant les DB handles per-player. Films
// chunks restent parallélisés en interne via backfill_weapons.go
// errgroup(24) — pas touché.
package v2

import (
	"context"
	gosync "sync"
	"time"

	"golang.org/x/sync/errgroup"
)

// PlayerPostSyncResult capture les compteurs du post-sync pour un joueur.
//
// Le détail des steps (heals, citations, dominance, etc.) est exposé pour
// l'observabilité expvar et le diagnostic. Si Err != nil, les compteurs
// peuvent être partiels (best-effort par step à l'intérieur du post-sync
// pipeline, qui continue sur warning).
type PlayerPostSyncResult struct {
	PlayerSlug             string
	CitationsComputed      int
	DominanceFlagsComputed int
	SkillHealed            int
	EventsHealed           int
	// WeaponKillsHealed RETIRÉ le 2026-09-01 : sa seule source
	// (domain.PostSyncResult.WeaponKillsProcessed) est partie avec l'étape 1.55.
	StatsHealed        int
	AchievementsSynced int
	Warnings           []string
	Duration           time.Duration
	Err                error
}

// PostSyncRunner exécute le post-sync pipeline complet pour UN joueur.
// L'implémentation V1-bridge (D6) acquiert les DB leases (player + shared
// reader), appelle runPostSyncPipeline, libère les leases, retourne les
// compteurs PostSyncResult mappés vers PlayerPostSyncResult.
//
// insertedIDs contient les match_ids fraîchement insérés en Phase 5 pour
// ce joueur. Utilisé par runPostSyncPipeline pour cibler les heals weapon
// kills + dominance flags sur les nouveaux matchs. Vide si le joueur n'a
// rien eu d'inséré (cas heal-only).
//
// Les tests utilisent un mock direct.
type PostSyncRunner interface {
	RunPostSync(ctx context.Context, p PlayerProfile, insertedIDs []string) (PlayerPostSyncResult, error)
}

// PostSyncCycleResult agrège le résultat de Phase 6.
//
// PerPlayer contient TOUS les joueurs même ceux en erreur (avec Err
// non-nil). Permet à l'orchestrator d'exposer le détail par joueur sans
// perdre l'info de qui a échoué.
type PostSyncCycleResult struct {
	PerPlayer map[string]PlayerPostSyncResult
	Duration  time.Duration
}

// RunPostSync exécute Phase 6 en parallèle, 1 goroutine par joueur.
//
// Sémantique :
//   - Parallélisme = len(players). Pas de borne artificielle : chaque
//     joueur écrit dans sa propre DB player, pas de contention.
//   - parallelism (optional override) <= 0 → parallélisme = len(players).
//     Si > 0, errgroup.SetLimit(parallelism). Utile pour tests ou
//     environnements contraints en goroutines.
//   - Erreurs par-joueur capturées dans PerPlayer[slug].Err, n'annulent
//     pas les autres.
//   - Retourne err != nil uniquement sur échec global (ctx annulé).
//   - insertedByPlayer (peut être nil) propage les match_ids insérés en
//     Phase 5 à chaque post-sync (cible weapon_kills/dominance sur les
//     nouveaux matchs).
func RunPostSync(
	ctx context.Context,
	players []PlayerProfile,
	runner PostSyncRunner,
	parallelism int,
	insertedByPlayer map[string][]string,
) (PostSyncCycleResult, error) {
	start := time.Now()
	res := PostSyncCycleResult{
		PerPlayer: make(map[string]PlayerPostSyncResult, len(players)),
	}
	if len(players) == 0 {
		res.Duration = time.Since(start)
		return res, nil
	}

	var mu gosync.Mutex
	eg, egCtx := errgroup.WithContext(ctx)
	if parallelism > 0 {
		eg.SetLimit(parallelism)
	}
	for _, p := range players {
		p := p
		eg.Go(func() error {
			pStart := time.Now()
			inserted := insertedByPlayer[p.PlayerSlug]
			pr, err := runner.RunPostSync(egCtx, p, inserted)
			pr.PlayerSlug = p.PlayerSlug // garde-rail invariant
			if pr.Duration == 0 {
				pr.Duration = time.Since(pStart)
			}
			if err != nil && pr.Err == nil {
				pr.Err = err
			}
			mu.Lock()
			res.PerPlayer[p.PlayerSlug] = pr
			mu.Unlock()
			return nil //nolint:nilerr // best-effort par joueur
		})
	}
	if err := eg.Wait(); err != nil {
		res.Duration = time.Since(start)
		return res, err
	}
	res.Duration = time.Since(start)
	return res, nil
}
