// Package domain — compare_weapons.go : LE PROFIL D'ARMES DU FACE-À-FACE
// (plan .ai/PLAN_COMPARE_PROFIL_ARMES_2026-09-17.md).
//
// # CE QUE CE FICHIER PORTE AU LOT 2, ET CE QU'IL PORTERA AU LOT 3
//
// Au lot 2, uniquement le SCOPE : l'ensemble de matchs sur lequel un côté de la comparaison
// est mesuré, et les totaux du joueur sur cet ensemble. Le contrat de réponse
// (`CompareWeaponProfile` et ses blocs) arrive au lot 3, avec son producteur — écrire des
// types que personne n'assemble encore en ferait du code mort le temps d'un lot.
package domain

// CompareWeaponScope est L'ENSEMBLE DE MATCHS d'un côté de la comparaison, et ce que le
// joueur y a fait.
//
// # POURQUOI LES MATCH_ID ET LES TOTAUX VOYAGENT ENSEMBLE
//
// Les deux lecteurs d'armes (frags par arme, frags mesurés) se bornent par `MatchIDs` — sans
// quoi ils balaieraient une table PARTAGÉE, tous joueurs confondus, ce que leur `Validate()`
// refuse. Les totaux, eux, sont les DÉNOMINATEURS de la publication : « N frags mesurés sur
// M » et la part de chaque classe de frags. Les tirer d'une seconde requête les ferait porter
// sur un scope potentiellement différent (une écriture concurrente entre les deux lectures
// suffit) ; ils sont donc agrégés SUR LES MÊMES LIGNES que les match_id.
//
// # LES DEUX SCOPES POSSIBLES, ET POURQUOI ILS NE SE MÉLANGENT PAS (D2)
//
// Un joueur LOCAL est mesuré sur TOUS ses matchs (campagne exclue) : c'est la même doctrine
// que les métriques déjà publiées par la page. Un joueur NON local n'existe localement que
// dans les matchs qu'il a joués AVEC le joueur courant : son scope est l'intersection, et ce
// n'est pas la même population — d'où `IsSample` côté réponse et la note « sur N matchs ».
// Confondre les deux publierait un profil d'arme de carrière là où seule une poignée de
// matchs a été observée.
type CompareWeaponScope struct {
	// MatchIDs borne les deux lectures d'armes. Jamais vide quand le scope existe : un
	// scope sans match n'est pas construit (le repo rend (nil, nil)).
	MatchIDs []string
	// Matches est le nombre de matchs du scope — publié tel quel (« sur N matchs »).
	Matches int
	// Kills, Deaths sont les totaux du joueur sur le scope : dénominateurs de la part des
	// frags par classe et de la couverture de la portée.
	Kills, Deaths int
	// MeleeKills, GrenadeKills sont les compteurs NATIFS de l'API. Ils ne se déduisent pas
	// du registre d'armes : la mêlée et la grenade sont servies par ces compteurs, jamais
	// par une classification d'arme (cf. service/fragdist).
	MeleeKills, GrenadeKills int
}
