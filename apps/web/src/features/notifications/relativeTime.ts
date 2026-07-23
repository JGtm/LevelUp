import { getNotificationsText } from './i18n'
import type { Locale } from '@/lib/i18n/locale'

/**
 * Formate un timestamp ISO en libellé relatif court (FR/EN).
 *
 * - < 1 min  → "à l'instant" / "just now"
 * - < 60 min → "il y a N min" / "N min ago"
 * - < 24 h   → "il y a N h" / "N h ago"
 * - < 7 j    → "il y a N j" / "N d ago"
 * - sinon    → date locale (jj/mm/aaaa ou m/d/yyyy)
 */
export function formatRelative(iso: string, locale: Locale): string {
  const t = getNotificationsText(locale)
  const date = new Date(iso)
  const now = Date.now()
  const diffMs = now - date.getTime()
  const minutes = Math.round(diffMs / 60_000)
  if (minutes < 1) return t.relJustNow
  if (minutes < 60) return t.relMinutesAgo(minutes)
  const hours = Math.round(minutes / 60)
  if (hours < 24) return t.relHoursAgo(hours)
  const days = Math.round(hours / 24)
  if (days < 7) return t.relDaysAgo(days)
  return t.relOnDate(iso)
}
