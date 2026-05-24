// Package analysis — math_safe.go : helpers de neutralisation des floats
// dangereux (NaN, +/-Inf) avant ecriture en JSON ou en DB.
//
// Contexte : un ratio Halo Infinite avec denominateur=0 (deaths=0 sur KDR,
// shots_fired=0 sur accuracy) produit NaN ou Inf en Go. Or json.Marshal
// rejette ces valeurs avec `json: unsupported value: NaN` — bug observe en
// prod sur les batches MatchBatch (cf. PLAN_FIX_SYNC_RELIABILITY_2026-05-24,
// Phase 4).
//
// Les helpers ici sont des algos purs (0 IO, 0 dependance) — testables sans
// setup. Ils doivent etre appeles au point de **production** des valeurs
// (BatchBuilder, perf_score compute, etc.), pas au moment du Marshal.
package analysis

import "math"

// IsBadFloat retourne true si f est NaN ou +/-Inf — valeurs interdites en
// JSON et qui produisent un panic ou un null silencieux selon le contexte.
func IsBadFloat(f float64) bool {
	return math.IsNaN(f) || math.IsInf(f, 0)
}

// SanitizeFloat retourne 0.0 si f est NaN ou +/-Inf, sinon f.
//
// Choix : 0.0 comme valeur neutre car la plupart des champs concernes sont
// des ratios (KDA, KDR, accuracy) ou un denominateur=0 signifie « pas de
// donnees significatives ». 0 est plus sur que NaN pour les calculs aval
// (moyennes, agregats SQL).
//
// Si tu veux distinguer « pas calcule » de « calcule=0 », utilise
// SanitizeNullableFloat qui retourne nil dans ce cas.
func SanitizeFloat(f float64) float64 {
	if IsBadFloat(f) {
		return 0.0
	}
	return f
}

// SanitizeNullableFloat retourne nil si la valeur pointee par p est NaN/Inf,
// si p est lui-meme nil, ou retourne p sinon.
//
// Cas d'usage : champs de domain row qui peuvent etre SQL NULL (ex:
// MatchParticipantRow.KDA = *float64). NaN devient NULL, ce qui est la
// semantique correcte (impossible a calculer).
func SanitizeNullableFloat(p *float64) *float64 {
	if p == nil {
		return nil
	}
	if IsBadFloat(*p) {
		return nil
	}
	return p
}

// SafeRatio calcule numerator/denominator en retournant 0 si denominator est
// proche de zero (|denom| < epsilon). Evite la production de NaN/Inf a la
// source plutot que d'avoir a sanitize apres.
//
// Equivalent de math.Inf-safe : utiliser ce helper systematiquement pour les
// ratios Halo (KDA, KDR, accuracy, etc.).
func SafeRatio(numerator, denominator float64) float64 {
	const epsilon = 1e-12
	if math.Abs(denominator) < epsilon {
		return 0.0
	}
	r := numerator / denominator
	if IsBadFloat(r) {
		return 0.0
	}
	return r
}
