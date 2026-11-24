/**
 * Tests — ficheViewpoint : ce que la fiche d'un joueur dit du point de vue. Il se branche sur l'état existant
 * (`useReplayViewpoint`) : la valeur du joueur est celle du menu de la frise (`viewpointValueOf`), jamais une clé à part.
 */
import { describe, expect, it, vi } from 'vitest'

import type { ReplayPlayer } from '@/lib/replay/rosterLogic'

import { ficheViewpointOf } from './ficheViewpoint'

const joueur = (over: Partial<ReplayPlayer>) => ({ xuid: 'film-1', board: { xuid: 'base-1' }, ...over }) as unknown as ReplayPlayer

describe('ficheViewpointOf', () => {
  it('prendre le point de vue pose le xuid de la BASE du joueur (celui du menu de la frise)', () => {
    const select = vi.fn()
    const v = ficheViewpointOf(joueur({}), { xuid: 'autre', select })
    v.onTake()
    expect(select).toHaveBeenCalledWith('base-1')
    expect(v).toMatchObject({ current: false, inert: false })
  })

  it('un bot (sans xuid de base) se prend par sa clé du film, comme dans le menu', () => {
    const select = vi.fn()
    ficheViewpointOf(joueur({ xuid: 'bot:Cortana', board: undefined }), { xuid: null, select }).onTake()
    expect(select).toHaveBeenCalledWith('bot:Cortana')
  })

  it('le joueur déjà regardé est « courant »', () => {
    expect(ficheViewpointOf(joueur({}), { xuid: 'base-1', select: vi.fn() }).current).toBe(true)
  })

  it('sans ligne de tableau de score : inerte', () => {
    expect(ficheViewpointOf(joueur({ board: undefined }), { xuid: null, select: vi.fn() }).inert).toBe(true)
  })
})
