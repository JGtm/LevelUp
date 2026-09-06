/**
 * replayCompose — L'ORDRE DE LA SCÈNE, EN DONNÉE.
 *
 * POURQUOI CE MODULE EXISTE (registre 2026-09-05, M3). La composition du rejeu — vingt-cinq
 * calques, du sol vers le sujet, chacun avec sa condition d'ouverture — vivait dans une
 * fonction `draw` de deux cents lignes au milieu de `ReplayCanvas.tsx`, et le SEUL test qui
 * nommait ce fichier comptait ses lignes. L'ordre des calques et les bascules du tiroir
 * n'étaient donc prouvés par rien : une inversion (les morts sous le sol, la chaleur
 * par-dessus les joueurs) ou un interrupteur qui cesse de couper ne se voyait qu'à l'oeil,
 * sur un rejeu, par quelqu'un qui savait ce qu'il cherchait.
 *
 * L'ORDRE EST UNE RÈGLE DE LECTURE, PAS UNE MISE EN PAGE. Du fond vers le sujet : le sol
 * porte le vocabulaire du terrain, qui porte les objets, qui portent les joueurs, qui portent
 * les événements. Inverser noierait les joueurs — et c'est exactement ce que `LAYER_ORDER`
 * fige ici, en un seul endroit lisible.
 *
 * DEUX RAISONS DE NE PAS PEINDRE, et une seule porte pour les deux : l'utilisateur a coupé le
 * calque au tiroir (`toggles`), ou la scène n'a rien à y mettre (`has`). Les distinguer dans
 * le canvas donnait des conditions mêlées à des appels ; les réunir ici les rend comptables —
 * `composeScene` rend la liste de ce qu'il a PEINT, et c'est cette liste que les tests lisent.
 *
 * CE MODULE NE DESSINE RIEN LUI-MÊME. Chaque `paint` est une fermeture liée par le canvas à
 * son propre état ; ici on ne connaît que leur ORDRE et leur CONDITION. C'est ce qui permet de
 * les remplacer par des témoins dans un test, sans canvas ni React.
 *
 * UNE SEULE CONVENTION, ET C'EST CELLE QUI EXISTAIT DÉJÀ. Onze calques du rejeu sont servis
 * par des hooks qui exposent `paint(ctx, frame, dpr)` — cinq lisent la densité de pixels, six
 * l'ignorent, et TypeScript accepte les seconds là où le premier est attendu. `LayerPaint`
 * n'invente donc rien : c'est la signature la plus large des onze. Les dix calques restants
 * sont des FONCTIONS LIBRES (`drawTracksLayer`, `drawShotsLayer`, `drawKillFxLayer`…), chacune
 * avec ses propres données, ses options et son encre : le canvas les LIE à cette convention au
 * lieu de les réécrire — leur signature riche est ce qui les rend testables une par une, et la
 * détruire pour uniformiser aurait coûté leurs tests sans rien apporter à la composition.
 */

/** Le contrat de calque, celui que portent déjà les onze `paint` des hooks du rejeu. */
export type LayerPaint = (ctx: CanvasRenderingContext2D, frame: number, dpr: number) => void

/**
 * L'ORDRE DE LA SCÈNE, du fond vers le sujet. Ce tableau EST la règle de lecture — le
 * déplacer déplace le calque, et le test d'ordre le dit.
 */
export const LAYER_ORDER = [
  // LE SOL. Deux replis exclusifs : l'image calée si la carte en a une, sinon les props
  // Forge — 3,4 % du terrain, et c'est mieux qu'un fond vide.
  'fond-carte',
  'sol-forge',
  // LE TERRAIN LU. La chaleur sous tout ce qui nomme ou se déplace ; les zones nommées sous
  // ce qui bouge (du vocabulaire, pas un événement) ; les objectifs par-dessus le
  // vocabulaire, l'enjeu primant sur les noms de lieux.
  'chaleur',
  'zones-nommees',
  'objectifs-cuits',
  // LES OBJETS DU TERRAIN, sous les joueurs : ils ne sont pas le sujet.
  'projectiles',
  'socles-armes',
  'armes-au-sol',
  'poses-equipement',
  'vehicules',
  // LES JOUEURS, puis ce qui se lit SUR eux.
  'trajectoires',
  'marques-de-tir',
  'gestes-capacite',
  // LES ÉVÉNEMENTS, qui se lisent sur les trajectoires.
  'tirs',
  'grenades',
  'fin-de-vol',
  'etat-zones',
  'drapeaux',
  'objets-objectif',
  'couronne-vip',
  'crane-porte',
  'bombe-portee',
  'deflagration',
  'pulses-objectif',
  // LES MORTS EN DERNIER : l'événement le plus lourd de sens du calque.
  'morts',
] as const

export type ReplayLayerId = (typeof LAYER_ORDER)[number]

/** Les interrupteurs du tiroir qui masquent un calque ENTIER, quoi qu'il ait à peindre. */
export interface SceneToggles {
  /** « Zones nommées » — le vocabulaire du terrain. */
  zones: boolean
  /** « Effets de tirs » — l'éclair de bouche ET le « ! » du tireur : un seul geste éteint. */
  shotFx: boolean
  /** « Poses d'équipement » — murs, capteurs, failles. */
  placements: boolean
  /** « Effets d'élimination » — le DESSIN s'éteint, jamais la mesure (cf. la chaleur). */
  killFx: boolean
}

/**
 * Ce que la scène a EFFECTIVEMENT à peindre. Un calque sans matière ne s'ouvre pas : ce n'est
 * pas une optimisation, c'est ce qui distingue « éteint » de « vide » quand on lit la liste
 * des calques peints.
 */
export interface SceneMatter {
  /** Le fond de carte calé existe et son emprise se projette. */
  background: boolean
  /** Le repli Forge : la carte n'a pas de sol figé mais le document porte des props. */
  floor: boolean
  heat: boolean
  zoneNames: boolean
  objectivesCooked: boolean
  projectiles: boolean
  placements: boolean
  fireMarks: boolean
  shotFx: boolean
  grenades: boolean
  zoneStates: boolean
  objectivePulses: boolean
  killFx: boolean
}

/** La scène : ce qu'il y a à peindre, ce qui est allumé, et comment peindre chaque calque. */
export interface ReplayScene {
  toggles: SceneToggles
  has: SceneMatter
  /** Le peintre de chaque calque, déjà lié à l'état du canvas. */
  paint: Record<ReplayLayerId, LayerPaint>
}

/**
 * Un CALQUE QUI SE NOMME LUI-MÊME : le hook qui le câble porte son identité à côté de son
 * geste, et la table de liaison du canvas n'a plus à la réécrire.
 *
 * POURQUOI (2026-09-06, revue R1, constat C2). La table de `buildScene` associait 25 ids à
 * 25 peintres, à la main. Les 25 valeurs ont le même type (`LayerPaint`) : intervertir
 * `'couronne-vip': skullCarrier.paint` et `'crane-porte': vipCrown.paint` compilait, passait
 * les 2 350 tests du rejeu, et remplaçait la couronne du VIP par le crâne d'Oddball à l'écran.
 * Un hook qui porte son `id` rend cette faute INÉCRIVABLE pour les onze calques câblés par
 * hook : l'id ne vient plus de la table, il vient du calque.
 */
export interface NamedLayerPainter<Id extends ReplayLayerId = ReplayLayerId> {
  /** Le nom du calque, tel qu'il figure dans `LAYER_ORDER`. */
  id: Id
  paint: LayerPaint
}

/**
 * bindPainters DÉRIVE la liaison id -> peintre des calques eux-mêmes.
 *
 * Le type de retour porte l'union EXACTE des ids passés : un calque oublié fait rougir le
 * compilateur chez l'appelant (la table `paint` de `ReplayScene` exige ses 25 clés), et un id
 * ne peut plus être écrit en face du mauvais peintre puisqu'il n'est plus écrit du tout.
 */
export function bindPainters<const L extends readonly NamedLayerPainter[]>(
  ...calques: L
): Record<L[number]['id'], LayerPaint> {
  const out = {} as Record<L[number]['id'], LayerPaint>
  for (const c of calques) out[c.id as L[number]['id']] = c.paint
  return out
}

/** Un calque prêt à peindre : son nom, sa condition, son geste. */
export interface ReplayLayer {
  id: ReplayLayerId
  /** Faux = ce calque ne peint pas cette image (coupé au tiroir, ou rien à peindre). */
  on: boolean
  paint: LayerPaint
}

/**
 * sceneLayers rend les vingt-cinq calques dans l'ordre, chacun avec sa condition.
 *
 * LES DEUX REPLIS DU SOL SONT EXCLUSIFS, et c'est la seule dépendance entre deux conditions :
 * les props Forge ne se peignent que si l'image calée manque. Partout ailleurs, une condition
 * ne regarde que son propre calque.
 *
 * LES CALQUES SANS CONDITION SONT SANS CONDITION POUR DE BON : leurs peintres (véhicules,
 * drapeaux, couronne, crâne, bombe, déflagration, socles, armes au sol, gestes de capacité,
 * fin de vol, trajectoires) décident eux-mêmes de ne rien poser quand ils n'ont rien — les
 * dédoubler ici donnerait deux vérités sur la même absence.
 */
export function sceneLayers(scene: ReplayScene): ReplayLayer[] {
  const { toggles: t, has: h, paint } = scene
  const on: Record<ReplayLayerId, boolean> = {
    'fond-carte': h.background,
    'sol-forge': !h.background && h.floor,
    chaleur: h.heat,
    'zones-nommees': t.zones && h.zoneNames,
    'objectifs-cuits': h.objectivesCooked,
    projectiles: h.projectiles,
    'socles-armes': true,
    'armes-au-sol': true,
    'poses-equipement': t.placements && h.placements,
    vehicules: true,
    trajectoires: true,
    'marques-de-tir': t.shotFx && h.fireMarks,
    'gestes-capacite': true,
    tirs: t.shotFx && h.shotFx,
    grenades: h.grenades,
    'fin-de-vol': true,
    'etat-zones': h.zoneStates,
    drapeaux: true,
    'objets-objectif': true,
    'couronne-vip': true,
    'crane-porte': true,
    'bombe-portee': true,
    deflagration: true,
    'pulses-objectif': h.objectivePulses,
    morts: t.killFx && h.killFx,
  }
  return LAYER_ORDER.map((id) => ({ id, on: on[id], paint: paint[id] }))
}

/**
 * composeScene peint les calques allumés, dans l'ordre, et rend la liste de ceux qu'il a
 * peints.
 *
 * LA LISTE RENDUE N'EST PAS UN JOURNAL DE DEBUG : c'est le contrat observable de la
 * composition, et le seul moyen de prouver un ordre ou une bascule sans regarder des pixels.
 */
export function composeScene(
  ctx: CanvasRenderingContext2D,
  layers: readonly ReplayLayer[],
  frame: number,
  dpr: number,
): ReplayLayerId[] {
  const peints: ReplayLayerId[] = []
  for (const layer of layers) {
    if (!layer.on) continue
    layer.paint(ctx, frame, dpr)
    peints.push(layer.id)
  }
  return peints
}
