// Package domain — match_elevation.go : LE DÉNIVELÉ DES ENGAGEMENTS D'UN MATCH, AU GRAIN DU FRAG.
//
// Décision D24 du plan .ai/V7.5/PLAN_AJUSTEMENTS_SUPP_PRE_V75_2026-09-21.md (forme M1 de la
// maquette MAQUETTE_DENIVELE_V2_2026-09-22.html) : « l'important c'est de voir où on meurt et
// où on frag ». La clé publiée n'est donc PAS (arme, côté) — celle de `WeaponRange`, faite pour
// des centaines de frags — mais LE FRAG LUI-MÊME. Un match en donne 10 à 25 par côté : à ce
// grain il n'y a ni seuil de mesure, ni percentile, ni binning à défendre, et chaque point
// reste un événement qu'un joueur se rappelle.
//
// # LE SIGNE EST CELUI DU JOUEUR CONSULTÉ, DES DEUX CÔTÉS
//
// `DeltaZM` répond toujours à la même question : « MOI, étais-je au-dessus ou en dessous de
// l'autre ? ». Positif = le joueur consulté était plus haut, POUR UN FRAG COMME POUR UNE MORT.
// C'est `analysis.signedElevation` qui pose cette convention, écrite une seule fois et testée
// aux deux bornes ; la base, elle, ne porte que la grandeur physique `killer_z - victim_z`.
//
// # LE LOBBY N'EST PAS UNE SÉRIE, C'EST UNE RÉFÉRENCE
//
// `Lobby` porte TOUS les frags mesurés du match, lus du côté du TUEUR, sans ceux du joueur
// consulté (qui sont déjà dans `Kills`). Il n'est pas une seconde lecture du match : il sert
// de fond de comparaison au bouton « comparer au lobby » de la vue match, avec
// `LobbyMedianDeltaZM` comme repère. Son signe est celui du tueur de chaque ligne — sur un
// nuage de fond, « qui » n'est pas la question.
package domain

// MatchElevationBlock — le bloc `combat_tab.elevation`.
//
// ABSENT (nil) quand le match n'a aucun frag mesuré : un nuage vide n'est pas un nuage, et
// l'UI n'a alors pas de carte à rendre. Une section qui s'affiche vide se lit « panne ».
type MatchElevationBlock struct {
	// Kills : les engagements du JOUEUR CONSULTÉ, ses frags et ses morts mélangés, chacun
	// portant son `Side`. Une seule liste parce que les deux séries partagent exactement
	// les mêmes axes et la même convention de signe : les séparer en deux champs ferait
	// deux formes à tenir en phase pour zéro information de plus.
	Kills []MatchElevationKill `json:"kills"`

	// Lobby : les frags mesurés des AUTRES joueurs, côté tueur. Vide quand le match ne
	// porte que ceux du joueur consulté.
	Lobby []MatchElevationKill `json:"lobby,omitempty"`

	// LobbyMedianDeltaZM : la médiane du dénivelé de `Lobby`, en mètres — le repère du
	// mode « comparer au lobby ». Vaut 0 quand `Lobby` est vide, ce qui est alors la
	// valeur d'un repère qui ne se trace pas (l'UI ne le trace que si `Lobby` a des points).
	LobbyMedianDeltaZM float64 `json:"lobby_median_delta_z_m"`

	// MeasuredKills : les FRAGS du joueur consulté qui portent une mesure ; TotalKills :
	// ceux que le tableau des scores lui compte. Les deux ensemble portent la RÉSERVE DE
	// COUVERTURE de la carte — un nuage sans elle laisserait croire à l'exhaustivité
	// (couverture plancher mesurée du corpus : 75,8 %). TotalKills vaut 0 quand le
	// scoreboard ne porte pas la ligne du joueur : l'UI n'écrit alors pas de fraction.
	MeasuredKills int `json:"measured_kills"`
	TotalKills    int `json:"total_kills"`
}

// MatchElevationKill — UN engagement mesuré : un point du nuage.
type MatchElevationKill struct {
	// DistanceM : la distance 3D tueur <-> victime à l'instant du coup fatal, en mètres.
	DistanceM float64 `json:"distance_m"`
	// DeltaZM : le dénivelé SIGNÉ DU CÔTÉ DU JOUEUR CONSULTÉ, en mètres (cf. l'en-tête).
	DeltaZM float64 `json:"delta_z_m"`
	// TimeMS : l'instant sur l'HORLOGE DU MATCH (`match_kill_events.time_ms`, cf.
	// TacticalClockMatch) — le web ouvre le rejeu avec `?t=<time_ms>&clock=match`.
	TimeMS int64 `json:"time_ms"`
	// Weapon, WeaponEN : le LIBELLÉ de l'arme du tueur, résolu par le référentiel du titre.
	// Jamais une clé brute ni un `source_tag` : une clé de registre n'est pas un mot
	// d'interface. Vides quand le titre n'a pas de libellé pour cette clé.
	Weapon   string `json:"weapon,omitempty"`
	WeaponEN string `json:"weapon_en,omitempty"`
	// Side : `MatchElevationSideKill` ou `MatchElevationSideDeath`, du point de vue du
	// joueur consulté. Les lignes de `Lobby` portent toujours le côté frag.
	Side string `json:"side"`
	// Opponent : le gamertag de L'AUTRE — la victime sur un frag, le tueur sur une mort.
	// Vide quand le film ne l'a pas résolu (bot non nommé, kill-feed muet).
	Opponent string `json:"opponent,omitempty"`
}

// Les deux côtés publiés. Ce sont des mots de CONTRAT, pas des libellés : l'UI les traduit.
const (
	MatchElevationSideKill  = "kill"
	MatchElevationSideDeath = "death"
)

// MatchElevationKillRaw — une mort mesurée telle que le LECTEUR la rend, avant tout point de
// vue : les deux identités, le libellé d'arme, et le dénivelé PHYSIQUE `killer_z - victim_z`.
//
// Elle existe parce que le repo ne sait pas POUR QUI il lit (il lit le match entier) et que
// l'inversion de signe est une règle d'analyse, écrite une seule fois. Le repo la remplit,
// `analysis.BuildMatchElevation` la consomme.
type MatchElevationKillRaw struct {
	KillerXUID     string
	KillerGamertag string
	VictimXUID     string
	VictimGamertag string
	Weapon         string
	WeaponEN       string
	TimeMS         int64
	DistanceM      float64
	// DeltaZ : `killer_z - victim_z`, en mètres. JAMAIS inversé ici.
	DeltaZ float64
}
