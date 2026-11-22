// Package v2 — persist_pending.go : un cycle dont l'attente de la file de persistance expire n'est
// pas un cycle perdu. Les matchs soumis sont durables et la file finit seule ; le cycle relit
// l'ensemble connu pour savoir lesquels sont déjà rangés, poursuit avec ceux-là (post-sync, état de
// l'historique) et laisse les autres à la file, sans les compter comme des échecs.
package v2

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/observability"
)

// counterPersistDrainTimeout : cycles dont l'attente de la file de persistance a expiré.
const counterPersistDrainTimeout = "sync_v2_persist_drain_timeout_total"

// persistProof est ce que le cycle sait de la persistance après l'expiration de l'attente. La
// valeur zéro (attente non expirée) ne dit rien : tout ce qui a été récupéré est rangé.
type persistProof struct {
	// timedOut : l'attente de la file a expiré.
	timedOut bool
	// proven : les matchs récupérés dont l'ensemble connu relu prouve le rangement.
	proven map[string]bool
	// queued : les matchs soumis dont le rangement n'est pas encore prouvé ; la file les finit.
	queued map[string]bool
}

// persisted dit si un match récupéré est à compter parmi les rangés : toujours quand l'attente n'a
// pas expiré (la persistance du cycle a réussi), sinon seulement s'il est prouvé.
func (p persistProof) persisted(matchID string) bool { return !p.timedOut || p.proven[matchID] }

// drainTimeoutOf rend l'erreur d'expiration de l'attente de la file que porte err, nil sinon.
func drainTimeoutOf(err error) *DrainTimeoutError {
	var dt *DrainTimeoutError
	if errors.As(err, &dt) {
		return dt
	}
	return nil
}

// proveAfterDrainTimeout relit l'ensemble connu de chaque joueur qui a listé des matchs : un match
// récupéré qui y figure est rangé (prouvé), un match soumis qui n'y figure pas est encore en file.
// Un joueur dont l'ensemble connu est illisible ne prouve rien : ses matchs soumis restent en file
// (ni comptés rangés, ni échecs), et l'état de son historique n'est pas touché (history_settle.go).
func (o *CycleOrchestratorImpl) proveAfterDrainTimeout(
	ctx context.Context,
	players []PlayerProfile,
	disc DiscoveryResult,
	fetched FetchSharedResult,
	dt *DrainTimeoutError,
) persistProof {
	proof := persistProof{timedOut: true, proven: map[string]bool{}, queued: map[string]bool{}}
	for _, p := range players {
		listed := disc.PerPlayer[p.PlayerSlug]
		if len(listed) == 0 || disc.Errors[p.PlayerSlug] != nil {
			continue
		}
		state, err := o.loader.LoadKnown(ctx, p)
		if err != nil {
			slog.WarnContext(ctx, "sync.v2: ensemble connu illisible après l'expiration de l'attente de la file — rangement non prouvé",
				"event", "sync.v2.persist.prove_failed", "player", p.PlayerSlug, "err", err)
			continue
		}
		for _, id := range listed {
			if _, ok := fetched.Matches[id]; ok && state.Known[id] {
				proof.proven[id] = true
			}
		}
	}
	for _, id := range dt.Submitted {
		if !proof.proven[id] {
			proof.queued[id] = true
		}
	}
	observability.IncCounterT(ctxkeys.TitleSlug(ctx), counterPersistDrainTimeout)
	slog.WarnContext(ctx, "sync.v2: attente de la file de persistance expirée — les matchs soumis finissent en arrière-plan, le cycle continue avec ceux déjà rangés",
		"event", "sync.v2.persist.drain_timeout", "waited_s", int(dt.Waited.Seconds()), "stalled", dt.Stalled, "submitted", len(dt.Submitted),
		"persisted", len(proof.proven), "still_queued", len(proof.queued), "parse_errors", dt.ParseErrors)
	return proof
}

// noteQueued ajoute aux avertissements de chaque joueur le nombre de ses matchs encore en file.
func noteQueued(per map[string]PlayerOutcome, players []PlayerProfile, disc DiscoveryResult, queued map[string]bool) {
	if len(queued) == 0 {
		return
	}
	for _, p := range players {
		n := 0
		for _, id := range disc.PerPlayer[p.PlayerSlug] {
			if queued[id] {
				n++
			}
		}
		if n == 0 {
			continue
		}
		out := per[p.PlayerSlug]
		out.Warnings = append(out.Warnings, fmt.Sprintf("persist: %d matchs encore dans la file d'écriture (attente expirée), rangés en arrière-plan", n))
		per[p.PlayerSlug] = out
	}
}
