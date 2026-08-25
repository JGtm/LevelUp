/**
 * Tests — ReplayTransport (la barre de lecture : icônes, vitesse, son, frise).
 *
 * Ce qu'ils protègent : les boutons en ICÔNE gardent leur libellé accessible (un symbole
 * sans nom serait une régression, pas une simplification), la vitesse vit À CÔTÉ de la
 * lecture avec son état dit (`aria-pressed`), et le son n'affiche aucune commande quand le
 * match n'en a aucun.
 */
import { createRef } from 'react'
import { describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'

import { ReplayTransport } from './ReplayTransport'
import type { ReplaySound } from './useReplaySound'

function makeSound(over: Partial<ReplaySound> = {}): ReplaySound {
  return {
    available: true,
    on: false,
    toggle: vi.fn(),
    volume: 0.7,
    setVolume: vi.fn(),
    mutedBySpeed: false,
    categories: { weapon: true, grenade: true, melee: true, equipment: true },
    toggleCategory: vi.fn(),
    tick: vi.fn(),
    ...over,
  }
}

function renderTransport(over: Partial<Parameters<typeof ReplayTransport>[0]> = {}) {
  const onTogglePlay = vi.fn()
  const onRestart = vi.fn()
  const onSetSpeed = vi.fn()
  const onToggleSettings = vi.fn()
  const utils = render(
    <ReplayTransport
      playing
      onTogglePlay={onTogglePlay}
      onRestart={onRestart}
      clockRef={createRef<HTMLSpanElement>()}
      sliderRef={createRef<HTMLInputElement>()}
      maxFrame={100}
      onScrub={vi.fn()}
      speed={1}
      onSetSpeed={onSetSpeed}
      sound={makeSound()}
      locale="fr"
      leadMarks={{
        changes: [],
        frameCount: 101,
        allyOf: () => null,
        labelOf: () => '',
        locale: 'fr',
      }}
      settingsOpen={false}
      onToggleSettings={onToggleSettings}
      settingsButtonRef={createRef<HTMLButtonElement>()}
      {...over}
    />,
  )
  return { ...utils, onTogglePlay, onRestart, onSetSpeed, onToggleSettings }
}

describe('ReplayTransport — lecture en icônes', () => {
  it('en lecture : le bouton dit « Pause » et bascule au clic', () => {
    const { onTogglePlay } = renderTransport({ playing: true })
    const btn = screen.getByRole('button', { name: 'Pause' })
    expect(btn.querySelector('svg')).toBeTruthy()
    fireEvent.click(btn)
    expect(onTogglePlay).toHaveBeenCalledTimes(1)
  })

  it('à l’arrêt : le bouton dit « Lecture »', () => {
    renderTransport({ playing: false })
    expect(screen.getByRole('button', { name: 'Lecture' }).querySelector('svg')).toBeTruthy()
  })

  it('« Recommencer » est une icône nommée, et appelle onRestart', () => {
    const { onRestart } = renderTransport()
    const btn = screen.getByRole('button', { name: 'Recommencer' })
    expect(btn.querySelector('svg')).toBeTruthy()
    fireEvent.click(btn)
    expect(onRestart).toHaveBeenCalledTimes(1)
  })
})

describe('ReplayTransport — vitesse à côté de la lecture', () => {
  it('propose les quatre multiplicateurs, aria-pressed sur celui en cours', () => {
    renderTransport({ speed: 2 })
    const pressed = ['0.5×', '1×', '2×', '4×'].filter(
      (label) => screen.getByRole('button', { name: label }).getAttribute('aria-pressed') === 'true',
    )
    expect(pressed).toEqual(['2×'])
  })

  it('cliquer un multiplicateur appelle onSetSpeed avec cette valeur', () => {
    const { onSetSpeed } = renderTransport({ speed: 1 })
    fireEvent.click(screen.getByRole('button', { name: '4×' }))
    expect(onSetSpeed).toHaveBeenCalledWith(4)
  })
})

describe('ReplayTransport — le son au niveau de la lecture', () => {
  it('l’interrupteur du son est dans la barre quand le match a des sons', () => {
    renderTransport()
    expect(screen.getByRole('button', { name: 'Son' })).toBeTruthy()
  })

  it('aucune commande de son quand le match n’en a aucun', () => {
    renderTransport({ sound: makeSound({ available: false }) })
    expect(screen.queryByRole('button', { name: 'Son' })).toBeNull()
  })

  it('son allumé : le curseur porte le volume réglé, et il est actionnable', () => {
    renderTransport({ sound: makeSound({ on: true, volume: 0.4 }) })
    const slider = screen.getByLabelText('Volume des sons') as HTMLInputElement
    expect(slider.value).toBe('40')
    expect(slider.disabled).toBe(false)
  })

  // DEMANDE UTILISATEUR DU 2026-08-25 : couper le son ne doit PLUS faire disparaître la barre
  // de volume. Ce cas tient les trois faits qui la composent — elle reste à l'écran, elle
  // montre zéro, et elle n'agit plus tant que le son est coupé.
  it('son coupé : le curseur RESTE, à zéro et inerte', () => {
    renderTransport({ sound: makeSound({ on: false, volume: 0.7 }) })
    const slider = screen.getByLabelText('Volume des sons') as HTMLInputElement
    expect(slider.value).toBe('0')
    expect(slider.disabled).toBe(true)
    expect(slider.getAttribute('title')).toMatch(/Son coupé/)
  })

  // LE NIVEAU RÉGLÉ N'EST PAS PERDU : le zéro affiché pendant la coupure est un AFFICHAGE, pas
  // une écriture. Le composant ne touche jamais `sound.volume` en basculant — la preuve est ici,
  // sur les deux rendus du même volume 0,7.
  it('rallumer le son rend le volume précédent — la coupure n’écrit rien', () => {
    const setVolume = vi.fn()
    const { unmount } = renderTransport({ sound: makeSound({ on: false, volume: 0.7, setVolume }) })
    expect(setVolume).not.toHaveBeenCalled()
    unmount()
    renderTransport({ sound: makeSound({ on: true, volume: 0.7, setVolume }) })
    expect((screen.getByLabelText('Volume des sons') as HTMLInputElement).value).toBe('70')
  })
})

describe('ReplayTransport — les réglages ferment la barre', () => {
  it('le bouton du tiroir est là, en icône nommée, et bascule au clic', () => {
    const { onToggleSettings } = renderTransport()
    const btn = screen.getByRole('button', { name: 'Réglages' })
    expect(btn.querySelector('svg')).toBeTruthy()
    expect(btn).toHaveAttribute('aria-expanded', 'false')
    fireEvent.click(btn)
    expect(onToggleSettings).toHaveBeenCalledTimes(1)
  })
})
