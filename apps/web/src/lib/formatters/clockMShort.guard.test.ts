/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (CLAUDE.md n° 6 : à la 3e copie, un helper canonique ET un garde-rail) —
 * LE FORMAT `MmSSs` A UN SEUL FOYER.
 *
 * POURQUOI CE GARDE (registre 2026-09-05, N3). Le littéral `${m}m${s.padStart(2,'0')}s`
 * était écrit QUATRE fois — trois dans `features/match-view/`, une dans `features/synthesis/`
 * — alors que `lib/formatters` porte déjà le foyer des durées. Aucune des quatre n'était
 * substituable telle quelle : deux prennent des millisecondes et deux des secondes, et elles
 * divergent au zéro (« - », `null`, ou « 0m00s »). Ce n'était donc pas une factorisation
 * abandonnée mais un helper INCOMPLET : la variante « instant » manquait. Elle existe
 * (`formatClockMShort`), les quatre sites l'appellent, et chacun garde CHEZ LUI la seule
 * décision qui lui appartient — l'unité d'entrée, ou ce qu'il fait d'une absence.
 *
 * `components/ui/match-card.tsx` n'est pas concerné : il écrit « 2m 05s », avec une espace,
 * un autre format — le motif ci-dessous ne le voit pas, et c'est voulu.
 */
import { describe, expect, it } from 'vitest'
import { readdirSync, readFileSync } from 'node:fs'
import { join, resolve } from 'node:path'

/**
 * La signature du format : `…}m${…padStart(2, '0')}s`. L'espace de `match-card` ne matche
 * pas (le motif exige `}m${` collés), et le zero-padding sur deux chiffres est ce qui
 * distingue cette famille de `formatDurationMinSec` (« 2min5s »).
 */
const LITTERAL = /\}m\$\{[^}]*padStart\(\s*2\s*,\s*'0'\s*\)\}s/

/** Le seul fichier autorisé à l'écrire, et le garde lui-même, qui le cite. */
const AUTORISES = new Set(['duration.ts', 'clockMShort.guard.test.ts'])

function walk(dir: string): string[] {
  const out: string[] = []
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const full = join(dir, entry.name)
    if (entry.isDirectory()) {
      if (entry.name === 'node_modules') continue
      out.push(...walk(full))
    } else if (/\.(ts|tsx)$/.test(entry.name)) {
      out.push(full)
    }
  }
  return out
}

describe('garde-rail : une seule écriture du format MmSSs', () => {
  it('le littéral n’apparaît nulle part ailleurs que dans lib/formatters/duration.ts', () => {
    const src = resolve(__dirname, '..', '..')
    const fautifs = walk(src).filter((f) => {
      const base = f.split(/[\\/]/).pop() ?? ''
      if (AUTORISES.has(base)) return false
      return LITTERAL.test(readFileSync(f, 'utf8'))
    })
    expect(fautifs).toEqual([])
  })

  it('et duration.ts, lui, le porte bien — sans quoi ce test ne garderait rien', () => {
    expect(LITTERAL.test(readFileSync(join(__dirname, 'duration.ts'), 'utf8'))).toBe(true)
  })
})
