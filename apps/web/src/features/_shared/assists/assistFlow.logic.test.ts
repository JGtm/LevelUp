/**
 * assistFlow.logic.test — les décisions pures du diagramme de flux « assistant -> finisseur » :
 * deux nœuds par joueur (un par côté), totaux de nœud, rubans, parts sur le total donné, ordre
 * des colonnes.
 */
import { describe, expect, it } from 'vitest'

import { buildAssistFlow, flowNodeId, type FlowPair } from './assistFlow.logic'

const pair = (assistant: string, finisher: string, count: number, stolen = 0): FlowPair => ({
  assistantKey: assistant,
  assistantName: assistant,
  finisherKey: finisher,
  finisherName: finisher,
  count,
  stolen,
})

const PAIRS = [pair('Madina', 'JGtm', 14, 3), pair('JGtm', 'Madina', 9), pair('JGtm', 'Kaya', 6), pair('Kaya', 'JGtm', 6)]

describe('buildAssistFlow', () => {
  it('aucune paire : aucun nœud ni ruban, total nul', () => {
    expect(buildAssistFlow([])).toEqual({ nodes: [], links: [], total: 0 })
    expect(buildAssistFlow([pair('A', 'B', 0)])).toEqual({ nodes: [], links: [], total: 0 })
  })

  it('chaque joueur a un nœud de chaque côté, aux identifiants distincts', () => {
    const { nodes } = buildAssistFlow(PAIRS)
    const jg = nodes.filter((n) => n.key === 'JGtm')
    expect(jg.map((n) => n.side).sort()).toEqual(['assistant', 'finisher'])
    expect(new Set(nodes.map((n) => n.id)).size).toBe(nodes.length)
    expect(jg.map((n) => n.id)).toContain(flowNodeId('assistant', 'JGtm'))
    expect(jg.map((n) => n.id)).toContain(flowNodeId('finisher', 'JGtm'))
  })

  it('total d’un nœud : assistances données (gauche) ou frags conclus sur une assistance (droite)', () => {
    const { nodes } = buildAssistFlow(PAIRS)
    const total = (side: string, key: string) => nodes.find((n) => n.side === side && n.key === key)?.total
    expect(total('assistant', 'JGtm')).toBe(15) // 9 + 6
    expect(total('finisher', 'JGtm')).toBe(20) // 14 + 6
    expect(total('assistant', 'Madina')).toBe(14)
    expect(total('finisher', 'Madina')).toBe(9)
  })

  it('un ruban par couple : largeur = assistances, éliminations volées conservées', () => {
    const { links } = buildAssistFlow(PAIRS)
    const l = links.find((x) => x.assistantKey === 'Madina' && x.finisherKey === 'JGtm')!
    expect(l).toMatchObject({ count: 14, stolen: 3, source: 'assistant:Madina', target: 'finisher:JGtm' })
    expect(links).toHaveLength(4)
  })

  it('deux paires du même couple s’additionnent', () => {
    const { links, total } = buildAssistFlow([pair('A', 'B', 2, 1), pair('A', 'B', 3, 1)])
    expect(links).toHaveLength(1)
    expect(links[0]).toMatchObject({ count: 5, stolen: 2 })
    expect(total).toBe(5)
  })

  it('parts sur le TOTAL DONNÉ (serveur), jamais sur la somme des rubans ; sans total, aucune part', () => {
    const withTotal = buildAssistFlow(PAIRS, { total: 100 })
    expect(withTotal.total).toBe(100)
    expect(withTotal.links.find((l) => l.assistantKey === 'Madina')!.share).toBe(0.14)
    const without = buildAssistFlow(PAIRS)
    expect(without.total).toBe(35)
    expect(without.links.every((l) => l.share === null)).toBe(true)
    expect(buildAssistFlow(PAIRS, { total: 0 }).links.every((l) => l.share === null)).toBe(true)
  })

  it('colonnes dans l’ordre donné, les inconnus à la suite par total décroissant', () => {
    const flow = buildAssistFlow([...PAIRS, pair('Dave', 'JGtm', 1), pair('Eve', 'JGtm', 4)], {
      assistantOrder: ['JGtm', 'Madina'],
      finisherOrder: ['Kaya'],
    })
    const left = flow.nodes.filter((n) => n.side === 'assistant').map((n) => n.key)
    expect(left).toEqual(['JGtm', 'Madina', 'Kaya', 'Eve', 'Dave'].filter((k) => left.includes(k)))
    expect(left.slice(0, 2)).toEqual(['JGtm', 'Madina'])
    // Inconnus : Kaya (6) puis Eve (4) puis Dave (1).
    expect(left.slice(2)).toEqual(['Kaya', 'Eve', 'Dave'])
    const right = flow.nodes.filter((n) => n.side === 'finisher').map((n) => n.key)
    expect(right[0]).toBe('Kaya')
  })

  it('les rubans suivent l’ordre des nœuds : tracé déterministe', () => {
    const a = buildAssistFlow(PAIRS, { assistantOrder: ['Madina', 'JGtm', 'Kaya'] })
    const b = buildAssistFlow([...PAIRS].reverse(), { assistantOrder: ['Madina', 'JGtm', 'Kaya'] })
    expect(a.links.map((l) => l.source + l.target)).toEqual(b.links.map((l) => l.source + l.target))
  })

  it('deux joueurs de même nom mais de clés distinctes restent distincts', () => {
    const flow = buildAssistFlow([
      { ...pair('Bot', 'X', 2), assistantKey: 'gt:Bot' },
      { ...pair('Bot', 'X', 3), assistantKey: 'xuid1' },
    ])
    expect(flow.nodes.filter((n) => n.side === 'assistant')).toHaveLength(2)
  })
})
