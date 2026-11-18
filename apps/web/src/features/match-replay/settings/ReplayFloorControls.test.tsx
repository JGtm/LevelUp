/**
 * Tests — ReplayFloorControls : le groupe « Étage » du panneau de la vue.
 *
 * Ce qu'ils protègent : (1) « Tout » par défaut, une pastille par étage en RANGS (« Bas », « Étage 1 »…),
 * jamais une hauteur ; (2) l'infobulle d'une pastille dit les anneaux d'un pion à cet étage ; (3) au-delà de six
 * pastilles la rangée défile ; (4) le curseur ne retient aucune pastille ; (5) une carte sans maillage affiche
 * un bloc d'explication en pointillé et AUCUNE commande ; (6) rien sans relief ; (7) libellés en anglais.
 */
import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

import { floorChips, type FloorState } from '../hooks/reliefLogic'
import type { ReplayLocale } from '../i18n/i18n'
import { FloorGroup, type ReplayFloorControls } from './ReplayFloorControls'

function ready(levels: number[], over: Partial<ReplayFloorControls> = {}): ReplayFloorControls {
  const state: FloorState = {
    state: 'ready',
    chips: floorChips(levels, { min: 0, max: 5 }),
    min: levels[0],
    max: levels[levels.length - 1],
  }
  return { state, cut: null, onSelectAll: vi.fn(), onSelectRank: vi.fn(), onHeight: vi.fn(), ...over }
}

function stateOnly(state: FloorState): ReplayFloorControls {
  return { state, cut: null, onSelectAll: vi.fn(), onSelectRank: vi.fn(), onHeight: vi.fn() }
}

function mount(floor: ReplayFloorControls, locale: ReplayLocale = 'fr') {
  return render(<FloorGroup locale={locale} floor={floor} />)
}

describe('FloorGroup — les pastilles', () => {
  it('« Tout » d’abord, retenue par défaut, puis une pastille par étage en rangs, sans unité', () => {
    mount(ready([0, 2.2, 4.4]))
    const names = screen.getAllByRole('button').map((b) => b.textContent)
    expect(names).toEqual(['Tout', 'Bas', 'Étage 1', 'Étage 2'])
    expect(screen.getByRole('button', { name: 'Tout' })).toHaveAttribute('aria-pressed', 'true')
    for (const n of names) expect(n).not.toMatch(/\d\s?m\b/)
  })

  it('l’infobulle d’une pastille dit le nombre d’anneaux d’un pion à cet étage', () => {
    mount(ready([0, 2.2, 4.4], {}))
    expect(screen.getByRole('button', { name: 'Bas' })).toHaveAttribute('title', 'Bas : aucun anneau')
    expect(screen.getByRole('button', { name: 'Étage 1' }).getAttribute('title')).toMatch(/^Étage 1 : \d anneaux?$/)
    expect(screen.getByRole('button', { name: 'Tout' })).toHaveAttribute('title', 'Aucune coupe : tous les étages en plein.')
  })

  it('un clic sur une pastille retient la hauteur exacte de l’étage ; « Tout » rend la coupe', () => {
    const onSelectRank = vi.fn()
    const onSelectAll = vi.fn()
    mount(ready([0, 2.2, 4.4], { onSelectRank, onSelectAll }))
    fireEvent.click(screen.getByRole('button', { name: 'Étage 2' }))
    expect(onSelectRank).toHaveBeenCalledWith(4.4)
    fireEvent.click(screen.getByRole('button', { name: 'Tout' }))
    expect(onSelectAll).toHaveBeenCalledTimes(1)
  })

  it('la pastille de la coupe courante est retenue, « Tout » ne l’est plus', () => {
    mount(ready([0, 2.2, 4.4], { cut: 2.2 }))
    expect(screen.getByRole('button', { name: 'Étage 1' })).toHaveAttribute('aria-pressed', 'true')
    expect(screen.getByRole('button', { name: 'Tout' })).toHaveAttribute('aria-pressed', 'false')
    expect(screen.getByRole('button', { name: 'Bas' })).toHaveAttribute('aria-pressed', 'false')
  })

  it('jusqu’à six pastilles d’étage la rangée passe à la ligne, au-delà elle défile horizontalement', () => {
    const six = mount(ready([0, 1.5, 3, 4.5, 6, 7.5]))
    expect(six.getByRole('group').className).toContain('flex-wrap')
    expect(six.getByRole('group').className).not.toContain('overflow-x-auto')
    six.unmount()
    const huit = mount(ready([0, 1.5, 3, 4.5, 6, 7.5, 9, 10.5]))
    expect(huit.getByRole('group').className).toContain('overflow-x-auto')
    expect(huit.getByRole('group').className).not.toContain('flex-wrap')
    expect(huit.getAllByRole('button')).toHaveLength(9)
  })
})

describe('FloorGroup — le curseur', () => {
  it('porte les bornes des sols, le nom accessible et le pas du quart d’unité', () => {
    mount(ready([0, 2.2, 4.4]))
    const slider = screen.getByRole('slider', { name: 'Hauteur de la coupe' })
    expect(slider).toHaveAttribute('min', '0')
    expect(slider).toHaveAttribute('max', '4.4')
    expect(slider).toHaveAttribute('step', '0.25')
  })

  it('un geste du curseur donne une hauteur libre, sans retenir de pastille', () => {
    const onHeight = vi.fn()
    mount(ready([0, 2.2, 4.4], { onHeight }))
    fireEvent.change(screen.getByRole('slider'), { target: { value: '1.25' } })
    expect(onHeight).toHaveBeenCalledWith(1.25)
  })

  it('dit « Tout » sans coupe, et le rang de l’étage le plus proche avec une coupe', () => {
    const tout = mount(ready([0, 2.2, 4.4]))
    expect(tout.container.querySelector('output')?.textContent).toBe('Tout')
    tout.unmount()
    const coupe = mount(ready([0, 2.2, 4.4], { cut: 2.25 }))
    expect(coupe.container.querySelector('output')?.textContent).toBe('Étage 1')
  })
})

describe('FloorGroup — carte sans maillage et sans relief', () => {
  it('sans maillage : un bloc d’explication en pointillé, aucune pastille ni curseur', () => {
    const { container } = mount(stateOnly({ state: 'noMesh' }))
    expect(screen.getByText('Carte sans maillage de navigation : le relief est affiché en entier, sans coupe par étage.')).toBeTruthy()
    expect(container.querySelector('p.border-dashed')).not.toBeNull()
    expect(screen.queryByRole('button')).toBeNull()
    expect(screen.queryByRole('slider')).toBeNull()
  })

  it('sans relief lisible : rien du tout', () => {
    const { container } = mount(stateOnly({ state: 'none' }))
    expect(container.innerHTML).toBe('')
  })
})

describe('FloorGroup — en anglais', () => {
  it('Floor, All, Low, Floor 1 et le nom du curseur', () => {
    mount(ready([0, 2.2]), 'en')
    expect(screen.getAllByRole('button').map((b) => b.textContent)).toEqual(['All', 'Low', 'Floor 1'])
    expect(screen.getByText('Floor')).toBeTruthy()
    expect(screen.getByRole('slider', { name: 'Cut height' })).toBeTruthy()
    expect(screen.getByRole('button', { name: 'Floor 1' }).getAttribute('title')).toMatch(/^Floor 1: \d rings?$/)
    expect(screen.getByRole('button', { name: 'Low' })).toHaveAttribute('title', 'Low: no rings')
  })
})
