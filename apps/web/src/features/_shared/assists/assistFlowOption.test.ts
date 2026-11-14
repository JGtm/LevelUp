/**
 * assistFlowOption.test — l'option du diagramme de flux « assistant -> finisseur » : série
 * `sankey`, deux nœuds par joueur, nœud à la couleur du joueur, ruban à celle de l'assistant,
 * total dans l'étiquette du nœud, phrases à la voix active, rendu réel.
 */
import { describe, expect, it } from 'vitest'

import { renderOptionToSvg } from '@/components/charts/svgRender.testutil'

import { buildAssistFlow, type FlowPair } from './assistFlow.logic'
import { assistFlowHeight, buildAssistFlowOption, type AssistFlowChartOpts } from './assistFlowOption'
import { ASSIST_FLOW_TEXT } from './assistFlowStrings'

const pair = (assistant: string, finisher: string, count: number, stolen = 0): FlowPair => ({
  assistantKey: assistant,
  assistantName: assistant,
  finisherKey: finisher,
  finisherName: finisher,
  count,
  stolen,
})

const COLORS: Record<string, string> = { Madina: '#aa3322', JGtm: '#2255cc', Kaya: '#22aa55' }

const opts = (locale: 'fr' | 'en' = 'fr'): AssistFlowChartOpts => ({
  colorOf: (n) => COLORS[n],
  fmtShare: (v) => `${Math.round(v * 100)} %`,
  texts: { ...ASSIST_FLOW_TEXT[locale], share: (p) => `${p} des assistances de l’escouade` },
})

const PAIRS = [pair('Madina', 'JGtm', 14, 3), pair('JGtm', 'Madina', 9), pair('JGtm', 'Kaya', 6), pair('Kaya', 'JGtm', 6)]
const FLOW = buildAssistFlow(PAIRS, { assistantOrder: ['JGtm', 'Madina', 'Kaya'], finisherOrder: ['JGtm', 'Madina', 'Kaya'], total: 56 })

type Opt = {
  series: {
    type: string
    layoutIterations: number
    data: { name: string; value: number; itemStyle: { color: string }; label: { position: string; formatter: string }; tip: string }[]
    links: { source: string; target: string; value: number; lineStyle: { color: string }; tip: string }[]
  }[]
  tooltip: { formatter: (p: unknown) => string }
}
const build = () => buildAssistFlowOption(FLOW, opts()) as unknown as Opt

describe('buildAssistFlowOption', () => {
  it('sans ruban : option vide', () => {
    expect(buildAssistFlowOption({ nodes: [], links: [], total: 0 }, opts())).toEqual({ backgroundColor: 'transparent' })
  })

  it('une série sankey, ordre des nœuds = ordre des données, six nœuds (deux par joueur)', () => {
    const s = build().series[0]
    expect(s.type).toBe('sankey')
    expect(s.layoutIterations).toBe(0)
    expect(s.data.map((n) => n.name)).toEqual([
      'assistant:JGtm',
      'assistant:Madina',
      'assistant:Kaya',
      'finisher:JGtm',
      'finisher:Madina',
      'finisher:Kaya',
    ])
  })

  it('nœud à la couleur du joueur, le même des deux côtés ; total du nœud dans son étiquette', () => {
    const s = build().series[0]
    const byId = (id: string) => s.data.find((n) => n.name === id)!
    expect(byId('assistant:JGtm').itemStyle.color).toBe('#2255cc')
    expect(byId('finisher:JGtm').itemStyle.color).toBe('#2255cc')
    expect(byId('assistant:JGtm').label).toEqual({ position: 'left', formatter: 'JGtm · 15' })
    expect(byId('finisher:JGtm').label).toEqual({ position: 'right', formatter: 'JGtm · 20' })
  })

  it('ruban à la couleur de l’ASSISTANT, largeur = assistances', () => {
    const s = build().series[0]
    const l = s.links.find((x) => x.source === 'assistant:Madina' && x.target === 'finisher:JGtm')!
    expect(l.value).toBe(14)
    expect(l.lineStyle.color).toBe('#aa3322')
    const back = s.links.find((x) => x.source === 'assistant:JGtm' && x.target === 'finisher:Madina')!
    expect(back.lineStyle.color).toBe('#2255cc')
  })

  it('infobulle d’un ruban : « Madina a assisté JGtm sur 14 frags (25 % des assistances de l’escouade) », puis « dont 3 … »', () => {
    const o = build()
    const l = o.series[0].links.find((x) => x.source === 'assistant:Madina' && x.target === 'finisher:JGtm')!
    expect(o.tooltip.formatter({ data: l })).toBe(
      'Madina a assisté JGtm sur 14 frags (25 % des assistances de l’escouade)<br/>dont 3 où l’assistant a fait plus de dégâts que le finisseur',
    )
    // Pas de « dont » quand aucune assistance du couple n’est de ce type.
    const plain = o.series[0].links.find((x) => x.source === 'assistant:JGtm' && x.target === 'finisher:Madina')!
    expect(o.tooltip.formatter({ data: plain })).toBe('JGtm a assisté Madina sur 9 frags (16 % des assistances de l’escouade)')
  })

  it('sans total de référence, aucune part n’est dite', () => {
    const flow = buildAssistFlow(PAIRS)
    const o = buildAssistFlowOption(flow, opts()) as unknown as Opt
    expect(o.tooltip.formatter({ data: o.series[0].links[0] })).not.toContain('%')
  })

  it('infobulle d’un nœud : gauche « assistances données », droite « frags conclus sur une assistance »', () => {
    const o = build()
    const left = o.series[0].data.find((n) => n.name === 'assistant:JGtm')!
    const right = o.series[0].data.find((n) => n.name === 'finisher:JGtm')!
    expect(o.tooltip.formatter({ data: left })).toBe('JGtm : 15 assistances données')
    expect(o.tooltip.formatter({ data: right })).toBe('JGtm : 20 frags conclus sur une assistance')
  })

  it('jamais « a été assisté » : le sujet est toujours l’assistant (FR et EN)', () => {
    for (const locale of ['fr', 'en'] as const) {
      const o = buildAssistFlowOption(FLOW, opts(locale)) as unknown as Opt
      const all = o.series[0].links.map((l) => o.tooltip.formatter({ data: l })).join(' ')
      expect(all).not.toMatch(/été assisté|was assisted|assisted by/i)
    }
  })

  it('le gamertag est échappé', () => {
    const flow = buildAssistFlow([pair('<b>x</b>', 'JGtm', 2)])
    const o = buildAssistFlowOption(flow, { ...opts(), colorOf: () => '#000000' }) as unknown as Opt
    expect(o.tooltip.formatter({ data: o.series[0].links[0] })).toContain('&lt;b&gt;x&lt;/b&gt;')
  })

  it('hauteur : un pas par nœud de la colonne la plus chargée, plancher de 220', () => {
    expect(assistFlowHeight({ nodes: [], links: [], total: 0 })).toBe(220)
    expect(assistFlowHeight(FLOW)).toBe(Math.max(220, 24 + 3 * 34))
    const big = buildAssistFlow(Array.from({ length: 9 }, (_, i) => pair(`A${i}`, 'F', 1)))
    expect(assistFlowHeight(big)).toBe(24 + 9 * 34)
  })

  it('le rendu réel écrit les noms, les totaux et peint les couleurs des joueurs', () => {
    const svg = renderOptionToSvg(buildAssistFlowOption(FLOW, opts()), 800, assistFlowHeight(FLOW))
    expect(svg).toContain('Madina · 14')
    expect(svg).toContain('JGtm · 20')
    expect(svg.toLowerCase()).toContain('#aa3322')
    expect(svg.toLowerCase()).toContain('#2255cc')
  })
})
