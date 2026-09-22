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
	// ElevationMedianM est la médiane du DÉNIVELÉ SIGNÉ de ses frags mesurés du match, en
	// mètres : `killer_z - victim_z` du côté tueur, POSITIF quand il frague depuis le haut
	// (cf. analysis.MeasuredKill.DeltaZ — le signe n'est jamais redressé). C'est l'axe
	// vertical du dénivelé (proposition E1, 2026-09-22), la distance au sol étant l'autre.
	//
	// POINTEUR, et pas un zéro : un 0 m se lirait « il frague à plat », ce qui est une
	// mesure ; l'absence dit « pas de dénivelé mesuré ».
	ElevationMedianM *float64 `json:"elevation_median_m,omitempty"`
	// ElevationLobbyDeltaM est `ElevationMedianM - LobbyElevationMedianM` : SIGNÉ, positif
	// au-dessus du lobby. MÊME normalisation que LobbyDeltaM, et pour la même raison — le
	// dénivelé typique d'un match dépend de la carte, pas du joueur. Absent dès que l'une
	// des deux médianes manque.
	ElevationLobbyDeltaM *float64 `json:"elevation_lobby_delta_m,omitempty"`
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
	// LobbyElevationMedianM est la médiane du dénivelé signé de TOUS les frags mesurés du
	// match — le référentiel vertical. Comme la médiane de portée, elle se calcule SUR LES
	// FRAGS, jamais comme la moyenne des médianes par joueur.
	LobbyElevationMedianM *float64 `json:"lobby_elevation_median_m,omitempty"`
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

// RangeReferenceBlock est la PÉRIODE DE RÉFÉRENCE de la portée, servie à la colonne de
// session (lot U du plan AJSUP, décision D23-4).
//
// # POURQUOI UN SECOND BLOC PLUTÔT QU'UN SCOPE PLUS LARGE SUR `RangeProfiles`
//
// `RangeProfiles` répond à « qu'ai-je fait CE SOIR » (un bâton par match de la session) ;
// celui-ci répond à « quel joueur suis-je SUR LA PÉRIODE », et c'est lui qui porte les
// bandes de rôle. Les deux se lisent sur le même axe (l'écart au lobby) mais pas sur le
// même scope : fusionner les deux aurait forcé le client à re-découper la période pour
// retrouver la session.
//
// # LE MÊME BLOC SERT LES DEUX COLONNES DU DRAWER
//
// La référence est celle du FILTRE de la page, pas de la session affichée — donc la session
// comparée partage exactement la même, et il n'y a pas de `compare_range_reference`. Le
// client met en surbrillance la fenêtre de chaque session dans le même nuage.
type RangeReferenceBlock struct {
	// Profiles sont les profils de la période, DU PLUS ANCIEN AU PLUS RÉCENT, ne publiant
	// que le joueur consulté (la médiane de lobby, elle, porte toujours sur le lobby
	// entier du match : c'est le référentiel).
	Profiles []MatchRangeProfile `json:"profiles"`
	// RoleLowM / RoleHighM sont les deux bandes de rôle : tiers 1/3 et 2/3 des écarts au
	// lobby des matchs de la période à au moins 5 frags mesurés. ABSENTS (nil) sous 3
	// points pleins — deux matchs ne font pas un habituel.
	RoleLowM  *float64 `json:"role_low_m,omitempty"`
	RoleHighM *float64 `json:"role_high_m,omitempty"`
	// PeriodMedianDeltaM est l'écart au lobby MÉDIAN de la période, sur les mêmes points
	// pleins. Absent avec les deux bandes.
	PeriodMedianDeltaM *float64 `json:"period_median_delta_m,omitempty"`
	// MatchesMeasured / MatchesTotal sont les DEUX TERMES de la couverture de la période
	// (matchs porteurs d'au moins un frag mesuré sur matchs du scope), jamais leur
	// quotient — le pourcentage se calcule là où il s'affiche.
	MatchesMeasured int `json:"matches_measured"`
	MatchesTotal    int `json:"matches_total"`
}
