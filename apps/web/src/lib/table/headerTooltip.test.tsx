/**
 * Tests — tooltips d'en-tête de colonne (V72-04, refondus V73-L2 2.4c).
 *
 * L'aide n'est plus une icône ⓘ SŒUR du contrôle de tri : elle est portée par le
 * LIBELLÉ lui-même. Ce qui doit rester vrai, et que ces tests verrouillent :
 *
 *  1. plus AUCUN bouton d'aide dans un en-tête (l'icône a disparu des deux
 *     familles) — sinon les deux registres coexisteraient ;
 *  2. le survol du libellé révèle l'aide ;
 *  3. le clic sur le libellé TRIE (il ne bascule pas une aide) — c'est la
 *     contrepartie du point 1 ;
 *  4. le bouton de tri n'est JAMAIS imbriqué dans un autre bouton (HTML invalide,
 *     le piège d'origine) ;
 *  5. un en-tête non triable garde l'aide atteignable au clavier (`focusable`).
 *
 * Famille 1 : `HeaderLabelTooltip` (les 8 <thead> TanStack).
 * Famille 2 : prop `tooltip` de `SortableTh` (tableaux HTML natifs admin/carrière).
 */
import { type ReactNode } from 'react'
import { describe, it, expect } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'

import { HeaderLabelTooltip } from './columnMeta'
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

describe('HeaderLabelTooltip (famille 1)', () => {
  it('rend le libellé inchangé quand il n’y a pas d’aide', () => {
    render(<HeaderLabelTooltip>Frags</HeaderLabelTooltip>)
    expect(screen.getByText('Frags')).toBeInTheDocument()
    // Aucun bouton parasite : la colonne sans aide ne gagne pas de contrôle.
    expect(screen.queryByRole('button')).toBeNull()
  })

  it('révèle l’aide au survol du libellé, sans icône ⓘ', () => {
    render(<HeaderLabelTooltip text="Frags réalisés durant le match.">Frags</HeaderLabelTooltip>)
    expect(screen.queryByRole('tooltip')).toBeNull()
    // Aucun bouton d'aide : c'est le libellé qui déclenche.
    expect(screen.queryByRole('button')).toBeNull()

    fireEvent.mouseEnter(screen.getByText('Frags'))
    expect(screen.getByRole('tooltip')).toHaveTextContent('Frags réalisés durant le match.')

    fireEvent.mouseLeave(screen.getByText('Frags'))
    expect(screen.queryByRole('tooltip')).toBeNull()
  })

  it('n’imbrique pas de bouton dans le bouton de tri, et le clic trie', () => {
    let sorted = 0
    renderInRow(
      <th>
        <HeaderLabelTooltip text="Impact par match.">
          <button type="button" onClick={() => (sorted += 1)}>
            FDA
          </button>
        </HeaderLabelTooltip>
      </th>,
    )
    // Un seul bouton dans l'en-tête : celui du tri (l'aide n'en ajoute aucun).
    const buttons = screen.getAllByRole('button')
    expect(buttons).toHaveLength(1)
    expect(buttons[0].querySelector('button')).toBeNull()

    // Le clic appartient au tri, pas à l'aide.
    fireEvent.click(buttons[0])
    expect(sorted).toBe(1)
  })

  it('ouvre l’aide au FOCUS clavier du bouton de tri enveloppé', () => {
    renderInRow(
      <th>
        <HeaderLabelTooltip text="Impact par match.">
          <button type="button" onClick={() => {}}>
            FDA
          </button>
        </HeaderLabelTooltip>
      </th>,
    )
    fireEvent.focus(screen.getByRole('button'))
    expect(screen.getByRole('tooltip')).toHaveTextContent('Impact par match.')
  })

  it('en-tête NON triable : le libellé devient atteignable au clavier (focusable)', () => {
    renderInRow(
      <th>
        <HeaderLabelTooltip text="Contexte du match." focusable>
          Contexte
        </HeaderLabelTooltip>
      </th>,
    )
    const trigger = screen.getByText('Contexte')
    expect(trigger).toHaveAttribute('tabindex', '0')
    fireEvent.focus(trigger)
    expect(screen.getByRole('tooltip')).toHaveTextContent('Contexte du match.')
  })
})

describe('SortableTh — prop tooltip (famille 2)', () => {
  it('sans tooltip : libellé + bouton de tri, aucune icône ⓘ', () => {
    renderInRow(<SortableTh label="Score" active={false} dir="desc" onClick={() => {}} />)
    expect(screen.getAllByRole('button')).toHaveLength(1)
    expect(screen.getByText('Score')).toBeInTheDocument()
  })

  it('avec tooltip : toujours UN SEUL bouton (le tri), aide au survol du libellé', () => {
    renderInRow(
      <SortableTh
        label="FDA"
        active={false}
        dir="desc"
        onClick={() => {}}
        tooltip="Impact par match, pas frags/morts."
      />,
    )
    // L'ancienne icône ⓘ ajoutait un 2e bouton : elle ne doit plus exister.
    const buttons = screen.getAllByRole('button')
    expect(buttons).toHaveLength(1)
    expect(buttons[0].textContent).toContain('FDA')

    fireEvent.mouseEnter(screen.getByText('FDA'))
    expect(screen.getByRole('tooltip')).toHaveTextContent('Impact par match, pas frags/morts.')
  })

  it('cliquer le libellé aidé déclenche bien le tri', () => {
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
    fireEvent.click(screen.getByRole('button'))
    expect(sorted).toBe(true)
  })
})
