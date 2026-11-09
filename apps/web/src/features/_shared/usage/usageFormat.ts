/**
 * usageFormat.ts — LE FORMATAGE DES PARTS des formes partagées (jauges et bandes de la carte
 * « Appui reçu » de Sessions).
 *
 * Pur : aucun React, aucune couleur en dur, aucune lecture de store.
 */
import { formatNumber } from '@/lib/formatters/number'
import { intlLocale } from '@/lib/formatters/intlLocale'
import type { Locale } from '@/lib/i18n/locale'

/** Suffixe d'un pourcentage : espace avant le « % » en français, collé en anglais. */
const PCT_SUFFIX = { fr: ' %', en: '%' } as const satisfies Record<Locale, string>

/** Un écart en points de pourcentage, signé : « +3,2 pts » / "+3.2 pts". */
export function formatUsageGap(deltaPts: number, locale: Locale): string {
  const abs = formatNumber(Math.abs(deltaPts), intlLocale(locale), 1)
  const sign = deltaPts < 0 ? '-' : '+'
  return `${sign}${abs} pts`
}

/** Une part en pourcentage : « 45,6 % » / "45.6%". `null`/absent → tiret. */
export function formatUsagePct(v: number | null | undefined, locale: Locale): string {
  if (v == null) return '—'
  return `${formatNumber(v, intlLocale(locale), 1)}${PCT_SUFFIX[locale]}`
}
