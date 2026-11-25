/**
 * Câblage — SEUL / MASQUER sur les répliques : la bulle (carte) et la voix (piste sonore) d'un joueur masqué disparaissent
 * ensemble, « Seul » ne garde que celles du joueur gardé, et réafficher un joueur les ramène. Le prédicat de la carte vient du
 * vrai `useReplayPlayerFocus` (celui que `ReplayCanvas` passe au calque), la voix reçoit la même liste de joueurs masqués.
 */
import { renderHook, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import type { ReplayPlayer } from '@/lib/replay/rosterLogic'

const hoisted = vi.hoisted(() => ({ reply: null as unknown }))

vi.mock('@/lib/api/client', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/lib/api/client')>()),
  api: { getStatic: vi.fn(async () => hoisted.reply) },
}))

// La position d'un joueur ne change rien à qui se montre : tout joueur est posé au même endroit.
vi.mock('../model/carrierPosition', async (importOriginal) => ({
  ...(await importOriginal<typeof import('../model/carrierPosition')>()),
  useCarrierPosAt: () => () => ({ x: 5, y: 5 }),
}))

import type { HiddenPlayers } from '../model/playerPanelLogic'
import type { CanvasView } from '../model/replayView'
import { useReplayChatter } from '../layers/useReplayChatter'
import { testReplayDoc } from '../test/testDoc'
import { useChatterVoices } from './useChatterVoices'
import { useReplayPlayerFocus } from './useReplayPlayerFocus'
import type { ReplayZoom } from './useReplayZoom'

const TABLE = {
  version: 1,
  sons: {
    '0000000a': { texte: { fr: 'Parole de A', en: 'A speaks' }, audio: ['fr'] },
    '0000000b': { texte: { fr: 'Parole de B', en: 'B speaks' }, audio: ['fr'] },
  },
}

const DOC = testReplayDoc({
  titleSlug: 'titre_seul_masquer',
  frameIntervalMs: 100,
  frameCount: 200,
  chatter: [
    { t: 10, player: 'A', sound: '0000000a', name: 'seeingenemynearby' },
    { t: 12, player: 'B', sound: '0000000b', name: 'seeingenemybelow' },
  ],
})

const life = (slot: number, xuid: string) =>
  ({ slot, team: 0, xuid, startFrame: 0, endFrame: 100, points: [{ t: 0, x: 0, y: 0 }, { t: 100, x: 1, y: 1 }] }) as never
const A: ReplayPlayer = { xuid: 'A', lives: [life(1, 'A')] }
const B: ReplayPlayer = { xuid: 'B', lives: [life(2, 'B')] }

const VIEW: CanvasView = { bounds: { minX: 0, maxX: 100, minY: 0, maxY: 100 }, width: 400, height: 400, pad: 0 }

const ZOOM = {
  level: 1, center: { x: 0, y: 0 }, canZoomIn: true, canZoomOut: false, canPan: false,
  zoomIn: vi.fn(), zoomOut: vi.fn(), reset: vi.fn(), panStep: vi.fn(), panBy: vi.fn(), zoomAt: vi.fn(), focusOn: vi.fn(),
} as unknown as ReplayZoom

/** Un contexte témoin : il note les textes écrits. */
function recorder() {
  const texts: string[] = []
  const state: Record<string, unknown> = {}
  const ctx = new Proxy(state, {
    get(target, prop: string) {
      if (prop === 'measureText') return (s: string) => ({ width: s.length * 6 })
      if (prop === 'fillText') return (s: string) => texts.push(s)
      if (prop === 'globalAlpha') return target.globalAlpha ?? 1
      if (typeof target[prop] !== 'undefined') return target[prop]
      return () => {}
    },
    set(target, prop: string, value) {
      target[prop] = value
      return true
    },
  }) as unknown as CanvasRenderingContext2D
  return { ctx, texts }
}

/** La carte ET la voix, pour une même liste de joueurs masqués. */
function mount(initial: HiddenPlayers | null) {
  return renderHook(
    ({ hidden }: { hidden: HiddenPlayers | null }) => {
      const focus = useReplayPlayerFocus({ doc: DOC, focus: null, hidden, followPlayer: null, frameRef: { current: 15 }, zoom: ZOOM })
      const chatter = useReplayChatter({
        doc: DOC, locale: 'fr', view: VIEW, showBubbles: true, showIcon: false,
        ink: { fill: '#fff', text: '#000' }, fallbackColor: '#888', identity: { colorOfXuid: () => '#abcdef' },
        shownAuthor: focus.shown?.author, reducedMotion: false,
      })
      const voices = useChatterVoices(DOC, 'fr', hidden)
      return { chatter, voices }
    },
    { initialProps: { hidden: initial } },
  )
}

/** Les phrases dont la bulle se dessine à l'image 15, et les voix que la piste peut jouer. */
function seen(r: ReturnType<typeof mount>['result']): { bubbles: string[]; voices: string[] } {
  const { ctx, texts } = recorder()
  r.current.chatter.layers[0].paint(ctx, 15, 1)
  return { bubbles: texts.sort(), voices: r.current.voices.map((v) => v.stem ?? '').sort() }
}

const BOTH = { bubbles: ['Parole de A', 'Parole de B'], voices: ['repliques/fr/0000000a', 'repliques/fr/0000000b'] }

beforeEach(() => {
  hoisted.reply = TABLE
  localStorage.clear()
})

describe('Seul / Masquer sur les répliques', () => {
  it('aucun joueur masqué : les deux joueurs ont leur bulle et leur voix', async () => {
    const { result } = mount(null)
    await waitFor(() => expect(result.current.voices).toHaveLength(2))
    expect(seen(result)).toEqual(BOTH)
  })

  it('un joueur masqué n’a ni bulle ni voix ; l’autre garde les siennes', async () => {
    const { result } = mount([A])
    await waitFor(() => expect(result.current.chatter.available.count).toBe(2))
    expect(seen(result)).toEqual({ bubbles: ['Parole de B'], voices: ['repliques/fr/0000000b'] })
  })

  it('« Seul A » (B masqué) : seules la bulle et la voix de A restent', async () => {
    const { result } = mount([B])
    await waitFor(() => expect(result.current.chatter.available.count).toBe(2))
    expect(seen(result)).toEqual({ bubbles: ['Parole de A'], voices: ['repliques/fr/0000000a'] })
  })

  it('les deux masqués : plus aucune bulle ni voix, mais le film porte toujours ses répliques', async () => {
    const { result } = mount([A, B])
    await waitFor(() => expect(result.current.chatter.available.count).toBe(2))
    expect(seen(result)).toEqual({ bubbles: [], voices: [] })
  })

  it('réafficher les joueurs ramène les bulles et les voix', async () => {
    const { result, rerender } = mount([A])
    await waitFor(() => expect(result.current.chatter.available.count).toBe(2))
    expect(seen(result).bubbles).toEqual(['Parole de B'])
    rerender({ hidden: null })
    expect(seen(result)).toEqual(BOTH)
  })
})
