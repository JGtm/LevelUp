/**
 * Tests — `useMatchReplay` demande la forme en colonnes et la défait avant la normalisation. Seul le
 * serveur de réseau est simulé : la fixture en colonnes est celle que produit l'encodeur Go.
 */
import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { gunzipSync } from 'node:zlib'
import { describe, expect, it } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'

import { server } from '@/test/setup'

import { goFixtureEntries, goFixturesDir, loadGoFixture } from '@/features/match-replay/test/goFixtures'

import { useMatchReplay } from './queries'
import { normalizeReplayDocument } from './replayNormalize'

const URL_REJEU = '/api/v1/players/jgtm/matches/match-1/replay'
const FILM = 'bcb6d393'

function wrapper({ children }: { children: ReactNode }) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })
  return <QueryClientProvider client={qc}>{children}</QueryClientProvider>
}

function normalFixture() {
  return loadGoFixture(goFixtureEntries().find((e) => e.film === FILM)!).doc
}

describe('useMatchReplay — forme en colonnes', () => {
  it('demande encoding=columns et rend le même document normalisé que la forme normale', async () => {
    let encodingAsked: string | null = null
    const encoded = JSON.parse(gunzipSync(readFileSync(join(goFixturesDir(), '..', 'go_columns', `columns_${FILM}.json.gz`))).toString('utf8'))
    server.use(
      http.get(URL_REJEU, ({ request }) => {
        encodingAsked = new URL(request.url).searchParams.get('encoding')
        return HttpResponse.json(encoded)
      }),
    )
    const { result } = renderHook(() => useMatchReplay('jgtm', 'match-1'), { wrapper })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(encodingAsked).toBe('columns')
    const data = result.current.data!
    expect(data).not.toHaveProperty('$enc')
    expect(data.contractIssue).toBeUndefined()
    const reference = normalizeReplayDocument(normalFixture())
    expect(data.tracks.length).toBe(reference.tracks.length)
    expect(JSON.stringify(data.tracks)).toBe(JSON.stringify(reference.tracks))
    expect(data.matchId).toBe(reference.matchId)
  })

  it('une réponse en forme normale (encodeur refusé côté serveur) se lit comme avant', async () => {
    const normal = normalFixture()
    server.use(http.get(URL_REJEU, () => HttpResponse.json(normal)))
    const { result } = renderHook(() => useMatchReplay('jgtm', 'match-1'), { wrapper })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data!.matchId).toBe(normalizeReplayDocument(normal).matchId)
  })
})
