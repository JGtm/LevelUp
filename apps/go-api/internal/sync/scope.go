// Package sync — scope.go : SyncScope centralise tous les flags de données backfill.
//
// Portage de src/data/sync/scope.py (SyncScope dataclass, ~96 champs booléens).
// Construit depuis les arguments CLI ou programmatiquement, puis résolu
// via Resolve() qui applique les implications (all_data → champs, force_X → X,
// groupes → sous-champs).
//
// Nouveau code : toujours passer un *SyncScope aux fonctions backfill.
package sync

// SyncScope centralise tous les flags de données partagés entre sync et backfill.
// Portage 1:1 de la dataclass Python src/data/sync/scope.py.
type SyncScope struct {
	// ── Options générales ────────────────────────────────────────────────
	DryRun         bool
	MaxMatches     int
	RequestsPerSec int
	DetectionMode  string // "or" (défaut) ou "and"

	// ── Types de données ────────────────────────────────────────────────
	Medals              bool
	Events              bool
	Skill               bool
	PersonalScores      bool
	PerformanceScores   bool
	Aliases             bool
	Accuracy            bool
	EnemyMMR            bool
	Assets              bool
	Participants        bool
	ParticipantsScores  bool
	ParticipantsKDA     bool
	ParticipantsShots   bool
	ParticipantsDamage  bool
	ParticipantsAvgLife bool
	KillerVictim        bool
	EndTime             bool
	Sessions            bool
	Shots               bool
	Citations           bool
	ParticipantsEnrich  bool
	TeammatesSig        bool

	// ── Flags force ─────────────────────────────────────────────────────
	ForceMedals              bool
	ForceEvents              bool
	ForceSkill               bool
	ForceAccuracy            bool
	ForceShots               bool
	ForcePersonalScores      bool
	ForcePerformanceScores   bool
	ForceParticipantsShots   bool
	ForceParticipantsDamage  bool
	ForceParticipantsAvgLife bool
	ForceEnemyMMR            bool
	ForceAliases             bool
	ForceAssets              bool
	ForceParticipants        bool
	ForceEndTime             bool
	ForceSessions            bool
	ForceCitations           bool
	ForceParticipantsEnrich  bool
	ForceTeammatesSig        bool

	// ── PVE (Firefight) — v5.2 ──────────────────────────────────────────
	PVEStats      bool
	ForcePVEStats bool

	// ── Weapon kills — RETIRÉ le 2026-09-01 ────────────────────────────
	// Les axes Weapons / ForceWeapons / ForceNoFilm sélectionnaient des matchs pour
	// l'étape 1.55 (corrélation tirs ↔ instant du kill), supprimée avec son producteur
	// (lot arme-source-unique, ADR à venir). Sur Halo Infinite le détail par arme vient
	// désormais de la source de dégât, produite par l'étape 1.57 qui porte SA PROPRE
	// sélection de retard. Un axe de scope sans exécuteur est un interrupteur menteur :
	// il est retiré plutôt que laissé branché sur rien.

	// ── LUSR / CSR / Skill Rank — v5.3 ─────────────────────────────────
	LUSR           bool
	ForceLUSR      bool
	CSR            bool
	ForceCSR       bool
	FetchCSR       bool
	SkillRank      bool
	ForceSkillRank bool

	// ── Badges narrative comeback — v6.2 ────────────────────────────────
	ComebackBadges      bool
	ForceComebackBadges bool

	// ── Playable duration — v6.3 ────────────────────────────────────────
	PlayableDuration      bool
	ForcePlayableDuration bool

	// ── EngagementScore — Phase 3 du plan engagement ───────────────────
	// Calcule le score d'engagement (cf .ai/REFLEXION_ENGAGEMENT_SCORE_*)
	// pour chaque match du joueur. Necessite que les highlight_events soient
	// deja charges (Events=true ou backfill_completed bit MBitEvents).
	EngagementScores      bool
	ForceEngagementScores bool

	// ── AssistsModel — régression OLS per-mode expected_assists ─────────
	// Calcule, par mode de jeu, un modèle linéaire multi-varié (6 features)
	// pour prédire les assistances. Stocké dans player_assists_model (stats.duckdb).
	// Seuil : 15 matchs par mode. Aucun appel API requis.
	AssistsModel      bool
	ForceAssistsModel bool

	// ── EngagementCoefficients — Phase recompute coefs ──────────────────
	// Recompute UNIQUEMENT les coefficients perso (mediane glissante des
	// paces). Suppose que EngagementScores a deja peuple les colonnes
	// engagement_pace_* (sinon skip silencieux). Tres rapide (~5ms par
	// joueur — 2 queries + median). Utile pour rafraichir apres un
	// ajustement de formule sans re-scanner toute l'historique.
	EngagementCoefficients bool
	// ForceEngagementCoefficients : pour le moment le recompute est deja
	// idempotent (UPSERT), donc le flag force n'a pas d'effet specifique.
	// Conserve pour symetrie API + extensibilite future (ex. ignorer le
	// seuil MinMatchesForCoef).
	ForceEngagementCoefficients bool

	// ── Méta-flag ───────────────────────────────────────────────────────
	AllData bool
}

// allDataFields are the field setters activated by AllData=true.
// Identique à _ALL_DATA_FIELDS Python.
var allDataFields = []func(*SyncScope){
	func(s *SyncScope) { s.Medals = true },
	func(s *SyncScope) { s.Events = true },
	func(s *SyncScope) { s.Skill = true },
	func(s *SyncScope) { s.PersonalScores = true },
	func(s *SyncScope) { s.PerformanceScores = true },
	func(s *SyncScope) { s.Aliases = true },
	func(s *SyncScope) { s.Accuracy = true },
	func(s *SyncScope) { s.EnemyMMR = true },
	func(s *SyncScope) { s.Assets = true },
	func(s *SyncScope) { s.Participants = true },
	func(s *SyncScope) { s.ParticipantsScores = true },
	func(s *SyncScope) { s.ParticipantsKDA = true },
	func(s *SyncScope) { s.ParticipantsShots = true },
	func(s *SyncScope) { s.ParticipantsDamage = true },
	func(s *SyncScope) { s.ParticipantsAvgLife = true },
	func(s *SyncScope) { s.KillerVictim = true },
	func(s *SyncScope) { s.EndTime = true },
	func(s *SyncScope) { s.Sessions = true },
	func(s *SyncScope) { s.Shots = true },
	func(s *SyncScope) { s.Citations = true },
	func(s *SyncScope) { s.ParticipantsEnrich = true },
	func(s *SyncScope) { s.TeammatesSig = true },
	func(s *SyncScope) { s.PVEStats = true },
	func(s *SyncScope) { s.LUSR = true },
	func(s *SyncScope) { s.CSR = true },
	func(s *SyncScope) { s.SkillRank = true },
	func(s *SyncScope) { s.ComebackBadges = true },
	func(s *SyncScope) { s.PlayableDuration = true },
	func(s *SyncScope) { s.EngagementScores = true },
	func(s *SyncScope) { s.AssistsModel = true },
}

// Resolve applique les implications : AllData → champs, Skill → EnemyMMR,
// SkillRank → LUSR+CSR, force_X → X. Doit être appelé une seule fois après
// construction. applyForceImplications est toujours appliqué en dernier.
func (s *SyncScope) Resolve() {
	// AllData active tous les champs data
	if s.AllData {
		for _, setter := range allDataFields {
			setter(s)
		}
	}

	// --skill active aussi le MMR adverse (fetch skill ⇒ fetch enemy MMR).
	if s.Skill {
		s.EnemyMMR = true
	}

	// ── SkillRank = LUSR + CSR ──
	if s.SkillRank {
		s.LUSR = true
		s.CSR = true
	}
	if s.ForceSkillRank {
		s.ForceLUSR = true
		s.ForceCSR = true
	}

	// ── force_X implique X (toujours en dernier) ──
	s.applyForceImplications()
}

// applyForceImplications assure que si ForceX=true alors X=true.
func (s *SyncScope) applyForceImplications() {
	imply := func(force *bool, data *bool) {
		if *force && !*data {
			*data = true
		}
	}
	imply(&s.ForceMedals, &s.Medals)
	imply(&s.ForceEvents, &s.Events)
	imply(&s.ForceSkill, &s.Skill)
	imply(&s.ForceAccuracy, &s.Accuracy)
	imply(&s.ForceShots, &s.Shots)
	imply(&s.ForcePersonalScores, &s.PersonalScores)
	imply(&s.ForcePerformanceScores, &s.PerformanceScores)
	imply(&s.ForceParticipantsShots, &s.ParticipantsShots)
	imply(&s.ForceParticipantsDamage, &s.ParticipantsDamage)
	imply(&s.ForceParticipantsAvgLife, &s.ParticipantsAvgLife)
	imply(&s.ForceEnemyMMR, &s.EnemyMMR)
	imply(&s.ForceAliases, &s.Aliases)
	imply(&s.ForceAssets, &s.Assets)
	imply(&s.ForceParticipants, &s.Participants)
	imply(&s.ForceEndTime, &s.EndTime)
	imply(&s.ForceSessions, &s.Sessions)
	imply(&s.ForceCitations, &s.Citations)
	imply(&s.ForceParticipantsEnrich, &s.ParticipantsEnrich)
	imply(&s.ForceTeammatesSig, &s.TeammatesSig)
	imply(&s.ForcePVEStats, &s.PVEStats)
	imply(&s.ForceLUSR, &s.LUSR)
	imply(&s.ForceCSR, &s.CSR)
	imply(&s.ForceSkillRank, &s.SkillRank)
	imply(&s.ForceComebackBadges, &s.ComebackBadges)
	imply(&s.ForcePlayableDuration, &s.PlayableDuration)
	imply(&s.ForceEngagementScores, &s.EngagementScores)
	imply(&s.ForceAssistsModel, &s.AssistsModel)
}

// NewScopeAll crée un scope avec AllData=true pré-résolu.
func NewScopeAll(maxMatches int) *SyncScope {
	s := &SyncScope{AllData: true, MaxMatches: maxMatches, DetectionMode: "or"}
	s.Resolve()
	return s
}

// HasAnyOption retourne true si au moins un type de données est activé.
func (s *SyncScope) HasAnyOption() bool {
	flags := []bool{
		s.Medals, s.Events, s.Skill, s.PersonalScores, s.PerformanceScores,
		s.Aliases, s.Accuracy, s.EnemyMMR, s.Assets, s.Participants,
		s.ParticipantsScores, s.ParticipantsKDA, s.ParticipantsShots,
		s.ParticipantsDamage, s.ParticipantsAvgLife, s.KillerVictim,
		s.EndTime, s.Sessions, s.Shots, s.Citations, s.ParticipantsEnrich,
		s.TeammatesSig, s.PVEStats, s.LUSR, s.CSR, s.SkillRank,
		s.ComebackBadges, s.PlayableDuration, s.EngagementScores, s.AssistsModel,
	}
	for _, f := range flags {
		if f {
			return true
		}
	}
	return false
}

// NeedsAPI retourne true si au moins un type nécessite un appel API.
func (s *SyncScope) NeedsAPI() bool {
	return s.Medals || s.Events || s.Skill || s.PersonalScores ||
		s.PerformanceScores || s.Aliases || s.Accuracy || s.EnemyMMR ||
		s.Assets || s.Participants || s.Shots || s.ParticipantsScores ||
		s.ParticipantsKDA || s.ParticipantsShots || s.ParticipantsDamage ||
		s.ParticipantsAvgLife || s.ParticipantsEnrich ||
		s.CSR || s.FetchCSR || s.SkillRank || s.PlayableDuration
}

// NeedsLocalOnly retourne true si des traitements locaux (sans API) sont demandés.
func (s *SyncScope) NeedsLocalOnly() bool {
	return s.KillerVictim || s.EndTime || s.Sessions || s.Citations ||
		s.LUSR || s.SkillRank || s.EngagementScores || s.AssistsModel
}

// Noms de champs SyncScope utilisés comme clés dans requestedTypeMap et fieldVals.
// Centralisés pour réduire la duplication littérale (lint goconst).
const (
	scopeFieldMedals = "Medals"
)

// requestedTypeMap maps scope field → bitmask key for backfill_completed tracking.
// Identique à _REQUESTED_TYPE_MAP Python.
var requestedTypeMap = map[string]string{
	scopeFieldMedals:      BackfillTypeMedals,
	"Events":              BackfillTypeEvents,
	"Skill":               BackfillTypeSkill,
	"PersonalScores":      BackfillTypePersonalScores,
	"PerformanceScores":   "performance_scores",
	"Aliases":             BackfillTypeAliases,
	"Accuracy":            MetricKeyAccuracy,
	"Shots":               BackfillTypeShots,
	"EnemyMMR":            BackfillTypeEnemyMMR,
	"Assets":              "assets",
	"Participants":        "participants",
	"ParticipantsScores":  "participants_scores",
	"ParticipantsKDA":     "participants_kda",
	"ParticipantsShots":   "participants_shots",
	"ParticipantsDamage":  "participants_damage",
	"ParticipantsAvgLife": "participants_avg_life",
	"PVEStats":            "pve_stats",
	"LUSR":                "lusr",
	"CSR":                 "csr",
	"EngagementScores":    "engagement_scores",
}

// RequestedTypes retourne la liste des noms de types demandés
// pour le bitmask backfill_completed.
func (s *SyncScope) RequestedTypes() []string {
	fieldVals := map[string]bool{
		scopeFieldMedals:      s.Medals,
		"Events":              s.Events,
		"Skill":               s.Skill,
		"PersonalScores":      s.PersonalScores,
		"PerformanceScores":   s.PerformanceScores,
		"Aliases":             s.Aliases,
		"Accuracy":            s.Accuracy,
		"Shots":               s.Shots,
		"EnemyMMR":            s.EnemyMMR,
		"Assets":              s.Assets,
		"Participants":        s.Participants,
		"ParticipantsScores":  s.ParticipantsScores,
		"ParticipantsKDA":     s.ParticipantsKDA,
		"ParticipantsShots":   s.ParticipantsShots,
		"ParticipantsDamage":  s.ParticipantsDamage,
		"ParticipantsAvgLife": s.ParticipantsAvgLife,
		"PVEStats":            s.PVEStats,
		"LUSR":                s.LUSR,
		"CSR":                 s.CSR,
		"EngagementScores":    s.EngagementScores,
	}
	var result []string
	for fieldName, typeKey := range requestedTypeMap {
		if fieldVals[fieldName] {
			result = append(result, typeKey)
		}
	}
	return result
}
