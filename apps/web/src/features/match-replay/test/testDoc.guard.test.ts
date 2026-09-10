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
import { cheminCourt, fichierNomme, lire, nomDe, testsDeLaFeature } from './featureFiles'

// La signature de la copie : normaliser soi-même un document dans un fichier de test.
const REBUILD = /normalizeReplayDocument\s*\(/

/**
 * Les tests qui appellent la frontière légitimement, hors fixture :
 *  - `replayContract.test.ts` : c'est le test DE la frontière.
 *
 * RETIRÉ le 2026-09-10 (lot hygiène 5.3, `.ai/V7.5/REGISTRE_REPORTS.md`, L577) :
 * `replayModel.bench.test.ts` (mesure E2.3 du plan « frise, point de vue », 2026-09-06,
 * chantier clos) lisait un artefact HORS DÉPÔT et se sautait donc toujours en CI — la mesure
 * qu'il produisait est déjà écrite en dur au-dessus de la mémo qu'elle a justifiée
 * (`useReplayModel.ts`). Un test qui se saute toujours en CI est du code mort (CLAUDE.md,
 * diagnostic n°1) : supprimé plutôt que converti en fixture, puisque son en-tête disait
 * lui-même n'être PAS un test de non-régression.
 */
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
