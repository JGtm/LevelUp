/**
 * Tests — LE FORMAT DES COMMANDES DE LA BARRE D'UNE LIGNE (Cinéma, Plein cadre) : chaque commande ordinaire est un
 * carré de 30 px au rayon de 8 px, sans bordure ni fond au repos (le fond ne vient qu'au survol) ; le bouton de
 * lecture reste rond et plein ; la capture et l'export sont deux boutons ordinaires, sans cartouche, qui gardent
 * leur progression, leur panneau et leur repli d'enregistrement ; l'état marqué (cadre courant, caméra qui suit)
 * garde son fond ; le curseur de volume reste visible. La barre complète du Standard garde ses formes rondes.
 */
import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

import { REPLAY_TEXT } from '../i18n/i18n'
import { COTES_BARRE_COMPACTE as C } from '../model/cotesBarreCompacte'
import { makeFrameControl } from '../test/frameControlStub'
import { makeCommands, makeExporter } from '../test/transportStubs'
import { ReplayCameraMenu } from './ReplayCameraMenu'
import { ReplayTransport } from './ReplayTransport'
import { ReplayTransportCompact, type ReplayTransportCompactProps } from './ReplayTransportCompact'

const T = REPLAY_TEXT.fr

const WITH_EXPORT = {
  captureImage: vi.fn(), recordingSupported: true, recording: false, toggleRecording: vi.fn(),
  videoExport: makeExporter({ phase: 'idle', done: 0, total: 0, pct: 0 }),
}

function renderBar(over: Partial<ReplayTransportCompactProps> = {}) {
  return render(<ReplayTransportCompact {...makeCommands({ autoRotation: null, ...over })} camera={over.camera} />)
}

/** Les classes de fond posées AU REPOS (hors variantes `hover:`). */
function restingBackgrounds(button: HTMLElement): string[] {
  return Array.from(button.classList).filter((c) => c.startsWith('bg-'))
}

/** Les commandes ordinaires de la barre : tout bouton sauf la lecture. */
function ordinaryButtons(container: HTMLElement): HTMLButtonElement[] {
  return Array.from(container.querySelectorAll('button')).filter(
    (b) => b.getAttribute('aria-label') !== T.pause && b.getAttribute('aria-label') !== T.play,
  )
}

describe('barre d’une ligne : le format des commandes', () => {
  it('chaque commande ordinaire est un carré de 30 px au rayon de 8 px, sans bordure ni arrondi complet', () => {
    const { container } = renderBar({ capture: WITH_EXPORT })
    const buttons = ordinaryButtons(container)
    expect(buttons.length).toBeGreaterThanOrEqual(12)
    for (const b of buttons) {
      const label = b.getAttribute('aria-label')
      expect(b.style.height, `${label}`).toBe(`${C.boutonPx}px`)
      expect(b.style.borderRadius, `${label}`).toBe(`${C.boutonRayonPx}px`)
      // Les commandes à valeur écrite (vitesse, export) peuvent s'élargir ; les autres sont des carrés.
      expect(b.style.width === `${C.boutonPx}px` || b.style.minWidth === `${C.boutonPx}px`, `${label}`).toBe(true)
      expect(b.classList.contains('rounded-full'), `${label}`).toBe(false)
      expect(b.classList.contains('border'), `${label}`).toBe(false)
    }
  })

  it('aucune commande n’a de fond au repos, sauf l’état marqué (le cadre courant)', () => {
    const { container } = renderBar({ capture: WITH_EXPORT, frameControl: makeFrameControl('cinema'), autoPlay: false })
    for (const b of ordinaryButtons(container)) {
      const label = b.getAttribute('aria-label')
      expect(restingBackgrounds(b), `${label}`).toEqual(label === 'Cinéma (T)' ? ['bg-accent'] : [])
    }
  })

  it('le cadre courant garde son fond et son marquage (aria-pressed) ; les autres cadres, non', () => {
    renderBar({ frameControl: makeFrameControl('plein-cadre') })
    expect(screen.getByRole('button', { name: 'Plein cadre (F)' })).toHaveAttribute('aria-pressed', 'true')
    expect(screen.getByRole('button', { name: 'Plein cadre (F)' }).classList.contains('bg-accent')).toBe(true)
    expect(screen.getByRole('button', { name: 'Cinéma (T)' })).toHaveAttribute('aria-pressed', 'false')
    expect(screen.getByRole('button', { name: 'Cinéma (T)' }).classList.contains('bg-accent')).toBe(false)
  })

  it('la lecture reste le rond plein de 36 px', () => {
    renderBar()
    const play = screen.getByRole('button', { name: T.pause })
    expect(play.style.width).toBe(`${C.lecturePx}px`)
    expect(play.style.height).toBe(`${C.lecturePx}px`)
  })

  it('la vitesse écrit sa valeur seule, sans chevron', () => {
    renderBar({ speed: 2 })
    const speed = screen.getByRole('button', { name: T.speed })
    expect(speed.textContent).toBe('2×')
    expect(speed.querySelector('svg')).toBeNull()
  })

  it('les sauts s’écrivent à 12,5 px', () => {
    renderBar()
    const skip = screen.getByRole('button', { name: `${T.skipForwardFmt(10)} (→)` })
    expect(skip.querySelector('span')?.className).toContain('text-[12.5px]')
  })

  it('le curseur de volume reste visible', () => {
    renderBar()
    expect(screen.getByRole('slider', { name: T.soundVolume })).toBeInTheDocument()
  })
})

describe('barre d’une ligne : la capture et l’export sont deux boutons ordinaires', () => {
  it('pas de cartouche : le conteneur n’a ni bordure, ni fond, ni arrondi complet', () => {
    renderBar({ capture: WITH_EXPORT })
    const holder = screen.getByRole('button', { name: T.captureImage }).parentElement as HTMLElement
    expect(holder.classList.contains('relative')).toBe(true)
    for (const cls of ['border', 'rounded-full', 'bg-muted/40', 'p-1']) expect(holder.classList.contains(cls)).toBe(false)
    expect(holder.contains(screen.getByRole('button', { name: T.exportVideo }))).toBe(true)
  })

  it('l’export ouvre son panneau, ancré sur le conteneur', () => {
    renderBar({ capture: WITH_EXPORT })
    const exportButton = screen.getByRole('button', { name: T.exportVideo })
    fireEvent.click(exportButton)
    expect(exportButton).toHaveAttribute('aria-expanded', 'true')
    const holder = exportButton.parentElement as HTMLElement
    expect(holder.children.length).toBe(3)
  })

  it('pendant l’export, le bouton s’élargit et porte le pourcentage', () => {
    renderBar({
      capture: { ...WITH_EXPORT, videoExport: makeExporter({ phase: 'encode', done: 5, total: 10, pct: 42 }) },
    })
    const exporting = screen.getByRole('button', { name: T.exportVideo })
    expect(exporting.textContent).toContain('42 %')
    expect(exporting.style.minWidth).toBe(`${C.boutonPx}px`)
    expect(exporting.style.width).toBe('')
  })

  it('le repli d’enregistrement garde sa pastille rouge et son état', () => {
    renderBar({ capture: { ...WITH_EXPORT, videoExport: null, recording: true } })
    const rec = screen.getByRole('button', { name: T.stopRecording })
    expect(rec).toHaveAttribute('aria-pressed', 'true')
    expect(rec.classList.contains('text-destructive')).toBe(true)
    expect(rec.style.width).toBe(`${C.boutonPx}px`)
  })
})

describe('le bouton caméra', () => {
  const groups = [
    { key: 'camp:0', label: 'Cobalt', options: [{ value: 'x1', rosterKey: 'x1', label: 'JGtm', disabled: false, title: 'JGtm' }] },
  ]
  const camera = (followed: string | null) => (
    <ReplayCameraMenu
      timeline={{ viewpoint: 'x1', viewpointGroups: groups, onSelectViewpoint: vi.fn() }}
      cameraFollow={{ player: followed ? { xuid: followed, lives: [] } : null, onFollow: vi.fn() }}
      locale="fr"
    />
  )

  it('carré de 30 px ; fond relevé tant qu’un joueur est suivi, aucun fond sinon', () => {
    const { unmount } = render(camera(null))
    const idle = screen.getByRole('button', { name: /^Caméra/ })
    expect(idle.style.width).toBe(`${C.boutonPx}px`)
    expect(idle.style.borderRadius).toBe(`${C.boutonRayonPx}px`)
    expect(restingBackgrounds(idle)).toEqual([])
    unmount()
    render(camera('x1'))
    expect(restingBackgrounds(screen.getByRole('button', { name: /^Caméra/ }))).toEqual(['bg-accent'])
  })
})

describe('la barre complète du Standard garde ses formes', () => {
  it('boutons ronds, sauts bordés, cartouche de capture et d’export', () => {
    render(<ReplayTransport {...makeCommands({ capture: WITH_EXPORT })} />)
    const skip = screen.getByRole('button', { name: `${T.skipForwardFmt(10)} (→)` })
    expect(skip.className).toContain('rounded-full')
    expect(skip.className).toContain('border')
    const holder = screen.getByRole('button', { name: T.captureImage }).parentElement as HTMLElement
    expect(holder.className).toContain('rounded-full')
    expect(holder.className).toContain('border')
  })
})
