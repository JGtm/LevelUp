/**
 * killFocusMotion.ts — LES GESTES DE LA MISE EN VALEUR D'UNE ÉLIMINATION, en fonction du temps
 * écoulé depuis le clic.
 *
 * Trois gestes, aux durées de la maquette validée : les équerres se VERROUILLENT (340 ms), les
 * étiquettes de nom POP (280 ms), le voile se FOND (240 ms). Aucun ne dépend de l'image du
 * rejeu : la lecture est en pause quand ils jouent, ils suivent donc le temps RÉEL, et c'est le
 * hook du calque qui le mesure. Ici, rien que des fonctions pures du temps écoulé.
 *
 * COMMENT LA MAQUETTE LES JOUE, ET CE QUE CE MODULE REPRODUIT : une courbe d'accélération unique
 * (`cubic-bezier(.2,.8,.2,1)`) transforme la progression AVANT l'interpolation des images-clés.
 * Les « 72 % » du verrouillage sont donc des 72 % de la progression ACCÉLÉRÉE, pas du temps
 * brut : le dépassement (échelle 0,88) arrive tôt et se résorbe lentement.
 *
 * `prefers-reduced-motion` : l'appelant passe un temps écoulé infini, et chaque geste rend sa
 * pose d'arrivée — des équerres posées, sans mouvement.
 */

/** Durée du verrouillage des équerres, en millisecondes. */
export const LOCK_MS = 340
/** Durée de l'apparition d'une étiquette de nom. */
export const POP_MS = 280
/** Durée du fondu du voile. */
export const FADE_MS = 240

/** La durée au bout de laquelle plus rien ne bouge : la boucle d'animation s'arrête. */
export const FOCUS_MOTION_MS = Math.max(LOCK_MS, POP_MS, FADE_MS)

/** L'état d'un élément animé : échelle, rotation (degrés, autour de son centre) et opacité. */
export interface MotionPose {
  scale: number
  rotationDeg: number
  opacity: number
}

/** Une image-clé : sa position dans la progression accélérée et la pose qu'elle impose. */
interface Keyframe extends MotionPose {
  offset: number
}

const LOCK_KEYS: readonly Keyframe[] = [
  { offset: 0, scale: 2.2, rotationDeg: 45, opacity: 0 },
  { offset: 0.72, scale: 0.88, rotationDeg: 0, opacity: 1 },
  { offset: 1, scale: 1, rotationDeg: 0, opacity: 1 },
]

const POP_KEYS: readonly Keyframe[] = [
  { offset: 0, scale: 0.6, rotationDeg: 0, opacity: 0 },
  { offset: 0.7, scale: 1.08, rotationDeg: 0, opacity: 1 },
  { offset: 1, scale: 1, rotationDeg: 0, opacity: 1 },
]

/**
 * cubicBezier rend la fonction d'accélération CSS `cubic-bezier(x1, y1, x2, y2)` : la progression
 * `t` (temps normalisé) vers la progression accélérée. Résolution de x(s) = t par bissection —
 * monotone pour des abscisses de contrôle dans [0, 1] —, puis lecture de y(s).
 */
export function cubicBezier(x1: number, y1: number, x2: number, y2: number): (t: number) => number {
  const axis = (a: number, b: number, s: number) =>
    3 * a * (1 - s) * (1 - s) * s + 3 * b * (1 - s) * s * s + s * s * s
  return (t) => {
    if (t <= 0) return 0
    if (t >= 1) return 1
    let lo = 0
    let hi = 1
    for (let i = 0; i < 40; i++) {
      const mid = (lo + hi) / 2
      if (axis(x1, x2, mid) < t) lo = mid
      else hi = mid
    }
    return axis(y1, y2, (lo + hi) / 2)
  }
}

/** La courbe de la maquette, commune aux trois gestes. */
const EASE = cubicBezier(0.2, 0.8, 0.2, 1)

/** Interpole les images-clés à la progression accélérée `p` (dans [0, 1]). */
function sample(keys: readonly Keyframe[], p: number): MotionPose {
  // Les deux bornes se rendent TELLES QUELLES : pas d'interpolation, donc pas d'écart d'arrondi
  // (ni de zéro négatif) sur la pose de départ ou d'arrivée.
  if (p <= 0) return pose(keys[0])
  if (p >= 1) return pose(keys[keys.length - 1])
  for (let i = 1; i < keys.length; i++) {
    const a = keys[i - 1]
    const b = keys[i]
    if (p > b.offset) continue
    const f = (p - a.offset) / (b.offset - a.offset)
    return {
      scale: a.scale + (b.scale - a.scale) * f,
      rotationDeg: a.rotationDeg + (b.rotationDeg - a.rotationDeg) * f,
      opacity: a.opacity + (b.opacity - a.opacity) * f,
    }
  }
  return pose(keys[keys.length - 1])
}

/** La pose d'une image-clé, sans son offset. */
function pose(k: Keyframe): MotionPose {
  return { scale: k.scale, rotationDeg: k.rotationDeg, opacity: k.opacity }
}

/** La progression accélérée d'un geste de `durationMs`, `elapsedMs` après son départ. */
function progress(elapsedMs: number, durationMs: number): number {
  return EASE(Math.min(1, Math.max(0, elapsedMs / durationMs)))
}

/** lockPose : la pose des équerres `elapsedMs` après le clic (large, tournée, puis posée). */
export function lockPose(elapsedMs: number): MotionPose {
  return sample(LOCK_KEYS, progress(elapsedMs, LOCK_MS))
}

/** popPose : la pose d'une étiquette de nom `elapsedMs` après le clic. */
export function popPose(elapsedMs: number): MotionPose {
  return sample(POP_KEYS, progress(elapsedMs, POP_MS))
}

/** fadeOpacity : l'opacité relative du voile (0 à 1) `elapsedMs` après le clic. */
export function fadeOpacity(elapsedMs: number): number {
  return progress(elapsedMs, FADE_MS)
}
