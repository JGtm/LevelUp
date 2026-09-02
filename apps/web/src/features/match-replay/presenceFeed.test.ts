/**
 * presenceFeed.test.ts — l'entrée/sortie de partie, dérivée des bornes de vie.
 *
 * Ce que ces tests verrouillent : les DEUX seuls signaux émis (première vie tardive,
 * dernière vie précoce), le SILENCE sur tout le reste — joueur ordinaire, trous entre
 * deux vies (manches d'élimination !), fenêtre inconnue, joueur innommable — et la
 * fusion triée dans le fil.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayFeedEntry } from './killFeedLogic'
import { mergeFeedWithPresence, presenceEntries } from './presenceFeed'
import type { ReplayDocumentReady, ReplayTrackReady } from './replayNormalize'
import type { ReplayWindowBounds } from './replayWindow'
import type { ReplayPlayer } from './rosterLogic'

/** Un document 10 Hz : frameToMs(t) = t * 100 ms. */
const DOC = { frameIntervalMs: 100, frameCount: 3_000 } as ReplayDocumentReady

/** Fenêtre de gameplay : de 10 s à 250 s (préambule de lecture 1 s avant, cadence 100 ms). */
const WINDOW: ReplayWindowBounds = {
  startFrame: 100,
  leadInFrame: 90,
  endFrame: 2_500,
  startMs: 10_000,
  endMs: 250_000,
}

function life(slot: number, start: number, end: number): ReplayTrackReady {
  return {
    slot,
    team: -1,
    startFrame: start,
    endFrame: end,
    points: [
      { t: start, x: 0, y: 0 },
      { t: end, x: 1, y: 1 },
    ],
  }
}

function player(xuid: string, lives: ReplayTrackReady[], over: Partial<ReplayPlayer> = {}): ReplayPlayer {
  return { xuid, filmName: xuid, lives, ...over }
}

describe('presenceEntries — ce qui parle, et ce qui se tait', () => {
  it('une PREMIÈRE vie tardive est une entrée en partie, datée à son début', () => {
    const [e] = presenceEntries([player('A', [life(600, 900, 2_500)])], WINDOW, DOC)
    expect(e.presence).toMatchObject({ kind: 'joined', xuid: 'A', name: 'A' })
    expect(e.replayMs).toBe(90_000)
  })

  it('une DERNIÈRE vie précoce dit « ne reviendra plus », datée à sa fin', () => {
    const [e] = presenceEntries([player('A', [life(512, 100, 1_200)])], WINDOW, DOC)
    expect(e.presence).toMatchObject({ kind: 'left', xuid: 'A' })
    expect(e.replayMs).toBe(120_000)
  })

  it('un joueur présent de bout en bout ne fait AUCUNE ligne', () => {
    expect(presenceEntries([player('A', [life(512, 100, 2_480)])], WINDOW, DOC)).toEqual([])
  })

  it("un TROU entre deux vies n'émet rien : rester mort entre deux manches est normal", () => {
    // 60 s sans vie au milieu — un mode à élimination en produit à chaque manche.
    const p = player('A', [life(512, 100, 1_000), life(700, 1_600, 2_480)])
    expect(presenceEntries([p], WINDOW, DOC)).toEqual([])
  })

  it('sans fenêtre de gameplay établie, rien ne se déduit', () => {
    expect(presenceEntries([player('A', [life(512, 900, 1_200)])], null, DOC)).toEqual([])
  })

  it('un bot porte son nom (suffixe de donnée retiré) et son drapeau ; un joueur innommable se tait', () => {
    const bot = player('bot:343 Oscar [bot]', [life(600, 900, 2_480)], {
      bot: true,
      filmName: '343 Oscar [bot]',
    })
    const anonyme = { xuid: 'X', lives: [life(601, 900, 2_480)] } as ReplayPlayer
    const out = presenceEntries([bot, anonyme], WINDOW, DOC)
    expect(out).toHaveLength(1)
    // Le film écrit le gamertag suffixé (marqueur de donnée, cf. killsource/roster.go) ;
    // la ligne de présence, elle, ne le répète pas (retour user 2026-09-02, lot D).
    expect(out[0].presence).toMatchObject({ kind: 'joined', name: '343 Oscar', bot: true })
  })
})

describe('presenceEntries — la PARTICIPATION API prime sur la dérivation film', () => {
  // Match démarré à minuit UTC ; artefact calé 4 s plus tard (originMs).
  const HEADER = { start_time: '2026-09-01T00:00:00Z' }
  const DOC_ORIGIN = { ...DOC, originMs: 4_000 } as ReplayDocumentReady
  const board = (over: Record<string, unknown>) =>
    ({ xuid: 'A', gamertag: 'Alpha', ...over }) as ReplayPlayer['board']

  it("un drapeau joined_in_progress pose l'entrée à l'horodatage API, recalé comme les médias", () => {
    // Rejoint à 00:01:40 → (100 000 − 0) − 4 000 = 96 000 ms sur l'axe du rejeu.
    const p = player('A', [life(600, 900, 2_400)], {
      board: board({
        joined_in_progress: true,
        first_joined_time: '2026-09-01T00:01:40Z',
        left_in_progress: false,
      }),
    })
    const [e] = presenceEntries([p], WINDOW, DOC_ORIGIN, HEADER)
    expect(e.presence).toMatchObject({ kind: 'joined', source: 'api', name: 'Alpha' })
    expect(e.replayMs).toBe(96_000)
  })

  it('left_in_progress pose la sortie API ; les DEUX drapeaux peuvent parler', () => {
    const p = player('A', [life(600, 900, 1_200)], {
      board: board({
        joined_in_progress: true,
        first_joined_time: '2026-09-01T00:01:40Z',
        left_in_progress: true,
        last_leave_time: '2026-09-01T00:03:20Z',
      }),
    })
    const out = presenceEntries([p], WINDOW, DOC_ORIGIN, HEADER)
    expect(out.map((e) => e.presence?.kind)).toEqual(['joined', 'left'])
    expect(out[1].replayMs).toBe(196_000)
    expect(out[1].presence?.source).toBe('api')
  })

  it("des drapeaux à FALSE font TAIRE le joueur : l'API affirme, le film ne la contredit pas", () => {
    // Les bornes de vie déclencheraient les deux lignes du repli — l'API dit non.
    const p = player('A', [life(600, 900, 1_200)], {
      board: board({ joined_in_progress: false, left_in_progress: false }),
    })
    expect(presenceEntries([p], WINDOW, DOC_ORIGIN, HEADER)).toEqual([])
  })

  it("sans drapeaux (colonnes NULL) ou sans en-tête, le REPLI film reprend la main", () => {
    const sansDrapeaux = player('A', [life(600, 900, 2_400)], { board: board({}) })
    const [e] = presenceEntries([sansDrapeaux], WINDOW, DOC_ORIGIN, HEADER)
    expect(e.presence).toMatchObject({ kind: 'joined', source: 'film' })
    const avecDrapeaux = player('A', [life(600, 900, 2_400)], {
      board: board({ joined_in_progress: true, first_joined_time: '2026-09-01T00:01:40Z' }),
    })
    const [f] = presenceEntries([avecDrapeaux], WINDOW, DOC_ORIGIN, null)
    expect(f.presence?.source).toBe('film')
  })

  it("un drapeau vrai SANS horodatage lisible ne pose rien : pas d'instant inventé", () => {
    const p = player('A', [life(600, 100, 2_480)], {
      board: board({ joined_in_progress: true, left_in_progress: false }),
    })
    expect(presenceEntries([p], WINDOW, DOC_ORIGIN, HEADER)).toEqual([])
  })

  it("l'instant API est borné à la fenêtre de gameplay", () => {
    const p = player('A', [life(600, 100, 2_480)], {
      board: board({
        joined_in_progress: true,
        first_joined_time: '2026-09-01T00:00:01Z', // avant le coup d'envoi recalé
        left_in_progress: false,
      }),
    })
    const [e] = presenceEntries([p], WINDOW, DOC_ORIGIN, HEADER)
    expect(e.replayMs).toBe(WINDOW.startMs)
  })
})

describe('mergeFeedWithPresence — un seul axe de temps', () => {
  it('fusionne et retrie ; sans présence, le fil ressort tel quel', () => {
    const feedLine = (key: string, replayMs: number): ReplayFeedEntry => ({
      key,
      replayMs,
      kill: null,
      medal: null,
      death: null,
    })
    const presence = presenceEntries([player('A', [life(600, 900, 2_500)])], WINDOW, DOC)
    const merged = mergeFeedWithPresence([feedLine('k1', 80_000), feedLine('k2', 100_000)], presence)
    expect(merged.map((e) => e.key)).toEqual(['k1', presence[0].key, 'k2'])
    expect(mergeFeedWithPresence([feedLine('k1', 1)], []).map((e) => e.key)).toEqual(['k1'])
  })
})
