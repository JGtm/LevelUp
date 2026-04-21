package assets

import "time"

// Metrics est l'interface d'observabilité pour le package assets.
// Implémentée par PrometheusMetrics (prod) et NoopMetrics (tests).
type Metrics interface {
	// IncHit incrémente le compteur de cache hits.
	IncHit(k Kind, src Source)
	// IncMiss incrémente le compteur de cache misses.
	IncMiss(k Kind)
	// IncFetchError incrémente le compteur d'erreurs de fetch distant.
	IncFetchError(k Kind)
	// IncIndexUnavailable incrémente le compteur d'accès à un index indisponible.
	IncIndexUnavailable()
	// IncIndexWriteDropped incrémente le compteur de jobs d'écriture droppés.
	IncIndexWriteDropped(k Kind)
	// IncIndexWriteOverflow incrémente le compteur de dépassements de capacité de la queue.
	IncIndexWriteOverflow()
	// ObserveLatency enregistre une latence de résolution.
	ObserveLatency(k Kind, src Source, d time.Duration)
}

// NoopMetrics est une implémentation no-op de Metrics (pour les tests et le mode dev).
type NoopMetrics struct{}

func (NoopMetrics) IncHit(_ Kind, _ Source)                          {}
func (NoopMetrics) IncMiss(_ Kind)                                   {}
func (NoopMetrics) IncFetchError(_ Kind)                             {}
func (NoopMetrics) IncIndexUnavailable()                             {}
func (NoopMetrics) IncIndexWriteDropped(_ Kind)                      {}
func (NoopMetrics) IncIndexWriteOverflow()                           {}
func (NoopMetrics) ObserveLatency(_ Kind, _ Source, _ time.Duration) {}
