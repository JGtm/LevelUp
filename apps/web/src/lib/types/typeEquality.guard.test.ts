/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (CLAUDE.md n° 6) : le motif d'égalité stricte de types a UN foyer,
 * `lib/types/typeEquality.ts`.
 *
 * POURQUOI CE GARDE. La centralisation SANS garde-rail re-diverge — c'est la leçon écrite du
 * dépôt (un prédicat passé de 8 à 36 copies après sa centralisation). Le motif est court, on le
 * réécrit de mémoire en dix secondes, et deux copies qui divergent d'un `extends` ne prouvent
 * plus la même chose : l'une attrape un `any` là où l'autre le laisse passer.
 */
import { readdirSync, readFileSync } from 'node:fs'
import { join, resolve } from 'node:path'

import { describe, expect, it } from 'vitest'

/**
 * La signature du motif — SA FORME, jamais les noms de ses paramètres.
 *
 * ELLE EXIGEAIT LE LITTÉRAL `A` COMME SECOND TERME, et c'est le constat C2 de la revue ronde 1
 * du lot 0.B : une copie écrite `<T>() => T extends L ? 1 : 2` (paramètres `L`/`R` au lieu de
 * `A`/`B`) traversait le garde, verte. Un garde qui ne reconnaît le motif que sous les noms
 * d'une seule copie ne garde que cette copie — et le helper re-diverge exactement comme la
 * règle n° 6 le prédit. Ce qui identifie le motif, c'est le conditionnel différé lui-même :
 * une fonction générique qui compare `T` à un identifiant et rend `1` ou `2`.
 */
const MOTIF = new RegExp(
  '<T>\\(\\)\\s*=>\\s*T\\s+extends\\s+[A-Za-z_$][\\w$]*\\s*\\?\\s*1\\s*:\\s*2',
)

/** Le seul fichier qui a le droit de le porter. */
const FOYER = 'typeEquality.ts'

function racineSrc(): string {
  return resolve(__dirname, '..', '..')
}

function fichiers(dir: string): string[] {
  const out: string[] = []
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const full = join(dir, entry.name)
    if (entry.isDirectory()) {
      if (entry.name === 'node_modules') continue
      out.push(...fichiers(full))
    } else if (/\.(ts|tsx)$/.test(entry.name)) {
      out.push(full)
    }
  }
  return out
}

describe('garde-rail : une seule égalité stricte de types', () => {
  it('aucun fichier ne réécrit le motif hors du foyer', () => {
    const fautifs = fichiers(racineSrc())
      .filter((f) => !f.endsWith(FOYER) && !f.endsWith('typeEquality.guard.test.ts'))
      .filter((f) => MOTIF.test(readFileSync(f, 'utf8')))
      .map((f) => f.slice(racineSrc().length + 1).split('\\').join('/'))
    expect(
      fautifs,
      `ces fichiers réécrivent l'égalité stricte de types : [${fautifs.join(', ')}]. ` +
        `Importer Equals / Expect depuis @/lib/types/typeEquality.`,
    ).toEqual([])
  })

  it('et le foyer, lui, le porte bien — sans quoi ce garde ne garderait rien', () => {
    expect(MOTIF.test(readFileSync(join(racineSrc(), 'lib', 'types', FOYER), 'utf8'))).toBe(true)
  })

  it('reconnaît le motif SOUS N’IMPORTE QUELS NOMS DE PARAMÈTRES (constat C2)', () => {
    const copies = [
      'type Equals<A, B> = (<T>() => T extends A ? 1 : 2) extends <T>() => T extends B ? 1 : 2',
      'type _Egaux<L, R> = (<T>() => T extends L ? 1 : 2) extends <T>() => T extends R ? 1 : 2',
      'type Same<Gauche, Droite> = (<T>() => T extends Gauche ? 1 : 2) extends never',
    ]
    for (const copie of copies) {
      expect(MOTIF.test(copie), copie).toBe(true)
    }
  })

  it('et ne crie pas sur ce qui n’est pas le motif', () => {
    for (const innocent of [
      "import type { Equals, Expect } from '@/lib/types/typeEquality'",
      'type Assignable<A, B> = A extends B ? true : false',
      'const x = items.filter((T) => T.extends)',
    ]) {
      expect(MOTIF.test(innocent), innocent).toBe(false)
    }
  })
})
