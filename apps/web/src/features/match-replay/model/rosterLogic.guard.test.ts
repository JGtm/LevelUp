/**
 * Garde-rail — foyer canonique du REPORT DE LECTURE (règle ≤2 copies, CLAUDE.md n°6).
 *
 * Trois calques du rejeu affichent, à une image donnée, la dernière lecture connue d'un
 * SLOT : les armes portées, l'inventaire, et la capacité d'armure. La logique est délicate
 * et identique aux trois — chercher par slot (jamais par joueur, sinon une dotation survit
 * à son porteur), puis, avant la première lecture d'une vie, replier sur la plus proche À
 * VENIR avec un âge NÉGATIF.
 *
 * Elle a été écrite deux fois avant d'être centralisée dans `nearestReading` ; ce test
 * interdit la troisième. Une factorisation sans garde-rail re-diverge — leçon du prédicat
 * bot, passé de 8 à 36 copies après centralisation.
 */
import { describe, it, expect } from 'vitest'

// import.meta.glob (Vite) charge chaque source comme chaîne brute — pas de dépendance à
// node:fs ni aux types node dans le tsconfig applicatif.
const sources = import.meta.glob('/src/**/*.{ts,tsx}', {
  query: '?raw',
  import: 'default',
  eager: true,
}) as Record<string, string>

const CANONICAL = '/src/features/match-replay/model/rosterLogic.ts'

/**
 * Le littéral qui signe une ré-inline : le repli « la plus proche à venir » compare deux âges
 * négatifs. Personne ne l'écrit par hasard, et il est le cœur de la règle.
 */
const AHEAD_FALLBACK = /!ahead\s*\|\|\s*age\s*>\s*ahead\.age/

describe('garde-rail report de lecture canonique', () => {
  it('le repli « lecture à venir » n’est écrit que dans rosterLogic.ts', () => {
    const offenders = Object.entries(sources)
      .filter(([path]) => path !== CANONICAL)
      .filter(([, code]) => AHEAD_FALLBACK.test(code))
      .map(([path]) => path)
    expect(
      offenders,
      `report de lecture ré-inliné hors du foyer canonique : ${offenders.join(', ')} — ` +
        'utiliser nearestReading()',
    ).toEqual([])
  })

  it('rosterLogic.ts n’en garde qu’UNE seule écriture', () => {
    const code = sources[CANONICAL] ?? ''
    const hits = code.match(new RegExp(AHEAD_FALLBACK.source, 'g')) ?? []
    expect(hits.length, 'nearestReading doit rester le seul porteur de la règle').toBe(1)
  })
})
