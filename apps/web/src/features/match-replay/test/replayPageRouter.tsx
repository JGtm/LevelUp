/**
 * replayPageRouter.tsx — LE ROUTEUR SIMULÉ des tests de la page du rejeu (`ReplayPage.*.test.tsx`).
 *
 * La page lit ses paramètres et sa recherche par `getRouteApi`, pousse le retour de l'historique par `useRouter`
 * et rend le lien « Fiche » par `Link` : ce module les remplace, sans importer la page (le test le charge dans la
 * fabrique de `vi.mock`, avant que la page ne soit lue) :
 *
 *   vi.mock('@tanstack/react-router', async (importOriginal) =>
 *     (await import('./test/replayPageRouter')).routerMock(await importOriginal()))
 *
 * `ROUTE` porte l'état que le test fait varier (recherche de l'adresse, match affiché) ; `goBack` est le retour de
 * l'historique que le bouton « retour » appelle.
 */
import type { ReactNode } from 'react'
import { vi } from 'vitest'

/** Ce que la route donne à la page : la recherche de l'adresse (`?t=&clock=`) et le match affiché. */
export const ROUTE = { search: {} as Record<string, string>, matchId: 'match-1' }

/** Le retour de l'historique (`router.history.back`). */
export const goBack = vi.fn()

/** Remet la route dans son état de départ : à appeler avant chaque test. */
export function resetRoute(): void {
  ROUTE.search = {}
  ROUTE.matchId = 'match-1'
}

/** Le module du routeur, avec la route, le routeur et le lien de la page remplacés. */
export function routerMock(actual: typeof import('@tanstack/react-router')) {
  return {
    ...actual,
    getRouteApi: () => ({
      useParams: () => ({ titleSlug: 'halo_infinite', playerSlug: 'jgtm', matchId: ROUTE.matchId }),
      useSearch: () => ROUTE.search,
    }),
    useRouter: () => ({ history: { length: 2, back: goBack }, navigate: vi.fn() }),
    Link: ({ children, ...rest }: { children?: ReactNode; 'aria-label'?: string }) => (
      <a href="#fiche" aria-label={rest['aria-label']}>
        {children}
      </a>
    ),
  }
}
