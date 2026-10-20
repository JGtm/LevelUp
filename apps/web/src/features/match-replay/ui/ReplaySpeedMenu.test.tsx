/**
 * Tests — ReplaySpeedMenu (la vitesse en menu déroulant).
 *
 * Ce qu'ils protègent : le déclencheur AFFICHE la valeur courante (c'est ce qui remplace
 * l'information que les quatre boutons donnaient d'un coup d'œil), le menu se ferme par les
 * mêmes sorties que le tiroir de réglages (Échap, clic dehors, et la sélection elle-même), et
 * la note « son coupé » se pose sur les vitesses où le lecteur se tait RÉELLEMENT — elle
 * interroge `soundPlaysAtSpeed`, elle ne redéfinit pas la borne.
 */
import { describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'

import { ReplaySpeedMenu } from './ReplaySpeedMenu'

function renderMenu(speed = 1) {
  const onSetSpeed = vi.fn()
  const utils = render(<ReplaySpeedMenu speed={speed} onSetSpeed={onSetSpeed} locale="fr" />)
  return { ...utils, onSetSpeed, trigger: screen.getByRole('button', { name: 'Vitesse' }) }
}

describe('ReplaySpeedMenu — le déclencheur dit où l’on en est', () => {
  it('affiche la vitesse courante, décimale comprise', () => {
    const { trigger } = renderMenu(0.5)
    expect(trigger.textContent).toContain('0.5×')
  })

  it('fermé, il ne rend AUCUNE option : le menu ne coûte rien tant qu’il dort', () => {
    const { trigger } = renderMenu(1)
    expect(trigger).toHaveAttribute('aria-expanded', 'false')
    expect(screen.queryByRole('button', { name: /4×/ })).toBeNull()
  })
})

describe('ReplaySpeedMenu — ouvrir, choisir, fermer', () => {
  it('ouvre au clic et propose les quatre multiplicateurs', () => {
    const { trigger } = renderMenu(1)
    fireEvent.click(trigger)
    expect(trigger).toHaveAttribute('aria-expanded', 'true')
    const options = screen.getAllByRole('button', { pressed: false })
    // Les quatre valeurs sont là ; celle en cours est `aria-pressed`, donc hors de cette liste.
    expect(options.length).toBeGreaterThanOrEqual(3)
    expect(screen.getByRole('button', { name: /^4×/ })).toBeTruthy()
  })

  it('marque la vitesse en cours, et une seule', () => {
    const { trigger } = renderMenu(2)
    fireEvent.click(trigger)
    const pressed = screen.getAllByRole('button', { pressed: true })
    expect(pressed).toHaveLength(1)
    expect(pressed[0].textContent).toContain('2×')
  })

  it('choisir applique la vitesse ET referme le menu', () => {
    const { trigger, onSetSpeed } = renderMenu(1)
    fireEvent.click(trigger)
    fireEvent.click(screen.getByRole('button', { name: /^4×/ }))
    expect(onSetSpeed).toHaveBeenCalledWith(4)
    expect(trigger).toHaveAttribute('aria-expanded', 'false')
  })

  it('Échap referme sans rien changer', () => {
    const { trigger, onSetSpeed } = renderMenu(1)
    fireEvent.click(trigger)
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(trigger).toHaveAttribute('aria-expanded', 'false')
    expect(onSetSpeed).not.toHaveBeenCalled()
  })

  it('un clic DEHORS referme — au geste, pas au relâché', () => {
    const { trigger } = renderMenu(1)
    fireEvent.click(trigger)
    fireEvent.pointerDown(document.body)
    expect(trigger).toHaveAttribute('aria-expanded', 'false')
  })

  it('un clic DANS le menu ne le referme pas', () => {
    const { trigger } = renderMenu(1)
    fireEvent.click(trigger)
    fireEvent.pointerDown(screen.getByRole('group', { name: 'Vitesse' }))
    expect(trigger).toHaveAttribute('aria-expanded', 'true')
  })
})

describe('ReplaySpeedMenu — les deux notes', () => {
  it('« normal » marque la vitesse de référence', () => {
    const { trigger } = renderMenu(4)
    fireEvent.click(trigger)
    expect(screen.getByRole('button', { name: /^1×/ }).textContent).toContain('normal')
  })

  // LA BORNE N'EST PAS RECOPIÉE ICI : le menu interroge `soundPlaysAtSpeed`. 2× joue, 4× non —
  // si la règle du son bougeait, ce test bougerait avec elle, ce qui est exactement le but.
  it('« son coupé » ne marque que les vitesses où le son se tait', () => {
    const { trigger } = renderMenu(1)
    fireEvent.click(trigger)
    expect(screen.getByRole('button', { name: /^4×/ }).textContent).toContain('son coupé')
    expect(screen.getByRole('button', { name: /^2×/ }).textContent).not.toContain('son coupé')
    expect(screen.getByRole('button', { name: /^0.5×/ }).textContent).not.toContain('son coupé')
  })
})
