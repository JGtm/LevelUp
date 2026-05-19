package sharedprovider

// Direction décrit le sens d'une transition d'état observable de l'extérieur.
// Cardinalité bornée — sert d'étiquette aux compteurs expvar et aux clés
// de filtrage pour les Subscribers.
type Direction string

const (
	// DirectionROToRW est émis quand un AcquireWriter passe RO → Draining → RW.
	DirectionROToRW Direction = "ro_to_rw"

	// DirectionRWToRO est émis quand un WriterHandle.Release passe RW → Reopening → RO.
	// C'est la transition qui intéresse le pool joueur : ses ATTACH RO sur shared
	// peuvent avoir été invalidés et il doit éventuellement purger ses conns idle.
	DirectionRWToRO Direction = "rw_to_ro"

	// DirectionErrorToRO est émis quand le retry loop async récupère depuis
	// StateError (reopen RO post-sync ayant échoué initialement).
	DirectionErrorToRO Direction = "error_to_ro"
)

// SwapEvent décrit une transition d'état terminée. Émis aux Subscribers
// après que la transition soit complète et que p.mu soit relâché — il est
// donc safe d'appeler Provider.Get/AcquireWriter depuis le callback.
type SwapEvent struct {
	Direction Direction
	From      State
	To        State
	Path      string
}

// Subscriber est invoqué de façon synchrone (sans tenir p.mu) à chaque
// SwapEvent. DOIT être rapide — pas d'I/O lourd, pas de query DuckDB —
// sinon ralentit le cycle de swap suivant et bloque les Get en attente.
//
// Cas d'usage typique : le pool joueur s'abonne pour purger ses conns idle
// dans la player DB après chaque DirectionRWToRO, garantissant que les
// ATTACH RO stales sont rejoués au prochain usage.
//
// Le Subscriber NE DOIT PAS appeler Subscribe ou Unsubscribe pendant son
// exécution — risque de deadlock sur subsMu.
type Subscriber func(SwapEvent)
