/**
 * useReplayRespawnForecast.test.tsx — LA PROBABILITÉ DE RÉAPPARITION CÂBLÉE : le chargement seulement
 * quand le calque est actif, un refus qui est un calque absent (jamais une action impossible), les deux
 * peintres qui portent leur id et lisent les mêmes morts à chaque image, Seul / Masquer, et les textes
 * dans la langue de la page (sans jamais le mot « prévision »).
 */
import { renderHook } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import type { CalloutZoneReady } from '@/lib/replay/calloutsPaint'
import type { ReplaySpawnForecast } from '@/lib/api/types'

import { REPLAY_TEXT } from '../i18n/i18n'
import type { CanvasView } from '../model/replayView'
import { count, recordingContext } from '../test/recordingContext'
import { forecastWording, useReplayRespawnForecast, type RespawnForecastOptions } from './useReplayRespawnForecast'

const query = vi.fn()
vi.mock('../../../lib/replay/queries', () => ({
  useReplaySpawnForecast: (...args: unknown[]) => query(...args),
}))

const VIEW: CanvasView = {
  bounds: { minX: 0, maxX: 100, minY: 0, maxY: 100 } as CanvasView['bounds'],
  width: 400,
  height: 400,
  pad: 0,
}

function zone(volumeIndex: number, x0: number, y0: number): CalloutZoneReady {
  return {
    names: {}, x: x0, y: y0, z: 0, zBottom: -1, zTop: 1, volumeIndex, big: true,
    polygon: [{ x: x0, y: y0 }, { x: x0 + 30, y: y0 }, { x: x0 + 30, y: y0 + 30 }, { x: x0, y: y0 + 30 }],
    parts: [], holes: [], labelX: x0 + 15, labelY: y0 + 15,
  }
}

const CALLOUTS = [zone(3, 0, 0), zone(7, 40, 40)]

const READY: ReplaySpawnForecast = {
  status: 'ready', candidates: 20, discarded_deaths: 0, matches_on_map: 4, sample_step_ms: 1000,
  zones: [{ volume_index: 3, name: 'Fosse' }, { volume_index: 7, name: 'Pont' }],
  points: [{ x: 10, y: 10, zone_index: 0 }, { x: 50, y: 50, zone_index: 1 }, { x: 90, y: 90, zone_index: -1 }],
  deaths: [
    {
      xuid: 'xuid(a)', death_frame: 100, respawn_frame: 130, death: { x: 10, y: 20, z: 0 }, actual: { x: 50, y: 60, z: 0 }, actual_zone: 7,
      samples: [{ frame: 100, outside_zone: 0, points_p: [0.5, 0.3, 0.2], zones: [{ zone: 3, p: 0.6 }, { zone: 7, p: 0.4 }] }],
    },
    {
      xuid: 'xuid(b)', death_frame: 100, respawn_frame: 130, death: { x: 70, y: 20, z: 0 }, actual: { x: 10, y: 10, z: 0 }, actual_zone: 3,
      samples: [{ frame: 100, outside_zone: 0, points_p: [0.5, 0.3, 0.2], zones: [{ zone: 3, p: 0.9 }] }],
    },
  ],
}

function options(over: Partial<RespawnForecastOptions> = {}): RespawnForecastOptions {
  return {
    doc: { frameIntervalMs: 100 },
    view: VIEW,
    enabled: true,
    draw: 'zones',
    playerSlug: 'jgtm',
    matchId: 'm1',
    locale: 'fr',
    callouts: CALLOUTS,
    identity: { colorOfXuid: (x) => (x === 'xuid(a)' ? 'ink-a' : 'ink-b'), nameOfXuid: (x) => (x === 'xuid(a)' ? 'Brume' : 'Orage') },
    ink: { scrim: 'ink-scrim', onScrim: 'ink-on-scrim', labelStroke: 'ink-stroke', neutral: 'ink-neutral' },
    ...over,
  }
}

function mount(over: Partial<RespawnForecastOptions> = {}) {
  return renderHook((o: RespawnForecastOptions) => useReplayRespawnForecast(o), { initialProps: options(over) })
}

beforeEach(() => {
  query.mockReset()
  query.mockReturnValue({ data: READY, isError: false })
})

describe('le chargement et la disponibilité', () => {
  it('carte à zones : la réponse est demandée, le calque est prêt et proposé', () => {
    const { result } = mount()
    expect(query).toHaveBeenCalledWith('jgtm', 'm1', true, 'fr')
    expect(result.current.ready).toBe(true)
    expect(result.current.available).toBe(true)
    expect(result.current.show).toBe(true)
  })

  it('calque éteint : la réponse est quand même demandée (la disponibilité ne se déduit pas de la préférence)', () => {
    mount({ enabled: false })
    expect(query).toHaveBeenCalledWith('jgtm', 'm1', true, 'fr')
  })

  it('carte sans zone nommée : rien n’est demandé, pas de calque, pas d’interrupteur', () => {
    query.mockReturnValue({ data: undefined, isError: false })
    const { result } = mount({ callouts: [] })
    expect(query).toHaveBeenCalledWith('jgtm', 'm1', false, 'fr')
    expect(result.current.ready).toBe(false)
    expect(result.current.available).toBe(false)
  })

  it('« unavailable » : calque absent et interrupteur absent, pas une erreur', () => {
    query.mockReturnValue({ data: { ...READY, status: 'unavailable', reason: 'not_enough_matches', deaths: null, zones: null }, isError: false })
    const { result } = mount()
    expect(result.current.ready).toBe(false)
    expect(result.current.available).toBe(false)
    const { ctx, ops } = recordingContext()
    result.current.layers[0].paint(ctx, 105, 1)
    result.current.layers[1].paint(ctx, 105, 1)
    expect(ops).toEqual([])
  })

  it('échec de chargement : calque et interrupteur absents', () => {
    query.mockReturnValue({ data: undefined, isError: true })
    const { result } = mount()
    expect(result.current.ready).toBe(false)
    expect(result.current.available).toBe(false)
  })

  it('en chargement et allumé : l’interrupteur attend la réponse', () => {
    query.mockReturnValue({ data: undefined, isError: false })
    const { result } = mount()
    expect(result.current.available).toBe(false)
  })

  it('éteint, en chargement : l’interrupteur n’est pas offert avant la réponse (jamais offert puis retiré)', () => {
    query.mockReturnValue({ data: undefined, isError: false })
    const { result } = mount({ enabled: false })
    expect(result.current.available).toBe(false)
    expect(result.current.ready).toBe(false)
  })

  it('éteint et « unavailable » : l’interrupteur n’est jamais offert (action impossible)', () => {
    query.mockReturnValue({ data: { ...READY, status: 'unavailable', reason: 'not_enough_matches', deaths: null, zones: null }, isError: false })
    const { result } = mount({ enabled: false })
    expect(result.current.available).toBe(false)
  })

  it('éteint puis allumé : la même réponse sert, l’interrupteur reste offert sans rechargement', () => {
    const { result, rerender } = mount({ enabled: false })
    expect(result.current.available).toBe(true)
    rerender(options({ enabled: true }))
    expect(result.current.available).toBe(true)
    expect(query.mock.calls.every((c) => c[2] === true)).toBe(true)
  })

  it('éteint avec une réponse déjà en cache : l’interrupteur suit la réponse', () => {
    const { result } = mount({ enabled: false })
    expect(result.current.available).toBe(true)
    expect(result.current.show).toBe(false)
  })

  it('une page sans match n’a pas de calque', () => {
    const { result } = mount({ playerSlug: undefined, matchId: undefined })
    expect(result.current.available).toBe(false)
  })

  it('les zones de la réponse sans forme au document ne rendent pas le calque prêt', () => {
    const { result } = mount({ callouts: [zone(99, 0, 0)] })
    expect(result.current.ready).toBe(false)
  })
})

describe('les deux peintres', () => {
  it('portent leur id, dans l’ordre teinte puis repères', () => {
    const { result } = mount()
    expect(result.current.layers.map((l) => l.id)).toEqual(['reapparition-fond', 'reapparition-reperes'])
  })

  it('lisent les mêmes morts à la même image : teinte, anneaux, étiquettes et légende, puis plus rien après l’effacement', () => {
    const { result } = mount()
    const [zones, reperes] = result.current.layers
    const z = recordingContext()
    const r = recordingContext()
    ;(r.ctx as unknown as { measureText: (t: string) => { width: number } }).measureText = (t) => ({ width: t.length * 6 })
    r.ops.length = 0
    zones.paint(z.ctx, 105, 1)
    reperes.paint(r.ctx, 105, 1)
    // Deux joueurs morts : deux jeux de zones teintées (2 + 1), deux anneaux, la légende.
    expect(count(z.ops, 'fill')).toBe(3)
    expect(count(r.ops, 'arc')).toBeGreaterThanOrEqual(4)
    expect(r.ops.filter((o) => o.op === 'fillText').map((o) => String(o.args[0]))).toContain('Probabilité de réapparition')

    const apres = recordingContext()
    zones.paint(apres.ctx, 1_000, 1)
    reperes.paint(apres.ctx, 1_000, 1)
    expect(apres.ops).toEqual([])
  })

  it('le choix du dessin décide ce que le calque « fond » peint : teinte, disques, ou les deux', () => {
    const paint = (draw: 'both' | 'zones' | 'points') => {
      const { result } = mount({ draw })
      const z = recordingContext()
      result.current.layers[0].paint(z.ctx, 105, 1)
      return { teintes: count(z.ops, 'fill') - count(z.ops, 'arc'), disques: count(z.ops, 'arc') }
    }
    // Deux joueurs morts à 105 : trois zones teintées (2 + 1) ; les disques : trois points par joueur sans probabilité nulle.
    expect(paint('zones')).toEqual({ teintes: 3, disques: 0 })
    expect(paint('points').disques).toBe(6)
    expect(paint('points').teintes).toBe(0)
    expect(paint('both').disques).toBe(6)
    expect(paint('both').teintes).toBe(3)
  })

  it('un changement de dessin repeint avec le nouveau choix', () => {
    const { result, rerender } = mount({ draw: 'zones' })
    rerender(options({ draw: 'points' }))
    const z = recordingContext()
    result.current.layers[0].paint(z.ctx, 105, 1)
    expect(count(z.ops, 'arc')).toBe(6)
  })

  it('Seul et Masquer : la probabilité du joueur mort suit le prédicat « auteur »', () => {
    const seul = mount({ shownAuthor: (x) => x === 'xuid(a)' })
    const z = recordingContext()
    seul.result.current.layers[0].paint(z.ctx, 105, 1)
    expect(count(z.ops, 'fill')).toBe(2)
    expect(z.ops.filter((o) => o.op === 'set fillStyle').every((o) => o.args[0] === 'ink-a')).toBe(true)

    const masque = mount({ shownAuthor: (x) => x !== 'xuid(a)' })
    const m = recordingContext()
    masque.result.current.layers[0].paint(m.ctx, 105, 1)
    expect(count(m.ops, 'fill')).toBe(1)
    expect(m.ops.filter((o) => o.op === 'set fillStyle').every((o) => o.args[0] === 'ink-b')).toBe(true)
  })

  it('un changement de filtre repeint avec la nouvelle règle (le cache d’image ne survit pas)', () => {
    const { result, rerender } = mount()
    const avant = recordingContext()
    result.current.layers[0].paint(avant.ctx, 105, 1)
    expect(count(avant.ops, 'fill')).toBe(3)
    rerender(options({ shownAuthor: (x) => x === 'xuid(b)' }))
    const apres = recordingContext()
    result.current.layers[0].paint(apres.ctx, 105, 1)
    expect(count(apres.ops, 'fill')).toBe(1)
  })
})

describe('les textes', () => {
  it('une probabilité s’écrit selon la langue de la page', () => {
    const fr = forecastWording('fr')
    const en = forecastWording('en')
    expect(fr.share(0.35).replace(/\s/g, ' ')).toBe('35 %')
    expect(en.share(0.35)).toBe('35%')
    expect(fr.share(0.001)).toBe('< 1 %')
    expect(en.share(0.001)).toBe('< 1%')
    expect(fr.share(0)).toMatch(/^0\s%$/)
  })

  it('les rangs : « 1er choix » en français, « 1st choice » en anglais', () => {
    expect(forecastWording('fr').rank(1)).toBe('1er choix')
    expect(forecastWording('fr').rank(3)).toBe('3e choix')
    expect(forecastWording('en').rank(1)).toBe('1st choice')
    expect(forecastWording('en').rank(2)).toBe('2nd choice')
    expect(forecastWording('en').rank(3)).toBe('3rd choice')
    expect(forecastWording('en').rank(11)).toBe('11th choice')
    expect(forecastWording('en').rank(22)).toBe('22nd choice')
  })

  it('JAMAIS « prévision » à l’écran, dans aucune langue : titre, aide, puce, légende et étiquettes', () => {
    for (const locale of ['fr', 'en'] as const) {
      const t = REPLAY_TEXT[locale].respawnForecast
      const w = forecastWording(locale)
      const lines = [
        t.title, t.hint, w.chip('X', 3), w.rank(2), w.share(0.3), w.legendCountdown('X', 3), w.appearedIn('X', 'Z'),
        w.appearedOutside('X'), w.appearedRank('2e', '30 %'), w.appearedUnlikely, w.legendScale, w.legendFooter,
        w.legendPoints, w.legendTintUnderPoints, t.drawLabel, ...Object.values(t.draw), ...Object.values(t.drawHint),
      ]
      for (const line of lines) expect(line.toLowerCase()).not.toMatch(/pr[ée]vision|pr[ée]dict|forecast/)
    }
    expect(REPLAY_TEXT.fr.respawnForecast.title).toBe('Probabilité de réapparition')
  })
})
