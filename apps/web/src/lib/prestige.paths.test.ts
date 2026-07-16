/**
 * prestige.paths.test.ts — garde-rail de câblage des routes Escouade.
 *
 * Le module Prestige est monté côté serveur SOUS /players/{player_slug}
 * (ownershipMW, ADR 0029). Les routes squad (roster CRUD + défis) DOIVENT
 * porter ce préfixe player-scoped ; un appel top-level (`/squads`) retourne
 * 404 — c'était la cause du bug « Enregistrer cette compo comme escouade »
 * (backlog C2). Ce test fige les chemins pour empêcher toute régression vers
 * un chemin nu.
 */
import { describe, it, expect, beforeEach, vi } from 'vitest'

const { calls } = vi.hoisted(() => ({ calls: [] as { method: string; path: string }[] }))

vi.mock('./api/client', () => {
  const record = (method: string) => (path: string) => {
    calls.push({ method, path })
    return Promise.resolve(undefined)
  }
  return {
    api: {
      get: record('get'),
      post: record('post'),
      patch: record('patch'),
      put: record('put'),
      delete: record('delete'),
    },
  }
})

import { prestigeApi } from './prestige'

const SLUG = 'Alice'
const ENC = encodeURIComponent(SLUG)

beforeEach(() => {
  calls.length = 0
})

describe('prestigeApi — routes Escouade player-scoped (/players/{slug})', () => {
  it('createSquad → POST /players/{created_by}/squads', () => {
    prestigeApi.createSquad({ name: 'Trio', created_by: SLUG, members: [] })
    expect(calls[0]).toEqual({ method: 'post', path: `/players/${ENC}/squads` })
  })

  it('listMySquads → GET /players/{userId}/squads', () => {
    prestigeApi.listMySquads(SLUG)
    expect(calls[0]).toEqual({ method: 'get', path: `/players/${ENC}/squads?user_id=${ENC}` })
  })

  it('renameSquad → PATCH /players/{requested_by}/squads/{id}', () => {
    prestigeApi.renameSquad('sq1', { name: 'X', requested_by: SLUG })
    expect(calls[0]).toEqual({ method: 'patch', path: `/players/${ENC}/squads/sq1` })
  })

  it('deleteSquad → DELETE /players/{requestedBy}/squads/{id}', () => {
    prestigeApi.deleteSquad('sq1', SLUG)
    expect(calls[0]).toEqual({
      method: 'delete',
      path: `/players/${ENC}/squads/sq1?requested_by=${ENC}`,
    })
  })

  it('addSquadMember → POST /players/{requested_by}/squads/{id}/members', () => {
    prestigeApi.addSquadMember('sq1', { xuid: 'x1', requested_by: SLUG })
    expect(calls[0].path).toBe(`/players/${ENC}/squads/sq1/members`)
  })

  it('removeSquadMember → DELETE /players/{requestedBy}/squads/{id}/members/{xuid}', () => {
    prestigeApi.removeSquadMember('sq1', 'x1', SLUG)
    expect(calls[0].path).toBe(`/players/${ENC}/squads/sq1/members/x1?requested_by=${ENC}`)
  })

  it('squadOrientation → GET /players/{requestedBy}/squads/{id}/orientation', () => {
    prestigeApi.squadOrientation('sq1', SLUG)
    expect(calls[0].path).toBe(`/players/${ENC}/squads/sq1/orientation?requested_by=${ENC}`)
  })

  it('listSquadChallenges → GET /players/{requestedBy}/squads/{id}/challenges', () => {
    prestigeApi.listSquadChallenges('sq1', SLUG)
    expect(calls[0].path).toBe(`/players/${ENC}/squads/sq1/challenges`)
  })

  it('createSquadChallenge → POST /players/{created_by}/squads/{id}/challenges', () => {
    prestigeApi.createSquadChallenge('sq1', {
      title_slug: 'halo_infinite',
      mode: 'collective',
      eval_type: 'threshold',
      window_type: 'session',
      created_by: SLUG,
    })
    expect(calls[0].path).toBe(`/players/${ENC}/squads/sq1/challenges`)
  })

  it('refreshSquadPool → POST /players/{requested_by}/squads/{id}/challenges/pool/refresh', () => {
    prestigeApi.refreshSquadPool('sq1', { title_slug: 'halo_infinite', requested_by: SLUG })
    expect(calls[0].path).toBe(`/players/${ENC}/squads/sq1/challenges/pool/refresh`)
  })

  it('joinSquadChallenge → POST /players/{user_id}/squad-challenges/{id}/join', () => {
    prestigeApi.joinSquadChallenge('sc1', { user_id: SLUG })
    expect(calls[0].path).toBe(`/players/${ENC}/squad-challenges/sc1/join`)
  })

  it('evaluateSquadChallenge → POST /players/{requestedBy}/squad-challenges/{id}/evaluate', () => {
    prestigeApi.evaluateSquadChallenge('sc1', SLUG)
    expect(calls[0].path).toBe(`/players/${ENC}/squad-challenges/sc1/evaluate`)
  })

  it('aucune route Escouade ne cible un chemin top-level nu', () => {
    prestigeApi.createSquad({ name: 'T', created_by: SLUG })
    prestigeApi.listMySquads(SLUG)
    prestigeApi.squadOrientation('sq1', SLUG)
    expect(calls.length).toBeGreaterThan(0)
    for (const c of calls) {
      expect(c.path.startsWith('/players/')).toBe(true)
      expect(c.path.startsWith('/squads')).toBe(false)
      expect(c.path.startsWith('/squad-challenges')).toBe(false)
    }
  })
})
