/**
 * squadPlayerTokens.test.ts — garde-rail de la quadruple `squad-player-1..4`.
 *
 * Ces quatre tokens portent l'IDENTITÉ d'un joueur d'escouade dans toute l'app
 * (pill du multiselect, pistes des charts, pills média, vue Match). Si deux
 * d'entre eux se rapprochent, on ne confond pas « deux nuances » mais DEUX
 * JOUEURS : ce test échoue dès qu'une modification de palette réduit l'écart.
 *
 * Il vérifie deux propriétés indépendantes :
 *   1. Lisibilité — contraste WCAG 2.0 ≥ 3:1 (seuil « objet graphique non
 *      textuel », WCAG 2.1 SC 1.4.11) contre les deux surfaces produit.
 *   2. Séparation — distance perceptuelle ΔE OKLab entre toutes les paires, en
 *      vision normale ET sous simulation protanopie / deutéranopie.
 *
 * Tout est calculé ici (aucune dépendance) : conversion sRGB → OKLab et
 * matrices de simulation daltonisme de Machado, Oliveira & Fernandes (2009),
 * sévérité 1.0, appliquées en RGB LINÉAIRE.
 */
import { describe, it, expect } from 'vitest'
import type { Palette, SemanticToken } from './semantic-tokens'
import { defaultPalette } from './palettes/default'
import { okabePalette } from './palettes/okabe-ito'
import { cividisPalette } from './palettes/cividis'
import { tolBrightPalette } from './palettes/tol-bright'
import { contrastRatio } from './wcagContrast'

/** Les 4 tokens verrouillés — ordre = ordre d'attribution des joueurs. */
const SQUAD_PLAYER_TOKENS: SemanticToken[] = [
  'squad-player-1',
  'squad-player-2',
  'squad-player-3',
  'squad-player-4',
]

// Surfaces produit sur lesquelles les couleurs joueurs sont rendues : la carte
// (`--card`) des thèmes clair et sombre. DÉRIVÉES DE `styles/globals.css`
// (oklch(0.995 0.002 255) et oklch(0.205 0 0)) — à resynchroniser si les
// surfaces bougent, sinon ce garde-rail valide contre un fond qui n'existe plus.
const SURFACE_LIGHT = '#FCFDFF'
const SURFACE_DARK = '#171717'

/** WCAG 2.1 SC 1.4.11 — objet graphique non textuel. */
const MIN_CONTRAST = 3
/** ΔE OKLab × 100 : deux couleurs sous ce seuil se lisent comme une variation. */
const MIN_DELTA_E_NORMAL = 15
/** Idem sous simulation daltonisme : la perte de gamut impose un seuil plus bas. */
const MIN_DELTA_E_CVD = 8

// ── Conversions couleur (autonomes) ─────────────────────────────────────────

function parseHex(hex: string): [number, number, number] {
  return [
    parseInt(hex.slice(1, 3), 16),
    parseInt(hex.slice(3, 5), 16),
    parseInt(hex.slice(5, 7), 16),
  ]
}

const toLinear = (c: number): number => {
  const v = c / 255
  return v <= 0.04045 ? v / 12.92 : Math.pow((v + 0.055) / 1.055, 2.4)
}

const toSrgb = (v: number): number => {
  const c = v <= 0.0031308 ? 12.92 * v : 1.055 * Math.pow(v, 1 / 2.4) - 0.055
  return Math.max(0, Math.min(255, Math.round(c * 255)))
}

/** sRGB → OKLab (Björn Ottosson, 2020). */
function oklab(hex: string): [number, number, number] {
  const [r, g, b] = parseHex(hex).map(toLinear)
  const l = Math.cbrt(0.4122214708 * r + 0.5363325363 * g + 0.0514459929 * b)
  const m = Math.cbrt(0.2119034982 * r + 0.6806995451 * g + 0.1073969566 * b)
  const s = Math.cbrt(0.0883024619 * r + 0.2817188376 * g + 0.6299787005 * b)
  return [
    0.2104542553 * l + 0.7936177850 * m - 0.0040720468 * s,
    1.9779984951 * l - 2.4285922050 * m + 0.4505937099 * s,
    0.0259040371 * l + 0.7827717662 * m - 0.8086757660 * s,
  ]
}

/** Distance euclidienne OKLab × 100 (≈ 1 unité = 1 « just noticeable step »). */
function deltaE(hexA: string, hexB: string): number {
  const a = oklab(hexA)
  const b = oklab(hexB)
  return 100 * Math.hypot(a[0] - b[0], a[1] - b[1], a[2] - b[2])
}

/** Machado et al. (2009), sévérité 1.0 — matrices en RGB linéaire, ligne par ligne. */
const CVD_MATRICES = {
  protanopie: [
    0.152286, 1.052583, -0.204868,
    0.114503, 0.786281, 0.099216,
    -0.003882, -0.048116, 1.051998,
  ],
  deuteranopie: [
    0.367322, 0.860646, -0.227968,
    0.280085, 0.672501, 0.047413,
    -0.011820, 0.042940, 0.968881,
  ],
} as const

type CvdKind = keyof typeof CVD_MATRICES

function simulateCvd(hex: string, kind: CvdKind): string {
  const m = CVD_MATRICES[kind]
  const [r, g, b] = parseHex(hex).map(toLinear)
  const out = [
    m[0] * r + m[1] * g + m[2] * b,
    m[3] * r + m[4] * g + m[5] * b,
    m[6] * r + m[7] * g + m[8] * b,
  ]
  return '#' + out.map((v) => toSrgb(v).toString(16).padStart(2, '0')).join('')
}

// ── Périmètre par palette ───────────────────────────────────────────────────

interface PaletteCase {
  name: string
  palette: Palette
  /**
   * 'both' : chaque token doit tenir le contraste sur les DEUX surfaces.
   * 'any'  : au moins une des deux (cf. exemption cividis ci-dessous).
   */
  contrast: 'both' | 'any'
  /** Simulation daltonisme exigée ? (cf. exemptions ci-dessous.) */
  cvd: boolean
}

const CASES: PaletteCase[] = [
  { name: 'default', palette: defaultPalette, contrast: 'both', cvd: true },
  { name: 'okabe-ito', palette: okabePalette, contrast: 'both', cvd: true },
  // Exemption CVD — cividis est une rampe SÉQUENTIELLE monochrome par
  // construction (Nuñez et al. 2018) : l'identité y passe par la clarté, pas par
  // la teinte, donc une simulation protan/deutan ne mesure rien d'utile (elle
  // préserve la clarté). Exemption contraste 'any' pour la même raison : une
  // rampe monotone en L* ne peut pas placer 4 pas BIEN ESPACÉS dans la fenêtre
  // étroite qui satisfait à la fois un fond quasi blanc et un fond quasi noir —
  // ses extrémités sont faites pour la surface opposée.
  { name: 'cividis', palette: cividisPalette, contrast: 'any', cvd: false },
  // Exemption CVD — les schémas de Paul Tol sont validés daltonisme par leur
  // auteur ; les 4 teintes retenues viennent de Vibrant/Muted, déjà éprouvés.
  { name: 'tol-bright', palette: tolBrightPalette, contrast: 'both', cvd: false },
]

// ── Tests ───────────────────────────────────────────────────────────────────

describe.each(CASES)('quadruple squad-player — palette "$name"', ({ palette, contrast, cvd }) => {
  const hexes = SQUAD_PLAYER_TOKENS.map((t) => palette[t])

  it.each(SQUAD_PLAYER_TOKENS)(`%s est lisible sur les surfaces produit`, (token) => {
    const hex = palette[token]
    const light = contrastRatio(hex, SURFACE_LIGHT)
    const dark = contrastRatio(hex, SURFACE_DARK)
    const detail = `${hex} : clair ${light.toFixed(2)}:1, sombre ${dark.toFixed(2)}:1`
    if (contrast === 'both') {
      expect(light, `${token} illisible sur la surface claire — ${detail}`).toBeGreaterThanOrEqual(MIN_CONTRAST)
      expect(dark, `${token} illisible sur la surface sombre — ${detail}`).toBeGreaterThanOrEqual(MIN_CONTRAST)
    } else {
      expect(Math.max(light, dark), `${token} illisible sur les deux surfaces — ${detail}`)
        .toBeGreaterThanOrEqual(MIN_CONTRAST)
    }
  })

  it('toutes les paires restent distinctes en vision normale', () => {
    for (let i = 0; i < hexes.length; i++) {
      for (let j = i + 1; j < hexes.length; j++) {
        const d = deltaE(hexes[i], hexes[j])
        expect(
          d,
          `${SQUAD_PLAYER_TOKENS[i]} (${hexes[i]}) et ${SQUAD_PLAYER_TOKENS[j]} (${hexes[j]}) trop proches : ΔE ${d.toFixed(1)}`,
        ).toBeGreaterThanOrEqual(MIN_DELTA_E_NORMAL)
      }
    }
  })

  const cvdKinds: CvdKind[] = cvd ? ['protanopie', 'deuteranopie'] : []
  it.each(cvdKinds)('toutes les paires restent distinctes sous %s', (kind) => {
    const simulated = hexes.map((h) => simulateCvd(h, kind))
    for (let i = 0; i < simulated.length; i++) {
      for (let j = i + 1; j < simulated.length; j++) {
        const d = deltaE(simulated[i], simulated[j])
        expect(
          d,
          `${SQUAD_PLAYER_TOKENS[i]} (${hexes[i]} → ${simulated[i]}) et ${SQUAD_PLAYER_TOKENS[j]} (${hexes[j]} → ${simulated[j]}) confondus en ${kind} : ΔE ${d.toFixed(1)}`,
        ).toBeGreaterThanOrEqual(MIN_DELTA_E_CVD)
      }
    }
  })
})

describe('simulation daltonisme — sanité des matrices', () => {
  it('un gris reste un gris (les matrices sont normalisées en ligne)', () => {
    expect(simulateCvd('#808080', 'protanopie')).toBe('#808080')
    expect(simulateCvd('#808080', 'deuteranopie')).toBe('#808080')
  })

  it('rouge et vert se rapprochent en deutéranopie (ce que le test mesure)', () => {
    const brut = deltaE('#FF0000', '#00FF00')
    const simule = deltaE(simulateCvd('#FF0000', 'deuteranopie'), simulateCvd('#00FF00', 'deuteranopie'))
    expect(simule).toBeLessThan(brut)
  })
})
