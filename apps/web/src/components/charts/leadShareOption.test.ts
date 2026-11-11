/**
 * leadShareOption.test.ts — l'option ECharts commune de la Mainmise : axe normalisé 0..100 %,
 * repère à 50 %, colonne en pointillé pour un match non mesuré (jamais zéro), infobulle qui dit la
 * raison du tiret, couleurs par jetons.
 */
import { describe, expect, it, vi } from 'vitest'

import type { LeadSharePoint } from '@/lib/charts/leadShare/model'
import { LEAD_SHARE_TEXT } from '@/lib/charts/leadShare/text'

vi.mock('@/lib/accessibility', () => ({
  resolveToken: (token: string) => `var(--ac-${token})`,
  tokenCssVar: (token: string) => `var(--ac-${token})`,
}))

const { buildLeadShareOption, leadShareTooltipLines } = await import('./leadShareOption')

const text = LEAD_SHARE_TEXT.fr

const POINTS: LeadSharePoint[] = [
  { key: 'a', label: '#1\nBazaar', share: 0.78, changes: 2 },
  { key: 'b', label: '#2\nRecharge', share: null },
  { key: 'c', label: '#3\nPrism', share: 0, changes: 0 },
]

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function option(variant: 'bars' | 'line' = 'bars', points = POINTS): Record<string, any> {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  return buildLeadShareOption(points, { text, variant }) as Record<string, any>
}

describe('buildLeadShareOption — l’axe est NORMALISÉ', () => {
  it('va toujours de 0 à 100 %, gradué tous les 25, et un seul axe des ordonnées', () => {
    const o = option()
    expect(Array.isArray(o.yAxis)).toBe(false)
    expect(o.yAxis).toMatchObject({ type: 'value', min: 0, max: 100, interval: 25 })
    expect(o.yAxis.axisLabel.formatter(50)).toBe('50 %')
  })

  it('pose le repère pointillé à 50 % sur la série de la part', () => {
    const o = option()
    expect(o.series[0].markLine.data).toEqual([{ yAxis: 50 }])
    expect(o.series[0].markLine.lineStyle.type).toBe('dashed')
  })

  it('met la légende en bas', () => {
    expect(option().legend.bottom).toBe(0)
  })
})

describe('buildLeadShareOption — un match non mesuré n’est PAS un zéro', () => {
  it('colonnes : la part est nulle (aucune barre), la colonne en pointillé occupe toute la hauteur', () => {
    const o = option('bars')
    expect(o.series[0].data).toEqual([78, null, 0])
    expect(o.series[1].data).toEqual([null, 100, null])
    expect(o.series[1].itemStyle).toMatchObject({ color: 'transparent', borderType: 'dashed' })
    // Une part mesurée à ZÉRO reste une barre de hauteur nulle, pas un trou.
    expect(o.series[0].data[2]).toBe(0)
  })

  it('courbe : la ligne se COUPE sur un match non mesuré, sans série de pointillé', () => {
    const o = option('line')
    expect(o.series).toHaveLength(1)
    expect(o.series[0].type).toBe('line')
    expect(o.series[0].connectNulls).toBe(false)
    expect(o.series[0].data).toEqual([78, null, 0])
  })

  it('sans aucun match non mesuré : pas de série ni de légende de pointillé', () => {
    const o = option('bars', [POINTS[0], POINTS[2]])
    expect(o.series).toHaveLength(1)
    expect(o.legend.data.map((e: { name: string }) => e.name)).toEqual([text.name])
  })

  it('la légende nomme la colonne en pointillé quand elle existe', () => {
    expect(option('bars').legend.data.map((e: { name: string }) => e.name)).toEqual([
      text.name,
      text.notMeasuredLegend,
    ])
  })

  it('aucun point : option vide', () => {
    expect(buildLeadShareOption([], { text, variant: 'bars' })).toEqual({ backgroundColor: expect.anything() })
  })
})

describe('buildLeadShareOption — couleurs', () => {
  it('l’équipe du joueur pour la part, le neutre pour le repère : des jetons, jamais un hex', () => {
    const o = option()
    expect(o.series[0].itemStyle.color).toBe('var(--ac-team-ally)')
    expect(o.series[0].markLine.lineStyle.color).toBe('var(--ac-divergent-neutral)')
  })
})

describe('leadShareTooltipLines — l’infobulle', () => {
  it('un match mesuré : part en pourcentage entier et retournements', () => {
    const lines = leadShareTooltipLines(POINTS[0], text)
    expect(lines[0]).toBe('<strong>#1 · Bazaar</strong>')
    expect(lines).toContain('Mainmise : <b>78 %</b>')
    expect(lines).toContain('2 retournements')
  })

  it('un match non mesuré : la raison du tiret, aucun pourcentage', () => {
    const lines = leadShareTooltipLines(POINTS[1], text)
    expect(lines).toContain(text.notMeasured)
    expect(lines.join('')).not.toMatch(/%/)
  })

  it('échappe les libellés de la donnée', () => {
    const lines = leadShareTooltipLines({ key: 'x', label: '<img src=x>', share: 0.5, details: ['<b>'] }, text)
    expect(lines.join('')).not.toContain('<img')
    expect(lines.join('')).toContain('&lt;img src=x&gt;')
    expect(lines.join('')).toContain('&lt;b&gt;')
  })
})
