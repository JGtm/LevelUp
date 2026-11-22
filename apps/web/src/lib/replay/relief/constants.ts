/**
 * relief/constants — LES SEUILS ET LES DRAPEAUX DU RELIEF, en un seul endroit.
 *
 * Les seuils de la coupe sont côté web parce qu'ils règlent une VUE, pas une donnée : la maquette
 * des commandes peut les bouger sans recuire le parc de cartes (conception §4.1). Les seuils de la
 * pile des sols (fusion, tolérance) voyagent par le JSON ; les valeurs ci-dessous ne servent que de
 * repli quand le fichier ne les porte pas.
 */

/** Drapeau (canal B) : la hauteur est connue ET le pixel est dans la zone gardée. */
export const FLAG_ON_MAP = 1
/** Drapeau (canal B) : la hauteur vient du maillage de navigation et non de la géométrie. */
export const FLAG_MESH_FLOOR = 2
/** Les bits que le fichier peut porter ; tout autre bit de B refuse l'image. */
export const FILE_FLAG_MASK = FLAG_ON_MAP | FLAG_MESH_FLOOR
/** Bit INTERNE (absent du fichier) : la hauteur est connue (A = 255). */
export const FLAG_KNOWN = 4

/** Demi-largeur de la bande d'un étage, en unités. */
export const CUT_BAND_U = 0.7
/** Largeur du fondu au-delà de la bande, en unités. */
export const CUT_FADE_U = 0.25
/** Opacité de ce qui est sous la coupe (dessous pâle). */
export const CUT_ALPHA_BELOW = 0.28
/** Opacité de ce qui est au-dessus de la coupe (presque effacé). */
export const CUT_ALPHA_ABOVE = 0.1
/** Rayon de « proche d'un sol de la coupe », en unités. */
export const CUT_NEAR_U = 0.25
/** Un dessus plus haut qu'un sol de cet écart est le toit de ce sol, en unités. */
export const CUT_ROOF_U = 1.0
/** Poids au-delà duquel un sol compte dans la coupe. */
export const CUT_MIN_WEIGHT = 0.5

/** Fusion des sols empilés, repli quand le JSON ne la porte pas (unités). */
export const DEFAULT_FLOOR_MERGE_U = 0.4
/** Ecart toléré entre un dessus et son sol, repli quand le JSON ne la porte pas (unités). */
export const DEFAULT_FLOOR_TOLERANCE_U = 0.3
/** Nombre de sols qu'un pixel porte au plus ; au-delà la pile est tronquée. */
export const STACK_LAYERS = 4
/** Marge de hauteur sous `z0` que la pile sait coder (unités) : un sol peut passer sous le dessus le plus bas. */
export const STACK_HEADROOM_BELOW_U = 32

/** Longueur maximale d'un tronçon de trait testé contre le masque de la coupe, en unités. */
export const LINE_PIECE_U = 0.5
/** Distance de part et d'autre d'une rupture où le masque est aussi interrogé, en pixels. */
export const LINE_SIDE_PX = 1.5

/** Pas du cache des coupes (unités) : une hauteur de coupe est arrondie à ce pas. */
export const CUT_CACHE_STEP_U = 0.25

/** Un sol compte sous les pieds jusqu'à cette hauteur au-dessus d'eux (unités). */
export const FOLLOW_REACH_U = 0.35
/** Ecart entre le sol porteur et la coupe au-delà duquel la coupe suit (unités). */
export const FOLLOW_THRESHOLD_U = 0.5
/** Durée d'écart ininterrompu avant que la coupe suive (temps du rejeu, ms). */
export const FOLLOW_DELAY_MS = 300
/** Un étage détecté attire la coupe jusqu'à cet écart (unités). */
export const FOLLOW_SNAP_REACH_U = 0.7
/** Ecart entre deux instants lus au-delà duquel c'est un saut de lecture, pas de la lecture (ms). */
export const FOLLOW_SEEK_GAP_MS = 1000
