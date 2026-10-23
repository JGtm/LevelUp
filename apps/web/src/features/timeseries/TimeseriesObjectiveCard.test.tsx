/**
 * TimeseriesObjectiveCard.test — les tuiles du cumul d'objectif (bloc « Objectif » de l'onglet Usages) :
 * une tuile par action qui vaut plus de zéro, aucune rangée sans action ni cumul.
 */
import { screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import type { ObjectiveAggregate } from '@/lib/api/types'
import { renderWithProviders } from '@/test/render-utils'

import { TimeseriesObjectiveCard } from './TimeseriesObjectiveCard'

const stats = (extra: Partial<ObjectiveAggregate>): ObjectiveAggregate => extra as ObjectiveAggregate

describe('TimeseriesObjectiveCard', () => {
  it('des actions d’objectif : une tuile par action non nulle', () => {
    renderWithProviders(<TimeseriesObjectiveCard stats={stats({ flag_captures: 3, flag_returns: 0 })} locale="fr" />)
    expect(screen.getByText('Captures de drapeau')).toBeInTheDocument()
    expect(screen.getByText('3')).toBeInTheDocument()
    expect(screen.queryByText('Retours de drapeau')).toBeNull()
  })

  it('aucune action d’objectif ou aucun cumul : la rangée se retire', () => {
    for (const s of [stats({ flag_captures: 0 }), undefined, null]) {
      const { container, unmount } = renderWithProviders(<TimeseriesObjectiveCard stats={s} locale="fr" />)
      expect(container).toBeEmptyDOMElement()
      unmount()
    }
  })

  it('en anglais, les libellés sont traduits', () => {
    renderWithProviders(<TimeseriesObjectiveCard stats={stats({ zone_captures: 2 })} locale="en" />)
    expect(screen.getByText('Zones captured')).toBeInTheDocument()
  })
})
