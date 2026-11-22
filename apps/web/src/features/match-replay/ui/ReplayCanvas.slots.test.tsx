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
import type { ComponentProps } from 'react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import type { ReplayDocumentReady } from '@/lib/replay/replayNormalize'
import type { ReplayPlayer } from '@/lib/replay/rosterLogic'

import { renderWithProviders } from '../../../test/render-utils'
import { REPLAY_TEXT } from '../i18n/i18n'
import { FOLLOW_MIN_LEVEL } from '../model/followLogic'
import { usesCompactBar } from '../model/replayFrame'
import { baseProps, canvasWith, installCanvasDoubles, smallDoc } from '../test/canvasTestKit'
import { makeFrameControl } from '../test/frameControlStub'
import { canvasSlotsOf } from './frames/replayFrameSlots'
import { ReplayCanvas } from './ReplayCanvas'
import type { ReplayCommandsRender, ReplaySlotCommands } from './replayTransportCommands'

const T = REPLAY_TEXT.fr

beforeEach(installCanvasDoubles)

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

  it('le Cinéma et le Plein cadre posent la barre d’une ligne, sans zoom flottant ; le Standard garde la barre complète et son zoom', () => {
    const complete = canvasSlotsOf('standard')
    const compacte = canvasSlotsOf('cinema')
    expect(complete.blockOverlays).toBeDefined()
    expect(compacte.transport).not.toBe(complete.transport)
    expect(compacte.blockOverlays).toBeUndefined()
    expect(canvasSlotsOf('plein-cadre')).toBe(compacte)
    // Le modèle décide, pas un nom de cadre : chaque cadre reçoit le jeu que `usesCompactBar` désigne.
    for (const frame of ['standard', 'cinema', 'plein-cadre'] as const) {
      expect(canvasSlotsOf(frame)).toBe(usesCompactBar(frame) ? compacte : complete)
    }
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
    const seen: ReplaySlotCommands[] = []
    renderWithProviders(
      canvasWith(smallDoc(), (c) => {
        seen.push(c)
        return null
      }),
    )
    const commands = seen[seen.length - 1]
    expect(Object.keys(commands).sort()).toEqual(
      [
        'autoPlay', 'cameraFollow', 'capture', 'clockRef', 'frameControl', 'fullscreen', 'locale', 'onRestart', 'onSeekBy',
        'onSetSpeed', 'onTogglePlay', 'onToggleAutoPlay', 'planFeedShown', 'playing', 'sound', 'speed', 'timeline', 'zoom', 'autoRotation', 'thinBar',
      ].sort(),
    )
    expect(typeof commands.zoom.zoomIn).toBe('function')
    // La caméra : sans page autour de la toile, personne n'est suivi et le geste est inerte.
    expect(commands.cameraFollow.player).toBeNull()
    expect(commands.planFeedShown).toBe(false)
    expect(commands.autoRotation).toBeNull()
  })
})

describe('ReplayCanvas : la barre se pose sur le plan au Plein cadre, préférence allumée', () => {
  const dockOf = () => screen.getByTestId('minimal-bar').parentElement as HTMLElement

  it('Plein cadre, préférence par défaut (allumée) : la barre est posée en bas du bloc, dans son liseré, hors du flux', () => {
    renderWithProviders(
      <ReplayCanvas {...baseProps(smallDoc())} frameControl={makeFrameControl('plein-cadre')} transport={minimalBar} />,
    )
    expect(dockOf().className).toContain('absolute')
    expect(dockOf().className).toContain('bottom-px')
  })

  it('Plein cadre, préférence éteinte : la barre reste dans le flux, sous le plan', () => {
    localStorage.setItem('replay-thin-bar-full-frame', 'false')
    renderWithProviders(
      <ReplayCanvas {...baseProps(smallDoc())} frameControl={makeFrameControl('plein-cadre')} transport={minimalBar} />,
    )
    expect(dockOf().className).not.toContain('absolute')
  })

  it.each(['cinema', 'plein-cadre'] as const)('%s : la barre est SOUS le bloc arrondi du plan, hors du bloc, quand elle n’est pas posée sur le plan', (frame) => {
    localStorage.setItem('replay-thin-bar-full-frame', 'false')
    const { container } = renderWithProviders(
      <ReplayCanvas {...baseProps(smallDoc())} frameControl={makeFrameControl(frame)} transport={minimalBar} />,
    )
    const bloc = container.querySelector('[data-replay-plan-block]') as HTMLElement
    expect(bloc.className).toContain('rounded-lg')
    expect(bloc.contains(container.querySelector('canvas'))).toBe(true)
    expect(bloc.contains(screen.getByTestId('minimal-bar'))).toBe(false)
    expect(bloc.nextElementSibling?.contains(screen.getByTestId('minimal-bar'))).toBe(true)
  })

  it.each(['standard', 'cinema'] as const)('%s : la barre reste dans le flux, quelle que soit la préférence', (frame) => {
    renderWithProviders(<ReplayCanvas {...baseProps(smallDoc())} frameControl={makeFrameControl(frame)} transport={minimalBar} />)
    expect(dockOf().className).not.toContain('absolute')
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

describe('ReplayCanvas : la barre d’une ligne (Cinéma, Plein cadre) pilote la toile', () => {
  const compact = canvasSlotsOf('cinema').transport
  const clock = () => screen.getByText(/^\d+:\d{2} \/ \d+:\d{2}$/).textContent

  it('lecture, pause et saut passent par l’objet de commandes', () => {
    renderWithProviders(canvasWith(smallDoc(), compact))
    fireEvent.click(screen.getByRole('button', { name: 'Lecture' }))
    expect(screen.getByRole('button', { name: 'Pause' })).toBeTruthy()
    fireEvent.click(screen.getByRole('button', { name: 'Pause' }))
    const before = clock()
    fireEvent.click(screen.getByRole('button', { name: /Avancer de 10 s/ }))
    expect(clock()).not.toBe(before)
  })

  it('aller à un instant : le champ de la frise compacte déplace l’horloge et pose la progression sur sa racine', () => {
    const { container } = renderWithProviders(canvasWith(smallDoc(), compact))
    const field = container.querySelector<HTMLInputElement>('input[data-replay-timeline]')!
    expect(field).not.toBeNull()
    const before = clock()
    fireEvent.change(field, { target: { value: field.max } })
    expect(clock()).not.toBe(before)
    const host = field.closest<HTMLElement>('[data-replay-cursor-host]')!
    expect(host.style.getPropertyValue('--played-r')).toBe('1')
  })

  it('le zoom de la barre pilote le cadrage de la carte', () => {
    renderWithProviders(canvasWith(smallDoc(), compact))
    expect(screen.getByRole('button', { name: T.zoomOut })).toBeDisabled()
    fireEvent.click(screen.getByRole('button', { name: T.zoomIn }))
    expect(screen.getByRole('button', { name: T.zoomOut })).toBeEnabled()
  })

  it('rien n’est remonté pendant la lecture : mêmes nœuds de barre, de frise et de champ', () => {
    const { container } = renderWithProviders(canvasWith(smallDoc(), compact))
    const play = screen.getByRole('button', { name: 'Lecture' })
    const field = container.querySelector('input[data-replay-timeline]')
    const bar = container.querySelector('[data-replay-transport="compact"]')
    fireEvent.click(play)
    fireEvent.click(screen.getByRole('button', { name: /Avancer de 10 s/ }))
    fireEvent.click(screen.getByRole('button', { name: T.zoomIn }))
    expect(screen.getByRole('button', { name: 'Pause' })).toBe(play)
    expect(container.querySelector('input[data-replay-timeline]')).toBe(field)
    expect(container.querySelector('[data-replay-transport="compact"]')).toBe(bar)
  })

  it('la lecture automatique est un interrupteur de la barre', () => {
    renderWithProviders(canvasWith(smallDoc(), compact))
    const toggle = screen.getByRole('switch', { name: T.autoPlay })
    const was = toggle.getAttribute('aria-checked')
    fireEvent.click(toggle)
    expect(screen.getByRole('switch', { name: T.autoPlay }).getAttribute('aria-checked')).not.toBe(was)
  })
})

describe('ReplayCanvas : la caméra suit le joueur de `cameraFollow`, fiche ouverte ou fermée', () => {
  /** Un joueur dont une vie couvre toute la durée du film : la carte a toujours une position à suivre. */
  function followable(doc: ReplayDocumentReady): ReplayPlayer {
    const end = doc.frameCount
    const points = [{ t: 0, x: 10, y: 20 }, { t: end, x: 30, y: 40 }]
    return { xuid: 'suivi', lives: [{ slot: 1, team: 0, xuid: 'suivi', startFrame: 0, endFrame: end, points }] as never }
  }

  function mount(doc: ReplayDocumentReady, extra: Partial<ComponentProps<typeof ReplayCanvas>>) {
    const seen: ReplaySlotCommands[] = []
    renderWithProviders(
      <ReplayCanvas
        {...baseProps(doc)}
        transport={(c) => {
          seen.push(c)
          return null
        }}
        {...extra}
      />,
    )
    return () => seen[seen.length - 1]
  }

  it('personne n’est suivi : la carte reste entière', () => {
    const doc = smallDoc()
    const last = mount(doc, {})
    expect(last().zoom.level).toBe(1)
  })

  it('un joueur suivi, AUCUNE fiche ouverte (playerFocus absent) : la carte se cadre sur lui', () => {
    const doc = smallDoc()
    const last = mount(doc, { cameraFollow: { player: followable(doc), onFollow: vi.fn() } })
    expect(last().zoom.level).toBeGreaterThanOrEqual(FOLLOW_MIN_LEVEL)
  })

  it('la fiche ouverte d’un AUTRE joueur, avec Seul, ne retire pas la caméra du joueur suivi', () => {
    const doc = smallDoc()
    const players = baseProps(doc).players ?? []
    const suivi = followable(doc)
    const autre = players.find((p) => p !== suivi) ?? suivi
    const last = mount(doc, {
      cameraFollow: { player: suivi, onFollow: vi.fn() },
      playerFocus: { player: autre, solo: true, hidden: false },
    })
    expect(last().zoom.level).toBeGreaterThanOrEqual(FOLLOW_MIN_LEVEL)
  })

  it('une fiche ouverte ne cadre rien sans joueur suivi côté caméra', () => {
    const doc = smallDoc()
    const last = mount(doc, { playerFocus: { player: followable(doc), solo: false, hidden: false } })
    expect(last().zoom.level).toBe(1)
  })

  it('un geste manuel de zoom arrête le suivi : la page est prévenue (onFollowStop) ; le cadrage du suivi, lui, ne l’est pas', () => {
    const doc = smallDoc()
    const onFollowStop = vi.fn()
    const last = mount(doc, { cameraFollow: { player: followable(doc), onFollow: vi.fn() }, onFollowStop })
    expect(onFollowStop).not.toHaveBeenCalled()
    act(() => last().zoom.zoomIn())
    expect(onFollowStop).toHaveBeenCalled()
  })
})
