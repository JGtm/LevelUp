/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (plan architecture, étape 7.11) : une feature n'importe jamais le module d'une route.
 *
 * Les routes importent les features (la page est le composant de la route) ; l'inverse ferme un cycle
 * `routes -> feature -> routes` que seul l'ordre d'évaluation des modules empêche d'exploser. Une page
 * qui lit ses paramètres ou sa recherche passe par `getRouteApi('<chemin>')`.
 *
 * Plafond 0 : aucune dérogation.
 */
import { describe, expect, it } from 'vitest'
import { moduleSpecifiers, parseSource, scanSources, SCAN_TIMEOUT_MS } from '@/lib/archScan.testutil'
import type { ScannedSource } from '@/lib/archScan.testutil'

const ROUTE_MODULE = /^@\/routes(?:\/|$)/

function routeImports(source: ScannedSource): string[] {
  return moduleSpecifiers(source)
    .filter(({ specifier }) => ROUTE_MODULE.test(specifier))
    .map(({ specifier, line }) => `${source.rel}:${line} importe ${specifier}`)
}

describe('garde-rail : aucune feature n’importe un module de route', () => {
  it('plafond 0 sous features/', () => {
    const hits = scanSources({ under: 'features' }).flatMap(routeImports)
    expect(hits, `Passer par getRouteApi('<chemin>') :\n${hits.join('\n')}`).toEqual([])
  }, SCAN_TIMEOUT_MS)

  it('auto-preuve : le détecteur voit l’import, la valeur comme le type, pas un autre alias', () => {
    const n = (code: string): number => routeImports(parseSource('features/x/a.tsx', code)).length
    expect(n("import { Route } from '@/routes/join'")).toBe(1)
    expect(n("import type { Route } from '@/routes/x/y'")).toBe(1)
    expect(n("import { x } from '@/routesmap/join'")).toBe(0)
    expect(n("import { x } from '@/lib/routes'")).toBe(0)
  })
})
