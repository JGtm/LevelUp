/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (CLAUDE.md n°6) : l'aide pédagogique du profil « Intensité » vit dans UNE
 * seule clé i18n (`common.charts.intensity_tooltip`), servie par
 * `intensityTooltipText`. Elle avait déjà divergé en trois copies (Sessions, Séries
 * temporelles, Escouade) avant d'être centralisée le 2026-09-09 : sans garde-rail, une
 * quatrième surface recopierait le texte plutôt que d'appeler le helper — c'est
 * exactement comme les trois premières sont nées.
 *
 * Le marqueur retenu est la phrase de découpage, présente dans les trois anciens
 * littéraux FR comme EN, et absente de tout autre texte du dépôt.
 *
 * Calqué sur `cumulativeFdaGap.guard.test.ts`.
 */
import { describe, it, expect } from 'vitest'
import { readdirSync, readFileSync } from 'node:fs'
import { join, resolve } from 'node:path'

/** Signature des anciens littéraux, FR et EN. */
const SLICES_LITERAL = /découpé en 10 tranches|split into 10 (equal )?slices/

/**
 * Le manifest TOML et son module généré PORTENT le texte : c'est leur rôle. Tout le
 * reste doit passer par `intensityTooltipText`.
 */
const ALLOWED = new Set(['common.toml', 'common.ts'])

function walk(dir: string): string[] {
  const out: string[] = []
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const full = join(dir, entry.name)
    if (entry.isDirectory()) {
      if (entry.name === 'node_modules') continue
      out.push(...walk(full))
    } else if (/\.(ts|tsx|toml)$/.test(entry.name) && !/\.test\.tsx?$/.test(entry.name)) {
      out.push(full)
    }
  }
  return out
}

describe('garde-rail intensityTooltipText (aide unique du profil Intensité)', () => {
  it('aucune recopie du texte hors du manifest et de son module généré', () => {
    const srcRoot = resolve(process.cwd(), 'src')
    const offenders: string[] = []
    for (const file of walk(srcRoot)) {
      const base = file.split(/[\\/]/).pop() ?? ''
      if (ALLOWED.has(base)) continue
      if (SLICES_LITERAL.test(readFileSync(file, 'utf8'))) {
        offenders.push(file.replace(srcRoot, 'src'))
      }
    }
    expect(offenders).toEqual([])
  })
})
