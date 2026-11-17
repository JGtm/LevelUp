/**
 * Tests — les trois cadres de la page du rejeu (lot A, item A.1 du plan « refonte UI du rejeu »).
 *
 * Ce qu'ils protègent :
 *  1. L'ÉTAT DE DÉPART : `standard`, à chaque montage, sans mémoire.
 *  2. TOUTES LES TRANSITIONS, en table (3 cadres x 3 événements, export libre puis en cours),
 *     dont le retour au cadre d'où l'on vient quand on quitte le Plein cadre.
 *  3. LE MÊME OBJET quand l'événement n'a aucun effet.
 *  4. LES LECTURES que les items suivants branchent : navigation de l'application, barre d'une
 *     ligne, rangée de fiches.
 */
import { describe, expect, it } from 'vitest'

import {
  canPlaceCardRow,
  INITIAL_REPLAY_FRAME,
  reduceReplayFrame,
  showsAppNavigation,
  usesCompactBar,
  type ReplayFrame,
  type ReplayFrameEvent,
  type ReplayFrameOrigin,
  type ReplayFrameState,
} from './replayFrame'

const FREE = { exportActive: false }
const EXPORTING = { exportActive: true }

const STANDARD: ReplayFrameState = { frame: 'standard', returnTo: 'standard' }
const CINEMA: ReplayFrameState = { frame: 'cinema', returnTo: 'cinema' }
const FULL_FROM_STANDARD: ReplayFrameState = { frame: 'plein-cadre', returnTo: 'standard' }
const FULL_FROM_CINEMA: ReplayFrameState = { frame: 'plein-cadre', returnTo: 'cinema' }

const CINEMA_TOGGLE: ReplayFrameEvent = { type: 'toggleCinema' }
const FULL_TOGGLE: ReplayFrameEvent = { type: 'toggleFullFrame' }
const EXIT: ReplayFrameEvent = { type: 'exit' }

/** Un cas : l'état de départ, l'événement, et l'état attendu SANS export puis PENDANT un export. */
interface Row {
  name: string
  from: ReplayFrameState
  event: ReplayFrameEvent
  free: ReplayFrameState
  exporting: ReplayFrameState
}

const TABLE: Row[] = [
  // Bascule Cinéma
  { name: 'standard + Cinéma', from: STANDARD, event: CINEMA_TOGGLE, free: CINEMA, exporting: STANDARD },
  { name: 'cinema + Cinéma', from: CINEMA, event: CINEMA_TOGGLE, free: STANDARD, exporting: CINEMA },
  {
    name: 'plein-cadre (depuis standard) + Cinéma',
    from: FULL_FROM_STANDARD,
    event: CINEMA_TOGGLE,
    free: CINEMA,
    exporting: FULL_FROM_STANDARD,
  },
  {
    name: 'plein-cadre (depuis cinema) + Cinéma',
    from: FULL_FROM_CINEMA,
    event: CINEMA_TOGGLE,
    free: CINEMA,
    exporting: FULL_FROM_CINEMA,
  },
  // Bascule Plein cadre
  {
    name: 'standard + Plein cadre',
    from: STANDARD,
    event: FULL_TOGGLE,
    free: FULL_FROM_STANDARD,
    exporting: STANDARD,
  },
  { name: 'cinema + Plein cadre', from: CINEMA, event: FULL_TOGGLE, free: FULL_FROM_CINEMA, exporting: CINEMA },
  {
    name: 'plein-cadre (depuis standard) + Plein cadre',
    from: FULL_FROM_STANDARD,
    event: FULL_TOGGLE,
    free: STANDARD,
    exporting: FULL_FROM_STANDARD,
  },
  {
    name: 'plein-cadre (depuis cinema) + Plein cadre',
    from: FULL_FROM_CINEMA,
    event: FULL_TOGGLE,
    free: CINEMA,
    exporting: FULL_FROM_CINEMA,
  },
  // Sortie : jamais bloquée par l'export
  { name: 'standard + sortie', from: STANDARD, event: EXIT, free: STANDARD, exporting: STANDARD },
  { name: 'cinema + sortie', from: CINEMA, event: EXIT, free: CINEMA, exporting: CINEMA },
  {
    name: 'plein-cadre (depuis standard) + sortie',
    from: FULL_FROM_STANDARD,
    event: EXIT,
    free: STANDARD,
    exporting: STANDARD,
  },
  {
    name: 'plein-cadre (depuis cinema) + sortie',
    from: FULL_FROM_CINEMA,
    event: EXIT,
    free: CINEMA,
    exporting: CINEMA,
  },
]

describe('état de départ', () => {
  it('chaque montage naît en standard, sans cadre mémorisé', () => {
    expect(INITIAL_REPLAY_FRAME).toEqual(STANDARD)
  })
})

describe('reduceReplayFrame — table des transitions', () => {
  it.each(TABLE)('$name, export libre', ({ from, event, free }) => {
    expect(reduceReplayFrame(from, event, FREE)).toEqual(free)
  })

  it.each(TABLE)('$name, export en cours', ({ from, event, exporting }) => {
    expect(reduceReplayFrame(from, event, EXPORTING)).toEqual(exporting)
  })

  it('la table couvre les trois cadres (dont les deux origines du Plein cadre) et les trois événements', () => {
    expect(new Set(TABLE.map((r) => r.from.frame))).toEqual(new Set<ReplayFrame>(['standard', 'cinema', 'plein-cadre']))
    expect(new Set(TABLE.map((r) => r.from.returnTo))).toEqual(new Set<ReplayFrameOrigin>(['standard', 'cinema']))
    expect(new Set(TABLE.map((r) => r.event.type))).toEqual(new Set(['toggleCinema', 'toggleFullFrame', 'exit']))
  })
})

describe('reduceReplayFrame — sans effet, le même objet', () => {
  it('la sortie hors Plein cadre rend l\'état reçu', () => {
    expect(reduceReplayFrame(STANDARD, EXIT, FREE)).toBe(STANDARD)
    expect(reduceReplayFrame(CINEMA, EXIT, FREE)).toBe(CINEMA)
  })

  it('une bascule pendant un export rend l\'état reçu', () => {
    for (const state of [STANDARD, CINEMA, FULL_FROM_STANDARD, FULL_FROM_CINEMA]) {
      expect(reduceReplayFrame(state, CINEMA_TOGGLE, EXPORTING)).toBe(state)
      expect(reduceReplayFrame(state, FULL_TOGGLE, EXPORTING)).toBe(state)
    }
  })
})

describe('reduceReplayFrame — le cadre d\'où l\'on vient', () => {
  it('Standard, Plein cadre, bascule Plein cadre : retour au Standard', () => {
    const inFull = reduceReplayFrame(STANDARD, FULL_TOGGLE, FREE)
    expect(reduceReplayFrame(inFull, FULL_TOGGLE, FREE)).toEqual(STANDARD)
  })

  it('Standard, Cinéma, Plein cadre, sortie : retour au Cinéma, pas au Standard', () => {
    const cinema = reduceReplayFrame(STANDARD, CINEMA_TOGGLE, FREE)
    const inFull = reduceReplayFrame(cinema, FULL_TOGGLE, FREE)
    expect(reduceReplayFrame(inFull, EXIT, FREE)).toEqual(CINEMA)
  })

  it('Standard, Plein cadre, Cinéma : le Cinéma, puis la bascule Cinéma rend le Standard', () => {
    const inFull = reduceReplayFrame(STANDARD, FULL_TOGGLE, FREE)
    const cinema = reduceReplayFrame(inFull, CINEMA_TOGGLE, FREE)
    expect(cinema).toEqual(CINEMA)
    expect(reduceReplayFrame(cinema, CINEMA_TOGGLE, FREE)).toEqual(STANDARD)
  })

  it('un cadre hors Plein cadre n\'a qu\'une représentation : returnTo est le cadre lui-même', () => {
    const states = [STANDARD, CINEMA, FULL_FROM_STANDARD, FULL_FROM_CINEMA]
    const events = [CINEMA_TOGGLE, FULL_TOGGLE, EXIT]
    for (const state of states) {
      for (const event of events) {
        const next = reduceReplayFrame(state, event, FREE)
        if (next.frame !== 'plein-cadre') expect(next.returnTo).toBe(next.frame)
      }
    }
  })
})

describe('lectures des cadres', () => {
  it('la navigation de l\'application n\'est recouverte qu\'en Plein cadre', () => {
    expect(showsAppNavigation('standard')).toBe(true)
    expect(showsAppNavigation('cinema')).toBe(true)
    expect(showsAppNavigation('plein-cadre')).toBe(false)
  })

  it('la barre d\'une ligne sert le Cinéma et le Plein cadre, pas le Standard', () => {
    expect(usesCompactBar('standard')).toBe(false)
    expect(usesCompactBar('cinema')).toBe(true)
    expect(usesCompactBar('plein-cadre')).toBe(true)
  })

  it('la rangée de fiches peut se poser en Cinéma et en Plein cadre, pas en Standard', () => {
    expect(canPlaceCardRow('standard')).toBe(false)
    expect(canPlaceCardRow('cinema')).toBe(true)
    expect(canPlaceCardRow('plein-cadre')).toBe(true)
  })
})
