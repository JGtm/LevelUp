/**
 * Tests — le statut du badge admin « version de schéma ».
 *
 * AUCUN NUMÉRO DE SCHÉMA N'EST ÉCRIT EN DUR ICI (2026-09-13, lot 0.B) : les versions se
 * dérivent de celle que Go publie (`test/goFixtures.ts`) et du seuil de compatibilité déclaré
 * par le web. Le ratchet `test/testDoc.guard.test.ts` l'interdit désormais dans toute la
 * feature — un littéral écrit à la main est ce qui a fait vivre pendant des mois une fixture
 * partagée figée à la version 1, que le serveur n'a jamais servie.
 */
/// <reference types="node" />
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

import { describe, expect, it } from 'vitest'

import type { ReplayContractIssue } from '@/lib/replay/replayDocumentSchema'

import { racineDuDepot } from '../test/featureFiles'
import { goFixtureEntries, goFixtureSchemaVersion } from '../test/goFixtures'
import { testReplayDoc } from '../test/testDoc'
import { calquePresent, couchesDesCalquesProduits, revisionDuCalque } from './calquePresent'
import {
  computeReplaySchemaStatus,
  MIN_RENDERABLE_SCHEMA_VERSION,
} from './replaySchemaStatusLogic'

/** La version que le producteur écrit aujourd'hui. */
const PRODUCTEUR = goFixtureSchemaVersion()

/** La version d AVANT celle du producteur : jamais un numero ecrit a la main. */
const AVANT_LAYERS = PRODUCTEUR - 1

/** Une version ANCIENNE, mais au-dessus du seuil d'affichage. */
const ANCIENNE = MIN_RENDERABLE_SCHEMA_VERSION + 1

describe('computeReplaySchemaStatus', () => {
  it('rend "unknown" quand la version courante du producteur est absente', () => {
    expect(computeReplaySchemaStatus(ANCIENNE, undefined)).toEqual({
      kind: 'unknown',
      schemaVersion: ANCIENNE,
    })
  })

  it('rend "upToDate" quand l\'artefact porte déjà la version courante', () => {
    expect(computeReplaySchemaStatus(PRODUCTEUR, PRODUCTEUR)).toEqual({
      kind: 'upToDate',
      schemaVersion: PRODUCTEUR,
    })
  })

  it('rend "stale" quand l\'artefact est en retard sur le producteur', () => {
    expect(computeReplaySchemaStatus(ANCIENNE, PRODUCTEUR)).toEqual({
      kind: 'stale',
      schemaVersion: ANCIENNE,
      latestSchemaVersion: PRODUCTEUR,
    })
  })

  it('rend "upToDate" quand l\'artefact est PLUS RÉCENT que le binaire courant (poste de dev)', () => {
    expect(computeReplaySchemaStatus(PRODUCTEUR, ANCIENNE)).toEqual({
      kind: 'upToDate',
      schemaVersion: PRODUCTEUR,
    })
  })
})

describe('la matrice de compatibilité (MIN_RENDERABLE_SCHEMA_VERSION)', () => {
  it('dit "à recuire" sous le seuil MÊME sans en-tête à comparer, jamais "inconnu"', () => {
    const status = computeReplaySchemaStatus(MIN_RENDERABLE_SCHEMA_VERSION - 1, undefined)
    expect(status.kind).toBe('stale')
    // Aucune cible n'est inventée : le seuil de compatibilité n'est pas la version du jour.
    expect(status).toEqual({
      kind: 'stale',
      schemaVersion: MIN_RENDERABLE_SCHEMA_VERSION - 1,
      latestSchemaVersion: undefined,
    })
  })

  it('nomme la cible sous le seuil quand l\'en-tête la donne', () => {
    expect(computeReplaySchemaStatus(MIN_RENDERABLE_SCHEMA_VERSION - 1, PRODUCTEUR)).toEqual({
      kind: 'stale',
      schemaVersion: MIN_RENDERABLE_SCHEMA_VERSION - 1,
      latestSchemaVersion: PRODUCTEUR,
    })
  })

  it('le seuil lui-même est AFFICHABLE : c\'est un minimum inclusif', () => {
    expect(computeReplaySchemaStatus(MIN_RENDERABLE_SCHEMA_VERSION, undefined).kind).toBe('unknown')
  })

  it('et le producteur écrit toujours au-dessus du seuil — sans quoi rien ne serait affichable', () => {
    expect(PRODUCTEUR).toBeGreaterThanOrEqual(MIN_RENDERABLE_SCHEMA_VERSION)
  })
})

describe('la borne de compatibilité est ÉPINGLÉE, pas seulement écrite', () => {
  /**
   * LA VALEUR EST ÉPINGLÉE ICI, ET C'EST LE CONSTAT C5 de la revue ronde 1 du lot 0.B : la
   * faire passer de 27 à 50 laissait 202 fichiers et 3 016 tests verts. La décision centrale
   * du garde-rail « matrice de compatibilité » n'était donc tenue par RIEN — une constante
   * qu'aucun test ne regarde est un commentaire.
   *
   * D'OÙ VIENT 27. La chronique du producteur (`replay/document_chronicle.go`) ne porte que
   * DEUX montées qui RETIRENT au client un champ qu'on lui avait donné :
   *   - v6  : `Inventory.a` retiré plutôt que réinterprété (il portait `rang − 16`) ;
   *   - v27 : `weaponChanges[].until` retiré — une durée de table (10/20/30 s), convention
   *           refusée par l'utilisateur.
   * À partir de 27, toute montée est ADDITIVE ou ne change que le contenu : un artefact plus
   * ancien se rend, plus pauvre, jamais faux.
   *
   * CHANGER CETTE VALEUR N'EST PAS UNE RETOUCHE DE CODE. C'est : (1) une décision produit —
   * on cesse d'afficher une génération d'artefacts ; (2) une entrée de chronique qui NOMME le
   * retrait ou le renommage justifiant la nouvelle borne ; (3) une fixture produite par Go à
   * cette borne, pour que « au-dessus, ça rend » reste prouvé et non affirmé. Ce test échoue
   * tant que les trois ne sont pas faits.
   */
  it('vaut 27, la dernière version qui retire un champ lu par le web', () => {
    expect(MIN_RENDERABLE_SCHEMA_VERSION).toBe(27)
  })

  it('et cette version EXISTE dans la chronique du producteur', () => {
    // Les trois formes d'en-tête attestées de `document_chronicle.go` (cf.
    // `testutil.ReplayChronicleVersions`, côté Go : le fichier n'en a pas une mais trois).
    const src = readFileSync(
      join(
        racineDuDepot(),
        'apps/go-api/internal/games/halo_infinite/film/replay/document_chronicle.go',
      ),
      'utf8',
    )
    const declarees = new Set<number>()
    for (const re of [
      /^\/\/ v(\d+) \(/gm,
      /^\/\/ SCHEMA (\d+) /gm,
      /^\/\/ CE QUE (?:LA VERSION|LE SCHEMA) (\d+) /gm,
    ]) {
      for (const m of src.matchAll(re)) declarees.add(Number(m[1]))
    }
    expect(declarees.size, 'la forme des en-têtes de chronique a-t-elle changé ?').toBeGreaterThan(40)
    expect(
      declarees.has(MIN_RENDERABLE_SCHEMA_VERSION),
      `la borne ${MIN_RENDERABLE_SCHEMA_VERSION} n'est déclarée par aucune entrée de chronique`,
    ).toBe(true)
  })
})

describe('le contrat non respecté prime sur toute question de version', () => {
  /**
   * LE MANQUEMENT EST UNE DONNÉE, pas une phrase (ronde 2 de la revue, constat R2-2) : ce
   * module ne le met jamais en mots — c'est le badge qui le fait, par `i18n.ts`, dans les deux
   * langues. Le type l'impose désormais, et c'est aussi ce qui a fait disparaître le cas
   * « chaîne vide » que ce fichier testait : une chaîne vide n'est plus représentable.
   */
  const MANQUEMENT: ReplayContractIssue = { kind: 'unknownKeys', keys: ['shotz'] }

  it('rend "invalid" et porte le manquement, même sur un artefact à jour', () => {
    expect(computeReplaySchemaStatus(PRODUCTEUR, PRODUCTEUR, MANQUEMENT)).toEqual({
      kind: 'invalid',
      schemaVersion: PRODUCTEUR,
      issue: MANQUEMENT,
    })
  })

  it('rend "invalid" aussi sous le seuil d\'affichage : la conformité passe avant', () => {
    expect(
      computeReplaySchemaStatus(MIN_RENDERABLE_SCHEMA_VERSION - 1, undefined, {
        kind: 'invalidField',
        path: 'bounds.maxX',
        detail: 'expected number',
      }).kind,
    ).toBe('invalid')
  })

  it('aucun manquement : le statut retombe sur la comparaison de versions', () => {
    expect(computeReplaySchemaStatus(PRODUCTEUR, PRODUCTEUR, undefined).kind).toBe('upToDate')
  })
})

describe('la matrice appliquée à CHAQUE fixture publiée par Go', () => {
  /**
   * POURQUOI ITÉRER LE JEU ENTIER (lot 0.B.7, 2026-09-13). Le manifeste ne porte plus un
   * document mais UN PAR BUILD du jeu. Une matrice de compatibilité qui ne statuerait que sur
   * la version du producteur dirait la même chose huit fois ; ce qu'on veut savoir est que
   * CHAQUE document publié est affichable par ce client, et que le badge sait quoi en dire.
   *
   * AUCUN DOCUMENT N'EST OUVERT ICI : les entrées du manifeste portent la version et le build,
   * et c'est exactement ce dont le badge a besoin. Le document complet, lui, traverse la
   * frontière dans `test/goFixtures.contract.test.ts` — deux preuves, deux coûts.
   */
  const FIXTURES = goFixtureEntries()

  it('le jeu publié n’est pas vide — sans quoi cette matrice ne statuerait sur rien', () => {
    expect(FIXTURES.length).toBeGreaterThan(0)
  })

  for (const f of FIXTURES) {
    describe(`${f.build} / ${f.film}`, () => {
      it('est affichable : sa version atteint le seuil de compatibilité', () => {
        expect(
          f.schemaVersion,
          `${f.file} est sous le seuil : le web ne prétend pas afficher cette génération`,
        ).toBeGreaterThanOrEqual(MIN_RENDERABLE_SCHEMA_VERSION)
      })

      it('se dit « à jour » face au producteur du jour', () => {
        expect(computeReplaySchemaStatus(f.schemaVersion, PRODUCTEUR)).toEqual({
          kind: 'upToDate',
          schemaVersion: f.schemaVersion,
        })
      })

      it('se dirait « à recuire » si le producteur prenait de l’avance', () => {
        const apres = f.schemaVersion + 1
        expect(computeReplaySchemaStatus(f.schemaVersion, apres)).toEqual({
          kind: 'stale',
          schemaVersion: f.schemaVersion,
          latestSchemaVersion: apres,
        })
      })

      it('et « inconnu » sans en-tête à comparer, jamais une exception', () => {
        expect(computeReplaySchemaStatus(f.schemaVersion, undefined)).toEqual({
          kind: 'unknown',
          schemaVersion: f.schemaVersion,
        })
      })
    })
  }
})

/**
 * LA MATRICE DE COMPATIBILITÉ GAGNE UN AXE (2026-09-17, lot 4.2.2, schéma 62) : la présence d'un
 * calque.
 *
 * POURQUOI ICI, À CÔTÉ DU STATUT DE VERSION. Les deux répondent à la même famille de question —
 * « que vaut CE document face à ce que le producteur sait faire aujourd'hui ? » — et ils ont la
 * même structure à trois issues, dont une est l'ABSENCE DE RÉPONSE. Un artefact antérieur à 62 ne
 * dit rien de ses calques, exactement comme une réponse sans en-tête ne dit rien de sa version :
 * dans les deux cas le badge doit se taire, pas trancher. Les regrouper est ce qui rend la
 * symétrie visible en revue.
 *
 * LE STATUT DE VERSION NE DÉPEND PAS DE `layers`, et ces cas le prouvent : les trois passent le
 * même document par `computeReplaySchemaStatus`, dont la sortie est IDENTIQUE dans les trois.
 */
describe('la présence d’un calque — le troisième axe de la matrice (schéma 62)', () => {
  const REVISION_GRAMMAIRE = 'grammar-2026-09-15.42'

  it('`layers` ABSENT : la question n’a pas de réponse, et le statut de version est intact', () => {
    const doc = testReplayDoc({ schemaVersion: AVANT_LAYERS })
    expect(calquePresent(doc, 'zoneStates')).toBe('inconnu')
    expect(couchesDesCalquesProduits(doc)).toEqual([])
    expect(computeReplaySchemaStatus(doc.schemaVersion, PRODUCTEUR).kind).toBe('stale')
  })

  it('entrée ABSENTE dans un `layers` présent : « pas produit » est une réponse', () => {
    const doc = testReplayDoc({ layers: { tracks: REVISION_GRAMMAIRE } })
    expect(calquePresent(doc, 'vipCrown')).toBe('nonProduit')
    expect(revisionDuCalque(doc, 'vipCrown')).toBeUndefined()
    expect(computeReplaySchemaStatus(doc.schemaVersion, PRODUCTEUR).kind).toBe('upToDate')
  })

  it('entrée PRÉSENTE : produit, sous la révision nommée, et la couche est lisible', () => {
    const doc = testReplayDoc({ layers: { tracks: REVISION_GRAMMAIRE } })
    expect(calquePresent(doc, 'tracks')).toBe('produit')
    expect(revisionDuCalque(doc, 'tracks')).toBe(REVISION_GRAMMAIRE)
    expect(couchesDesCalquesProduits(doc)).toEqual([REVISION_GRAMMAIRE])
    expect(computeReplaySchemaStatus(doc.schemaVersion, PRODUCTEUR).kind).toBe('upToDate')
  })

  it('les fixtures que Go publie portent TOUTES leur table de calques', () => {
    for (const f of goFixtureEntries()) {
      expect(f.schemaVersion, `${f.file} : version de la fixture`).toBe(PRODUCTEUR)
    }
  })
})
