// Package domain — match_view_raw.go : types *Raw de transfert
// platform/duckdb -> service (lignes brutes des requêtes Q12-Q29). Aucune logique,
// uniquement des structs. Séparé de match_view.go qui ne conserve que les DTO de
// réponse JSON (refactor god-file, revue 2026-06-02).
package domain

import "time"

// MatchHistAvgRow : ligne brute de Q29 (historique récent pour moyennes).
type MatchHistAvgRow struct {
	Kills           int
	Deaths          int
	Assists         int
	HeadshotKills   *int
	MaxKillingSpree *int
	PerfectKills    int
	PairName        string
	IsFirefight     bool
	IsRanked        bool
	DurationSeconds int
}

// MatchMetaRaw : données brutes de la requête Q13 (match_registry).
type MatchMetaRaw struct {
	MatchID                 string
	StartTime               *time.Time
	DurationSeconds         *float64
	MapName                 *string
	PairName                *string
	PlaylistName            *string
	IsFirefight             bool
	IsRanked                bool
	PlayableDurationSeconds *int64
	MapAssetID              *string
	// MapVersionID : version de la map jouée (match_registry.map_version_id), pour
	// le fetch DiscoveryUGC de l'image (endpoint /api/v1/assets/maps/.../image?v=).
	MapVersionID    *string
	GameVariantName *string
	// PlaylistAssetID : identifiant stable de la playlist (match_registry.playlist_id).
	// Nil pour les matchs custom games sans playlist enregistrée.
	PlaylistAssetID *string
	// MapNameFR / ModeNameFR / PlaylistNameFR : traductions FR enrichies post-scan
	// via asset_translations (map, playlist) et mode_name_tr (mode). Nil si non
	// disponibles — le service retombe alors sur le label brut EN.
	MapNameFR      *string
	ModeNameFR     *string
	PlaylistNameFR *string
	// MapNameEN : nom canonique EN résolu via asset_translations en-US.
	// Utilisé pour les lookups d'asset image (l'adapter AssetURLAdapter
	// indexe `static/maps/halo_infinite/{name}.{ext}` par nom EN). Sans ça,
	// le fallback adapter recevrait l'UUID brut de match_registry.map_name
	// et échouerait silencieusement (warn "map image missing for known map").
	MapNameEN *string
	// PairNameFR : traduction FR stockée en DB (match_registry.pair_name_fr).
	// Utilisé comme fallback quand mode_name_tr ne contient pas le mode EN normalisé.
	PairNameFR *string
	// PairAssetID : identifiant stable du pair_name (match_registry.pair_id).
	// Permet la résolution via asset_translations quand pair_name est un UUID brut.
	PairAssetID *string
	// GameVariantAssetID : identifiant stable du game variant (match_registry.game_variant_id).
	GameVariantAssetID *string
	// MapImageURL : URL résolue depuis map_images_registry par map_id (stable UUID).
	// Nil si map_id absent du registry — le service retombe sur l'adapter name-based.
	MapImageURL *string
	// Team0Score / Team1Score : score de jeu de chaque équipe (match_registry).
	// Ex: 50/47 pour Slayer, 3/1 pour CTF. Nil si FFA ou custom sans score.
	Team0Score *int16
	Team1Score *int16
	// Team0RoundsWon / Team1RoundsWon / RoundsTotal : les MANCHES du match
	// (match_registry, source CoreStats.RoundsWon/Lost/Tied). Sur une variante déclarée
	// dans regulation.toml [rounds_decide], ce sont ELLES qui disent le résultat : le
	// score en points y est un cumul sur toutes les manches, qui peut donner la victoire
	// au perdant (mesure du 2026-08-29). Nil = inconnu → l'affichage garde les points.
	Team0RoundsWon *int16
	Team1RoundsWon *int16
	RoundsTotal    *int16
	// T0Ms : offset countdown pré-match en ms (Match Timeline T0, Phase 3).
	// Dérivé de `epoch_ms(real_start_time AT TIME ZONE 'UTC') − epoch_ms(start_time)`.
	// Nil si real_start_time absent → BuildForMatchMs retombe sur T0=0.
	T0Ms *int64
	// ElapsedSeconds : durée de JEU observée du match (médiane du temps joué des
	// participants présents du début à la fin ; repli MAX). Chargée séparément de
	// Q13 et best-effort (cf. duckdb/elapsed_seconds.go). Nil = non estimable →
	// aucun flag « Prolongation ».
	ElapsedSeconds *int
}

// PlayerMatchStatsRaw : données brutes de Q17 (match_participants filtré par xuid).
type PlayerMatchStatsRaw struct {
	OutcomeCode       int
	TeamID            *int
	RankInTeam        *int
	Kills             int
	Deaths            int
	Assists           int
	KDA               *float64
	Accuracy          *float64
	PersonalScore     *float64
	AvgLifeSeconds    *float64
	TimePlayedSeconds *float64
	ShotsFired        *int
	ShotsHit          *int
	DamageDealt       *float64
	DamageTaken       *float64
	TeamMMR           *float64
	EnemyMMR          *float64
	HeadshotKills     *int
	MaxKillingSpree   *int
	// BackfillBits (INTEGER) — bitmask match_participants.backfill_bits pour ce joueur×match.
	// Bit 9 (PBitMedals = 512) indique si les médailles ont été fetchées.
	// Nil si la colonne n'est pas disponible (anciens matchs ou erreur de scan).
	BackfillBits *int
}

// ScoreboardRaw : données brutes de Q12 (une ligne du scoreboard).
type ScoreboardRaw struct {
	XUID     string
	Gamertag string
	IsBot    bool
	// Participation (API PlayerParticipationInfo, colonnes match_participants) : nil sur
	// les matchs antérieurs aux colonnes. Les horodatages sont des instants ABSOLUS (UTC —
	// bug TZ des matchs anciens corrigé en base le 2026-05-29).
	JoinedInProgress *bool
	LeftInProgress   *bool
	FirstJoinedTime  *time.Time
	LastLeaveTime    *time.Time
	TeamID           *int
	RankInTeam       *int
	OutcomeCode      int
	PersonalScore    *float64
	Kills            int
	Deaths           int
	Assists          int
	KDA              *float64
	Accuracy         *float64
	TimePlayed       *float64
	TeamMMR          *float64
	EnemyMMR         *float64
	ShotsFired       *int
	ShotsHit         *int
	DamageDealt      *float64
	DamageTaken      *float64
	AvgLifeSeconds   *float64
	HeadshotKills    *int
	MaxKillingSpree  *int
	GrenadeKills     *int
	MeleeKills       *int
	PowerWeaponKills *int
	// Mécaniques de kill NATIVES Halo 5 (assassinats + compétences spartiate :
	// ground pound, shoulder bash). Colonnes match_participants (Q12) — nil hors h5
	// (Infinite ne les fournit pas). Alimentent la FragDistribution v2 du viewer
	// (classe Capacités spartanes + rôle Assassinat) et les colonnes scoreboard.
	AssassinationKills *int
	GroundPoundKills   *int
	ShoulderBashKills  *int
	PerfectKills       int
	TopWeaponID        *int64
	TopWeaponLabel     string
	// Expected stats depuis match_participants (API) + calcul à la volée (service).
	// AssistsExpected peuplé par le service layer (assists_model_coefs), pas par l'API.
	KillsExpected   *float64
	DeathsExpected  *float64
	KillsStdDev     *float64
	DeathsStdDev    *float64
	AssistsExpected *float64
	// LocallyEstimated : K/D attendus issus du modèle local (count∝durée, Halo 5),
	// pas de l'API skill. Posé sur la ligne is_me → label « Estimé localement » du drawer.
	LocallyEstimated bool
	// Objectifs (match_objective_stats_latest, requête SÉPARÉE Q12bObjectiveStats
	// jointe côté Go par xuid) — nil hors mode à objectif, titre non supporté (table
	// vide) OU vue absente (dégradation best-effort G1 : le scoreboard reste servi).
	// Blocs mutuellement exclusifs par mode : seul le bloc du mode joué est non-nil.
	// Le service construit MatchScoreboardRow.Objective uniquement si un bloc est présent.
	Obj ObjectiveRaw
}

// ObjectiveRaw : stats objectifs par joueur (Q12bObjectiveStats sur
// match_objective_stats_latest, chargée séparément du scoreboard).
// Blocs mutuellement exclusifs par mode (CTF / Zones / Oddball / Stockpile / Extraction /
// VIP). HasObjective() vrai dès qu'un discriminant de bloc est non-nil.
type ObjectiveRaw struct {
	// CTF (CaptureTheFlagStats)
	FlagCaptures             *int
	FlagCaptureAssists       *int
	FlagGrabs                *int
	FlagSecures              *int
	FlagSteals               *int
	FlagReturns              *int
	FlagCarriersKilled       *int
	FlagReturnersKilled      *int
	KillsAsFlagCarrier       *int
	KillsAsFlagReturner      *int
	TimeAsFlagCarrierSeconds *float64
	// Zones (ZonesStats — Strongholds + KOTH)
	ZoneCaptures       *int
	ZoneSecures        *int
	ZoneOffensiveKills *int
	ZoneDefensiveKills *int
	ZoneScoringTicks   *int
	TimeInZonesSeconds *float64
	// Oddball (OddballStats)
	KillsAsSkullCarrier              *int
	SkullCarriersKilled              *int
	SkullGrabs                       *int
	SkullScoringTicks                *int
	TimeAsSkullCarrierSeconds        *float64
	LongestTimeAsSkullCarrierSeconds *float64
	// Stockpile (StockpileStats — V721-02)
	KillsAsPowerSeedCarrier       *int
	PowerSeedCarriersKilled       *int
	PowerSeedsDeposited           *int
	PowerSeedsStolen              *int
	TimeAsPowerSeedCarrierSeconds *float64
	TimeAsPowerSeedDriverSeconds  *float64
	// Extraction (ExtractionStats — V721-02, aucune durée)
	ExtractionConversionsCompleted *int
	ExtractionConversionsDenied    *int
	ExtractionInitiationsCompleted *int
	ExtractionInitiationsDenied    *int
	SuccessfulExtractions          *int
	// VIP (VipStats — V721-02)
	KillsAsVip              *int
	VipKills                *int
	VipAssists              *int
	TimesSelectedAsVip      *int
	MaxKillingSpreeAsVip    *int
	TimeAsVipSeconds        *float64
	LongestTimeAsVipSeconds *float64
	// Assaut (match_bomb_stats_latest — RECONSTRUITES DU FILM, pas de l'API)
	//
	// SEUL BLOC DE CETTE STRUCTURE QUI NE VIENT PAS DE `match_objective_stats_latest`, et c'est
	// une décision de schéma : l'API 343 ne publie AUCUNE statistique d'objectif pour l'Assaut
	// (la famille `BombStats` du moteur est de la télémétrie Bond, jamais répliquée dans le
	// film — mesure du 2026-09-04, cause unique du silence des deux côtés). Celles-ci sont
	// décodées du film Theater et vivent dans une TABLE DÉDIÉE : deux producteurs sur la même
	// table s'écraseraient dans la vue `_latest`, et un re-sync API effacerait les stats de
	// bombe. Elles sont donc chargées par une SECONDE requête, dégradable indépendamment, et
	// gatée par la capability `film.bomb_stats`.
	//
	// NULL = NON MESURÉ, JAMAIS ZÉRO : les cinq colonnes sont nullables au DDL.
	// `BombCarriersKilled` est nul PARTOUT à ce jour — la paire tueur/victime qu'il demande
	// n'existe pas dans la chaîne de cuisson (report au registre).
	BombDetonations          *int
	BombArms                 *int
	BombGrabs                *int
	TimeAsBombCarrierSeconds *float64
	BombCarriersKilled       *int
}

// HasCTF / HasZones / HasOddball / HasStockpile / HasExtraction / HasVip : discriminants
// de bloc (un compteur non-nil suffit — une valeur 0 reste un pointeur non-nil).
func (o ObjectiveRaw) HasCTF() bool     { return o.FlagGrabs != nil || o.FlagCaptures != nil }
func (o ObjectiveRaw) HasZones() bool   { return o.ZoneCaptures != nil || o.ZoneScoringTicks != nil }
func (o ObjectiveRaw) HasOddball() bool { return o.SkullGrabs != nil || o.SkullScoringTicks != nil }
func (o ObjectiveRaw) HasVip() bool     { return o.TimesSelectedAsVip != nil || o.KillsAsVip != nil }
func (o ObjectiveRaw) HasStockpile() bool {
	return o.PowerSeedsDeposited != nil || o.PowerSeedCarriersKilled != nil
}
func (o ObjectiveRaw) HasExtraction() bool {
	return o.SuccessfulExtractions != nil || o.ExtractionInitiationsCompleted != nil
}

// HasBomb : le bloc ASSAUT, reconstruit du FILM et NON de l'API (cf. le bloc de champs
// ci-dessus). Le discriminant prend les deux compteurs que la chaîne publie toujours ensemble
// dès qu'une source est lue — `bomb_detonations` (statborg) et `bomb_arms` (anneau + jointure).
// `bomb_carriers_killed` n'y entre PAS : il est nul partout, il ne discriminerait rien.
func (o ObjectiveRaw) HasBomb() bool {
	return o.BombDetonations != nil || o.BombArms != nil || o.BombGrabs != nil
}

// HasObjective : au moins un bloc objectif présent.
func (o ObjectiveRaw) HasObjective() bool {
	return o.HasCTF() || o.HasZones() || o.HasOddball() ||
		o.HasStockpile() || o.HasExtraction() || o.HasVip() || o.HasBomb()
}

// BulkMedalRaw : une ligne de Q27 (médailles de tous les joueurs du match).
type BulkMedalRaw struct {
	XUID       string
	MedalID    int64
	Count      int
	Label      string
	Difficulty string
}

// BulkWeaponKillRaw : une ligne de Q28 (kills par arme de tous les joueurs du match).
type BulkWeaponKillRaw struct {
	XUID        string
	WeaponID    int64
	Kills       int
	WeaponLabel string
	// Class / Role / Family : dimensions du registre d'armes (axe manipulation + fonction
	// de combat + famille), résolues dans la même passe que le label (resolveWeaponMeta).
	// Vides si l'arme est absente du registre. Alimentent la FragDistribution par-match
	// (sunburst v2) : Class recolore le breakdown par arme + la ventilation gun → classe ;
	// Family alimente le TYPE de grenade au niveau 2 (V72-15.2 : frag/plasma/dynamo/splinter).
	Class  string
	Role   string
	Family string
	// WeaponKey : clé canonique du registre ("h5_vehicle_warthog"), niveau 2 des classes
	// ventilées par ENGIN (véhicule/tourelle — V73-3.2, où Class/Role/Family valent tous
	// « vehicle » et ne distinguent donc aucun engin).
	WeaponKey string
	// MechanicKills : sous-ensemble de Kills non-arme (mêlée/assassinat/coup au sol/charge
	// d'épaule attribués à l'arme TENUE, kill_kind <> 'weapon' sur H5 ; 0 sur Infinite).
	// buildFragDistribution les retire des classes gun (anti-double-comptage V72-15.3).
	MechanicKills int
}

// MatchEnrichmentRaw : données brutes de Q18.
type MatchEnrichmentRaw struct {
	PerformanceScore *float64
	IsWithFriends    bool
	IsExcluded       bool
	// DominanceFlag : 0=none, 1=domination, 2=humiliation, 3=remontada,
	// 4=débandade, 5=contre-remontada (cf. canonical.DominanceFlag).
	// Peuplé par sync.BackfillDominanceFlags via engine.RunBackfillComebackBadges.
	DominanceFlag int
}

// MedalRaw : données brutes de Q14.
type MedalRaw struct {
	MedalID     int64
	Count       int
	Label       string
	Description string
	Difficulty  string
}

// EventRaw : données brutes de Q21.
type EventRaw struct {
	EventType string
	TimeMS    *int64
	XUID      *string
	// Gamertag : résolu côté repo via v_gamertag_lookup (bots gérés, fallback
	// xuid raw). Nil uniquement si le xuid est orphelin (jamais en
	// match_participants ni xuid_aliases) ; dans ce cas le service affichera
	// le XUID brut.
	Gamertag *string
	// MedalName : nom ANGLAIS de la médaille (events `medal` uniquement), extrait
	// côté repo du raw_json de highlight_events (champ medal_name). Nil pour les
	// autres event_types, et pour un event medal dont le raw_json est absent ou
	// illisible — le service dégrade alors en event medal anonyme.
	MedalName *string
}

// MedalNameMeta : l'identité d'une médaille résolue depuis son nom ANGLAIS
// (medal_definitions.name_en, metadata.duckdb). Label/Description suivent la
// locale de requête — même chaîne BCP-47 que lookupMedalMeta (GH-5b).
type MedalNameMeta struct {
	MedalNameID int64
	Label       string
	Description string
}

// KillSourceRaw : la SOURCE DE DÉGÂT d'une mort (Q21b), telle que le décodeur de film
// l'a inscrite dans `match_kill_events`.
//
// Une ligne par couple (tueur, instant) et SEULEMENT quand la source est UNANIME : deux
// morts au même millisecond pour le même tueur (un double kill) donnent deux lignes dans
// la table ; si elles ne portent pas la même source, Q21b n'en publie aucune. La règle est
// la même que celle du pont vers les icônes — jamais une arme fausse. Mesuré sur la base
// de production : 0 cas contradictoire sur 152 009 kills, mais la garde reste.
type KillSourceRaw struct {
	// XUID du tueur, tel que le kill-feed le crédite (`feed_killer_xuid`).
	XUID string
	// TimeMS : instant de la mort, même échelle que highlight_events.time_ms.
	TimeMS int64
	// SourceTag : identifiant `jpt!` de l'effet de dégât fatal. Ne dépend d'aucune table
	// de nommage — c'est l'adapter du titre qui le traduit en icône.
	SourceTag uint32
	// Headshot : le modificateur de dégât fatal (`source_category`) valait EXACTEMENT
	// `killscope.CategoryHeadshot`, ET était UNANIME sur le couple (tueur, instant) — même
	// garde que SourceTag (Q21b : `count(DISTINCT source_category) = 1`, même clause `HAVING`
	// que le tag). PAS de pointeur : la seule PRÉSENCE de cette ligne dans le résultat de
	// Q21b affirme déjà « connu et non ambigu » (comme SourceTag) — l'absence de ligne EST
	// l'état « non mesurable/ambigu », jamais `false`.
	//
	// G.1 (2026-08-29) : oracle contre `match_participants.headshot_kills` (API officielle),
	// filtre STRICT `= 'Headshot'` = 99,3 % d'accord ; `HeadshotMultiplier` INCLUS fait chuter
	// à 84,4 % — jamais l'ajouter (verrouillé par killscope.IsHeadshotCategory +
	// internal/archlint/no_raw_headshot_category_literal_test.go).
	Headshot bool
}

// KillAssistRaw : l'ASSISTANCE d'une mort (Q21c), telle que le décodeur de film l'a
// inscrite dans `match_kill_events`.
//
// UNE LIGNE PRÉSENTE = ON SAIT (`assist_known` en base). AssistGamertag nil dit alors
// « PAS d'assistant », MESURÉ — à ne jamais confondre avec l'état « on ne sait pas », qui
// n'a PAS de ligne : c'est l'absence d'appariement qui le porte, et le feed doit préserver
// cette distinction jusqu'à l'écran.
type KillAssistRaw struct {
	// XUID du tueur, tel que le kill-feed le crédite (`feed_killer_xuid`).
	XUID string
	// TimeMS : instant de la mort, même échelle que highlight_events.time_ms.
	TimeMS int64
	// AssistGamertag : l'assistant nommé, nil quand la mort n'en porte pas (mesuré).
	AssistGamertag *string
	// AssistXUID : le xuid de l'assistant nommé, pour résoudre son équipe. Nil sans
	// assistant.
	AssistXUID *string
	// KillerDamagePct / AssistDamagePct : parts de dégâts en pourcentage ENTIER, non
	// bornées à 100 (valeurs mesurées jusqu'à 228 — dégât excédentaire, hypothèse non
	// établie). Nil = non mesurée, jamais 0. AssistDamagePct n'existe que si l'assistant
	// est nommé (le persister refuse une part sans porteur).
	KillerDamagePct *int
	AssistDamagePct *int
}

// KVPairRaw : données brutes de Q20 (single-match, MatchID non peuplé) et du
// loader batch SquadRepository.LoadKVPairs (Q32c, MatchID peuplé pour le
// regroupement multi-match de la synthèse d'events kill/death title-agnostic).
//
// UN XUID VIDE N'EST PAS UN XUID INCONNU : C'EST UNE ABSENCE D'IDENTITÉ. La canonique
// `match_kill_events` porte les morts de BOT avec un `xuid` NULL et un `gamertag`
// renseigné — un bot n'a pas de XUID. Q20 sert ce NULL TEL QUEL (jamais de COALESCE vers
// la chaîne vide au SQL : il fusionnerait tous les bots en un joueur fantôme, piège gardé
// par TestPasDeXuidNormaliseEnChaineVide) et le scanner le rend en "".
//
// Ce que "" impose aux consommateurs, et c'est une règle, pas une précaution :
//   - Tout AGRÉGAT par acteur (Dominance/tug, KD timeline, antagonistes, némésis,
//     synthèse d'events kill/death) SAUTE la paire dès qu'un des deux xuid est vide.
//     Les duels et les courbes sont HUMAINS SEULEMENT — c'est la sémantique d'avant
//     la bascule du 2026-08-03, où l'ancienne table ne savait pas représenter un bot.
//   - Seul le KILL FEED, qui NOMME au lieu d'agréger, exploite une ligne à xuid vide :
//     il affiche le gamertag du bot. Un journal cite ce qui s'est passé ; un agrégat
//     répond sur des joueurs.
type KVPairRaw struct {
	// MatchID : peuplé uniquement par les lectures batch (LoadKVPairs). Vide ("")
	// pour le chemin single-match Q20 qui scope déjà la requête par match_id.
	MatchID string
	// KillerXUID / VictimXUID : "" = acteur SANS identité (bot), cf. doctrine ci-dessus.
	KillerXUID string
	// KillerGT : peut être vide (colonne NULLABLE au DDL) — un tueur non nommé.
	KillerGT   string
	VictimXUID string
	// VictimGT : NOT NULL au DDL, donc toujours renseigné. C'est ce qui permet au feed
	// de nommer une victime bot que son xuid ne désigne pas.
	VictimGT  string
	KillCount int
	TimeMS    int64
}

// MatchNeighbors : matchs adjacents (prev/next) pour la navigation de la page détail.
type MatchNeighbors struct {
	PreviousMatchID *string `json:"previous_match_id"`
	NextMatchID     *string `json:"next_match_id"`
	CurrentIndex    int     `json:"current_index"`
	TotalMatches    int     `json:"total_matches"`
	// AppliedFilters : echo des filtres effectivement appliqués (Phase 2b).
	// Permet au front de reconstruire un contextLabel localisé sans dupliquer
	// la résolution côté backend. Nil si aucun filtre n'a été passé.
	AppliedFilters *MatchFilterSpec `json:"applied_filters,omitempty"`
}

// SkillRankRaw : données brutes de Q22 (match_skill_rank — player DB).
type SkillRankRaw struct {
	RatingType    string
	TierLabel     *string
	RatingValue   *float64
	RatingDelta   *float64
	PlaylistGroup *string
	// Tier (EN, ex: "Diamond", "Onyx") et SubTier (1..6 ou 0 pour Onyx) sont
	// extraits depuis match_skill_rank. Utilisés par buildRankBlock pour
	// construire l'URL du badge via TitleAssetURLAdapter.CSRRankImageURL.
	// Nil quand absent (LUSR ou rang non encore résolu côté sync).
	Tier    *string
	SubTier *int
	// ExpectedWinProb : proba de victoire pré-match de l'équipe du joueur
	// (LUSR v2, ∈ [0,1]). Lue depuis match_skill_rank_latest.expected_win_prob.
	// Nil pour les matchs pré-v2 / sans donnée (Stratégie C : posée sur les rows LUSR).
	ExpectedWinProb *float64
}

// EncounterRaw : données brutes de Q23 (participants du match + historique commun).
type EncounterRaw struct {
	XUID          string
	Gamertag      string
	IsBot         bool
	CountTogether int
	IsAlly        bool
}

// EncounterStatsRaw : stats riches par encounter chargees via Q23b
// (chunk MV4.C'). Permet a narrative.ComputeEncounterBadges d'attribuer
// les badges ally_plus + tough_enemy.
//
// L'agregation se fait sur l'historique commun entre le joueur courant
// (myXUID) et tm.xuid. Le service compose ces stats avec les EncounterRaw
// (CountTogether + IsAlly) pour produire les EncounterStats consommees par
// narrative.ComputeEncounterBadges.
type EncounterStatsRaw struct {
	XUID           string
	AllyCount      int
	EnemyCount     int
	WinsAsAlly     int
	LossesAsAlly   int
	WinsVsEnemy    int
	LossesVsEnemy  int
	KillsDealt     int
	DeathsSuffered int
	// FirstSeen : start_time UTC du match commun le plus ANCIEN. Zéro time
	// quand l'historique est vide ou sans start_time. Alimente les badges de
	// relation temporels (recrue / ancien) réutilisés depuis le hub Relations.
	FirstSeen time.Time
	// LastSeenAt : start_time UTC du match commun le plus récent. Zéro time
	// quand l'historique est vide ou que match_registry n'a pas de start_time
	// pour les matchs concernés.
	LastSeenAt time.Time
}

// MediaAssocRaw : données brutes de Q24 (media_files + media_match_associations).
type MediaAssocRaw struct {
	FileID        string
	FileName      string
	FilePath      string
	Kind          string
	ThumbnailPath *string
	// CaptureStartTime : début de capture (RFC3339), nil si la base ne le connaît
	// pas. CaptureTime, lui, est la FIN de la capture — les deux sont distincts
	// pour un clip et confondus pour une image.
	CaptureStartTime *string
	CaptureTime      *string
	// DurationSeconds : durée du média arrondie à la seconde (la base la stocke
	// en DOUBLE). Nil pour une image ou quand ffprobe n'a rien pu dériver.
	DurationSeconds *int
	Liked           bool
}

// PlayerAssistsModel contient les coefs OLS per-mode d'un joueur pour expected_assists.
// Chargé depuis player_assists_model (stats.duckdb) par MatchViewRepository.GetPlayerAssistsModel.
// Nil si le mode n'est pas encore modélisé (< 15 matchs).
type PlayerAssistsModel struct {
	GameVariantName string
	Intercept       float64
	CoefKills       float64
	CoefDeaths      float64
	CoefDamageDealt float64
	CoefDamageTaken float64
	CoefMMRDelta    float64
	R2              float64
	N               int
}

// ExpectedStatsRaw : données brutes de Q26 (match_participants expected columns).
// AssistsExpected est peuplé par le service layer (formule assists_model_coefs), pas par Q26.
type ExpectedStatsRaw struct {
	KillsExpected   *float64
	DeathsExpected  *float64
	KillsStddev     *float64
	DeathsStddev    *float64
	AssistsExpected *float64
}
