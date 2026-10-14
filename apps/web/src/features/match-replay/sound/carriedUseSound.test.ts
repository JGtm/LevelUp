/**
 * Tests — carriedUseSound (l'utilisation d'un équipement porté), sur les relevés réels figés de
 * `test/carriedUseWitnesses.ts` (cuisson unitaire au schéma 95 de `5676a9ba` et `1f8e8849`).
 */
import { describe, expect, it } from 'vitest'

import { doc1f8e8849, doc5676a9ba, PORTES_5676A9BA, POSES_DU_MUR_565 } from '../test/carriedUseWitnesses'
import { carriedUseSoundEvents } from './carriedUseSound'
import { buildSoundTimeline } from './replaySound'

function parStem(ev: { ms: number; stem: string }[]): Record<string, number[]> {
  const out: Record<string, number[]> = {}
  for (const e of ev) (out[e.stem] ??= []).push(e.ms)
  for (const v of Object.values(out)) v.sort((a, b) => a - b)
  return out
}

describe('carriedUseSoundEvents — un son à chaque utilisation datée', () => {
  it('5676a9ba : le capteur, le champ et l écran sonnent à l image de chaque utilisation, le mur jamais', () => {
    const ev = parStem(carriedUseSoundEvents(doc5676a9ba()))
    const usages = (r: number) =>
      PORTES_5676A9BA.filter((c) => c.r === r)
        .flatMap((c) => c.uses)
        .map((u) => u * 100)
        .sort((a, b) => a - b)
    expect(ev.sensor_activate).toEqual(usages(1))
    expect(ev.repair_field_activate).toEqual(usages(23))
    expect(ev.shroud_deploy).toEqual([630_700])
    expect(ev.sensor_activate).toHaveLength(14)
    expect(ev.repair_field_activate).toHaveLength(11)
    expect(ev.wall_activate).toBeUndefined()
  })

  it('1f8e8849 : le traqueur prend le son du capteur, à ses deux utilisations', () => {
    expect(carriedUseSoundEvents(doc1f8e8849())).toEqual([
      expect.objectContaining({ ms: 162_200, stem: 'sensor_activate' }),
      expect.objectContaining({ ms: 244_700, stem: 'sensor_activate' }),
    ])
  })
})

describe('aucun doublon : le mur sonne par sa pose, une fois', () => {
  it('5676a9ba : les deux utilisations du mur de 565 sonnent par leurs deux poses deployed, pas en plus', () => {
    const ev = buildSoundTimeline(doc5676a9ba(true), [])
    const murs = ev.filter((e) => e.stem === 'wall_activate').map((e) => e.ms)
    expect(murs).toEqual(POSES_DU_MUR_565.map((p) => p.t0 * 100))
  })
})
