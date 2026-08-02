/**
 * Garde-rail (règle n°6 CLAUDE.md) : aucune clé de métrique technique `Field*`
 * ne doit apparaître dans le JSX rendu des features Prestige et Ascension. Les
 * libellés passent par le hook unique `useMetricLabel()` (qui résout depuis les
 * manifests TOML) ; les valeurs `Field*` (contrat fil) vivent dans
 * lib/i18n/metricLabel.ts, hors composants.
 *
 * Le motif `\bField[A-Z][a-zA-Z]+\b` cible les jetons de type FieldKDA /
 * FieldWinRate. Il EXCLUT volontairement `formFieldMetric`, `<Field …>` (pas de
 * limite de mot avant « Field », ou pas de majuscule collée après). Les fichiers
 * de test sont exclus : ils référencent légitimement la valeur fil dans des mocks.
 */
import { describe, it, expect } from 'vitest'
import { readdirSync, readFileSync } from 'node:fs'
import { join, dirname } from 'node:path'
import { fileURLToPath } from 'node:url'

const FEATURES_ROOT = dirname(fileURLToPath(import.meta.url))
const RAW_FIELD_KEY = /\bField[A-Z][a-zA-Z]+\b/
const SCANNED_FEATURES = ['prestige', 'ascension'] as const

function collectSourceFiles(dir: string): string[] {
  const out: string[] = []
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const full = join(dir, entry.name)
    if (entry.isDirectory()) {
      out.push(...collectSourceFiles(full))
      continue
    }
    if (!/\.(ts|tsx)$/.test(entry.name)) continue
    if (/\.test\.(ts|tsx)$/.test(entry.name)) continue
    out.push(full)
  }
  return out
}

describe('garde-rail : aucune clé métrique Field* dans le JSX rendu', () => {
  const files = SCANNED_FEATURES.flatMap((f) => collectSourceFiles(join(FEATURES_ROOT, f)))

  it('scanne au moins un fichier par feature (le walker fonctionne)', () => {
    expect(files.length).toBeGreaterThan(0)
  })

  for (const file of files) {
    const rel = file.slice(FEATURES_ROOT.length + 1).replace(/\\/g, '/')
    it(`${rel} n'affiche aucune clé Field*`, () => {
      const src = readFileSync(file, 'utf8')
      expect(src).not.toMatch(RAW_FIELD_KEY)
    })
  }
})
