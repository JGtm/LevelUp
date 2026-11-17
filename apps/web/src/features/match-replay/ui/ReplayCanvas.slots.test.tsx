/**
 * Tests — LES DEUX EMPLACEMENTS DE LA TOILE (`transport`, `blockOverlays`) : la toile reçoit sa barre de
 * lecture et les superpositions de son bloc comme des FONCTIONS DE RENDU, et leur remet un seul objet de
 * commandes (`replayTransportCommands`). Ce que ces tests prouvent :
 *  - la barre du Standard, rendue par l'emplacement, pilote la toile (lecture, pause, saut) ;
 *  - une AUTRE barre, minimale, branchée sur le même emplacement, pilote la même toile : la toile ne
 *    connaît pas sa barre ;
 *  - `blockOverlays` rend son contenu dans le bloc de la toile, après la légende, hors de la barre ;
 *  - ni la lecture, ni le changement de fonction de rendu ne remontent la barre : c'est le même nœud DOM.
 */
import { act, cleanup, fireEvent, screen } from '@testing-library/react'
import type { ComponentProps, ReactElement } from 'react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { buildFilmAllegiance } from '@/lib/replay/filmAllegiance'
import type { ReplayDocumentReady } from '@/lib/replay/replayNormalize'

import { renderWithProviders } from '../../../test/render-utils'
import { buildPlayers } from '../../../lib/replay/rosterLogic'
import { REPLAY_TEXT } from '../i18n/i18n'
import { createPlaybackStore } from '../model/playbackStore'
import { goFixtureEntries, loadGoFixture } from '../test/goFixtures'
import { recordingContext } from '../test/recordingContext'
import { testReplayDoc } from '../test/testDoc'
import { canvasSlotsOf } from './frames/replayFrameSlots'
import { ReplayCanvas } from './ReplayCanvas'
import type { ReplayCommandsRender, ReplayTransportCommands } from './replayTransportCommands'

const T = REPLAY_TEXT.fr

function smallDoc(): ReplayDocumentReady {
  const smallest = [...goFixtureEntries()].sort((a, b) => a.bytes - b.bytes)[0]
  return testReplayDoc(loadGoFixture(smallest).doc)
}

/** Monte la toile avec les emplacements donnés ; le reste des propriétés est le minimum requis. */
function canvasWith(doc: ReplayDocumentReady, transport: ReplayCommandsRender, blockOverlays?: ReplayCommandsRender): ReactElement {
  return <ReplayCanvas {...baseProps(doc)} transport={transport} blockOverlays={blockOverlays} />
}

/** Les propriétés requises, mémorisées par document : deux montages du même document reçoivent les mêmes valeurs. */
const BASE_PROPS = new WeakMap<ReplayDocumentReady, BaseProps>()
type BaseProps = Omit<ComponentProps<typeof ReplayCanvas>, 'transport' | 'blockOverlays'>

function baseProps(doc: ReplayDocumentReady): BaseProps {
  const known = BASE_PROPS.get(doc)
  if (known) return known
  const players = buildPlayers(doc, [])
  const viewpoint = players.find((p) => p.team !== undefined && p.team >= 0 && !p.xuid.startsWith('bot:'))?.xuid ?? null
  const props: BaseProps = {
    doc,
    locale: 'fr',
    playWindow: null,
    playbackStore: createPlaybackStore(),
    allegiance: buildFilmAllegiance(players, viewpoint),
    players,
    viewpoint,
    fullscreen: { active: false, disabled: false, toggle: () => {} },
    frameControl: { toggleCinema: () => {}, toggleFullFrame: () => {} },
  }
  BASE_PROPS.set(doc, props)
  return props
}

beforeEach(() => {
  try {
    localStorage.clear()
  } catch {
    // Pas de stockage : les réglages gardent leurs défauts, ce que ces tests supposent déjà.
  }
  // Aucune boucle d'animation : la lecture change d'état, l'image ne bouge que sur un saut.
  vi.stubGlobal('requestAnimationFrame', () => 0)
  vi.stubGlobal('cancelAnimationFrame', () => {})
  // Une largeur mesurée : sans elle la toile ne peint rien et l'horloge reste vide.
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

/** Une barre minimale, qui ne connaît de la toile que l'objet de commandes. */
const minimalBar: ReplayCommandsRender = (c) => (
  <div data-testid="minimal-bar">
    <button type="button" onClick={c.onTogglePlay}>
      {c.playing ? 'minimal-pause' : 'minimal-lecture'}
    </button>
    <button type="button" onClick={() => c.onSeekBy(10)}>
      minimal-avance
    </button>
    <button type="button" onClick={() => c.onSetSpeed(2)}>
      minimal-vitesse
    </button>
    <span data-testid="minimal-speed">{c.speed}</span>
    <span data-testid="minimal-clock" ref={c.clockRef} />
  </div>
)

describe('ReplayCanvas : la barre du Standard passe par l’emplacement `transport`', () => {
  it('elle pilote la toile : lecture, pause, saut', () => {
    const doc = smallDoc()
    renderWithProviders(canvasWith(doc, canvasSlotsOf('standard').transport))
    expect(screen.getByRole('button', { name: 'Lecture' })).toBeTruthy()
    fireEvent.click(screen.getByRole('button', { name: 'Lecture' }))
    expect(screen.getByRole('button', { name: 'Pause' })).toBeTruthy()
    fireEvent.click(screen.getByRole('button', { name: 'Pause' }))
    expect(screen.getByRole('button', { name: 'Lecture' })).toBeTruthy()

    const clock = () => screen.getByText(/^\d+:\d{2} \/ \d+:\d{2}$/).textContent
    const before = clock()
    fireEvent.click(screen.getByRole('button', { name: /Avancer de 10 s/ }))
    expect(clock()).not.toBe(before)
  })

  it('le Cinéma et le Plein cadre posent la même barre', () => {
    expect(canvasSlotsOf('cinema').transport).toBe(canvasSlotsOf('standard').transport)
    expect(canvasSlotsOf('plein-cadre').transport).toBe(canvasSlotsOf('standard').transport)
  })
})

describe('ReplayCanvas : une autre barre pilote la même toile', () => {
  it('lecture, saut et vitesse passent par l’objet de commandes, sans que la toile connaisse la barre', () => {
    renderWithProviders(canvasWith(smallDoc(), minimalBar))
    expect(screen.queryByRole('button', { name: 'Lecture' })).toBeNull()
    expect(screen.getByRole('button', { name: 'minimal-lecture' })).toBeTruthy()
    fireEvent.click(screen.getByRole('button', { name: 'minimal-lecture' }))
    expect(screen.getByRole('button', { name: 'minimal-pause' })).toBeTruthy()

    const clock = () => screen.getByTestId('minimal-clock').textContent ?? ''
    const before = clock()
    fireEvent.click(screen.getByRole('button', { name: 'minimal-avance' }))
    expect(clock()).not.toBe(before)

    expect(screen.getByTestId('minimal-speed').textContent).toBe('1')
    fireEvent.click(screen.getByRole('button', { name: 'minimal-vitesse' }))
    expect(screen.getByTestId('minimal-speed').textContent).toBe('2')
  })

  it('l’objet de commandes porte ce que la barre actuelle reçoit, zoom compris', () => {
    const seen: ReplayTransportCommands[] = []
    renderWithProviders(
      canvasWith(smallDoc(), (c) => {
        seen.push(c)
        return null
      }),
    )
    const commands = seen[seen.length - 1]
    expect(Object.keys(commands).sort()).toEqual(
      [
        'autoPlay', 'capture', 'clockRef', 'fullscreen', 'locale', 'onRestart', 'onSeekBy', 'onSetSpeed',
        'onTogglePlay', 'onToggleAutoPlay', 'playing', 'sound', 'speed', 'timeline', 'zoom',
      ].sort(),
    )
    expect(typeof commands.zoom.zoomIn).toBe('function')
  })
})

describe('ReplayCanvas : l’emplacement `blockOverlays`', () => {
  it('rend son contenu dans le bloc de la toile, hors de la barre', () => {
    const { container } = renderWithProviders(
      canvasWith(smallDoc(), minimalBar, (c) => <div data-testid="overlay">{c.locale}</div>),
    )
    const overlay = screen.getByTestId('overlay')
    expect(overlay.textContent).toBe('fr')
    const canvas = container.querySelector('canvas')!
    const block = canvas.parentElement!.parentElement!
    expect(overlay.parentElement).toBe(block)
    expect(screen.getByTestId('minimal-bar').contains(overlay)).toBe(false)
    expect(block.contains(screen.getByTestId('minimal-bar'))).toBe(false)
  })

  it('absent, il ne rend rien', () => {
    renderWithProviders(canvasWith(smallDoc(), minimalBar))
    expect(screen.queryByTestId('overlay')).toBeNull()
  })

  it('la commande de zoom flottante est celle que les cadres y posent, et elle pilote le cadrage', () => {
    const { container } = renderWithProviders(
      canvasWith(smallDoc(), minimalBar, canvasSlotsOf('standard').blockOverlays),
    )
    const group = screen.getByRole('group', { name: T.zoomGroup })
    const block = container.querySelector('canvas')!.parentElement!.parentElement!
    expect(group.parentElement).toBe(block)
    expect(screen.getByText(T.zoomLevelFmt(1))).toBeTruthy()
    fireEvent.click(screen.getByRole('button', { name: T.zoomIn }))
    expect(screen.queryByText(T.zoomLevelFmt(1))).toBeNull()
  })
})

describe('ReplayCanvas : la barre n’est jamais remontée', () => {
  it('la barre du Standard garde ses nœuds pendant la lecture et à chaque nouvelle fonction de rendu', () => {
    const doc = smallDoc()
    const bar = canvasSlotsOf('standard').transport
    const view = renderWithProviders(canvasWith(doc, bar))
    const play = screen.getByRole('button', { name: 'Lecture' })
    const seek = screen.getByRole('button', { name: /Avancer de 10 s/ })

    fireEvent.click(play)
    expect(screen.getByRole('button', { name: 'Pause' })).toBe(play)
    fireEvent.click(seek)
    expect(screen.getByRole('button', { name: /Avancer de 10 s/ })).toBe(seek)

    // Une fonction de rendu d'identité nouvelle à chaque rendu : le même élément, donc le même nœud.
    act(() => {
      view.rerender(canvasWith(doc, (c) => bar(c)))
    })
    expect(screen.getByRole('button', { name: 'Pause' })).toBe(play)
    expect(screen.getByRole('button', { name: /Avancer de 10 s/ })).toBe(seek)
  })

  it('une autre barre garde aussi son nœud pendant la lecture', () => {
    renderWithProviders(canvasWith(smallDoc(), minimalBar))
    const bar = screen.getByTestId('minimal-bar')
    fireEvent.click(screen.getByRole('button', { name: 'minimal-lecture' }))
    fireEvent.click(screen.getByRole('button', { name: 'minimal-vitesse' }))
    expect(screen.getByTestId('minimal-bar')).toBe(bar)
  })
})
