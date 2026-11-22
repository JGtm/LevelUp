/// <reference types="node" />
// @vitest-environment node
/**
 * departureLogic.test.ts — un départ n'est pas une mort.
 *
 * Un rejeu réel aminci (`test/departFixture.ts`) porte le cas : un joueur dont le corps est supprimé
 * sans lâcher ni compte à rebours, puis dont la présence se ferme. Les autres cas sont écrits à la main
 * sur un document court : une vraie mort suivie d'un départ, une fin de manche, un document sans
 * présence, un corps supprimé suivi d'une autre vie. Identifiants fictifs : aucun xuid réel.
 */
import { describe, expect, it } from 'vitest'

import type { KillEvent } from '@/features/match-view/_momentum'
import type { ReplayDocument } from '@/lib/api/types'
import { frameToMs } from '@/lib/replay/replayLogic'
import { buildPlayers, vitalityPresence } from '@/lib/replay/rosterLogic'

import { departDoc, departDocument, DEPART, PARTANT } from '../test/departFixture'
import { testReplayDoc, type TestReplayDocOverrides } from '../test/testDoc'
import { departuresOf, documentHorizonFrame, hasLeftAt, SURVIVOR_GRACE_MS } from './departureLogic'
import { reduceFeed } from '../hooks/useReplayTimeline'
import { alignFeedByOrigin, alignFeedToTracks, buildFeedEntries } from './killFeedLogic'
import { playerNowReadings } from './playerNowReadings'
import { replayClock } from './replayClock'

type Track = NonNullable<ReplayDocument['tracks']>[number]

/** Un frag de `killer` sur `victim` à l'instant `tMs` du film (aucune arme : seule la fin de vie compte). */
function frag(tMs: number, victim: string, killer = 'fx-depart-2'): KillEvent {
  return {
    tMs, xuid: killer, ally: true, teamID: 0, weaponKey: '', weaponLabel: '', weaponImageUrl: '', weaponTinted: false,
    assistState: '', assistGamertag: '', assistTeamID: null, killerDamagePct: null, assistDamagePct: null,
    victimXuid: victim, victimGamertag: '',
  }
}

/**
 * Les frags qui revendiquent chaque vie finie par un lâcher de commande du partant : seul son corps supprimé
 * reste sans frag. L'instant d'un frag est celui de la vie plus l'origine du film (le fil la retranche).
 */
function fragsOnReleasedLives(doc: ReturnType<typeof departDoc>): KillEvent[] {
  const clock = replayClock(doc)!
  return doc.tracks
    .filter((t) => t.xuid === PARTANT && t.endReason === 'released')
    .map((t) => frag(frameToMs(t.endFrame ?? 0, doc) + clock.originMs, PARTANT))
}

describe('le rejeu réel : un partant dont le corps est supprimé', () => {
  const doc = departDoc()

  it('le reconnaît comme un départ : dernière vie, corps supprimé, aucun compte à rebours, présence fermée avant la fin', () => {
    const d = departuresOf(doc).get(PARTANT)
    expect(d?.deletedBody).toBe(true)
    expect(d?.fromFrame).toBe(DEPART.corpsSupprime)
    expect([...departuresOf(doc).keys()]).toEqual([PARTANT])
  })

  it('il est parti de l’image qui suit la fin de sa vie jusqu’à la fin du document, au-delà de la fermeture de sa présence', () => {
    const left = (f: number) => hasLeftAt(departuresOf(doc), PARTANT, f)
    expect([DEPART.mort, 9700, DEPART.reapparition, DEPART.corpsSupprime].map(left)).toEqual([false, false, false, false])
    expect([DEPART.corpsSupprime + 1, 9900, DEPART.presenceFermee].map(left)).toEqual([true, true, true])
    expect([DEPART.presenceFermee + 1, 9940, 14_000, departDoc().frameCount - 1].map(left)).toEqual([true, true, true, true])
  })

  it('aucun autre joueur n’est parti : ni le bot, ni les présents jusqu’au bout', () => {
    expect(hasLeftAt(departuresOf(doc), 'bot:Robot Un [bot]', 9900)).toBe(false)
    expect(hasLeftAt(departuresOf(doc), 'fx-depart-2', 9900)).toBe(false)
  })

  it('sur les deux chemins d’alignement, sa fin de vie ne fait plus de ligne « mort »', () => {
    const kills = fragsOnReleasedLives(doc)
    const parOrigine = alignFeedByOrigin(kills, 0, doc, replayClock(doc)!)
    const parPistes = alignFeedToTracks(kills, 0, doc)
    for (const feed of [parOrigine, parPistes]) {
      expect(feed.deaths.filter((d) => d.xuid === PARTANT)).toEqual([])
    }
  })

  it('sans la règle (document sans présence), la même fin de vie reste une ligne « mort » non prouvée', () => {
    const sansPresence = departDoc({ roster: departDocument().roster?.map((r) => ({ ...r, presence: [] })) })
    expect(departuresOf(sansPresence).size).toBe(0)
    const kills = fragsOnReleasedLives(sansPresence)
    for (const feed of [
      alignFeedByOrigin(kills, 0, sansPresence, replayClock(sansPresence)!),
      alignFeedToTracks(kills, 0, sansPresence),
    ]) {
      const morts = feed.deaths.filter((d) => d.xuid === PARTANT)
      expect(morts.map((d) => [d.replayMs, d.proven])).toEqual([[frameToMs(DEPART.corpsSupprime, sansPresence), false]])
    }
  })

  it('le fil assemblé n’a aucune ligne « mort » pour lui, ni marque de mort sur sa piste de la frise', () => {
    const entries = buildFeedEntries(fragsOnReleasedLives(doc), [], 0, doc)
    expect(entries.filter((e) => e.death?.xuid === PARTANT)).toEqual([])
    const corps = frameToMs(DEPART.corpsSupprime, doc)
    expect(reduceFeed(entries, PARTANT).deaths.map((d) => d.replayMs)).not.toContain(corps)
  })

  it('la fiche dit « mort » avec son compte à rebours à 9700, puis « parti » aux images 9786, 9900, 9940 et à la dernière', () => {
    const players = buildPlayers(doc, [])
    const player = players.find((p) => p.xuid === PARTANT)!
    const presence = vitalityPresence(doc)
    const lire = (frame: number) => playerNowReadings({ player, doc, frame, presence, locale: 'fr' })
    const a9700 = lire(9700)
    expect([a9700.state.departed, a9700.state.respawnFrame, a9700.eliminated]).toEqual([false, DEPART.reapparition, true])
    for (const frame of [9786, 9900, 9940, doc.frameCount - 1]) {
      const r = lire(frame)
      expect([frame, r.state.alive, r.state.departed, r.eliminated]).toEqual([frame, false, true, true])
    }
  })
})

describe('les règles, sur un document court', () => {
  const STEP = 100
  const FRAMES = 1_000
  const countdown = (from: number) => ({ filmIndex: 0, from, to: from + 80, seconds: 8, startRead: true, endRead: true })
  const track = (over: Partial<Track>): Track => ({
    slot: 1, team: 0, xuid: 'X1', startFrame: 0, endFrame: 200, points: [{ t: 0, x: 0, y: 0 }, { t: 200, x: 1, y: 1 }],
    ...over,
  })
  // X2 est le témoin du chemin par les pistes : le frag qui lui revient fait mesurer le décalage (nul) du document.
  const temoin = track({ xuid: 'X2', slot: 5, endFrame: 500, points: [{ t: 0, x: 0, y: 0 }, { t: 500, x: 1, y: 1 }] })
  const spec = (over: Omit<TestReplayDocOverrides, 'tracks'> & { tracks?: Track[]; presenceEnd?: number | null }): TestReplayDocOverrides => {
    const { presenceEnd = 400, tracks = [track({ endReason: 'deleted' })], ...rest } = over
    return {
      frameCount: FRAMES,
      frameIntervalMs: STEP,
      originMs: 0,
      roster: [{ xuid: 'X1', filmIndex: 0, name: 'Un', team: 0, presence: presenceEnd === null ? [] : [{ from: 0, to: presenceEnd }] }],
      tracks: [...tracks, temoin],
      ...rest,
    }
  }
  const doc = (over: Parameters<typeof spec>[0] = {}) => testReplayDoc(spec(over))
  /** Le départ par corps supprimé : la fin de vie n'est pas une mort. */
  const departed = (d: ReturnType<typeof doc>) => departuresOf(d).get('X1')?.deletedBody === true
  /** « A quitté la partie » à une image, par la lecture de la fiche. */
  const leftAt = (d: ReturnType<typeof doc>, frame: number) => hasLeftAt(departuresOf(d), 'X1', frame)
  /** Les lignes « mort » de X1 par les deux chemins ; le frag sur X2 est le seul de la partie. */
  const deaths = (d: ReturnType<typeof doc>) => {
    const kills = [frag(50_000, 'X2', 'K')]
    const seuls = (l: ReturnType<typeof alignFeedByOrigin>['deaths']) => l.filter((x) => x.xuid === 'X1')
    return {
      origine: seuls(alignFeedByOrigin(kills, 0, d, replayClock(d)!).deaths),
      pistes: seuls(alignFeedToTracks(kills, 0, d).deaths),
    }
  }

  it('un corps supprimé, dernière vie, sans compte à rebours, présence fermée tôt : un départ, sans ligne « mort »', () => {
    const d = doc()
    expect(departed(d)).toBe(true)
    expect(deaths(d)).toEqual({ origine: [], pistes: [] })
  })

  it('une vraie mort suivie d’un départ (lâcher de commande) : pas un départ, la ligne « mort » reste', () => {
    const d = doc({ tracks: [track({ endReason: 'released', releaseFrame: 201 })] })
    expect(departed(d)).toBe(false)
    for (const l of Object.values(deaths(d))) expect(l.map((x) => [x.xuid, x.proven])).toEqual([['X1', true]])
  })

  it('une mort au compte à rebours lu, corps supprimé : pas un départ', () => {
    const d = doc({ respawnCountdowns: [countdown(202)] })
    expect(departed(d)).toBe(false)
    for (const l of Object.values(deaths(d))) expect(l.map((x) => [x.xuid, x.proven])).toEqual([['X1', true]])
  })

  it('un compte à rebours hors de la tolérance ne fait pas une mort : toujours un départ', () => {
    // La tolérance va de -66 à +200 ms autour de la fin de vie, soit de -0,66 à +2 images de 100 ms.
    expect(departed(doc({ respawnCountdowns: [countdown(202)] }))).toBe(false)
    expect(departed(doc({ respawnCountdowns: [countdown(203)] }))).toBe(true)
  })

  it('une fin de manche (corps supprimé, le joueur a une vie ensuite) : inchangée, ligne non prouvée', () => {
    const d = doc({
      tracks: [
        track({ endReason: 'deleted' }),
        track({ slot: 2, startFrame: 300, endFrame: 700, endReason: 'released', releaseFrame: 700, points: [{ t: 300, x: 0, y: 0 }, { t: 700, x: 1, y: 1 }] }),
      ],
    })
    expect(departed(d)).toBe(false)
    // La fin de la vie suivante (lâcher de commande) est une mort prouvée, comme avant.
    for (const l of Object.values(deaths(d))) expect(l.map((x) => [x.replayMs, x.proven])).toEqual([[20_000, false], [70_000, true]])
  })

  it('un document sans présence publiée : aucun départ, comportement actuel', () => {
    const d = doc({ presenceEnd: null })
    expect(departed(d)).toBe(false)
    for (const l of Object.values(deaths(d))) expect(l.map((x) => [x.xuid, x.proven])).toEqual([['X1', false]])
  })

  it('une dernière vie sans raison publiée : comportement actuel', () => {
    const d = doc({ tracks: [track({ endReason: undefined })] })
    expect(departed(d)).toBe(false)
    for (const l of Object.values(deaths(d))) expect(l).toHaveLength(1)
  })

  it('présent jusqu’au bout, ou jusque dans la marge de fin de partie : un survivant, pas un partant', () => {
    const limite = FRAMES - 1 - SURVIVOR_GRACE_MS / STEP
    expect(departed(doc({ presenceEnd: FRAMES - 1 }))).toBe(false)
    expect(departed(doc({ presenceEnd: limite }))).toBe(false)
    expect(departed(doc({ presenceEnd: limite - 1 }))).toBe(true)
  })

  it('une présence fermée 5 s avant la fin du document, corps supprimé : un départ (hors marge de 3 s)', () => {
    const cinqSecondes = FRAMES - 1 - 5_000 / STEP
    const d = doc({ presenceEnd: cinqSecondes })
    expect(departed(d)).toBe(true)
    expect(deaths(d)).toEqual({ origine: [], pistes: [] })
  })

  it('parti jusqu’à la fin : le corps supprimé, de l’image qui suit la fin de la vie à la dernière image', () => {
    const d = doc()
    expect([200, 201, 400, 401, FRAMES - 1].map((f) => leftAt(d, f))).toEqual([false, true, true, true, true])
  })

  it('un joueur mort puis parti : « mort » jusqu’à la fermeture de sa présence, « parti » après, sa ligne « mort » gardée', () => {
    const d = doc({ tracks: [track({ endReason: 'released', releaseFrame: 201 })] })
    expect([200, 201, 300, 400].map((f) => leftAt(d, f))).toEqual([false, false, false, false])
    expect([401, 500, FRAMES - 1].map((f) => leftAt(d, f))).toEqual([true, true, true])
    for (const l of Object.values(deaths(d))) expect(l.map((x) => [x.xuid, x.proven])).toEqual([['X1', true]])
  })

  it('un joueur parti puis revenu n’est jamais « parti » : par une vie, ou par une période de présence', () => {
    const vie = track({ slot: 2, startFrame: 600, endFrame: 800, endReason: 'released', releaseFrame: 800, points: [{ t: 600, x: 0, y: 0 }, { t: 800, x: 1, y: 1 }] })
    const reviens = doc({ tracks: [track({ endReason: 'released', releaseFrame: 201 }), vie] })
    expect([300, 450, 599, 700, 850, FRAMES - 1].map((f) => leftAt(reviens, f))).toEqual([false, false, false, false, false, false])
    // Une période de présence plus tard : il est mort, puis absent un temps, puis de retour — parti seulement après la dernière.
    const presence = testReplayDoc({
      ...spec({ tracks: [track({ endReason: 'released', releaseFrame: 201 })] }),
      roster: [{ xuid: 'X1', filmIndex: 0, name: 'Un', team: 0, presence: [{ from: 0, to: 400 }, { from: 600, to: 800 }] }],
    })
    expect([300, 450, 599, 700, 800].map((f) => leftAt(presence, f))).toEqual([false, false, false, false, false])
    expect([801, FRAMES - 1].map((f) => leftAt(presence, f))).toEqual([true, true])
  })

  it('une fermeture dans la marge de fin de partie : pas « parti » ; un document sans présence : comportement actuel', () => {
    const limite = FRAMES - 1 - SURVIVOR_GRACE_MS / STEP
    const dansLaMarge = doc({ presenceEnd: limite, tracks: [track({ endReason: 'released', releaseFrame: 201 })] })
    expect([limite + 1, FRAMES - 1].map((f) => leftAt(dansLaMarge, f))).toEqual([false, false])
    const sansPresence = doc({ presenceEnd: null })
    expect([201, 500, FRAMES - 1].map((f) => leftAt(sansPresence, f))).toEqual([false, false, false])
  })

  it('un bot n’est jamais un partant', () => {
    const d = testReplayDoc({
      ...spec({ tracks: [] }),
      roster: [{ xuid: '', bot: true, filmIndex: 1, name: 'Robot [bot]', presence: [{ from: 0, to: 400 }] }],
      tracks: [{ slot: 9, team: 0, bot: 'Robot [bot]', startFrame: 0, endFrame: 200, endReason: 'deleted', points: [{ t: 0, x: 0, y: 0 }] }],
    })
    expect(departuresOf(d).size).toBe(0)
    expect(hasLeftAt(departuresOf(d), 'bot:Robot [bot]', 500)).toBe(false)
  })

  it('l’horizon est la dernière image publiée', () => {
    expect(documentHorizonFrame(doc())).toBe(FRAMES - 1)
  })
})
