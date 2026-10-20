/**
 * Test de non-régression — toucher un filtre de la barre Escouade ne re-rend PAS
 * le contenu de la page.
 *
 * Symptôme corrigé (utilisateur, 2026-09-20) : « dès que je touche à un filtre
 * dans la barre, c'est comme si la page rechargeait mais sans changer ce qui
 * s'affiche ; quand je clique sur Analyser ça re-recharge et là ça affiche les
 * bonnes stats ». Cause : l'état en attente de la barre vivait dans SquadLayout,
 * PARENT de `<Outlet />` et du `SquadContext.Provider` — chaque case cochée
 * re-rendait l'arbre entier et faisait rejouer l'animation de tous les graphes.
 *
 * Le test reproduit le mécanisme réel : `<Outlet />` est mémoïsé côté routeur
 * (React.memo sans props), donc un rendu du layout ne le traverse PAS ; seule
 * une nouvelle IDENTITÉ de la valeur de `SquadContext` re-rend le contenu. La
 * sonde compte ses rendus et sert d'oracle des deux côtés :
 *  - cocher une playlist, ouvrir/fermer un popover et choisir une période EN
 *    ATTENTE ne doivent produire AUCUN rendu du contenu ;
 *  - « Analyser » doit commiter dans le store (le hash change) — c'est le seul
 *    moment où la page a le droit de se recharger.
 */
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { act, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { http, HttpResponse } from 'msw'
import type { ReactNode } from 'react'
import { renderWithProviders } from '@/test/render-utils'
import { server } from '@/test/setup'
import { useSquadFilterStore } from '@/stores/squadFilterStore'
import { SquadLayout } from './SquadLayout'

// Compteur partagé entre la sonde (créée dans la factory de mock, hoistée) et
// les assertions. `vi.hoisted` est le seul moyen d'y accéder des deux côtés.
const sonde = vi.hoisted(() => ({ rendus: 0 }))

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  const { memo } = await import('react')
  const { useSquadContext } = await import('./SquadContext')
  // Consommateur de SquadContext : exactement ce que sont SquadSynergiesPage,
  // SquadContributionsPage et SquadDynamiquePage.
  function SondeDeContexte() {
    const ctx = useSquadContext()
    sonde.rendus += 1
    return <div data-testid="contenu-onglet">{ctx.confirmedGamertags.length}</div>
  }
  // `memo` sans props : reproduit l'Outlet du routeur, qui ne re-rend pas sur un
  // simple rendu de son parent.
  const OutletMemo = memo(function Outlet() {
    return <SondeDeContexte />
  })
  return {
    ...actual,
    useParams: () => ({ playerSlug: 'p' }),
    useSearch: () => ({}),
    useMatchRoute: () => () => null,
    useNavigate: () => vi.fn(),
    Outlet: OutletMemo,
    Link: ({ children }: { children?: ReactNode }) => <a>{children}</a>,
  }
})

/** Réponse teammates minimale mais NON vide : le contenu (donc l'Outlet) est monté. */
const teammatesReponse = {
  options: [],
  teammates: [],
  total_matches: 3,
  session_labels: { solo: [], squad: [] },
  friends_count: 0,
  composition_sessions: [],
  latest_composition_session: '',
  match_history: [],
}

/** /filters/resolve : une playlist cochable (count > 0, sinon repliée) + les presets. */
const resolveReponse = {
  effective: {
    filter_mode: 'period',
    period: { start_date: null, end_date: null },
    sessions: { picked_sessions: [], gap_minutes: 120 },
    cascade: { experience_types: [], playlists: [], modes: [], maps: [] },
  },
  available_options: {
    experience_types: [],
    playlists: [{ label: 'Arene classee', value: 'ranked-arena', count: 3 }],
    modes: [],
    maps: [],
  },
  // Ni la liste de sessions de repli ni le bloc `counts` ne sont posés ici : la
  // page n'en a pas besoin pour ce scénario, et le garde-rail « source unique du
  // compte de session » (singleCountSource.guard.test.ts) interdit leurs
  // littéraux partout dans features/squad — y compris dans un fixture de test.
  period_presets: [
    { preset_id: '7d', count: 1 },
    { preset_id: '30d', count: 3 },
    { preset_id: '90d', count: 3 },
    { preset_id: 'all', count: 3 },
  ],
}

beforeEach(() => {
  localStorage.clear()
  useSquadFilterStore.getState().resetFilters()
  sonde.rendus = 0
  server.use(
    http.post('/api/v1/players/:playerSlug/pages/teammates', () =>
      HttpResponse.json(teammatesReponse),
    ),
    http.post('/api/v1/players/:playerSlug/filters/resolve', () =>
      HttpResponse.json(resolveReponse),
    ),
  )
})

/** Laisse retomber les requêtes en vol (MSW répond en quelques ms, en process). */
async function laisserRetomberLesRequetes() {
  await act(async () => {
    await new Promise((resolve) => setTimeout(resolve, 150))
  })
}

/** Monte la page et rend le nombre de rendus du contenu une fois tout posé. */
async function monterEtStabiliser(): Promise<number> {
  renderWithProviders(<SquadLayout />)
  await screen.findByTestId('contenu-onglet')
  await laisserRetomberLesRequetes()
  return sonde.rendus
}

describe('SquadLayout — la barre de filtres ne re-rend pas le contenu', () => {
  it('cocher une playlist, ouvrir/fermer un popover et choisir une période en attente ne re-rendent pas le consommateur de SquadContext', async () => {
    const user = userEvent.setup()
    const rendusInitiaux = await monterEtStabiliser()

    // 1. Ouvrir le popover Filtres et cocher une playlist.
    await user.click(screen.getByRole('button', { name: /Filtres/ }))
    const casePlaylist = await screen.findByLabelText(/Arene classee/)
    await user.click(casePlaylist)
    expect((screen.getByLabelText(/Arene classee/) as HTMLInputElement).checked).toBe(true)

    // 2. Fermer le popover Filtres, ouvrir celui de la Période.
    await user.click(screen.getByRole('button', { name: /Filtres/ }))
    await user.click(screen.getByRole('button', { name: /Toutes les périodes/ }))

    // 3. Choisir un preset de période EN ATTENTE (pas encore commité).
    await user.click(await screen.findByRole('button', { name: /30 jours/ }))

    // Le preview repart sur le nouveau pending : on le laisse répondre.
    await laisserRetomberLesRequetes()

    // Rien n'a été commité : le store n'a pas bougé.
    expect(useSquadFilterStore.getState().filterContext.cascade?.playlists ?? []).toEqual([])
    expect(useSquadFilterStore.getState().filterContext.period?.start_date ?? null).toBeNull()

    // L'oracle : le contenu de la page n'a pas été re-rendu une seule fois.
    expect(sonde.rendus).toBe(rendusInitiaux)
  })

  it('« Analyser » commit les filtres en attente dans le store (le hash change)', async () => {
    const user = userEvent.setup()
    await monterEtStabiliser()
    const hashAvant = useSquadFilterStore.getState().filterContextHash

    await user.click(screen.getByRole('button', { name: /Filtres/ }))
    await user.click(await screen.findByLabelText(/Arene classee/))
    await user.click(screen.getByRole('button', { name: /Filtres/ }))
    await user.click(screen.getByRole('button', { name: 'Analyser' }))

    expect(useSquadFilterStore.getState().filterContext.cascade?.playlists).toEqual([
      'ranked-arena',
    ])
    expect(useSquadFilterStore.getState().filterContextHash).not.toBe(hashAvant)
  })
})
