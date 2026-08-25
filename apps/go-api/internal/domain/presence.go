package domain

// PlayerPresence est l'état de présence d'un joueur suivi, tel qu'exposé par
// GET /api/v1/presence au shell (sélecteur de joueur de la navigation).
//
// InGame répond à « ce joueur est-il en jeu sur un titre suivi par LevelUp ? »,
// quel que soit le titre configuré pour lui : un joueur configuré Halo 5 qui
// lance Halo Infinite est en jeu, et TitleSlug dit sur lequel. Champs de titre
// vides ⇔ InGame faux.
type PlayerPresence struct {
	PlayerSlug string `json:"player_slug"`
	Gamertag   string `json:"gamertag"`
	InGame     bool   `json:"in_game"`
	TitleSlug  string `json:"title_slug,omitempty"`
	TitleName  string `json:"title_name,omitempty"`
}

// PresenceSnapshot est la réponse de GET /api/v1/presence.
//
// Players ne contient que les joueurs accessibles à l'utilisateur courant
// (ADR 0029).
//
// FriendsInGame compte MES AMIS EN JEU, au sens produit du 2026-08-25 : les
// joueurs inscrits que l'utilisateur courant voit dans son cercle (les siens et
// ceux de ses co-membres de groupe) SANS en être le propriétaire, actuellement
// vus sur un titre suivi. Le compte est donc PERSONNEL — deux utilisateurs de la
// même instance obtiennent deux valeurs différentes — et un joueur étranger à
// son cercle n'y entre jamais. Seul l'entier est exposé : les identités servies
// restent celles de Players.
//
// Watcher éteint ou indisponible : liste vide et compteur à zéro — jamais une
// erreur, la présence est une information d'agrément.
type PresenceSnapshot struct {
	Players       []PlayerPresence `json:"players"`
	FriendsInGame int              `json:"friends_in_game" doc:"Nombre de joueurs du cercle de l'utilisateur (visibles mais non possédés par lui) actuellement en jeu sur un titre suivi."`
}
