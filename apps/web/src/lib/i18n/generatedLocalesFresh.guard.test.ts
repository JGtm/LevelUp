/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail CI — les fichiers générés depuis `config/locales.toml` (ADR 0040) sont À JOUR :
 * `lib/i18n/generated/locales.ts` (ce module web) et `apps/go-api/internal/i18n/locales_gen.go`
 * (le miroir Go). La logique vit dans `tools/check-generated-types-fresh.mjs` (partagée avec
 * `make openapi-check`) ; le texte attendu est produit par `scripts/locales.mjs`, le module de
 * l'écrivain.
 *
 * Ce test est l'unique verrou joué en CI côté web, et il couvre aussi le fichier Go : la liste
 * web et la liste Go sont donc toutes deux comparées au MÊME fichier source. Le test Go
 * `internal/i18n/locales_gen_test.go` relit le même TOML : un écart se voit des deux côtés.
 *
 * Réparation d'un échec :  node apps/web/scripts/build_i18n_manifests.mjs
 */
import { describe, it, expect } from 'vitest'
// @ts-expect-error — script Node hors du programme tsc d'apps/web (frontière ESM/TS).
import { checkGeneratedLocalesFresh } from '../../../../../tools/check-generated-types-fresh.mjs'

describe('langues générées — locales.ts et locales_gen.go à jour vis-à-vis de config/locales.toml', () => {
  it('la régénération ne produit AUCUN diff', () => {
    const problem = checkGeneratedLocalesFresh() as string | null
    expect(problem, problem ?? '').toBeNull()
  })
})
