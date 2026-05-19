package sharedprovider

// Direction décrit le sens d'une transition d'état observable de l'extérieur.
// Cardinalité bornée — sert d'étiquette aux compteurs expvar et aux clés
// de filtrage pour les Subscribers.
type Direction string

const (
	// DirectionPreSwapToRW est émis SYNCHRONIQUEMENT par AcquireWriter AVANT
	// la phase de drain et le swap effectif (state encore RO à ce moment).
	// Le Provider attend que TOUS les Subscribers retournent avant de
	// poursuivre.
	//
	// Cas d'usage critique : le pool joueur libère son ATTACH RO sur shared
	// (via Reopen des conns player+social) pour permettre au Provider de
	// faire OpenReadWrite sans "Unique file handle conflict".
	//
	// Les Subscribers DOIVENT être rapides et idempotents — un Subscriber
	// lent retarde tout le swap RW.
	DirectionPreSwapToRW Direction = "pre_swap_to_rw"

	// DirectionROToRW est émis quand un AcquireWriter passe RO → Draining → RW.
	// (Pas encore émis activement — réservé pour future observabilité.)
	DirectionROToRW Direction = "ro_to_rw"

	// DirectionRWToRO est émis quand un WriterHandle.Release passe RW → Reopening → RO.
	// C'est la transition qui intéresse le pool joueur : ses conns peuvent
	// re-attachShared pour servir les requêtes shared.X via la conn player.
	DirectionRWToRO Direction = "rw_to_ro"

	// DirectionErrorToRO est émis quand le retry loop async récupère depuis
	// StateError (reopen RO post-sync ayant échoué initialement). Les
	// Subscribers doivent re-attacher comme à RWToRO.
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
