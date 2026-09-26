/**
 * tacticalReplayInstant.test.ts — le lien tactique (`?t=&clock=`, lot M1b, 2026-09-08) : de
 * l'instant reçu du contrat `TacticalContribution` à la frame que `ReplayCanvas.openAtFrame`
 * consomme. Extrait de `replayLogic.test.ts` (seuil des 500 lignes du dépôt) — les fonctions
 * testées vivent toujours dans `replayLogic.ts`, aux côtés de `frameToMs`/`msToFrames` dont
 * elles se servent.
 */
import { describe, expect, it } from 'vitest'

import { resolveTacticalOpenAtFrame, resolveTacticalReplayInstant } from './replayLogic'
import { testReplayDoc as makeDoc } from '../../features/match-replay/test/testDoc'
import type { ReplayDocument } from '@/lib/api/types'

describe('resolveTacticalReplayInstant — lien tactique -> instant sur l’horloge du film', () => {
  it('clock=film : aucune conversion, t est déjà sur l’axe du film', () => {
    expect(resolveTacticalReplayInstant({ t: 4200, clock: 'film' }, null)).toEqual({
      status: 'ms',
      ms: 4200,
    })
    // Même absence de calage : « film » n'en a jamais eu besoin.
    expect(resolveTacticalReplayInstant({ t: 4200, clock: 'film' }, undefined)).toEqual({
      status: 'ms',
      ms: 4200,
    })
  })

  it('clock=match, calage connu : horlogeFilm = horlogeMatch + deathOffsetMs', () => {
    expect(resolveTacticalReplayInstant({ t: 7000, clock: 'match' }, 3600)).toEqual({
      status: 'ms',
      ms: 10_600,
    })
  })

  it('clock=match, calage connu et NÉGATIF : additionné quand même (pas un repli)', () => {
    expect(resolveTacticalReplayInstant({ t: 7000, clock: 'match' }, -500)).toEqual({
      status: 'ms',
      ms: 6500,
    })
  })

  it('clock=match, calage MESURÉ À ZÉRO : une valeur, pas une absence', () => {
    expect(resolveTacticalReplayInstant({ t: 7000, clock: 'match' }, 0)).toEqual({
      status: 'ms',
      ms: 7000,
    })
  })

  it('clock=match, calage inconnu (null ou undefined) : refuse plutôt que de deviner', () => {
    expect(resolveTacticalReplayInstant({ t: 7000, clock: 'match' }, null)).toEqual({
      status: 'unknown-offset',
    })
    expect(resolveTacticalReplayInstant({ t: 7000, clock: 'match' }, undefined)).toEqual({
      status: 'unknown-offset',
    })
  })
})

/** Coverage minimale, valide, pour un document de test — seul `deathOffsetMs` varie entre
 *  les cas ; le reste n'a aucune incidence sur `resolveTacticalOpenAtFrame`. */
function coverageFixture(deathOffsetMs?: number): NonNullable<ReplayDocument['coverage']> {
  const layer = { available: 0, attached: 0, noSlot: 0, ambiguous: 0, outOfWindow: 0, unpublished: 0 }
  return {
    shots: layer,
    grenades: layer,
    objectives: layer,
    originResolved: true,
    bridge: {
      slots: 0,
      fromReading: 0,
      livesNamed: 0,
      livesTotal: 0,
      indexReadings: 0,
      indexDisagreements: 0,
      slotCollisions: 0,
      namedByPreviousLife: 0,
      namedByNextLife: 0,
      namedBySlotBridge: 0,
      unnamedLives: 0,
      unnamedLivesContested: 0,
      deathOffsetMatched: deathOffsetMs != null ? 1 : 0,
      deathOffsetRunnerUp: 0,
      deathOffsetMs,
      closedByShot: 0,
      closedByRespawn: 0,
      closedContested: 0,
      closedRefused: 0,
      // Lien direct corps ↔ joueur (schéma 50 amendé, lot E2) : sans incidence ici non plus,
      // mais le type les exige — la fixture doit rester un document VALIDE, pas un fragment.
      concordant: 0,
      discordant: 0,
      bridgeNamedLives: 0,
      directByCreation: 0,
      directByCreationPropagated: 0,
      bodiesWithCreation: 0,
    },
  }
}

describe('resolveTacticalOpenAtFrame — le lien tactique, du texte de l’URL à la frame', () => {
  const doc = makeDoc({ frameIntervalMs: 100, coverage: coverageFixture(3600) })
  const docSansCalage = makeDoc({ frameIntervalMs: 100, coverage: coverageFixture(undefined) })

  it('pas de lien (t ou clock absent) : rien à ouvrir, aucun avis', () => {
    expect(resolveTacticalOpenAtFrame({}, doc)).toEqual({
      openAtFrame: null,
      showUncalibratedNotice: false,
    })
    expect(resolveTacticalOpenAtFrame({ t: '7000' }, doc)).toEqual({
      openAtFrame: null,
      showUncalibratedNotice: false,
    })
  })

  it('document pas encore chargé : rien à ouvrir, PAS un avis (la page charge, elle n’est pas en défaut)', () => {
    expect(resolveTacticalOpenAtFrame({ t: '7000', clock: 'match' }, undefined)).toEqual({
      openAtFrame: null,
      showUncalibratedNotice: false,
    })
  })

  it('clock=match, calage CONNU : frame attendue = (t + deathOffsetMs) / frameIntervalMs', () => {
    // (7000 + 3600) / 100 = 106.
    expect(resolveTacticalOpenAtFrame({ t: '7000', clock: 'match' }, doc)).toEqual({
      openAtFrame: 106,
      showUncalibratedNotice: false,
    })
  })

  it('clock=match, calage INCONNU : aucune frame, avis affiché (jamais un saut approximatif)', () => {
    expect(resolveTacticalOpenAtFrame({ t: '7000', clock: 'match' }, docSansCalage)).toEqual({
      openAtFrame: null,
      showUncalibratedNotice: true,
    })
  })

  it('clock=film : frame DIRECTE, aucun offset appliqué même si le document en publie un', () => {
    // 7000 / 100 = 70, PAS 106 : le calage de match ne concerne pas l'horloge du film.
    expect(resolveTacticalOpenAtFrame({ t: '7000', clock: 'film' }, doc)).toEqual({
      openAtFrame: 70,
      showUncalibratedNotice: false,
    })
  })

  it('un `t` non numérique (lien corrompu) : rien à ouvrir, aucun avis', () => {
    expect(resolveTacticalOpenAtFrame({ t: 'pas-un-nombre', clock: 'match' }, doc)).toEqual({
      openAtFrame: null,
      showUncalibratedNotice: false,
    })
  })
})
