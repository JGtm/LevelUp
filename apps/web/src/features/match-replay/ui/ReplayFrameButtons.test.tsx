/**
 * Tests — ReplayFrameButtons (les icônes de cadre et le plein écran du système, à droite de la barre).
 *
 * Ce qu'ils protègent : l'icône du cadre courant est marquée ; un clic envoie le geste du modèle (jamais
 * une règle de cadre écrite dans le composant) ; les trois icônes sont inertes pendant un export, sauf la
 * sortie du Plein cadre ; les infobulles et noms accessibles rappellent les touches T et F, dans les deux
 * langues ; le plein écran du système reste pilotable, et inerte pendant un export.
 */
import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

import type { ReplayFrameControl } from '../hooks/useReplayFrame'
import type { ReplayFullscreen } from '../hooks/useReplayFullscreen'
import type { ReplayLocale } from '../i18n/i18n'
import type { ReplayFrame } from '../model/replayFrame'
import { makeFrameControl } from '../test/frameControlStub'
import { ReplayFrameButtons } from './ReplayFrameButtons'

const FRAMES: readonly ReplayFrame[] = ['standard', 'cinema', 'plein-cadre']

function renderButtons(
  frameControl: ReplayFrameControl,
  over: { fullscreen?: ReplayFullscreen; locale?: ReplayLocale } = {},
) {
  const fullscreen = over.fullscreen ?? { active: false, disabled: false, toggle: vi.fn() }
  return render(<ReplayFrameButtons frameControl={frameControl} fullscreen={fullscreen} locale={over.locale ?? 'fr'} />)
}

describe('ReplayFrameButtons — le cadre courant', () => {
  it.each(FRAMES)('%s : seule son icône est marquée', (frame) => {
    renderButtons(makeFrameControl(frame))
    const pressed = screen
      .getAllByRole('button', { pressed: true })
      .map((b) => b.getAttribute('aria-label'))
    const names = { standard: 'Standard', cinema: 'Cinéma (T)', 'plein-cadre': 'Plein cadre (F)' }
    expect(pressed).toEqual([names[frame]])
  })

  it('les trois icônes sont dans un groupe nommé « Cadre du lecteur »', () => {
    renderButtons(makeFrameControl())
    const groupe = screen.getByRole('group', { name: 'Cadre du lecteur' })
    expect(groupe.querySelectorAll('button')).toHaveLength(3)
    expect(groupe.querySelectorAll('button svg')).toHaveLength(3)
  })
})

describe('ReplayFrameButtons — les touches sont rappelées', () => {
  it('en français : « Cinéma (T) », « Plein cadre (F) », « Standard » sans touche, en nom et en infobulle', () => {
    renderButtons(makeFrameControl())
    for (const nom of ['Standard', 'Cinéma (T)', 'Plein cadre (F)']) {
      expect(screen.getByRole('button', { name: nom })).toHaveAttribute('title', nom)
    }
  })

  it('en anglais : « Cinema (T) », « Full frame (F) »', () => {
    renderButtons(makeFrameControl(), { locale: 'en' })
    expect(screen.getByRole('group', { name: 'Player layout' })).toBeInTheDocument()
    for (const nom of ['Standard', 'Cinema (T)', 'Full frame (F)']) {
      expect(screen.getByRole('button', { name: nom })).toHaveAttribute('title', nom)
    }
  })
})

describe('ReplayFrameButtons — les gestes', () => {
  it('depuis le Standard : Cinéma bascule le Cinéma, Plein cadre bascule le Plein cadre', () => {
    const control = makeFrameControl('standard')
    renderButtons(control)
    fireEvent.click(screen.getByRole('button', { name: 'Cinéma (T)' }))
    expect(control.toggleCinema).toHaveBeenCalledTimes(1)
    fireEvent.click(screen.getByRole('button', { name: 'Plein cadre (F)' }))
    expect(control.toggleFullFrame).toHaveBeenCalledTimes(1)
    expect(control.exit).not.toHaveBeenCalled()
  })

  it('depuis le Cinéma : Standard bascule le Cinéma (retour), l’icône courante ne fait rien', () => {
    const control = makeFrameControl('cinema')
    renderButtons(control)
    fireEvent.click(screen.getByRole('button', { name: 'Cinéma (T)' }))
    expect(control.toggleCinema).not.toHaveBeenCalled()
    fireEvent.click(screen.getByRole('button', { name: 'Standard' }))
    expect(control.toggleCinema).toHaveBeenCalledTimes(1)
  })

  it('depuis le Plein cadre : l’icône courante en SORT (exit), Cinéma passe par la bascule Cinéma', () => {
    const control = makeFrameControl('plein-cadre')
    renderButtons(control)
    fireEvent.click(screen.getByRole('button', { name: 'Plein cadre (F)' }))
    expect(control.exit).toHaveBeenCalledTimes(1)
    expect(control.toggleFullFrame).not.toHaveBeenCalled()
    fireEvent.click(screen.getByRole('button', { name: 'Cinéma (T)' }))
    expect(control.toggleCinema).toHaveBeenCalledTimes(1)
  })

  it('depuis un Plein cadre ouvert du Cinéma : Standard sort puis bascule le Cinéma (deux gestes)', () => {
    let state: ReplayFrame = 'plein-cadre'
    const exit = vi.fn(() => {
      state = 'cinema'
    })
    const toggleCinema = vi.fn(() => {
      state = 'standard'
    })
    const control = makeFrameControl('plein-cadre', {
      exit,
      toggleCinema,
      peek: () => ({ frame: state, returnTo: 'cinema' }),
    })
    renderButtons(control)
    fireEvent.click(screen.getByRole('button', { name: 'Standard' }))
    expect(exit).toHaveBeenCalledTimes(1)
    expect(toggleCinema).toHaveBeenCalledTimes(1)
    expect(state).toBe('standard')
  })
})

describe('ReplayFrameButtons — la sortie du Plein cadre est celle de « Quitter »', () => {
  it('l’icône courante rend le cadre ET referme le plein écran du système quand il est atteint', () => {
    const control = makeFrameControl('plein-cadre')
    const fullscreen: ReplayFullscreen = { active: true, disabled: false, toggle: vi.fn() }
    renderButtons(control, { fullscreen })
    fireEvent.click(screen.getByRole('button', { name: 'Plein cadre (F)' }))
    expect(control.exit).toHaveBeenCalledTimes(1)
    expect(fullscreen.toggle).toHaveBeenCalledTimes(1)
  })

  it('sans plein écran du système atteint, seule la sortie du cadre a lieu', () => {
    const control = makeFrameControl('plein-cadre')
    const fullscreen: ReplayFullscreen = { active: false, disabled: false, toggle: vi.fn() }
    renderButtons(control, { fullscreen })
    fireEvent.click(screen.getByRole('button', { name: 'Plein cadre (F)' }))
    expect(control.exit).toHaveBeenCalledTimes(1)
    expect(fullscreen.toggle).not.toHaveBeenCalled()
  })
})

describe('ReplayFrameButtons — pendant un export', () => {
  it('les trois icônes sont inertes et leurs clics ne font rien', () => {
    const control = makeFrameControl('standard', { disabled: true })
    renderButtons(control)
    for (const nom of ['Standard', 'Cinéma (T)', 'Plein cadre (F)']) {
      const bouton = screen.getByRole('button', { name: nom })
      expect(bouton).toBeDisabled()
      fireEvent.click(bouton)
    }
    expect(control.toggleCinema).not.toHaveBeenCalled()
    expect(control.toggleFullFrame).not.toHaveBeenCalled()
    expect(control.exit).not.toHaveBeenCalled()
  })

  it('en Plein cadre, SEULE la sortie reste active', () => {
    const control = makeFrameControl('plein-cadre', { disabled: true })
    renderButtons(control)
    expect(screen.getByRole('button', { name: 'Plein cadre (F)' })).toBeEnabled()
    expect(screen.getByRole('button', { name: 'Standard' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Cinéma (T)' })).toBeDisabled()
    fireEvent.click(screen.getByRole('button', { name: 'Plein cadre (F)' }))
    expect(control.exit).toHaveBeenCalledTimes(1)
    fireEvent.click(screen.getByRole('button', { name: 'Cinéma (T)' }))
    expect(control.toggleCinema).not.toHaveBeenCalled()
  })
})

describe('ReplayFrameButtons — plein écran du système', () => {
  it('fermé : « Plein écran », et le clic bascule le mode', () => {
    const toggle = vi.fn()
    renderButtons(makeFrameControl(), { fullscreen: { active: false, disabled: false, toggle } })
    const btn = screen.getByRole('button', { name: 'Plein écran' })
    expect(btn).toHaveAttribute('title', 'Plein écran')
    fireEvent.click(btn)
    expect(toggle).toHaveBeenCalledTimes(1)
  })

  it('ouvert : le nom dit la sortie', () => {
    renderButtons(makeFrameControl(), { fullscreen: { active: true, disabled: false, toggle: vi.fn() } })
    expect(screen.getByRole('button', { name: 'Quitter le plein écran' })).toHaveAttribute('title', 'Quitter le plein écran')
  })

  it('pendant un export : désactivé, le clic ne fait rien', () => {
    const toggle = vi.fn()
    renderButtons(makeFrameControl(), { fullscreen: { active: false, disabled: true, toggle } })
    const btn = screen.getByRole('button', { name: 'Plein écran' })
    expect(btn).toBeDisabled()
    fireEvent.click(btn)
    expect(toggle).not.toHaveBeenCalled()
  })

  it('reste pilotable quel que soit le cadre', () => {
    const toggle = vi.fn()
    renderButtons(makeFrameControl('plein-cadre'), { fullscreen: { active: false, disabled: false, toggle } })
    fireEvent.click(screen.getByRole('button', { name: 'Plein écran' }))
    expect(toggle).toHaveBeenCalledTimes(1)
  })

  it('en anglais : « Full screen » et « Exit full screen »', () => {
    const ferme = renderButtons(makeFrameControl(), { locale: 'en' })
    expect(screen.getByRole('button', { name: 'Full screen' })).toBeTruthy()
    ferme.unmount()
    renderButtons(makeFrameControl(), { locale: 'en', fullscreen: { active: true, disabled: false, toggle: vi.fn() } })
    expect(screen.getByRole('button', { name: 'Exit full screen' })).toBeTruthy()
  })
})
