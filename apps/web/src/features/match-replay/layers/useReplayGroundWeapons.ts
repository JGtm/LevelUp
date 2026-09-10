/**
 * useReplayGroundWeapons — LE CÂBLAGE DU CALQUE DES ARMES AU SOL, en un seul point.
 *
 * MÊME PARTI QUE `useReplayWeaponPads` ET `useReplayFlagCarries`, et pour la même raison :
 * `ReplayCanvas.tsx` porte un SEUIL DE TAILLE (`max-lines` eslint, R5) qui ne
 * remonte pas. Un calque de plus s'y branche par un import, un appel et une ligne de peinture —
 * tout le reste (cuisson des vignettes, mémoïsation, tracé, survol) vit ici.
 *
 * LES VIGNETTES SONT LES MÊMES QUE CELLES DES SOCLES, ET C'EST LA DONNÉE QUI LE DIT :
 * `groundWeapons[].w` est le MÊME identifiant de famille que `weaponPads[].weapon` et
 * `loadouts[].w`, donc la même clé dans `weaponLabels` — MODULO SA FORME (cf.
 * `weaponLabelKeyOf`, lot 6.5 du 2026-09-10 : les artefacts cuits l'écrivent en minuscules
 * sans préfixe ici, en majuscules préfixées côté socles et côté `weaponLabels` — un défaut de
 * jointure, pas un manque). La résolution passe par `padIconRefFor`/`padNameFor` — les
 * fonctions PURES qui portent déjà la règle (silhouette pleine plutôt que contour, miroir des
 * atlas, masques de HUD des power-ups, ET DÉSORMAIS la normalisation de clé). Elles sont
 * RÉUTILISÉES et non recopiées : une seconde règle de résolution divergerait au premier
 * ajustement, et l'écran montrerait la même arme de deux façons selon qu'elle est sur son
 * socle ou par terre.
 *
 * DEUXIÈME COPIE DE LA CUISSON (CLAUDE.md n°6) : l'effet ci-dessous a le même corps que celui
 * de `useReplayWeaponPads` — charger, teindre deux fois (corps et liseré), repeindre. Il en
 * diverge par ses ENCRES, qui sont le sujet de chaque calque. À la TROISIÈME copie, centraliser
 * et poser le garde-rail.
 *
 * LE SURVOL EXISTE DEPUIS LE LOT 6.5 (2026-09-10) — la réserve du 2026-08-30 ci-dessous est
 * DATÉE et LEVÉE, elle reste par exigence CLAUDE.md n°9 (doc inversée) : « PAS DE SURVOL, ET
 * C'EST DÉLIBÉRÉ (périmètre du lot du 2026-08-30) : le calque affiche, il n'interroge pas. »
 * C'était vrai jusqu'à ce lot ; ce n'est plus le cas. L'infobulle lit `dropper`/`picker` — DÉJÀ
 * publiés, déjà servis à chaque rejeu — via `nameOfSlot`, LA MÊME résolution frame-aware que
 * les poses d'équipement (`ownerNameOf` de `ReplayCanvasTips`). UNE LIGNE, jamais deux (cf.
 * `ReplayGroundWeaponTip.tsx`) : « <arme> · lâchée par X · reprise par Y ».
 */
import { useCallback, useEffect, useRef, useState, type PointerEvent, type RefObject } from 'react'

import { useTitleSlug } from '@/lib/title-routing'
import type { ReplayGroundWeapon } from '@/lib/api/types'

import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'
import { tintedIconCanvas } from './replayDraw'
import {
  drawGroundWeaponsLayer,
  groundWeaponAt,
  type GroundWeaponIcon,
  type GroundWeaponView,
} from './groundWeaponsLayer'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import type { XY } from '../../../lib/replay/replayLogic'
import { padIconRefFor, padNameFor } from './useReplayWeaponPads'

/** Ce qui est survolé : l'arme, où poser l'infobulle, et les DEUX lignes déjà composées. */
export interface GroundWeaponHover {
  item: ReplayGroundWeapon
  at: XY
  /** Le nom de l'arme, résolu comme celui d'un socle (bilingue, ou son identifiant à défaut). */
  weaponName: string
  /** « apparue », « lâchée par X », ou « lâchée par un joueur non nommé » — jamais les deux. */
  originLine: string
  /** « reprise par Y » (ou « ... non nommé ») ; `null` tant qu'aucun ramassage n'est mesuré. */
  pickupLine: string | null
}

export interface GroundWeaponsInput {
  doc: ReplayDocumentReady
  /** Le cadrage PARTAGÉ du canvas : tous les calques projettent la même scène. */
  view: GroundWeaponView
  /** Faux quand le calque est éteint : rien n'est dessiné, rien ne se survole. */
  enabled: boolean
  /**
   * Les deux encres : le CORPS à l'encre du marquage, le LISERÉ à l'encre NEUTRE.
   *
   * LE LISERÉ N'EST NI NOIR NI COLORÉ PAR NATURE. Noir, il se confondrait avec les contours des
   * cartes en niveaux de gris (retour utilisateur du 2026-08-28 sur les socles) ; teint d'une
   * couleur d'enjeu — l'or d'un power-up, l'orange d'une arme de puissance — il ferait lire une
   * arme abandonnée comme un socle. L'encre du « aucun camp » dit exactement ce qu'est l'objet.
   */
  ink: { fill: string; outline: string }
  /** Repeindre la scène : les vignettes arrivent après coup (chargement asynchrone). */
  redraw: () => void
  /** L'image courante, telle que la boucle de lecture la tient (pour le SURVOL uniquement). */
  frameRef: RefObject<number>
  /**
   * Le nom du joueur qui occupait un slot à une image donnée — LA MÊME résolution frame-aware
   * que celle des poses d'équipement (`useSlotIdentity.nameOfSlot`). `null` = slot libre ou
   * vie sans propriétaire connu : l'infobulle dit alors « joueur non nommé », jamais un nom
   * deviné.
   */
  nameOfSlot: (slot: number, frame: number) => string | null
  locale: ReplayLocale
}

export interface GroundWeapons {
  /** Le film porte-t-il des armes au sol ? Une bascule qui ne commande rien ne s'affiche pas. */
  available: boolean
  /**
   * LE NOM DU CALQUE, porté par le calque et non par la table de liaison du canvas
   * (2026-09-06, revue R1 constat C2) : c'est ce qui rend impossible de peindre ce geste
   * sous l'identité d'un autre calque.
   */
  id: 'armes-au-sol'
  /** Trace le calque à l'image demandée ; ne fait rien quand il est éteint. */
  paint: (ctx: CanvasRenderingContext2D, frame: number, k: number) => void
  hover: GroundWeaponHover | null
  onPointerMove: (event: PointerEvent<HTMLCanvasElement>) => void
  onPointerLeave: () => void
}

/**
 * groundWeaponOriginLine — LA PREMIÈRE LIGNE DE L'INFOBULLE : ce que l'origine mesurée permet
 * de dire, et rien de plus (cf. l'en-tête pour la mesure qui la justifie).
 */
function groundWeaponOriginLine(
  item: ReplayGroundWeapon,
  nameOfSlot: (slot: number, frame: number) => string | null,
  t: (typeof REPLAY_TEXT)['fr'],
): string {
  if (item.origin === 'spawned') return t.groundWeaponSpawned
  const name = nameOfSlot(item.dropper, item.t0)
  return name ? t.groundWeaponDroppedByFmt(name) : t.groundWeaponDropperUnknown
}

/** groundWeaponPickupLine — LA SECONDE LIGNE, ou `null` tant qu'aucune reprise n'est mesurée. */
function groundWeaponPickupLine(
  item: ReplayGroundWeapon,
  nameOfSlot: (slot: number, frame: number) => string | null,
  t: (typeof REPLAY_TEXT)['fr'],
): string | null {
  if (item.picker < 0) return null
  const name = nameOfSlot(item.picker, item.t1)
  return name ? t.groundWeaponPickedByFmt(name) : t.groundWeaponPickerUnknown
}

export function useReplayGroundWeapons({
  doc,
  view,
  enabled,
  ink,
  redraw,
  frameRef,
  nameOfSlot,
  locale,
}: GroundWeaponsInput): GroundWeapons {
  const items = doc.groundWeapons
  const labels = doc.weaponLabels
  const titleSlug = useTitleSlug()
  const t = REPLAY_TEXT[locale]

  // UNE TABLE PAR RÉFÉRENCE, pas un état : la remplir ne doit pas re-rendre la page (la boucle
  // de dessin la lit pendant qu'elle peint). Même règle que les vignettes de socle et de
  // grenade — le repeint passe par `redraw`, jamais par un `setState`.
  const iconsRef = useRef<Map<string, GroundWeaponIcon>>(new Map())

  useEffect(() => {
    // Une table NEUVE à chaque cuisson : garder l'ancienne servirait des vignettes teintes au
    // thème précédent le temps que les images se rechargent.
    const map = new Map<string, GroundWeaponIcon>()
    iconsRef.current = map
    const seen = new Set<string>()
    for (const item of items) {
      if (seen.has(item.w)) continue
      const ref = padIconRefFor(item.w, labels, titleSlug)
      if (!ref) continue
      seen.add(item.w)
      const { url, tinted, mirrored } = ref
      const weapon = item.w
      const im = new Image()
      im.onload = () => {
        map.set(weapon, {
          // Une image FINIE garde ses couleurs (`tinted` faux) sauf si le miroir l'oblige à
          // repasser par le canvas hors écran — exactement la règle des socles.
          fill: tinted || mirrored ? tintedIconCanvas(im, ink.fill, { mirrored, tinted }) : im,
          // Le LISERÉ ne demande que la SILHOUETTE : `source-in` la rend de n'importe quelle
          // image à alpha, image finie comprise.
          outline: tintedIconCanvas(im, ink.outline, { mirrored }),
        })
        redraw()
      }
      im.src = url
    }
  }, [items, labels, titleSlug, ink.fill, ink.outline, redraw])

  const paint = useCallback(
    (ctx: CanvasRenderingContext2D, frame: number, k: number) => {
      if (!enabled || items.length === 0) return
      drawGroundWeaponsLayer(ctx, items, view, { frame, k }, {
        iconOf: (weapon) => iconsRef.current.get(weapon) ?? null,
      })
    },
    [enabled, items, view],
  )

  const [hover, setHover] = useState<GroundWeaponHover | null>(null)

  const onPointerMove = useCallback(
    (event: PointerEvent<HTMLCanvasElement>) => {
      if (!enabled || items.length === 0 || view.width === 0) {
        setHover((prev) => (prev === null ? prev : null))
        return
      }
      const rect = event.currentTarget.getBoundingClientRect()
      // Le contexte dessine en pixels CSS ; le rapport ne vaut 1 que si la mise en page ne
      // remet pas le canevas à l'échelle — on le calcule plutôt que de le supposer (même
      // amorce que le survol des socles, des poses et des drapeaux).
      const kx = rect.width > 0 ? view.width / rect.width : 1
      const ky = rect.height > 0 ? view.height / rect.height : 1
      const at = { x: (event.clientX - rect.left) * kx, y: (event.clientY - rect.top) * ky }
      const found = groundWeaponAt(items, view, frameRef.current, window.devicePixelRatio || 1, at)
      setHover((prev) => {
        if (!found) return prev === null ? prev : null
        const next: GroundWeaponHover = {
          item: found,
          at,
          weaponName: padNameFor(found.w, labels, t, locale),
          originLine: groundWeaponOriginLine(found, nameOfSlot, t),
          pickupLine: groundWeaponPickupLine(found, nameOfSlot, t),
        }
        if (
          prev &&
          prev.item === next.item &&
          prev.at.x === at.x &&
          prev.at.y === at.y &&
          prev.originLine === next.originLine &&
          prev.pickupLine === next.pickupLine
        ) {
          return prev
        }
        return next
      })
    },
    [enabled, items, view, frameRef, labels, t, locale, nameOfSlot],
  )

  const onPointerLeave = useCallback(() => {
    setHover((prev) => (prev === null ? prev : null))
  }, [])

  return { id: 'armes-au-sol', available: items.length > 0, paint, hover, onPointerMove, onPointerLeave }
}
