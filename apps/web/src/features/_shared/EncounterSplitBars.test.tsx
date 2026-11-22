import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'

import { WinLossSplitBar } from './EncounterSplitBars'

describe('WinLossSplitBar', () => {
  it('montre les victoires et les défaites de part et d’autre de la barre', () => {
    const { container } = render(<WinLossSplitBar wins={3} losses={1} locale="fr" />)
    expect(container.textContent).toBe('31')
    expect(container.querySelector('span[style*="outcome-win"]')).not.toBeNull()
    expect(container.querySelector('span[style*="outcome-loss"]')).not.toBeNull()
  })

  it('porte une infobulle accordée au nombre, dans la langue demandée', async () => {
    render(<WinLossSplitBar wins={1} losses={2} locale="en" />)
    await userEvent.hover(screen.getByText('1'))
    expect(await screen.findByText('1 win')).toBeInTheDocument()
    await userEvent.hover(screen.getByText('2'))
    expect(await screen.findByText('2 losses')).toBeInTheDocument()
  })

  it('rend un tiret quand il n’y a ni victoire ni défaite', () => {
    const { container } = render(<WinLossSplitBar wins={0} losses={0} locale="fr" />)
    expect(container.textContent).toBe('—')
  })
})
