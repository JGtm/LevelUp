/**
 * usageFormat.ts — LE FORMATAGE DES PARTS des formes partagées (jauges et bandes de la carte
 * « Appui reçu » de Sessions).
 *
 * Pur : aucun React, aucune couleur en dur, aucune lecture de store.
 */
import type { Locale } from '@/lib/i18n/locale'

/** Une part en pourcentage : « 45,6 % » / "45.6%". `null`/absent → tiret. */
export function formatUsagePct(v: number | null | undefined, locale: Locale): string {
  if (v == null) return '—'
  const s = v.toFixed(1)
  return locale === 'fr' ? `${s.replace('.', ',')} %` : `${s}%`
}
