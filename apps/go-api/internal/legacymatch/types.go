// Package legacymatch contient les types de match transitionnels utilisés par
// les helpers d'analyse et les repos legacy. P4.3 finale (ADR 0011).
//
// Ces types ont été déplacés depuis `internal/domain` pour libérer le package
// domain de ses artefacts transitionnels. Ils restent utilisés par :
//   - Analyses legacy (BuildHeroCard, ComputeKPIs, build*Tab, etc.) qui sont
//     du code mort en path canonical actif mais consomment encore ces types.
//   - Helpers internes squad/teammates (extractSynthesisSessionLabels,
//     filterSynthesisByCascade, etc.) qui reçoivent des rows converties depuis
//     canonical via `analysis.{Stats,Synthesis,Home}MatchRowsFromCanonical`.
//   - Repos legacy (`platform/duckdb/{home,stats,squad,synthesis}_repo.go`)
//     qui exposent encore Load*Matches comme public API non-port pour les
//     tests d'intégration.
//
// Sprint cleanup post-P4.3 : ces types seront supprimés quand les helpers
// seront portés full canonical et les analyses legacy retirées.
package legacymatch

import "time"

// HomeMatchRow est une ligne brute chargée depuis Q26 (matchs du home).
type HomeMatchRow struct {
	MatchID   string
	StartTime time.Time
	MapID     string
	MapName   string
	MapNameFR string
	// MapImageURL est résolue par HomeRepo depuis map_images_registry
	// (lookup par map_id, pattern asset kinds). Empty si pas d'entrée
	// dans le registry — l'analysis layer émet alors nil et le frontend
	// dégrade gracieusement (placeholder map inconnue).
	MapImageURL        string
	PairID             string
	PairName           string
	PairNameFR         string
	GameVariantID      string
	GameVariantName    string
	GameVariantNameFR  string
	PlaylistID         string
	PlaylistName       string
	PlaylistNameFR     string
	IsFirefight        bool
	IsRanked           bool
	SessionLabel       *string
	IsWithFriends      bool
	Outcome            int
	TeamID             int
	Team0Score         int
	Team1Score         int
	DominanceFlag      int
	Kills              int
	Deaths             int
	Assists            int
	KDA                *float64
	Ratio              *float64
	Accuracy           *float64
	AvgLifeSeconds     *float64
	TimePlayedSecs     *int
	DamageDealt        *float64
	DamageTaken        *float64
	TeamMMR            *float64
	EnemyMMR           *float64
	PerformanceScore   *float64
	SkillRatingValue   *float64
	SkillRatingType    string
	SkillTier          *string
	SkillSubTier       int
	SkillTierLabel     *string
	SkillRatingDelta   *float64
	SkillPlaylistGroup *string
	SkillRankImageURL  *string
	RankInTeam         *int
	HeadshotKills      int
	PerfectKills       int
	MaxKillingSpree    *int
}

// HomeSessionRow est une ligne brute chargée depuis Q27 (sessions enrichment).
type HomeSessionRow struct {
	MatchID       string
	SessionID     *int
	SessionLabel  *string
	IsWithFriends bool
	StartTime     *time.Time
}

// StatsMatchRow est le type de transfert entre platform/duckdb et les services de stats.
// Contient toutes les métriques nécessaires au calcul du performance score (Q23).
type StatsMatchRow struct {
	MatchID           string
	StartTime         time.Time
	Outcome           *int
	Kills             int
	Deaths            int
	Assists           int
	KDA               *float64
	Accuracy          *float64
	PersonalScore     *int
	DamageDealt       *float64
	DamageTaken       *float64
	TimePlayedSeconds *int
	AvgLifeSeconds    *float64
	TeamMMR           *float64
	EnemyMMR          *float64
	KillsExpected     *float64
	DeathsExpected    *float64
	// ShotsHit : tirs au but du match (match_participants.shots_hit). Alimente le
	// fallback populationnel du modèle d'assists attendus (slope × (personal_score +
	// shots_hit) + intercept). nil si non chargé.
	ShotsHit       *int
	Rank           *int
	IsRanked       bool
	IsFirefight    bool
	IsWithFriends  bool
	PlaylistName   string
	PlaylistNameFR string
	PairName       string
	PairNameFR     string
	// GameVariant : source de repli pour le mode FR (les game_variant sont localisés,
	// contrairement à asset_translations[pair]). Cf. buildSessionDetailRows.
	GameVariantName     string
	GameVariantNameFR   string
	MapName             string
	MapNameFR           string
	TeamID              *int
	PerfScoreComputed   *float64
	SessionID           *string
	SessionLabel        *string
	MedalExploitScore   *float64
	OffensiveConversion *float64
	DefensiveResistance *float64
	MaxKillingSpree     *int
	HeadshotKills       *int
	PerfectKills        *int
	// Kill-type breakdown (donut « répartition des frags » timeseries) : types
	// d'arme de base + mécaniques natives Halo 5 (assassinats + compétences
	// spartiate). Les 3 mécaniques sont nil hors h5.
	MeleeKills         *int
	GrenadeKills       *int
	PowerWeaponKills   *int
	AssassinationKills *int
	GroundPoundKills   *int
	ShoulderBashKills  *int
	// SkillRatingValue : rating CSR ou LUSR du match (depuis match_skill_rank).
	// Nil si le titre/match n'a pas de skill snapshot.
	SkillRatingValue          *float64
	SkillRatingType           string   // "csr" | "lusr" | ""
	SkillPlaylistGroup        *string  // groupe normalisé (ex: "ranked-arena")
	SkillSeasonID             *string  // saison Halo (ex: "Elan") — rupture de courbe si changement
	SkillMeasurementRemaining *int     // matchs de placement restants (>0 = placement)
	SkillRatingDelta          *float64 // gain/perte de rating du match (depuis SkillSnapshot.Delta)
	SkillExpectedWinProb      *float64 // proba de victoire pré-match ∈ [0,1] (LUSR v2, SkillSnapshot.ExpectedWinProb)
	// Codes de palier (pour reconstruire le libellé "Or III" comme l'Explorer) :
	SkillTierCode       *string  // code EN stable (ex: "gold", "diamond", "onyx")
	SkillTierCodeFR     *string  // libellé FR du tier (ex: "Or", "Diamant") depuis match_skill_rank.tier_fr
	SkillSubTier        *int     // 1..6, nil pour Onyx
	EngagementScoreBrut *float64 // résidu brut engagement, nil si non calculé
	EngagementPaceRatio *float64 // engagement absolu = pace_joueur/pace_lobby, nil si indispo
}

// SynthesisMatchRow est une ligne brute chargée depuis Q33b.
type SynthesisMatchRow struct {
	MatchID        string
	StartTime      time.Time
	Outcome        int
	Kills          int
	Deaths         int
	KDA            *float64
	IsWithFriends  bool
	Accuracy       *float64
	TimePlayedSecs *int
	// AvgLifeSeconds : durée de vie moyenne (valeur API match_participants),
	// PAS un dérivé de time_played/n. Nil si non chargée. Cf. squad_breakdown.go.
	AvgLifeSeconds   *float64
	PerformanceScore *float64
	SessionLabel     *string
	IsRanked         bool
	IsFirefight      bool
	PlaylistName     string
}
