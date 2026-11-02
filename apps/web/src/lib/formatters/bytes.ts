/**
 * Volume d'octets en unité lisible, par langue (décimal : 1 Ko = 1 000 o).
 * Partagé par l'administration (ressources, bases) et les réglages (archive des films).
 */
import { intlLocale } from './intlLocale'
import type { Locale } from '@/lib/i18n/locale'

/** Unités d'octets par langue (puissances de 1 000) : octet, kilo, méga, giga. */
const BYTE_UNITS = {
  fr: ['o', 'Ko', 'Mo', 'Go'],
  en: ['B', 'KB', 'MB', 'GB'],
} as const satisfies Record<Locale, readonly [string, string, string, string]>

/** Formate un volume d'octets en unité lisible ; `undefined` ou NaN rend un tiret. */
export function formatBytes(bytes: number | undefined, locale: Locale): string {
  if (bytes == null || Number.isNaN(bytes)) return '—'
  const abs = Math.abs(bytes)
  const fmt = (v: number, unit: string) =>
    `${v.toLocaleString(intlLocale(locale), { maximumFractionDigits: 1 })} ${unit}`
  const [unit, kilo, mega, giga] = BYTE_UNITS[locale]
  if (abs < 1_000) return fmt(bytes, unit)
  if (abs < 1_000_000) return fmt(bytes / 1_000, kilo)
  if (abs < 1_000_000_000) return fmt(bytes / 1_000_000, mega)
  return fmt(bytes / 1_000_000_000, giga)
}
