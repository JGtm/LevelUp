// Package persist — batch.go : structs représentant un batch atomique à
// persister (1 match complet par convention §6.bis du design).

package persist

import (
	"time"

	"levelup/go-api/internal/domain"
)

// DBTarget identifie la DB cible d'un sous-batch. Permet au BatchQueue de
// router vers le bon worker.
type DBTarget string

const (
	// TargetShared : shared_matches_v2.duckdb (registry, participants, medals,
	// events, weapon_kills, killer_victim, xuid_aliases).
	TargetShared DBTarget = "shared"

	// TargetPlayer : stats.duckdb d'UN joueur (enrichments, skill_rank,
	// citations, awards, career_progression, sessions).
	TargetPlayer DBTarget = "player"

	// TargetPVE : shared_pve.duckdb (Firefight stats).
	TargetPVE DBTarget = "pve"

	// TargetMetadata : metadata.duckdb (rare en sync, ex. mode_name_tr update).
	TargetMetadata DBTarget = "metadata"
)

// MatchBatch est l'unité atomique de persistance — 1 match complet
// (data API + tous les enrichments locaux computed). Cf. §6.bis du design.
//
// Property critique : si un MatchBatch est persisté, le match est
// entièrement complet (pas d'état partiel). Si crash mid-persist, la
// transaction rollback préserve l'atomicité.
//
// Sérialisable JSON pour le WAL (durabilité crash).
type MatchBatch struct {
	// Métadonnées identifiant le batch et son origine.
	BatchID   string    `json:"batch_id"`   // UUID v4 généré par BatchBuilder
	TitleSlug string    `json:"title_slug"` // ex. "halo_infinite" (multi-titres ready)
	Player    string    `json:"player"`     // gamertag (route vers le bon player DB)
	XUID      string    `json:"xuid"`
	CreatedAt time.Time `json:"created_at"`
	Source    string    `json:"source"` // "sync_delta", "backfill_cli", "manual_recompute", etc.

	// Données pour shared_matches_v2.duckdb.
	Shared SharedBatch `json:"shared"`

	// Données pour stats.duckdb du joueur (Player).
	PlayerData PlayerBatch `json:"player_data"`

	// Données pour shared_pve.duckdb (nil si match non-Firefight).
	PVE *PVEBatch `json:"pve,omitempty"`

	// Données pour metadata.duckdb (rare, nil si pas d'update mappings).
	Metadata *MetadataBatch `json:"metadata,omitempty"`
}

// SharedBatch contient les écritures pour shared_matches_v2.duckdb
// liées à 1 match.
type SharedBatch struct {
	// Match : la row registry du match (1 seule).
	// Nil si on ne crée pas de nouveau match (cas rare : recompute pur).
	Match *domain.MatchRegistryRow `json:"match,omitempty"`

	// Participants : les 10-16 participants du match (1 par joueur).
	Participants []domain.MatchParticipantRow `json:"participants,omitempty"`

	// Medals : médailles gagnées par tous les participants du match.
	Medals []domain.MedalRow `json:"medals,omitempty"`

	// HighlightEvents : events depuis le film (kill, death, etc.).
	HighlightEvents []HighlightEventInsert `json:"highlight_events,omitempty"`

	// WeaponKills : attributions kills par arme depuis le film.
	WeaponKills []WeaponKillInsert `json:"weapon_kills,omitempty"`

	// WeaponAccuracy : tirs par arme agrégés par (match, joueur, arme) — précision
	// par arme reconstruite des events WeaponDrop (Halo 5 ; carnage WeaponStats vide).
	WeaponAccuracy []WeaponAccuracyInsert `json:"weapon_accuracy,omitempty"`

	// KillerVictim : agrégats killer→victim pour le match.
	KillerVictim []KillerVictimInsert `json:"killer_victim,omitempty"`

	// KillPositions : positions monde tueur/victime par kill (Halo 5 natif,
	// Infinite plus tard). Jointes au kill par (match_id, killer_xuid, time_ms).
	KillPositions []KillPositionInsert `json:"kill_positions,omitempty"`

	// XUIDAliases : nouveaux mappings xuid→gamertag observés dans le match.
	XUIDAliases []XUIDAliasInsert `json:"xuid_aliases,omitempty"`

	// Commendations : commendations NATIVES progressées sur le match (Halo 5
	// natif, AXE B). 1 row par (xuid, commendation_id) avec le compte gagné CE
	// match. Vide pour les titres sans commendations natives (Infinite).
	Commendations []CommendationInsert `json:"commendations,omitempty"`

	// MatchCSRs : CSR de tous les participants d'un match ranked (lobby
	// context pour match view / compare). Vide pour les matchs non-ranked.
	MatchCSRs []MatchCSRInsert `json:"match_csrs,omitempty"`
}

// PlayerBatch contient les écritures pour stats.duckdb (du joueur Player).
type PlayerBatch struct {
	// Enrichment : 1 row complète pour ce match (tous les enrichments
	// computed dans le même struct — performance, dominance, engagement,
	// session, is_with_friends, etc.).
	Enrichment *EnrichmentRow `json:"enrichment,omitempty"`

	// SkillRank : rating LUSR ou CSR pour ce match (1 row, PK=match_id).
	SkillRank *SkillRankInsert `json:"skill_rank,omitempty"`

	// LUSRComponents : breakdown des 8 composantes (si LUSR computed).
	LUSRComponents []LUSRComponentInsert `json:"lusr_components,omitempty"`

	// Citations : citations gagnées pour ce match.
	Citations []CitationInsert `json:"citations,omitempty"`

	// PersonalScoreAwards : awards PersonalScore API pour ce match.
	PersonalScoreAwards []PersonalScoreAwardInsert `json:"personal_score_awards,omitempty"`

	// CareerProgression : snapshot rang post-match (nil si rang inchangé).
	CareerProgression *CareerProgressionInsert `json:"career_progression,omitempty"`

	// Session : assignation session pour ce match (1 row).
	Session *SessionInsert `json:"session,omitempty"`
}

// PVEBatch — Firefight stats (1 match Firefight). Une row par participant
// (PK shared_pve.pve_match_stats = (match_id, xuid)). Le sync collecte tous
// les participants présents dans le payload Halo, pas seulement le joueur sync.
type PVEBatch struct {
	Stats []PVEMatchStatsInsert `json:"stats,omitempty"`
}

// MetadataBatch — rare en sync, ex. nouvelles traductions mode/playlist.
type MetadataBatch struct {
	ModeNameTranslations []ModeNameTranslationInsert `json:"mode_name_translations,omitempty"`
	// (extensible : autres updates metadata)
}
