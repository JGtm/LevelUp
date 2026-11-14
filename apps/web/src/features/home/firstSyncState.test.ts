import { describe, expect, it } from 'vitest'
import { resolveFirstSyncState, type FirstSyncInput } from './firstSyncState'

const base: FirstSyncInput = {
  reauthRequired: false,
  authState: 'ready',
  syncEnabled: true,
  activeJobId: null,
  jobStatus: undefined,
}

describe('resolveFirstSyncState', () => {
  it('rien de lancé : « idle », jamais « en cours »', () => {
    expect(resolveFirstSyncState(base).kind).toBe('idle')
  })

  it.each([
    ['reconnexion exigée', { reauthRequired: true }],
    ['compte non connecté', { authState: 'missing' }],
    ['identité partielle', { authState: 'partial' }],
  ] as const)('%s : en attente de connexion, même avec un job connu', (_nom, patch) => {
    expect(
      resolveFirstSyncState({
        ...base,
        ...patch,
        activeJobId: 'j1',
        jobStatus: 'running',
      }).kind,
    ).toBe('waiting_connection')
  })

  it('titre en pause : « paused » avant le job', () => {
    expect(
      resolveFirstSyncState({
        ...base,
        syncEnabled: false,
        activeJobId: 'j1',
        jobStatus: 'failed',
      }).kind,
    ).toBe('paused')
  })

  it('profil inconnu : aucune pause déclarée', () => {
    expect(resolveFirstSyncState({ ...base, syncEnabled: undefined }).kind).toBe('idle')
  })

  it.each([
    ['job pas encore lu', undefined, 'running'],
    ['en file', 'queued', 'running'],
    ['en cours', 'running', 'running'],
    ['terminé', 'succeeded', 'empty'],
    ['échoué', 'failed', 'failed'],
    ['annulé', 'cancelled', 'failed'],
    ['interrompu', 'interrupted', 'failed'],
  ] as const)('job %s', (_nom, jobStatus, kind) => {
    expect(resolveFirstSyncState({ ...base, activeJobId: 'j1', jobStatus }).kind).toBe(kind)
  })
})
