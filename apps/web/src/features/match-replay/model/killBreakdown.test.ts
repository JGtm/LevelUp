/**
 * killBreakdown.test.ts — LA RÉPARTITION DES ÉLIMINATIONS D'UN JOUEUR À L'IMAGE LUE.
 *
 * Un match écrit à la main : Alpha fait huit éliminations au fil du temps (deux armes d'épaule, une arme
 * lourde à deux armes, une grenade, une mêlée sans arme, deux éliminations sans classe servie). Chaque frag
 * porte la classe que le serveur a servie sur lui, avec la clé d'arme du registre du titre. Une élimination de
 * Bravo et une de bot ne comptent que pour leur propre joueur.
 */
import { describe, expect, it } from 'vitest'

import type { FragClassKey } from '@/lib/accessibility/scales/fragClass'
import type { ReplayPlayer } from '@/lib/replay/rosterLogic'

import type { ReplayFeedEntry, ReplayKill } from './killFeedLogic'
import { killBreakdownAt, readKillBreakdownSeries, type KillBreakdown } from './killBreakdown'

const STEP_MS = 100
const DOC = { frameIntervalMs: STEP_MS }

const ALPHA: ReplayPlayer = { xuid: 'A', lives: [] }
const BOT: ReplayPlayer = { xuid: 'bot:Marc', bot: true, lives: [] }

/** Un frag du fil : `weaponClass` est celle que le serveur a servie sur l'élimination (absente = non établie). */
function killEntry(
  replayMs: number,
  killer: string,
  weaponKey: string,
  weaponClass?: FragClassKey,
): ReplayFeedEntry {
  const kill = { xuid: killer, weaponKey, weaponClass, replayMs, medals: [] } as unknown as ReplayKill
  return { key: `k${replayMs}`, replayMs, kill, medal: null, death: null }
}

/** Alpha : huit éliminations de 1 s à 8 s, puis une mêlée à 9 s. Bravo et le bot tuent aussi. */
const FEED: ReplayFeedEntry[] = [
  killEntry(1_000, 'A', 'hinf_ma40_ar', 'shoulder'),
  killEntry(1_500, 'B', 'hinf_ma40_ar', 'shoulder'),
  killEntry(2_000, 'A', 'hinf_ma40_ar', 'shoulder'),
  killEntry(3_000, 'A', 'hinf_s7_sniper', 'heavy'),
  killEntry(4_000, 'A', 'hinf_frag_grenade', 'grenade'),
  killEntry(5_000, 'A', ''),
  killEntry(6_000, 'A', 'hinf_arme_hors_table'),
  killEntry(7_000, 'A', 'hinf_gravity_hammer', 'heavy'),
  killEntry(8_000, 'A', 'hinf_ma40_ar', 'shoulder'),
  killEntry(9_000, 'A', '', 'melee'),
  killEntry(1_200, 'bot:Marc', 'hinf_ma40_ar', 'shoulder'),
]

function lire(frame: number, feed: ReplayFeedEntry[] | null = FEED, player: ReplayPlayer = ALPHA): KillBreakdown | null {
  const series = readKillBreakdownSeries({ doc: DOC, player, feed })
  return killBreakdownAt(series, frame)
}

function classe(b: KillBreakdown | null, key: FragClassKey) {
  const found = b?.classes.find((c) => c.classKey === key)
  if (!found) throw new Error(`classe ${key} absente`)
  return found
}

/** Invariants : classes plus non classé = total, armes d'une classe = compte de la classe. */
function verifierSommes(b: KillBreakdown): void {
  const parClasse = b.classes.reduce((s, c) => s + c.atInstant, 0)
  const nonClasse = b.unclassified?.atInstant ?? 0
  expect(parClasse + nonClasse).toBe(b.totals.atInstant)
  for (const c of b.classes) {
    expect(c.weapons.reduce((s, w) => s + w.atInstant, 0)).toBe(c.atInstant)
    expect(c.weapons.reduce((s, w) => s + w.atMatch, 0)).toBe(c.atMatch)
  }
  const parClasseMatch = b.classes.reduce((s, c) => s + c.atMatch, 0)
  expect(parClasseMatch + (b.unclassified?.atMatch ?? 0)).toBe(b.totals.atMatch)
}

describe('killBreakdown — début de match', () => {
  const b = lire(0)

  it('les classes du match sont listées à zéro, sans part', () => {
    expect(b).not.toBeNull()
    expect(b?.totals).toEqual({ atInstant: 0, atMatch: 9 })
    for (const c of b?.classes ?? []) {
      expect(c.atInstant).toBe(0)
      expect(c.share).toBeNull()
      for (const w of c.weapons) {
        expect(w.atInstant).toBe(0)
        expect(w.share).toBeNull()
      }
    }
  })

  it('la catégorie non classée est présente à zéro à l\'instant, sans part', () => {
    expect(b?.unclassified).toEqual({ atInstant: 0, atMatch: 2, share: null })
  })

  it('les sommes tiennent', () => {
    verifierSommes(b as KillBreakdown)
  })
})

describe('killBreakdown — milieu de match', () => {
  // Frame 60 = 6 s : huit éliminations ont eu lieu ou plus (1 s à 6 s : six éliminations d'Alpha).
  const b = lire(60) as KillBreakdown

  it('le total à l\'instant compte les six éliminations d\'Alpha, le bot et Bravo exclus', () => {
    expect(b.totals).toEqual({ atInstant: 6, atMatch: 9 })
  })

  it('l\'épaule, l\'arme lourde et la grenade ont leur compte et leur part du total à l\'instant', () => {
    expect(classe(b, 'shoulder')).toMatchObject({ atInstant: 2, atMatch: 3, share: 2 / 6 })
    expect(classe(b, 'heavy')).toMatchObject({ atInstant: 1, atMatch: 2, share: 1 / 6 })
    expect(classe(b, 'grenade')).toMatchObject({ atInstant: 1, atMatch: 1, share: 1 / 6 })
  })

  it('la mêlée, à 9 s, est encore listée à zéro', () => {
    expect(classe(b, 'melee')).toMatchObject({ atInstant: 0, atMatch: 1, share: 0 })
  })

  it('une arme encore à zéro à l\'instant reste listée dans sa classe', () => {
    const lourdes = classe(b, 'heavy').weapons
    expect(lourdes.map((w) => [w.weaponKey, w.atInstant, w.atMatch])).toEqual([
      ['hinf_gravity_hammer', 0, 1],
      ['hinf_s7_sniper', 1, 1],
    ])
  })

  it('la catégorie non classée compte les deux éliminations sans arme établie', () => {
    expect(b.unclassified).toEqual({ atInstant: 2, atMatch: 2, share: 2 / 6 })
  })

  it('les sommes tiennent', () => {
    verifierSommes(b)
  })
})

describe('killBreakdown — fin de match', () => {
  const b = lire(100) as KillBreakdown

  it('à l\'instant, chaque compte égale celui du match', () => {
    expect(b.totals).toEqual({ atInstant: 9, atMatch: 9 })
    for (const c of b.classes) {
      expect(c.atInstant).toBe(c.atMatch)
      for (const w of c.weapons) expect(w.atInstant).toBe(w.atMatch)
    }
    expect(b.unclassified?.atInstant).toBe(b.unclassified?.atMatch)
  })

  it('les parts somment à 100 %', () => {
    const parts = [...b.classes.map((c) => c.share), b.unclassified?.share]
    expect(parts.reduce<number>((s, p) => s + (p ?? 0), 0)).toBeCloseTo(1, 10)
  })

  it('classes par compte au match décroissant, à égalité par identifiant', () => {
    expect(b.classes.map((c) => c.classKey)).toEqual(['shoulder', 'heavy', 'grenade', 'melee'])
  })

  it('les armes d\'une classe suivent le même ordre', () => {
    expect(classe(b, 'heavy').weapons.map((w) => w.weaponKey)).toEqual(['hinf_gravity_hammer', 'hinf_s7_sniper'])
    expect(classe(b, 'shoulder').weapons.map((w) => [w.weaponKey, w.atMatch])).toEqual([['hinf_ma40_ar', 3]])
  })

  it('les sommes tiennent', () => {
    verifierSommes(b)
  })
})

describe('killBreakdown — mêlée, grenade et arme lourde', () => {
  it('la mêlée est classée à part, sa part vaut 1 sur 9 à 9 s', () => {
    expect(classe(lire(90) as KillBreakdown, 'melee')).toMatchObject({ atInstant: 1, share: 1 / 9 })
  })

  it("la mêlée sans arme est une entrée d'arme à clé vide dans sa classe", () => {
    const b = lire(100) as KillBreakdown
    expect(classe(b, 'melee').weapons).toEqual([{ weaponKey: '', atInstant: 1, atMatch: 1, share: 1 / 9 }])
  })

  it("une classe servie sur une arme hors du registre range quand même l'élimination, sous sa clé", () => {
    const feed = [killEntry(1_000, 'A', 'hinf_arme_inconnue', 'sidearm')]
    const b = lire(100, feed) as KillBreakdown
    expect(classe(b, 'sidearm').weapons.map((w) => w.weaponKey)).toEqual(['hinf_arme_inconnue'])
    expect(b.unclassified).toBeNull()
  })

  it('la grenade est classée à part, par sa clé d\'arme', () => {
    const b = lire(40) as KillBreakdown
    expect(classe(b, 'grenade').weapons).toEqual([{ weaponKey: 'hinf_frag_grenade', atInstant: 1, atMatch: 1, share: 1 / 4 }])
  })

  it('l\'arme lourde regroupe toutes ses armes de la classe heavy', () => {
    const b = lire(100) as KillBreakdown
    expect(classe(b, 'heavy').atMatch).toBe(2)
  })
})

describe('killBreakdown — arme non établie et bot', () => {
  it('une élimination sans classe servie, avec ou sans clé d\'arme, tombe dans la catégorie non classée', () => {
    const b = lire(70) as KillBreakdown
    expect(b.unclassified).toEqual({ atInstant: 2, atMatch: 2, share: 2 / 7 })
    expect(b.classes.flatMap((c) => c.weapons).map((w) => w.weaponKey)).not.toContain('hinf_arme_hors_table')
  })

  it('un bot compte pour ses propres éliminations, pas pour celles d\'Alpha', () => {
    const b = lire(15, FEED, BOT) as KillBreakdown
    expect(b.totals).toEqual({ atInstant: 1, atMatch: 1 })
    expect(classe(b, 'shoulder')).toMatchObject({ atInstant: 1, atMatch: 1, share: 1 })
  })

  it('les sommes tiennent pour le bot', () => {
    verifierSommes(lire(15, FEED, BOT) as KillBreakdown)
  })
})

describe('killBreakdown — absence', () => {
  it('fil absent : null, jamais zéro', () => {
    expect(lire(50, null)).toBeNull()
  })

  it('fil sans aucune élimination : null', () => {
    expect(lire(50, [])).toBeNull()
  })

  it('joueur sans éliminations dans un fil qui en porte : totaux à zéro, aucune classe', () => {
    const b = lire(100, FEED, { xuid: 'Z', lives: [] }) as KillBreakdown
    expect(b).toEqual({ classes: [], unclassified: null, totals: { atInstant: 0, atMatch: 0 } })
  })
})
