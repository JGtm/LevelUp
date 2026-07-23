/**
 * Helpers de formatage Ascension : interpolation simple, palier PP, date FR/EN.
 *
 * Pas d'ICU/plural — règle MVP cohérente avec notifications/format.ts :
 * remplacement de `{plural}` par 's' si n > 1, vide sinon. Suffisant pour
 * la complexité actuelle ; on migrera à intl-messageformat si plural complexe.
 */
import type { Locale } from '@/lib/i18n/locale'
import { intlLocale } from '@/lib/formatters'

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
export function formatAscensionDate(iso: string | null | undefined, locale: Locale): string {
  if (!iso) return ''
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return ''
  return d.toLocaleDateString(intlLocale(locale), {
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

/**
 * Progression (0..100) dans la bande de paliers PP courante : du dernier palier
 * atteint (ou 0) vers le prochain. 100 si déjà au multiplicateur max (30+).
 */
export function streakTierProgressPct(length: number): number {
  const next = nextPPTier(length)
  if (!next) return 100
  let prev = 0
  for (const t of STREAK_PP_TIERS) {
    if (t.length <= length) prev = t.length
    else break
  }
  const span = next.length - prev
  if (span <= 0) return 100
  return Math.max(0, Math.min(100, ((length - prev) / span) * 100))
}

/** Format multiplier en string court (×1.25, ×1.75). */
export function formatMultiplier(value: number): string {
  return `×${value.toFixed(2).replace(/\.00$/, '').replace(/0$/, '')}`
}

/**
 * Formate une valeur numérique selon la sémantique de la métrique :
 * - accuracy / ratio 0..1 → pourcentage (55.0 %)
 * - KDA / KPM / PSPM → 2 décimales, trailing zeros strippés (1.45 / 0.8 / 87.5)
 * - compteurs entiers (matches_played, wins, kills, etc.) → arrondi entier
 *   avec séparateur de milliers en-US (virgule) — déterministe inter-env
 * - défaut → 2 décimales avec trailing strip
 */
export function formatMetricValue(metric: string, value: number): string {
  const intMetrics = new Set([
    'matches_played',
    'wins',
    'kills',
    'headshots',
    'assists',
    'accuracy_threshold_days',
  ])
  if (intMetrics.has(metric)) {
    // Locale fixée à en-US pour produire un séparateur déterministe.
    return Math.round(value).toLocaleString('en-US')
  }
  if (metric === 'accuracy') {
    return `${(value * 100).toFixed(1)} %`
  }
  // Score, KDA, KPM, PSPM → 2 décimales puis strip trailing zeros.
  return stripTrailingZeros(value.toFixed(2))
}

// stripTrailingZeros enlève les zéros de fin et le point final éventuel.
// "87.50" → "87.5" ; "1.00" → "1" ; "0.80" → "0.8" ; "1.45" → "1.45"
function stripTrailingZeros(s: string): string {
  if (!s.includes('.')) return s
  return s.replace(/0+$/, '').replace(/\.$/, '')
}

