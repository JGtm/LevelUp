/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (CLAUDE.md n°6) : `components/charts/LazyECharts.tsx` est le SEUL fichier qui
 * importe `echarts-for-react`. Un montage direct reprendrait le suivi de taille de la
 * bibliothèque, qui laisse à sa largeur de montage un graphe monté pendant que son
 * conteneur change de largeur (graphes débordant de leurs cartes à l'ouverture de la
 * comparaison des Sessions) — cf. `lib/echarts/followContainerSize.ts`.
 */
import { describe, it, expect } from 'vitest'
import { readdirSync, readFileSync } from 'node:fs'
import { join, resolve, sep } from 'node:path'

const IMPORT_LITERAL = /(from\s+|import\()\s*['"]echarts-for-react['"]/

const CANONICAL_FILE = 'components/charts/LazyECharts.tsx'

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

describe('garde-rail echartsReactSingleMount (source unique components/charts/LazyECharts.tsx)', () => {
  const srcRoot = resolve(process.cwd(), 'src')
  const rel = (f: string) => f.split(sep).join('/').replace(`${srcRoot.split(sep).join('/')}/`, '')

  it("aucun import de echarts-for-react hors de LazyECharts", () => {
    const porteurs = walk(srcRoot)
      .filter((f) => IMPORT_LITERAL.test(readFileSync(f, 'utf8')))
      .map(rel)
      .filter((relPath) => relPath !== CANONICAL_FILE)
    expect(
      porteurs,
      `Montage direct de echarts-for-react : ${porteurs.join(', ')}. Passer par components/charts/LazyECharts.tsx.`,
    ).toEqual([])
  })

  it('le fichier canonique porte bien l import (pas de garde-rail fantôme)', () => {
    expect(IMPORT_LITERAL.test(readFileSync(join(srcRoot, ...CANONICAL_FILE.split('/')), 'utf8'))).toBe(true)
  })
})
