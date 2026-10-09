// Package service — bootstrap_privacy.go : lecture live de la privacy du joueur
// courant pour /bootstrap, bornée dans le temps et mémorisée en échec (plan perf
// 2026-09-23, lot L5b, décision D5b.7). Extrait de bootstrap_service.go.
//
// /bootstrap bloque le rendu du shell : l'appel live est borné (privacyFetchBudget).
// Seul le succès était mis en cache (30 min, côté provider) : tant que Waypoint ne
// répondait pas dans le budget, CHAQUE /bootstrap attendait 2 s pour rien. Un échec
// ouvre désormais une attente de privacyFailureBackoff pendant laquelle l'appel
// n'est plus tenté — /bootstrap sert directement le state persisté (repli E3),
// exactement ce qu'il servait après les 2 s d'attente. Les échecs rapides (statut
// HTTP, réponse illisible) sont, eux, mémorisés par le provider, qui resservait
// déjà son repli « fetch_error » : le comportement de la page ne change pas.
package service

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/observability/timing"
)

const (
	// privacyFetchBudget : attente maximale de l'appel live par /bootstrap.
	privacyFetchBudget = 2 * time.Second
	// privacyFailureBackoff : après un échec (budget dépassé ou erreur du provider),
	// plus d'appel live pour ce xuid pendant cette durée.
	privacyFailureBackoff = 5 * time.Minute
)

// privacyLiveFetch porte le budget de l'appel live et la mémoire de ses échecs, par
// xuid. Sûr en concurrence ; un pointeur nil se comporte comme sans mémoire.
type privacyLiveFetch struct {
	budget time.Duration
	now    func() time.Time

	mu         sync.Mutex
	retryAfter map[string]time.Time // xuid → pas d'appel live avant cet instant
}

func newPrivacyLiveFetch() *privacyLiveFetch {
	return &privacyLiveFetch{budget: privacyFetchBudget, now: time.Now, retryAfter: make(map[string]time.Time)}
}

// waiting dit si le xuid est dans l'attente qui suit un échec.
func (l *privacyLiveFetch) waiting(xuid string) bool {
	if l == nil {
		return false
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.now().Before(l.retryAfter[xuid])
}

// recordFailure ouvre l'attente du xuid.
func (l *privacyLiveFetch) recordFailure(xuid string) {
	if l == nil {
		return
	}
	l.mu.Lock()
	l.retryAfter[xuid] = l.now().Add(privacyFailureBackoff)
	l.mu.Unlock()
}

func (l *privacyLiveFetch) fetchBudget() time.Duration {
	if l == nil || l.budget <= 0 {
		return privacyFetchBudget
	}
	return l.budget
}

// fetchPrivacyNonBlocking fetche la privacy avec un budget court (2 s). En cas
// d'échec, renvoie nil sans bloquer le bootstrap (repli E3 sur le state persisté) et
// mémorise l'échec : pendant privacyFailureBackoff, aucun appel live, nil immédiat
// (marqueur timing `privacy_live_backoff`). L'annulation de la requête elle-même
// n'est pas un échec de Waypoint et n'ouvre pas d'attente.
func (s *BootstrapService) fetchPrivacyNonBlocking(ctx context.Context, xuid string) *domain.MatchPrivacyInfo {
	if s.privacyLive.waiting(xuid) {
		timing.FromContext(ctx).Section("privacy_live_backoff")()
		slog.DebugContext(ctx, "bootstrap: privacy live en attente après un échec — state persisté", "xuid", xuid)
		return nil
	}
	type result struct {
		info *domain.MatchPrivacyInfo
	}
	ch := make(chan result, 1)
	timeoutCtx, cancel := context.WithTimeout(ctx, s.privacyLive.fetchBudget())
	defer cancel()

	go func() {
		info, err := s.privacyProvider.GetMatchPrivacy(timeoutCtx, xuid)
		if err != nil {
			s.privacyLive.recordFailure(xuid)
			slog.ErrorContext(ctx, "bootstrap: privacy fetch échoué — pas de nouvel appel avant l'échéance",
				"xuid", xuid, "retry_after", privacyFailureBackoff, "err", err)
			ch <- result{nil}
			return
		}
		ch <- result{info}
	}()

	select {
	case r := <-ch:
		return r.info
	case <-timeoutCtx.Done():
		// Budget écoulé alors que la requête vit encore : Waypoint n'a pas répondu à
		// temps. Une requête elle-même annulée ou échue n'ouvre pas d'attente.
		if ctx.Err() == nil && errors.Is(timeoutCtx.Err(), context.DeadlineExceeded) {
			s.privacyLive.recordFailure(xuid)
			slog.ErrorContext(ctx, "bootstrap: privacy fetch hors budget — pas de nouvel appel avant l'échéance",
				"xuid", xuid, "budget", s.privacyLive.fetchBudget(), "retry_after", privacyFailureBackoff,
				"err", timeoutCtx.Err())
			return nil
		}
		slog.DebugContext(ctx, "bootstrap: privacy fetch annulé avec la requête", "xuid", xuid, "err", timeoutCtx.Err())
		return nil
	}
}
