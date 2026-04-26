/**
 * SquadSessionSelector.test.tsx — Le sélecteur de session escouade ne
 * doit PLUS exposer le sélecteur "Solo" (retiré : page solo dédiée
 * séparée). Les libellés viennent de getSquadText.
 */
import { describe, it, expect, vi, afterEach } from 'vitest'
import { screen, fireEvent } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'
import { SquadSessionSelector } from './SquadSessionSelector'

const SQUAD_SESSIONS = ['Session A', 'Session B', 'Session C']

afterEach(() => {
  useAppShellStore.setState({ locale: 'fr' })
})

describe('SquadSessionSelector', () => {
  it('ne rend rien quand aucune session escouade', () => {
    const { container } = renderWithProviders(
      <SquadSessionSelector
        sessionLabels={{ solo: ['ignored'], squad: [] }}
        squadSession={null}
        onSquadChange={() => {}}
      />,
    )
    expect(container.firstChild).toBeNull()
  })

  it('rend "Escouade" mais PAS "Solo" comme label de selector', () => {
    renderWithProviders(
      <SquadSessionSelector
        sessionLabels={{ solo: ['Solo session'], squad: SQUAD_SESSIONS }}
        squadSession={null}
        onSquadChange={() => {}}
      />,
    )
    expect(screen.getByText('Escouade')).toBeInTheDocument()
    expect(screen.queryByText('Solo')).toBeNull()
  })

  it('passe en EN quand le store locale = en', () => {
    useAppShellStore.setState({ locale: 'en' })
    renderWithProviders(
      <SquadSessionSelector
        sessionLabels={{ solo: [], squad: SQUAD_SESSIONS }}
        squadSession={null}
        onSquadChange={() => {}}
      />,
    )
    expect(screen.getByText('Squad')).toBeInTheDocument()
    expect(screen.queryByText('Escouade')).toBeNull()
  })

  it('appelle onSquadChange avec la valeur sélectionnée via le select', () => {
    const onChange = vi.fn()
    renderWithProviders(
      <SquadSessionSelector
        sessionLabels={{ solo: [], squad: SQUAD_SESSIONS }}
        squadSession={null}
        onSquadChange={onChange}
      />,
    )
    const select = screen.getByRole('combobox')
    fireEvent.change(select, { target: { value: 'Session B' } })
    expect(onChange).toHaveBeenCalledWith('Session B')
  })

  it('appelle onSquadChange(null) quand on choisit "(toutes)"', () => {
    const onChange = vi.fn()
    renderWithProviders(
      <SquadSessionSelector
        sessionLabels={{ solo: [], squad: SQUAD_SESSIONS }}
        squadSession={'Session A'}
        onSquadChange={onChange}
      />,
    )
    const select = screen.getByRole('combobox')
    fireEvent.change(select, { target: { value: '' } })
    expect(onChange).toHaveBeenCalledWith(null)
  })

  it('affiche le bouton Réinitialiser uniquement si une session est sélectionnée', () => {
    const { rerender } = renderWithProviders(
      <SquadSessionSelector
        sessionLabels={{ solo: [], squad: SQUAD_SESSIONS }}
        squadSession={null}
        onSquadChange={() => {}}
      />,
    )
    expect(screen.queryByText(/Réinitialiser/)).toBeNull()

    rerender(
      <SquadSessionSelector
        sessionLabels={{ solo: [], squad: SQUAD_SESSIONS }}
        squadSession={'Session A'}
        onSquadChange={() => {}}
      />,
    )
    expect(screen.getByText(/Réinitialiser/)).toBeInTheDocument()
  })

  it('navigation ← / → passe à la session précédente / suivante', () => {
    const onChange = vi.fn()
    renderWithProviders(
      <SquadSessionSelector
        sessionLabels={{ solo: [], squad: SQUAD_SESSIONS }}
        squadSession={'Session B'}
        onSquadChange={onChange}
      />,
    )
    const prev = screen.getByTitle('Session précédente')
    const next = screen.getByTitle('Session suivante')
    fireEvent.click(prev)
    expect(onChange).toHaveBeenCalledWith('Session A')
    onChange.mockClear()
    fireEvent.click(next)
    expect(onChange).toHaveBeenCalledWith('Session C')
  })
})
