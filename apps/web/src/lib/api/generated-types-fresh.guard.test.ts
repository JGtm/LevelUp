/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail CI — `generated.ts` DÉRIVE bien de `apps/go-api/api/openapi.yaml`.
 *
 * Chaînon manquant du verrouillage du contrat (contre-revue V72) : `openapi-check`
 * (Go) verrouille openapi.yaml ← code Go, `contract-surface.guard.test.ts` verrouille
 * les DISPARITIONS de surface — mais rien n'imposait de rejouer `make generate-types`
 * après une évolution de contrat. Un openapi.yaml committé sans régénération laissait
 * le front typé sur l'ANCIEN contrat, tsc vert (types périmés ≠ types incohérents).
 *
 * Point d'accroche : la CI n'exécute AUCUNE cible make ; le job Frontend, lui, lance
 * `npm run test:coverage`. Ce test est donc l'unique verrou réellement joué en CI. La
 * logique vit dans `tools/check-generated-types-fresh.mjs` (script partagé avec
 * `make openapi-check`) — pas de seconde implémentation ici.
 *
 * Réparation d'un échec :  cd apps/web && npm run generate-types
 * (précédé de `make openapi-gen` si c'est le contrat Go qui a bougé).
 */
import { describe, it, expect } from 'vitest'
// @ts-expect-error — script Node hors du programme tsc d'apps/web (frontière ESM/TS).
import { checkGeneratedTypesFresh } from '../../../../../tools/check-generated-types-fresh.mjs'

describe('contrat généré — generated.ts à jour vis-à-vis de openapi.yaml', () => {
  it(
    'la régénération openapi-typescript ne produit AUCUN diff',
    () => {
      const problem = checkGeneratedTypesFresh() as string | null
      expect(problem, problem ?? '').toBeNull()
    },
    30_000, // le générateur relit tout le contrat (~1 s en local, marge pour la CI)
  )
})
