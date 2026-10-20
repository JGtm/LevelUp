/**
 * La route `ascension/tendances` redirige vers `stats/tendances` : la page Tendances vit sous la
 * section Solo. Un lien existant garde ses paramètres de chemin (langue, titre, joueur) ;
 * l'historique n'empile pas l'ancienne adresse.
 */
import { describe, expect, it, vi } from 'vitest'
import { isRedirect } from '@tanstack/react-router'

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    createFileRoute: () => (opts: Record<string, unknown>) => ({ options: opts }),
  }
})

// Import APRÈS le mock : la route lit `createFileRoute` au chargement.
import { Route } from './tendances'

type BeforeLoad = (ctx: { params: Record<string, string> }) => void
const beforeLoad = (Route as unknown as { options: { beforeLoad: BeforeLoad } }).options.beforeLoad

describe('route ascension/tendances → stats/tendances', () => {
  it('redirige vers la page Solo, mêmes paramètres, sans empiler', () => {
    const params = { lang: 'en', titleSlug: 'halo_infinite', playerSlug: 'jgtm' }
    let opts: Record<string, unknown> | undefined
    try {
      beforeLoad({ params })
    } catch (e) {
      if (!isRedirect(e)) throw e
      opts = e.options as Record<string, unknown>
    }
    expect(opts?.to).toBe('/{-$lang}/t/$titleSlug/players/$playerSlug/stats/tendances')
    expect(opts?.params).toEqual(params)
    expect(opts?.replace).toBe(true)
  })
})
