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
import { join } from 'node:path'

import { describe, expect, it } from 'vitest'
import {
  cheminCourt,
  fichierNomme,
  fichiersSous,
  lire,
  nomDe,
  racineWeb,
  testsDeLaFeature,
} from './featureFiles'

// La signature de la copie : normaliser soi-même un document dans un fichier de test.
const REBUILD = /normalizeReplayDocument\s*\(/

/**
 * LES TROIS FORMES sous lesquelles un numéro de schéma s'écrit à la main.
 *
 * IL N'Y EN AVAIT QU'UNE, ET C'EST LE CONSTAT C3 de la revue ronde 1 : le garde ne voyait ni
 * l'appel POSITIONNEL (`computeReplaySchemaStatus(48, 51)` — la forme même que ce lot venait de
 * retirer de `replaySchemaStatusLogic.test.ts`) ni la CLÉ CITÉE (`"schemaVersion": 3`, celle
 * d'une réponse d'API simulée). Un garde qui ne connaît qu'une orthographe du défaut laisse
 * passer les deux autres, en restant vert.
 *
 * AUCUNE NE VISE AUTRE CHOSE QU'UN NOMBRE : `schemaVersion: number` (annotation de type),
 * `schemaVersion={PRODUCTEUR}` et `computeReplaySchemaStatus(PRODUCTEUR, ANCIENNE)` sont
 * exactement ce qu'on veut voir à la place, et ne doivent pas rougir.
 */
const VERSIONS_EN_DUR: { nom: string; motif: RegExp }[] = [
  { nom: "litteral d'objet ou prop JSX", motif: /(?:latestS|s)chemaVersion\s*(?::|=\{)\s*-?\d/ },
  { nom: 'cle citee', motif: /['"](?:latestS|s)chemaVersion['"]\s*:\s*-?\d/ },
  { nom: 'appel positionnel', motif: /computeReplaySchemaStatus\s*\(\s*-?\d/ },
]

/** Un fichier porte-t-il l'une des trois formes ? */
function ecritUneVersionEnDur(src: string): boolean {
  return VERSIONS_EN_DUR.some((s) => s.motif.test(src))
}

/**
 * LES TESTS DU REJEU, et ils ne vivent pas tous dans la feature — constat C4 de la même revue.
 * Le balayage s'arrêtait à `features/match-replay/`, si bien que deux tests du rejeu écrivaient
 * encore un numéro à la main sans rougir : `lib/replay/heatPaint.test.ts` et la porte de la
 * route `.../matches/$matchId/replay.gate.test.tsx`. L'item 0.B.6 dit « les tests WEB », pas
 * « les tests de la feature ».
 *
 * TROIS RACINES, et pas `src/` entier : la frontière du rejeu (`lib/replay/`), sa page
 * (`routes/`) et sa feature. Un `schemaVersion` ailleurs dans le dépôt parlerait d'autre chose
 * — le jour où il en apparaîtrait un, il se discutera, il ne se subira pas.
 */
function testsDuRejeu(): string[] {
  const racines = [join(racineWeb(), 'src', 'lib', 'replay'), join(racineWeb(), 'src', 'routes')]
  const hors = racines.flatMap(fichiersSous).filter((f) => /\.test\.(ts|tsx)$/.test(f))
  return [...testsDeLaFeature(), ...hors]
}

/** Le chemin affiché : relatif à `src/`, pour situer un fautif hors de la feature. */
function cheminAffiche(f: string): string {
  const src = join(racineWeb(), 'src')
  return f.startsWith(src) ? f.slice(src.length + 1).split('\\').join('/') : cheminCourt(f)
}

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
  it('aucun test du rejeu ne fixe une version de schéma en dur', () => {
    const fautifs = testsDuRejeu()
      .filter((f) => !ALLOWED.has(nomDe(f)))
      .filter((f) => ecritUneVersionEnDur(lire(f)))
      .map(cheminAffiche)
    expect(
      fautifs,
      `ces tests écrivent une version de schéma en dur : [${fautifs.join(', ')}]. ` +
        `La seule source est le manifeste produit par Go — cf. goFixtureSchemaVersion().`,
    ).toEqual([])
  })

  it('et le balayage sort bien de la feature — sans quoi C4 se rejouerait', () => {
    const hors = testsDuRejeu()
      .map(cheminAffiche)
      .filter((c) => c.startsWith('lib/replay/') || c.startsWith('routes/'))
    expect(hors.length).toBeGreaterThan(0)
    expect(hors.some((c) => c.startsWith('lib/replay/'))).toBe(true)
    expect(hors.some((c) => c.startsWith('routes/'))).toBe(true)
  })

  it('et la fixture partagée prend bien la sienne du producteur', () => {
    const src = lire(fichierNomme('testDoc.ts'))
    expect(ecritUneVersionEnDur(src)).toBe(false)
    expect(src).toContain('goFixtureSchemaVersion()')
  })

  it('les trois signatures attrapent bien ce qu’elles visent — sinon ce garde serait inerte', () => {
    const fautifs = [
      'schemaVersion: 1',
      'schemaVersion={48}',
      'latestSchemaVersion: 51',
      '"schemaVersion": 3',
      "'latestSchemaVersion': 12",
      'computeReplaySchemaStatus(48, 51)',
      'expect(computeReplaySchemaStatus( 48 , undefined))',
    ]
    for (const fautif of fautifs) {
      expect(ecritUneVersionEnDur(fautif), fautif).toBe(true)
    }
    const legitimes = [
      'schemaVersion: number',
      'schemaVersion={PRODUCTEUR}',
      'schemaVersion: MIN_RENDERABLE_SCHEMA_VERSION - 1',
      'computeReplaySchemaStatus(PRODUCTEUR, ANCIENNE)',
      'schemaVersion: goFixtureSchemaVersion(),',
    ]
    for (const legitime of legitimes) {
      expect(ecritUneVersionEnDur(legitime), legitime).toBe(false)
    }
  })
})
