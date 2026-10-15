/**
 * segmentInk — L'ENCRE D'UNE ÉCRITURE POSÉE SUR UN APLAT DE SEGMENT : blanche, cernée d'un contour
 * sombre fin, dans les deux thèmes (décision du 2026-10-15 ; plus de bascule noir/blanc par
 * contraste). Les valeurs viennent des jetons CSS `--segment-ink` et `--segment-ink-outline`
 * (`styles/globals.css`), lus ici pour les graphes ECharts, dont le canvas ne lit pas `var()`.
 * Les pistes HTML posent la classe `segment-ink` (`SegmentLabel`).
 */

/** Valeurs de repli (SSR, jsdom) : les mêmes que les jetons. */
const FALLBACK = { ink: '#ffffff', outline: '#0b1220' } as const

export interface SegmentInk {
  ink: string
  outline: string
}

function readToken(name: string, fallback: string): string {
  if (typeof document === 'undefined') return fallback
  const value = getComputedStyle(document.documentElement).getPropertyValue(name).trim()
  return value || fallback
}

/** L'encre et son contour, résolus depuis les jetons CSS. */
export function segmentInk(): SegmentInk {
  return { ink: readToken('--segment-ink', FALLBACK.ink), outline: readToken('--segment-ink-outline', FALLBACK.outline) }
}
