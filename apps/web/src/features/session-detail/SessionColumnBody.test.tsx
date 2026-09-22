/**
 * SessionColumnBody.test.tsx — LES QUATRE TITRES DE SECTION de la page session, et la
 * seule chose qui doit les faire bouger : la donnée.
 *
 * CE QUE CE FICHIER FIXE, et pourquoi chaque point a sa raison d'être :
 *   1. QUATRE titres, dans l'ORDRE — « Bilan », « Match par match », « Frags et usages »,
 *      « Détail des matchs ». Ni cinq, ni un onglet.
 *   2. UN TITRE NE SE POSE JAMAIS AU-DESSUS DE RIEN : sans frags ni usages, la section 3
 *      disparaît ENTIÈREMENT, titre compris. C'est le seul groupe dont tous les blocs
 *      peuvent manquer à la fois.
 *   3. L'état « aucun film » COMPTE POUR UN BLOC : il porte une phrase et un dénominateur,
 *      la section reste donc titrée au-dessus de lui.
 *   4. IDENTITÉ compact / non compact — le drawer de comparaison monte le MÊME composant ;
 *      des titres différents d'un côté décaleraient les deux colonnes d'une ligne, ce qui
 *      est exactement ce que la comparaison côte à côte doit empêcher.
 */
import { describe, expect, it, vi } from 'vitest'
import { screen, within } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'

import type { SessionCompareEntry, SessionUsageBlock } from '@/lib/api/types'

import { USAGE_TEXT } from '@/features/_shared/usage/usageI18n'

import { SessionColumnBody } from './SessionColumnBody'

// ECharts (canvas) ne peint pas dans jsdom : ce fichier ne lit que des titres.
vi.mock('echarts-for-react', () => ({ default: () => null }))

// Le tableau des matchs monte des liens de route ; on n'en teste rien ici.
vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    useParams: () => ({ playerSlug: 'moi' }),
    useSearch: () => ({}),
    useNavigate: () => vi.fn(),
  }
})

const SECTIONS = ['Bilan', 'Match par match', 'Frags et usages', 'Détail des matchs'] as const

/** Session dont la répartition des frags est servie → la carte des frags se dessine. */
const ENTRY_AVEC_FRAGS = {
  session_label: '2026-04-21 19h30',
  start_time: '2026-04-21T19:30:00Z',
  end_time: '2026-04-21T20:05:00Z',
  total_matches: 2,
  wins: 2,
  losses: 0,
  kda: 2.4,
  kdr: 1.8,
  kills_per_match: 13,
  win_rate: 100,
  performance_score: 68.5,
  with_friends: false,
  dominant_category: 'Ranked',
  matches: null,
  match_series: null,
  participation: null,
  frag_distribution: {
    total_kills: 26,
    classes: [{ class: 'shoulder', kills: 26, roles: [] }],
  },
  top_weapon_kills: [{ label: 'MA40', kills: 12, class: 'shoulder' }],
} as unknown as SessionCompareEntry

/** Même session, sans aucune donnée de frags : la carte des frags rend `null`. */
const ENTRY_SANS_FRAGS = {
  ...ENTRY_AVEC_FRAGS,
  frag_distribution: undefined,
  top_weapon_kills: [],
} as unknown as SessionCompareEntry

/** Bloc d'usage mesuré : la carte « Contrôle des armes spéciales » se dessine. */
const USAGE_MESURE: SessionUsageBlock = {
  available: true,
  matches_measured: 4,
  matches_total: 6,
  team_parity_pct: 25,
  metrics: [
    {
      key: 'pad_pickups',
      player_total: 9,
      team_total: 20,
      lobby_total: 43,
      matches_above_lobby_parity: 1,
      player_share_of_team_pct: 45,
      per_match: [{ match_id: 'm1', player_share_of_team_pct: 45 }],
    },
  ],
}

function titresRendus(): string[] {
  return screen
    .getAllByRole('heading', { level: 3 })
    .map((h) => h.textContent ?? '')
    .filter((label) => (SECTIONS as readonly string[]).includes(label))
}

function monter(props: Partial<React.ComponentProps<typeof SessionColumnBody>> = {}) {
  return renderWithProviders(
    <SessionColumnBody
      entry={ENTRY_AVEC_FRAGS}
      matches={[]}
      playerSlug="moi"
      compact={false}
      usage={USAGE_MESURE}
      {...props}
    />,
  )
}

describe('SessionColumnBody — les quatre titres de section', () => {
  it('donnée complète : les quatre titres, dans l’ordre', () => {
    monter()
    expect(titresRendus()).toEqual([...SECTIONS])
  })

  it('la bande KPI reste sans titre (elle n’est pas une 5e section)', () => {
    monter()
    // Exactement quatre titres de section reconnus, pas un de plus.
    expect(titresRendus()).toHaveLength(4)
  })

  it('la section 3 coiffe BIEN les cartes d’usage (et pas une section voisine)', () => {
    monter()
    const section = screen.getByText('Frags et usages').closest('section')
    expect(section).not.toBeNull()
    expect(
      within(section as HTMLElement).getByText(USAGE_TEXT.fr.blockPadControl),
    ).toBeInTheDocument()
  })
})

describe('SessionColumnBody — un titre ne se pose pas au-dessus de rien', () => {
  it('ni frags ni usages : la section « Frags et usages » disparaît ENTIÈREMENT', () => {
    monter({ entry: ENTRY_SANS_FRAGS, usage: undefined })
    expect(titresRendus()).toEqual(['Bilan', 'Match par match', 'Détail des matchs'])
    expect(screen.queryByText('Frags et usages')).not.toBeInTheDocument()
  })

  it('titre sans décodeur de film (unsupported) et sans frags : toujours pas de section 3', () => {
    monter({
      entry: ENTRY_SANS_FRAGS,
      usage: { available: false, unavailable_reason: 'unsupported', matches_measured: 0, matches_total: 6 },
    })
    expect(screen.queryByText('Frags et usages')).not.toBeInTheDocument()
  })

  it('frags seuls (aucun bloc d’usage) : la section 3 existe quand même', () => {
    monter({ usage: undefined })
    expect(titresRendus()).toEqual([...SECTIONS])
  })

  it('usages seuls, état « aucun film » : c’est un bloc visible, la section 3 le coiffe', () => {
    monter({
      entry: ENTRY_SANS_FRAGS,
      usage: { available: true, matches_measured: 0, matches_total: 6 },
    })
    expect(titresRendus()).toEqual([...SECTIONS])
    // Le vocabulaire des états vides a été refondu le 2026-09-21 (D8, `usageEmptyMessage`) :
    // la cause est NOMMÉE (« aucun film décodé ») au lieu du « aucune donnée » d'avant.
    expect(screen.getByText(USAGE_TEXT.fr.emptyNoFilm)).toBeInTheDocument()
  })
})

describe('SessionColumnBody — le mode comparer monte les mêmes sections', () => {
  it('compact et non compact : mêmes titres, même ordre', () => {
    const plein = monter()
    const titresPlein = titresRendus()
    plein.unmount()

    monter({ compact: true, participationSide: 'left' })
    expect(titresRendus()).toEqual(titresPlein)
    expect(titresRendus()).toEqual([...SECTIONS])
  })

  it('compact : la section 3 disparaît aux mêmes conditions que la colonne pleine', () => {
    monter({ compact: true, entry: ENTRY_SANS_FRAGS, usage: undefined })
    expect(titresRendus()).toEqual(['Bilan', 'Match par match', 'Détail des matchs'])
  })
})
