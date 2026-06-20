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
	AvgLifeTimeOfPlayer   string      `json:"AvgLifeTimeOfPlayer"` // ISO8601 "PT..S"
	TotalTimePlayed       string      `json:"TotalTimePlayed"`     // ISO8601 "PT..S"
}
