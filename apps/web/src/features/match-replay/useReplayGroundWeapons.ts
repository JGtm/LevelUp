/**
 * useReplayGroundWeapons — LE CÂBLAGE DU CALQUE DES ARMES AU SOL, en un seul point.
 *
 * MÊME PARTI QUE `useReplayWeaponPads` ET `useReplayFlagCarries`, et pour la même raison :
 * `ReplayCanvas.tsx` porte un SEUIL DE TAILLE (`max-lines` eslint, R5) qui ne
 * remonte pas. Un calque de plus s'y branche par un import, un appel et une ligne de peinture —
 * tout le reste (cuisson des vignettes, mémoïsation, tracé) vit ici.
 *
 * LES VIGNETTES SONT LES MÊMES QUE CELLES DES SOCLES, ET C'EST LA DONNÉE QUI LE DIT :
 * `groundWeapons[].w` est le MÊME identifiant de famille que `weaponPads[].weapon` et
 * `loadouts[].w`, donc la même clé dans `weaponLabels`. La résolution passe par
 * `padIconRefFor` — la fonction PURE qui porte déjà la règle (silhouette pleine plutôt que
 * contour, miroir des atlas, masques de HUD des power-ups). Elle est RÉUTILISÉE et non
 * recopiée : une seconde règle de résolution divergerait au premier ajustement, et l'écran
 * montrerait la même arme de deux façons selon qu'elle est sur son socle ou par terre.
 *
 * DEUXIÈME COPIE DE LA CUISSON (CLAUDE.md n°6) : l'effet ci-dessous a le même corps que celui
 * de `useReplayWeaponPads` — charger, teindre deux fois (corps et liseré), repeindre. Il en
 * diverge par ses ENCRES, qui sont le sujet de chaque calque. À la TROISIÈME copie, centraliser
 * et poser le garde-rail.
 *
 * PAS DE SURVOL, ET C'EST DÉLIBÉRÉ (périmètre du lot du 2026-08-30) : le calque affiche, il
 * n'interroge pas. Le rendre survolable demanderait un quatrième calque dans `hoverLayers` et
 * une quatrième infobulle dans `ReplayCanvasTips` — un lot en soi, à ouvrir si le besoin se
 * confirme à l'écran. Ce qu'on perd est nommé : le NOM de l'arme au sol ne se lit nulle part,
 * seule sa silhouette la dit.
 */
import { useCallback, useEffect, useRef } from 'react'

import { useTitleSlug } from '@/lib/title-routing'

import { tintedIconCanvas } from './replayDraw'
import {
  drawGroundWeaponsLayer,
  type GroundWeaponIcon,
  type GroundWeaponView,
} from './groundWeaponsLayer'
import type { ReplayDocumentReady } from './replayNormalize'
import { padIconRefFor } from './useReplayWeaponPads'

export interface GroundWeaponsInput {
  doc: ReplayDocumentReady
  /** Le cadrage PARTAGÉ du canvas : tous les calques projettent la même scène. */
  view: GroundWeaponView
  /** Faux quand le calque est éteint : rien n'est dessiné. */
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
}

export interface GroundWeapons {
  /** Le film porte-t-il des armes au sol ? Une bascule qui ne commande rien ne s'affiche pas. */
  available: boolean
  /** Trace le calque à l'image demandée ; ne fait rien quand il est éteint. */
  paint: (ctx: CanvasRenderingContext2D, frame: number, k: number) => void
}

export function useReplayGroundWeapons({
  doc,
  view,
  enabled,
  ink,
  redraw,
}: GroundWeaponsInput): GroundWeapons {
  const items = doc.groundWeapons
  const labels = doc.weaponLabels
  const titleSlug = useTitleSlug()
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

  return { available: items.length > 0, paint }
}
