/**
 * _positionsZone.ts — LA ZONE D'AFFICHAGE DE « OCCUPATION DU TERRAIN » : la boîte que le bloc offre au plan,
 * celle sur laquelle la règle Auto de l'orientation (celle du rejeu) juge si la carte gagne à être tournée.
 *
 * Le cadre du plan ne prend pas toute la largeur de la carte : il est borné DEUX FOIS — par une part de la
 * largeur utile et par un plafond de hauteur — et c'est la LARGEUR qui se réduit pour tenir la hauteur, jamais
 * le rapport du monde (un plan étiré désalignerait le calque de chaleur de son fond). Ces deux bornes décident
 * du style du cadre (`MatchPositionsHeatmap`) ET de la boîte de la règle Auto : elles vivent ici, une seule fois.
 *
 * Pur : aucun React, aucun canvas.
 */
import type { Dimensions } from '@/lib/replay/orientation'

import type { MapFrame } from './_positionsHeat'

/**
 * PLAFOND DE HAUTEUR DU PLAN. Le cadre prenait toute la largeur de la carte : sur un plan carré, cela donnait un
 * pavé aussi haut que large, qui poussait le reste de l'onglet hors de l'écran.
 */
export const PLAN_MAX_HEIGHT_REM = 21.6

/** PART DE LA LARGEUR DISPONIBLE que le plan peut prendre ; le rapport avec le plafond de hauteur fixe la forme du cadre. */
export const PLAN_MAX_WIDTH_PCT = 72

/** La taille de base d'un `rem`, en px : le plafond de hauteur se compare à une largeur mesurée en px. */
export const PX_PAR_REM = 16

/**
 * boiteDeLOccupation rend la boîte que le bloc offre au plan, en px : `PLAN_MAX_WIDTH_PCT` % de la largeur utile
 * de sa zone (`useContentWidth`) sur `PLAN_MAX_HEIGHT_REM` de haut. Une largeur utile pas encore mesurée (0)
 * rend une boîte nulle : la règle Auto laisse alors le plan droit.
 */
export function boiteDeLOccupation(largeurUtile: number): Dimensions {
  if (!(largeurUtile > 0)) return { largeur: 0, hauteur: 0 }
  return { largeur: (largeurUtile * PLAN_MAX_WIDTH_PCT) / 100, hauteur: PLAN_MAX_HEIGHT_REM * PX_PAR_REM }
}

/** dimensionsDuCadre rend la carte posée DROITE, en unités monde, d'après le cadre du fond ; `null` sans cadre. */
export function dimensionsDuCadre(frame: MapFrame | null): Dimensions | null {
  return frame && frame.widthM > 0 && frame.heightM > 0 ? { largeur: frame.widthM, hauteur: frame.heightM } : null
}
