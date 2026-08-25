import { describe, expect, it } from 'vitest'

import {
  EXPECTED_REPLAY_SCHEMA_VERSION,
  replaySchemaState,
} from './replaySchemaLogic'

describe('replaySchemaState', () => {
  it('rend "current" quand les deux versions concordent', () => {
    expect(replaySchemaState(EXPECTED_REPLAY_SCHEMA_VERSION)).toBe('current')
  })

  it('rend "stale" pour un artefact ANTÉRIEUR — cas de l’audit (backfill non rejoué)', () => {
    expect(replaySchemaState(EXPECTED_REPLAY_SCHEMA_VERSION - 1)).toBe('stale')
    expect(replaySchemaState(1)).toBe('stale')
    expect(replaySchemaState(0)).toBe('stale')
  })

  it('rend "ahead" pour un artefact POSTÉRIEUR — le client, pas l’artefact, est en retard', () => {
    expect(replaySchemaState(EXPECTED_REPLAY_SCHEMA_VERSION + 1)).toBe('ahead')
    expect(replaySchemaState(999)).toBe('ahead')
  })
})
