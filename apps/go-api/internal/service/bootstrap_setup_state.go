// Package service — bootstrap_setup_state.go : l'état d'installation (setup_state) exposé par /bootstrap.
package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/domain"
)

// setupCountBudget borne le temps d'attente du décompte des matchs servant à
// distinguer ready vs profile_ready_no_sync. Court volontairement : ce décompte
// est cosmétique (signal du wizard), et sous contention DB (sync tenant le writer
// RW) une lecture shared peut pendre jusqu'au readyTimeout du provider (~30s), ce
// qui ferait échouer/rollback un switch de titre. Au-delà du budget, on dégrade.
const setupCountBudget = 2 * time.Second

func (s *BootstrapService) resolveSetupState(ctx context.Context, sess *domain.SessionData, currentTitleSlug string, players []domain.PlayerSummary) string {
	if len(players) == 0 {
		// SSO terminé (identité Halo liée en session) mais aucun profil local créé :
		// router vers StepPlayer (confirmation/création de profil) au lieu de
		// reboucler sur StepDeviceCode. Le Device Code Flow finit en "authorized"
		// sans auto-provisionner — le profil est créé à l'étape suivante.
		if ResolveLinkedIdentity(sess) != nil {
			return bootstrapSetupHaloLinkedNoProfile
		}
		return bootstrapSetupNoHaloLink
	}
	// Compter les matchs du shared du TITRE COURANT, best-effort borné : sous
	// contention (sync en cours) la dégradation profile_ready_no_sync est bénigne
	// (le routing /setup est gaté par players==0, pas par setup_state) et n'empêche
	// jamais un switch de titre d'aboutir.
	count, ok := s.matchCountForSetup(ctx, currentTitleSlug)
	if !ok {
		return bootstrapSetupProfileReadyNoSync
	}
	if count > 0 {
		return bootstrapSetupReady
	}
	return bootstrapSetupProfileReadyNoSync
}

// matchCountForSetup retourne (count, true) si le décompte des matchs du titre
// courant a été obtenu dans setupCountBudget, (0, false) sinon (dégradation). Le
// décompte passe par le résolveur title-aware s'il est câblé, sinon par bootRepo
// (legacy). Exécuté dans une goroutine bornée pour qu'un provider contendu
// (StateError, swap RW long) ne puisse jamais faire pendre le bootstrap.
func (s *BootstrapService) matchCountForSetup(ctx context.Context, currentTitleSlug string) (int, bool) {
	countFn := func(cctx context.Context) (int, error) {
		if s.matchCountForTitle != nil {
			return s.matchCountForTitle(cctx, currentTitleSlug)
		}
		if s.bootRepo != nil {
			return s.bootRepo.GetMatchCount(cctx)
		}
		return 0, fmt.Errorf("matchCountForSetup: aucun décompte disponible")
	}

	cctx, cancel := context.WithTimeout(ctx, setupCountBudget)
	defer cancel()

	type result struct {
		n   int
		err error
	}
	ch := make(chan result, 1) // bufferisé : la goroutine ne fuit pas si on timeout
	go func() {
		n, err := countFn(cctx)
		ch <- result{n: n, err: err}
	}()

	select {
	case r := <-ch:
		if r.err != nil {
			slog.WarnContext(ctx, "bootstrap: décompte matchs setup_state indisponible (dégradé)",
				"err", r.err, "title", currentTitleSlug)
			return 0, false
		}
		return r.n, true
	case <-cctx.Done():
		slog.WarnContext(ctx, "bootstrap: décompte matchs setup_state au-delà du budget, dégradation",
			"title", currentTitleSlug, "budget_ms", setupCountBudget.Milliseconds())
		return 0, false
	}
}
