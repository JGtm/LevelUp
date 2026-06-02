package pool

// metrics.go — compteurs expvar du cooldown 429/503 (namespace levelup.auth_pool.*).
// Exposés via /debug/vars (ADR 0009, pas de Prometheus). Pattern aligné sur
// skill_v2_metrics.go.

import "expvar"

var (
	cooldownsTotal         = expvar.NewInt("levelup.auth_pool.cooldowns_total")
	cooldowns429Total      = expvar.NewInt("levelup.auth_pool.cooldowns_429_total")
	cooldowns503Total      = expvar.NewInt("levelup.auth_pool.cooldowns_503_total")
	retryAfterHonoredTotal = expvar.NewInt("levelup.auth_pool.retry_after_honored_total")
	cooldownExtendedTotal  = expvar.NewInt("levelup.auth_pool.cooldown_extended_total")
	lastCooldownSeconds    = expvar.NewInt("levelup.auth_pool.last_cooldown_seconds")
)

// recordCooldownMetrics incrémente les compteurs cumulés à chaque déclenchement
// (ou extension) d'un cooldown 429/503.
func recordCooldownMetrics(statusCode int, honored bool) {
	cooldownsTotal.Add(1)
	switch statusCode {
	case 429:
		cooldowns429Total.Add(1)
	case 503:
		cooldowns503Total.Add(1)
	}
	if honored {
		retryAfterHonoredTotal.Add(1)
	}
}
