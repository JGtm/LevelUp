/**
 * segmentInk — L'ENCRE D'UNE ÉCRITURE POSÉE SUR UN APLAT : noire ou blanche, celle qui contraste
 * le mieux (WCAG) avec la couleur RÉSOLUE du segment. Source unique des pistes HTML
 * (`SegmentLabel`) et des étiquettes ECharts (`barLabel`) ; aucune valeur d'encre ailleurs.
 *
 * La couleur du segment arrive sous la forme qu'elle a chez l'appelant : hex, `rgb()`, `var(--x)`
 * d'un jeton, `color-mix(...)`, `oklch(...)`. Elle est résolue par le navigateur (sonde `color`
 * puis `cssColorToHex`), jamais par table de conversion. Couleur non résolue (jsdom, SSR) :
 * encre claire, l'état d'avant ce module.
 */
import { useMemo } from 'react'

import { cssColorToHex } from '@/lib/echarts/cssColorToHex'
import { useThemeVersion } from '@/lib/echarts/useThemeVersion'

import { useColorPaletteVersion } from './useColorPaletteVersion'
import { contrastRatio } from './wcagContrast'

/** Les deux encres possibles (jetons d'encre : seule déclaration de ces valeurs). */
export const SEGMENT_INK = { light: '#ffffff', dark: '#000000' } as const

const HEX6 = /^#[0-9a-f]{6}$/i
const HEX3 = /^#[0-9a-f]{3}$/i
const RGB = /^rgba?\(\s*(\d{1,3})[\s,]+(\d{1,3})[\s,]+(\d{1,3})/i

function toHex6(css: string): string | null {
  const c = css.trim()
  if (HEX6.test(c)) return c
  if (HEX3.test(c)) return `#${c[1]}${c[1]}${c[2]}${c[2]}${c[3]}${c[3]}`
  const m = RGB.exec(c)
  if (m) return `#${[m[1], m[2], m[3]].map((v) => Math.min(255, Number(v)).toString(16).padStart(2, '0')).join('')}`
  return null
}

/** Couleur CSS quelconque -> `#rrggbb`, ou null si elle ne se résout pas. */
export function resolveCssColor(css: string): string | null {
  const direct = toHex6(css)
  if (direct) return direct
  if (typeof document === 'undefined' || !css) return null
  const probe = document.createElement('span')
  probe.style.color = css
  if (!probe.style.color) return null
  probe.style.display = 'none'
  document.body.appendChild(probe)
  try {
    // `var()` et `color-mix()` se résolvent à la lecture, depuis l'élément posé dans le document.
    const computed = getComputedStyle(probe).color
    // jsdom n'a pas de canvas : la normalisation y bruit sans rien résoudre.
    if (toHex6(computed) == null && /jsdom/i.test(navigator.userAgent)) return null
    return toHex6(computed) ?? toHex6(cssColorToHex(computed))
  } finally {
    probe.remove()
  }
}

/** Aplat translucide (`itemStyle.opacity` d'ECharts) : la couleur vue est son mélange avec le fond. */
export interface InkBackdrop {
  /** Couleur du fond sur lequel l'aplat est posé (fond de la carte). */
  color: string
  /** Opacité de l'aplat, 0..1. */
  opacity: number
}

function compositeOver(fillHex: string, backdrop: InkBackdrop): string {
  const under = resolveCssColor(backdrop.color)
  if (!under) return fillHex
  const a = Math.max(0, Math.min(1, backdrop.opacity))
  const channel = (i: number) => Math.round(parseInt(fillHex.slice(i, i + 2), 16) * a + parseInt(under.slice(i, i + 2), 16) * (1 - a))
  return `#${[1, 3, 5].map((i) => channel(i).toString(16).padStart(2, '0')).join('')}`
}

/**
 * L'encre de meilleur contraste sur `fill` (hex, `rgb()`, `var()`, `color-mix()`...), mélangé au
 * `backdrop` quand l'aplat est translucide.
 */
export function inkOn(fill: string, backdrop?: InkBackdrop): string {
  const resolved = resolveCssColor(fill)
  const hex = resolved && backdrop ? compositeOver(resolved, backdrop) : resolved
  if (!hex) return SEGMENT_INK.light
  return contrastRatio(hex, SEGMENT_INK.dark) > contrastRatio(hex, SEGMENT_INK.light) ? SEGMENT_INK.dark : SEGMENT_INK.light
}

/** `inkOn` qui suit la palette et le thème (la couleur d'un jeton change avec eux). */
export function useSegmentInk(fill: string): string {
  const paletteVersion = useColorPaletteVersion()
  const themeVersion = useThemeVersion()
  // eslint-disable-next-line react-hooks/exhaustive-deps
  return useMemo(() => inkOn(fill), [fill, paletteVersion, themeVersion])
}
