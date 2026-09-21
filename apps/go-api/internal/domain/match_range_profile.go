package domain

// match_range_profile.go — LA PORTÉE D'UN JOUEUR, RAPPORTÉE À SON LOBBY (plan
// .ai/V7.5/PLAN_AJUSTEMENTS_SUPP_PRE_V75_2026-09-21.md, lot N2, décisions D22-4 et D22-5).
//
// # POURQUOI L'ÉCART AU LOBBY, ET JAMAIS LA MÉDIANE BRUTE
//
// La médiane de frag d'un joueur ne dit rien seule : sur un BTB à Fragmentation le lobby
// entier frague autour de 24 m, sur une arène à Live Fire autour de 11 m. Un même 30 m est
// un tireur d'élite modeste dans le premier cas, une anomalie extrême dans le second. Tracée
// brute, la courbe d'un joueur ne raconte que la playlist de sa soirée. C'est `LobbyDeltaM`
// — l'écart à la médiane de TOUS les joueurs du match, camp adverse compris — qui neutralise
// la carte et le mode et laisse apparaître la façon de jouer.
//
// # LA MÉDIANE DU LOBBY N'EST PAS LA MOYENNE DES MÉDIANES
//
// Elle se calcule sur LES FRAGS du match, pas sur les joueurs. Moyenner les médianes par
// joueur pèserait autant un joueur à trois frags qu'un joueur à trente, et déplacerait le
// référentiel au gré des effectifs décodés. Garde-rail : match_range_profile_test.go.
//
// # `Measured` VOYAGE AVEC CHAQUE MÉDIANE, ET CE N'EST PAS DÉCORATIF
//
// La couverture des positions décodées tourne autour de 76 %, et elle est INÉGALE d'un
// joueur à l'autre du même match. Une médiane sans son effectif laisserait croire à
// l'exhaustivité ; c'est aussi ce qui permet au client de creuser le point sous son seuil
// (5 frags mesurés, D22-5) au lieu de l'effacer.

import "time"

// MatchRangePlayer est la portée d'UN joueur sur UN match.
type MatchRangePlayer struct {
	XUID string `json:"xuid"`
	// Gamertag est vide quand l'appelant ne nomme pas ce joueur (il n'est alors jamais
	// inventé côté Go : le client retombe sur le xuid).
	Gamertag string `json:"gamertag,omitempty"`
	// MedianM est la médiane, en mètres, des distances de ses frags MESURÉS du match.
	MedianM float64 `json:"median_m"`
	// LobbyDeltaM est `MedianM - LobbyMedianM` : SIGNÉ, positif au-dessus du lobby. C'est
	// l'axe du nuage de l'Escouade et la hauteur du bâton de la session.
	LobbyDeltaM float64 `json:"lobby_delta_m"`
	// Measured est le nombre de frags mesurés du joueur sur ce match — le dénominateur de
	// sa médiane. Toujours >= 1 : un joueur sans frag mesuré est ABSENT, jamais à zéro.
	Measured int `json:"measured"`
}

// MatchRangeProfile est le profil de portée d'UN match : le référentiel du lobby et les
// joueurs publiés.
//
// Un match dont AUCUN frag n'est mesuré (film non décodé, positions absentes) est ABSENT de
// la liste — il n'a pas de référentiel, donc aucun écart n'y a de sens.
type MatchRangeProfile struct {
	MatchID  string    `json:"match_id"`
	PlayedAt time.Time `json:"played_at"`
	// MapName nomme la carte pour l'étiquette « #N · carte » des graphes de la page.
	MapName string `json:"map_name,omitempty"`
	// Players sont les joueurs PUBLIÉS par l'appelant (le roster affiché pour l'Escouade,
	// le seul joueur consulté pour la Session), dans l'ordre qu'il a demandé. Le lobby
	// complet n'est JAMAIS publié : il sert de référentiel, pas de tableau nominatif.
	Players []MatchRangePlayer `json:"players"`
	// LobbyMedianM est la médiane des distances de TOUS les frags mesurés du match — le
	// référentiel. Elle porte sur le lobby entier même quand `Players` n'a qu'une ligne.
	LobbyMedianM float64 `json:"lobby_median_m"`
	// LobbyMeasured est le nombre de frags mesurés du match, tous joueurs confondus.
	LobbyMeasured int `json:"lobby_measured"`
}

// MatchRangeBlock est le bloc servi aux pages : les profils et la couverture de la mesure.
//
// Bloc nil (jamais un bloc vide) quand le titre n'a pas de décodeur de film, quand le repo
// n'est pas câblé ou quand la lecture échoue : une OMISSION, jamais des zéros qui se
// liraient comme une mesure.
type MatchRangeBlock struct {
	Profiles []MatchRangeProfile `json:"profiles"`
	// KillsMeasured / KillsTotal sont les DEUX TERMES de la couverture (frags mesurés sur
	// frags publiables du scope), jamais leur quotient : le pourcentage se calcule là où il
	// s'affiche, et un dénominateur à zéro s'y lit « couverture inconnue ».
	KillsMeasured int `json:"kills_measured"`
	KillsTotal    int `json:"kills_total"`
}
