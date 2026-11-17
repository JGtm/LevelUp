/**
 * Tests — planFeed : ce que le fil posé sur le plan montre. Les cinq dernières éliminations déjà survenues, de la
 * plus ancienne à la plus récente ; rien d'autre qu'une élimination ; l'estompage des deux plus anciennes
 * seulement quand le fil est plein.
 */
import { describe, expect, it } from 'vitest'

import { COTES_FIL_PLAN } from './cotesFilPlan'
import type { ReplayDeath, ReplayFeedEntry, ReplayKill } from './killFeedLogic'
import { dernieresEliminations, opaciteLigneFilPlan } from './planFeed'

function elimination(i: number): ReplayFeedEntry {
  const replayMs = i * 1_000
  const kill = { tMs: replayMs, replayMs, xuid: 'k', victimXuid: 'v', medals: [] } as unknown as ReplayKill
  return { key: `k${i}`, replayMs, kill, medal: null, death: null }
}

/** `n` éliminations aux instants 1 s, 2 s, ... n s. */
function eliminations(n: number): ReplayFeedEntry[] {
  return Array.from({ length: n }, (_, i) => elimination(i + 1))
}

const FIN = 1_000_000
const cles = (lignes: readonly ReplayFeedEntry[]) => lignes.map((l) => l.key)

describe('dernieresEliminations : combien de lignes', () => {
  it('aucune élimination passée : rien', () => {
    expect(dernieresEliminations([], FIN)).toEqual([])
    expect(dernieresEliminations(eliminations(4), 500)).toEqual([])
  })

  it('3 éliminations passées : 3 lignes', () => {
    expect(cles(dernieresEliminations(eliminations(3), FIN))).toEqual(['k1', 'k2', 'k3'])
  })

  it('5 éliminations passées : 5 lignes', () => {
    expect(cles(dernieresEliminations(eliminations(5), FIN))).toEqual(['k1', 'k2', 'k3', 'k4', 'k5'])
  })

  it('8 éliminations passées : les 5 dernières', () => {
    expect(cles(dernieresEliminations(eliminations(8), FIN))).toEqual(['k4', 'k5', 'k6', 'k7', 'k8'])
  })

  it('le nombre de lignes est celui de la cote', () => {
    expect(dernieresEliminations(eliminations(20), FIN)).toHaveLength(COTES_FIL_PLAN.lignes)
  })
})

describe('dernieresEliminations : ordre et instant lu', () => {
  it('la plus récente est la DERNIÈRE de la liste (affichée en bas)', () => {
    const lignes = dernieresEliminations(eliminations(8), FIN)
    expect(lignes[lignes.length - 1].key).toBe('k8')
    expect(lignes[0].replayMs).toBeLessThan(lignes[lignes.length - 1].replayMs)
  })

  it('suit l\'instant lu : les éliminations postérieures n\'y sont pas, celle à l\'instant exact y est', () => {
    const fil = eliminations(8)
    expect(cles(dernieresEliminations(fil, 6_000))).toEqual(['k2', 'k3', 'k4', 'k5', 'k6'])
    expect(cles(dernieresEliminations(fil, 5_999))).toEqual(['k1', 'k2', 'k3', 'k4', 'k5'])
    expect(cles(dernieresEliminations(fil, 2_000))).toEqual(['k1', 'k2'])
  })

  it('un retour en arrière rend les mêmes lignes que la première fois (aucun état gardé)', () => {
    const fil = eliminations(8)
    const avant = cles(dernieresEliminations(fil, 7_000))
    dernieresEliminations(fil, 3_000)
    expect(cles(dernieresEliminations(fil, 7_000))).toEqual(avant)
  })

  it('ne modifie pas le fil de la page', () => {
    const fil = eliminations(8)
    const copie = [...fil]
    dernieresEliminations(fil, FIN)
    expect(fil).toEqual(copie)
  })
})

describe('dernieresEliminations : seulement des éliminations', () => {
  it("une médaille seule, une mort sans tueur, une entrée de partie et un événement n'y entrent pas, et ne prennent pas de place", () => {
    const mort = { xuid: 'x', replayMs: 7_500 } as unknown as ReplayDeath
    const autres: ReplayFeedEntry[] = [
      { key: 'm', replayMs: 7_200, kill: null, medal: { tMs: 7_200 } as never, death: null },
      { key: 'd', replayMs: 7_500, kill: null, medal: null, death: mort },
      { key: 'p', replayMs: 7_600, kill: null, medal: null, death: null, presence: { xuid: 'x' } as never },
      { key: 'e', replayMs: 7_700, kill: null, medal: null, death: null, event: { family: 'objectives' } as never },
    ]
    const fil = [...eliminations(8), ...autres].sort((a, b) => a.replayMs - b.replayMs)
    const lignes = dernieresEliminations(fil, FIN)
    expect(cles(lignes)).toEqual(['k4', 'k5', 'k6', 'k7', 'k8'])
    expect(cles(dernieresEliminations(autres, FIN))).toEqual([])
  })
})

describe('opaciteLigneFilPlan : l\'estompage des deux plus anciennes', () => {
  it('fil plein (5 lignes) : 70 % pour la plus ancienne, 85 % pour la suivante, 100 % pour les autres', () => {
    expect([0, 1, 2, 3, 4].map((i) => opaciteLigneFilPlan(i, 5))).toEqual([0.7, 0.85, 1, 1, 1])
  })

  it('fil de moins de 5 lignes : aucun estompage', () => {
    for (const total of [1, 2, 3, 4]) {
      for (let i = 0; i < total; i++) expect(opaciteLigneFilPlan(i, total)).toBe(1)
    }
  })

  it('les opacités sont celles des cotes', () => {
    expect(COTES_FIL_PLAN.estompage).toEqual([0.7, 0.85])
    expect(COTES_FIL_PLAN.lignes).toBe(5)
  })
})
