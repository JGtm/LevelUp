import { QueryClient } from '@tanstack/react-query'
import { describe, expect, it } from 'vitest'

import { SANS_BARRE_META, compteDansLaBarre } from './sansBarre'

function enAttente() {
  let fin!: (v: string) => void
  const promesse = new Promise<string>((r) => {
    fin = r
  })
  return { promesse, fin }
}

describe('compteDansLaBarre (vrai QueryClient)', () => {
  it('compte une requête pending ordinaire', async () => {
    const client = new QueryClient()
    const { promesse, fin } = enAttente()
    const p = client.fetchQuery({ queryKey: ['a'], queryFn: () => promesse })
    expect(client.isFetching({ predicate: compteDansLaBarre })).toBe(1)
    fin('ok')
    await p
    expect(client.isFetching({ predicate: compteDansLaBarre })).toBe(0)
  })

  it('ignore une requête marquée sans barre', async () => {
    const client = new QueryClient()
    const { promesse, fin } = enAttente()
    const p = client.fetchQuery({ queryKey: ['b'], queryFn: () => promesse, meta: SANS_BARRE_META })
    expect(client.isFetching()).toBe(1)
    expect(client.isFetching({ predicate: compteDansLaBarre })).toBe(0)
    fin('ok')
    await p
  })

  it('ne compte pas un refetch (données déjà en cache)', async () => {
    const client = new QueryClient()
    client.setQueryData(['c'], 'v')
    const { promesse, fin } = enAttente()
    const p = client.fetchQuery({ queryKey: ['c'], queryFn: () => promesse, staleTime: 0 })
    expect(client.isFetching({ predicate: compteDansLaBarre })).toBe(0)
    fin('ok')
    await p
  })
})
