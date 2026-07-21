package skill

// lusr_eligibility.go — prédicat d'éligibilité LUSR PARTAGÉ entre le scoreur
// (processOneShadowMatch, skill_v2_shadow.go) et le détecteur de trous
// (ScanLUSRGaps, lusr_gap_scan.go). Source unique anti-dérive (CLAUDE.md n°6) :
// sans ça le détecteur re-diverge du scoreur (over/under-count). Garde-rail
// TestNoDuplicateLUSREligibilityFilter (lusr_eligibility_guardrail_test.go).

import (
	"context"
	"database/sql"

	skillv2 "levelup/go-api/internal/analysis/skill_v2"
)

// lusrSkipReason classe la raison d'inéligibilité LUSR d'un match dépendant des
// rosters. Partagée par le scoreur et le détecteur : une seule définition
// d'« éligible ».
type lusrSkipReason int

const (
	lusrEligibleReason lusrSkipReason = iota // éligible (pas un skip)
	// lusrSkipNonTwoTeam : owner sans team_id, match ≠ 2 équipes humaines, ou
	// outcome non scorable (ni Win/Draw/Loss). Regroupé car tous comptés dans le
	// même bucket skippedNonTwoTeam par le scoreur (comportement historique).
	lusrSkipNonTwoTeam
	lusrSkipImbalance // |nA - nB| > 1 en effectif concurrent
)

// lusrEligibility porte le verdict d'éligibilité d'un match + les rosters / outcome
// déjà résolus, pour que le scoreur les réutilise sans re-query.
type lusrEligibility struct {
	teamA    []rosterMember
	teamB    []rosterMember
	outcomeA skillv2.TeamResult
	eligible bool
	reason   lusrSkipReason
}

// classifyLUSREligibility applique les filtres d'éligibilité LUSR DÉPENDANT DES
// ROSTERS, partagés entre le scoreur (processOneShadowMatch) et le détecteur de
// trous (ScanLUSRGaps). La résolution de chaîne (GetLUSRChainForTitle) et le
// filtre SQL (loadShadowMatches) sont les deux autres maillons mono-source de
// l'éligibilité, appelés par les callers en amont.
//
// Checks, ordre identique à l'ancien inline de processOneShadowMatch :
//  1. ownerHasTeam (team_id de l'owner renseigné) ;
//  2. exactement 2 équipes humaines (buildTwoTeamRosters) ;
//  3. équilibre concurrent |nA - nB| ≤ 1 — on compte les présents au coup
//     d'envoi (present_at_beginning), pas len(team) : dans Halo l'effectif par
//     camp est constant, un quitter est REMPLACÉ jamais ajouté ; len()
//     sur-compterait quitter + remplaçant comme 2 joueurs fictifs (cf.
//     concurrentTeamSize) → ~32% des 4v4 avec subs sautés à tort ;
//  4. outcome owner ∈ {Win, Draw, Loss} (DNF / inconnu → non scorable).
//
// Read-only (1 query buildTwoTeamRosters), n'écrit rien, ne consulte pas le
// watermark : la distinction intérieur/pending est du ressort du caller.
func classifyLUSREligibility(ctx context.Context, sharedDB *sql.DB, m shadowMatch) lusrEligibility {
	if !m.ownerHasTeam {
		return lusrEligibility{reason: lusrSkipNonTwoTeam}
	}
	teamA, teamB, ok := buildTwoTeamRosters(ctx, sharedDB, m.matchID, m.ownerTeamID)
	if !ok {
		return lusrEligibility{reason: lusrSkipNonTwoTeam}
	}
	if isTeamImbalanceTooHigh(concurrentTeamSize(teamA), concurrentTeamSize(teamB)) {
		return lusrEligibility{reason: lusrSkipImbalance}
	}
	outcomeA, ok := outcomeToTeamResult(m.ownerOutcome)
	if !ok {
		return lusrEligibility{reason: lusrSkipNonTwoTeam}
	}
	return lusrEligibility{teamA: teamA, teamB: teamB, outcomeA: outcomeA, eligible: true, reason: lusrEligibleReason}
}
