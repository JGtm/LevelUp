/**
 * Helpers de formatage Ascension : interpolation simple, palier PP, date FR/EN.
 *
 * Pas d'ICU/plural — règle MVP cohérente avec notifications/format.ts :
 * remplacement de `{plural}` par 's' si n > 1, vide sinon. Suffisant pour
 * la complexité actuelle ; on migrera à intl-messageformat si plural complexe.
 */
import type { AscensionLocale } from './i18n'

/**
 * Substitue les placeholders `{key}` par la valeur correspondante.
 * Cas particulier : `{plural}` retourne 's' si n > 1, vide sinon.
 */
export function interpolate(
  template: string,
  params: Record<string, string | number>,
): string {
  return template.replace(/\{(\w+)\}/g, (_, key) => {
    if (key === 'plural') {
      const n = typeof params.n === 'number' ? params.n : Number(params.count ?? 0)
      return n > 1 ? 's' : ''
    }
    const v = params[key]
    return v === undefined ? '' : String(v)
  })
}

/** Formate une date ISO en JJ/MM/AAAA (FR) ou MM/DD/YYYY (EN). */
export function formatDate(iso: string | null | undefined, locale: AscensionLocale): string {
  if (!iso) return ''
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return ''
  return d.toLocaleDateString(locale === 'fr' ? 'fr-FR' : 'en-US', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
  })
}

/**
 * Paliers de streak qui déclenchent un palier PP (alignés sur le backend —
 * apps/go-api/internal/progression/streaks/types.go PPMultiplier).
 */
export const STREAK_PP_TIERS: ReadonlyArray<{ length: number; multiplier: number }> = [
  { length: 4, multiplier: 1.1 },
  { length: 8, multiplier: 1.25 },
  { length: 15, multiplier: 1.5 },
  { length: 30, multiplier: 1.75 },
]

/**
 * Retourne le prochain palier PP au-dessus de la longueur courante, ou null
 * si déjà au max (30+).
 */
export function nextPPTier(length: number): { length: number; multiplier: number } | null {
  for (const t of STREAK_PP_TIERS) {
    if (length < t.length) return t
  }
  return null
}

/** Format multiplier en string court (×1.25, ×1.75). */
export function formatMultiplier(value: number): string {
  return `×${value.toFixed(2).replace(/\.00$/, '').replace(/0$/, '')}`
}
