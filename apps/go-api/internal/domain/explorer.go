// Package domain — types pour l'explorer (matchs communs, recherche joueurs).
//
// Port Go de apps/api/app/schemas/explorer.py.
// Routes : POST /players/{slug}/pages/explorer/player-query
//
//	POST /players/{slug}/pages/explorer/matches-query
//	GET  /directory/gamertags/search (déjà implémenté dans gamertag_repo)
package domain

import "time"

// ---------------------------------------------------------------------------
// Explorer — Player Query
// ---------------------------------------------------------------------------

// PageSizeCommonMatches est la taille de page fixe pour l'historique commun.
const PageSizeCommonMatches = 20

// ExplorerPlayerQueryRequest : corps de la requête POST player-query.
type ExplorerPlayerQueryRequest struct {
	TargetGamertag string `json:"target_gamertag"`
	// TargetXUID (optionnel) : quand fourni (ex. ligne du Classement), le service
	// l'utilise directement et saute la résolution gamertag→xuid locale — qui
	// échouerait pour un joueur absent des données locales.
	TargetXUID string `json:"target_xuid,omitempty"`
	Page       int    `json:"page,omitempty"` // 1-indexé, défaut 1
}

// CommonMatchRow : un match en commun entre 2 joueurs.
type CommonMatchRow struct {
	MatchID       string    `json:"match_id"`
	StartTime     time.Time `json:"start_time"`
	MapUI         string    `json:"map_ui"`
	ModeUI        string    `json:"mode_ui"`
	WereTeammates bool      `json:"were_teammates"`
	PlayerOutcome int       `json:"player_outcome"`
	OutcomeLabel  string    `json:"outcome_label"`
	Kills         int       `json:"kills"`
	Deaths        int       `json:"deaths"`
	KDA           float64   `json:"kda"`
}

// ExplorerEncounterStats : stats agrégées du couple (player_courant, target),
// mirror partiel de MatchEncounterRow (mêmes 7 colonnes que le tableau
// MatchEncountersTable de la page match-view, sans `is_ally` car pas de match
// courant en Explorer).
type ExplorerEncounterStats struct {
	// CountTogether = nombre de matchs joués ensemble (alliés + ennemis).
	CountTogether int `json:"count_together"`
	// AllyCount = matchs en tant qu'alliés. Nil si calcul indisponible.
	AllyCount *int `json:"ally_count,omitempty"`
	// EnemyCount = matchs en tant qu'ennemis. Nil si calcul indisponible.
	EnemyCount *int `json:"enemy_count,omitempty"`
	// WinrateAsAlly = ratio victoires en tant qu'allié (0..1). Nil si AllyCount=0.
	WinrateAsAlly *float64 `json:"winrate_as_ally,omitempty"`
	// WinrateVsEnemy = ratio victoires en tant qu'ennemi (0..1). Nil si EnemyCount=0.
	WinrateVsEnemy *float64 `json:"winrate_vs_enemy,omitempty"`
	// KillsDealt = kills par moi sur la cible (toutes occurrences).
	KillsDealt *int `json:"kills_dealt,omitempty"`
	// DeathsSuffered = morts subies par moi causées par la cible.
	DeathsSuffered *int `json:"deaths_suffered,omitempty"`
	// LastSeenAt = date du dernier match commun (toutes occurrences).
	LastSeenAt *time.Time `json:"last_seen_at,omitempty"`
}

// ExplorerPlayerQueryResponse : réponse de la requête player-query.
type ExplorerPlayerQueryResponse struct {
	TargetGamertag string                  `json:"target_gamertag"`
	TargetXUID     string                  `json:"target_xuid"`
	CommonMatches  []CommonMatchRow        `json:"common_matches"`
	Badges         []MatchEncounterBadge   `json:"badges,omitempty"`
	EncounterStats *ExplorerEncounterStats `json:"encounter_stats,omitempty"`
	Total          int                     `json:"total"`       // items sur la page courante
	TotalCount     int                     `json:"total_count"` // total tous matchs confondus
	WinsTogether   int                     `json:"wins_together"`
	LossesTogether int                     `json:"losses_together"`
	Page           int                     `json:"page"`
	PageSize       int                     `json:"page_size"`
	// ActivityHeatmap : agrégat jour × heure des matchs communs (Explorer mode
	// Joueur). Calculé sur l'ensemble des matchs communs (avant pagination).
	// Coloration front pilotée par `count` (intensité d'activité).
	ActivityHeatmap []TemporalHeatmapCell `json:"activity_heatmap,omitempty"`
	// TargetProfile : encart "Profil joueur cible" affiché en haut des résultats
	// Explorer. Composite de 4 sous-blocs best-effort (chacun peut être nil).
	// Voir ExplorerTargetProfile pour la sémantique des champs.
	TargetProfile *ExplorerTargetProfile `json:"target_profile,omitempty"`
}

// ExplorerTargetProfile : encart synthétique sur le joueur cible recherché
// dans l'Explorer mode Joueur. Composite de 4 sources :
//
//  1. Identity : identité Spartan (banner / emblem / service tag / rang
//     carrière) — fetch live Halo via CareerLiveService.GetSpartanIdentityFor.
//     Fallback DB locale si live indisponible. Nil si totalement inconnu.
//  2. CareerStats : stats agrégées carrière entière du joueur (KDA / KDR /
//     win rate / accuracy / etc.) — fetch live Halo via PlayerStatsProvider.
//     FetchRemoteStats. Nil si endpoint inaccessible ou no-tokens.
//  3. SampleStats : stats calculées localement sur les matchs en commun avec
//     le user connecté (échantillon). Toujours calculable depuis DuckDB.
//     Nil seulement si common_matches=0.
//  4. PrivacyWarning : niveau de privacy du profil cible (none / partial /
//     full) — fetch via PrivacyProvider.GetMatchPrivacy. Nil si no-tokens.
//
// AuthAvailable : vrai si le user connecté a des tokens OAuth Halo. Quand
// faux, les 3 premiers sous-blocs basculent en mode dégradé (identity peut
// venir de la DB locale, career_stats et privacy sont nil). Sert au front à
// afficher un hint "Connexion Halo requise" sur les sections masquées.
type ExplorerTargetProfile struct {
	// Identity est sérialisé en DTO snake_case (career_rank imbriqué) comme la
	// Home — PAS la struct brute HomeSpartanIdentityRow (clés PascalCase, champs
	// de rang plats) qui divergeait du contrat front. Construit via
	// analysis.BuildSpartanIdentity dans le service.
	Identity    *HomeSpartanIdentity       `json:"identity,omitempty"`
	CareerStats *NormalizedPlayerStats     `json:"career_stats,omitempty"`
	SampleStats *ExplorerTargetSampleStats `json:"sample_stats,omitempty"`
	// TopMedals : médailles lifetime du joueur cible (top, triées par count
	// décroissant, cap 20) issues du service record Waypoint + métadonnées
	// locales (label/description/image). Le front affiche un top 5 + expander.
	TopMedals []MedalDigestItem `json:"top_medals,omitempty"`
	// SeasonCSRs : classements CSR par playlist ranked de la saison courante du
	// joueur cible (live, endpoint skill public — fonctionne pour tout xuid).
	SeasonCSRs []CareerPlaylistCSR `json:"season_csrs,omitempty"`
	// MatchesPerSeason : nombre de matchs matchmade par saison (service record
	// par saison). Vide si non calculé/indisponible.
	MatchesPerSeason []SeasonMatchCount `json:"matches_per_season,omitempty"`
	// CombatProfile : les ~20 derniers matchs PvP du joueur cible, fetchés en LIVE
	// via l'API Halo (RecentMatchesProvider, cache 20 min, aucune ingestion) — c'est
	// la source affichée PAR DÉFAUT pour les graphes "profil de combat". Vide si pas
	// d'auth / pas de données live. Cf. PLAN_explorer_combat_profile_charts.md.
	CombatProfile []ExplorerTargetRecentMatch `json:"combat_profile,omitempty"`
	// CombatProfileLocal : les N derniers matchs PvP de la cible présents en base
	// locale (shared.match_participants — essentiellement les matchs en commun).
	// Alimente le TOGGLE "local" de la section profil de combat. Vide si aucun.
	CombatProfileLocal []ExplorerTargetRecentMatch `json:"combat_profile_local,omitempty"`
	// PrivacyWarning : conservé (toujours nil — la privacy n'est plus fetchée
	// pour l'Explorer ; champ gardé pour compat de schéma).
	PrivacyWarning *MatchPrivacyWarning `json:"privacy_warning,omitempty"`
	AuthAvailable  bool                 `json:"auth_available"`
}

// ExplorerTargetRecentMatch est un match PvP récent du joueur cible, projeté
// pour les graphes "profil de combat" de l'Explorer (FDA, dégâts, score +
// placement, folie/frags parfaits, donut modes). Une ligne = un match.
//
// Source : shared.match_participants (+ match_registry pour map/mode/date) et
// shared.medals_earned (perfect kills). La résolution outcome→couleur/label se
// fait au front (cohérent avec TimeseriesKdaBars qui mappe l'outcome int).
type ExplorerTargetRecentMatch struct {
	MatchID   string    `json:"match_id"`
	StartTime time.Time `json:"start_time"`
	MapUI     string    `json:"map_ui"`
	ModeUI    string    `json:"mode_ui"`
	// Outcome : 1=tie, 2=win, 3=loss, 4=DNF (convention produit).
	Outcome int `json:"outcome"`
	// Rank : placement du joueur (1-based). nil si DNF/non classé — le front
	// laisse alors un trou dans la courbe (ne pas tracer 0, qui fausserait
	// l'axe inversé du graphe score+placement).
	Rank            *int    `json:"rank,omitempty"`
	Kills           int     `json:"kills"`
	Deaths          int     `json:"deaths"`
	Assists         int     `json:"assists"`
	KDA             float64 `json:"kda"`   // ratio FDA pré-calculé (match_participants.kda)
	Score           int     `json:"score"` // match_participants.personal_score
	DamageDealt     int     `json:"damage_dealt"`
	DamageTaken     int     `json:"damage_taken"`
	MaxKillingSpree int     `json:"max_killing_spree"`
	PerfectKills    int     `json:"perfect_kills"`
	// ModePairAssetID : AssetId du PlaylistMapModePair (source LIVE). Transient
	// (non sérialisé) — le MatchInfo live n'a pas de PublicName, donc on résout le
	// mode depuis cet AssetId via shared.match_registry (pair_id → pair_name) côté
	// repo (translateModeUIsFR). Vide pour la source locale (mode déjà résolu).
	ModePairAssetID string `json:"-"`
	// MapAssetID / GameVariantAssetID : AssetIds bruts du registre (source LOCALE),
	// transient (non sérialisés). Servent au fallback de résolution des libellés via
	// asset_translations quand les colonnes noms du registre sont NULL — cas Halo 5,
	// où map_name/pair_name sont vides sur 100 % des matchs (cf. resolveTargetRecentAssetNames).
	// Vides pour la source live (résolution via ModePairAssetID).
	MapAssetID         string `json:"-"`
	GameVariantAssetID string `json:"-"`
}

// SeasonMatchCount : nombre de matchs matchmade joués sur une saison donnée par
// le joueur cible (alimente le graphe "matchs par saison"). En mode live, Matches
// est le total de la saison (service record filtré par seasonId) et le pic de
// rang CSR de la saison est exposé (tier + image du badge, rendu au-dessus de la
// barre). En mode dégradé (bucketing local sans auth), seul Matches est rempli.
//
// Le split classé/non-classé n'est pas exposé : l'API Waypoint rejette
// `seasonId + isRanked` seul (cf. fetchSeasonRow).
type SeasonMatchCount struct {
	SeasonID   string `json:"season_id"`
	SeasonName string `json:"season_name"`
	Matches    int    `json:"matches"`
	// CSRTier / CSRSubTier / CSRBadgeImageURL : pic de rang CSR du joueur sur la
	// saison (plus haut tier parmi les playlists ranked engagées). Vide si aucune
	// donnée CSR pour la saison.
	CSRTier          string  `json:"csr_tier,omitempty"`
	CSRSubTier       int     `json:"csr_sub_tier,omitempty"`
	CSRBadgeImageURL *string `json:"csr_badge_image_url,omitempty"`
}

// ExplorerTargetSampleStats : stats agrégées du joueur cible calculées sur
// les matchs joués en commun avec le user connecté. Toutes les valeurs en
// pointer signifient "indisponible" lorsque le dénominateur est nul (deaths=0,
// shots_fired=0, etc.) — le frontend rend "—" dans ce cas.
type ExplorerTargetSampleStats struct {
	// SampleSize : nombre de matchs sur lesquels l'agrégation porte.
	SampleSize int `json:"sample_size"`

	// Totaux bruts.
	Kills       int `json:"kills"`
	Deaths      int `json:"deaths"`
	Assists     int `json:"assists"`
	Wins        int `json:"wins"`
	Losses      int `json:"losses"`
	Draws       int `json:"draws"`
	ShotsFired  int `json:"shots_fired"`
	ShotsHit    int `json:"shots_hit"`
	DamageDealt int `json:"damage_dealt"`
	DamageTaken int `json:"damage_taken"`

	// Breakdown kill types (somme sur le sample).
	HeadshotKills    int `json:"headshot_kills"`
	MeleeKills       int `json:"melee_kills"`
	PowerWeaponKills int `json:"power_weapon_kills"`
	GrenadeKills     int `json:"grenade_kills"`

	// Médailles : total et nombre de types distincts gagnés sur le sample.
	TotalMedals  int `json:"total_medals"`
	UniqueMedals int `json:"unique_medals"`

	// Ratios calculés. Nil si dénominateur nul.
	KDA                 *float64 `json:"kda,omitempty"`
	KDR                 *float64 `json:"kdr,omitempty"`
	WinRate             *float64 `json:"win_rate,omitempty"`             // 0..1
	Accuracy            *float64 `json:"accuracy,omitempty"`             // 0..1
	HeadshotRate        *float64 `json:"headshot_rate,omitempty"`        // headshots/kills
	OffensiveConversion *float64 `json:"offensive_conversion,omitempty"` // cf. combat_yield.go
	DefensiveResistance *float64 `json:"defensive_resistance,omitempty"` // cf. combat_yield.go

	// Cadence par minute — KPI dérivés (kills/deaths/assists ÷ minutes jouées),
	// même définition que la page Coéquipiers/Squad (teammates.14). Nil si la
	// durée jouée cumulée est nulle.
	KillsPerMin   *float64 `json:"kills_per_min,omitempty"`
	DeathsPerMin  *float64 `json:"deaths_per_min,omitempty"`
	AssistsPerMin *float64 `json:"assists_per_min,omitempty"`

	// AvgPersonalScore : score Halo moyen par match (AVG(personal_score), la
	// colonne "Score" du scoreboard). Nil si sample vide.
	AvgPersonalScore *float64 `json:"avg_personal_score,omitempty"`

	// PerfectKills : total de "frags parfaits" sur le sample (set de médailles
	// « frag parfait » du titre, cf. analysis.PerfectKillMedalIDs, agrégé sur
	// shared.medals_earned).
	PerfectKills int `json:"perfect_kills"`

	// TopWeapons : top armes (par kills) de la cible sur les matchs communs
	// (shared.v_weapon_kills + labels metadata.weapon_labels). Vide si indispo.
	TopWeapons []WeaponHighlight `json:"top_weapons,omitempty"`
}

// ParticipantStatsAggregate : agrégat brut renvoyé par le repo DuckDB pour le
// joueur cible sur la liste des common_matches. Sert d'entrée à
// analysis.BuildSampleStats — pas exposé au front.
type ParticipantStatsAggregate struct {
	Kills            int
	Deaths           int
	Assists          int
	Wins             int
	Losses           int
	Draws            int
	ShotsFired       int
	ShotsHit         int
	DamageDealt      float64
	DamageTaken      float64
	HeadshotKills    int
	MeleeKills       int
	PowerWeaponKills int
	GrenadeKills     int
	// TimePlayedSeconds : somme du temps joué (s) sur le sample — alimente la
	// cadence par minute. Colonne match_participants.time_played_seconds.
	TimePlayedSeconds int
	// PersonalScore : somme du score Halo (personal_score) sur le sample.
	PersonalScore int
}

// MedalCountsAggregate : agrégat brut médailles pour un joueur sur un sample
// de matchs (total occurrences et nombre de types distincts).
type MedalCountsAggregate struct {
	Total  int
	Unique int
	// PerfectKills : occurrences des médailles « frag parfait » du titre
	// (cf. analysis.PerfectKillMedalIDs) — exposé en "frags parfaits".
	PerfectKills int
}

// KillerVictimAggregate : kills croisés agrégés entre deux joueurs.
type KillerVictimAggregate struct {
	KillsDealt     int // kills du joueur courant sur le joueur cible
	DeathsSuffered int // kills du joueur cible sur le joueur courant
}

// ---------------------------------------------------------------------------
// Explorer — Matches Query
// ---------------------------------------------------------------------------

// ExplorerMatchesQueryRequest : corps de POST matches-query.
// Accepte les mêmes filtres que MatchHistoryQueryRequest.
type ExplorerMatchesQueryRequest struct {
	Filters           FilterContextInput `json:"filters"`
	Pagination        PaginationRequest  `json:"pagination"`
	SortField         string             `json:"sort_field"`
	SortDir           string             `json:"sort_dir"`
	IncludeExportHint bool               `json:"include_export_hint,omitempty"`
	// PerfTiers filtre par palier de performance (1=Excellent … 5=Mauvais).
	// Liste vide = pas de filtre.
	PerfTiers []int `json:"perf_tiers,omitempty"`
	// SkillTiers filtre par tier skill ("Bronze"…"Onyx"). Requiert RankedContext.
	SkillTiers []string `json:"skill_tiers,omitempty"`
	// RankedContext : "ranked" | "unranked" | "" (tous). Requiert pour activer SkillTiers.
	RankedContext string `json:"ranked_context,omitempty"`
	// OutcomeFilter : codes résultat acceptés (1=Égalité,2=Victoire,3=Défaite,4=Abandon).
	OutcomeFilter []int `json:"outcome_filter,omitempty"`
	// Filtres Explorer additionnels (date, type d'expérience, playlist, carte, mode, squad, match ID).
	MatchStartDate  *time.Time `json:"match_start_date,omitempty"`
	MatchEndDate    *time.Time `json:"match_end_date,omitempty"`
	ExperienceTypes []string   `json:"experience_types,omitempty"`
	Playlists       []string   `json:"playlists,omitempty"`
	MapNames        []string   `json:"map_names,omitempty"`
	ModeNames       []string   `json:"mode_names,omitempty"`
	SquadScope      string     `json:"squad_scope,omitempty"`
	MatchIDSearch   string     `json:"match_id_search,omitempty"`
	// MatchIDs : whitelist exacte de match_id à conserver (Explorer mode Joueur).
	MatchIDs []string `json:"match_ids,omitempty"`
	// IncludeBriefing : opt-in du bandeau de briefing (mode Matchs). Propagé au
	// service via MatchHistoryQueryRequest.IncludeExplorerBriefing. Faux (défaut)
	// = réponse sans le bloc Briefing (aucun surcoût). L'Explorer l'envoie à true.
	IncludeBriefing bool `json:"include_briefing,omitempty"`
}

// ExplorerMatchesRow : une ligne dans la liste des matchs filtrés (Explorer).
type ExplorerMatchesRow struct {
	MatchID             string    `json:"match_id"`
	StartTime           time.Time `json:"start_time"`
	StartTimeLabel      string    `json:"start_time_label"`
	MapUI               *string   `json:"map_ui"`
	ModeUI              *string   `json:"mode_ui"`
	PlaylistLabel       *string   `json:"playlist_label"`
	OutcomeCode         int       `json:"outcome_code"`
	OutcomeLabel        string    `json:"outcome_label"`
	ScoreLabel          string    `json:"score_label"`
	IsWithFriends       bool      `json:"is_with_friends"`
	ExperienceTypeLabel string    `json:"experience_type_label"`
	MatchURL            string    `json:"match_url"`
	// Combat stats
	Kills   int `json:"kills,omitempty"`
	Deaths  int `json:"deaths,omitempty"`
	Assists int `json:"assists,omitempty"`
	// PerfScore : score de performance 0-100 (nil si non calculé).
	PerfScore *int `json:"perf_score,omitempty"`
	// PerfTier : palier de performance 1-5 (0 si score absent).
	PerfTier int `json:"perf_tier,omitempty"`
	// DeltaPerf : déviation du score de perf depuis la médiane (perf_score - 50).
	DeltaPerf *int `json:"delta_perf,omitempty"`
	// SkillTierLabel : label formaté du tier ranked/LUSR (ex. "Diamant IV"), nil si absent.
	SkillTierLabel *string `json:"skill_tier_label,omitempty"`
	// RatingType : "CSR" (classé officiel Microsoft) ou "LUSR" (interne LevelUp).
	// Nil pour les matchs sans skill rank (PvE, Custom). Source : match_skill_rank.rating_type.
	RatingType *string `json:"rating_type,omitempty"`
	// ExpectedWinProb : proba de victoire pré-match de l'équipe ∈ [0,1] (LUSR v2).
	// Alimente la colonne « Prob. vic. ». Nil si pré-v2 / non disponible.
	ExpectedWinProb *float64 `json:"expected_win_prob,omitempty"`
	// PlacementDone/PlacementTotal : progression placement (X/Y).
	// Si présents, l'UI affiche "X/Y" dans la colonne Rang à la place du SkillTierLabel.
	// CSR : remaining parsé depuis "Placement (N restants)" + threshold csr_placement_thresholds.
	// LUSR : 10 plus anciens matchs sans LUSR par chaîne (arena_slayer/arena_objectif/btb/chaos).
	PlacementDone  *int `json:"placement_done,omitempty"`
	PlacementTotal *int `json:"placement_total,omitempty"`
	// DeltaMMR : variation de MMR/CSR/LUSR pour ce match (nil si non rankée).
	DeltaMMR *float64 `json:"delta_mmr,omitempty"`
	// TeamMMR : MMR moyen de l'équipe du joueur sur ce match (nil si non rankée).
	TeamMMR *float64 `json:"team_mmr,omitempty"`
	// EnemyMMR : MMR moyen de l'équipe adverse sur ce match (nil si non rankée).
	EnemyMMR *float64 `json:"enemy_mmr,omitempty"`
	// KDA per-match : valeur API native (Infinite) ou FDA net (k+a/3)-d
	// (Halo 5), nette (peut être négative), lue telle quelle ; jamais un
	// quotient/deaths.
	KDA *float64 `json:"kda,omitempty"`
	// DurationSeconds : durée du match en secondes (nil si manquante).
	DurationSeconds *int `json:"duration_seconds,omitempty"`
	// DominanceFlag : 0=none, 1=domination, 2=humiliation, 3=remontada,
	// 4=débandade, 5=contre-remontada (cf. canonical.DominanceFlag). Le
	// front résout le label via narrative.dominance.* (manifest match_view)
	// et affiche "-" pour 0.
	DominanceFlag int `json:"dominance_flag,omitempty"`
	// HadBotTeammate : un coéquipier (même team_id) avait un xuid bid(...).
	// Propagé pour la section Career best_matches (les LOSS avec bot ne sont
	// pas exposés ici : exclus côté repo, cf. CareerRepo.GetHighlightMatchIDs).
	// Le front affiche une pill "bot" sur la card du match.
	HadBotTeammate bool `json:"had_bot_teammate,omitempty"`
}

// ExplorerMatchesSummary : résumé de la requête Explorer.
type ExplorerMatchesSummary struct {
	TotalMatches int `json:"total_matches"`
	// Options disponibles pour les filtres Explorer (valeurs distinctes triées).
	// AvailableExperienceTypes : LABEL localisé (locale de requête), VALUE FR
	// canonique = clé de filtre intacte (GH6-1, miroir GH5-2 Omnibar).
	AvailableExperienceTypes []LabelValue `json:"available_experience_types,omitempty"`
	AvailablePlaylists       []string     `json:"available_playlists,omitempty"`
	AvailableMaps            []string     `json:"available_maps,omitempty"`
	AvailableModes           []string     `json:"available_modes,omitempty"`
	// Options Explorer-spécifiques avec count cascade-aware (sémantique OR).
	// Permettent au front d'afficher "Win (42)" et de griser les options à 0.
	AvailableOutcomes       []LabelValue `json:"available_outcomes,omitempty"`
	AvailablePerfTiers      []LabelValue `json:"available_perf_tiers,omitempty"`
	AvailableSkillTiers     []LabelValue `json:"available_skill_tiers,omitempty"`
	AvailableRankedContexts []LabelValue `json:"available_ranked_contexts,omitempty"`
	AvailableSquadScopes    []LabelValue `json:"available_squad_scopes,omitempty"`
}

// ExplorerMatchesTable : table paginée de l'Explorer.
type ExplorerMatchesTable struct {
	Items      []ExplorerMatchesRow `json:"items"`
	Pagination PaginationMeta       `json:"pagination"`
}

// ExplorerMatchesQueryResponse : réponse de POST matches-query.
type ExplorerMatchesQueryResponse struct {
	Summary    ExplorerMatchesSummary `json:"summary"`
	Table      ExplorerMatchesTable   `json:"table"`
	ExportHint *ExportHint            `json:"export_hint,omitempty"`
	// Briefing : bandeau de briefing au-dessus du tableau. Nil quand la requête
	// ne l'a pas demandé (include_briefing=false) ou scope vide.
	Briefing *ExplorerBriefing `json:"briefing,omitempty"`
}

// ---------------------------------------------------------------------------
// Common matches raw DB row
// ---------------------------------------------------------------------------

// CommonMatchRaw : données brutes de Q19.
type CommonMatchRaw struct {
	MatchID        string
	StartTime      time.Time
	MapUI          string
	ModeUI         string
	Player1TeamID  *int
	Player2TeamID  *int
	Player1Outcome int
	Player1Kills   int
	Player1Deaths  int
	Player1KDA     float64
}
