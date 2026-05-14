// Package service — career_live_metrics.go : compteurs expvar pour le flow
// live carrière. Variables exposées sous `/debug/vars` (cf. ADR 0009 :
// monitoring expvar stdlib).
//
// Granularité par cas pour permettre de tracer en prod :
//   - taux de cache hit (cache_hit / (cache_hit + live))
//   - taux d'erreur (fetch_error / total)
//   - taux de fallback DB (db_fallback / total)
//   - taux d'insert effectif (snapshot_inserted / live)
package service

import "expvar"

var (
	careerLiveProgressLive    = expvar.NewInt("career_live.progress.live")
	careerLiveProgressCache   = expvar.NewInt("career_live.progress.cache_hit")
	careerLiveProgressFail    = expvar.NewInt("career_live.progress.fetch_error")
	careerLiveCustomLive      = expvar.NewInt("career_live.customization.live")
	careerLiveCustomCache     = expvar.NewInt("career_live.customization.cache_hit")
	careerLiveCustomFail      = expvar.NewInt("career_live.customization.fetch_error")
	careerLiveInsertChanged   = expvar.NewInt("career_live.snapshot.inserted")
	careerLiveInsertSkipped   = expvar.NewInt("career_live.snapshot.skipped_no_delta")
	careerLiveDBFallback      = expvar.NewInt("career_live.fallback.db_row")
	careerLivePerFieldMerge   = expvar.NewInt("career_live.fallback.per_field")
	careerLiveEmptyResult     = expvar.NewInt("career_live.fallback.empty")
	careerLiveIdentityServed  = expvar.NewInt("career_live.identity.served")
	careerLiveIdentityMissing = expvar.NewInt("career_live.identity.missing")
	// Budget de latence : combien de requêtes home ont vu le live tronqué
	// par CareerLiveBudget (~2.5 s). Si élevé, signe que l'API Halo est
	// lente — la home reste rapide mais sert des données un peu moins
	// fraîches que prévu.
	careerLiveBudgetExceeded = expvar.NewInt("career_live.budget.exceeded")
	// Background refresh déclenchés après un budget exceeded.
	careerLiveBgRefresh = expvar.NewInt("career_live.bg.refresh_kicked")
)
