/**
 * Garde-rail — UN SEUL DÉCODEUR DE LA FORME EN COLONNES du rejeu (règle ≤ 2 copies, CLAUDE.md n° 6).
 *
 * POURQUOI. La forme en colonnes est un contrat entre le serveur (`replayview.EncodeColumns`) et ce
 * module : ses marqueurs, `$enc` à la racine et `$c` sur une table, sont ce que le décodeur reconnaît.
 * Un second lecteur de ces marqueurs ferait diverger les deux côtés sans qu'aucun test ne le voie.
 *
 * RÈGLE. Aucun fichier de production ne lit `$enc` ni `$c` (hors commentaires) sauf
 * `lib/replay/replayColumns.ts`. Allowlist d'exceptions : VIDE. Les tests (`*.test.ts[x]`) ne sont pas
 * balayés : ils portent des documents en colonnes comme données. Le pendant Go est
 * `archlint/no_second_columns_codec_test.go`.
 */
import { describe, expect, it } from 'vitest'

const sources = import.meta.glob('/src/**/*.{ts,tsx}', {
  query: '?raw',
  import: 'default',
  eager: true,
}) as Record<string, string>

const CANONICAL = '/src/lib/replay/replayColumns.ts'
const ALLOWLIST: readonly string[] = []

/** Les marqueurs de la forme, lus comme identifiants ou comme clés : `$enc`, `$c`. */
const MARKER = /\$(?:enc|c)(?![A-Za-z0-9_$])/

/** Le code sans ses commentaires (blocs et lignes), chaînes de commentaire comprises. */
function withoutComments(code: string): string {
  return code.replace(/\/\*[\s\S]*?\*\//g, '').replace(/(^|[^:])\/\/.*$/gm, '$1')
}

function isTest(path: string): boolean {
  return /\.test\.tsx?$/.test(path)
}

describe('garde-rail : un seul décodeur de la forme en colonnes (lib/replay/replayColumns.ts)', () => {
  it('aucun autre fichier de production ne lit $enc ni $c', () => {
    const offenders = Object.entries(sources)
      .filter(([path]) => path !== CANONICAL && !isTest(path) && !ALLOWLIST.includes(path))
      .filter(([, code]) => MARKER.test(withoutComments(code)))
      .map(([path]) => path)
    expect(
      offenders,
      `marqueur de la forme en colonnes lu hors du décodeur canonique : ${offenders.join(', ')} — ` +
        'passer par decodeReplayColumns (lib/replay/replayColumns).',
    ).toEqual([])
  })

  it('et le décodeur, lui, lit bien les deux marqueurs — sans quoi ce garde ne garderait rien', () => {
    const code = withoutComments(sources[CANONICAL] ?? '')
    expect(code).toMatch(/\$enc/)
    expect(code).toMatch(/\$c(?![A-Za-z0-9_$])/)
  })

  it('le motif reconnaît les formes à garder et ignore les mots voisins', () => {
    for (const s of ['$cache', '$encode', 'price$', 'a$c1']) expect(MARKER.test(s), s).toBe(false)
  })
})
