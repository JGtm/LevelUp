package sessionusage

// flag_grabs_net.go — LA LIGNE DES PRISES NETTES DE DRAPEAU lue dans `match_flag_grabs_net_latest`.
//
// Elle est jointe, match par match et joueur par joueur, à la colonne `flag_grabs_net` de
// l'objectif des formes retenues (squadagg/squad_formes.go, joindrePrisesNettes) : un match sans
// prise lue ne reçoit aucune valeur — « non mesuré », jamais un zéro.

// FlagGrabsNetRow — une ligne (match, joueur) de `match_flag_grabs_net_latest`.
type FlagGrabsNetRow struct {
	MatchID string
	XUID    string
	// Raw / Net : les prises brutes LUES DU FILM, et les mêmes jonglage replié. Le « brut »
	// est ici celui du film — le compteur de l'API (`match_objective_stats.flag_grabs`) est
	// une AUTRE chaîne, et il n'entre jamais dans cette table.
	Raw, Net int
	// Openings : les ouvertures de portage comptées par l'oracle du film sur CE match. Valeur
	// de match, identique sur toutes les lignes du match. C'est le dénominateur de Raw.
	Openings int
	// WindowMS : la fenêtre sous laquelle Net a été calculé.
	WindowMS int
}
