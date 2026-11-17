/**
 * Tests — les lignes d'objectif du fil : repliages (porteur, assistances de capture) et lignes
 * nouvelles (fin de manche, crâne, colline).
 *
 * Ce qu'ils protègent :
 *  1. « PORTEUR ÉLIMINÉ » n'est plus une ligne : c'est une mention dans l'infobulle de la ligne du frag
 *     du même tueur, que le filtre Objectifs retient ; sans frag apparié l'action garde sa ligne.
 *  2. LES ASSISTANCES DE CAPTURE nomment leurs crédités sur la ligne de la capture de leur équipe ;
 *     sans capture de leur équipe elles gardent leur ligne.
 *  3. LA FIN DE MANCHE écrit le score de manche de chaque camp, jamais le meneur du match, et
 *     seulement à plusieurs manches.
 *  4. LE CRÂNE (prise ; perte d'un portage fermé seulement) et LA COLLINE (début de tenue) sont des
 *     lignes d'objectif : visibles dans Tout et dans Objectifs.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'

import type { KillEvent } from '@/features/match-view/_momentum'
import { filmAllegianceOf } from '@/lib/replay/filmAllegiance'

import { buildFeedEntries } from '../model/killFeedLogic'
import { FEED_FILTER_KEY } from '../settings/replayPreferences'
import { testReplayDoc } from '../test/testDoc'
import { ReplayKillFeed } from './ReplayKillFeed'

vi.mock('@/lib/accessibility', () => ({
  resolveToken: (token: string) => `var(${token})`,
  tokenCssVar: (token: string) => `var(--ac-${token})`,
}))

const FRAME_MS = 100

const META = new Map([
  ['me', { gamertag: 'JGtm' }],
  ['mate', { gamertag: 'Ally02' }],
  ['foe', { gamertag: 'Cobra01' }],
])

const ROSTER = [
  { xuid: 'me', filmIndex: 0, name: 'JGtm', team: 0 },
  { xuid: 'mate', filmIndex: 1, name: 'Ally02', team: 0 },
  { xuid: 'foe', filmIndex: 2, name: 'Cobra01', team: 1 },
]

function killEvent(over: Partial<KillEvent>): KillEvent {
  return {
    tMs: 2_000, xuid: 'me', ally: true, teamID: 0, weaponKey: '', weaponLabel: '', weaponImageUrl: '',
    weaponTinted: false, assistState: '', assistGamertag: '', assistTeamID: null, killerDamagePct: null,
    assistDamagePct: null, victimXuid: 'foe', victimGamertag: 'Cobra01', ...over,
  }
}

const KILLS = [killEvent({ tMs: 2_000 }), killEvent({ tMs: 9_000, xuid: 'foe', victimXuid: 'me', victimGamertag: 'JGtm' })]

const act = (stat: string, t: number, xuid: string) => ({ stat, t, xuid, timeMs: t * FRAME_MS })

/** Deux manches : 100/78 (fin f50) puis 100/43 (fin f150). */
const ROUND_TEAMS = [
  {
    teamId: 0,
    rounds: [
      { round: 0, points: [{ t: 0, v: 0 }, { t: 50, v: 100 }] },
      { round: 1, points: [{ t: 100, v: 0 }, { t: 150, v: 100 }] },
    ],
    total: [{ t: 50, v: 100 }, { t: 150, v: 200 }],
  },
  {
    teamId: 1,
    rounds: [
      { round: 0, points: [{ t: 0, v: 0 }, { t: 50, v: 78 }] },
      { round: 1, points: [{ t: 100, v: 0 }, { t: 150, v: 43 }] },
    ],
    total: [{ t: 50, v: 78 }, { t: 150, v: 121 }],
  },
]

const DOC = testReplayDoc({
  frameIntervalMs: FRAME_MS,
  roster: ROSTER,
  objectives: [
    act('flag_carriers_killed', 20, 'me'),
    act('flag_carriers_killed', 120, 'me'),
    act('flag_captures', 60, 'me'),
    act('flag_capture_assists', 60, 'mate'),
    act('flag_capture_assists', 90, 'foe'),
  ] as never,
  skullCarries: [
    { xuid: 'mate', t0: 70, t1: 80, closed: true },
    { xuid: 'foe', t0: 85, t1: 140, closed: false },
  ] as never,
  scoreTimeline: {
    teams: ROUND_TEAMS,
    players: [],
  } as never,
  zoneStates: [
    {
      zoneRef: 0,
      spans: [
        { t0: 90, t1: 99, owner: null, active: true },
        { t0: 100, t1: 119, owner: 1, active: true },
        { t0: 120, t1: 129, owner: null, active: true },
        { t0: 130, t1: 160, owner: 1, active: true },
      ],
      gauge: [],
      gaugeRamps: [],
    },
  ] as never,
})

const ALLEGIANCE = filmAllegianceOf(DOC, [], 'me')

function renderFeed(locale: 'fr' | 'en' = 'fr') {
  return render(
    <ReplayKillFeed
      entries={buildFeedEntries(KILLS, [], 0, null)}
      nowMs={1_000_000}
      playWindow={null}
      scoreboard={[]}
      xuidMeta={META}
      allegiance={ALLEGIANCE}
      locale={locale}
      doc={DOC}
    />,
  )
}

const rowsText = () => screen.getAllByRole('listitem').map((r) => r.textContent ?? '')
/** Les lignes de frag dont l'infobulle dit que la victime portait le drapeau (fr ou en). */
const carrierRows = (hint: string) =>
  screen.getAllByRole('listitem').filter((r) => (r.getAttribute('title') ?? '').includes(hint))

function choose(name: RegExp) {
  fireEvent.click(screen.getByRole('button', { name: /Filtrer le fil|Filter the feed/ }))
  fireEvent.click(screen.getByRole('menuitemradio', { name }))
}

beforeEach(() => window.localStorage.removeItem(FEED_FILTER_KEY))
afterEach(() => window.localStorage.removeItem(FEED_FILTER_KEY))

describe('porteur éliminé : une mention en infobulle sur le frag, plus une ligne', () => {
  it('l’infobulle du frag le dit, et la ligne d’objectif correspondante disparaît', () => {
    renderFeed()
    const frag = carrierRows('La victime portait le drapeau')
    expect(frag).toHaveLength(1)
    expect(frag[0].textContent).toContain('Cobra01')
    expect(frag[0].textContent).toContain('JGtm')
    // La mention n'ajoute aucun texte à la ligne : le badge a quitté la rangée.
    expect(frag[0].textContent).not.toContain('Porteur')
    // Le second frag n'est pas celui d'un porteur : pas de mention.
    expect(rowsText().filter((r) => r.includes('Porteur'))).toHaveLength(0)
  })

  it('une action sans frag apparié garde sa ligne : on ne perd jamais une information', () => {
    renderFeed()
    expect(rowsText().filter((r) => r.includes('a éliminé un porteur de drapeau'))).toHaveLength(1)
  })

  it('le filtre Objectifs retient le frag du porteur', () => {
    renderFeed()
    choose(/Objectifs/)
    expect(carrierRows('La victime portait le drapeau')).toHaveLength(1)
  })

  it('en anglais, la mention et la phrase suivent la langue', () => {
    renderFeed('en')
    expect(carrierRows('The victim was carrying the flag')).toHaveLength(1)
    expect(rowsText().some((r) => r.includes('killed a flag carrier'))).toBe(true)
  })
})

describe('assistances de capture : sur la ligne de la capture', () => {
  it('la capture nomme les crédités de son équipe, et leur ligne propre disparaît', () => {
    renderFeed()
    const capture = rowsText().filter((r) => r.includes('a capturé le drapeau'))
    expect(capture).toHaveLength(1)
    expect(capture[0]).toContain('Ally02')
    expect(rowsText().filter((r) => r.includes('Ally02') && r.includes('a aidé'))).toHaveLength(0)
  })

  it('l’assistance d’un joueur de l’autre camp, sans capture de son équipe, garde sa ligne', () => {
    renderFeed()
    expect(rowsText().filter((r) => r.includes('Cobra01') && r.includes('a aidé à capturer le drapeau'))).toHaveLength(1)
  })
})

describe('fin de manche, crâne, colline : des lignes d’objectif', () => {
  it('chaque manche écrit le score de manche de chaque camp, jamais le total du match', () => {
    renderFeed()
    // Le fil se lit du plus récent au plus ancien : la manche 1 est la dernière rangée.
    const rounds = rowsText().filter((r) => r.includes('Fin de la manche')).reverse()
    expect(rounds).toHaveLength(2)
    expect(rounds[0]).toContain('Fin de la manche 1')
    expect(rounds[0]).toContain('100')
    expect(rounds[0]).toContain('78')
    expect(rounds[1]).toContain('43')
    expect(rounds.join(' ')).not.toContain('200')
    expect(rounds.join(' ')).not.toContain('121')
  })

  it('le crâne : une prise par portage, une perte pour le portage fermé seulement', () => {
    renderFeed()
    const rows = rowsText()
    expect(rows.filter((r) => r.includes('a pris le crâne'))).toHaveLength(2)
    expect(rows.filter((r) => r.includes('a perdu le crâne'))).toHaveLength(1)
    expect(rows.find((r) => r.includes('a perdu le crâne'))).toContain('Ally02')
  })

  it('la colline : une ligne à la prise, aucune à la reprise du même camp après un neutre', () => {
    renderFeed()
    expect(rowsText().filter((r) => r.includes('a pris la colline'))).toHaveLength(1)
  })

  it('toutes ces lignes sont de la famille Objectifs : le filtre les garde', () => {
    renderFeed()
    choose(/Objectifs/)
    const rows = screen.getAllByRole('listitem')
    const lines = rows.filter((r) => r.getAttribute('data-feed-family') === 'objectives')
    const text = lines.map((r) => r.textContent ?? '').join(' | ')
    expect(text).toContain('Fin de la manche 1')
    expect(text).toContain('a pris le crâne')
    expect(text).toContain('a pris la colline')
  })

  it('en anglais', () => {
    renderFeed('en')
    const text = rowsText().join(' | ')
    expect(text).toContain('Round 1 over')
    expect(text).toContain('picked up the skull')
    expect(text).toContain('lost the skull')
    expect(text).toContain('took the hill')
  })
})
