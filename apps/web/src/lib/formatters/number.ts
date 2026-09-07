/**
 * Helpers canoniques pour le formatage de nombres (revue 2026-04-29 P2.6bis).
 *
 * Centralise les patterns dispersés dans :
 *   - features/lab/LabPage.tsx::formatNumber (toLocaleString)
 *   - features/session-detail/SessionDetailPage.tsx::formatNumber (toFixed)
 *   - features/palmares/PalmaresRelationsPage.tsx::formatKDA (sub-unitaire 2 décimales)
 *   - components/charts/_utils.ts::formatNumber (toFixed pour tooltips)
 *
 * Convention décimale (ADR 0006 + revue axe 1) :
 *   - Compteurs simples (kills, deaths) : 0 décimales
 *   - Pourcentages affichés (win_rate * 100) : 1 décimale (cf. formatPercent)
 *   - Ratios sub-unitaires (KDA, KDR) : 2 décimales
 */

import type { Locale } from './date'

/**
 * Format chiffre arrondi avec N décimales. Utilise toLocaleString (locale-
 * sensitive : séparateurs FR vs EN). Pour les charts qui veulent un format
 * stable indépendant de la locale, utiliser `formatNumberFixed` à la place.
 *
 * @example
 *   formatNumber(12345.678, 'fr-FR', 1)   // "12 345,7"
 *   formatNumber(12345.678, 'en-US', 0)   // "12,346"
 *   formatNumber(null, 'fr-FR')           // "—"
 */
export function formatNumber(
  value: number | null | undefined,
  locale: Locale,
  decimals = 0,
  fallback = '—',
): string {
  if (value == null || Number.isNaN(value)) return fallback
  return value.toLocaleString(locale, {
    minimumFractionDigits: decimals,
    maximumFractionDigits: decimals,
  })
}

/**
 * Format chiffre avec toFixed (pas de séparateurs locale-sensitive).
 * Convient aux tooltips chart qui veulent un format stable.
 *
 * @example
 *   formatNumberFixed(12.345, 1)   // "12.3"
 *   formatNumberFixed(null)        // "—"
 *   formatNumberFixed(NaN)         // "—"
 */
export function formatNumberFixed(
  value: number | null | undefined,
  decimals = 1,
  fallback = '—',
): string {
  if (value == null || Number.isNaN(value) || !Number.isFinite(value)) {
    return fallback
  }
  return value.toFixed(decimals)
}

/**
 * Format d'un delta SIGNÉ à N décimales avec préfixe explicite et glyphe UNIQUE
 * '−' (U+2212, jamais le tiret ASCII) : le signe doit rester lisible sans dépendre
 * de la seule couleur. Zéro → '±0' / '±0.00' (jamais '-0'). null/undefined/non
 * fini → `fallback` (défaut '' — les deltas absents n'affichent rien).
 *
 * Source UNIQUE des deltas signés à décimales fixes (delta de rang CSR/LUSR, deltas
 * de briefing) — garde-rail `signed-format.guard.test.ts`. NE PAS confondre avec
 * `formatSignedPoints` (points entiers '±0 pts', @/lib/baseline) ni avec le
 * `formatDelta` de delta-card (précision dynamique selon magnitude + objet couleur).
 *
 * @example
 *   formatSignedFixed(0.3, 2)   // "+0.30"
 *   formatSignedFixed(-1.5, 2)  // "−1.50"  (U+2212)
 *   formatSignedFixed(0, 2)     // "±0.00"
 *   formatSignedFixed(45, 0)    // "+45"
 *   formatSignedFixed(null, 2)  // ""
 */
export function formatSignedFixed(
  value: number | null | undefined,
  decimals: number,
  fallback = '',
): string {
  if (value == null || !Number.isFinite(value)) return fallback
  const rounded = Number(value.toFixed(decimals))
  if (rounded === 0) return `±${(0).toFixed(decimals)}`
  const abs = Math.abs(rounded).toFixed(decimals)
  return rounded > 0 ? `+${abs}` : `−${abs}`
}

/**
 * Format ratio sub-unitaire (KDA, KDR) avec 2 décimales locale-sensitive.
 *
 * @example
 *   formatRatio(2.345, 'fr-FR')   // "2,35"
 *   formatRatio(null, 'fr-FR')    // "—"
 */
export function formatRatio(
  value: number | null | undefined,
  locale: Locale,
  fallback = '—',
): string {
  return formatNumber(value, locale, 2, fallback)
}

/** Alias domaine — KDA/KDR sont des ratios sub-unitaires. */
export const formatKDA = formatRatio
