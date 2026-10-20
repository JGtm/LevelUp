/**
 * Tests de l'identité de la page Tendances : un autre joueur ou un autre titre repart de zéro,
 * et le filtre « Type de partie » affiche toujours le type appliqué, même absent de la réponse.
 *
 * ECharts est simulé (jsdom n'a pas de canvas). Le joueur de la route est modifiable.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, screen, waitFor, within } from '@testing-library/react'

import { useAppShellStore } from '@/stores/appShellStore'
import { renderWithProviders } from '@/test/render-utils'

import { response } from './tendances.fixture'
import { squadResponse } from './tendancesSquad.fixture'
import { TendancesTab } from './TendancesTab'

const route = vi.hoisted(() => ({ playerSlug: 'jgtm' }))

vi.mock('echarts-for-react', () => ({
  default: ({ option }: { option: unknown }) => (
    <div data-testid="echarts-stub">{JSON.stringify(option)}</div>
  ),
}))

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return { ...actual, useParams: () => ({ playerSlug: route.playerSlug }) }
})

const post = vi.fn()
const get = vi.fn()
vi.mock('@/lib/api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/api/client')>()
  return {
    ...actual,
    api: {
      ...actual.api,
      post: (...args: unknown[]) => post(...args),
      get: (...args: unknown[]) => get(...args),
    },
  }
})

beforeEach(() => {
  localStorage.clear()
  route.playerSlug = 'jgtm'
  post.mockReset()
  get.mockReset()
  get.mockImplementation(async (path: string) => {
    if (path.includes('career/encounters')) {
      return { teammates: [{ gamertag: 'Alice', xuid: 'x-alice', match_count: 20 }] }
    }
    if (path.includes('squads')) return { squads: [] }
    return { title_slug: 'halo_infinite', schema_version: 1, locale: 'fr', fields: {} }
  })
  post.mockImplementation(async (_path: string, body: { view?: string }) =>
    body.view === 'squad' ? squadResponse({ game_types: [] }) : response(),
  )
  useAppShellStore.setState({
    locale: 'fr',
    currentTitleSlug: 'halo_infinite',
    isBootstrapped: true,
  })
})
afterEach(() => useAppShellStore.setState({ locale: 'fr', currentTitleSlug: 'halo_infinite' }))

const select = () => screen.getByLabelText('Type de partie') as HTMLSelectElement

describe('TendancesTab — identité', () => {
  it('un autre joueur remet le type de partie à « Toutes les parties »', async () => {
    const { rerender } = renderWithProviders(<TendancesTab />)
    await screen.findByTestId('tendances-matrix')
    fireEvent.change(select(), { target: { value: 'ranked_slayer' } })
    await waitFor(() =>
      expect(post.mock.calls.at(-1)?.[1]).toMatchObject({ game_type: 'ranked_slayer' }),
    )

    route.playerSlug = 'autre'
    rerender(<TendancesTab />)
    await waitFor(() => expect(select().value).toBe(''))
    await waitFor(() => {
      const [chemin, corps] = post.mock.calls.at(-1) ?? []
      expect(chemin).toBe('/players/autre/pages/trends')
      expect(corps).toMatchObject({ game_type: '' })
    })
  })

  it('un autre titre remet le type de partie à zéro', async () => {
    renderWithProviders(<TendancesTab />)
    await screen.findByTestId('tendances-matrix')
    fireEvent.change(select(), { target: { value: 'ranked_slayer' } })
    await waitFor(() =>
      expect(post.mock.calls.at(-1)?.[1]).toMatchObject({ game_type: 'ranked_slayer' }),
    )

    useAppShellStore.setState({ currentTitleSlug: 'halo_5' })
    await waitFor(() => expect(select().value).toBe(''))
  })
})

describe('TendancesTab — le filtre affiche toujours le type appliqué', () => {
  it('après bascule vers une réponse sans ce type, la valeur et le libellé restent', async () => {
    localStorage.setItem('squad-teammates-jgtm', JSON.stringify(['Alice']))
    renderWithProviders(<TendancesTab />)
    await screen.findByTestId('tendances-matrix')
    fireEvent.change(select(), { target: { value: 'ranked_slayer' } })
    await waitFor(() =>
      expect(post.mock.calls.at(-1)?.[1]).toMatchObject({ game_type: 'ranked_slayer' }),
    )

    fireEvent.click(
      within(screen.getByRole('group', { name: 'Vue' })).getByRole('button', { name: 'Escouade' }),
    )
    await waitFor(() => expect(post.mock.calls.at(-1)?.[1]).toMatchObject({ view: 'squad' }))
    await screen.findByTestId('tendances-matrix')
    expect(select().value).toBe('ranked_slayer')
    expect(select().selectedOptions[0].textContent).toBe('Classé · Assassin')
  })
})
