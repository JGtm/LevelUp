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

import { racineDuDepot } from '../test/featureFiles'
import { goFixtureSchemaVersion } from '../test/goFixtures'
import {
  computeReplaySchemaStatus,
  MIN_RENDERABLE_SCHEMA_VERSION,
} from './replaySchemaStatusLogic'

/** La version que le producteur écrit aujourd'hui. */
const PRODUCTEUR = goFixtureSchemaVersion()

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
  it('rend "invalid" et porte le manquement, même sur un artefact à jour', () => {
    expect(computeReplaySchemaStatus(PRODUCTEUR, PRODUCTEUR, 'matchId : champ requis')).toEqual({
      kind: 'invalid',
      schemaVersion: PRODUCTEUR,
      issue: 'matchId : champ requis',
    })
  })

  it('rend "invalid" aussi sous le seuil d\'affichage : la conformité passe avant', () => {
    expect(
      computeReplaySchemaStatus(MIN_RENDERABLE_SCHEMA_VERSION - 1, undefined, 'bounds : absent').kind,
    ).toBe('invalid')
  })

  it('une chaîne VIDE n\'est pas un manquement : elle ne doit pas allumer le badge', () => {
    expect(computeReplaySchemaStatus(PRODUCTEUR, PRODUCTEUR, '').kind).toBe('upToDate')
  })
})
