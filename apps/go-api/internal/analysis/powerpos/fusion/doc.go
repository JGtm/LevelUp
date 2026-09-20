// Package fusion combine, CELLULE PAR CELLULE, les deux voies de derivation des positions
// de force : la GEOMETRIE (ce que la carte permet — `powerpos/geo`, un score par noeud du
// sol derive) et l'EMPIRIQUE (ce que les matchs montrent — `powerpos`, un score par cellule
// scorable). Item 2bis.D du plan `.ai/PLAN_POSITIONS_DE_FORCE_2026-09-20.md`, decision D9 :
// trois angles, un seul score final ; l'oracle reste le juge, jamais une entree du score.
//
// PUR : aucune I/O. Les deux entrees sont deja des scores ; ce paquet ne recalcule ni l'un
// ni l'autre (`ReglageV2` et `ReglageGeoV1` sont FIGES, D12 : la fusion est un reglage NEUF
// au-dessus d'eux, `ReglageFusionV1`).
//
// # LA MEME CELLULE DES DEUX COTES
//
// Les deux voies adressent la grille tactique (`tactical.Grille`, 0,5 m, ancree sur
// l'origine du monde) : une cellule geometrique et une cellule empirique de la meme carte
// sont la meme cellule. Verifie sur pieces le 2026-09-20 (CSV de Recharge, les deux voies :
// centre = (col + 0,5) x 0,5 sur toutes les lignes, ecart nul) — aucun decalage d'une
// demi-cellule a corriger. La geometrie porte PLUSIEURS noeuds par cellule (un par etage :
// 1 a 2 sur Recharge) ; l'empirique n'a pas de Z. La cellule geometrique prend le MEILLEUR
// de ses etages (cf. AgregeCellules) : c'est l'etage qu'on tient, et c'est ce que l'oracle
// nomme (« le haut de la tour », pas la tour).
//
// # LA FORME DU SCORE
//
//	score = PoidsGeo x geo_norm + PoidsEmp x emp_norm
//
// avec geo_norm et emp_norm ramenes dans [0, 1] PAR CARTE par le min-max robuste p5..p95 du
// depot (`geo.Normalise`) : les deux scores bruts vivent dans des bandes differentes (0,50 a
// 0,65 pour l'empirique v2, 0,30 a 0,63 pour la geometrie) et ne se sommeraient pas.
//
// UNE ABSENCE EST EXPLICITE, JAMAIS UN ZERO PAR DEFAUT. Une cellule du sol derive sans
// engagement mesure (non scorable : moins de 3 matchs ou de 40 engagements dans son disque)
// recoit `EmpAbsent` comme emp_norm ; une cellule scorable hors du sol derive (le sol est
// DERIVE du rendu, il a des trous — GEOMETRIE_2026-09-20.md §5) recoit `GeoAbsent` comme
// geo_norm. Les deux valeurs font partie du reglage et se choisissent au calibrage : un
// zero dirait « mauvais lieu » la ou l'on ne sait rien.
//
// # LA SELECTION EST CELLE DE LA V2, TELLE QUELLE
//
// Hysteresis amorce / croissance, fermeture morphologique, 8-connexite apres fermeture,
// taille minimale, plafond (`powerpos.Selectionne` avec la partie « selection » du
// reglage). Pas la selection geometrique (maxima locaux sur le GRAPHE de deplacement) : le
// graphe n'existe pas par cellule, et une fusion par cellule se selectionne par cellule.
// Les quantiles de l'hysteresis sont, eux, propres a la fusion : le score fusionne n'a pas
// la distribution du score v2.
package fusion
