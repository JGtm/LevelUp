/**
 * La route `squad/usages` redirige vers `squad/emprise` (D1 du plan
 * PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26 : l'onglet Usages s'appelle « Emprise »). Un lien
 * existant garde ses paramètres de chemin (langue, titre, joueur) ET sa recherche (`session`,
 * `teammates`, lues par le layout Escouade) ; l'historique n'empile pas l'ancienne adresse.
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
import { Route } from './usages'

type BeforeLoad = (ctx: { params: Record<string, string>; search: Record<string, string> }) => void
const beforeLoad = (Route as unknown as { options: { beforeLoad: BeforeLoad } }).options.beforeLoad

function redirectOf(ctx: Parameters<BeforeLoad>[0]) {
  try {
    beforeLoad(ctx)
  } catch (e) {
    if (isRedirect(e)) return e.options
    throw e
  }
  throw new Error('squad/usages devait rediriger')
}

describe('route squad/usages → squad/emprise', () => {
  it('redirige vers l’onglet Emprise, mêmes paramètres, recherche transmise, sans empiler', () => {
    const params = { lang: 'en', titleSlug: 'halo_infinite', playerSlug: 'jgtm' }
    const search = { session: '2026-09-22', teammates: 'Chocoboflor,Madina97294' }
    const opts = redirectOf({ params, search })
    expect(opts.to).toBe('/{-$lang}/t/$titleSlug/players/$playerSlug/squad/emprise')
    expect(opts.params).toEqual(params)
    expect(opts.search).toEqual(search)
    expect(opts.replace).toBe(true)
  })

  it('sans recherche : la redirection n’en invente pas', () => {
    const opts = redirectOf({ params: { titleSlug: 'halo_5', playerSlug: 'p' }, search: {} })
    expect(opts.params).toEqual({ titleSlug: 'halo_5', playerSlug: 'p' })
    expect(opts.search).toEqual({})
  })
})
