// Package domain — compare.go : types pour l'endpoint Compare joueur vs joueur.
//
// Sprint 54 C : NormalizedPlayerStats, CompareRequest/Response, CompareMetricRow.
package domain

import "fmt"

// NormalizedPlayerStats est une projection normalisée des stats d'un joueur,
// utilisée pour la comparaison multi-titre.
type NormalizedPlayerStats struct {
	TitleSlug string `json:"title_slug"`
	XUID      string `json:"xuid"`
	Gamertag  string `json:"gamertag"`
	// IsLocal : le joueur a sa propre stats.duckdb (toutes les métriques sont fiables).
	IsLocal bool `json:"is_local"`
	// IsLocalSample : le joueur est *remote* (pas de stats.duckdb), mais les
	// métriques locale-only (spree, life, perfect, headshots) ont été calculées
	// sur l'échantillon des matchs croisés avec le joueur A.
	IsLocalSample  bool    `json:"is_local_sample,omitempty"`
	Matches        int     `json:"matches"`
	WinRate        float64 `json:"win_rate"`
	KDA            float64 `json:"kda"`
	KDR            float64 `json:"kdr"`
	KillsPerGame   float64 `json:"kills_per_game"`
	DeathsPerGame  float64 `json:"deaths_per_game"`
	AssistsPerGame float64 `json:"assists_per_game"`
	Accuracy       float64 `json:"accuracy"`
	DamagePerGame  float64 `json:"damage_per_game"`

	// Phase 2 — métriques enrichies.
	DamageTakenPerGame   float64 `json:"damage_taken_per_game"`
	PerfectKillsPerGame  float64 `json:"perfect_kills_per_game"`
	MaxKillingSpree      int     `json:"max_killing_spree"`
	AvgLifeSecs          float64 `json:"avg_life_secs"`
	HeadshotKillsPerGame float64 `json:"headshot_kills_per_game"`

	// ATH — disponibles uniquement pour le joueur A (local).
	PerfATH float64 `json:"perf_ath"`
	LusrATH float64 `json:"lusr_ath"`

	CareerRank int `json:"career_rank"`
	// CareerRankLabel : titre localisé du rang carrière ("Général Platine VI"),
	// résolu via le RankCatalog (même helper que le profil de combat). Vide si rang
	// inconnu (non-local sans rang récupéré).
	CareerRankLabel string `json:"career_rank_label,omitempty"`
	// HighestCSR : meilleur CSR courant (max sur les playlists ranked) de la saison
	// en cours, récupéré en live (même provider que le profil de combat). 0 si non
	// classé / indisponible. Comparable A vs B (les deux côtés via le même endpoint).
	HighestCSR float64 `json:"highest_csr"`
	// HighestCSRLabel : libellé tier du meilleur CSR courant ("Platine IV"). Vide si non classé.
	HighestCSRLabel string `json:"highest_csr_label,omitempty"`
	// HighestCSRAllTime : meilleur CSR de tous les temps (max all_time sur les playlists). 0 si aucun.
	HighestCSRAllTime float64 `json:"highest_csr_all_time"`
	// HighestCSRAllTimeLabel : libellé tier du meilleur CSR all-time ("Onyx", "Diamant II"). Vide si aucun.
	HighestCSRAllTimeLabel string `json:"highest_csr_all_time_label,omitempty"`
	// TimePlayedSeconds : temps de jeu cumulé (lifetime) issu du service record
	// Waypoint (champ TimePlayed ISO-8601). 0 si indisponible.
	TimePlayedSeconds int64          `json:"time_played_seconds,omitempty"`
	Extended          map[string]any `json:"extended,omitempty"`
}

// RemoteMedalCount est une médaille agrégée (lifetime) issue du service record
// Waypoint : identifiant + nombre d'occurrences. Enrichie ensuite en
// MedalDigestItem (label/description/image) via les métadonnées locales.
type RemoteMedalCount struct {
	NameID int64 `json:"name_id"`
	Count  int   `json:"count"`
}

// RemoteServiceRecord regroupe ce que l'endpoint service record Waypoint expose
// en un seul appel : les stats normalisées (avec TimePlayedSeconds) et la liste
// des médailles lifetime. Évite de re-fetcher pour les médailles / le temps joué.
type RemoteServiceRecord struct {
	Stats  NormalizedPlayerStats `json:"stats"`
	Medals []RemoteMedalCount    `json:"medals,omitempty"`
	// SeasonIDs : chemins CMS des saisons effectivement jouées par le joueur,
	// issus du bloc Subqueries.SeasonIds du service record lifetime (ex.
	// "Seasons/Season7.json", "Csr/Seasons/CsrSeason9-1.json"). Permet de boucler
	// sur les saisons réelles d'un joueur arbitraire (cf. season breakdown Explorer).
	SeasonIDs []string `json:"season_ids,omitempty"`
	// PlaylistAssetIDs : asset IDs des playlists effectivement engagées par le
	// joueur (Subqueries.PlaylistAssetIds). Intersecté avec les playlists ranked
	// pour ne requêter le CSR par saison que sur les playlists réellement jouées
	// (0 appel pour un joueur social — cf. season breakdown Explorer).
	PlaylistAssetIDs []string `json:"playlist_asset_ids,omitempty"`
}

// WeaponHighlight représente une arme mise en avant (top arme par kills),
// utilisée notamment par le profil Explorer (top_weapons).
type WeaponHighlight struct {
	WeaponID int64  `json:"weapon_id"`
	LabelFR  string `json:"label_fr"`
	LabelEN  string `json:"label_en"`
	Kills    int    `json:"kills"`
}

// PlayerATH regroupe les métriques all-time lues depuis stats.duckdb du joueur A.
type PlayerATH struct {
	CareerRank int
	PerfATH    float64
	LusrATH    float64
}

// CompareRequest est le body de POST .../pages/compare.
type CompareRequest struct {
	TargetGamertag string             `json:"target_gamertag"`
	Filters        FilterContextInput `json:"filters,omitempty"`
}

// Validate valide les champs de CompareRequest.
func (r CompareRequest) Validate() error {
	if r.TargetGamertag == "" {
		return fmt.Errorf("CompareRequest: target_gamertag requis")
	}
	return r.Filters.Validate()
}

// CompareMetricRow est une ligne de la table de comparaison.
//
// Metric est une CLÉ (pas un libellé) : le libellé affiché est résolu côté
// front à partir du registre canonique de champs (config/titles/{slug}/mappings/
// fields.toml exposé par /titles/{slug}/field-mappings) puis du dictionnaire
// local de la feature. Le contrat ne transporte donc AUCUN libellé de métrique
// (règle « jamais de label FR/EN en dur côté Go » — le champ label_fr a été
// retiré le 2026-08-04).
//
// ValueAAvailable / ValueBAvailable indiquent si la donnée est exploitable :
// false signifie « pas de source locale pour cette métrique » (joueur remote
// pour les métriques calculées depuis stats.duckdb / shared.match_participants,
// ou ATH non encore calculé). Le frontend rend « N/A » dans ce cas.
type CompareMetricRow struct {
	Metric          string  `json:"metric"`
	ValueA          float64 `json:"value_a"`
	ValueB          float64 `json:"value_b"`
	ValueAAvailable bool    `json:"value_a_available"`
	ValueBAvailable bool    `json:"value_b_available"`
	Delta           float64 `json:"delta"`                   // value_b - value_a
	Winner          string  `json:"winner"`                  // "a" | "b" | "tie"
	LessIsBetter    bool    `json:"less_is_better"`          // true = valeur basse meilleure (deaths_per_game, rendement, damage_taken_per_game)
	SampleSizeB     int     `json:"sample_size_b,omitempty"` // nb matchs B si joueur local croisé
	// DisplayA/DisplayB : libellé d'affichage prêt-à-rendre quand la valeur brute
	// n'est pas parlante (rang carrière → titre "Général Platine VI", CSR → "Or III").
	// Vide → le front formate ValueA/ValueB. La barre/winner restent sur les valeurs.
	DisplayA string `json:"display_a,omitempty"`
	DisplayB string `json:"display_b,omitempty"`
}

// CrossMatchSample regroupe les métriques locale-only calculées pour un joueur
// remote (B) sur les matchs croisés avec A (présents dans shared.match_participants
// pour les deux xuids). MatchesCount = 0 signifie aucun croisement exploitable.
type CrossMatchSample struct {
	MatchesCount         int
	MaxKillingSpree      int
	AvgLifeSecs          float64
	PerfectKillsPerGame  float64
	HeadshotKillsPerGame float64
}

// CompareEncounterStats : stats de rencontres historiques entre joueur A et joueur B.
// Alimenté depuis shared.match_participants + shared.killer_victim_pairs — best-effort.
type CompareEncounterStats struct {
	TotalEncounters int
	AllyCount       int
	EnemyCount      int
	WinrateAsAlly   *float64
	WinrateVsEnemy  *float64
	KillsDealt      int // frags de A sur B (toutes occurrences)
	DeathsSuffered  int // frags de B sur A (toutes occurrences)
}

// CompareResponse est la réponse de POST .../pages/compare.
type CompareResponse struct {
	PlayerA         NormalizedPlayerStats `json:"player_a"`
	PlayerB         NormalizedPlayerStats `json:"player_b"`
	Metrics         []CompareMetricRow    `json:"metrics"`
	TitleSlug       string                `json:"title_slug"`
	EncounterBadges []MatchEncounterBadge `json:"encounter_badges,omitempty"`
}
