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

	// ── Types granulaires v5.2 (nouveaux champs bitmask) ────────────────
	// Skill / MMR — Halo Infinite n'expose pas Assists dans StatPerformances.
	TeamMMR        bool
	KillsExpected  bool
	DeathsExpected bool
	// Combat
	Damage  bool
	AvgLife bool
	// Kills détaillés
	GrenadeKills     bool
	MeleeKills       bool
	PowerWeaponKills bool
	HeadshotKills    bool
	MaxSpree         bool
	// Divers
	KDARecalc  bool
	TimePlayed bool

	// ── Groupes (alias résolus dans Resolve()) ──────────────────────────
	MMR         bool // = TeamMMR + EnemyMMR
	Expected    bool // = KillsExpected + DeathsExpected (Halo Infinite n'a pas d'Assists)
	Combat      bool // = Accuracy + Shots + Damage
	KillsDetail bool // = GrenadeKills + MeleeKills + PowerWeaponKills + HeadshotKills
	CoreStats   bool // = Combat + AvgLife + KillsDetail + KDARecalc + TimePlayed

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

	// ── Flags force granulaires v5.2 ────────────────────────────────────
	ForceTeamMMR          bool
	ForceKillsExpected    bool
	ForceDeathsExpected   bool
	ForceAvgLife          bool
	ForceDamage           bool
	ForceGrenadeKills     bool
	ForceMeleeKills       bool
	ForcePowerWeaponKills bool
	ForceHeadshotKills    bool
	ForceMaxSpree         bool
	ForceKDARecalc        bool
	ForceTimePlayed       bool
	ForceMMR              bool
	ForceExpected         bool
	ForceCombat           bool
	ForceKillsDetail      bool
	ForceCoreStats        bool

	// ── PVE (Firefight) — v5.2 ──────────────────────────────────────────
	PVEStats      bool
	ForcePVEStats bool

	// ── Weapon kills — v5.5 / v5.6 ─────────────────────────────────────
	Weapons      bool
	ForceWeapons bool
	ForceNoFilm  bool

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
	func(s *SyncScope) { s.TeamMMR = true },
	func(s *SyncScope) { s.KillsExpected = true },
	func(s *SyncScope) { s.DeathsExpected = true },
	func(s *SyncScope) { s.Damage = true },
	func(s *SyncScope) { s.AvgLife = true },
	func(s *SyncScope) { s.GrenadeKills = true },
	func(s *SyncScope) { s.MeleeKills = true },
	func(s *SyncScope) { s.PowerWeaponKills = true },
	func(s *SyncScope) { s.HeadshotKills = true },
	func(s *SyncScope) { s.MaxSpree = true },
	func(s *SyncScope) { s.KDARecalc = true },
	func(s *SyncScope) { s.TimePlayed = true },
	func(s *SyncScope) { s.MMR = true },
	func(s *SyncScope) { s.Expected = true },
	func(s *SyncScope) { s.Combat = true },
	func(s *SyncScope) { s.KillsDetail = true },
	func(s *SyncScope) { s.CoreStats = true },
	func(s *SyncScope) { s.PVEStats = true },
	func(s *SyncScope) { s.Weapons = true },
	func(s *SyncScope) { s.LUSR = true },
	func(s *SyncScope) { s.CSR = true },
	func(s *SyncScope) { s.SkillRank = true },
	func(s *SyncScope) { s.ComebackBadges = true },
	func(s *SyncScope) { s.PlayableDuration = true },
	func(s *SyncScope) { s.EngagementScores = true },
	func(s *SyncScope) { s.AssistsModel = true },
}

// Resolve applique les implications : AllData → champs, groupes → sous-champs,
// force_X → X. Doit être appelé une seule fois après construction.
//
// Ordre critique des groupes v5.2 (extérieur → intérieur) :
//  1. CoreStats  → Combat, KillsDetail, AvgLife, KDARecalc, TimePlayed
//  2. Combat     → Accuracy, Shots, Damage
//  3. KillsDetail → GrenadeKills, MeleeKills, PowerWeaponKills, HeadshotKills
//  4. MMR        → TeamMMR, EnemyMMR
//  5. Expected   → KillsExpected, DeathsExpected (Halo Infinite n'expose pas Assists)
//  6. forceMap   (toujours en dernier)
func (s *SyncScope) Resolve() {
	// AllData active tous les champs data
	if s.AllData {
		for _, setter := range allDataFields {
			setter(s)
		}
	}

	// ── 1. Groupes de haut niveau ──
	if s.CoreStats {
		s.Combat = true
		s.AvgLife = true
		s.KillsDetail = true
		s.KDARecalc = true
		s.TimePlayed = true
	}

	// ── 2. Groupes intermédiaires ──
	if s.Combat {
		s.Accuracy = true
		s.Shots = true
		s.Damage = true
	}
	if s.KillsDetail {
		s.GrenadeKills = true
		s.MeleeKills = true
		s.PowerWeaponKills = true
		s.HeadshotKills = true
	}

	// ── 3. Groupes skill ──
	if s.MMR {
		s.TeamMMR = true
		s.EnemyMMR = true
	}
	if s.Expected {
		s.KillsExpected = true
		s.DeathsExpected = true
	}
	// --skill active aussi les champs granulaires (rétrocompatibilité)
	if s.Skill {
		s.TeamMMR = true
		s.EnemyMMR = true
		s.KillsExpected = true
		s.DeathsExpected = true
	}

	// ── 3b. SkillRank = LUSR + CSR ──
	if s.SkillRank {
		s.LUSR = true
		s.CSR = true
	}
	if s.ForceSkillRank {
		s.ForceLUSR = true
		s.ForceCSR = true
	}

	// ── 4. force_X implique X (toujours en dernier) ──
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
	imply(&s.ForceTeamMMR, &s.TeamMMR)
	imply(&s.ForceKillsExpected, &s.KillsExpected)
	imply(&s.ForceDeathsExpected, &s.DeathsExpected)
	imply(&s.ForceAvgLife, &s.AvgLife)
	imply(&s.ForceDamage, &s.Damage)
	imply(&s.ForceGrenadeKills, &s.GrenadeKills)
	imply(&s.ForceMeleeKills, &s.MeleeKills)
	imply(&s.ForcePowerWeaponKills, &s.PowerWeaponKills)
	imply(&s.ForceHeadshotKills, &s.HeadshotKills)
	imply(&s.ForceMaxSpree, &s.MaxSpree)
	imply(&s.ForceKDARecalc, &s.KDARecalc)
	imply(&s.ForceTimePlayed, &s.TimePlayed)
	imply(&s.ForceMMR, &s.MMR)
	imply(&s.ForceExpected, &s.Expected)
	imply(&s.ForceCombat, &s.Combat)
	imply(&s.ForceKillsDetail, &s.KillsDetail)
	imply(&s.ForceCoreStats, &s.CoreStats)
	imply(&s.ForcePVEStats, &s.PVEStats)
	imply(&s.ForceWeapons, &s.Weapons)
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
		s.TeammatesSig, s.TeamMMR, s.KillsExpected, s.DeathsExpected,
		s.Damage, s.AvgLife, s.GrenadeKills, s.MeleeKills,
		s.PowerWeaponKills, s.HeadshotKills, s.MaxSpree, s.KDARecalc,
		s.TimePlayed, s.MMR, s.Expected, s.Combat, s.KillsDetail, s.CoreStats,
		s.PVEStats, s.Weapons, s.LUSR, s.CSR, s.SkillRank,
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

// requestedTypeMap maps scope field → bitmask key for backfill_completed tracking.
// Identique à _REQUESTED_TYPE_MAP Python.
var requestedTypeMap = map[string]string{
	"Medals":              "medals",
	"Events":              "events",
	"Skill":               "skill",
	"PersonalScores":      "personal_scores",
	"PerformanceScores":   "performance_scores",
	"Aliases":             "aliases",
	"Accuracy":            "accuracy",
	"Shots":               "shots",
	"EnemyMMR":            "enemy_mmr",
	"Assets":              "assets",
	"Participants":        "participants",
	"ParticipantsScores":  "participants_scores",
	"ParticipantsKDA":     "participants_kda",
	"ParticipantsShots":   "participants_shots",
	"ParticipantsDamage":  "participants_damage",
	"ParticipantsAvgLife": "participants_avg_life",
	"PVEStats":            "pve_stats",
	"Weapons":             "weapons",
	"LUSR":                "lusr",
	"CSR":                 "csr",
	"EngagementScores":    "engagement_scores",
}

// RequestedTypes retourne la liste des noms de types demandés
// pour le bitmask backfill_completed.
func (s *SyncScope) RequestedTypes() []string {
	fieldVals := map[string]bool{
		"Medals":              s.Medals,
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
		"Weapons":             s.Weapons,
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
