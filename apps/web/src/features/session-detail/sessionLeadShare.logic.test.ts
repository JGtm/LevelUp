import { describe, expect, it } from 'vitest'

import type { SessionCompareEntry, SessionDetailMatchRow } from '@/lib/api/types'
import { LEAD_SHARE_TEXT } from '@/lib/charts/leadShare/text'

import { sessionLeadSharePoints, sessionLeadShareSummary } from './sessionLeadShare.logic'

function rows(over: Array<Partial<SessionDetailMatchRow>>): SessionDetailMatchRow[] {
  return over as unknown as SessionDetailMatchRow[]
}

describe('sessionLeadSharePoints — une colonne par match', () => {
  it('garde l’ordre des matchs, la part mesurée et les retournements', () => {
    const pts = sessionLeadSharePoints(
      rows([
        { match_id: 'm1', map_name: 'Bazaar', lead_share: 0.78, lead_changes: 2, mode_ui: 'Assassin' },
        { match_id: 'm2', map_name: 'Recharge', lead_share: 0.41, lead_changes: 0 },
      ]),
    )
    expect(pts.map((p) => p.key)).toEqual(['m1', 'm2'])
    expect(pts[0]).toMatchObject({ share: 0.78, changes: 2, details: ['Assassin'] })
    expect(pts[0].label).toBe('#1\nBazaar')
    // Zéro retournement MESURÉ reste un zéro.
    expect(pts[1].changes).toBe(0)
  })

  it('un match non mesuré a une part NULLE (colonne en pointillé), jamais zéro', () => {
    const [p] = sessionLeadSharePoints(rows([{ match_id: 'm3', map_name: 'Aquarius' }]))
    expect(p.share).toBeNull()
    expect(p.changes).toBeNull()
  })

  it('une part mesurée à zéro reste zéro : c’est une mesure', () => {
    const [p] = sessionLeadSharePoints(rows([{ match_id: 'm4', lead_share: 0, lead_changes: 0 }]))
    expect(p.share).toBe(0)
  })
})

describe('sessionLeadShareSummary — la moyenne des matchs mesurés', () => {
  const entry = (lead_share: number | undefined, lead_share_matches: number | undefined) =>
    ({ lead_share, lead_share_matches }) as unknown as SessionCompareEntry

  it('FR : pourcentage entier et nombre de matchs mesurés', () => {
    expect(sessionLeadShareSummary(entry(0.54, 5), LEAD_SHARE_TEXT.fr)).toBe('Moyenne 54 % sur 5 matchs mesurés')
  })

  it('singulier d’un seul match mesuré, et EN', () => {
    expect(sessionLeadShareSummary(entry(0.5, 1), LEAD_SHARE_TEXT.fr)).toBe('Moyenne 50 % sur 1 match mesuré')
    expect(sessionLeadShareSummary(entry(0.5, 3), LEAD_SHARE_TEXT.en)).toBe('Average 50 % over 3 measured matches')
  })

  it('aucun match mesuré : pas de ligne (ni « 0 % », ni « N non mesurés »)', () => {
    expect(sessionLeadShareSummary(entry(undefined, undefined), LEAD_SHARE_TEXT.fr)).toBeNull()
    expect(sessionLeadShareSummary(null, LEAD_SHARE_TEXT.fr)).toBeNull()
  })
})
