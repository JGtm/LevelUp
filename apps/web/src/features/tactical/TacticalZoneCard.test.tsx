/**
 * TacticalZoneCard et TacticalRejeuTile — la zone sélectionnée et ses mini-tuiles « Rejeu »
 * (plan Tactique v2, L6.3).
 *
 * Ce que ces tests cadenassent :
 *   - sans sélection : « Zone sélectionnée » et « Aucune zone sélectionnée », rien d'autre ;
 *   - une zone : son nom en jeu (ou « Zone sans nom »), ses coordonnées, sa valeur (signée sur une
 *     lecture signée), sa sous-ligne par lecture, l'intertitre « Rejeu » et ses tuiles ;
 *   - la tuile : deux lignes, l'arme seule tronquée, le texte complet en infobulle, le bouton de
 *     rejeu seulement si l'artefact existe ET que le titre sert le rejeu, l'ordre du serveur (du
 *     plus récent au plus ancien) ;
 *   - le lien de rejeu est un `<Link>` du routeur portant `?t=&clock=`.
 */
import type React from 'react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { screen, within } from '@testing-library/react'

import type { CelluleTactique, TacticalCelluleReponse, TacticalContribution } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'
import { renderWithProviders } from '@/test/render-utils'

import { getTacticalText } from './i18n'
import { TacticalZoneCard } from './TacticalZoneCard'

vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return {
    ...actual,
    Link: ({
      params,
      search,
      children,
      ...rest
    }: {
      to: string
      params: { titleSlug: string; playerSlug: string; matchId: string }
      search?: Record<string, string>
      children?: React.ReactNode
    } & Record<string, unknown>) => {
      const base = `/${params.titleSlug}/players/${params.playerSlug}/matches/${params.matchId}/replay`
      const qs = search ? `?${new URLSearchParams(search).toString()}` : ''
      return (
        <a href={`${base}${qs}`} {...(rest as Record<string, unknown>)}>
          {children}
        </a>
      )
    },
  }
})

vi.mock('@/lib/i18n/fieldMappings', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/lib/i18n/fieldMappings')>()
  return {
    ...actual,
    useOutcomeMapping: (key: string) =>
      key === 'win' ? { label: 'Victoire' } : key === 'loss' ? { label: 'Défaite' } : undefined,
  }
})

vi.mock('@/lib/title-routing', () => ({ useTitleSlug: () => 'halo_infinite' }))

const t = getTacticalText('fr')

const CELLULE: CelluleTactique = {
  col: -7,
  lig: 2,
  valeur: 3,
  brut: 3,
  matchs: 4,
  matchs_victoire: 2,
  matchs_defaite: 2,
  frags: 3,
  morts: 1,
  centre_x: -13,
  centre_y: 5,
}

function contribution(extra: Partial<TacticalContribution> = {}): TacticalContribution {
  return {
    match_id: 'm1',
    instant_ms: 83_400,
    clock: 'match',
    xuid: '2533274000000001',
    match_started_at: '2026-09-01T22:30:00Z',
    replay_available: true,
    resultat: 'win',
    face: 'mort',
    autre_gamertag: 'Rival',
    arme_label: 'Fusil de combat à longue portée',
    mode_label: 'Assassin',
    score_label: '50 - 42',
    placement: { seul: true, distance_m: 30.7 },
    ...extra,
  }
}

const DETAIL: TacticalCelluleReponse = {
  contributions: [
    contribution(),
    contribution({ match_id: 'm2', instant_ms: 5_000, resultat: 'loss', replay_available: false, match_started_at: '2026-08-30T20:00:00Z' }),
  ],
  matchs_non_ouvrables: 0,
  zone: { nom_fr: 'Nid blindé', nom_en: 'Armored Nest' },
}

function rendre(
  props: Partial<{
    cellule: CelluleTactique | null
    detail: { data?: TacticalCelluleReponse; isPending: boolean; isError: boolean }
    question: 'morts' | 'solde' | 'gagne' | 'temps'
    signee: boolean
    locale: 'fr' | 'en'
  }> = {},
) {
  const locale = props.locale ?? 'fr'
  return renderWithProviders(
    <TacticalZoneCard
      t={getTacticalText(locale)}
      locale={locale}
      playerSlug="JGtm"
      question={props.question ?? 'morts'}
      signee={props.signee ?? false}
      pasM={2}
      cellule={props.cellule === undefined ? CELLULE : props.cellule}
      detail={props.detail ?? { data: DETAIL, isPending: false, isError: false }}
    />,
  )
}

beforeEach(() => {
  useAppShellStore.setState({
    locale: 'fr',
    userTimezone: 'UTC',
    currentTitleSlug: 'halo_infinite',
    availableTitles: [{ slug: 'halo_infinite', name: 'Halo Infinite', capabilities: ['replay'] }] as never,
  })
})

describe('TacticalZoneCard — la zone sélectionnée', () => {
  it('sans sélection : le titre « Zone sélectionnée » et une ligne, rien d’autre', () => {
    rendre({ cellule: null, detail: { isPending: false, isError: false } })
    const carte = screen.getByTestId('tactical-zone-card')
    expect(carte).toHaveTextContent(t.zoneTitle)
    expect(within(carte).getByText(t.zoneNone)).toBeInTheDocument()
    expect(within(carte).queryByText(t.zoneReplayHeading)).toBeNull()
  })

  it('une zone : nom en jeu, coordonnées, valeur et unité, sous-ligne, « Rejeu » et ses tuiles', () => {
    rendre()
    const carte = screen.getByTestId('tactical-zone-card')
    expect(screen.getByTestId('tactical-zone-title')).toHaveTextContent('Nid blindé')
    expect(carte).toHaveTextContent('x −14…−12 m · y 4…6 m')
    expect(screen.getByTestId('tactical-zone-value')).toHaveTextContent('3')
    expect(carte).toHaveTextContent(t.units.morts)
    expect(carte).toHaveTextContent('4 matchs distincts')
    expect(within(carte).getByText(t.zoneReplayHeading)).toBeInTheDocument()
    expect(within(carte).getAllByTestId('tactical-rejeu-tile')).toHaveLength(2)
  })

  it('le nom de la zone dans la langue de la page', () => {
    rendre({ locale: 'en' })
    expect(screen.getByTestId('tactical-zone-title')).toHaveTextContent('Armored Nest')
  })

  it('pendant la lecture du détail : « Zone sélectionnée », la valeur déjà servie', () => {
    rendre({ detail: { isPending: true, isError: false } })
    expect(screen.getByTestId('tactical-zone-title')).toHaveTextContent(t.zoneTitle)
    expect(screen.getByTestId('tactical-zone-value')).toHaveTextContent('3')
  })

  it('aucune zone ne la nomme : « Zone sans nom »', () => {
    rendre({ detail: { data: { ...DETAIL, zone: undefined }, isPending: false, isError: false } })
    expect(screen.getByTestId('tactical-zone-title')).toHaveTextContent(t.zoneUnnamed)
  })

  it('lecture « victoires − défaites » : valeur signée, victoires et défaites', () => {
    rendre({ question: 'gagne', signee: true, cellule: { ...CELLULE, valeur: 0.25 } })
    expect(screen.getByTestId('tactical-zone-value')).toHaveTextContent('+ 0,25')
    expect(screen.getByTestId('tactical-zone-card')).toHaveTextContent('2 victoires, 2 défaites · 4 matchs distincts')
  })

  it('lecture « solde » : frags et morts', () => {
    rendre({ question: 'solde', signee: true, cellule: { ...CELLULE, valeur: -0.5 } })
    expect(screen.getByTestId('tactical-zone-value')).toHaveTextContent('− 0,5')
    expect(screen.getByTestId('tactical-zone-card')).toHaveTextContent('3 frags, 1 mort · 4 matchs distincts')
  })

  it('lecture d’artefact : la tuile dit l’entrée dans la zone', () => {
    rendre({
      question: 'temps',
      detail: { data: { ...DETAIL, contributions: [contribution({ face: 'entree', clock: 'film', autre_gamertag: undefined, placement: undefined })] }, isPending: false, isError: false },
    })
    expect(screen.getByTestId('tactical-rejeu-tile')).toHaveTextContent('Entrée dans la zone')
  })

  it('aucun match ouvrable : la liste le dit', () => {
    rendre({ detail: { data: { ...DETAIL, contributions: [] }, isPending: false, isError: false } })
    expect(screen.getByText(t.zoneContributionsEmpty)).toBeInTheDocument()
  })
})

describe('TacticalRejeuTile — la mini-tuile d’une contribution', () => {
  it('deux lignes : mode, score, issue, date ; instant, fait, arme, badge', () => {
    rendre()
    const tuile = screen.getAllByTestId('tactical-rejeu-tile')[0]
    const l1 = within(tuile).getByTestId('tactical-rejeu-l1')
    const l2 = within(tuile).getByTestId('tactical-rejeu-l2')
    expect(l1).toHaveTextContent('Assassin')
    expect(l1).toHaveTextContent('50 - 42')
    expect(l1).toHaveTextContent('Victoire')
    expect(l1).toHaveTextContent('01/09/2026 · 22:30')
    expect(l2).toHaveTextContent('1:23')
    expect(l2).toHaveTextContent('Tué par Rival')
    expect(l2).toHaveTextContent('Fusil de combat à longue portée')
    expect(l2).toHaveTextContent('seul · 30 m')
  })

  it('l’arme seule se tronque ; le texte complet est en infobulle', () => {
    rendre()
    const tuile = screen.getAllByTestId('tactical-rejeu-tile')[0]
    expect(within(tuile).getByText('Fusil de combat à longue portée').className).toContain('truncate')
    expect(within(tuile).getByText('Tué par Rival').className).toContain('flex-none')
    expect(tuile.getAttribute('title')).toBe(
      'Assassin · 50 - 42 · Victoire · 01/09/2026 · 22:30 · 1:23 · Tué par Rival · Fusil de combat à longue portée · seul · 30 m',
    )
  })

  it('le bouton de rejeu : à l’instant, seulement si l’artefact existe', () => {
    rendre()
    const [avec, sans] = screen.getAllByTestId('tactical-rejeu-tile')
    const lien = within(avec).getByRole('link', { name: 'Ouvrir le rejeu à 1:23' })
    expect(lien.getAttribute('href')).toBe('/halo_infinite/players/JGtm/matches/m1/replay?t=83400&clock=match')
    expect(within(sans).queryByRole('link')).toBeNull()
  })

  it('titre sans rejeu : aucun bouton', () => {
    useAppShellStore.setState({
      availableTitles: [{ slug: 'halo_infinite', name: 'Halo Infinite', capabilities: [] }] as never,
    })
    rendre()
    expect(screen.queryByRole('link')).toBeNull()
  })

  it('l’ordre du serveur : du plus récent au plus ancien', () => {
    rendre()
    const liens = screen.getAllByTestId('tactical-rejeu-tile').map((n) => n.textContent ?? '')
    expect(liens[0]).toContain('01/09/2026')
    expect(liens[1]).toContain('30/08/2026')
  })
})
