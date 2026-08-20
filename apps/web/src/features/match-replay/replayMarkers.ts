/**
 * replayMarkers.ts — LE JOUEUR SUR LA CARTE : traînée, marqueur d'étage, cône de visée,
 * apparition et mort.
 *
 * CE QUE CE CALQUE NE DESSINE PLUS. Une jauge de BOUCLIER surmontait chaque marqueur ; elle a
 * été retirée le 2026-08-15 à la demande de l'utilisateur — « je veux pas ces barres
 * horizontales au dessus des points de joueurs » — et cette demande remplace la décision du
 * 2026-08-13 qui l'avait rendue permanente. Le champ `sh` continue de voyager : ce sont les
 * FICHES de joueurs qui le lisent (rosterLogic.ts). C'est le dessin sur la carte qui est
 * refusé, pas la mesure.
 *
 * Une trace = UNE VIE, jamais un joueur : le slot de biped est réattribué à chaque
 * réapparition. Les trois instants d'une vie se lisent donc séparément — elle S'OUVRE (anneau),
 * elle DURE (marqueur + traînée), elle SE FERME (croix). Sans ces repères, une vie qui
 * disparaît et une autre qui commence deux mètres plus loin sont indiscernables.
 *
 * TOUT CE QUI S'ADRESSE À L'ŒIL EST À L'ÉCHELLE DE L'ÉCRAN, pas du canevas. Le canevas est
 * rendu à la densité du périphérique : un rayon déclaré en pixels de canevas apparaîtrait deux
 * fois plus petit sur un écran à forte densité. Les positions, elles, appartiennent au MONDE et
 * n'y touchent pas. C'est la distinction que le POC avait dû introduire après avoir constaté
 * que des points « grossis » restaient invisibles.
 *
 * CE QUE CHAQUE MARQUEUR PORTE DEPUIS LE 2026-08-16 (plan d'habillage, décisions D1-D5) : la
 * couleur d'ÉQUIPE de son propriétaire (jamais une teinte par vie), sa FORME d'identité
 * (losange pour un ami, disque cerclé pour le joueur de la page) et son NOM écrit dessous
 * (replayLabels.ts). Le trait qui sortait du point — l'axe du cône — a été SUPPRIMÉ à la
 * demande de l'utilisateur (« supprimer le bâton qui sort de ces points ») : le cône dégradé
 * dit déjà la direction du regard.
 *
 * Aucun littéral de couleur : les teintes d'équipe arrivent résolues depuis les tokens, les
 * encres de lisibilité depuis le thème (cf. canvasInk.ts).
 */
import type { PlayerMarkKind } from './playerMarks'
import { drawAimCone } from './replayAimCone'
import { drawNameLabel } from './replayLabels'
import type { ReplayTrackReady } from './replayNormalize'

import {
  altitudeAt,
  floorOf,
  isAliveAt,
  positionAt,
  trackWindow,
  trailAt,
  worldToCanvas,
  type XY,
} from './replayLogic'

/** Cadrage du canvas (mêmes paramètres que worldToCanvas). */
export interface CanvasView {
  bounds: { minX: number; minY: number; maxX: number; maxY: number }
  width: number
  height: number
  pad: number
}

// --- Durées, en frames, converties par l'appelant depuis le temps réel ---------------------

/**
 * Réglages temporels du calque, tous exprimés en frames de la grille du rejeu.
 * Les valeurs de référence viennent du POC, où elles ont été réglées à l'écran :
 * traînée 7 s, cône maintenu 5 s (la couverture de la visée passe de 43,6 % à 93,5 % du temps
 * de jeu), apparition 0,8 s. La mort, elle, a été RALLONGÉE depuis : 2,5 s et non plus les
 * 1,5 s du POC, où la croix passait inaperçue en lecture accélérée (cf. TIMING_MS dans
 * `useReplayTiming.ts`, seule source des durées déclarées).
 */
export interface MarkerTiming {
  trail: number
  aimHold: number
  death: number
  spawn: number
}

// --- Constantes de rendu (pixels d'ÉCRAN, sauf mention) ------------------------------------

/**
 * LA TRAÎNÉE S'ÉCLAIRCIT VERS LE PASSÉ. Un seul trait à opacité constante disait « le joueur
 * est passé par là » sans dire DANS QUEL SENS ; l'alpha qui monte vers la tête donne la
 * direction du mouvement sans ajouter la moindre flèche. La polyligne se dessine donc segment
 * par segment, chacun avec son opacité.
 */
const TRAIL_WIDTH = 1.6
const TRAIL_ALPHA_TAIL = 0.08
const TRAIL_ALPHA_HEAD = 0.63

/**
 * LES TAILLES ET LE STYLE DU MARQUEUR SUIVENT LA PLANCHE VALIDÉE LE 2026-08-16 (item A1),
 * dont les valeurs sont transcrites au §1bis du plan d'habillage — verdict utilisateur :
 * « Parfait. Je veux exactement ce style pour les points, la croix de mort et la traînée. »
 * Ce que la planche change par rapport au POC : le point rétrécit (4,6 -> 3,4), le halo diffus
 * DISPARAÎT, et l'étage cesse d'être une couleur de trace pour devenir un anneau fin à l'encre
 * du thème.
 */
/** Rayon du noyau du marqueur, et son accroissement par étage. */
const CORE_RADIUS = 3.4
const CORE_PER_FLOOR = 0.7
/**
 * Anneaux concentriques : un par étage au-dessus du sol (règle du COMPTE inchangée). Le
 * premier est posé à un rayon FIXE — la planche le veut détaché du point, pas collé à lui.
 */
const RING_RADIUS = 6.5
const RING_GAP = 2.8
const RING_WIDTH = 1
const RING_ALPHA = 0.9
const RING_ALPHA_DECAY = 0.18
/** Liseré de lisibilité : la carte va du clair au sombre, un point coloré s'y perd sans lui. */
const OUTLINE_PAD = 1.0
const OUTLINE_ALPHA = 0.62
/**
 * LE JOUEUR DE LA PAGE (forme 'ring') — DOUBLE CONTOUR ET HALO depuis le 2026-08-18.
 *
 * CE QUE LE RETOUR DEMANDE : « avoir l'icône du joueur actif qui se démarque de tous les
 * autres aussi (j'aurais bien aimé du vert mais pour l'accessibilité je sais pas si ça peut
 * le faire) ». La réponse tient en une règle : LA COULEUR NE PORTE JAMAIS SEULE. Le marqueur
 * du joueur de la page était déjà cerclé d'UN anneau ; il en porte DEUX, plus un halo diffus,
 * et c'est cette FORME qui le distingue — un lecteur qui ne voit pas la teinte voit toujours
 * le seul point de la carte à deux anneaux. La couleur (`selfInk`, token `success`) vient EN
 * PLUS, sur le contour et le halo ; le NOYAU garde la couleur d'ÉQUIPE, qui dit le camp.
 *
 * LE HALO EST DE RETOUR ICI, ET SEULEMENT ICI. Il avait été supprimé de TOUS les marqueurs le
 * 16/08 (« la lueur diffuse doublait l'emprise et empâtait la carte ») — sur un seul point de
 * la carte, il ne l'empâte pas, il le désigne.
 */
const SELF_RING_WIDTH = 1.5
const SELF_RING_GAP = 1.6
/** Écart entre les deux anneaux d'identité, en pixels d'écran. */
const SELF_RING_GAP_2 = 2.4
/** Le halo : un disque diffus posé sous le marqueur, au rayon du second anneau. */
const SELF_HALO_PAD = 2.2
const SELF_HALO_ALPHA = 0.22

/**
 * CROIX DE MORT — demi-taille FIXE qui s'estompe (elle ne grandit plus, cf. §1bis).
 *
 * PLUS PETITE ET PLUS ÉPAISSE depuis le 2026-08-18 (A1) : « les croix de mort plus petites,
 * plus épaisses, toujours rouges, et qui persistent plus longtemps ». À 5 px de demi-taille
 * pour 1,6 px de trait, la croix avait l'emprise d'un marqueur vivant et le poids d'un trait
 * de traînée — on la voyait large et pâle. À 3,6 pour 2,6 elle occupe moitié moins de surface
 * et frappe deux fois plus : c'est une marque, pas un joueur.
 */
const DEATH_RADIUS = 3.6
const DEATH_WIDTH = 2.6
const DEATH_ALPHA = 0.9
const SPAWN_RADIUS = 2
const SPAWN_GROWTH = 12
const SPAWN_WIDTH = 1.2
const SPAWN_ALPHA = 0.8
/**
 * Style du calque : ce qu'une VIE emprunte à son PROPRIÉTAIRE, plus les encres du thème.
 *
 * TOUT SE LIT PAR SLOT, jamais par index de trace. Le slot est la clé d'une vie, et le
 * propriétaire de la vie porte sa couleur d'équipe, sa marque d'identité et son nom
 * (rosterLogic.ts). Une vie SANS propriétaire rend `null` en couleur, et le calque ne la
 * dessine PAS : ce sont les caméras et les spectateurs de fin de partie, qui ne désignent
 * personne (2026-08-20). La marque et le nom, eux, peuvent manquer sur une vie qui se
 * dessine — elle sort alors sans étiquette ni marque, pas sans marqueur.
 */
export interface MarkerStyle {
  /** Couleur d'équipe du propriétaire de la vie. `null` = ne rien dessiner pour ce slot. */
  colorOfSlot: (slot: number) => string | null
  /** Encre qui contraste avec la page dans les deux thèmes : liseré, anneau du joueur. */
  ink: string
  frame: number
  timing: MarkerTiming
  z: { min: number; max: number }
  /** Densité du canevas : tout ce qui s'adresse à l'œil est multiplié par ce facteur. */
  k: number
  showAim: boolean
  /** Marque d'identité du propriétaire : elle décide de la FORME du marqueur. */
  markOfSlot: (slot: number) => PlayerMarkKind | undefined
  /** Nom à écrire sous le marqueur ; `null` = vie sans propriétaire, donc sans étiquette. */
  nameOfSlot: (slot: number) => string | null
  /** Calque des noms (bouton « Noms », allumé par défaut) : un BTB doit pouvoir l'éteindre. */
  showNames: boolean
  /** Calque de la TRAÎNÉE (bouton « Traînée », allumé par défaut) — retour du 2026-08-18. */
  showTrail: boolean
  /** Encre du DOUBLE CONTOUR et du halo du joueur de la page (token `success`, cf. useReplayInks). */
  selfInk: string
  /**
   * Encre de la CROIX DE MORT — token `destructive`, la même que les effets de mort du
   * calque d'événements (cf. useReplayInks). « Toujours rouges » (A1, 2026-08-18) : une mort
   * ne dit plus le camp de celui qui meurt, elle dit qu'on est mort. Le camp reste porté par
   * la traînée qui vient de s'éteindre, par la fiche et par le fil.
   */
  deathInk: string
  /** Encre du CONTOUR des noms — sombre dans les deux thèmes (cf. replayLabels.ts). */
  labelStroke: string
}

/**
 * La FORME dit l'identité, la couleur dit le camp (décision D5) : un ami se repère sans lire
 * son nom, le joueur de la page aussi, et aucune des deux marques ne touche à la couleur qui
 * porte déjà l'équipe.
 */
export type MarkerShape = 'circle' | 'diamond' | 'ring'

/** shapeOfMark : ami = losange, joueur de la page = disque cerclé, tout le monde = disque. */
function shapeOfMark(mark: PlayerMarkKind | undefined): MarkerShape {
  if (mark === 'friend') return 'diamond'
  if (mark === 'me') return 'ring'
  return 'circle'
}

/**
 * drawTracksLayer dessine chaque vie à la frame courante : celles qui vivent, et celles qui
 * viennent de se fermer (la croix survit 2,5 s à la vie qu'elle termine).
 */
export function drawTracksLayer(
  ctx: CanvasRenderingContext2D,
  tracks: ReplayTrackReady[],
  view: CanvasView,
  style: MarkerStyle,
): void {
  ctx.lineCap = 'round'
  ctx.lineJoin = 'round'
  tracks.forEach((track) => {
    const color = style.colorOfSlot(track.slot)
    if (!color) return
    if (!isAliveAt(track, style.frame)) {
      // La couleur du slot reste la PORTE (une vie sans couleur ne se dessine pas), mais la
      // croix ne la porte plus : elle est rouge (cf. `deathInk`).
      drawDeathMark(ctx, track, view, style)
      return
    }
    drawLivingTrack(ctx, track, view, style, color)
  })
  ctx.globalAlpha = 1
}

/**
 * drawDeathMark marque l'endroit de la dernière position transmise pendant `timing.death`,
 * puis plus rien : le joueur a disparu jusqu'à sa réapparition, qui est une AUTRE vie.
 */
function drawDeathMark(
  ctx: CanvasRenderingContext2D,
  track: ReplayTrackReady,
  view: CanvasView,
  style: MarkerStyle,
): void {
  const age = style.frame - trackWindow(track).end
  if (age < 0 || age > style.timing.death) return
  const last = track.points[track.points.length - 1]
  if (!last) return
  const c = project(last, view)
  const fade = 1 - age / style.timing.death
  // LA CROIX NE GRANDIT PAS (planche du 2026-08-16) : une croix qui enfle attire l'œil sur un
  // événement déjà passé. Elle garde sa taille et s'efface.
  const r = DEATH_RADIUS * style.k
  // TOUJOURS ROUGE (A1, 2026-08-18) : la couleur d'équipe du défunt ne passe plus ici.
  ctx.strokeStyle = style.deathInk
  ctx.globalAlpha = DEATH_ALPHA * fade
  ctx.lineWidth = DEATH_WIDTH * style.k
  ctx.beginPath()
  ctx.moveTo(c.x - r, c.y - r)
  ctx.lineTo(c.x + r, c.y + r)
  ctx.moveTo(c.x + r, c.y - r)
  ctx.lineTo(c.x - r, c.y + r)
  ctx.stroke()
  ctx.globalAlpha = 1
}

/** drawLivingTrack : traînée, cône, apparition, marqueur d'étage. */
function drawLivingTrack(
  ctx: CanvasRenderingContext2D,
  track: ReplayTrackReady,
  view: CanvasView,
  style: MarkerStyle,
  color: string,
): void {
  const head = positionAt(track.points, style.frame)
  if (!head) return
  const c = project(head, view)
  const fl = floorIndex(track, style)

  if (style.showTrail) drawTrail(ctx, track, view, style, color)
  if (style.showAim) drawAimCone(ctx, track, c, style, color)
  drawSpawnRing(ctx, track, c, style, color)
  const shape = shapeOfMark(style.markOfSlot(track.slot))
  drawMarker(ctx, c, style, color, fl, shape)
  // LE NOM SOUS LE POINT, jamais à côté d'une croix de mort : la ligne ci-dessus n'est
  // atteinte que pour une vie EN COURS (drawDeathMark rend avant).
  const name = style.showNames ? style.nameOfSlot(track.slot) : null
  if (name) drawNameLabel(ctx, c, name, style, color, markerEdge(fl, style.k, shape))
}

/**
 * markerEdge : distance du centre au bord EXTERNE de tout ce que le marqueur dessine, en
 * pixels d'écran — c'est SOUS ce bord que le nom se pose (replayLabels.ts).
 *
 * Le bord n'est pas toujours celui du liseré : au-dessus du sol, l'anneau d'étage est posé à
 * un rayon FIXE (6,5) qui dépasse le disque, et le joueur de la page porte en plus son anneau
 * d'identité. Prendre le maximum des trois est la seule façon d'être sûr que l'étiquette ne
 * chevauche aucun trait.
 */
function markerEdge(fl: number, k: number, shape: MarkerShape): number {
  const outline = CORE_RADIUS + CORE_PER_FLOOR * fl + OUTLINE_PAD
  const ring = fl > 0 ? ringRadius(fl) + RING_WIDTH / 2 : 0
  const self = shape === 'ring' ? selfRingRadius2(fl) + SELF_HALO_PAD : 0
  return Math.max(outline, ring, self) * k
}

/** ringRadius : rayon de l'anneau d'étage n° `r` (1 = le premier au-dessus du sol). */
function ringRadius(r: number): number {
  return RING_RADIUS + RING_GAP * (r - 1)
}

/** selfRingRadius : rayon du PREMIER anneau d'identité du joueur de la page (forme 'ring'). */
function selfRingRadius(fl: number): number {
  return CORE_RADIUS + CORE_PER_FLOOR * fl + OUTLINE_PAD + SELF_RING_GAP
}

/** selfRingRadius2 : rayon du SECOND anneau — c'est lui qui fait le « double contour ». */
function selfRingRadius2(fl: number): number {
  return selfRingRadius(fl) + SELF_RING_GAP_2
}

/**
 * drawTrail : la polyligne des 7 s écoulées, dessinée SEGMENT PAR SEGMENT.
 *
 * Un seul `stroke` à opacité constante coûterait moins cher, mais il ne dirait pas le SENS du
 * déplacement. L'opacité monte linéairement de la queue (0,08 — presque effacé) vers la tête
 * (0,63) : la trace la plus visible est toujours celle de l'instant.
 */
function drawTrail(
  ctx: CanvasRenderingContext2D,
  track: ReplayTrackReady,
  view: CanvasView,
  style: MarkerStyle,
  color: string,
): void {
  const trail = trailAt(track.points, style.frame, style.timing.trail)
  if (trail.length < 2) return
  const segments = trail.length - 1
  const span = TRAIL_ALPHA_HEAD - TRAIL_ALPHA_TAIL
  ctx.strokeStyle = color
  ctx.lineWidth = TRAIL_WIDTH * style.k
  let from = project(trail[0], view)
  for (let i = 1; i < trail.length; i++) {
    const to = project(trail[i], view)
    ctx.globalAlpha = TRAIL_ALPHA_TAIL + (span * i) / segments
    ctx.beginPath()
    ctx.moveTo(from.x, from.y)
    ctx.lineTo(to.x, to.y)
    ctx.stroke()
    from = to
  }
  ctx.globalAlpha = 1
}

/** drawSpawnRing : l'anneau qui s'ouvre au premier instant de la vie. */
function drawSpawnRing(
  ctx: CanvasRenderingContext2D,
  track: ReplayTrackReady,
  c: XY,
  style: MarkerStyle,
  color: string,
): void {
  const age = style.frame - trackWindow(track).start
  if (age < 0 || age > style.timing.spawn) return
  const f = 1 - age / style.timing.spawn
  ctx.beginPath()
  ctx.arc(c.x, c.y, (SPAWN_RADIUS + SPAWN_GROWTH * (1 - f)) * style.k, 0, Math.PI * 2)
  ctx.strokeStyle = color
  ctx.globalAlpha = SPAWN_ALPHA * f
  ctx.lineWidth = SPAWN_WIDTH * style.k
  ctx.stroke()
  ctx.globalAlpha = 1
}

/**
 * drawMarker dessine le joueur : anneaux d'étage, liseré, noyau — et sa FORME dit son identité
 * (disque, losange pour un ami, disque cerclé pour le joueur de la page).
 *
 * IL N'Y A PLUS DE HALO (planche du 2026-08-16) : la lueur diffuse sous chaque point doublait
 * l'emprise du marqueur et empâtait la carte dès qu'un nom s'écrivait dessous. Le liseré
 * sombre suffit à détacher un point coloré d'un fond qui va du blanc au noir.
 *
 * L'ÉTAGE SE LIT PAR DES ANNEAUX CONCENTRIQUES, jamais par un décalage du marqueur : en vue de
 * dessus, déplacer un point vers le haut de l'écran voudrait dire « plus au nord » et
 * fausserait la position. L'altitude est un palier, pas un dégradé — l'histogramme des z montre
 * des pics nets, la carte a trois niveaux de jeu.
 *
 * L'ANNEAU D'ÉTAGE PREND LA COULEUR DU PION depuis le 2026-08-18 (A1 : « le cercle d'altitude à
 * la couleur du pion du joueur »). Il était à l'encre du thème, pour ne pas faire dire deux
 * choses à la couleur ; à l'écran, cette neutralité le détachait de son point et, sur une carte
 * à trois étages peuplée, on lisait des anneaux orphelins. La couleur ne dit toujours que le
 * camp — c'est le NOMBRE d'anneaux qui dit la hauteur, et lui seul.
 *
 * LE LOSANGE A LE MÊME RAYON CIRCONSCRIT QUE LE DISQUE : la forme change, la taille non —
 * sinon un ami paraîtrait plus proche ou plus gros que ses coéquipiers.
 */
function drawMarker(
  ctx: CanvasRenderingContext2D,
  c: XY,
  style: MarkerStyle,
  color: string,
  fl: number,
  shape: MarkerShape,
): void {
  const core = (CORE_RADIUS + CORE_PER_FLOOR * fl) * style.k

  // LE HALO EN PREMIER, sous tout le reste : c'est une lueur, pas un trait.
  if (shape === 'ring') drawSelfHalo(ctx, c, style, fl)

  ctx.strokeStyle = color
  ctx.lineWidth = RING_WIDTH * style.k
  for (let r = 1; r <= fl; r++) {
    ctx.globalAlpha = RING_ALPHA - RING_ALPHA_DECAY * (r - 1)
    ctx.beginPath()
    ctx.arc(c.x, c.y, ringRadius(r) * style.k, 0, Math.PI * 2)
    ctx.stroke()
  }

  ctx.globalAlpha = OUTLINE_ALPHA
  ctx.fillStyle = style.ink
  corePath(ctx, c, core + OUTLINE_PAD * style.k, shape)
  ctx.fill()

  ctx.globalAlpha = 1
  ctx.fillStyle = color
  corePath(ctx, c, core, shape)
  ctx.fill()

  if (shape === 'ring') drawSelfRings(ctx, c, style, fl)
}

/**
 * drawSelfRings — LES DEUX ANNEAUX du joueur de la page, à `selfInk`.
 *
 * DEUX, PAS UN : c'est le double contour qui porte l'identité quand la couleur ne peut pas
 * (cf. la note de SELF_RING_WIDTH). Le second est plus fin et plus pâle — l'œil lit « un
 * anneau souligné », pas « deux cercles ».
 */
function drawSelfRings(
  ctx: CanvasRenderingContext2D,
  c: XY,
  style: MarkerStyle,
  fl: number,
): void {
  ctx.strokeStyle = style.selfInk
  ctx.globalAlpha = 1
  ctx.lineWidth = SELF_RING_WIDTH * style.k
  ctx.beginPath()
  ctx.arc(c.x, c.y, selfRingRadius(fl) * style.k, 0, Math.PI * 2)
  ctx.stroke()
  ctx.globalAlpha = 0.75
  ctx.lineWidth = (SELF_RING_WIDTH / 2) * style.k
  ctx.beginPath()
  ctx.arc(c.x, c.y, selfRingRadius2(fl) * style.k, 0, Math.PI * 2)
  ctx.stroke()
  ctx.globalAlpha = 1
}

/** drawSelfHalo — la lueur diffuse sous le marqueur du joueur de la page (dégradé radial). */
function drawSelfHalo(
  ctx: CanvasRenderingContext2D,
  c: XY,
  style: MarkerStyle,
  fl: number,
): void {
  const r = (selfRingRadius2(fl) + SELF_HALO_PAD) * style.k
  const gradient = ctx.createRadialGradient(c.x, c.y, 0, c.x, c.y, r)
  gradient.addColorStop(0, style.selfInk)
  gradient.addColorStop(1, 'transparent')
  ctx.globalAlpha = SELF_HALO_ALPHA
  ctx.fillStyle = gradient
  ctx.beginPath()
  ctx.arc(c.x, c.y, r, 0, Math.PI * 2)
  ctx.fill()
  ctx.globalAlpha = 1
}

/** corePath trace le noyau (ou son liseré) : cercle, ou losange de même rayon circonscrit. */
function corePath(ctx: CanvasRenderingContext2D, c: XY, r: number, shape: MarkerShape): void {
  ctx.beginPath()
  if (shape !== 'diamond') {
    ctx.arc(c.x, c.y, r, 0, Math.PI * 2)
    return
  }
  ctx.moveTo(c.x, c.y - r)
  ctx.lineTo(c.x + r, c.y)
  ctx.lineTo(c.x, c.y + r)
  ctx.lineTo(c.x - r, c.y)
  ctx.lineTo(c.x, c.y - r)
  ctx.closePath()
}

function project(p: XY, view: CanvasView): XY {
  return worldToCanvas(p, view.bounds, view.width, view.height, view.pad)
}

function floorIndex(track: ReplayTrackReady, style: MarkerStyle): number {
  const z = altitudeAt(track.points, style.frame)
  return z === null ? 0 : floorOf(z, style.z.min, style.z.max)
}
