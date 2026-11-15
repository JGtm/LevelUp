import { describe, expect, it } from 'vitest'
import { resolveFirstSyncState, type FirstSyncInput } from './firstSyncState'

const base: FirstSyncInput = {
  reauthRequired: false,
  canWrite: true,
  syncEnabled: true,
  server: { syncing: false, jobStatus: undefined },
}

describe('resolveFirstSyncState', () => {
  it('état du serveur pas encore arrivé : « loading », rien à dire', () => {
    expect(resolveFirstSyncState({ ...base, server: null }).kind).toBe('loading')
  })

  it('rien en vol, aucun job : « idle », jamais « en cours »', () => {
    expect(resolveFirstSyncState(base).kind).toBe('idle')
  })

  it('jeton mort sur son propre profil : en attente de connexion, même avec un job connu', () => {
    expect(
      resolveFirstSyncState({ ...base, reauthRequired: true, server: { syncing: true, jobStatus: 'running' } }).kind,
    ).toBe('waiting_connection')
  })

  it("jeton mort du visiteur sur le profil d'un autre : aucun « en attente de connexion »", () => {
    expect(resolveFirstSyncState({ ...base, reauthRequired: true, canWrite: false }).kind).toBe('idle')
  })

  it('titre en pause : « paused » avant le job', () => {
    expect(
      resolveFirstSyncState({ ...base, syncEnabled: false, server: { syncing: false, jobStatus: 'failed' } }).kind,
    ).toBe('paused')
  })

  it('profil inconnu : aucune pause déclarée', () => {
    expect(resolveFirstSyncState({ ...base, syncEnabled: undefined }).kind).toBe('idle')
  })

  it("une synchronisation en vol d'une autre source (sans job) : « running »", () => {
    expect(resolveFirstSyncState({ ...base, server: { syncing: true, jobStatus: undefined } }).kind).toBe('running')
  })

  it.each([
    ['en file', 'queued', 'running'],
    ['en cours', 'running', 'running'],
    ['terminé', 'succeeded', 'empty'],
    ['échoué', 'failed', 'failed'],
    ['annulé', 'cancelled', 'failed'],
    ['interrompu', 'interrupted', 'failed'],
  ] as const)('job %s', (_nom, jobStatus, kind) => {
    expect(resolveFirstSyncState({ ...base, server: { syncing: false, jobStatus } }).kind).toBe(kind)
  })

  it('un job terminé alors qu’une autre source synchronise : « running »', () => {
    expect(resolveFirstSyncState({ ...base, server: { syncing: true, jobStatus: 'failed' } }).kind).toBe('running')
  })
})
