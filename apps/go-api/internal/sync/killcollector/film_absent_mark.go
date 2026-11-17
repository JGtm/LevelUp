package killcollector

// film_absent_mark.go — l'entrée exportée du marqueur terminal « film absent » (MBitFilmAbsent) pour un appelant
// qui constate lui-même qu'un film n'existe plus (la génération à la demande d'un rejeu). La requête, le bit et le
// lease sont ceux de la passe film ([marquerRegistre]) : aucune écriture neuve.

import (
	"context"
	"errors"
	"fmt"

	"levelup/go-api/internal/persist"
	"levelup/go-api/internal/sync/matchflags"
)

// ErrNoSharedWriter : aucun écrivain de la base partagée n'est disponible (acquire nul).
var ErrNoSharedWriter = errors.New("killcollector: aucun écrivain de la base partagée")

// MarquerFilmAbsent pose MBitFilmAbsent sur le match matchID (identifiant COMPLET du registre), sous un segment
// d'écriture court pris par acquire (lease RW, ADR 0013). Idempotent, ne touche aucun autre bit ; un match absent du
// registre n'est pas une erreur (l'UPDATE ne trouve aucune ligne).
func MarquerFilmAbsent(ctx context.Context, acquire persist.SharedWriterFn, matchID string) error {
	if acquire == nil {
		return ErrNoSharedWriter
	}
	db, release, err := acquire(ctx)
	if err != nil {
		return fmt.Errorf("marqueur film absent (%s): écrivain indisponible: %w", matchID, err)
	}
	defer release()
	return marquerRegistre(ctx, db, matchID, matchflags.MBitFilmAbsent)
}
