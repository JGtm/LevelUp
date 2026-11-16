/**
 * lifeEndProof.test.ts — la preuve qu'une fin de vie sans frag est une mort : lâcher de commande (S1) ou
 * début du compte à rebours de réapparition (S2), chacun à -2 / +6 images de la fin de piste (frame de 33 ms).
 * Identifiants fictifs : aucun xuid réel.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayDocument } from '@/lib/api/types'

import { testReplayDoc, type TestReplayDocOverrides } from '../test/testDoc'
import {
  countdownStartsByXuid,
  lifeEndProof,
  RESPAWN_COUNTDOWN_TOL_MS,
  UNIT_RELEASE_TOL_MS,
} from './lifeEndProof'

type Track = NonNullable<ReplayDocument['tracks']>[number]
type Countdown = NonNullable<ReplayDocument['respawnCountdowns']>[number]

const FIN = 300
const STEP = 33

function track(over: Partial<Track> = {}): Track {
  return {
    slot: 1,
    team: 0,
    xuid: 'X1',
    startFrame: 0,
    endFrame: FIN,
    points: [{ t: 0, x: 0, y: 0 }, { t: FIN, x: 1, y: 1 }],
    ...over,
  }
}

function countdown(from: number, filmIndex = 0): Countdown {
  return { filmIndex, from, to: from + 240, seconds: 8, startRead: true, endRead: true }
}

/** Le document : un joueur X1 (index 0), un X2 (index 1), un bot (index 8). */
function doc(over: TestReplayDocOverrides = {}) {
  return testReplayDoc({
    frameIntervalMs: STEP,
    roster: [
      { xuid: 'X1', filmIndex: 0, name: 'Un' },
      { xuid: 'X2', filmIndex: 1, name: 'Deux' },
      { xuid: '', bot: true, filmIndex: 8, name: 'Bot [bot]' },
    ],
    ...over,
  })
}

function proof(t: Track, d: ReturnType<typeof doc>) {
  const ready = testReplayDoc({ tracks: [t], frameIntervalMs: d.frameIntervalMs })
  return lifeEndProof(ready.tracks[0], d, countdownStartsByXuid(d).starts)
}

describe('lifeEndProof — S1, le lâcher de commande', () => {
  it('prouve la mort de -2 à +6 images de la fin de piste, aux deux bornes', () => {
    for (const delta of [-2, -1, 0, 3, 6]) {
      const t = track({ endReason: 'released', releaseFrame: FIN + delta })
      expect(proof(t, doc()), `delta ${delta}`).toBe('release')
    }
  })

  it('ne prouve rien hors de la tolérance : -3 et +7 images', () => {
    for (const delta of [-3, 7, 40]) {
      const t = track({ endReason: 'released', releaseFrame: FIN + delta })
      expect(proof(t, doc()), `delta ${delta}`).toBeNull()
    }
  })

  it('un corps supprimé encore commandé (deleted) sans compte à rebours : fin de manche, aucune preuve', () => {
    expect(proof(track({ endReason: 'deleted' }), doc())).toBeNull()
  })

  it('un lâcher sans releaseFrame ne prouve rien', () => {
    expect(proof(track({ endReason: 'released' }), doc())).toBeNull()
  })

  it('une frame de lâcher nulle est une lecture, pas une absence', () => {
    const t = track({ endFrame: 0, startFrame: 0, points: [{ t: 0, x: 0, y: 0 }], endReason: 'released', releaseFrame: 0 })
    expect(proof(t, doc())).toBe('release')
  })

  it('prouve par le lâcher le joueur absent du roster (S1 ne passe pas par le roster)', () => {
    const t = track({ xuid: 'INCONNU', endReason: 'released', releaseFrame: FIN + 1 })
    expect(proof(t, doc())).toBe('release')
  })
})

describe('lifeEndProof — S2, le compte à rebours de réapparition', () => {
  it('prouve la mort quand le compte du même joueur commence de -2 à +6 images de la fin', () => {
    for (const delta of [-2, 0, 3, 6]) {
      const d = doc({ respawnCountdowns: [countdown(FIN + delta)] })
      expect(proof(track(), d), `delta ${delta}`).toBe('countdown')
    }
  })

  it('ne prouve rien hors de la tolérance : -3 et +7 images', () => {
    for (const delta of [-3, 7]) {
      const d = doc({ respawnCountdowns: [countdown(FIN + delta)] })
      expect(proof(track(), d), `delta ${delta}`).toBeNull()
    }
  })

  it('le compte d’un autre joueur ne prouve rien', () => {
    const d = doc({ respawnCountdowns: [countdown(FIN, 1)] })
    expect(proof(track(), d)).toBeNull()
  })

  it('un compte plus ancien ou plus tardif du joueur ne masque pas celui qui tombe dans la tolérance', () => {
    const d = doc({ respawnCountdowns: [countdown(10), countdown(FIN + 2), countdown(900)] })
    expect(proof(track(), d)).toBe('countdown')
  })

  it('un index de film porté par deux entrées du roster : ses comptes ne servent pas, et sont comptés', () => {
    const d = doc({
      roster: [
        { xuid: 'X1', filmIndex: 0, name: 'Un' },
        { xuid: 'X9', filmIndex: 0, name: 'Autre' },
      ],
      respawnCountdowns: [countdown(FIN)],
    })
    expect(proof(track(), d)).toBeNull()
    expect(countdownStartsByXuid(d).unresolved).toBe(1)
  })

  it('un compte dont l’entrée est un bot, sans xuid ou absente du roster ne sert pas, et est compté', () => {
    const d = doc({ respawnCountdowns: [countdown(FIN, 8), countdown(FIN, 77)] })
    expect(proof(track(), d)).toBeNull()
    const { starts, unresolved } = countdownStartsByXuid(d)
    expect(starts.size).toBe(0)
    expect(unresolved).toBe(2)
  })

  it('un joueur sans index de film (absent du roster) n’a que S1', () => {
    const d = doc({ respawnCountdowns: [countdown(FIN)] })
    expect(proof(track({ xuid: 'INCONNU' }), d)).toBeNull()
    expect(proof(track({ xuid: 'INCONNU', endReason: 'released', releaseFrame: FIN }), d)).toBe('release')
  })
})

describe('lifeEndProof — S1 ou S2, anciens documents, cadence', () => {
  it('le lâcher l’emporte et le compte seul suffit : S1 ou S2', () => {
    const d = doc({ respawnCountdowns: [countdown(FIN)] })
    expect(proof(track({ endReason: 'released', releaseFrame: FIN }), d)).toBe('release')
    expect(proof(track({ endReason: 'released', releaseFrame: FIN + 40 }), d)).toBe('countdown')
    expect(proof(track({ endReason: 'deleted' }), d)).toBe('countdown')
  })

  it('un lâcher hors tolérance et un compte hors tolérance ne prouvent rien', () => {
    const d = doc({ respawnCountdowns: [countdown(FIN + 20)] })
    expect(proof(track({ endReason: 'released', releaseFrame: FIN + 20 }), d)).toBeNull()
  })

  it('un ancien document (ni endReason ni respawnCountdowns) ne prouve rien', () => {
    expect(proof(track(), doc())).toBeNull()
  })

  it('un document intermédiaire (comptes sans endReason) : S2 seul', () => {
    const d = doc({ respawnCountdowns: [countdown(FIN + 1)] })
    expect(proof(track(), d)).toBe('countdown')
    expect(proof(track({ endFrame: FIN + 500 }), d)).toBeNull()
  })

  it('un document avec endReason mais sans respawnCountdowns : S1 seul', () => {
    expect(proof(track({ endReason: 'released', releaseFrame: FIN + 2 }), doc())).toBe('release')
    expect(proof(track({ endReason: 'deleted' }), doc())).toBeNull()
  })

  it('la tolérance suit les millisecondes, pas les images : à 100 ms par image, 200 ms valent 2 images', () => {
    const d = doc({ frameIntervalMs: 100 })
    const tol = UNIT_RELEASE_TOL_MS.after
    expect(tol).toBe(RESPAWN_COUNTDOWN_TOL_MS.after)
    expect(proof(track({ endReason: 'released', releaseFrame: FIN + 2 }), d)).toBe('release')
    expect(proof(track({ endReason: 'released', releaseFrame: FIN + 3 }), d)).toBeNull()
    expect(proof(track({ endReason: 'released', releaseFrame: FIN - 1 }), d)).toBeNull()
    const dc = doc({ frameIntervalMs: 100, respawnCountdowns: [countdown(FIN + 2)] })
    expect(proof(track(), dc)).toBe('countdown')
  })

  it('la fin de piste est celle du rejeu : le dernier point quand endFrame manque', () => {
    const t = track({ endFrame: undefined, endReason: 'released', releaseFrame: FIN + 1 })
    expect(proof(t, doc())).toBe('release')
  })
})
