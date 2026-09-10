/**
 * Garde-rail — clé d'identité canonique d'une entrée de roster (règle ≤2 copies, CLAUDE.md n°6).
 *
 * `rosterEntryKey` (ce fichier) dérive la clé d'un bot par le littéral `` `bot:${name}` ``
 * (`botKey`). Avant ce lot, `equipmentUsageLogic` et `playerCardReadings` en portaient chacun
 * une copie inline (centralisées par le lot R6), et `seatLogic.filmIndexByIdentity` en portait
 * une 3e (migrée le 2026-09-10, lot hygiène 5.3, `.ai/V7.5/REGISTRE_REPORTS.md`, L595). Ce test
 * interdit une 4e réapparition du littéral hors du foyer canonique.
 */
import { describe, it, expect } from 'vitest'

// import.meta.glob (Vite) charge chaque source comme chaîne brute — pas de dépendance à
// node:fs ni aux types node dans le tsconfig applicatif.
const sources = import.meta.glob('/src/**/*.{ts,tsx}', {
  query: '?raw',
  import: 'default',
  eager: true,
}) as Record<string, string>

const CANONICAL = '/src/lib/replay/rosterLogic.ts'

/** Le littéral qui signe une ré-inline de la clé d'identité d'un bot de roster. */
const BOT_KEY_LITERAL = /`bot:\$\{/

describe('garde-rail clé de roster canonique (rosterEntryKey / botKey)', () => {
  it('le littéral `bot:${…}` n’est écrit que dans rosterLogic.ts', () => {
    const offenders = Object.entries(sources)
      .filter(([path]) => path !== CANONICAL)
      .filter(([path]) => !path.endsWith('.test.ts') && !path.endsWith('.test.tsx'))
      .filter(([, code]) => BOT_KEY_LITERAL.test(code))
      .map(([path]) => path)
    expect(
      offenders,
      `dérivation de clé de bot ré-inlinée hors du foyer canonique : ${offenders.join(', ')} — ` +
        'utiliser rosterEntryKey()/botKey() (lib/replay/rosterLogic.ts)',
    ).toEqual([])
  })

  it('rosterLogic.ts n’en garde qu’UNE seule écriture', () => {
    const code = sources[CANONICAL] ?? ''
    const hits = code.match(new RegExp(BOT_KEY_LITERAL.source, 'g')) ?? []
    expect(hits.length, 'botKey doit rester le seul porteur du littéral').toBe(1)
  })
})
