// Package invariants — contrats de données déclarés du pipeline de sync.
//
// Chaque invariant est une règle vérifiable par requêtes SQL sur (playerDB,
// sharedDB) pour un joueur donné. Le package est volontairement minimal
// (database/sql + context, zéro dépendance interne) pour être consommable par :
//
//  1. le test d'intégration multi-joueurs (gate au commit, CI tag integration) —
//     cf. internal/sync/invariants_gate_integration_test.go ;
//  2. à terme, une sentinelle runtime (généralisation de RunDualRowSentinel)
//     branchée sur les MÊMES définitions — une seule source de vérité.
//
// Sévérités :
//   - SeverityFail : violation d'un contrat dur — le gate DOIT échouer.
//   - SeverityWarn : dérive tolérée/documentée (cold-start, trous catalogue) —
//     reportée mais non bloquante.
//
// Historique des incidents couverts : delta-skip sans enrichment (2026-05-27 et
// 2026-06-10), orphelins registry/participants (corruption ART), pair_name UUID
// (trou catalogue 2026-06-09), sessions absentes.
package invariants

import (
	"context"
	"database/sql"
	"fmt"
)

// Severity d'une violation.
type Severity string

const (
	SeverityFail Severity = "fail"
	SeverityWarn Severity = "warn"
)

const sampleCap = 5

// Violation décrit un invariant violé pour un joueur.
type Violation struct {
	// Key stable (snake_case) — corrèle logs, tests et dashboards.
	Key      string
	Severity Severity
	// Count = nombre d'entités en violation (matchs, rows...).
	Count int
	// Sample = jusqu'à 5 identifiants pour diagnostic direct.
	Sample []string
	// Description humaine du contrat violé.
	Description string
}

func (v Violation) String() string {
	return fmt.Sprintf("[%s] %s: count=%d sample=%v — %s", v.Severity, v.Key, v.Count, v.Sample, v.Description)
}

// Report agrège les violations d'un joueur.
type Report struct {
	XUID       string
	Violations []Violation
}

// Failures retourne uniquement les violations de sévérité fail.
func (r Report) Failures() []Violation {
	out := make([]Violation, 0, len(r.Violations))
	for _, v := range r.Violations {
		if v.Severity == SeverityFail {
			out = append(out, v)
		}
	}
	return out
}

// CheckPlayer exécute tous les invariants déclarés pour un joueur.
// playerDB = stats.duckdb du joueur ; sharedDB = shared_matches_v2.duckdb
// (RO suffit). Les erreurs SQL sont remontées (un invariant invérifiable est
// un échec du harnais, pas un succès silencieux).
func CheckPlayer(ctx context.Context, playerDB, sharedDB *sql.DB, xuid string) (Report, error) {
	rep := Report{XUID: xuid}

	checks := []func(context.Context, *sql.DB, *sql.DB, string) (*Violation, error){
		// FAIL — contrats durs.
		checkEnrichmentMissing,
		checkParticipantsWithoutRegistry,
		checkRegistryWithoutParticipants,
		checkMedalsWithoutRegistry,
		checkLUSRV2Orphan,
		// WARN — dérives tolérées/documentées, à surveiller en tendance.
		checkSessionMissing,
		checkPerformanceScoreMissing,
		checkSkillRankMissing,
		checkCitationsMissing,
		checkPersonalScoreAwardsMissing,
		checkXuidAliasMissing,
		checkPairNameUUID,
	}
	for _, check := range checks {
		v, err := check(ctx, playerDB, sharedDB, xuid)
		if err != nil {
			return rep, err
		}
		if v != nil {
			rep.Violations = append(rep.Violations, *v)
		}
	}
	return rep, nil
}

// ─── I1 — enrichment_missing (FAIL) ─────────────────────────────────────────
//
// Contrat : tout match présent dans shared.match_participants pour ce xuid a
// une row player_match_enrichment dans la player DB. C'est LE contrat du
// delta-skip cross-player (loadKnownMatchIDs source 2 + ensurePlayerEnrichmentRows) :
// un match peut être inséré en shared par le sync d'un coéquipier, mais la
// convergence DOIT créer la row enrichment du joueur courant.
// Incidents : 2026-05-27 (Madina/Choco/XxDaemon), 2026-06-10 (session du 09/06).
func checkEnrichmentMissing(ctx context.Context, playerDB, sharedDB *sql.DB, xuid string) (*Violation, error) {
	enriched, err := collectIDs(ctx, playerDB, `SELECT match_id FROM player_match_enrichment`)
	if err != nil {
		return nil, fmt.Errorf("invariants/enrichment_missing: player query: %w", err)
	}
	enrichedSet := make(map[string]struct{}, len(enriched))
	for _, id := range enriched {
		enrichedSet[id] = struct{}{}
	}
	sharedIDs, err := collectIDs(ctx, sharedDB,
		`SELECT DISTINCT match_id FROM match_participants WHERE xuid || '' = ? AND match_id IS NOT NULL`, xuid)
	if err != nil {
		return nil, fmt.Errorf("invariants/enrichment_missing: shared query: %w", err)
	}
	var missing []string
	for _, id := range sharedIDs {
		if _, ok := enrichedSet[id]; !ok {
			missing = append(missing, id)
		}
	}
	if len(missing) == 0 {
		return nil, nil
	}
	return &Violation{
		Key:         "enrichment_missing",
		Severity:    SeverityFail,
		Count:       len(missing),
		Sample:      capSample(missing),
		Description: "matchs en shared.match_participants sans row player_match_enrichment (delta-skip non convergé)",
	}, nil
}

// ─── I5 — participants_without_registry (FAIL) ──────────────────────────────
//
// Contrat : toute row match_participants référence un match_registry existant.
// Un orphelin = écriture partielle (classe d'incidents corruption ART).
func checkParticipantsWithoutRegistry(ctx context.Context, _ *sql.DB, sharedDB *sql.DB, _ string) (*Violation, error) {
	ids, err := collectIDs(ctx, sharedDB, `
		SELECT DISTINCT p.match_id
		FROM match_participants p
		LEFT JOIN match_registry r ON r.match_id = p.match_id
		WHERE r.match_id IS NULL`)
	if err != nil {
		return nil, fmt.Errorf("invariants/participants_without_registry: %w", err)
	}
	if len(ids) == 0 {
		return nil, nil
	}
	return &Violation{
		Key:         "participants_without_registry",
		Severity:    SeverityFail,
		Count:       len(ids),
		Sample:      capSample(ids),
		Description: "rows match_participants orphelines (match_id absent de match_registry)",
	}, nil
}

// ─── I4 — session_missing (WARN) ────────────────────────────────────────────
//
// Contrat : après post-sync, toute row enrichment porte un session_id.
// WARN (pas FAIL) : une fenêtre transitoire existe entre l'ensure-rows et
// l'étape sessions du même pipeline si celui-ci est interrompu.
func checkSessionMissing(ctx context.Context, playerDB, _ *sql.DB, _ string) (*Violation, error) {
	ids, err := collectIDs(ctx, playerDB,
		`SELECT match_id FROM player_match_enrichment WHERE session_id IS NULL`)
	if err != nil {
		return nil, fmt.Errorf("invariants/session_missing: %w", err)
	}
	if len(ids) == 0 {
		return nil, nil
	}
	return &Violation{
		Key:         "session_missing",
		Severity:    SeverityWarn,
		Count:       len(ids),
		Sample:      capSample(ids),
		Description: "rows enrichment sans session_id (étape sessions du post-sync non passée)",
	}, nil
}

// ─── I2 — performance_score_missing (WARN) ──────────────────────────────────
//
// Contrat souple : les scores de performance convergent via
// batchComputePerformanceScores. Le cold-start par chaîne (< 10 matchs dans la
// chaîne) laisse légitimement des NULL → WARN seulement. Un Count qui croît
// au fil des cycles signale en revanche un batch qui ne converge plus.
func checkPerformanceScoreMissing(ctx context.Context, playerDB, _ *sql.DB, _ string) (*Violation, error) {
	ids, err := collectIDs(ctx, playerDB,
		`SELECT match_id FROM player_match_enrichment WHERE performance_score IS NULL`)
	if err != nil {
		return nil, fmt.Errorf("invariants/performance_score_missing: %w", err)
	}
	if len(ids) == 0 {
		return nil, nil
	}
	return &Violation{
		Key:         "performance_score_missing",
		Severity:    SeverityWarn,
		Count:       len(ids),
		Sample:      capSample(ids),
		Description: "rows enrichment sans performance_score (cold-start toléré, croissance = batch en panne)",
	}, nil
}

// ─── I6 — pair_name_uuid (WARN) ─────────────────────────────────────────────
//
// Contrat : match_registry.pair_name est un libellé humain, jamais un UUID brut
// (trou catalogue assets — observé sur les 12 matchs du 2026-06-09 au soir).
func checkPairNameUUID(ctx context.Context, _ *sql.DB, sharedDB *sql.DB, _ string) (*Violation, error) {
	ids, err := collectIDs(ctx, sharedDB, `
		SELECT match_id FROM match_registry
		WHERE pair_name IS NOT NULL
		  AND regexp_matches(pair_name, '^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$')`)
	if err != nil {
		return nil, fmt.Errorf("invariants/pair_name_uuid: %w", err)
	}
	if len(ids) == 0 {
		return nil, nil
	}
	return &Violation{
		Key:         "pair_name_uuid",
		Severity:    SeverityWarn,
		Count:       len(ids),
		Sample:      capSample(ids),
		Description: "match_registry.pair_name est un UUID brut (catalogue assets non résolu)",
	}, nil
}

// ─── I7 — registry_without_participants (FAIL) ──────────────────────────────
//
// Contrat : tout match_registry a au moins une row match_participants.
// Un registre sans participants = écriture partielle (classe ART, miroir de I5).
func checkRegistryWithoutParticipants(ctx context.Context, _ *sql.DB, sharedDB *sql.DB, _ string) (*Violation, error) {
	ids, err := collectIDs(ctx, sharedDB, `
		SELECT r.match_id
		FROM match_registry r
		LEFT JOIN match_participants p ON p.match_id = r.match_id
		WHERE p.match_id IS NULL`)
	if err != nil {
		return nil, fmt.Errorf("invariants/registry_without_participants: %w", err)
	}
	if len(ids) == 0 {
		return nil, nil
	}
	return &Violation{
		Key:         "registry_without_participants",
		Severity:    SeverityFail,
		Count:       len(ids),
		Sample:      capSample(ids),
		Description: "rows match_registry sans aucun participant (écriture partielle)",
	}, nil
}

// ─── I8 — medals_without_registry (FAIL) ────────────────────────────────────
//
// Contrat : toute row medals_earned référence un match_registry existant.
func checkMedalsWithoutRegistry(ctx context.Context, _ *sql.DB, sharedDB *sql.DB, _ string) (*Violation, error) {
	ids, err := collectIDs(ctx, sharedDB, `
		SELECT DISTINCT m.match_id
		FROM medals_earned m
		LEFT JOIN match_registry r ON r.match_id = m.match_id
		WHERE r.match_id IS NULL`)
	if err != nil {
		return nil, fmt.Errorf("invariants/medals_without_registry: %w", err)
	}
	if len(ids) == 0 {
		return nil, nil
	}
	return &Violation{
		Key:         "medals_without_registry",
		Severity:    SeverityFail,
		Count:       len(ids),
		Sample:      capSample(ids),
		Description: "rows medals_earned orphelines (match_id absent de match_registry)",
	}, nil
}

// ─── I9 — lusr_v2_orphan (FAIL) ─────────────────────────────────────────────
//
// Contrat dual-row LUSR v2 (mêmes sémantiques que RunDualRowSentinel) : un
// match noté porte soit LUSR seul (héritage v1), soit LUSR + LUSR_V2 (nominal
// post-bascule). LUSR_V2 seul = bug d'écriture canonique.
func checkLUSRV2Orphan(ctx context.Context, playerDB, _ *sql.DB, _ string) (*Violation, error) {
	ids, err := collectIDs(ctx, playerDB, `
		SELECT match_id
		FROM match_skill_rank
		GROUP BY match_id
		HAVING SUM(CASE WHEN rating_type = 'LUSR_V2' THEN 1 ELSE 0 END) > 0
		   AND SUM(CASE WHEN rating_type = 'LUSR' THEN 1 ELSE 0 END) = 0`)
	if err != nil {
		return nil, fmt.Errorf("invariants/lusr_v2_orphan: %w", err)
	}
	if len(ids) == 0 {
		return nil, nil
	}
	return &Violation{
		Key:         "lusr_v2_orphan",
		Severity:    SeverityFail,
		Count:       len(ids),
		Sample:      capSample(ids),
		Description: "matchs avec row LUSR_V2 sans row LUSR (contrat dual-row violé, cf. RunDualRowSentinel)",
	}, nil
}

// ─── I10 — skill_rank_missing (WARN) ────────────────────────────────────────
//
// Contrat souple : tout match du joueur devrait porter une row match_skill_rank
// (LUSR pour le social, CSR pour le ranked). WARN car certains contextes
// (PvE/Firefight, DNF précoces) peuvent légitimement en manquer ; une
// CROISSANCE du count signale en revanche la classe « désync watermark v2 »
// (watermark shared avancé sans row player DB, incident 2026-06-03).
func checkSkillRankMissing(ctx context.Context, playerDB, sharedDB *sql.DB, xuid string) (*Violation, error) {
	rated, err := collectIDs(ctx, playerDB, `SELECT DISTINCT match_id FROM match_skill_rank`)
	if err != nil {
		return nil, fmt.Errorf("invariants/skill_rank_missing: player query: %w", err)
	}
	ratedSet := make(map[string]struct{}, len(rated))
	for _, id := range rated {
		ratedSet[id] = struct{}{}
	}
	sharedIDs, err := collectIDs(ctx, sharedDB,
		`SELECT DISTINCT match_id FROM match_participants WHERE xuid || '' = ? AND match_id IS NOT NULL`, xuid)
	if err != nil {
		return nil, fmt.Errorf("invariants/skill_rank_missing: shared query: %w", err)
	}
	var missing []string
	for _, id := range sharedIDs {
		if _, ok := ratedSet[id]; !ok {
			missing = append(missing, id)
		}
	}
	if len(missing) == 0 {
		return nil, nil
	}
	return &Violation{
		Key:         "skill_rank_missing",
		Severity:    SeverityWarn,
		Count:       len(missing),
		Sample:      capSample(missing),
		Description: "matchs du joueur sans row match_skill_rank (PvE toléré ; croissance = désync watermark LUSR)",
	}, nil
}

// ─── I11 — citations_missing (WARN) ─────────────────────────────────────────
//
// Contrat souple : un match où le joueur a gagné des médailles devrait avoir
// des rows match_citations (pipeline BackfillMatchCitations). WARN : dépend de
// metadata.citation_mappings (absent sur env minimal) et du passage du
// post-sync citations.
func checkCitationsMissing(ctx context.Context, playerDB, sharedDB *sql.DB, xuid string) (*Violation, error) {
	cited, err := collectIDs(ctx, playerDB, `SELECT DISTINCT match_id FROM match_citations`)
	if err != nil {
		return nil, fmt.Errorf("invariants/citations_missing: player query: %w", err)
	}
	citedSet := make(map[string]struct{}, len(cited))
	for _, id := range cited {
		citedSet[id] = struct{}{}
	}
	medaled, err := collectIDs(ctx, sharedDB,
		`SELECT DISTINCT match_id FROM medals_earned WHERE xuid || '' = ?`, xuid)
	if err != nil {
		return nil, fmt.Errorf("invariants/citations_missing: shared query: %w", err)
	}
	var missing []string
	for _, id := range medaled {
		if _, ok := citedSet[id]; !ok {
			missing = append(missing, id)
		}
	}
	if len(missing) == 0 {
		return nil, nil
	}
	return &Violation{
		Key:         "citations_missing",
		Severity:    SeverityWarn,
		Count:       len(missing),
		Sample:      capSample(missing),
		Description: "matchs avec médailles mais sans match_citations (pipeline citations non passé)",
	}, nil
}

// ─── I12 — psa_missing (WARN) ───────────────────────────────────────────────
//
// Contrat souple : un match enrichi devrait avoir ses personal_score_awards
// (écrits par le traitement per-match — PAS convergés après delta-skip à date,
// cf. audit Phase 3 du plan). Alimente l'axe « objectif » du radar synergie.
func checkPersonalScoreAwardsMissing(ctx context.Context, playerDB, _ *sql.DB, _ string) (*Violation, error) {
	enriched, err := collectIDs(ctx, playerDB, `SELECT match_id FROM player_match_enrichment`)
	if err != nil {
		return nil, fmt.Errorf("invariants/psa_missing: enrichment query: %w", err)
	}
	awarded, err := collectIDs(ctx, playerDB, `SELECT DISTINCT match_id FROM personal_score_awards`)
	if err != nil {
		return nil, fmt.Errorf("invariants/psa_missing: psa query: %w", err)
	}
	awardedSet := make(map[string]struct{}, len(awarded))
	for _, id := range awarded {
		awardedSet[id] = struct{}{}
	}
	var missing []string
	for _, id := range enriched {
		if _, ok := awardedSet[id]; !ok {
			missing = append(missing, id)
		}
	}
	if len(missing) == 0 {
		return nil, nil
	}
	return &Violation{
		Key:         "psa_missing",
		Severity:    SeverityWarn,
		Count:       len(missing),
		Sample:      capSample(missing),
		Description: "matchs enrichis sans personal_score_awards (delta-skip non convergé pour les PSA)",
	}, nil
}

// ─── I13 — xuid_alias_missing (WARN) ────────────────────────────────────────
//
// Contrat : tout xuid humain présent dans match_participants a un alias
// gamertag dans xuid_aliases (sinon l'UI retombe sur le xuid brut — classe
// « GUID/XUID partout »). Les bots (préfixe bid() sont exclus.
func checkXuidAliasMissing(ctx context.Context, _ *sql.DB, sharedDB *sql.DB, _ string) (*Violation, error) {
	ids, err := collectIDs(ctx, sharedDB, `
		SELECT DISTINCT p.xuid || '' AS xid
		FROM match_participants p
		LEFT JOIN xuid_aliases a ON a.xuid || '' = p.xuid || ''
		WHERE a.xuid IS NULL
		  AND p.xuid IS NOT NULL
		  AND NOT starts_with(p.xuid || '', 'bid(')`)
	if err != nil {
		return nil, fmt.Errorf("invariants/xuid_alias_missing: %w", err)
	}
	if len(ids) == 0 {
		return nil, nil
	}
	return &Violation{
		Key:         "xuid_alias_missing",
		Severity:    SeverityWarn,
		Count:       len(ids),
		Sample:      capSample(ids),
		Description: "xuids humains de match_participants sans alias gamertag (affichage xuid brut)",
	}, nil
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func collectIDs(ctx context.Context, db *sql.DB, query string, args ...any) ([]string, error) {
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func capSample(ids []string) []string {
	if len(ids) <= sampleCap {
		return ids
	}
	return ids[:sampleCap]
}
