/**
 * tacticalReplayInstant.test.ts — le lien tactique (`?t=&clock=`) : de l'instant reçu du contrat
 * `TacticalContribution` à la frame que `ReplayCanvas.openAtFrame` consomme. Extrait de
 * `replayLogic.test.ts` (seuil des 500 lignes du dépôt) — la fonction testée vit dans
 * `replayLogic.ts`, aux côtés de `msToFrames` dont elle se sert.
 *
 * Les valeurs viennent d'un artefact réel (`data/cache/replays/halo_infinite/01e1f945.json`) :
 * originMs 12 588, frameIntervalMs 100, frameCount 5 343. L'horloge `match` se convertit par
 * l'origine du film (`t − originMs`), jamais par `deathOffsetMs` (un décalage de l'horloge moteur,
 * de l'ordre de 6 592 057 ms sur cet artefact : il aurait rejeté l'ouverture à la dernière image).
 */
import { describe, expect, it } from 'vitest'

import { OUVERTURE_AVANT_EVENEMENT_MS, resolveTacticalOpenAtFrame } from './replayLogic'
import { testReplayDoc as makeDoc } from '../../features/match-replay/test/testDoc'

describe('resolveTacticalOpenAtFrame — le lien tactique, du texte de l’URL à la frame', () => {
  const doc = makeDoc({ frameIntervalMs: 100, frameCount: 5343, originMs: 12_588 })
  const docSansOrigine = makeDoc({ frameIntervalMs: 100, frameCount: 5343 })

  it('l’ouverture se place 5 s avant l’événement', () => {
    expect(OUVERTURE_AVANT_EVENEMENT_MS).toBe(5000)
  })

  it('pas de lien (t ou clock absent) : rien à ouvrir, aucun avis', () => {
    expect(resolveTacticalOpenAtFrame({}, doc)).toEqual({ openAtFrame: null, showMissingOriginNotice: false })
    expect(resolveTacticalOpenAtFrame({ t: '7000' }, doc)).toEqual({ openAtFrame: null, showMissingOriginNotice: false })
  })

  it('document pas encore chargé : rien à ouvrir, PAS un avis (la page charge, elle n’est pas en défaut)', () => {
    expect(resolveTacticalOpenAtFrame({ t: '7000', clock: 'match' }, undefined)).toEqual({
      openAtFrame: null,
      showMissingOriginNotice: false,
    })
  })

  it('clock=match : film = t − originMs (60 000 − 12 588 = 47 412 ms), puis 5 s de recul -> frame 424', () => {
    // Sans recul ce serait la frame 474 ; le recul de 5 s la ramène à (47 412 − 5 000) / 100 = 424.
    expect(resolveTacticalOpenAtFrame({ t: '60000', clock: 'match' }, doc)).toEqual({
      openAtFrame: 424,
      showMissingOriginNotice: false,
    })
  })

  it('clock=film : t est déjà sur l’axe du film, seul le recul de 5 s s’applique', () => {
    // (7000 − 5000) / 100 = 20 ; l’origine n’intervient pas.
    expect(resolveTacticalOpenAtFrame({ t: '7000', clock: 'film' }, doc)).toEqual({
      openAtFrame: 20,
      showMissingOriginNotice: false,
    })
    expect(resolveTacticalOpenAtFrame({ t: '7000', clock: 'film' }, docSansOrigine)).toEqual({
      openAtFrame: 20,
      showMissingOriginNotice: false,
    })
  })

  it('clock=match sans origine du film : aucune frame, avis affiché (jamais un saut approximatif)', () => {
    expect(resolveTacticalOpenAtFrame({ t: '60000', clock: 'match' }, docSansOrigine)).toEqual({
      openAtFrame: null,
      showMissingOriginNotice: true,
    })
  })

  it('un instant à moins de 5 s du début du film : plancher à la frame 0', () => {
    // match 14 000 -> film 1 412 ms ; film 3 000 ms : le recul passerait sous zéro.
    expect(resolveTacticalOpenAtFrame({ t: '14000', clock: 'match' }, doc).openAtFrame).toBe(0)
    expect(resolveTacticalOpenAtFrame({ t: '3000', clock: 'film' }, doc).openAtFrame).toBe(0)
  })

  it('un `t` non numérique (lien corrompu) : rien à ouvrir, aucun avis', () => {
    expect(resolveTacticalOpenAtFrame({ t: 'pas-un-nombre', clock: 'match' }, doc)).toEqual({
      openAtFrame: null,
      showMissingOriginNotice: false,
    })
  })
})
