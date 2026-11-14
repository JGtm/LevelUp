/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail : la couche des infobulles `title` est montée UNE fois, à la racine (`AppProviders`), et nulle
 * part ailleurs. Sans elle, les 250 `title` natifs de l'application redeviennent invisibles dans un partage
 * de fenêtre (Discord, Teams) ; deux montages afficheraient chaque infobulle en double.
 */
import { describe, expect, it } from 'vitest'
import { readdirSync, readFileSync } from 'node:fs'
import { join, resolve } from 'node:path'

const SRC = resolve(__dirname, '..', '..')

function* sources(dir: string): Generator<string> {
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const full = join(dir, entry.name)
    if (entry.isDirectory()) yield* sources(full)
    else if (/\.tsx?$/.test(entry.name) && !/\.test\.tsx?$/.test(entry.name)) yield full
  }
}

describe('couche des infobulles title', () => {
  it('est montée dans AppProviders et seulement là', () => {
    const mounts: string[] = []
    for (const file of sources(SRC)) {
      if (/<TitleTooltipLayer\s*\/>/.test(readFileSync(file, 'utf8'))) mounts.push(file.slice(SRC.length + 1).replaceAll('\\', '/'))
    }
    expect(mounts).toEqual(['app/providers/index.tsx'])
  })
})
