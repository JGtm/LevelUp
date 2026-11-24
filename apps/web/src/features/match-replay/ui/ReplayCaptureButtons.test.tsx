/**
 * Tests — ReplayCaptureButtons (capture d'image, enregistrement ou export vidéo, panneau d'export).
 *
 * Ce qu'ils protègent : les boutons sont des icônes mais gardent leur nom accessible (un symbole sans
 * nom serait une régression), l'export prend la place de l'enregistrement quand le navigateur sait
 * encoder (et l'enregistrement reste le repli), le bouton d'export SE RÉ-ÉLARGIT pour porter la
 * progression pendant le calcul, et le panneau est monté dans le cartouche positionné qui l'ancre.
 */
import { fireEvent, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'

import type { ReplayCapture } from '../export/useReplayCapture'
import type { ReplayExport } from '../export/useReplayExport'
import type { ReplayLocale } from '../i18n/i18n'
import { ReplayCaptureButtons } from './ReplayCaptureButtons'

function makeExport(over: Partial<ReplayExport> = {}): ReplayExport {
  return {
    supported: true,
    state: { phase: 'idle', done: 0, total: 0, pct: 0 },
    defaultBounds: () => ({ startFrame: 0, endFrame: 100 }),
    run: vi.fn(async () => {}),
    cancel: vi.fn(),
    clockOf: (f) => `0:0${f % 10}`,
    lengthClock: () => '1:40',
    zoomLevel: 1,
    soundOn: true,
    ...over,
  }
}

function makeCapture(over: Partial<ReplayCapture> = {}): ReplayCapture {
  return {
    captureImage: vi.fn(),
    recordingSupported: true,
    recording: false,
    toggleRecording: vi.fn(),
    videoExport: null,
    ...over,
  }
}

function renderButtons(capture: ReplayCapture, locale: ReplayLocale = 'fr') {
  return render(<ReplayCaptureButtons capture={capture} locale={locale} />)
}

describe('ReplayCaptureButtons — capture et enregistrement', () => {
  it('« Capturer l’image » est une icône nommée, et commande la capture', () => {
    const capture = makeCapture()
    renderButtons(capture)
    const btn = screen.getByRole('button', { name: "Capturer l'image" })
    expect(btn.querySelector('svg')).toBeTruthy()
    fireEvent.click(btn)
    expect(capture.captureImage).toHaveBeenCalledTimes(1)
  })

  it('les pastilles sont des icônes nues, mais gardent leur nom accessible long', () => {
    renderButtons(makeCapture())
    const image = screen.getByRole('button', { name: "Capturer l'image" })
    const rec = screen.getByRole('button', { name: 'Enregistrer la vidéo' })
    expect(image.textContent).not.toContain('Image')
    expect(rec.textContent).not.toContain('REC')
    expect(image.getAttribute('title')).toBeTruthy()
    expect(rec.getAttribute('title')).toBeTruthy()
  })

  it('à l’arrêt : le bouton dit « Enregistrer la vidéo » et lance au clic', () => {
    const capture = makeCapture()
    renderButtons(capture)
    const btn = screen.getByRole('button', { name: 'Enregistrer la vidéo' })
    expect(btn).toHaveAttribute('aria-pressed', 'false')
    fireEvent.click(btn)
    expect(capture.toggleRecording).toHaveBeenCalledTimes(1)
  })

  it('en cours : le nom accessible dit ce que le CLIC fera, pas l’état', () => {
    renderButtons(makeCapture({ recording: true }))
    expect(screen.getByRole('button', { name: "Arrêter l'enregistrement" })).toHaveAttribute('aria-pressed', 'true')
  })

  it('navigateur sans enregistrement : PAS de bouton vidéo, mais le bouton image reste', () => {
    renderButtons(makeCapture({ recordingSupported: false }))
    expect(screen.queryByRole('button', { name: 'Enregistrer la vidéo' })).toBeNull()
    expect(screen.getByRole('button', { name: "Capturer l'image" })).toBeTruthy()
  })
})

describe('ReplayCaptureButtons — export hors temps réel', () => {
  it('offre l’export et RETIRE l’enregistrement quand le navigateur sait encoder', () => {
    renderButtons(makeCapture({ videoExport: makeExport() }))
    expect(screen.getByRole('button', { name: 'Exporter la vidéo' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Enregistrer la vidéo' })).not.toBeInTheDocument()
  })

  it('garde l’enregistrement en REPLI quand l’export n’est pas possible', () => {
    renderButtons(makeCapture({ videoExport: makeExport({ supported: false }) }))
    expect(screen.queryByRole('button', { name: 'Exporter la vidéo' })).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Enregistrer la vidéo' })).toBeInTheDocument()
  })

  it('ouvre le dialogue au clic, et le referme au second', async () => {
    const user = userEvent.setup()
    renderButtons(makeCapture({ videoExport: makeExport() }))
    const bouton = screen.getByRole('button', { name: 'Exporter la vidéo' })
    await user.click(bouton)
    expect(screen.getByRole('dialog', { name: 'Exporter le rejeu en vidéo' })).toBeInTheDocument()
    await user.click(bouton)
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  // Le panneau se positionne en `absolute bottom-full right-0` : monté hors du cartouche, seul élément
  // à porter `relative`, il se résoudrait sur la carte entière du rejeu et serait découpé. jsdom ne
  // mesure rien ; ce que le test tient, c'est la parenté dont la mise en page dépend.
  it('monte le panneau DANS l’élément positionné qui porte le bouton', async () => {
    const user = userEvent.setup()
    renderButtons(makeCapture({ videoExport: makeExport() }))
    await user.click(screen.getByRole('button', { name: 'Exporter la vidéo' }))
    const ancre = screen.getByRole('dialog').parentElement
    expect(ancre).not.toBeNull()
    expect(ancre).toHaveClass('relative')
    expect(ancre).toContainElement(screen.getByRole('button', { name: 'Exporter la vidéo' }))
  })
})

describe('ReplayCaptureButtons — le bouton d’export porte la progression', () => {
  const avecExport = (state: ReplayExport['state']) => makeCapture({ videoExport: makeExport({ state }) })

  it('au repos : aucun pourcentage dans le bouton', () => {
    renderButtons(avecExport({ phase: 'idle', done: 0, total: 0, pct: 0 }))
    expect(screen.getByRole('button', { name: 'Exporter la vidéo' }).textContent).not.toContain('%')
  })

  it('affiche le pourcentage DANS le bouton pendant l’encodage, panneau fermé', () => {
    renderButtons(avecExport({ phase: 'encode', done: 300, total: 1200, pct: 25 }))
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Exporter la vidéo' })).toHaveTextContent('25 %')
  })

  it('affiche une ellipse pendant la préparation, où aucun pourcentage n’est vrai', () => {
    renderButtons(avecExport({ phase: 'prepare', done: 0, total: 1200, pct: 0 }))
    expect(screen.getByRole('button', { name: 'Exporter la vidéo' })).toHaveTextContent('…')
  })

  it('le bouton élargi garde son nom accessible, que le panneau soit ouvert ou non', async () => {
    const user = userEvent.setup()
    renderButtons(avecExport({ phase: 'encode', done: 600, total: 1200, pct: 50 }))
    const bouton = screen.getByRole('button', { name: 'Exporter la vidéo' })
    expect(bouton).toHaveTextContent('50 %')
    await user.click(bouton)
    expect(screen.getByRole('dialog')).toBeInTheDocument()
    await user.click(bouton)
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Exporter la vidéo' })).toHaveTextContent('50 %')
  })
})
