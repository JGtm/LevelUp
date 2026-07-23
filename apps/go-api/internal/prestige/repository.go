package prestige

import (
	"context"
	"time"
)

// repository.go — interfaces de persistance du module Prestige.
//
// Les implémentations vivent dans internal/platform/duckdb/ et sont injectées
// dans le service via constructor (DI). La couche `domain` (palier, baseline,
// lifecycle, evaluator) ne doit jamais importer ces interfaces — elle reçoit
// des données déjà chargées.

// ---------- ChallengeRepo (stats.duckdb par joueur) ----------

// ChallengeRepo gère la persistance des défis individuels.
type ChallengeRepo interface {
	Create(ctx context.Context, c Challenge) error
	Get(ctx context.Context, id string) (Challenge, error)
	List(ctx context.Context, filter ChallengeFilter) ([]Challenge, error)
	UpdateStatus(ctx context.Context, id string, status ChallengeStatus, at time.Time) error
	UpdateLabel(ctx context.Context, id, label string) error
	UpdateTarget(ctx context.Context, id string, target float64, tier Tier, dataTier DataTier, at time.Time) error
	CountActiveByCadence(ctx context.Context, userID, titleSlug string, cadence Cadence) (int, error)
	CountActiveTotal(ctx context.Context, userID, titleSlug string) (int, error)
	CountCreatedSince(ctx context.Context, userID, titleSlug string, mode ChallengeMode, since time.Time) (int, error)
	// DetachFromArc dissocie tous les défis d'un arc (arc_id = NULL) sans les
	// supprimer — utilisé quand on supprime l'arc en gardant ses objectifs.
	DetachFromArc(ctx context.Context, arcID string) error
	// DeleteByArc supprime physiquement tous les défis d'un arc — utilisé pour
	// l'exemption « arc à peine créé » (zéro trace, donc zéro cooldown).
	DeleteByArc(ctx context.Context, arcID string) error
}

// ChallengeFilter filtre la liste des défis.
type ChallengeFilter struct {
	UserID    string
	TitleSlug string
	Status    *ChallengeStatus  // nil = tous statuts (filtre mono-statut)
	Statuses  []ChallengeStatus // vide = ignoré ; sinon status IN (...) — prioritaire sur Status
	ArcID     *string           // nil = tous, "" interdit (utilise NoArc à la place)
	Mode      *ChallengeMode
	Metric    *string // nil = toutes métriques (filtre cooldown par métrique)
	Limit     int
}

// ---------- ArcRepo (stats.duckdb par joueur) ----------

// ArcRepo gère la persistance des arcs (preset ou libres).
type ArcRepo interface {
	Create(ctx context.Context, a Arc) error
	Get(ctx context.Context, id string) (Arc, error)
	ListByUser(ctx context.Context, userID, titleSlug string) ([]Arc, error)
	MarkCompleted(ctx context.Context, id string, at time.Time) error
	// Delete supprime un arc (conteneur léger sans sémantique PP/cooldown :
	// le hard delete est justifié). Les objectifs sont traités en amont par le
	// service (détachés ou supprimés/abandonnés selon l'option).
	Delete(ctx context.Context, id string) error
}

// ArcTitlesRepo expose la voie cross-titre (table de jointure arc_titles) EN
// LECTURE, sans changer la sémantique mono-titre existante. La nouvelle voie est
// un sur-ensemble strict des lectures `WHERE title_slug = ?` actuelles, qui
// restent fonctionnelles (cf. PLAN_CROSS_TITLE_ARCS_BACKEND Phase 2).
//
// Garde-fou pré-backfill : si arc_titles est vide pour un arc, les deux méthodes
// retombent sur arc.title_slug (titre primaire) — donc identiques à l'historique.
type ArcTitlesRepo interface {
	// ArcTitles retourne les title_slug couverts par un arc (>=1). Fallback sur
	// [arc.title_slug] si la jointure ne contient aucune ligne pour cet arc.
	ArcTitles(ctx context.Context, arcID string) ([]string, error)
	// ArcsByTitle liste les arcs couvrant un titre via arc_titles. En mono-titre
	// (1 ligne par arc), strictement équivalent à l'ancienne requête sur
	// arc.title_slug.
	ArcsByTitle(ctx context.Context, userID, titleSlug string) ([]Arc, error)
}

// ---------- MomentCardRepo (stats.duckdb par joueur) ----------

// MomentCardRepo gère la persistance des cartes générées.
type MomentCardRepo interface {
	Create(ctx context.Context, mc MomentCard) error
	GetByChallenge(ctx context.Context, challengeID string) (MomentCard, error)
	ListRecent(ctx context.Context, userID, titleSlug string, limit int) ([]MomentCard, error)
}

// ---------- PrestigeRepo (shared_social.duckdb, cross-joueurs) ----------

// PrestigeRepo gère les événements PP, totaux et leaderboards.
type PrestigeRepo interface {
	EmitEvent(ctx context.Context, ev PrestigeEvent) error
	GetUserPrestige(ctx context.Context, userID, titleSlug string) (UserPrestige, error)
	GetUserPrestigeCrossTitle(ctx context.Context, userID string) (UserPrestige, error)
	UpsertUserPrestige(ctx context.Context, up UserPrestige) error
	ListEvents(ctx context.Context, userID, titleSlug string, since time.Time) ([]PrestigeEvent, error)
	GetLeaderboard(ctx context.Context, userIDs []string, titleSlug *string, since time.Time) ([]LeaderboardEntry, error)
}

// LeaderboardEntry est une ligne du leaderboard PP.
//
// Affichage décomposé brut/bonus/total (Axe 5). RawScore vient du module
// stats existant (rating brut), BonusScore est calculé par Prestige
// (somme des bonus de paliers complétés sur la période).
type LeaderboardEntry struct {
	UserID     string  `json:"user_id"`
	TitleSlug  string  `json:"title_slug,omitempty"`
	TotalPP    int     `json:"total_pp"`
	Level      Level   `json:"level"`
	RawScore   float64 `json:"raw_score"`
	BonusScore float64 `json:"bonus_score"`
}

// ---------- TelemetryRepo (stats.duckdb par joueur) ----------

// TelemetryRepo gère le journal d'événements pour le calage post-alpha.
type TelemetryRepo interface {
	Emit(ctx context.Context, ev PrestigeTelemetry) error
}

// ---------- BaselineStateRepo (stats.duckdb par joueur) ----------

// BaselineStateRepo gère l'état de fraîcheur des baselines pour le reset 60j.
type BaselineStateRepo interface {
	Get(ctx context.Context, userID, titleSlug, metric string) (BaselineState, error)
	Upsert(ctx context.Context, st BaselineState) error
}

// ---------- TemplateRepo (metadata.duckdb) ----------

// TemplateRepo gère le catalogue de défis pré-calibrés.
type TemplateRepo interface {
	ListByTitle(ctx context.Context, titleSlug string) ([]Template, error)
	GetByID(ctx context.Context, id string) (Template, error)
	Suggest(ctx context.Context, titleSlug string, excludeIDs []string, count int) ([]Template, error)
	Replace(ctx context.Context, titleSlug string, templates []Template) error
	// UpsertOne insère ou met à jour un seul template (par ID). Sémantique
	// identique à Replace mais explicite pour l'usage isolé (notamment par
	// coach_advisor.Synthesizer qui persiste un nouveau template synthétisé).
	UpsertOne(ctx context.Context, template Template) error
}

// ---------- PresetArcRepo (metadata.duckdb) ----------

// PresetArcRepo gère le catalogue d'arcs preset.
type PresetArcRepo interface {
	ListByTitle(ctx context.Context, titleSlug string) ([]PresetArc, error)
	GetByID(ctx context.Context, id string) (PresetArc, error)
	GetSteps(ctx context.Context, presetArcID string) ([]PresetArcStep, error)
	Replace(ctx context.Context, titleSlug string, arcs []PresetArc, steps []PresetArcStep) error
}

// ---------- SquadChallengeRepo (shared_social.duckdb) ----------

// SquadChallengeRepo gère les défis d'escouade.
type SquadChallengeRepo interface {
	Create(ctx context.Context, sc SquadChallenge) error
	Get(ctx context.Context, id string) (SquadChallenge, error)
	ListBySquad(ctx context.Context, squadID string) ([]SquadChallenge, error)
	// Archive marque un défi comme archivé (abandon). ListBySquad ne renvoie
	// plus les défis archivés. Idempotent.
	Archive(ctx context.Context, id string) error

	AddParticipant(ctx context.Context, p SquadChallengeParticipant) error
	UpdateParticipantProgress(ctx context.Context, challengeID, userID string, value float64, completedAt *time.Time) error
	ListParticipants(ctx context.Context, challengeID string) ([]SquadChallengeParticipant, error)
	CountActiveParticipants(ctx context.Context, challengeID string) (int, error)
}

// ---------- SquadRepo (shared_social.duckdb) ----------

// SquadRepo gère les groupes Prestige.
//
// Membres clés par xuid (cf. SquadMember). RemoveMember prend donc un xuid.
// ListSquadsForUser prend un userID = player_slug : ne remonte que les escouades
// où le membre est un utilisateur de l'app (squad_member.user_id renseigné).
type SquadRepo interface {
	Create(ctx context.Context, s Squad) error
	Get(ctx context.Context, id string) (Squad, error)
	// Rename change le nom d'une escouade (UPDATE basse fréquence sur squad).
	Rename(ctx context.Context, id, name string) error
	AddMember(ctx context.Context, m SquadMember) error
	RemoveMember(ctx context.Context, squadID, xuid string) error
	ListMembers(ctx context.Context, squadID string) ([]SquadMember, error)
	ListSquadsForUser(ctx context.Context, userID string) ([]Squad, error)
}
