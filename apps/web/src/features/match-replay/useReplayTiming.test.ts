/**
 * useReplayTiming.test.ts — LES DURÉES DÉCLARÉES DU REJEU, et celle qui a été tranchée.
 *
 * `REPLAY_TIMING_MS` se disait « exposée pour les tests » sans qu'aucun test ne la lise : la
 * décision produit qu'elle porte n'était donc défendue par rien. Ce fichier lui donne le
 * garde-rail annoncé.
 *
 * LA CROIX DE MORT DURE 2,5 s, et c'est un arbitrage, pas un réglage libre. 1,5 s (valeur du
 * POC) rendait la mort invisible en lecture accélérée — à 4x, une croix vivait moins de dix
 * images. 4 s, l'autre durée proposée sur la planche, encombre la carte : un repère qui reste
 * à l'écran sans que rien ne l'y appelle. La valeur du milieu est celle qu'on livre, et la
 * changer doit rendre ce test rouge.
 */
import { renderHook } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { testReplayDoc } from './test/testDoc'
import { REPLAY_TIMING_MS, useReplayTiming } from './useReplayTiming'

describe('REPLAY_TIMING_MS — les durées déclarées', () => {
  it('la croix de mort dure 2,5 s : ni les 1,5 s du POC, ni les 4 s de la planche', () => {
    expect(REPLAY_TIMING_MS.death).toBe(2_500)
  })

  it('les autres durées du calque des joueurs sont celles réglées à l écran', () => {
    expect(REPLAY_TIMING_MS.trail).toBe(7_000)
    expect(REPLAY_TIMING_MS.aimHold).toBe(5_000)
    expect(REPLAY_TIMING_MS.spawn).toBe(800)
  })
})

describe('useReplayTiming — la conversion en images du document', () => {
  it('la croix de mort vaut bien 2,5 s d images sur la grille du document', () => {
    const doc = testReplayDoc()
    const { result } = renderHook(() => useReplayTiming(doc))
    const parSeconde = result.current.baseFps
    // La durée en images doit correspondre à 2,5 s de la cadence NATIVE du document : c'est
    // le seul lien qui garantit qu'une croix dure le même TEMPS quel que soit le film.
    expect(result.current.timing.death).toBe(Math.round((REPLAY_TIMING_MS.death / 1000) * parSeconde))
  })
})
