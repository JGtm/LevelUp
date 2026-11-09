/**
 * placementShroud — LA BULLE de l'écran occultant (shroud screen) : sa portée, sa durée de
 * repli, et sa forme sur la carte.
 *
 * POURQUOI UN FICHIER À PART. La bulle a gagné deux encres et des éclairs qui scintillent :
 * comme la faille (`placementRift.ts`) et le mur (`placementWall.ts`), elle sort de
 * `placementShapes.ts` pour garder celui-ci sous le seuil de 500 lignes.
 *
 * Comme le reste du dossier : pas de React, pas de couleur écrite. Les encres arrivent du
 * calque appelant, qui les tient des tokens du thème (`useReplayInks`).
 */
import type { XY } from '../../../lib/replay/replayLogic'
import type { ShapeStyle } from './placementShapes'

/**
 * Rayon de l'écran occultant, en mètres monde. VALEUR DÉCLARÉE, PAS MESURÉE, PAS OFFICIELLE —
 * même statut que celui du champ de réparation, et il faut le dire aussi franchement.
 *
 * LES TROIS SOURCES SONT VIDES, ET C'EST LA RAISON D'ÊTRE DU BORD FLOU. Le film ne porte
 * aucune portée pour cet objet ; la source officielle qui chiffre le détecteur (4,25 m) ne
 * chiffre pas cet écran ; et rien n'a été mesuré dans le corpus. 6 m — 12 m de diamètre — est
 * un choix d'ÉCRAN, calibré sur ce que la bulle du jeu couvre en pratique (un couloir, un
 * passage étroit), et volontairement DEUX FOIS le champ de réparation pour que les trois
 * disques du calque (capteur 4,25 m, champ 3 m, écran 6 m) ne se confondent jamais.
 *
 * LE BORD FLOU PORTE CETTE RÉSERVE À L'ÉCRAN, comme le pointillé la porte pour le champ. Deux
 * conventions différentes pour la même réserve, et c'est voulu : le champ a une BORNE dont on
 * doute (un anneau, en pointillé), l'écran n'a pas de borne du tout (un dégradé, sans anneau).
 * Une limite nette dirait « au-delà de ce trait on se voit », ce que personne n'a établi.
 */
export const SHROUD_RADIUS_M = 6

/**
 * LA DURÉE DE REPLI DE L'ÉCRAN OCCULTANT, en ms : 7,5 s, LUE DANS LE FILM. À l'impact, le film fait naître un
 * objet que la pile i28 ne nomme pas et le supprime 7,51 s plus tard (38 écrans sur 39, cinq films de 2025-11 à
 * 2026-07), durée confirmée en jeu par l'utilisateur. Depuis le schéma 106 le document porte cet objet
 * (`tracked[].effect`) : sa naissance et sa suppression LUES bornent la pose (`spawnedObjects.ts`). Cette durée ne
 * sert plus qu'à un écran dont la bulle n'est pas lue (artefact antérieur, association ambiguë, suppression non
 * lue). Le `t1` de l'objet engendré, lui, est la fin de la GRENADE (0,1 à 0,4 s après le repos). La fenêtre de la
 * pose (`placementWindow.ts`) lit cette durée à cet endroit et nulle part ailleurs.
 */
export const SHROUD_DURATION_MS = 7_500

/** Les deux encres de la bulle : le bleu nuit du cœur, le bleu électrique du bord et des éclairs. */
export interface ShroudInk {
  core: string
  glow: string
}

/** Part du rayon occupée par le fondu du bord (verdict R2-6 : 22 %). */
const SHROUD_FADE_RATIO = 0.22
/** Opacité du cœur. Presque plein : le verdict du 2026-08-27 est « opaque ». */
const SHROUD_CORE_ALPHA = 0.94
/** Où, dans le fondu, l'encre électrique culmine avant de s'éteindre. */
const SHROUD_GLOW_STOP = 0.45

/**
 * LE SCINTILLEMENT DES ÉCLAIRS — la MÊME courbe que ceux de la fiche (`@keyframes
 * replay-zone-bolt`, globals.css) : un double battement bref dans un cycle de 2,6 s. Les deux
 * copies (CSS pour la fiche, canevas ici) doivent rester identiques, sinon la carte et la fiche
 * d'un joueur dans la bulle ne battraient plus ensemble.
 */
const SHROUD_BOLT_CYCLE_MS = 2_600
const SHROUD_BOLT_KEYS: readonly (readonly [number, number])[] = [
  [0, 0], [0.06, 0], [0.08, 0.95], [0.11, 0.15], [0.14, 0.8], [0.22, 0], [1, 0],
]

/**
 * Décalage de chaque éclair dans le cycle, en secondes — partagé avec la fiche
 * (`ReplayPlayerCard.SHROUD_BOLTS`), pour que les éclairs de la carte et ceux de la fiche
 * s'allument au même instant.
 */
export const SHROUD_BOLT_OFFSETS_S = [0, 1.3, 2.05] as const

/**
 * La géométrie des éclairs, en fractions du rayon plein : abscisse du tracé et zigzag de haut
 * en bas. Ils restent DANS le cœur — un éclair qui déborderait dessinerait la borne que le
 * fondu refuse de poser.
 */
const SHROUD_BOLT_X = [-0.38, 0.3, -0.02] as const
const SHROUD_BOLT_PATH: readonly XY[] = [
  { x: 0.1, y: -0.62 }, { x: -0.12, y: -0.04 }, { x: 0.06, y: -0.04 }, { x: -0.1, y: 0.62 },
]
const SHROUD_BOLT_WIDTH = 1.6
const SHROUD_BOLT_BLUR = 6

/**
 * shroudBoltAlpha — l'opacité d'un éclair `ageMs` après la pose, décalé de `offsetS` dans le
 * cycle. L'horloge est celle de la POSE, comme sur la fiche : après un saut de lecture, chaque
 * éclair reprend à son avancement réel, jamais sur un rythme inventé.
 */
export function shroudBoltAlpha(ageMs: number, offsetS: number): number {
  const t = (ageMs + offsetS * 1000) / SHROUD_BOLT_CYCLE_MS
  const phase = t - Math.floor(t)
  for (let i = 1; i < SHROUD_BOLT_KEYS.length; i++) {
    const [p1, a1] = SHROUD_BOLT_KEYS[i]
    if (phase > p1) continue
    const [p0, a0] = SHROUD_BOLT_KEYS[i - 1]
    return a0 + ((a1 - a0) * (phase - p0)) / (p1 - p0)
  }
  return 0
}

/** La bulle à dessiner : son centre projeté, son rayon en pixels, son âge depuis la pose. */
export interface ShroudShape {
  c: XY
  radiusPx: number
  ageMs: number
}

/**
 * drawShroud — LA BULLE de l'écran occultant.
 *
 * OPAQUE, ET LES PIONS AU-DESSUS (verdict utilisateur du 2026-08-27, parmi trois propositions
 * mises côte à côte sur la planche). L'opacité dit « ici, on ne se voit pas » — c'est la
 * fonction même de l'objet ; les pions passent au-dessus sans une ligne de code ici, parce que
 * le calque des poses est tracé AVANT celui des pistes.
 *
 * BLEU NUIT, BORD ÉLECTRIQUE, ÉCLAIRS (décision utilisateur du 2026-10-30, qui remplace l'encre
 * neutre) : la bulle reprend l'électricité bleue de l'objet du jeu, déjà portée par la fiche
 * d'un joueur qui s'y trouve. Les éclairs sont un SIGNE, pas une mesure : rien dans le film ne
 * bat à ce rythme, d'où leur cycle emprunté à la fiche. Sous mouvement réduit ils restent
 * éteints, comme sur la fiche : le disque dit déjà l'état sans clignoter.
 */
export function drawShroud(
  ctx: CanvasRenderingContext2D,
  shape: ShroudShape,
  style: ShapeStyle,
  ink: ShroudInk,
): void {
  const { c, radiusPx } = shape
  if (!(radiusPx > 0)) return
  const plein = radiusPx * (1 - SHROUD_FADE_RATIO)
  ctx.save()
  ctx.fillStyle = ink.core
  ctx.globalAlpha = SHROUD_CORE_ALPHA
  ctx.beginPath()
  ctx.arc(c.x, c.y, plein, 0, Math.PI * 2)
  ctx.fill()
  // Le fondu : du cœur, par l'encre électrique, jusqu'au néant — sans jamais poser de trait.
  // C'est l'ABSENCE d'anneau qui dit que la portée n'est pas connue.
  const g = ctx.createRadialGradient(c.x, c.y, plein, c.x, c.y, radiusPx)
  g.addColorStop(0, ink.core)
  g.addColorStop(SHROUD_GLOW_STOP, ink.glow)
  g.addColorStop(1, 'transparent')
  ctx.fillStyle = g
  ctx.beginPath()
  ctx.arc(c.x, c.y, radiusPx, 0, Math.PI * 2)
  ctx.fill()
  if (!style.reducedMotion) drawShroudBolts(ctx, { ...shape, radiusPx: plein }, style, ink.glow)
  ctx.restore()
}

/** drawShroudBolts — les éclairs allumés à cet instant ; rien entre deux battements. */
function drawShroudBolts(
  ctx: CanvasRenderingContext2D,
  shape: ShroudShape,
  style: ShapeStyle,
  color: string,
): void {
  const { c, radiusPx: r, ageMs } = shape
  ctx.strokeStyle = color
  ctx.shadowColor = color
  ctx.shadowBlur = SHROUD_BOLT_BLUR * style.k
  ctx.lineWidth = SHROUD_BOLT_WIDTH * style.k
  ctx.lineJoin = 'miter'
  SHROUD_BOLT_OFFSETS_S.forEach((offsetS, i) => {
    const alpha = shroudBoltAlpha(ageMs, offsetS)
    if (!(alpha > 0)) return
    const x0 = c.x + SHROUD_BOLT_X[i] * r
    ctx.globalAlpha = alpha
    ctx.beginPath()
    SHROUD_BOLT_PATH.forEach((p, j) => {
      if (j === 0) ctx.moveTo(x0 + p.x * r, c.y + p.y * r)
      else ctx.lineTo(x0 + p.x * r, c.y + p.y * r)
    })
    ctx.stroke()
  })
}
