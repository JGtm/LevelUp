/**
 * Tests — ReliefPills : la pilule « Fond », le bouton « Traits du relief » et la pilule « Étage » des vues 2D sans
 * panneau « Vue ». Ce que les commandes DISENT et ce qu'elles COMMANDENT : l'ordre des fonds, des étages en rangs
 * sans unité, jamais une commande qui ne commande rien (carte sans maillage).
 */
import { fireEvent, render, screen, within } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

import type { ReliefViewFloor } from '@/lib/replay/useReliefView'

import { FLOOR_CHIPS_SCROLL_MIN, ReliefBackgroundPill, ReliefFloorPill, ReliefLinesButton } from './ReliefPills'
import { RELIEF_VIEW_TEXT } from './text'

const T = RELIEF_VIEW_TEXT.fr

function floor(over: Partial<ReliefViewFloor> = {}): ReliefViewFloor {
  return {
    state: {
      state: 'ready',
      chips: [
        { rank: 0, z: 0 },
        { rank: 1, z: 2.2 },
        { rank: 2, z: 4.4 },
      ],
      min: -0.5,
      max: 5,
    },
    cut: null,
    selectAll: vi.fn(),
    selectRank: vi.fn(),
    setHeight: vi.fn(),
    ...over,
  }
}

describe('ReliefBackgroundPill', () => {
  it('propose Plan, Rendu, Relief dans cet ordre et retient la valeur choisie', () => {
    const onSelect = vi.fn()
    render(<ReliefBackgroundPill text={T} options={['plan', 'rendu', 'relief']} selected="rendu" onSelect={onSelect} meshPercent={null} />)
    const group = screen.getByRole('group', { name: 'Fond' })
    expect(within(group).getAllByRole('button').map((b) => b.textContent)).toEqual(['Plan', 'Rendu', 'Relief'])
    expect(within(group).getByRole('button', { name: 'Rendu' })).toHaveAttribute('aria-pressed', 'true')
    fireEvent.click(within(group).getByRole('button', { name: 'Relief' }))
    expect(onSelect).toHaveBeenCalledWith('relief')
  })

  it('ne pose rien quand il n’y a pas de choix', () => {
    const { container } = render(<ReliefBackgroundPill text={T} options={[]} selected="rendu" onSelect={vi.fn()} meshPercent={null} />)
    expect(container).toBeEmptyDOMElement()
  })

  it('l’infobulle de « Relief » dit la part tirée du maillage quand elle est connue (D-16)', () => {
    render(<ReliefBackgroundPill text={T} options={['rendu', 'relief']} selected="rendu" onSelect={vi.fn()} meshPercent={46} />)
    expect(screen.getByRole('button', { name: 'Relief' }).title).toContain('46 % de la carte')
  })

  it('l’infobulle dit la lumière de l’ÉCRAN, jamais un point cardinal de la carte (D-15), dans les deux langues', () => {
    for (const [locale, upperLeft] of [['fr', 'haut à gauche'], ['en', 'upper left']] as const) {
      expect(RELIEF_VIEW_TEXT[locale].reliefHint).toContain(upperLeft)
      expect(RELIEF_VIEW_TEXT[locale].reliefHint).not.toMatch(/nord|north/i)
    }
  })

  it('sans part connue, l’infobulle ne cite aucun pourcentage', () => {
    render(<ReliefBackgroundPill text={T} options={['rendu', 'relief']} selected="rendu" onSelect={vi.fn()} meshPercent={null} />)
    expect(screen.getByRole('button', { name: 'Relief' }).title).not.toContain('%')
  })

  it('se dit en anglais', () => {
    render(<ReliefBackgroundPill text={RELIEF_VIEW_TEXT.en} options={['plan', 'rendu']} selected="plan" onSelect={vi.fn()} meshPercent={null} />)
    expect(screen.getByRole('group', { name: 'Background' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Render' })).toBeInTheDocument()
  })
})

describe('ReliefLinesButton', () => {
  it('reflète l’état et le bascule', () => {
    const onToggle = vi.fn()
    render(<ReliefLinesButton text={T} available show onToggle={onToggle} />)
    const button = screen.getByRole('button', { name: 'Traits du relief' })
    expect(button).toHaveAttribute('aria-pressed', 'true')
    fireEvent.click(button)
    expect(onToggle).toHaveBeenCalledTimes(1)
  })

  it('ne pose rien tant que les données du relief ne sont pas lues', () => {
    const { container } = render(<ReliefLinesButton text={T} available={false} show={false} onToggle={vi.fn()} />)
    expect(container).toBeEmptyDOMElement()
  })
})

describe('ReliefFloorPill', () => {
  it('« Tout » puis un rang par étage, sans unité : Bas, Étage 1, Étage 2', () => {
    render(<ReliefFloorPill text={T} floor={floor()} />)
    const group = screen.getByRole('group', { name: 'Étage' })
    expect(within(group).getAllByRole('button').map((b) => b.textContent)).toEqual(['Tout', 'Bas', 'Étage 1', 'Étage 2'])
    expect(within(group).getByRole('button', { name: 'Tout' })).toHaveAttribute('aria-pressed', 'true')
    expect(group.textContent).not.toMatch(/\d\s*m\b/)
  })

  it('une pastille retient son étage à sa hauteur exacte, « Tout » lève la coupe', () => {
    const f = floor()
    render(<ReliefFloorPill text={T} floor={f} />)
    fireEvent.click(screen.getByRole('button', { name: 'Étage 1' }))
    expect(f.selectRank).toHaveBeenCalledWith(2.2)
    fireEvent.click(screen.getByRole('button', { name: 'Tout' }))
    expect(f.selectAll).toHaveBeenCalledTimes(1)
  })

  it('la coupe retenue allume sa pastille et le curseur la nomme', () => {
    render(<ReliefFloorPill text={T} floor={floor({ cut: 2.2 })} />)
    expect(screen.getByRole('button', { name: 'Étage 1' })).toHaveAttribute('aria-pressed', 'true')
    expect(screen.getByRole('button', { name: 'Tout' })).toHaveAttribute('aria-pressed', 'false')
    expect(screen.getByRole('slider', { name: 'Hauteur de la coupe' })).toHaveValue('2.2')
    expect(screen.getByRole('slider').closest('label')?.querySelector('output')).toHaveTextContent('Étage 1')
  })

  it('une hauteur libre au curseur ne retient aucune pastille', () => {
    const f = floor({ cut: 1.25 })
    render(<ReliefFloorPill text={T} floor={f} />)
    for (const name of ['Tout', 'Bas', 'Étage 1', 'Étage 2']) {
      expect(screen.getByRole('button', { name })).toHaveAttribute('aria-pressed', 'false')
    }
    fireEvent.change(screen.getByRole('slider'), { target: { value: '3' } })
    expect(f.setHeight).toHaveBeenCalledWith(3)
  })

  it('carte sans maillage : la pilule « Étage » n’est pas posée', () => {
    const { container } = render(<ReliefFloorPill text={T} floor={floor({ state: { state: 'noMesh' } })} />)
    expect(container).toBeEmptyDOMElement()
  })

  it('relief pas encore lu : rien', () => {
    const { container } = render(<ReliefFloorPill text={T} floor={floor({ state: { state: 'none' } })} />)
    expect(container).toBeEmptyDOMElement()
  })

  it('au-delà de six étages, la rangée défile au lieu de s’étendre', () => {
    const chips = Array.from({ length: FLOOR_CHIPS_SCROLL_MIN + 1 }, (_, rank) => ({ rank, z: rank * 2 }))
    render(<ReliefFloorPill text={T} floor={floor({ state: { state: 'ready', chips, min: 0, max: 14 } })} />)
    expect(screen.getByRole('button', { name: 'Tout' }).parentElement?.className).toContain('overflow-x-auto')
  })

  it('sans étage détecté : « Tout » et le curseur seuls, sans nom de rang pour une hauteur libre', () => {
    render(<ReliefFloorPill text={T} floor={floor({ state: { state: 'ready', chips: [], min: 0, max: 6 }, cut: 2.5 })} />)
    expect(screen.getAllByRole('button')).toHaveLength(1)
    expect(screen.getByRole('slider')).toBeInTheDocument()
    expect(screen.getByRole('slider').closest('label')?.querySelector('output')).toBeEmptyDOMElement()
  })

  it('au maximum que la brique cuit (cinq étages), tout est posé et rien ne défile', () => {
    const chips = Array.from({ length: 5 }, (_, rank) => ({ rank, z: rank * 2 }))
    render(<ReliefFloorPill text={T} floor={floor({ state: { state: 'ready', chips, min: 0, max: 8 } })} />)
    expect(screen.getByRole('button', { name: 'Tout' }).parentElement?.className).not.toContain('overflow-x-auto')
    expect(screen.getByRole('button', { name: 'Étage 4' })).toBeInTheDocument()
    expect(screen.getByRole('slider')).toBeInTheDocument()
  })

  it('jusqu’à six étages, aucun défilement', () => {
    render(<ReliefFloorPill text={T} floor={floor()} />)
    expect(screen.getByRole('button', { name: 'Tout' }).parentElement?.className).not.toContain('overflow-x-auto')
  })
})
