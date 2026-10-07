/**
 * Tests — MatchViewTabArsenal, l'onglet « Armes et terrain » reconstruit aux formes de l'Emprise.
 *
 * Les cartes de l'Emprise sont RÉELLES (modèles, textes, briques partagées) sur les deux témoins de
 * `matchEmprise.fixtures.ts` ; seules les feuilles qui lisent l'artefact de rejeu, les positions ou
 * ECharts sont mockées. Couvre : l'ordre des cartes (plan §3), l'intertitre et sa couverture, la
 * rangée G | H, le repli des râteliers, la ligne non identifiée, l'ordre des lignes d'« Isolement »,
 * le retrait par carte, un match sans film, l'anglais et l'état vide de l'onglet.
 */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, within } from '@testing-library/react'

import type { MatchEmpriseBlock } from '@/lib/api/types'

import { MATCH_VIEW_TEXT } from './i18n'
import { FLOOD_GULCH, STARBOARD, STARBOARD_LIVES, STARBOARD_SCOREBOARD, XUID } from './matchEmprise.fixtures'
import { MatchViewTabArsenal } from './MatchViewTabArsenal'

const hoisted = vi.hoisted(() => ({ equipment: true }))

vi.mock('@/lib/replay/queries', () => ({ useMatchReplay: () => ({ data: hoisted.equipment ? {} : undefined }) }))
vi.mock('@/features/match-replay/model/equipmentUsageLogic', () => ({
  buildEquipmentUsage: () => ({}),
  hasEquipmentUsage: () => hoisted.equipment,
}))
vi.mock('@/features/match-replay/MatchEquipmentUsageSection', () => ({
  MatchEquipmentUsageSection: () => (hoisted.equipment ? <div data-testid="equipment-usage" /> : null),
}))
vi.mock('./MatchFragCard', () => ({ MatchFragCard: () => <div data-testid="frag-card" /> }))
vi.mock('./MatchKillDistanceSection', () => ({ MatchKillDistanceSection: () => <div data-testid="kill-distance" /> }))
vi.mock('./MatchPositionsHeatmap', () => ({
  MatchPositionsHeatmap: ({ positions }: { positions?: unknown[] }) => (positions?.length ? <div data-testid="positions-heatmap" /> : null),
}))
vi.mock('./blockPredicates', async (importOriginal) => ({
  ...(await importOriginal<typeof import('./blockPredicates')>()),
  useHasKillDistanceSection: () => false,
}))

beforeEach(() => {
  hoisted.equipment = true
})

interface Over {
  emprise?: MatchEmpriseBlock | null
  lives?: typeof STARBOARD_LIVES | null
  locale?: 'fr' | 'en'
  positions?: boolean
}

function afficher({ emprise = STARBOARD, lives = STARBOARD_LIVES, locale = 'fr', positions = true }: Over = {}) {
  return render(
    <MatchViewTabArsenal
      playerSlug="jgtm"
      matchId="ab526724"
      replayAvailable
      scoreboard={STARBOARD_SCOREBOARD}
      roster={[]}
      fragDistribution={null}
      weaponTools={null}
      killDistance={null}
      emprise={emprise}
      livesNearTeammate={lives}
      meXUID={XUID.jgtm}
      friendGamertags={[]}
      matchPositions={positions ? [{ timeMs: 0, x: 1, y: 2, z: 0, team: 0 }] : []}
      locale={locale}
      t={MATCH_VIEW_TEXT[locale]}
    />,
  )
}

/** Les cartes de « Équipement et terrain », dans l'ordre du document. */
const CARDS = ['match-emprise-control', 'emprise-sheets', 'equipment-usage', 'emprise-production', 'emprise-yield', 'match-emprise-lives', 'positions-heatmap']
const order = (container: HTMLElement) =>
  [...container.querySelectorAll<HTMLElement>('[data-testid]')].map((n) => n.dataset.testid).filter((id) => CARDS.includes(id!))

describe('MatchViewTabArsenal — « Équipement et terrain »', () => {
  it('22/09 : les sept cartes dans l’ordre du plan (D, E, F, G, H, I, J)', () => {
    const { container } = afficher()
    expect(order(container)).toEqual(CARDS)
  })

  it('l’intertitre porte sa couverture : film décodé, joueurs présents à la fin', () => {
    afficher()
    expect(screen.getByText(MATCH_VIEW_TEXT.fr.sectionEquipmentTerrain)).toBeInTheDocument()
    expect(screen.getByTestId('match-emprise-coverage').textContent).toBe('film décodé · 8 joueurs présents à la fin')
  })

  it('« Frags par ressource » et « Rendement par ressource » côte à côte', () => {
    afficher()
    const row = screen.getByTestId('emprise-production').parentElement!
    expect(row.className).toContain('lg:grid-cols-2')
    expect(row.contains(screen.getByTestId('emprise-yield'))).toBe(true)
  })

  it('contrôle : les armes de râtelier repliées derrière leur bouton, dépliées au clic', () => {
    afficher()
    expect(screen.queryByTestId('piste-camps-row-rack|vk78')).toBeNull()
    const bouton = screen.getByTestId('match-emprise-racks-toggle')
    expect(bouton.textContent).toContain('(3, repliées)')
    fireEvent.click(bouton)
    expect(screen.getByTestId('piste-camps-row-rack|vk78')).toBeInTheDocument()
    expect(bouton.getAttribute('aria-expanded')).toBe('true')
  })

  it('contrôle : les objets pris en retrait sous leur ressource, les bonus avec leurs socles vidés', () => {
    afficher()
    const control = within(screen.getByTestId('match-emprise-control'))
    expect((control.getByTestId('piste-camps-row-powerup|overshield').firstElementChild as HTMLElement).dataset.indent).toBe('true')
    const bonus = control.getByTestId('piste-camps-row-powerup').textContent
    expect(bonus).toContain('Prises de bonus')
    expect(bonus).toContain('10 socles vidés')
    expect(bonus).not.toContain('prises ·')
  })

  it('24/07 : la ligne des prises non identifiées sous les pistes', () => {
    afficher({ emprise: FLOOD_GULCH, lives: null })
    expect(screen.getByTestId('match-emprise-unclassified').textContent).toBe(
      '31 prises sur un emplacement non identifié, hors des pistes (équipe 19, adversaire 12)',
    )
  })

  it('24/07 : « Frags par ressource » dit pourquoi les frags pendant l’effet manquent', () => {
    afficher({ emprise: FLOOD_GULCH, lives: null })
    expect(screen.getByTestId('piste-camps-pending-powerup').textContent).toBe('Frags pendant l’effet non mesurés : journal des morts non publiable')
    expect(screen.getByTestId('emprise-yield-pending-powerup').textContent).toContain('Non mesuré : frags pendant l’effet non publiés')
  })

  it('chaque état du journal des morts a son texte (frags et rendement par ressource)', () => {
    const textes = {
      publishable: ['Aucun frag pendant l’effet d’un bonus', null],
      not_publishable: ['Frags pendant l’effet non mesurés : journal des morts non publiable', 'Non mesuré : frags pendant l’effet non publiés'],
      unavailable: ['Non mesuré : lecture indisponible', 'Non mesuré : lecture indisponible'],
    } as const
    for (const [etat, [frags, rendement]] of Object.entries(textes)) {
      const vue = afficher({ emprise: { ...FLOOD_GULCH, kill_journal: etat as MatchEmpriseBlock['kill_journal'] }, lives: null })
      expect(within(screen.getByTestId('emprise-production')).getByTestId('piste-camps-pending-powerup').textContent).toBe(frags)
      const ligne = screen.queryByTestId('emprise-yield-pending-powerup')
      if (rendement == null) expect(ligne).toBeNull()
      else expect(ligne?.textContent).toContain(rendement)
      if (etat === 'unavailable') expect(document.body.textContent).not.toContain('non publiable')
      vue.unmount()
    }
  })
  it('« Isolement » : une ligne par joueur dans l’ordre des fiches de « Prises par joueur »', () => {
    afficher()
    const lives = screen.getByTestId('match-emprise-lives')
    const names = ['JGtm', 'XL JACOB', 'Madina97294', 'Chocoboflor']
    const positions = names.map((n) => lives.textContent!.indexOf(n))
    expect(positions.every((p) => p >= 0)).toBe(true)
    expect([...positions].sort((a, b) => a - b)).toEqual(positions)
  })

  it('chaque carte se retire seule : sans vies, pas d’« Isolement » ; sans usage d’équipements, pas de grille', () => {
    hoisted.equipment = false
    const { container } = afficher({ lives: null })
    expect(order(container)).toEqual(CARDS.filter((id) => id !== 'match-emprise-lives' && id !== 'equipment-usage'))
  })

  it('sans film (Halo 5 : feuille seule) : la barre épaisse des armes spéciales, rien d’autre de l’Emprise', () => {
    hoisted.equipment = false
    const halo5: MatchEmpriseBlock = {
      ...STARBOARD,
      film_unavailable: 'film_unsupported',
      objects: [],
      resources: [],
      matches: [{ match_id: 'h5', has_film: false, team_known: false, resources: [] }],
      production: [{ resource: 'power_weapon', kills: { us: 3, them: 5 } }],
    }
    const { container } = afficher({ emprise: halo5, lives: null, positions: false })
    expect(order(container)).toEqual(['emprise-production'])
    expect(screen.getByTestId('piste-camps-row-power_weapon')).toBeInTheDocument()
    expect(screen.queryByTestId('piste-camps-pending-powerup')).toBeNull()
    expect(screen.getByTestId('match-emprise-coverage').textContent).toBe('sans film')
  })

  it('EN : intertitre, titres des cartes et couverture en anglais', () => {
    afficher({ locale: 'en' })
    expect(screen.getByText(MATCH_VIEW_TEXT.en.sectionEquipmentTerrain)).toBeInTheDocument()
    expect(screen.getByText('Resource control, by match')).toBeInTheDocument()
    expect(screen.getByText('Isolation, by player')).toBeInTheDocument()
    expect(screen.getByTestId('match-emprise-coverage').textContent).toBe('film decoded · 8 players present at the end')
  })

  it('rien à montrer : l’état vide nommé de l’onglet, aucun intertitre', () => {
    hoisted.equipment = false
    afficher({ emprise: null, lives: null, positions: false })
    expect(screen.queryByText(MATCH_VIEW_TEXT.fr.sectionEquipmentTerrain)).toBeNull()
    expect(screen.getByText(MATCH_VIEW_TEXT.fr.arsenalEmptyTitle)).toBeInTheDocument()
  })
})
