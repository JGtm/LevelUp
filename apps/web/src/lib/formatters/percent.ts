/**
 * Helper canonique pour formater un ratio (0..1) en pourcentage affiché.
 *
 * ADR 0006 (canonical-indicators-and-units) : l'API renvoie toujours un
 * ratio 0..1 pour les indicateurs (win_rate, accuracy, etc.). Le formatage
 * `*100` + arrondi décimal se fait UNIQUEMENT à l'affichage côté front.
 *
 * Convention décimale (revue 2026-04-29 axe 1 amendé) :
 *   - 1 décimale par défaut (taux principaux)
 *   - 2 décimales pour ratios sub-unitaires (KDA, KDR — passer decimals=2)
 *   - 0 décimale pour des compteurs simples (passer decimals=0)
 */

/**
 * Formate un ratio 0..1 en chaîne pourcentage (ex: 0.553 → "55.3 %").
 *
 * @param ratio    valeur 0..1 ; null/undefined/NaN → fallback "—"
 * @param decimals nombre de décimales (défaut 1)
 * @param fallback chaîne renvoyée si la valeur est invalide (défaut "—")
 *
 * @example
 *   formatPercent(0.5532)        // "55.3 %"
 *   formatPercent(0.5532, 2)     // "55.32 %"
 *   formatPercent(1)             // "100.0 %"
 *   formatPercent(null)          // "—"
 *   formatPercent(0.5, 0)        // "50 %"
 */
export function formatPercent(
  ratio: number | null | undefined,
  decimals = 1,
  fallback = '—',
): string {
  if (ratio == null || Number.isNaN(ratio)) {
    return fallback
  }
  return `${(ratio * 100).toFixed(decimals)} %`
}

/**
 * Variante sans le suffixe "%". Utile quand l'affichage est composé
 * (ex: "<value>% <label>").
 *
 * @example
 *   formatPercentValue(0.5532)   // "55.3"
 */
export function formatPercentValue(
  ratio: number | null | undefined,
  decimals = 1,
  fallback = '—',
): string {
  if (ratio == null || Number.isNaN(ratio)) {
    return fallback
  }
  return (ratio * 100).toFixed(decimals)
}

/**
 * Pourcentage ENTIER compact SANS espace avant le « % » — pour les cellules de
 * tableaux/briefings denses (ex: "55%"). Distinct de formatPercent (décimales +
 * espace typographique FR). Prend un ratio 0..1.
 *
 * @example
 *   formatPercentInt(0.5532)   // "55%"
 */
export function formatPercentInt(ratio: number | null | undefined, fallback = '—'): string {
  if (ratio == null || !Number.isFinite(ratio)) {
    return fallback
  }
  return `${Math.round(ratio * 100)}%`
}
