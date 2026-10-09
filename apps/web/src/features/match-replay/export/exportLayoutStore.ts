/**
 * exportLayoutStore.ts — LA MISE EN PAGE D'EXPORT EN VIGUEUR, partagee entre la boucle d'export
 * (hors React) et le cadrage de la toile (`useReplayView`, dans React).
 *
 * # POURQUOI UN MAGASIN DE MODULE (remplace `exportRenderScale`, 2026-09-16)
 *
 * L'ancien suréchantillonnage n'etait qu'un FACTEUR lu au dimensionnement du backing store : un
 * objet mutable suffisait. Le rendu au format cible change la GEOMETRIE — largeur et hauteur de
 * dessin, donc le cadre de la scene, la projection, les calques cuits et chaque calque lie a
 * `canvasView`. Tout cela est memoise dans React : il faut un re-rendu, donc un magasin auquel
 * `useReplayView` s'abonne (`useSyncExternalStore`). Le faire descendre en prop traverserait
 * `ReplayCanvas`, qui est a son plafond de taille.
 *
 * # DEMANDEE / APPLIQUEE : POURQUOI DEUX VALEURS
 *
 * La boucle d'export DEMANDE une mise en page, puis ne doit peindre sa premiere image qu'une fois
 * React l'a APPLIQUEE — rendu, calques statiques recuits, `draw` republie. `useReplayView` le
 * signale depuis un effet ; les effets du meme composant qui suivent (cuisson des calques,
 * publication de `draw`) s'executent dans le meme passage, avant que la boucle ne reprenne la
 * main. La densite de pixels suit la valeur APPLIQUEE : ainsi le backing store et les calques
 * cuits ne changent de densite qu'avec la geometrie qui va avec.
 *
 * UNE SEULE TOILE DE REJEU PAR PAGE : c'est deja l'hypothese de l'ancien facteur de module.
 */
import { useEffect, useSyncExternalStore } from 'react'

import { yieldToEvents } from './eventLoopYield'
import type { ExportFormat, ExportLayout } from './exportFormats'

let requested: ExportLayout | null = null
let applied: ExportLayout | null = null
const listeners = new Set<() => void>()

function subscribe(fn: () => void): () => void {
  listeners.add(fn)
  return () => listeners.delete(fn)
}

function snapshot(): ExportLayout | null {
  return requested
}

/**
 * requestExportLayout — pose (ou retire, `null`) la mise en page d'export. Seul ecrivain : la
 * boucle d'export, qui la retire dans un `finally` sur tous les chemins.
 */
export function requestExportLayout(layout: ExportLayout | null): void {
  requested = layout
  for (const fn of listeners) fn()
}

/** La mise en page d'export demandee, et un re-rendu a chaque changement. */
export function useExportLayout(): ExportLayout | null {
  const layout = useSyncExternalStore(subscribe, snapshot, snapshot)
  // L'ACCUSE D'APPLICATION, apres le commit du rendu qui a lu cette valeur (cf. l'en-tete).
  useEffect(() => {
    applied = layout
  }, [layout])
  return layout
}

/** La mise en page demandee (`null` hors export) : son identite marque UN export. */
export function exportLayoutRequested(): ExportLayout | null {
  return requested
}

function activeSnapshot(): boolean {
  return requested !== null
}

/**
 * useExportActive — un re-rendu a l'entree et a la sortie de chaque export, SANS accuser
 * d'application (c'est le role de `useExportLayout`, tenu par le cadrage). Sert aux memos qui
 * dependent du theme : les encres de l'export sont celles du theme sombre (cf. `themeInk.ts`).
 */
export function useExportActive(): boolean {
  return useSyncExternalStore(subscribe, activeSnapshot, activeSnapshot)
}

/**
 * isExportActive — un export tient-il la toile ? VRAI DES LA DEMANDE, avant meme que React
 * l'applique : c'est ce que lisent les gestes hors React (survol, cf. `hoverLayers.ts`) pour
 * se taire pendant `prepare` et `encode` (decision D7).
 */
export function isExportActive(): boolean {
  return requested !== null
}

/** La mise en page demandee est-elle celle que la toile a deja prise ? */
export function isExportLayoutApplied(): boolean {
  return applied === requested
}

/**
 * canvasPixelRatio — la densite a laquelle la toile et ses calques cuits se rendent : celle de
 * la mise en page d'export appliquee, sinon celle de l'ecran. PENDANT UN EXPORT, LE
 * `devicePixelRatio` N'ENTRE NULLE PART : c'est ce qui rend le fichier identique d'une machine
 * a l'autre.
 */
export function canvasPixelRatio(): number {
  return applied ? applied.pixelRatio : window.devicePixelRatio || 1
}

/**
 * Le delai au-dela duquel une mise en page demandee et jamais appliquee est une panne. En pratique
 * elle s'applique au premier retour a la boucle d'evenements (un rendu synchrone de React) ;
 * sans toile montee, elle ne s'appliquera jamais, et l'export doit le dire plutot qu'attendre.
 */
export const EXPORT_LAYOUT_TIMEOUT_MS = 5000

/**
 * waitForExportLayout — rend la main quand la toile est REELLEMENT au format : mise en page
 * appliquee par React, puis backing store redimensionne par un trace a exactement
 * `format.width` x `format.height`.
 *
 * LA TAILLE DU BACKING STORE EST LA PREUVE, PAS UNE SUPPOSITION : l'encodeur est configure au
 * format, et une toile d'une autre taille serait rognee ou refusee. Leve apres
 * `EXPORT_LAYOUT_TIMEOUT_MS` — la boucle d'export en fait un echec affiche.
 */
export async function waitForExportLayout(
  canvas: HTMLCanvasElement,
  format: ExportFormat,
  redraw: () => void,
): Promise<void> {
  const deadline = performance.now() + EXPORT_LAYOUT_TIMEOUT_MS
  for (;;) {
    if (isExportLayoutApplied()) {
      redraw()
      if (canvas.width === format.width && canvas.height === format.height) return
    }
    if (performance.now() > deadline) {
      throw new Error(
        `mise en page d'export non appliquee (toile ${canvas.width}x${canvas.height}, attendu ${format.width}x${format.height})`,
      )
    }
    await yieldToEvents()
  }
}
