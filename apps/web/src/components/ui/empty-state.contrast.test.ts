/// <reference types="node" />
// @vitest-environment node
/**
 * Le bord tireté du bloc placeholder (`EmptyStateNotice`) se voit en thème sombre.
 *
 * Avec `--border` (L 0.275) sur le fond muted du bloc, le contraste tombait à ~1,1:1 : le bloc
 * semblait absent. `--placeholder-border` doit tenir 3:1 (contraste des bords d'un composant,
 * WCAG 1.4.11) face au fond du bloc (`--muted`), à la carte qui le porte (`--card`) et au fond de
 * page (`--background`). En clair, il reste `--border` (rendu inchangé).
 *
 * Les valeurs sombres sont des gris OKLCH (chroma nulle) : leur luminance relative vaut L³.
 */
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

import { describe, expect, it } from 'vitest'

const CSS = readFileSync(resolve(process.cwd(), 'src', 'styles', 'globals.css'), 'utf8')

/** Le bloc du thème : sombre (défaut) ou clair. */
function bloc(theme: 'dark' | 'light'): string {
  const debut = theme === 'dark' ? CSS.indexOf(":root[data-theme='dark'] {") : CSS.indexOf(":root[data-theme='light'] {")
  return CSS.slice(debut, CSS.indexOf('\n}', debut))
}

function valeur(theme: 'dark' | 'light', nom: string): string {
  const m = new RegExp(`\\s${nom}:\\s*([^;]+);`).exec(bloc(theme))
  if (!m) throw new Error(`${nom} absente du thème ${theme}`)
  return m[1].trim()
}

/** Luminance relative d'un gris OKLCH `oklch(L 0 0)`. */
function luminanceGris(v: string): number {
  const m = /^oklch\(([\d.]+) 0 0\)$/.exec(v)
  if (!m) throw new Error(`gris OKLCH attendu : ${v}`)
  return Number(m[1]) ** 3
}

function contraste(a: number, b: number): number {
  return (Math.max(a, b) + 0.05) / (Math.min(a, b) + 0.05)
}

describe('EmptyStateNotice — bord tireté', () => {
  it('sombre : 3:1 au moins face au fond du bloc, à la carte et à la page', () => {
    const bord = luminanceGris(valeur('dark', '--placeholder-border'))
    for (const fond of ['--muted', '--card', '--background']) {
      expect(contraste(bord, luminanceGris(valeur('dark', fond))), fond).toBeGreaterThanOrEqual(3)
    }
  })

  it('sombre : l’ancien bord (--border) ne tenait pas — ce test prouve quelque chose', () => {
    const ancien = luminanceGris(valeur('dark', '--border'))
    expect(contraste(ancien, luminanceGris(valeur('dark', '--muted')))).toBeLessThan(1.5)
  })

  it('clair : le bord reste celui de --border', () => {
    expect(valeur('light', '--placeholder-border')).toBe('var(--border)')
  })
})
