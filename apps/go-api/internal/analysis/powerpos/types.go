package powerpos

// KillSample est une elimination, les DEUX extremites en metres monde.
//
// Le match est porte par l'echantillon (et non par l'appel) pour la meme raison que dans
// `tactical.PositionSample` : le plancher de rarete se compte en matchs DISTINCTS par
// cellule.
//
// Z EST PRESENT, contrairement a la lecture tactique qui est une vue du dessus pure : le
// DENIVELE d'un engagement est l'un des signaux (une position de force tient la hauteur).
// Il ne sert PAS a adresser la cellule — l'adressage reste en XY.
// RangTueur est le rang competitif du TUEUR AU MOMENT DU MATCH (CSR de
// `match_csrs_latest`), ou nil quand il est inconnu — ce qui est le cas de TOUS les
// joueurs d'un match non classe, la table ne couvrant que les playlists classees. Un rang
// inconnu n'est pas un rang bas : il pese neutre (cf. ponderation.go).
type KillSample struct {
	MatchID                   string
	KillerX, KillerY, KillerZ float64
	VictimX, VictimY, VictimZ float64
	RangTueur                 *float64
}

// PresenceSample est un SEGMENT d'occupation : un joueur d'une equipe se tient en (X, Y)
// pendant DurMS millisecondes.
//
// POURQUOI UNE DUREE ET PAS UN POINT. Les points d'une piste de rejeu ne sont pas
// equidistants dans le temps (mesure sur les artefacts : t = 0, 13, 26, 49, 51, 60...).
// Compter les points ferait peser une zone ou le decodeur a echantillonne serre autant
// qu'une zone reellement tenue. C'est l'appelant qui convertit l'ecart entre deux points
// en millisecondes (`frameIntervalMs` du document de rejeu).
//
// `Gagnant` est l'issue du MATCH pour l'equipe du joueur, pas pour le joueur : une position
// de force se juge sur qui remporte la partie, pas sur qui y meurt le moins.
type PresenceSample struct {
	MatchID string
	Team    int
	Gagnant bool
	X, Y    float64
	DurMS   float64
}

// Cellule est une cellule ALIMENTEE, avec ses accumulateurs. Une cellule jamais atteinte
// n'existe pas (meme decision produit que `tactical.Raster` : la peindre a zero
// inventerait une mesure).
type Cellule struct {
	// Col, Lig : l'adresse entiere sur la grille, ancree sur l'ORIGINE DU MONDE.
	Col int `json:"col"`
	Lig int `json:"lig"`
	// CentreX, CentreY : le centre de la cellule en metres monde.
	CentreX float64 `json:"centre_x"`
	CentreY float64 `json:"centre_y"`

	// KillsDepuis : eliminations dont le tueur etait ici.
	KillsDepuis int `json:"kills_depuis"`
	// MortsDedans : eliminations dont la victime etait ici.
	MortsDedans int `json:"morts_dedans"`
	// PorteeMedianeM : mediane des distances 3D tueur-victime des kills partis d'ici.
	// Zero quand KillsDepuis vaut zero.
	PorteeMedianeM float64 `json:"portee_mediane_m"`
	// DeniveleMedianM : mediane de (killer_z - victim_z) des kills partis d'ici. Positif =
	// on tire vers le bas.
	DeniveleMedianM float64 `json:"denivele_median_m"`

	// MatchsKills : matchs distincts ayant alimente la cellule par un kill (depuis ou dedans).
	MatchsKills int `json:"matchs_kills"`
	// MatchsPresence : matchs distincts ayant alimente la cellule par une presence.
	MatchsPresence int `json:"matchs_presence"`
	// MatchsDistincts : l'union des deux. Sert au rapport, jamais de plancher a lui seul
	// (cf. doc.go : les deux sources n'ont pas la meme densite).
	MatchsDistincts int `json:"matchs_distincts"`

	// OccupationGagnantsMS / OccupationPerdantsMS : temps passe ici par l'equipe qui a
	// gagne le match et par celle qui a perdu, en millisecondes cumulees sur le corpus.
	OccupationGagnantsMS float64 `json:"occupation_gagnants_ms"`
	OccupationPerdantsMS float64 `json:"occupation_perdants_ms"`

	// DirSortante : directions d'ici VERS LA VICTIME, pour les kills partis d'ici. Sa
	// dispersion mesure la COUVERTURE : combien de voies distinctes le lieu tient.
	DirSortante SommeAngulaire `json:"dir_sortante"`
	// DirEntrante : directions d'ici VERS LE TUEUR, pour les morts subies ici. Sa
	// dispersion mesure l'EXPOSITION : par combien d'angles le lieu se prend.
	DirEntrante SommeAngulaire `json:"dir_entrante"`

	// KillsPonderes / MortsPonderees : les memes eliminations, comptees au POIDS DU RANG
	// DU TUEUR (cf. ponderation.go). Avec une ponderation neutre elles valent exactement
	// KillsDepuis et MortsDedans.
	KillsPonderes  float64 `json:"kills_ponderes"`
	MortsPonderees float64 `json:"morts_ponderees"`

	// KillsRangConnu / MortsRangConnu : parmi ces eliminations, celles dont le rang du
	// tueur est connu. C'est la MESURE DE COUVERTURE du rang, et elle se lit par cellule
	// parce que la couverture n'est pas uniforme (une carte jouee surtout en classe la
	// voit partout, une carte de playlist sociale nulle part).
	KillsRangConnu int `json:"kills_rang_connu"`
	MortsRangConnu int `json:"morts_rang_connu"`

	// KillsTueurFort / MortsTueurFort : celles dont le tueur a un rang connu au moins egal
	// a la mediane du corpus. Variante « au-dessus du rang median », mesuree a part.
	KillsTueurFort int `json:"kills_tueur_fort"`
	MortsTueurFort int `json:"morts_tueur_fort"`
}

// TotalEngagements rend le nombre d'eliminations qui touchent la cellule d'un cote ou de
// l'autre. C'est le denominateur du rapport de duel, et la taille d'echantillon sur
// laquelle ce rapport est retreci (cf. score.go).
func (c Cellule) TotalEngagements() int { return c.KillsDepuis + c.MortsDedans }

// TotalOccupationMS rend le temps d'occupation toutes issues confondues.
func (c Cellule) TotalOccupationMS() float64 {
	return c.OccupationGagnantsMS + c.OccupationPerdantsMS
}
