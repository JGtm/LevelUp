/**
 * Tests des SEGMENTS moteur de l'export (vehicleEngineMix.ts) : la chaîne
 * enter -> loop x N -> exit posée sur l'axe du clip, les jonctions de 150 ms, la reprise
 * directe sur loop quand la plage coupe un épisode, et la queue d'un exit gravé.
 */
import { describe, expect, it } from 'vitest'

import {
  engineStemsOf,
  engineTailSeconds,
  planEngineSegments,
} from './vehicleEngineMix'
import { ENGINE_CROSSFADE_S, type EnginePlan } from './vehicleEngineSound'

/** Durées plausibles de la banque (secondes), par stem. */
const DUREES: Record<string, number> = {
  vehicle_warthog_enter: 2.1,
  vehicle_warthog_loop: 3.25,
  vehicle_warthog_exit: 2.25,
  vehicle_scorpion_enter: 3.6,
  vehicle_scorpion_loop: 3.26,
  vehicle_scorpion_idle: 1.78,
  vehicle_scorpion_exit: 5.6,
}
const durOf = (stem: string) => DUREES[stem] ?? null

const plan = (family: string, episodes: EnginePlan['episodes']): EnginePlan => ({ family, episodes })

describe('planEngineSegments — la chaîne complète d un épisode dans la plage', () => {
  const p = plan('warthog', [{ t0Ms: 2_000, t1Ms: 10_000, idle: [] }])
  const segs = planEngineSegments(p, { startMs: 0, endMs: 20_000 }, durOf)

  it('enter à T0, loop à la jonction, exit à T1 — trois segments', () => {
    expect(segs.map((s) => s.stem)).toEqual([
      'vehicle_warthog_enter',
      'vehicle_warthog_loop',
      'vehicle_warthog_exit',
    ])
  })

  it('la jonction enter->loop est un fondu croisé de 150 ms qui finit à la fin de l enter', () => {
    const [enter, loop] = segs
    expect(enter.atS).toBe(2)
    expect(enter.loop).toBe(false)
    expect(enter.stopS).toBeCloseTo(2 + 2.1, 6)
    expect(enter.fadeOutS).toBe(ENGINE_CROSSFADE_S)
    expect(loop.loop).toBe(true)
    expect(loop.atS).toBeCloseTo(2 + 2.1 - ENGINE_CROSSFADE_S, 6)
    expect(loop.fadeInS).toBe(ENGINE_CROSSFADE_S)
  })

  it('l exit part à T1, sans fin forcée : ses fins sont GRAVÉES (contrat de la banque)', () => {
    const exit = segs[2]
    expect(exit.atS).toBe(10)
    expect(exit.stopS).toBeUndefined()
    const loop = segs[1]
    expect(loop.stopS).toBeCloseTo(10 + ENGINE_CROSSFADE_S, 6)
  })
})

describe('planEngineSegments — plage qui COUPE un épisode', () => {
  const p = plan('warthog', [{ t0Ms: 2_000, t1Ms: 30_000, idle: [] }])

  it('début coupé : reprise DIRECTEMENT sur loop, jamais de re-enter (règle du seek)', () => {
    const segs = planEngineSegments(p, { startMs: 5_000, endMs: 12_000 }, durOf)
    expect(segs.map((s) => s.stem)).toEqual(['vehicle_warthog_loop'])
    expect(segs[0].atS).toBe(0)
    expect(segs[0].fadeInS).toBe(0)
  })

  it('fin coupée : pas d exit, la boucle s éteint par le fondu PILE sur la borne', () => {
    const segs = planEngineSegments(p, { startMs: 0, endMs: 12_000 }, durOf)
    const loop = segs[segs.length - 1]
    expect(loop.stem).toBe('vehicle_warthog_loop')
    expect(loop.stopS).toBe(12)
    expect(loop.fadeOutS).toBe(ENGINE_CROSSFADE_S)
    expect(segs.some((s) => s.stem === 'vehicle_warthog_exit')).toBe(false)
  })

  it('épisode entièrement hors plage : rien', () => {
    expect(planEngineSegments(p, { startMs: 31_000, endMs: 40_000 }, durOf)).toEqual([])
  })
})

describe('planEngineSegments — le ralenti du Scorpion', () => {
  const p = plan('scorpion', [
    { t0Ms: 0, t1Ms: 12_000, idle: [{ t0Ms: 4_000, t1Ms: 7_000 }] },
  ])

  it('loop, idle, loop — trois voix tenues raccordées par des fondus de 150 ms', () => {
    const segs = planEngineSegments(p, { startMs: 0, endMs: 20_000 }, durOf)
    const tenues = segs.filter((s) => s.loop)
    expect(tenues.map((s) => s.stem)).toEqual([
      'vehicle_scorpion_loop',
      'vehicle_scorpion_idle',
      'vehicle_scorpion_loop',
    ])
    const [, idle, retour] = tenues
    expect(idle.atS).toBe(4)
    expect(idle.fadeInS).toBe(ENGINE_CROSSFADE_S)
    expect(tenues[0].stopS).toBeCloseTo(4 + ENGINE_CROSSFADE_S, 6)
    expect(retour.atS).toBe(7)
    expect(idle.stopS).toBeCloseTo(7 + ENGINE_CROSSFADE_S, 6)
  })
})

describe('les absences restent des silences propres', () => {
  it('un fichier manquant saute son segment sans casser la chaîne', () => {
    const p = plan('ghost', [{ t0Ms: 0, t1Ms: 5_000, idle: [] }]) // aucune durée déclarée
    expect(planEngineSegments(p, { startMs: 0, endMs: 10_000 }, durOf)).toEqual([])
  })

  it('une famille hors table ne produit rien', () => {
    expect(planEngineSegments(plan('shade_turret', [{ t0Ms: 0, t1Ms: 5_000, idle: [] }]), { startMs: 0, endMs: 10_000 }, durOf)).toEqual([])
  })
})

describe('engineTailSeconds — la queue d un exit gravé', () => {
  it('un exit qui part près de la borne déborde de sa durée', () => {
    const p = plan('scorpion', [{ t0Ms: 0, t1Ms: 9_000, idle: [] }])
    const segs = planEngineSegments(p, { startMs: 0, endMs: 10_000 }, durOf)
    // Exit à 9 s, durée 5,6 s : fin à 14,6 s sur un clip de 10 s -> queue de 4,6 s.
    expect(engineTailSeconds(segs, durOf, 10)).toBeCloseTo(4.6, 6)
  })

  it('aucune queue quand tout finit dans le clip', () => {
    const p = plan('warthog', [{ t0Ms: 1_000, t1Ms: 4_000, idle: [] }])
    const segs = planEngineSegments(p, { startMs: 0, endMs: 60_000 }, durOf)
    expect(engineTailSeconds(segs, durOf, 60)).toBe(0)
  })
})

describe('engineStemsOf — les stems à décoder, dédupliqués', () => {
  it('deux plans de la même famille ne décodent qu une fois', () => {
    const plans = [plan('warthog', []), plan('warthog', []), plan('scorpion', [])]
    expect(engineStemsOf(plans).sort()).toEqual([
      'vehicle_scorpion_enter', 'vehicle_scorpion_exit', 'vehicle_scorpion_idle',
      'vehicle_scorpion_loop', 'vehicle_warthog_enter', 'vehicle_warthog_exit',
      'vehicle_warthog_loop',
    ])
  })
})
