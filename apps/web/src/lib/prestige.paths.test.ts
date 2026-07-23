/**
 * prestige.paths.test.ts — garde-rail de câblage de TOUTES les routes Prestige.
 *
 * Le module Prestige (défis, arcs, templates, prestige/PP, escouade) est monté
 * côté serveur SOUS /players/{player_slug} (ownershipMW, ADR 0029). CHAQUE route
 * — pas seulement les routes squad — DOIT porter ce préfixe player-scoped ; un
 * appel top-level (`/challenges`, `/arcs`, `/squads`, …) retourne 404. C'était la
 * cause du bug « Enregistrer cette compo comme escouade » (backlog C2) côté squad
 * ET du 404 silencieux des écritures/lectures unitaires défis/arcs/templates.
 *
 * Ce test fige les chemins pour empêcher toute régression vers un chemin nu. La
 * complétude est garantie AU COMPILE : `invokeAll` est typé
 * `Record<keyof typeof prestigeApi, …>` → ajouter une fonction à prestigeApi sans
 * l'enregistrer ici casse la compilation (anti-oubli de câblage).
 */
import { describe, it, expect, beforeEach, vi } from 'vitest'
import type { CreateArcBody, CreateChallengeBody } from './prestige'

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

const CHALLENGE_BODY: CreateChallengeBody = {
  user_id: SLUG,
  title_slug: 'halo_infinite',
  metric: 'kills',
  target: 10,
  window_type: 'session',
  cadence: 'free',
  eval_type: 'threshold',
  mode: 'libre',
}

const ARC_BODY: CreateArcBody = {
  user_id: SLUG,
  title_slug: 'halo_infinite',
  title: 'Mon arc',
}

beforeEach(() => {
  calls.length = 0
})

describe('prestigeApi — routes Défis/Arcs/Prestige/Templates player-scoped', () => {
  it('createChallenge → POST /players/{user_id}/prestige/challenges', () => {
    prestigeApi.createChallenge(CHALLENGE_BODY)
    expect(calls[0]).toEqual({ method: 'post', path: `/players/${ENC}/prestige/challenges` })
  })

  it('getChallenge → GET /players/{actor}/prestige/challenges/{id}', () => {
    prestigeApi.getChallenge('c1', SLUG)
    expect(calls[0]).toEqual({ method: 'get', path: `/players/${ENC}/prestige/challenges/c1` })
  })

  it('listActiveChallenges → GET /players/{userId}/prestige/challenges', () => {
    prestigeApi.listActiveChallenges(SLUG, 'halo_infinite')
    expect(calls[0].path).toBe(`/players/${ENC}/prestige/challenges?user_id=${ENC}&title_slug=halo_infinite`)
  })

  it('updateChallenge → PATCH /players/{actor}/prestige/challenges/{id}', () => {
    prestigeApi.updateChallenge('c1', { target: 2 }, SLUG)
    expect(calls[0]).toEqual({ method: 'patch', path: `/players/${ENC}/prestige/challenges/c1` })
  })

  it('abandonChallenge → DELETE /players/{actor}/prestige/challenges/{id}', () => {
    prestigeApi.abandonChallenge('c1', SLUG)
    expect(calls[0]).toEqual({ method: 'delete', path: `/players/${ENC}/prestige/challenges/c1` })
  })

  it('suggestNext → POST /players/{actor}/prestige/challenges/{id}/suggest-next', () => {
    prestigeApi.suggestNext('c1', SLUG)
    expect(calls[0]).toEqual({ method: 'post', path: `/players/${ENC}/prestige/challenges/c1/suggest-next` })
  })

  it('createArc → POST /players/{user_id}/arcs', () => {
    prestigeApi.createArc(ARC_BODY)
    expect(calls[0]).toEqual({ method: 'post', path: `/players/${ENC}/arcs` })
  })

  it('listArcs → GET /players/{userId}/arcs', () => {
    prestigeApi.listArcs(SLUG, 'halo_infinite')
    expect(calls[0].path).toBe(`/players/${ENC}/arcs?user_id=${ENC}&title_slug=halo_infinite`)
  })

  it('getArc → GET /players/{actor}/arcs/{id}', () => {
    prestigeApi.getArc('a1', SLUG)
    expect(calls[0]).toEqual({ method: 'get', path: `/players/${ENC}/arcs/a1` })
  })

  it('deleteArc → DELETE /players/{userId}/arcs/{id}', () => {
    prestigeApi.deleteArc('a1', SLUG, true)
    expect(calls[0].path).toBe(`/players/${ENC}/arcs/a1?user_id=${ENC}&objectives=delete`)
  })

  it('listArcPresets → GET /players/{userId}/arcs/presets', () => {
    prestigeApi.listArcPresets(SLUG, 'halo_infinite')
    expect(calls[0].path).toBe(`/players/${ENC}/arcs/presets?user_id=${ENC}&title_slug=halo_infinite`)
  })

  it('adoptArcPreset → POST /players/{userId}/arcs/presets/{id}/adopt', () => {
    prestigeApi.adoptArcPreset('p1', SLUG, 'halo_infinite')
    expect(calls[0].path).toBe(`/players/${ENC}/arcs/presets/p1/adopt`)
  })

  it('getMyPrestige → GET /players/{userId}/prestige/me', () => {
    prestigeApi.getMyPrestige(SLUG, 'halo_infinite')
    expect(calls[0].path).toBe(`/players/${ENC}/prestige/me?user_id=${ENC}&title_slug=halo_infinite`)
  })

  it('suggestTemplates → GET /players/{userId}/templates/suggest', () => {
    prestigeApi.suggestTemplates(SLUG, 'halo_infinite')
    expect(calls[0].path).toBe(
      `/players/${ENC}/templates/suggest?user_id=${ENC}&title_slug=halo_infinite&count=3`,
    )
  })
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
})

/**
 * Invocation minimale de CHAQUE fonction exportée de prestigeApi. Typé
 * `Record<keyof typeof prestigeApi, …>` : toute nouvelle fonction non enregistrée
 * ici échoue à la compilation → la complétude du garde-rail ci-dessous ne peut
 * pas régresser silencieusement.
 */
const invokeAll: Record<keyof typeof prestigeApi, () => unknown> = {
  createChallenge: () => prestigeApi.createChallenge(CHALLENGE_BODY),
  getChallenge: () => prestigeApi.getChallenge('c1', SLUG),
  listActiveChallenges: () => prestigeApi.listActiveChallenges(SLUG, 'halo_infinite'),
  listChallenges: () => prestigeApi.listChallenges(SLUG, 'halo_infinite', ['completed']),
  updateChallenge: () => prestigeApi.updateChallenge('c1', { target: 2 }, SLUG),
  abandonChallenge: () => prestigeApi.abandonChallenge('c1', SLUG),
  suggestNext: () => prestigeApi.suggestNext('c1', SLUG),
  enablePilotMode: () => prestigeApi.enablePilotMode(SLUG, 'halo_infinite'),
  disablePilotMode: () => prestigeApi.disablePilotMode(SLUG, 'halo_infinite'),
  createArc: () => prestigeApi.createArc(ARC_BODY),
  listArcs: () => prestigeApi.listArcs(SLUG, 'halo_infinite'),
  getArc: () => prestigeApi.getArc('a1', SLUG),
  deleteArc: () => prestigeApi.deleteArc('a1', SLUG, true),
  listArcPresets: () => prestigeApi.listArcPresets(SLUG, 'halo_infinite'),
  adoptArcPreset: () => prestigeApi.adoptArcPreset('p1', SLUG, 'halo_infinite'),
  getMyPrestige: () => prestigeApi.getMyPrestige(SLUG, 'halo_infinite'),
  suggestTemplates: () => prestigeApi.suggestTemplates(SLUG, 'halo_infinite'),
  createSquadChallenge: () =>
    prestigeApi.createSquadChallenge('sq1', {
      title_slug: 'halo_infinite',
      mode: 'collective',
      eval_type: 'threshold',
      window_type: 'session',
      created_by: SLUG,
    }),
  listSquadChallenges: () => prestigeApi.listSquadChallenges('sq1', SLUG),
  joinSquadChallenge: () => prestigeApi.joinSquadChallenge('sc1', { user_id: SLUG }),
  createSquad: () => prestigeApi.createSquad({ name: 'T', created_by: SLUG }),
  listMySquads: () => prestigeApi.listMySquads(SLUG),
  addSquadMember: () => prestigeApi.addSquadMember('sq1', { xuid: 'x1', requested_by: SLUG }),
  removeSquadMember: () => prestigeApi.removeSquadMember('sq1', 'x1', SLUG),
  renameSquad: () => prestigeApi.renameSquad('sq1', { name: 'X', requested_by: SLUG }),
  deleteSquad: () => prestigeApi.deleteSquad('sq1', SLUG),
  evaluateSquadChallenge: () => prestigeApi.evaluateSquadChallenge('sc1', SLUG),
  refreshSquadPool: () =>
    prestigeApi.refreshSquadPool('sq1', { title_slug: 'halo_infinite', requested_by: SLUG }),
  squadOrientation: () => prestigeApi.squadOrientation('sq1', SLUG),
}

describe('prestigeApi — aucune route ne cible un chemin top-level nu', () => {
  // Préfixes nus interdits : toute route prestige DOIT passer par /players/{slug}.
  const bareTopLevelPrefixes = [
    '/challenges',
    '/arcs',
    '/templates',
    '/prestige',
    '/squads',
    '/squad-challenges',
  ]

  it.each(Object.keys(invokeAll))('%s reste sous /players/{slug}', (name) => {
    calls.length = 0
    invokeAll[name as keyof typeof prestigeApi]()
    expect(calls.length).toBeGreaterThan(0)
    for (const c of calls) {
      expect(c.path.startsWith('/players/')).toBe(true)
      for (const bare of bareTopLevelPrefixes) {
        expect(c.path.startsWith(bare)).toBe(false)
      }
    }
  })
})
