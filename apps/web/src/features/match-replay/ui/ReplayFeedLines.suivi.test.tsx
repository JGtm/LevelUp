/**
 * Tests — LE MARQUAGE DES LIGNES DU FIL QUI CONCERNENT LE JOUEUR SUIVI (`FeedLine`, `KillLine`, le fil posé sur le
 * plan et le fil de la colonne).
 *
 * Ce qu'ils protègent : chaque forme de ligne (frag, mort sans tueur, médaille seule, entrée ou sortie, objectif,
 * prise, véhicule, crâne) porte `data-followed` et le calque de fond léger quand le joueur suivi y figure, et jamais
 * sinon ; aucun joueur suivi, aucune ligne marquée ; le marquage se CUMULE avec la ligne choisie (`bg-muted`,
 * `data-selected`) et ne la remplace pas ; il n'y a AUCUN liseré ; la forme posée sur le plan et le fil de la colonne
 * le portent comme la ligne d'un fil ordinaire.
 */
import { cleanup, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'

import type { KillEvent } from '@/features/match-view/_momentum'
import { filmAllegianceOf } from '@/lib/replay/filmAllegiance'

import type { ReplayTeamCascades } from '../hooks/useTeamCascades'
import type { FeedEventLine } from '../model/feedEventLines'
import { FEED_FOLLOWED } from '../model/feedRow'
import { playerKeys } from '../model/feedFollowed'
import { buildFeedEntries, type ReplayFeedEntry } from '../model/killFeedLogic'
import { testReplayDoc } from '../test/testDoc'
import { ReplayFeedOverlay } from './ReplayFeedOverlay'
import { FeedLine } from './ReplayFeedLines'
import { ReplayKillFeed } from './ReplayKillFeed'

vi.mock('@/lib/accessibility', () => ({
  resolveToken: (token: string) => `var(${token})`,
  tokenCssVar: (token: string) => `var(--ac-${token})`,
}))

afterEach(cleanup)

const DOC = testReplayDoc({
  frameIntervalMs: 100,
  roster: [
    { xuid: 'me', filmIndex: 0, name: 'JGtm', team: 0 },
    { xuid: 'foe', filmIndex: 1, name: 'Cobra01', team: 1 },
    { xuid: 'pal', filmIndex: 2, name: 'Alpha', team: 0 },
  ],
})
const ALLEGIANCE = filmAllegianceOf(DOC, [], 'me')
const META = new Map([
  ['me', { gamertag: 'JGtm' }],
  ['foe', { gamertag: 'Cobra01' }],
  ['pal', { gamertag: 'Alpha' }],
])
const TEAMS: ReplayTeamCascades = { allyOf: () => null, labelOf: (id) => `Camp ${id}` }

const FOLLOWED_ME = playerKeys({ xuid: 'me' } as never)

const base: ReplayFeedEntry = { key: 'l', replayMs: 5_000, kill: null, medal: null, death: null }

function killEntry(xuid: string, victimXuid: string, key = 'kill'): ReplayFeedEntry {
  const victim = META.get(victimXuid)?.gamertag ?? ''
  return {
    ...base,
    key,
    kill: {
      tMs: 5_000, xuid, teamID: 0, weaponKey: '', weaponLabel: '', weaponImageUrl: '', weaponTinted: false,
      assistState: '', assistGamertag: '', assistTeamID: null, killerDamagePct: null, assistDamagePct: null,
      victimXuid, victimGamertag: victim, replayMs: 5_000, medals: [],
    } as never,
  }
}

const eventEntry = (line: unknown): ReplayFeedEntry => ({ ...base, key: 'event', event: line as FeedEventLine })

/** Rend UNE ligne du fil, dans sa liste, avec les clés du joueur suivi données. */
function ligne(entry: ReplayFeedEntry, followedKeys: ReadonlySet<string> | undefined, focus?: { selected: boolean }) {
  const { container } = render(
    <ul>
      <FeedLine
        entry={entry}
        inkOf={() => 'var(--ac-neutral)'}
        xuidMeta={META}
        marks={new Map()}
        locale="fr"
        teams={TEAMS}
        followedKeys={followedKeys}
        focus={focus ? { selected: focus.selected, onSelect: () => {} } : undefined}
      />
    </ul>,
  )
  const li = container.querySelector('li')
  if (!li) throw new Error('aucune ligne rendue')
  return li
}

const marquee = (li: HTMLElement) => li.getAttribute('data-followed') === 'true'
const calque = (li: HTMLElement) => FEED_FOLLOWED.split(' ').every((c) => li.classList.contains(c))

describe('chaque forme de ligne porte le marquage quand le joueur suivi y figure', () => {
  const cas: Array<[string, ReplayFeedEntry, ReplayFeedEntry]> = [
    ['frag (tueur)', killEntry('me', 'foe'), killEntry('pal', 'foe')],
    ['frag (victime)', killEntry('foe', 'me'), killEntry('foe', 'pal')],
    ['mort sans tueur', { ...base, death: { replayMs: 5_000, xuid: 'me', kind: '', img: '', tinted: false, proven: true } }, { ...base, death: { replayMs: 5_000, xuid: 'foe', kind: '', img: '', tinted: false, proven: true } }],
    ['médaille seule', { ...base, medal: { tMs: 5_000, xuid: 'me', gamertag: 'JGtm', name: 'M', label: 'M', description: '', imageUrl: '' } }, { ...base, medal: { tMs: 5_000, xuid: 'foe', gamertag: 'Cobra01', name: 'M', label: 'M', description: '', imageUrl: '' } }],
    ['entrée de partie', { ...base, presence: { kind: 'joined', xuid: 'me', name: 'JGtm', bot: false, source: 'api' } }, { ...base, presence: { kind: 'joined', xuid: 'foe', name: 'Cobra01', bot: false, source: 'api' } }],
    ['objectif', eventEntry({ family: 'objectives', xuid: 'me', stat: 'flag_captures' }), eventEntry({ family: 'objectives', xuid: 'foe', stat: 'flag_captures' })],
    ['prise', eventEntry({ family: 'pickups', xuid: 'me', name: 'BR75', nature: 'rack' }), eventEntry({ family: 'pickups', xuid: 'foe', name: 'BR75', nature: 'rack' })],
    ['véhicule', eventEntry({ family: 'vehicles', xuid: 'me', name: 'Warthog' }), eventEntry({ family: 'vehicles', xuid: 'foe', name: 'Warthog' })],
    ['crâne', eventEntry({ family: 'objectives', kind: 'skull', xuid: 'me', change: 'taken' }), eventEntry({ family: 'objectives', kind: 'skull', xuid: 'foe', change: 'taken' })],
  ]

  it.each(cas)('%s : marquée pour le joueur suivi, pas pour un autre', (_nom, concernee, autre) => {
    const li = ligne(concernee, FOLLOWED_ME)
    expect(marquee(li)).toBe(true)
    expect(calque(li)).toBe(true)
    cleanup()
    const autreLi = ligne(autre, FOLLOWED_ME)
    expect(marquee(autreLi)).toBe(false)
    expect(autreLi.hasAttribute('data-followed')).toBe(false)
    expect(calque(autreLi)).toBe(false)
  })

  it.each(cas)('%s : aucun joueur suivi, aucune ligne marquée', (_nom, concernee) => {
    expect(marquee(ligne(concernee, playerKeys(null)))).toBe(false)
    cleanup()
    expect(marquee(ligne(concernee, undefined))).toBe(false)
  })

  it('la fin de manche et la colline n\'ont pas de joueur : jamais marquées', () => {
    const manche = eventEntry({ family: 'objectives', kind: 'roundEnd', index: 1, scores: [{ teamId: 0, score: 1 }] })
    expect(marquee(ligne(manche, FOLLOWED_ME))).toBe(false)
    cleanup()
    expect(marquee(ligne(eventEntry({ family: 'objectives', kind: 'hill', teamId: 0 }), FOLLOWED_ME))).toBe(false)
  })
})

describe('un fond léger, jamais un liseré, et cumulable avec la ligne choisie', () => {
  it('le marquage est un calque `ring` sous le contenu : aucune bordure, aucune classe de bordure colorée', () => {
    const li = ligne(killEntry('me', 'foe'), FOLLOWED_ME)
    expect(FEED_FOLLOWED).toContain('before:bg-ring/15')
    expect(FEED_FOLLOWED).not.toMatch(/\bborder/)
    expect(li.className).not.toMatch(/\bborder/)
  })

  it('la ligne de frag CHOISIE et concernant le joueur suivi porte les deux marques à la fois', () => {
    const li = ligne(killEntry('me', 'foe'), FOLLOWED_ME, { selected: true })
    expect(li.getAttribute('data-selected')).toBe('true')
    expect(li.classList.contains('bg-muted')).toBe(true)
    expect(marquee(li)).toBe(true)
    expect(calque(li)).toBe(true)
  })

  it('choisie sans concerner le joueur suivi : `bg-muted` seul ; concernant sans être choisie : le calque seul', () => {
    const choisie = ligne(killEntry('pal', 'foe'), FOLLOWED_ME, { selected: true })
    expect(choisie.classList.contains('bg-muted')).toBe(true)
    expect(marquee(choisie)).toBe(false)
    cleanup()
    const suivie = ligne(killEntry('me', 'foe'), FOLLOWED_ME, { selected: false })
    expect(suivie.classList.contains('bg-muted')).toBe(false)
    expect(suivie.hasAttribute('data-selected')).toBe(false)
    expect(marquee(suivie)).toBe(true)
  })

  it('une ligne de frag cliquable reste un bouton à état quand elle est marquée', () => {
    ligne(killEntry('me', 'foe'), FOLLOWED_ME, { selected: false })
    expect(screen.getByRole('button', { pressed: false })).toBeTruthy()
  })
})

describe('les trois formes du fil', () => {
  const KILLS: KillEvent[] = [
    {
      tMs: 2_000, xuid: 'me', ally: true, teamID: 0, weaponKey: '', weaponLabel: '', weaponImageUrl: '', weaponTinted: false,
      assistState: '', assistGamertag: '', assistTeamID: null, killerDamagePct: null, assistDamagePct: null,
      victimXuid: 'foe', victimGamertag: 'Cobra01',
    },
    {
      tMs: 4_000, xuid: 'pal', ally: true, teamID: 0, weaponKey: '', weaponLabel: '', weaponImageUrl: '', weaponTinted: false,
      assistState: '', assistGamertag: '', assistTeamID: null, killerDamagePct: null, assistDamagePct: null,
      victimXuid: 'foe', victimGamertag: 'Cobra01',
    },
  ]
  const entries = buildFeedEntries(KILLS, [], 0, null)

  it('la colonne : seules les lignes du joueur suivi sont marquées, la liste reste dans le même ordre', () => {
    const { container } = render(
      <ReplayKillFeed
        entries={entries} nowMs={100_000} playWindow={null} scoreboard={[]} xuidMeta={META}
        allegiance={ALLEGIANCE} locale="fr" followedKeys={FOLLOWED_ME}
      />,
    )
    const lignes = [...container.querySelectorAll<HTMLElement>('li')]
    expect(lignes).toHaveLength(2)
    // La plus récente en tête : Alpha (non suivi), puis JGtm (suivi).
    expect(lignes.map(marquee)).toEqual([false, true])
  })

  it('la colonne sans joueur suivi : aucune ligne marquée', () => {
    const { container } = render(
      <ReplayKillFeed
        entries={entries} nowMs={100_000} playWindow={null} scoreboard={[]} xuidMeta={META}
        allegiance={ALLEGIANCE} locale="fr"
      />,
    )
    expect(container.querySelectorAll('[data-followed]')).toHaveLength(0)
  })

  it('le fil posé sur le plan : la ligne nue du joueur suivi est marquée, estompée comme les autres', () => {
    const { container } = render(
      <ReplayFeedOverlay
        entries={entries} nowMs={100_000} playWindow={null} xuidMeta={META} allegiance={ALLEGIANCE} locale="fr"
        followedKeys={FOLLOWED_ME}
      />,
    )
    const lignes = [...container.querySelectorAll<HTMLElement>('li')]
    // La plus récente EN BAS : JGtm (suivi), puis Alpha, dans l'ordre du temps.
    expect(lignes.map(marquee)).toEqual([true, false])
    expect(calque(lignes[0])).toBe(true)
    expect(lignes[0].style.opacity).toBe('1')
  })

  it('le fil posé sur le plan sans joueur suivi : aucune ligne marquée', () => {
    const { container } = render(
      <ReplayFeedOverlay entries={entries} nowMs={100_000} playWindow={null} xuidMeta={META} allegiance={ALLEGIANCE} locale="fr" />,
    )
    expect(container.querySelectorAll('[data-followed]')).toHaveLength(0)
  })
})
