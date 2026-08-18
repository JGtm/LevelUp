/**
 * equipmentPlacementsLayer.ts — LES POSES D'ÉQUIPEMENT sur la carte : ce qui se dessine, et
 * surtout ce qui ne se dessine PAS.
 *
 * LA SOURCE est le document (schéma 10) : `equipmentPlacements`, une pose par vie d'objet, avec
 * sa fenêtre [t0, t1] sur l'axe des frames, sa position monde, la FAMILLE que le manifeste du
 * titre lui donne (`replay_labels.toml`), l'IDENTIFIANT du tag `eqip`, le SLOT de son poseur
 * (`owner`, -1 si aucun bipède contemporain à moins de 3 m), le CAP de visée de ce poseur (`h`,
 * absent une fois sur sept) et — depuis le schéma 10 — l'ORIGINE MESURÉE de la pose.
 *
 * `equipmentPlacements` N'EST PAS CE QUE SON NOM DIT, ET C'EST LA MESURE QUI L'A ÉTABLI (phase G
 * du 2026-08-18, 11 films calibrés, 3 661 poses à poseur mesuré). 88,6 % de ces poses naissent
 * dans les 2 frames et les 1,5 m qui suivent le DERNIER point de leur poseur : ce ne sont pas
 * des objets posés sur la carte, ce sont les objets que le joueur PORTAIT, relâchés quand il
 * MEURT. D'où le filtre `origin === 'deployed'` : la seule origine qui décrive un geste.
 * Le témoin qui valide la méthode est interne — si les 88,6 % étaient un artefact de fenêtre,
 * les PANNEAUX du mur les porteraient aussi ; ils en ont zéro sur 48 et un sur 43.
 *
 * LE RENDU EST UNE TABLE PAR FAMILLE (`PLACEMENT_RENDER`), et c'est la règle qui compte : une
 * famille absente de la table ne dessine RIEN, une famille à `null` ne dessine rien NON PLUS
 * mais l'a décidé. Jamais le dessin d'une voisine, même principe que les libellés et les sons.
 *
 * CE FICHIER DÉCIDE, IL NE TRACE PRESQUE PLUS. Le MUR (géométrie monde + identifiants des
 * panneaux) vit dans `placementWall.ts` ; les formes qui tiennent dans un centre projeté
 * (balise, impulsion du traqueur, disque du champ, point neutre, marque de révélation) vivent
 * dans `placementShapes.ts` avec le socle du cadrage. Reste ici le CAPTEUR — sa portée est en
 * mètres et il est le seul à révéler — et l'aiguillage.
 *
 * LE CAPTEUR PINGE, et tout ce qu'il affirme — portée, cadence, durée de révélation, camp
 * révélé — vit dans `threatSensor.ts` : les chiffres y sont OFFICIELS et sourcés, la logique y
 * est pure et testée. Ce fichier n'en garde que le TRACÉ (disque, anneau, onde).
 *
 * Pas de React : géométrie pure + un CanvasRenderingContext2D (même règle que replayDraw).
 */
import type { ReplayEquipmentPlacement } from '@/lib/api/types'

import {
  BEACON_RADIUS_PX,
  drawBeacon,
  drawRepairField,
  drawRevealMark,
  drawSeekerImpulse,
  drawUnnamedDot,
  type PlacementView,
  project,
  REPAIR_FIELD_RADIUS_M,
  SEEKER_IMPULSE_RADIUS_PX,
  UNNAMED_DOT_RADIUS_PX,
  viewScale,
} from './placementShapes'
import { drawWall, wallHeading, WALL_PANEL_IDS, wallRadiusM } from './placementWall'
// La FENÊTRE d'une pose vit à part (cf. le réexport plus haut) : elle est aussi LUE ici, et un
// réexport ne met rien dans la portée locale. L'import inverse, lui, ne porte que des TYPES —
// il n'y a donc aucun cycle à l'exécution.
import { isPlacementActive, placementShows } from './placementWindow'
import type { XY } from './replayLogic'
import type { ReplayTrackReady } from './replayNormalize'
import {
  SENSOR_FILL_ALPHA,
  SENSOR_PING_WIDTH,
  SENSOR_RADIUS_M,
  SENSOR_RING_ALPHA,
  SENSOR_RING_WIDTH,
  sensorPing,
  sensorReveals,
} from './threatSensor'

/**
 * LA FENÊTRE D'UNE POSE vit dans `placementWindow.ts` depuis le 2026-08-18 (seuil de taille),
 * et se réexporte ici : le survol et les tests la lisent par ce module, la découpe interne n'a
 * pas à remonter jusqu'à eux.
 */
export { isPlacementActive, placementEndFrame, placementShows } from './placementWindow'

/**
 * Le cadrage est réexporté ici : il est un ARGUMENT de ce calque (`drawEquipmentPlacementsLayer`,
 * `placementAt`), donc il appartient à sa surface publique — ses appelants n'ont pas à connaître
 * le fichier où la découpe interne l'a rangé.
 */
export type { PlacementView } from './placementShapes'

/**
 * Les façons de dessiner une pose. Ce ne sont PAS les familles : le jour où deux familles
 * partageront un rendu (deux murs de palettes différentes, par exemple), la table les enverra
 * toutes deux sur la même valeur sans qu'on duplique un tracé.
 */
export type PlacementKind = 'wall' | 'sensor' | 'beacon' | 'seeker' | 'field' | 'unnamed'

/** Famille du MUR — nommée parce que la règle des panneaux la vise explicitement. */
const PLACEMENT_FAMILY_WALL = 'wall'

/**
 * LA TABLE PAR FAMILLE — l'unique porte d'entrée du rendu. Les clés sont les identifiants
 * STABLES publiés par le document (`family`, posés par le manifeste du titre), jamais des
 * libellés.
 *
 * TROIS ÉTATS, ET LEUR DIFFÉRENCE EST DU SENS, pas du code :
 *  - une RÈGLE DE RENDU : la famille se dessine, chacune avec sa forme ;
 *  - `null` : la famille est CONNUE et la décision est de ne rien dessiner — voir les objets
 *    portés ci-dessous ;
 *  - ABSENTE : on n'a rien décidé, et le calque se tait. C'est le cas des familles qu'un futur
 *    manifeste ajouterait, et celui — assumé — des power-ups (voir la note en fin de table).
 *
 * ELLE EST ÉNUMÉRÉE AILLEURS, et un garde-rail le vérifie : `equipmentFamilies` du valideur Go
 * (`internal/games/mappings/loader_replay_labels_equipment.go`) et `placementFamily` de l'i18n
 * doivent coïncider avec elle — cf. `placementFamily.guard.test.ts`, qui LIT le fichier Go.
 */
export const PLACEMENT_RENDER: Readonly<Record<string, PlacementKind | null>> = {
  // Les objets que la mesure voit DÉPLOYÉS, chacun avec sa forme propre.
  wall: 'wall',
  sensor: 'sensor',
  translocator_beacon: 'beacon',
  threat_seeker: 'seeker',
  repair_field: 'field',
  // L'objet posé dont la nature n'est pas établie : un point neutre, et seulement sur bascule.
  other: 'unnamed',
  // LES OBJETS QUE LE JOUEUR PORTE — `null` EXPLICITE, ET C'EST UNE DÉCISION MESURÉE.
  //
  // Les quatre grenades font 59 à 63 % des poses du corpus, et 95 % d'entre elles sont des
  // LÂCHERS À LA MORT (`dropped`). Les quelques `deployed` qui restent sont des grenades
  // lancées — or les lancers ont déjà leur calque, avec leur trajectoire et leur type. Le
  // grappin, le propulseur et le répulseur sont des capacités qui agissent SUR LEUR PORTEUR :
  // ce que le film en publie ici est l'appareil lui-même, pas un objet posé sur le terrain (le
  // grappin a d'ailleurs son propre calque de traction). Les dessiner noierait le mur et le
  // capteur sous des points qui ne disent rien du terrain.
  grenade_frag: null,
  grenade_plasma: null,
  grenade_dynamo: null,
  grenade_spike: null,
  grapple: null,
  thruster: null,
  repulsor: null,
  // POWER-UPS (`powerup_overshield`, `powerup_camo`) : DÉLIBÉRÉMENT ABSENTS, PAS MÊME À `null`.
  //
  // Le plan prévoyait de les dessiner comme des objets de la CARTE, sans poseur, à leur socle.
  // LA MESURE A RÉFUTÉ L'HYPOTHÈSE SUR LE PEU QU'ON A : le corpus entier (11 films) porte UNE
  // pose de surbouclier et UNE de camouflage, et les DEUX ont un poseur et sont `dropped` (23 et
  // 38 ms, 0,53 et 0,67 m de la fin de vie de leur poseur). Aucune position récurrente n'a pu
  // être cherchée. Coder une règle dont aucun membre mesuré ne relève serait du vocabulaire
  // mort (CLAUDE.md n°7) : la famille entrera dans cette table le jour où un film portera de
  // vrais power-ups de socle, et pas avant.
}

/**
 * L'ORIGINE QUI SE DESSINE. Identifiant STABLE du document, jamais un libellé.
 *
 * Le vocabulaire publié est `deployed` / `dropped` / `unknown` — `spawn` n'existe pas : le
 * critère « dotation reçue au spawn » que le plan avait écrit avant la mesure compte 4 poses sur
 * 3 661 (0,1 %), et les quatre sont des vies de 0,13 s et 1,49 s où début et fin se confondent.
 */
export const PLACEMENT_ORIGIN_DEPLOYED = 'deployed'

/** Origine non établie : aucun poseur mesuré, ou artefact antérieur au schéma 10. */
export const PLACEMENT_ORIGIN_UNKNOWN = 'unknown'

/**
 * placementOrigin — l'origine d'une pose, telle qu'on a le droit de la lire.
 *
 * LE CHAMP EST OPTIONNEL AU CONTRAT, ET SON ABSENCE SE LIT `unknown` — JAMAIS `deployed`. Les
 * artefacts antérieurs au schéma 10 sont encore sur disque et en production jusqu'à la
 * re-cuisson complète : un repli sur `deployed` ferait dessiner, sur tout ce parc, exactement le
 * mur fantôme que ce filtre existe pour supprimer.
 */
export function placementOrigin(p: ReplayEquipmentPlacement): string {
  return p.origin ?? PLACEMENT_ORIGIN_UNKNOWN
}

/**
 * placementIsDeployedObject — cette pose est-elle l'objet DÉPLOYÉ qu'on a le droit de montrer ?
 *
 * DEUX CONDITIONS, TOUTES DEUX MESURÉES : l'origine est un déploiement, et — pour le mur
 * seulement — l'identifiant est celui des PANNEAUX (cf. WALL_PANEL_IDS : un mur déployé produit
 * deux poses, l'appareil qui vole et ses panneaux, et les dessiner toutes deux ferait deux arcs
 * pour un seul mur).
 *
 * EXPORTÉE PARCE QUE LE SON LIT LA MÊME RÈGLE (`replaySound.ts`) : le geste de pose qui sonne
 * est le même que celui qui se dessine. Deux prédicats divergeraient au premier changement.
 */
export function placementIsDeployedObject(p: ReplayEquipmentPlacement): boolean {
  if (placementOrigin(p) !== PLACEMENT_ORIGIN_DEPLOYED) return false
  if (p.family !== PLACEMENT_FAMILY_WALL) return true
  return WALL_PANEL_IDS.includes(p.id)
}

/** Rayon minimal de la zone sensible au survol, en pixels : un point de 2,5 px ne se vise pas. */
const HOVER_MIN_RADIUS_PX = 9

/**
 * Ce que le calque a besoin de savoir de l'instant courant. `frameMs` est la durée RÉELLE
 * d'une frame (cf. `frameToMs(1, doc)`) : c'est elle qui donne son âge à la pulsation, jamais
 * un compteur d'images — la cadence d'échantillonnage est choisie au build.
 */
export interface PlacementTime {
  frame: number
  frameMs: number
  /** Nombre total d'images du rejeu (`doc.frameCount`) : la borne des poses sans fin connue. */
  frames: number
  /** Densité de pixels : les épaisseurs d'écran la suivent (même règle que les marqueurs). */
  k: number
  reducedMotion: boolean
  /** Les objets non identifiés se dessinent-ils ? Bascule du tiroir, ÉTEINTE par défaut. */
  showUnnamed: boolean
}

/** Ce qu'il faut connaître du temps pour borner une pose — rien de plus (règle des 5 params). */
export type PlacementWindowTime = Pick<PlacementTime, 'frameMs' | 'frames'>

/** Ce que le SURVOL a besoin de savoir de l'instant : l'image, sa fenêtre, et la bascule. */
export type PlacementHoverTime = Pick<
  PlacementTime,
  'frame' | 'frameMs' | 'frames' | 'showUnnamed'
>

/**
 * Ce que le calque a besoin de LIRE : les poses, et — pour le seul capteur de menaces — les
 * vies et leur camp.
 *
 * POURQUOI LES TROIS VOYAGENT ENSEMBLE. Le capteur ne se dessine pas seul : son ping RÉVÈLE
 * des adversaires, et « adversaire » est une relation entre deux vies. Les grouper garde
 * `drawEquipmentPlacementsLayer` à cinq paramètres (seuil CLAUDE.md n°5) et dit ce qu'ils sont :
 * une SCÈNE, pas trois arguments indépendants.
 */
export interface PlacementScene {
  placements: readonly ReplayEquipmentPlacement[]
  /** Toutes les vies du film ; le balayage de révélation filtre lui-même sur leur fenêtre. */
  lives: readonly ReplayTrackReady[]
  /** Camp d'une vie (`team_side`) ; null = camp inconnu, donc jamais révélé ni révélateur. */
  sideOfSlot: (slot: number) => string | null
}

/** Les encres du calque : la couleur d'équipe du poseur, et le neutre quand il n'y en a pas. */
export interface PlacementInk {
  /** Couleur de la vie qui a posé l'objet ; null = vie inconnue, on tombe sur `neutral`. */
  colorOfSlot: (slot: number) => string | null
  /** Encre neutre du thème (`--muted-foreground`), servie aux poses sans poseur. */
  neutral: string
}

/**
 * placementKind rend la règle de rendu d'une pose, ou null quand il n'y en a pas.
 *
 * QUATRE RAISONS DE NE RIEN RENDRE, et elles se lisent dans cet ordre : famille hors table,
 * famille à `null` (objet porté), objet non identifié alors que la bascule est éteinte, et
 * enfin pose qui n'est pas un DÉPLOIEMENT.
 *
 * L'OBJET NON IDENTIFIÉ EST LE SEUL À ÉCHAPPER AU FILTRE D'ORIGINE, et c'est délibéré : sa
 * bascule est un outil de diagnostic (« un objet d'équipement est ici, sa nature n'est pas
 * établie »), et pour ce service-là l'origine n'apporte rien — on cherche à voir ce qu'on ne
 * sait pas nommer, d'où qu'il vienne. Elle reste éteinte par défaut, comportement inchangé.
 */
export function placementKind(
  p: ReplayEquipmentPlacement,
  showUnnamed: boolean,
): PlacementKind | null {
  const kind = PLACEMENT_RENDER[p.family]
  if (!kind) return null
  if (kind === 'unnamed') return showUnnamed ? kind : null
  return placementIsDeployedObject(p) ? kind : null
}

/** inkOf — la couleur d'équipe du poseur, ou le neutre quand la pose n'a pas de poseur connu. */
function inkOf(p: ReplayEquipmentPlacement, ink: PlacementInk): string {
  if (p.owner < 0) return ink.neutral
  return ink.colorOfSlot(p.owner) ?? ink.neutral
}

/**
 * drawSensor — la ZONE du capteur (disque à alpha très bas, borné par son anneau) et, quand
 * une onde est en cours, LE PING : un anneau parti du centre qui s'ouvre jusqu'au rayon en
 * s'effaçant.
 *
 * LA ZONE NE BOUGE PLUS, seule l'onde bouge : c'est ce qui distingue un capteur qui PINGE d'un
 * capteur qui respire. Le disque dit la portée officielle, à demeure ; l'onde dit l'instant.
 *
 * `ctx.arc` sur le centre PROJETÉ et le rayon à l'échelle : la projection du rejeu est une
 * similitude (même facteur en X et en Y, cf. worldToCanvas), donc un cercle monde reste un
 * cercle à l'écran. Ce qui se calcule en monde, c'est ce qui porte une ORIENTATION — le mur.
 */
function drawSensor(
  ctx: CanvasRenderingContext2D,
  p: ReplayEquipmentPlacement,
  view: PlacementView,
  time: PlacementTime,
  color: string,
): void {
  const c = project({ x: p.x, y: p.y }, view)
  const r = SENSOR_RADIUS_M * viewScale(view)
  if (!(r > 0)) return
  ctx.save()
  ctx.fillStyle = color
  ctx.strokeStyle = color
  ctx.globalAlpha = SENSOR_FILL_ALPHA
  ctx.beginPath()
  ctx.arc(c.x, c.y, r, 0, Math.PI * 2)
  ctx.fill()
  ctx.globalAlpha = SENSOR_RING_ALPHA
  ctx.lineWidth = SENSOR_RING_WIDTH * time.k
  ctx.beginPath()
  ctx.arc(c.x, c.y, r, 0, Math.PI * 2)
  ctx.stroke()
  const wave = sensorPing((time.frame - p.t0) * time.frameMs, time.reducedMotion)
  if (wave && wave.reach > 0) {
    ctx.globalAlpha = wave.alpha
    ctx.lineWidth = SENSOR_PING_WIDTH * time.k
    ctx.beginPath()
    ctx.arc(c.x, c.y, r * wave.reach, 0, Math.PI * 2)
    ctx.stroke()
  }
  ctx.restore()
}

/**
 * Ce qu'il faut pour tracer UNE pose : la pose, sa règle de rendu déjà décidée, et LES VIES.
 *
 * LES VIES SONT LÀ POUR LE SEUL MUR, et c'est assumé : depuis le 2026-08-18, un mur sans cap
 * de pose emprunte la TRAJECTOIRE de son poseur (cf. `wallHeading`). Les grouper dans un objet
 * plutôt que d'ajouter deux paramètres garde `drawPlacement` sous le seuil de la règle des
 * cinq (CLAUDE.md n°5) et évite de recalculer `placementKind`, que l'appelant connaît déjà.
 */
interface PlacementTarget {
  p: ReplayEquipmentPlacement
  kind: PlacementKind
  lives: readonly ReplayTrackReady[]
}

/**
 * drawPlacement — la forme d'UNE pose, aiguillée par sa règle de rendu.
 *
 * Le mur et le capteur reçoivent la pose et le cadrage : ils travaillent en MONDE. Les autres
 * reçoivent un centre déjà projeté — c'est la frontière de `placementShapes.ts`.
 */
function drawPlacement(
  ctx: CanvasRenderingContext2D,
  target: PlacementTarget,
  view: PlacementView,
  time: PlacementTime,
  ink: PlacementInk,
): void {
  const { p, kind } = target
  const color = inkOf(p, ink)
  if (kind === 'wall') {
    drawWall(ctx, { p, heading: wallHeading(p, target.lives) }, view, time, color)
    return
  }
  if (kind === 'sensor') {
    drawSensor(ctx, p, view, time, color)
    return
  }
  const c = project({ x: p.x, y: p.y }, view)
  const ageMs = (time.frame - p.t0) * time.frameMs
  if (kind === 'beacon') drawBeacon(ctx, c, time, color)
  else if (kind === 'seeker') drawSeekerImpulse(ctx, c, ageMs, time, color)
  else if (kind === 'field')
    drawRepairField(
      ctx,
      { c, radiusPx: REPAIR_FIELD_RADIUS_M * viewScale(view), ageMs },
      time,
      color,
    )
  else drawUnnamedDot(ctx, c, time, color)
}

/**
 * drawEquipmentPlacementsLayer trace les poses ACTIVES à l'image courante, puis les marques de
 * révélation que les pings de capteur produisent à cet instant.
 *
 * La fenêtre dessinée n'est PAS [t0, t1] : `t1` date la mise au repos de l'objet, pas sa
 * disparition, que le film ne porte pas (cf. placementEndFrame). Le capteur se tient à sa durée
 * officielle, les autres poses restent jusqu'à la fin du rejeu.
 *
 * LES MARQUES PASSENT APRÈS TOUTES LES POSES, et jamais dans la boucle : une marque est un
 * signe posé sur un JOUEUR, elle doit se lire par-dessus les zones — y compris celle d'un
 * second capteur qui recouvrirait le premier.
 */
export function drawEquipmentPlacementsLayer(
  ctx: CanvasRenderingContext2D,
  scene: PlacementScene,
  view: PlacementView,
  time: PlacementTime,
  ink: PlacementInk,
): void {
  const sensors: ReplayEquipmentPlacement[] = []
  for (const p of scene.placements) {
    const kind = placementKind(p, time.showUnnamed)
    if (!kind || !isPlacementActive(p, kind, time.frame, time)) continue
    drawPlacement(ctx, { p, kind, lives: scene.lives }, view, time, ink)
    if (kind === 'sensor') sensors.push(p)
  }
  if (sensors.length === 0) return
  for (const reveal of sensorReveals(sensors, scene, time)) {
    const color = ink.colorOfSlot(reveal.owner) ?? ink.neutral
    drawRevealMark(ctx, project({ x: reveal.x, y: reveal.y }, view), reveal.sinceMs, time, color)
  }
}

/**
 * Rayon de la zone sensible au survol d'une pose, en pixels d'écran.
 *
 * Les familles dont la forme est déjà en PIXELS (balise, impulsion du traqueur, point neutre)
 * gardent leur taille, relevée au plancher de visée ; celles dont la forme est en MÈTRES
 * (capteur, champ de réparation, mur) suivent l'échelle de la carte — leur zone sensible est
 * celle qu'on voit.
 */
function hoverRadiusPx(kind: PlacementKind, view: PlacementView): number {
  const scale = viewScale(view)
  if (kind === 'sensor') return SENSOR_RADIUS_M * scale
  if (kind === 'field') return REPAIR_FIELD_RADIUS_M * scale
  if (kind === 'wall') return Math.max(wallRadiusM(view) * scale, HOVER_MIN_RADIUS_PX)
  if (kind === 'seeker') return Math.max(SEEKER_IMPULSE_RADIUS_PX, HOVER_MIN_RADIUS_PX)
  if (kind === 'beacon') return Math.max(BEACON_RADIUS_PX, HOVER_MIN_RADIUS_PX)
  return Math.max(UNNAMED_DOT_RADIUS_PX, HOVER_MIN_RADIUS_PX)
}

/**
 * placementAt — la pose sous le curseur, à l'image courante, ou null.
 *
 * LA PLUS PETITE GAGNE, et c'est ce qui rend le survol utilisable : le disque du capteur
 * couvre 4,25 m de rayon, un mur posé dedans en couvre 1,6. Prendre la première trouvée
 * montrerait toujours le capteur, et le mur serait inatteignable au pointeur.
 *
 * `point` est en pixels CSS du canvas, le même repère que le dessin (le facteur de densité est
 * déjà absorbé par la transformation du contexte).
 */
export function placementAt(
  placements: readonly ReplayEquipmentPlacement[],
  view: PlacementView,
  time: PlacementHoverTime,
  point: XY,
): ReplayEquipmentPlacement | null {
  let best: ReplayEquipmentPlacement | null = null
  let bestR = Infinity
  for (const p of placements) {
    const kind = placementKind(p, time.showUnnamed)
    if (!kind || !isPlacementActive(p, kind, time.frame, time)) continue
    if (!placementShows(p, kind, time)) continue
    const r = hoverRadiusPx(kind, view)
    if (r >= bestR) continue
    const c = project({ x: p.x, y: p.y }, view)
    const dx = point.x - c.x
    const dy = point.y - c.y
    if (dx * dx + dy * dy > r * r) continue
    best = p
    bestR = r
  }
  return best
}

/**
 * countDrawablePlacements — ce qu'un document donne À DESSINER, compté par la MÊME PORTE que
 * le tracé (`placementKind`).
 *
 * POURQUOI ICI ET PAS DANS LE COMPOSANT. Une bascule d'interface ne doit s'allumer que si
 * quelque chose se dessinerait derrière elle (même règle que le bouton Zones), et « quelque
 * chose se dessinerait » est une décision de CE module — famille hors table, objet porté,
 * lâcher à la mort. La recompter dans le composant, c'est deux réponses possibles à la même
 * question. `showUnnamed` vaut `true` au comptage : on compte ce qui SERAIT dessiné, les deux
 * bascules allumées.
 */
export function countDrawablePlacements(placements: readonly ReplayEquipmentPlacement[]): {
  drawable: number
  unnamed: number
} {
  let drawable = 0
  let unnamed = 0
  for (const p of placements) {
    const kind = placementKind(p, true)
    if (!kind) continue
    drawable++
    if (kind === 'unnamed') unnamed++
  }
  return { drawable, unnamed }
}
