/**
 * format.ts — l'heure locale d'un match, lue par les cartes de l'Emprise et de l'Objectif
 * (axe des matchs, infobulles) et par les tests de la page Emprise.
 */
import type { Locale } from '@/lib/i18n/locale'

/** L'heure locale d'un match (« 19:22 »), depuis son horodatage ISO. */
export function formatMatchTime(iso: string | undefined, locale: Locale): string {
  if (!iso) return ''
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return ''
  return d.toLocaleTimeString(locale === 'fr' ? 'fr-FR' : 'en-GB', {
    hour: '2-digit',
    minute: '2-digit',
  })
}
