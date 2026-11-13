/**
 * viewPresetLogic — LES PRÉRÉGLAGES DE VUE du rejeu : quels calques chacun allume, ce qu'un
 * préréglage fait d'un état de calques, et la vue que l'état courant décrit.
 *
 * LOGIQUE PURE : aucune lecture de stockage, aucun React. Le hook `useReplayView` branche cette
 * table sur les bascules réelles de `useReplaySettings`.
 *
 * UN « CALQUE », ICI, EST UNE BASCULE QUI MONTRE OU CACHE UNE FAMILLE D'OBJETS (treize au
 * total). Les AFFINAGES d'un calque ne sont pas des calques et un préréglage ne les touche pas :
 * objets non identifiés des poses, armes spéciales seulement, armes de départ au sol. Ils
 * restreignent ou élargissent ce qu'un calque montre, et leurs défauts (éteints) sont ce qui
 * garde « Complet » égal à l'état d'ouverture de la page — les allumer noierait un
 * Grande Bataille de points neutres.
 *
 * LES EFFETS, LA CARTE DE CHALEUR, LA COULEUR DES POINTS ET LE SON ne sont pas des calques non
 * plus : un préréglage les laisse comme l'utilisateur les a réglés.
 *
 * DEUX RÈGLES TIENNENT TOUT LE RESTE :
 *  - un préréglage n'ACTIVE jamais un calque indisponible pour le match, et ne le touche pas du
 *    tout (sa préférence mémorisée vaut pour le prochain match qui le porte) ;
 *  - la vue active se DÉDUIT de l'état, sur les seuls calques disponibles : « Personnalisé »
 *    dès qu'un calque diffère de tous les préréglages.
 */

/** Les treize calques, dans l'ordre où le panneau les propose. */
export const LAYER_KEYS = [
  'aim',
  'trail',
  'zones',
  'weaponPads',
  'groundWeapons',
  'vehicles',
  'placements',
  'droppedPlacements',
  'modeObjectives',
  'flagCarries',
  'vipCrown',
  'skullCarrier',
  'bombCarrier',
] as const

export type LayerKey = (typeof LAYER_KEYS)[number]

/** Un booléen par calque : l'état affiché, ou ce que le film porte. */
export type LayerFlags = Readonly<Record<LayerKey, boolean>>

/** Les préréglages, dans l'ordre où le panneau les propose (et où une égalité se tranche). */
export const VIEW_PRESET_IDS = ['complet', 'combat', 'deplacements', 'objectif'] as const

export type ViewPresetId = (typeof VIEW_PRESET_IDS)[number]

/** « Personnalisé » : l'état n'est celui d'aucun préréglage. Jamais choisi, toujours déduit. */
export const CUSTOM_VIEW = 'personnalise' as const

export type ViewId = ViewPresetId | typeof CUSTOM_VIEW

/**
 * LES OBJECTIFS D'UN MODE, d'un bloc : le terrain de l'enjeu (zones de capture, collines,
 * bases) et ce qui se porte (drapeau, couronne, crâne, bombe). Un mode n'en porte que
 * quelques-uns, la disponibilité écarte les autres.
 */
const OBJECTIVE_LAYERS: readonly LayerKey[] = [
  'modeObjectives',
  'flagCarries',
  'vipCrown',
  'skullCarrier',
  'bombCarrier',
]

/**
 * LA TABLE : pour chaque préréglage, les calques ALLUMÉS ; tous les autres sont éteints.
 *
 *  - Complet : tout.
 *  - Combat : visée, traînée, véhicules, objectifs.
 *  - Déplacements : traînée et zones. Le réglage « traînée longue / vies entières » de la
 *    maquette n'existe pas dans l'application : la traînée existante en tient lieu.
 *  - Objectif : traînée, zones, objectifs.
 */
export const PRESET_LAYERS: Readonly<Record<ViewPresetId, readonly LayerKey[]>> = {
  complet: LAYER_KEYS,
  combat: ['aim', 'trail', 'vehicles', ...OBJECTIVE_LAYERS],
  deplacements: ['trail', 'zones'],
  objectif: ['trail', 'zones', ...OBJECTIVE_LAYERS],
}

/** L'état complet (un booléen par calque) que décrit un préréglage. */
export function presetFlags(id: ViewPresetId): LayerFlags {
  const on = PRESET_LAYERS[id]
  return Object.fromEntries(LAYER_KEYS.map((k) => [k, on.includes(k)])) as LayerFlags
}

/** Ce que le film porte, tel que le canvas le publie (les mêmes questions que le tiroir). */
export interface FilmLayerAvailability {
  zones: boolean
  placements: { drawable: boolean; unnamed: boolean; dropped: boolean }
  weaponPads: boolean
  groundWeapons: boolean
  modeObjectives: boolean
  flagCarries: boolean
  vipCrown: boolean
  skullCarrier: boolean
  bombCarrier: boolean
  vehicles: boolean
  /**
   * Le fond retenu porte un plan des murs. Ce n'est pas un calque de préréglage (`LAYER_KEYS`) :
   * il ne fait que conditionner l'affichage de sa bascule.
   */
  wallPlan: boolean
}

/**
 * Ce que le film porte, calque par calque. La visée et la traînée n'ont aucune condition (une
 * vie a toujours un regard et un passé). Les objets lâchés ne se proposent qu'avec leur mère :
 * des poses dessinables ET au moins un objet lâché.
 */
export function layerAvailability(film: FilmLayerAvailability): LayerFlags {
  return {
    aim: true,
    trail: true,
    zones: film.zones,
    weaponPads: film.weaponPads,
    groundWeapons: film.groundWeapons,
    vehicles: film.vehicles,
    placements: film.placements.drawable,
    droppedPlacements: film.placements.drawable && film.placements.dropped,
    modeObjectives: film.modeObjectives,
    flagCarries: film.flagCarries,
    vipCrown: film.vipCrown,
    skullCarrier: film.skullCarrier,
    bombCarrier: film.bombCarrier,
  }
}

/**
 * Les calques à BASCULER pour passer de `current` au préréglage `id` : ceux dont l'état diffère
 * de la table ET que le match porte. Un calque indisponible n'est jamais dans la liste.
 */
export function layersToToggle(
  id: ViewPresetId,
  current: LayerFlags,
  available: LayerFlags,
): LayerKey[] {
  const target = presetFlags(id)
  return LAYER_KEYS.filter((k) => available[k] && current[k] !== target[k])
}

/** Les calques disponibles de `current` sont-ils exactement ceux du préréglage `id` ? */
function matchesPreset(id: ViewPresetId, current: LayerFlags, available: LayerFlags): boolean {
  return layersToToggle(id, current, available).length === 0
}

/**
 * LA VUE ACTIVE, déduite de l'état. Plusieurs préréglages peuvent être égaux sur les calques
 * d'un match donné (sans objectif, zone ni véhicule, Complet et Combat se confondent) : le
 * préréglage MÉMORISÉ gagne s'il est encore égal à l'état, sinon le premier de la liste.
 * Aucun préréglage égal : Personnalisé.
 */
export function activeView(
  current: LayerFlags,
  available: LayerFlags,
  remembered?: ViewPresetId,
): ViewId {
  if (remembered !== undefined && matchesPreset(remembered, current, available)) return remembered
  return VIEW_PRESET_IDS.find((id) => matchesPreset(id, current, available)) ?? CUSTOM_VIEW
}
