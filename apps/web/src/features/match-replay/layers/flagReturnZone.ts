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
 * quelle équipe appartient chaque joueur — la VIE ne le dit pas (`Track.Team` vaut -1) ; c'est
 * l'entrée de roster du film qui le porte, et la page l'a déjà jointe pour colorer les camps
 * (`FilmAllegiance.membersOf`). Le serveur publie donc la RÈGLE (`doc.flagReturnZone` : un
 * rayon et deux durées), le client compte et intègre.
 *
 * LE MODÈLE DONNE LA FORME, L'OBSERVATION DONNE LES BORNES. Quand le rejeu SAIT à quelle image le
 * drapeau est rentré (le lâcher est suivi d'un état `home`), la jauge est remise à l'échelle pour
 * atteindre exactement 1 à cette image : on ne prétend pas prédire ce qui est observé. Sans
 * retour observé (le drapeau est repris, ou l'axe s'arrête), la jauge court telle quelle et se
 * plafonne à 1.
 */
import type { XY } from '../../../lib/replay/replayLogic'

import type { ReplayFlagCarryReady } from '../../../lib/replay/replayNormalize'

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
   * Les clés des joueurs de l'équipe donnée, telles que le FILM les nomme (`ReplayPlayer.xuid`,
   * celles des relectures de position — `FilmAllegiance.membersOf`).
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
 * LE RENDU — une ZONE AU SOL à la place du drapeau tombé, et sa jauge à l'extérieur.
 *
 * CE QUI SE DESSINE, de dessous en dessus :
 *
 *  - LE DISQUE de la zone, à l'encre du camp PROPRIÉTAIRE (opacité 0,2 ; 0,3 quand un défenseur
 *    est dedans) : c'est la surface où l'on se place pour renvoyer le drapeau ;
 *  - L'ANNEAU net de son bord (opacité 0,9, 2 px ; 3 px quand un défenseur est dedans) : il
 *    délimite la zone, rien d'autre ;
 *  - LA JAUGE, un arc DISTINCT sur son propre rayon, toujours au-delà de l'anneau (cf.
 *    `gaugeRadiusPx`) : elle ne se pose jamais sur lui, sinon l'œil lit une minuterie et non une
 *    zone. C'est elle qui se vide, du haut dans le sens des aiguilles : ce qui reste est le temps
 *    qui reste avant que le drapeau rentre.
 *
 * LE CERCLE EST LA ZONE RÉELLE, à l'échelle de la carte, centrée sur le pied du drapeau. Elle est
 * petite — on marche littéralement sur le drapeau pour le renvoyer — et la dessiner plus grande
 * « pour qu'on la voie » mentirait sur la portée du geste. Seul un plancher (`MIN_RADIUS_PX`)
 * empêche qu'elle disparaisse sous le glyphe du drapeau aux petits zooms.
 *
 * L'OCCUPATION SE VOIT SANS CHIFFRE : dès qu'un défenseur est dedans, le disque et l'anneau
 * s'épaississent. C'est le seul signal qui dise « ça va plus vite en ce moment » — la vitesse
 * d'une jauge ne se lit pas sur une image fixe.
 *
 * UN DRAPEAU NEUTRE (équipe -1) N'A PAS DE ZONE : ni disque ni anneau, la jauge seule.
 */
export interface FlagReturnStyle {
  /** L'encre du camp PROPRIÉTAIRE du drapeau, résolue par l'appelant (règle color-tokens). */
  colorOfTeam: (team: number) => string
}

const ZONE_FILL_ALPHA = 0.2
const ZONE_FILL_ALPHA_BUSY = 0.3
const ZONE_RING_ALPHA = 0.9
const ZONE_RING_WIDTH = 2
const ZONE_RING_WIDTH_BUSY = 3
const GAUGE_ALPHA = 0.95
const GAUGE_WIDTH = 3.75
/**
 * MIN_RADIUS_PX — plancher du rayon de la ZONE : sur les grandes cartes, la zone réelle ne tient
 * que dans quelques pixels. Il ne couvre PAS le glyphe (hampe de 13 × 1,45 ≈ 18,9 px, demi-hauteur
 * ≈ 9,4 px) : il garantit un disque de 16 px centré sur le pied, dont la moitié basse et le flanc
 * gauche restent hors du glyphe, qui monte du pied vers le haut et la droite.
 */
const MIN_RADIUS_PX = 8
/** GAUGE_GAP_PX — l'écart minimal entre le bord de la zone et l'arc de la jauge. */
const GAUGE_GAP_PX = 5
/**
 * GAUGE_MIN_PX — plancher du rayon de la JAUGE : un arc de quelques pixels de rayon ne se lit
 * pas. Elle se pose à l'extérieur du glyphe du drapeau, comme le HUD du jeu.
 */
const GAUGE_MIN_PX = 20

/**
 * gaugeRadiusPx — le rayon de l'arc de jauge pour une zone de rayon `zoneR` : toujours STRICTEMENT
 * plus grand que celui de l'anneau, jamais confondu avec lui.
 */
export function gaugeRadiusPx(zoneR: number): number {
  return Math.max(zoneR + GAUGE_GAP_PX, GAUGE_MIN_PX)
}

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

/** drawOne peint UNE zone : le disque, l'anneau, puis l'arc restant sur son propre rayon. */
function drawOne(ctx: CanvasRenderingContext2D, c: XY, now: FlagReturnNow, paint: ZonePaint): void {
  const busy = now.occupants > 0
  ctx.save()
  ctx.fillStyle = paint.ink
  ctx.strokeStyle = paint.ink
  // UN DRAPEAU NEUTRE N'A PAS DE ZONE, et le cercle disparaît avec elle. Personne ne le possède,
  // donc personne ne le renvoie : dessiner un anneau autour de lui promettrait une action qui
  // n'existe pas. La JAUGE, elle, reste — c'est la minuterie, et elle est bien réelle.
  if (now.team >= 0) {
    ctx.globalAlpha = busy ? ZONE_FILL_ALPHA_BUSY : ZONE_FILL_ALPHA
    ctx.beginPath()
    ctx.arc(c.x, c.y, paint.r, 0, Math.PI * 2)
    ctx.fill()
    ctx.globalAlpha = ZONE_RING_ALPHA
    ctx.lineWidth = busy ? ZONE_RING_WIDTH_BUSY : ZONE_RING_WIDTH
    ctx.beginPath()
    ctx.arc(c.x, c.y, paint.r, 0, Math.PI * 2)
    ctx.stroke()
  }
  const left = 1 - Math.min(Math.max(now.progress, 0), 1)
  if (left > 0) {
    ctx.globalAlpha = GAUGE_ALPHA
    ctx.lineWidth = GAUGE_WIDTH
    ctx.beginPath()
    ctx.arc(c.x, c.y, gaugeRadiusPx(paint.r), -Math.PI / 2, -Math.PI / 2 + left * Math.PI * 2)
    ctx.stroke()
  }
  ctx.restore()
}
