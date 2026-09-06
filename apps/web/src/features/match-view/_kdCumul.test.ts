/**
 * _kdCumul.test.ts — la géométrie des FRAGS CUMULÉS, sur des oracles écrits à la main.
 *
 * CE QUE CES TESTS PROTÈGENT, ET QUI N'AVAIT AUCUN ORACLE JUSQU'ICI (registre 2026-09-05,
 * N1) : les deux cumuls et leur ordre, l'attribution du camp d'un fait marquant — un badge
 * de MORT appartient à l'équipe qui a fragué, pas à la victime —, le placement
 * anti-collision des pastilles, et l'élargissement des axes qui les laisse respirer.
 *
 * Chaque nombre attendu est recalculé à part, jamais recopié du module : l'écart de base
 * vaut `max(2,5 ; 0,18 × plus haut cumul)`, la hauteur d'une pastille `1,05 ×` cet écart, et
 * la fenêtre de voisinage `6 %` de l'étendue de l'axe des temps.
 */
import { describe, expect, it } from 'vitest'

import type { MatchHighlightEvent, MatchImpactBadge } from '@/lib/api/types'

import { buildKdCumul, KD_CUMUL_MIN_SPAN_MS, valueAtMs, type KdCumulInput } from './_kdCumul'
import type { XuidMeta } from './xuidMeta'

/**
 * DEUX ALLIÉS, UN ADVERSAIRE. `A` et `B` sont du camp du joueur de la page, `E` en face ;
 * `Z` n'est dans aucune ligne de scoreboard — ses frags ne comptent pour personne.
 */
const META: XuidMeta = new Map([
  ['A', { gamertag: 'Alpha', ally: true }],
  ['B', { gamertag: 'Bravo', ally: true }],
  ['E', { gamertag: 'Echo', ally: false }],
])

function kill(tMs: number, actor: string): MatchHighlightEvent {
  return { event_type: 'kill', actor_xuid: actor, event_time_ms: tMs } as MatchHighlightEvent
}

function badge(key: string, tMs: number, xuid: string): MatchImpactBadge {
  return { key, time_ms: tMs, player_xuid: xuid } as MatchImpactBadge
}

function input(over: Partial<KdCumulInput> = {}): KdCumulInput {
  return {
    // Trois frags : deux alliés (10 s et 30 s), un adverse (20 s).
    events: [kill(10_000, 'A'), kill(20_000, 'E'), kill(30_000, 'B')],
    badges: [],
    scoreboard: [],
    meXUID: 'A',
    xuidMeta: META,
    objectiveEvents: null,
    ...over,
  }
}

describe('buildKdCumul — quand le graphe existe, et quand il n’existe pas', () => {
  it('rend null sans event du tout', () => {
    expect(buildKdCumul(input({ events: [] }))).toBeNull()
    expect(buildKdCumul(input({ events: null }))).toBeNull()
  })

  it('rend null quand aucun event n’est un FRAG (médailles seules)', () => {
    const medailles = [{ event_type: 'medal', actor_xuid: 'A', event_time_ms: 1_000 }]
    expect(buildKdCumul(input({ events: medailles as MatchHighlightEvent[] }))).toBeNull()
  })

  it('rend null quand aucun tueur n’est rattaché à un camp — jamais un camp par défaut', () => {
    expect(buildKdCumul(input({ events: [kill(10_000, 'Z')] }))).toBeNull()
  })
})

describe('buildKdCumul — les deux cumuls', () => {
  it('compte chaque camp à part, dans l’ordre du temps', () => {
    const m = buildKdCumul(input())!
    expect(m.ally).toEqual([
      { tMs: 10_000, y: 1 },
      { tMs: 30_000, y: 2 },
    ])
    expect(m.enemy).toEqual([{ tMs: 20_000, y: 1 }])
  })

  it('TRIE les events, même servis à l’envers', () => {
    const m = buildKdCumul(input({ events: [kill(30_000, 'B'), kill(10_000, 'A')] }))!
    expect(m.ally.map((p) => p.tMs)).toEqual([10_000, 30_000])
  })

  it('IGNORE un tueur hors scoreboard — il ne compte pour aucun des deux', () => {
    const m = buildKdCumul(input({ events: [kill(10_000, 'A'), kill(15_000, 'Z')] }))!
    expect(m.ally).toEqual([{ tMs: 10_000, y: 1 }])
    expect(m.enemy).toEqual([])
  })

  it('garde une minute d’échelle sur un match plus court que ça', () => {
    expect(buildKdCumul(input())!.totalMs).toBe(KD_CUMUL_MIN_SPAN_MS)
    // Au-delà, l'axe suit le dernier frag.
    const long = buildKdCumul(input({ events: [kill(10_000, 'A'), kill(125_000, 'E')] }))!
    expect(long.totalMs).toBe(125_000)
  })
})

describe('valueAtMs — l’ancrage d’une pastille sur sa courbe', () => {
  const courbe = [
    { tMs: 10_000, y: 1 },
    { tMs: 30_000, y: 2 },
  ]

  it('rend 0 avant le premier frag : la courbe commence à zéro', () => {
    expect(valueAtMs(courbe, 0)).toBe(0)
    expect(valueAtMs(courbe, 9_999)).toBe(0)
  })

  it('rend la valeur DÈS l’instant du frag, et la MAINTIENT jusqu’au suivant', () => {
    expect(valueAtMs(courbe, 10_000)).toBe(1)
    expect(valueAtMs(courbe, 29_999)).toBe(1)
    expect(valueAtMs(courbe, 30_000)).toBe(2)
    expect(valueAtMs(courbe, 900_000)).toBe(2)
  })

  it('rend 0 sur une courbe vide, sans jamais lever', () => {
    expect(valueAtMs([], 10_000)).toBe(0)
  })
})

describe('buildKdCumul — le placement des pastilles', () => {
  /**
   * Le montage de référence : deux pastilles de FRAG à 1 s d'écart (donc en collision) et
   * une pastille de MORT. Plus haut cumul = 2, donc écart de base = max(2,5 ; 0,36) = 2,5 ;
   * hauteur d'une pastille = 2,625 ; fenêtre de voisinage = 6 % de 60 000 = 3 600 ms.
   */
  const avecBadges = () =>
    buildKdCumul(
      input({
        badges: [
          badge('first_blood', 10_000, 'A'),
          badge('top_gun', 11_000, 'B'),
          badge('first_group_death', 20_000, 'A'),
        ],
      }),
    )!

  it('ancre une pastille de FRAG sur la courbe de son tueur, au-dessus', () => {
    const b = avecBadges().badges.find((x) => x.key === 'first_blood')!
    expect(b.team).toBe('ally')
    expect(b.yAt).toBe(1) // le cumul allié vaut 1 à 10 s
    expect(b.yChip).toBeCloseTo(3.5, 6) // 1 + 2,5
    expect(b.tone).toBe('good') // `auto` : le fait profite au camp du joueur
    expect(b.label).toBe('⚡ Alpha')
  })

  it('ÉCARTE la voisine trop proche d’exactement une hauteur de pastille', () => {
    const b = avecBadges().badges.find((x) => x.key === 'top_gun')!
    expect(b.yAt).toBe(1)
    expect(b.yChip).toBeCloseTo(6.125, 6) // 3,5 + 2,625
  })

  it('range une pastille de MORT sur la courbe ADVERSE, en dessous', () => {
    // `first_group_death` date la mort d'un ALLIÉ : le frag revient à l'adversaire, et
    // c'est sur SA courbe que la pastille s'ancre.
    const b = avecBadges().badges.find((x) => x.key === 'first_group_death')!
    expect(b.team).toBe('enemy')
    expect(b.yAt).toBe(1) // le cumul adverse vaut 1 à 20 s
    expect(b.yChip).toBeCloseTo(-1.5, 6) // 1 − 2,5
    expect(b.tone).toBe('bad')
  })

  it('N’EN POSE AUCUNE pour un joueur hors scoreboard, ni pour une clé inconnue', () => {
    const m = buildKdCumul(
      input({
        badges: [badge('first_blood', 10_000, 'Z'), badge('cle_inconnue', 10_000, 'A')],
      }),
    )!
    expect(m.badges).toEqual([])
  })

  it('N’EN POSE AUCUNE sans instant mesuré, ni pour `kamikaze` (badge de match entier)', () => {
    const sansInstant = { key: 'first_blood', label: '', player_xuid: 'A' } as MatchImpactBadge
    const m = buildKdCumul(input({ badges: [sansInstant, badge('kamikaze', 10_000, 'A')] }))!
    expect(m.badges).toEqual([])
  })
})

describe('buildKdCumul — les bornes de l’axe vertical', () => {
  it('sans pastille : de 0 à 30 % au-dessus du plus haut cumul', () => {
    const m = buildKdCumul(input())!
    expect(m.yMin).toBe(0)
    expect(m.yMax).toBe(3) // ceil(max(2 × 1,3 ; 2 + 1)) = 3
  })

  it('MONTE au-dessus de la pastille la plus haute', () => {
    const m = buildKdCumul(
      input({ badges: [badge('first_blood', 10_000, 'A'), badge('top_gun', 11_000, 'B')] }),
    )!
    expect(m.yMax).toBe(9) // ceil(6,125 + 2,625) = ceil(8,75)
    expect(m.yMin).toBe(0) // aucune pastille sous la courbe
  })

  it('DESCEND sous zéro seulement quand une pastille de mort l’exige', () => {
    const m = buildKdCumul(input({ badges: [badge('first_group_death', 20_000, 'A')] }))!
    expect(m.yMin).toBe(-5) // floor(−1,5 − 2,625) = floor(−4,125)
  })
})
