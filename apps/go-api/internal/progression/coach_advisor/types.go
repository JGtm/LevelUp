// Package coach_advisor — pont entre le coach (détecteur passif) et Prestige
// (système de défis calibrés).
//
// Le coach_advisor consomme les signaux du coach (LOWESS positive, combat
// patterns, near-miss records/milestones, comeback) et propose au joueur des
// challenges ou des arcs Prestige calibrés. C'est le troisième acteur introduit
// par l'ADR 0020.
//
// Cinq invariants gravés ici par les tests d'intégration (cf. ADR 0020 §1) :
//
//	I1. Tout challenge passe par prestige.CalculatePalier() — pas de bypass.
//	I2. La baseline reste l'ancre — pas de cible absolue inventée.
//	I3. Tout template synthétisé reste un prestige.Template valide.
//	I4. Tout arc dynamique reste un prestige.Arc standard (IsPreset=false).
//	I5. Aucune génération sans signal fort (strength >= synthesis_min_strength).
//
// Ce package est PUR sur ses fonctions matcher/signals/synthesizer/composer —
// l'I/O est confiné à service.go.
package coach_advisor

import (
	"time"

	"levelup/go-api/internal/prestige"
)

// SignalKind identifie la source d'un signal coach.
//
// Mappé un-à-un sur les détecteurs du package internal/progression/coach et
// internal/progression/patterns (cf. signals.go pour le mapping).
type SignalKind string

const (
	// SignalLOWESSPositive : pente LOWESS positive soutenue sur une métrique
	// (typiquement >= 14 jours), indique une amélioration à consolider.
	SignalLOWESSPositive SignalKind = "lowess_positive"
	// SignalLOWESSSoftNegative : pente LOWESS négative soutenue (>= 14 jours) —
	// opportunité de STABILISER l'axe (registre soft, non-culpabilisant). Symétrique
	// de SignalLOWESSPositive ; strength basée sur la magnitude de la pente.
	SignalLOWESSSoftNegative SignalKind = "lowess_soft_negative"
	// SignalCombatPatternActive : OC élevé + résidu engagement > +5 — joueur
	// performant en combat, prêt pour un challenge offensif.
	SignalCombatPatternActive SignalKind = "combat_pattern_active"
	// SignalCombatPatternDiscreet : résidu engagement < -5 — joueur discret,
	// support à valoriser.
	SignalCombatPatternDiscreet SignalKind = "combat_pattern_discreet"
	// SignalCombatPatternFragile : DR < 70% P80 — survie à solidifier.
	SignalCombatPatternFragile SignalKind = "combat_pattern_fragile"
	// SignalRecordApproach : valeur courante à moins de 5% du PB — pousser
	// vers le franchissement.
	SignalRecordApproach SignalKind = "record_approach"
	// SignalMilestoneApproach : milestone à >= 90% — pousser au déblocage.
	SignalMilestoneApproach SignalKind = "milestone_approach"
	// SignalComebackWelcome : retour après >= 5 jours d'inactivité — challenge
	// soft pour réamorcer.
	SignalComebackWelcome SignalKind = "comeback_welcome"
	// SignalLUSRTierApproach : μ à ±10 points du sub-tier suivant — challenge
	// court pour valider le passage.
	SignalLUSRTierApproach SignalKind = "lusr_tier_approach"
)

// Signal est l'entrée du matcher / synthesizer.
//
// Construit par signals.SignalsFromCoachInput() à partir des sorties du coach
// existant ([]coach.Alert, LUSRSnapshot, PatternReport). Ce package ne dépend
// pas du package coach pour éviter le couplage inverse — c'est le service
// orchestrateur qui fait la traduction.
type Signal struct {
	Kind   SignalKind
	Metric string

	// LUSRComponent et RadarAxis permettent le matching contre les templates
	// Prestige (Template.LUSRComponents et Template.RadarAxes). L'un OU l'autre
	// peut être vide selon la nature du signal.
	LUSRComponent string
	RadarAxis     string

	// Strength normalisé sur [0, 1] : indique la force du signal. Filtré par
	// l'invariant I5 (synthesis_min_strength).
	Strength float64

	// Source garde une référence opaque vers la donnée d'origine (alert,
	// pattern report entry, near-miss record). Utilisée pour i18n params et
	// debugging — pas pour la logique.
	Source any
}

// MatchScore est le résultat du matching d'un signal contre un template.
//
// Score = lusr_component_weight * (1 si Template.LUSRComponents contient
//
//	  Signal.LUSRComponent sinon 0) +
//	radar_axis_weight * (1 si Template.RadarAxes contient Signal.RadarAxis sinon 0) +
//	metric_match_weight * (1 si Template.Metric == Signal.Metric sinon 0)
//
// Voir matcher.go pour le détail.
type MatchScore struct {
	Template prestige.Template
	Score    float64
}

// ProposalKind discrimine entre les deux artefacts qu'un advisor peut proposer.
type ProposalKind string

const (
	// ProposalKindChallenge : matérialisation d'un seul template (catalogue ou
	// synthétisé) en challenge Prestige à l'acceptance.
	ProposalKindChallenge ProposalKind = "challenge"
	// ProposalKindArc : matérialisation d'un Arc Prestige dynamique avec N
	// challenges (étapes) à l'acceptance.
	ProposalKindArc ProposalKind = "arc"
)

// ProposalStatus suit le cycle de vie d'une proposal (cf. ADR 0020 §3).
//
// Transitions :
//
//	pending ─accept────────> accepted   (resolved_ref = challenge_id ou arc_id)
//	pending ─dismiss───────> dismissed
//	pending ─supersession──> superseded (superseded_by = id de la nouvelle)
//	pending ─completion────> obsoleted  (challenge accepté sur même axis complété)
//	pending ─60j sans signal> stale     (job GC optionnel V2.1)
type ProposalStatus string

const (
	ProposalPending    ProposalStatus = "pending"
	ProposalAccepted   ProposalStatus = "accepted"
	ProposalDismissed  ProposalStatus = "dismissed"
	ProposalSuperseded ProposalStatus = "superseded"
	ProposalObsoleted  ProposalStatus = "obsoleted"
	ProposalStale      ProposalStatus = "stale"
)

// ProposalOrigin trace si le template référencé vient du catalogue Prestige
// ou d'une synthèse coach (cf. ADR 0021).
type ProposalOrigin string

const (
	OriginCatalog     ProposalOrigin = "catalog"
	OriginSynthesized ProposalOrigin = "synthesized"
)

// Proposal est l'artefact persisté par le coach_advisor — différé jusqu'à
// l'action joueur (accept / dismiss) ou supersession automatique.
//
// Persistance : table `coach_proposal` dans stats.duckdb (par joueur).
// Cf. ADR 0020 §"Schéma DuckDB" pour le SQL exact.
type Proposal struct {
	ID        string
	UserID    string
	TitleSlug string

	Kind ProposalKind

	// TemplateID est renseigné si Kind=challenge. Pour Kind=arc, il reste
	// vide et les étapes sont décrites dans ChallengesSpec.
	TemplateID string

	// ChallengesSpec n'est renseigné que pour Kind=arc — JSON sérialisé
	// décrivant les étapes (chaque étape référence un template_id catalogue
	// ou synthétisé déjà persisté).
	ChallengesSpec string

	// SuggestedTier est purement indicatif UI. Prestige recalcule via baseline
	// au moment du CreateChallenge (cf. I1).
	SuggestedTier prestige.Tier

	// Tracking origine
	SourceSignal SignalKind
	SourceMetric string
	RadarAxis    string
	Strength     float64
	Origin       ProposalOrigin

	// I18n
	ReasonKeyEN  string
	ReasonKeyFR  string
	ReasonParams string // JSON

	Status ProposalStatus

	CreatedAt  time.Time
	ExpiresAt  *time.Time // nullable (pas d'expiration par âge — cf. ADR 0020 §3)
	ResolvedAt *time.Time
	// ResolvedRef est challenge_id ou arc_id selon Kind si accepted.
	ResolvedRef string

	SupersededBy string
	SupersededAt *time.Time
	ObsoletedAt  *time.Time
}
