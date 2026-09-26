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
// FriendsInGame compte MES AMIS EN JEU, au sens produit du 2026-08-25.
//
// PÉRIMÈTRE, deux axes à ne pas confondre :
//
//   - les joueurs CANDIDATS sont exactement ceux de Players — donc ceux du TITRE
//     COURANT de la requête (en-tête X-LevelUp-Title), visibles par l'utilisateur
//     au sens ADR 0029 (les siens et ceux de ses co-membres de groupe). Changer
//     de titre change le parc compté, comme il change la liste servie ;
//   - « en jeu » se lit, lui, sur N'IMPORTE QUEL titre suivi : un joueur du parc
//     Halo Infinite vu sur Halo 5 compte, exactement comme sa manette s'allume
//     dans le sélecteur.
//
// De ces candidats sont retirés les profils dont l'utilisateur est PROPRIÉTAIRE
// direct — sur une instance sans propriété appliquée (auth désactivée, mode
// démo), personne ne possède rien et tous les visibles en jeu comptent. Le
// compte est donc PERSONNEL : deux utilisateurs de la même instance obtiennent
// deux valeurs différentes, et un joueur étranger au cercle n'y entre jamais.
// Seul l'entier est exposé : les identités servies restent celles de Players.
//
// Watcher éteint ou indisponible : liste vide et compteur à zéro — jamais une
// erreur, la présence est une information d'agrément.
type PresenceSnapshot struct {
	Players       []PlayerPresence `json:"players"`
	FriendsInGame int              `json:"friends_in_game" doc:"Amis en jeu : joueurs visibles du titre courant (même périmètre que players), non possédés par l'utilisateur, en jeu sur l'un des titres suivis."`
}
