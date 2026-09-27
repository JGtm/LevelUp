/**
 * SquadEmprisePage.test.tsx — l'onglet « Emprise » (lot L5 du plan
 * PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26), sur la soirée témoin du 22/09 (chiffres de la
 * maquette de l'onglet) : blocs et ordre du débrief, puis chaque carte — « Contrôle des
 * ressources » (compte · part dans chaque segment, repli S3, trait 50 %), « … au fil de la
 * session » (graphe tracé, légende), « Répartition des prises dans l'escouade » (fiches, bonus
 * perdus en couleurs d'équipe, pastilles pleines et vides), « … match par match » (résultat,
 * dominance, sans film, râteliers repliés). États vides hérités de l'onglet remplacé.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, screen, within } from '@testing-library/react'

import type { TeammateRow, TeammatesPageResponse } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'
import { renderWithProviders } from '@/test/render-utils'

import { EMPRISE_2209, HISTORY_2209, XUID } from './emprise/emprise.fixtures'
import { formatMatchTime } from './formes/format'
import * as squadContextModule from './SquadContext'
import { SquadEmprisePage } from './SquadEmprisePage'

vi.mock('echarts-for-react', () => ({
  default: () => <div data-testid="echarts-mock" />,
}))

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

const text = (id: string) => screen.getByTestId(id).textContent ?? ''

describe('SquadEmprisePage — structure', () => {
  it('trois blocs dans l’ordre du débrief ; le bilan en deux cartes côte à côte', () => {
    mount()
    const sections = ['emprise-section-bilan', 'emprise-section-roles', 'emprise-section-carte'].map((id) => screen.getByTestId(id))
    for (let i = 1; i < sections.length; i++) {
      expect(sections[i - 1].compareDocumentPosition(sections[i]) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
    }
    expect(within(sections[0]).getByText('Bilan de la soirée')).toBeInTheDocument()
    expect(within(sections[1]).getByText('Rôles dans l’escouade')).toBeInTheDocument()
    expect(within(sections[2]).getByText('Carte par carte')).toBeInTheDocument()
    const control = screen.getByTestId('emprise-control')
    const fil = screen.getByTestId('emprise-fil')
    expect(control.parentElement).toBe(fil.parentElement)
    expect(control.parentElement?.className).toContain('lg:grid-cols-2')
    expect(control.compareDocumentPosition(fil) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
  })

  it('titres factuels (S1)', () => {
    mount()
    for (const title of [
      'Contrôle des ressources',
      'Contrôle des ressources au fil de la session',
      'Répartition des prises dans l’escouade',
      'Contrôle des ressources, match par match',
    ]) {
      expect(screen.getByText(title)).toBeInTheDocument()
    }
  })

  it('sans coéquipier : l’invitation à choisir ; sélection invalide : son message', () => {
    mount({ gamertags: [] })
    expect(screen.getByText('Analyse de synergies')).toBeInTheDocument()
    vi.restoreAllMocks()
    mount({ gamertags: ['Inconnu'], rows: [] })
    expect(screen.queryByTestId('squad-emprise-page')).toBeNull()
  })

  it('sans bloc Emprise : l’onglet le dit (aucun film décodé)', () => {
    mount({ pageData: { ...page(), squad_emprise: undefined } as TeammatesPageResponse })
    expect(screen.getByText('Aucun film décodé')).toBeInTheDocument()
  })
})

describe('Contrôle des ressources', () => {
  it('une piste par ressource, pastille de couleur, « compte · part » dans chaque segment', () => {
    mount()
    const bonus = screen.getByTestId('piste-camps-row-powerup')
    expect(bonus.textContent).toContain('Bonus')
    expect(bonus.textContent).toContain('prises · camouflage, surbouclier')
    expect(bonus.textContent).toContain('12 · 60 %')
    expect(bonus.textContent).toContain('40 % · 8')
    const power = screen.getByTestId('piste-camps-row-power_weapon')
    expect(power.textContent).toContain('Armes spéciales')
    expect(power.textContent).toContain('prises sur les socles')
    expect(power.textContent).toContain('23 · 44,2 %')
    expect(power.textContent).toContain('55,8 % · 29')
  })

  it('repli S3 : hors navigateur rien ne « tient », les deux camps passent au-dessus, pastille devant', () => {
    mount()
    const repli = screen.getByTestId('piste-camps-repli-powerup')
    expect(repli.textContent).toContain('12 · 60 %')
    expect(repli.textContent).toContain('8 · 40 %')
  })

  it('légende en pied de carte : notre camp, adversaire, « 50 % : autant que l’adversaire »', () => {
    mount()
    const legend = within(screen.getByTestId('emprise-control')).getByTestId('objectif-legend')
    expect(legend.textContent).toContain('Notre camp')
    expect(legend.textContent).toContain('Adversaire')
    expect(legend.textContent).toContain('50 % : autant que l’adversaire')
    expect(legend.textContent).not.toContain('parité')
  })
})

describe('Contrôle des ressources au fil de la session', () => {
  it('le graphe est tracé ; légende : ressources, victoire / défaite, dominance, 50 %', () => {
    mount()
    const fil = screen.getByTestId('emprise-fil')
    expect(within(fil).getByTestId('echarts-mock')).toBeInTheDocument()
    const legend = within(fil).getByTestId('chart-card-legend')
    for (const label of ['Bonus', 'Armes spéciales', 'Victoire, défaite', 'Drapeau de dominance', '50 % : autant que l’adversaire']) {
      expect(legend.textContent).toContain(label)
    }
    expect(legend.textContent).not.toContain('Véhicules')
  })
})

describe('Répartition des prises dans l’escouade', () => {
  it('bonus perdus : 2 sur 12 (17 %) et 2 sur 8 (25 %), pastilles d’équipe au lieu de « nous » / « eux »', () => {
    mount()
    expect(text('emprise-losses')).toContain('Bonus perdus')
    expect(text('emprise-losses-us')).toBe('2 sur 12 (17 %)')
    expect(text('emprise-losses-them')).toBe('2 sur 8 (25 %)')
    expect(text('emprise-losses')).not.toMatch(/nous|eux/)
    expect(within(screen.getByTestId('emprise-losses-us')).getByRole('img', { name: 'Notre camp' })).toBeInTheDocument()
  })

  it('fiches JGtm, Chocoboflor, Madina97294 puis le reste du camp ; JGtm : 3 bonus · 9 armes spéciales', () => {
    mount()
    for (const id of [XUID.jgtm, XUID.choco, XUID.madina, 'rest']) {
      expect(screen.getByTestId(`emprise-sheet-${id}`)).toBeInTheDocument()
    }
    expect(text(`emprise-sheet-foot-${XUID.jgtm}-powerup`)).toBe('3 bonus')
    expect(text(`emprise-sheet-foot-${XUID.jgtm}-power_weapon`)).toBe('9 armes spéciales')
    expect(text('emprise-sheet-foot-rest-power_weapon')).toBe('8 armes spéciales')
    const jgtm = screen.getByTestId(`emprise-sheet-${XUID.jgtm}`)
    expect(jgtm.textContent).toContain('Ressource dominante')
    expect(jgtm.textContent).toContain('Armes spéciales')
    expect(screen.getByTestId(`emprise-sheet-${XUID.madina}`).textContent).toContain('Bonus')
    expect(screen.getByTestId('emprise-sheet-rest').textContent).toContain('Reste du camp')
  })

  it('une pastille par prise, pastille vide = bonus perdu ; un zéro reste une ligne atténuée', () => {
    mount()
    const madinaCamo = screen.getByTestId(`emprise-sheet-line-${XUID.madina}-powerup_camo`)
    expect(madinaCamo.querySelectorAll('[data-dot="taken"]')).toHaveLength(3)
    expect(madinaCamo.querySelectorAll('[data-dot="lost"]')).toHaveLength(1)
    const jgtmSurb = screen.getByTestId(`emprise-sheet-line-${XUID.jgtm}-powerup_overshield`)
    expect(jgtmSurb.querySelectorAll('[data-dot="taken"]')).toHaveLength(2)
    const restCamo = screen.getByTestId('emprise-sheet-line-rest-powerup_camo')
    expect(restCamo.getAttribute('data-zero')).toBe('true')
    expect(restCamo.querySelectorAll('[data-dot]')).toHaveLength(0)
  })

  it('mêmes lignes, même ordre sur toutes les fiches', () => {
    mount()
    const order = (id: string) =>
      Array.from(screen.getByTestId(`emprise-sheet-${id}`).querySelectorAll('[data-testid^="emprise-sheet-line-"]')).map((el) =>
        el.getAttribute('data-testid')!.split('-').pop(),
      )
    const ref = order(XUID.jgtm)
    expect(ref).toHaveLength(9)
    for (const id of [XUID.choco, XUID.madina, 'rest']) expect(order(id)).toEqual(ref)
  })
})

describe('Contrôle des ressources, match par match', () => {
  it('colonnes : heure, carte, mode, « Victoire 3–0 » et « Domination » à Starboard', () => {
    mount()
    const head = text('emprise-grid-head-m1')
    expect(head).toContain(formatMatchTime(HISTORY_2209[0].start_time, 'fr'))
    expect(head).toContain('Starboard')
    expect(head).toContain('Drapeau')
    expect(text('emprise-grid-result-m1')).toBe('Victoire 3–0')
    expect(text('emprise-grid-dominance-m1')).toBe('Domination')
    expect(text('emprise-grid-result-m3')).toBe('Défaite 1–3')
    expect(screen.queryByTestId('emprise-grid-dominance-m3')).toBeNull()
  })

  it('cases « 5–2 », « — » sans objet, « sans film » à Detachment ; frags aux armes spéciales lus sans film', () => {
    mount()
    const table = screen.getByTestId('emprise-grid-table')
    expect(within(table).getAllByText('5–2').length).toBeGreaterThan(0)
    expect(within(table).getAllByText('sans film').length).toBeGreaterThan(0)
    expect(within(table).getAllByText('—').length).toBeGreaterThan(0)
    expect(within(table).getByText('9–13')).toBeInTheDocument()
    expect(within(table).getByText('frags obtenus avec')).toBeInTheDocument()
  })

  it('armes de râtelier repliées derrière un bouton ; il les déplie', () => {
    mount()
    const toggle = screen.getByTestId('emprise-grid-racks-toggle')
    expect(toggle.textContent).toContain('Armes de râtelier')
    expect(toggle.textContent).toContain('(13, prises)')
    expect(toggle.getAttribute('aria-expanded')).toBe('false')
    expect(screen.queryByText('Déchiqueteur')).toBeNull()
    fireEvent.click(toggle)
    expect(toggle.getAttribute('aria-expanded')).toBe('true')
    expect(screen.getByText('Déchiqueteur')).toBeInTheDocument()
  })

  it('légende : plus / moins que l’adversaire, rien à prendre, sans film', () => {
    mount()
    const legend = within(screen.getByTestId('emprise-grid')).getByTestId('objectif-legend')
    for (const label of ['Plus que l’adversaire', 'Moins', 'Rien à prendre', 'Sans film']) {
      expect(legend.textContent).toContain(label)
    }
  })
})

describe('SquadEmprisePage — anglais (S12)', () => {
  it('titres et libellés en anglais', () => {
    useAppShellStore.setState({ locale: 'en' })
    mount()
    for (const title of ['Resource control', 'Resource control over the session', 'Pickups within the squad', 'Resource control, match by match']) {
      expect(screen.getByText(title)).toBeInTheDocument()
    }
    expect(text('emprise-losses-us')).toBe('2 of 12 (17%)')
    expect(text('emprise-grid-result-m1')).toBe('Win 3–0')
  })
})
