package domain

// Types de l'onglet Tactique — lectures de PLACEMENT agregees par carte (ou je meurs, ou
// je tue, ou je gagne). Structs purs : aucune I/O, aucun SQL, aucune dependance au
// document de rejeu.
//
// FRONTIERE VOULUE : `internal/analysis/tactical` ne connait QUE ces types. Il n'importe
// ni `analysis/replay` (le document de rejeu) ni `platform/duckdb` — c'est l'appelant qui
// PROJETTE ce qu'il a (positions de `kill_positions`, points des pistes d'un artefact)
// vers `PositionSample`. Le rasterisage reste donc consommable sans artefact.

// PositionSample est l'entree minimale du rasterisage : un point du monde, en METRES, et
// le match qui l'a produit.
//
// Le match est porte par le point (et non par l'appel) parce que le plancher de rarete se
// compte en MATCHS DISTINCTS par cellule : sans l'identifiant, un joueur immobile gonfle
// une cellule sans rien prouver de plus (mesure de cmd/mappos-build, 2026-08-30).
//
// Z est volontairement absent : toutes les lectures de l'onglet sont des vues du dessus.
type PositionSample struct {
	MatchID string
	X, Y    float64
}

// BornesMonde est le rectangle englobant d'une lecture, en metres monde. `Valide` est faux
// tant qu'aucun point n'a ete vu : un rectangle vide n'est pas un rectangle a l'origine.
type BornesMonde struct {
	MinX, MinY float64
	MaxX, MaxY float64
	Valide     bool
}

// CelluleTactique est une cellule ALIMENTEE d'une lecture agregee. Une cellule jamais
// atteinte n'existe pas : elle n'apparait dans aucune liste (decision produit 2026-09-05,
// « cellule jamais atteinte = VIDE, jamais peinte en froid »).
type CelluleTactique struct {
	// Col, Lig : l'adresse entiere de la cellule sur la grille, ancree sur l'ORIGINE DU
	// MONDE (et non sur les bornes de la lecture) — deux lectures filtrees differemment
	// nomment donc la meme cellule pareil, et deux rasters de matchs differents se
	// somment sans re-projection.
	Col, Lig int

	// CentreX, CentreY : le centre de la cellule en metres monde, pour le peintre.
	CentreX, CentreY float64

	// Valeur est la valeur PAR MATCH (decision produit : jamais un cumul brut, qui ne se
	// compare pas d'un filtre a l'autre). Lecture simple : occurrences / nombre de matchs
	// retenus. Lecture signee : taux du cote victoire moins taux du cote defaite.
	Valeur float64

	// Brut est le cumul non normalise qui a produit `Valeur` — servi AVEC elle, jamais a
	// sa place (doctrine « jamais un taux seul »).
	Brut float64

	// Matchs est le nombre de matchs DISTINCTS ayant alimente la cellule ; MatchsVictoire
	// et MatchsDefaite le detaillent par cote (non nuls seulement en lecture signee).
	Matchs         int
	MatchsVictoire int
	MatchsDefaite  int
}

// EchelleTactique porte les reperes de coloration d'une lecture. Les quantiles sont
// calcules sur les cellules ALIMENTEES uniquement : inclure des zeros implicites
// ecraserait toute la dynamique vers le bas.
type EchelleTactique struct {
	// P50, P95 : les quantiles des valeurs des cellules. En lecture signee, ils portent
	// sur la VALEUR ABSOLUE — un quantile sur le signe n'a pas de sens (il depend de la
	// proportion de cellules favorables, pas de l'intensite).
	P50, P95 float64

	// Borne est le haut de l'echelle. En lecture signee, l'echelle est SYMETRIQUE et va
	// de -Borne a +Borne : sans cela, un cote parait plus intense que l'autre a valeur
	// egale.
	Borne float64

	// Symetrique dit laquelle des deux lectures ci-dessus s'applique.
	Symetrique bool

	// NCellules est le nombre de cellules ayant servi au calcul.
	NCellules int
}

// ---------------------------------------------------------------------------
// Lecture tactique — ce qu'un lecteur de base rend, et ce qu'une page en publie.
//
// Les types ci-dessous vivent ICI et pas dans `internal/analysis/tactical` :
// ils traversent la frontiere port -> service -> handler, et un DTO de reponse
// est un type de `domain`, jamais un type d'`analysis` (un algo qui exporte sa
// forme de sortie fige son appelant sur son implementation).
// ---------------------------------------------------------------------------

// TacticalQuery est la demande adressee au lecteur tactique.
//
// Le FILTRE est un `MatchFilterSpec` — le vocabulaire de l'Explorateur, et le seul
// du depot. Aucun axe nouveau n'est invente ici : une page qui filtrerait
// autrement que l'Explorateur donnerait deux comptes de matchs differents pour la
// meme question.
type TacticalQuery struct {
	// PlayerXUID est le joueur dont on lit les matchs. L'univers est TOUJOURS le
	// sien : la portee « tout le monde » n'existe pas en V1.
	PlayerXUID string

	// MapID restreint a une carte. Vide = toutes les cartes (ecran d'entree).
	MapID string

	// Filtre : nil = aucun filtre (tout l'historique du joueur).
	Filtre *MatchFilterSpec
}

// TacticalMatch est un match RETENU par le filtre : l'unite de l'univers.
type TacticalMatch struct {
	MatchID string

	// Outcome porte OutcomeWin / OutcomeLoss / OutcomeDraw / OutcomeDNF, ou
	// OutcomeUnknown quand le substrat ne le sait pas. Un resultat inconnu compte
	// au denominateur « par match » et dans aucun des deux cotes de la lecture
	// signee (cf. analysis/tactical.RasteriseAvecResultats).
	Outcome int
}

// TacticalUnivers est L'ENSEMBLE DES MATCHS RETENUS par le filtre, avec la
// composition des equipes de chacun.
//
// POURQUOI IL VOYAGE AVEC LES POINTS ET N'EN EST JAMAIS DEDUIT : un match retenu
// peut n'avoir AUCUN point sur la lecture courante (aucune mort, aucun kill), et
// c'est un zero legitime qui doit compter au denominateur. Le deduire des points
// l'effacerait — c'est le defaut corrige en phase 1 (12 victoires dont 2 muettes
// lues +0,10 au lieu de 0,00).
type TacticalUnivers struct {
	Matchs []TacticalMatch

	// Equipes : matchID -> xuid -> numero d'equipe. La composition change d'un
	// match a l'autre ; une table globale melangerait deux compositions au premier
	// joueur ayant change de camp.
	Equipes EquipesParMatch
}

// TacticalKillPosition est UNE mort mesuree : la position du tueur et celle de la
// victime, en metres monde, avec les deux identites.
//
// Z est absent : toutes les lectures de l'onglet sont des vues du dessus. Une
// ligne n'existe que si les DEUX positions sont connues — une position partielle
// n'est jamais approchee (meme prudence que KillDistanceRepo).
type TacticalKillPosition struct {
	MatchID string

	// KillerXUID est vide quand personne ne revendique la mort (chute, hors-limites,
	// environnement) ou quand le tueur est un bot. VictimXUID est vide pour une
	// victime bot.
	KillerXUID string
	VictimXUID string

	KillerX, KillerY float64
	VictimX, VictimY float64
}

// TacticalPositions : l'univers ET les positions mesurees de ses matchs.
type TacticalPositions struct {
	Univers TacticalUnivers
	Points  []TacticalKillPosition
}

// TacticalKillEvents : l'univers ET le journal des morts de ses matchs, sous la
// forme que `analysis/coordination` consomme.
type TacticalKillEvents struct {
	Univers TacticalUnivers
	Events  []KillEvent
}

// TacticalMapRow est une carte JOUEE, telle que le lecteur la rend : le compte de
// matchs et sa decomposition en victoires / defaites. Le plancher de lisibilite
// est pose par le service, pas par le lecteur — une regle produit n'appartient pas
// a une requete SQL.
type TacticalMapRow struct {
	MapID     string
	MapName   string
	MapNameFR string

	Matchs    int
	Victoires int
	Defaites  int
}

// ---------------------------------------------------------------------------
// Ce que la page publie
// ---------------------------------------------------------------------------

// TacticalMapCard est une carte de l'ecran d'entree : la ligne du lecteur, plus
// le verdict de lisibilite.
type TacticalMapCard struct {
	MapID     string `json:"map_id"`
	MapName   string `json:"map_name"`
	MapNameFR string `json:"map_name_fr"`

	Matchs    int `json:"matchs"`
	Victoires int `json:"victoires"`
	Defaites  int `json:"defaites"`

	// SousPlancher : la carte compte moins de matchs que le plancher par carte.
	// Elle reste affichee (le joueur doit voir qu'il y a joue) mais desaturee et
	// non ouvrable — une lecture de placement sur trois matchs est du bruit.
	SousPlancher bool `json:"sous_plancher"`
}

// TacticalMapsPage est la reponse de l'ecran d'entree.
type TacticalMapsPage struct {
	Cartes []TacticalMapCard `json:"cartes"`

	// PlancherMatchs est le seuil qui a decide de `SousPlancher`. Publie parce que
	// l'ecran doit pouvoir le NOMMER a l'utilisateur, pas le recopier.
	PlancherMatchs int `json:"plancher_matchs"`
}

// TacticalRaster est la reponse d'une lecture de placement sur une carte.
type TacticalRaster struct {
	MapID    string `json:"map_id"`
	Question string `json:"question"`
	Qui      string `json:"qui"`

	// MatchsRetenus est la taille de l'univers — le denominateur de la valeur
	// « par match » de chaque cellule. Publie AVEC les cellules : une intensite
	// sans son denominateur ne se compare pas d'un filtre a l'autre.
	MatchsRetenus int `json:"matchs_retenus"`

	// PasM est le pas de la grille en metres, et Bornes le rectangle englobant les
	// cellules LISIBLES — le cadre que le peintre doit couvrir.
	PasM   float64     `json:"pas_m"`
	Bornes BornesMonde `json:"bornes"`

	Cellules []CelluleTactique `json:"cellules"`
	Echelle  EchelleTactique   `json:"echelle"`

	// PointsIgnores : les positions ecartees faute de coordonnees finies. Publie
	// plutot qu'avale — un decodage qui derape se voit ici.
	PointsIgnores int `json:"points_ignores"`

	// Echange est le taux de morts vengees de mon equipe SUR CETTE CARTE. nil quand
	// le titre ne sait pas lire la source des morts (capability `film.kill_source`
	// absente) : la lecture de placement reste servie, le KPI est simplement
	// silencieux — jamais un zero, qui se lirait comme une contre-performance.
	Echange *Couverture `json:"echange,omitempty"`
}
