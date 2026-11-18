/**
 * Tests — LE FOND RELIEF DANS LA TOILE : le canvas branche `useReplayRelief` sur le panneau de la vue, le
 * fond posé et le calque des traits.
 *
 * Les fichiers et le calculateur du relief sont remplacés par des doubles (la décision, pas le calcul, se teste
 * ici : `useReplayRelief.test.tsx` et la bibliothèque `lib/replay/relief` tiennent le reste). Ce que ces tests
 * prouvent, toile montée et panneau ouvert :
 *  - par défaut (fond Plan) le relief ne coûte que ses données : aucune image des hauteurs, aucune demande de rendu ;
 *  - le panneau propose Plan, Rendu, Relief et la commande d'étage ; choisir Relief le retient et demande le rendu teinté ;
 *  - choisir un étage demande la coupe (qu'aucun joueur ne soit filtré est tenu par `reliefNoPlayerMask.guard.test.ts`) ;
 *  - une lecture du relief en échec retombe sur le plan : plus d'option Relief, ni de commande d'étage.
 */
import { act, cleanup, fireEvent, screen, within } from '@testing-library/react'
import type { ComponentProps } from 'react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { buildFilmAllegiance } from '@/lib/replay/filmAllegiance'
import { fourBlocksScene } from '@/lib/replay/relief/reliefScene.testutil'
import type { ReliefFiles } from '@/lib/replay/reliefQueries'
import type { ReliefRender } from '@/lib/replay/useReliefRender'
import type { ReplayDocumentReady } from '@/lib/replay/replayNormalize'

import { buildPlayers } from '../../../lib/replay/rosterLogic'
import { renderWithProviders } from '../../../test/render-utils'
import type { ReplayMapBackgroundLayer } from '../hooks/useReplayView'
import { createPlaybackStore } from '../model/playbackStore'
import { makeFrameControl } from '../test/frameControlStub'
import { goFixtureEntries, loadGoFixture } from '../test/goFixtures'
import { recordingContext } from '../test/recordingContext'
import { testReplayDoc } from '../test/testDoc'
import { canvasSlotsOf } from './frames/replayFrameSlots'
import { ReplayCanvas } from './ReplayCanvas'

const mocks = vi.hoisted(() => ({ files: vi.fn(), render: vi.fn() }))
vi.mock('@/lib/replay/reliefQueries', () => ({ useReplayReliefFiles: mocks.files }))
vi.mock('@/lib/replay/useReliefRender', () => ({ useReliefRender: mocks.render }))

/** ImageData minimal : jsdom n'en fournit pas, et le relief teinté en construit un. */
class FakeImageData {
  readonly data: Uint8ClampedArray
  readonly width: number
  readonly height: number

  constructor(data: Uint8ClampedArray, width: number, height: number) {
    this.data = data
    this.width = width
    this.height = height
  }
}

function smallDoc(): ReplayDocumentReady {
  const smallest = [...goFixtureEntries()].sort((a, b) => a.bytes - b.bytes)[0]
  return testReplayDoc(loadGoFixture(smallest).doc)
}

/** Un fond qui couvre toute la zone jouée (sinon le canvas l'écarte) et porte plan ET relief. */
function background(over: Partial<ReplayMapBackgroundLayer> = {}): ReplayMapBackgroundLayer {
  return {
    calibration: { widthPx: 200_000, heightPx: 200_000, metersPerPixel: 1, originX: -100_000, originY: 100_000, convention: 'x' } as never,
    image: document.createElement('canvas'),
    plan: null,
    variante: 'plan',
    planAvailable: true,
    reliefAvailable: true,
    zonesClip: null,
    ...over,
  }
}

type Props = ComponentProps<typeof ReplayCanvas>

function props(doc: ReplayDocumentReady, bg: ReplayMapBackgroundLayer | null): Props {
  const players = buildPlayers(doc, [])
  const viewpoint = players.find((p) => p.team !== undefined && p.team >= 0 && !p.xuid.startsWith('bot:'))?.xuid ?? null
  return {
    doc,
    locale: 'fr',
    playWindow: null,
    playbackStore: createPlaybackStore(),
    allegiance: buildFilmAllegiance(players, viewpoint),
    players,
    viewpoint,
    fullscreen: { active: false, disabled: false, toggle: () => {} },
    frameControl: makeFrameControl(),
    background: bg,
    playerSlug: 'jg',
    matchId: 'm1',
    mapId: 'carte-a',
    transport: canvasSlotsOf('standard').transport,
  }
}

function readyFiles(): ReliefFiles {
  return { data: fourBlocksScene().data, blob: new Blob(), loading: false, error: null }
}

function readyRender(): ReliefRender {
  const empty = { points: new Float32Array(0), offsets: new Uint32Array(1) }
  return {
    status: 'ready',
    info: { width: 40, height: 10, hasMesh: true },
    frame: {
      width: 40, height: 10, cut: null, image: new Uint8ClampedArray(40 * 10 * 4),
      alpha: new Uint8Array(400), planAlpha: new Uint8Array(400), lines: { steps: empty, contours: empty, walls: empty },
    },
    error: null,
  }
}

const lastRequest = () => mocks.render.mock.calls.at(-1)?.[2] ?? null
const lastNeedImage = (): boolean => mocks.files.mock.calls.at(-1)?.[3]

function openPanel() {
  fireEvent.click(screen.getByRole('button', { name: /^Vue/ }))
  return screen.getByRole('region', { name: 'Vue' })
}

beforeEach(() => {
  try {
    localStorage.clear()
  } catch {
    // Pas de stockage : les réglages gardent leurs défauts, ce que ces tests supposent déjà.
  }
  mocks.files.mockReset()
  mocks.render.mockReset()
  mocks.files.mockReturnValue(readyFiles())
  mocks.render.mockReturnValue(readyRender())
  vi.stubGlobal('requestAnimationFrame', () => 0)
  vi.stubGlobal('cancelAnimationFrame', () => {})
  vi.stubGlobal('ImageData', FakeImageData)
  vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockReturnValue({
    x: 0, y: 0, top: 0, left: 0, bottom: 540, right: 960, width: 960, height: 540, toJSON: () => ({}),
  })
  vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockImplementation(() => recordingContext().ctx as never)
  vi.spyOn(console, 'warn').mockImplementation(() => {})
})

afterEach(() => {
  cleanup()
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

describe('ReplayCanvas — le fond Plan par défaut', () => {
  it('ne lit que les données du relief : ni image des hauteurs, ni demande de rendu', () => {
    renderWithProviders(<ReplayCanvas {...props(smallDoc(), background())} />)
    expect(mocks.files.mock.calls.at(-1)?.slice(0, 3)).toEqual(['jg', 'm1', true])
    expect(lastNeedImage()).toBe(false)
    expect(lastRequest()).toBeNull()
  })

  it('un fond sans relief ne demande rien du tout', () => {
    renderWithProviders(<ReplayCanvas {...props(smallDoc(), background({ reliefAvailable: false }))} />)
    expect(mocks.files.mock.calls.at(-1)?.[2]).toBe(false)
  })
})

describe('ReplayCanvas — le panneau de la vue', () => {
  it('propose Plan, Rendu, Relief, le Plan retenu, les traits éteints et la commande d’étage en rangs', () => {
    renderWithProviders(<ReplayCanvas {...props(smallDoc(), background())} />)
    const panel = openPanel()
    const group = within(panel).getByRole('radiogroup', { name: 'Fond' })
    expect(within(group).getAllByRole('radio').map((r) => r.textContent)).toEqual(['Plan', 'Rendu', 'Relief'])
    expect(within(panel).getByRole('radio', { name: 'Plan' })).toHaveAttribute('aria-checked', 'true')
    expect(within(panel).getByRole('switch', { name: 'Traits du relief' })).toHaveAttribute('aria-checked', 'false')
    expect(within(within(panel).getByRole('group', { name: 'Étage' })).getAllByRole('button').map((b) => b.textContent)).toEqual(['Tout', 'Bas', 'Étage 1'])
  })

  it('choisir Relief le retient, allume les traits d’office et demande le rendu teinté, sans masque', () => {
    renderWithProviders(<ReplayCanvas {...props(smallDoc(), background())} />)
    fireEvent.click(within(openPanel()).getByRole('radio', { name: 'Relief' }))
    expect(localStorage.getItem('replay-background-style')).toBe('relief')
    expect(lastNeedImage()).toBe(true)
    const request = lastRequest()
    expect(request.base).not.toBeNull()
    expect(request).toMatchObject({ cut: null, lines: true, masks: false })
    expect(within(screen.getByRole('region', { name: 'Vue' })).getByRole('switch', { name: 'Traits du relief' })).toHaveAttribute('aria-checked', 'true')
  })

  it('choisir un étage demande la coupe, avec les masques sur un fond Plan (le garde-rail dit qu’aucun joueur n’est filtré)', () => {
    renderWithProviders(<ReplayCanvas {...props(smallDoc(), background())} />)
    fireEvent.click(within(openPanel()).getByRole('button', { name: 'Étage 1' }))
    expect(lastRequest()).toMatchObject({ cut: 2.2, base: null, masks: true })
  })

  it('une lecture du relief en échec replie sur le Plan : plus d’option Relief ni de commande d’étage', () => {
    mocks.render.mockReturnValue({ status: 'failed', info: null, frame: null, error: new Error('data-invalid') })
    renderWithProviders(<ReplayCanvas {...props(smallDoc(), background())} />)
    const panel = openPanel()
    expect(within(within(panel).getByRole('radiogroup', { name: 'Fond' })).getAllByRole('radio').map((r) => r.textContent)).toEqual(['Plan', 'Rendu'])
    expect(within(panel).queryByRole('group', { name: 'Étage' })).toBeNull()
    expect(within(panel).queryByRole('switch', { name: 'Traits du relief' })).toBeNull()
  })

  it('un fond sans relief garde le panneau d’avant : Plan et Rendu, aucune commande de relief', () => {
    mocks.files.mockReturnValue({ data: null, blob: null, loading: false, error: null })
    mocks.render.mockReturnValue({ status: 'idle', info: null, frame: null, error: null })
    renderWithProviders(<ReplayCanvas {...props(smallDoc(), background({ reliefAvailable: false }))} />)
    const panel = openPanel()
    expect(within(within(panel).getByRole('radiogroup', { name: 'Fond' })).getAllByRole('radio').map((r) => r.textContent)).toEqual(['Plan', 'Rendu'])
    expect(within(panel).queryByRole('slider')).toBeNull()
  })
})

describe('ReplayCanvas — le fond Relief posé', () => {
  it('le relief voulu se peint : le fond est un canvas, les traits sont cuits', () => {
    localStorage.setItem('replay-background-style', 'relief')
    const { container } = renderWithProviders(<ReplayCanvas {...props(smallDoc(), background())} />)
    act(() => {})
    expect(container.querySelector('canvas')).not.toBeNull()
    expect(lastRequest()).toMatchObject({ lines: true, masks: false })
    expect(lastRequest().base).not.toBeNull()
  })
})
