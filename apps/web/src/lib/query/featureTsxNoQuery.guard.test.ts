/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (plan architecture, étape 7.2) : aucun `useQuery` / `useQueries` (ni leurs
 * variantes infinie et suspense) écrit dans un composant `.tsx` de `features/`.
 *
 * Une requête écrite dans un composant duplique le hook de la feature : même clé de cache,
 * options qui divergent (la copie de l'Accueil n'avait pas le garde `enabled` du hook et
 * pouvait partir avec un identifiant de joueur vide). Les lectures vivent dans `queries.ts`
 * ou `hooks/` de la feature ; le composant appelle le hook.
 *
 * Plafond 0, sans allowlist. Périmètre : `src/features/**.tsx` hors tests.
 */
import { describe, expect, it } from 'vitest'
import { callsNamed, formatHits, parseSource, SCAN_TIMEOUT_MS, scanSources } from '../archScan.testutil'

const QUERY_HOOKS: ReadonlySet<string> = new Set([
  'useQuery',
  'useQueries',
  'useInfiniteQuery',
  'useSuspenseQuery',
  'useSuspenseQueries',
])

const HINT =
  "Déplacer la lecture dans le `queries.ts` ou le dossier `hooks/` de la feature et appeler le hook depuis le composant (clé dans lib/query/keys.ts)."

describe('garde-rail : pas de requête dans un composant de feature', () => {
  it('aucun useQuery / useQueries dans un .tsx de features/', () => {
    const hits = scanSources({ under: 'features', tsxOnly: true }).flatMap((s) => callsNamed(s, QUERY_HOOKS))
    expect(hits.length, formatHits(hits, HINT)).toBe(0)
  }, SCAN_TIMEOUT_MS)

  it('auto-preuve : le détecteur voit les appels visés et ignore commentaires, chaînes et autres noms', () => {
    const seen = (code: string): number => callsNamed(parseSource('features/x/Comp.tsx', code), QUERY_HOOKS).length
    expect(seen("const q = useQuery({ queryKey: ['a'] })")).toBe(1)
    expect(seen('const r = useQueries({ queries: [] })')).toBe(1)
    expect(seen('const r = rq.useQuery({})')).toBe(1)
    expect(seen("const q = useInfiniteQuery({ queryKey: ['a'] })")).toBe(1)
    expect(seen('// useQuery({})\nconst a = 1')).toBe(0)
    expect(seen("const s = 'useQuery('")).toBe(0)
    expect(seen('const q = useChallenges(id)')).toBe(0)
    expect(seen('const useQueryLike = 1')).toBe(0)
  })
})
