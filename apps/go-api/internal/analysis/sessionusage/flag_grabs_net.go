package sessionusage

// flag_grabs_net.go — LA LIGNE DES PRISES NETTES DE DRAPEAU lue dans `match_flag_grabs_net_latest`.
//
// Elle est jointe, match par match et joueur par joueur, à la colonne `flag_grabs_net` de
// l'objectif des formes retenues (squadagg/squad_formes.go, joindrePrisesNettes) : un match sans
// prise lue ne reçoit aucune valeur — « non mesuré », jamais un zéro.

// FlagGrabsNetRow — une ligne (match, joueur) de `match_flag_grabs_net_latest`, réduite aux
// colonnes que la jointure lit (les prises brutes et les ouvertures de portage restent en base).
type FlagGrabsNetRow struct {
	MatchID string
	XUID    string
	// Net : les prises du film, jonglage replié.
	Net int
	// WindowMS : la fenêtre sous laquelle Net a été calculé.
	WindowMS int
}
