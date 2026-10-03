/**
 * Tests — AssetDrawer : aucune requête tiroir fermé (lot perf L4a, D4.5, 2026-09-23).
 *
 * Le tiroir est monté par l'AppShell sur TOUTES les pages : ses trois catalogues
 * (cartes, armes, médailles) partaient à chaque chargement de l'application, tiroir
 * fermé. Ouvert, le comportement ne change pas : les trois onglets se chargent.
 */
import { describe, it, expect, beforeEach } from 'vitest'
import { act, waitFor } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import { renderWithProviders } from '@/test/render-utils'
import { server } from '@/test/setup'
import { useAppShellStore } from '@/stores/appShellStore'
import { useAssetDrawerStore } from './assetDrawer.store'
import { AssetDrawer } from './AssetDrawer'

const catalogues: string[] = []

beforeEach(() => {
  catalogues.length = 0
  useAppShellStore.setState({ locale: 'fr', currentTitleSlug: 'halo_infinite' })
  useAssetDrawerStore.setState({ isOpen: false, activeTab: 'maps', search: '' })
  server.use(
    http.get('/api/v1/assets/:titleSlug/:kind', ({ params }) => {
      catalogues.push(String(params.kind))
      return HttpResponse.json([])
    }),
  )
})

async function laisserRetomber() {
  await act(async () => {
    await new Promise((resolve) => setTimeout(resolve, 100))
  })
}

describe('AssetDrawer — catalogues chargés seulement tiroir ouvert (D4.5)', () => {
  it('fermé : aucune requête ; ouvert : les trois catalogues, comme avant', async () => {
    renderWithProviders(<AssetDrawer />)
    await laisserRetomber()
    expect(catalogues).toEqual([])

    act(() => useAssetDrawerStore.getState().open())
    await waitFor(() => expect([...catalogues].sort()).toEqual(['maps', 'medals', 'weapons']))
  })

  it('ouvert au montage (état persisté) : les trois catalogues partent aussitôt', async () => {
    useAssetDrawerStore.setState({ isOpen: true })
    renderWithProviders(<AssetDrawer />)
    await waitFor(() => expect([...catalogues].sort()).toEqual(['maps', 'medals', 'weapons']))
  })
})
