/// <reference types="node" />
// @vitest-environment node
/**
 * sansAccents : la normalisation sans accents (tests unitaires) + garde-rail (CLAUDE.md n°6) —
 * `.normalize('NFD')` ne s'écrit que dans `lib/text/sansAccents.ts`. Quatre copies vivaient dans la
 * colonne des cartes de l'onglet Tactique, le glossaire (deux) et les noms d'équipe ; migrées le
 * 2026-10-07 (plan Tactique v2, lot L13 F6).
 */
import { describe, expect, it } from 'vitest'
import { readdirSync, readFileSync } from 'node:fs'
import { join, resolve } from 'node:path'

import { sansAccents } from './sansAccents'

describe('sansAccents', () => {
  it('retire les accents, garde la casse et les blancs', () => {
    expect(sansAccents('Écluse')).toBe('Ecluse')
    expect(sansAccents('  Béhémoth ')).toBe('  Behemoth ')
    expect(sansAccents('Ça, où, déjà')).toBe('Ca, ou, deja')
    expect(sansAccents('Streets')).toBe('Streets')
  })
})

function walk(dir: string): string[] {
  const out: string[] = []
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const full = join(dir, entry.name)
    if (entry.isDirectory()) {
      if (entry.name === 'node_modules' || entry.name === 'generated') continue
      out.push(...walk(full))
    } else if (/\.(ts|tsx)$/.test(entry.name) && !/\.test\.tsx?$/.test(entry.name)) {
      out.push(full)
    }
  }
  return out
}

/** La décomposition NFD écrite à la main, sous ses deux guillemets. */
const NFD_A_LA_MAIN = /\.normalize\(\s*['"`]NFD['"`]\s*\)/

describe('garde-rail sansAccents (lib/text/sansAccents.ts source unique)', () => {
  it('reconnaît la décomposition écrite à la main', () => {
    expect(NFD_A_LA_MAIN.test("texte.normalize('NFD').replace(/x/g, '')")).toBe(true)
    expect(NFD_A_LA_MAIN.test('nom.normalize("NFD")')).toBe(true)
    expect(NFD_A_LA_MAIN.test("texte.normalize('NFC')")).toBe(false)
  })

  it('.normalize(\'NFD\') ne s’écrit nulle part hors lib/text/sansAccents.ts', () => {
    const srcRoot = resolve(process.cwd(), 'src')
    const offenders: string[] = []
    for (const file of walk(srcRoot)) {
      const rel = file.replace(srcRoot, 'src').replace(/\\/g, '/')
      if (rel === 'src/lib/text/sansAccents.ts') continue
      if (NFD_A_LA_MAIN.test(readFileSync(file, 'utf8'))) offenders.push(rel)
    }
    expect(offenders, `Normalisation sans accents à migrer vers @/lib/text/sansAccents : ${offenders.join(', ')}`).toEqual([])
  })
})
