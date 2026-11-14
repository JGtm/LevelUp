/**
 * ReplayViewPanel.terrain.test.tsx — LE PANNEAU DE LA VUE : la bascule « Plan des murs » (visible
 * seulement quand le fond porte un plan) et la commande d'orientation de la carte.
 */
import { describe, expect, it, vi } from 'vitest'
import { fireEvent, screen } from '@testing-library/react'

import { makeBackground, makeOrientation, makeWallPlan, renderPanel } from './replayViewPanelFixtures'

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

describe('ReplayViewPanel — fond de carte (rendu ou plan)', () => {
  it('fond avec image plan : « Fond » propose Rendu et Plan en choix exclusif, le courant coché', () => {
    renderPanel({ background: makeBackground({ value: 'plan' }) })
    expect(screen.getByRole('radiogroup', { name: 'Fond' })).toBeTruthy()
    expect(screen.getByRole('radio', { name: 'Plan' })).toHaveAttribute('aria-checked', 'true')
    expect(screen.getByRole('radio', { name: 'Rendu' })).toHaveAttribute('aria-checked', 'false')
  })

  it('un clic choisit SA variante', () => {
    const onSelect = vi.fn()
    renderPanel({ background: makeBackground({ onSelect }) })
    fireEvent.click(screen.getByRole('radio', { name: 'Plan' }))
    expect(onSelect).toHaveBeenCalledWith('plan')
    fireEvent.click(screen.getByRole('radio', { name: 'Rendu' }))
    expect(onSelect).toHaveBeenLastCalledWith('rendu')
  })

  it('fond sans image plan : la commande est absente', () => {
    renderPanel({ background: makeBackground({ available: false }) })
    expect(screen.queryByRole('radiogroup', { name: 'Fond' })).toBeNull()
  })

  it('en anglais, la commande se nomme Background : Render / Plan', () => {
    renderPanel({ locale: 'en' })
    expect(screen.getByRole('radiogroup', { name: 'Background' })).toBeTruthy()
    expect(screen.getByRole('radio', { name: 'Render' })).toBeTruthy()
    expect(screen.getByRole('radio', { name: 'Plan' })).toBeTruthy()
  })

  it('en mode plan, le plan des murs est désactivé, éteint, avec « Déjà dans le plan » en infobulle', () => {
    const onToggle = vi.fn()
    renderPanel({ wallPlan: makeWallPlan({ show: true, inPlan: true, onToggle }) })
    const btn = screen.getByRole('switch', { name: 'Plan des murs' })
    expect(btn).toBeDisabled()
    expect(btn).toHaveAttribute('aria-checked', 'false')
    expect(btn).toHaveAttribute('title', 'Déjà dans le plan')
    fireEvent.click(btn)
    expect(onToggle).not.toHaveBeenCalled()
  })

  it('en mode plan et en anglais, l infobulle dit Already in the plan', () => {
    renderPanel({ locale: 'en', wallPlan: makeWallPlan({ inPlan: true }) })
    expect(screen.getByRole('switch', { name: 'Wall plan' })).toHaveAttribute('title', 'Already in the plan')
  })

  it('sur le rendu, le plan des murs reste actif et garde son état', () => {
    renderPanel({ wallPlan: makeWallPlan({ show: true, inPlan: false }) })
    const btn = screen.getByRole('switch', { name: 'Plan des murs' })
    expect(btn).not.toBeDisabled()
    expect(btn).toHaveAttribute('aria-checked', 'true')
  })
})
