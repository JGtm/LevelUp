/**
 * Tests — le statut du badge admin « version de schéma ».
 *
 * AUCUN NUMÉRO DE SCHÉMA N'EST ÉCRIT EN DUR ICI (2026-09-13, lot 0.B) : les versions se
 * dérivent de celle que Go publie (`test/goFixtures.ts`) et du seuil de compatibilité déclaré
 * par le web. Le ratchet `test/testDoc.guard.test.ts` l'interdit désormais dans toute la
 * feature — un littéral écrit à la main est ce qui a fait vivre pendant des mois une fixture
 * partagée figée à la version 1, que le serveur n'a jamais servie.
 */
import { describe, expect, it } from 'vitest'

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
