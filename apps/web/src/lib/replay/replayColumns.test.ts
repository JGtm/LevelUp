/// <reference types="node" />
// @vitest-environment node
/**
 * Tests — `decodeReplayColumns` : le décodeur de la forme en colonnes du rejeu.
 *
 * La preuve qui compte vient du serveur : les fixtures `fixtures/go_columns/` sont ÉCRITES par
 * l'encodeur Go (`replayview.EncodeColumns`, avec un test de fraîcheur côté Go) à partir des documents
 * de `fixtures/go/`. Décodées ici, elles doivent redonner EXACTEMENT ces documents : mêmes clés dans le
 * même ordre, mêmes valeurs (`-0` devient `0`, décision documentée de l'encodeur).
 */
import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { gunzipSync } from 'node:zlib'
import { describe, expect, it } from 'vitest'

import { goFixtureEntries, goFixturesDir, loadGoFixture } from '@/features/match-replay/test/goFixtures'

import { decodeReplayColumns } from './replayColumns'

const columnsDir = join(goFixturesDir(), '..', 'go_columns')

function readColumnsFixture(film: string): unknown {
  return JSON.parse(gunzipSync(readFileSync(join(columnsDir, `columns_${film}.json.gz`))).toString('utf8'))
}

/** Premier chemin où deux valeurs diffèrent (ordre des clés compris ; `-0` et `0` sont égaux), ou null. */
function firstDifference(a: unknown, b: unknown, path = '$'): string | null {
  if (typeof a === 'number' && typeof b === 'number') return a === b ? null : `${path} (${a} / ${b})`
  if (a === null || b === null || typeof a !== 'object' || typeof b !== 'object') {
    return a === b ? null : `${path} (${String(a)} / ${String(b)})`
  }
  if (Array.isArray(a) !== Array.isArray(b)) return `${path} (liste / objet)`
  if (Array.isArray(a) && Array.isArray(b)) {
    if (a.length !== b.length) return `${path} (longueur ${a.length} / ${b.length})`
    for (let i = 0; i < a.length; i++) {
      const d = firstDifference(a[i], b[i], `${path}[${i}]`)
      if (d) return d
    }
    return null
  }
  const ka = Object.keys(a)
  const kb = Object.keys(b)
  if (ka.length !== kb.length) return `${path} (${ka.length} clés / ${kb.length})`
  for (let i = 0; i < ka.length; i++) {
    if (ka[i] !== kb[i]) return `${path} (clé ${ka[i]} / ${kb[i]})`
    const d = firstDifference((a as Record<string, unknown>)[ka[i]], (b as Record<string, unknown>)[kb[i]], `${path}.${ka[i]}`)
    if (d) return d
  }
  return null
}

describe('decodeReplayColumns — fixtures écrites par le serveur', () => {
  const films = ['bcb6d393', '000d5950']
  for (const film of films) {
    it(`${film} : le document décodé est le document de la forme normale`, () => {
      const entry = goFixtureEntries().find((e) => e.film === film)
      expect(entry, `fixture ${film} absente du manifeste`).toBeDefined()
      const encoded = readColumnsFixture(film) as Record<string, unknown>
      expect(encoded.$enc).toBe(1)
      const normal = loadGoFixture(entry!).doc
      const decoded = decodeReplayColumns(encoded)
      expect(firstDifference(normal, decoded)).toBeNull()
      expect(decoded as Record<string, unknown>).not.toHaveProperty('$enc')
    })
  }
})

describe('decodeReplayColumns — cas écrits à la main', () => {
  it('un document sans $enc est rendu tel quel, même référence', () => {
    const doc = { matchId: 'm', tracks: [{ points: [{ x: 1 }] }] }
    expect(decodeReplayColumns(doc)).toBe(doc)
    const liste: unknown[] = []
    expect(decodeReplayColumns(liste)).toBe(liste)
  })

  it('une version de forme inconnue est refusée, jamais décodée à tort', () => {
    expect(() => decodeReplayColumns({ $enc: 2, a: 1 })).toThrow(/inconnue/)
  })

  it('absent, null et 0 restent trois choses (colonne creuse en longueurs alternées)', () => {
    const encoded = {
      $enc: 1,
      items: { $c: 4, k: ['a', 'b'], c: [{ v: [0, null], a: [0, 1, 2, 1] }, [1, 2, 3, 4]] },
    }
    expect(decodeReplayColumns(encoded)).toStrictEqual({
      items: [{ b: 1 }, { a: 0, b: 2 }, { a: null, b: 3 }, { b: 4 }],
    })
  })

  it('absence en chaîne de 0 et de 1', () => {
    const encoded = { $enc: 1, items: { $c: 5, k: ['h'], c: [{ v: [7, 8], a: '01001' }] } }
    expect(decodeReplayColumns(encoded)).toStrictEqual({ items: [{}, { h: 7 }, {}, {}, { h: 8 }] })
  })

  it('échelle et différences : 1000, 1033, 1066 à 2 décimales', () => {
    const encoded = {
      $enc: 1,
      items: { $c: 3, k: ['t', 'x'], c: [{ v: [1000, 33, 33], d: 1 }, { v: [125, 200, 5], s: 2, d: 1 }] },
    }
    const out = decodeReplayColumns<unknown>(encoded) as { items: Array<{ t: number; x: number }> }
    expect(out.items.map((r) => r.t)).toEqual([1000, 1033, 1066])
    expect(out.items.map((r) => r.x)).toEqual([1.25, 3.25, 3.3])
  })

  it('la division par 10^s rend le double du jeton décimal (valeurs non représentables comprises)', () => {
    const jetons = ['0.1', '0.2', '0.3', '2.675', '123.456', '-0.07', '1234567.891']
    const ints = [100, 200, 300, 2675, 123456, -70, 1234567891]
    const encoded = { $enc: 1, items: { $c: ints.length, k: ['v'], c: [{ v: ints, s: 3 }] } }
    const out = decodeReplayColumns<unknown>(encoded) as { items: Array<{ v: number }> }
    expect(out.items.map((r) => r.v)).toEqual(jetons.map(Number))
  })

  it('n-uplets numériques', () => {
    const encoded = { $enc: 1, pts: { $c: 3, a: 2, c: [[1, 2, 3], { v: [5, 15, 25], s: 1 }] } }
    expect(decodeReplayColumns(encoded)).toStrictEqual({ pts: [[1, 0.5], [2, 1.5], [3, 2.5]] })
  })

  it('tables imbriquées : une colonne brute porte des valeurs elles-mêmes en colonnes', () => {
    const inner = { $c: 2, k: ['x'], c: [[1, 2]] }
    const encoded = { $enc: 1, items: { $c: 2, k: ['id', 'pts'], c: [[10, 20], [inner, inner]] } }
    expect(decodeReplayColumns(encoded)).toStrictEqual({
      items: [{ id: 10, pts: [{ x: 1 }, { x: 2 }] }, { id: 20, pts: [{ x: 1 }, { x: 2 }] }],
    })
  })

  it('des listes de scalaires et des objets simples restent intacts', () => {
    const encoded = { $enc: 1, a: [1, 2, 3], b: { c: 'x', d: null }, e: [] }
    expect(decodeReplayColumns(encoded)).toStrictEqual({ a: [1, 2, 3], b: { c: 'x', d: null }, e: [] })
  })

  it('une colonne mal formée lève une erreur claire', () => {
    expect(() => decodeReplayColumns({ $enc: 1, t: { $c: 1, k: ['a'], c: [{ nope: 1 }] } })).toThrow(/colonne mal formée/)
  })
})
