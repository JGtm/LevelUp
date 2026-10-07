/**
 * SessionColumnBody.test.tsx — LES TITRES DE LA PAGE SESSION et les cartes « Frags et usages » qu'ils
 * coiffent (plan PLAN_SESSIONS_EMPRISE_2026-10-06, S4.8), et la seule chose qui doit les faire bouger :
 * la donnée.
 *
 * CE QUE CE FICHIER FIXE :
 *   1. QUATRE titres de groupe, dans l'ORDRE — « Bilan », « Match par match », « Frags et usages »,
 *      « Détail des matchs » ; sous « Frags et usages », les intertitres des sous-groupes dans l'ordre
 *      de la maquette, « Ressources » avec sa couverture (pleine page seulement).
 *   2. Les cartes A à L dans l'ordre de la maquette, chacune se retirant seule sans donnée ; un
 *      sous-groupe sans carte n'a pas d'intertitre ; sans aucune carte, « Frags et usages » disparaît.
 *   3. Halo 5 (sans film) : A, B, B' et G seulement, aucun intertitre de ressources.
 *   4. IDENTITÉ compact / non compact des titres de groupe (le tiroir monte le MÊME composant).
 *   5. L'anglais.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { screen } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import { soloEmpriseSansFilm } from '@/features/timeseries/usages/usages.fixtures'
import type { SessionCompareEntry } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'

import { SessionColumnBody } from './SessionColumnBody'
import { session0709, session2209, sessionSolo } from './sessionEmprise.fixtures'
import type { SessionColumnBlocks } from './sessionEmprise.logic'

// ECharts (canvas) ne peint pas dans jsdom : ce fichier lit des titres et des cadres de cartes.
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

afterEach(() => useAppShellStore.setState({ locale: 'fr' }))

const SECTIONS = ['Bilan', 'Match par match', 'Frags et usages', 'Détail des matchs'] as const

/** Les cadres des cartes A à L, par leur repère de test (B et B' : le titre de leur carte de graphe). */
const CARD_IDS: [string, string][] = [
  ['frag_bar', '[data-testid="squad-frag-breakdown"]'],
  ['control', '[data-testid="emprise-control"]'],
  ['fil', '[data-testid="emprise-fil"]'],
  ['grid', '[data-testid="emprise-grid"]'],
  ['mine', '[data-testid="usages-mine"]'],
  ['production', '[data-testid="emprise-production"]'],
  ['yield', '[data-testid="emprise-yield"]'],
  ['lives', '[data-testid="usages-lives"]'],
  ['objective_balance', '[data-testid="objective-balance"]'],
  ['objective_sheet', '[data-testid="objective-solo-sheet"]'],
  ['equipment', '[data-testid="usages-equipment"]'],
]

/** Les cartes rendues, dans l'ordre du document ; B repérée par son titre. */
function cartes(container: HTMLElement): string[] {
  const sel = CARD_IDS.map(([, s]) => s).join(', ')
  const nodes = Array.from(container.querySelectorAll(`${sel}, [data-testid="chart-card"]`))
  return nodes
    .map((n) => {
      const hit = CARD_IDS.find(([, s]) => n.matches(s))
      if (hit) return hit[0]
      return n.textContent?.includes('Outils de destruction') || n.textContent?.includes('Tools of destruction') ? 'tools' : ''
    })
    .filter((k, i, all) => k !== '' && all.indexOf(k) === i)
}

function titresRendus(): string[] {
  return screen
    .getAllByRole('heading', { level: 3 })
    .map((h) => h.textContent ?? '')
    .filter((label) => (SECTIONS as readonly string[]).includes(label))
}

function intertitres(container: HTMLElement): string[] {
  return Array.from(container.querySelectorAll('[data-session-subgroup]')).map((h) => h.textContent ?? '')
}

function monter(blocks: SessionColumnBlocks, compact = false) {
  return renderWithProviders(<SessionColumnBody blocks={blocks} playerSlug="moi" compact={compact} />)
}

describe('SessionColumnBody — titres de groupe et intertitres', () => {
  it('soirée du 22/09 : quatre titres de groupe, intertitres dans l’ordre, couverture des ressources', () => {
    const { container } = monter(session2209())
    expect(titresRendus()).toEqual([...SECTIONS])
    expect(intertitres(container)).toEqual([
      'Ressources6 matchs filmés sur 7 · frags de la feuille de match sur les 7',
      'Rendement des ressources',
      'Isolement',
      'Objectif',
    ])
  })

  it('les cartes A à L, dans l’ordre de la maquette ; l’équipement non servi n’a ni carte ni intertitre', () => {
    const { container } = monter(session2209())
    expect(cartes(container)).toEqual([
      'frag_bar',
      'tools',
      'control',
      'fil',
      'grid',
      'mine',
      'production',
      'yield',
      'lives',
      'objective_balance',
      'objective_sheet',
    ])
    expect(screen.queryByText('Équipement')).not.toBeInTheDocument()
  })

  it('les paires A|B, C|D, G|H partagent une rangée en pleine page', () => {
    const { container } = monter(session2209())
    const pairs = Array.from(container.querySelectorAll('[data-session-pair]')).map((n) => n.getAttribute('data-session-pair'))
    expect(pairs).toEqual(['frag_bar|tools', 'control|fil', 'production|yield'])
  })
})

describe('SessionColumnBody — chaque carte se retire seule', () => {
  it('session solo sans match à objectif : ni rapport de force, ni fiche, ni intertitre « Objectif »', () => {
    const { container } = monter(sessionSolo())
    expect(cartes(container)).not.toContain('objective_balance')
    expect(cartes(container)).not.toContain('objective_sheet')
    expect(intertitres(container)).not.toContain('Objectif')
    // Aucune prise de ressource : ni contrôle, ni fil, ni « Contribution aux prises ».
    expect(cartes(container)).not.toContain('control')
    expect(cartes(container)).not.toContain('mine')
  })

  it('aucune carte : « Frags et usages » disparaît ENTIÈREMENT, titre compris', () => {
    monter({ entry: { ...session2209().entry, frag_distribution: undefined, weapon_tools: undefined } as unknown as SessionCompareEntry, matches: [] })
    expect(titresRendus()).toEqual(['Bilan', 'Match par match', 'Détail des matchs'])
  })

  it('Halo 5 (sans film) : A, B, B’ et G seulement, sans intertitre de ressources', () => {
    const base = session2209()
    const { container } = monter({
      entry: { ...base.entry, weapon_accuracy: [{ weapon_name: 'BR', shots_fired: 10, shots_hit: 5, accuracy: 50 }] } as unknown as SessionCompareEntry,
      matches: base.matches,
      emprise: soloEmpriseSansFilm(),
    })
    expect(cartes(container)).toEqual(['frag_bar', 'tools', 'production'])
    expect(screen.getByText('Précision par arme')).toBeInTheDocument()
    expect(intertitres(container)).toEqual(['Rendement des ressources'])
  })
})

describe('SessionColumnBody — le tiroir monte les mêmes groupes', () => {
  it('compact et non compact : mêmes titres de groupe, même ordre', () => {
    const plein = monter(session2209())
    const titresPlein = titresRendus()
    plein.unmount()
    monter(session2209(), true)
    expect(titresRendus()).toEqual(titresPlein)
  })

  it('compact : chaque carte est en vue compacte (A : parts et total en sous-libellé)', () => {
    monter(session2209(), true)
    expect(screen.getByText('65 frags')).toBeInTheDocument()
    expect(screen.queryByTestId('frag-breakdown-total-JGtm')).not.toBeInTheDocument()
    // Le jeu de textes de la vue compacte : légende de la grille réduite (maquette `makeGrid`, `cp`).
    expect(screen.getByText('Plus de 50 %')).toBeInTheDocument()
    expect(screen.queryByText('Plus que l’adversaire')).not.toBeInTheDocument()
  })

  it('compact, 07/09 : le rapport de force par rôle (Bases, prendre : 46 % / 54 %, MESURES §3)', () => {
    monter(session0709(), true)
    const row = screen.getByTestId('objective-balance-row-zones_strongholds-take')
    expect([...row.querySelectorAll('[data-fit-label]')].map((l) => l.textContent)).toEqual(['46 %', '54 %'])
  })

  it('pleine page, 07/09 : les actions, pas la barre par rôle', () => {
    monter(session0709())
    expect(screen.queryByTestId('objective-balance-row-zones_strongholds-take')).not.toBeInTheDocument()
    expect(screen.getByTestId('objective-balance')).toBeInTheDocument()
  })
})

describe('SessionColumnBody — anglais', () => {
  it('titres et intertitres en anglais', () => {
    useAppShellStore.setState({ locale: 'en' })
    const { container } = monter(session2209())
    expect(intertitres(container)).toEqual([
      'Resources6 filmed matches of 7 · kills from the match sheet over all 7',
      'Resource efficiency',
      'Isolation',
      'Objective',
    ])
    // L'intertitre et le titre de la carte portent le même nom (Séries temporelles).
    expect(screen.getAllByText('Isolation')).toHaveLength(2)
  })
})
