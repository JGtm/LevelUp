// Package domain — match_rows.go : structs neutres représentant les rows
// DB des tables critiques (shared.match_registry, shared.match_participants,
// shared.medals_earned, shared.weapon_kills, etc.).
//
// **Localisation domain** : ces structs sont utilisées par PLUSIEURS packages
// — `internal/sync/` (collecte + parsing depuis l'API Halo), `internal/persist/`
// (batch persistence). Pour éviter les cycles d'import sync ⇄ persist,
// elles vivent ici en zone neutre.
//
// Refactor 2026-05-23 (refactor/collect-persist) — déplacé depuis
// `internal/sync/transforms.go` et `internal/sync/writes.go`. Les anciens
// emplacements conservent des type aliases pour préserver les callers
// existants jusqu'à la fin du refactor.

package domain

import "time"

// MatchRegistryRow représente une ligne dans shared.match_registry.
//
// Champs PK : MatchID.
// Le champ SeasonID est NULL pour les anciens matchs (la migration
// `shared_backfill_is_ranked_and_season` populate via dérivation start_time).
type MatchRegistryRow struct {
	MatchID                 string
	StartTime               time.Time
	EndTime                 *time.Time
	PlaylistID              *string
	PlaylistName            *string
	PlaylistVersionID       *string
	MapID                   *string
	MapName                 *string
	MapVersionID            *string
	PairID                  *string
	PairName                *string
	PairVersionID           *string
	GameVariantID           *string
	GameVariantName         *string
	GameVariantVersionID    *string
	ModeCategory            string
	IsRanked                bool
	IsFirefight             bool
	DurationSeconds         *int
	PlayableDurationSeconds *int
	RealStartTime           *time.Time
	Team0Score              *int
	Team1Score              *int
	Team0PSScore            *int // somme des PersonalScore équipe 0
	Team1PSScore            *int // somme des PersonalScore équipe 1

	// Team0RoundsWon / Team1RoundsWon / RoundsTotal — les MANCHES du match
	// (`CoreStats.RoundsWon/RoundsLost/RoundsTied`). Sur un mode qui se décide aux
	// manches, Team{0,1}Score est un cumul de points qui ne dit PAS le résultat (4 matchs
	// Oddball du corpus donnent la victoire à l'équipe qui a le moins de points, cf.
	// `.ai/V7.5/RAPPORT_MANCHES_2026-08-29.md`). RoundsTotal est le MAX des totaux des deux
	// camps (un abandon crédite 1 manche à un seul camp). Nil = inconnu (ligne antérieure au
	// backfill, FFA, camp absent) → le lecteur retombe sur les points ; jamais un zéro
	// substitué, qui se lirait « zéro manche gagnée ».
	Team0RoundsWon *int
	Team1RoundsWon *int
	RoundsTotal    *int
	FirstSyncBy    string
	// SeasonID est l'identifiant CSR de la saison du match (ex. "CsrSeason13-1").
	// Lu depuis matchInfo["SeasonId"] (payload Halo officiel). Permet le lookup
	// threshold dynamique côté display (cf. csr_placement_thresholds).
	SeasonID *string

	// MatchIntensity (DOUBLE) — events/min/joueur du lobby. Caractéristique
	// permanente du match (calculée 1 fois, ne change pas). Set au moment de
	// l'INSERT par le sync engine quand les events sont déjà fetched.
	// Cf. internal/migration/steps_engagement.go.
	MatchIntensity *float64

	// PlayerCount (SMALLINT) — TAILLE du roster reporté par l'API pour ce match
	// (nb de joueurs dans le carnage/scoreboard), AVANT tout resolve-or-skip xuid.
	// Oracle d'intégrité : roster attendu vs lignes match_participants réellement
	// persistées (un écart = joueurs droppés faute de résolution xuid). 0 = inconnu
	// (titre/chemin qui ne le renseigne pas) → la colonne reste à son DEFAULT 0.
	PlayerCount int

	// BackfillCompleted (INTEGER, bitmask) — état d'avancement du backfill pour
	// ce match. En mode INSERT-only (Phase 1 refactor Collect→Persist), la
	// valeur finale doit être calculée AVANT le Submit du batch (cumulative
	// OR sur tous les MBit* positionnés pendant la collecte). Plus de UPDATE
	// post-coup possible. Cf. internal/sync/writes.go::MBit* constantes.
	BackfillCompleted *int64
}

// MatchParticipantRow représente une ligne COMPLÈTE dans
// shared.match_participants (30 colonnes), utilisée pour les INSERT depuis
// l'API Halo.
//
// **Distinction** avec `domain.ParticipantRow` (déclaré dans stats.go) :
// celui-ci est le type MINIMAL (5 colonnes) utilisé par `internal/analysis/`
// pour le compute LUSR. Les deux coexistent — ne pas les confondre.
//
// Champs PK : (MatchID, XUID).
// Tous les champs métier sont pointers pour permettre le COALESCE / l'absence
// (l'API Halo ne retourne pas toujours toutes les valeurs, ex. shots_fired
// pour les vieux matchs).
type MatchParticipantRow struct {
	MatchID           string
	XUID              string
	Gamertag          *string
	TeamID            *int
	Outcome           *int
	Rank              *int
	Score             *int
	Kills             *int
	Deaths            *int
	Assists           *int
	ShotsFired        *int
	ShotsHit          *int
	DamageDealt       *float64
	DamageTaken       *float64
	KDA               *float64
	Accuracy          *float64
	PersonalScore     *int
	TimePlayedSeconds *int
	AvgLifeSeconds    *float64
	KillsExpected     *float64
	DeathsExpected    *float64
	KillsStddev       *float64
	TeamMMR           *float64
	EnemyMMR          *float64
	HeadshotKills     *int
	MaxKillingSpree   *int
	GrenadeKills      *int
	MeleeKills        *int
	PowerWeaponKills  *int

	// Mécaniques de kill NATIVES Halo 5 (assassinats + compétences spartiate).
	// Agrégats par-joueur du carnage h5, DISJOINTS de MeleeKills. nil pour les
	// autres titres (Infinite ne les fournit pas). Cf. halo_5/dto_carnage.go.
	AssassinationKills *int
	GroundPoundKills   *int
	ShoulderBashKills  *int

	DeathsStddev *float64

	// BackfillBits (INTEGER, bitmask) — état des colonnes par joueur×match
	// (PBitTeamMMR, PBitEnemyMMR, PBitKillsExp, PBitDeathsExp, PBitAccuracy,
	// PBitShots, PBitDamage, PBitAvgLife, PBitMedals, PBitKDA, PBitTimePlayed,
	// PBitKillerVictim). Set à l'INSERT en mode Collect→Persist (calculé par
	// le collecteur au moment où il sait quelles colonnes sont fiables).
	BackfillBits *int

	// ParticipationInfo booleans (LUSR v2 — Mini-Phase 0.5).
	// NULL pour les matchs antérieurs à la capture.
	// quit_real = PresentAtBeginning && !PresentAtCompletion
	// late_join = JoinedInProgress
	PresentAtBeginning  *bool
	PresentAtCompletion *bool
	JoinedInProgress    *bool
	LeftInProgress      *bool

	// ParticipationInfo timestamps (LUSR v2 Phase 3-quit — 2026-05-27).
	// Permet d'ordonner précisément les quitters d'un match (1er parti =
	// LastLeaveTime ASC). NULL pour les anciens matchs non backfillés.
	FirstJoinedTime *time.Time
	LastLeaveTime   *time.Time
}

// MedalRow représente une ligne dans shared.medals_earned.
//
// Champs PK : (MatchID, XUID, MedalNameID).
type MedalRow struct {
	MatchID     string
	XUID        string
	MedalNameID int64
	Count       int
}

// WeaponKillRow représente une ligne dans shared.weapon_kills.
//
// Pas de PK explicite (table append-only par (MatchID, XUID, TimeMS)).
// WeaponID et ReconciledAs sont des UBIGINT (uint64) — certains IDs Halo
// dépassent 2^63, donc on les sérialise en string décimale au moment de
// l'INSERT puis cast côté DuckDB pour préserver la valeur exacte.
type WeaponKillRow struct {
	TimeMS          int
	WeaponID        *uint64
	ReconciledAs    *uint64
	DeltaMS         *int
	Confidence      string
	AttributionPath string
	SwapDetected    bool
	DelayedDamage   bool
	PlayerIndex     *int
}
