/**
 * Helpers canoniques pour le formatage de dates (revue 2026-04-29 P2.6bis).
 *
 * Centralise les patterns dispersés dans :
 *   - features/lab/LabPage.tsx::formatDate (avec text fallback)
 *   - features/palmares/PalmaresRelationsPage.tsx::formatDate (medium)
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
 * Format date court DD/MM pour les axes de chart timeseries.
 *
 * VERROU 'fr-FR' DÉLIBÉRÉ (décision I2b, 2026-07-04 ; DETTE_ASSUMEE §2 I2b ; revu
 * V7d/VF-14 2026-07-07 — maintenu). Le rendu est NUMÉRIQUE PUR `DD/MM` (day+month
 * en 2-digit, sans nom de mois) : identique en FR et EN (`29/04` dans les deux
 * locales), donc locale-invariant à l'affichage. On fige explicitement pour
 * garantir l'ordre jour/mois sur l'axe quelle que soit la locale runtime (un
 * 'en-US' rendrait MM/DD et casserait la lecture de l'axe). Ne PAS threader la
 * locale ici : ce serait introduire une divergence d'ordre sans gain d'i18n.
 * Pour une date lisible localisée (nom de mois), utiliser formatDate(value, locale).
 *
 * @example
 *   formatDateShort('2026-04-29')   // "29/04" (FR comme EN)
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
