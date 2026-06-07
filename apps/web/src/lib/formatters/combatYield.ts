/**
 * Formatage canonique du rendement combat (OC) et de la résistance (DR).
 *
 * Aligné sur la KPI card « Rendement / Résistance » de l'Explorer
 * (ExplorerTargetSampleStats.YieldTile) :
 *   - OC (rendement)  : valeur × 100, arrondi entier → "42%"
 *   - DR (résistance) : (valeur − 1) × 100, baseline 1.0 → "+18%" / "0%"
 *                       valeur < 0 = sentinelle ∞ (0 mort) → "∞"
 *
 * Renvoie un fallback (par défaut "—") pour null / undefined.
 */

import type { SemanticToken } from '@/lib/accessibility'

const DASH = '—'

/** Poids d'un assist en frag-équivalent (convention Halo : 1 assist = 1/3 de frag). */
const ASSIST_FRAG_WEIGHT = 3

/**
 * Dégâts par frag-équivalent = damage / (kills + assists/3). C'est l'inverse exact
 * du rendement offensif normalisé : offensive_conversion = 225 / effectiveDmgPerFrag.
 * Compte les assists au dénominateur pour rester aligné avec le % affiché.
 * Renvoie null si les dégâts sont absents ou si le dénominateur ≤ 0.
 */
export function effectiveDmgPerFrag(
  damage: number | null | undefined,
  kills: number,
  assists: number,
): number | null {
  if (damage == null || !Number.isFinite(damage)) return null
  const fragEquivalents = kills + assists / ASSIST_FRAG_WEIGHT
  if (fragEquivalents <= 0) return null
  return damage / fragEquivalents
}

/** OC = offensive_conversion (≥ 0). Affiché en pourcentage entier. */
export function formatOffensiveConversion(oc: number | null | undefined, fallback = DASH): string {
  if (oc == null || !Number.isFinite(oc)) return fallback
  return `${Math.round(oc * 100)}%`
}

/**
 * DR = defensive_resistance (baseline 1.0). Affiché en écart au baseline
 * ("+18%", "0%"). Valeur négative = sentinelle ∞ (aucune mort).
 */
export function formatDefensiveResistance(dr: number | null | undefined, fallback = DASH): string {
  if (dr == null || !Number.isFinite(dr)) return fallback
  if (dr < 0) return '∞'
  const pct = Math.round((dr - 1) * 100)
  return `${dr >= 1 ? '+' : ''}${pct}%`
}

/**
 * Accent de QUALITÉ du combat yield, pour la barre des KPI cards.
 * Référence = baseline affichée : rendement ≥ 100 % (oc ≥ 1.0) et résistance
 * ≥ +0 % (dr ≥ 1.0). « Plus le % est haut, mieux c'est. »
 *   - vert   : toutes les métriques présentes sont ≥ référence
 *   - rouge  : toutes en-dessous
 *   - neutre : mixte
 *   - undefined : aucune donnée
 */
export function combatYieldToken(
  oc: number | null | undefined,
  dr: number | null | undefined,
): SemanticToken | undefined {
  const states = [
    oc != null && Number.isFinite(oc) ? oc >= 1.0 : null,
    // dr < 0 = sentinelle ∞ (0 mort) → excellent.
    dr != null && Number.isFinite(dr) ? dr < 0 || dr >= 1.0 : null,
  ].filter((v): v is boolean => v !== null)
  if (states.length === 0) return undefined
  if (states.every((v) => v)) return 'divergent-pos'
  if (states.every((v) => !v)) return 'divergent-neg'
  return 'divergent-neutral'
}
