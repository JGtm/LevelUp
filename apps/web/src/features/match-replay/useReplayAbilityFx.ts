/**
 * useReplayAbilityFx — LE CÂBLAGE DES GESTES DE CAPACITÉ SUR LEUR PORTEUR, en un point.
 *
 * DEUX GESTES, UNE MÊME NATURE : la LIGNE DE GRAPPIN (schéma 8, `doc.grappleLines`) et la
 * POUSSÉE DU PROPULSEUR (schéma 38, `doc.abilityImpulses`). Ni l'une ni l'autre ne pose quoi
 * que ce soit sur le terrain — ce sont des capacités qui agissent SUR LE JOUEUR, elles se
 * dessinent donc sur son pion, juste au-dessus des trajectoires et sous les effets de tir.
 * `equipmentPlacementsLayer` les met déjà toutes deux à `null` pour la même raison.
 *
 * POURQUOI UN HOOK PLUTÔT QUE DEUX APPELS DANS LE CANVAS. Le canvas du rejeu porte une dette
 * de taille GELÉE par un cliquet (`placementFamily.guard.test.ts`) : toute addition s'y paie
 * par une EXTRACTION — c'est la doctrine que ce cliquet documente depuis sa dix-septième.
 * Le dash du propulseur demandait ses lignes de glue alors que le fichier était à trois lignes
 * de son plafond. Regrouper les deux gestes de capacité rend au canvas plus qu'il ne lui prend,
 * et le regroupement n'est pas de circonstance : c'est la même famille de geste, le même
 * moment de la pile de calques, la même absence d'objet posé.
 *
 * AUCUNE LOGIQUE NE SE DÉPLACE : la fenêtre mesurée du grappin, l'orientation du dash et leurs
 * tracés restent dans `grappleLayer.ts` et `thrusterDashFx.ts`. Ce fichier ne fait que joindre
 * et appeler.
 */
import { useCallback, useMemo } from 'react'

import { buildGrappleFx, drawGrappleLayer } from './grappleLayer'
import { frameToMs, msToFrames } from './replayLogic'
import type { ReplayDocumentReady } from './replayNormalize'
import {
  buildThrusterDashFx,
  drawThrusterDashLayer,
  THRUSTER_DASH_HEADING_MS,
  type CanvasView,
} from './thrusterDashFx'

export interface AbilityFxHookInput {
  doc: ReplayDocumentReady
  view: CanvasView
  /** Encre du câble de grappin : l'encre neutre du thème (cf. useReplayInks). */
  grappleInk: string
  /** Couleur d'équipe de la vie, à une image donnée — le dash porte celle de son porteur. */
  colorOfSlot: (slot: number, frame: number) => string | null
  reducedMotion: boolean
}

export interface ReplayAbilityFx {
  /**
   * Peint les gestes de capacité actifs à l'image demandée. No-op quand il n'y en a aucun.
   *
   * PAS DE DRAPEAU `available` ICI, à la différence de `useReplayVipCrown` / `bombBlast` : leur
   * drapeau a un lecteur (le bandeau de disponibilité des calques, `ReplayCanvas.tsx`), celui-ci
   * n'en aurait aucun — ni bascule de tiroir, ni ligne de disponibilité pour ces deux gestes.
   * Un booléen sans lecteur est du code mort (CLAUDE.md n°7) ; la garde de vide vit dans
   * `paint`, où elle sert vraiment.
   */
  paint: (ctx: CanvasRenderingContext2D, frame: number, k: number) => void
}

export function useReplayAbilityFx({
  doc,
  view,
  grappleInk,
  colorOfSlot,
  reducedMotion,
}: AbilityFxHookInput): ReplayAbilityFx {
  // Les tractions de grappin, jointes une fois aux points de leur vie (schéma 8).
  const grappleFx = useMemo(() => buildGrappleFx(doc), [doc])
  // Les poussées de propulseur : la fenêtre de lecture de direction est déclarée en TEMPS RÉEL
  // et convertie ici, parce que la durée d'une image dépend du document.
  const dashFx = useMemo(
    () => buildThrusterDashFx(doc, msToFrames(THRUSTER_DASH_HEADING_MS, doc)),
    [doc],
  )
  const frameMs = useMemo(() => frameToMs(1, doc), [doc])

  const paint = useCallback(
    (ctx: CanvasRenderingContext2D, frame: number, k: number) => {
      // LA LIGNE DE GRAPPIN d'abord : c'est un lien joueur -> point d'accroche, il se lit sur la
      // trajectoire sans couvrir les événements. Fenêtre MESURÉE [t0, t1] — la ligne suit le
      // joueur qui se déplace vers l'ancre, puis disparaît à l'arrivée. Statique par image
      // (mouvement réduit respecté par construction).
      if (grappleFx.length > 0) drawGrappleLayer(ctx, grappleFx, view, frame, grappleInk)
      // LE DASH DU PROPULSEUR ensuite : un sillage bref derrière le pion, orienté par le
      // déplacement lu autour de l'instant publié.
      if (dashFx.length > 0) {
        drawThrusterDashLayer(ctx, dashFx, view, { frame, frameMs, k, reducedMotion }, {
          colorOfSlot,
        })
      }
    },
    [grappleFx, dashFx, view, grappleInk, frameMs, reducedMotion, colorOfSlot],
  )

  return { paint }
}
