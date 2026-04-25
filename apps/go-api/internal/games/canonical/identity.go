package canonical

// PlayerIdentity représente une identité joueur résolue par un titre.
//
// xuid est la chaîne canonique externe (string et non uint64) pour préserver
// les bit-patterns Halo qui débordent en int64 signé.
type PlayerIdentity struct {
	XUID               string
	Gamertag           string
	GamertagNormalized string
	ServiceTag         string
	EmblemURL          string
	AvatarURL          string
	IsBot              bool
}

// PlayerStats agrège les statistiques d'un joueur sur une période donnée.
//
// Les pointeurs flottants sur les ratios autorisent la distinction
// "non disponible" vs "0 explicite" demandée par le canonique.
type PlayerStats struct {
	Identity      PlayerIdentity
	MatchesPlayed int
	Wins          int
	Losses        int
	Ties          int
	WinRate       *float64
	Kills         int
	Deaths        int
	Assists       int
	KDR           *float64
	KDA           *float64
	Accuracy      *float64
}
