/**
 * Tests — tooltips d'en-tête de colonne (V72-04).
 *
 * Famille 1 : helper `HeaderInfoTooltip` (utilisé par les 8 <thead> TanStack) —
 * rend `null` sans texte, rend une icône ⓘ (portal InfoTooltip) qui révèle le
 * contenu au clic, et stoppe la propagation du clic pour ne PAS déclencher le
 * tri des <th> qui portent l'`onClick` (MatchScoreboard, DetectionsPanel…).
 *
 * Famille 2 : prop `tooltip` de `SortableTh` — icône ⓘ rendue en SŒUR du bouton
 * de tri (jamais imbriquée dedans : deux <button> imbriqués = HTML invalide).
 */
import { type ReactNode } from 'react'
import { describe, it, expect } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'

import { HeaderInfoTooltip } from './columnMeta'
import { SortableTh } from '@/components/ui/sortable-th'

function renderInRow(ui: ReactNode) {
  return render(
    <table>
      <thead>
        <tr>{ui}</tr>
      </thead>
    </table>,
  )
}

describe('HeaderInfoTooltip (famille 1)', () => {
  it('ne rend rien quand le texte est absent', () => {
    render(<HeaderInfoTooltip />)
    expect(screen.queryByRole('button')).toBeNull()
  })

  it('rend une icône ⓘ qui révèle le contenu au clic', () => {
    render(<HeaderInfoTooltip text="Frags réalisés durant le match." />)
    expect(screen.queryByRole('tooltip')).toBeNull()
    fireEvent.click(screen.getByRole('button'))
    expect(screen.getByRole('tooltip')).toHaveTextContent('Frags réalisés durant le match.')
  })

  it('stoppe la propagation du clic (le <th> cliquable ne trie pas)', () => {
    let thClicked = false
    renderInRow(
      <th
        onClick={() => {
          thClicked = true
        }}
      >
        Frags
        <HeaderInfoTooltip text="Frags réalisés." />
      </th>,
    )
    fireEvent.click(screen.getByRole('button'))
    expect(thClicked).toBe(false)
  })
})

describe('SortableTh — prop tooltip (famille 2)', () => {
  it('sans tooltip : libellé + bouton de tri, aucune icône ⓘ', () => {
    renderInRow(<SortableTh label="Score" active={false} dir="desc" onClick={() => {}} />)
    expect(screen.getAllByRole('button')).toHaveLength(1)
    expect(screen.getByText('Score')).toBeInTheDocument()
  })

  it('avec tooltip : icône ⓘ sœur du bouton de tri (jamais imbriquée), contenu au clic', () => {
    renderInRow(
      <SortableTh
        label="FDA"
        active={false}
        dir="desc"
        onClick={() => {}}
        tooltip="Impact par match, pas frags/morts."
      />,
    )
    const buttons = screen.getAllByRole('button')
    expect(buttons).toHaveLength(2)
    const sortButton = buttons.find((b) => b.textContent?.includes('FDA'))!
    const infoButton = buttons.find((b) => b !== sortButton)!
    // L'icône n'est PAS un descendant du bouton de tri (pas de bouton imbriqué).
    expect(sortButton.contains(infoButton)).toBe(false)
    fireEvent.click(infoButton)
    expect(screen.getByRole('tooltip')).toHaveTextContent('Impact par match, pas frags/morts.')
  })

  it('cliquer l’icône ⓘ ne déclenche pas le tri', () => {
    let sorted = false
    renderInRow(
      <SortableTh
        label="FDA"
        active={false}
        dir="desc"
        onClick={() => {
          sorted = true
        }}
        tooltip="Aide."
      />,
    )
    const infoButton = screen.getAllByRole('button').find((b) => !b.textContent?.includes('FDA'))!
    fireEvent.click(infoButton)
    expect(sorted).toBe(false)
  })
})
