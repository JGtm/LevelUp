/**
 * useReplayFullscreen — LE PLEIN ÉCRAN DU SYSTÈME et la sortie du Plein cadre par Échap.
 *
 * # DEUX GESTES DISTINCTS
 *
 * 1. Le PLEIN CADRE (touche F, bouton, `useReplayFrame`) est la superposition seule : la page pose
 *    `REPLAY_FULLSCREEN_FRAME` sur le conteneur de sa grille, qui recouvre alors toute la fenêtre
 *    (en-tête et fil d'Ariane compris). Il ne demande rien au navigateur.
 * 2. Le PLEIN ÉCRAN DU SYSTÈME (`toggle`) est le natif seul : la page entière
 *    (`document.documentElement`), jamais la grille — le tiroir de réglages est rendu par portail
 *    dans `document.body`, hors de la grille, et un plein écran sur la grille le laisserait derrière.
 *    Il s'ajoute à n'importe quel cadre. API absente (navigateur, cadre sans permission) ou demande
 *    refusée : aucun natif, le cadre reste ce qu'il est, et le natif n'est plus voulu : le geste suivant
 *    le redemande.
 *
 * DEPUIS LE STANDARD, le geste du système entre d'abord en Plein cadre : le natif seul sur la page
 * à largeur bornée n'a aucun sens. Ce Plein cadre-là appartient au geste : il se referme avec lui
 * (second geste, ou natif qui tombe), et le natif part avec lui par quelque chemin que le Plein cadre
 * se referme (F, icône du Standard, Échap, « Quitter »). Un Plein cadre ouvert autrement (F) ne se
 * referme jamais avec le natif ; et le geste du système depuis le Cinéma ou le Plein cadre n'en change pas.
 *
 * # COMMENT ON EN SORT
 *
 * - `fullscreenchange` est la source de vérité du natif : Échap en plein écran natif, le NAVIGATEUR
 *   en sort de lui-même, sans toujours livrer la frappe à la page. Un natif qui tombe est suivi.
 * - Échap en Plein cadre : ce qui est ouvert LE PLUS À L'INTÉRIEUR se ferme d'abord (tiroir, média
 *   agrandi, menu de vitesse), le Plein cadre ensuite, qui rend le cadre d'où l'on vient. Ces couches
 *   se signalent par l'attribut `REPLAY_ESCAPE_LAYER_ATTR` sur leur racine ; l'écouteur est posé en
 *   CAPTURE, donc il voit la frappe AVANT elles — la couche est encore dans le DOM, il s'abstient.
 *   En bulle, l'ordre dépendrait de l'ordre d'abonnement, et React vide ses mises à jour entre deux
 *   écouteurs : la couche aurait déjà disparu, et une seule frappe fermerait les deux.
 * - Échap hors Plein cadre (Standard, Cinéma) : rien. T et F en sortent.
 *
 * # PENDANT UN EXPORT VIDÉO, LE GESTE NE BASCULE PAS
 *
 * Le bouton est désactivé et le geste sans effet tant qu'un export tient la toile (`disabled`, lu
 * sur le cadre). Échap, lui, sort toujours : en natif le navigateur le fait de toute façon, et
 * l'export dessine à son propre format — la mise en page de l'écran ne touche pas le clip.
 *
 * # AUCUN ÉTAT DANS L'URL
 *
 * Une manière de regarder, pas un endroit : ni un rechargement ni une navigation ne le gardent, et
 * quitter la page rend le plein écran natif au navigateur.
 */
import { useCallback, useEffect, useRef, useState } from 'react'

import type { ReplayFrameControl } from './useReplayFrame'

/**
 * L'ATTRIBUT D'UNE COUCHE QU'ÉCHAP FERME AVANT LE PLEIN CADRE. Posé en clair (`data-replay-escape-layer=""`)
 * sur la racine du panneau de la vue, de la fiche de joueur, du média agrandi et des menus ; le garde-rail
 * `escapeLayers.guard.test.ts` exige qu'aucun écouteur d'Échap du rejeu ne s'en passe.
 */
export const REPLAY_ESCAPE_LAYER_ATTR = 'data-replay-escape-layer'

/**
 * LE CADRE DU PLEIN CADRE, posé sur le conteneur de la grille de la page. Il DÉFILE LUI-MÊME
 * (`overflow-y-auto`) : `useReplayViewport` mesure la hauteur offerte au terrain par rapport au
 * premier ancêtre qui défile, et c'est ce cadre qu'il doit trouver. `z-40` : au-dessus des
 * languettes fixes de l'application, au-dessous du tiroir de réglages (`z-50`, en portail).
 *
 * `[align-content:safe_center]` CENTRE LA RANGÉE DANS LE CADRE (régression constatée le
 * 2026-10-21) : la hauteur de la rangée suit la carte, plafonnée par `CANVAS_HEIGHT_CEILING`
 * (720 px, garde-fou mémoire) — sur un écran plus haut que ça en plein écran, la grille (une
 * seule rangée, pas de `grid-template-rows`) restait plus courte que le cadre, et un conteneur
 * bloc aligne son contenu en haut par défaut : la carte se retrouvait collée en haut, avec du
 * vide sous elle au lieu d'être centrée. `safe` évite le symétrique sur un cadre plus COURT que
 * la rangée (BTB, fiches nombreuses) : sans lui, un centrage pur pousserait le haut du contenu
 * hors champ au premier rendu, avant tout défilement.
 */
export const REPLAY_FULLSCREEN_FRAME =
  'fixed inset-0 z-40 overflow-y-auto overscroll-contain bg-background p-3 [align-content:safe_center]'

export interface ReplayFullscreen {
  /** Le plein écran du système est atteint, et par nous (le seul demandeur de la racine). */
  active: boolean
  /** Un export vidéo tient la toile : le geste du système est sans effet. */
  disabled: boolean
  /** Entre dans le plein écran du système, ou en sort. À appeler DANS le geste (clic) : le natif l'exige. */
  toggle: () => void
}

/** La page est-elle en plein écran natif, et par nous (le seul demandeur de la racine) ? */
function nativeIsOurs(): boolean {
  return document.fullscreenElement === document.documentElement
}

/** Rend le plein écran natif au navigateur, s'il est le nôtre. */
function leaveNative(): void {
  if (!nativeIsOurs()) return
  document.exitFullscreen().catch((err: unknown) => {
    console.warn('[replay-fullscreen] sortie du plein écran natif refusée', err)
  })
}

/**
 * Demande le plein écran natif de la page ; sans API, aucun natif et le cadre reste ce qu'il est.
 * `onUnavailable` est appelé quand le natif ne sera pas atteint : API absente (aussitôt) ou demande refusée.
 */
function requestNative(onUnavailable: () => void): void {
  const root = document.documentElement
  if (!document.fullscreenEnabled || typeof root.requestFullscreen !== 'function') {
    onUnavailable()
    return
  }
  root.requestFullscreen().catch((err: unknown) => {
    console.warn('[replay-fullscreen] plein écran natif refusé', err)
    onUnavailable()
  })
}

export function useReplayFullscreen(control: ReplayFrameControl): ReplayFullscreen {
  const { frame, disabled, peek, toggleFullFrame, exit } = control
  const [active, setActive] = useState(false)
  // LA VÉRITÉ DU GESTE, lue hors rendu : `fullscreenchange` et Échap arrivent entre deux rendus.
  // Le natif est-il VOULU (demandé, pas forcément encore accordé) ?
  const wantedRef = useRef(false)
  // LE NATIF A-T-IL ÉTÉ ATTEINT depuis la demande ? Seule sa chute est suivie : un
  // `fullscreenchange` qui ne nous concerne pas (une vidéo mise en plein écran) n'y touche pas.
  const reachedRef = useRef(false)
  // LE PLEIN CADRE EN COURS A-T-IL ÉTÉ OUVERT PAR CE GESTE (depuis le Standard) ? Il se referme alors avec lui.
  const ownsFrameRef = useRef(false)

  const release = useCallback(() => {
    wantedRef.current = false
    reachedRef.current = false
    setActive(false)
    leaveNative()
    if (!ownsFrameRef.current) return
    ownsFrameRef.current = false
    exit()
  }, [exit])

  const toggle = useCallback(() => {
    if (disabled) return
    if (wantedRef.current || ownsFrameRef.current) {
      release()
      return
    }
    if (peek().frame === 'standard') {
      toggleFullFrame()
      ownsFrameRef.current = peek().frame === 'plein-cadre'
    }
    wantedRef.current = true
    // Un natif refusé ou absent n'est plus voulu : sans cela le geste suivant « quitterait » un natif jamais atteint.
    requestNative(() => {
      if (!reachedRef.current) wantedRef.current = false
    })
  }, [disabled, release, peek, toggleFullFrame])

  useEffect(() => {
    function onChange() {
      if (nativeIsOurs()) {
        reachedRef.current = true
        // LA DEMANDE A ABOUTI APRÈS UNE SORTIE (geste répété avant la transition) : on la défait,
        // sans quoi la page resterait en plein écran natif sans que personne ne le veuille.
        if (wantedRef.current) setActive(true)
        else leaveNative()
        return
      }
      if (document.fullscreenElement === null && reachedRef.current) release()
    }
    document.addEventListener('fullscreenchange', onChange)
    return () => document.removeEventListener('fullscreenchange', onChange)
  }, [release])

  // UN PLEIN CADRE DU GESTE REFERMÉ AILLEURS (F, icône du Standard, « Quitter ») REND LE NATIF : il n'est plus celui
  // du geste, et le natif qui l'accompagnait n'a plus de raison d'être. `release` n'appelle pas `exit` : le cadre
  // est déjà refermé.
  useEffect(() => {
    if (frame === 'plein-cadre' || !ownsFrameRef.current) return
    ownsFrameRef.current = false
    release()
  }, [frame, release])

  useEffect(() => {
    if (frame !== 'plein-cadre') return
    function onKeyDown(e: KeyboardEvent) {
      if (e.key !== 'Escape') return
      if (document.querySelector(`[${REPLAY_ESCAPE_LAYER_ATTR}]`)) return
      if (ownsFrameRef.current) release()
      else exit()
    }
    window.addEventListener('keydown', onKeyDown, true)
    return () => window.removeEventListener('keydown', onKeyDown, true)
  }, [frame, release, exit])

  // QUITTER LA PAGE REND LE NATIF : la page suivante ne l'a pas demandé.
  useEffect(() => () => leaveNative(), [])

  return { active, disabled, toggle }
}
