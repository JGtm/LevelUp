/**
 * useReplayWeaponPads — TOUT LE CÂBLAGE DU CALQUE DES SOCLES, en un seul point.
 *
 * POURQUOI UN HOOK ET NON QUATRE MORCEAUX DANS `ReplayCanvas`. Le canvas du rejeu porte une
 * dette de taille GELÉE par un cliquet (cf. placementFamily.guard.test.ts) : toute addition s'y
 * fait par extraction, jamais par empilement. Ce hook réunit donc les quatre préoccupations du
 * calque — la cuisson des vignettes, le tracé par image, le survol et l'infobulle — et n'en
 * rend au canvas que trois lignes utiles : `paint`, les deux gestionnaires de pointeur, et le
 * survol courant. Même parti que `usePlacementHover` et `useSlotIdentity` avant lui.
 *
 * LES VIGNETTES SONT CUITES UNE FOIS PAR THÈME, pas par image. Ce sont les MÊMES icônes que
 * les fiches joueur — celles du document (`weaponLabels[id].img`, 168 PNG extraits du jeu) —
 * et la plupart sont des MASQUES à teindre : un canvas ne connaît pas le `mask-image` du CSS,
 * la teinte se fait donc hors écran par composition (`tintedIconCanvas`, le même utilitaire que
 * les vignettes de grenade). Une famille sans visuel n'en emprunte aucun : le calque lui dessine
 * un glyphe neutre.
 *
 * L'IMAGE COURANTE EST LUE DANS UNE RÉFÉRENCE, jamais dans un état React (même règle et même
 * conséquence assumée que `usePlacementHover`) : si l'état d'un socle change SOUS un pointeur
 * immobile, son infobulle attend le prochain mouvement. À l'arrêt — le cas où l'on inspecte —
 * la lecture est exacte.
 */
import { useCallback, useEffect, useMemo, useRef, useState, type PointerEvent, type RefObject } from 'react'

import { catalogText } from './catalogLabel'
import { REPLAY_TEXT, type ReplayLocale } from './i18n'
import type { PlacementView } from './placementShapes'
import { tintedIconCanvas } from './replayDraw'
import { frameToMs, type XY } from './replayLogic'
import type { ReplayDocumentReady, ReplayWeaponPadReady } from './replayNormalize'
import { padScaleOf } from './weaponPadFamilies'
import {
  drawWeaponPadsLayer,
  padAt,
  padRespawnSecondsAt,
  padStateAt,
  type PadState,
} from './weaponPadsLayer'

/** Ce qui est survolé : le socle, son état LU À CET INSTANT, et où poser l'infobulle. */
export interface WeaponPadHover {
  pad: ReplayWeaponPadReady
  at: XY
  /** Nom de l'arme dans la langue du lecteur, ou son identifiant quand rien ne la nomme. */
  name: string
  state: PadState
  /** Secondes avant la réapparition attendue, ou null (cycle non établi, socle non vide). */
  respawnS: number | null
}

export interface WeaponPadsInput {
  doc: ReplayDocumentReady
  view: PlacementView
  /** L'image courante, telle que la boucle de lecture la tient. */
  frameRef: RefObject<number>
  /** Faux quand le calque est éteint : rien n'est dessiné, rien ne se survole. */
  enabled: boolean
  /** Encre neutre du thème et contour des étiquettes (les deux viennent du canvas). */
  ink: { neutral: string; labelStroke: string }
  locale: ReplayLocale
  /** Repeindre la scène : les vignettes arrivent après coup (chargement asynchrone). */
  redraw: () => void
}

export interface WeaponPads {
  /** Le film porte-t-il des socles ? Une bascule qui ne commande rien ne s'affiche pas. */
  available: boolean
  /** Trace le calque à l'image demandée ; ne fait rien quand il est éteint. */
  paint: (ctx: CanvasRenderingContext2D, frame: number, k: number) => void
  hover: WeaponPadHover | null
  onPointerMove: (event: PointerEvent<HTMLCanvasElement>) => void
  onPointerLeave: () => void
}

export function useReplayWeaponPads({
  doc,
  view,
  frameRef,
  enabled,
  ink,
  locale,
  redraw,
}: WeaponPadsInput): WeaponPads {
  const pads = doc.weaponPads
  const [hover, setHover] = useState<WeaponPadHover | null>(null)
  const iconsRef = useRef<Map<string, CanvasImageSource>>(new Map())

  // LA TAILLE SUIT LA CLÉ CANONIQUE de l'arme (`weaponLabels[id].key`, posée à la requête),
  // jamais son hexadécimal ni son libellé : c'est le vocabulaire commun aux tables du client.
  const labels = doc.weaponLabels
  const scaleOf = useCallback(
    (weapon: string) => padScaleOf(labels?.[weapon]?.key),
    [labels],
  )
  const nameOf = useCallback(
    (weapon: string) => catalogText(labels?.[weapon], locale) ?? weapon,
    [labels, locale],
  )

  // LES VIGNETTES, cuites une fois par document ET par encre : un masque se teint hors écran,
  // une image finie se pose telle quelle (même contrat `tinted` que WeaponIcon).
  useEffect(() => {
    const map = new Map<string, CanvasImageSource>()
    iconsRef.current = map
    const seen = new Set<string>()
    for (const pad of pads) {
      const label = labels?.[pad.weapon]
      if (!label?.img || seen.has(pad.weapon)) continue
      seen.add(pad.weapon)
      const weapon = pad.weapon
      const tinted = label.tinted
      const im = new Image()
      im.onload = () => {
        map.set(weapon, tinted ? tintedIconCanvas(im, ink.neutral) : im)
        redraw()
      }
      im.src = label.img
    }
  }, [pads, labels, ink.neutral, redraw])

  const frameMs = useMemo(() => frameToMs(1, doc), [doc])
  const t = REPLAY_TEXT[locale]

  const paint = useCallback(
    (ctx: CanvasRenderingContext2D, frame: number, k: number) => {
      if (!enabled || pads.length === 0) return
      drawWeaponPadsLayer(
        ctx,
        pads,
        view,
        { frame, frameMs, k },
        {
          ink: ink.neutral,
          labelStroke: ink.labelStroke,
          iconOf: (weapon) => iconsRef.current.get(weapon) ?? null,
          scaleOf,
          countdownLabel: t.padCountdownFmt,
        },
      )
    },
    [enabled, pads, view, frameMs, ink.neutral, ink.labelStroke, scaleOf, t.padCountdownFmt],
  )

  const onPointerMove = useCallback(
    (event: PointerEvent<HTMLCanvasElement>) => {
      if (!enabled || pads.length === 0 || view.width === 0) {
        setHover((prev) => (prev === null ? prev : null))
        return
      }
      const rect = event.currentTarget.getBoundingClientRect()
      // Le contexte dessine en pixels CSS ; le rapport ne vaut 1 que si la mise en page ne
      // remet pas le canevas à l'échelle — on le calcule plutôt que de le supposer (même
      // amorce que le survol des poses).
      const kx = rect.width > 0 ? view.width / rect.width : 1
      const ky = rect.height > 0 ? view.height / rect.height : 1
      const at = { x: (event.clientX - rect.left) * kx, y: (event.clientY - rect.top) * ky }
      const style = {
        ink: ink.neutral,
        labelStroke: ink.labelStroke,
        iconOf: () => null,
        scaleOf,
        countdownLabel: t.padCountdownFmt,
      }
      const found = padAt(pads, view, style, window.devicePixelRatio || 1, at)
      setHover((prev) => {
        if (!found) return prev === null ? prev : null
        const frame = frameRef.current
        const next: WeaponPadHover = {
          pad: found,
          at,
          name: nameOf(found.weapon),
          state: padStateAt(found, frame),
          respawnS: padRespawnSecondsAt(found, frame, frameMs),
        }
        if (
          prev &&
          prev.pad === next.pad &&
          prev.state === next.state &&
          prev.respawnS === next.respawnS &&
          prev.at.x === at.x &&
          prev.at.y === at.y
        ) {
          return prev
        }
        return next
      })
    },
    [enabled, pads, view, ink.neutral, ink.labelStroke, scaleOf, t.padCountdownFmt, frameRef, frameMs, nameOf],
  )

  const onPointerLeave = useCallback(() => {
    setHover((prev) => (prev === null ? prev : null))
  }, [])

  return { available: pads.length > 0, paint, hover, onPointerMove, onPointerLeave }
}
