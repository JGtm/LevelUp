/**
 * TimeseriesObjectiveCard.test — le bandeau « Objectifs » : KPI du cumul d'objectif quand le périmètre
 * en a, constat « Aucune action d'objectif sur ce périmètre » dans un bloc quand il n'en a aucun, absent
 * quand le titre ne mesure pas les objectifs (capability `objective_stats`).
 */
import { screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

import type { ObjectiveAggregate } from '@/lib/api/types'
import { renderWithProviders } from '@/test/render-utils'

import { TimeseriesObjectiveCard } from './TimeseriesObjectiveCard'

const capability = vi.hoisted(() => ({ has: true }))
vi.mock('@/lib/capabilities/capabilities', () => ({ useCapability: () => capability.has }))

const stats = (extra: Partial<ObjectiveAggregate>): ObjectiveAggregate => extra as ObjectiveAggregate

describe('TimeseriesObjectiveCard', () => {
  it('des actions d’objectif : les KPI, pas de constat', () => {
    capability.has = true
    renderWithProviders(<TimeseriesObjectiveCard stats={stats({ flag_captures: 3 })} locale="fr" />)
    expect(screen.getByText('3')).toBeInTheDocument()
    expect(screen.queryByTestId('timeseries-objectives-empty')).toBeNull()
  })

  it('aucune action d’objectif ou aucun cumul : le bandeau reste, titre gardé, avec le constat dans un bloc', () => {
    capability.has = true
    for (const s of [stats({ flag_captures: 0 }), undefined, null]) {
      const { unmount } = renderWithProviders(<TimeseriesObjectiveCard stats={s} locale="fr" />)
      const bloc = screen.getByTestId('timeseries-objectives-empty')
      expect(bloc).toHaveTextContent('Aucune action d’objectif sur ce périmètre')
      expect(bloc.querySelector('.border-dashed')).not.toBeNull()
      expect(screen.getByRole('region')).toContainElement(bloc)
      unmount()
    }
  })

  it('en anglais, le constat est traduit', () => {
    capability.has = true
    renderWithProviders(<TimeseriesObjectiveCard stats={null} locale="en" />)
    expect(screen.getByTestId('timeseries-objectives-empty')).toHaveTextContent('No objective action in this scope')
  })

  it('titre sans statistiques d’objectif (capability absente) : le bandeau est retiré', () => {
    capability.has = false
    const { container } = renderWithProviders(<TimeseriesObjectiveCard stats={null} locale="fr" />)
    expect(container).toBeEmptyDOMElement()
  })
})
