package replay

// document_bomb_armings.go — L'ARMEMENT DE LA BOMBE : la forme que le compte à rebours
// d'Assaut prend dans l'artefact, et ce que la mesure a refusé d'y mettre.
//
// # D'où vient ce qui est publié
//
// L'anneau du marqueur d'objectif (`ti=12 i14`, `managed-navpoint-radial-progress`) est la
// JAUGE D'ARMEMENT — protocole du 2026-09-01 avec tirage nul, fixé avant la mesure
// (`filmdec/navpoint_ti12_plancher_test.go`) :
//
//	le DÉBUT d'une montée CONTIGUË de l'anneau  = un joueur commence à armer (le hold) ;
//	la FIN de la montée (quantum plein)          = la bombe est ARMÉE ;
//	fin + 4,93 s                                 = l'EXPLOSION (la mèche, constante moteur —
//	                                               13/13 Neutral Bomb CV 0,016, 4/4 Husky Raid
//	                                               CV 0,016, 0/1000 tirages nuls aussi bien).
//
// Chaque entrée porte les DEUX événements de la percée — `bomb_arming_start` (StartT/StartMs)
// et `bomb_armed` (T/TimeMs) — plus la mèche (`fuseMs`) : le client dessine le remplissage sur
// [startT, t] et le compte à rebours sur [t, t + fuseMs].
//
// # Ce qui n'est PAS publié, et pourquoi c'est la moitié du résultat
//
//	ONE BOMB. Le signal n'y tient PAS (CV 0,725, 87/1000 tirages nuls font aussi bien — mesure
//	  du 2026-09-01). La variante est écartée par DEUX gardes indépendantes : le nom de variante
//	  chez l'appelant (`replaybuild`), et la CONFRONTATION LOCALE aux explosions du même film
//	  (cf. bomb_armings.go) — pas d'événement publié plutôt qu'un événement faux.
//	QUI ARME. Le navpoint est un marqueur d'écran, pas un acteur : aucun xuid n'est publié.
//	  Le détonateur, lui, est déjà nommé par le statborg (`bomb_detonations` dans `objectives`).
//	LES MONTÉES SOUS LE PLEIN : jamais publiées, comptées dans la couverture. Le « quantum
//	  plein » est MESURÉ à q=254 (diagnostic du gate, 2026-09-01) : les 7 montées confirmées
//	  par une explosion finissent toutes à 254 ; les montées plafonnées à 253 (~4,9 s,
//	  130 -> 253, après chaque explosion et à chaque apparition de bombe) sont l'ANIMATION DE
//	  RECHARGE du marqueur — le même anneau, un autre sens — et les fins plus basses des holds
//	  relâchés. Sans ce filtre, le film 35b75a31 publiait 19 « armements » pour 3 explosions.
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
	// FuseMS est la mèche moteur : l'explosion attendue à TimeMS + FuseMS. Publiée avec
	// l'événement plutôt que codée côté client — si une variante future porte une autre
	// mèche, l'artefact la dira sans changer de schéma.
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
	// Rises est le nombre de montées contiguës détectées (avant tout filtre).
	Rises int `json:"rises"`
	// BelowFull compte les montées écartées parce que leur dernier échantillon n'atteint pas
	// le QUANTUM PLEIN (q=254) : les holds relâchés ET l'animation de recharge du marqueur
	// (plafonnée à 253 — mesure du gate, 12/12 hors mèche). Une montée sous le plein n'a pas
	// armé — la publier serait un compte à rebours faux.
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
	// `objectives`, et celles qu'un armement précède dans la fenêtre de la mèche. C'est la
	// CONFRONTATION LOCALE : une explosion sans armement à ~4,93 s contredit la lecture.
	Detonations        int `json:"detonations"`
	DetonationsCovered int `json:"detonationsCovered"`
	// Suppressed dit que la confrontation locale a ÉCHOUÉ et que le calque entier a été
	// retenu à la source — aucun événement publié plutôt que des événements faux.
	Suppressed bool `json:"suppressed,omitempty"`
}
