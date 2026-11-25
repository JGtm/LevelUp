/**
 * chatterBubbles.ts — LA BULLE D'UNE RÉPLIQUE sur la carte : combien de temps elle vit, quelle réplique porte chaque joueur à
 * cet instant, et où poser les bulles pour qu'elles ne se recouvrent pas. Logique pure, en pixels de toile ; le dessin vit
 * dans `layers/chatterLayer.ts`.
 *
 * LE TEMPS EST CELUI DU FILM (maquette validée) : 2,6 s de lecture, fondu d'entrée de 0,15 s, fondu de sortie de 0,5 s. À 0,5×
 * la bulle reste donc plus longtemps à l'écran, à 4× moins ; ce qu'on lit, c'est ce qui s'est dit à ce moment du match, et le fil
 * garde la phrase entière. Le mouvement réduit supprime les fondus : la bulle est pleine pendant toute sa vie.
 *
 * UNE BULLE PAR JOUEUR : si un joueur parle de nouveau avant la fin de la première, la plus récente remplace l'ancienne.
 */
import type { ChatterLine } from './chatterLines'

/** La vie d'une bulle, et ses deux fondus (en ms de film). */
export const CHATTER_LIFE_MS = 2600
export const CHATTER_FADE_IN_MS = 150
export const CHATTER_FADE_OUT_MS = 500

/** L'opacité d'une bulle `ageMs` après la parole ; 0 avant et après sa vie. */
export function chatterAlpha(ageMs: number, reducedMotion: boolean): number {
  if (ageMs < 0 || ageMs > CHATTER_LIFE_MS) return 0
  if (reducedMotion) return 1
  if (ageMs < CHATTER_FADE_IN_MS) return ageMs / CHATTER_FADE_IN_MS
  if (ageMs > CHATTER_LIFE_MS - CHATTER_FADE_OUT_MS) return Math.max(0, (CHATTER_LIFE_MS - ageMs) / CHATTER_FADE_OUT_MS)
  return 1
}

/** Une bulle vivante : la réplique, son âge, son opacité. */
export interface LiveChatter {
  line: ChatterLine
  ageMs: number
  alpha: number
}

/**
 * Les bulles vivantes à `frame` : pour chaque joueur, sa réplique la plus récente encore dans sa vie. Une réplique à venir
 * (postérieure à l'image) ne vit pas. `lines` est trié par image, `frameMs` la durée d'une image.
 */
export function liveChatter(
  lines: readonly ChatterLine[],
  frame: number,
  frameMs: number,
  reducedMotion: boolean,
): LiveChatter[] {
  const latest = new Map<string, LiveChatter>()
  for (const line of lines) {
    if (line.frame > frame) break
    const ageMs = (frame - line.frame) * frameMs
    const alpha = chatterAlpha(ageMs, reducedMotion)
    if (alpha <= 0) continue
    latest.set(line.xuid, { line, ageMs, alpha })
  }
  return [...latest.values()]
}

/** Un rectangle de toile. */
export interface Rect {
  x: number
  y: number
  w: number
  h: number
}

/** Ce que le placement lit d'une bulle : sa taille et le point de la carte qu'elle désigne. */
export interface BubbleSpec {
  w: number
  h: number
  anchorX: number
  anchorY: number
}

/** L'espace entre le pion et la bulle, et le jeu minimal entre deux bulles. */
const BUBBLE_GAP = 24
const BUBBLE_MARGIN = 2
const OVERLAP_BUBBLE_WEIGHT = 10
const OVERLAP_MARKER_WEIGHT = 2

/** La surface de recouvrement de `a` et `b`, chacun élargi de `gap`. */
function overlap(a: Rect, b: Rect, gap: number): number {
  const dx = Math.max(0, Math.min(a.x + a.w + gap, b.x + b.w) - Math.max(a.x - gap, b.x))
  const dy = Math.max(0, Math.min(a.y + a.h + gap, b.y + b.h) - Math.max(a.y - gap, b.y))
  return dx * dy
}

/** Les positions essayées autour du pion, de la plus naturelle (au-dessus) à la plus lointaine. */
function candidates(w: number, h: number, g: number): ReadonlyArray<readonly [number, number]> {
  return [
    [-w / 2, -g - h], [-w / 2, g], [g, -h / 2], [-w - g, -h / 2],
    [g - 6, -g - h], [-w - g + 6, -g - h], [g - 6, g], [-w - g + 6, g],
    [-w / 2, -g - 2 * h - 6], [-w / 2, g + h + 6], [g + 30, -h / 2], [-w - g - 30, -h / 2],
  ]
}

/**
 * placeBubbles pose chaque bulle près de son pion en évitant les bulles déjà posées et les pions : la première position qui ne
 * recouvre rien, sinon la moins mauvaise. La bulle reste dans la toile. Les bulles sont posées dans l'ordre reçu.
 */
export function placeBubbles(
  specs: readonly BubbleSpec[],
  markers: readonly Rect[],
  canvas: { width: number; height: number },
): Rect[] {
  const placed: Rect[] = []
  for (const s of specs) {
    let best: Rect | null = null
    let bestScore = Infinity
    for (const [dx, dy] of candidates(s.w, s.h, BUBBLE_GAP)) {
      const x = Math.min(Math.max(BUBBLE_MARGIN, s.anchorX + dx), Math.max(BUBBLE_MARGIN, canvas.width - s.w - BUBBLE_MARGIN))
      const y = Math.min(Math.max(BUBBLE_MARGIN, s.anchorY + dy), Math.max(BUBBLE_MARGIN, canvas.height - s.h - BUBBLE_MARGIN))
      const r = { x, y, w: s.w, h: s.h }
      let score = 0
      for (const q of placed) score += overlap(r, q, 3) * OVERLAP_BUBBLE_WEIGHT
      for (const m of markers) score += overlap(r, m, 0) * OVERLAP_MARKER_WEIGHT
      if (score < bestScore) {
        bestScore = score
        best = r
        if (score === 0) break
      }
    }
    placed.push(best ?? { x: BUBBLE_MARGIN, y: BUBBLE_MARGIN, w: s.w, h: s.h })
  }
  return placed
}
