/**
 * bombBlastFx.ts — LA DÉFLAGRATION D'UNE BOMBE D'ASSAUT. Logique pure, pas de React.
 *
 * POURQUOI CET EFFET EXISTE (retour utilisateur du 2026-08-31 : « faudrait un truc bien voyant
 * quand même »). L'Assaut publie depuis ce jour ses explosions (`bomb_detonations`, cf.
 * `objectiveevents/named.go`), et elles n'avaient qu'un FILIGRANE de fiche — 22 % d'opacité,
 * derrière le contenu, à l'autre bout de l'écran par rapport à la carte. Or en Assaut
 * l'explosion n'est pas un détail de score : c'est le SEUL événement qui décide de la manche,
 * et il n'y en a qu'une poignée par match. Il lui fallait la carte.
 *
 * TROIS COUCHES, ET C'EST CE QUI LE REND VOYANT — un anneau seul se lit comme une capture de
 * drapeau, ce qu'une explosion n'est pas :
 *
 *   1. l'ÉCLAT — un disque plein qui naît large et meurt vite (le premier quart de la vie) ;
 *   2. l'ONDE — un anneau épais qui s'ouvre bien plus loin que celui d'une capture
 *      (44 px contre 24) et s'amincit en se diluant ;
 *   3. les ÉCLATS — huit traits radiaux qui partent du point et s'éteignent. C'est eux qui
 *      disent « ça a explosé » plutôt que « ça a été capturé ».
 *
 * CE N'EST PAS UN PULSE D'OBJECTIF, et la différence est la même que pour l'onde de capture
 * (`flagCaptureFx`) : le pulse (`objectivesLayer.buildObjectivePulses`) pose l'action sur
 * l'ÉLÉMENT SERVI le plus proche. En Assaut il ne pose rien du tout — le catalogue d'objectifs
 * ne porte AUCUNE forme de site sur les cinq cartes d'Assaut (phase A3 du lot A). Cet effet-ci
 * lit la position RELUE de l'auteur, comme l'onde de capture.
 *
 * LA GARDE D'HORLOGE EST LA MÊME QUE CELLE DE L'ONDE DE CAPTURE : `a.t` n'est une frame du
 * document que si l'origine du film a été résolue (`filmClockTrusted`). Sinon l'écart est
 * INCONNU — mesuré de 3,6 s à 50,8 s selon le match — et la déflagration s'allumerait sur une
 * explosion qui n'a pas encore eu lieu. Muet vaut mieux que faux : la liste sort vide.
 *
 * AUCUNE POSITION DEVINÉE : une explosion dont l'auteur n'est pas localisable à cette image est
 * ÉCARTÉE. La poser au centre de la carte, ou sur le socle le plus proche, désignerait un lieu
 * que la mesure ne dit pas.
 *
 * AUCUN TOKEN ICI (règle color-tokens) : l'encre arrive résolue par l'appelant. C'est celle du
 * CAMP de l'auteur, comme pour l'onde de capture — sur la carte, le glyphe dit QUOI et la
 * couleur dit QUI, et en Assaut savoir quel camp a fait sauter la bombe est la moitié de
 * l'information.
 */
import { filmClockTrusted } from '@/lib/replay/scoreTimeline'

import { type CanvasView, projectTo } from '../replayView'
import { type XY } from '../replayLogic'
import type { ReplayDocumentReady } from '../replayNormalize'

/**
 * La stat d'une explosion, telle que le serveur la nomme (`objectiveevents/named.go`). En
 * Assaut un point de mode vaut UNE explosion, et rien d'autre ne fait bouger le score — c'est
 * la seule stat de la famille que le film réplique.
 */
export const BOMB_DETONATION_STAT = 'bomb_detonations'

/** Une déflagration prête à dessiner : son instant, le lieu RELU de son auteur, et son auteur. */
export interface BombBlastFx {
  /** Image du document (l'axe de `Point.T` — aucun recalage ici, cf. l'en-tête). */
  frame: number
  x: number
  y: number
  /** L'auteur : c'est lui qui donne l'encre du camp, résolue par l'appelant. */
  xuid: string
}

/**
 * buildBombBlastFx rend les explosions de bombe du film, posées au lieu de leur auteur.
 *
 * `posOf` PORTE DÉJÀ SA FENÊTRE APRÈS-MORT (patron `posOfPlayerAt`, les mêmes vies indexées que
 * le drapeau porté et la couronne) — et cette fenêtre compte ici plus qu'ailleurs : le poseur
 * d'une bombe meurt souvent DANS son explosion.
 */
export function buildBombBlastFx(
  doc: ReplayDocumentReady,
  posOf: (xuid: string, frame: number) => XY | null,
): BombBlastFx[] {
  if (doc.objectives.length === 0) return []
  if (!filmClockTrusted(doc)) return []
  const out: BombBlastFx[] = []
  for (const a of doc.objectives) {
    if (a.stat !== BOMB_DETONATION_STAT) continue
    const pos = posOf(a.xuid, a.t)
    if (!pos) continue
    out.push({ frame: a.t, x: pos.x, y: pos.y, xuid: a.xuid })
  }
  return out
}

/**
 * Durée de l'effet, EN IMAGES — 12 images ≈ 1,2 s au pas de 100 ms documenté du rejeu.
 *
 * DEUX FOIS l'onde de capture (6 images), et c'est délibéré : une capture de drapeau se répète
 * plusieurs fois par manche, une explosion d'Assaut arrive une à quatre fois par MATCH (relevé
 * A0.3 : 28 explosions sur 9 films). Un événement rare a le droit de tenir l'écran plus
 * longtemps — le risque de recouvrement, lui, est nul à cette densité.
 */
export const BOMB_BLAST_HOLD_FRAMES = 12

/** Fenêtre d'affichage (même forme que celle de l'onde de capture). */
export interface BombBlastWindow {
  frame: number
  hold: number
}

/** Style de l'effet : l'encre du camp de l'auteur est RÉSOLUE par l'appelant. */
export interface BombBlastStyle {
  /** Encre du camp de l'auteur vu de la page ; le neutre du thème quand il est inconnu. */
  inkOf: (xuid: string) => string
  /** Mouvement réduit : rien ne s'ouvre, l'empreinte ne fait que s'éteindre. */
  reducedMotion: boolean
}

// Géométrie, en pixels d'écran.
/** L'ÉCLAT : disque plein, large d'emblée, éteint au quart de la vie. */
const FLASH_R = 13
const FLASH_LIFE = 0.25
const FLASH_ALPHA = 0.55
/** L'ONDE : bien plus ample que celle d'une capture (24 px) — c'est ce qui la distingue de loin. */
const WAVE_START_R = 8
const WAVE_SPREAD = 36
const WAVE_START_W = 4
const WAVE_END_W = 0.8
const WAVE_ALPHA = 0.95
/** Les ÉCLATS : huit traits radiaux, du bord de l'éclat vers l'extérieur. */
const SHARD_COUNT = 8
const SHARD_INNER = 10
const SHARD_LEN = 16
const SHARD_W = 2
const SHARD_ALPHA = 0.8
/** Mouvement réduit : une empreinte FIXE — disque plein et anneau, aucun mouvement. */
const STATIC_R = 15
const STATIC_W = 3

/**
 * drawBombBlastFx dessine les explosions de la fenêtre courante.
 *
 * SOUS « MOUVEMENT RÉDUIT », UNE EMPREINTE FIXE QUI S'ÉTEINT — disque plein cerclé, sans
 * expansion ni éclats. C'est la règle de tous les effets du rejeu (`drawObjectivePulses`,
 * `drawFlagCaptureFx`) : ce qui disparaît est le MOUVEMENT, pas l'information. L'explosion
 * reste marquée, à sa place, pendant la même durée — et reste voyante, parce que le disque
 * plein l'est par lui-même.
 */
export function drawBombBlastFx(
  ctx: CanvasRenderingContext2D,
  fx: readonly BombBlastFx[],
  view: CanvasView,
  win: BombBlastWindow,
  style: BombBlastStyle,
): void {
  for (const f of fx) {
    const age = win.frame - f.frame
    if (age < 0 || age > win.hold) continue
    const k = age / Math.max(win.hold, 1)
    const c = projectTo(view, f)
    const ink = style.inkOf(f.xuid)
    ctx.fillStyle = ink
    ctx.strokeStyle = ink

    if (style.reducedMotion) {
      ctx.globalAlpha = FLASH_ALPHA * (1 - k)
      ctx.beginPath()
      ctx.arc(c.x, c.y, STATIC_R, 0, Math.PI * 2)
      ctx.fill()
      ctx.globalAlpha = WAVE_ALPHA * (1 - k)
      ctx.lineWidth = STATIC_W
      ctx.beginPath()
      ctx.arc(c.x, c.y, STATIC_R, 0, Math.PI * 2)
      ctx.stroke()
      continue
    }

    // 1. L'ÉCLAT — il ne vit que le premier quart, et son alpha meurt sur CE quart, pas sur la
    // vie entière : au-delà il ne resterait qu'un voile gris qui empâterait l'onde.
    if (k < FLASH_LIFE) {
      ctx.globalAlpha = FLASH_ALPHA * (1 - k / FLASH_LIFE)
      ctx.beginPath()
      ctx.arc(c.x, c.y, FLASH_R, 0, Math.PI * 2)
      ctx.fill()
    }
    // 2. L'ONDE.
    ctx.globalAlpha = WAVE_ALPHA * (1 - k)
    ctx.lineWidth = WAVE_START_W + (WAVE_END_W - WAVE_START_W) * k
    ctx.beginPath()
    ctx.arc(c.x, c.y, WAVE_START_R + WAVE_SPREAD * k, 0, Math.PI * 2)
    ctx.stroke()
    // 3. LES ÉCLATS — ils partent avec l'onde et s'éteignent plus tôt qu'elle (le carré du
    // temps), pour que la fin de l'effet soit un anneau seul et non un soleil qui s'efface.
    ctx.globalAlpha = SHARD_ALPHA * (1 - k) * (1 - k)
    ctx.lineWidth = SHARD_W
    const r0 = SHARD_INNER + WAVE_SPREAD * k
    ctx.beginPath()
    for (let i = 0; i < SHARD_COUNT; i++) {
      const a = (i * 2 * Math.PI) / SHARD_COUNT
      const cos = Math.cos(a)
      const sin = Math.sin(a)
      ctx.moveTo(c.x + cos * r0, c.y + sin * r0)
      ctx.lineTo(c.x + cos * (r0 + SHARD_LEN), c.y + sin * (r0 + SHARD_LEN))
    }
    ctx.stroke()
  }
  ctx.globalAlpha = 1
}
