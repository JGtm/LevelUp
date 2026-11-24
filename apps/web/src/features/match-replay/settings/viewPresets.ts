/**
 * viewPresets — LES PRÉRÉGLAGES DE VUE : un point de départ, pas un mode fermé. Une vue (préréglage ou vue
 * enregistrée) est une table de valeurs pour les réglages que la table des réglages dit « couverts »
 * (`SettingDef.preset`) ; on l'applique d'un geste, puis on ajuste librement.
 *
 * UN ÉTAT « MODIFIÉ » SE DÉDUIT, il ne s'écrit pas : la vue active est « modifiée » quand au moins un réglage qu'elle
 * couvre diffère de sa table, sur les SEULS réglages que le match propose (`isProposed`). Un réglage sans objet
 * sur ce match ne rend donc jamais la vue « modifiée », et appliquer une vue ne le touche pas : sa préférence vaut
 * pour le prochain match qui le porte.
 *
 * UNE VUE ENREGISTRÉE PEUT IGNORER UN RÉGLAGE : une table qui ne dit rien d'un réglage (ajouté depuis par la table
 * des réglages) ne le compare pas et ne l'écrit pas.
 *
 * LOGIQUE PURE : aucun React, aucun stockage.
 */
import { isProposed, type SettingsContext } from './settingsAvailability'
import { PRESET_COVERED, type SettingId, type SettingValue, type SettingValues } from './settingsTable'

/** Les préréglages, dans l'ordre où le panneau les propose. */
export const PRESET_IDS = ['complet', 'combat', 'deplacements', 'objectif'] as const
export type PresetId = (typeof PRESET_IDS)[number]

/** Une table de valeurs : le préréglage ou la vue enregistrée. Un réglage absent n'est ni comparé ni écrit. */
export type ViewValues = Readonly<Partial<Record<SettingId, SettingValue>>>

export function isPresetId(raw: unknown): raw is PresetId {
  return typeof raw === 'string' && (PRESET_IDS as readonly string[]).includes(raw)
}

/** Les trois couches portées de l'enjeu (VIP, crâne, bombe) suivent les drapeaux : elles s'allument ensemble. */
const WITH_CARRIED: readonly SettingId[] = ['flags', 'vip', 'skull', 'bomb']

/** Une table de préréglage : les interrupteurs allumés, tous les autres éteints, et le filtre du fil. */
function table(on: readonly SettingId[], feedFilter = 'tout'): ViewValues {
  const out: Partial<Record<SettingId, SettingValue>> = {}
  for (const id of PRESET_COVERED) out[id] = id === 'feedFilter' ? feedFilter : on.includes(id)
  return out
}

/**
 * LES TABLES.
 *  - Complet : tout ce qui se dessine ou se dit, sauf les traces de vies et la chaleur ; c'est l'état d'ouverture ;
 *  - Combat : regard, tirs, véhicules, objectifs, morts et tueurs ; le fil ne montre que les éliminations ;
 *  - Déplacements : trajets, apparitions, zones et chaleur ;
 *  - Objectif : trajets, zones et objectifs ; le fil ne montre que les objectifs.
 */
const PRESETS: Readonly<Record<PresetId, ViewValues>> = {
  complet: table([
    'aim', 'trail', 'shotFx', 'killFx', 'forecast', 'reliefLines', 'wallPlan', 'zones', 'weaponPads', 'groundWeapons',
    'vehicles', 'placements', 'placeDropped', 'objectives', ...WITH_CARRIED, 'planFeed',
    'catWeapon', 'catGrenade', 'catMelee', 'catEquip',
  ]),
  combat: table([
    'aim', 'trail', 'shotFx', 'killFx', 'deaths', 'killers', 'wallPlan', 'weaponPads', 'groundWeapons', 'vehicles',
    'objectives', ...WITH_CARRIED, 'planFeed', 'catWeapon', 'catGrenade', 'catMelee',
  ], 'kills'),
  deplacements: table(['trail', 'spawns', 'forecast', 'zones', 'reliefLines', 'heatmap', 'vehicles', 'planFeed']),
  objectif: table([
    'trail', 'zones', 'objectives', ...WITH_CARRIED, 'vehicles', 'weaponPads', 'planFeed', 'catWeapon',
  ], 'objectives'),
}

/** La table d'un préréglage. */
export function presetValues(id: PresetId): ViewValues {
  return PRESETS[id]
}

/** La photographie des réglages couverts : ce que « Enregistrer comme ma vue » retient. */
export function snapshotValues(values: SettingValues): ViewValues {
  return Object.fromEntries(PRESET_COVERED.map((id) => [id, values[id]])) as ViewValues
}

/** La valeur que la vue vise pour ce réglage, ou `undefined` quand elle n'en dit rien ou que ce match ne la propose pas. */
function targetOf(id: SettingId, target: ViewValues, c: SettingsContext): SettingValue | undefined {
  const want = target[id]
  if (want === undefined) return undefined
  // Un filtre du fil que ce match ne propose pas (Objectifs sans ligne d'objectif) ne se vise pas.
  if (id === 'feedFilter' && !c.feedFilters.includes(want as string)) return undefined
  return want
}

/** Les écritures qui amènent l'état à la vue : les réglages couverts, proposés, dont la valeur diffère. */
export function writesToReach(target: ViewValues, values: SettingValues, c: SettingsContext): ReadonlyArray<readonly [SettingId, SettingValue]> {
  const out: Array<readonly [SettingId, SettingValue]> = []
  for (const id of PRESET_COVERED) {
    if (!isProposed(id, c)) continue
    const want = targetOf(id, target, c)
    if (want !== undefined && values[id] !== want) out.push([id, want])
  }
  return out
}

/** Les réglages qui diffèrent de la vue : sa « modification ». Vide = la vue est telle quelle. */
export function changedFrom(target: ViewValues, values: SettingValues, c: SettingsContext): readonly SettingId[] {
  return writesToReach(target, values, c).map(([id]) => id)
}
