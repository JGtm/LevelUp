/**
 * screenSizes.ts — LE MODÈLE DE TAILLE DU REJEU 2D : une taille RÉELLE quand l'échelle le
 * permet, une taille MINIMALE GARANTIE à l'écran sinon.
 *
 * ## La décision, et ce qu'elle remplace (utilisateur, 2026-09-19)
 *
 * Les véhicules étaient dessinés à une taille CONSTANTE en pixels d'écran, calculée une fois
 * pour toutes contre une cible de cadrage (« un Mongoose fait 1,75 pion de long ») et SANS
 * aucun rapport avec l'échelle de la carte. Un Ghost avait donc la même taille à l'écran sur
 * une carte de 54 m et sur une de 273 m — ce qui est faux dans les deux sens : trop gros sur la
 * petite, trop petit sur la grande, et c'est le retour utilisateur du 2026-09-19.
 *
 * LE MODÈLE RETENU EST CELUI DES MARQUEURS DE POINT DE PASSAGE D'UN JEU : un objet prend sa
 * taille RÉELLE — sa longueur monde × l'échelle du cadrage — dès que cette taille est lisible,
 * et se pose sur un PLANCHER D'ÉCRAN quand elle ne l'est plus.
 *
 *     taille = plafond_doux( max( longueur_monde × échelle_px_par_m , minimum_écran[famille] ) )
 *
 * TROIS PROPRIÉTÉS, ET CHACUNE EST CE QUI FAIT L'INTÉRÊT DU MODÈLE :
 *
 *   PROPORTIONS VRAIES   au-dessus du seuil, deux familles sont entre elles dans le rapport de
 *                        leurs longueurs monde, exactement. Un Scorpion (3,88 m) mesure 3,03
 *                        fois un Mongoose (1,28 m) à l'écran, quelle que soit la carte.
 *   LISIBILITÉ TENUE     sous le seuil, plus rien ne rétrécit : un Mongoose sur Flood Gulch
 *                        vaudrait 4,5 px de long à l'échelle réelle — un point, pas un
 *                        véhicule. Le minimum le tient à 18,5 px.
 *   LE ZOOM RÉVÈLE       zoomer augmente l'échelle, donc fait passer les familles au-dessus du
 *                        seuil une à une : sur Flood Gulch, 3 familles sur 18 sont à leur
 *                        taille réelle à 1×, 13 sur 18 à 3×. L'utilisateur récupère la vérité
 *                        des tailles en zoomant, sans jamais perdre la lisibilité en dézoomant.
 *
 * ## Les pixels sont LOGIQUES, jamais physiques
 *
 * Tous les minimums de ce fichier sont en pixels CSS. La densité du périphérique (`k`,
 * `devicePixelRatio`) est appliquée AU TRACÉ, par l'appelant, exactement comme pour les
 * épaisseurs de trait des marqueurs — un écran à `dpr = 2` rend donc la MÊME taille logique
 * avec deux fois plus de pixels physiques, et jamais un objet deux fois plus petit. L'échelle
 * du cadrage (`scaleOf(view)`) est elle aussi en pixels CSS par mètre : `view.width` est la
 * largeur du conteneur, pas celle de la toile physique.
 *
 * ## Les familles qui n'ont PAS de longueur monde mesurée
 *
 * Le modèle a deux termes ; une famille sans longueur monde mesurée n'a que le second, et sa
 * taille EST son minimum. C'est le cas, mesuré, de toutes les familles hors véhicules :
 *
 *   pion / bipède          `replayMarkers.CORE_RADIUS` — aucune longueur monde n'est publiée
 *                          pour un joueur, et le pion n'en est de toute façon pas une
 *                          représentation : c'est un MARQUEUR. Sa référence ne bouge pas
 *                          (décision utilisateur du 2026-09-19), et c'est elle qui sert
 *                          d'unité à tous les minimums ci-dessous.
 *   socle d'arme           `weaponPadsLayer.PAD_DOT_PX` / `PAD_ICON_H_PX` — un socle est un
 *                          point d'apparition, pas un objet ; le document n'en publie aucune
 *                          dimension.
 *   drapeau / objectif     `placementShapes` — glyphes de forme, sans dimension publiée.
 *   équipement             idem.
 *
 * Ces quatre-là gardent donc leur constante là où elle vit, et ce fichier les INVENTORIE pour
 * que le modèle se lise en un seul endroit — il n'y a aucune dimension à en sortir tant que le
 * document n'en publie pas.
 */
import { PION_VISIBLE_DIAMETER_PX } from '../layers/replayMarkers'

/**
 * PION_SCREEN_PX — L'UNITÉ DE TOUS LES MINIMUMS : le diamètre RÉELLEMENT VU d'un pion au
 * rez-de-chaussée, liseré compris.
 *
 * POURQUOI LE PION ET PAS UN NOMBRE DE PIXELS. Le pion est la seule grandeur de ce calque que
 * l'utilisateur voit en permanence et à laquelle il compare tout le reste (« le Ghost est plus
 * petit que le pion du joueur, ça fait hyper bizarre », 2026-09-09). Exprimer les minimums en
 * pions les rend lisibles ET solidaires : le jour où le pion change de taille, l'échelle des
 * minimums suit sans qu'aucune valeur ne soit retouchée.
 */
export const PION_SCREEN_PX = PION_VISIBLE_DIAMETER_PX

/**
 * VEHICLE_MIN_SCREEN_PX — LE MINIMUM GARANTI D'UN VÉHICULE, toutes familles de châssis
 * confondues : 2,1 pions de long, soit 18,48 px CSS.
 *
 * LA VALEUR EST CELLE QUE L'UTILISATEUR A VALIDÉE LE 2026-09-19 (« les véhicules sont trop
 * petits, +20 % » — 1,75 pion × 1,2 = 2,1), et elle change de RÔLE : ce n'est plus la taille
 * des véhicules, c'est le plancher sous lequel aucun ne descend. Sur une grande carte l'effet
 * est exactement celui du +20 % demandé (Flood Gulch, 3,49 px/m : toutes les familles sous
 * 5,3 m de long y sont au plancher) ; sur une petite carte la taille réelle passe devant et les
 * véhicules y sont plus gros encore (Snowbound, 17,56 px/m : Ghost 20,3 -> 29,7 px, aucune
 * famille au plancher).
 *
 * UN SEUL MINIMUM POUR LES DIX-HUIT FAMILLES, et c'est délibéré : un minimum par famille
 * rétablirait sous le seuil une hiérarchie de tailles que la mesure ne soutient plus à cette
 * échelle — le seuil est précisément l'endroit où l'écran cesse de pouvoir distinguer un
 * Mongoose d'un Warthog (4,5 px contre 7,7 px sur Flood Gulch).
 */
export const VEHICLE_MIN_SCREEN_PX = 2.1 * PION_SCREEN_PX

/**
 * VEHICLE_SOFT_CEIL_PX — le plafond DOUX, INCHANGÉ en valeur (7 pions, 61,60 px) : au-delà, la
 * croissance ralentit en racine carrée au lieu de s'arrêter net — un véhicule très long reste
 * visiblement plus grand qu'un plus petit, mais cesse de dominer l'écran.
 *
 * IL DEVIENT UTILE, ALORS QU'IL NE SERVAIT JAMAIS. Sous l'ancien modèle à taille constante,
 * aucune famille ne l'atteignait sauf les deux dropships. Avec la taille réelle, il mord dès
 * qu'on regarde une petite carte : sur Snowbound (17,56 px/m) le Pélican y passe de 197 px
 * — un tiers de la largeur de la scène — à 73 px, et le Scorpion de 68 à 64.
 */
export const VEHICLE_SOFT_CEIL_PX = 7 * PION_SCREEN_PX

/**
 * VEHICLE_UNKNOWN_MIN_HALF_PX — la demi-diagonale du losange d'un châssis NON RÉSOLU : un demi
 * pion, soit une diagonale d'UN pion.
 *
 * IL N'A PAS DE TERME RÉALISTE, et ce n'est pas un oubli : le losange est précisément ce qu'on
 * dessine quand la famille du châssis est inconnue — donc quand aucune longueur monde n'est
 * disponible. Il reste petit par construction : il dit « un objet est là, on ne sait pas
 * lequel », et un glyphe d'ignorance ne doit pas peser autant qu'un véhicule nommé.
 *
 * MONTÉ DE 3,40 À 4,40 PX LE 2026-09-20 avec le reste du calque : à 3,40 il valait le NOYAU
 * d'un pion sans son liseré, une ancre que la mesure du 2026-09-09 avait déjà établie comme
 * trop petite pour les véhicules.
 */
export const VEHICLE_UNKNOWN_MIN_HALF_PX = 0.5 * PION_SCREEN_PX

/**
 * VEHICLE_TURRET_MIN_HALF_PX — le demi-côté minimal du pictogramme de tourelle : 0,75 pion,
 * soit 6,60 px. INCHANGÉ en valeur, mais il devient un MINIMUM : quand la famille de la
 * tourelle porte une longueur monde au manifeste (Shade 1,53 m, tourelle montée 1,39 m), le
 * pictogramme prend sa demi-longueur réelle dès qu'elle dépasse ce seuil — sur Snowbound la
 * Shade passe ainsi de 6,60 à 13,4 px de demi-côté.
 *
 * UNE TOURELLE DOIT SE LIRE COMME UN OBJET DU TERRAIN, pas comme un pion de joueur : elle est
 * donc sensiblement plus grande que le losange d'un châssis inconnu, sans atteindre la taille
 * d'un châssis conduit.
 */
export const VEHICLE_TURRET_MIN_HALF_PX = 0.75 * PION_SCREEN_PX

/**
 * screenLengthPx — LE MODÈLE, en une fonction : la taille réelle si l'échelle la rend lisible,
 * le minimum de la famille sinon, le plafond doux au-dessus.
 *
 * `worldLengthM` est la longueur MONDE de l'objet (mètres) ; `scalePxPerM` l'échelle du cadrage
 * (`scaleOf(view)`, pixels CSS par mètre) ; `minPx` le minimum garanti de sa famille. Un cadrage
 * dégénéré ou une longueur nulle ne rendent PAS zéro mais le minimum : un objet dont on connaît
 * la position se dessine, même quand on ne sait pas encore le mesurer.
 */
export function screenLengthPx(worldLengthM: number, scalePxPerM: number, minPx: number): number {
  const realiste = worldLengthM > 0 && scalePxPerM > 0 ? worldLengthM * scalePxPerM : 0
  const plancher = Math.max(realiste, minPx)
  return plancher <= VEHICLE_SOFT_CEIL_PX
    ? plancher
    : VEHICLE_SOFT_CEIL_PX + Math.sqrt(plancher - VEHICLE_SOFT_CEIL_PX)
}

/**
 * spriteWorldLengthM — la longueur MONDE d'un sprite de véhicule, en mètres, lue dans le COUPLE
 * sprite × manifeste : la hauteur native du PNG (nez en haut, donc la hauteur EST l'axe de
 * longueur) × les millimètres-monde par pixel de sprite que le manifeste déclare pour SA
 * famille (`static/vehicles-assets/halo_infinite/replay/index.json`, dix-huit familles au statut
 * « valide », mesurées le 2026-09-02 — Mongoose 1,28 m, Warthog 2,22 m, Scorpion 3,88 m,
 * Pélican 11,22 m).
 *
 * C'EST LA SEULE LONGUEUR MONDE MESURÉE DU CALQUE, et c'est pour cela que les véhicules sont la
 * seule famille à avoir un terme réaliste (cf. l'en-tête).
 */
export function spriteWorldLengthM(naturalHeightPx: number, mmPerPx: number): number {
  if (naturalHeightPx <= 0 || mmPerPx <= 0) return 0
  return (naturalHeightPx * mmPerPx) / 1000
}
