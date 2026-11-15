/**
 * killFocusMotion.test.ts — LES TROIS GESTES DE LA MISE EN VALEUR, en fonction du temps écoulé.
 *
 * Les valeurs attendues sont celles de la maquette validée : verrouillage 340 ms (échelle 2,2 et
 * rotation de 45 degrés vers 1 et 0, dépassement à 0,88 à 72 % du temps, opacité 0 vers 1),
 * étiquette 280 ms (échelle 0,6 vers 1,08 à 70 % puis 1), voile 240 ms. Une courbe d'accélération
 * unique, `cubic-bezier(.2,.8,.2,1)`, s'applique AVANT l'interpolation des images-clés.
 */
import { describe, expect, it } from 'vitest'

import {
  cubicBezier,
  FADE_MS,
  FOCUS_MOTION_MS,
  LOCK_MS,
  fadeOpacity,
  lockPose,
  popPose,
} from './killFocusMotion'

describe('cubicBezier — la courbe d’accélération', () => {
  const ease = cubicBezier(0.2, 0.8, 0.2, 1)

  it('part de 0, arrive à 1, et reste dans [0, 1]', () => {
    expect(ease(0)).toBe(0)
    expect(ease(1)).toBe(1)
    for (let t = 0; t <= 1; t += 0.05) {
      const y = ease(t)
      expect(y).toBeGreaterThanOrEqual(0)
      expect(y).toBeLessThanOrEqual(1)
    }
  })

  it('démarre vite (la courbe est au-dessus de la diagonale) et ne recule jamais', () => {
    expect(ease(0.25)).toBeGreaterThan(0.25)
    let prev = 0
    for (let t = 0; t <= 1; t += 0.02) {
      const y = ease(t)
      expect(y).toBeGreaterThanOrEqual(prev - 1e-9)
      prev = y
    }
  })
})

describe('lockPose — le verrouillage des équerres', () => {
  it('commence large, tournée et invisible', () => {
    expect(lockPose(0)).toEqual({ scale: 2.2, rotationDeg: 45, opacity: 0 })
  })

  it('finit à l’échelle 1, sans rotation, opaque — et y reste', () => {
    expect(lockPose(LOCK_MS)).toEqual({ scale: 1, rotationDeg: 0, opacity: 1 })
    expect(lockPose(LOCK_MS * 10)).toEqual({ scale: 1, rotationDeg: 0, opacity: 1 })
  })

  it('DÉPASSE : passe sous l’échelle 1 (jusqu’à 0,88) avant de se poser', () => {
    let min = Infinity
    for (let ms = 0; ms <= LOCK_MS; ms += 5) min = Math.min(min, lockPose(ms).scale)
    expect(min).toBeLessThan(1)
    expect(min).toBeGreaterThanOrEqual(0.88)
  })

  it('la rotation et l’échelle décroissent dès les premiers instants (la courbe démarre vite)', () => {
    const mi = lockPose(LOCK_MS * 0.05)
    expect(mi.rotationDeg).toBeLessThan(45)
    expect(mi.rotationDeg).toBeGreaterThan(0)
    expect(mi.scale).toBeLessThan(2.2)
  })

  it('un temps négatif vaut le départ', () => {
    expect(lockPose(-50)).toEqual(lockPose(0))
  })
})

describe('popPose — l’étiquette', () => {
  it('commence à 0,6 et invisible, finit à 1', () => {
    expect(popPose(0)).toEqual({ scale: 0.6, rotationDeg: 0, opacity: 0 })
    expect(popPose(10_000)).toEqual({ scale: 1, rotationDeg: 0, opacity: 1 })
  })

  it('dépasse 1 (jusqu’à 1,08) avant de se poser', () => {
    let max = 0
    for (let ms = 0; ms <= 280; ms += 5) max = Math.max(max, popPose(ms).scale)
    expect(max).toBeGreaterThan(1)
    expect(max).toBeLessThanOrEqual(1.08 + 1e-9)
  })
})

describe('fadeOpacity — le voile', () => {
  it('monte de 0 à 1 en 240 ms', () => {
    expect(fadeOpacity(0)).toBe(0)
    expect(fadeOpacity(FADE_MS)).toBe(1)
    expect(fadeOpacity(FADE_MS / 2)).toBeGreaterThan(0)
    expect(fadeOpacity(FADE_MS / 2)).toBeLessThan(1)
  })
})

describe('FOCUS_MOTION_MS — la durée au bout de laquelle la boucle d’animation s’arrête', () => {
  it('vaut la plus longue des trois animations', () => {
    expect(FOCUS_MOTION_MS).toBe(LOCK_MS)
    expect(FOCUS_MOTION_MS).toBeGreaterThanOrEqual(FADE_MS)
  })
})
