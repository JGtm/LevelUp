/**
 * mapReliefTokens.test.ts — garde-fou du fond « Relief » et du cerne deux tons des pions
 * (jetons `map-relief-base`, `marker-outline-band`, `marker-outline-rim`), pour chaque palette et
 * chaque thème, sous les quatre visions (normale, protanopie, deutéranopie, tritanopie).
 *
 * Trois propriétés, et chacune a une raison de ne pas se contenter de l'œil :
 *   1. Rampe — les paliers voisins de la rampe du Relief (`reliefRamp.ts`, tirée de
 *      `map-relief-base`) restent à ΔE ≥ 6 : le relief se lit à la clarté, et deux paliers qui
 *      fondent effacent une marche. Même seuil que la rampe du fond Plan (6,7 en son pire).
 *   2. Cerne — le MEILLEUR des deux tons (bande sombre, filet clair) tient 3:1 contre chaque
 *      palier de la rampe du Relief, contre chaque teinte du fond Plan, contre la surface du fond
 *      Rendu (style « encre ») et contre la carte du thème (là où aucun fond n'est peint). Le
 *      contour d'avant (`--muted-foreground` à 62 %) n'atteignait 3:1 dans aucune des 1 280
 *      combinaisons mesurées : c'est ce qui motive le cerne.
 *   3. Valeurs — une teinte par palette (la même dans les deux thèmes), les deux encres du cerne
 *      identiques partout, la bande plus sombre que le filet.
 *
 * Les paliers de la rampe (gris 0,36 / 0,52 / 0,68 / 0,84 / 1,0) sont ceux de la mesure de teinte :
 * les cinq paliers où se trouve le sol des quatre cartes témoins. Les teintes du fond Plan sont
 * LES SIX POINTS de `rampePlanHauteurs` (brique `halo-infinite-map-images`, `himap/plan_hauteurs.go`,
 * v1.3.1), plus leurs milieux et leurs versions dans l'ombre portée (× 0,72 rouge et vert, × 0,76
 * bleu) ; la surface du fond Rendu est le gris du style « encre » (`himap/rendu_couleur.go`,
 * `TeinteEncre`), de son plus sombre à son plus clair, avec le gris du sol supposé (178) et les
 * arêtes (un tiers de la teinte).
 */
import { describe, expect, it } from 'vitest'

import { deltaE, simulateCvd, type CvdKind } from './colorDistance'
import { MAP_MARKER_OUTLINE } from './palettes/_mapMarkerOutline'
import { cividisPalette } from './palettes/cividis'
import { defaultPalette } from './palettes/default'
import { okabePalette } from './palettes/okabe-ito'
import { tolBrightPalette } from './palettes/tol-bright'
import { reliefRampHex, reliefRampStops } from './reliefRamp'
import type { Palette } from './semantic-tokens'
import { contrastRatio, relLuminance } from './wcagContrast'

/** Teinte de base décidée par palette (une seule valeur, les deux thèmes). */
const CASES: Record<string, { palette: Palette; base: string }> = {
  default: { palette: defaultPalette, base: '#6B7078' },
  'okabe-ito': { palette: okabePalette, base: '#7559B5' },
  cividis: { palette: cividisPalette, base: '#3F8C4F' },
  'tol-bright': { palette: tolBrightPalette, base: '#7559B5' },
}

const THEMES: ('light' | 'dark')[] = ['light', 'dark']
/** Normale plus les trois daltonismes simulés. */
const VISIONS: (CvdKind | 'normale')[] = ['normale', 'protanopie', 'deuteranopie', 'tritanopie']

const seen = (hex: string, vision: CvdKind | 'normale'): string => (vision === 'normale' ? hex : simulateCvd(hex, vision))

/** Les cinq paliers où se trouve le sol des cartes témoins. */
const RELIEF_STEPS = [0.36, 0.52, 0.68, 0.84, 1.0] as const

/** Écart minimal entre paliers voisins de la rampe, sous chaque vision. */
const MIN_RAMP_DELTA_E = 6
/** Contraste minimal du meilleur des deux tons du cerne contre n'importe quel fond. */
const MIN_OUTLINE_CONTRAST = 3

/** Surface de la carte (`--card`) des deux thèmes : le fond là où rien n'est peint. */
const CARD = { light: '#FCFDFF', dark: '#171717' } as const

// --- Fond Plan : `rampePlanHauteurs` (himap/plan_hauteurs.go) --------------------------------

const PLAN_STOPS: ReadonlyArray<readonly [number, number, number]> = [
  [0x60, 0x70, 0x8c], [0x7e, 0x8c, 0x96], [0x9e, 0xa6, 0xa0],
  [0xbe, 0xbe, 0xa0], [0xdc, 0xd6, 0xa4], [0xf6, 0xec, 0xb0],
]
const SHADOW_RG = 0.72
const SHADOW_B = 0.76

const hex = (rgb: readonly number[]): string =>
  '#' + rgb.map((v) => Math.round(v).toString(16).padStart(2, '0')).join('')

function planGrounds(): string[] {
  const points: Array<readonly number[]> = [...PLAN_STOPS]
  for (let i = 0; i + 1 < PLAN_STOPS.length; i++) {
    points.push(PLAN_STOPS[i].map((v, c) => (v + PLAN_STOPS[i + 1][c]) / 2))
  }
  const shadowed = points.map((p) => [Math.floor(p[0] * SHADOW_RG), Math.floor(p[1] * SHADOW_RG), Math.floor(p[2] * SHADOW_B)])
  return [...points, ...shadowed].map(hex)
}

// --- Fond Rendu, style « encre » (himap/rendu_couleur.go) -----------------------------------

const RENDU_UNKNOWN_GROUND_GRAY = 178
const RENDU_EDGE_DIVISOR = 3

/** `TeinteEncre` : v = (0,30 + 0,70 × éclairement) × (1 − 0,45 × écart au niveau de jeu), éclairement dans [0,25 ; 1]. */
function renduGrounds(): string[] {
  const grays = new Set<number>([RENDU_UNKNOWN_GROUND_GRAY])
  for (const e of [0.25, 0.5, 0.75, 1]) {
    for (const x of [0, 0.5, 1]) {
      const g = Math.round(255 * Math.max(0, Math.min(1, (0.3 + 0.7 * e) * (1 - 0.45 * x))))
      grays.add(g)
      grays.add(Math.floor(g / RENDU_EDGE_DIVISOR))
    }
  }
  return [...grays].map((g) => hex([g, g, g]))
}

/** Contraste du meilleur des deux tons du cerne contre `ground`, sous `vision`. */
function bestOutlineContrast(ground: string, vision: CvdKind | 'normale'): number {
  const g = seen(ground, vision)
  return Math.max(
    contrastRatio(seen(MAP_MARKER_OUTLINE['marker-outline-band'], vision), g),
    contrastRatio(seen(MAP_MARKER_OUTLINE['marker-outline-rim'], vision), g),
  )
}

describe.each(Object.entries(CASES))('palette « %s »', (_name, { palette: base, base: expectedBase }) => {
  it('porte la teinte de base décidée, en hex plein', () => {
    expect(base['map-relief-base']).toBe(expectedBase)
  })

  describe.each(THEMES)('thème %s', (theme) => {
    // Les jetons ne dépendent que de la palette : les deux thèmes partagent la teinte et le cerne.
    const palette = base
    const ramp = RELIEF_STEPS.map((g) => reliefRampHex(palette['map-relief-base'], g))

    it.each(VISIONS)('rampe : paliers voisins à ΔE ≥ 6 (%s)', (vision) => {
      for (let i = 1; i < ramp.length; i++) {
        const de = deltaE(seen(ramp[i - 1], vision), seen(ramp[i], vision))
        expect(de, `${RELIEF_STEPS[i - 1]} / ${RELIEF_STEPS[i]} : ${ramp[i - 1]} / ${ramp[i]}`).toBeGreaterThanOrEqual(MIN_RAMP_DELTA_E)
      }
    })

    describe.each(VISIONS)('cerne, vision %s', (vision) => {
      const grounds: Array<[string, string[]]> = [
        ['rampe du Relief', ramp],
        ['fond Plan', planGrounds()],
        ['fond Rendu', renduGrounds()],
        ['carte du thème', [CARD[theme]]],
      ]
      it.each(grounds)('le meilleur des deux tons tient 3:1 contre %s', (_what, list) => {
        for (const ground of list) {
          expect(bestOutlineContrast(ground, vision), `fond ${ground}`).toBeGreaterThanOrEqual(MIN_OUTLINE_CONTRAST)
        }
      })
    })
  })
})

describe('encres du cerne', () => {
  const band = MAP_MARKER_OUTLINE['marker-outline-band']
  const rim = MAP_MARKER_OUTLINE['marker-outline-rim']

  it('la bande est plus sombre que le filet, avec un écart de luminance franc', () => {
    expect(relLuminance(band)).toBeLessThan(relLuminance(rim))
    expect(contrastRatio(band, rim)).toBeGreaterThanOrEqual(10)
  })

  it.each(Object.entries(CASES))('palette « %s » : jetons du cerne repris de la source commune', (_n, { palette }) => {
    expect(palette['marker-outline-band']).toBe(band)
    expect(palette['marker-outline-rim']).toBe(rim)
  })
})

describe('reliefRamp — source unique de la rampe du Relief', () => {
  it('reliefRampStops donne les trois points, et reliefRampHex les retrouve en 0, 0,5 et 1', () => {
    const { low, mid, high } = reliefRampStops('#6B7078')
    expect(reliefRampHex('#6B7078', 0)).toBe(low)
    expect(reliefRampHex('#6B7078', 0.5)).toBe(mid)
    expect(reliefRampHex('#6B7078', 1)).toBe(high)
    expect(relLuminance(low)).toBeLessThan(relLuminance(mid))
    expect(relLuminance(mid)).toBeLessThan(relLuminance(high))
  })
})
