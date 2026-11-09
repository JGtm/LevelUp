import { describe, expect, it } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import { focusByKeyboard, focusByMouse } from '@/test/focus'
import { Tooltip } from './tooltip'

function renderTooltip() {
  render(
    <Tooltip content="Explication">
      <button type="button">Cible</button>
    </Tooltip>,
  )
  return screen.getByRole('button', { name: 'Cible' })
}

describe('Tooltip', () => {
  it('s’ouvre au survol', () => {
    const cible = renderTooltip()
    fireEvent.mouseEnter(cible.parentElement as HTMLElement)
    expect(screen.getByRole('tooltip')).toHaveTextContent('Explication')
    fireEvent.mouseLeave(cible.parentElement as HTMLElement)
    expect(screen.queryByRole('tooltip')).not.toBeInTheDocument()
  })

  it('s’ouvre au focus clavier et se ferme quand le focus part', () => {
    const cible = renderTooltip()
    focusByKeyboard(cible)
    expect(screen.getByRole('tooltip')).toHaveTextContent('Explication')
    fireEvent.blur(cible)
    expect(screen.queryByRole('tooltip')).not.toBeInTheDocument()
  })

  it('n’apparaît pas sur un focus qui ne vient pas du clavier', () => {
    const cible = renderTooltip()
    focusByMouse(cible)
    expect(screen.queryByRole('tooltip')).not.toBeInTheDocument()
  })
})
