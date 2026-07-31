/**
 * coverageLogic.test.ts — l'écran doit dire ce qu'il ne montre pas.
 *
 * Ces tests portent sur des PROPRIÉTÉS, pas sur des valeurs figées : un taux qui ne vaut jamais
 * 1 quand il n'y a rien, une somme qui doit tomber juste, un pont qui n'est « lu » que s'il
 * l'est entièrement. Ce sont les trois façons dont un écran peut mentir sans qu'on le voie.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayDocument, ReplayLayerCoverage } from '@/lib/api/types'

import { bridgeIsRead, isBalanced, ratioOf, summarizeAll } from './coverageLogic'

const layer = (o: Partial<ReplayLayerCoverage>): ReplayLayerCoverage => ({
  available: 0,
  attached: 0,
  noSlot: 0,
  ambiguous: 0,
  outOfWindow: 0,
  unpublished: 0,
  ...o,
})

describe('ratioOf', () => {
  it('rend 0 — et jamais 1 — quand il n’y a rien à rattacher', () => {
    // « rien à rattacher » n'est pas « tout rattaché » : afficher 100 % sur un calque vide
    // serait le mensonge le plus facile à commettre.
    expect(ratioOf(layer({ available: 0, attached: 0 }))).toBe(0)
  })

  it('rend le rapport rattachés / disponibles', () => {
    expect(ratioOf(layer({ available: 519, attached: 475 }))).toBeCloseTo(0.9152, 4)
  })
})

describe('isBalanced', () => {
  it('accepte un comptage dont la somme fait le total', () => {
    expect(isBalanced(layer({ available: 519, attached: 475, noSlot: 44 }))).toBe(true)
  })

  it('refuse un comptage qui perd des événements en route', () => {
    // 475 + 20 = 495, pas 519 : 24 événements ont disparu sans cause nommée. C'est exactement
    // le défaut que la couverture existe pour rendre visible.
    expect(isBalanced(layer({ available: 519, attached: 475, noSlot: 20 }))).toBe(false)
  })
})

describe('summarizeAll', () => {
  it('rend une liste vide quand l’artefact ne porte pas de couverture', () => {
    // Les artefacts construits avant cette version n'ont pas le champ : l'écran doit se taire,
    // pas inventer un dénominateur.
    expect(summarizeAll({ coverage: undefined } as ReplayDocument)).toEqual([])
  })

  it('marque nominal uniquement si le verdict l’est ET que la somme tombe juste', () => {
    const doc = {
      coverage: {
        shots: layer({ available: 519, attached: 475, noSlot: 44 }),
        grenades: layer({ available: 70, attached: 70 }),
        verdict: { shots: 'nominal', grenades: 'nominal' },
        bridge: {
          slots: 90,
          fromReading: 90,
          livesNamed: 90,
          livesTotal: 105,
          indexReadings: 26,
          indexDisagreements: 0,
          slotCollisions: 0,
        },
      },
    } as unknown as ReplayDocument
    const out = summarizeAll(doc)
    expect(out).toHaveLength(2)
    expect(out[0].nominal).toBe(true)
    expect(out[0].rejects).toEqual([{ cause: 'noSlot', n: 44 }])
    expect(out[1].rejects).toEqual([])
  })

  it('refuse « nominal » quand le serveur le dit mais que le comptage fuit', () => {
    // Le verdict vient du serveur ; la somme se vérifie ici. Si les deux se contredisent,
    // l'écran suit le comptage — un verdict ne rattrape pas une fuite.
    const doc = {
      coverage: {
        shots: layer({ available: 519, attached: 475, noSlot: 10 }),
        grenades: layer({ available: 0 }),
        verdict: { shots: 'nominal' },
        bridge: {
          slots: 0,
          fromReading: 0,
          livesNamed: 0,
          livesTotal: 0,
          indexReadings: 0,
          indexDisagreements: 0,
          slotCollisions: 0,
        },
      },
    } as unknown as ReplayDocument
    expect(summarizeAll(doc)[0].nominal).toBe(false)
  })
})

describe('bridgeIsRead', () => {
  const bridge = (o: Record<string, number>) =>
    ({
      coverage: {
        shots: layer({}),
        grenades: layer({}),
        bridge: {
          slots: 90,
          fromReading: 90,
          livesNamed: 90,
          livesTotal: 105,
          indexReadings: 26,
          indexDisagreements: 0,
          slotCollisions: 0,
          ...o,
        },
      },
    }) as unknown as ReplayDocument

  it('est vrai quand tout le pont vient de la lecture', () => {
    expect(bridgeIsRead(bridge({}))).toBe(true)
  })

  it('est faux dès qu’une autre source alimente le pont', () => {
    // Un écart entre `slots` et `fromReading` signifie qu'une méthode autre que la lecture a
    // nommé des traces. Le rejeu en a retiré une le 2026-07-28 ; ce test garde la porte fermée.
    expect(bridgeIsRead(bridge({ fromReading: 84 }))).toBe(false)
  })

  it('est faux si une identité est lue de deux façons', () => {
    expect(bridgeIsRead(bridge({ indexDisagreements: 1 }))).toBe(false)
  })

  it('est faux si une trace change de porteur', () => {
    expect(bridgeIsRead(bridge({ slotCollisions: 1 }))).toBe(false)
  })

  it('est faux sans pont du tout', () => {
    expect(bridgeIsRead(bridge({ slots: 0 }))).toBe(false)
    expect(bridgeIsRead(undefined)).toBe(false)
  })
})
