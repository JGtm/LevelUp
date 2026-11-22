// Package v2 — persist_drain.go : délai d'attente de la file de persistance à la fin d'un cycle, et
// l'erreur typée de son expiration.
package v2

import (
	"context"
	"fmt"
	"time"
)

// Délai d'attente de la file d'écriture à la fin de la Phase 5 : `base + parMatch × soumis`, borné
// par `max`. Contrat : la file persiste en série, à raison d'environ une seconde par match (mesure
// d'une passe de rattrapage de 200 matchs : 3,7 minutes, soit 1,1 seconde par match) ; `parMatch`
// en prend plus du triple, `base` couvre les lots minuscules et le démarrage du worker, `max` borne
// une attente qui ne finirait jamais (file bloquée) à vingt minutes. Au budget d'une passe de
// rattrapage (200 matchs) l'attente est de 14 minutes.
const (
	persistDrainBase     = 60 * time.Second
	persistDrainPerMatch = 4 * time.Second
	persistDrainMax      = 20 * time.Minute
)

// persistDrainTimeout rend le délai d'attente de la file pour `submitted` matchs soumis.
func persistDrainTimeout(submitted int) time.Duration {
	return scaledDrainTimeout(persistDrainBase, persistDrainPerMatch, persistDrainMax, submitted)
}

// scaledDrainTimeout est la formule du délai : base + parMatch × soumis, plafonné à ceiling.
func scaledDrainTimeout(base, perMatch, ceiling time.Duration, submitted int) time.Duration {
	return min(base+time.Duration(max(submitted, 0))*perMatch, ceiling)
}

// DrainTimeoutError dit que l'attente de la file de persistance a expiré : les matchs soumis sont
// durables (journal d'écriture) et la file finit seule en arrière-plan, mais le cycle ne sait pas
// lesquels sont déjà rangés. Le cycle le relit (ensemble connu) au lieu de tout abandonner.
type DrainTimeoutError struct {
	// Waited est le délai accordé à la file.
	Waited time.Duration
	// Submitted : les matchs effectivement soumis à la file (ni refusés au parsage, ni sans
	// récupérateur connu), dans un ordre quelconque.
	Submitted []string
	// ParseErrors : matchs récupérés mais refusés au parsage, jamais soumis.
	ParseErrors int
}

func (e *DrainTimeoutError) Error() string {
	return fmt.Sprintf("PersistCycle: drain expiré après %s (submitted=%d parse_errors=%d): %v",
		e.Waited, len(e.Submitted), e.ParseErrors, context.DeadlineExceeded)
}

// Unwrap garde la chaîne de context.DeadlineExceeded.
func (e *DrainTimeoutError) Unwrap() error { return context.DeadlineExceeded }
