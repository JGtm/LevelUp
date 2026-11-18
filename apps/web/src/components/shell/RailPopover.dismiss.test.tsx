/**
 * RailMenu — fermeture : Échap ferme le menu ouvert (au survol ou épinglé) et rend le focus au déclencheur
 * quand il était dans l'entrée ; un appui hors de l'entrée ne ferme que le menu épinglé.
 */
import { describe, expect, it } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'

import { RailMenu } from './RailPopover'

function rendu() {
  render(
    <div>
      <button data-testid="dehors">dehors</button>
      <RailMenu label="Stats" trigger="S" triggerClassName="rail">
        {(close) => (
          <button type="button" role="menuitem" onClick={close}>
            Séries
          </button>
        )}
      </RailMenu>
    </div>,
  )
  return screen.getByRole('button', { name: 'Stats' })
}

/** Un clic à la souris (`detail` 1) épingle le menu. */
const epingle = (declencheur: HTMLElement) => fireEvent.click(declencheur, { detail: 1 })

describe('RailMenu — fermeture', () => {
  it('menu épinglé : Échap ferme, un appui dehors ferme, un appui dedans non', () => {
    const declencheur = rendu()
    epingle(declencheur)
    expect(screen.getByRole('menu')).toBeInTheDocument()
    fireEvent.mouseDown(screen.getByRole('menu'))
    expect(screen.getByRole('menu')).toBeInTheDocument()
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(screen.queryByRole('menu')).toBeNull()

    epingle(declencheur)
    fireEvent.mouseDown(screen.getByTestId('dehors'))
    expect(screen.queryByRole('menu')).toBeNull()
  })

  it('menu ouvert au survol : Échap ferme, un appui dehors ne ferme pas', () => {
    const declencheur = rendu()
    fireEvent.mouseEnter(declencheur.parentElement as HTMLElement)
    expect(screen.getByRole('menu')).toBeInTheDocument()
    fireEvent.mouseDown(screen.getByTestId('dehors'))
    expect(screen.getByRole('menu')).toBeInTheDocument()
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(screen.queryByRole('menu')).toBeNull()
  })

  it('Échap depuis une entrée du menu rend le focus au déclencheur ; sans focus dans l’entrée, il n’est pas volé', () => {
    const declencheur = rendu()
    epingle(declencheur)
    screen.getByRole('menuitem').focus()
    fireEvent.keyDown(screen.getByRole('menuitem'), { key: 'Escape' })
    expect(screen.queryByRole('menu')).toBeNull()
    expect(declencheur).toHaveFocus()

    declencheur.blur()
    fireEvent.mouseEnter(declencheur.parentElement as HTMLElement)
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(screen.queryByRole('menu')).toBeNull()
    expect(declencheur).not.toHaveFocus()
  })

  it('une flèche ne ferme pas le menu', () => {
    const declencheur = rendu()
    epingle(declencheur)
    fireEvent.keyDown(screen.getByRole('menuitem'), { key: 'ArrowDown' })
    expect(screen.getByRole('menu')).toBeInTheDocument()
  })
})
