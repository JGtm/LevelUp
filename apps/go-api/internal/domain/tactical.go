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
