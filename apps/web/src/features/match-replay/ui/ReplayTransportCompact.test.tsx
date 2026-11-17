/**
 * Tests — ReplayTransportCompact (la barre de lecture d'une ligne du Cinéma et du Plein cadre).
 *
 * Ce qu'ils protègent : toutes les commandes de la barre complète y sont, dans l'ordre de la maquette, et
 * chacune pilote la toile par l'objet de commandes (lecture, pause, recommencer, sauts, vitesse, son, zoom,
 * lecture automatique, capture, export, cadres, plein écran du système) ; l'emplacement de la caméra ne rend
 * rien tant qu'il n'est pas fourni ; un export en cours neutralise la lecture et la frise ; les cadres posent
 * la barre qui leur revient et la commande de zoom flottante n'existe que dans le Standard.
 */
import { fireEvent, render, screen, within } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

import { REPLAY_TEXT } from '../i18n/i18n'
import { makeFrameControl } from '../test/frameControlStub'
import { makeCommands, makeExporter, makeTimeline, makeZoom } from '../test/transportStubs'
import { ReplayTransportCompact, type ReplayTransportCompactProps } from './ReplayTransportCompact'
import { ReplayTransport } from './ReplayTransport'
import { canvasSlotsOf } from './frames/replayFrameSlots'

const T = REPLAY_TEXT.fr

function renderBar(over: Partial<ReplayTransportCompactProps> = {}) {
  const commands = makeCommands(over)
  const view = render(<ReplayTransportCompact {...commands} camera={over.camera} />)
  return { ...view, commands }
}

/** Les noms des boutons de la barre, dans l'ordre du document. */
function buttonNames(container: HTMLElement): string[] {
  return Array.from(container.querySelectorAll('button')).map((b) => b.getAttribute('aria-label') ?? '')
}

describe('ReplayTransportCompact — les commandes, dans l’ordre de la maquette', () => {
  it('son, lecture, recommencer, sauts, lecture automatique, vitesse, zoom, capture, export, cadres, plein écran', () => {
    const { container } = renderBar({
      capture: {
        captureImage: vi.fn(), recordingSupported: true, recording: false, toggleRecording: vi.fn(),
        videoExport: makeExporter({ phase: 'idle', done: 0, total: 0, pct: 0 }),
      },
    })
    expect(buttonNames(container)).toEqual([
      T.sound,
      T.pause,
      `${T.restart} (R)`,
      `${T.skipBackFmt(10)} (←)`,
      `${T.skipForwardFmt(10)} (→)`,
      T.autoPlay,
      T.speed,
      T.zoomOut,
      T.zoomIn,
      T.captureImage,
      T.exportVideo,
      'Standard',
      'Cinéma (T)',
      'Plein cadre (F)',
      T.fullscreenEnter,
    ])
    // Le temps est écrit par la boucle de dessin dans l'élément que la toile fournit ; la frise est le champ de temps.
    expect(screen.getByRole('slider', { name: T.time })).toHaveAttribute('data-replay-timeline')
  })

  it('une rangée, 56 px, des séparateurs verticaux entre les groupes', () => {
    const { container } = renderBar()
    const bar = container.querySelector('[data-replay-transport="compact"]') as HTMLElement
    expect(bar.style.height).toBe('56px')
    expect(bar.querySelectorAll(':scope > span[aria-hidden="true"]').length).toBeGreaterThanOrEqual(3)
  })
})

describe('ReplayTransportCompact — chaque commande pilote la toile', () => {
  it('lecture / pause, recommencer, sauts de ±10 s', () => {
    const { commands } = renderBar()
    fireEvent.click(screen.getByRole('button', { name: T.pause }))
    expect(commands.onTogglePlay).toHaveBeenCalledTimes(1)
    fireEvent.click(screen.getByRole('button', { name: `${T.restart} (R)` }))
    expect(commands.onRestart).toHaveBeenCalledTimes(1)
    fireEvent.click(screen.getByRole('button', { name: `${T.skipBackFmt(10)} (←)` }))
    fireEvent.click(screen.getByRole('button', { name: `${T.skipForwardFmt(10)} (→)` }))
    expect(commands.onSeekBy).toHaveBeenNthCalledWith(1, -10)
    expect(commands.onSeekBy).toHaveBeenNthCalledWith(2, 10)
  })

  it('à l’arrêt, le bouton plein dit « Lecture »', () => {
    renderBar({ playing: false })
    expect(screen.getByRole('button', { name: T.play })).toBeEnabled()
  })

  it('la vitesse se choisit dans le menu', () => {
    const { commands } = renderBar()
    fireEvent.click(screen.getByRole('button', { name: T.speed }))
    fireEvent.click(screen.getByRole('button', { name: /^2×/ }))
    expect(commands.onSetSpeed).toHaveBeenCalledWith(2)
  })

  it('le son, la lecture automatique (interrupteur) et le zoom', () => {
    const { commands } = renderBar({ zoom: makeZoom({ canZoomOut: false }) })
    fireEvent.click(screen.getByRole('button', { name: T.sound }))
    expect(commands.sound.toggle).toHaveBeenCalledTimes(1)
    const auto = screen.getByRole('switch', { name: T.autoPlay })
    expect(auto).toHaveAttribute('aria-checked', 'false')
    fireEvent.click(auto)
    expect(commands.onToggleAutoPlay).toHaveBeenCalledTimes(1)
    // Au palier minimal, « − » est éteint ; « + » zoome.
    expect(screen.getByRole('button', { name: T.zoomOut })).toBeDisabled()
    fireEvent.click(screen.getByRole('button', { name: T.zoomIn }))
    expect(commands.zoom.zoomIn).toHaveBeenCalledTimes(1)
  })

  it('le zoom − agit quand il est permis', () => {
    const { commands } = renderBar()
    fireEvent.click(screen.getByRole('button', { name: T.zoomOut }))
    expect(commands.zoom.zoomOut).toHaveBeenCalledTimes(1)
  })

  it('capture d’image, export vidéo, icônes de cadre et plein écran du système', () => {
    const frameControl = makeFrameControl('cinema')
    const toggle = vi.fn()
    const { commands } = renderBar({
      frameControl,
      fullscreen: { active: false, disabled: false, toggle },
      capture: {
        captureImage: vi.fn(), recordingSupported: true, recording: false, toggleRecording: vi.fn(),
        videoExport: makeExporter({ phase: 'idle', done: 0, total: 0, pct: 0 }),
      },
    })
    fireEvent.click(screen.getByRole('button', { name: T.captureImage }))
    expect(commands.capture.captureImage).toHaveBeenCalledTimes(1)
    fireEvent.click(screen.getByRole('button', { name: T.exportVideo }))
    expect(screen.getByRole('button', { name: T.exportVideo })).toHaveAttribute('aria-expanded', 'true')
    fireEvent.click(screen.getByRole('button', { name: 'Plein cadre (F)' }))
    expect(frameControl.toggleFullFrame).toHaveBeenCalledTimes(1)
    fireEvent.click(screen.getByRole('button', { name: T.fullscreenEnter }))
    expect(toggle).toHaveBeenCalledTimes(1)
  })

  it('l’enregistrement en direct prend la place de l’export quand le navigateur ne sait pas encoder', () => {
    const { commands } = renderBar()
    fireEvent.click(screen.getByRole('button', { name: T.recordVideo }))
    expect(commands.capture.toggleRecording).toHaveBeenCalledTimes(1)
  })
})

describe('ReplayTransportCompact — l’emplacement de la caméra', () => {
  it('absent : aucun bouton « Caméra », le son est le premier de la barre', () => {
    const { container } = renderBar()
    expect(screen.queryByRole('button', { name: 'Caméra' })).toBeNull()
    expect(buttonNames(container)[0]).toBe(T.sound)
  })

  it('fourni : le nœud se pose à gauche du son, dans la barre', () => {
    const { container } = renderBar({ camera: <button type="button" aria-label="Caméra">C</button> })
    expect(buttonNames(container).slice(0, 2)).toEqual(['Caméra', T.sound])
  })
})

describe('ReplayTransportCompact — pendant un export', () => {
  const encoding = { phase: 'encode', done: 300, total: 1200, pct: 25 } as const

  it('la lecture, les sauts et recommencer sont inertes ; l’export affiche le pourcentage', () => {
    const { commands } = renderBar({
      capture: {
        captureImage: vi.fn(), recordingSupported: true, recording: false, toggleRecording: vi.fn(),
        videoExport: makeExporter(encoding),
      },
    })
    for (const name of [T.pause, `${T.restart} (R)`, `${T.skipBackFmt(10)} (←)`, `${T.skipForwardFmt(10)} (→)`]) {
      const button = screen.getByRole('button', { name })
      expect(button).toBeDisabled()
      fireEvent.click(button)
    }
    expect(commands.onTogglePlay).not.toHaveBeenCalled()
    expect(commands.onRestart).not.toHaveBeenCalled()
    expect(commands.onSeekBy).not.toHaveBeenCalled()
    expect(screen.getByRole('button', { name: T.exportVideo }).textContent).toContain('25 %')
  })

  it('la frise est retirée du pointeur et des technologies d’assistance', () => {
    const { container } = renderBar({
      capture: {
        captureImage: vi.fn(), recordingSupported: true, recording: false, toggleRecording: vi.fn(),
        videoExport: makeExporter(encoding),
      },
    })
    const field = container.querySelector('[data-replay-timeline]') as HTMLElement
    const wrapper = field.closest('[aria-hidden="true"]') as HTMLElement
    expect(wrapper).not.toBeNull()
    expect(wrapper.className).toContain('pointer-events-none')
  })

  it('au repos, la frise reste saisissable', () => {
    const { container } = renderBar()
    const field = container.querySelector('[data-replay-timeline]') as HTMLElement
    expect(field.closest('[aria-hidden="true"]')).toBeNull()
  })

  it('les icônes de cadre suivent la règle de ReplayFrameButtons : inertes sauf la sortie du Plein cadre', () => {
    renderBar({
      frameControl: makeFrameControl('plein-cadre', { disabled: true }),
      capture: {
        captureImage: vi.fn(), recordingSupported: true, recording: false, toggleRecording: vi.fn(),
        videoExport: makeExporter(encoding),
      },
    })
    expect(screen.getByRole('button', { name: 'Plein cadre (F)' })).toBeEnabled()
    expect(screen.getByRole('button', { name: 'Standard' })).toBeDisabled()
  })
})

describe('Les cadres posent leur barre : Standard = complète + zoom flottant, Cinéma et Plein cadre = d’une ligne', () => {
  function renderSlots(frame: 'standard' | 'cinema' | 'plein-cadre') {
    const slots = canvasSlotsOf(frame)
    const commands = makeCommands({ frameControl: makeFrameControl(frame) })
    return render(
      <div>
        {slots.transport(commands)}
        <div data-testid="block">{slots.blockOverlays?.(commands)}</div>
      </div>,
    )
  }

  it('Standard : la barre complète et la croix de zoom flottante', () => {
    const { container } = renderSlots('standard')
    expect(container.querySelector('[data-replay-transport="compact"]')).toBeNull()
    expect(within(screen.getByTestId('block')).getByRole('group', { name: T.zoomGroup })).toBeInTheDocument()
  })

  it.each(['cinema', 'plein-cadre'] as const)('%s : la barre d’une ligne, aucune croix de zoom flottante', (frame) => {
    const { container } = renderSlots(frame)
    expect(container.querySelector('[data-replay-transport="compact"]')).not.toBeNull()
    expect(within(screen.getByTestId('block')).queryByRole('group', { name: T.zoomGroup })).toBeNull()
    // Le zoom est dans la barre : « − » et « + » y sont, une seule fois.
    expect(screen.getAllByRole('button', { name: T.zoomIn })).toHaveLength(1)
  })

  it('la barre complète n’est pas la barre d’une ligne : elle ne porte pas de zoom', () => {
    const commands = makeCommands()
    const { zoom, ...complete } = commands
    void zoom
    render(<ReplayTransport {...complete} timeline={makeTimeline()} />)
    expect(screen.queryByRole('button', { name: T.zoomIn })).toBeNull()
  })
})
