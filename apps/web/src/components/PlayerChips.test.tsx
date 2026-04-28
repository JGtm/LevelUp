/**
 * Tests Vitest pour PlayerChips (Phase 8).
 *
 * Couvre :
 * - rendu liste de chips
 * - click sur chip inactive -> onChange avec id
 * - click sur chip active -> onChange avec null (deselect)
 * - aria-pressed reflete l'etat actif
 */
import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

import { PlayerChips, type PlayerChipItem } from './PlayerChips'

vi.mock('@/lib/accessibility', () => ({
  tokenCssVar: (token: string) => `var(--${token})`,
}))

describe('PlayerChips', () => {
  const players: PlayerChipItem[] = [
    { id: 'xuid-1', label: 'Alice', colorToken: 'chart-series-1' },
    { id: 'xuid-2', label: 'Bob', colorToken: 'chart-series-2' },
  ]

  it('rend les chips avec leur label', () => {
    render(<PlayerChips players={players} selectedId={null} onChange={() => {}} />)
    expect(screen.getByText('Alice')).toBeInTheDocument()
    expect(screen.getByText('Bob')).toBeInTheDocument()
  })

  it('aria-pressed=false sur tous les chips si selectedId=null', () => {
    render(<PlayerChips players={players} selectedId={null} onChange={() => {}} />)
    const aliceBtn = screen.getByText('Alice').closest('button')
    const bobBtn = screen.getByText('Bob').closest('button')
    expect(aliceBtn).toHaveAttribute('aria-pressed', 'false')
    expect(bobBtn).toHaveAttribute('aria-pressed', 'false')
  })

  it('aria-pressed=true sur le chip selectionne', () => {
    render(<PlayerChips players={players} selectedId="xuid-1" onChange={() => {}} />)
    const aliceBtn = screen.getByText('Alice').closest('button')
    const bobBtn = screen.getByText('Bob').closest('button')
    expect(aliceBtn).toHaveAttribute('aria-pressed', 'true')
    expect(bobBtn).toHaveAttribute('aria-pressed', 'false')
  })

  it('click sur chip inactif appelle onChange avec son id', () => {
    const onChange = vi.fn()
    render(<PlayerChips players={players} selectedId={null} onChange={onChange} />)
    fireEvent.click(screen.getByText('Bob'))
    expect(onChange).toHaveBeenCalledWith('xuid-2')
  })

  it('click sur chip actif appelle onChange avec null (deselect)', () => {
    const onChange = vi.fn()
    render(<PlayerChips players={players} selectedId="xuid-1" onChange={onChange} />)
    fireEvent.click(screen.getByText('Alice'))
    expect(onChange).toHaveBeenCalledWith(null)
  })

  it('affiche le groupLabel si fourni', () => {
    render(
      <PlayerChips
        players={players}
        selectedId={null}
        onChange={() => {}}
        groupLabel="Afficher joueur :"
      />,
    )
    expect(screen.getByText('Afficher joueur :')).toBeInTheDocument()
  })
})
