/**
 * format.ts — helpers de formatage purs du dashboard admin (testables sans
 * React). Strings paramétrées en dictionnaire local (pattern
 * notifications/relativeTime.ts) — les manifests TOML ne gèrent pas les
 * placeholders.
 */
import { intlLocale } from '@/lib/formatters'

export type AdminLocale = 'fr' | 'en'

const REL = {
  fr: {
    justNow: "à l'instant",
    minutesAgo: (n: number) => `il y a ${n} min`,
    hoursAgo: (n: number) => `il y a ${n} h`,
    daysAgo: (n: number) => `il y a ${n} j`,
  },
  en: {
    justNow: 'just now',
    minutesAgo: (n: number) => `${n} min ago`,
    hoursAgo: (n: number) => `${n} h ago`,
    daysAgo: (n: number) => `${n} d ago`,
  },
} as const

/**
 * Formate un timestamp ISO en libellé relatif court. Au-delà de 7 jours →
 * date locale. ISO vide/invalide → '—' (les snapshots « jamais couru »
 * arrivent avec une string vide).
 */
export function adminRelativeTime(iso: string | undefined, locale: AdminLocale, now = Date.now()): string {
  if (!iso) return '—'
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) return '—'
  const t = REL[locale]
  const minutes = Math.round((now - date.getTime()) / 60_000)
  if (minutes < 1) return t.justNow
  if (minutes < 60) return t.minutesAgo(minutes)
  const hours = Math.round(minutes / 60)
  if (hours < 24) return t.hoursAgo(hours)
  const days = Math.round(hours / 24)
  if (days < 7) return t.daysAgo(days)
  return date.toLocaleDateString(intlLocale(locale))
}

/**
 * Formate une durée en millisecondes en libellé compact : "850 ms",
 * "2,4 s" / "2.4 s", "1 min 05 s", "1 h 12 min". Négatif/NaN → '—'.
 */
export function formatDurationMs(ms: number | undefined, locale: AdminLocale): string {
  if (ms === undefined || Number.isNaN(ms) || ms < 0) return '—'
  if (ms < 1000) return `${Math.round(ms)} ms`
  const seconds = ms / 1000
  if (seconds < 60) {
    const rounded = Math.round(seconds * 10) / 10
    const text = locale === 'fr' ? String(rounded).replace('.', ',') : String(rounded)
    return `${text} s`
  }
  const totalMinutes = Math.floor(seconds / 60)
  if (totalMinutes < 60) {
    const rest = Math.round(seconds - totalMinutes * 60)
    return rest > 0 ? `${totalMinutes} min ${String(rest).padStart(2, '0')} s` : `${totalMinutes} min`
  }
  const hours = Math.floor(totalMinutes / 60)
  const restMin = totalMinutes - hours * 60
  return restMin > 0 ? `${hours} h ${String(restMin).padStart(2, '0')} min` : `${hours} h`
}

/**
 * Formate un horodatage ISO en datetime locale complète (pour les `title=`
 * au survol des temps relatifs). ISO vide → ''.
 */
export function adminAbsoluteTime(iso: string | undefined, locale: AdminLocale): string {
  if (!iso) return ''
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) return ''
  return date.toLocaleString(intlLocale(locale))
}

/** Formate un intervalle scheduler en minutes vers un libellé lisible. */
export function formatIntervalMinutes(minutes: number | undefined, locale: AdminLocale): string {
  if (!minutes || minutes <= 0) return '—'
  if (minutes < 60) return `${minutes} min`
  const hours = Math.floor(minutes / 60)
  const rest = minutes % 60
  if (rest === 0) return `${hours} h`
  return locale === 'fr' ? `${hours} h ${rest} min` : `${hours} h ${rest} min`
}
