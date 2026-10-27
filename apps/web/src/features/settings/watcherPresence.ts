/**
 * Formatters de présence du Watcher — extraits de WatcherCard.tsx pour que le
 * module de composant n'exporte que des composants
 * (react-refresh/only-export-components).
 */
import { intlLocale } from '@/lib/formatters'
import type { SettingsText } from '@/features/settings/i18n'
import { DEFAULT_LOCALE, type Locale } from '@/lib/i18n/locale'

/**
 * Mappe les titleNames Xbox spéciaux vers leurs labels UI. Xbox utilise
 * "Online" comme titleName du Dashboard, ce qui donnerait "Vu il y a 2h
 * sur Online" — peu parlant. On remap vers "l'accueil Xbox" / "the Xbox
 * home" pour clarifier.
 *
 * Tout autre titleName est passé tel quel (Halo Infinite, CS2, etc.).
 */
export function resolveTitleDisplayName(titleName: string, t: SettingsText): string {
  if (titleName === 'Online') return t.watcherTitleXboxDashboard
  return titleName
}

/**
 * Format un "vu il y a X sur Y" lisible côté UI.
 *
 * Logique :
 *  - < 60 s     → "à l'instant" / "just now"
 *  - < 60 min   → "5 min"  / "5 min ago"
 *  - < 24 h     → "3 h"    / "3 hr ago"
 *  - < 7 j      → "2 j"    / "2 days ago"
 *  - sinon      → format date absolu localisé
 *
 * Le timestamp en entrée est en RFC3339 UTC (renvoyé par l'API Go).
 * `now` est injectable pour faciliter les tests.
 */
export function formatLastSeen(
  timestamp: string,
  titleName: string,
  t: SettingsText,
  locale: Locale = DEFAULT_LOCALE,
  now: Date = new Date(),
): string {
  const past = new Date(timestamp)
  if (Number.isNaN(past.getTime())) {
    return t.watcherNeverSeen
  }
  const diffMs = now.getTime() - past.getTime()
  const diffMin = Math.floor(diffMs / 60_000)
  const diffH = Math.floor(diffMs / 3_600_000)
  const diffD = Math.floor(diffMs / 86_400_000)

  let duration: string
  if (diffMin < 1) {
    duration = t.watcherDurationUnderMin
  } else if (diffMin < 60) {
    duration = `${diffMin} min`
  } else if (diffH < 24) {
    duration = t.watcherDurationHours.replace('{n}', String(diffH))
  } else if (diffD < 7) {
    duration = (diffD > 1 ? t.watcherDurationDayMany : t.watcherDurationDayOne).replace('{n}', String(diffD))
  } else {
    // Format absolu pour les dates anciennes.
    const date = past.toLocaleDateString(intlLocale(locale), {
      day: 'numeric',
      month: 'short',
      year: 'numeric',
    })
    return t.watcherLastSeenAbsolute
      .replace('{date}', date)
      .replace('{title}', titleName)
  }

  return t.watcherLastSeenRelative
    .replace('{duration}', duration)
    .replace('{title}', titleName)
}
