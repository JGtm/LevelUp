/**
 * useReplayKillFocus — LA MISE EN VALEUR D'UNE ÉLIMINATION, câblée au canvas : elle lit la demande du
 * magasin de lecture, la résout sur le film, peint ses deux calques et fait tourner leur animation.
 *
 * CE QUE LE HOOK PORTE, ET QUI N'EST PAS DANS LE DESSIN (`killFocusLayer`, `killFocusLogic`) :
 *
 *  - LA RÉSOLUTION : la demande ne porte que des xuids et une image ; ce hook y met les positions
 *    (celles du porteur : le bipède, ou son véhicule), la fiche occupée, la couleur et le nom, tels
 *    que le canvas les résout pour les pions — l'étiquette dit le même nom que le pion qu'elle
 *    remplace, dans la même couleur.
 *  - L'HORLOGE DE L'ANIMATION. La lecture est en PAUSE quand les équerres se verrouillent : ni
 *    l'image du rejeu ni la boucle de lecture ne tournent. Le temps est donc le temps RÉEL
 *    (`performance.now`), mesuré depuis le jeton du clic, et une boucle d'images d'écran ne
 *    demande de repeindre que PENDANT l'animation — elle s'arrête d'elle-même à `FOCUS_MOTION_MS`.
 *    `prefers-reduced-motion` : pas de boucle, les poses d'arrivée tout de suite.
 *  - L'ICÔNE DE L'ARME, chargée et teintée à l'encre d'étiquette (`loadedImage`, `tintedIconCanvas`),
 *    comme les vignettes des autres calques : le repeint passe par `redraw`, jamais par un rendu.
 *  - LE NOM DU PION : l'étiquette le REMPLACE. `nameHiddenAtSlot` dit au calque des joueurs quels
 *    slots ne portent plus leur nom pendant la mise en valeur.
 *  - LE NETTOYAGE : un autre document (autre match) ou le démontage efface la mise en valeur.
 *
 * Deux calques, une seule résolution : le voile et les équerres (`mise-en-valeur`), puis les
 * étiquettes (`etiquettes-mise-en-valeur`), dans l'ordre de `replayCompose`.
 */
import { useCallback, useEffect, useMemo, useRef } from 'react'

import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import { isAliveAt } from '../../../lib/replay/replayLogic'
import { useCarrierPosAt } from '../model/carrierPosition'
import {
  hiddenNameSlots,
  resolveKillFocus,
  type KillFocusResolvers,
} from '../model/killFocusLogic'
import { FOCUS_MOTION_MS } from '../model/killFocusMotion'
import { buildLivesByXuid } from '../model/livesPosition'
import { usePlaybackFocus, type PlaybackStore } from '../model/playbackStore'
import type { CanvasView } from '../model/replayView'
import { drawKillFocusLabels, drawKillFocusVeil, type KillFocusInk } from './killFocusLayer'
import { withLoadedImage } from './loadedImage'
import type { NamedLayerPainter } from './replayCompose'
import { tintedIconCanvas } from './replayDraw'

export interface KillFocusOptions {
  doc: ReplayDocumentReady
  view: CanvasView
  store: PlaybackStore
  /** Les encres : le voile (jeton `scrim`) et l'encre sombre des étiquettes (`labelStroke`). */
  scrimInk: string
  labelInk: string
  /** Les lectures de pion du canvas : fiche occupée, couleur et nom par joueur (`useSlotIdentity`). */
  seatedOfXuid: KillFocusResolvers['seated']
  colorOfXuid: KillFocusResolvers['colorOf']
  nameOfXuid: KillFocusResolvers['nameOf']
  reducedMotion: boolean
  /** Seul et Masquer (`useReplayPlayerFocus`) : un acteur non affiché n'est pas mis en valeur. */
  shownAtSlot?: KillFocusResolvers['shownAtSlot']
  /** Repeindre la scène (hors boucle de lecture) : le canvas la fournit. */
  redraw: () => void
}

export interface KillFocusLayers {
  veil: NamedLayerPainter<'mise-en-valeur'>
  labels: NamedLayerPainter<'etiquettes-mise-en-valeur'>
  /** Vrai pour un slot dont l'étiquette pleine remplace le nom sous le pion. */
  nameHiddenAtSlot: (slot: number) => boolean
}

export function useReplayKillFocus(o: KillFocusOptions): KillFocusLayers {
  const { doc, view, store, scrimInk, labelInk, seatedOfXuid, colorOfXuid, nameOfXuid } = o
  const { reducedMotion, redraw, shownAtSlot } = o
  const ink = useMemo<KillFocusInk>(() => ({ veil: scrimInk, label: labelInk }), [scrimInk, labelInk])
  const request = usePlaybackFocus(store)
  const posOf = useCarrierPosAt(doc)
  const lives = useMemo(() => buildLivesByXuid(doc.tracks), [doc.tracks])

  const resolved = useMemo(() => {
    if (!request) return null
    return resolveKillFocus(request, {
      posOf,
      seated: seatedOfXuid,
      colorOf: colorOfXuid,
      nameOf: nameOfXuid,
      slotOf: (xuid, frame) => lives.get(xuid)?.find((life) => isAliveAt(life, frame))?.slot ?? null,
      shownAtSlot,
    })
  }, [request, posOf, lives, seatedOfXuid, colorOfXuid, nameOfXuid, shownAtSlot])

  // UN AUTRE MATCH OU LE DÉMONTAGE : la mise en valeur d'un document n'a pas de sens dans un autre.
  useEffect(() => () => store.clearFocus(), [doc, store])

  // L'HORLOGE : le temps réel du clic, et la boucle qui repeint tant que ça bouge.
  const startedAt = useRef(0)
  const token = resolved?.token
  useEffect(() => {
    if (token === undefined) return
    const start = performance.now()
    startedAt.current = start
    if (reducedMotion) return
    let raf = 0
    const step = () => {
      redraw()
      if (performance.now() - start < FOCUS_MOTION_MS) raf = requestAnimationFrame(step)
    }
    raf = requestAnimationFrame(step)
    return () => cancelAnimationFrame(raf)
  }, [token, reducedMotion, redraw])
  const elapsed = useCallback(
    () => (reducedMotion ? FOCUS_MOTION_MS : performance.now() - startedAt.current),
    [reducedMotion],
  )

  // L'ICÔNE DE L'ARME DU TUEUR, teinte à l'encre d'étiquette (un masque) ou telle quelle (image finie).
  const weaponUrl = resolved?.weapon.imageUrl ?? ''
  const weaponTinted = resolved?.weapon.tinted ?? false
  const iconRef = useRef<HTMLCanvasElement | HTMLImageElement | null>(null)
  useEffect(() => {
    iconRef.current = null
    if (!weaponUrl) return
    let stale = false
    withLoadedImage(weaponUrl, (im) => {
      if (stale) return
      iconRef.current = weaponTinted ? tintedIconCanvas(im, ink.label) : im
      redraw()
    })
    return () => {
      stale = true
    }
  }, [weaponUrl, weaponTinted, ink.label, redraw])

  const veilPaint = useCallback(
    (ctx: CanvasRenderingContext2D, _frame: number, k: number) =>
      drawKillFocusVeil(ctx, resolved, view, { k, elapsedMs: elapsed(), ink }),
    [resolved, view, ink, elapsed],
  )
  const labelsPaint = useCallback(
    (ctx: CanvasRenderingContext2D, _frame: number, k: number) =>
      drawKillFocusLabels(ctx, resolved, view, {
        k, elapsedMs: elapsed(), ink, weaponIcon: iconRef.current,
      }),
    [resolved, view, ink, elapsed],
  )
  const hidden = useMemo(() => hiddenNameSlots(resolved), [resolved])
  const nameHiddenAtSlot = useCallback((slot: number) => hidden.has(slot), [hidden])

  return useMemo(
    () => ({
      veil: { id: 'mise-en-valeur' as const, paint: veilPaint },
      labels: { id: 'etiquettes-mise-en-valeur' as const, paint: labelsPaint },
      nameHiddenAtSlot,
    }),
    [veilPaint, labelsPaint, nameHiddenAtSlot],
  )
}
