/**
 * HelpSplitButton — fermeture : Échap et un appui hors du bouton ferment le menu ; un appui dedans non.
 */
import type { ComponentPropsWithoutRef } from 'react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, screen } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'

import { HelpSplitButton } from './HelpSplitButton'

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    Link: ({ children, to, ...props }: ComponentPropsWithoutRef<'a'> & { to: string }) => (
      <a href={to} {...props}>
        {children}
      </a>
    ),
  }
})

function ouvre() {
  renderWithProviders(
    <div>
      <button data-testid="dehors">dehors</button>
      <HelpSplitButton isActive={false} />
    </div>,
  )
  const declencheur = screen.getByRole('button', { name: 'Onglets Aide' })
  fireEvent.click(declencheur)
  expect(screen.getByRole('menu')).toBeInTheDocument()
  return declencheur
}

describe('HelpSplitButton — fermeture', () => {
  beforeEach(() => useAppShellStore.setState({ locale: 'fr' }))

  it('Échap ferme le menu', () => {
    const declencheur = ouvre()
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(screen.queryByRole('menu')).toBeNull()
    expect(declencheur).toHaveAttribute('aria-expanded', 'false')
  })

  it('un appui hors du bouton ferme, un appui dans le menu ne ferme pas', () => {
    ouvre()
    fireEvent.mouseDown(screen.getByRole('menu'))
    expect(screen.getByRole('menu')).toBeInTheDocument()
    fireEvent.mouseDown(screen.getByTestId('dehors'))
    expect(screen.queryByRole('menu')).toBeNull()
  })
})
