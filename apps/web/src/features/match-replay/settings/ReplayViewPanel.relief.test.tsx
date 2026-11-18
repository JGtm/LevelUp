/**
 * Tests — le panneau de la vue et le fond Relief : le segmenté « Fond » à trois options, l'interrupteur
 * « Traits du relief » et la commande d'étage, dans le groupe « Terrain ».
 *
 * Ce qu'ils protègent (maquette D.1, choix validés) : l'ordre Plan, Rendu, Relief ; l'infobulle du Relief qui dit
 * la part tirée du maillage (D-16) ; « Plan des murs » disponible sur le Relief ; l'absence de toute commande
 * d'étage sans maillage ; l'absence de toute commande de relief sans relief ; fr ET en.
 */
import { cleanup, fireEvent, screen, within } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

import { floorRanks } from '@/lib/replay/relief'

import { floorChips } from '../hooks/reliefLogic'
import type { ReplayFloorControls } from './ReplayFloorControls'
import { makeBackground, makeRelief, makeWallPlan, renderPanel } from './replayViewPanelFixtures'

const THREE = ['plan', 'rendu', 'relief'] as const

function floorReady(over: Partial<ReplayFloorControls> = {}): ReplayFloorControls {
  return {
    state: { state: 'ready', chips: floorChips(floorRanks([0, 2.2, 4.4]), { min: 0, max: 5 }), min: 0, max: 4.4 },
    cut: null,
    onSelectAll: vi.fn(),
    onSelectRank: vi.fn(),
    onHeight: vi.fn(),
    follow: { available: false, on: false, onToggle: vi.fn() },
    ...over,
  }
}

describe('panneau de la vue — le segmenté « Fond » à trois options', () => {
  it('propose Plan, Rendu, Relief dans cet ordre, le fond retenu pressé', () => {
    renderPanel({ background: makeBackground({ options: THREE, value: 'relief' }) })
    const group = screen.getByRole('radiogroup', { name: 'Fond' })
    expect(within(group).getAllByRole('radio').map((r) => r.textContent)).toEqual(['Plan', 'Rendu', 'Relief'])
    expect(screen.getByRole('radio', { name: 'Relief' })).toHaveAttribute('aria-checked', 'true')
    expect(screen.getByRole('radio', { name: 'Plan' })).toHaveAttribute('aria-checked', 'false')
  })

  it('un clic sur Relief le choisit', () => {
    const onSelect = vi.fn()
    renderPanel({ background: makeBackground({ options: THREE, value: 'plan', onSelect }) })
    fireEvent.click(screen.getByRole('radio', { name: 'Relief' }))
    expect(onSelect).toHaveBeenCalledWith('relief')
  })

  it('l’infobulle du Relief dit la part tirée du maillage de navigation (D-16), quand elle est connue', () => {
    renderPanel({ background: makeBackground({ options: THREE, meshPercent: 46 }) })
    const title = screen.getByRole('radio', { name: 'Relief' }).getAttribute('title') ?? ''
    expect(title).toContain('Le terrain vu de dessus, en gris teinté')
    expect(title).toContain('Relief tiré du maillage de navigation sur 46 % de la carte.')
  })

  it('sans part connue, l’infobulle ne dit rien du maillage', () => {
    renderPanel({ background: makeBackground({ options: THREE, meshPercent: null }) })
    expect(screen.getByRole('radio', { name: 'Relief' }).getAttribute('title')).not.toContain('maillage')
  })

  it('sans choix (aucune option) : pas de segmenté', () => {
    renderPanel({ background: makeBackground({ options: [] }) })
    expect(screen.queryByRole('radiogroup', { name: 'Fond' })).toBeNull()
  })

  it('un fond avec relief mais sans plan propose Rendu, Relief', () => {
    renderPanel({ background: makeBackground({ options: ['rendu', 'relief'], value: 'rendu' }) })
    expect(within(screen.getByRole('radiogroup', { name: 'Fond' })).getAllByRole('radio').map((r) => r.textContent)).toEqual(['Rendu', 'Relief'])
  })
})

describe('panneau de la vue — les traits du relief', () => {
  it('l’interrupteur reflète l’état retenu pour le fond et bascule', () => {
    const onToggle = vi.fn()
    renderPanel({ relief: makeRelief({ lines: { available: true, show: true, onToggle } }) })
    const sw = screen.getByRole('switch', { name: 'Traits du relief' })
    expect(sw).toHaveAttribute('aria-checked', 'true')
    fireEvent.click(sw)
    expect(onToggle).toHaveBeenCalledTimes(1)
  })

  it('éteint sur un fond où il ne l’est pas par défaut', () => {
    renderPanel({ relief: makeRelief({ lines: { available: true, show: false, onToggle: vi.fn() } }) })
    expect(screen.getByRole('switch', { name: 'Traits du relief' })).toHaveAttribute('aria-checked', 'false')
  })

  it('« Plan des murs » reste disponible sur le Relief, indépendant des traits', () => {
    renderPanel({
      background: makeBackground({ options: THREE, value: 'relief' }),
      wallPlan: makeWallPlan({ show: true, inPlan: false }),
      relief: makeRelief({ lines: { available: true, show: true, onToggle: vi.fn() } }),
    })
    const walls = screen.getByRole('switch', { name: 'Plan des murs' })
    expect(walls).not.toBeDisabled()
    expect(walls).toHaveAttribute('aria-checked', 'true')
  })

  it('sans relief : ni traits ni étage', () => {
    renderPanel({ relief: makeRelief() })
    expect(screen.queryByRole('switch', { name: 'Traits du relief' })).toBeNull()
    expect(screen.queryByRole('slider')).toBeNull()
  })
})

describe('panneau de la vue — la commande d’étage', () => {
  it('pose « Étage » dans le groupe Terrain : Tout, une pastille par étage en rangs, un curseur', () => {
    renderPanel({ relief: makeRelief({ floor: floorReady() }) })
    expect(screen.getByText('Étage')).toBeTruthy()
    const chips = within(screen.getByRole('group', { name: 'Étage' })).getAllByRole('button')
    expect(chips.map((b) => b.textContent)).toEqual(['Tout', 'Bas', 'Étage 1', 'Étage 2'])
    expect(screen.getByRole('slider', { name: 'Hauteur de la coupe' })).toBeTruthy()
  })

  it('un clic sur une pastille retient l’étage', () => {
    const onSelectRank = vi.fn()
    renderPanel({ relief: makeRelief({ floor: floorReady({ onSelectRank }) }) })
    fireEvent.click(screen.getByRole('button', { name: 'Étage 1' }))
    expect(onSelectRank).toHaveBeenCalledWith(2.2)
  })

  it('carte sans maillage : le bloc d’explication, les traits restent proposés, aucune commande d’étage', () => {
    renderPanel({
      relief: makeRelief({
        lines: { available: true, show: true, onToggle: vi.fn() },
        floor: { state: { state: 'noMesh' }, cut: null, onSelectAll: vi.fn(), onSelectRank: vi.fn(), onHeight: vi.fn(), follow: { available: false, on: false, onToggle: vi.fn() } },
      }),
    })
    expect(screen.getByText(/Carte sans maillage de navigation/)).toBeTruthy()
    expect(screen.getByRole('switch', { name: 'Traits du relief' })).toBeTruthy()
    expect(screen.queryByRole('slider')).toBeNull()
    expect(screen.queryByRole('button', { name: 'Tout' })).toBeNull()
  })

  it('« Suivre le joueur ouvert » ferme le groupe Étage : désactivé sans fiche ouverte, il bascule avec une fiche', () => {
    renderPanel({ relief: makeRelief({ floor: floorReady() }) })
    expect(screen.getByRole('switch', { name: 'Suivre le joueur ouvert' })).toBeDisabled()
    cleanup()
    const onToggle = vi.fn()
    renderPanel({ relief: makeRelief({ floor: floorReady({ follow: { available: true, on: false, onToggle } }) }) })
    fireEvent.click(screen.getByRole('switch', { name: 'Suivre le joueur ouvert' }))
    expect(onToggle).toHaveBeenCalledTimes(1)
  })
})

describe('panneau de la vue — en anglais', () => {
  it('Background : Plan, Render, Relief ; Relief lines ; Floor, All, Low, Floor 1', () => {
    renderPanel({
      locale: 'en',
      background: makeBackground({ options: THREE, value: 'relief', meshPercent: 46 }),
      relief: makeRelief({ lines: { available: true, show: true, onToggle: vi.fn() }, floor: floorReady() }),
    })
    expect(within(screen.getByRole('radiogroup', { name: 'Background' })).getAllByRole('radio').map((r) => r.textContent)).toEqual(['Plan', 'Render', 'Relief'])
    expect(screen.getByRole('radio', { name: 'Relief' }).getAttribute('title')).toContain('Relief taken from the navigation mesh on 46% of the map.')
    expect(screen.getByRole('switch', { name: 'Relief lines' })).toBeTruthy()
    expect(within(screen.getByRole('group', { name: 'Floor' })).getAllByRole('button').map((b) => b.textContent)).toEqual(['All', 'Low', 'Floor 1', 'Floor 2'])
  })
})
