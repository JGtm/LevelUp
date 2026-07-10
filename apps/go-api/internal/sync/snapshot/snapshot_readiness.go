package snapshot

import "levelup/go-api/internal/sync/matchflags"

// snapshot_readiness.go — prédicat de complétude « snapshot-ready » (Phase 2).
//
// Un match est SNAPSHOT-READY quand TOUTES ses dérivations sont TERMINALES :
// soit calculées, soit terminalement-absentes (film perdu, LUSR inéligible).
// Le but est de n'inclure dans le snapshot immuable que des matchs complets,
// SANS jamais cacher à vie un match légitimement partiel ni geler un match dont
// une dérivation reste bloquée (grâce par âge → pose forcée).
//
// Ce fichier ne contient QUE la logique PURE (aucune DB) : isMatchSnapshotReady
// + l'enum FERMÉ des raisons. L'orchestration (chargement des faits, INSERT
// stage='snapshot') et le câblage post-sync vivent ailleurs.
//
// Corrections issues de la vérif adversariale (Phase 2 design) :
//   - weapon-kills gaté par CapWeaponKills (PAS CapNativeKillMechanics, Halo-5-only) ;
//   - LUSR : un match fact-éligible SANS row LUSR (imbalance/DNF/group-hold) =
//     lusr_skipped NON bloquant — jamais un blocage/gel ;
//   - n'exiger que performance_score (engagement_score NULL = légitime) ;
//   - enum de raisons FERMÉ (cardinalité expvar bornée, ADR 0009) ;
//   - media-scan EXCLU du prédicat (best-effort, asynchrone).

// Raisons de complétude — vocabulaire FERMÉ (jamais de littéral dispersé ni de
// valeur dérivée d'un match_id : la cardinalité expvar en dépend). Deux familles :
//   - terminal-partiel (le match est READY, mais une dérivation est légitimement
//     absente) : lusr_ineligible, lusr_skipped, weapons_absent ;
//   - bloqué (dérivation non-terminale) : blocked_* — n'apparaît QUE sur une pose
//     FORCÉE après dépassement de la fenêtre de grâce, précédé de `forced`.
const (
	snapReasonLUSRIneligible = "lusr_ineligible" // ranked / firefight / FFA / non-2-team
	snapReasonLUSRSkipped    = "lusr_skipped"    // 2-team éligible mais pas de row (imbalance/DNF/group-hold)
	snapReasonWeaponsAbsent  = "weapons_absent"  // film 404/expiré (matchflags.MBitWeaponKillsNoFilm) — terminal
	snapReasonForced         = "forced"          // pose forcée : grâce dépassée malgré dérivation(s) bloquée(s)

	snapReasonBlockedEvents    = "blocked_events"
	snapReasonBlockedPerf      = "blocked_perf"
	snapReasonBlockedDominance = "blocked_dominance"
	snapReasonBlockedPSA       = "blocked_psa"
	snapReasonBlockedCitations = "blocked_citations"
	snapReasonBlockedWeapons   = "blocked_weapons"
	snapReasonBlockedPVE       = "blocked_pve"
)

// matchReadinessFacts agrège les marqueurs terminaux d'UN match (faits lus en
// base par l'orchestrateur). Aucun pointeur : les bool « *Set » disent si la
// dérivation est terminale (NOT NULL / bit posé / row existante).
type matchReadinessFacts struct {
	// shared.match_registry
	eventsLoaded      bool  // events_loaded = TRUE (chargé OU no-film définitif)
	backfillCompleted int64 // match_registry.backfill_completed (bits matchflags.MBitWeaponKills/NoFilm/PVEStats)
	isRanked          bool
	isFirefight       bool
	durationSeconds   int
	humanTeamCount    int // nb d'équipes humaines distinctes (LUSR exige exactement 2)
	// player_match_enrichment_latest + vues player
	perfScoreSet   bool // performance_score NOT NULL
	dominanceSet   bool // dominance_flag NOT NULL
	psaCheckedSet  bool // psa_checked_at NOT NULL
	citationsExist bool // EXISTS match_citations_latest
	lusrRowExists  bool // EXISTS match_skill_rank_latest rating_type='LUSR'
}

// titleReadinessCaps porte les capabilities du titre pertinentes pour le prédicat
// (résolues une fois par l'orchestrateur, pour garder isMatchSnapshotReady pur).
type titleReadinessCaps struct {
	hasLUSR        bool // CapLUSR
	hasWeaponKills bool // CapWeaponKills (film-decoder par-arme ; Infinite oui, Halo 5 non)
	hasFirefight   bool // CapFirefight
}

// isMatchSnapshotReady décide si un match est snapshot-ready et retourne les
// partial_reasons (vocabulaire fermé). agedOut = âge du match > fenêtre de grâce.
//
//   - blocked vide → READY, reasons = terminal-partiels (lusr_*, weapons_absent) ;
//   - blocked non vide & agedOut → READY FORCÉ, reasons = partiels + `forced` + blocked_* ;
//   - blocked non vide & !agedOut → NON ready.
func isMatchSnapshotReady(f matchReadinessFacts, c titleReadinessCaps, agedOut bool) (bool, []string) {
	var partial, blocked []string

	if !f.eventsLoaded {
		blocked = append(blocked, snapReasonBlockedEvents)
	}
	if !f.perfScoreSet {
		blocked = append(blocked, snapReasonBlockedPerf)
	}
	if !f.dominanceSet {
		blocked = append(blocked, snapReasonBlockedDominance)
	}
	if !f.psaCheckedSet {
		blocked = append(blocked, snapReasonBlockedPSA)
	}
	if !f.citationsExist {
		blocked = append(blocked, snapReasonBlockedCitations)
	}

	// Weapon-kills : seulement si le titre les produit (CapWeaponKills).
	if c.hasWeaponKills {
		switch {
		case f.backfillCompleted&matchflags.MBitWeaponKills != 0:
			// calculé → rien
		case f.backfillCompleted&matchflags.MBitWeaponKillsNoFilm != 0:
			partial = append(partial, snapReasonWeaponsAbsent) // terminal-absent
		default:
			blocked = append(blocked, snapReasonBlockedWeapons)
		}
	}

	// LUSR : seulement si le titre le supporte (CapLUSR).
	if c.hasLUSR {
		eligible := !f.isRanked && !f.isFirefight && f.durationSeconds >= 30 && f.humanTeamCount == 2
		switch {
		case !eligible:
			partial = append(partial, snapReasonLUSRIneligible) // terminal
		case f.lusrRowExists:
			// calculé → rien
		default:
			// fact-éligible mais pas de row (imbalance/DNF/group-hold) : NON bloquant.
			partial = append(partial, snapReasonLUSRSkipped)
		}
	}

	// PvE : seulement pour les matchs Firefight d'un titre qui les supporte.
	if f.isFirefight && c.hasFirefight && f.backfillCompleted&matchflags.MBitPVEStats == 0 {
		blocked = append(blocked, snapReasonBlockedPVE)
	}

	if len(blocked) == 0 {
		return true, partial
	}
	if agedOut {
		return true, append(append(partial, snapReasonForced), blocked...)
	}
	return false, nil
}
