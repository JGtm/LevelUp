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
import { drawNameLabel } from './replayLabels'
import type { ReplayProjectileReady, ReplayTrackReady } from './replayNormalize'

import {
  altitudeAt,
  floorOf,
  freshness,
  heldReading,
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
 * de jeu), mort marquée 1,5 s, apparition 0,8 s.
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
/** Anneau externe du joueur DE LA PAGE (forme 'ring') : le seul trait qui cercle un marqueur. */
const SELF_RING_WIDTH = 1.5
const SELF_RING_GAP = 1.6

const AIM_LENGTH = 52
const AIM_HALF_ANGLE = 0.42
const AIM_CONE_ALPHA = 0.55
/** Une visée de 5 s ne vaut pas une visée de l'instant : elle perd 62 % de son opacité. */
const AIM_FADE = 0.62
/**
 * L'ÉLÉVATION DE VISÉE (schéma 13) : le cône garde son ANGLE et raccourcit.
 *
 * `AIM_LENGTH` reste la longueur MAXIMALE — celle d'un joueur qui vise à plat. Le facteur est
 * `cos(p)`, la part horizontale d'un regard incliné, borné en bas pour que le marqueur reste
 * lisible quand la visée est verticale.
 *
 * Ce que la mesure dit du champ (lot E, 3 films) : médiane −4,7 / −3,4 / −3,6°, 67 à 77 % des
 * visées vers le BAS, extrêmes −85,5 à +82°. Le cône passe donc son temps à 99,5 % de sa
 * longueur, et se contracte franchement dans les instants qui comptent — un tir en plongée,
 * un joueur qui couvre une passerelle.
 */
const AIM_PITCH_FLOOR = 0.35
/** Trait de sens, à la pointe du cône : vers l'extérieur = vers le haut, intérieur = vers le bas. */
const AIM_TICK_LENGTH = 6
const AIM_TICK_WIDTH = 1.4
/** Zone morte du trait : sous 2°, la visée se lit à plat et le cône n'a pas bougé (cos 2° = 0,9994). */
const AIM_TICK_DEAD_DEG = 2

/** Croix de mort : demi-taille FIXE qui s'estompe (elle ne grandit plus, cf. §1bis). */
const DEATH_RADIUS = 5
const DEATH_WIDTH = 1.6
const DEATH_ALPHA = 0.9
const SPAWN_RADIUS = 2
const SPAWN_GROWTH = 12
const SPAWN_WIDTH = 1.2
const SPAWN_ALPHA = 0.8

const PROJECTILE_ALPHA = 0.5
const PROJECTILE_WIDTH = 1.2
/** Le vol reste visible brièvement après son dernier point répliqué, puis s'efface. */
const PROJECTILE_TAIL_FRAMES = 7

/**
 * Style du calque : ce qu'une VIE emprunte à son PROPRIÉTAIRE, plus les encres du thème.
 *
 * TOUT SE LIT PAR SLOT, jamais par index de trace. Le slot est la clé d'une vie, et le
 * propriétaire de la vie porte sa couleur d'équipe, sa marque d'identité et son nom
 * (rosterLogic.ts). Une vie sans propriétaire rend `null` / `undefined` : le calque la
 * dessine quand même — elle existe — mais sans nom ni marque.
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
 * viennent de se fermer (la croix survit 1,5 s à la vie qu'elle termine).
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
      drawDeathMark(ctx, track, view, style, color)
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
  color: string,
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
  ctx.strokeStyle = color
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

  drawTrail(ctx, track, view, style, color)
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
  const self = shape === 'ring' ? selfRingRadius(fl) + SELF_RING_WIDTH / 2 : 0
  return Math.max(outline, ring, self) * k
}

/** ringRadius : rayon de l'anneau d'étage n° `r` (1 = le premier au-dessus du sol). */
function ringRadius(r: number): number {
  return RING_RADIUS + RING_GAP * (r - 1)
}

/** selfRingRadius : rayon de l'anneau d'identité du joueur de la page (forme 'ring'). */
function selfRingRadius(fl: number): number {
  return CORE_RADIUS + CORE_PER_FLOOR * fl + OUTLINE_PAD + SELF_RING_GAP
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

/**
 * drawAimCone dessine la DIRECTION DU REGARD, décodée du même record que la position.
 *
 * Le cône se dégrade du centre vers le bord — dense à l'origine, où il faut lire QUI vise,
 * transparent au bout, où il ne faut pas masquer le décor. Il pâlit avec l'âge de la mesure et
 * n'est PAS dessiné au-delà du maintien : passé ce délai, on ne sait plus où le joueur regarde,
 * et une direction périmée affirmerait ce qu'on ignore.
 *
 * IL N'Y A PLUS D'AXE (décision D3, 2026-08-16) : les deux traits qui prolongeaient le point
 * — « le bâton » — ont été supprimés à la demande de l'utilisateur. Ce que le cône seul perd
 * en précision d'angle, la carte le regagne en lisibilité : huit bâtons sur un 4v4 se
 * croisaient au-dessus des noms.
 *
 * SES DIMENSIONS SONT CELLES DE LA PLANCHE, « un peu plus prononcées » (verdict du soir du
 * 2026-08-16, §1bis du plan) : rayon 52, demi-ouverture 0,42 rad, alpha 0,55. Le cône avait
 * d'abord été rétréci à 30 px / 0,30 — trop timide pour se lire une fois les noms posés.
 *
 * DEPUIS LE SCHÉMA 13, IL DIT LES DEUX AXES DU REGARD. Le cap oriente le secteur ; l'ÉLÉVATION
 * (`p`, degrés, positif = vers le haut) le RACCOURCIT — `AIM_LENGTH × max(0,35 ; cos p)` — et
 * un trait posé à sa pointe dit de quel côté. Il faut les deux : le cosinus est pair, donc la
 * longueur seule confond « vise le ciel » et « vise ses pieds ». Un artefact antérieur au
 * schéma 13 ne porte pas `p` : le cône y garde sa pleine longueur, sans tick, et c'est le
 * comportement voulu — absent se lit « à plat », jamais « inconnu ».
 */
function drawAimCone(
  ctx: CanvasRenderingContext2D,
  track: ReplayTrackReady,
  c: XY,
  style: MarkerStyle,
  color: string,
): void {
  const read = heldReading(track.points, style.frame, (p) => p.h, style.timing.aimHold)
  if (!read) return
  const fresh = freshness(read.age, style.timing.aimHold, AIM_FADE)
  // Monde -> canevas : l'axe Y est inversé, donc l'angle l'est aussi.
  const ang = (-read.value * Math.PI) / 180
  const pitch = heldPitch(track.points, style, read.age)
  const R = AIM_LENGTH * pitchScale(pitch) * style.k
  const gradient = ctx.createRadialGradient(c.x, c.y, 0, c.x, c.y, R)
  gradient.addColorStop(0, color)
  gradient.addColorStop(1, 'transparent')
  ctx.globalAlpha = AIM_CONE_ALPHA * fresh
  ctx.beginPath()
  ctx.moveTo(c.x, c.y)
  ctx.arc(c.x, c.y, R, ang - AIM_HALF_ANGLE, ang + AIM_HALF_ANGLE)
  ctx.closePath()
  ctx.fillStyle = gradient
  ctx.fill()
  drawPitchTick(ctx, c, ang, R, pitch, style, color)
  ctx.globalAlpha = 1
}

/**
 * heldPitch rend l'ÉLÉVATION en vigueur, ou 0 (à plat).
 *
 * LA RÈGLE D'ÂGE N'EST PAS UNE PRÉCAUTION DÉCORATIVE. `p` est omis quand la visée s'arrondit
 * à plat (contrat du champ, cf. `Point.P` côté Go), et `heldReading` remonterait alors
 * jusqu'à un point PLUS ANCIEN qui, lui, porte une élévation : le marqueur afficherait une
 * plongée périmée sur une visée à plat actuelle. Les deux angles venant du MÊME
 * enregistrement, une élévation trouvée plus loin dans le passé que le cap appartient
 * forcément à une autre visée — on la refuse, et « absent » redevient ce qu'il doit être.
 */
function heldPitch(
  points: ReplayTrackReady['points'],
  style: MarkerStyle,
  headingAge: number,
): number {
  const read = heldReading(points, style.frame, (p) => p.p, style.timing.aimHold)
  return read && read.age <= headingAge ? read.value : 0
}

/**
 * pitchScale : ce que l'élévation fait à la LONGUEUR du cône.
 *
 * Le cône est la projection au sol d'un regard qui, lui, vit en trois dimensions. Plus le
 * joueur pique ou lève la tête, moins ce regard porte LOIN SUR LE PLAN — d'où le cosinus, qui
 * est exactement la part horizontale d'une direction inclinée. L'ANGLE, lui, ne bouge pas :
 * c'est toujours le même cap.
 *
 * LE PLANCHER EXISTE POUR QUE LE MARQUEUR RESTE LISIBLE : à ±90° le cosinus s'annule et le
 * cône disparaîtrait, alors qu'un joueur qui vise ses pieds ou le ciel est précisément ce
 * qu'on veut voir. On s'arrête donc à 35 % de la longueur maximale.
 */
function pitchScale(pitchDeg: number): number {
  return Math.max(AIM_PITCH_FLOOR, Math.cos((pitchDeg * Math.PI) / 180))
}

/**
 * drawPitchTick dit le SENS de l'élévation, que la longueur ne peut pas dire.
 *
 * Le cosinus est PAIR : viser 30° au-dessus et 30° en dessous raccourcissent le cône
 * exactement pareil. Un repère de sens est donc nécessaire, et c'est un trait court posé sur
 * l'axe du regard, au BORD du cône — vers l'EXTÉRIEUR quand le joueur lève la tête, vers
 * l'INTÉRIEUR quand il pique. Il ne part JAMAIS du point : « le bâton » reste supprimé
 * (décision D3), et ce trait-ci vit à la pointe du cône, à des dizaines de pixels de là.
 *
 * LA ZONE MORTE (2°) N'EST PAS UN ARRONDI DE CONFORT : sous 2° le cône perd 0,06 % de sa
 * longueur, donc l'œil ne peut RIEN vérifier de ce que le trait affirmerait, et l'affirmation
 * changerait de sens à chaque image. Une visée est à plat quand elle se lit à plat.
 */
function drawPitchTick(
  ctx: CanvasRenderingContext2D,
  c: XY,
  ang: number,
  R: number,
  pitchDeg: number,
  style: MarkerStyle,
  color: string,
): void {
  if (Math.abs(pitchDeg) < AIM_TICK_DEAD_DEG) return
  const len = AIM_TICK_LENGTH * style.k
  const far = R + (pitchDeg > 0 ? len : -len)
  ctx.beginPath()
  ctx.moveTo(c.x + Math.cos(ang) * R, c.y + Math.sin(ang) * R)
  ctx.lineTo(c.x + Math.cos(ang) * far, c.y + Math.sin(ang) * far)
  ctx.strokeStyle = color
  ctx.lineWidth = AIM_TICK_WIDTH * style.k
  ctx.stroke()
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
 * des pics nets, la carte a trois niveaux de jeu. L'anneau est tracé à l'ENCRE DU THÈME et non
 * à la couleur du joueur : la couleur dit le camp, elle ne doit pas aussi dire la hauteur.
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

  ctx.strokeStyle = style.ink
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

  if (shape === 'ring') {
    // MOI : un anneau externe à l'encre du thème — le seul marqueur cerclé de la carte.
    ctx.strokeStyle = style.ink
    ctx.lineWidth = SELF_RING_WIDTH * style.k
    ctx.beginPath()
    ctx.arc(c.x, c.y, selfRingRadius(fl) * style.k, 0, Math.PI * 2)
    ctx.stroke()
  }
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

/**
 * drawProjectilesLayer dessine les vols de projectile en cours.
 *
 * LE DERNIER POINT N'EST PAS UN IMPACT : le film ne porte aucun événement de détonation. C'est
 * la dernière position RÉPLIQUÉE — pour une grenade, la réplication cesse ~1,4 s après le
 * lancer alors que la mèche court jusqu'à ~3 s. Le vol s'efface donc, il n'explose pas.
 */
export function drawProjectilesLayer(
  ctx: CanvasRenderingContext2D,
  projectiles: ReplayProjectileReady[],
  view: CanvasView,
  frame: number,
  color: string,
): void {
  ctx.strokeStyle = color
  ctx.lineWidth = PROJECTILE_WIDTH
  for (const pr of projectiles) {
    const pts = pr.p
    if (pts.length < 2) continue
    const end = pr.t0 + pts[pts.length - 1][0]
    if (frame < pr.t0 || frame > end + PROJECTILE_TAIL_FRAMES) continue
    const fade = frame > end ? 1 - (frame - end) / PROJECTILE_TAIL_FRAMES : 1
    ctx.globalAlpha = PROJECTILE_ALPHA * fade
    ctx.beginPath()
    let started = false
    for (const [dt, x, y] of pts) {
      if (pr.t0 + dt > frame) break
      const c = project({ x, y }, view)
      if (!started) {
        ctx.moveTo(c.x, c.y)
        started = true
      } else ctx.lineTo(c.x, c.y)
    }
    if (started) ctx.stroke()
  }
  ctx.globalAlpha = 1
}

function project(p: XY, view: CanvasView): XY {
  return worldToCanvas(p, view.bounds, view.width, view.height, view.pad)
}

function floorIndex(track: ReplayTrackReady, style: MarkerStyle): number {
  const z = altitudeAt(track.points, style.frame)
  return z === null ? 0 : floorOf(z, style.z.min, style.z.max)
}
