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
 *
 * SECOND VOLET (2026-09-13, lot 0.B) : AUCUN NUMÉRO DE SCHÉMA ÉCRIT À LA MAIN dans les tests
 * de la feature. La fixture partagée a déclaré `schemaVersion: 1` pendant des mois — un
 * document que le serveur n'a JAMAIS servi — sous un commentaire qui énonçait pourtant le
 * principe inverse. Un numéro écrit à la main est une version inventée : il ne suit aucune
 * montée, et le test qui le porte finit par décrire un contrat périmé en restant vert. La
 * seule source est désormais ce que Go publie (`test/fixtures/go/`, lu par `goFixtures.ts`) ;
 * ces fixtures sont du JSON, donc hors du balayage de ce garde par construction.
 */
import { describe, expect, it } from 'vitest'
import { cheminCourt, fichierNomme, lire, nomDe, testsDeLaFeature } from './featureFiles'

// La signature de la copie : normaliser soi-même un document dans un fichier de test.
const REBUILD = /normalizeReplayDocument\s*\(/

/**
 * La signature du numéro écrit à la main : `schemaVersion` (ou `latestSchemaVersion`)
 * immédiatement suivi d'un NOMBRE, par `:` (littéral d'objet) ou par `={` (prop JSX).
 *
 * ELLE NE VISE QUE LES NOMBRES : `schemaVersion: number` (annotation de type) et
 * `schemaVersion={PRODUCTEUR}` (valeur dérivée du manifeste Go) sont exactement ce qu'on veut
 * voir à la place, et ne doivent pas rougir.
 */
const VERSION_EN_DUR = /(?:latestS|s)chemaVersion\s*(?::|=\{)\s*-?\d/

/**
 * Les tests qui appellent la frontière légitimement, hors fixture :
 *  - `replayContract.test.ts` : c'est le test DE la frontière ;
 *  - `goFixtures.contract.test.ts` (2026-09-13, lot 0.B) : il fait passer les documents
 *    PRODUITS PAR GO par cette même frontière — c'est tout son objet, et le faire via la
 *    fixture minimale reviendrait à ne plus tester le document réel.
 *
 * RETIRÉ le 2026-09-10 (lot hygiène 5.3, `.ai/V7.5/REGISTRE_REPORTS.md`, L577) :
 * `replayModel.bench.test.ts` (mesure E2.3 du plan « frise, point de vue », 2026-09-06,
 * chantier clos) lisait un artefact HORS DÉPÔT et se sautait donc toujours en CI — la mesure
 * qu'il produisait est déjà écrite en dur au-dessus de la mémo qu'elle a justifiée
 * (`useReplayModel.ts`). Un test qui se saute toujours en CI est du code mort (CLAUDE.md,
 * diagnostic n°1) : supprimé plutôt que converti en fixture, puisque son en-tête disait
 * lui-même n'être PAS un test de non-régression.
 */
const ALLOWED = new Set([
  'testDoc.guard.test.ts',
  'replayContract.test.ts',
  'goFixtures.contract.test.ts',
])

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

describe('garde-rail : aucun numéro de schéma écrit à la main', () => {
  it('aucun test de la feature ne fixe une version de schéma en dur', () => {
    const fautifs = testsDeLaFeature()
      .filter((f) => !ALLOWED.has(nomDe(f)))
      .filter((f) => VERSION_EN_DUR.test(lire(f)))
      .map(cheminCourt)
    expect(
      fautifs,
      `ces tests écrivent une version de schéma en dur : [${fautifs.join(', ')}]. ` +
        `La seule source est le manifeste produit par Go — cf. goFixtureSchemaVersion().`,
    ).toEqual([])
  })

  it('et la fixture partagée prend bien la sienne du producteur', () => {
    const src = lire(fichierNomme('testDoc.ts'))
    expect(VERSION_EN_DUR.test(src)).toBe(false)
    expect(src).toContain('goFixtureSchemaVersion()')
  })

  it('la signature attrape bien ce qu’elle vise — sans quoi ce garde serait inerte', () => {
    for (const fautif of ['schemaVersion: 1', 'schemaVersion={48}', 'latestSchemaVersion: 51']) {
      expect(VERSION_EN_DUR.test(fautif), fautif).toBe(true)
    }
    for (const legitime of [
      'schemaVersion: number',
      'schemaVersion={PRODUCTEUR}',
      'schemaVersion: MIN_RENDERABLE_SCHEMA_VERSION - 1',
    ]) {
      expect(VERSION_EN_DUR.test(legitime), legitime).toBe(false)
    }
  })
})
