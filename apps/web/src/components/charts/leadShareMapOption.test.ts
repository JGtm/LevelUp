/**
 * leadShareMapOption.test.ts — « Mainmise par carte » : sélection contre historique, axe normalisé
 * 0..100 %, cartes sans mesure écartées, infobulle avec le nombre de matchs mesurés.
 */
import { describe, expect, it, vi } from 'vitest'

import type { MapBreakdownRow } from '@/lib/api/types'
import { LEAD_SHARE_TEXT } from '@/lib/charts/leadShare/text'

vi.mock('@/lib/accessibility', () => ({
  resolveToken: (token: string) => `var(--ac-${token})`,
  tokenCssVar: (token: string) => `var(--ac-${token})`,
}))

const { buildLeadShareMapOption, leadShareMapRows } = await import('./leadShareMapOption')

const text = LEAD_SHARE_TEXT.fr

function row(map_ui: string, over: Partial<MapBreakdownRow> = {}): MapBreakdownRow {
  return { map_ui, match_count: 5, win_rate: 0.5, ...over }
}

const ROWS: MapBreakdownRow[] = [
  row('Bazaar', { lead_share: 0.55, lead_share_matches: 4, historical_lead_share: 0.5, historical_lead_share_matches: 20 }),
  row('Recharge'), // aucune mesure : écartée
  row('Streets', { historical_lead_share: 0.6, historical_lead_share_matches: 1 }), // historique seul
]

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function option(rows = ROWS): Record<string, any> {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  return buildLeadShareMapOption(rows, { text, mapLabelOf: (m) => m.toUpperCase(), scopeLabel: 'Session', historyLabel: 'Historique' }) as Record<string, any>
}

describe('leadShareMapRows — les cartes retenues', () => {
  it('écarte une carte sans aucune mesure, garde l’ordre reçu', () => {
    expect(leadShareMapRows(ROWS).map((r) => r.map_ui)).toEqual(['Bazaar', 'Streets'])
  })

  it('au plus vingt cartes, les plus jouées', () => {
    const many = Array.from({ length: 25 }, (_, i) =>
      row(`Carte ${i}`, { match_count: i + 1, lead_share: 0.5, lead_share_matches: 1 }),
    )
    const kept = leadShareMapRows(many)
    expect(kept).toHaveLength(20)
    expect(kept.map((r) => r.map_ui)).not.toContain('Carte 0')
  })
})

describe('buildLeadShareMapOption', () => {
  it('axe normalisé de 0 à 100 %, repère pointillé à 50 %, légende en bas', () => {
    const o = option()
    expect(o.xAxis).toMatchObject({ type: 'value', min: 0, max: 100, interval: 25 })
    expect(o.series[0].markLine.data).toEqual([{ xAxis: 50 }])
    expect(o.legend.bottom).toBe(0)
    expect(o.legend.data.map((e: { name: string }) => e.name)).toEqual(['Historique', 'Session'])
  })

  it('une valeur absente reste ABSENTE (null), jamais zéro', () => {
    const o = option()
    expect(o.series[0].data).toEqual([50, 60]) // historique
    expect(o.series[1].data).toEqual([55, null]) // sélection : Streets sans mesure
  })

  it('l’infobulle donne la part et le nombre de matchs mesurés de chaque barre', () => {
    const o = option()
    const html = o.tooltip.formatter([{ dataIndex: 0 }]) as string
    expect(html).toContain('BAZAAR')
    expect(html).toContain('Session : <b>55 %</b> (4 matchs)')
    expect(html).toContain('Historique : <b>50 %</b> (20 matchs)')
    const streets = o.tooltip.formatter([{ dataIndex: 1 }]) as string
    expect(streets).not.toContain('Session')
    expect(streets).toContain('Historique : <b>60 %</b> (1 match)')
  })

  it('aucune carte mesurée : option vide', () => {
    expect(option([row('Recharge')])).toEqual({ backgroundColor: expect.anything() })
  })
})
