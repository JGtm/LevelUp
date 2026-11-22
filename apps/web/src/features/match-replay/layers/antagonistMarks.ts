/**
 * antagonistMarks.ts : LES TITRES D'ANTAGONISTE SUR LA CARTE DU REJEU. L'icône némésis ou
 * souffre-douleur devant le nom du joueur qui tient le titre, avec son compte, et la pulsation
 * quand le titre change de main.
 *
 * CE QUE CE FICHIER PORTE, ET CE QU'IL NE DÉCIDE PAS. Qui tient quel titre à quelle image, et où
 * en est la pulsation, vient de `model/antagonistTitles` (`SlotTitle`). Ici on ne fait que dessiner
 * un titre : l'icône teintée (`createTitleIcons`), la suite « icône, compte » qui précède un
 * texte (`drawTitleRun`, partagée par le nom sous le pion et l'étiquette de la flèche hors cadre) et
 * les deux anneaux autour du pion (`drawTitlePulse`).
 *
 * L'ICÔNE EST UN MASQUE. `nemesis-black.png` et `victim-black.png` sont du noir sur transparent
 * (alpha à 1 bit) : on ne garde que l'alpha, réduit à la taille d'écran par paliers (un PNG de 1369 px
 * ramené d'un coup à 12 px crénèle), puis teinté à la couleur du joueur (le camp, ou sa couleur
 * distincte) passée par l'appelant, et cerné de l'encre du contour des noms : même technique que les
 * vignettes d'arme (silhouette reposée tout autour du corps), même épaisseur que les lettres. La
 * toile rendue est mise en cache par (rôle, couleur, contour, densité) ; la taille d'écran est fixe.
 *
 * TOUT S'ADRESSE À L'ŒIL : tailles en pixels d'ÉCRAN, multipliées par la densité `k` du canevas.
 * Aucune couleur écrite ici.
 */
import type { AntagonistRole } from '@/lib/antagonists/antagonists'

import type { SlotTitle, SlotTitlesOf } from '../model/antagonistTitles'
import type { XY } from '../../../lib/replay/replayLogic'

/** Les deux images, du noir sur transparent : le masque que l'on teint. */
export const TITLE_ICON_URL: Readonly<Record<AntagonistRole, string>> = {
  nemesis: '/icons/nemesis-black.png',
  victim: '/icons/victim-black.png',
}

/** Hauteur de l'icône, en pixels d'écran. La largeur suit le rapport de l'image. */
export const TITLE_ICON_HEIGHT_PX = 12

/** Dimensions naturelles des deux PNG : elles fixent la largeur de l'icône avant son chargement. */
const NATURAL_SIZE: Readonly<Record<AntagonistRole, { w: number; h: number }>> = {
  nemesis: { w: 714, h: 670 },
  victim: { w: 1099, h: 1369 },
}

/** Écart entre l'icône et son compte, et entre un titre et ce qui le suit, en pixels d'écran. */
export const TITLE_ICON_GAP_PX = 2
export const TITLE_RUN_GAP_PX = 3

/** Rayon du contour de l'icône : la moitié de l'épaisseur du contour des lettres (le trait est centré sur le bord). */
export const TITLE_OUTLINE_PX = 1.3
/** Nombre de silhouettes posées en cercle pour obtenir ce contour. */
const OUTLINE_STEPS = 12

/** Une pulsation grossit l'icône de cette fraction de sa taille, au plus fort. */
const PULSE_GROW = 0.8
/** Les anneaux : écart de départ du second, course, trait et opacité de départ. */
const RING_DELAY = 0.25
const RING_TRAVEL_PX = 14
const RING_WIDTH_PX = 1.6
const RING_ALPHA = 0.9
/** Au-delà de cette taille, le cache est vidé : il ne garde que les encres de la palette courante. */
const CACHE_MAX = 48

/** La largeur de l'icône d'un rôle en pixels d'écran, hors contour : la place qu'elle réserve dans l'étiquette. */
export function titleInnerWidthPx(role: AntagonistRole): number {
  const n = NATURAL_SIZE[role]
  return (TITLE_ICON_HEIGHT_PX * n.w) / n.h
}

/** Une icône prête à poser : la toile (corps et contour) et sa taille en pixels d'écran. */
export interface TitleIcon {
  image: CanvasImageSource
  wPx: number
  hPx: number
}

export interface TitleIcons {
  /** Pose l'image chargée d'un rôle ; les icônes déjà bâties de ce rôle sont écartées. */
  setSource(role: AntagonistRole, image: HTMLImageElement): void
  /** L'icône du rôle à cette teinte, ce contour et cette densité ; `null` tant que l'image n'est pas chargée. */
  iconOf(role: AntagonistRole, ink: string, outline: string, k: number): TitleIcon | null
}

function makeCanvas(w: number, h: number): { canvas: HTMLCanvasElement; ctx: CanvasRenderingContext2D } | null {
  const canvas = document.createElement('canvas')
  canvas.width = w
  canvas.height = h
  const ctx = canvas.getContext('2d')
  return ctx ? { canvas, ctx } : null
}

/** shrink : l'alpha de l'image à `w` par `h`, par moitiés successives. */
function shrink(src: HTMLImageElement, w: number, h: number): HTMLCanvasElement | null {
  let from: CanvasImageSource = src
  let cw = src.naturalWidth || src.width
  let ch = src.naturalHeight || src.height
  while (cw >= 2 * w && ch >= 2 * h) {
    cw = Math.ceil(cw / 2)
    ch = Math.ceil(ch / 2)
    const step = makeCanvas(cw, ch)
    if (!step) return null
    step.ctx.imageSmoothingEnabled = true
    step.ctx.imageSmoothingQuality = 'high'
    step.ctx.drawImage(from, 0, 0, cw, ch)
    from = step.canvas
  }
  const out = makeCanvas(w, h)
  if (!out) return null
  out.ctx.imageSmoothingEnabled = true
  out.ctx.imageSmoothingQuality = 'high'
  out.ctx.drawImage(from, 0, 0, w, h)
  return out.canvas
}

/** tint : le masque rempli de `color`, son alpha conservé (`source-in`). */
function tint(mask: HTMLCanvasElement, color: string): HTMLCanvasElement | null {
  const out = makeCanvas(mask.width, mask.height)
  if (!out) return null
  out.ctx.drawImage(mask, 0, 0)
  out.ctx.globalCompositeOperation = 'source-in'
  out.ctx.fillStyle = color
  out.ctx.fillRect(0, 0, mask.width, mask.height)
  return out.canvas
}

function buildIcon(src: HTMLImageElement, role: AntagonistRole, ink: string, outline: string, k: number): TitleIcon | null {
  const bw = Math.max(1, Math.round(titleInnerWidthPx(role) * k))
  const bh = Math.max(1, Math.round(TITLE_ICON_HEIGHT_PX * k))
  const pad = Math.ceil(TITLE_OUTLINE_PX * k)
  const mask = shrink(src, bw, bh)
  const body = mask && tint(mask, ink)
  const edge = mask && outline ? tint(mask, outline) : null
  const out = makeCanvas(bw + 2 * pad, bh + 2 * pad)
  if (!body || !out || (outline && !edge)) return null
  if (edge) {
    const r = TITLE_OUTLINE_PX * k
    for (let i = 0; i < OUTLINE_STEPS; i++) {
      const a = (2 * Math.PI * i) / OUTLINE_STEPS
      out.ctx.drawImage(edge, pad + Math.cos(a) * r, pad + Math.sin(a) * r)
    }
  }
  out.ctx.drawImage(body, pad, pad)
  return { image: out.canvas, wPx: out.canvas.width / k, hPx: out.canvas.height / k }
}

export function createTitleIcons(): TitleIcons {
  const sources = new Map<AntagonistRole, HTMLImageElement>()
  const cache = new Map<string, TitleIcon>()
  return {
    setSource(role, image) {
      sources.set(role, image)
      for (const key of [...cache.keys()]) if (key.startsWith(`${role}|`)) cache.delete(key)
    },
    iconOf(role, ink, outline, k) {
      const src = sources.get(role)
      if (!src) return null
      const key = `${role}|${ink}|${outline}|${k}`
      const known = cache.get(key)
      if (known) return known
      const built = buildIcon(src, role, ink, outline, k)
      if (!built) return null
      if (cache.size >= CACHE_MAX) cache.clear()
      cache.set(key, built)
      return built
    },
  }
}

/** Ce que le calque des pions reçoit : les titres par slot et par image, et les icônes pour les dessiner. */
export interface AntagonistMarks {
  titlesOfSlot: SlotTitlesOf
  icons: TitleIcons
}

/**
 * Un titre prêt à être écrit devant un nom : l'icône (ou `null` tant qu'elle n'est pas chargée : sa
 * place est réservée, le nom ne bouge pas à son arrivée), son compte, son opacité et son échelle.
 */
export interface LabelTitle {
  icon: TitleIcon | null
  /** Largeur de l'icône hors contour, en pixels d'écran : la place qu'elle prend dans la ligne. */
  innerWPx: number
  count: number
  /** 1 = plein ; baisse pour l'ancien tenant qui s'efface. */
  alpha: number
  /** 1 = au repos ; plus grand pendant la pulsation. */
  scale: number
}

/** Les titres d'un pion à une image, prêts à dessiner, et où en est la pulsation de ses anneaux. */
export interface TitleDraw {
  titles: readonly LabelTitle[]
  /** Progression des anneaux (0 à 1) ; `null` = pas de prise de titre en cours sur ce pion. */
  pulse: number | null
  /** La couleur du joueur : celle des anneaux. */
  color: string
  k: number
}

/** pulseScale : 1 au repos, grossit puis revient exactement à 1 au bout de la pulsation. */
export function pulseScale(progress: number): number {
  if (progress <= 0 || progress >= 1) return 1
  return 1 + PULSE_GROW * Math.sin(Math.PI * progress)
}

/** titleFade : l'opacité de l'ancien tenant, de 1 à 0 pendant la pulsation de son successeur. */
export function titleFade(progress: number): number {
  return Math.min(1, Math.max(0, 1 - progress))
}

/** Ce que `titleDrawOf` lit du style du calque des pions (structurel : `MarkerStyle` le satisfait). */
export interface TitleStyleSource {
  k: number
  labelStroke: string
  frame: number
  antagonists?: AntagonistMarks
}

function labelTitleOf(title: SlotTitle, marks: AntagonistMarks, color: string, style: TitleStyleSource): LabelTitle {
  const motion = title.motion
  return {
    icon: marks.icons.iconOf(title.role, color, style.labelStroke, style.k),
    innerWPx: titleInnerWidthPx(title.role),
    count: title.count,
    alpha: motion?.kind === 'loss' ? titleFade(motion.progress) : 1,
    scale: motion?.kind === 'gain' ? pulseScale(motion.progress) : 1,
  }
}

/**
 * titleDrawOf : les titres que porte le slot à l'image du style, prêts à dessiner ; `undefined` quand
 * il n'en porte aucun (ou que le calque ne reçoit pas de titres) : le pion se dessine alors
 * exactement comme avant.
 */
export function titleDrawOf(style: TitleStyleSource, slot: number, color: string): TitleDraw | undefined {
  const marks = style.antagonists
  const titles = marks?.titlesOfSlot(slot, style.frame)
  if (!marks || !titles || titles.length === 0) return undefined
  let pulse: number | null = null
  for (const t of titles) if (t.motion?.kind === 'gain') pulse = t.motion.progress
  return { titles: titles.map((t) => labelTitleOf(t, marks, color, style)), pulse, color, k: style.k }
}

/** Le style d'une étiquette : celui du calque, avec les titres à écrire devant le texte quand il y en a. */
export function labelStyleOf(
  style: { k: number; labelStroke: string },
  draw: TitleDraw | undefined,
): { k: number; labelStroke: string; titles?: readonly LabelTitle[] } {
  return draw ? { k: style.k, labelStroke: style.labelStroke, titles: draw.titles } : style
}

/**
 * drawTitlePulse : les deux anneaux d'une prise de titre, à la couleur du joueur, qui partent du bord
 * du pion (`edge`, en pixels de canevas) et s'éteignent en s'élargissant ; le second part un peu après.
 */
export function drawTitlePulse(ctx: CanvasRenderingContext2D, c: XY, draw: TitleDraw | undefined, edge: number): void {
  if (!draw || draw.pulse === null) return
  for (let ring = 0; ring < 2; ring++) {
    const p = (draw.pulse - ring * RING_DELAY) / (1 - RING_DELAY)
    if (p <= 0 || p >= 1) continue
    ctx.globalAlpha = RING_ALPHA * (1 - p)
    ctx.lineWidth = RING_WIDTH_PX * draw.k * (1 - 0.5 * p)
    ctx.strokeStyle = draw.color
    ctx.beginPath()
    ctx.arc(c.x, c.y, edge + RING_TRAVEL_PX * draw.k * (1 - (1 - p) * (1 - p)), 0, Math.PI * 2)
    ctx.stroke()
  }
  ctx.globalAlpha = 1
}

/** Le style d'une suite de titres : la densité et l'encre du contour des lettres. */
export interface TitleRunStyle {
  k: number
  labelStroke: string
}

const countWidth = (ctx: CanvasRenderingContext2D, t: LabelTitle): number => ctx.measureText(String(t.count)).width

/**
 * titleRunWidth : la largeur de la suite « icône, compte » de tous les titres, écarts compris, en pixels de canevas. La police
 * du texte doit être posée sur `ctx` : le compte se mesure avec elle.
 */
export function titleRunWidth(ctx: CanvasRenderingContext2D, titles: readonly LabelTitle[], k: number): number {
  let width = 0
  for (const t of titles) width += (t.innerWPx + TITLE_ICON_GAP_PX + TITLE_RUN_GAP_PX) * k + countWidth(ctx, t)
  return width
}

/**
 * drawTitleRun écrit, à partir de `at.x` et centrée verticalement sur `at.y`, la suite « icône, compte »
 * de chaque titre : l'icône (grossie par la pulsation autour de son centre, sans pousser ce qui suit),
 * puis son compte, à la couleur du joueur et cerné comme le nom. Rend l'abscisse où le texte suivant commence.
 */
export function drawTitleRun(
  ctx: CanvasRenderingContext2D,
  titles: readonly LabelTitle[],
  at: XY,
  style: TitleRunStyle,
  color: string,
): number {
  const k = style.k
  const yMid = at.y
  let x = at.x
  ctx.textAlign = 'left'
  ctx.textBaseline = 'middle'
  ctx.lineJoin = 'round'
  for (const t of titles) {
    ctx.globalAlpha = t.alpha
    if (t.icon) {
      const w = t.icon.wPx * t.scale * k
      const h = t.icon.hPx * t.scale * k
      ctx.drawImage(t.icon.image, x + (t.innerWPx * k) / 2 - w / 2, yMid - h / 2, w, h)
    }
    x += (t.innerWPx + TITLE_ICON_GAP_PX) * k
    const text = String(t.count)
    if (style.labelStroke) {
      ctx.lineWidth = TITLE_OUTLINE_PX * 2 * k
      ctx.strokeStyle = style.labelStroke
      ctx.strokeText(text, x, yMid)
    }
    ctx.fillStyle = color
    ctx.fillText(text, x, yMid)
    x += countWidth(ctx, t) + TITLE_RUN_GAP_PX * k
  }
  ctx.globalAlpha = 1
  return x
}
