/**
 * Tests — MatchKillFeed (kill feed DOM de la carte Dominance).
 *
 * Ce que ces tests protègent, dans l'ordre d'importance :
 *  1. AUCUNE ICÔNE FAUSSE. Un kill dont le backend n'a pas résolu l'arme rend un repère
 *     neutre, jamais l'icône d'un autre kill.
 *  2. La couleur d'équipe suit la MÊME cascade que l'en-tête du scoreboard.
 *  3. Aucun hex ni classe Tailwind de couleur ne sort de ce composant (règle color-tokens) :
 *     les couleurs viennent de `lib/halo/` (référentiel de jeu) ou d'un token sémantique.
 */
import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'

import { MatchKillFeed } from './MatchKillFeed'
import { MATCH_VIEW_TEXT } from './i18n'
import type { MomentumKill } from './_momentum'
import type { MatchScoreboardRow, MatchTugOfWarBin } from '@/lib/api/types'

vi.mock('@/lib/accessibility', () => ({
  resolveToken: (token: string) => `var(${token})`,
  tokenCssVar: (token: string) => `var(--ac-${token})`,
}))

const t = MATCH_VIEW_TEXT.fr

const BINS: MatchTugOfWarBin[] = [
  { bin_start: 0, bin_end: 60, team_kills: 0, enemy_kills: 0, net_kills: 0 },
  { bin_start: 60, bin_end: 120, team_kills: 0, enemy_kills: 0, net_kills: 0 },
]

function kill(over: Partial<MomentumKill>): MomentumKill {
  return {
    tMs: 1000,
    xuid: 'me',
    ally: true,
    binIdx: 0,
    fracInBin: 0.5,
    teamID: 0,
    weaponKey: '',
    weaponLabel: '',
    weaponImageUrl: '',
    weaponTinted: false,
    assistState: '',
    assistGamertag: '',
    assistTeamID: null,
    killerDamagePct: null,
    assistDamagePct: null,
    victimXuid: '',
    victimGamertag: '',
    victimTeamID: null,
    ...over,
  }
}

const META = new Map([
  ['me', { gamertag: 'JGtm', ally: true }],
  ['foe', { gamertag: 'Cobra01', ally: false }],
])

function sbRow(over: Partial<MatchScoreboardRow>): MatchScoreboardRow {
  return { xuid: 'x', gamertag: 'GT', team_side: 't0', is_me: false, ...over } as MatchScoreboardRow
}

describe('MatchKillFeed — arme du kill', () => {
  it('rend l’icône de l’arme quand le backend l’a résolue', () => {
    render(
      <MatchKillFeed
        bins={BINS}
        kills={[kill({ weaponLabel: 'BR75', weaponImageUrl: '/static/w/killfeed-00.png', weaponTinted: true })]}
        scoreboard={[]}
        xuidMeta={META}
        t={t}
      />,
    )
    expect(screen.getByRole('img', { name: 'BR75' })).toBeTruthy()
  })

  it('sans arme résolue, aucun <img> n’est rendu — repère neutre, jamais une icône fausse', () => {
    const { container } = render(
      <MatchKillFeed bins={BINS} kills={[kill({})]} scoreboard={[]} xuidMeta={META} t={t} />,
    )
    expect(container.querySelectorAll('img')).toHaveLength(0)
    expect(screen.queryAllByRole('img')).toHaveLength(0)
    // Le kill existe quand même dans la lane : il n'est pas escamoté.
    expect(screen.getAllByRole('listitem')).toHaveLength(1)
  })

  it('affiche la couverture mesurée (le repli est compté, pas caché)', () => {
    render(
      <MatchKillFeed
        bins={BINS}
        kills={[
          kill({ tMs: 1000, weaponImageUrl: '/static/w/killfeed-00.png', weaponLabel: 'BR75' }),
          kill({ tMs: 2000 }),
          kill({ tMs: 3000, xuid: 'foe', ally: false, teamID: 1 }),
        ]}
        scoreboard={[]}
        xuidMeta={META}
        t={t}
      />,
    )
    expect(screen.getByText('1/3 avec arme')).toBeTruthy()
  })

  it('le survol expose le nom du tueur, le nom de l’arme et l’instant', () => {
    render(
      <MatchKillFeed
        bins={BINS}
        kills={[kill({ tMs: 65_000, weaponLabel: 'Needler', weaponImageUrl: '/static/w/killfeed-19.png' })]}
        scoreboard={[]}
        xuidMeta={META}
        t={t}
      />,
    )
    expect(screen.getByTitle('JGtm — Needler — 1:05')).toBeTruthy()
  })

  it('sans arme, le survol le DIT au lieu de laisser croire à une arme', () => {
    render(
      <MatchKillFeed bins={BINS} kills={[kill({ tMs: 0 })]} scoreboard={[]} xuidMeta={META} t={t} />,
    )
    expect(screen.getByTitle('JGtm — Arme inconnue — 0:00')).toBeTruthy()
  })
})

describe('MatchKillFeed — couleur d’équipe', () => {
  it('utilise la couleur officielle du team_id (même cascade que l’en-tête du scoreboard)', () => {
    render(
      <MatchKillFeed
        bins={BINS}
        kills={[kill({ teamID: 1 })]} // Cobra
        scoreboard={[]}
        xuidMeta={META}
        t={t}
      />,
    )
    const item = screen.getByRole('listitem')
    // #FE3939 = couleur officielle Cobra (lib/halo/teamNames), rendue en rgb par jsdom.
    expect(item.getAttribute('style')).toContain('rgb(254, 57, 57)')
  })

  it('la couleur fournie par le backend prime sur la map officielle', () => {
    render(
      <MatchKillFeed
        bins={BINS}
        kills={[kill({ teamID: 1 })]}
        scoreboard={[sbRow({ team_side: 't1', team_color: '#00FF00' })]}
        xuidMeta={META}
        t={t}
      />,
    )
    expect(screen.getByRole('listitem').getAttribute('style')).toContain('rgb(0, 255, 0)')
  })

  it('sans team_id, retombe sur le token sémantique allié/ennemi', () => {
    render(
      <MatchKillFeed
        bins={BINS}
        kills={[kill({ teamID: null, ally: false, xuid: 'foe' })]}
        scoreboard={[]}
        xuidMeta={META}
        t={t}
      />,
    )
    expect(screen.getByRole('listitem').getAttribute('style')).toContain('var(--ac-team-enemy)')
  })
})

describe('MatchKillFeed — dégradations', () => {
  it('aucun kill → rien rendu (pas de bandeau vide sous le graphe)', () => {
    const { container } = render(
      <MatchKillFeed bins={BINS} kills={[]} scoreboard={[]} xuidMeta={META} t={t} />,
    )
    expect(container.firstChild).toBeNull()
  })

  it('sépare les lanes alliée et ennemie', () => {
    render(
      <MatchKillFeed
        bins={BINS}
        kills={[kill({ tMs: 1000 }), kill({ tMs: 2000, xuid: 'foe', ally: false, teamID: 1 })]}
        scoreboard={[]}
        xuidMeta={META}
        t={t}
      />,
    )
    expect(screen.getByRole('list', { name: t.combatTeamLabel }).querySelectorAll('[role="listitem"]')).toHaveLength(1)
    expect(screen.getByRole('list', { name: t.combatEnemyLabel }).querySelectorAll('[role="listitem"]')).toHaveLength(1)
  })
})
