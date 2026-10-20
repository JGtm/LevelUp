import { describe, expect, it } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'

import type { RelationAssists } from '@/lib/api/types'

import { AssistExchangeCell } from './AssistExchangeCell'

// JGtm × Chocoboflor (base du 08/10) : 476 assistances données, 658 reçues, sur les frags
// officiels des matchs ensemble dont le film porte l'assistance.
const ASSISTS: RelationAssists = {
  my_frags: 4247,
  partner_frags: 3912,
  received: { total: 658, low: 200, mid: 300, high: 158 },
  given: { total: 476, low: 150, mid: 200, high: 126 },
}

describe('AssistExchangeCell', () => {
  it('dit données puis reçues, et son infobulle ne porte aucune mention de couverture', () => {
    render(<AssistExchangeCell assists={ASSISTS} locale="fr" />)
    const cell = screen.getByTestId('assist-cell')
    expect(cell.textContent).toBe('476658')
    fireEvent.mouseEnter(cell.parentElement as HTMLElement)
    const tip = screen.getByRole('tooltip')
    expect(tip.textContent).toContain("Tu l'as assisté 476 fois")
    expect(tip.textContent).toContain("Il t'a assisté 658 fois")
    expect(tip.textContent).not.toMatch(/mesur/i)
  })

  it('rend « — » seul quand aucun match ensemble ne porte l’assistance', () => {
    const { container } = render(<AssistExchangeCell assists={null} locale="fr" />)
    expect(container.textContent).toBe('—')
    expect(screen.queryByRole('tooltip')).toBeNull()
  })
})
