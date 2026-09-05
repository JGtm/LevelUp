package filmproc

// solo_wait.go — LE SECOND REGIME DU VERROU SOLO : ATTENDRE SON TOUR, BORNE.
//
// AcquireSolo REFUSE tout de suite quand un autre decodage tient la machine — le bon
// comportement pour un outil d'operateur (`cmd/replay-build`) ou pour le post-sync (le match
// manque revient au cycle suivant). Une PASSE n'a pas ce luxe : l'enfant de backfill, l'ouvrier
// et le harnais d'equivalence (`cmd/replay-equiv`) doivent finir par traiter leur film, et un
// refus les ferait echouer sur un simple chevauchement. Ils attendent donc, jusqu'a une borne :
// au-dela, c'est le refus ordinaire, detenteur nomme (PLAN_CUISSON_PERF §3 D7).

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// soloWaitPoll : cadence des nouvelles tentatives. Plus court que le battement du detenteur,
// pour qu'un verrou rendu soit vu au plus tard une demi-seconde apres.
const soloWaitPoll = 500 * time.Millisecond

// AcquireSoloWait prend le verrou de decodage comme [AcquireSolo], mais attend jusqu'a `max`
// quand un autre decodage le tient. Rend [ErrDecodeBusy] (enveloppe, detenteur nomme) une fois
// la borne atteinte, ou l'erreur du contexte s'il est annule avant.
func AcquireSoloWait(ctx context.Context, cacheRoot, tool, matchID string, max time.Duration) (*SoloLock, error) {
	deadline := time.Now().Add(max)
	for {
		l, err := AcquireSolo(cacheRoot, tool, matchID)
		if err == nil || !errors.Is(err, ErrDecodeBusy) {
			return l, err
		}
		if !time.Now().Before(deadline) {
			return nil, fmt.Errorf("apres %s d'attente : %w", max, err)
		}
		select {
		case <-ctx.Done():
			// DEUX %w : l'appelant doit pouvoir tester l'annulation (context.Canceled,
			// context.DeadlineExceeded) ET le motif du refus (ErrDecodeBusy).
			return nil, fmt.Errorf("attente du verrou interrompue (%w) : %w", ctx.Err(), err)
		case <-time.After(soloWaitPoll):
		}
	}
}
