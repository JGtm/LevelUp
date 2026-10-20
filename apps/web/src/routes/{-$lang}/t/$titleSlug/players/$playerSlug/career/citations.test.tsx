/**
 * Tests — la route career/citations monte la page des citations CALCULÉES seulement pour un
 * titre qui déclare la capability `citations`. Sans elle (Halo 5 : commendations natives),
 * l'URL tapée à la main affiche « indisponible pour ce titre » et ne monte pas la page, donc
 * n'émet aucune requête POST /pages/citations.
 */
import type { ComponentType } from 'react'
import { beforeAll, beforeEach, describe, expect, it, vi } from 'vitest'
import { screen } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'

const pageMontee = vi.fn()

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return { ...actual, createFileRoute: () => (opts: Record<string, unknown>) => opts }
})
vi.mock('@/features/citations/UnifiedCitationsPage', () => ({
  UnifiedCitationsPage: (props: { source: string }) => {
    pageMontee(props.source)
    return <div data-testid="page-citations" />
  },
}))

// Import APRÈS les mocks (le module lit createFileRoute au chargement).
import { Route } from './citations'

type Titles = ReturnType<typeof useAppShellStore.getState>['availableTitles']

function setTitle(slug: string, capabilities: string[]) {
  useAppShellStore.setState({
    locale: 'fr',
    currentTitleSlug: slug,
    availableTitles: [
      { slug, name: slug, status: 'active', capabilities, is_default: false, effective_hp_to_kill: 225 },
    ] as unknown as Titles,
  })
}

// autoCodeSplitting : le composant de route est un wrapper lazy — préchargé une fois.
const Page = (Route as unknown as { component: ComponentType & { preload?: () => Promise<unknown> } })
  .component

beforeAll(async () => {
  await Page.preload?.()
})

describe('route career/citations', () => {
  beforeEach(() => pageMontee.mockReset())

  it('monte la page des citations calculées pour un titre qui déclare citations', async () => {
    setTitle('halo_infinite', ['career', 'citations'])
    renderWithProviders(<Page />)

    expect(await screen.findByTestId('page-citations')).toBeInTheDocument()
    expect(pageMontee).toHaveBeenCalledWith('infinite')
  })

  it('affiche « indisponible » sans monter la page pour un titre sans citations (halo_5)', async () => {
    setTitle('halo_5', ['career'])
    renderWithProviders(<Page />)

    expect(await screen.findByText('Indisponible pour ce titre')).toBeInTheDocument()
    expect(screen.getByText('Ce titre ne fournit pas les citations calculées.')).toBeInTheDocument()
    expect(screen.queryByTestId('page-citations')).not.toBeInTheDocument()
    expect(pageMontee).not.toHaveBeenCalled()
  })

  it('reste fermée sans la capability career, même avec citations', async () => {
    setTitle('halo_infinite', ['citations'])
    renderWithProviders(<Page />)

    expect(await screen.findByText('Indisponible pour ce titre')).toBeInTheDocument()
    expect(screen.queryByTestId('page-citations')).not.toBeInTheDocument()
  })
})
