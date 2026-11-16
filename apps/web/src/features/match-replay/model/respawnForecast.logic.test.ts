/**
 * respawnForecast.logic.test.ts — LA PROBABILITÉ DE RÉAPPARITION LUE À UN INSTANT : normalisation,
 * échantillon courant, trois premières zones, rang du point réel, et surtout le DÉFILEMENT du temps
 * (avant la mort, pendant le compte à rebours, à la réapparition, après l'effacement).
 */
import { describe, expect, it } from 'vitest'

import type { ReplaySpawnForecast } from '@/lib/api/types'

import {
  momentOf,
  momentsAt,
  normalizeSpawnForecast,
  primaryMoment,
  rankOfZone,
  REVEAL_FADE_MS,
  REVEAL_HOLD_MS,
  revealOpacity,
  sampleAt,
  tintAlpha,
  discRadiusPx,
  DISC_BASE_RADIUS_PX,
  DISC_SPAN_RADIUS_PX,
  TINT_AMPLITUDE_ALPHA_UNDER_POINTS,
  TINT_BASE_ALPHA_UNDER_POINTS,
  TINT_AMPLITUDE_ALPHA,
  TINT_BASE_ALPHA,
  type ForecastDeath,
  type SpawnForecastReady,
} from './respawnForecast.logic'

/** 100 ms par image : une seconde de compte à rebours = 10 images. */
const FRAME_MS = 100

type WireDeath = NonNullable<ReplaySpawnForecast['deaths']>[number]

function wireDeath(over: Partial<WireDeath> = {}): WireDeath {
  return {
    xuid: 'xuid(1)',
    death_frame: 100,
    respawn_frame: 130,
    death: { x: 10, y: 20, z: 0 },
    actual: { x: 50, y: 60, z: 0 },
    actual_zone: 7,
    samples: [
      { frame: 100, outside_zone: 0, points_p: [0.5, 0.3, 0.2], zones: [{ zone: 3, p: 0.5 }, { zone: 7, p: 0.3 }, { zone: 4, p: 0.1 }, { zone: 9, p: 0.05 }] },
      { frame: 110, outside_zone: 0, points_p: [0.5, 0.3, 0.2], zones: [{ zone: 7, p: 0.6 }, { zone: 3, p: 0.2 }] },
      { frame: 120, outside_zone: 0, points_p: [0.5, 0.3, 0.2], zones: [{ zone: 7, p: 0.7 }, { zone: 3, p: 0.1 }] },
    ],
    ...over,
  }
}

function wire(over: Partial<ReplaySpawnForecast> = {}): ReplaySpawnForecast {
  return {
    status: 'ready',
    candidates: 40,
    discarded_deaths: 0,
    matches_on_map: 5,
    sample_step_ms: 1000,
    zones: [{ volume_index: 3, name: 'Fosse' }, { volume_index: 7, name: 'Pont' }],
    // Trois points : un en « Fosse » (zones[0]), un en « Pont » (zones[1]), un hors de toute zone servie.
    points: [{ x: 1, y: 2, zone_index: 0 }, { x: 3, y: 4, zone_index: 1 }, { x: 5, y: 6, zone_index: -1 }],
    deaths: [wireDeath()],
    ...over,
  }
}

function ready(over: Partial<ReplaySpawnForecast> = {}): SpawnForecastReady {
  const f = normalizeSpawnForecast(wire(over))
  if (!f) throw new Error('prevision attendue prete')
  return f
}

describe('normalizeSpawnForecast', () => {
  it('absente ou indisponible : aucun calque', () => {
    expect(normalizeSpawnForecast(undefined)).toBeNull()
    expect(normalizeSpawnForecast(null)).toBeNull()
    expect(normalizeSpawnForecast(wire({ status: 'unavailable', reason: 'no_model', deaths: null, zones: null }))).toBeNull()
  })

  it('prête mais sans mort exploitable : aucun calque', () => {
    expect(normalizeSpawnForecast(wire({ deaths: null }))).toBeNull()
    expect(normalizeSpawnForecast(wire({ deaths: [wireDeath({ samples: null })] }))).toBeNull()
    expect(normalizeSpawnForecast(wire({ deaths: [wireDeath({ respawn_frame: 100 })] }))).toBeNull()
  })

  it('écarte une mort sans coordonnées finies, garde les autres', () => {
    const f = normalizeSpawnForecast(
      wire({ deaths: [wireDeath({ xuid: 'xuid(bad)', death: { x: Number.NaN, y: 0, z: 0 } }), wireDeath({ xuid: 'xuid(ok)' })] }),
    )
    expect(f?.deaths.map((d) => d.xuid)).toEqual(['xuid(ok)'])
  })

  it('trie les morts par image et les échantillons par image, les zones de la plus à la moins probable', () => {
    const f = normalizeSpawnForecast(
      wire({
        deaths: [
          wireDeath({ xuid: 'xuid(late)', death_frame: 300, respawn_frame: 330, samples: [{ frame: 310, outside_zone: 0, points_p: [0.5, 0.3, 0.2], zones: [{ zone: 1, p: 0.1 }, { zone: 2, p: 0.4 }] }, { frame: 300, outside_zone: 0, points_p: [0.5, 0.3, 0.2], zones: [] }] }),
          wireDeath({ xuid: 'xuid(early)' }),
        ],
      }),
    )
    expect(f?.deaths.map((d) => d.xuid)).toEqual(['xuid(early)', 'xuid(late)'])
    const late = f!.deaths[1]
    expect(late.samples.map((s) => s.frame)).toEqual([300, 310])
    expect(late.samples[1].zones.map((z) => z.zone)).toEqual([2, 1])
  })

  it('ne garde pas une zone de probabilité nulle', () => {
    const f = normalizeSpawnForecast(wire({ deaths: [wireDeath({ samples: [{ frame: 100, outside_zone: 1, points_p: null, zones: [{ zone: 1, p: 0 }] }] })] }))
    expect(f?.deaths[0].samples[0].zones).toEqual([])
  })

  it('la zone réelle absente devient null, et les noms de zone suivent le volume_index', () => {
    const f = normalizeSpawnForecast(wire({ deaths: [wireDeath({ actual_zone: undefined })] }))
    expect(f?.deaths[0].actualZone).toBeNull()
    expect(f?.zones).toEqual([{ volumeIndex: 3, name: 'Fosse' }, { volumeIndex: 7, name: 'Pont' }])
  })
})

describe('sampleAt et rankOfZone', () => {
  const death: ForecastDeath = ready().deaths[0]

  it('prend l’échantillon le plus récent qui ne dépasse pas l’image', () => {
    expect(sampleAt(death, 100).frame).toBe(100)
    expect(sampleAt(death, 109.9).frame).toBe(100)
    expect(sampleAt(death, 110).frame).toBe(110)
    expect(sampleAt(death, 125).frame).toBe(120)
    expect(sampleAt(death, 500).frame).toBe(120)
  })

  it('avant le premier échantillon, retombe sur le premier', () => {
    expect(sampleAt(death, 50).frame).toBe(100)
  })

  it('le rang du point réel : 1 = première zone, null hors des zones ou sans zone nommée', () => {
    expect(rankOfZone(death.samples[0], 7)).toBe(2)
    expect(rankOfZone(death.samples[2], 7)).toBe(1)
    expect(rankOfZone(death.samples[2], 4)).toBeNull()
    expect(rankOfZone(death.samples[2], null)).toBeNull()
  })
})

describe('momentOf — le temps défile', () => {
  const death = ready().deaths[0]
  const at = (frame: number) => momentOf(death, frame, FRAME_MS)

  it('avant la mort : rien', () => {
    expect(at(99.9)).toBeNull()
    expect(at(0)).toBeNull()
  })

  it('pendant le compte à rebours : l’échantillon courant, les trois premières zones, le temps restant', () => {
    const m = at(100)!
    expect(m.phase).toBe('countdown')
    expect(m.sample.frame).toBe(100)
    expect(m.top.map((z) => z.zone)).toEqual([3, 7, 4])
    expect(m.peak).toBe(0.5)
    expect(m.remainingMs).toBeCloseTo(3000, 6)
    expect(m.fraction).toBeCloseTo(1, 6)
    expect(m.rank).toBeNull()
    expect(m.opacity).toBe(1)

    const later = at(115)!
    expect(later.sample.frame).toBe(110)
    expect(later.top.map((z) => z.zone)).toEqual([7, 3])
    expect(later.remainingMs).toBeCloseTo(1500, 6)
    expect(later.fraction).toBeCloseTo(0.5, 6)
  })

  it('à la dernière image avant la réapparition : toujours le compte à rebours', () => {
    const m = at(129.9)!
    expect(m.phase).toBe('countdown')
    expect(m.sample.frame).toBe(120)
    expect(m.fraction).toBeGreaterThan(0)
  })

  it('à la réapparition : le point réel, son rang dans la dernière probabilité, dessin plein', () => {
    const m = at(130)!
    expect(m.phase).toBe('reveal')
    expect(m.sample.frame).toBe(120)
    expect(m.rank).toBe(1)
    expect(m.actualP).toBe(0.7)
    expect(m.remainingMs).toBe(0)
    expect(m.fraction).toBe(0)
    expect(m.opacity).toBe(1)
  })

  it('un point réel hors des zones probables : sans rang ni probabilité', () => {
    const d = ready({ deaths: [wireDeath({ actual_zone: 4 })] }).deaths[0]
    const m = momentOf(d, 130, FRAME_MS)!
    expect(m.rank).toBeNull()
    expect(m.actualP).toBe(0)
  })

  it('le dessin s’éteint en fondu puis s’efface après environ 3 s', () => {
    const framesOf = (ms: number) => 130 + ms / FRAME_MS
    const fadeStart = REVEAL_HOLD_MS - REVEAL_FADE_MS
    expect(at(framesOf(fadeStart - 1))!.opacity).toBe(1)
    const mid = at(framesOf(fadeStart + REVEAL_FADE_MS / 2))!
    expect(mid.opacity).toBeGreaterThan(0)
    expect(mid.opacity).toBeLessThan(1)
    expect(at(framesOf(REVEAL_HOLD_MS - 1))!.opacity).toBeGreaterThan(0)
    expect(at(framesOf(REVEAL_HOLD_MS))).toBeNull()
    expect(at(framesOf(REVEAL_HOLD_MS + 5000))).toBeNull()
  })

  it('revealOpacity : bornée à [0, 1], nulle avant la réapparition et après la tenue', () => {
    expect(revealOpacity(-1)).toBe(0)
    expect(revealOpacity(0)).toBe(1)
    expect(revealOpacity(REVEAL_HOLD_MS)).toBe(0)
  })
})

describe('momentsAt — Seul / Masquer et plusieurs morts à la fois', () => {
  const forecast = ready({
    deaths: [
      wireDeath({ xuid: 'xuid(a)', death_frame: 100, respawn_frame: 130 }),
      wireDeath({ xuid: 'xuid(b)', death_frame: 110, respawn_frame: 150, samples: [{ frame: 110, outside_zone: 0, points_p: [0.5, 0.3, 0.2], zones: [{ zone: 3, p: 0.9 }] }] }),
    ],
  })

  it('sans filtre : toutes les morts actives', () => {
    expect(momentsAt(forecast, 120, FRAME_MS).map((m) => m.death.xuid)).toEqual(['xuid(a)', 'xuid(b)'])
    expect(momentsAt(forecast, 140, FRAME_MS).map((m) => m.death.xuid)).toEqual(['xuid(a)', 'xuid(b)'])
  })

  it('Seul : la probabilité du joueur gardé, celle des autres retirée', () => {
    const seulA = (xuid: string) => xuid === 'xuid(a)'
    expect(momentsAt(forecast, 120, FRAME_MS, seulA).map((m) => m.death.xuid)).toEqual(['xuid(a)'])
  })

  it('Masquer : la probabilité du joueur masqué retirée, celle des autres gardée', () => {
    const masquerA = (xuid: string) => xuid !== 'xuid(a)'
    expect(momentsAt(forecast, 120, FRAME_MS, masquerA).map((m) => m.death.xuid)).toEqual(['xuid(b)'])
  })

  it('le fil du temps : rien, une mort, deux morts, une réapparition, puis plus rien', () => {
    const noms = (f: number) => momentsAt(forecast, f, FRAME_MS).map((m) => `${m.death.xuid}:${m.phase}`)
    expect(noms(90)).toEqual([])
    expect(noms(105)).toEqual(['xuid(a):countdown'])
    expect(noms(115)).toEqual(['xuid(a):countdown', 'xuid(b):countdown'])
    expect(noms(135)).toEqual(['xuid(a):reveal', 'xuid(b):countdown'])
    expect(noms(155)).toEqual(['xuid(a):reveal', 'xuid(b):reveal'])
    expect(noms(130 + REVEAL_HOLD_MS / FRAME_MS + 1)).toEqual(['xuid(b):reveal'])
    expect(noms(150 + REVEAL_HOLD_MS / FRAME_MS + 1)).toEqual([])
  })

  it('primaryMoment : la mort la plus récente ; rien sans mort active', () => {
    expect(primaryMoment([])).toBeNull()
    const moments = momentsAt(forecast, 120, FRAME_MS)
    expect(primaryMoment(moments)?.death.xuid).toBe('xuid(b)')
  })
})

describe('tintAlpha', () => {
  it('un fond léger qui monte avec la part de la première zone', () => {
    expect(tintAlpha(0, 0.5)).toBeCloseTo(TINT_BASE_ALPHA, 9)
    expect(tintAlpha(0.5, 0.5)).toBeCloseTo(TINT_BASE_ALPHA + TINT_AMPLITUDE_ALPHA, 9)
    expect(tintAlpha(0.25, 0.5)).toBeGreaterThan(TINT_BASE_ALPHA)
    expect(tintAlpha(0.25, 0.5)).toBeLessThan(TINT_BASE_ALPHA + TINT_AMPLITUDE_ALPHA)
  })

  it('sans première zone, le fond seul', () => {
    expect(tintAlpha(0.3, 0)).toBe(TINT_BASE_ALPHA)
  })
})

describe('les points candidats et leur probabilité', () => {
  it('sont servis une fois, `zone_index` devient le volume_index de la zone, -1 aucune zone', () => {
    const f = ready()
    expect(f.points).toEqual([
      { x: 1, y: 2, zone: 3 },
      { x: 3, y: 4, zone: 7 },
      { x: 5, y: 6, zone: null },
    ])
  })

  it('chaque échantillon porte la probabilité des points, alignée, et chaque instant la reprend', () => {
    const f = ready()
    const d = f.deaths[0]
    expect(d.samples.every((s) => s.pointsP.length === f.points.length)).toBe(true)
    const m = momentOf(d, 105, FRAME_MS, f.points)!
    expect(m.points).toBe(f.points)
    expect(m.sample.pointsP).toEqual([0.5, 0.3, 0.2])
    expect(momentsAt(f, 105, FRAME_MS)[0].points).toBe(f.points)
    // À la réapparition : les points de la dernière probabilité.
    expect(momentsAt(f, 131, FRAME_MS)[0].sample.pointsP).toHaveLength(3)
  })

  it('la probabilité d’une zone reste la somme de ses points (ici une zone par point)', () => {
    const f = ready()
    const s = f.deaths[0].samples[0]
    expect(s.pointsP[0]).toBeCloseTo(s.zones.find((z) => z.zone === 3)!.p, 9)
    expect(s.pointsP[1]).toBeCloseTo(s.zones.find((z) => z.zone === 7)!.p, 9)
  })

  it('une liste de probabilités qui n’est pas alignée sur les points est écartée, jamais devinée', () => {
    const f = normalizeSpawnForecast(wire({ deaths: [wireDeath({ samples: [{ frame: 100, outside_zone: 0, points_p: [0.5, 0.5], zones: [{ zone: 3, p: 0.5 }] }] })] }))!
    expect(f.deaths[0].samples[0].pointsP).toEqual([])
    expect(f.deaths[0].samples[0].zones).toHaveLength(1)
  })

  it('une réponse sans point : aucune probabilité de point, le reste intact', () => {
    const f = normalizeSpawnForecast(wire({ points: null, deaths: [wireDeath({ samples: [{ frame: 100, outside_zone: 0, points_p: null, zones: [{ zone: 3, p: 1 }] }] })] }))!
    expect(f.points).toEqual([])
    expect(f.deaths[0].samples[0].pointsP).toEqual([])
  })
})

describe('le disque d’un point : le rayon suit la racine de la probabilité', () => {
  it('croît avec la probabilité, base à 0, base + étendue à 1', () => {
    expect(discRadiusPx(0)).toBe(DISC_BASE_RADIUS_PX)
    expect(discRadiusPx(1)).toBeCloseTo(DISC_BASE_RADIUS_PX + DISC_SPAN_RADIUS_PX, 9)
    expect(discRadiusPx(0.05)).toBeLessThan(discRadiusPx(0.2))
    expect(discRadiusPx(0.2)).toBeLessThan(discRadiusPx(0.45))
    // La racine : quadrupler la probabilité double la part variable du rayon.
    expect(discRadiusPx(0.16) - DISC_BASE_RADIUS_PX).toBeCloseTo(2 * (discRadiusPx(0.04) - DISC_BASE_RADIUS_PX), 9)
  })

  it('une probabilité négative ne donne pas de rayon sous la base', () => {
    expect(discRadiusPx(-0.5)).toBe(DISC_BASE_RADIUS_PX)
  })
})

describe('tintAlpha selon le dessin', () => {
  it('sous les points (« Zones et points »), la teinte reste plus légère que seule', () => {
    expect(tintAlpha(0.5, 0.5, 'both')).toBeCloseTo(TINT_BASE_ALPHA_UNDER_POINTS + TINT_AMPLITUDE_ALPHA_UNDER_POINTS, 9)
    expect(tintAlpha(0.5, 0.5, 'both')).toBeLessThan(tintAlpha(0.5, 0.5, 'zones'))
    expect(tintAlpha(0, 0.5, 'both')).toBeLessThan(tintAlpha(0, 0.5, 'zones'))
  })
})
