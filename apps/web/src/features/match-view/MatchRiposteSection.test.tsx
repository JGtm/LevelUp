/**
 * MatchRiposteSection — les deux graphes, l'échelle commune et les deux états vides.
 *
 * Ce que ces tests cadenassent : « on ne sait pas » ne s'écrit jamais « aucune », les deux
 * camps sortent côte à côte, la borne des barres est celle du MATCH (pas celle du camp), et
 * l'infobulle d'une barre nomme les couples plutôt qu'un compte de plus.
 */
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'

import type { MatchRiposteBlock, MatchScoreboardRow } from '@/lib/api/types'
import { MatchRiposteSection } from './MatchRiposteSection'
import { MATCH_VIEW_TEXT } from './i18n'

vi.mock('@/lib/accessibility', async (orig) => ({
  ...(await orig<Record<string, unknown>>()),
  tokenCssVar: (token: string) => `var(--${token})`,
  resolveToken: (token: string) => `var(--${token})`,
}))

const t = MATCH_VIEW_TEXT.fr

const scoreboard = [
  { xuid: 'me', gamertag: 'JGtm', team_side: 't1', is_me: true },
  { xuid: 'kaya', gamertag: 'Kaya', team_side: 't1', is_me: false },
  { xuid: 'vex', gamertag: 'Vex', team_side: 't0', is_me: false },
] as unknown as MatchScoreboardRow[]

const block = (over: Partial<MatchRiposteBlock> = {}): MatchRiposteBlock => ({
  fenetre_ms: 5000,
  measured_deaths: 6,
  deaths: [
    {
      victim_xuid: 'me',
      victim_gamertag: 'JGtm',
      victim_team_id: 1,
      killer_xuid: 'vex',
      time_ms: 1000,
      avenged: true,
      avenger_xuid: 'kaya',
      avenger_gamertag: 'Kaya',
      delai_ms: 3100,
      vengeable: true,
    },
  ],
  players: [
    { xuid: 'me', gamertag: 'JGtm', team_id: 1, deaths_avenged: 1, ripostes: 1 },
    { xuid: 'kaya', gamertag: 'Kaya', team_id: 1, deaths_avenged: 0, ripostes: 4 },
    { xuid: 'vex', gamertag: 'Vex', team_id: 0, deaths_avenged: 0, ripostes: 2 },
  ],
  ...over,
})

describe('MatchRiposteSection — états', () => {
  it('ne rend RIEN quand le bloc est absent (aucune ligne de journal)', () => {
    const { container } = render(
      <MatchRiposteSection block={undefined} scoreboard={scoreboard} meXUID="me" locale="fr" t={t} />,
    )
    expect(container).toBeEmptyDOMElement()
  })

  it('nomme l’état quand aucune mort n’est lisible (measured_deaths = 0)', () => {
    render(
      <MatchRiposteSection
        block={block({ measured_deaths: 0, deaths: [], players: [] })}
        scoreboard={scoreboard}
        meXUID="me"
        locale="fr"
        t={t}
      />,
    )
    expect(screen.getByText(t.riposteNotUsable)).toBeInTheDocument()
  })

  it('distingue « mesuré, mais personne n’a riposté »', () => {
    render(
      <MatchRiposteSection
        block={block({
          deaths: [
            {
              victim_xuid: 'me',
              victim_team_id: 1,
              time_ms: 10,
              avenged: false,
              vengeable: true,
            },
          ],
          players: [{ xuid: 'me', gamertag: 'JGtm', team_id: 1, deaths_avenged: 0, ripostes: 0 }],
        })}
        scoreboard={scoreboard}
        meXUID="me"
        locale="fr"
        t={t}
      />,
    )
    expect(screen.getByText(t.riposteNoData)).toBeInTheDocument()
  })
})

describe('MatchRiposteSection — rendu', () => {
  it('rend UN graphe PAR CAMP, le mien en premier', () => {
    render(
      <MatchRiposteSection block={block()} scoreboard={scoreboard} meXUID="me" locale="fr" t={t} />,
    )
    const camps = screen.getAllByTestId(/^riposte-camp-/)
    expect(camps.map((c) => c.getAttribute('data-testid'))).toEqual([
      'riposte-camp-t1',
      'riposte-camp-t0',
    ])
  })

  it('emploie UNE SEULE échelle pour les deux camps et les deux côtés', () => {
    render(
      <MatchRiposteSection block={block()} scoreboard={scoreboard} meXUID="me" locale="fr" t={t} />,
    )
    // Borne du match = 4 (les ripostes de Kaya). 2 ripostes dans l'autre camp = la moitié,
    // et 1 mort vengée = le quart — si l'échelle était par camp, Vex tiendrait 100 %.
    expect(screen.getByTestId('riposte-bar-did-kaya')).toHaveStyle({ width: '100%' })
    expect(screen.getByTestId('riposte-bar-did-vex')).toHaveStyle({ width: '50%' })
    expect(screen.getByTestId('riposte-bar-avenged-me')).toHaveStyle({ width: '25%' })
  })

  it('trie chaque camp sur les ripostes portées', () => {
    render(
      <MatchRiposteSection block={block()} scoreboard={scoreboard} meXUID="me" locale="fr" t={t} />,
    )
    const mien = screen.getByTestId('riposte-camp-t1')
    const barres = within(mien)
      .getAllByTestId(/^riposte-bar-did-/)
      .map((b) => b.getAttribute('data-testid'))
    expect(barres).toEqual(['riposte-bar-did-kaya', 'riposte-bar-did-me'])
  })

  it('écrit le pied : morts vengées sur morts mesurées', () => {
    render(
      <MatchRiposteSection block={block()} scoreboard={scoreboard} meXUID="me" locale="fr" t={t} />,
    )
    expect(screen.getByText(t.riposteFooterFmt(1, 6))).toBeInTheDocument()
  })

  it('nomme le couple dans l’infobulle d’une barre', async () => {
    const user = userEvent.setup()
    render(
      <MatchRiposteSection block={block()} scoreboard={scoreboard} meXUID="me" locale="fr" t={t} />,
    )
    await user.hover(screen.getByTestId('riposte-bar-avenged-me'))
    expect(await screen.findByText(t.riposteAvengedByFmt('Kaya', '3,1'))).toBeInTheDocument()
  })
})
