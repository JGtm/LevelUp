/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (CLAUDE.md n° 6, règle « ≤ 2 copies » → helper + garde-rail) : la fixture
 * « document de rejeu minimal passé par la frontière » a UNE source, `test/testDoc.ts`.
 *
 * POURQUOI CE GARDE. Elle était écrite deux fois à l'identique (rosterLogic.test,
 * replayLogic.test) et les fiches enrichies en réclamaient deux de plus. Quatre copies
 * d'un document minimal, ce sont quatre défauts qui divergent en silence — et un test
 * qui croit décrire le document du serveur alors qu'il décrit celui d'un autre test.
 *
 * Ce test échoue si un test de la feature rappelle `normalizeReplayDocument` à la main
 * au lieu de passer par la fixture.
 */
import { describe, expect, it } from 'vitest'
import { cheminCourt, fichierNomme, lire, nomDe, testsDeLaFeature } from './test/featureFiles'

// La signature de la copie : normaliser soi-même un document dans un fichier de test.
const REBUILD = /normalizeReplayDocument\s*\(/

// Seul le test de la frontière elle-même l'appelle légitimement hors fixture.
const ALLOWED = new Set(['testDoc.guard.test.ts', 'replayContract.test.ts'])

describe('garde-rail : une seule fixture de document de rejeu', () => {
  it('aucun test de la feature ne renormalise un document à la main', () => {
    const fautifs = testsDeLaFeature()
      .filter((f) => !ALLOWED.has(nomDe(f)))
      .filter((f) => REBUILD.test(lire(f)))
      .map(cheminCourt)
    expect(fautifs).toEqual([])
  })

  it('et la fixture, elle, passe bien par la frontière — sans quoi ce test ne garderait rien', () => {
    expect(REBUILD.test(lire(fichierNomme('testDoc.ts')))).toBe(true)
  })
})
