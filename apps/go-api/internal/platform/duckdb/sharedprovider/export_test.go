package sharedprovider

import "time"

// Helpers test-only exposés dans le scope `_test` du package (compilés
// uniquement quand `go test` est invoqué). Permettent aux tests black-box
// (`package sharedprovider_test`) de manipuler des champs internes sans
// élargir l'API publique.

// SetReadyTimeoutForTest configure le délai max d'attente d'un Get pendant
// un swap. Par défaut 30s — réduit à ~100ms en tests pour ne pas allonger
// inutilement la suite.
func SetReadyTimeoutForTest(p Provider, d time.Duration) {
	p.(*providerImpl).readyTimeout = d
}

// SetRetryBaseBackoffForTest configure le délai initial du retry loop reopen
// RO. Par défaut 1s (5 tentatives sur ~30s, aligné production). Réduit à
// ~50ms en tests pour valider la récupération sans attendre des secondes.
func SetRetryBaseBackoffForTest(p Provider, d time.Duration) {
	p.(*providerImpl).retryBaseBackoff = d
}

// SetFailNextReopenForTest arme un hook qui fait échouer le prochain appel
// à tryReopenROLocked sans tenter l'OpenReadOnly. Le flag est consommé après
// un seul échec (CompareAndSwap), permettant au retry loop de réussir
// ensuite — utile pour valider le contrat dégradé/recovery (T11).
func SetFailNextReopenForTest(p Provider, fail bool) {
	p.(*providerImpl).failNextReopen.Store(fail)
}
