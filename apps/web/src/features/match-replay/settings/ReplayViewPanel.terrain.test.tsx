/**
 * ReplayViewPanel.terrain.test.tsx — LE PANNEAU DE LA VUE : la bascule « Plan des murs » (visible
 * seulement quand le fond porte un plan) et la commande d'orientation de la carte.
 */
import { describe, expect, it, vi } from 'vitest'
import { fireEvent, screen } from '@testing-library/react'

import { makeOrientation, makeWallPlan, renderPanel } from './replayViewPanelFixtures'

describe('ReplayViewPanel — orientation de la carte', () => {
  it('propose les quatre positions en choix exclusif, la courante cochée', () => {
    renderPanel({ orientation: makeOrientation({ turns: 2 }) })
    const group = screen.getByRole('radiogroup', { name: 'Orientation' })
    expect(group).toBeTruthy()
    for (const nom of ['0°', '90°', '180°', '270°']) {
      expect(screen.getByRole('radio', { name: nom })).toBeTruthy()
    }
    expect(screen.getByRole('radio', { name: '180°' })).toHaveAttribute('aria-checked', 'true')
    expect(screen.getByRole('radio', { name: '0°' })).toHaveAttribute('aria-checked', 'false')
  })

  it('un clic choisit SA position, en quarts de tour', () => {
    const onSelect = vi.fn()
    renderPanel({ orientation: makeOrientation({ onSelect }) })
    fireEvent.click(screen.getByRole('radio', { name: '270°' }))
    expect(onSelect).toHaveBeenCalledWith(3)
    fireEvent.click(screen.getByRole('radio', { name: '90°' }))
    expect(onSelect).toHaveBeenLastCalledWith(1)
  })

  it('sans carte connue, la commande est absente', () => {
    renderPanel({ orientation: makeOrientation({ available: false }) })
    expect(screen.queryByRole('radiogroup', { name: 'Orientation' })).toBeNull()
  })

  it('le titre est dans la langue du panneau', () => {
    renderPanel({ locale: 'en' })
    expect(screen.getByRole('radiogroup', { name: 'Orientation' })).toBeTruthy()
  })
})

describe('ReplayViewPanel — plan des murs', () => {
  it('carte dont le fond porte un plan : la bascule est proposée, reflète son état et appelle SON callback', () => {
    const onToggle = vi.fn()
    const { onToggleZones } = renderPanel({ wallPlan: makeWallPlan({ show: false, onToggle }) })
    const btn = screen.getByRole('switch', { name: 'Plan des murs' })
    expect(btn).toHaveAttribute('aria-checked', 'false')
    fireEvent.click(btn)
    expect(onToggle).toHaveBeenCalledTimes(1)
    expect(onToggleZones).not.toHaveBeenCalled()
  })

  it('carte sans plan (Live Fire, deux cartes Forge) : la bascule est absente', () => {
    renderPanel({ wallPlan: makeWallPlan({ available: false }) })
    expect(screen.queryByRole('switch', { name: 'Plan des murs' })).toBeNull()
  })

  it('en anglais, la bascule se nomme Wall plan', () => {
    renderPanel({ locale: 'en' })
    expect(screen.getByRole('switch', { name: 'Wall plan' })).toBeTruthy()
  })
})
