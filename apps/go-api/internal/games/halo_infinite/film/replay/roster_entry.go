package replay

// roster_entry.go — LE JOUEUR DU FILM TEL QUE LE DOCUMENT LE PUBLIE.
//
// EXTRAIT DE `document.go` AU LOT 1.9.14, par DEPLACEMENT PUR : ce fichier-la passait 627
// lignes et le lot lui ajoute le SIEGE (cf. sieges.go). Le type et son siege vivent desormais
// cote a cote — la regle de lecture d'une fiche se lit d'un seul fichier.

// RosterEntry est un joueur du film : son identité, et l'index sous lequel le film le désigne.
//
// LES DEUX NE SONT PAS INTERCHANGEABLES. Le xuid IDENTIFIE ; l'index ORDONNE, et il n'a de
// sens qu'à l'intérieur de ce film. Les garder côte à côte est ce qui permet de traduire un
// événement (qui porte l'index) sans jamais confondre l'un avec l'autre.
type RosterEntry struct {
	// XUID en décimal, même forme que Track.XUID et que la base.
	XUID string `json:"xuid"`
	// FilmIndex est l'index du joueur DANS CE FILM, lu dans les cinq bits qui précèdent son
	// xuid (cf. player_index.go, 26 chunks concordants sur le film de référence).
	FilmIndex int `json:"filmIndex"`
	// Name est le gamertag TEL QUE LE FILM L'ÉCRIT, dans le même enregistrement que le xuid.
	//
	// CE N'EST PAS UNE RÉSOLUTION : rien n'est allé le chercher ailleurs, donc rien ne peut
	// l'avoir mal apparié. Il rend le rejeu lisible sans base de données. Ce qu'il ne donne
	// PAS, et que seule la base porte : les compteurs du match. Vide si l'enregistrement ne le
	// portait pas.
	Name string `json:"name,omitempty"`
	// Team est le DÉSIGNATEUR D'ÉQUIPE QUE LE FILM ÉCRIT (schéma 57, lot 1.7), par index de
	// joueur : `0..8` pour les huit camps de `mp_team_designator`, `-1` pour « AUCUNE ÉQUIPE »
	// (ce que le moteur écrit sur un mode sans camps).
	//
	// IL EST UN POINTEUR, ET C'EST LA SEULE FORME JUSTE. Trois états existent, pas deux :
	// ABSENT = le film n'a pas nommé ce joueur (ou l'artefact est antérieur au schéma 57) ;
	// `-1` = le film dit « aucune équipe » ; `0..8` = le camp. Un entier nu les confondrait —
	// un artefact 56 relu vaudrait `0` partout, c'est-à-dire « tout le monde dans le camp 0 »,
	// et `MIN_RENDERABLE_SCHEMA_VERSION` vaut 27 : le client lit encore des artefacts anciens.
	//
	// MÊME SOURCE QUE `Track.Team`, à la clé près : celui-ci est indexé par `filmIndex`, donc il
	// vaut aussi pour un joueur dont aucune vie n'est publiée et pour un BOT — que le film
	// assoit au même index.
	//
	// DEPUIS LE LOT 1.9.14 (2026-09-15), LE REJEU COLORE ET GROUPE PAR CE CHAMP quand il est
	// présent, et retombe sur `team_side` de la feuille de match — affiché comme tel — quand il
	// est absent. C'était l'inverse jusque-là (la feuille décidait, le film n'était qu'une
	// donnée), et V4 l'avait laissé ainsi pour que le lot 1.7 ne change aucun rendu. Il vaut
	// aussi pour un joueur que la feuille ne porte PAS — un remplaçant absent du tableau de
	// score y gagne son camp.
	Team *int `json:"team,omitempty"`
	// Bot est vrai pour une entrée déclarée par BOT_METADATA (schéma 36) : son XUID est VIDE
	// — un bot n'en a pas, et le normaliser en pseudo-identifiant fusionnerait des bots — et
	// son Name porte le suffixe « [bot] », comme la base l'écrit. FilmIndex est le slot du
	// roster de réplication que le paquet type 12 déclare.
	Bot bool `json:"bot,omitempty"`
	// Bid est l'identifiant STABLE d'un bot, forme `bid(N.0)` — la même que `match_participants`
	// écrit (schéma 50).
	//
	// IL ÉTAIT LU DEPUIS TOUJOURS ET N'ÉTAIT PAS PUBLIÉ (inventaire P1, E9). Faute de l'avoir,
	// la jointure web d'un bot se faisait sur le NOM NU — égalité de chaîne des deux côtés —,
	// et le commentaire de `rosterLogic.ts` le disait lui-même : deux bots homonymes fusionnent.
	// Vide pour un humain, et vide pour un bot dont la déclaration ne portait pas d'identifiant :
	// un `bid(0.0)` inventé joindrait deux bots distincts.
	Bid string `json:"bid,omitempty"`
	// Seat est LE SIÈGE : la fiche que cette entrée occupe à l'écran (lot 1.9.14).
	//
	// IL VAUT `FilmIndex` DANS L'ÉCRASANTE MAJORITÉ DES CAS, et il en diffère exactement quand
	// cette entrée CONTINUE le siège d'un partant sans que le film ait réutilisé son index —
	// le chaînage est alors un repli, et [RosterEntry.SeatSource] le dit. Deux entrées qui
	// portent le même `Seat` sont deux occupants SUCCESSIFS d'une même fiche ; leurs présences
	// (les vies de `tracks[]`) ne se recouvrent pas.
	//
	// POURQUOI UN CHAMP DE PLUS PLUTÔT QUE `FilmIndex` RÉUTILISÉ. L'index est une LECTURE du
	// film et ne doit jamais être écrasé par une déduction : un événement qui porte un index
	// se rattache par `FilmIndex`, une fiche se dessine par `Seat`. Les confondre rendrait le
	// repli indétectable et casserait le rattachement.
	//
	// TOUJOURS ÉMIS, SANS `omitempty` : le siège 0 est un siège comme un autre, et `omitempty`
	// l'effacerait — le premier joueur du roster perdrait sa fiche. Absent des artefacts
	// antérieurs au lot 1.9.14 : un client qui ne le trouve pas retombe
	// sur `FilmIndex`, ce qui est le comportement d'un film sans relais.
	Seat int `json:"seat"`
	// SeatSource dit D'OÙ VIENT le siège : [SeatSourceLu] (l'index que le film écrit, reprise
	// écrite comprise) ou [SeatSourceApparie] (l'appariement ordinal par camp, un repli nommé
	// et compté — cf. sieges.go). Vide sur un artefact antérieur au lot 1.9.14.
	SeatSource string `json:"seatSource,omitempty"`
}
