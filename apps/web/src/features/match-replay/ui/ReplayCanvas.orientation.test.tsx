/**
 * Tests — L'ORIENTATION « AUTO » DE LA TOILE ET LE BOUTON « PLAN TOURNÉ » : sans choix retenu pour la carte,
 * la toile tourne un plan haut dans une boîte large (`useReplayView`) ; la barre d'une ligne propose alors de
 * le redresser, et son clic retient 0 quart de tour pour la carte. Le bouton n'existe ni pour un choix
 * retenu, ni quand le plan est droit, ni dans la barre complète du Standard.
 */
import { cleanup, fireEvent, screen } from '@testing-library/react'
import type { ComponentProps, ReactElement } from 'react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { buildFilmAllegiance } from '@/lib/replay/filmAllegiance'
import type { ReplayDocumentReady } from '@/lib/replay/replayNormalize'

import { renderWithProviders } from '../../../test/render-utils'
import { buildPlayers } from '../../../lib/replay/rosterLogic'
import { REPLAY_TEXT } from '../i18n/i18n'
import { createPlaybackStore } from '../model/playbackStore'
import { makeFrameControl } from '../test/frameControlStub'
import { recordingContext } from '../test/recordingContext'
import { testReplayDoc } from '../test/testDoc'
import { orientationKey } from '../settings/replayPreferences'
import { canvasSlotsOf } from './frames/replayFrameSlots'
import { ReplayCanvas } from './ReplayCanvas'
import type { ReplayCommandsRender } from './replayTransportCommands'

const T = REPLAY_TEXT.fr
const BOUTON = { name: T.frames.rotatedPlan }
const MAP = 'ridgeline'

/** Un plan haut (10 x 40) et un plan carré : la boîte mesurée (960 x 720) tourne le premier, pas le second. */
const HAUT = testReplayDoc({ frameIntervalMs: 50, frameCount: 10, bounds: { minX: 0, minY: 0, maxX: 10, maxY: 40 } })
const CARRE = testReplayDoc({ frameIntervalMs: 50, frameCount: 10, bounds: { minX: 0, minY: 0, maxX: 20, maxY: 20 } })

function canvasWith(doc: ReplayDocumentReady, transport: ReplayCommandsRender): ReactElement {
  const players = buildPlayers(doc, [])
  const props: Omit<ComponentProps<typeof ReplayCanvas>, 'transport'> = {
    doc,
    locale: 'fr',
    playWindow: null,
    playbackStore: createPlaybackStore(),
    allegiance: buildFilmAllegiance(players, null),
    players,
    viewpoint: null,
    fullscreen: { active: false, disabled: false, toggle: () => {} },
    frameControl: makeFrameControl(),
    mapId: MAP,
  }
  return <ReplayCanvas {...props} transport={transport} />
}

beforeEach(() => {
  window.localStorage.clear()
  vi.stubGlobal('requestAnimationFrame', () => 0)
  vi.stubGlobal('cancelAnimationFrame', () => {})
  vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockReturnValue({
    x: 0, y: 0, top: 0, left: 0, bottom: 540, right: 960, width: 960, height: 540, toJSON: () => ({}),
  })
  vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockImplementation(() => recordingContext().ctx as never)
})

afterEach(() => {
  cleanup()
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

describe('ReplayCanvas : « Plan tourné » dans la barre d’une ligne', () => {
  it('l’Auto tourne un plan haut : le bouton est là, avec son infobulle', () => {
    renderWithProviders(canvasWith(HAUT, canvasSlotsOf('cinema').transport))
    const bouton = screen.getByRole('button', BOUTON)
    expect(bouton).toHaveAttribute('title', T.frames.rotatedPlanTip)
    // L'Auto n'écrit rien : seul un clic retient un choix.
    expect(window.localStorage.getItem(orientationKey(MAP))).toBeNull()
  })

  it('son clic retient 0 quart de tour pour la carte : le plan revient droit et le bouton disparaît', () => {
    renderWithProviders(canvasWith(HAUT, canvasSlotsOf('cinema').transport))
    fireEvent.click(screen.getByRole('button', BOUTON))
    expect(window.localStorage.getItem(orientationKey(MAP))).toBe('0')
    expect(screen.queryByRole('button', BOUTON)).toBeNull()
  })

  it('un plan droit n’a pas de bouton', () => {
    renderWithProviders(canvasWith(CARRE, canvasSlotsOf('cinema').transport))
    expect(screen.queryByRole('button', BOUTON)).toBeNull()
  })

  it.each(['0', '1', '2', '3'])('un choix retenu (%s) n’a jamais de bouton, même quand l’Auto tournerait le plan', (valeur) => {
    window.localStorage.setItem(orientationKey(MAP), valeur)
    renderWithProviders(canvasWith(HAUT, canvasSlotsOf('cinema').transport))
    expect(screen.queryByRole('button', BOUTON)).toBeNull()
  })

  it('la barre complète du Standard ne change pas : pas de bouton, même quand l’Auto tourne le plan', () => {
    renderWithProviders(canvasWith(HAUT, canvasSlotsOf('standard').transport))
    expect(screen.queryByRole('button', BOUTON)).toBeNull()
  })

  it('le Plein cadre pose la même barre que le Cinéma : le bouton y est aussi', () => {
    renderWithProviders(canvasWith(HAUT, canvasSlotsOf('plein-cadre').transport))
    expect(screen.getByRole('button', BOUTON)).toBeTruthy()
  })
})
