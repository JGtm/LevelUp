package canonical

import "time"

// AssetReference est l'objet canonique pour les assets Halo référencés par
// le produit (maps, playlists, modes, ranks, médailles).
type AssetReference struct {
	Kind         string            // "map", "playlist", "game_variant", "career_rank", ...
	ID           string            // identifiant source stable
	VersionID    string            // optionnel
	DefaultLabel string            // libellé par défaut sans i18n locale avancée
	Labels       map[string]string // locale -> label
	IconURL      string
}

// MatchSummary est la version légère d'un match pour un historique paginé.
type MatchSummary struct {
	MatchID         string
	StartedAtUTC    time.Time
	DurationSeconds *int
	MatchType       MatchType
	Playlist        *AssetReference
	Map             *AssetReference
	GameVariant     *AssetReference
	// PairMode est la catégorie humaine du mode (ex: "Slayer", "CTF") extraite
	// de pair_name / pair_name_fr. Distinct de GameVariant (variante technique
	// avec préfixe titre, ex: "HaloMultiplayer:Slayer"). Nil si non disponible.
	PairMode *AssetReference
	IsRanked *bool
	IsPvE    *bool
	Outcome  Outcome
	// Teams est le snapshot léger des scores d'équipes pour l'affichage en
	// liste / hero card (ADR 0011, P4.1). MatchDetail expose la même info
	// avec ParticipantsXUIDs en plus ; ici on garde une version sans les
	// participants. Nil si non chargé / non team-based.
	Teams []TeamSnapshot

	// T0Ms est l'offset en millisecondes du countdown pré-match (Match Timeline
	// T0, Phase 3) : durée entre le start_time officiel du match et le début
	// réel du gameplay. Dérivé en SQL de
	// `epoch_ms(real_start_time AT TIME ZONE 'UTC') − epoch_ms(start_time_utc)`.
	// Nil si real_start_time absent (T0 non calculable) → les builders timeline
	// retombent sur T0=0 (chronologie brute, comportement pré-Phase 3).
	// Champ additif (politique d'évolution canonical, cf. ADR 0005).
	T0Ms *int64
}

// GameplayDurationSeconds retourne la VRAIE durée de gameplay (countdown
// pré-match retranché) : DurationSeconds − T0Ms/1000, bornée à ≥ 0. Source
// unique pour l'affichage "durée du match" homogène (vs durée brute du film).
// Retourne nil si DurationSeconds est nil. Si T0Ms est nil (countdown inconnu),
// retombe sur la durée brute.
func (m MatchSummary) GameplayDurationSeconds() *int {
	if m.DurationSeconds == nil {
		return nil
	}
	gp := *m.DurationSeconds
	if m.T0Ms != nil {
		gp -= int(*m.T0Ms / 1000)
	}
	if gp < 0 {
		gp = 0
	}
	return &gp
}

// MatchDetail est l'objet canonique central d'un match côté services.
type MatchDetail struct {
	MatchID      string
	StartedAtUTC time.Time
	EndedAtUTC   *time.Time
	Playlist     *AssetReference
	Map          *AssetReference
	GameVariant  *AssetReference
	IsRanked     *bool
	IsPvE        *bool
	MatchType    MatchType
	Participants []MatchParticipant
	Teams        []TeamSnapshot
	Skill        *MatchSkillSnapshot
	Limitations  []CapabilityGap

	// Commendations : commendations NATIVES progressées sur ce match (Halo 5 natif,
	// AXE B prod-gate). Affichage NATIF tel quel (PAS le moteur de citations dérivé
	// d'Infinite) — chaque entrée porte le compte gagné CE match. Vide pour les
	// titres sans commendations natives par match (Infinite). Centré sur le viewer
	// (le joueur dont la consultation déclenche le détail).
	Commendations []Commendation
}

// Commendation est une commendation NATIVE Halo 5 progressée sur un match (AXE B).
// Donnée brute affichée TELLE QUELLE (pas de reconstruction par tier/composite).
type Commendation struct {
	// ID = UUID natif de la commendation (clé naturelle h5). Toujours présent.
	ID string
	// Count = nombre de progressions gagnées CE match (Progress − PreviousProgress).
	Count int
	// Name = libellé natif si une définition est connue (metadata h5), sinon
	// "" → le consommateur dégrade en "Commendation {ID}" (donnée brute suffit en
	// Phase 1, cf. AXE B : fetch des définitions natives = suite).
	Name string
	// IconURL = icône native si connue, sinon nil (Phase 1 : pas de définitions).
	IconURL *string
	// Progress = total cumulatif À VIE de la commendation À CE match (valeur absolue
	// de 343, shared.match_commendations.progress). Permet de dériver le palier
	// atteint + masterisé (parité Cumulative des citations Infinite). 0 si la donnée
	// native ne le porte pas (lignes pré-migration / source live) → anneau vide.
	Progress int
	// TierTargets = CSV croissant des seuils de paliers de la commendation
	// (commendation_definitions.tier_targets, ex. "5,15,30,55,105"). Vide → pas de
	// paliers connus (Meta/Daily ou définition non seedée) → ProgressPct=0.
	TierTargets string
}

// CommendationDefinition — référentiel natif (nom + icône CDN + catégorie) d'une
// commendation, par UUID. Résolu depuis la metadata par titre (Halo 5 : table
// commendation_definitions, API Metadata officielle /commendations) pour enrichir les
// Commendation brutes (AXE B définitions natives). ID = UUID = Commendation.ID.
type CommendationDefinition struct {
	ID       string
	Name     string
	IconURL  string
	Category string // ex. "MULTIPLAYER", "GAME MODE" (groupe d'affichage des totaux)
}

// CommendationTotal — total À VIE d'une commendation NATIVE pour un joueur : le
// progress ABSOLU du match le plus récent (AXE B totaux). ID = UUID natif ; Name /
// IconURL / Category résolus via CommendationDefinition (vides si définition absente).
type CommendationTotal struct {
	ID       string
	Name     string
	Category string
	Total    int
	IconURL  *string
}

// MatchParticipant représente un joueur d'un match dans le canonique.
type MatchParticipant struct {
	Identity        PlayerIdentity
	TeamID          *int
	RankInMatch     *int
	Outcome         Outcome
	Score           *int
	Kills           *int
	Deaths          *int
	Assists         *int
	HeadshotKills   *int
	Accuracy        *float64
	DamageDealt     *int
	DamageTaken     *int
	ShotsFired      *int
	ShotsHit        *int
	MaxKillingSpree *int
	PersonalScore   *int

	// Champs étendus scoreboard (null si non chargés par LoadMatchScoreboard)
	KDA              *float64
	TimePlayed       *int
	MeleeKills       *int
	GrenadeKills     *int
	PowerWeaponKills *int
	AvgLifeSeconds   *float64
	PerfectKills     *int
	TopWeaponID      *string // effective_weapon_id converti en string
	IsBot            *bool

	// Mécaniques de kill NATIVES Halo 5 (assassinats + compétences spartiate :
	// ground pound, shoulder bash). nil hors h5 (capability-gated à l'affichage).
	AssassinationKills *int
	GroundPoundKills   *int
	ShoulderBashKills  *int
}

// HighlightEvent est un événement horodaté issu de highlight_events.
//
// Étendu en Phase 0 méta-plan (loader unifié `port.HighlightEventsRepository`)
// pour permettre une consommation unifiée par Squad (impact 8 rôles), MatchView
// (kill feed, cadence, dominance), Timeseries (first events rolling, intensity,
// cadence). Les nouveaux champs sont additifs : un consommateur de l'ancienne
// API (EventType, TimeMS, XUID) reste valide.
type HighlightEvent struct {
	EventType string // valeurs canoniques : voir HighlightEventType (canonical/enums.go)
	TimeMS    int64

	// XUID est conservé pour compatibilité ascendante. Convention historique :
	// pour un event "kill", XUID = tueur ; pour "death", XUID = victime.
	//
	// Deprecated: utiliser KillerXUID / VictimXUID / PlayerXUID selon le sens
	// sémantique du EventType. Le champ sera retiré en v(N+2) après migration
	// des consommateurs (cf. docs/adr/0005-canonical-player-match-row-evolution.md).
	XUID string

	// MatchID rattache l'event à un match précis. Permet de batcher la
	// récupération des events sur N matchs sans perdre l'origine.
	MatchID string

	// KillerXUID est renseigné pour les events impliquant un tueur identifié
	// (kill, finisher, first_kill, clutch). Nil sinon.
	KillerXUID *string

	// VictimXUID est renseigné pour les events impliquant une victime identifiée
	// (kill, death, finisher, first_kill, first_death). Nil sinon.
	VictimXUID *string

	// PlayerXUID est renseigné pour les events centrés sur un joueur sans
	// notion tueur/victime (medal, assist). Nil sinon.
	PlayerXUID *string

	// WeaponID identifie l'arme utilisée pour un event "kill" / "finisher" si
	// l'information est disponible.
	WeaponID *string

	// Detail porte les payloads typés par EventType (ex. medal ID + tier
	// pour event "medal", clutch context pour "clutch"). La structure exacte
	// dépend du type ; les helpers de consommation (analysis/narrative)
	// connaissent les clés attendues.
	Detail map[string]any
}

// PlayerMatchRow est le contrat partagé par les services qui agrègent les
// matchs d'un joueur (Squad, MatchView, Career, Synthesis, Citations, Timeseries).
// Compose les 3 facets canoniques :
//
//   - Summary : metadata du match (map, mode, playlist, durée, outcome global).
//   - Self : stats du joueur principal sur ce match.
//   - Enrichment : metadata LevelUp-specific (session, performance, dominance,
//     friends, MMR enemy).
//
// Politique d'évolution : additive uniquement. La suppression ou le renommage
// d'un champ est un breaking change qui doit être documenté en commit message
// + ADR + dépréciation préalable. Voir docs/adr/0005-canonical-player-match-row-evolution.md
// (livré en Phase 4 méta-plan).
type PlayerMatchRow struct {
	Summary    MatchSummary
	Self       MatchParticipant
	Enrichment PlayerMatchEnrichment
}

// PlayerMatchEnrichment porte les metadata LevelUp-specific qui ne sont pas
// dans le core canonical (sessions, performance score interne, dominance flag
// calculé au sync, contexte friends, MMR enemy si head-to-head).
type PlayerMatchEnrichment struct {
	SessionID           *string
	SessionLabel        *string
	PerformanceScore    *float64
	DominanceFlag       DominanceFlag
	HadBotTeammate      bool
	IsWithFriends       bool
	FriendsXUIDs        []string // sous-ensemble présent ce match (peut être nil)
	TeamMMR             *float64
	EnemyMMR            *float64       // si dispo (head-to-head, sinon nil)
	SkillSnapshot       *SkillSnapshot // ADR 0011, P4 — rating + tier + sub-tier + delta
	EngagementScoreBrut *float64       // résidu brut engagement (player - attendu), nil si non calculé
	EngagementPaceRatio *float64       // engagement absolu = pace_joueur/pace_lobby (1.0 = rythme lobby), nil si indispo
}

// SkillSnapshot est la projection canonique du rating de skill pour un match.
//
// ADR 0011 (P4) : tout titre PvP compétitif expose un rating + tier code +
// sub-tier + delta points. Les LIBELLÉS localisés (`SkillTierLabel = "Diamant 3"`)
// restent dans `TitleSemanticAdapter.Ranks()`. L'image (`SkillRankImageURL`)
// reste dans `TitleAssetURLAdapter.CSRRankImageURL`.
//
// Tous les champs sont optionnels (pointeurs) — un titre sans système de
// tiers (ex: Halo MCC 1-50) peut ne renseigner que `RatingValue`.
type SkillSnapshot struct {
	RatingType           RatingType // "csr" | "lusr" (enum existant canonical/enums.go)
	RatingValue          *float64   // valeur brute du rating (CSR points, LUSR mu)
	TierCode             *string    // code stable cross-titre (ex: "diamond", "onyx") — EN
	TierCodeFR           *string    // libellé localisé FR (ex: "Or", "Platine") depuis match_skill_rank.tier_fr
	SubTier              *int       // 1..6 ou nil pour Onyx (max tier sans sub-tier)
	Delta                *float64   // points gagnés/perdus ce match (positif/négatif)
	PlaylistGroup        *string    // groupe normalisé (ex: "ranked-arena")
	SeasonID             *string    // saison Halo associée au rating (ex: "Elan")
	MeasurementRemaining *int       // matchs de placement restants (>0 = phase placement)
	KillsExpected        *float64   // depuis MatchSkillSnapshot — utilisé par Stats
	DeathsExpected       *float64   // idem
	// ExpectedWinProb : proba de victoire pré-match de l'équipe du joueur (LUSR v2
	// Sprint 1.A), ∈ [0,1]. nil pour les matchs pré-v2 / non-LUSR. Lue depuis
	// match_skill_rank.expected_win_prob (Stratégie C la pose sur les rows LUSR).
	ExpectedWinProb *float64
}

// ImpactBadge est un badge d'impact calculé sur les événements d'un match.
type ImpactBadge struct {
	BadgeKey   string // identifiant technique : first_blood, clutch_finisher…
	BadgeFR    string // libellé français affiché
	PlayerXUID string
}

// TeamSnapshot est la vue légère d'une équipe d'un match.
type TeamSnapshot struct {
	TeamID            int
	Score             *int
	MMR               *float64
	ParticipantsXUIDs []string
}

// MatchSkillSnapshot contient les données natives de skill d'un match.
type MatchSkillSnapshot struct {
	PlayerXUID      string
	TeamMMR         *float64
	EnemyMMR        *float64
	KillsExpected   *float64
	KillsStdDev     *float64
	DeathsExpected  *float64
	DeathsStdDev    *float64
	AssistsExpected *float64
	AssistsStdDev   *float64
}

// CapabilityGap représente une limitation explicite reportée à un consommateur.
type CapabilityGap struct {
	CapabilityKey string // ex: "match.skill.snapshot"
	ReasonCode    string // cause normalisée
	Severity      string // "info" | "warning" | "blocking"
	Message       string
	Retryable     bool
}
