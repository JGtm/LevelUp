/**
 * flagReturnZone.ts — LA ZONE DE RETOUR D'UN DRAPEAU TOMBÉ, et sa jauge.
 *
 * CE QUE LE JEU FAIT, ET QUE LE REJEU IGNORAIT. Un drapeau de CTF resté au sol ne s'y éternise
 * pas : une zone l'entoure, les coéquipiers de son camp qui s'y tiennent VIDENT sa jauge de
 * retour plus vite, et sans personne elle se vide seule jusqu'à ce qu'il rentre. Jusqu'au schéma
 * 29 le rejeu laissait le drapeau au sol jusqu'à sa reprise — des lâchers de plus de deux minutes
 * qui n'ont jamais existé à l'écran.
 *
 * LE MODÈLE VIENT DU JEU, PAS D'UNE INTUITION. Son propre script nomme la fonction qui accélère :
 * `CalculateReturnRateHarmonic`. La jauge se remplit donc au taux
 *
 *     1 / resetSeconds  +  H(n) / soloSeconds        H(n) = 1 + 1/2 + … + 1/n
 *
 * où `n` est le nombre de DÉFENSEURS dans la zone. Deux défenseurs valent 1 + 1/2, trois
 * 1 + 1/2 + 1/3 : le rendement décroît — être plus nombreux aide, mais de moins en moins.
 *
 * CE QUE LE REJEU NE MODÉLISE PAS, ET C'EST UNE DÉCISION MESURÉE : la CONTESTATION. Le script du
 * jeu décrit un état `Contested` (un ennemi du camp propriétaire dans la zone bloque le retour)
 * puis `ContestedRefilling` (la jauge repart en arrière). Trois faits l'ont écarté du rendu, et
 * ils vont tous dans le même sens :
 *
 *  - l'utilisateur ne l'a JAMAIS observé en jeu — « pas d'arrêt de la jauge ni de reset, sauf si
 *    on reprend le drapeau adverse » ;
 *  - ni le réglage qui l'active (`flagContestedStateEnabled`) ni son taux
 *    (`flagContestRefillRate`) ne sont lisibles : leurs constantes sont dédupliquées dans le pool
 *    du script ;
 *  - la MESURE explique le silence — sur 72 lâchers où un ennemi entre dans la zone, 56 finissent
 *    par une REPRISE, et son séjour dure 1,65 s en moyenne. À 1,3 m d'un drapeau tombé, un ennemi
 *    ne conteste pas : il RAMASSE.
 *
 * LA SEULE INTERRUPTION VISIBLE EST DONC LA REPRISE, et le rejeu la rend déjà sans rien de plus :
 * un intervalle `carried` ferme le lâcher, la zone disparaît, et le lâcher SUIVANT rouvre une
 * jauge NEUVE repartie de zéro (cf. `mergedDrops`).
 *
 * UN DRAPEAU NEUTRE N'A PAS DE DÉFENSEUR, et rien ici ne le teste : son équipe vaut -1, la liste
 * revient vide, et la jauge se réduit à la minuterie. C'est exactement la règle du jeu — un
 * drapeau neutre ne se renvoie pas, il revient tout seul.
 *
 * POURQUOI LE CALCUL EST ICI ET NON SUR LE SERVEUR. Compter les défenseurs exige de savoir à
 * quelle équipe appartient chaque joueur — et **l'équipe n'est pas dans le film** (`Track.Team`
 * vaut -1). Le constructeur du rejeu est hors ligne et n'ouvre aucune base ; la page, elle, a
 * déjà joint le tableau de bord pour colorer les camps. Le serveur publie donc la RÈGLE
 * (`doc.flagReturnZone` : un rayon et deux durées), le client compte et intègre.
 *
 * LE MODÈLE DONNE LA FORME, L'OBSERVATION DONNE LES BORNES. Quand le rejeu SAIT à quelle image le
 * drapeau est rentré (le lâcher est suivi d'un état `home`), la jauge est remise à l'échelle pour
 * atteindre exactement 1 à cette image : on ne prétend pas prédire ce qui est observé. Sans
 * retour observé (le drapeau est repris, ou l'axe s'arrête), la jauge court telle quelle et se
 * plafonne à 1.
 */
import type { XY } from '../model/replayLogic'

import type { ReplayFlagCarryReady } from '../model/replayNormalize'

/** La règle de retour du mode, telle que `doc.flagReturnZone` la publie (schéma 35). */
export interface FlagReturnRule {
  /** Rayon de la zone de retour, dans les coordonnées monde du rejeu. */
  radiusM: number
  /** Durée qu'un drapeau au sol met à rentrer TOUT SEUL. */
  resetSeconds: number
  /** Durée qu'il met avec UN défenseur dans la zone. */
  soloSeconds: number
}

/** Un lâcher instrumenté : sa position image par image, sa jauge et son occupation. */
export interface FlagReturnDrop {
  /** Le rayon de la zone, dans les coordonnées monde. */
  radiusM: number
  /** L'équipe PROPRIÉTAIRE du drapeau — ce sont ses joueurs qui le renvoient. */
  team: number
  /** Bornes du lâcher, en images. `t1` est INCLUSE. */
  t0: number
  t1: number
  /** La position du drapeau à chaque image de `[t0, t1]` (il roule : elle bouge). */
  x: Float32Array
  y: Float32Array
  /** La jauge de retour à chaque image, dans `[0, 1]`. */
  progress: Float32Array
  /** Le nombre de défenseurs dans la zone à chaque image. */
  occupants: Uint8Array
  /** L'image du retour OBSERVÉ, `null` quand le lâcher finit autrement (reprise, fin d'axe). */
  returnFrame: number | null
}

/** H(n) = 1 + 1/2 + … + 1/n, et 0 pour n <= 0. Le taux d'accélération que le jeu applique. */
export function harmonic(n: number): number {
  let h = 0
  for (let i = 1; i <= n; i += 1) h += 1 / i
  return h
}

/** Ce qu'un lâcher montre à une image donnée. */
export interface FlagReturnNow {
  /** L'équipe PROPRIÉTAIRE du drapeau : c'est elle qui donne l'encre. */
  team: number
  x: number
  y: number
  radiusM: number
  progress: number
  occupants: number
}

/** flagReturnAt rend les lâchers ACTIFS à une image — leur position, leur jauge, leur monde. */
export function flagReturnAt(drops: readonly FlagReturnDrop[], frame: number): FlagReturnNow[] {
  const out: FlagReturnNow[] = []
  for (const d of drops) {
    if (frame < d.t0 || frame > d.t1) continue
    const i = frame - d.t0
    out.push({
      team: d.team,
      x: d.x[i],
      y: d.y[i],
      radiusM: d.radiusM,
      progress: d.progress[i],
      occupants: d.occupants[i],
    })
  }
  return out
}

/** Ce qu'il faut pour instrumenter les lâchers : la position d'un joueur, et qui défend. */
export interface FlagReturnInput {
  rule: FlagReturnRule | null
  frameIntervalMs: number
  posOf: (xuid: string, frame: number) => XY | null
  /**
   * Les xuid des joueurs de l'équipe donnée, tels que le tableau de bord les nomme.
   *
   * VIDE POUR UN DRAPEAU NEUTRE (équipe -1), et ce n'est pas un oubli : un drapeau que personne
   * ne possède n'a pas de défenseur. Il revient tout seul, à la minuterie, et le modèle le rend
   * ainsi sans qu'aucune branche ne le dise.
   */
  defendersOf: (team: number) => readonly string[]
}

/**
 * buildFlagReturnDrops instrumente chaque lâcher publié.
 *
 * LES SPANS `dropped` CONTIGUS SONT UN SEUL LÂCHER : le calque en ouvre un nouveau dès que la
 * POSITION change (le drapeau roule). Les traiter séparément couperait la jauge en morceaux.
 */
export function buildFlagReturnDrops(
  carries: readonly ReplayFlagCarryReady[],
  input: FlagReturnInput,
): FlagReturnDrop[] {
  const { rule, frameIntervalMs } = input
  if (!rule || rule.radiusM <= 0 || rule.resetSeconds <= 0 || rule.soloSeconds <= 0) return []
  if (frameIntervalMs <= 0) return []
  const out: FlagReturnDrop[] = []
  for (const carry of carries) {
    for (const run of mergedDrops(carry)) out.push(instrument(run, carry.team ?? -1, input, rule))
  }
  return out
}

/** Un lâcher AVANT instrumentation : ses bornes, ses poses successives, et sa fin observée. */
interface DropRun {
  t0: number
  t1: number
  poses: { t0: number; x: number; y: number }[]
  returnFrame: number | null
}

/**
 * mergedDrops fusionne les intervalles `dropped` contigus d'un drapeau.
 *
 * ET C'EST ICI QUE LA REPRISE REMET LA JAUGE À ZÉRO, sans qu'une ligne ne le dise : un intervalle
 * qui n'est pas `dropped` ferme le lâcher courant, et le suivant ouvre un `DropRun` NEUF dont
 * l'accumulateur repart de zéro. C'est le seul « reset » que le joueur observe en jeu.
 */
function mergedDrops(carry: ReplayFlagCarryReady): DropRun[] {
  const spans = carry.spans
  const out: DropRun[] = []
  let cur: DropRun | null = null
  for (let i = 0; i < spans.length; i += 1) {
    const s = spans[i]
    if (s.state !== 'dropped') {
      cur = null
      continue
    }
    const pose = { t0: s.t0, x: s.x, y: s.y }
    if (cur && cur.t1 + 1 === s.t0) {
      cur.t1 = s.t1
      cur.poses.push(pose)
    } else {
      cur = { t0: s.t0, t1: s.t1, poses: [pose], returnFrame: null }
      out.push(cur)
    }
    const next = spans[i + 1]
    cur.returnFrame = next && next.state === 'home' ? next.t0 : null
  }
  return out
}

/** instrument calcule, image par image, la position, l'occupation et la jauge d'un lâcher. */
function instrument(
  run: DropRun,
  team: number,
  input: FlagReturnInput,
  rule: FlagReturnRule,
): FlagReturnDrop {
  const n = run.t1 - run.t0 + 1
  const x = new Float32Array(n)
  const y = new Float32Array(n)
  const occupants = new Uint8Array(n)
  const progress = new Float32Array(n)
  const defenders = input.defendersOf(team)
  const dt = input.frameIntervalMs / 1000
  const r2 = rule.radiusM * rule.radiusM
  let pose = 0
  let acc = 0
  for (let i = 0; i < n; i += 1) {
    const frame = run.t0 + i
    while (pose + 1 < run.poses.length && run.poses[pose + 1].t0 <= frame) pose += 1
    x[i] = run.poses[pose].x
    y[i] = run.poses[pose].y
    occupants[i] = countInside(defenders, frame, x[i], y[i], r2, input.posOf)
    acc += (1 / rule.resetSeconds + harmonic(occupants[i]) / rule.soloSeconds) * dt
    progress[i] = acc
  }
  rescale(progress, run)
  return {
    radiusM: rule.radiusM,
    team,
    t0: run.t0,
    t1: run.t1,
    x,
    y,
    progress,
    occupants,
    returnFrame: run.returnFrame,
  }
}

/** countInside compte les défenseurs dont la position connue tombe dans la zone. */
function countInside(
  defenders: readonly string[],
  frame: number,
  fx: number,
  fy: number,
  r2: number,
  posOf: FlagReturnInput['posOf'],
): number {
  let n = 0
  for (const xuid of defenders) {
    const p = posOf(xuid, frame)
    if (!p) continue
    const dx = p.x - fx
    const dy = p.y - fy
    if (dx * dx + dy * dy <= r2) n += 1
  }
  return n > 255 ? 255 : n
}

/**
 * rescale fait ATTERRIR la jauge sur le retour OBSERVÉ.
 *
 * Le modèle donne la forme — le rythme, l'accélération quand la zone se remplit — mais c'est le
 * film qui dit à quelle image le drapeau est rentré. Sans ce recalage la jauge finirait à 0,8 ou
 * à 1,4 selon la finesse du comptage, et l'œil lirait un retour prématuré ou en retard. Sans
 * retour observé, rien à recaler : la jauge est simplement plafonnée.
 */
function rescale(progress: Float32Array, run: DropRun): void {
  const at = run.returnFrame === null ? -1 : run.returnFrame - 1 - run.t0
  const end = at >= 0 && at < progress.length ? progress[at] : 0
  const k = end > 0 ? 1 / end : 0
  for (let i = 0; i < progress.length; i += 1) {
    const v = k > 0 ? progress[i] * k : progress[i]
    progress[i] = v > 1 ? 1 : v
  }
}

/**
 * LE RENDU — un anneau à la place du drapeau tombé, et une jauge qui se vide dessus.
 *
 * TROIS DÉCISIONS DE LECTURE, et chacune répond à ce que l'œil doit comprendre en une image :
 *
 *  - LE CERCLE EST LA ZONE RÉELLE, à l'échelle de la carte. Elle est petite — on marche
 *    littéralement sur le drapeau pour le renvoyer — et la dessiner plus grande « pour qu'on la
 *    voie » mentirait sur la portée du geste. Un plancher de quelques pixels évite seulement
 *    qu'elle disparaisse aux petits zooms.
 *  - LA JAUGE EST UN ARC QUI SE VIDE, parti du haut et tournant dans le sens des aiguilles : ce
 *    qui reste à l'écran est le temps qui reste avant que le drapeau rentre.
 *  - L'OCCUPATION SE VOIT SANS CHIFFRE : dès qu'un défenseur est dedans, l'anneau s'épaissit et
 *    s'éclaircit. C'est le seul signal qui dise « ça va plus vite en ce moment » — la vitesse
 *    d'une jauge ne se lit pas sur une image fixe.
 */
export interface FlagReturnStyle {
  /** L'encre du camp PROPRIÉTAIRE du drapeau, résolue par l'appelant (règle color-tokens). */
  colorOfTeam: (team: number) => string
}

/** Réglages du tracé : francs sans concurrencer le glyphe du drapeau qui se pose dessus. */
const ZONE_FILL_ALPHA = 0.1
const ZONE_RING_ALPHA = 0.45
const ZONE_RING_ALPHA_BUSY = 0.9
const GAUGE_ALPHA = 0.95
const GAUGE_WIDTH_K = 2.5
const MIN_RADIUS_PX = 4
/**
 * GAUGE_MIN_PX — la JAUGE a son propre plancher, PLUS GRAND que celui de la zone, et les deux
 * disent deux choses différentes.
 *
 * La zone est une VÉRITÉ DE CARTE : 1,3 m à l'échelle du rejeu, c'est-à-dire quelques pixels —
 * la grossir mentirait sur la portée du geste. La jauge, elle, est une LECTURE : un arc de trois
 * pixels de rayon ne se lit pas. Elle se pose donc juste à l'extérieur du glyphe du drapeau
 * (rayon de touche 12 × 1,45), comme le fait le HUD du jeu — attachée à l'objet, pas au terrain.
 */
const GAUGE_MIN_PX = 20
/** Épaisseur de base de l'anneau, en pixels. La jauge la multiplie pour passer devant. */
const RING_WIDTH = 1.5

/** drawFlagReturnZones peint les zones de retour ACTIVES à une image. */
export function drawFlagReturnZones(
  ctx: CanvasRenderingContext2D,
  drops: readonly FlagReturnDrop[],
  project: (p: XY) => XY,
  pxPerM: number,
  frame: number,
  style: FlagReturnStyle,
): void {
  for (const now of flagReturnAt(drops, frame)) {
    drawOne(ctx, project({ x: now.x, y: now.y }), now, {
      r: Math.max(pxPerM * now.radiusM, MIN_RADIUS_PX),
      ink: style.colorOfTeam(now.team),
    })
  }
}

/** Ce que le tracé d'une zone a besoin de savoir de l'écran : sa taille et son encre. */
interface ZonePaint {
  r: number
  ink: string
}

/** drawOne peint UNE zone : le disque, l'anneau, puis l'arc restant. */
function drawOne(ctx: CanvasRenderingContext2D, c: XY, now: FlagReturnNow, paint: ZonePaint): void {
  const busy = now.occupants > 0
  ctx.save()
  ctx.fillStyle = paint.ink
  ctx.strokeStyle = paint.ink
  // UN DRAPEAU NEUTRE N'A PAS DE ZONE, et le cercle disparaît avec elle. Personne ne le possède,
  // donc personne ne le renvoie : dessiner un anneau autour de lui promettrait une action qui
  // n'existe pas. La JAUGE, elle, reste — c'est la minuterie, et elle est bien réelle.
  if (now.team >= 0) {
    ctx.globalAlpha = ZONE_FILL_ALPHA
    ctx.beginPath()
    ctx.arc(c.x, c.y, paint.r, 0, Math.PI * 2)
    ctx.fill()
    ctx.globalAlpha = busy ? ZONE_RING_ALPHA_BUSY : ZONE_RING_ALPHA
    ctx.lineWidth = RING_WIDTH * (busy ? 2 : 1)
    ctx.beginPath()
    ctx.arc(c.x, c.y, paint.r, 0, Math.PI * 2)
    ctx.stroke()
  }
  const left = 1 - Math.min(Math.max(now.progress, 0), 1)
  if (left > 0) {
    const rg = Math.max(paint.r, GAUGE_MIN_PX)
    ctx.globalAlpha = GAUGE_ALPHA
    ctx.lineWidth = RING_WIDTH * GAUGE_WIDTH_K
    ctx.beginPath()
    ctx.arc(c.x, c.y, rg, -Math.PI / 2, -Math.PI / 2 + left * Math.PI * 2)
    ctx.stroke()
  }
  ctx.restore()
}
