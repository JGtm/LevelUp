import { describe, expect, it } from 'vitest'

import { FOLLOW_DELAY_MS } from './constants'
import { FOLLOW_INITIAL, stepFollow, type FollowContext, type FollowSample, type FollowState } from './follow'
import { reliefData, type FloorRect } from './reliefScene.testutil'

const FRAME_MS = 33
const meshOf = (rects: FloorRect[]) => reliefData({ width: 100, height: 100, rects }).floors.mesh

/** Un sol bas partout (monde de 10 × 10 u) et un étage sur la moitié droite (x de 5 à 10). */
const TWO_FLOORS: FollowContext = {
  mesh: meshOf([
    { i0: 0, j0: 0, i1: 99, j1: 99, z: 0 },
    { i0: 50, j0: 0, i1: 99, j1: 99, z: 2.2 },
  ]),
  levels: [0, 2.2],
}
/** Une rampe de 0 à 2,2 u sur dix unités. */
const RAMP: FollowContext = {
  mesh: meshOf([{ i0: 0, j0: 0, i1: 99, j1: 99, z: [0, 2.2, 2.2, 0] }]),
  levels: [0, 2.2],
}

interface Change {
  atMs: number
  cut: number | null
}

/** Rejoue des échantillons ; rend l'état final et chaque changement de la coupe. */
function run(ctx: FollowContext, samples: FollowSample[], start: FollowState = FOLLOW_INITIAL) {
  const changes: Change[] = []
  let state = start
  for (const sample of samples) {
    const next = stepFollow(state, sample, ctx)
    if (next.cut !== state.cut) changes.push({ atMs: sample.atMs, cut: next.cut })
    state = next
  }
  return { state, changes }
}

/** Un joueur vivant : `at(t)` donne sa position et sa hauteur à l'instant `t` (ms). */
function life(
  lifeKey: number,
  fromMs: number,
  toMs: number,
  at: (t: number) => { x: number; y?: number; z?: number | null },
): FollowSample[] {
  const out: FollowSample[] = []
  for (let t = fromMs; t < toMs; t += FRAME_MS) {
    const p = at(t)
    out.push({ alive: true, lifeKey, x: p.x, y: p.y ?? 5, z: p.z, atMs: t })
  }
  return out
}

const dead = (fromMs: number, toMs: number): FollowSample[] => {
  const out: FollowSample[] = []
  for (let t = fromMs; t < toMs; t += FRAME_MS) out.push({ alive: false, lifeKey: 0, x: 0, y: 0, atMs: t })
  return out
}

describe('suivi du joueur ouvert : la coupe prend l’étage sous ses pieds', () => {
  it('à la première image d’une vie, la coupe prend l’étage du sol porteur, sans délai', () => {
    const { changes } = run(TWO_FLOORS, life(1, 0, 100, () => ({ x: 7, z: 2.2 })))
    expect(changes).toEqual([{ atMs: 0, cut: 2.2 }])
  })

  it('marche à plat : aucun changement', () => {
    const { changes } = run(TWO_FLOORS, life(1, 0, 5000, (t) => ({ x: 1 + t / 1500, z: 0 })))
    expect(changes).toEqual([{ atMs: 0, cut: 0 }])
  })

  it('saut sur place de 0,9 u : aucun changement, au sol comme sur l’étage', () => {
    const jump = (z0: number, x: number) => (t: number) => ({ x, z: z0 + 0.9 * Math.sin((Math.PI * (t % 1000)) / 1000) })
    expect(run(TWO_FLOORS, life(1, 0, 4000, jump(0, 2))).changes).toEqual([{ atMs: 0, cut: 0 }])
    expect(run(TWO_FLOORS, life(1, 0, 4000, jump(2.2, 7))).changes).toEqual([{ atMs: 0, cut: 2.2 }])
  })

  it('chute d’un étage : un changement, 300 ms après le passage du bord', () => {
    // Sur l'étage jusqu'à 1 s, puis vers la gauche à 2 u/s : le bord est en x = 5. Les pieds ne tombent qu'ensuite.
    const at = (t: number) => ({ x: t < 1000 ? 7 : 7 - (2 * (t - 1000)) / 1000, z: 2.2 })
    const samples = life(1, 0, 4000, at)
    const crossing = samples.find((s) => s.x < 5)!.atMs
    const { changes } = run(TWO_FLOORS, samples)
    expect(changes.map((c) => c.cut)).toEqual([2.2, 0])
    const delay = changes[1].atMs - crossing
    expect(delay).toBeGreaterThanOrEqual(FOLLOW_DELAY_MS)
    expect(delay).toBeLessThan(FOLLOW_DELAY_MS + FRAME_MS)
  })

  it('un écart plus court que 300 ms ne change rien, et la minuterie repart de zéro', () => {
    // 200 ms au-dessus du sol bas (x = 4), puis retour sur l'étage, deux fois de suite : jamais 300 ms d'affilée.
    const at = (t: number) => ({ x: Math.floor(t / 200) % 2 === 1 ? 4 : 7, z: 2.2 })
    const { changes } = run(TWO_FLOORS, life(1, 0, 4000, at))
    expect(changes).toEqual([{ atMs: 0, cut: 2.2 }])
  })

  it('rampe de 2,2 u : deux à trois changements, croissants, jamais d’aller-retour', () => {
    const { changes } = run(RAMP, life(1, 0, 10_000, (t) => ({ x: t / 1000, z: 0.22 * (t / 1000) })))
    const cuts = changes.map((c) => c.cut as number)
    const afterStart = cuts.slice(1)
    expect(afterStart.length).toBeGreaterThanOrEqual(2)
    expect(afterStart.length).toBeLessThanOrEqual(3)
    expect([...cuts].sort((a, b) => a - b)).toEqual(cuts)
    expect(cuts[cuts.length - 1]).toBe(2.2)
  })

  it('un point hors maillage garde le dernier sol porteur de la vie', () => {
    const outside = life(1, 1000, 4000, () => ({ x: 50, y: 50, z: 2.2 }))
    const { state, changes } = run(TWO_FLOORS, [...life(1, 0, 1000, () => ({ x: 7, z: 2.2 })), ...outside])
    expect(changes).toEqual([{ atMs: 0, cut: 2.2 }])
    expect(state.lastFloor).toBeCloseTo(2.2, 2)
  })

  it('première image d’une vie hors maillage et sans sol connu : la coupe prend la hauteur des pieds', () => {
    const { changes } = run(TWO_FLOORS, life(1, 0, 100, () => ({ x: 50, y: 50, z: 1.3 })))
    expect(changes).toEqual([{ atMs: 0, cut: 1.25 }])
  })

  it('z absent vaut zéro', () => {
    for (const z of [undefined, null]) {
      const { changes } = run(TWO_FLOORS, life(1, 0, 100, () => ({ x: 7, z })))
      expect(changes).toEqual([{ atMs: 0, cut: 0 }])
    }
  })
})

describe('suivi du joueur ouvert : mort et réapparition', () => {
  it('à la mort l’étage reste ; à la réapparition à un autre étage, un changement immédiat', () => {
    const samples = [
      ...life(1, 0, 1000, () => ({ x: 2, z: 0 })),
      ...dead(1000, 6000),
      ...life(2, 6000, 7000, () => ({ x: 7, z: 2.2 })),
    ]
    const { changes, state } = run(TWO_FLOORS, samples)
    const respawn = samples.find((s) => s.lifeKey === 2)!.atMs
    expect(changes).toEqual([
      { atMs: 0, cut: 0 },
      { atMs: respawn, cut: 2.2 },
    ])
    expect(state.cut).toBe(2.2)
  })

  it('pendant la mort, la coupe ne bouge pas, quelle que soit la durée', () => {
    const samples = [...life(1, 0, 500, () => ({ x: 7, z: 2.2 })), ...dead(500, 60_000)]
    const { changes, state } = run(TWO_FLOORS, samples)
    expect(changes).toEqual([{ atMs: 0, cut: 2.2 }])
    expect(state.cut).toBe(2.2)
    expect(state.lifeKey).toBeNull()
  })

  it('réapparaître au même étage ne change pas la coupe', () => {
    const samples = [...life(1, 0, 500, () => ({ x: 7, z: 2.2 })), ...dead(500, 2000), ...life(2, 2000, 2500, () => ({ x: 8, z: 2.2 }))]
    expect(run(TWO_FLOORS, samples).changes).toEqual([{ atMs: 0, cut: 2.2 }])
  })

  it('un autre numéro de vie est une première image, même sans échantillon de mort entre les deux', () => {
    const samples = [...life(1, 0, 500, () => ({ x: 2, z: 0 })), ...life(2, 500, 1000, () => ({ x: 7, z: 2.2 }))]
    expect(run(TWO_FLOORS, samples).changes.map((c) => c.cut)).toEqual([0, 2.2])
  })
})

describe('suivi du joueur ouvert : saut de lecture et cas limites', () => {
  const onGround = life(1, 0, 500, () => ({ x: 2, z: 0 }))

  it('un saut de lecture (clic sur la barre) pose la coupe sans délai', () => {
    const jump: FollowSample = { alive: true, lifeKey: 1, x: 7, y: 5, z: 2.2, atMs: 500 + FRAME_MS, seek: true }
    expect(run(TWO_FLOORS, [...onGround, jump]).changes.map((c) => c.cut)).toEqual([0, 2.2])
  })

  it('un instant qui recule, ou qui avance de plus d’une seconde, est un saut de lecture', () => {
    const back: FollowSample = { alive: true, lifeKey: 1, x: 7, y: 5, z: 2.2, atMs: 100 }
    expect(run(TWO_FLOORS, [...onGround, back]).changes.map((c) => c.cut)).toEqual([0, 2.2])
    const far: FollowSample = { alive: true, lifeKey: 1, x: 7, y: 5, z: 2.2, atMs: 60_000 }
    expect(run(TWO_FLOORS, [...onGround, far]).changes.map((c) => c.cut)).toEqual([0, 2.2])
  })

  it('sans maillage il n’y a pas de coupe : l’état ne bouge pas', () => {
    const { state } = run({ mesh: null, levels: [] }, life(1, 0, 1000, () => ({ x: 7, z: 2.2 })))
    expect(state.cut).toBeNull()
  })

  it('ne modifie jamais l’état reçu', () => {
    const frozen = Object.freeze({ ...FOLLOW_INITIAL })
    const next = stepFollow(frozen, { alive: true, lifeKey: 1, x: 7, y: 5, z: 2.2, atMs: 0 }, TWO_FLOORS)
    expect(frozen).toEqual(FOLLOW_INITIAL)
    expect(next).not.toBe(frozen)
    expect(next.cut).toBe(2.2)
  })
})
