package sharedprovider

import "context"

// Direction décrit le sens d'une transition d'état observable de l'extérieur.
// Cardinalité bornée — sert d'étiquette aux compteurs expvar et aux clés
// de filtrage pour les Subscribers.
type Direction string

const (
	// DirectionPreSwapToRW est émis SYNCHRONIQUEMENT par AcquireWriter en
	// Phase 3, ENTRE la fermeture du handle Provider RO et OpenReadWrite.
	// State courant : Draining (la transition vers RW n'a pas encore eu lieu).
	//
	// Le timing précis est critique à cause de l'auto-attach DuckDB-Go : si
	// shared est ouvert quelque part dans le process, toute nouvelle conn
	// DuckDB l'auto-attache. Pour que Subscribers (pool) puissent Reopen
	// leurs conns player sans auto-attach, il faut que le handle Provider
	// soit DÉJÀ fermé. La notif arrive donc juste après ce close, juste
	// avant le OpenReadWrite.
	//
	// Cas d'usage critique : le pool joueur ferme pdb.Shared et Reopen
	// pdb.Player (sans auto-attach car file totalement libéré côté Provider).
	//
	// Les Subscribers DOIVENT être rapides et idempotents — exécutés sous
	// p.mu, ils NE DOIVENT PAS appeler Get/AcquireWriter (deadlock garanti).
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
//
// Sprint B1 commit 19 : ctx est passé pour propager l'event_id du caller du
// swap (typiquement sync.RunDelta) au callback pool. Permet de grep le
// timeline complet d'un swap : Provider notify → pool DETACH → Provider
// open RW → pool RW write → Provider notify back → pool re-attach.
// En pratique, ce ctx peut être context.Background() pour les transitions
// RWToRO/ErrorToRO (déclenchées par Release() sans ctx caller) ; dans ce
// cas l'event_id du caller initial AcquireWriter est capturé et propagé
// via ctxkeys.WithEventID (cf. providerImpl.releaseWriter).
type Subscriber func(ctx context.Context, evt SwapEvent)
