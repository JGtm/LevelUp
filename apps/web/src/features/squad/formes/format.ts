/**
 * format.ts — l'heure locale d'un match, lue par les cartes de l'Emprise et de l'Objectif
 * (axe des matchs, infobulles) et par les tests de la page Emprise.
 */
import type { Locale } from '@/lib/i18n/locale'
import { intlLocale } from '@/lib/formatters/intlLocale'

/** L'heure locale d'un match (FR « 19:22 », EN « 07:22 PM »), depuis son horodatage ISO. */
export function formatMatchTime(iso: string | undefined, locale: Locale): string {
  if (!iso) return ''
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return ''
  return d.toLocaleTimeString(intlLocale(locale), {
    hour: '2-digit',
    minute: '2-digit',
  })
}
