/**
 * D16 — rangees partagees en vue comparaison (une rangee par CARTE depuis le plan
 * PLAN_SESSIONS_EMPRISE_2026-10-06, D11).
 *
 * Verifie l'INVARIANT de structure, pas le pixel : les deux colonnes emettent la MEME
 * liste ordonnee de cles de section (meme index = meme rangee de la grille racine),
 * un cote qui n'a pas la carte rend le placeholder « Sans equivalent dans cette
 * session », les titres de groupe et les intertitres de sous-groupe se posent dans la meme
 * rangee des deux cotes, et les cartes sont en vue compacte DES DEUX cotes. En pleine page
 * (drawer ferme) : pile simple, aucune rangee, aucun placeholder.
 *
 * Temoin (MESURES §0) : la soiree d'escouade du 22/09 (deux CTF) face a la session solo du meme
 * soir (Super Fiesta, aucun match a objectif).
 */
import { describe, expect, it, vi } from 'vitest'
import { fireEvent, screen, waitFor } from '@testing-library/react'
import { http, HttpResponse } from 'msw'

import { renderWithProviders } from '@/test/render-utils'
import { server } from '@/test/setup'

import { SessionDetailPage } from './SessionDetailPage'
import { session2209, sessionSolo } from './sessionEmprise.fixtures'

vi.mock('echarts-for-react', () => ({ default: () => null }))

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    useParams: () => ({ playerSlug: 'test-player' }),
    useSearch: () => ({}),
    useNavigate: () => vi.fn(),
  }
})

const KPI = {
  start_time: '2026-09-22T19:23:00Z',
  end_time: '2026-09-22T20:40:00Z',
  kda: 2.4,
  performance_score: 68.5,
  win_rate: 43,
  kdr: 1.8,
  kills_per_match: 9,
  dominant_category: 'Ranked',
}

const CURRENT = session2209()
const COMPARE = sessionSolo()
const CURRENT_LABEL = '2026-09-22 21h23'
const COMPARE_LABEL = '2026-09-22 19h27'

function baseResponse() {
  return {
    current_session: { ...KPI, ...CURRENT.entry, session_label: CURRENT_LABEL },
    available_sessions: [CURRENT_LABEL, COMPARE_LABEL],
    matches: CURRENT.matches,
    emprise: CURRENT.emprise,
    lives_near_teammate: CURRENT.lives,
    formes_retenues: CURRENT.formes,
    suggested_compare: { session_label: COMPARE_LABEL, strategy: 's', reason: 'r' },
    compare_enabled: false,
    compare_session: null,
    compare_metrics: [],
  }
}

/** Bloc « Coordination » minimal et MESURE : sa couverture et son taux identifient la colonne. */
function coordinationBlock(measured: number) {
  const couverture = (taux: number) => ({ taux, brut: 10, n: 20, par_match: 5, echantillon_faible: false })
  return {
    available: true,
    matches_measured: measured,
    matches_total: 7,
    appui: { on_me_prepare: couverture(measured / 10), ma_part_des_appuis: couverture(0.3), parity_pct: 25 },
    per_match: [],
  }
}

function mockDetail(withCoordination = false) {
  server.use(
    http.post('/api/v1/players/:playerSlug/pages/sessions/detail', async ({ request }) => {
      const body = (await request.json()) as { enable_compare?: boolean }
      const payload: Record<string, unknown> = baseResponse()
      if (withCoordination) payload.coordination = coordinationBlock(6)
      if (body.enable_compare) {
        payload.compare_enabled = true
        payload.compare_session = { ...KPI, ...COMPARE.entry, session_label: COMPARE_LABEL }
        payload.compare_matches = COMPARE.matches
        payload.compare_emprise = COMPARE.emprise
        payload.compare_lives_near_teammate = COMPARE.lives
        payload.compare_formes_retenues = COMPARE.formes
        if (withCoordination) payload.compare_coordination = coordinationBlock(5)
      }
      return HttpResponse.json(payload)
    }),
  )
}

async function openCompare() {
  await waitFor(() => {
    expect(screen.getByRole('button', { name: /Comparer/i })).toBeInTheDocument()
  })
  fireEvent.click(screen.getByRole('button', { name: /Comparer/i }))
  await waitFor(() => {
    expect(screen.getByRole('heading', { name: 'Comparaison' })).toBeInTheDocument()
  })
}

/** Cles de section dans l'ordre du DOM — colonne principale d'abord, puis drawer. */
function sectionKeys(container: HTMLElement): string[] {
  return Array.from(container.querySelectorAll('[data-session-section]')).map(
    (node) => node.getAttribute('data-session-section') ?? '',
  )
}

/** Les deux cellules (gauche, droite) d'une rangee partagee. */
function cells(container: HTMLElement, key: string): Element[] {
  return Array.from(container.querySelectorAll(`[data-session-section="${key}"]`))
}

const placeholder = (cell: Element) => cell.querySelector('[data-testid="session-section-placeholder"]')

async function openedCompare(withCoordination = false) {
  mockDetail(withCoordination)
  const view = renderWithProviders(<SessionDetailPage />)
  await openCompare()
  await waitFor(() => {
    expect(cells(view.container, 'objective_balance')).toHaveLength(2)
  })
  return view
}

describe('SessionDetailPage — rangees partagees (D16), une par carte', () => {
  it('emet la meme liste ordonnee de sections dans les deux colonnes, une rangee par carte', async () => {
    const { container } = await openedCompare()
    const keys = sectionKeys(container)
    expect(keys.length % 2).toBe(0)
    const half = keys.length / 2
    const left = keys.slice(0, half)
    // Meme index = meme rangee de la grille : l'egalite des deux listes EST l'alignement.
    expect(left).toEqual(keys.slice(half))
    expect(new Set(left).size).toBe(left.length)
    const kills = left.slice(left.indexOf('frag_donut'), left.indexOf('matches'))
    expect(kills).toEqual(['frag_donut', 'tools', 'control', 'fil', 'grid', 'mine', 'production', 'yield', 'lives', 'objective_balance', 'objective_sheet'])
    expect(left).not.toContain('usage')
    expect(left).not.toContain('frags')
  })

  it('le cote sans la carte porte le marqueur : la session solo n’a pas de carte d’objectif', async () => {
    const { container } = await openedCompare()
    for (const key of ['objective_balance', 'objective_sheet', 'control', 'mine']) {
      const [left, right] = cells(container, key)
      expect(placeholder(left), `${key} a gauche`).toBeNull()
      expect(placeholder(right), `${key} a droite`).not.toBeNull()
    }
    // Les cartes que les deux sessions ont : aucun marqueur.
    for (const key of ['frag_donut', 'tools', 'production', 'lives']) {
      for (const cell of cells(container, key)) expect(placeholder(cell), key).toBeNull()
    }
    expect(screen.getAllByText('Sans équivalent dans cette session').length).toBeGreaterThan(0)
  })

  it('les titres de groupe et de sous-groupe se posent dans la meme rangee des deux cotes', async () => {
    const { container } = await openedCompare()
    const [gauche, droite] = cells(container, 'frag_donut')
    for (const cell of [gauche, droite]) expect(cell.querySelector('h3')?.textContent).toBe('Frags et usages')
    for (const [key, sub] of [['control', 'resources'], ['production', 'prendre'], ['lives', 'lives'], ['objective_balance', 'objectif']]) {
      const pair = cells(container, key)
      for (const cell of pair) {
        expect(cell.querySelector(`[data-session-subgroup="${sub}"]`), `${key} / ${sub}`).not.toBeNull()
      }
    }
    // Aucune couverture en petit dans la vue de comparaison.
    expect(container.querySelector('[data-session-subgroup="resources"] small')).toBeNull()
  })

  it('A : l’anneau des DEUX cotes, chacun avec SON total, sans gamertag', async () => {
    const { container } = await openedCompare()
    const [gauche, droite] = cells(container, 'frag_donut')
    expect(gauche.querySelector('[data-testid="frag-sunburst"]')?.textContent).toContain('65')
    expect(droite.querySelector('[data-testid="frag-sunburst"]')?.textContent).toContain('72')
    for (const cell of [gauche, droite]) expect(cell.textContent).not.toContain('JGtm')
  })

  it('rend « Appui reçu » des DEUX cotes, chacune avec SES donnees, et plus de Riposte', async () => {
    const { container } = await openedCompare(true)
    const pair = cells(container, 'coordination')
    expect(pair).toHaveLength(2)
    for (const cell of pair) {
      expect(placeholder(cell)).toBeNull()
      expect(cell.querySelector('[data-session-coordination]')).not.toBeNull()
      expect(cell.textContent).toContain('Appui reçu')
      expect(cell.textContent).not.toContain('Riposte')
    }
    // Chaque colonne parle de SA session : la couverture differe des deux cotes.
    expect(pair[0].innerHTML).not.toEqual(pair[1].innerHTML)
  })

  it('ne pose ni rangees ni placeholder en pleine page (drawer ferme)', async () => {
    mockDetail()
    const { container } = renderWithProviders(<SessionDetailPage />)
    await waitFor(() => {
      expect(screen.getByRole('button', { name: /Comparer/i })).toBeInTheDocument()
    })
    expect(sectionKeys(container)).toHaveLength(0)
    expect(screen.queryByTestId('session-section-placeholder')).not.toBeInTheDocument()
  })
})
