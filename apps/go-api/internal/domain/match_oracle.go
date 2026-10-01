package domain

// match_oracle.go — CE QUE L'API DIT D'UN MATCH ET QUE LE CONSTRUCTEUR DE REJEU NE LIT PAS.
//
// POURQUOI UN TYPE DISTINCT DE MatchFacts. `MatchFacts` est une ENTREE de la cuisson : le triplet
// (frags, morts, assistances) apparie les slots, `teamScores` rattache les camps. Tout ce qu'il
// porte peut donc etre CIRCULAIRE pour un banc qui juge l'artefact (cf.
// .ai/V7.5/film_re/BANC_DE_VERITE_CONCEPTION_2026-09-30.md §0). `MatchOracle` porte l'inverse :
// des verites officielles que la cuisson n'a JAMAIS vues, et qu'elle ne doit pas voir — les
// ajouter a `MatchFacts` changerait la cle de cuisson et les digests d'equivalence sans rien
// changer a l'artefact (decision D-2 du banc de verite).
//
// CHAQUE CHAMP EST UN POINTEUR : nil = la colonne est NULL en base, ce qui n'est PAS zero. Un
// oracle qui dirait « 0 tir » pour une colonne absente jugerait faux un artefact juste.

// MatchPlayerOracle est la ligne officielle d'un joueur (`match_participants`), plus ses stats
// d'objectif (`match_objective_stats_latest`).
type MatchPlayerOracle struct {
	// XUID en decimal, meme forme que la base et que le rejeu (`bid(N.0)` pour un bot).
	XUID string `json:"xuid"`
	// PersonalScore / Score : les deux colonnes de score de la ligne de match. Laquelle egale le
	// score « affiche » du statborg reste a mesurer (banc de verite, O-S3) : les deux sont gardees.
	PersonalScore *int `json:"personalScore,omitempty"`
	Score         *int `json:"score,omitempty"`
	// ShotsFired / ShotsHit : les tirs officiels (O-X1).
	ShotsFired *int `json:"shotsFired,omitempty"`
	ShotsHit   *int `json:"shotsHit,omitempty"`
	// Les kills par categorie (rapprochement futur avec la classe de degat du kill-feed).
	HeadshotKills    *int `json:"headshotKills,omitempty"`
	MeleeKills       *int `json:"meleeKills,omitempty"`
	GrenadeKills     *int `json:"grenadeKills,omitempty"`
	PowerWeaponKills *int `json:"powerWeaponKills,omitempty"`
	// TimePlayedSeconds : duree officielle. NON DATEE sur l'horloge du film (plan §8.30).
	TimePlayedSeconds *int `json:"timePlayedSeconds,omitempty"`
	// PresentAtBeginning / PresentAtCompletion : presence BOOLEENNE, insensible au decalage
	// d'horloge (jusqu'a 25 s) qui interdit d'employer les instants d'arrivee et de depart.
	PresentAtBeginning  *bool `json:"presentAtBeginning,omitempty"`
	PresentAtCompletion *bool `json:"presentAtCompletion,omitempty"`
	// Objectives : les colonnes NON NULLES de `match_objective_stats_latest` pour ce joueur,
	// indexees par leur NOM DE COLONNE (ex. `flag_captures`, `time_as_skull_carrier_seconds`).
	// Une carte plutot que des champs : la table est large (un bloc par mode) et s'etend par
	// ALTER ; recopier ses colonnes ici ferait une troisieme liste a tenir synchronisee.
	Objectives map[string]float64 `json:"objectives,omitempty"`
}

// MatchOracle est l'ensemble des verites officielles d'un match qui NE SONT PAS des entrees de la
// cuisson du rejeu.
type MatchOracle struct {
	MatchID string              `json:"matchId"`
	Players []MatchPlayerOracle `json:"players,omitempty"`
}

// Empty dit qu'aucune ligne n'a ete trouvee.
func (o MatchOracle) Empty() bool { return len(o.Players) == 0 }
