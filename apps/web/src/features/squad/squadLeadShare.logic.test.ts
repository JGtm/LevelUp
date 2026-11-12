import { describe, expect, it } from 'vitest'

import type { SquadLeadSharePoint } from '@/lib/api/types'

import { squadLeadSharePoints } from './squadLeadShare.logic'

function pt(over: Partial<SquadLeadSharePoint>): SquadLeadSharePoint {
  return { match_id: 'm', start_time: '2026-11-03T20:00:00Z', outcome: 2, ...over }
}

describe('squadLeadSharePoints — un point par match, dans l’ordre du serveur', () => {
  it('numérote les matchs, garde la carte, la soirée, la part et les retournements', () => {
    const pts = squadLeadSharePoints([
      pt({ match_id: 'm1', map_name: 'Bazaar', session_label: '3 nov.', lead_share: 0.7, lead_changes: 2 }),
      pt({ match_id: 'm2', lead_share: 0, lead_changes: 0 }),
    ])
    expect(pts[0]).toMatchObject({ key: 'm1', label: '#1\nBazaar', share: 0.7, changes: 2, details: ['3 nov.'] })
    expect(pts[1]).toMatchObject({ key: 'm2', label: '#2', share: 0, changes: 0, details: [] })
  })

  it('un match non mesuré : part NULLE (la courbe se coupe), jamais zéro', () => {
    const [p] = squadLeadSharePoints([pt({ match_id: 'm3' })])
    expect(p.share).toBeNull()
    expect(p.changes).toBeNull()
  })

  it('absent ou null : aucun point', () => {
    expect(squadLeadSharePoints(undefined)).toEqual([])
    expect(squadLeadSharePoints(null)).toEqual([])
  })
})
