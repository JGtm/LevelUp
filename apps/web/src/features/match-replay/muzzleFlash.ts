/**
 * muzzleFlash.ts — L'ÉCLAIR DE BOUCHE D'UN TIR.
 *
 * PORTÉ, PAS RÉINVENTÉ. La référence est `apps/web/src/components/replay/ReplayCanvas.tsx`
 * du projet Csstat : « petite flamme dans la direction du REGARD du joueur qui tire »,
 * `MUZZLE_DURATION = 6` frames, mêlée exclue, dessinée EN PLUS du cône de visée. La
 * mécanique en est reprise telle quelle : composition additive (`lighter`), une bouffée
 * étirée dans l'axe (translate + rotate + scale), un halo large sous un cœur incandescent,
 * et une opacité qui tombe avec l'âge.
 *
 * CE QU'IL DIT, ET RIEN D'AUTRE : d'où part le tir, quand, et de quelle NATURE est la
 * décharge. Il ne pointe AUCUNE victime — les événements de tir n'en portent pas (mesuré),
 * et le trait vers une cible serait une invention. Il ne porte pas non plus la couleur du
 * tireur : correction utilisateur du 2026-08-15, « les couleurs des effets de tirs [...]
 * prennent seulement l'ARME en compte ». L'identité du joueur reste au marqueur et au cône.
 *
 * IL NE REMPLACE PAS LE CÔNE DE VISÉE (demande explicite) : le cône reste ce qu'il est, le
 * flash se pose au marqueur, court et vif.
 *
 * LA MÊLÉE N'A PAS D'ÉCLAIR — un coup de marteau n'est pas un tir. La référence l'exclut
 * aussi. Le filtrage se fait en amont (`buildShotFx`), pas ici : un effet qu'on dessine
 * puis qu'on efface est un effet qu'on a payé.
 *
 * Pas de React, pas d'état : une fonction du temps, donc un retour en arrière redonne
 * exactement la même image.
 */
import type { FxInk, FxTint } from './fxInk'
import type { ShotFamily } from './shotEffects'

/** Géométrie d'un éclair, en pixels d'ÉCRAN (le facteur de densité est appliqué par `k`). */
export interface MuzzleShape {
  x: number
  y: number
  /** Radians canevas. null = le regard n'était pas lisible : une bouffée RONDE, sans axe. */
  angle: number | null
  /** 1 = à l'instant du tir, 0 = fin de rémanence. */
  fade: number
  /** Sous « mouvement réduit », l'éclair ne progresse ni ne palpite. */
  reduced: boolean
  /** Départage les formes irrégulières : deux tirs voisins ne se superposent pas. */
  seed: number
  /** Densité du canevas : tout ce qui s'adresse à l'œil est multiplié par ce facteur. */
  k: number
}

/** Rayon de référence de l'éclair, en pixels d'écran (réglé à l'œil, item 2.4). */
const FLASH_R = 9
/** Rayon du cœur incandescent, plus petit et plus vif que le halo. */
const CORE_R = 3.6
/** Étirement dans l'axe de tir : la référence Csstat étire 1,7 en long, 0,45 en large. */
const STRETCH_LONG = 1.7
const STRETCH_WIDE = 0.45
/** L'éclair naît un peu DEVANT le marqueur : au bout du canon, pas dans le torse. */
const MUZZLE_OFFSET = 5

/**
 * drawMuzzleFlash dessine l'éclair d'un tir selon la FAMILLE de son arme (la forme) et la
 * TEINTE de sa décharge (la couleur). Les deux sont orthogonales : deux armes de la même
 * forme n'ont pas la même couleur dans le jeu, et l'inverse est vrai aussi.
 */
export function drawMuzzleFlash(
  ctx: CanvasRenderingContext2D,
  family: ShotFamily,
  tint: FxTint,
  shape: MuzzleShape,
  ink: FxInk,
): void {
  const color = ink.tint[tint] || ink.tint.neutral
  if (!color) return
  ctx.save()
  // ADDITIF : deux éclairs qui se recouvrent s'ajoutent au lieu de se masquer, et un éclair
  // sur une carte sombre brûle au lieu de la teinter. C'est la composition de la référence.
  ctx.globalCompositeOperation = 'lighter'
  ctx.lineCap = 'round'
  ctx.lineJoin = 'round'
  ctx.strokeStyle = color
  ctx.fillStyle = color
  // L'ÉCLAT TOMBE VITE (carré du temps restant) : une détonation ne s'éteint pas
  // linéairement. Sous mouvement réduit, l'intensité ne varie plus du tout.
  const heat = shape.reduced ? 0.6 : shape.fade * shape.fade
  const grow = shape.reduced ? 0.5 : 1 - shape.fade
  if (shape.angle === null) {
    drawPuff(ctx, shape, heat, color, ink.core)
  } else {
    const s = { ...shape, angle: shape.angle, heat, grow, color, core: ink.core }
    switch (family) {
      case 'ballistic':
        drawFlame(ctx, s)
        break
      case 'plasma':
        drawBloom(ctx, s)
        break
      case 'light':
        drawRay(ctx, s)
        break
      case 'shock':
        drawArc(ctx, s)
        break
      case 'explosive':
        drawBlast(ctx, s)
        break
      case 'needles':
        drawSpray(ctx, s)
        break
      default:
        drawPuff(ctx, shape, heat, color, ink.core)
    }
  }
  ctx.restore()
}

/** Un éclair orienté : sa géométrie, sa chaleur, son avancement et ses deux encres. */
interface Oriented extends Omit<MuzzleShape, 'angle'> {
  angle: number
  heat: number
  grow: number
  color: string
  core: string
}

/** muzzle — le point d'où part l'éclair : devant le marqueur, dans l'axe du regard. */
function muzzle(s: Oriented): { x: number; y: number } {
  const d = MUZZLE_OFFSET * s.k
  return { x: s.x + Math.cos(s.angle) * d, y: s.y + Math.sin(s.angle) * d }
}

/**
 * glow — le halo et le cœur de la référence : deux dégradés radiaux concentriques, le
 * second plus petit et plus clair. `long`/`wide` étirent la bouffée dans l'axe.
 */
function glow(
  ctx: CanvasRenderingContext2D,
  at: { x: number; y: number },
  s: Oriented,
  radius: number,
  long: number,
  wide: number,
): void {
  ctx.save()
  ctx.translate(at.x, at.y)
  ctx.rotate(s.angle)
  ctx.scale(long, wide)
  paintRadial(ctx, radius * s.k, s.color, 0.75 * s.heat)
  paintRadial(ctx, CORE_R * s.k * (radius / FLASH_R), s.core, s.heat)
  ctx.restore()
}

/** paintRadial peint un disque dégradé centré sur l'origine du repère courant. */
function paintRadial(
  ctx: CanvasRenderingContext2D,
  radius: number,
  color: string,
  alpha: number,
): void {
  if (!color || radius <= 0) return
  const g = ctx.createRadialGradient(0, 0, 0, 0, 0, radius)
  g.addColorStop(0, color)
  // « transparent » plutôt qu'une couleur à alpha zéro : le canevas accepte n'importe quel
  // format de couleur du thème (oklch compris) sans qu'on ait à le désassembler.
  g.addColorStop(1, 'transparent')
  ctx.globalAlpha = Math.min(1, alpha)
  ctx.fillStyle = g
  ctx.beginPath()
  ctx.arc(0, 0, radius, 0, Math.PI * 2)
  ctx.fill()
}

/**
 * drawFlame — LA POUDRE : la bouffée étirée de la référence, brève et chaude. C'est la
 * forme portée telle quelle de Csstat (halo + cœur, étirement 1,7 / 0,45).
 */
function drawFlame(ctx: CanvasRenderingContext2D, s: Oriented): void {
  glow(ctx, muzzle(s), s, FLASH_R, STRETCH_LONG, STRETCH_WIDE)
}

/**
 * drawBloom — LE PLASMA : l'énergie ne claque pas, elle s'épanouit. La bouffée est plus
 * ronde, elle GRANDIT en s'éteignant, et sa décroissance est molle (racine carrée) — le
 * contraste voulu avec la poudre, déjà établi par les effets de mort.
 */
function drawBloom(ctx: CanvasRenderingContext2D, s: Oriented): void {
  const soft = { ...s, heat: Math.sqrt(Math.max(0, s.fade)) * (s.reduced ? 0.6 : 1) }
  glow(ctx, muzzle(s), soft, FLASH_R * (1 + 0.45 * s.grow), 1.15, 0.9)
}

/**
 * drawRay — LE DUR-LUMIÈRE : un rai CONTINU, deux traits superposés (large et pâle sous
 * fin et franc). L'épaisseur ne varie pas avec l'âge, seule l'opacité baisse : un faisceau
 * ne se disperse pas, il s'éteint.
 */
function drawRay(ctx: CanvasRenderingContext2D, s: Oriented): void {
  const m = muzzle(s)
  const len = 26 * s.k
  const ex = m.x + Math.cos(s.angle) * len
  const ey = m.y + Math.sin(s.angle) * len
  ctx.beginPath()
  ctx.moveTo(m.x, m.y)
  ctx.lineTo(ex, ey)
  ctx.globalAlpha = 0.3 * s.fade
  ctx.lineWidth = 5 * s.k
  ctx.stroke()
  ctx.globalAlpha = 0.95 * s.fade
  ctx.lineWidth = 1.6 * s.k
  ctx.strokeStyle = s.core || s.color
  ctx.stroke()
  ctx.strokeStyle = s.color
  glow(ctx, m, s, FLASH_R * 0.6, 1, 1)
}

/**
 * drawArc — L'ARC ÉLECTRIQUE : une ligne BRISÉE, jamais droite, regénérée d'un germe
 * stable (revenir en arrière redonne la même image, comme la nappe de la Dynamo).
 */
function drawArc(ctx: CanvasRenderingContext2D, s: Oriented): void {
  const m = muzzle(s)
  const nx = -Math.sin(s.angle)
  const ny = Math.cos(s.angle)
  const segments = 4
  const reach = 18 * s.k
  ctx.globalAlpha = Math.min(1, 1.1 * s.heat)
  ctx.lineWidth = 1.6 * s.k
  ctx.beginPath()
  for (let i = 0; i <= segments; i++) {
    const u = i / segments
    const amp = i === segments ? 0 : (1.5 + (s.seed % 4)) * (i % 2 ? 1 : -1) * s.k
    ctx.lineTo(m.x + Math.cos(s.angle) * reach * u + nx * amp, m.y + Math.sin(s.angle) * reach * u + ny * amp)
  }
  ctx.stroke()
  glow(ctx, m, s, FLASH_R * 0.55, 1, 1)
}

/**
 * drawBlast — LA DÉFLAGRATION : une bouffée large au départ, plus une onde qui s'ouvre AU
 * CANON. L'onde ne se pose jamais au bout : le film ne date aucun impact, et l'affirmer
 * inventerait un point de chute.
 */
function drawBlast(ctx: CanvasRenderingContext2D, s: Oriented): void {
  const m = muzzle(s)
  glow(ctx, m, s, FLASH_R * 1.25, 1.3, 0.85)
  ctx.globalAlpha = 0.45 * s.fade
  ctx.lineWidth = 1.4 * s.k
  ctx.beginPath()
  ctx.arc(m.x, m.y, (4 + 16 * s.grow) * s.k, 0, Math.PI * 2)
  ctx.stroke()
}

/**
 * drawSpray — LES AIGUILLES : la signature est la GERBE, pas le trait. Cinq brins courts
 * qui s'écartent du canon, sans rien affirmer de leur point de chute.
 */
function drawSpray(ctx: CanvasRenderingContext2D, s: Oriented): void {
  const m = muzzle(s)
  const nx = -Math.sin(s.angle)
  const ny = Math.cos(s.angle)
  ctx.globalAlpha = 0.9 * s.fade
  ctx.lineWidth = 1.1 * s.k
  for (let i = 0; i < 5; i++) {
    const spread = (i - 2) / 2
    const reach = (10 + 2 * i) * s.k
    ctx.beginPath()
    ctx.moveTo(m.x, m.y)
    ctx.lineTo(
      m.x + Math.cos(s.angle) * reach + nx * spread * 5 * s.k,
      m.y + Math.sin(s.angle) * reach + ny * spread * 5 * s.k,
    )
    ctx.stroke()
  }
  glow(ctx, m, s, FLASH_R * 0.5, 1, 1)
}

/**
 * drawPuff — LE TIR DONT LE REGARD N'EST PAS LISIBLE : une bouffée RONDE, posée sur le
 * marqueur. Aucune direction n'est inventée — c'est la règle qui vaut pour tout le rejeu.
 */
function drawPuff(
  ctx: CanvasRenderingContext2D,
  shape: MuzzleShape,
  heat: number,
  color: string,
  core: string,
): void {
  ctx.save()
  ctx.translate(shape.x, shape.y)
  paintRadial(ctx, FLASH_R * 0.85 * shape.k, color, 0.7 * heat)
  paintRadial(ctx, CORE_R * 0.85 * shape.k, core, heat)
  ctx.restore()
}
