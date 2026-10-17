/**
 * La Synthèse sans aucun frag : l'anneau « Répartition des frags » reste dans sa rangée avec son
 * constat « Aucun frag sur ce périmètre » (bloc parent + pointillé), au lieu de disparaître.
 */
import { screen } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import { describe, expect, it, vi } from 'vitest'
import type { ComponentPropsWithoutRef } from 'react'

import { synthesisFixture } from '@/test/handlers'
import { renderWithProviders } from '@/test/render-utils'
import { server } from '@/test/setup'

import { SynthesisPage } from './SynthesisPage'

vi.mock('@/components/charts/ChartCard', () => ({ ChartCard: () => <div data-testid="chart-card" /> }))
vi.mock('@/components/charts/Heatmap2DChart', () => ({ Heatmap2DChart: () => <div data-testid="chart-card" /> }))
vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    useNavigate: () => vi.fn(),
    useParams: () => ({ playerSlug: 'test-player' }),
    Link: ({ children, to, ...props }: ComponentPropsWithoutRef<'a'> & { to: string }) => (
      <a href={to} {...props}>{children}</a>
    ),
  }
})

const detailed = {
  total_betrayals: 0,
  total_suicides: 0,
  total_perfect_kills: 0,
  total_headshot_kills: 0,
  total_shots_fired: 0,
  total_shots_hit: 0,
  total_damage_dealt: 0,
  total_damage_taken: 0,
  total_assassinations: 0,
  total_ground_pound_kills: 0,
  total_shoulder_bash_kills: 0,
  total_vehicles_destroyed: 0,
  total_hijacks: 0,
}

describe('Synthèse — aucun frag', () => {
  it('l’anneau des frags reste, avec le constat dans un bloc', async () => {
    server.use(
      http.post('/api/v1/players/:playerSlug/pages/synthesis', () =>
        HttpResponse.json({ ...synthesisFixture, detailed_stats: detailed, frag_distribution: null }),
      ),
    )
    renderWithProviders(<SynthesisPage />)
    const bloc = await screen.findByTestId('frag-sunburst-empty')
    expect(bloc).toHaveTextContent('Répartition des frags')
    expect(bloc).toHaveTextContent('Aucun frag sur ce périmètre')
    expect(bloc.querySelector('.border-dashed')).not.toBeNull()
  })
})
