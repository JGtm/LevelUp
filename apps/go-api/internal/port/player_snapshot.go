package port

// player_snapshot.go — la capture de l'état d'un joueur avant et après une synchronisation, pour détecter ce
// qui a changé (rang, awards, défis, niveaux de classement, médailles...).

import (
	"context"
	"time"
)

// PlayerSnapshot : l'état d'un joueur avant ou après une synchronisation, pour détecter ce qui a changé.
type PlayerSnapshot struct {
	// Rang carrière Halo lifetime (career_progression).
	CurrentRank     int
	CurrentRankName string

	// Awards / objectifs personnels.
	PersonalAwardCount int

	// Total brut de citations (legacy — utilisé pour la rétro-compat des tests existants, plus émis comme
	// challenge_completed depuis 2026-05-16).
	CitationsCount int

	// Défis daily/weekly (challenge_snapshots).
	ChallengePathsCount     int // nb challenge_path distinct connus
	ChallengeCompletedCount int // nb challenge_path dont le dernier status = 'Completed'

	// CSR / LUSR : dernière entrée par playlist_group dans match_skill_rank.
	// clé = playlist_group, valeur = "rating_type|tier|sub_tier".
	SkillTierByPlaylist map[string]string

	// Battle pass : tracks ayant atteint leur rang max (has_reached_max_rank=TRUE dans le dernier snapshot par
	// track).
	BattlepassCompletedTracks int

	// Citations / commendations agrégées via CitationsService.
	CitationTotalEarnedTiers int // somme des EarnedTiers sur toutes les citations
	CitationMasteryCount     int // nb de citations avec MasteryPct >= 100

	// Métriques agrégées.
	KDRatio        float64 // KD agrégé sur tous les matchs ingérés
	Winrate        float64 // 0..1 — fraction de matchs gagnés
	BestKDA        float64 // record matériel (kills+assists)/max(deaths,1) sur 1 match
	BestKDAMatchID string  // match associé au record

	// LastMatchStartTime : start canonique (UTC) du match le plus récent du joueur dans le shared. Watermark de
	// la détection « rival croisé » : after > before ⇒ nouveau match ⇒ détection ; sinon SKIP (coût nul sur
	// les syncs à vide). Zéro si aucun match (aucune détection possible).
	LastMatchStartTime time.Time

	// EarnedMedalIDs : ensemble des medal_name_id à SUM(count) > 0 pour le joueur dans le shared du titre.
	// Base de la détection « médaille inédite » : after \ before = médailles décrochées pour la première
	// fois. nil/vide si medals_earned est absente (titre sans la capability) ou illisible → la garde
	// cold-start sème alors sans notifier.
	EarnedMedalIDs map[int64]struct{}
}

// PlayerSnapshotReader remplit un PlayerSnapshot depuis les bases d'UN joueur. Les lectures sont best-effort :
// une table vide, absente ou illisible laisse les champs à zéro (journalisé au niveau Debug).
type PlayerSnapshotReader interface {
	// FillPlayerCounters remplit ce que la base du joueur sait : rang carrière, awards, citations, défis,
	// niveaux de classement, battle pass.
	FillPlayerCounters(ctx context.Context, s *PlayerSnapshot)
	// FillSharedAggregates remplit ce que la base partagée sait du joueur : KD, taux de victoire, meilleur
	// KDA, dernier match, médailles obtenues. Sans joueur identifié ou sans lecteur partagé, rien n'est lu.
	FillSharedAggregates(ctx context.Context, s *PlayerSnapshot)
}
