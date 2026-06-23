package halo_5

// dto_carnage.go — projection du carnage report Halo 5 (/h5/{mode}/matches/{id}).
//
// Le carnage est la SEULE source du ROSTER complet d'un match : la liste de matchs
// (/players/{gt}/matches) ne porte que le joueur self. On ne projette que les
// champs consommés par l'ingestion participants (scoreboard par-joueur + équipes).
// Identité GAMERTAG-keyée (Player.Xuid toujours null).

// H5CarnageResponse — racine du carnage report (scoreboard étendu).
type H5CarnageResponse struct {
	PlayerStats []H5CarnagePlayer `json:"PlayerStats"`
	TeamStats   []H5CarnageTeam   `json:"TeamStats"`
	IsTeamGame  bool              `json:"IsTeamGame"`
}

// H5CarnageTeam — score + rang d'équipe (Rank 1 = équipe gagnante).
type H5CarnageTeam struct {
	TeamId int `json:"TeamId"`
	Score  int `json:"Score"`
	Rank   int `json:"Rank"`
}

// H5CarnagePlayer — stats par joueur du carnage. ⚠ PAS de KDA/Accuracy/DamageTaken
// natifs : l'API h5 ne fournit que les comptes bruts + les dégâts INFLIGÉS
// (TotalWeaponDamage). Le KDA est une stat d'API (jamais fabriquée) ; la résistance
// dégrade proprement faute de dégâts subis.
type H5CarnagePlayer struct {
	Player                H5PlayerRef `json:"Player"`
	TeamId                int         `json:"TeamId"`
	Rank                  int         `json:"Rank"`
	DNF                   bool        `json:"DNF"`
	PlayerScore           int         `json:"PlayerScore"`
	TotalKills            int         `json:"TotalKills"`
	TotalDeaths           int         `json:"TotalDeaths"`
	TotalAssists          int         `json:"TotalAssists"`
	TotalHeadshots        int         `json:"TotalHeadshots"`
	TotalShotsFired       int         `json:"TotalShotsFired"`
	TotalShotsLanded      int         `json:"TotalShotsLanded"`
	TotalWeaponDamage     float64     `json:"TotalWeaponDamage"`
	TotalMeleeKills       int         `json:"TotalMeleeKills"`
	TotalGrenadeKills     int         `json:"TotalGrenadeKills"`
	TotalPowerWeaponKills int         `json:"TotalPowerWeaponKills"`
	// Mécaniques de kill NATIVES Halo 5 (agrégats par-joueur du carnage, DISJOINTS de
	// TotalMeleeKills — l'assassinat/ground pound/shoulder bash ont leur propre compteur).
	// Absents des autres titres (→ nil côté canonical/domain pour Infinite).
	// NB : WeaponStats[] (tirs par arme) est documenté au schéma mais 343 le sert VIDE en
	// pratique (sonde live JGtm 2026-06-23 : n=0) → non modélisé. Le per-kill arme est
	// dans /events, sans compteur de tirs → pas de précision par-arme calculable.
	TotalAssassinations    int    `json:"TotalAssassinations"`
	TotalGroundPoundKills  int    `json:"TotalGroundPoundKills"`
	TotalShoulderBashKills int    `json:"TotalShoulderBashKills"`
	AvgLifeTimeOfPlayer    string `json:"AvgLifeTimeOfPlayer"` // ISO8601 "PT..S"
	TotalTimePlayed        string `json:"TotalTimePlayed"`     // ISO8601 "PT..S"
	// XpInfo : progression SR (rang XP de compte) du joueur. SEULE source du SR —
	// ni la liste de matchs ni le service record ne le portent (cf. PLAN_H5_ASSETS).
	XpInfo *H5XpInfo `json:"XpInfo"`
	// ProgressiveCommendationDeltas : commendations NATIVES Halo 5 progressées sur CE
	// match (AXE B prod-gate). Liste de {Id, PreviousProgress, Progress} par
	// commendation touchée — le COMPTE de CE match = Progress − PreviousProgress
	// (analogue au compteur par-match de medals_earned). Source per-match CONFIRMÉE
	// (sonde live) : seul endroit où l'API h5 expose la progression de commendations.
	ProgressiveCommendationDeltas []H5CommendationDelta `json:"ProgressiveCommendationDeltas"`
	// MetaCommendationDeltas : même forme (commendations « méta »/agrégées). Vide dans
	// la sonde — ignoré en Phase 1 (cf. AXE B), mappé sur le même chemin si peuplé.
	MetaCommendationDeltas []H5CommendationDelta `json:"MetaCommendationDeltas"`
}

// H5CommendationDelta — progression d'UNE commendation native Halo 5 sur un match.
// Id = UUID de commendation (clé naturelle, jamais résolu en numérique côté h5).
// Le compte gagné CE match = Progress − PreviousProgress (≥ 0 attendu ; un delta
// ≤ 0 est une commendation présente sans progression → ignoré à l'extraction).
type H5CommendationDelta struct {
	Id               string `json:"Id"`
	PreviousProgress int    `json:"PreviousProgress"`
	Progress         int    `json:"Progress"`
}

// H5XpInfo — progression SR Halo 5 du joueur dans le match. SpartanRank = niveau SR
// courant (1..152, 152 = MAX) ; TotalXP = XP de compte cumulé. Lu dans la carnage
// (PlayerStats[].XpInfo) pour alimenter le rang XP de la page Carrière.
type H5XpInfo struct {
	SpartanRank int `json:"SpartanRank"`
	TotalXP     int `json:"TotalXP"`
}
