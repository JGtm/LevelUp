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

// SetDrainTimeoutForTest configure le délai max d'attente du drain des
// readers en vol pendant la phase 2 d'AcquireWriter. Par défaut 5s — réduit
// à ~200ms en tests pour valider rapidement le rollback sur drain expiré.
func SetDrainTimeoutForTest(p Provider, d time.Duration) {
	p.(*providerImpl).drainTimeout = d
}

// SetRWHoldWatchdogForTest configure le seuil du watchdog de détention du
// writer RW. Par défaut 2s (defaultRWHoldWatchdog) — réduit à ~50ms en tests
// pour valider le fire sans allonger la suite. À appeler AVANT AcquireWriter
// (le timer s'arme à l'acquisition). d <= 0 désactive le watchdog.
func SetRWHoldWatchdogForTest(p Provider, d time.Duration) {
	p.(*providerImpl).rwHoldWatchdog = d
}

// SetSlowSwapThresholdForTest configure le seuil au-delà duquel une phase du
// cycle B-swap est journalisée en INFO au lieu de DEBUG. Par défaut 2s
// (defaultSlowSwapThreshold) — abaissé à ~1ns en tests pour qualifier n'importe
// quel cycle de « lent » sans devoir en fabriquer un vraiment long. d <= 0
// désactive la remontée INFO (tout le nominal reste en DEBUG).
func SetSlowSwapThresholdForTest(p Provider, d time.Duration) {
	p.(*providerImpl).slowSwapThreshold = d
}
