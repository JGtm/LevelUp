import { describe, expect, it } from 'vitest'

import type { LeadSharePeriodPoint } from '@/lib/api/types'
import { LEAD_SHARE_TEXT } from '@/lib/charts/leadShare/text'

import { formatMonthShort, leadShareMonthPoints } from './timeseriesLeadShare.logic'

const MOIS: LeadSharePeriodPoint[] = [
  { month: '2026-09', start_date: '2026-09-01', lead_share: 0.63, measured_matches: 26, matches: 30 },
  { month: '2026-10', start_date: '2026-10-01', measured_matches: 0, matches: 4 },
]

describe('leadShareMonthPoints — une colonne par mois', () => {
  it('garde l’ordre, la moyenne mesurée et le nombre de matchs mesurés sur le total', () => {
    const [sept] = leadShareMonthPoints(MOIS, LEAD_SHARE_TEXT.fr, 'fr')
    expect(sept.key).toBe('2026-09')
    expect(sept.share).toBe(0.63)
    expect(sept.details).toEqual(['26 matchs sur 30 avec un chiffre'])
  })

  it('un mois sans match mesuré : part NULLE (pointillé), jamais zéro', () => {
    const [, oct] = leadShareMonthPoints(MOIS, LEAD_SHARE_TEXT.fr, 'fr')
    expect(oct.share).toBeNull()
  })

  it('absent ou null : aucune colonne', () => {
    expect(leadShareMonthPoints(undefined, LEAD_SHARE_TEXT.fr, 'fr')).toEqual([])
    expect(leadShareMonthPoints(null, LEAD_SHARE_TEXT.fr, 'fr')).toEqual([])
  })
})

describe('formatMonthShort', () => {
  it('suit la langue du shell, en UTC (pas de décalage de fuseau)', () => {
    expect(formatMonthShort('2026-01-01', 'fr')).toMatch(/janv\.? 26/)
    expect(formatMonthShort('2026-01-01', 'en')).toMatch(/Jan 26|Jan ’26|Jan 2026/)
  })
})
