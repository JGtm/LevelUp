package sharedprovider

// State représente l'état courant du provider vis-à-vis du handle DuckDB
// sous-jacent. Encodé en int32 pour permettre les load/store atomiques
// (cf. providerImpl.state).
type State int32

const (
	// StateRO — steady state nominal : handle ouvert en read_only, Get sert
	// directement la conn sous-jacente.
	StateRO State = iota

	// StateDraining — un sync a demandé un writer ; on attend que les Get en
	// vol terminent avant de fermer la conn RO. Les nouveaux Get attendent.
	// Introduit au commit 3.
	StateDraining

	// StateRW — handle ouvert en read_write, réservé au writer en cours.
	// Get attend la fin du swap. Introduit au commit 3.
	StateRW

	// StateReopening — release writer en cours : on ferme RW et on rouvre RO.
	// Get attend. Introduit au commit 3.
	StateReopening

	// StateError — la réouverture RO post-sync a échoué ; un retry loop
	// interne tente de récupérer. Get retourne ErrSwapFailed.
	// Introduit au commit 3.
	StateError

	// StateClosed — Close a été appelé ; Get et AcquireWriter retournent
	// ErrProviderClosed.
	StateClosed
)

// String retourne le nom canonique de l'état, utilisé comme clé pour les
// métriques expvar (cardinalité bornée — cf. ADR-0009).
func (s State) String() string {
	switch s {
	case StateRO:
		return "ro"
	case StateDraining:
		return "draining"
	case StateRW:
		return "rw"
	case StateReopening:
		return "reopening"
	case StateError:
		return "error"
	case StateClosed:
		return "closed"
	default:
		return "unknown"
	}
}

// allStates liste toutes les valeurs de State. Sert à pré-initialiser les
// compteurs expvar pour que toutes les clés apparaissent dans /debug/vars
// dès le boot (sinon seules les clés vues au moins une fois sont exposées).
var allStates = []State{
	StateRO,
	StateDraining,
	StateRW,
	StateReopening,
	StateError,
	StateClosed,
}
