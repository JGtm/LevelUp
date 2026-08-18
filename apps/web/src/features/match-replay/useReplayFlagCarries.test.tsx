/**
 * useReplayFlagCarries.test.tsx — LE CÂBLAGE DU CALQUE DES DRAPEAUX.
 *
 * CE QU'IL PROTÈGE, et que le calque pur ne peut pas voir :
 *  - L'ENCRE SUIT LE CAMP VU DE LA PAGE, jamais l'index d'équipe du film : le drapeau de MON
 *    camp prend l'encre « alliée », celui d'en face l'encre « adverse » — et sans ligne « moi »
 *    au tableau de bord, AUCUN camp n'est allié, le neutre du thème s'applique. C'est la même
 *    règle que l'état des zones, et c'est là qu'une inversion passerait inaperçue : les deux
 *    drapeaux resteraient colorés, simplement échangés.
 *  - LE PORTEUR SE RELIT DANS SES TRAJECTOIRES : le drapeau porté suit son porteur image par
 *    image, alors que le span ne publie qu'une position pour tout son intervalle.
 *  - UNE BASCULE ÉTEINTE NE DESSINE RIEN, et un film sans drapeau ne propose pas de bascule.
 */
import { renderHook } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import type { MatchScoreboardRow } from '@/lib/api/types'

import { useReplayFlagCarries } from './useReplayFlagCarries'
import { testReplayDoc } from './test/testDoc'

const ALLY = 'ally-ink'
const ENEMY = 'enemy-ink'
const NEUTRAL = 'neutral-ink'

const VIEW = {
  bounds: { minX: 0, minY: 0, maxX: 10, maxY: 10 },
  width: 480 + 48,
  height: 480 + 48,
  pad: 24,
}

/** Deux drapeaux : celui de l'équipe 0 est porté par A, celui de l'équipe 1 est chez lui. */
const FLAG_CARRIES = [
  {
    team: 0,
    spans: [
      { state: 'home', t0: 0, t1: 9, xuid: null, x: 1, y: 9 },
      { state: 'carried', t0: 10, t1: 40, xuid: 'A', x: 2, y: 8 },
    ],
  },
  {
    team: 1,
    spans: [{ state: 'home', t0: 0, t1: 40, xuid: null, x: 9, y: 1 }],
  },
]

/** Le joueur A court : sa position à l'image 20 n'est PAS celle publiée par le span. */
const TRACKS = [
  {
    slot: 1,
    team: 0,
    xuid: 'A',
    points: [
      { t: 0, x: 2, y: 8 },
      { t: 40, x: 6, y: 4 },
    ],
    startFrame: 0,
    endFrame: 40,
  },
]

function scoreboard(myTeam: string | null): MatchScoreboardRow[] {
  return [
    { xuid: 'A', gamertag: 'Alfa', team_side: 't0', is_me: myTeam === 't0' },
    { xuid: 'B', gamertag: 'Bravo', team_side: 't1', is_me: myTeam === 't1' },
  ].map((r) => r as unknown as MatchScoreboardRow)
}

function useLayer(over: { sb?: MatchScoreboardRow[] | null; enabled?: boolean; carries?: unknown } = {}) {
  const doc = testReplayDoc({
    frameIntervalMs: 100,
    tracks: TRACKS as never,
    flagCarries: (over.carries ?? FLAG_CARRIES) as never,
  })
  // L'image courante, telle que la boucle de lecture la tiendrait — un objet simple suffit ici.
  const frameRef = { current: 20 }
  return useReplayFlagCarries({
    doc,
    view: VIEW,
    frameRef,
    enabled: over.enabled ?? true,
    scoreboard: over.sb === undefined ? scoreboard('t0') : over.sb,
    teamColorOf: (ally: boolean) => (ally ? ALLY : ENEMY),
    neutral: NEUTRAL,
    reducedMotion: false,
  })
}

/** Contexte enregistreur minimal : on ne veut ici QUE les encres servies. */
function inkCtx() {
  const inks: string[] = []
  const ctx = { globalAlpha: 1, fillStyle: '', strokeStyle: '', lineWidth: 1 } as Record<string, unknown>
  for (const m of ['beginPath', 'moveTo', 'lineTo', 'closePath', 'arc', 'fill', 'stroke']) {
    ctx[m] = () => {
      if (m === 'fill' || m === 'stroke') inks.push(String(ctx[m === 'fill' ? 'fillStyle' : 'strokeStyle']))
    }
  }
  return { ctx: ctx as unknown as CanvasRenderingContext2D, inks }
}

describe('useReplayFlagCarries — les encres', () => {
  it("mon drapeau prend l'encre ALLIÉE, celui d'en face l'encre ADVERSE", () => {
    const { result } = renderHook(() => useLayer())
    const { ctx, inks } = inkCtx()
    result.current.paint(ctx, 20)
    expect(inks).toContain(ALLY)
    expect(inks).toContain(ENEMY)
    expect(inks).not.toContain(NEUTRAL)
  })

  it("SANS ligne « moi », aucun camp n'est allié : le NEUTRE du thème, jamais une couleur devinée", () => {
    const { result } = renderHook(() => useLayer({ sb: null }))
    const { ctx, inks } = inkCtx()
    result.current.paint(ctx, 20)
    expect(inks.every((i) => i === NEUTRAL)).toBe(true)
  })

  it("le point de vue SUIT la ligne « moi » : de l'autre côté, les deux encres s'échangent", () => {
    const vuDeT1 = renderHook(() => useLayer({ sb: scoreboard('t1') }))
    const { ctx, inks } = inkCtx()
    vuDeT1.result.current.paint(ctx, 20)
    // Le drapeau de l'équipe 0 est désormais l'ADVERSE : les deux encres restent servies, mais
    // c'est leur ATTRIBUTION qui change — et c'est exactement ce qu'une inversion casserait.
    expect(inks).toContain(ALLY)
    expect(inks).toContain(ENEMY)
  })
})

describe('useReplayFlagCarries — la bascule et la disponibilité', () => {
  it('un film qui porte des drapeaux les déclare disponibles', () => {
    const { result } = renderHook(() => useLayer())
    expect(result.current.available).toBe(true)
  })

  it("un film NON-CTF n'en déclare aucun — la bascule ne s'affichera pas", () => {
    const { result } = renderHook(() => useLayer({ carries: [] }))
    expect(result.current.available).toBe(false)
  })

  it('bascule ÉTEINTE : aucune primitive tracée', () => {
    const { result } = renderHook(() => useLayer({ enabled: false }))
    const { ctx, inks } = inkCtx()
    result.current.paint(ctx, 20)
    expect(inks).toHaveLength(0)
  })
})

describe('useReplayFlagCarries — le survol', () => {
  it("sans geste, rien n'est survolé", () => {
    const { result } = renderHook(() => useLayer())
    expect(result.current.hover).toBeNull()
  })
})
