/**
 * 2g — Héritage du segment optionnel `{-$lang}` par les Links / navigate title-scoped.
 *
 * Vérité recherchée (plan §7, à vérifier sur pièces) : depuis une URL AVEC segment
 * langue (`/en/t/halo_infinite/players/x/home`), une navigation title-scoped SANS
 * `lang` dans `params` PRÉSERVE-t-elle le segment `/en` (héritage des params courants
 * par TanStack) ? On teste via un vrai routeur + `buildLocation` (pas de rendu DOM).
 */
import { describe, it, expect } from 'vitest'
import { createRouter, createMemoryHistory } from '@tanstack/react-router'

import { routeTree } from '@/routeTree.gen'
import { queryClient } from '@/app/queryClient'

function routerAt(pathname: string) {
  return createRouter({
    routeTree,
    context: { queryClient },
    history: createMemoryHistory({ initialEntries: [pathname] }),
  })
}

describe('héritage du segment lang (2g)', () => {
  it('navigate title-scoped SANS lang depuis /en/… préserve /en', async () => {
    const router = routerAt('/en/t/halo_infinite/players/x/home')
    await router.load()
    const loc = router.buildLocation({
      to: '/{-$lang}/t/$titleSlug/players/$playerSlug/home',
      params: { titleSlug: 'halo_infinite', playerSlug: 'y' },
    })
    expect(loc.pathname).toBe('/en/t/halo_infinite/players/y/home')
  })

  it('navigate title-scoped depuis une URL SANS lang reste sans lang (/t/…)', async () => {
    const router = routerAt('/t/halo_infinite/players/x/home')
    await router.load()
    const loc = router.buildLocation({
      to: '/{-$lang}/t/$titleSlug/players/$playerSlug/home',
      params: { titleSlug: 'halo_infinite', playerSlug: 'y' },
    })
    expect(loc.pathname).toBe('/t/halo_infinite/players/y/home')
  })

  it('ÉMISSION par défaut (I10) : passer `lang` dans params depuis /t/… produit /{lang}/t/…', async () => {
    // Le pendant du test ci-dessus : dès qu'un point d'entrée (index, backstop, Settings)
    // ÉMET la locale via le param `lang`, l'URL porte le segment — puis il est hérité.
    const router = routerAt('/t/halo_infinite/players/x/home')
    await router.load()
    const fr = router.buildLocation({
      to: '/{-$lang}/t/$titleSlug/players/$playerSlug/home',
      params: { lang: 'fr', titleSlug: 'halo_infinite', playerSlug: 'y' },
    })
    expect(fr.pathname).toBe('/fr/t/halo_infinite/players/y/home')

    const en = router.buildLocation({
      to: '/{-$lang}/t/$titleSlug/players/$playerSlug/home',
      params: { lang: 'en', titleSlug: 'halo_5', playerSlug: 'y' },
    })
    expect(en.pathname).toBe('/en/t/halo_5/players/y/home')
  })
})
