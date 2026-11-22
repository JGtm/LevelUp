/**
 * _mapMarkerOutline.ts — Les deux encres du CERNE des pions du rejeu 2D : une bande sombre bordée
 * d'un filet clair.
 *
 * Pourquoi deux tons et pourquoi invariantes. Le fond d'une carte va du presque noir (fond Rendu
 * ombré, creux du Relief) au presque blanc (haut de la rampe), et un contour d'un seul ton y
 * échoue d'un côté ou de l'autre. La bande sombre porte le contraste contre un sol clair, le filet
 * clair le porte contre un sol sombre : le MEILLEUR des deux atteint au moins 4,4:1 contre
 * n'importe quel gris (point de croisement de deux tons presque noir / blanc), quelle que soit la
 * vision. Un cerne qui dépendrait de la palette ou du thème perdrait cette garantie ; les deux
 * valeurs sont donc les mêmes dans toutes les palettes et dans les deux thèmes, et c'est
 * `mapReliefTokens.test.ts` qui les mesure contre chaque fond de carte.
 *
 * La bande reprend la teinte du contour des noms (`--replay-label-stroke`, rgb(8 12 18)) : le nom
 * et le pion qu'il décrit sont cernés de la même encre.
 */
import type { SemanticToken } from '../semantic-tokens'

type MapMarkerOutlineToken = Extract<SemanticToken, `marker-outline-${string}`>

export const MAP_MARKER_OUTLINE: Record<MapMarkerOutlineToken, string> = {
  'marker-outline-band': '#080C12', // bande sombre, posée contre le noyau
  'marker-outline-rim': '#FFFFFF', // filet clair, au bord extérieur
}
