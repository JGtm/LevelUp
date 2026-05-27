package domain

import "time"

// skill_v2.go — types LUSR v2 (TrueSkill 2 inspired).
//
// Coexistence avec le LUSR v1 (cf. LUSRMatchRating dans match_skill_rank) :
// suffixe _V2 partout, tables séparées (player_skill_state_v2, lusr_hyperparams_v2),
// aucune dépendance croisée. Voir internal/analysis/skill_v2/doc.go pour la math.

// SkillV2State représente le posterior courant d'un joueur sur un groupe de modes.
// Une row dans player_skill_state_v2 (append-only) ; la lecture passe par la vue
// player_skill_state_v2_latest qui filtre sur written_at max par (xuid, playlist_group).
type SkillV2State struct {
	XUID          string
	PlaylistGroup string
	Mu            float64
	Sigma         float64
	// Experience : # de matchs LUSR-éligibles joués dans ce groupe avant ce snapshot.
	// Utilisé par Phase 2 (experienceOffset TS2 §7) ; en Phase 1 c'est purement informatif.
	Experience    int
	LastMatchID   *string
	LastMatchAt   *time.Time
	WrittenAt     time.Time
}

// SkillV2Hyperparam représente un paramètre global appris (ou figé par défaut)
// pour un groupe de modes : Mu0, Sigma0, Beta, Tau, DrawProbability (Phase 1)
// puis squadOffset[k], experienceOffset[k], w_p^kill etc. (Phase 2+).
//
// Modèle clé/valeur volontairement plat plutôt qu'une struct riche par groupe :
// (a) facile à dumper/importer en CSV pour QA, (b) ajout d'un nouveau param =
// nouvelle row, pas de migration de schéma, (c) source explicite pour distinguer
// les defaults "halo5_default" d'un éventuel batch "batch_2026_06".
type SkillV2Hyperparam struct {
	PlaylistGroup string
	Name          string
	Value         float64
	Source        string
	WrittenAt     time.Time
}
