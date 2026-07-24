/**
 * Garde-rail — foyer canonique de SortableTh (règle ≤2 copies, CLAUDE.md n°6).
 *
 * SortableTh (en-tête de colonne triable — bouton + flèche + aria-sort) est défini
 * UNE seule fois dans components/ui/sortable-th.tsx et importé par ses consommateurs
 * (tableaux Carrière + petites tables admin, I16). Ce test interdit toute RÉ-INLINE
 * de la primitive : re-déclarer `function SortableTh` ailleurs re-diverge (leçon
 * prédicat bot : 8 → 36 copies — cf. metric-trend.guard.test.ts, même contrat).
 *
 * EXEMPTION datée : features/leaderboard/LeaderboardBlock.tsx a sa PROPRE copie
 * locale de SortableTh, antérieure à ce garde-rail (2026-07 ou avant) et HORS
 * périmètre du chantier I16 (cf. plan — LeaderboardBlock cité comme référence de
 * pattern, pas comme fichier à migrer). Dette notée, pas traitée ici — à fusionner
 * dans un futur passage si LeaderboardBlock est retouché.
 */
import { describe, it, expect } from 'vitest'

// import.meta.glob (Vite) charge chaque source comme chaîne brute — pas de
// dépendance à node:fs ni aux types node dans le tsconfig applicatif.
const sources = import.meta.glob('/src/**/*.{ts,tsx}', {
  query: '?raw',
  import: 'default',
  eager: true,
}) as Record<string, string>

const CANONICAL = '/src/components/ui/sortable-th.tsx'
// Legacy pré-existant, hors périmètre I16 (cf. doc ci-dessus) — daté 2026-07-24.
const DATED_ALLOWLIST = new Set(['/src/features/leaderboard/LeaderboardBlock.tsx'])

describe('garde-rail SortableTh canonique', () => {
  it('SortableTh n’est déclaré que dans components/ui/sortable-th.tsx (+ allowlist datée)', () => {
    const decl = /function\s+SortableTh\b|const\s+SortableTh\s*=/
    const offenders = Object.entries(sources)
      .filter(([path]) => path !== CANONICAL && !DATED_ALLOWLIST.has(path))
      .filter(([, code]) => decl.test(code))
      .map(([path]) => path)
    expect(offenders, `SortableTh ré-inliné hors du foyer canonique : ${offenders.join(', ')}`).toEqual([])
  })
})
