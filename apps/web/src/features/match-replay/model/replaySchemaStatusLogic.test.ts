import { describe, expect, it } from 'vitest'

import { computeReplaySchemaStatus } from './replaySchemaStatusLogic'

describe('computeReplaySchemaStatus', () => {
  it('rend "unknown" quand la version courante du producteur est absente', () => {
    expect(computeReplaySchemaStatus(48, undefined)).toEqual({
      kind: 'unknown',
      schemaVersion: 48,
    })
  })

  it('rend "upToDate" quand l\'artefact porte déjà la version courante', () => {
    expect(computeReplaySchemaStatus(51, 51)).toEqual({
      kind: 'upToDate',
      schemaVersion: 51,
    })
  })

  it('rend "stale" quand l\'artefact est en retard sur le producteur', () => {
    expect(computeReplaySchemaStatus(48, 51)).toEqual({
      kind: 'stale',
      schemaVersion: 48,
      latestSchemaVersion: 51,
    })
  })

  it('rend "upToDate" quand l\'artefact est PLUS RÉCENT que le binaire courant (poste de dev)', () => {
    expect(computeReplaySchemaStatus(51, 48)).toEqual({
      kind: 'upToDate',
      schemaVersion: 51,
    })
  })
})
