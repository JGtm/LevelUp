/**
 * shotEffects.ts — LA FORME D'UN TIR DIT SON ARME.
 *
 * CE QUI EST MESURÉ, ET CE QUI EST CHOISI. Ce qui est mesuré, c'est le LIBELLÉ de l'arme : le
 * film le porte sur chaque événement de tir, et l'artefact le nomme (22 familles nommées sur
 * 22 pour ce match). Ce qui est choisi, c'est la mise en scène — une famille d'arme se dessine
 * autrement qu'une autre. Aucune famille n'est déduite d'une observation : une arme hors
 * catalogue tombe sur le rendu neutre, jamais sur un rendu approchant.
 *
 * OÙ VIT LE CLASSEMENT : dans les mappings du titre, publiés par l'artefact
 * (`weaponLabels[id].fx`). Ce fichier ne connaît plus une seule arme Halo.
 *
 * POURQUOI LA FORME ET NON LA TEINTE. Le POC distinguait les familles par sept teintes `hsla`
 * en dur. Ici la couleur appartient au TIREUR — c'est elle qui permet de suivre un joueur des
 * yeux — et les couleurs sémantiques passent par des tokens (règle du dépôt). Faire porter la
 * famille par la teinte obligerait soit à écraser l'identité du tireur, soit à inventer sept
 * couleurs hors du système. La forme, elle, est libre : une gerbe d'aiguilles ne ressemble à
 * rien d'autre, et elle reste lisible dans les deux thèmes sans qu'on ait à la décliner.
 *
 * ACCESSIBILITÉ : sous « mouvement réduit », les effets ne s'animent pas. La géométrie et
 * l'opacité restent constantes pendant toute la rémanence — le marqueur reste, il ne bouge
 * plus. La feuille de style ne peut rien pour un canvas dessiné en JS : la préférence se lit
 * donc ici.
 */

/** Familles de rendu. `sobre` n'est pas une famille : c'est l'absence de famille connue. */
export type ShotFamily =
  | 'ballistic'
  | 'plasma'
  | 'light'
  | 'shock'
  | 'explosive'
  | 'melee'
  | 'needles'
  | 'plain'

/**
 * DRAWN_FAMILIES — les familles que ce fichier sait DESSINER.
 *
 * LE CATALOGUE DES ARMES N'EST PLUS ICI (lot 3.2). Il vivait en dur, par nom canonique
 * d'arme — 22 noms Halo dans du code de rendu, donc un catalogue de jeu recopié côté web,
 * qui se serait tu le jour où une arme est renommée et qui n'aurait jamais rien su d'un
 * second titre. Le classement vient désormais du DOCUMENT (`weaponLabels[id].fx`), résolu
 * hors ligne depuis `config/titles/{slug}/mappings/replay_labels.toml`.
 *
 * Ce qui reste ici est ce qui APPARTIENT au rendu : la géométrie de chaque famille, et le
 * refus de dessiner ce qu'on ne connaît pas.
 */
const DRAWN_FAMILIES: Record<string, ShotFamily> = {
  ballistic: 'ballistic',
  plasma: 'plasma',
  light: 'light',
  shock: 'shock',
  explosive: 'explosive',
  melee: 'melee',
  needles: 'needles',
}

/**
 * familyOf valide la famille annoncée par le document.
 *
 * SANS FAMILLE, OU HORS DES FORMES CONNUES : `plain`. On ne rapproche pas d'une famille
 * voisine — un rendu emprunté affirmerait une arme qu'on ignore, exactement comme le ferait
 * un visuel par défaut. Une famille inconnue du client (document plus récent que l'app)
 * tombe donc sur le trait neutre, pas sur une forme approchante.
 */
export function familyOf(effect: string | undefined): ShotFamily {
  if (!effect) return 'plain'
  return DRAWN_FAMILIES[effect] ?? 'plain'
}

/** Géométrie d'un effet : origine, direction (radians canvas) et longueur en pixels. */
export interface ShotShape {
  x: number
  y: number
  /** null = la visée n'était pas lisible : on ne dessine aucune direction. */
  angle: number | null
  length: number
  /** 1 = à l'instant du tir, 0 = fin de rémanence. */
  fade: number
  /** Sous « mouvement réduit », la forme ne progresse pas avec `fade`. */
  reduced: boolean
  /** Départage les formes irrégulières pour que deux tirs voisins ne se superposent pas. */
  seed: number
  /**
   * true SEULEMENT quand l'extrémité est une position RÉELLE — celle de la victime d'une
   * mort (règle POC : couple tueur-victime complet). Sur un TIR, `length` n'est qu'une
   * longueur de trace : y poser un halo d'impact inventerait un point de chute que le film
   * ne donne pas. Les familles `explosive`, `needles` et `melee` s'en servent pour ne pas
   * mentir sur leur extrémité.
   */
  target?: boolean
  /**
   * Mêlée UNIQUEMENT : la victime est à PORTÉE du geste (distance monde < 8 m, seuil
   * mesuré côté killFx). L'arc de liaison tueur -> victime ne se dessine qu'alors — au
   * delà, le geste garde son arc ouvert vers la victime, sans trait qui la relie.
   */
  meleeLink?: boolean
}

/**
 * drawShotEffect dessine un tir selon sa famille, dans la couleur DÉJÀ RÉSOLUE du tireur.
 *
 * La direction n'est tracée que si le film la porte : sans elle, seul l'éclat d'origine est
 * dessiné. Tracer un trait par défaut ferait croire à une visée connue.
 */
export function drawShotEffect(
  ctx: CanvasRenderingContext2D,
  family: ShotFamily,
  shape: ShotShape,
  color: string,
): void {
  ctx.save()
  ctx.strokeStyle = color
  ctx.fillStyle = color
  ctx.lineCap = 'round'
  ctx.lineJoin = 'round'
  const advance = shape.reduced ? 0.5 : 1 - shape.fade
  if (shape.angle === null) {
    drawOrigin(ctx, shape, shape.fade)
  } else {
    const s = { ...shape, angle: shape.angle, advance }
    switch (family) {
      case 'ballistic':
        drawBallistic(ctx, s)
        break
      case 'plasma':
        drawPlasma(ctx, s)
        break
      case 'light':
        drawLight(ctx, s)
        break
      case 'shock':
        drawShock(ctx, s)
        break
      case 'explosive':
        drawExplosive(ctx, s)
        break
      case 'melee':
        drawMelee(ctx, s)
        break
      case 'needles':
        drawNeedles(ctx, s)
        break
      default:
        drawPlain(ctx, s)
    }
  }
  ctx.restore()
}

/** Forme orientée : la géométrie plus son avancement, une fois la direction connue. */
interface Oriented extends Omit<ShotShape, 'angle'> {
  angle: number
  advance: number
}

/**
 * drawDeathMarker — la mort dont AUCUN axe n'est lisible (couple incomplet, règle 89/93) :
 * un anneau POINTILLÉ, daté et localisé, qui n'affirme aucune direction. Convention déjà
 * établie par le POC — le pointillé est la marque du « lu mais non orientable ».
 */
export function drawDeathMarker(
  ctx: CanvasRenderingContext2D,
  shape: ShotShape,
  color: string,
): void {
  ctx.save()
  ctx.strokeStyle = color
  ctx.fillStyle = color
  ctx.globalAlpha = 0.75 * shape.fade
  ctx.beginPath()
  ctx.arc(shape.x, shape.y, 5.5, 0, Math.PI * 2)
  ctx.lineWidth = 1.3
  ctx.setLineDash([2, 2])
  ctx.stroke()
  ctx.setLineDash([])
  drawOrigin(ctx, shape, shape.fade)
  ctx.restore()
}

/** drawOrigin — l'éclat d'un tir dont la visée n'est pas lisible. Aucune direction inventée. */
function drawOrigin(ctx: CanvasRenderingContext2D, shape: ShotShape, fade: number): void {
  ctx.globalAlpha = 0.85 * fade
  ctx.beginPath()
  ctx.arc(shape.x, shape.y, 2.5, 0, Math.PI * 2)
  ctx.fill()
}

function endOf(s: Oriented): { x: number; y: number } {
  return { x: s.x + Math.cos(s.angle) * s.length, y: s.y + Math.sin(s.angle) * s.length }
}

/**
 * drawBallistic — une poudre : trait net et bref, qui s'éteint sec.
 *
 * DEUX HORLOGES, recalées sur le POC : l'ÉCLAT s'éteint vite (carré du temps restant) —
 * c'est la détonation sèche demandée ; le TRAIT décroît plus lentement (puissance 1,5)
 * parce qu'il porte QUI a tiré sur QUI, et une information de ce prix ne doit pas
 * disparaître en deux images. Le POC avait mesuré le défaut inverse : tout au cube, et
 * l'effet était absent des vitesses 4x et 8x.
 */
function drawBallistic(ctx: CanvasRenderingContext2D, s: Oriented): void {
  const e = endOf(s)
  const k = s.fade * s.fade
  const kt = Math.pow(Math.max(0, s.fade), 1.5)
  ctx.globalAlpha = 0.9 * kt
  ctx.lineWidth = 2.2
  ctx.beginPath()
  ctx.moveTo(s.x, s.y)
  ctx.lineTo(e.x, e.y)
  ctx.stroke()
  // L'éclat de bouche : il grandit en s'éteignant (POC), au carré — une détonation.
  ctx.globalAlpha = Math.min(1, 1.1 * k)
  ctx.beginPath()
  ctx.arc(s.x, s.y, 2 + 4.2 * k, 0, Math.PI * 2)
  ctx.fill()
}

/**
 * drawPlasma — l'énergie ne s'éteint pas comme une poudre : décroissance MOLLE (racine
 * carrée, recalée POC — c'est le contraste voulu avec la balistique : même geste,
 * extinction opposée), et le trait ONDULE.
 */
function drawPlasma(ctx: CanvasRenderingContext2D, s: Oriented): void {
  const nx = -Math.sin(s.angle)
  const ny = Math.cos(s.angle)
  const phase = (s.seed % 13) * 0.48
  const k = Math.sqrt(Math.max(0, s.fade))
  ctx.globalAlpha = 0.8 * k
  ctx.lineWidth = 2.3
  ctx.beginPath()
  const steps = 14
  for (let i = 0; i <= steps; i++) {
    const u = i / steps
    const off = Math.sin(u * 7.5 + phase + s.advance * 2.2) * 3.6 * (1 - u) * k
    const px = s.x + Math.cos(s.angle) * s.length * u + nx * off
    const py = s.y + Math.sin(s.angle) * s.length * u + ny * off
    if (i === 0) ctx.moveTo(px, py)
    else ctx.lineTo(px, py)
  }
  ctx.stroke()
}

/**
 * drawLight — un RAI CONTINU : deux traits superposés, large et pâle sous fin et franc.
 * L'épaisseur ne varie pas avec l'âge, seule l'opacité baisse : un faisceau ne se
 * disperse pas, il s'éteint (largeurs recalées POC : 6,5 / 2,0).
 */
function drawLight(ctx: CanvasRenderingContext2D, s: Oriented): void {
  const e = endOf(s)
  ctx.beginPath()
  ctx.moveTo(s.x, s.y)
  ctx.lineTo(e.x, e.y)
  ctx.globalAlpha = 0.28 * s.fade
  ctx.lineWidth = 6.5
  ctx.stroke()
  ctx.globalAlpha = 0.95 * s.fade
  ctx.lineWidth = 2
  ctx.stroke()
}

/**
 * drawShock — un arc BRISÉ : une ligne en zigzag, jamais droite. Décroissance recalée
 * POC (puissance 1,6) : plus vif qu'un plasma, moins sec qu'une poudre.
 */
function drawShock(ctx: CanvasRenderingContext2D, s: Oriented): void {
  const nx = -Math.sin(s.angle)
  const ny = Math.cos(s.angle)
  const segments = 6
  const k = Math.pow(Math.max(0, s.fade), 1.6)
  ctx.globalAlpha = Math.min(1, 1.1 * k)
  ctx.lineWidth = 1.7
  ctx.beginPath()
  for (let i = 0; i <= segments; i++) {
    const u = i / segments
    const amp = i === segments ? 0 : (2 + (s.seed % 5)) * (i % 2 ? 1 : -1)
    const px = s.x + Math.cos(s.angle) * s.length * u + nx * amp
    const py = s.y + Math.sin(s.angle) * s.length * u + ny * amp
    if (i === 0) ctx.moveTo(px, py)
    else ctx.lineTo(px, py)
  }
  ctx.stroke()
}

/**
 * drawExplosive — DEUX TEMPS : un départ épais et bref, puis une onde qui s'ouvre.
 * L'onde ne se pose à l'EXTRÉMITÉ que si elle est RÉELLE (`target`, la victime d'une
 * mort) ; sur un tir elle reste centrée sur le tireur — le film ne date aucun impact.
 */
function drawExplosive(ctx: CanvasRenderingContext2D, s: Oriented): void {
  const burst = Math.max(0, (s.fade - 0.7) / 0.3)
  if (burst > 0) {
    const dl = Math.min(s.length, 22)
    const e = { x: s.x + Math.cos(s.angle) * dl, y: s.y + Math.sin(s.angle) * dl }
    ctx.globalAlpha = 0.9 * burst
    ctx.lineWidth = 3.4
    ctx.beginPath()
    ctx.moveTo(s.x, s.y)
    ctx.lineTo(e.x, e.y)
    ctx.stroke()
  }
  const cx = s.target ? s.x + Math.cos(s.angle) * s.length : s.x
  const cy = s.target ? s.y + Math.sin(s.angle) * s.length : s.y
  ctx.globalAlpha = 0.5 * s.fade
  ctx.lineWidth = 1.4
  ctx.beginPath()
  ctx.arc(cx, cy, 5 + 24 * s.advance, 0, Math.PI * 2)
  ctx.stroke()
}

/**
 * drawMelee — AUCUN éclair de bouche : le geste n'est pas un tir, c'est un arc court,
 * ouvert du côté de la victime. L'ARC DE LIAISON entre les deux ne se dessine que si la
 * victime est réellement à portée (`meleeLink`, seuil 8 m mesuré côté killFx) : relier un
 * marteau à une victime à 20 m affirmerait un contact qui n'a pas eu lieu.
 */
function drawMelee(ctx: CanvasRenderingContext2D, s: Oriented): void {
  const r = 7 + 13 * s.advance
  ctx.globalAlpha = 0.9 * s.fade
  ctx.lineWidth = 2.4
  ctx.beginPath()
  ctx.arc(s.x, s.y, r, s.angle - 0.9, s.angle + 0.9)
  ctx.stroke()
  ctx.globalAlpha = 0.45 * s.fade
  ctx.lineWidth = 1.4
  ctx.beginPath()
  ctx.arc(s.x, s.y, r * 0.6, s.angle - 1.25, s.angle + 1.25)
  ctx.stroke()
  if (s.target && s.meleeLink) {
    const ex = s.x + Math.cos(s.angle) * s.length
    const ey = s.y + Math.sin(s.angle) * s.length
    const nx = -Math.sin(s.angle)
    const ny = Math.cos(s.angle)
    const b = s.length * 0.3
    ctx.globalAlpha = 0.75 * s.fade
    ctx.lineWidth = 1.9
    ctx.beginPath()
    ctx.moveTo(s.x, s.y)
    ctx.quadraticCurveTo((s.x + ex) / 2 + nx * b, (s.y + ey) / 2 + ny * b, ex, ey)
    ctx.stroke()
  }
}

/**
 * drawNeedles — la signature est la GERBE, pas le trait : cinq brins qui s'écartent.
 * La petite détonation à l'extrémité ne se joue QUE si elle est réelle (`target`, une
 * mort) : un tir du Needler ne dit pas où ses aiguilles ont fini.
 */
function drawNeedles(ctx: CanvasRenderingContext2D, s: Oriented): void {
  const nx = -Math.sin(s.angle)
  const ny = Math.cos(s.angle)
  ctx.globalAlpha = 0.85 * s.fade
  ctx.lineWidth = 1.1
  for (let i = 0; i < 5; i++) {
    const spread = (i - 2) / 2
    const reach = s.length * (0.6 + 0.1 * i)
    const ex = s.x + Math.cos(s.angle) * reach + nx * spread * 9
    const ey = s.y + Math.sin(s.angle) * reach + ny * spread * 9
    ctx.beginPath()
    ctx.moveTo(s.x, s.y)
    ctx.lineTo(ex, ey)
    ctx.stroke()
  }
  if (s.target && s.advance > 0.55) {
    const ex = s.x + Math.cos(s.angle) * s.length
    const ey = s.y + Math.sin(s.angle) * s.length
    ctx.globalAlpha = 0.65 * s.fade
    ctx.beginPath()
    ctx.arc(ex, ey, 4 + 15 * ((s.advance - 0.55) / 0.45), 0, Math.PI * 2)
    ctx.lineWidth = 1.2
    ctx.stroke()
  }
}

/** drawPlain — arme non cataloguée : un trait sobre, qui n'affirme aucune famille. */
function drawPlain(ctx: CanvasRenderingContext2D, s: Oriented): void {
  const e = endOf(s)
  ctx.globalAlpha = 0.7 * s.fade
  ctx.lineWidth = 1
  ctx.beginPath()
  ctx.moveTo(s.x, s.y)
  ctx.lineTo(e.x, e.y)
  ctx.stroke()
  drawOrigin(ctx, s, s.fade)
}
