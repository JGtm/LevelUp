/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (CLAUDE.md n° 6) : LA COPIE LOCALE DE `replay.SchemaVersion` NE DOIT JAMAIS
 * DIVERGER DE LA CONSTANTE GO.
 *
 * `EXPECTED_REPLAY_SCHEMA_VERSION` (replaySchemaLogic.ts) est une copie ÉCRITE À LA MAIN de
 * `replay.SchemaVersion` (apps/go-api/internal/analysis/replay/document.go) — le contrat
 * généré ne porte aucune valeur littérale pour `schemaVersion` (son type est `number`, la
 * valeur variant précisément d'un artefact à l'autre). Sans ce garde-rail, un bump de schéma
 * côté Go qui oublierait la copie ferait lire le client une version PÉRIMÉE comme
 * `current` — la garde du lot 2 se tairait exactement là où elle devrait parler.
 *
 * Même patron que `placementFamily.guard.test.ts` : lire le fichier Go source par
 * `readFileSync` et comparer, plutôt que dupliquer la constante sans lien vérifiable.
 */
import { describe, expect, it } from 'vitest'

import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

import { EXPECTED_REPLAY_SCHEMA_VERSION } from './replaySchemaLogic'

const REPO = resolve(__dirname, '..', '..', '..', '..', '..')
const GO_DOCUMENT = resolve(
  REPO,
  'apps/go-api/internal/analysis/replay/document.go',
)

/** La constante Go `SchemaVersion` — la seule source de vérité. */
function goSchemaVersion(): number {
  const src = readFileSync(GO_DOCUMENT, 'utf8')
  const m = src.match(/^const SchemaVersion = (\d+)$/m)
  expect(m, 'const SchemaVersion introuvable dans document.go').not.toBeNull()
  return Number(m?.[1])
}

describe('garde-rail : parité EXPECTED_REPLAY_SCHEMA_VERSION <-> replay.SchemaVersion', () => {
  it('la copie locale vaut EXACTEMENT la constante Go', () => {
    expect(
      EXPECTED_REPLAY_SCHEMA_VERSION,
      'EXPECTED_REPLAY_SCHEMA_VERSION (replaySchemaLogic.ts) a divergé de replay.SchemaVersion ' +
        '(document.go) — mettre à jour la copie locale ET vérifier que la garde du lot 2 reste ' +
        'juste pour les artefacts déjà écrits (cf. LOT2_TELEMETRIE_GARDE_SCHEMA_2026-08-25.md).',
    ).toBe(goSchemaVersion())
  })
})
