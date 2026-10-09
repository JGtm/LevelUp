package replay

// document_bomb_armings.go — L'ARMEMENT DE LA BOMBE : la forme que le compte à rebours
// d'Assaut prend dans l'artefact, et ce que la mesure a refusé d'y mettre.
//
// # D'où vient ce qui est publié
//
// L'anneau du marqueur d'objectif (`ti=12 i14`, `managed-navpoint-radial-progress`) est la
// JAUGE D'ARMEMENT — protocole du 2026-09-01 avec tirage nul, fixé avant la mesure
// (`filmdec/navpoint_ti12_plancher_test.go`), étendu le même jour par la lecture « mèche
// pausable » (`filmdec/navpoint_ti12_meche_test.go`) et portée en production le 2026-09-04 :
//
//	le DÉBUT d'un SEGMENT contigu de l'anneau  = un joueur commence à armer (le hold) ;
//	la FIN du segment, à son sommet plein      = la bombe est ARMÉE ;
//	fin + la MÈCHE, les PAUSES déduites        = l'EXPLOSION. Une tenue de désarmement
//	                                             SUSPEND la mèche ; le compte à rebours
//	                                             reprend où il en était.
//
// LA MÈCHE EST MESURÉE SUR LE FILM, pas supposée : médiane des délais corrigés, publiée avec
// chaque événement (`fuseMs`). Elle vaut 4,93 s en Neutral Bomb (13/13, CV 0,016), 5,1 s en
// Husky Raid (4/4) et 16,2 s en One Bomb (9/9 explosions portées, CV 0,017, 0/1000 tirages
// nuls) — trois valeurs, UNE règle, aucun branchement sur le nom de la variante.
//
// Chaque entrée porte les DEUX événements de la percée — `bomb_arming_start` (StartT/StartMs)
// et `bomb_armed` (T/TimeMs) — plus la mèche (`fuseMs`) : le client dessine le remplissage sur
// [startT, t] et le compte à rebours sur [t, t + fuseMs].
//
// # Ce qui n'est PAS publié, et pourquoi c'est la moitié du résultat
//
//	UN FILM QUI CONTREDIT LA LECTURE. La garde de NOM qui écartait One Bomb est LEVÉE le
//	  2026-09-04 (la lecture pausable l'explique) : il ne reste que la CONFRONTATION LOCALE
//	  aux explosions du même film (cf. bomb_armings.go), TOUT-OU-RIEN — une explosion sans
//	  armement dans la fenêtre de sens, ou des mèches qui se contredisent, et le calque entier
//	  est retenu. Pas d'événement publié plutôt qu'un événement faux.
//	QUI ARME. Le navpoint est un marqueur d'écran, pas un acteur : aucun xuid n'est publié.
//	  Le détonateur, lui, est déjà nommé par le statborg (`bomb_detonations` dans `objectives`).
//	LES SEGMENTS SOUS LE PLEIN : jamais publiés, comptés dans la couverture. Le « quantum
//	  plein » est MESURÉ à q=254 (diagnostic du gate, 2026-09-01) : les 7 montées confirmées
//	  par une explosion finissent toutes à 254, et l'inspection One Bomb du même jour mesure
//	  la même fin pleine (131 -> 254). Les segments plafonnés à 253 sont l'ANIMATION DE
//	  RECHARGE du marqueur — le même anneau, un autre sens — et les fins plus basses sont des
//	  holds relâchés. Sans ce filtre, le film 35b75a31 publiait 19 « armements » pour 3
//	  explosions.
type BombArming struct {
	// T est l'index de frame de l'instant ARMÉ (`bomb_armed`), sur le même axe que Point.T.
	T int `json:"t"`
	// TimeMS est l'instant armé exact sur l'horloge du film — même convention que
	// ObjectiveAction.TimeMS : la grille de frames perd de la précision (100 ms par défaut).
	TimeMS int `json:"timeMs"`
	// StartT est l'index de frame du début du hold d'armement (`bomb_arming_start`).
	StartT int `json:"startT"`
	// StartMS est l'instant exact du début du hold, sur l'horloge du film.
	StartMS int `json:"startMs"`
	// FuseMS est la mèche du FILM, MESURÉE sur ses explosions (médiane des délais corrigés
	// des pauses) : l'explosion attendue à TimeMS + FuseMS, tenue de désarmement mise à
	// part. Publiée avec l'événement plutôt que codée côté client — c'est ce qui permet à
	// One Bomb (16,2 s) et à Neutral Bomb (4,93 s) de sortir de la même chaîne. Sur un film
	// SANS explosion il n'y a rien à mesurer : la valeur est alors la référence DÉDUITE
	// (`BombFuseMS`), et `coverage.bombArmings.detonations == 0` dit ce cas.
	FuseMS int `json:"fuseMs"`
}

// BombArmingsCoverage dit ce que la lecture de l'anneau a rendu — et pourquoi le calque
// peut être vide alors que le film a été lu (même doctrine que les autres couvertures).
type BombArmingsCoverage struct {
	// Scanned dit que l'appelant a reconnu le match armable ET que le film a été balayé.
	// Faux = mode non armable (ou film illisible) : un calque vide ne dit alors rien.
	Scanned bool `json:"scanned"`
	// Reads est le nombre de lectures de l'anneau (toutes voies).
	Reads int `json:"reads"`
	// Rises est le nombre de SEGMENTS contigus de l'anneau détectés (avant tout filtre) — le
	// fait brut d'où la lecture part. Le nom de la clé JSON est antérieur au passage des
	// « montées » aux segments (2026-09-04) : il est gardé pour ne pas casser les artefacts
	// déjà cuits, la grandeur comptée est celle décrite ici.
	Rises int `json:"rises"`
	// BelowFull compte les segments qui ont la FORME d'un armement (ils finissent à leur
	// sommet) mais dont le sommet n'atteint pas le QUANTUM PLEIN (q=254) : les holds relâchés
	// ET l'animation de recharge du marqueur (plafonnée à 253 — mesure du gate, 12/12 hors
	// mèche). Un segment sous le plein n'a pas armé — le publier serait un compte à rebours
	// faux. Les tenues de DÉSARMEMENT n'y entrent pas : elles ne descendent pas d'un
	// armement manqué, elles servent à corriger la mèche.
	BelowFull int `json:"belowFull"`
	// Armed est le nombre d'armements retenus après le filtre du plein et la déduplication
	// des paires de navpoints.
	Armed int `json:"armed"`
	// PairMerged compte les montées fondues dans un armement déjà retenu (les navpoints vont
	// par paires, +12 d'écart de slot, un par camp — le même anneau répliqué deux fois).
	PairMerged int `json:"pairMerged"`
	// Published est le nombre d'armements posés sur la grille de frames.
	Published int `json:"published"`
	// OutOfWindow compte les armements hors de l'axe de temps du document.
	OutOfWindow int `json:"outOfWindow"`
	// Detonations / DetonationsCovered : les explosions du statborg visibles dans
	// `objectives`, et celles qu'un armement précède dans la FENÊTRE DE SENS, délai corrigé
	// des pauses. C'est la CONFRONTATION LOCALE. `Detonations == 0` dit aussi que la mèche
	// publiée est la référence DÉDUITE, faute d'explosion à mesurer.
	Detonations        int `json:"detonations"`
	DetonationsCovered int `json:"detonationsCovered"`
	// Suppressed dit que la confrontation locale a ÉCHOUÉ — explosion orpheline, ou mèches du
	// film qui se contredisent — et que le calque entier a été retenu à la source : aucun
	// événement publié plutôt que des événements faux.
	Suppressed bool `json:"suppressed,omitempty"`
}
