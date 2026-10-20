/**
 * Tests du builder du calendrier : fenêtre aux bornes, colonnes de semaines, lundi en haut,
 * jours hors fenêtre à `null`, étiquettes uniques, fuseau, infobulle.
 */
import { describe, expect, it } from 'vitest'

import type { TrendsCalendarDay } from '@/lib/api/types'

import {
  buildCalendarGrid,
  formatCalendarTooltip,
  localDay,
  shiftDay,
  weekdayNames,
  type CalendarCellDetail,
  type CalendarInput,
} from './tendancesCalendar.logic'

/** Un lundi. */
const AS_OF = '2026-10-05T12:00:00Z'

function day(date: string, wins: number, losses: number, extra: Partial<TrendsCalendarDay> = {}) {
  const matches = wins + losses
  return { date, matches, wins, losses, win_rate: wins / matches, ...extra }
}

function input(overrides: Partial<CalendarInput> = {}): CalendarInput {
  return {
    calendar: [],
    asOf: AS_OF,
    timeZone: 'Europe/Paris',
    horizon: 7,
    locale: 'fr',
    ...overrides,
  }
}

const detail = (p: { detail?: unknown }) => p.detail as CalendarCellDetail

describe('buildCalendarGrid — fenêtre et colonnes', () => {
  it('7 j : du mardi 29/09 au lundi 05/10, deux colonnes de sept cases émises colonne par colonne', () => {
    const grid = buildCalendarGrid(input())
    expect(grid.columns).toBe(2)
    expect(grid.points).toHaveLength(14)
    expect(grid.points.slice(0, 7).map((p) => detail(p).date)).toEqual([
      '2026-09-28',
      '2026-09-29',
      '2026-09-30',
      '2026-10-01',
      '2026-10-02',
      '2026-10-03',
      '2026-10-04',
    ])
    expect(grid.points[7].y).toBe(grid.points[0].y)
    expect(grid.points.slice(0, 7).every((p) => p.x === grid.points[0].x)).toBe(true)
    expect(grid.points[7].x).not.toBe(grid.points[0].x)
  })

  it('lundi en tête : les sept lignes vont de lundi à dimanche', () => {
    const grid = buildCalendarGrid(input())
    const lignes = grid.points.slice(0, 7).map((p) => p.y)
    expect(lignes).toEqual(weekdayNames('fr'))
    expect(lignes[0]).toMatch(/^lun/)
    expect(lignes[6]).toMatch(/^dim/)
  })

  it('étiquette d’une colonne : le lundi en « jour mois court »', () => {
    const grid = buildCalendarGrid(input())
    expect(grid.points[0].x).toMatch(/^28 sept/)
    expect(grid.points[7].x).toMatch(/^5 oct/)
  })

  it('jour joué dans la fenêtre : son taux de victoire ; jour non joué, avant ou après : null', () => {
    const grid = buildCalendarGrid(
      input({ calendar: [day('2026-10-01', 3, 1), day('2026-09-20', 1, 1), day('2026-09-28', 1, 0)] }),
    )
    const byDate = Object.fromEntries(grid.points.map((p) => [detail(p).date, p.value]))
    expect(byDate['2026-10-01']).toBe(0.75)
    expect(byDate['2026-09-29']).toBeNull() // dans la fenêtre, non joué
    expect(byDate['2026-09-28']).toBeNull() // joué, mais un jour AVANT la fenêtre
    expect(byDate['2026-10-06']).toBeNull() // après as_of
    expect(grid.playedDays).toBe(1)
  })

  it('bornes : le jour de début et le jour de fin sont dans la fenêtre', () => {
    const grid = buildCalendarGrid(input({ calendar: [day('2026-09-29', 1, 0), day('2026-10-05', 0, 1)] }))
    const byDate = Object.fromEntries(grid.points.map((p) => [detail(p).date, p.value]))
    expect(byDate['2026-09-29']).toBe(1)
    expect(byDate['2026-10-05']).toBe(0)
    expect(grid.playedDays).toBe(2)
  })

  it('taux absent de la réponse : victoires sur matchs', () => {
    const grid = buildCalendarGrid(
      input({ calendar: [{ date: '2026-10-01', matches: 4, wins: 1, losses: 3 }] }),
    )
    expect(grid.points.find((p) => detail(p).date === '2026-10-01')?.value).toBe(0.25)
  })

  it('calendrier nul ou vide : aucun jour joué, toutes les cases à null', () => {
    for (const calendar of [null, undefined, []]) {
      const grid = buildCalendarGrid(input({ calendar }))
      expect(grid.playedDays).toBe(0)
      expect(grid.points.every((p) => p.value === null)).toBe(true)
    }
  })

  it('365 j : 53 semaines, jamais plus de douze étiquettes affichées', () => {
    const grid = buildCalendarGrid(input({ horizon: 365 }))
    expect(grid.columns).toBe(53)
    expect(grid.points).toHaveLength(53 * 7)
    expect(Math.ceil(grid.columns / (grid.labelInterval + 1))).toBeLessThanOrEqual(12)
  })

  it('7 j : toutes les étiquettes sont affichées (interval 0)', () => {
    expect(buildCalendarGrid(input()).labelInterval).toBe(0)
  })

  it('étiquettes uniques ; deux lundis de même jour et mois portent l’année', () => {
    const grid = buildCalendarGrid(input({ horizon: 3000 }))
    const colonnes = grid.points.filter((_, i) => i % 7 === 0).map((p) => p.x)
    expect(new Set(colonnes).size).toBe(grid.columns)
    expect(colonnes.filter((x) => /2020/.test(x) && /^5 oct/.test(x))).toHaveLength(1)
    expect(colonnes.filter((x) => /2026/.test(x) && /^5 oct/.test(x))).toHaveLength(1)
  })
})

describe('buildCalendarGrid — fuseau', () => {
  it('as_of tardif en UTC : le jour local de Paris est déjà le lendemain', () => {
    expect(localDay('2026-10-05T23:30:00Z', 'Europe/Paris')).toBe('2026-10-06')
    expect(localDay('2026-10-05T23:30:00Z', 'America/New_York')).toBe('2026-10-05')
    const paris = buildCalendarGrid(input({ asOf: '2026-10-05T23:30:00Z' }))
    const ny = buildCalendarGrid(input({ asOf: '2026-10-05T23:30:00Z', timeZone: 'America/New_York' }))
    // Paris : fenêtre du 30/09 au 06/10 ; New York : du 29/09 au 05/10.
    expect(paris.points.map((p) => detail(p).date)).toContain('2026-10-11')
    expect(paris.columns).toBe(2)
    expect(ny.columns).toBe(2)
  })

  it('un jour joué le 06/10 compte pour Paris, pas pour New York', () => {
    const calendar = [day('2026-10-06', 1, 0)]
    const asOf = '2026-10-05T23:30:00Z'
    expect(buildCalendarGrid(input({ asOf, calendar })).playedDays).toBe(1)
    expect(buildCalendarGrid(input({ asOf, calendar, timeZone: 'America/New_York' })).playedDays).toBe(0)
  })

  it('shiftDay franchit les fins de mois et d’année', () => {
    expect(shiftDay('2026-03-01', -1)).toBe('2026-02-28')
    expect(shiftDay('2026-12-31', 1)).toBe('2027-01-01')
  })
})

describe('formatCalendarTooltip', () => {
  const played = (extra: Partial<CalendarCellDetail> = {}) => ({
    x: 'x',
    y: 'y',
    value: 0.75,
    detail: {
      date: '2026-10-05',
      played: true,
      matches: 4,
      wins: 3,
      losses: 1,
      winRate: 0.75,
      performance: 52.34,
      ...extra,
    },
  })

  it('date longue, matchs, victoires, défaites, taux et score de performance', () => {
    const html = formatCalendarTooltip(played(), 'fr')
    expect(html).toContain('5 octobre 2026')
    expect(html).toContain('4 matchs')
    expect(html).toContain('Victoires : 3')
    expect(html).toContain('Défaites : 1')
    expect(html).toContain('Taux de victoire : 75 %')
    expect(html).toContain('Score de performance : 52,3')
  })

  it('sans score de performance : la ligne n’existe pas', () => {
    expect(formatCalendarTooltip(played({ performance: undefined }), 'fr')).not.toContain(
      'Score de performance',
    )
  })

  it('case non jouée : pas d’infobulle', () => {
    expect(
      formatCalendarTooltip(
        { x: 'x', y: 'y', value: null, detail: { date: '2026-10-05', played: false } },
        'fr',
      ),
    ).toBe('')
  })

  it('anglais', () => {
    const html = formatCalendarTooltip(played(), 'en')
    expect(html).toContain('October 5, 2026')
    expect(html).toContain('Wins: 3')
  })
})
