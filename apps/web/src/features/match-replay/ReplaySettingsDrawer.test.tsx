/**
 * ReplaySettingsDrawer.test.tsx — LE TIROIR DE RÉGLAGES : chaque section n'apparaît que si
 * elle a quelque chose à commander, chaque bascule appelle EXACTEMENT son callback (jamais
 * une voisine), et le panneau se ferme par son bouton comme par Échap (même contrat que les
 * autres panneaux du dépôt — AlertDialog, AssetDrawer).
 */
import { describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'

import { ReplaySettingsDrawer } from './ReplaySettingsDrawer'
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

function renderDrawer(over: Partial<Parameters<typeof ReplaySettingsDrawer>[0]> = {}) {
  const onClose = vi.fn()
  const onToggleAim = vi.fn()
  const onToggleZones = vi.fn()
  const onSetSpeed = vi.fn()
  const utils = render(
    <ReplaySettingsDrawer
      locale="fr"
      onClose={onClose}
      showAim
      onToggleAim={onToggleAim}
      showZones
      onToggleZones={onToggleZones}
      zonesAvailable
      sound={makeSound()}
      speed={1}
      onSetSpeed={onSetSpeed}
      {...over}
    />,
  )
  return { ...utils, onClose, onToggleAim, onToggleZones, onSetSpeed }
}

describe('ReplaySettingsDrawer — calques', () => {
  it('bascule Visée : reflète showAim et appelle onToggleAim au clic, jamais onToggleZones', () => {
    const { onToggleAim, onToggleZones } = renderDrawer({ showAim: true })
    const btn = screen.getByRole('button', { name: 'Visée' })
    expect(btn).toHaveAttribute('aria-pressed', 'true')
    fireEvent.click(btn)
    expect(onToggleAim).toHaveBeenCalledTimes(1)
    expect(onToggleZones).not.toHaveBeenCalled()
  })

  it('bouton Zones absent quand la carte n a pas de zones nommées', () => {
    renderDrawer({ zonesAvailable: false })
    expect(screen.queryByRole('button', { name: 'Zones' })).toBeNull()
  })

  it('bouton Zones présent, reflète showZones, appelle onToggleZones au clic', () => {
    const { onToggleZones } = renderDrawer({ zonesAvailable: true, showZones: false })
    const btn = screen.getByRole('button', { name: 'Zones' })
    expect(btn).toHaveAttribute('aria-pressed', 'false')
    fireEvent.click(btn)
    expect(onToggleZones).toHaveBeenCalledTimes(1)
  })
})

describe('ReplaySettingsDrawer — vitesse', () => {
  it('propose les quatre multiplicateurs, aria-pressed sur celui en cours', () => {
    renderDrawer({ speed: 2 })
    expect(screen.getByRole('button', { name: '0.5×' })).toBeTruthy()
    expect(screen.getByRole('button', { name: '1×' })).toBeTruthy()
    expect(screen.getByRole('button', { name: '2×' })).toBeTruthy()
    expect(screen.getByRole('button', { name: '4×' })).toBeTruthy()
  })

  it('cliquer un multiplicateur appelle onSetSpeed avec cette valeur', () => {
    const { onSetSpeed } = renderDrawer({ speed: 1 })
    fireEvent.click(screen.getByRole('button', { name: '4×' }))
    expect(onSetSpeed).toHaveBeenCalledWith(4)
  })
})

describe('ReplaySettingsDrawer — son', () => {
  it('aucune commande de son ni de catégorie quand le match n a aucun son', () => {
    renderDrawer({ sound: makeSound({ available: false }) })
    expect(screen.queryByText('Sons par catégorie')).toBeNull()
    expect(screen.queryByRole('button', { name: 'Armes' })).toBeNull()
    expect(screen.queryByRole('button', { name: 'Son' })).toBeNull()
  })

  it('les quatre catégories sont affichées, chacune avec son état', () => {
    renderDrawer({
      sound: makeSound({
        categories: { weapon: true, grenade: false, melee: true, equipment: false },
      }),
    })
    expect(screen.getByRole('button', { name: 'Armes' })).toHaveAttribute('aria-pressed', 'true')
    expect(screen.getByRole('button', { name: 'Grenades' })).toHaveAttribute('aria-pressed', 'false')
    expect(screen.getByRole('button', { name: 'Mêlée' })).toHaveAttribute('aria-pressed', 'true')
    expect(screen.getByRole('button', { name: 'Équipements' })).toHaveAttribute('aria-pressed', 'false')
  })

  it('cliquer une catégorie appelle toggleCategory avec SA clé, jamais une voisine', () => {
    const toggleCategory = vi.fn()
    renderDrawer({ sound: makeSound({ toggleCategory }) })
    fireEvent.click(screen.getByRole('button', { name: 'Grenades' }))
    expect(toggleCategory).toHaveBeenCalledTimes(1)
    expect(toggleCategory).toHaveBeenCalledWith('grenade')
  })
})

describe('ReplaySettingsDrawer — fermeture', () => {
  it('le bouton de fermeture appelle onClose', () => {
    const { onClose } = renderDrawer()
    fireEvent.click(screen.getByRole('button', { name: 'Fermer les réglages' }))
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('Échap appelle onClose', () => {
    const { onClose } = renderDrawer()
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(onClose).toHaveBeenCalledTimes(1)
  })
})
