/**
 * Tests — `useMatchReplay` : le chemin réel des deux métas HTTP du rejeu. Le serveur publie
 * `X-Replay-Latest-Schema-Version` et `X-Replay-Cadence-Stale` ; le vrai client (`api.getWithHeaders`)
 * les lit sur une vraie réponse, et la requête les recopie dans `latestSchemaVersion` / `cadenceStale`,
 * que `ReplayPage` transmet au badge admin. Seul le serveur de réseau est simulé.
 */
import { describe, expect, it } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'

import { server } from '@/test/setup'

import { goFixtureSchemaVersion } from '@/features/match-replay/test/goFixtures'

import { useMatchReplay } from './queries'

const URL_REJEU = '/api/v1/players/jgtm/matches/match-1/replay'

function wrapper({ children }: { children: ReactNode }) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })
  return <QueryClientProvider client={qc}>{children}</QueryClientProvider>
}

/** Sert un document minimal avec les en-têtes donnés, puis lit ce que la requête en tire. */
async function lire(headers: Record<string, string>) {
  server.use(http.get(URL_REJEU, () => HttpResponse.json({ schemaVersion: goFixtureSchemaVersion(), tracks: [] }, { headers })))
  const { result } = renderHook(() => useMatchReplay('jgtm', 'match-1'), { wrapper })
  await waitFor(() => expect(result.current.isSuccess).toBe(true))
  return result.current.data!
}

describe('useMatchReplay — en-têtes de méta', () => {
  it('« true » dans X-Replay-Cadence-Stale : le pas est périmé, et la version courante est lue', async () => {
    const data = await lire({ 'X-Replay-Cadence-Stale': 'true', 'X-Replay-Latest-Schema-Version': '9' })
    expect(data.cadenceStale).toBe(true)
    expect(data.latestSchemaVersion).toBe(9)
  })

  it('« false » : le pas n’est pas périmé', async () => {
    const data = await lire({ 'X-Replay-Cadence-Stale': 'false', 'X-Replay-Latest-Schema-Version': '9' })
    expect(data.cadenceStale).toBeUndefined()
    expect(data.latestSchemaVersion).toBe(9)
  })

  it('en-têtes absents : ni pas périmé ni version courante', async () => {
    const data = await lire({})
    expect(data.cadenceStale).toBeUndefined()
    expect(data.latestSchemaVersion).toBeUndefined()
  })
})
