/**
 * explosionFx.ts — L'EXPLOSION D'UNE GRENADE, en particules.
 *
 * PORTÉE, PAS RÉINVENTÉE. La référence est `apps/web/src/components/replay/ExplosionFx.tsx`
 * du projet Csstat : une timeline de 1,4 s — flash radial très bref, boule de feu en
 * composition additive, onde de choc en anneau, éclats en traînées, poussière résiduelle —
 * et trois familles de particules typées (braises, éclats, bouffées). La mécanique, les
 * temps et les coefficients de traînée en sont repris ; ce qui change tient à deux
 * contraintes de CE canevas, écrites plus bas : le rendu doit être une FONCTION DU TEMPS, et
 * il doit être BORNÉ.
 *
 * CE QUE L'EXPLOSION AFFIRME, ET CE QU'ELLE N'AFFIRME PAS. Le film ne porte AUCUN événement
 * de détonation : ce qui est connu, c'est la DERNIÈRE POSITION RÉPLIQUÉE du vol (pour une
 * frag, la réplication cesse ~1,4 s après le lancer alors que la mèche court jusqu'à ~3 s).
 * L'explosion est donc une MISE EN SCÈNE assumée à cet endroit — et c'est pourquoi l'écran
 * continue de dire « dernière position connue », jamais « impact ».
 *
 * FONCTION DU TEMPS, PAS ANIMATION À ÉTAT. La référence fait avancer ses particules image
 * par image. Ici la lecture se déplace : on avance, on recule, on saute au curseur. Une
 * explosion à état afficherait n'importe quoi après un retour en arrière. Chaque particule
 * est donc REJOUÉE depuis son germe à chaque image — même germe, même âge, même image.
 *
 * BORNÉE, ET LA BORNE EST MESURÉE (item 3.4) : 12 braises + 10 éclats + 6 bouffées = 28
 * particules par explosion, au plus 84 pas de simulation (1,4 s à 60 Hz). Le coût d'UNE
 * explosion, relevé sur toute sa timeline et figé par test : **19 dégradés radiaux et 227
 * primitives** au pire instant. Le corpus des 23 artefacts locaux (801 explosions posées)
 * ne montre jamais plus de **5 explosions simultanées** dans la fenêtre de 1,4 s : le pire
 * cas réel coûte donc 95 dégradés et ~1 135 primitives sur une image, sur un canevas dont
 * le fond, la structure, les zones et les objectifs sont déjà cuits hors écran.
 */

/** Durée totale de la timeline, en millisecondes (celle de la référence). */
export const EXPLOSION_MS = 1_400

/** Pas de simulation, en millisecondes : les coefficients de la référence sont par image 60 Hz. */
const STEP_MS = 1000 / 60

/** Bornes de coût — la raison d'être de l'item 3.4. */
const EMBERS = 12
const SPARKS = 10
const DUST = 6
const MAX_STEPS = Math.ceil(EXPLOSION_MS / STEP_MS)

/** Rayon de référence de la boule de feu, en pixels d'ÉCRAN. */
const BLAST_R = 13
/** Durée du flash radial (référence : ~70 ms, puis extinction sur trois fois autant). */
const FLASH_MS = 70
/** Durée de l'onde de choc. */
const WAVE_MS = 380

/** Les encres d'une explosion, résolues depuis le thème (aucune couleur écrite ici). */
export interface ExplosionInk {
  /** Teinte de la déflagration : elle suit le TYPE de grenade (feu, plasma, cristal). */
  fire: string
  /** Cœur incandescent, commun à tous les effets de tir et d'explosion. */
  core: string
  /** Poussière résiduelle : une encre de mise en page, pas une couleur de donnée. */
  smoke: string
}

/** Où et quand l'explosion se joue. */
export interface ExplosionShape {
  x: number
  y: number
  /** Âge de l'explosion, en millisecondes. Au-delà de EXPLOSION_MS, plus rien n'est dessiné. */
  ageMs: number
  /** Germe stable : deux lectures du même instant redonnent exactement la même image. */
  seed: number
  /** Densité du canevas : tout ce qui s'adresse à l'œil est multiplié par ce facteur. */
  k: number
  /** Sous « mouvement réduit », l'explosion ne se joue pas : une empreinte fixe la remplace. */
  reduced: boolean
}

/** Générateur pseudo-aléatoire à germe — le même que la nappe électrique (LCG 32 bits). */
function rng(seed: number): () => number {
  let s = (seed | 0) || 1
  return () => {
    s = (s * 1103515245 + 12345) & 0x7fffffff
    return s / 0x7fffffff
  }
}

/**
 * drawExplosion joue l'explosion à l'âge demandé. Rien à l'extérieur de la fenêtre : une
 * explosion terminée ne coûte pas une opération.
 */
export function drawExplosion(
  ctx: CanvasRenderingContext2D,
  shape: ExplosionShape,
  ink: ExplosionInk,
): void {
  if (shape.ageMs < 0 || shape.ageMs > EXPLOSION_MS) return
  if (!ink.fire) return
  ctx.save()
  if (shape.reduced) {
    drawStill(ctx, shape, ink)
    ctx.restore()
    return
  }
  const steps = Math.min(MAX_STEPS, Math.floor(shape.ageMs / STEP_MS))
  drawDust(ctx, shape, ink, steps)
  drawWave(ctx, shape, ink)
  drawEmbers(ctx, shape, ink, steps)
  drawSparks(ctx, shape, ink, steps)
  drawFlash(ctx, shape, ink)
  ctx.restore()
}

/**
 * drawStill — SOUS MOUVEMENT RÉDUIT : une empreinte fixe, ni particules ni progression.
 * L'information (« quelque chose a explosé ici ») est gardée ; la stimulation ne l'est pas.
 */
function drawStill(ctx: CanvasRenderingContext2D, s: ExplosionShape, ink: ExplosionInk): void {
  ctx.globalCompositeOperation = 'source-over'
  ctx.globalAlpha = 0.55
  ctx.strokeStyle = ink.fire
  ctx.lineWidth = 1.6 * s.k
  ctx.beginPath()
  ctx.arc(s.x, s.y, BLAST_R * s.k, 0, Math.PI * 2)
  ctx.stroke()
  ctx.globalAlpha = 0.8
  ctx.fillStyle = ink.fire
  ctx.beginPath()
  ctx.arc(s.x, s.y, 2.2 * s.k, 0, Math.PI * 2)
  ctx.fill()
}

/**
 * drawEmbers — LA BOULE DE FEU : des bouffées projetées radialement, freinées (traînée 0,86
 * par pas, référence) et légèrement tirées vers le bas. Elles GRANDISSENT en s'éteignant, et
 * leur cœur passe de l'incandescent à la teinte du type : c'est la scène de feu, en deux
 * encres du thème plutôt qu'en quatre couleurs écrites en dur.
 */
function drawEmbers(
  ctx: CanvasRenderingContext2D,
  s: ExplosionShape,
  ink: ExplosionInk,
  steps: number,
): void {
  ctx.globalCompositeOperation = 'lighter'
  const rand = rng(s.seed)
  for (let i = 0; i < EMBERS; i++) {
    const ang = rand() * Math.PI * 2
    const sp = (0.35 + rand() * 1.1) * s.k
    const life = 16 + rand() * 16
    const r0 = (2.4 + rand() * 2.6) * s.k
    const el = steps / life
    if (el >= 1) continue
    let x = s.x
    let y = s.y
    let vx = Math.cos(ang) * sp
    let vy = Math.sin(ang) * sp * 0.85
    for (let n = 0; n < steps; n++) {
      x += vx
      y += vy
      vx *= 0.86
      vy = vy * 0.86 + 0.02 * s.k
    }
    const r = r0 * (1 + el * 1.3)
    // Jeune = cœur incandescent, vieille = teinte du type : le refroidissement de la
    // référence, dit avec les deux encres dont on dispose.
    paintRadial(ctx, x, y, r, el < 0.35 ? ink.core : ink.fire, (1 - el) * 0.8)
  }
}

/**
 * drawSparks — LES ÉCLATS : des traînées fines et rapides, freinées et attirées vers le bas.
 * Ce sont des traits, pas des dégradés — c'est ce qui rend la gerbe bon marché.
 */
function drawSparks(
  ctx: CanvasRenderingContext2D,
  s: ExplosionShape,
  ink: ExplosionInk,
  steps: number,
): void {
  ctx.globalCompositeOperation = 'lighter'
  ctx.lineCap = 'round'
  ctx.strokeStyle = ink.fire
  const rand = rng(s.seed ^ 0x5f5f)
  for (let i = 0; i < SPARKS; i++) {
    const ang = rand() * Math.PI * 2
    const sp = (1.1 + rand() * 1.6) * s.k
    const life = 22 + rand() * 18
    const sl = steps / life
    if (sl >= 1) continue
    let x = s.x
    let y = s.y
    let vx = Math.cos(ang) * sp
    let vy = Math.sin(ang) * sp
    for (let n = 0; n < steps; n++) {
      x += vx
      y += vy
      vx *= 0.9
      vy = vy * 0.9 + 0.04 * s.k
    }
    ctx.globalAlpha = 1 - sl
    ctx.lineWidth = (1 - sl) * 1.4 * s.k + 0.3
    ctx.beginPath()
    ctx.moveTo(x - vx * 1.8, y - vy * 1.8)
    ctx.lineTo(x, y)
    ctx.stroke()
  }
}

/**
 * drawDust — LA POUSSIÈRE RÉSIDUELLE : quelques bouffées qui s'élargissent et s'attardent,
 * peintes SOUS le feu (composition normale). C'est ce qui empêche l'explosion de disparaître
 * d'un coup — la référence les fait vivre bien plus longtemps que les braises.
 */
function drawDust(
  ctx: CanvasRenderingContext2D,
  s: ExplosionShape,
  ink: ExplosionInk,
  steps: number,
): void {
  if (!ink.smoke) return
  ctx.globalCompositeOperation = 'source-over'
  const rand = rng(s.seed ^ 0x2b2b)
  for (let i = 0; i < DUST; i++) {
    const ang = rand() * Math.PI * 2
    const sp = (0.1 + rand() * 0.4) * s.k
    const life = 60 + rand() * 50
    const r0 = (3.5 + rand() * 3) * s.k
    const grow = (0.05 + rand() * 0.09) * s.k
    const lf = steps / life
    if (lf >= 1) continue
    let x = s.x
    let y = s.y
    let vx = Math.cos(ang) * sp
    let vy = Math.sin(ang) * sp
    for (let n = 0; n < steps; n++) {
      x += vx
      y += vy
      vx *= 0.95
      vy *= 0.95
    }
    // Elle monte en opacité au tout début (elle se forme), puis s'efface : c'est la rampe
    // de la référence, qui évite une bouffée qui « apparaît » d'un bloc.
    const a = (1 - lf) * Math.min(1, lf / 0.12) * 0.4
    paintRadial(ctx, x, y, r0 + grow * steps, ink.smoke, a)
  }
}

/** drawWave — L'ONDE DE CHOC : un anneau qui s'ouvre vite puis se calme (easeOutCubic). */
function drawWave(ctx: CanvasRenderingContext2D, s: ExplosionShape, ink: ExplosionInk): void {
  if (s.ageMs >= WAVE_MS) return
  const pr = s.ageMs / WAVE_MS
  const ease = 1 - Math.pow(1 - pr, 3)
  ctx.globalCompositeOperation = 'lighter'
  ctx.strokeStyle = ink.fire
  ctx.globalAlpha = (1 - pr) * 0.8
  ctx.lineWidth = (1 - pr) * 2.4 * s.k + 0.4
  ctx.beginPath()
  ctx.arc(s.x, s.y, ease * BLAST_R * 2.2 * s.k, 0, Math.PI * 2)
  ctx.stroke()
}

/** drawFlash — LE FLASH RADIAL, très bref, par-dessus tout le reste. */
function drawFlash(ctx: CanvasRenderingContext2D, s: ExplosionShape, ink: ExplosionInk): void {
  const fl =
    s.ageMs < FLASH_MS ? 1 : s.ageMs < FLASH_MS * 4 ? 1 - (s.ageMs - FLASH_MS) / (FLASH_MS * 3) : 0
  if (fl <= 0) return
  ctx.globalCompositeOperation = 'lighter'
  paintRadial(ctx, s.x, s.y, BLAST_R * 1.8 * s.k, ink.core || ink.fire, fl * 0.9)
}

/** paintRadial — un disque dégradé, de la couleur donnée vers la transparence. */
function paintRadial(
  ctx: CanvasRenderingContext2D,
  x: number,
  y: number,
  radius: number,
  color: string,
  alpha: number,
): void {
  if (!color || radius <= 0 || alpha <= 0) return
  const g = ctx.createRadialGradient(x, y, 0, x, y, radius)
  g.addColorStop(0, color)
  // « transparent » plutôt qu'une couleur à alpha zéro : le canevas accepte n'importe quel
  // format du thème (oklch compris) sans qu'on ait à le désassembler.
  g.addColorStop(1, 'transparent')
  ctx.globalAlpha = Math.min(1, alpha)
  ctx.fillStyle = g
  ctx.beginPath()
  ctx.arc(x, y, radius, 0, Math.PI * 2)
  ctx.fill()
}
