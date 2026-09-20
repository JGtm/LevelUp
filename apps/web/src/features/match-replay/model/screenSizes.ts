/**
 * screenSizes.ts — LE MODÈLE DE TAILLE DU REJEU 2D : l'échelle de la carte quand elle est
 * généreuse, un PLANCHER D'ÉCHELLE quand elle ne l'est pas — et les proportions entre familles
 * conservées dans les deux cas.
 *
 * ## La décision, et ce qu'elle remplace (utilisateur, 2026-09-19 puis 2026-09-20)
 *
 * LE DÉFAUT D'ORIGINE (constat du 2026-09-19 : « les véhicules sont trop petits »). Les
 * véhicules étaient dessinés à une taille CONSTANTE en pixels d'écran : un millimètre-monde
 * valait `VEHICLE_PX_PER_MM` pixels, POUR TOUTE CARTE. Un Ghost avait donc la même taille à
 * l'écran sur une carte de 54 m et sur une de 273 m — trop gros sur la petite, trop petit sur
 * la grande, et c'est la grande que l'utilisateur regardait.
 *
 * LE MODÈLE RETENU LE 2026-09-20, et c'est un modèle en ESPACE ÉCRAN, pas un modèle
 * « réaliste » : l'échelle appliquée aux longueurs monde est le MAXIMUM de deux échelles —
 * celle de la carte, et un PLANCHER qui vaut l'ancien cadrage grossi de 20 %.
 *
 *     échelle_px_par_mm = max( échelle_carte_px_par_mm , PX_PAR_MM_MINIMUM_ECRAN )
 *     taille_px         = plafond_doux( longueur_mm[famille] × échelle_px_par_mm )
 *
 * CE QUE CETTE FORME GARANTIT, ET QU'UN PLANCHER EN PIXELS NE GARANTISSAIT PAS. Le plancher
 * porte sur l'ÉCHELLE, pas sur la taille : toutes les familles le franchissent ENSEMBLE, donc
 * elles restent entre elles dans le rapport exact de leurs longueurs monde À TOUTE ÉCHELLE. Un
 * plancher exprimé en pixels (la première écriture de ce fichier, remplacée le jour même)
 * écrasait au contraire quinze familles sur dix-huit à la même taille sur une grande carte.
 *
 * TROIS PROPRIÉTÉS, chacune vérifiée par un test :
 *
 *   JAMAIS PLUS PETIT   sur TOUTE carte, aucune famille ne descend sous sa taille d'avant le
 *   QU'AVANT            2026-09-20 : l'échelle effective vaut au minimum l'ancienne × 1,2.
 *   GRANDE CARTE =      là où l'échelle de la carte est sous le plancher (toute carte de plus
 *   ANCIEN × 1,2        de 65,9 m de large, donc toutes les grandes), le rendu est EXACTEMENT
 *                       l'ancien grossi de 20 % — ce que l'utilisateur a demandé le 2026-09-19.
 *   PETITE CARTE =      là où l'échelle de la carte dépasse le plancher, c'est elle qui sert :
 *   LA PLUS GRANDE      les véhicules y sont plus gros encore, et à leur taille réelle.
 *
 * ## Les pixels sont LOGIQUES, jamais physiques
 *
 * Tout ce fichier est en pixels CSS. La densité du périphérique (`k`, `devicePixelRatio`) est
 * appliquée AU TRACÉ, par l'appelant, exactement comme pour les épaisseurs de trait des
 * marqueurs — un écran à `dpr = 2` rend donc la MÊME taille logique avec deux fois plus de
 * pixels physiques. L'échelle du cadrage (`scaleOf(view)`) est elle aussi en pixels CSS par
 * mètre : `view.width` est la largeur du CONTENEUR, pas celle de la toile physique.
 *
 * ## Les familles qui n'ont PAS de longueur monde mesurée
 *
 * Le modèle a besoin d'une longueur monde ; une famille qui n'en a pas mesurée garde la sienne,
 * là où elle vit, et ne passe pas par ici :
 *
 *   pion / bipède          `replayMarkers.CORE_RADIUS` — aucune longueur monde n'est publiée
 *                          pour un joueur, et le pion n'en est de toute façon pas une
 *                          représentation : c'est un MARQUEUR. Sa référence NE BOUGE PAS
 *                          (décision utilisateur du 2026-09-19), et c'est elle qui sert d'unité
 *                          à l'ancien cadrage, donc au plancher d'échelle ci-dessous.
 *   socle d'arme           `weaponPadsLayer.PAD_DOT_PX` / `PAD_ICON_H_PX` — un socle est un
 *                          point d'apparition, pas un objet ; le document n'en publie aucune
 *                          dimension.
 *   drapeau / objectif     `placementShapes` — glyphes de forme, sans dimension publiée.
 *   équipement             idem.
 *
 * LES DEUX GLYPHES DU CALQUE VÉHICULES, EUX, SUIVENT LE MÊME FACTEUR (décision du 2026-09-20 :
 * « même facteur ») : le losange d'un châssis non résolu et le pictogramme de tourelle n'ont pas
 * de longueur monde propre, mais on leur en donne une ÉQUIVALENTE — celle qui redonne leur
 * taille d'avant à l'ancienne échelle. Ils grossissent donc de 20 % au minimum, et suivent la
 * carte comme les châssis, sans jamais rétrécir par rapport à un véhicule voisin.
 */
import { CORE_RADIUS, PION_VISIBLE_DIAMETER_PX } from '../layers/replayMarkers'

/**
 * PION_SCREEN_PX — L'UNITÉ HISTORIQUE DU CADRAGE : le diamètre RÉELLEMENT VU d'un pion au
 * rez-de-chaussée, liseré compris. Le pion ne change pas de taille (décision du 2026-09-19) ;
 * il reste l'ancre par laquelle l'ancienne échelle, donc le plancher, est définie.
 */
export const PION_SCREEN_PX = PION_VISIBLE_DIAMETER_PX

/**
 * MONGOOSE_REFERENCE_LENGTH_MM — la longueur RÉELLE (nez-en-haut) du sprite Mongoose : 128 px
 * de sprite × 10 mm/px. Elle n'est plus une cible de cadrage, seulement l'ancre par laquelle
 * l'ANCIENNE échelle se reconstitue.
 */
const MONGOOSE_REFERENCE_LENGTH_MM = 1280

/**
 * MONGOOSE_TO_PION_RATIO_V1 — la cible de cadrage d'AVANT le 2026-09-20 : « un Mongoose fait
 * 1,75 pion de long », milieu de la fourchette 1,5-2 demandée à l'origine. Elle ne décide plus
 * de rien ; elle est conservée parce que le PLANCHER d'échelle se définit par rapport à elle, et
 * qu'un plancher dont on ne peut pas reconstituer l'origine n'est plus vérifiable.
 */
const MONGOOSE_TO_PION_RATIO_V1 = 1.75

/**
 * PX_PAR_MM_CADRAGE_V1 — L'ANCIENNE ÉCHELLE, en pixels CSS par millimètre-monde :
 * 0,01203125 px/mm. C'est l'ex-`VEHICLE_PX_PER_MM`, et elle s'appliquait quelle que soit la
 * carte.
 */
const PX_PAR_MM_CADRAGE_V1 = (PION_SCREEN_PX * MONGOOSE_TO_PION_RATIO_V1) / MONGOOSE_REFERENCE_LENGTH_MM

/**
 * VEHICLE_SCREEN_BUMP — LE +20 % DEMANDÉ PAR L'UTILISATEUR LE 2026-09-19, en UN seul littéral.
 * Il ne s'applique plus aux tailles une par une : il s'applique à l'ÉCHELLE PLANCHER, ce qui le
 * propage aux dix-huit familles ET aux deux glyphes du calque sans qu'aucune valeur ne soit
 * recopiée. Une révision ultérieure du cadrage change CE nombre, rien d'autre.
 */
export const VEHICLE_SCREEN_BUMP = 1.2

/**
 * PX_PAR_MM_MINIMUM_ECRAN — LE PLANCHER D'ÉCHELLE : 0,0144375 px/mm, c'est-à-dire l'ancienne
 * échelle grossie de 20 %. Sous ce plancher, l'échelle de la carte ne décide plus.
 *
 * L'ÉCHELLE DE BASCULE VAUT 14,44 px/m, soit une scène de 65,9 m de large dans un conteneur de
 * 1 000 px CSS (marge de 24). Mesuré sur les documents cuits : Flood Gulch (272,8 m, 3,49 px/m)
 * est très en dessous — le plancher y sert, et le rendu y est exactement l'ancien +20 % ;
 * Snowbound (54,2 m, 17,56 px/m) est au-dessus — l'échelle de la carte y sert, et les véhicules
 * y sont plus gros que l'ancien +20 %.
 */
export const PX_PAR_MM_MINIMUM_ECRAN = PX_PAR_MM_CADRAGE_V1 * VEHICLE_SCREEN_BUMP

/**
 * VEHICLE_SOFT_CEIL_PX — le plafond DOUX, INCHANGÉ (7 pions, 61,60 px) : au-delà, la croissance
 * ralentit en racine carrée au lieu de s'arrêter net — un véhicule très long reste visiblement
 * plus grand qu'un plus petit, mais cesse de dominer l'écran.
 *
 * IL NE CASSE PAS L'INVARIANT « jamais plus petit qu'avant », et c'est vérifiable : la
 * compression est MONOTONE croissante, donc une entrée plus grande sort toujours plus grande.
 * Mesure sur Flood Gulch : le Pélican passe de 70,17 à 71,62 px, le Phantom de 69,87 à 71,32.
 */
export const VEHICLE_SOFT_CEIL_PX = 7 * PION_SCREEN_PX

/**
 * VEHICLE_UNKNOWN_HALF_MM — la LONGUEUR MONDE ÉQUIVALENTE de la demi-diagonale du losange d'un
 * châssis NON RÉSOLU : celle qui, à l'ANCIENNE échelle, redonne exactement ses 3,40 px
 * historiques (le noyau d'un pion). Elle vaut donc 282,6 mm, et le losange grossit de 20 % au
 * minimum comme tout le reste — 3,40 -> 4,08 px sur une grande carte.
 *
 * POURQUOI UNE ÉQUIVALENCE PLUTÔT QU'UNE MESURE : le losange est précisément ce qu'on dessine
 * quand la famille du châssis est INCONNUE, donc quand aucune longueur monde n'existe. Il dit
 * « un objet est là, on ne sait pas lequel », et un glyphe d'ignorance ne doit pas peser autant
 * qu'un véhicule nommé — d'où une équivalence calée sur ce qu'il valait, jamais sur un châssis.
 */
const VEHICLE_UNKNOWN_HALF_MM = CORE_RADIUS / PX_PAR_MM_CADRAGE_V1

/**
 * VEHICLE_TURRET_HALF_MM — même équivalence pour le demi-côté du pictogramme de tourelle : celle
 * qui redonne ses 6,60 px historiques (0,75 pion) à l'ancienne échelle, soit 548,5 mm. 6,60 ->
 * 7,92 px sur une grande carte.
 *
 * UNE TOURELLE DOIT SE LIRE COMME UN OBJET DU TERRAIN, pas comme un pion de joueur : elle reste
 * sensiblement plus grande que le losange d'un châssis inconnu, sans atteindre la taille d'un
 * châssis conduit — et ce rapport est désormais tenu à TOUTE échelle, puisque les trois suivent
 * la même.
 */
const VEHICLE_TURRET_HALF_MM = (PION_SCREEN_PX * 0.75) / PX_PAR_MM_CADRAGE_V1

/**
 * VEHICLE_EDGE_FALLBACK_MM — la longueur monde équivalente du demi-gabarit qu'un véhicule dont
 * le sprite ou le manifeste n'est PAS ENCORE CHARGÉ prête à son étiquette : celle qui redonne
 * ses 4,40 px historiques (un demi-pion) à l'ancienne échelle. 4,40 -> 5,28 px sur une grande
 * carte.
 */
const VEHICLE_EDGE_FALLBACK_MM = (PION_SCREEN_PX / 2) / PX_PAR_MM_CADRAGE_V1

/**
 * screenScalePxPerMm — L'ÉCHELLE EFFECTIVE du calque : celle de la carte, ou le plancher, la
 * plus grande des deux.
 *
 * `scalePxPerM` est l'échelle du cadrage (`scaleOf(view)`, pixels CSS par mètre). Un cadrage
 * dégénéré — toile pas encore mesurée — rend le plancher, jamais zéro : un objet dont on connaît
 * la position se dessine, même avant que la scène ne soit cadrée.
 */
export function screenScalePxPerMm(scalePxPerM: number): number {
  const carte = scalePxPerM > 0 ? scalePxPerM / 1000 : 0
  return Math.max(carte, PX_PAR_MM_MINIMUM_ECRAN)
}

/**
 * screenLengthPx — LE MODÈLE, en une fonction : une longueur monde (millimètres) rendue à
 * l'échelle effective, plafond doux appliqué.
 */
export function screenLengthPx(worldLengthMm: number, scalePxPerM: number): number {
  if (worldLengthMm <= 0) return 0
  const brut = worldLengthMm * screenScalePxPerMm(scalePxPerM)
  return brut <= VEHICLE_SOFT_CEIL_PX
    ? brut
    : VEHICLE_SOFT_CEIL_PX + Math.sqrt(brut - VEHICLE_SOFT_CEIL_PX)
}

/**
 * spriteWorldLengthMm — la longueur MONDE d'un sprite de véhicule, en millimètres, lue dans le
 * COUPLE sprite × manifeste : la hauteur native du PNG (nez en haut, donc la hauteur EST l'axe
 * de longueur) × les millimètres-monde par pixel de sprite que le manifeste déclare pour SA
 * famille (`static/vehicles-assets/halo_infinite/replay/index.json`, dix-huit familles au statut
 * « valide », mesurées le 2026-09-02 — Mongoose 1 280 mm, Warthog 2 220, Scorpion 3 880,
 * Pélican 11 220).
 *
 * C'EST LA SEULE LONGUEUR MONDE MESURÉE DU CALQUE (cf. l'en-tête) : les deux glyphes n'en ont
 * qu'une ÉQUIVALENTE, et les autres familles du rejeu n'en ont aucune.
 */
export function spriteWorldLengthMm(naturalHeightPx: number, mmPerPx: number): number {
  if (naturalHeightPx <= 0 || mmPerPx <= 0) return 0
  return naturalHeightPx * mmPerPx
}

/** vehicleUnknownHalfPx — la demi-diagonale du losange d'un châssis non résolu, à l'échelle. */
export function vehicleUnknownHalfPx(scalePxPerM: number): number {
  return screenLengthPx(VEHICLE_UNKNOWN_HALF_MM, scalePxPerM)
}

/** vehicleTurretHalfPx — le demi-côté du pictogramme de tourelle, à l'échelle. */
export function vehicleTurretHalfPx(scalePxPerM: number): number {
  return screenLengthPx(VEHICLE_TURRET_HALF_MM, scalePxPerM)
}

/** vehicleEdgeFallbackPx — le demi-gabarit d'un véhicule dont le sprite n'est pas chargé. */
export function vehicleEdgeFallbackPx(scalePxPerM: number): number {
  return screenLengthPx(VEHICLE_EDGE_FALLBACK_MM, scalePxPerM)
}
