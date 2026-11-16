package tactical

// reapparitions_types.go — LES TYPES DE LA LECTURE « REAPPARITION » : ce qu'une vie sait de
// la maniere dont elle commence et dont la precedente a fini, et l'instantane des autres
// joueurs a l'instant ou elle commence. La logique est dans reapparitions.go.

// SourceReapparition dit D'OU VIENT la decision « cette vie est une reapparition ».
const (
	// SourceFilm : la fin de la vie precedente est LUE dans le film (liberation du corps ou
	// debut de compte a rebours de reapparition).
	SourceFilm = "film"
)

// Raisons de fin de vie que le film ecrit pour un corps.
const (
	// RaisonLiberee : le corps est LIBERE (le joueur est mort et attend son compte a rebours).
	RaisonLiberee = "released"
	// RaisonSupprimee : le corps est SUPPRIME alors que le joueur le commande encore. C'est la
	// fin d'une MANCHE : la vie qui suit est un spawn de debut de manche, pas une reapparition.
	RaisonSupprimee = "deleted"
)

// FinLue est la fin d'une vie telle que le film la raconte : pourquoi, et a quelle frame.
type FinLue struct {
	// Raison vaut RaisonLiberee ou RaisonSupprimee ; toute autre valeur est ignoree.
	Raison string
	// Frame est la frame de la liberation ou de la suppression du corps.
	Frame int
}

// DebutCompte est le DEBUT d'un compte a rebours de reapparition : le joueur est mort a cet
// instant et attend. Le xuid est celui du joueur qui attend.
type DebutCompte struct {
	XUID  string
	Frame int
}

// Vivant est la position d'un joueur VIVANT a un instant : la matiere de la prevision
// (qui est ou quand quelqu'un reapparait). Un xuid vide est une vie que le film n'a pas
// nommee (bot, vie anonyme) : elle est vivante et compte, sans avoir d'identite.
type Vivant struct {
	XUID   string  `json:"xuid,omitempty"`
	Equipe int     `json:"equipe"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Z      float64 `json:"z"`
}

// PointMort est l'endroit et l'instant ou la vie precedente s'est arretee.
type PointMort struct {
	Frame int     `json:"frame"`
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	Z     float64 `json:"z"`
}

// SpawnPiste est le PREMIER point d'une vie : la reapparition.
type SpawnPiste struct {
	Frame int     `json:"frame"`
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	Z     float64 `json:"z"`
	// PremiereVie marque le spawn de DEPART — celui de la premiere vie du joueur sur le
	// match, et le seul que la lecture « ou je spawne » regarde (decision produit du plan :
	// les reapparitions suivantes dependent de ou l'on vient de mourir, pas du placement
	// d'ouverture).
	//
	// C'est la premiere vie NOMMEE : une vie que le fil des morts n'a pas nommee n'a pas de
	// porteur, et les premieres vies d'un film sont parfois anterieures au debut reel du
	// match. La borne est donc « la plus precoce que le film sache attribuer », pas « la
	// premiere du film ».
	PremiereVie bool `json:"premiere_vie,omitempty"`

	// Reapparition : cette vie commence apres une MORT du joueur (et non au debut du match
	// ou d'une manche). Source dit comment on l'a decide (SourceFilm).
	Reapparition bool   `json:"reapparition,omitempty"`
	Source       string `json:"source,omitempty"`
	// DebutDeManche : la vie precedente a ete close par une suppression de corps (fin de
	// manche). Cette vie n'est PAS une reapparition et n'entre pas dans le modele, au meme
	// titre qu'une premiere vie.
	DebutDeManche bool `json:"debut_de_manche,omitempty"`

	// Mort et Instantane ne sont portes que par une reapparition dont l'equipe du joueur est
	// connue. Mort : la fin de la vie precedente. Instantane : les AUTRES joueurs vivants a la
	// frame de la reapparition, ceux apparus dans la meme seconde exclus (leur premier point est
	// un spawn, pas une position de jeu).
	Mort       *PointMort `json:"mort,omitempty"`
	Instantane []Vivant   `json:"instantane,omitempty"`
}
