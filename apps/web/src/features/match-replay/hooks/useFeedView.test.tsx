/**
 * Tests — useFeedView : la liste du fil vue à l'instant lu, lue à l'identique par tout affichage.
 *
 * Ce qu'ils protègent : la coupe à l'instant lu (la plus récente en tête), les lignes d'événement du
 * document, le filtre (appliqué, retenu, replié sur Tout quand il n'est pas proposé), la ligne choisie
 * du magasin de lecture et le geste de clic (nul en lecture seule).
 */
import { act, renderHook } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it } from 'vitest'

import type { KillEvent } from '@/features/match-view/_momentum'
import { filmAllegianceOf } from '@/lib/replay/filmAllegiance'

import { buildFeedEntries } from '../model/killFeedLogic'
import { createPlaybackStore } from '../model/playbackStore'
import { FEED_FILTER_KEY } from '../settings/replayPreferences'
import { testReplayDoc } from '../test/testDoc'
import { useFeedView, type FeedViewInput } from './useFeedView'

const FRAME_MS = 100

function kill(over: Partial<KillEvent>): KillEvent {
  return {
    tMs: 1_000, xuid: 'me', ally: true, teamID: 0, weaponKey: 'br75', weaponLabel: 'BR75',
    weaponImageUrl: '/br75.png', weaponTinted: true, assistState: '', assistGamertag: '',
    assistTeamID: null, killerDamagePct: null, assistDamagePct: null, victimXuid: 'foe',
    victimGamertag: 'Cobra01', ...over,
  }
}

const KILLS = [
  kill({ tMs: 2_050 }),
  kill({ tMs: 9_000, xuid: 'foe', victimXuid: 'me', victimGamertag: 'JGtm' }),
]

const DOC = testReplayDoc({
  frameIntervalMs: FRAME_MS,
  roster: [
    { xuid: 'me', filmIndex: 0, name: 'JGtm', team: 0 },
    { xuid: 'foe', filmIndex: 1, name: 'Cobra01', team: 1 },
  ],
  // Une capture à 6 s : une ligne d'événement, entre les deux éliminations.
  objectives: [{ stat: 'flag_captures', t: 60, xuid: 'me', timeMs: 6_000 }] as never,
})
const ALLEGIANCE = filmAllegianceOf(DOC, [], 'me')
const ENTRIES = buildFeedEntries(KILLS, [], 0, null)

function vue(over: Partial<FeedViewInput> = {}) {
  return renderHook(
    (props: Partial<FeedViewInput>) =>
      useFeedView({ entries: ENTRIES, nowMs: 100_000, doc: DOC, locale: 'fr', allegiance: ALLEGIANCE, ...props }),
    { initialProps: over },
  )
}

beforeEach(() => window.localStorage.removeItem(FEED_FILTER_KEY))
afterEach(() => window.localStorage.removeItem(FEED_FILTER_KEY))

describe('la liste', () => {
  it('rend les éliminations ET les lignes d’événement du document, la plus récente en tête', () => {
    const { result } = vue()
    expect(result.current.filter).toBe('tout')
    expect(result.current.lines.map((l) => l.replayMs)).toEqual([9_000, 6_000, 2_050])
    expect(result.current.lines[1].event).toBeTruthy()
  })

  it('se coupe à l’instant lu et suit son avancée', () => {
    const { result, rerender } = vue({ nowMs: 3_000 })
    expect(result.current.lines.map((l) => l.replayMs)).toEqual([2_050])
    rerender({ nowMs: 7_000 })
    expect(result.current.lines.map((l) => l.replayMs)).toEqual([6_000, 2_050])
  })

  it('sans document : seulement le fil de la page, sans ligne d’événement', () => {
    const { result } = vue({ doc: null })
    expect(result.current.lines.map((l) => l.replayMs)).toEqual([9_000, 2_050])
    expect(result.current.select).toBeNull()
  })

  it('ne modifie jamais le fil de la page', () => {
    const avant = [...ENTRIES]
    vue()
    expect(ENTRIES).toEqual(avant)
  })

  it('deux affichages qui lisent le même fil lisent la même liste', () => {
    const a = vue()
    const b = vue()
    expect(a.result.current.lines.map((l) => l.key)).toEqual(b.result.current.lines.map((l) => l.key))
  })
})

describe('le filtre', () => {
  it('le geste change la liste, la retient dans le navigateur et propose ses options', () => {
    const { result } = vue()
    expect(result.current.options.map((o) => o.id)).toContain('kills')
    act(() => result.current.chooseFilter('kills'))
    expect(result.current.filter).toBe('kills')
    expect(result.current.lines.map((l) => l.replayMs)).toEqual([9_000, 2_050])
    expect(window.localStorage.getItem(FEED_FILTER_KEY)).toBe('kills')
  })

  it('un filtre retenu mais non proposé pour ce match se replie sur Tout, sans réécrire la préférence', () => {
    // Objectifs n'est proposé que si le match a une ligne d'objectif : sans document, il n'y en a aucune.
    window.localStorage.setItem(FEED_FILTER_KEY, 'objectives')
    const { result } = vue({ doc: null })
    expect(result.current.filter).toBe('tout')
    expect(result.current.options.map((o) => o.id)).not.toContain('objectives')
    expect(window.localStorage.getItem(FEED_FILTER_KEY)).toBe('objectives')
  })

  it('matchHasNone : vrai seulement quand le filtre n’a aucune ligne dans tout le match', () => {
    const plein = vue({ nowMs: 0 })
    expect(plein.result.current.lines).toEqual([])
    expect(plein.result.current.matchHasNone).toBe(false)
    const vide = vue({ entries: [], doc: null })
    expect(vide.result.current.matchHasNone).toBe(true)
  })
})

describe('la ligne choisie et le clic', () => {
  it('sans magasin de lecture, aucune ligne choisie et le geste est nul', () => {
    const { result } = vue()
    expect(result.current.selectedKey).toBeNull()
    expect(result.current.select).toBeNull()
  })

  it('le clic pose la mise en valeur puis demande le saut ; la ligne choisie suit le magasin', () => {
    const store = createPlaybackStore()
    const sauts: number[] = []
    store.onSeekRequest((f) => sauts.push(f))
    const { result } = vue({ playbackStore: store })
    const cible = result.current.lines.find((l) => l.kill && l.replayMs === 2_050)!
    expect(result.current.select).not.toBeNull()
    act(() => result.current.select!(cible))
    expect(sauts).toEqual([21])
    expect(store.focus()).toMatchObject({ frame: 21, killerXuid: 'me', victimXuid: 'foe' })
    expect(result.current.selectedKey).toBe(cible.key)
    act(() => store.clearFocus())
    expect(result.current.selectedKey).toBeNull()
  })

  it('une ligne qui n’est pas un frag ne demande rien', () => {
    const store = createPlaybackStore()
    const sauts: number[] = []
    store.onSeekRequest((f) => sauts.push(f))
    const { result } = vue({ playbackStore: store })
    const evenement = result.current.lines.find((l) => l.event)!
    act(() => result.current.select!(evenement))
    expect(sauts).toEqual([])
    expect(store.focus()).toBeNull()
  })
})
