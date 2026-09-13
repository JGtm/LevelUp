/**
 * format.ts — LE FORMATAGE du bloc « formes retenues » : parts, écarts signés,
 * durées, et le palier « rond » d'une échelle.
 *
 * Les trois premiers délèguent à `features/_shared/usage/usageFormat.ts` — le
 * formatage du bloc d'usage est déjà le bon (virgule en FR, tiret pour le non
 * mesuré), et une seconde règle d'arrondi ferait deux nombres différents pour la
 * même mesure sur deux écrans voisins.
 */
import { formatUsageCount, formatUsageDecimal, formatUsagePct } from '@/features/_shared/usage/usageFormat'
import type { Locale } from '@/lib/i18n/locale'

/** Une part en pourcentage. `null` → tiret (non mesuré). */
export function formatPct(v: number | null | undefined, locale: Locale): string {
  return formatUsagePct(v, locale)
}

/** Un nombre à une décimale (virgule en FR). */
export function formatNumber(v: number, locale: Locale): string {
  return formatUsageDecimal(v, locale)
}

/** Un compte brut, ou une durée en m:ss quand la colonne en est une. */
export function formatCount(
  v: number | null | undefined,
  locale: Locale,
  isDuration = false,
): string {
  return formatUsageCount(v, locale, isDuration)
}

/**
 * UN ÉCART SIGNÉ, toujours avec son signe — c'est la grandeur que la forme
 * « écart à la parité » mesure, et un écart sans signe ne veut rien dire. Le
 * signe moins est le vrai (U+2212), pas le trait d'union.
 */
export function formatSigned(v: number, locale: Locale): string {
  const sign = v > 0 ? '+' : v < 0 ? '−' : '±'
  return `${sign}${formatUsageDecimal(Math.abs(v), locale)}`
}

/**
 * niceBound — le palier « rond » au-dessus d'un maximum, pour qu'une barre ne
 * colle jamais au bord de sa piste et qu'une graduation intermédiaire se lise.
 * Mêmes régimes que l'artefact (unité, paire, dizaine, quart de cent).
 */
export function niceBound(max: number): number {
  if (!Number.isFinite(max) || max <= 1) return 1
  if (max <= 5) return Math.ceil(max)
  if (max <= 12) return Math.ceil(max / 2) * 2
  if (max <= 60) return Math.ceil(max / 10) * 10
  return Math.ceil(max / 25) * 25
}

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
