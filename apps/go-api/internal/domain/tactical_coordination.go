package domain

// tactical_coordination.go — LA SECTION « COORDINATION D'EQUIPE » DE LA LECTURE TACTIQUE.
//
// # CE QU'ELLE AJOUTE, ET CE QU'ELLE N'AJOUTE PAS
//
// Le TAUX d'isolement et le taux d'ECHANGE vivent deja sur `TacticalRaster`
// (`Isolement`, `Echange`) : ils ne sont pas redits ici. Ce bloc porte la seule chose que
// le taux ne dit pas — LA FORME de la distance a l'equipier au moment de mes morts : sa
// mediane, et sa distribution par intervalles.
//
// # LE BINNING EST SERVEUR (ADR 0010)
//
// Les intervalles sont decides ici et servis comptes : le web dessine ce qu'il recoit, il
// ne re-bucket rien. Deux clients ne peuvent donc pas rendre deux histogrammes differents
// de la meme mesure.
//
// # UNE DISTANCE ABSENTE N'EST PAS UNE GRANDE DISTANCE
//
// `MortAExaminer.PlusProcheM` est nil quand AUCUN coequipier n'etait VISIBLE a cet
// instant : il n'y a alors aucune distance a mesurer. Ces morts comptent dans le taux
// d'isolement (elles sont isolees par construction) mais N'ENTRENT PAS dans la
// distribution — les glisser dans un intervalle « 50+ » inventerait une mesure. Elles
// sortent donc a part, dans `MortsSansDistance`.

// TacticalBinDistance est UN intervalle de l'histogramme des distances.
type TacticalBinDistance struct {
	// MinM est la borne basse de l'intervalle, en metres (incluse).
	MinM float64 `json:"min_m"`

	// MaxM est la borne haute, en metres (exclue). nil pour le DERNIER intervalle, qui
	// est ouvert — « 50+ » n'a pas de borne haute, et lui en inventer une ferait croire
	// que les morts au-dela ont ete ecartees.
	MaxM *float64 `json:"max_m,omitempty"`

	// N est le nombre de morts tombant dans l'intervalle.
	N int `json:"n"`
}

// TacticalCoordination est la lecture de coordination servie avec CHAQUE raster.
type TacticalCoordination struct {
	// DistanceMedianeM est la mediane des distances MESUREES, en metres. nil quand
	// aucune mort n'a de distance mesuree — jamais un zero, qui se lirait comme « colle
	// a son equipier ».
	DistanceMedianeM *float64 `json:"distance_mediane_m,omitempty"`

	// Distribution est l'histogramme, un element par intervalle, dans l'ordre croissant.
	// Toujours servi en entier (intervalles vides compris) : un histogramme dont les
	// colonnes vides disparaissent change de forme d'un filtre a l'autre.
	Distribution []TacticalBinDistance `json:"distribution_distances"`

	// NDistances est le nombre de morts ayant une distance mesuree — le denominateur de
	// la mediane et la somme des intervalles.
	NDistances int `json:"n_distances"`

	// MortsSansDistance : les morts EXAMINEES sans coequipier visible, donc sans distance.
	MortsSansDistance int `json:"morts_sans_distance"`

	// RayonsM sont les portees de radar DISTINCTES des matchs lus (regulation.toml :
	// 18 m en Arene, 24 m en BTB), triees croissant.
	//
	// PLUSIEURS VALEURS QUAND LE FILTRE MELANGE DES FORMATS, ET JAMAIS UNE MOYENNE : une
	// moyenne de deux regles du jeu n'est la regle d'aucun match. Le web affiche la ou les
	// valeurs, et pose autant de seuils sur l'histogramme.
	RayonsM []float64 `json:"rayons_m"`

	// MatchsMesures est l'univers de CETTE mesure : les matchs dont la variante a une
	// portee connue. Publie parce que la note de lecture le nomme (« lecture sur N matchs
	// mesures sur M »).
	MatchsMesures int `json:"matchs_mesures"`

	// FenetreEchangeSecondes est la fenetre pendant laquelle une mort peut encore etre
	// vengee, en secondes (coordination.FenetreEchangeMs). Publiee parce que la note de
	// la section la nomme — jamais recopiee cote web, ou elle divergerait du calcul.
	FenetreEchangeSecondes int `json:"fenetre_echange_secondes"`
}

// TacticalBornesDistanceM sont les bornes des intervalles de l'histogramme, en metres.
// Le dernier intervalle est OUVERT au-dela de la derniere borne (« 50+ »).
//
// CES BORNES SONT CELLES DE LA MAQUETTE 034b1915 (0-10, 10-20, 20-30, 30-40, 40-50, 50+) :
// des dizaines de metres, lisibles, et assez fines pour que les deux portees de radar du
// titre (18 m et 24 m) tombent a l'interieur d'un intervalle plutot que sur un bord.
var TacticalBornesDistanceM = []float64{0, 10, 20, 30, 40, 50}

// TaCoordDistances est ce que `coordination.Distances` rend : la part MESUREE de la
// section, sans les denominateurs que seule la lecture connait (rayons, matchs).
type TaCoordDistances struct {
	Mediane           *float64
	Distribution      []TacticalBinDistance
	N                 int
	MortsSansDistance int
}
