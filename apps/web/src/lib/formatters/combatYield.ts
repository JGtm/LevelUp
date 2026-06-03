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

const DASH = '—'

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
