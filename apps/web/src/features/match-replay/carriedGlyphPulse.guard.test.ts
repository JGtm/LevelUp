/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (CLAUDE.md n° 6) — UNE SEULE RESPIRATION POUR LES GLYPHES PORTÉS.
 *
 * POURQUOI CE GARDE (registre 2026-09-05, K4). Les trois calques qui posent un objet SUR son
 * porteur — la couronne du VIP, le crâne d'Oddball, la bombe d'Assaut — réécrivaient à
 * l'identique quatre constantes et une sinusoïde. Le risque n'était pas une désynchronisation
 * visible (les trois relèvent de modes distincts et ne co-occurrent jamais) mais la DÉRIVE :
 * un réglage d'oeil sur l'un, jamais reporté sur les deux autres, et trois respirations pour
 * un seul geste de lecture — que personne ne peut comparer puisqu'on ne les voit jamais
 * ensemble. C'est exactement le genre de divergence qu'aucun test de rendu n'attrape.
 *
 * CE QU'IL DÉTECTE : la sinusoïde, et la période de pulsation redéclarée.
 */
import { describe, expect, it } from 'vitest'
import { cheminCourt, fichierNomme, lire, nomDe, tousLesFichiers } from './test/featureFiles'

/** La respiration elle-même : la sinusoïde normalisée entre 0 et 1. */
const SINUSOIDE = /0\.5 \+ 0\.5 \* Math\.sin\(/

/** Sa période, redéclarée. */
const PERIODE = /PULSE_PERIOD_FRAMES\s*=/

const AUTORISES = new Set(['carriedGlyphPulse.ts', 'carriedGlyphPulse.guard.test.ts'])

function fautifs(motif: RegExp): string[] {
  return tousLesFichiers()
    .filter((f) => !AUTORISES.has(nomDe(f)))
    .filter((f) => motif.test(lire(f)))
    .map(cheminCourt)
}

describe('garde-rail : une seule pulsation de glyphe porté', () => {
  it('personne ne réécrit la sinusoïde de respiration', () => {
    expect(fautifs(SINUSOIDE)).toEqual([])
  })

  it('personne ne redéclare la période de pulsation', () => {
    expect(fautifs(PERIODE)).toEqual([])
  })

  it('et `carriedGlyphPulse` porte bien les deux — sans quoi ce test ne garderait rien', () => {
    const src = lire(fichierNomme('carriedGlyphPulse.ts'))
    expect(SINUSOIDE.test(src)).toBe(true)
    expect(PERIODE.test(src)).toBe(true)
  })
})
