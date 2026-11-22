/**
 * useReplayReliefFollow — « SUIVRE LE JOUEUR OUVERT » : à chaque image lue, la coupe prend l'étage de ses pieds.
 *
 * Le hook rend UN pas, `step(frame)`, que la toile appelle à chaque dessin (cadence de l'écran en lecture,
 * un dessin à chaque saut en pause) : l'image courante n'est pas un état React, elle vit dans la cellule de
 * dessin. Il lit le joueur (`buildFollowSampler`), avance `stepFollow` (sol porteur, seuil, délai : voir
 * `lib/replay/relief/follow.ts`) et ne PREND LA PAROLE que quand la coupe change (`onCut`) — rare : une
 * montée, une chute, une réapparition. Le reste du temps, rien ne se re-rend.
 *
 * L'ÉTAT DU SUIVI REPART DE L'ÉTAT INITIAL (`FOLLOW_INITIAL`) quand quoi que ce soit de ce qu'il lit change :
 * un autre joueur ouvert, un suivi rallumé, une autre carte (maillage). Le premier pas d'une vie prend alors
 * l'étage sans délai. Un pas est fait aussitôt, à l'image courante : allumer le suivi en pause pose la coupe
 * sans attendre un dessin.
 *
 * LE DÉLAI SE COMPTE EN TEMPS DE REJEU, pas en temps réel (`reliefFollowSample.ts`) : il suit la vitesse de
 * lecture et se fige en pause. Un recul ou un saut de plus d'une seconde est un saut de lecture : prise
 * d'étage immédiate.
 */
import { useCallback, useEffect, useMemo, useRef } from 'react'

import { FOLLOW_INITIAL, stepFollow, type FollowContext, type FollowState, type ReliefData } from '@/lib/replay/relief'
import type { ReplayDocumentReady } from '@/lib/replay/replayNormalize'
import type { ReplayPlayer } from '@/lib/replay/rosterLogic'

import { buildFollowSampler, type FollowSampler } from '../model/reliefFollowSample'

/** Ce que le suivi lit : le document, le joueur ouvert, la cellule de dessin et les sols de la carte. */
export interface ReliefFollowInput {
  doc: ReplayDocumentReady
  /** Le joueur dont la fiche est ouverte ; `null` = personne. */
  player: ReplayPlayer | null
  /** La cellule de dessin du canvas (l'image montrée) : lue, jamais écrite. */
  frameRef: { readonly current: number }
}

export interface ReliefFollowOptions extends ReliefFollowInput {
  /** Le suivi tourne : allumé, joueur ouvert, carte avec maillage. */
  on: boolean
  /** Les données du relief (sols et étages) ; `null` tant qu'elles ne sont pas lues. */
  data: ReliefData | null
  /** La coupe change : l'étage à poser. */
  onCut: (z: number) => void
}

/** Ce que le pas lit, tenu dans une cellule pour que `step` garde son identité. */
interface Live {
  state: FollowState
  sampler: FollowSampler | null
  ctx: FollowContext | null
  onCut: (z: number) => void
}

export function useReplayReliefFollow({ doc, player, frameRef, on, data, onCut }: ReliefFollowOptions): (frame: number) => void {
  const sampler = useMemo(() => (on && player ? buildFollowSampler(doc, player) : null), [on, doc, player])
  const ctx = useMemo<FollowContext | null>(() => (data ? { mesh: data.floors.mesh, levels: data.floors.levels } : null), [data])
  const live = useRef<Live>({ state: FOLLOW_INITIAL, sampler: null, ctx: null, onCut })
  const step = useCallback((frame: number) => {
    const l = live.current
    if (!l.sampler || !l.ctx) return
    const next = stepFollow(l.state, l.sampler(frame), l.ctx)
    const cut = next.cut
    const moved = cut !== null && cut !== l.state.cut
    l.state = next
    if (moved) l.onCut(cut)
  }, [])
  useEffect(() => {
    live.current.onCut = onCut
  }, [onCut])
  useEffect(() => {
    live.current.state = FOLLOW_INITIAL
    live.current.sampler = sampler
    live.current.ctx = ctx
    step(frameRef.current)
  }, [sampler, ctx, step, frameRef])
  return step
}
