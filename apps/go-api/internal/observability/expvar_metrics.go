// Package observability fournit le squelette minimal de metriques expvar
// pour LevelUp.
//
// Conformement au meta-plan PLAN_META_FOUNDATIONS_GO § 4.7 : pas de Prometheus
// / OpenTelemetry pour cette iteration. On expose 4 categories de metriques de
// base via expvar (stdlib) sur l'endpoint /debug/vars :
//
//	service_duration_ms{service,status}  -- moyenne + max + sum + count
//	repo_query_duration_ms{query}        -- meme structure
//	cache_hit_ratio{cache}               -- compteurs hits + misses
//	error_count{service}                 -- compteur simple
//
// Les helpers sont threadsafe (atomic.Int64 + sync.Map). expvar.Map publie
// chaque metrique sous la cle "levelup" du JSON renvoye par /debug/vars.
package observability

import (
	"expvar"
	"sync"
	"sync/atomic"
)

// metricsMap est le namespace expvar racine. Initialise au package init().
var metricsMap *expvar.Map

// counters stocke les compteurs simples (atomic.Int64). Cle = nom de metrique.
var counters sync.Map

// durations stocke les agregats de durees (count + sum + max).
var durations sync.Map

func init() {
	metricsMap = expvar.NewMap("levelup")
}

// PublishDuckDBPoolStats enregistre sous "levelup"/"duckdb_pool_stats" une metrique
// expvar dont la valeur (sql.DBStats par handle) est produite a la demande par
// snapshot(). Le snapshot est injecte (func() any) pour eviter un cycle
// observability -> platform/duckdb. J1 (2026-07-05, ADR 0009) : rend visibles
// WaitCount/WaitDuration/InUse du pool sur /debug/vars.
func PublishDuckDBPoolStats(snapshot func() any) {
	if metricsMap == nil || snapshot == nil {
		return
	}
	metricsMap.Set("duckdb_pool_stats", expvar.Func(snapshot))
}

// PublishDuckDBBudgets enregistre sous "levelup"/"duckdb_budgets" les bornes
// ressources DuckDB effectives (memory_limit/threads/tailles de pool). J2 : rend
// la config mémoire/threads réellement appliquée visible sur /debug/vars, à côté
// de duckdb_pool_stats. snapshot() injecté (anti-cycle observability -> duckdb).
func PublishDuckDBBudgets(snapshot func() any) {
	if metricsMap == nil || snapshot == nil {
		return
	}
	metricsMap.Set("duckdb_budgets", expvar.Func(snapshot))
}

// IncCounter incremente de 1 le compteur nomme. Cree le compteur a la
// premiere appel.
//
// Convention de nommage : "<categorie>_<sous_cle>" en snake_case, ex.
// "cache_hit_player_matches", "error_count_squad_service".
func IncCounter(name string) {
	AddInt(name, 1)
}

// AddInt augmente le compteur nomme de delta (peut etre negatif).
func AddInt(name string, delta int64) {
	if v, ok := counters.Load(name); ok {
		v.(*atomic.Int64).Add(delta)
		return
	}
	c := &atomic.Int64{}
	actual, loaded := counters.LoadOrStore(name, c)
	actual.(*atomic.Int64).Add(delta)
	if !loaded {
		// Premiere creation : publier le compteur dans le map expvar.
		key := name
		metricsMap.Set(key, expvar.Func(func() any {
			if v, ok := counters.Load(key); ok {
				return v.(*atomic.Int64).Load()
			}
			return int64(0)
		}))
	}
}

// SetInt fixe la valeur du compteur nomme (semantique gauge : etat du dernier
// run, pas un cumul). Ex. "invariants_fail_last".
func SetInt(name string, value int64) {
	if v, ok := counters.Load(name); ok {
		v.(*atomic.Int64).Store(value)
		return
	}
	c := &atomic.Int64{}
	actual, loaded := counters.LoadOrStore(name, c)
	actual.(*atomic.Int64).Store(value)
	if !loaded {
		key := name
		metricsMap.Set(key, expvar.Func(func() any {
			if v, ok := counters.Load(key); ok {
				return v.(*atomic.Int64).Load()
			}
			return int64(0)
		}))
	}
}

// LoadCounter retourne la valeur courante du compteur nomme (0 si absent).
// Threadsafe ; utile pour tests et snapshots.
func LoadCounter(name string) int64 {
	if v, ok := counters.Load(name); ok {
		return v.(*atomic.Int64).Load()
	}
	return 0
}

// durationStats agrege les mesures de duree (count + sum + max).
type durationStats struct {
	count atomic.Int64
	sumMS atomic.Int64
	maxMS atomic.Int64
}

// RecordDurationMS enregistre une mesure de duree en ms pour la metrique nommee.
// Garde count + sum + max ; une moyenne est calculee a la lecture.
//
// Les valeurs negatives sont ignorees (defensive). Pas d'histogramme complet
// -- pour avoir des p95/p99, il faudra installer Prometheus plus tard.
func RecordDurationMS(name string, ms int64) {
	if ms < 0 {
		return
	}
	stats := getOrCreateDurationStats(name)
	stats.count.Add(1)
	stats.sumMS.Add(ms)
	for {
		cur := stats.maxMS.Load()
		if ms <= cur {
			break
		}
		if stats.maxMS.CompareAndSwap(cur, ms) {
			break
		}
	}
}

func getOrCreateDurationStats(name string) *durationStats {
	if v, ok := durations.Load(name); ok {
		return v.(*durationStats)
	}
	s := &durationStats{}
	actual, loaded := durations.LoadOrStore(name, s)
	stats := actual.(*durationStats)
	if !loaded {
		key := name
		metricsMap.Set(key, expvar.Func(func() any {
			if v, ok := durations.Load(key); ok {
				s := v.(*durationStats)
				cnt := s.count.Load()
				sum := s.sumMS.Load()
				var avg int64
				if cnt > 0 {
					avg = sum / cnt
				}
				return map[string]int64{
					"count":  cnt,
					"sum_ms": sum,
					"avg_ms": avg,
					"max_ms": s.maxMS.Load(),
				}
			}
			return nil
		}))
	}
	return stats
}

// LoadDurationStats retourne un snapshot (count, sum, avg, max) en ms pour
// la metrique nommee. Tous zero si la metrique n'a jamais ete enregistree.
func LoadDurationStats(name string) (count, sumMS, avgMS, maxMS int64) {
	v, ok := durations.Load(name)
	if !ok {
		return 0, 0, 0, 0
	}
	s := v.(*durationStats)
	count = s.count.Load()
	sumMS = s.sumMS.Load()
	maxMS = s.maxMS.Load()
	if count > 0 {
		avgMS = sumMS / count
	}
	return
}

// Reset remet a zero toutes les metriques. Reserve aux tests : ne pas
// utiliser en production (on perd l'historique du process).
func Reset() {
	counters.Range(func(k, _ any) bool {
		counters.Delete(k)
		return true
	})
	durations.Range(func(k, _ any) bool {
		durations.Delete(k)
		return true
	})
	metricsMap.Init()
}
