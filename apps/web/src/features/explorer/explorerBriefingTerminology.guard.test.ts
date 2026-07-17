/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail terminologie briefing Explorer V2 (CLAUDE.md n°1 « FR sans
 * anglicismes » + n°6 anti-divergence). Les libellés retirés en V2 ne doivent
 * PLUS réapparaître dans le manifest `explorer.toml` ni dans les composants du
 * briefing :
 *   - FR « Par playlist »  → remplacé par « Par sélection » (dimension playlist) ;
 *   - « Pronostic » / « Prognosis » → remplacés par « Classement » / « Ranking »
 *     (le module ex-« Pronostic » devient la carte « Classement », D-A 2026-07-16).
 *
 * NB : l'EN « By playlist » (dimension) reste valide et n'est pas prohibé.
 */
import { describe, it, expect } from 'vitest'
import { readdirSync, readFileSync } from 'node:fs'
import { join, resolve } from 'node:path'

const FORBIDDEN: RegExp[] = [/Par playlist/, /Pronostic/, /Prognosis/]

/** Manifest i18n explorer + composants nommés « *briefing* » (hors fichiers de test). */
function briefingSources(): string[] {
  const root = resolve(process.cwd(), 'src')
  const out: string[] = [join(root, 'lib', 'i18n', 'manifests', 'explorer.toml')]
  const featDir = join(root, 'features', 'explorer')
  for (const entry of readdirSync(featDir, { withFileTypes: true })) {
    if (!entry.isFile()) continue
    if (!/\.(ts|tsx)$/.test(entry.name) || /\.test\.tsx?$/.test(entry.name)) continue
    if (/briefing/i.test(entry.name)) out.push(join(featDir, entry.name))
  }
  return out
}

describe('garde-rail terminologie briefing Explorer V2', () => {
  it('aucun libellé retiré (« Par playlist » / « Pronostic » / « Prognosis »)', () => {
    const offenders: string[] = []
    for (const file of briefingSources()) {
      const content = readFileSync(file, 'utf8')
      for (const rx of FORBIDDEN) {
        if (rx.test(content)) offenders.push(`${file.split(/[\\/]/).pop()} :: ${rx.source}`)
      }
    }
    expect(offenders, `Libellés retirés à corriger : ${offenders.join(', ')}`).toEqual([])
  })
})
