/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail des ENCRES du canvas : chaque `InkVar` que le rendu peut demander existe dans le
 * thème. Une variable absente rendrait '' (fillStyle inchangé) et un contour de nom
 * invisible, en silence — `readInk` le journalise, ce test le fait échouer avant.
 */
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const CSS = resolve(__dirname, '..', '..', 'styles', 'globals.css')

/** Les encres déclarées par `InkVar` (canvasInk.ts) qui appartiennent au rejeu lui-même. */
const REPLAY_INKS = ['--replay-label-stroke'] as const

describe('garde-rail : encres du canvas de rejeu', () => {
  it('le contour des noms est défini dans les DEUX thèmes', () => {
    const css = readFileSync(CSS, 'utf8')
    for (const v of REPLAY_INKS) {
      const decl = css.indexOf(`${v}:`)
      expect(decl, `${v} absente de globals.css`).toBeGreaterThan(-1)
      // Le bloc qui la porte doit viser :root, le thème sombre ET le thème clair.
      const block = css.slice(css.lastIndexOf('}', decl), decl)
      expect(block.includes(":root[data-theme='dark']"), `${v} : thème sombre`).toBe(true)
      expect(block.includes(":root[data-theme='light']"), `${v} : thème clair`).toBe(true)
    }
  })
})
