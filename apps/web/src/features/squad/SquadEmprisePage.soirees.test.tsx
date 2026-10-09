/**
 * SquadEmprisePage.soirees.test.tsx — l'onglet Emprise, sa fin : « Contrôle des ressources, par
 * soirée » (soirées hors comparaison grisées, bloc placeholder) puis les sections d'objectif
 * arrivées de Contributions (présentes aussi sans film). Même montage que SquadEmprisePage.test.tsx.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { screen, within } from '@testing-library/react'

import type { SquadEmpriseBlock, TeammateRow, TeammatesPageResponse } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'
import { renderWithProviders } from '@/test/render-utils'

import { empriseTabHasContent } from './emprise/empriseContent'
import { EMPRISE_2209, HISTORY_2209, XUID } from './emprise/emprise.fixtures'
import { block0709, history0709 } from './objectif/objectif.fixtures'
import * as squadContextModule from './SquadContext'
import { SquadEmprisePage } from './SquadEmprisePage'

vi.mock('echarts-for-react', () => ({
  default: () => <div data-testid="echarts-mock" />,
}))

// Libellés d'issue du manifest du titre (`outcomes.toml`), dans la langue de l'interface,
// servis sans attendre le chargement des correspondances.
const { OUTCOMES } = vi.hoisted(() => ({
  OUTCOMES: {
    fr: { win: 'Victoire', loss: 'Défaite', tie: 'Égalité', dnf: 'Abandon' },
    en: { win: 'Win', loss: 'Loss', tie: 'Tie', dnf: 'DNF' },
  } as Record<string, Record<string, string>>,
}))
vi.mock('@/lib/i18n/fieldMappings', async (importOriginal) => {
  const { useAppShellStore: store } = await import('@/stores/appShellStore')
  return {
    ...(await importOriginal<typeof import('@/lib/i18n/fieldMappings')>()),
    useOutcomeLabel: (key: string) => OUTCOMES[store.getState().locale]?.[key] ?? key,
  }
})

const ROW = (gamertag: string): TeammateRow => ({
  gamertag,
  xuid: 'x',
  encounter_count: 7,
  last_seen_at: undefined,
  with_kpis: {
    match_count: 7,
    wins: 3,
    kd_ratio: 1.1,
    win_rate: 0.43,
    accuracy: 0.45,
    kills_per_game: 12,
    assists_per_game: 4,
    headshot_kills_per_game: 3,
    perfect_kills_per_game: 1,
  },
  without_kpis: undefined,
})

function page(block = EMPRISE_2209): TeammatesPageResponse {
  return {
    options: [],
    teammates: [],
    total_matches: 7,
    session_labels: { solo: [], squad: [] },
    friends_count: 2,
    main_player: 'JGtm',
    match_history: HISTORY_2209,
    squad_emprise: block,
  } as TeammatesPageResponse
}

function mount(opts: { gamertags?: string[]; rows?: TeammateRow[]; pageData?: TeammatesPageResponse | null } = {}) {
  const gamertags = opts.gamertags ?? ['Chocoboflor', 'Madina97294']
  vi.spyOn(squadContextModule, 'useSquadContext').mockReturnValue({
    selectedRows: opts.rows ?? gamertags.map(ROW),
    confirmedGamertags: gamertags,
    pageData: opts.pageData === undefined ? page() : opts.pageData,
    playerSlug: 'jgtm',
    currentPlayerXuid: XUID.jgtm,
  })
  return renderWithProviders(<SquadEmprisePage />)
}

beforeEach(() => {
  useAppShellStore.setState({ locale: 'fr' })
})

afterEach(() => {
  vi.restoreAllMocks()
})

describe('Soirées précédentes', () => {
  it('après le rendement ; « Contrôle des ressources, par soirée » en demi-largeur à gauche, rien à sa droite', async () => {
    mount()
    const prendre = screen.getByTestId('emprise-section-prendre')
    const habitude = screen.getByTestId('emprise-section-habitude')
    expect(prendre.compareDocumentPosition(habitude) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
    const card = screen.getByTestId('emprise-habit')
    expect(card.parentElement?.className).toContain('lg:grid-cols-2')
    expect(card.parentElement?.children).toHaveLength(1)
    expect(await within(card).findByTestId('echarts-mock')).toBeInTheDocument()
    const legend = within(card).getByTestId('chart-card-legend')
    for (const label of ['Bonus', 'Armes spéciales', '50 % : autant que l’adversaire']) expect(legend.textContent).toContain(label)
  })

  it('sans soirée précédente : le graphe reste, ce soir seul ; aucune part nulle part : le bloc placeholder', async () => {
    mount({ pageData: page({ ...EMPRISE_2209, habit: { ...EMPRISE_2209.habit!, previous: [] } }) })
    expect(await within(screen.getByTestId('emprise-habit')).findByTestId('echarts-mock')).toBeInTheDocument()
    expect(screen.queryByTestId('emprise-habit-note')).toBeNull()
    vi.restoreAllMocks()
    const sansPart = { ...EMPRISE_2209.habit!, previous: [], current: { ...EMPRISE_2209.habit!.current, shares: [] } }
    mount({ pageData: page({ ...EMPRISE_2209, habit: sansPart }) })
    const note = screen.getByTestId('emprise-habit-note')
    expect(note.textContent).toContain('Aucune prise')
    expect(note.querySelector('.border-dashed')).not.toBeNull()
  })

  it('une soirée d’autres modes que ce soir : la légende nomme le point gris', () => {
    const previous = EMPRISE_2209.habit!.previous!.map((e, i) => (i === 0 ? { ...e, comparable: false, families: ['Bases'] } : e))
    mount({ pageData: page({ ...EMPRISE_2209, habit: { ...EMPRISE_2209.habit!, previous } }) })
    const legend = within(screen.getByTestId('emprise-habit')).getByTestId('chart-card-legend')
    expect(legend.textContent).toContain('Autres modes que ce soir')
  })

  it('sans habitude publiée : le bloc se retire', () => {
    mount({ pageData: page({ ...EMPRISE_2209, habit: undefined }) })
    expect(screen.queryByTestId('emprise-section-habitude')).toBeNull()
  })
})

describe('Objectif (arrivé de Contributions)', () => {
  const withObjective = (block: SquadEmpriseBlock | undefined): TeammatesPageResponse =>
    ({ ...page(), squad_emprise: block, formes_retenues: block0709(), match_history: history0709() }) as TeammatesPageResponse

  it('les sections « Objectif » puis « Répartition de l’objectif dans l’escouade » ferment l’onglet', () => {
    mount({ pageData: withObjective(EMPRISE_2209) })
    const habitude = screen.getByTestId('emprise-section-habitude')
    const objectif = screen.getByTestId('squad-objective-section')
    const fiches = screen.getByTestId('squad-objective-sheets-section')
    expect(habitude.compareDocumentPosition(objectif) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
    expect(objectif.compareDocumentPosition(fiches) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
  })

  it('sans film mais avec de l’objectif : l’onglet montre l’objectif, pas « aucun film décodé »', () => {
    mount({ pageData: withObjective(undefined) })
    expect(screen.getByTestId('squad-objective-section')).toBeInTheDocument()
    expect(screen.queryByText('Aucun film décodé')).toBeNull()
    expect(empriseTabHasContent({ squad_emprise: undefined, formes_retenues: block0709() })).toBe(true)
    expect(empriseTabHasContent({ squad_emprise: undefined, formes_retenues: undefined })).toBe(false)
  })
})
