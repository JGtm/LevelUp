// Package dbfault classe les pannes de lecture d'une base pour la couche HTTP : une base
// momentanément indisponible (bascule du fournisseur partagé, poignée fermée ou invalidée) se
// répond par un 503 que le client retente, jamais par un 500. La couche HTTP ne connaît ni le
// moteur de base ni ses paquets : elle interroge ces prédicats.
package dbfault

import (
	"errors"
	"strings"

	duckdbpkg "levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
)

// IsHandleClosedOrInvalidated reconnaît les erreurs de base qui justifient un 503 +
// Retry-After au lieu d'un 500. Couvre :
//   - `sql: database is closed` (poignée fermée, réouverture possible) ;
//   - `database has been invalidated...` (FATAL DuckDB, cf. duckdb.IsInvalidatedError).
//
// L'appelant retente dans quelques secondes, une fois la réouverture faite.
func IsHandleClosedOrInvalidated(err error) bool {
	if err == nil {
		return false
	}
	if duckdbpkg.IsInvalidatedError(err) {
		return true
	}
	return strings.Contains(err.Error(), "database is closed")
}

// IsSharedSwapContention reconnaît les erreurs du fournisseur de la base partagée qui signalent
// qu'elle est momentanément indisponible car une synchronisation tient l'écrivain (bascule en
// cours ou fournisseur en récupération). La lecture est idempotente et repassera dès le retour
// en lecture seule.
func IsSharedSwapContention(err error) bool {
	return errors.Is(err, sharedprovider.ErrSwapTimeout) ||
		errors.Is(err, sharedprovider.ErrSwapFailed) ||
		errors.Is(err, sharedprovider.ErrProviderClosed)
}

// IsFileLock reconnaît la contention fichier : un AUTRE détenteur tient la base (autre process, handle pas
// encore libérée). Transitoire : l'appelant la compte à part d'une panne.
func IsFileLock(err error) bool {
	return duckdbpkg.IsFileLockError(err)
}
