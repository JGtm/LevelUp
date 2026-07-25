import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'

import { ReviewBadge } from './ReviewBadge'
import { useAppShellStore } from '@/stores/appShellStore'
import type { ChartReview } from '@/lib/review/chart-review'

const KNOWN_KEY = 'test.chart'

// Le manifeste réel se vide à la fin d'une tournée de revue : le badge est donc
// testé contre un manifeste STUBÉ (son contenu réel est couvert par
// lib/review/chart-review.test.ts).
const stub = vi.hoisted(() => ({ entry: undefined as unknown }))

vi.mock('@/lib/review/chart-review', () => ({
  chartReview: (key?: string) => (key === 'test.chart' ? stub.entry : undefined),
}))

function setEntry(entry: ChartReview | undefined) {
  stub.entry = entry
}

describe('ReviewBadge', () => {
  beforeEach(() => {
    useAppShellStore.setState({ locale: 'fr' })
    setEntry(undefined)
  })

  it('ne rend RIEN sans clé (mécanisme inerte)', () => {
    const { container } = render(<ReviewBadge />)
    expect(container.innerHTML).toBe('')
  })

  it('ne rend RIEN pour une clé absente du manifeste (manifeste vide = badge inerte)', () => {
    const { container } = render(<ReviewBadge reviewKey="chart.absente" />)
    expect(container.innerHTML).toBe('')
  })

  it('rend le libellé FR selon le statut', () => {
    setEntry({ status: 'verify', note: { fr: 'note fr', en: 'note en' } })
    render(<ReviewBadge reviewKey={KNOWN_KEY} />)
    const badge = screen.getByTestId('chart-review-badge')
    expect(badge.textContent).toBe('À vérifier')
    expect(badge.getAttribute('data-review-status')).toBe('verify')
    expect(badge.getAttribute('title')).toContain('note fr')
  })

  it('rend le libellé EN quand la locale bascule', () => {
    useAppShellStore.setState({ locale: 'en' })
    setEntry({ status: 'new', note: { fr: 'note fr', en: 'note en' } })
    render(<ReviewBadge reviewKey={KNOWN_KEY} />)
    const badge = screen.getByTestId('chart-review-badge')
    expect(badge.textContent).toBe('New')
    expect(badge.getAttribute('title')).toContain('note en')
  })

  it('rend le statut « suppression » avec son libellé dédié', () => {
    setEntry({ status: 'removal', note: { fr: 'note fr', en: 'note en' } })
    render(<ReviewBadge reviewKey={KNOWN_KEY} />)
    expect(screen.getByTestId('chart-review-badge').textContent).toBe('Suppression ?')
  })
})
