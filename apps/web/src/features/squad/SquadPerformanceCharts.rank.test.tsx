/**
 * SquadPerformanceCharts — le sélecteur de chaîne du sous-graphe « Rang » (décision du 14/11) :
 * un rang ne se compare qu'à l'intérieur de sa chaîne (CSR classé, un groupe LUSR par chaîne).
 * Une rangée de boutons au-dessus du graphe, libellés des chaînes du manifeste de la Carrière avec
 * leur nombre de matchs ; la chaîne la plus jouée par défaut ; une seule chaîne : pas de sélecteur.
 */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, screen, waitFor, within } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'
import type { SquadPerformanceSeriesPoint } from '@/lib/api/types'

import { getSquadText } from './i18n'
import { SquadPerformanceCharts } from './SquadPerformanceCharts'

const options: Record<string, unknown>[] = []
vi.mock('echarts-for-react', () => ({
  default: (props: { option: Record<string, unknown> }) => {
    options.push(props.option)
    return <div data-testid="echarts-stub" />
  },
}))

beforeEach(() => {
  options.length = 0
  useAppShellStore.setState({ locale: 'fr' })
})

const point = (order: number, rating: number, group: string | undefined, delta: number): SquadPerformanceSeriesPoint =>
  ({
    match_order: order,
    match_id: `m${order}`,
    start_time: '2026-09-10T20:00:00Z',
    kills: 10,
    deaths: 8,
    assists: 3,
    skill_rating: rating,
    skill_rating_type: group ? 'lusr' : 'csr',
    skill_playlist_group: group,
    skill_delta: delta,
  }) as SquadPerformanceSeriesPoint

const TWO_CHAINS = {
  JGtm: [point(0, 60, 'arena_slayer', 2), point(1, 62, 'arena_slayer', 2), point(2, 61, 'arena_slayer', -1), point(3, 140, 'arena_objectif', 5)],
}
const ONE_CHAIN = { JGtm: [point(0, 1500, undefined, 4), point(1, 1510, undefined, 10)] }

function mount(rows: Record<string, SquadPerformanceSeriesPoint[]>) {
  const labels = getSquadText('fr').performanceCharts
  return renderWithProviders(
    <SquadPerformanceCharts rowsByPlayer={rows} playerOrder={['JGtm']} colorByPlayer={{ JGtm: '#2255cc' }} labels={labels} />,
  )
}

/** La dernière option du graphe de rang (la seule à porter des `title` de bandeau). */
const rankOption = () =>
  [...options].reverse().find((o) => Array.isArray((o as { title?: unknown }).title)) as unknown as {
    title: { text: string; subtext: string }[]
    yAxis: { min: number; max: number }[]
  }

describe('sélecteur de chaîne du rang', () => {
  it('une rangée de boutons, un par chaîne présente, libellé de la Carrière et nombre de matchs', () => {
    mount(TWO_CHAINS)
    const group = screen.getByRole('group', { name: 'Chaîne de rang' })
    expect(within(group).getAllByRole('button').map((b) => b.textContent)).toEqual([
      'Social · Assassin (3)',
      'Social · Objectif (1)',
    ])
  })

  it('par défaut la chaîne la plus jouée, et le graphe ne trace que ses points', async () => {
    mount(TWO_CHAINS)
    const group = screen.getByRole('group', { name: 'Chaîne de rang' })
    expect(within(group).getByRole('button', { name: /Assassin/ })).toHaveAttribute('aria-pressed', 'true')
    await waitFor(() => expect(rankOption()).toBeTruthy())
    // 61 − (60 − 2) = 3 : l'écart de la chaîne Assassin, sans les 140 de l'autre chaîne.
    expect(rankOption().title[0].subtext).toBe('61 (+3)')
  })

  it('choisir une autre chaîne rebascule le graphe, écart et échelle calculés sur elle seule', async () => {
    mount(TWO_CHAINS)
    fireEvent.click(screen.getByRole('button', { name: /Objectif/ }))
    expect(screen.getByRole('button', { name: /Objectif/ })).toHaveAttribute('aria-pressed', 'true')
    await waitFor(() => expect(rankOption().title[0].subtext).toBe('140 (+5)'))
  })

  it('une seule chaîne dans la session : pas de sélecteur', () => {
    mount(ONE_CHAIN)
    expect(screen.queryByRole('group', { name: 'Chaîne de rang' })).toBeNull()
  })

  it('EN : libellés de la chaîne dans la langue de la page', () => {
    useAppShellStore.setState({ locale: 'en' })
    const labels = getSquadText('en').performanceCharts
    renderWithProviders(
      <SquadPerformanceCharts rowsByPlayer={TWO_CHAINS} playerOrder={['JGtm']} colorByPlayer={{ JGtm: '#2255cc' }} labels={labels} />,
    )
    const group = screen.getByRole('group', { name: 'Rank chain' })
    expect(within(group).getAllByRole('button').map((b) => b.textContent)).toEqual(['Social · Slayer (3)', 'Social · Objective (1)'])
  })
})
