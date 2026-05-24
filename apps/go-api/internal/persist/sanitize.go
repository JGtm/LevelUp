// Package persist — sanitize.go : neutralise les floats dangereux (NaN, +/-Inf)
// dans un MatchBatch avant le json.Marshal du WAL.
//
// Phase 4 du PLAN_FIX_SYNC_RELIABILITY_2026-05-24. Contexte :
//
// Un ratio Halo Infinite avec denominateur=0 (deaths=0 sur KDR, shots_fired=0
// sur accuracy) produit NaN ou Inf en Go. Or json.Marshal rejette ces valeurs
// avec `json: unsupported value: NaN` — bug observe en prod 2026-05-24 sur
// 2 matchs/cycle de Chocoboflor + 1 match XxDaemonGamerxX, droppes
// silencieusement avant insertion DB.
//
// Strategie : sanitize defensif au point de soumission a la queue (Submit())
// pour garantir que tout batch ecrit dans le WAL est marshal-able. Les
// helpers underlying sont dans internal/analysis (algos purs sans dependance
// DB, testes via 15 sous-tests en math_safe_test.go).
//
// Couches de protection :
//
//  1. Source ideale : le builder devrait produire des valeurs sanitize via
//     analysis.SafeRatio() lors du calcul (cf. accuracy/KDA computes). A
//     enforcer progressivement par PR dedie.
//  2. Frontiere defensive (ce fichier) : sanitize tout batch avant marshal.
//     Garantit l'invariant "tout WAL est valide" meme si layer 1 oublie un site.
//
// Couvert par TestMatchBatch_Marshal_FailsOnNaN (sentinelle bug) et
// TestMatchBatch_Marshal_SafeAfterSanitize (preuve qu'apres sanitize, marshal
// reussit).

package persist

import (
	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/domain"
)

// SanitizeBatch parcourt tous les champs flottants d'un MatchBatch et
// neutralise les valeurs NaN et +/-Inf qui feraient echouer json.Marshal.
//
// Strategie par type de champ :
//   - *float64 (nullable, ex: KDA, Accuracy) → nil si NaN/Inf (semantique
//     "non calculable") via analysis.SanitizeNullableFloat.
//   - float64 (non-nullable, ex: LUSRComponent.Value) → 0.0 si NaN/Inf via
//     analysis.SanitizeFloat (l'absence de valeur n'est pas representable,
//     donc on tombe sur le neutre).
//
// Idempotent : ré-appeler sur un batch deja sanitize est un no-op.
// Sans effet de bord visible si batch est nil.
func SanitizeBatch(batch *MatchBatch) {
	if batch == nil {
		return
	}

	// Shared.Match : MatchIntensity (float ratio) — peut etre NaN si la
	// formule (kills+deaths)/duration calculee avec duration=0.
	if batch.Shared.Match != nil {
		sanitizeMatchRegistryRow(batch.Shared.Match)
	}

	// Shared.Participants : KDA, Accuracy, KillsExpected, DeathsExpected,
	// KillsStddev, DeathsStddev, TeamMMR, EnemyMMR, DamageDealt, DamageTaken,
	// AvgLifeSeconds — tous *float64 nullables.
	for i := range batch.Shared.Participants {
		sanitizeMatchParticipantRow(&batch.Shared.Participants[i])
	}

	// Shared.MatchCSRs : RatingValue, RatingDelta (parfois transmis avec
	// NaN si le skill endpoint Halo renvoie un cas degenere).
	for i := range batch.Shared.MatchCSRs {
		batch.Shared.MatchCSRs[i].RatingValue = analysis.SanitizeNullableFloat(batch.Shared.MatchCSRs[i].RatingValue)
		batch.Shared.MatchCSRs[i].RatingDelta = analysis.SanitizeNullableFloat(batch.Shared.MatchCSRs[i].RatingDelta)
	}

	// PlayerData.Enrichment : performance_score + engagement_* + paces.
	if batch.PlayerData.Enrichment != nil {
		sanitizeEnrichmentRow(batch.PlayerData.Enrichment)
	}

	// PlayerData.SkillRank : rating_value, rating_deviation, rating_delta.
	if batch.PlayerData.SkillRank != nil {
		batch.PlayerData.SkillRank.RatingValue = analysis.SanitizeNullableFloat(batch.PlayerData.SkillRank.RatingValue)
		batch.PlayerData.SkillRank.RatingDeviation = analysis.SanitizeNullableFloat(batch.PlayerData.SkillRank.RatingDeviation)
		batch.PlayerData.SkillRank.RatingDelta = analysis.SanitizeNullableFloat(batch.PlayerData.SkillRank.RatingDelta)
	}

	// PlayerData.LUSRComponents : Value + Weight (float64 directs).
	// NaN dans une composante LUSR signifie qu'elle n'a pas contribue —
	// on ramene a 0 (neutre additif).
	for i := range batch.PlayerData.LUSRComponents {
		batch.PlayerData.LUSRComponents[i].Value = analysis.SanitizeFloat(batch.PlayerData.LUSRComponents[i].Value)
		batch.PlayerData.LUSRComponents[i].Weight = analysis.SanitizeFloat(batch.PlayerData.LUSRComponents[i].Weight)
	}
}

// sanitizeMatchRegistryRow neutralise les floats de match_registry.
func sanitizeMatchRegistryRow(r *domain.MatchRegistryRow) {
	r.MatchIntensity = analysis.SanitizeNullableFloat(r.MatchIntensity)
}

// sanitizeMatchParticipantRow neutralise les floats du participant.
// Liste alignee sur domain/match_rows.go champs *float64.
func sanitizeMatchParticipantRow(p *domain.MatchParticipantRow) {
	p.DamageDealt = analysis.SanitizeNullableFloat(p.DamageDealt)
	p.DamageTaken = analysis.SanitizeNullableFloat(p.DamageTaken)
	p.KDA = analysis.SanitizeNullableFloat(p.KDA)
	p.Accuracy = analysis.SanitizeNullableFloat(p.Accuracy)
	p.AvgLifeSeconds = analysis.SanitizeNullableFloat(p.AvgLifeSeconds)
	p.KillsExpected = analysis.SanitizeNullableFloat(p.KillsExpected)
	p.DeathsExpected = analysis.SanitizeNullableFloat(p.DeathsExpected)
	p.KillsStddev = analysis.SanitizeNullableFloat(p.KillsStddev)
	p.DeathsStddev = analysis.SanitizeNullableFloat(p.DeathsStddev)
	p.TeamMMR = analysis.SanitizeNullableFloat(p.TeamMMR)
	p.EnemyMMR = analysis.SanitizeNullableFloat(p.EnemyMMR)
}

// sanitizeEnrichmentRow neutralise les floats de player_match_enrichment.
func sanitizeEnrichmentRow(e *EnrichmentRow) {
	e.PerformanceScore = analysis.SanitizeNullableFloat(e.PerformanceScore)
	e.EngagementScore = analysis.SanitizeNullableFloat(e.EngagementScore)
	e.EngagementScoreBrut = analysis.SanitizeNullableFloat(e.EngagementScoreBrut)
	e.EngagementPacePlayer = analysis.SanitizeNullableFloat(e.EngagementPacePlayer)
	e.EngagementPaceTeam = analysis.SanitizeNullableFloat(e.EngagementPaceTeam)
	e.EngagementPaceLobby = analysis.SanitizeNullableFloat(e.EngagementPaceLobby)
}
