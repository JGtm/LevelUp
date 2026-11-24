/**
 * settingsTable — LA TABLE DES RÉGLAGES D'AFFICHAGE du rejeu : une ligne par réglage, et rien d'autre que des
 * données. Le panneau « Vue », l'accès rapide, le compte « N sur M » des rubriques, l'interrupteur maître et la
 * comparaison aux préréglages se déduisent tous de cette table ; ajouter un réglage, c'est ajouter une ligne ici
 * (et son libellé dans `settings.i18n.ts`), sans retoucher un composant.
 *
 * LOGIQUE PURE : aucun React, aucun stockage. Les clés de stockage ne sont pas ici : les réglages gardent leurs
 * préférences d'avant (`replayPreferences`, les hooks de calques), la table ne dit que ce qu'un réglage EST.
 *
 * LE VOCABULAIRE DE LA TABLE
 *  - `rubric` : la rubrique du panneau qui le porte ; `null` = hors rubriques (la barre de lecture fine, au pied
 *    du panneau) ;
 *  - `sub` : un sous-titre de rubrique (« Vies », « Catégories ») ; les réglages qui le partagent se suivent ;
 *  - `parent` : la mère, dont la fille se lit dessous, en retrait ; `hideWhenParentOff` dit si la fille disparaît
 *    quand la mère est éteinte (un filtre n'a rien à filtrer ; une couche indépendante, si : sa commande doit
 *    rester atteignable) ;
 *  - `preset` : un préréglage la couvre (le panneau la compare à sa table) ;
 *  - `pinnable` : elle peut se poser au bord de la carte (accès rapide) ;
 *  - `master` : elle compte dans l'interrupteur maître de sa rubrique et dans son « N sur M ».
 */
import type { MapBackgroundVariant } from '@/lib/replay/mapBackgroundVariant'

/** Les cinq rubriques du panneau, dans l'ordre où il les propose. */
export const RUBRIC_IDS = ['joueurs', 'carte', 'objets', 'fil', 'son'] as const
export type RubricId = (typeof RUBRIC_IDS)[number]

/** Les sous-titres de rubrique (une pile de réglages qui parlent d'une même chose). */
export type SubGroupId = 'vies' | 'categories'

/** Tous les réglages de la table, dans l'ordre de leur rubrique. */
export const SETTING_IDS = [
  'aim', 'trail', 'colors', 'shotFx', 'killFx',
  'spawns', 'deaths', 'killers', 'livesWho', 'livesWhen', 'forecast', 'forecastDraw',
  'bg', 'reliefLines', 'wallPlan', 'floor', 'floorFollow', 'orientation', 'zones',
  'heatmap', 'heatMode', 'heatSpan',
  'weaponPads', 'groundWeapons', 'groundSpecial', 'groundStarting', 'vehicles',
  'placements', 'placeDropped', 'placeUnnamed', 'objectives', 'flags', 'vip', 'skull', 'bomb',
  'feedFilter', 'planFeed',
  'sound', 'volume', 'catWeapon', 'catGrenade', 'catMelee', 'catEquip',
  'thinBar',
] as const
export type SettingId = (typeof SETTING_IDS)[number]

/** La valeur d'un réglage : un oui/non, un choix (chaîne), un niveau (nombre) ou la coupe d'étage (`null` = Tout). */
export type SettingValue = boolean | string | number | null

/** L'état courant de tous les réglages. */
export type SettingValues = Readonly<Record<SettingId, SettingValue>>

/** La forme de la commande : un interrupteur, un segmenté, une liste, la coupe d'étage ou un curseur. */
export type ControlKind = 'toggle' | 'choice' | 'select' | 'floor' | 'slider'

export interface SettingDef {
  readonly id: SettingId
  readonly rubric: RubricId | null
  readonly control: ControlKind
  readonly sub?: SubGroupId
  readonly parent?: SettingId
  readonly hideWhenParentOff?: boolean
  /** Les valeurs d'un choix, dans l'ordre où le segmenté les propose. */
  readonly options?: readonly string[]
  readonly default: SettingValue
  readonly preset: boolean
  readonly pinnable: boolean
  readonly master: boolean
}

const BACKGROUNDS: readonly MapBackgroundVariant[] = ['plan', 'rendu', 'relief']

/** Un interrupteur de rubrique : couvert par les préréglages, épinglable, compté par le maître. */
function toggle(
  id: SettingId, rubric: RubricId | null, def: boolean,
  more: Partial<Omit<SettingDef, 'id' | 'rubric' | 'control' | 'default'>> = {},
): SettingDef {
  return { id, rubric, control: 'toggle', default: def, preset: true, pinnable: true, master: true, ...more }
}

/** Un interrupteur qui reste hors du préréglage, de l'accès rapide ou du maître. */
const APART = { preset: false, pinnable: false, master: false } as const
const CHILD = { pinnable: false } as const

function choice(
  id: SettingId, rubric: RubricId, options: readonly string[], def: string,
  more: Partial<Omit<SettingDef, 'id' | 'rubric' | 'control' | 'default' | 'options'>> = {},
): SettingDef {
  return { id, rubric, control: 'choice', options, default: def, preset: false, pinnable: false, master: false, ...more }
}

/** LA TABLE. L'ordre des lignes est l'ordre d'affichage dans la rubrique. */
export const SETTING_DEFS: readonly SettingDef[] = [
  // JOUEURS : ce que montre chaque joueur.
  toggle('aim', 'joueurs', true),
  toggle('trail', 'joueurs', true),
  choice('colors', 'joueurs', ['team', 'player'], 'team'),
  toggle('shotFx', 'joueurs', true),
  toggle('killFx', 'joueurs', true),
  toggle('spawns', 'joueurs', false, { sub: 'vies' }),
  toggle('deaths', 'joueurs', false, { sub: 'vies' }),
  toggle('killers', 'joueurs', false, { sub: 'vies' }),
  choice('livesWho', 'joueurs', ['all', 'team', 'followed'], 'all', { sub: 'vies' }),
  choice('livesWhen', 'joueurs', ['all', 'sofar'], 'all', { sub: 'vies' }),
  toggle('forecast', 'joueurs', true, { sub: 'vies' }),
  choice('forecastDraw', 'joueurs', ['both', 'zones', 'points'], 'both', { sub: 'vies', parent: 'forecast', hideWhenParentOff: true }),

  // CARTE : le terrain.
  choice('bg', 'carte', BACKGROUNDS, 'plan'),
  toggle('reliefLines', 'carte', true, { parent: 'bg' }),
  toggle('wallPlan', 'carte', true, { parent: 'bg' }),
  { id: 'floor', rubric: 'carte', control: 'floor', default: null, preset: false, pinnable: false, master: false },
  toggle('floorFollow', 'carte', false, { ...APART, parent: 'floor' }),
  choice('orientation', 'carte', ['auto', '0', '1', '2', '3'], 'auto'),
  toggle('zones', 'carte', true),
  toggle('heatmap', 'carte', false),
  choice('heatMode', 'carte', ['presence', 'kills'], 'presence', { parent: 'heatmap', hideWhenParentOff: true }),
  choice('heatSpan', 'carte', ['match', 'live'], 'match', { parent: 'heatmap', hideWhenParentOff: true }),

  // OBJETS : armes, véhicules, équipements posés, objectifs du mode.
  toggle('weaponPads', 'objets', true),
  toggle('groundWeapons', 'objets', true),
  toggle('groundSpecial', 'objets', false, { ...CHILD, parent: 'groundWeapons', hideWhenParentOff: true }),
  toggle('groundStarting', 'objets', false, { ...CHILD, parent: 'groundWeapons', hideWhenParentOff: true }),
  toggle('vehicles', 'objets', true),
  toggle('placements', 'objets', true),
  toggle('placeDropped', 'objets', true, { ...CHILD, parent: 'placements', hideWhenParentOff: true }),
  toggle('placeUnnamed', 'objets', false, { ...CHILD, parent: 'placements', hideWhenParentOff: true }),
  toggle('objectives', 'objets', true),
  toggle('flags', 'objets', true, { parent: 'objectives' }),
  toggle('vip', 'objets', true, { ...CHILD, parent: 'objectives' }),
  toggle('skull', 'objets', true, { ...CHILD, parent: 'objectives' }),
  toggle('bomb', 'objets', true, { ...CHILD, parent: 'objectives' }),

  // FIL : le fil des événements.
  { id: 'feedFilter', rubric: 'fil', control: 'select', default: 'tout', preset: true, pinnable: false, master: false },
  toggle('planFeed', 'fil', true),

  // SON : un seul endroit de réglage ; la barre garde l'interrupteur et le volume, sur le même état.
  toggle('sound', 'son', false, { preset: false }),
  { id: 'volume', rubric: 'son', control: 'slider', default: 0.7, preset: false, pinnable: false, master: false },
  toggle('catWeapon', 'son', true, { ...CHILD, sub: 'categories' }),
  toggle('catGrenade', 'son', true, { ...CHILD, sub: 'categories' }),
  toggle('catMelee', 'son', true, { ...CHILD, sub: 'categories' }),
  toggle('catEquip', 'son', true, { ...CHILD, sub: 'categories' }),

  // HORS RUBRIQUES : au pied du panneau, au Plein cadre seulement.
  toggle('thinBar', null, true, APART),
]

const BY_ID = new Map<SettingId, SettingDef>(SETTING_DEFS.map((d) => [d.id, d]))

/** La ligne d'un réglage. Un identifiant hors table n'existe pas : le typage l'interdit, la table le garantit. */
export function defOf(id: SettingId): SettingDef {
  const def = BY_ID.get(id)
  if (!def) throw new Error(`réglage inconnu : ${id}`)
  return def
}

/** Les réglages d'une rubrique, dans l'ordre de la table. */
export function defsOfRubric(rubric: RubricId): readonly SettingDef[] {
  return SETTING_DEFS.filter((d) => d.rubric === rubric)
}

/** Les valeurs d'usine : la valeur par défaut de chaque ligne de la table. */
export function factoryValues(): SettingValues {
  return Object.fromEntries(SETTING_DEFS.map((d) => [d.id, d.default])) as SettingValues
}

/** Les réglages que les préréglages couvrent, dans l'ordre de la table. */
export const PRESET_COVERED: readonly SettingId[] = SETTING_DEFS.filter((d) => d.preset).map((d) => d.id)
