/**
 * useReplayFullscreen — LE MODE PLEIN ÉCRAN DU REJEU : la carte, les fiches et le fil, seuls à
 * l'écran.
 *
 * # DEUX ÉTAGES, ET LE SECOND EST UN REPLI
 *
 * 1. Une SUPERPOSITION plein cadre : la page pose `REPLAY_FULLSCREEN_FRAME` sur le conteneur de
 *    sa grille, qui recouvre alors toute la fenêtre (en-tête et fil d'Ariane compris). C'est le
 *    mode lui-même, et il marche partout.
 * 2. Le plein écran NATIF de TOUTE LA PAGE (`document.documentElement`), demandé en même temps.
 *    La page entière et non la grille : le tiroir de réglages est rendu par portail dans
 *    `document.body`, hors de la grille — un plein écran sur la grille le laisserait derrière.
 *    API absente (navigateur, cadre sans permission) ou demande refusée : la superposition seule
 *    reste, et c'est déjà le mode.
 *
 * # COMMENT ON EN SORT
 *
 * - Le bouton ou la touche F : sortie du mode ET du plein écran natif.
 * - Échap en plein écran natif : le NAVIGATEUR en sort de lui-même, sans toujours livrer la
 *   frappe à la page. `fullscreenchange` est donc la source de vérité : le natif qui tombe ferme
 *   le mode.
 * - Échap en superposition seule : ce qui est ouvert LE PLUS À L'INTÉRIEUR se ferme d'abord
 *   (tiroir, média agrandi, menu de vitesse), le mode ensuite. Ces couches se signalent par
 *   l'attribut `REPLAY_ESCAPE_LAYER_ATTR` sur leur racine ; l'écouteur du mode est posé en
 *   CAPTURE, donc il voit la frappe AVANT elles — la couche est encore dans le DOM, il s'abstient.
 *   En bulle, l'ordre dépendrait de l'ordre d'abonnement, et React vide ses mises à jour entre
 *   deux écouteurs : la couche aurait déjà disparu, et une seule frappe fermerait les deux.
 *
 * # PENDANT UN EXPORT VIDÉO, LE MODE NE BASCULE PAS
 *
 * Le bouton est désactivé et la touche F sans effet tant qu'un export tient la toile
 * (`useExportActive`). Échap, lui, sort toujours : en natif le navigateur le fait de toute façon,
 * et l'export dessine à son propre format — la mise en page de l'écran ne touche pas le clip.
 *
 * # AUCUN ÉTAT DANS L'URL
 *
 * Le mode est une manière de regarder, pas un endroit : il ne survit ni à un rechargement ni à
 * une navigation, et quitter la page rend le plein écran natif au navigateur.
 */
import { useCallback, useEffect, useRef, useState } from 'react'

import { useExportActive } from '../export/exportLayoutStore'

/**
 * L'ATTRIBUT D'UNE COUCHE QU'ÉCHAP FERME AVANT LE MODE. Posé en clair (`data-replay-escape-layer=""`)
 * sur la racine du tiroir, du média agrandi et du menu de vitesse ; le garde-rail
 * `escapeLayers.guard.test.ts` exige qu'aucun écouteur d'Échap du rejeu ne s'en passe.
 */
export const REPLAY_ESCAPE_LAYER_ATTR = 'data-replay-escape-layer'

/**
 * LE CADRE DU MODE, posé sur le conteneur de la grille de la page. Il DÉFILE LUI-MÊME
 * (`overflow-y-auto`) : `useReplayViewport` mesure la hauteur offerte au terrain par rapport au
 * premier ancêtre qui défile, et c'est ce cadre qu'il doit trouver. `z-40` : au-dessus des
 * languettes fixes de l'application, au-dessous du tiroir de réglages (`z-50`, en portail).
 */
export const REPLAY_FULLSCREEN_FRAME =
  'fixed inset-0 z-40 overflow-y-auto overscroll-contain bg-background p-3'

export interface ReplayFullscreen {
  /** Le mode est ouvert : la page pose `REPLAY_FULLSCREEN_FRAME` sur sa grille. */
  active: boolean
  /** Un export vidéo tient la toile : le mode ne s'ouvre ni ne se ferme au bouton ou à F. */
  disabled: boolean
  /** Ouvre ou ferme le mode. À appeler DANS le geste (clic, touche) : le natif l'exige. */
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

/** Demande le plein écran natif de la page ; sans API, la superposition seule fait le mode. */
function requestNative(): void {
  const root = document.documentElement
  if (!document.fullscreenEnabled || typeof root.requestFullscreen !== 'function') return
  root.requestFullscreen().catch((err: unknown) => {
    console.warn('[replay-fullscreen] plein écran natif refusé : superposition seule', err)
  })
}

export function useReplayFullscreen(): ReplayFullscreen {
  const disabled = useExportActive()
  const [active, setActive] = useState(false)
  // LA VÉRITÉ DU GESTE, lue hors rendu : `fullscreenchange` et Échap arrivent entre deux rendus.
  const activeRef = useRef(false)
  // LE NATIF A-T-IL ÉTÉ ATTEINT depuis l'ouverture ? Seule sa chute ferme le mode : un
  // `fullscreenchange` qui ne nous concerne pas (une vidéo mise en plein écran) n'y touche pas.
  const reachedRef = useRef(false)

  const close = useCallback(() => {
    activeRef.current = false
    reachedRef.current = false
    setActive(false)
    leaveNative()
  }, [])

  const toggle = useCallback(() => {
    if (disabled) return
    if (activeRef.current) {
      close()
      return
    }
    activeRef.current = true
    setActive(true)
    requestNative()
  }, [disabled, close])

  useEffect(() => {
    function onChange() {
      if (nativeIsOurs()) {
        reachedRef.current = true
        // LA DEMANDE A ABOUTI APRÈS UNE SORTIE (F pressé deux fois avant la transition) : on
        // la défait, sans quoi la page resterait en plein écran natif hors du mode.
        if (!activeRef.current) leaveNative()
        return
      }
      if (document.fullscreenElement === null && reachedRef.current) close()
    }
    document.addEventListener('fullscreenchange', onChange)
    return () => document.removeEventListener('fullscreenchange', onChange)
  }, [close])

  useEffect(() => {
    if (!active) return
    function onKeyDown(e: KeyboardEvent) {
      if (e.key !== 'Escape') return
      if (document.querySelector(`[${REPLAY_ESCAPE_LAYER_ATTR}]`)) return
      close()
    }
    window.addEventListener('keydown', onKeyDown, true)
    return () => window.removeEventListener('keydown', onKeyDown, true)
  }, [active, close])

  // LE CADRE CHANGE SANS QUE LA FENÊTRE CHANGE : en superposition seule, rien ne prévient ceux
  // qui se calent sur la fenêtre (la hauteur offerte au terrain, la position du tiroir). Le
  // basculement leur est donc annoncé comme un redimensionnement — c'en est un pour eux. Le
  // premier rendu n'annonce rien : la page n'a pas encore changé de cadre.
  const announcedRef = useRef(active)
  useEffect(() => {
    if (announcedRef.current === active) return
    announcedRef.current = active
    window.dispatchEvent(new Event('resize'))
  }, [active])

  // QUITTER LA PAGE REND LE NATIF : le mode n'existe plus, la page suivante ne l'a pas demandé.
  useEffect(() => () => leaveNative(), [])

  return { active, disabled, toggle }
}
