/**
 * MatchViewPage.t0.test.tsx — LA PAGE TRANSMET LE COUP D'ENVOI À L'ONGLET CHRONOLOGIE.
 *
 * POURQUOI CE FICHIER (2026-09-06, revue R1 du lot v2 D, constat C1). La correction P0-7 vit
 * dans TROIS lignes de câblage : `t0Ms={header.t0_ms}` ici, puis `t0Ms={t0Ms}` sur les deux
 * lectures du bloc de score (couvertes par `MatchViewTabChronology.t0.test.tsx`). Retirer
 * celle-ci laissait la suite entière verte : l'onglet recevait `undefined`, l'horloge retenait
 * `t0Ms = 0`, et le bloc « Score dans le temps » repassait sur l'axe du film — 18 à 28 s
 * d'écart avec « Frags cumulés » juste au-dessus, sans un seul rouge.
 *
 * L'onglet est ici un ESPION : ce qui est éprouvé est la VALEUR transmise, pas le rendu de
 * l'onglet (testé chez lui). Le montage reprend le patron de `MatchViewTabs.test.tsx`.
 */
import { describe, expect, it, vi } from 'vitest'

import { renderWithProviders } from '@/test/render-utils'

/** Le countdown d'avant-match publié par l'API pour ce témoin. */
const T0_MS = 30_000

const hoisted = vi.hoisted(() => ({
  propsOnglet: [] as Record<string, unknown>[],
  matchView: {
    data: {
      header: {
        map_ui: 'Forest',
        mode_ui: 'Slayer',
        start_time_label: null,
        replay_available: true,
        t0_ms: 30_000,
        score_timeline_kind: undefined,
      },
      rank: null,
      summary_tab: { kpis: {}, expected_stats: null, medals: [], citations: [] },
      combat_tab: {},
      team_tab: {},
      media_tab: { media_items: [] },
      citations_tab: { native_commendations: [] },
      radar: null,
    } as unknown,
    isPending: false,
    isError: false,
    error: null as unknown,
    refetch: vi.fn(),
  },
}))

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    useParams: () => ({ playerSlug: 'test-player', matchId: 'm1' }),
    useSearch: () => ({ tab: 'chronology' }),
    useNavigate: () => vi.fn(),
    useRouter: () => ({ history: { length: 1, back: vi.fn() } }),
  }
})

vi.mock('./queries', () => ({
  useMatchView: () => hoisted.matchView,
  useMatchObjectiveEvents: () => ({ data: undefined }),
  useMatchPositions: () => ({ data: undefined }),
}))

vi.mock('@/features/settings/queries', () => ({
  useSettings: () => ({ data: { friend_gamertags: [] } }),
}))

vi.mock('@/stores/appShellStore', () => ({
  useAppShellStore: (
    selector: (s: {
      locale: 'fr' | 'en'
      availableTitles: unknown[]
      currentTitleSlug: string
    }) => unknown,
  ) => selector({ locale: 'fr', availableTitles: [], currentTitleSlug: 'halo_infinite' }),
}))

vi.mock('./MatchHeader', () => ({
  MatchBreadcrumb: () => <div />,
  MatchNavigationBar: () => <div />,
  MatchHeaderCard: () => <div />,
}))

// L'ESPION : il enregistre ce que la page lui passe, et ne rend rien.
vi.mock('./MatchViewTabChronology', () => ({
  MatchViewTabChronology: (p: Record<string, unknown>) => {
    hoisted.propsOnglet.push(p)
    return <div data-testid="onglet-espion" />
  },
}))

const { MatchViewPage } = await import('./MatchViewPage')

describe('page match — le coup d’envoi descend jusqu’à l’onglet Chronologie (P0-7)', () => {
  it('l’onglet reçoit `header.t0_ms`, pas `undefined`', () => {
    hoisted.propsOnglet.length = 0
    renderWithProviders(<MatchViewPage />)
    expect(hoisted.propsOnglet.length).toBeGreaterThan(0)
    expect(hoisted.propsOnglet.at(-1)?.t0Ms).toBe(T0_MS)
  })
})
