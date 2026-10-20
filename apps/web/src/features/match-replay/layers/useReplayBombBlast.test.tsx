/**
 * Tests — useReplayBombBlast : L'ENCRE D'UNE DÉFLAGRATION EST L'ALLÉGEANCE DU FILM DE SON AUTEUR
 * (2026-10-06), vue du joueur regardé ; sans allégeance, le neutre du thème — jamais une équipe
 * devinée. Le dessin lui-même est couvert par `bombBlastFx.test.ts`.
 */
import { renderHook } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { buildFilmAllegiance, type FilmAllegiance } from '@/lib/replay/filmAllegiance'
import type { ReplayPlayer } from '@/lib/replay/rosterLogic'

import { testReplayDoc } from '../test/testDoc'
import { useReplayBombBlast } from './useReplayBombBlast'

const VIEW = { bounds: { minX: 0, minY: 0, maxX: 10, maxY: 10 }, width: 528, height: 528, pad: 24 }
const ALLIE = 'encre-alliee'
const ADVERSE = 'encre-adverse'
const NEUTRE = 'encre-neutre'

/** Le joueur A, immobile en (1,9), fait détonner la bombe à l'image 10. */
const DOC = testReplayDoc({
  frameIntervalMs: 100,
  originMs: 0,
  tracks: [
    { slot: 1, team: 0, xuid: 'A', points: [{ t: 0, x: 1, y: 9 }, { t: 100, x: 1, y: 9 }], startFrame: 0, endFrame: 100 },
  ] as never,
  objectives: [{ t: 10, xuid: 'A', stat: 'bomb_detonations', timeMs: 1_000 }] as never,
})

/** A au camp 0, B au camp 1, Muet sans équipe du film ; vu de `reference`. */
function vueDe(reference: string): FilmAllegiance {
  const joueurs = [
    { xuid: 'A', team: 0, lives: [] },
    { xuid: 'B', team: 1, lives: [] },
    { xuid: 'Muet', lives: [] },
  ] as unknown as ReplayPlayer[]
  return buildFilmAllegiance(joueurs, reference)
}

/** Les encres posées en peignant l'image de la détonation. */
function encres(allegiance: FilmAllegiance): Set<string> {
  const { result } = renderHook(() =>
    useReplayBombBlast({
      doc: DOC,
      view: VIEW,
      allegiance,
      teamColorOf: (ally) => (ally ? ALLIE : ADVERSE),
      neutral: NEUTRE,
      reducedMotion: true,
    }),
  )
  const vues = new Set<string>()
  const ctx = { globalAlpha: 1, strokeStyle: '', fillStyle: '', lineWidth: 1 } as Record<string, unknown>
  for (const m of ['beginPath', 'arc', 'moveTo', 'lineTo']) ctx[m] = () => {}
  ctx.stroke = () => vues.add(String(ctx.strokeStyle))
  ctx.fill = () => vues.add(String(ctx.fillStyle))
  result.current.paint(ctx as unknown as CanvasRenderingContext2D, 10)
  return vues
}

describe('useReplayBombBlast — l’encre de l’auteur, lue dans le film', () => {
  it('vue de son camp : l’encre ALLIÉE ; vue de l’autre : l’encre ADVERSE', () => {
    expect(encres(vueDe('A'))).toEqual(new Set([ALLIE]))
    expect(encres(vueDe('B'))).toEqual(new Set([ADVERSE]))
  })

  it('vue d’un joueur dont le film tait l’équipe : le NEUTRE — aucun camp deviné', () => {
    expect(encres(vueDe('Muet'))).toEqual(new Set([NEUTRE]))
  })
})
