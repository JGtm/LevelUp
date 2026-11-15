package sharedprovider

import (
	"context"
	"database/sql"
	"time"
)

// ReadSegmentFunc prête un segment de LECTURE de la base partagée : fn s'exécute sous un lecteur du
// fournisseur, rendu au retour. Il n'appelle pas fn quand la base est indisponible (l'indisponibilité est
// signalée à l'appelant de ReadSegment).
type ReadSegmentFunc func(ctx context.Context, step string, fn func(sharedDB *sql.DB))

// ReadSegment construit le segment de lecture d'un fournisseur ; nil quand p est nil. L'attente d'un swap est
// bornée par budget (WithSwapWaitBudget). Quand le lecteur est indisponible, unavailable reçoit l'étape et
// l'erreur, et fn n'est pas appelé.
func ReadSegment(p Provider, budget time.Duration, unavailable func(ctx context.Context, step string, err error)) ReadSegmentFunc {
	if p == nil {
		return nil
	}
	return func(ctx context.Context, step string, fn func(sharedDB *sql.DB)) {
		db, release, err := p.Get(WithSwapWaitBudget(ctx, budget))
		if err != nil {
			unavailable(ctx, step, err)
			return
		}
		defer release()
		fn(db)
	}
}
