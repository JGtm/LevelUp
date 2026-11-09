/**
 * AscensionLayout : le layout ne rend plus ni en-tête ni barre d'onglets (posés au-dessus par
 * `SectionHeader`, voir `components/shell/SectionHeader.test.tsx` pour les onglets, dont
 * « Tactique » derrière la capacité `replay`). Il garde le bandeau d'astuces, sous l'en-tête, puis
 * le contenu de l'onglet.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { screen } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'

import { AscensionLayout } from './AscensionLayout'

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    Outlet: () => <div data-testid="contenu-onglet" />,
  }
})

beforeEach(() => useAppShellStore.setState({ locale: 'fr' }))
afterEach(() => useAppShellStore.setState({ locale: 'fr' }))

describe('AscensionLayout — le contenu sous l’en-tête de section', () => {
  it('rend le contenu de l’onglet et aucun titre ni onglet de son cru', () => {
    renderWithProviders(<AscensionLayout />)
    expect(screen.getByTestId('contenu-onglet')).toBeInTheDocument()
    expect(screen.queryByRole('heading', { level: 1 })).toBeNull()
    expect(screen.queryAllByRole('tab')).toHaveLength(0)
    expect(screen.queryByRole('navigation')).toBeNull()
  })

  it('le bandeau d’astuces précède le contenu de l’onglet', () => {
    const { container } = renderWithProviders(<AscensionLayout />)
    const main = container.querySelector('main')
    const contenu = screen.getByTestId('contenu-onglet')
    expect(main?.firstElementChild).not.toBe(contenu)
    expect(main?.contains(contenu)).toBe(true)
  })
})

// LA LARGEUR (plan Tactique v2, L4.1) : la page occupe toute la largeur de la fenêtre, avec la
// marge de 24 px des pages pleine largeur — plus de conteneur centré borné à 6xl.
describe('AscensionLayout — la largeur de la page', () => {
  it('pleine largeur : ni conteneur centré ni borne 6xl, marge p-6', () => {
    const { container } = renderWithProviders(<AscensionLayout />)
    const main = container.querySelector('main')
    expect(main).not.toBeNull()
    const classes = (main?.className ?? '').split(/\s+/)
    expect(classes).not.toContain('container')
    expect(classes).not.toContain('max-w-6xl')
    expect(classes).toContain('p-6')
  })
})
