/**
 * weaponSoundLogic.test.ts — LE CALCUL DES SONS D'ARMES, SANS WEBAUDIO.
 *
 * Tout ce qui décide de ce qu'on entend est pur : le tirage de variation, la conversion des
 * unités et le mapping du curseur de distance. Ces tests fixent les deux invariants qui
 * comptent pour l'utilisateur — les réglages par défaut rendent le fichier extrait TEL QUEL,
 * et un manifeste absent ne produit ni erreur ni silence anormal.
 */
import { describe, expect, it } from 'vitest'

import {
  clampPercent,
  distanceChain,
  drawRange,
  drawVariation,
  gainFromDb,
  indexManifest,
  normalizeRange,
  playbackRateFromCents,
} from './weaponSoundLogic'

/** Générateur déterministe : un tirage testé au hasard ne prouverait rien. */
function tirages(...valeurs: number[]): () => number {
  let i = 0
  return () => valeurs[Math.min(i++, valeurs.length - 1)]
}

describe('conversions d’unités', () => {
  it('0 dB ne change rien, et les décibels suivent la formule de puissance', () => {
    expect(gainFromDb(0)).toBe(1)
    expect(gainFromDb(-20)).toBeCloseTo(0.1, 10)
    expect(gainFromDb(20)).toBeCloseTo(10, 10)
  })

  it('une octave vaut 1200 centièmes, et 0 centième laisse la vitesse à 1', () => {
    expect(playbackRateFromCents(0)).toBe(1)
    expect(playbackRateFromCents(1200)).toBeCloseTo(2, 10)
    expect(playbackRateFromCents(-1200)).toBeCloseTo(0.5, 10)
  })

  it('une fourchette aberrante est bornée plutôt que rendue inaudible', () => {
    expect(playbackRateFromCents(100_000)).toBe(4)
    expect(playbackRateFromCents(-100_000)).toBe(0.25)
    expect(playbackRateFromCents(Number.NaN)).toBe(1)
  })
})

describe('fourchettes', () => {
  it('une fourchette absente, non finie ou nulle vaut « pas de variation »', () => {
    expect(normalizeRange(undefined)).toBeNull()
    expect(normalizeRange({ bas: 0, haut: 0 })).toBeNull()
    expect(normalizeRange({ bas: Number.NaN, haut: 1 })).toBeNull()
  })

  it('des bornes inversées sont remises dans l’ordre plutôt que de vider le tirage', () => {
    expect(normalizeRange({ bas: 3, haut: -3 })).toEqual({ bas: -3, haut: 3 })
  })

  it('le tirage balaie la fourchette de bout en bout', () => {
    const plage = { bas: -4, haut: 2 }
    expect(drawRange(plage, 1, tirages(0))).toBeCloseTo(-4, 10)
    expect(drawRange(plage, 1, tirages(1))).toBeCloseTo(2, 10)
    expect(drawRange(plage, 1, tirages(0.5))).toBeCloseTo(-1, 10)
  })

  it('le réglage réduit les BORNES, il ne décale pas le résultat', () => {
    const plage = { bas: -4, haut: 2 }
    // À 50 %, le pire cas vaut la moitié du pire cas d'origine — et le tirage médian reste
    // médian. Appliquer le ratio au résultat aurait tiré tout le son vers le grave.
    expect(drawRange(plage, 0.5, tirages(0))).toBeCloseTo(-2, 10)
    expect(drawRange(plage, 0.5, tirages(1))).toBeCloseTo(1, 10)
  })
})

describe('drawVariation', () => {
  const variation = { volume_db: { bas: -2, haut: 2 }, pitch_cents: { bas: -1200, haut: 1200 } }

  it('à 100 %, applique la fourchette du jeu telle quelle', () => {
    const tirage = drawVariation(variation, 100, tirages(1, 1))
    expect(tirage.gainDb).toBeCloseTo(2, 10)
    expect(tirage.playbackRate).toBeCloseTo(2, 10)
  })

  it('à 0 %, rend le NEUTRE EXACT — le fichier est joué tel qu’il a été extrait', () => {
    expect(drawVariation(variation, 0, tirages(1, 1))).toEqual({ gainDb: 0, playbackRate: 1 })
  })

  it('sans manifeste de variation, lecture pure et aucune erreur', () => {
    expect(drawVariation(undefined, 100, tirages(1))).toEqual({ gainDb: 0, playbackRate: 1 })
    expect(drawVariation({}, 100, tirages(1))).toEqual({ gainDb: 0, playbackRate: 1 })
  })

  it('une fourchette de volume seule ne touche pas la hauteur', () => {
    const tirage = drawVariation({ volume_db: { bas: -6, haut: 0 } }, 100, tirages(0))
    expect(tirage.gainDb).toBeCloseTo(-6, 10)
    expect(tirage.playbackRate).toBe(1)
  })

  it('un réglage illisible ou hors bornes est ramené dans [0, 100]', () => {
    expect(clampPercent(Number.NaN)).toBe(0)
    expect(clampPercent(-10)).toBe(0)
    expect(clampPercent(250)).toBe(100)
    expect(drawVariation(variation, 250, tirages(1, 1)).gainDb).toBeCloseTo(2, 10)
  })
})

describe('distanceChain', () => {
  it('à 0 %, AUCUNE chaîne — le lecteur ne doit insérer aucun nœud', () => {
    expect(distanceChain(0)).toBeNull()
    expect(distanceChain(-5)).toBeNull()
    expect(distanceChain(Number.NaN)).toBeNull()
  })

  it('à 100 %, atténue et assourdit dans les bornes annoncées', () => {
    const chaine = distanceChain(100)
    expect(chaine).not.toBeNull()
    expect(chaine?.gainDb).toBeCloseTo(-24, 10)
    expect(chaine?.cutoffHz).toBeCloseTo(500, 6)
  })

  it('le gain décroît en dB et la coupure GÉOMÉTRIQUEMENT — sinon la moitié du curseur ne s’entend pas', () => {
    const moitie = distanceChain(50)
    expect(moitie?.gainDb).toBeCloseTo(-12, 10)
    // Milieu géométrique de 20 000 et 500, pas leur moyenne (10 250).
    expect(moitie?.cutoffHz).toBeCloseTo(Math.sqrt(20_000 * 500), 6)
  })

  it('la chaîne est monotone : plus loin ne doit jamais sonner plus fort ni plus clair', () => {
    let precedentGain = 0
    let precedenteCoupure = 20_000
    for (let d = 10; d <= 100; d += 10) {
      const c = distanceChain(d)
      expect(c!.gainDb).toBeLessThan(precedentGain)
      expect(c!.cutoffHz).toBeLessThan(precedenteCoupure)
      precedentGain = c!.gainDb
      precedenteCoupure = c!.cutoffHz
    }
  })
})

describe('indexManifest', () => {
  it('indexe par arme et garde la PREMIÈRE entrée d’une arme à plusieurs modes', () => {
    const index = indexManifest({
      sons: [
        { arme: 'un_assaultrifle', fichier: 'a.wav', mode: 'normal' },
        { arme: 'un_assaultrifle', fichier: 'b.wav', mode: 'charge' },
      ],
    })
    expect(index.size).toBe(1)
    expect(index.get('un_assaultrifle')?.fichier).toBe('a.wav')
  })

  it('ignore les entrées inexploitables, et un manifeste absent rend un index vide', () => {
    const index = indexManifest({
      sons: [
        { arme: '', fichier: 'a.wav' },
        { arme: 'x', fichier: '' },
        { arme: 'ok', fichier: 'ok.wav' },
      ],
    })
    expect([...index.keys()]).toEqual(['ok'])
    expect(indexManifest(null).size).toBe(0)
    expect(indexManifest(undefined).size).toBe(0)
  })
})
