/**
 * Helpers canoniques pour le formatage de dates (revue 2026-04-29 P2.6bis).
 *
 * Centralise les patterns dispersés dans :
 *   - features/lab/LabPage.tsx::formatDate (avec text fallback)
 *   - features/palmares/PalmaresRelationsPage.tsx::formatDate (medium)
 *   - features/squad/v2/components/HistoryTable.tsx::formatDate (DD/MM/YY)
 *   - components/charts/_utils.ts::formatDateShort (DD/MM)
 *
 * Convention : passer la locale en argument explicite (pas de defaut FR/EN
 * hardcoded — le composant doit la résoudre via le store i18n).
 */

/** Locale BCP-47 (ex: 'fr-FR', 'en-US'). */
export type Locale = string

/**
 * Formate une date ISO en chaîne lisible selon la locale.
 *
 * @param value    Date | string ISO | timestamp ms | null/undefined
 * @param locale   BCP-47 (ex: 'fr-FR')
 * @param opts     Intl.DateTimeFormatOptions (défaut : dateStyle='medium')
 * @param fallback chaîne renvoyée si value invalide (défaut "—")
 *
 * @example
 *   formatDate('2026-04-29T12:00:00Z', 'fr-FR')      // "29 avr. 2026"
 *   formatDate('2026-04-29T12:00:00Z', 'fr-FR', { dateStyle: 'short' })
 *                                                      // "29/04/2026"
 *   formatDate(null, 'fr-FR')                        // "—"
 */
export function formatDate(
  value: Date | string | number | null | undefined,
  locale: Locale,
  opts: Intl.DateTimeFormatOptions = { dateStyle: 'medium' },
  fallback = '—',
): string {
  if (value == null || value === '') return fallback
  const d = value instanceof Date ? value : new Date(value)
  if (Number.isNaN(d.getTime())) return fallback
  return new Intl.DateTimeFormat(locale, opts).format(d)
}

/**
 * Format date court FR (DD/MM) pour les axes de chart timeseries.
 * Verrouillé sur fr-FR — usage chart uniquement.
 *
 * @example
 *   formatDateShort('2026-04-29')   // "29/04"
 */
export function formatDateShort(value: Date | string | number): string {
  const d = value instanceof Date ? value : new Date(value)
  return d.toLocaleDateString('fr-FR', { day: '2-digit', month: '2-digit' })
}

/**
 * Format date+time complet selon la locale (toLocaleString avec defaults).
 *
 * @example
 *   formatDateTime('2026-04-29T20:30:00Z', 'fr-FR')  // "29/04/2026 22:30:00"
 */
export function formatDateTime(
  value: Date | string | number | null | undefined,
  locale: Locale,
  fallback = '—',
): string {
  if (value == null || value === '') return fallback
  const d = value instanceof Date ? value : new Date(value)
  if (Number.isNaN(d.getTime())) return fallback
  return d.toLocaleString(locale)
}
