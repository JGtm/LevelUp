/**
 * Garde-rail (release v8, T1-b) — aucun choix de comportement par COMPARAISON DE SLUG de titre
 * dans `apps/web/src`.
 *
 * Règle transverse (CLAUDE.md, ADR 0025) : on branche sur une capability (`useCapability`,
 * `hasCapabilityIn`, `RouteCapabilityGate`, `useDataCapability`), jamais sur
 * `slug === 'halo_5'` / `slug === 'halo_infinite'`. Une comparaison de slug code en dur la liste
 * des titres connus : un troisième titre, ou un titre qui obtient la même capacité, ne suit pas.
 * Le cas déclencheur : l'onglet Citations / Commendations, la home et la route de la page
 * Citations se décidaient sur `=== 'halo_5'` pendant que l'API n'avait aucune porte.
 *
 * Complète la règle ESLint `@levelup/no-title-slug-literal` (littéral `halo_infinite` seul,
 * `features/` + `components/` seulement) : ici les DEUX slugs, toute la source non-test, et la
 * FORME comparaison (`===`, `!==`, `==`, `!=`, `case 'x':`), sans toucher aux affectations ni
 * aux valeurs par défaut (`?? 'halo_infinite'`), qui ne choisissent aucun comportement.
 * Modèle côté Go : `internal/archlint/no_slug_comparison_test.go`.
 *
 * Pour ajouter une exception : le chemin exact dans `ALLOWLIST`, avec la date et la raison pour
 * laquelle une capability ne convient pas. Une entrée dont le fichier ne contient plus de
 * comparaison est refusée (liste blanche périmée).
 */
import { describe, it, expect } from 'vitest'

// import.meta.glob (Vite) charge chaque source comme chaîne brute (cf. spartan-customizer-strict.guard).
const sources = import.meta.glob('/src/**/*.{ts,tsx}', {
  query: '?raw',
  import: 'default',
  eager: true,
}) as Record<string, string>

const SLUG = String.raw`['"\`](?:halo_5|halo_infinite)['"\`]`
const OP = String.raw`(?:===|!==|==|!=)`

/** `x === 'halo_5'`, `'halo_infinite' !== x`, `case 'halo_5':`. */
const SLUG_COMPARISON = new RegExp(
  String.raw`${OP}\s*${SLUG}|${SLUG}\s*${OP}|(?:^|[^\w$.])case\s+${SLUG}\s*:`,
)

/** Sources qui ne sont pas du code livré : tests, mesures, témoins, garde-rails, fixtures, généré. */
const NOT_SHIPPED = /\.(?:test|spec|mesure|witness|guard)\.tsx?$|\/src\/test\/|\/generated(?:\/|\.ts$)/

/**
 * Exceptions datées. Le motif d'une exception est qu'aucune capability ne porte la décision.
 *
 * - `lib/match-nav/waypointUrl.ts` (2026-10-19) : table de correspondance slug -> segment d'URL
 *   publique de Halo Waypoint (`halo-5-guardians` / `halo-infinite`, `arena/`). La décision
 *   « ce titre a-t-il une page Waypoint ? » est déjà portée par la capability
 *   `waypoint_match_url` côté appelant ; ce fichier ne fait que traduire un identifiant interne
 *   en identifiant externe, ce qu'une capability (booléenne) ne sait pas exprimer.
 */
const ALLOWLIST = new Set<string>(['/src/lib/match-nav/waypointUrl.ts'])

function isCommentLine(line: string): boolean {
  const t = line.trim()
  return t.startsWith('//') || t.startsWith('*') || t.startsWith('/*')
}

function comparisonsIn(code: string): string[] {
  return code
    .split('\n')
    .map((line, i) => ({ line, n: i + 1 }))
    .filter(({ line }) => !isCommentLine(line) && SLUG_COMPARISON.test(line))
    .map(({ line, n }) => `${n}: ${line.trim()}`)
}

describe('garde-rail : aucune comparaison de slug de titre dans src/', () => {
  const shipped = Object.entries(sources).filter(([path]) => !NOT_SHIPPED.test(path))

  it('aucune comparaison `=== halo_5` / `=== halo_infinite` hors liste blanche', () => {
    expect(shipped.length).toBeGreaterThan(100) // le glob a bien trouvé le code
    const offenders = shipped
      .filter(([path]) => !ALLOWLIST.has(path))
      .flatMap(([path, code]) => comparisonsIn(code).map((hit) => `${path}:${hit}`))
    expect(
      offenders,
      `Choix de comportement par slug de titre : passer par une capability (useCapability / ` +
        `hasCapabilityIn / RouteCapabilityGate), ou exception datée dans ALLOWLIST :\n  ${offenders.join('\n  ')}`,
    ).toEqual([])
  })

  it('chaque entrée de la liste blanche contient encore une comparaison (pas de liste périmée)', () => {
    for (const path of ALLOWLIST) {
      const code = sources[path]
      expect(code, `fichier de la liste blanche introuvable : ${path}`).toBeDefined()
      expect(comparisonsIn(code).length, `liste blanche périmée : ${path}`).toBeGreaterThan(0)
    }
  })

  // Le garde-rail est lui-même testé : un filet qui ne mord plus est pire qu'aucun.
  it('détecte les formes de comparaison', () => {
    const cases = [
      `currentTitleSlug === 'halo_5'`,
      `s.currentTitleSlug !== "halo_infinite"`,
      `if (slug == 'halo_5') {`,
      `'halo_5' === titleSlug`,
      "title.slug === `halo_infinite`",
      `case 'halo_5':`,
    ]
    for (const code of cases) expect(SLUG_COMPARISON.test(code), `non détecté : ${code}`).toBe(true)
  })

  it('ne signale PAS les usages sans choix de comportement', () => {
    const cases = [
      `currentTitleSlug: 'halo_infinite',`,
      `const slug = data.current_title_slug ?? 'halo_infinite'`,
      `export const DEFAULT_TITLE_SLUG = 'halo_infinite'`,
      `registreDuTitre(slug = 'halo_infinite')`,
      `titleSlug === otherSlug`,
      `const kind = x === 'halo_5_extra'`,
    ]
    for (const code of cases) expect(SLUG_COMPARISON.test(code), `faux positif : ${code}`).toBe(false)
  })
})
