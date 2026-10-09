package domain

// replay_link.go — CE QUE LA BASE SAIT D'UN MATCH POUR EN FAIRE UN LIEN.
//
// La notification « rejeux prêts » (lot B v7.5) énumère des matchs. Un identifiant nu ne
// sert à rien : le lecteur veut cliquer. Or la page de rejeu du front est JOUEUR-SCOPÉE
// (/t/{titre}/players/{joueur}/matches/{match}/replay) — il n'existe aucune route de match
// sans joueur. Il faut donc, pour chaque match, UN joueur connu de l'instance qui y a
// participé ; le nom de carte s'y ajoute parce qu'il est dans la même ligne de registre et
// qu'il rend la liste lisible.
//
// TOUT EST OPTIONNEL PAR CONSTRUCTION. Un match sans participant connu (film du cache d'un
// match jamais synchronisé, joueur retiré de db_profiles) rend un XUID vide : la ligne
// s'affiche sans lien plutôt que de ne pas s'afficher.

// ReplayLinkTarget : la cible de lien d'un match.
type ReplayLinkTarget struct {
	// MatchID : identité COMPLÈTE, telle qu'indexée par match_registry.
	MatchID string
	// XUID d'un joueur connu ayant participé ; vide si aucun.
	XUID string
	// MapName : nom de carte brut du registre ; vide si absent.
	MapName string
}
