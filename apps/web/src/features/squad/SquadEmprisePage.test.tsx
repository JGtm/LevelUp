/**
 * SquadEmprisePage.test.tsx — l'onglet « Emprise » (lot L5 du plan
 * PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26), sur la soirée témoin du 22/09 (chiffres de la
 * maquette de l'onglet) : blocs et ordre du débrief, puis chaque carte — « Contrôle des ressources »
 * (compte · part dans chaque segment, repli S3, trait 50 %), « … cumul par match » (graphe tracé,
 * légende), « Prises par joueur » (fiches, bonus perdus en couleurs d'équipe, pastilles pleines et
 * vides), « … par match » (résultat, dominance, sans film, râteliers repliés). États vides hérités
 * de l'onglet remplacé.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, screen, within } from '@testing-library/react'

import type { SquadEmpriseBlock, TeammateRow, TeammatesPageResponse } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'
import { renderWithProviders } from '@/test/render-utils'

import { empriseHasContent } from './emprise/empriseContent'
import { EMPRISE_2209, HISTORY_2209, XUID } from './emprise/emprise.fixtures'
import { PLACEMENT_2209 } from './emprise/placement.fixtures'
import { formatMatchTime } from './formes/format'
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

const text = (id: string) => screen.getByTestId(id).textContent ?? ''

describe('SquadEmprisePage — structure', () => {
  it('trois blocs dans l’ordre du débrief ; le bilan en deux cartes côte à côte', () => {
    mount()
    const sections = ['emprise-section-bilan', 'emprise-section-roles', 'emprise-section-carte'].map((id) => screen.getByTestId(id))
    for (let i = 1; i < sections.length; i++) {
      expect(sections[i - 1].compareDocumentPosition(sections[i]) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
    }
    expect(within(sections[0]).getByText('Ressources')).toBeInTheDocument()
    // « Prises par joueur » : l'intertitre seul — les fiches sont à même la section, sans carte
    // englobante ni titre redondant.
    expect(within(sections[1]).getAllByText('Prises par joueur')).toHaveLength(1)
    expect(screen.getByTestId('emprise-sheets').className).not.toContain('border')
    expect(within(sections[2]).getByText('Par match')).toBeInTheDocument()
    const control = screen.getByTestId('emprise-control')
    const fil = screen.getByTestId('emprise-fil')
    expect(control.parentElement).toBe(fil.parentElement)
    expect(control.parentElement?.className).toContain('lg:grid-cols-2')
    expect(control.compareDocumentPosition(fil) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
  })

  it('titres factuels (S1)', () => {
    mount()
    for (const title of ['Contrôle des ressources', 'Contrôle des ressources, cumul par match', 'Contrôle des ressources, par match']) {
      expect(screen.getByText(title)).toBeInTheDocument()
    }
    expect(screen.getAllByText('Prises par joueur').length).toBeGreaterThan(0)
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
    const bonus = within(screen.getByTestId('emprise-control')).getByTestId('piste-camps-row-powerup')
    expect(bonus.textContent).toContain('Prises de bonus')
    expect(bonus.textContent).toContain('camouflage, surbouclier')
    expect(bonus.textContent).not.toContain('prises ·')
    expect(bonus.textContent).toContain('12 · 60 %')
    expect(bonus.textContent).toContain('40 % · 8')
    const power = within(screen.getByTestId('emprise-control')).getByTestId('piste-camps-row-power_weapon')
    expect(power.textContent).toContain('Prises d’armes spéciales')
    expect(power.textContent).not.toContain('prises sur les socles')
    expect(power.textContent).toContain('23 · 44,2 %')
    expect(power.textContent).toContain('55,8 % · 29')
  })

  it('repli S3 : hors navigateur rien ne « tient », les deux camps passent au-dessus, pastille devant', () => {
    mount()
    const repli = within(screen.getByTestId('emprise-control')).getByTestId('piste-camps-repli-powerup')
    expect(repli.textContent).toContain('12 · 60 %')
    expect(repli.textContent).toContain('8 · 40 %')
  })

  it('légende en pied de carte : équipe, adversaire, « 50 % : autant que l’adversaire »', () => {
    mount()
    const legend = within(screen.getByTestId('emprise-control')).getByTestId('objectif-legend')
    expect(legend.textContent).toContain('Équipe')
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
  it('bonus perdus : 2 sur 12 (17 %) et 2 sur 8 (25 %), pastilles d’équipe au lieu de « équipe » / « adversaire » écrits', () => {
    mount()
    expect(text('emprise-losses')).toContain('Bonus perdus')
    expect(text('emprise-losses-us')).toBe('2 sur 12 (17 %)')
    expect(text('emprise-losses-them')).toBe('2 sur 8 (25 %)')
    expect(text('emprise-losses')).not.toMatch(/nous|eux/)
    expect(within(screen.getByTestId('emprise-losses-us')).getByRole('img', { name: 'Équipe' })).toBeInTheDocument()
  })

  it('fiches JGtm, Chocoboflor, Madina97294, sans fiche du reste de l’équipe ; JGtm : 3 bonus · 9 armes spéciales', () => {
    mount()
    for (const id of [XUID.jgtm, XUID.choco, XUID.madina]) {
      expect(screen.getByTestId(`emprise-sheet-${id}`)).toBeInTheDocument()
    }
    expect(screen.queryByTestId('emprise-sheet-rest')).toBeNull()
    expect(screen.queryByText('Reste de l’équipe')).toBeNull()
    expect(text(`emprise-sheet-foot-${XUID.jgtm}-powerup`)).toBe('3 bonus')
    expect(text(`emprise-sheet-foot-${XUID.jgtm}-power_weapon`)).toBe('9 armes spéciales')
    const jgtm = screen.getByTestId(`emprise-sheet-${XUID.jgtm}`)
    expect(jgtm.textContent).toContain('Ressource dominante')
    expect(jgtm.textContent).toContain('Armes spéciales')
    expect(screen.getByTestId(`emprise-sheet-${XUID.madina}`).textContent).toContain('Bonus')
  })

  it('une pastille par prise, pastille vide = bonus perdu ; un zéro reste une ligne atténuée', () => {
    mount()
    const madinaCamo = screen.getByTestId(`emprise-sheet-line-${XUID.madina}-powerup_camo`)
    expect(madinaCamo.querySelectorAll('[data-dot="taken"]')).toHaveLength(3)
    expect(madinaCamo.querySelectorAll('[data-dot="lost"]')).toHaveLength(1)
    const jgtmSurb = screen.getByTestId(`emprise-sheet-line-${XUID.jgtm}-powerup_overshield`)
    expect(jgtmSurb.querySelectorAll('[data-dot="taken"]')).toHaveLength(2)
    const jgtmCamo = screen.getByTestId(`emprise-sheet-line-${XUID.jgtm}-powerup_camo`)
    expect(jgtmCamo.getAttribute('data-zero')).toBeNull()
    const zero = screen.getByTestId(`emprise-sheet-${XUID.madina}`).querySelector('[data-zero="true"]')
    expect(zero).not.toBeNull()
    expect(zero!.querySelectorAll('[data-dot]')).toHaveLength(0)
  })

  it('mêmes lignes, même ordre sur toutes les fiches', () => {
    mount()
    const order = (id: string) =>
      Array.from(screen.getByTestId(`emprise-sheet-${id}`).querySelectorAll('[data-testid^="emprise-sheet-line-"]')).map((el) =>
        el.getAttribute('data-testid')!.split('-').pop(),
      )
    const ref = order(XUID.jgtm)
    // Neuf objets pris par l'équipe, moins les deux que seul le reste de l'équipe a pris.
    expect(ref).toHaveLength(7)
    for (const id of [XUID.choco, XUID.madina]) expect(order(id)).toEqual(ref)
  })
})

describe('Contrôle des ressources, match par match', () => {
  it('colonnes : heure, carte, mode, « Victoire 3–0 », sans badge de dominance', () => {
    mount()
    const head = text('emprise-grid-head-m1')
    expect(head).toContain(formatMatchTime(HISTORY_2209[0].start_time, 'fr'))
    expect(head).toContain('Starboard')
    expect(head).toContain('Drapeau')
    expect(text('emprise-grid-result-m1')).toBe('Victoire 3–0')
    expect(screen.queryByTestId('emprise-grid-dominance-m1')).toBeNull()
    expect(text('emprise-grid-result-m3')).toBe('Défaite 1–3')
  })

  it('cases « 5–2 », « — » sans objet, « sans film » à Detachment ; frags aux armes spéciales lus sans film', () => {
    mount()
    const table = screen.getByTestId('emprise-grid-table')
    expect(within(table).getAllByText('5–2').length).toBeGreaterThan(0)
    expect(within(table).getAllByText('sans film').length).toBeGreaterThan(0)
    expect(within(table).getAllByText('—').length).toBeGreaterThan(0)
    expect(within(table).getByText('9–13')).toBeInTheDocument()
    expect(within(table).getByText('Frags avec arme spéciale')).toBeInTheDocument()
    expect(within(table).getAllByText('Prises de bonus').length).toBeGreaterThan(0)
    expect(within(table).queryByText('frags obtenus avec')).toBeNull()
  })

  it('constat R2 (revue L6.1) : un match filmé à équipe inconnue dit « équipe inconnue », pas « — »', () => {
    const block: SquadEmpriseBlock = {
      ...EMPRISE_2209,
      matches: EMPRISE_2209.matches!.map((m) => (m.match_id === 'm2' ? { ...m, team_known: false } : m)),
    }
    mount({ pageData: page(block) })
    const table = screen.getByTestId('emprise-grid-table')
    const cells = table.querySelectorAll('[data-cell="noteam"]')
    // Synthèse bonus, deux objets bonus, synthèse des armes spéciales et ses armes (râteliers repliés).
    expect(cells.length).toBeGreaterThanOrEqual(3)
    expect(cells[0].textContent).toBe('équipe inconnue')
    expect(within(table).queryByText('4–0')).toBeNull()
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
    for (const title of ['Resource control', 'Resource control, cumulative by match', 'Resource control, by match']) {
      expect(screen.getByText(title)).toBeInTheDocument()
    }
    expect(screen.getAllByText('Pickups by player').length).toBeGreaterThan(0)
    expect(text('emprise-losses-us')).toBe('2 of 12 (17%)')
    expect(text('emprise-grid-result-m1')).toBe('Win 3–0')
  })
})

describe('Rendement des ressources', () => {
  it('après « Par match » ; Frags par ressource | Rendement par ressource côte à côte, même rangée', () => {
    mount()
    const carte = screen.getByTestId('emprise-section-carte')
    const prendre = screen.getByTestId('emprise-section-prendre')
    expect(carte.compareDocumentPosition(prendre) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
    expect(within(prendre).getByText('Rendement des ressources')).toBeInTheDocument()
    const production = screen.getByTestId('emprise-production')
    const rendement = screen.getByTestId('emprise-yield')
    expect(production.parentElement).toBe(rendement.parentElement)
    expect(production.parentElement?.className).toContain('lg:grid-cols-2')
    expect(production.compareDocumentPosition(rendement) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
  })

  it('Frags par ressource : 8 · 61,5 % / 38,5 % · 5 pendant l’effet, temps d’effet 2 min 39 · 58,5 % … 1 min 53 ; armes spéciales 38 / 41', () => {
    mount()
    const card = screen.getByTestId('emprise-production')
    const bonus = within(card).getByTestId('piste-camps-row-powerup')
    expect(bonus.textContent).toContain('Frags pendant l’effet d’un bonus')
    expect(bonus.textContent).toContain('8 · 61,5 %')
    expect(bonus.textContent).toContain('38,5 % · 5')
    expect(text('emprise-production-exposure-powerup')).toBe('temps d’effet : 2 min 39 · 58,5 %1 min 53')
    const armes = within(card).getByTestId('piste-camps-row-power_weapon')
    expect(armes.textContent).toContain('Frags avec arme spéciale')
    expect(armes.textContent).not.toContain('frags obtenus avec')
    // Constat R3 (revue L6.1) : la barre épaisse des armes spéciales porte sur la population de la
    // fine et du rendement (38 / 41, matchs aux prises mesurées), pas sur la feuille entière (47 / 54).
    expect(armes.textContent).toContain('38 · 48,1 %')
    expect(armes.textContent).toContain('51,9 % · 41')
    expect(armes.textContent).not.toContain('47')
    expect(text('emprise-production-exposure-power_weapon')).toBe('prises sur les socles : 23 prises · 44,2 %29 prises')
    // Une barre fine sous chaque barre épaisse (rôle img, nom = la ligne d'exposition).
    expect(within(bonus).getByRole('img', { name: /temps d’effet : 2 min 39/ })).toBeInTheDocument()
    const legend = within(card).getByTestId('objectif-legend')
    for (const label of ['Équipe', 'Adversaire', '50 % : autant que l’adversaire', 'Barre fine : temps d’effet, prises ou temps à bord']) {
      expect(legend.textContent).toContain(label)
    }
  })

  it('Rendement par ressource : bonus +14 % (3,0 contre 2,7), armes spéciales +17 % (1,7 contre 1,4), axe −50 / +50 %', () => {
    mount()
    expect(text('emprise-yield-gap-powerup')).toBe('+14 %')
    expect(text('emprise-yield-raw-powerup')).toBe('3,0 contre 2,7')
    expect(text('emprise-yield-gap-power_weapon')).toBe('+17 %')
    expect(text('emprise-yield-raw-power_weapon')).toBe('1,7 contre 1,4')
    const card = screen.getByTestId('emprise-yield')
    expect(card.textContent).toContain('frags par minute d’effet')
    expect(card.textContent).toContain('frags par prise')
    for (const tick of ['−50 %', '−25', 'autant', '+25', '+50 %']) expect(card.textContent).toContain(tick)
    // Barre divergente depuis le zéro : à droite pour un écart positif.
    const bar = screen.getByTestId('emprise-yield-bar-powerup').closest('[style*="left"]') as HTMLElement
    expect(bar.style.left).toBe('50%')
    const legend = within(card).getByTestId('objectif-legend')
    expect(legend.textContent).toContain('Équipe plus productive')
    expect(legend.textContent).toContain('Moins')
  })
})

describe('Halo 5, sans film (D10)', () => {
  // Le bloc tel que le Go le publie pour un titre sans résumé d'usage du film : la feuille seule.
  const sansFilm: SquadEmpriseBlock = {
    matches_total: 7,
    matches_measured: 0,
    film_unavailable: 'film_unsupported',
    players: EMPRISE_2209.players,
    resources: [],
    objects: [],
    matches: (EMPRISE_2209.matches ?? []).map((m) => ({
      match_id: m.match_id,
      has_film: false,
      team_known: true,
      resources: [],
      power_weapon_kills: m.power_weapon_kills,
    })),
    production: [{ resource: 'power_weapon', kills: { us: 47, them: 54 } }],
  }

  it('l’onglet ne garde que les frags aux armes spéciales, barre épaisse seule, sur toute la rangée', () => {
    mount({ pageData: page(sansFilm) })
    for (const id of ['emprise-section-bilan', 'emprise-section-roles', 'emprise-section-carte', 'emprise-section-habitude']) {
      expect(screen.queryByTestId(id)).toBeNull()
    }
    const card = screen.getByTestId('emprise-production')
    const armes = within(card).getByTestId('piste-camps-row-power_weapon')
    expect(armes.textContent).toContain('47 · 46,5 %')
    expect(within(card).queryByTestId('piste-camps-row-powerup')).toBeNull()
    expect(screen.queryByTestId('emprise-production-exposure-power_weapon')).toBeNull()
    expect(within(armes).getAllByRole('img')).toHaveLength(1)
    expect(screen.queryByTestId('emprise-yield')).toBeNull()
    expect(card.parentElement?.className).toContain('lg:[&>*:only-child]:col-span-2')
    expect(within(card).getByTestId('objectif-legend').textContent).not.toContain('Barre fine')
  })

  it('sans aucune donnée : rien à montrer (la barre d’onglets masque l’onglet)', () => {
    const vide: SquadEmpriseBlock = {
      ...sansFilm,
      production: [],
      matches: (sansFilm.matches ?? []).map((m) => ({ ...m, power_weapon_kills: undefined })),
    }
    expect(empriseHasContent(vide)).toBe(false)
    expect(empriseHasContent(sansFilm)).toBe(true)
    expect(empriseHasContent(EMPRISE_2209)).toBe(true)
    expect(empriseHasContent(undefined)).toBe(false)
  })
})

describe('Prendre et habitude — anglais (S12)', () => {
  it('titres et valeurs en anglais', () => {
    useAppShellStore.setState({ locale: 'en' })
    mount()
    for (const title of ['Kills by resource', 'Efficiency by resource', 'Resource control, by session']) {
      expect(screen.getByText(title)).toBeInTheDocument()
    }
    expect(text('emprise-yield-raw-powerup')).toBe('3.0 vs 2.7')
    expect(text('emprise-yield-gap-powerup')).toBe('+14%')
    expect(text('emprise-production-exposure-powerup')).toBe('effect time: 2 min 39 · 58.5%1 min 53')
  })
})

describe('Isolement (lot V4 du plan Emprise vies)', () => {
  const withPlacement = (placement = PLACEMENT_2209) => page({ ...EMPRISE_2209, placement })
  const sansMesure = () => ({ ...PLACEMENT_2209, players: (PLACEMENT_2209.players ?? []).map((p) => ({ ...p, lives_measured: 0, lives: [] })) })

  it('le bloc se montre entre « Rendement des ressources » et « Soirées précédentes », le nuage au-dessus de la barre des quarts', () => {
    mount({ pageData: withPlacement() })
    const prendre = screen.getByTestId('emprise-section-prendre')
    const section = screen.getByTestId('emprise-section-placement')
    const habitude = screen.getByTestId('emprise-section-habitude')
    expect(prendre.compareDocumentPosition(section) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
    expect(section.compareDocumentPosition(habitude) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
    expect(within(section).getByText('Isolement')).toBeInTheDocument()
    const vie = within(section).getByTestId('emprise-placement-vie')
    const quarts = within(section).getByTestId('emprise-placement-quarts')
    expect(vie.compareDocumentPosition(quarts) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
    // Pleine largeur : les deux cartes s'empilent, aucune grille à deux colonnes.
    expect(vie.parentElement).toBe(quarts.parentElement)
    expect(vie.parentElement?.className).not.toContain('grid-cols-2')
    expect(within(vie).getByText('Placement et rendement de chaque vie')).toBeInTheDocument()
    expect(within(quarts).getByText('Part des vies par placement')).toBeInTheDocument()
    for (const card of [vie, quarts]) expect(within(card).getByTestId('echarts-mock')).toBeInTheDocument()
  })

  it('titres en anglais', () => {
    useAppShellStore.setState({ locale: 'en' })
    mount({ pageData: withPlacement() })
    const section = screen.getByTestId('emprise-section-placement')
    expect(within(section).getByText('Isolation')).toBeInTheDocument()
    expect(within(section).getByText('Placement and yield of each life')).toBeInTheDocument()
    expect(within(section).getByText('Share of lives by placement')).toBeInTheDocument()
  })

  it('sans le bloc placement : le bloc est absent, le reste de l’onglet est là', () => {
    mount()
    expect(screen.queryByTestId('emprise-section-placement')).toBeNull()
    expect(screen.getByTestId('emprise-section-prendre')).toBeInTheDocument()
  })

  it('placement sans aucune vie mesurée : le bloc est absent', () => {
    mount({ pageData: withPlacement(sansMesure()) })
    expect(screen.queryByTestId('emprise-section-placement')).toBeNull()
  })

  it('l’onglet reste masqué sans aucun contenu, y compris avec un placement sans vie mesurée ; une vie mesurée suffit à le montrer', () => {
    const rien: SquadEmpriseBlock = { matches_total: 7, matches_measured: 0, players: EMPRISE_2209.players, resources: [], objects: [], matches: [], production: [] }
    const sansVie = sansMesure()
    expect(empriseHasContent(rien)).toBe(false)
    expect(empriseHasContent({ ...rien, placement: sansVie })).toBe(false)
    expect(empriseHasContent({ ...rien, placement: PLACEMENT_2209 })).toBe(true)
  })
})
