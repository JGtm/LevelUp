package types

// grammar_tir_continu.go — LE TIR CONTINU, TEL QUE LA VUE DE CONTROLE LE PORTE (lot M4b de la
// campagne « retours rejeu », 2026-09-24).
//
// # CE QUE LE FILM ECRIT, ET CE QU IL N ECRIT PAS (sonde P1-S3)
//
// Une arme dont le barillet est de type de prediction 1 ou 3 n emet pas un record de tir par coup :
// l ecrivain de la vue de controle pose, dans l entree du joueur, le bit de la GACHETTE TENUE
// (`m0` / `m2`, type 1) ou du BARILLET EN TIR (`m4` / `m5`, type 3). Theater rejoue ce bit comme
// une gachette tenue et fait tirer le barillet a SA CADENCE (celle du tag) : les instants des
// coups ne sont pas dans le film ; le DEBUT et la FIN de la rafale le sont, au tick.
//
// Une RAFALE est donc un intervalle lu : de la premiere entree lue qui tient la gachette a la
// premiere entree lue qui ne la tient plus. Un paquet dont la vue de controle n est pas lue est
// un TROU : de lui a la prochaine entree lue du joueur, l etat de la gachette est INCONNU. Le trou
// est porte par la rafale (`Holes`) et le rejeu se TAIT sur ce passage — jamais il ne suppose que
// la gachette est restee tenue ; si l entree lue apres le trou ne la tient plus, la rafale finit
// au debut du trou. Tout trou est COMPTE ([ContinuousFireStats]).

// Les causes des bornes d une rafale. Etiquettes STABLES, persistees dans les faits.
const (
	// ContinuousFireBoundPressed : la rafale commence sur une entree lue qui tient la gachette,
	// apres une entree lue qui ne la tenait pas (ou apres le debut du film).
	ContinuousFireBoundPressed = "pressed"
	// ContinuousFireBoundReleased : la rafale finit sur une entree lue qui ne tient plus la
	// gachette — comme Theater, qui simule tant que le bit est pose.
	ContinuousFireBoundReleased = "released"
	// ContinuousFireBoundHole : la borne est un TROU — un paquet dont la vue de controle n est pas
	// lue. En fin de rafale : l instant du premier paquet non lu (l entree lue apres le trou ne tient
	// plus la gachette : le lacher est DANS le trou) ; en debut : la premiere entree lue apres le trou.
	ContinuousFireBoundHole = "hole"
	// ContinuousFireBoundFilmEnd : la rafale tenait encore au dernier paquet du film.
	ContinuousFireBoundFilmEnd = "filmEnd"
)

// ContinuousFireBurst est UNE rafale de tir continu lue dans la vue de controle.
type ContinuousFireBurst struct {
	// FilmIndex est l index de controle de l entree : l index de joueur du film, c est-a-dire
	// la PLACE (le meme espace que le tireur d un record de tir).
	FilmIndex int
	// Hand est la main (0, 1) ; Barrel dit si le bit est un BARILLET de type 3 (`m4` / `m5`) plutot
	// qu une GACHETTE de type 1 (`m0` / `m2`) ; Input est l entree (0..2) ou le barillet (0..1).
	// L entree 0 de la main 0 est la gachette principale.
	Hand   int
	Barrel bool
	Input  int
	// Weapon est l index d arme de la main lu dans la premiere entree de la rafale
	// (`FUN_1406d00ec` : 0..3, -1 sentinelle, -2 absent).
	Weapon int
	// StartUS / EndUS bornent la rafale, horloge du film ; StartBound / EndBound disent ce qui les
	// a posees (cf. les etiquettes ci-dessus).
	StartUS, EndUS       uint64
	StartBound, EndBound string
	// Entries est le nombre d entrees lues qui tiennent ce bit pendant la rafale.
	Entries int
	// Holes sont les TROUS de la rafale : les passages ou la gachette, tenue avant et apres, n est
	// pas lue (du premier paquet non lu a la prochaine entree lue du joueur). Le rejeu s y TAIT.
	Holes []ContinuousFireHole
}

// ContinuousFireHole est un passage INCONNU d une rafale, horloge du film : [StartUS, EndUS).
type ContinuousFireHole struct {
	StartUS, EndUS uint64
}

// ContinuousFireStats dit ce que la lecture de la vue de controle a vu, et ce qu elle n a PAS pu
// lire. Sans ces denominateurs, « N rafales » ne se juge pas.
type ContinuousFireStats struct {
	// Scanned dit que la marche du frame-processeur a eu lieu.
	Scanned bool
	// Packets est le nombre de paquets delta marches (liste d evenements localisee ou non).
	Packets int
	// Reached est le nombre de paquets dont la vue B a clos sa liste (la vue C y a ete lue) ;
	// Closed ceux dont la vue C se lit jusqu a son terminateur ET ferme le paquet. SEULS les
	// paquets fermes rendent des entrees.
	Reached, Closed int
	// Holes est le nombre de paquets NON LUS (Packets - Closed) : liste d evenements non
	// localisee, vue B ouverte, ou vue C qui ne ferme pas. HoleRuns compte leurs suites maximales.
	Holes, HoleRuns int
	// Unlocated est la part des trous due a une liste d evenements non localisee ; OpenViewB celle
	// d une vue B qui n a pas clos ; les quatre suivants ventilent les vues C atteintes qui ne
	// ferment pas par leur cause (`grammar.ArretVueC`), NotClosing celles qui se lisent jusqu a
	// leur terminateur sans fermer le paquet (une fin de vue B fausse).
	Unlocated, OpenViewB                         int
	StopOverflow, StopKind, StopBlockBC, StopCap int
	NotClosing                                   int
	// Entries est le nombre d entrees de controle lues (vues fermees) ; WithAction celles qui
	// portent le bloc de 0x68 ; Firing celles qui tiennent au moins une gachette ou un barillet.
	Entries, WithAction, Firing int
	// Bursts est le nombre de rafales rendues ; BurstsWithHole celles qu un trou touche (une borne,
	// ou un passage interieur) ; InnerHoles le nombre de passages interieurs.
	Bursts, BurstsWithHole, InnerHoles int
	// HeldHoleMS est la duree cumulee des trous tombes PENDANT qu une gachette etait tenue (du
	// premier paquet non lu a la prochaine entree lue du joueur) : le temps de tir que le rejeu
	// TAIT, par joueur et par bit, faute de lecture.
	HeldHoleMS int64
}
