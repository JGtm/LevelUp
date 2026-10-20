/**
 * Tests — L'ENCRE DU FIL EST L'ALLÉGEANCE DU FILM (2026-10-06).
 *
 * Chaque nom du fil (tueur et icône de son arme, victime, décoré, défunt, arrivant ou partant)
 * prend l'encre de son allégeance du film vue du joueur regardé (`FilmAllegiance.ofXuid`) : les
 * acteurs du fil portent les xuid de la BASE, et un bot s'y relie par sa ligne de feuille
 * (`bid(N.0)`). Avant, l'encre venait de la feuille (`KillEvent.ally`, table d'identité) avec des
 * replis devinés : une victime inconnue prenait le camp opposé au tueur, un décoré inconnu
 * l'encre alliée.
 */
import { describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'

import type { KillEvent } from '@/features/match-view/_momentum'
import { filmAllegianceOf } from '@/lib/replay/filmAllegiance'

import type { MedalEvent, ReplayFeedEntry, ReplayKill } from '../model/killFeedLogic'
import { ReplayKillFeed } from './ReplayKillFeed'
import { scoreboardRow } from '../test/scoreboardRow'
import { testReplayDoc } from '../test/testDoc'

vi.mock('@/lib/accessibility', () => ({
  resolveToken: (token: string) => `var(${token})`,
  tokenCssVar: (token: string) => `var(--ac-${token})`,
}))

const ALLIE = 'var(--ac-team-ally)'
const ADVERSE = 'var(--ac-team-enemy)'
const NEUTRE = 'var(--muted-foreground)'

/**
 * Alpha (la référence) et le bot Sandwolf au camp 0, Bravo au camp 1, Muet sans équipe du film.
 * La feuille porte le bot comme la base le publie (`bid(44.0)`, `is_bot`) et met Muet du côté
 * d'Alpha : ce côté-là ne doit RIEN décider.
 */
const DOC = testReplayDoc({
  roster: [
    { xuid: 'A', filmIndex: 0, name: 'Alpha', team: 0 },
    { xuid: '', bot: true, filmIndex: 8, name: 'Sandwolf [bot]', team: 0 },
    { xuid: 'B', filmIndex: 1, name: 'Bravo', team: 1 },
    { xuid: 'M', filmIndex: 2, name: 'Muet' },
  ],
})
const BOARD = [
  scoreboardRow('A', 'Alpha', 't0', { is_me: true }),
  scoreboardRow('bid(44.0)', 'Sandwolf', 't0', { is_bot: true }),
  scoreboardRow('B', 'Bravo', 't1'),
  scoreboardRow('M', 'Muet', 't0'),
]
const NOMS = new Map([
  ['A', { gamertag: 'Alpha' }],
  ['bid(44.0)', { gamertag: 'Sandwolf' }],
  ['B', { gamertag: 'Bravo' }],
  ['M', { gamertag: 'Muet' }],
])

function kill(xuid: string, victimXuid: string, victimGamertag: string): ReplayKill {
  const k: KillEvent = {
    tMs: 1_000, xuid, ally: false, teamID: null, weaponKey: '', weaponLabel: 'BR75',
    weaponImageUrl: '/static/weapons/br75.png', weaponTinted: true, assistState: '',
    assistGamertag: '', assistTeamID: null, killerDamagePct: null, assistDamagePct: null,
    victimXuid, victimGamertag,
  }
  return { ...k, replayMs: 1_000, medals: [] }
}

const medal = (xuid: string, gamertag: string): MedalEvent => ({
  tMs: 1_000, xuid, gamertag, name: 'Double Kill', label: 'Doublé',
  description: '', imageUrl: '',
})

function ligne(over: Partial<ReplayFeedEntry> & { key: string }): ReplayFeedEntry {
  return { replayMs: 1_000, kill: null, medal: null, death: null, ...over }
}

function afficher(entries: ReplayFeedEntry[], reference = 'A') {
  return render(
    <ReplayKillFeed
      entries={entries}
      nowMs={60_000}
      playWindow={null}
      scoreboard={BOARD}
      xuidMeta={NOMS}
      allegiance={filmAllegianceOf(DOC, BOARD, reference)}
      locale="fr"
    />,
  )
}

const encreDe = (nom: string) => (screen.getByText(nom) as HTMLElement).style.color

describe('ReplayKillFeed — l’encre de chaque nom est l’allégeance du film', () => {
  it('un BOT du camp de la référence tue : son nom et son arme prennent l’encre ALLIÉE, sa victime l’adverse', () => {
    afficher([ligne({ key: 'k', kill: kill('bid(44.0)', 'B', 'Bravo') })])
    expect(encreDe('Sandwolf')).toBe(ALLIE)
    expect(screen.getByRole('img', { name: 'BR75' }).getAttribute('style') ?? '').toContain(ALLIE)
    expect(encreDe('Bravo')).toBe(ADVERSE)
  })

  it('un joueur dont le film TAIT l’équipe : encre NEUTRE, malgré sa ligne de feuille du camp allié', () => {
    afficher([ligne({ key: 'k', kill: kill('M', 'B', 'Bravo') })])
    expect(encreDe('Muet')).toBe(NEUTRE)
  })

  it('une victime absente du film : encre NEUTRE — plus de « camp opposé au tueur » deviné', () => {
    afficher([ligne({ key: 'k', kill: kill('A', 'X', 'Inconnu') })])
    expect(encreDe('Alpha')).toBe(ALLIE)
    expect(encreDe('Inconnu')).toBe(NEUTRE)
  })

  it('médaille seule, mort neutre et ligne de présence suivent la même règle', () => {
    afficher([
      ligne({ key: 'm', medal: medal('bid(44.0)', 'Sandwolf') }),
      ligne({ key: 'd', replayMs: 2_000, death: { replayMs: 2_000, xuid: 'B', kind: '', img: '', tinted: false } }),
      ligne({ key: 'p', replayMs: 3_000, presence: { kind: 'joined', xuid: 'M', name: 'Muet', bot: false, source: 'api' } }),
    ])
    expect(encreDe('Sandwolf')).toBe(ALLIE)
    expect(encreDe('Bravo')).toBe(ADVERSE)
    expect(encreDe('Muet')).toBe(NEUTRE)
  })

  it('vu d’une référence dont le film tait l’équipe, aucun nom n’a d’encre de camp', () => {
    afficher([ligne({ key: 'k', kill: kill('bid(44.0)', 'B', 'Bravo') })], 'M')
    expect(encreDe('Sandwolf')).toBe(NEUTRE)
    expect(encreDe('Bravo')).toBe(NEUTRE)
  })
})
