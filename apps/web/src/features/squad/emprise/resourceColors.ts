/**
 * resourceColors.ts — le code couleur des RESSOURCES de l'onglet Emprise (D9 du plan
 * PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26) : bonus sarcelle, armes spéciales violet, véhicules
 * orange, armes de râtelier bleu. Une pastille de cette
 * couleur précède chaque nom de ressource (spec S8). Jetons seulement : aucune valeur en dur.
 */
import { resolveToken, tokenCssVar, type SemanticToken } from '@/lib/accessibility'

import { RESOURCE_POWERUP, RESOURCE_POWER_WEAPON, RESOURCE_RACK, RESOURCE_VEHICLE } from './emprise.logic'

const RESOURCE_TOKENS: Record<string, SemanticToken> = {
  [RESOURCE_POWERUP]: 'resource-powerup',
  [RESOURCE_POWER_WEAPON]: 'resource-power-weapon',
  [RESOURCE_VEHICLE]: 'resource-vehicle',
  [RESOURCE_RACK]: 'resource-rack',
}

/** Repli neutre pour une ressource sans jeton (jamais rendue : voir `RESOURCE_ORDER`). */
const FALLBACK: SemanticToken = 'frag-unattributed'

/** La couleur d'une ressource, en variable CSS (DOM). */
export function resourceInk(resource: string): string {
  return tokenCssVar(RESOURCE_TOKENS[resource] ?? FALLBACK)
}

/** La couleur d'une ressource, résolue (graphes ECharts, relue à chaque rendu). */
export function resolveResourceColor(resource: string): string {
  return resolveToken(RESOURCE_TOKENS[resource] ?? FALLBACK)
}
