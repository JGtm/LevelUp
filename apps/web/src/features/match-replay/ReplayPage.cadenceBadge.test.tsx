/**
 * Tests — la PAGE du rejeu transmet le verdict de cadence du serveur au badge admin.
 *
 * `lib/replay/queries.cadence.test.tsx` s'arrête à la valeur lue par la requête (l'en-tête
 * `X-Replay-Cadence-Stale`). Ce test monte la vraie page, la vraie requête et le vrai badge : seuls
 * le routeur et le serveur de réseau sont simulés. Retirer la prop `cadenceStale={data.cadenceStale}`
 * de `ReplayPage` fait rougir le premier cas (le badge dirait « à jour » au lieu de « à régénérer »).
 *
 * Le document servi n'a aucune piste : la page est à l'état « vide », sans canvas ni fil, mais la
 * tête (fil d'Ariane et badge) se monte dès que le document est là.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { act, screen } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import type { ReactNode } from 'react'

import { goFixtureEntries, goFixtureSchemaVersion, loadGoFixture } from '@/features/match-replay/test/goFixtures'
import { renderWithProviders } from '@/test/render-utils'
import { server } from '@/test/setup'
import { useAppShellStore } from '@/stores/appShellStore'

import { ReplayPage } from './ReplayPage'

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    getRouteApi: () => ({
      useParams: () => ({ titleSlug: 'halo_infinite', playerSlug: 'jgtm', matchId: 'match-1' }),
      useSearch: () => ({}),
    }),
    useRouter: () => ({ history: { length: 1, back: vi.fn() }, navigate: vi.fn() }),
    Link: ({ children }: { children?: ReactNode }) => <a>{children}</a>,
  }
})

const URL_REJEU = '/api/v1/players/jgtm/matches/match-1/replay'
const VERSION = goFixtureSchemaVersion()

/**
 * Un VRAI document produit par Go (le plus petit du manifeste), sans piste : il passe le contrôle de contrat
 * (aucun « contrat non respecté » ne se glisse dans le badge) et la page reste à l'état « vide », sans canvas.
 */
function documentSansPiste() {
  const plusPetit = [...goFixtureEntries()].sort((a, b) => a.bytes - b.bytes)[0]
  return { ...loadGoFixture(plusPetit).doc, tracks: [] }
}

function servirLeRejeu(headers: Record<string, string>) {
  const doc = documentSansPiste()
  server.use(http.get(URL_REJEU, () => HttpResponse.json(doc, { headers })))
}

describe('ReplayPage — badge de schéma admin', () => {
  beforeEach(() => {
    act(() => useAppShellStore.setState({ locale: 'fr', isAdmin: true }))
  })
  afterEach(() => {
    act(() => useAppShellStore.setState({ isAdmin: false }))
  })

  it('pas périmé côté serveur : le badge dit « à jour »', async () => {
    servirLeRejeu({ 'X-Replay-Latest-Schema-Version': String(VERSION), 'X-Replay-Cadence-Stale': 'false' })
    renderWithProviders(<ReplayPage />)
    expect(await screen.findByText(`Schéma ${VERSION} · à jour`)).toBeInTheDocument()
  })

  it('X-Replay-Cadence-Stale = true : le badge dit « à régénérer », même schéma que le producteur', async () => {
    servirLeRejeu({ 'X-Replay-Latest-Schema-Version': String(VERSION), 'X-Replay-Cadence-Stale': 'true' })
    renderWithProviders(<ReplayPage />)
    // Le document réel porte la table des calques : le badge nomme la couche de publication.
    expect(
      await screen.findByText(`Schéma ${VERSION} · à régénérer (dernier : ${VERSION}) — seule la publication a changé`),
    ).toBeInTheDocument()
    expect(screen.queryByText(`Schéma ${VERSION} · à jour`)).not.toBeInTheDocument()
  })

  it("un joueur ordinaire ne voit aucun badge, même avec un pas périmé", async () => {
    act(() => useAppShellStore.setState({ isAdmin: false }))
    servirLeRejeu({ 'X-Replay-Latest-Schema-Version': String(VERSION), 'X-Replay-Cadence-Stale': 'true' })
    renderWithProviders(<ReplayPage />)
    await screen.findByText('Aucun rejeu 2D disponible pour ce match.')
    expect(screen.queryByText(/Schéma/)).not.toBeInTheDocument()
  })
})
