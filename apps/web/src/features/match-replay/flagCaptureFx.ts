/**
 * flagCaptureFx.ts — L'ONDE DE CHOC D'UNE CAPTURE DE DRAPEAU. Logique pure, pas de React.
 *
 * POURQUOI CET EFFET EXISTE (retour utilisateur du 2026-08-27 : « effet mini onde de choc à la
 * capture »). La vie des drapeaux (`flagCarriesLayer.ts`) décrit des DURÉES — porté, au sol, à la
 * base — et une durée ne se voit pas commencer. La capture, elle, est l'INSTANT qui compte dans
 * une partie de CTF : sans marqueur ponctuel, elle se lit après coup, quand le glyphe est déjà
 * rentré à sa base. Deux anneaux qui s'ouvrent et s'éteignent la posent à l'endroit et à la
 * seconde où elle a eu lieu.
 *
 * CE N'EST PAS LE RETOUR DES PULSES RETIRÉS, et la différence est nette. Le pulse de CTF
 * (`objectivesLayer.buildObjectivePulses`) était un SUBSTITUT faute d'objet : il posait TOUTE
 * action de la famille `flag_` sur l'élément statique le plus proche de son auteur — un socle
 * voisin, jamais le lieu de l'action. Il reste retiré (`flagPulsesRetired`), et rien ici ne le
 * rallume. Cet effet-ci ne connaît qu'UNE stat, `flag_captures`, et il la pose à la position
 * RELUE de son auteur : ni la même source, ni le même lieu, ni le même nombre.
 *
 * LA GARDE D'HORLOGE EST LA MÊME QUE CELLE DES PULSES, et pour la même raison : `a.t` n'est une
 * frame du document que si l'origine du film a été résolue (`filmClockTrusted`). Sinon les
 * instants sont décalés d'un écart INCONNU — mesuré de 3,6 s à 50,8 s selon le match — et l'onde
 * s'ouvrirait sur une capture qui n'a pas encore eu lieu. Un effet muet vaut mieux qu'un effet
 * faux : la liste sort vide.
 *
 * AUCUNE POSITION DEVINÉE : une capture dont l'auteur n'est pas localisable à cette image est
 * ÉCARTÉE. Poser l'anneau au centre de la carte, ou sur le socle le plus proche, désignerait un
 * lieu que la mesure ne dit pas — c'est exactement le défaut du substitut qu'on a retiré.
 *
 * AUCUN TOKEN ICI (règle color-tokens) : l'encre arrive résolue par l'appelant, comme pour le
 * calque des drapeaux.
 */
import { filmClockTrusted } from '@/lib/replay/scoreTimeline'

import { type CanvasView, projectTo } from './replayView'
import { type XY } from './replayLogic'
import type { ReplayDocumentReady } from './replayNormalize'

/**
 * La stat d'une capture, telle que le serveur la nomme (`objectiveevents/named.go`). C'est la
 * SEULE de la famille `flag_` qui donne une onde : un `flag_grabs` ou un `flag_returns` est déjà
 * lisible sur le glyphe vivant (il change d'état), une capture, elle, ne laisse rien derrière.
 */
const FLAG_CAPTURE_STAT = 'flag_captures'

/** Une capture prête à dessiner : son instant, le lieu RELU de son auteur, et son auteur. */
export interface FlagCaptureFx {
  /** Image du document (l'axe de `Point.T` — aucun recalage ici, cf. l'en-tête). */
  frame: number
  x: number
  y: number
  /** L'auteur : c'est lui qui donne l'encre du camp, résolue par l'appelant. */
  xuid: string
}

/**
 * buildFlagCaptureFx rend les captures de drapeau du film, posées au lieu de leur auteur.
 *
 * `posOf` PORTE DÉJÀ SA FENÊTRE APRÈS-MORT : c'est la relecture du hook appelant (patron
 * `posOfPlayerAt`, mêmes vies indexées que le glyphe porté). Elle n'est pas repassée en
 * paramètre — deux fenêtres pour la même lecture divergeraient au premier réglage, et le
 * décalage ne se verrait pas à l'écran.
 */
export function buildFlagCaptureFx(
  doc: ReplayDocumentReady,
  posOf: (xuid: string, frame: number) => XY | null,
): FlagCaptureFx[] {
  if (doc.objectives.length === 0) return []
  if (!filmClockTrusted(doc)) return []
  const out: FlagCaptureFx[] = []
  for (const a of doc.objectives) {
    if (a.stat !== FLAG_CAPTURE_STAT) continue
    const pos = posOf(a.xuid, a.t)
    if (!pos) continue
    out.push({ frame: a.t, x: pos.x, y: pos.y, xuid: a.xuid })
  }
  return out
}

/**
 * Durée de l'effet, EN IMAGES — 6 images ≈ 600 ms au pas de 100 ms documenté du rejeu.
 *
 * En images pour la même raison que le clignotement du calque voisin : le tracé ne reçoit que
 * l'index d'image, jamais l'horloge du document. Assez long pour être vu à 1x, assez court pour
 * ne pas se superposer à la capture suivante (la plus rapide mesurée sur les films CTF témoins
 * laisse plusieurs secondes entre deux).
 */
export const FLAG_CAPTURE_HOLD_FRAMES = 6

/** Fenêtre d'affichage (même forme que celle des pulses d'objectif). */
export interface FlagCaptureWindow {
  frame: number
  hold: number
}

/** Style de l'effet : l'encre du camp de l'auteur est RÉSOLUE par l'appelant. */
export interface FlagCaptureStyle {
  /** Encre du camp de l'auteur vu de la page ; le neutre du thème quand il est inconnu. */
  inkOf: (xuid: string) => string
  /** Mouvement réduit : l'onde ne s'ouvre plus, elle ne fait que s'éteindre. */
  reducedMotion: boolean
}

// Géométrie de l'onde, en pixels d'écran. DEUX anneaux et non un seul : un anneau unique qui
// s'écarte se lit comme un halo, deux anneaux décalés se lisent comme une onde qui part d'un
// point — c'est ce que le retour demandait. Ils s'amincissent en s'ouvrant, comme une onde qui
// se dilue, et non l'inverse.
const RING_START_R = 6
const RING_SPREAD = 18
/** Retard du second anneau, en pixels de rayon : c'est lui qui donne le sens de propagation. */
const RING_GAP = 5
const RING_START_W = 2.6
const RING_END_W = 0.6
const RING_ALPHA = 0.9
/** Mouvement réduit : un rayon FIXE, celui des pulses d'objectif (même famille de repère). */
const RING_STATIC_R = 11

/**
 * drawFlagCaptureFx dessine les captures de la fenêtre courante : deux anneaux concentriques qui
 * s'ouvrent en s'éteignant.
 *
 * SOUS « MOUVEMENT RÉDUIT », UN SEUL ANNEAU, SANS EXPANSION — il ne fait que s'éteindre. C'est la
 * règle de tous les effets du rejeu (`drawObjectivePulses`) : ce qui doit disparaître, c'est le
 * MOUVEMENT, pas l'information. L'instant reste marqué, à sa place, pendant la même durée.
 */
export function drawFlagCaptureFx(
  ctx: CanvasRenderingContext2D,
  fx: readonly FlagCaptureFx[],
  view: CanvasView,
  win: FlagCaptureWindow,
  style: FlagCaptureStyle,
): void {
  for (const f of fx) {
    const age = win.frame - f.frame
    if (age < 0 || age > win.hold) continue
    const k = age / Math.max(win.hold, 1)
    const c = projectTo(view, f)
    ctx.strokeStyle = style.inkOf(f.xuid)
    ctx.globalAlpha = RING_ALPHA * (1 - k)
    if (style.reducedMotion) {
      ctx.lineWidth = RING_START_W
      ctx.beginPath()
      ctx.arc(c.x, c.y, RING_STATIC_R, 0, Math.PI * 2)
      ctx.stroke()
      continue
    }
    ctx.lineWidth = RING_START_W + (RING_END_W - RING_START_W) * k
    const r = RING_START_R + RING_SPREAD * k
    ctx.beginPath()
    ctx.arc(c.x, c.y, r, 0, Math.PI * 2)
    ctx.stroke()
    // Le second anneau TRAÎNE derrière le premier, sans jamais s'inverser : au tout début de la
    // course son rayon serait négatif, et un rayon négatif fait lever `arc`.
    ctx.beginPath()
    ctx.arc(c.x, c.y, Math.max(r - RING_GAP, 1), 0, Math.PI * 2)
    ctx.stroke()
  }
  ctx.globalAlpha = 1
}
