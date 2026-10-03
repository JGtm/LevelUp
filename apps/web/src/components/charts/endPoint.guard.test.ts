/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (CLAUDE.md n° 6, règle « ≤ 2 copies ») : « grossir le point final d'une courbe »
 * a une SOURCE UNIQUE, `components/charts/endPoint.ts` (`withEndPoint`). Le motif était écrit à
 * la main dans quatre graphes (écart cumulé au FDA attendu, fil de l'objectif, fil et soirée
 * après soirée des ressources) et une cinquième variante (soirée après soirée de l'objectif) ;
 * la revue L6.1 du plan PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26 (constat R7) les a migrés.
 * Ce test échoue si une copie locale réapparaît dans `src/` hors du helper.
 *
 * Les trois empreintes du motif, telles qu'il était écrit :
 *   1. la recherche du dernier point non nul, boucle d'index (`if (v != null) last = i`) ;
 *   2. la taille conditionnée à l'index final (`symbolSize: i === last ? …`, `symbolSize: end ? …`,
 *      `symbolSize: i === n - 1 ? …`) ;
 *   3. la comparaison d'index au dernier point (`i === last`).
 * Le test du motif lui-même (`reconnaît les copies migrées`) prouve que les expressions régulières
 * attrapent bien les anciennes copies : un garde-rail qui ne voit rien ne garde rien.
 *
 * Pas d'allowlist : aucun fichier n'a de raison de ré-écrire le motif.
 */
import { describe, expect, it } from 'vitest'
import { readdirSync, readFileSync } from 'node:fs'
import { join, resolve, sep } from 'node:path'

const FOOTPRINTS: { name: string; re: RegExp }[] = [
  { name: 'recherche locale du dernier point', re: /if\s*\([^\n]*\)\s*last\w*\s*=\s*i\b/ },
  { name: 'taille conditionnée au point final', re: /symbolSize:\s*\(?\s*(?:i\s*===|end\b|isEnd\b|isLast\b|last\b)/ },
  { name: 'index comparé au dernier point', re: /\bi\s*===\s*last\b/ },
]

const HELPER = 'components/charts/endPoint.ts'

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

/** Les empreintes du motif présentes dans un texte source. */
function footprintsOf(source: string): string[] {
  return FOOTPRINTS.filter((f) => f.re.test(source)).map((f) => f.name)
}

describe('garde-rail endPoint (source unique components/charts/endPoint.ts)', () => {
  it('reconnaît les copies migrées (le garde-rail voit ce qu’il interdit)', () => {
    const copies = [
      // squadFdaGapChart.ts (avant la revue L6.1)
      'data.forEach((v, i) => {\n    if (v != null) last = i\n  })',
      // objectifCharts.ts, fil de la session
      'const data = cum.map((v, i) => (i === last && v != null ? endPoint(v, color, tc.card) : v))',
      // empriseCharts.ts, fil des ressources
      'matches.forEach((m, i) => {\n    if (m.points[resource]) last = i\n  })',
      // empriseCharts.ts, soirée après soirée
      '      symbolSize: end ? 11 : 6,',
      // objectifCharts.ts, soirée après soirée
      '        symbolSize: i === n - 1 ? 10 : 6,',
    ]
    for (const c of copies) expect(footprintsOf(c), c).not.toEqual([])
  })

  it('aucune copie locale du motif dans src/ hors du helper', () => {
    const srcRoot = resolve(process.cwd(), 'src')
    const offenders = walk(srcRoot)
      .map((f) => ({ rel: f.split(sep).join('/').replace(`${srcRoot.split(sep).join('/')}/`, ''), f }))
      .filter(({ rel }) => rel !== HELPER)
      .flatMap(({ rel, f }) => footprintsOf(readFileSync(f, 'utf8')).map((name) => `${rel} : ${name}`))
    expect(offenders, `Grossir le point final : importer withEndPoint de ${HELPER}`).toEqual([])
  })
})
