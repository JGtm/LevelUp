/**
 * tendancesCalendar.logic — le builder PUR du « Calendrier des résultats » : de la liste des
 * jours joués aux cases du wrapper `Heatmap2DChart`, semaines en colonnes et jours en lignes.
 *
 * FENÊTRE. Du jour local de `as_of − (horizon − 1) jours` au jour local de `as_of`, dans le
 * fuseau de la réponse. Les dates se manipulent en chaînes « AAAA-MM-JJ » et en jours
 * civils (UTC pur) : aucune heure d'été n'entre dans l'arithmétique.
 *
 * COLONNES. Une par semaine (lundi en tête) qui recoupe la fenêtre ; LIGNES : lundi en haut
 * (le wrapper est monté avec `yAxisInverse`). Une case vaut le taux de victoire du jour (0..1)
 * si le jour est joué ET dans la fenêtre, `null` sinon.
 *
 * TOUTES LES CASES SONT ÉMISES, colonne par colonne : le wrapper déduit ses axes de l'ordre
 * d'apparition des points, en omettre une décalerait les catégories.
 */
import type { ChartPointHeatmap } from '@/components/charts/Heatmap2DChart'
import { escapeHtml } from '@/components/charts/_utils'
import { intlLocale } from '@/lib/formatters'
import type { TrendsCalendarDay } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'

import { getTendancesText } from './i18n'
import { formatTrendValue } from './tendances.logic'

const DAY_MS = 24 * 60 * 60 * 1000
const DAYS_PER_WEEK = 7
/** Une douzaine d'étiquettes de colonnes visibles au plus. */
const MAX_VISIBLE_LABELS = 12

export interface CalendarInput {
  calendar: readonly TrendsCalendarDay[] | null | undefined
  /** Instant de référence de la réponse (`as_of`). */
  asOf: string | Date
  /** Fuseau IANA de la réponse. */
  timeZone: string
  horizon: number
  locale: Locale
}

export interface CalendarGrid {
  points: ChartPointHeatmap[]
  /** Nombre de colonnes (semaines). */
  columns: number
  /** Jours joués dans la fenêtre : 0 = rien à tracer. */
  playedDays: number
  /** Écart d'étiquettes (`axisLabel.interval`) qui garde une douzaine d'étiquettes au plus. */
  labelInterval: number
}

/** Détail d'une case, lu par l'infobulle. */
export interface CalendarCellDetail extends Record<string, unknown> {
  date: string
  played: boolean
  matches: number
  wins: number
  losses: number
  winRate?: number
  performance?: number
}

/** Jour civil « AAAA-MM-JJ » de `instant` dans `timeZone`. */
export function localDay(instant: string | Date, timeZone: string): string {
  const parts = new Intl.DateTimeFormat('en-CA', {
    timeZone,
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  }).formatToParts(new Date(instant))
  const get = (type: string) => parts.find((p) => p.type === type)?.value ?? ''
  return `${get('year')}-${get('month')}-${get('day')}`
}

function dayToMs(day: string): number {
  const [y, m, d] = day.split('-').map(Number)
  return Date.UTC(y, m - 1, d)
}

function msToDay(ms: number): string {
  return new Date(ms).toISOString().slice(0, 10)
}

/** `day` décalé de `delta` jours civils. */
export function shiftDay(day: string, delta: number): string {
  return msToDay(dayToMs(day) + delta * DAY_MS)
}

/** Lundi de la semaine de `day` (0 = lundi ... 6 = dimanche pour le décalage). */
function mondayOf(day: string): string {
  const jour = new Date(dayToMs(day)).getUTCDay() // 0 = dimanche
  return shiftDay(day, -((jour + 6) % DAYS_PER_WEEK))
}

/** Noms courts des jours, lundi en tête, dans la locale. */
export function weekdayNames(locale: Locale): string[] {
  const fmt = new Intl.DateTimeFormat(intlLocale(locale), { weekday: 'short', timeZone: 'UTC' })
  // 2024-01-01 est un lundi.
  return Array.from({ length: DAYS_PER_WEEK }, (_, i) =>
    fmt.format(new Date(Date.UTC(2024, 0, 1 + i))),
  )
}

/** Étiquettes des lundis : « jour mois court » ; l'année s'ajoute à celles qui se répètent. */
function columnLabels(mondays: readonly string[], locale: Locale): string[] {
  const loc = intlLocale(locale)
  const short = new Intl.DateTimeFormat(loc, { day: 'numeric', month: 'short', timeZone: 'UTC' })
  const withYear = new Intl.DateTimeFormat(loc, {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
    timeZone: 'UTC',
  })
  const courtes = mondays.map((d) => short.format(new Date(dayToMs(d))))
  const comptes = new Map<string, number>()
  for (const label of courtes) comptes.set(label, (comptes.get(label) ?? 0) + 1)
  return mondays.map((d, i) =>
    (comptes.get(courtes[i]) ?? 0) > 1 ? withYear.format(new Date(dayToMs(d))) : courtes[i],
  )
}

function winRateOf(day: TrendsCalendarDay): number | null {
  if (day.win_rate != null) return day.win_rate
  return day.matches > 0 ? day.wins / day.matches : null
}

/** Construit la grille du calendrier. */
export function buildCalendarGrid(input: CalendarInput): CalendarGrid {
  const end = localDay(input.asOf, input.timeZone)
  const start = shiftDay(end, -(input.horizon - 1))
  const played = new Map<string, TrendsCalendarDay>()
  for (const d of input.calendar ?? []) {
    if (d.date >= start && d.date <= end && d.matches > 0) played.set(d.date, d)
  }

  const firstMonday = mondayOf(start)
  const lastMonday = mondayOf(end)
  const columns = Math.round((dayToMs(lastMonday) - dayToMs(firstMonday)) / (DAYS_PER_WEEK * DAY_MS)) + 1
  const mondays = Array.from({ length: columns }, (_, i) => shiftDay(firstMonday, i * DAYS_PER_WEEK))
  const labels = columnLabels(mondays, input.locale)
  const rows = weekdayNames(input.locale)

  const points: ChartPointHeatmap[] = []
  mondays.forEach((monday, c) => {
    rows.forEach((row, r) => {
      const date = shiftDay(monday, r)
      const day = played.get(date)
      const detail: CalendarCellDetail = day
        ? {
            date,
            played: true,
            matches: day.matches,
            wins: day.wins,
            losses: day.losses,
            winRate: winRateOf(day) ?? undefined,
            performance: day.performance_score,
          }
        : { date, played: false, matches: 0, wins: 0, losses: 0 }
      points.push({ x: labels[c], y: row, value: day ? winRateOf(day) : null, detail })
    })
  })

  return {
    points,
    columns,
    playedDays: played.size,
    labelInterval: Math.max(0, Math.ceil(columns / MAX_VISIBLE_LABELS) - 1),
  }
}

/** Infobulle d'une case jouée (texte échappé) ; vide pour une case sans match. */
export function formatCalendarTooltip(point: ChartPointHeatmap, locale: Locale): string {
  const d = point.detail as CalendarCellDetail | undefined
  if (!d || !d.played) return ''
  const t = getTendancesText(locale)
  const date = new Intl.DateTimeFormat(intlLocale(locale), {
    dateStyle: 'full',
    timeZone: 'UTC',
  }).format(new Date(dayToMs(d.date)))
  const lignes = [`<b>${escapeHtml(date)}</b>`, escapeHtml(t.matchCount(d.matches))]
  lignes.push(escapeHtml(t.calendarWins(d.wins)), escapeHtml(t.calendarLosses(d.losses)))
  if (d.winRate != null) {
    lignes.push(escapeHtml(t.calendarWinRate(formatTrendValue(d.winRate, 'ratio', 2, locale))))
  }
  if (d.performance != null) {
    lignes.push(
      escapeHtml(t.calendarPerformance(formatTrendValue(d.performance, 'number', 1, locale))),
    )
  }
  return lignes.join('<br/>')
}
