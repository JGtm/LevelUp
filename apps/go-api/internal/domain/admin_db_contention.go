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
	// Étape 0 attribution — fenêtre RW stricte (durée de détention du writer) :
	// agrégat + ventilation PAR DÉTENTEUR (label posé au call-site), triée
	// total_ms décroissant, + watchdog (détentions au-delà du seuil ~2s).
	AvgRWWindowMs int64                `json:"avg_rw_window_ms"` // détention writer moyenne
	MaxRWWindowMs int64                `json:"max_rw_window_ms"` // pic de détention writer
	WatchdogFired int64                `json:"watchdog_fired"`   // détentions > seuil (WARN émis)
	Holders       []DBContentionHolder `json:"holders"`          // ventilation par détenteur
}

// DBContentionHolder est la ventilation de la fenêtre RW pour UN détenteur du
// writer (label constant posé via ctxkeys.WithDBWriterLabel au call-site).
type DBContentionHolder struct {
	Label         string `json:"label"`          // ex: sync_v2_postsync, persist_worker
	Count         int64  `json:"count"`          // nb de détentions mesurées
	TotalMs       int64  `json:"total_ms"`       // temps cumulé de détention
	AvgMs         int64  `json:"avg_ms"`         // détention moyenne
	MaxMs         int64  `json:"max_ms"`         // pic de détention
	WatchdogFired int64  `json:"watchdog_fired"` // détentions > seuil pour ce label
}
