// Package domain — admin_db_contention.go : payload du dashboard admin
// « Contention DB (sync) ».
//
// Reflète les compteurs du sharedprovider B-swap (lecture seule des métriques
// expvar). Permet de diagnostiquer si le swap RO↔RW pendant le sync est un
// problème de cadence (beaucoup de bascules / lectures rejetées) ou non.
package domain

// DBContentionResponse est la réponse de GET /admin/db-contention.
type DBContentionResponse struct {
	GeneratedAt   string `json:"generated_at"`   // RFC3339
	State         string `json:"state"`          // ro | draining | rw | reopening | error | closed
	Swaps         int64  `json:"swaps"`          // écritures shared (bascules RO→RW) depuis le boot
	AvgAcquireMs  int64  `json:"avg_acquire_ms"` // durée moyenne d'une bascule RO→RW (drain inclus)
	AvgReleaseMs  int64  `json:"avg_release_ms"` // durée moyenne d'une bascule RW→RO
	DrainMsTotal  int64  `json:"drain_ms_total"` // temps cumulé d'attente du drain des lecteurs
	ReadsRejected int64  `json:"reads_rejected"` // lectures rejetées en 503 pendant un swap
	ReadersInUse  int64  `json:"readers_in_use"` // lecteurs en vol à l'instant T
	SwapFailures  int64  `json:"swap_failures"`  // échecs de swap (reopen/acquire/panic)
	AvgBlockedMs  int64  `json:"avg_blocked_ms"` // blocage lecteurs moyen par swap (drain+RW+reopen)
	MaxBlockedMs  int64  `json:"max_blocked_ms"` // pic de blocage lecteurs
}
