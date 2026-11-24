/**
 * savedViews — « MES VUES » : jusqu'à trois vues nommées que le lecteur enregistre depuis son état courant, qu'il
 * renomme et supprime, dont l'une peut s'ouvrir d'office avec tout rejeu, et dont chacune peut se rattacher à un
 * mode de jeu. Le contrat des réducteurs :
 *
 *  - une vue enregistrée est la photographie des réglages couverts par les préréglages (`snapshotValues`) ; elle se
 *    compare, s'applique et se met à jour comme un préréglage (`viewPresets`) ;
 *  - la vue d'ouverture est un préréglage ou une vue enregistrée ; sans choix, le rejeu s'ouvre avec les réglages
 *    tels qu'ils ont été laissés (le comportement d'avant) ;
 *  - une vue rattachée à un mode passe avant la vue d'ouverture, pour les matchs de ce mode seulement ;
 *  - la vue active est un préréglage ou une vue enregistrée ; supprimer la vue active ramène à « Complet ».
 *
 * LOGIQUE PURE : aucun React, aucun stockage. Les identifiants sont fabriqués par l'appelant (`newId`), jamais ici.
 */
import { MATCH_MODE_KEYS, type MatchModeKey } from './settingsAvailability'
import { SETTING_IDS, type SettingId, type SettingValue } from './settingsTable'
import { isPresetId, presetValues, type PresetId, type ViewValues } from './viewPresets'

/** Combien de vues enregistrées au plus, et la longueur d'un nom. */
export const VIEW_LIMIT = 3
export const VIEW_NAME_MAX = 24

/** La vue d'ouverture quand rien n'est choisi : Complet. */
export const DEFAULT_ACTIVE: PresetId = 'complet'

export interface SavedView {
  readonly id: string
  readonly name: string
  readonly values: ViewValues
  /** Le mode de jeu auquel la vue est rattachée, ou `null`. */
  readonly mode: MatchModeKey | null
}

export interface ViewsState {
  readonly views: readonly SavedView[]
  /** La vue d'ouverture (préréglage ou vue enregistrée), ou `null` : « dernière vue utilisée ». */
  readonly defaultId: string | null
  /** La vue active : un préréglage ou une vue enregistrée. */
  readonly activeId: string
}

export const EMPTY_VIEWS: ViewsState = { views: [], defaultId: null, activeId: DEFAULT_ACTIVE }

/** Le nom retenu : sans espaces autour, coupé à la longueur permise ; vide, `null`. */
export function cleanName(raw: string): string | null {
  const name = raw.trim().slice(0, VIEW_NAME_MAX).trim()
  return name === '' ? null : name
}

/** La vue `id` existe-t-elle (préréglage ou vue enregistrée) ? */
export function viewExists(state: ViewsState, id: string): boolean {
  return isPresetId(id) || state.views.some((v) => v.id === id)
}

/** La table d'une vue (préréglage ou enregistrée), ou `null` si elle n'existe pas. */
export function valuesOfView(state: ViewsState, id: string): ViewValues | null {
  if (isPresetId(id)) return presetValues(id)
  return state.views.find((v) => v.id === id)?.values ?? null
}

/** La vue active, telle qu'elle est (`null` pour un préréglage : son nom est dans les libellés). */
export function userViewOf(state: ViewsState, id: string): SavedView | null {
  return state.views.find((v) => v.id === id) ?? null
}

export type SaveResult =
  | { readonly ok: true; readonly state: ViewsState; readonly view: SavedView }
  | { readonly ok: false; readonly reason: 'name' | 'limit' }

/** Enregistre l'état courant comme une nouvelle vue, qui devient la vue active. */
export function saveView(state: ViewsState, name: string, values: ViewValues, id: string): SaveResult {
  if (state.views.length >= VIEW_LIMIT) return { ok: false, reason: 'limit' }
  const clean = cleanName(name)
  if (clean === null) return { ok: false, reason: 'name' }
  const view: SavedView = { id, name: clean, values, mode: null }
  return { ok: true, state: { ...state, views: [...state.views, view], activeId: id }, view }
}

/** Met à jour la photographie d'une vue enregistrée (« Mettre à jour cette vue »). */
export function updateView(state: ViewsState, id: string, values: ViewValues): ViewsState {
  return { ...state, views: state.views.map((v) => (v.id === id ? { ...v, values } : v)) }
}

/** Renomme une vue ; un nom vide laisse l'ancien. */
export function renameView(state: ViewsState, id: string, name: string): ViewsState {
  const clean = cleanName(name)
  if (clean === null) return state
  return { ...state, views: state.views.map((v) => (v.id === id ? { ...v, name: clean } : v)) }
}

/** Supprime une vue ; si elle était la vue d'ouverture ou la vue active, ces rôles sont rendus. */
export function deleteView(state: ViewsState, id: string): ViewsState {
  return {
    views: state.views.filter((v) => v.id !== id),
    defaultId: state.defaultId === id ? null : state.defaultId,
    activeId: state.activeId === id ? DEFAULT_ACTIVE : state.activeId,
  }
}

/** Rattache une vue enregistrée à un mode de jeu, ou la détache (`null`). */
export function attachMode(state: ViewsState, id: string, mode: MatchModeKey | null): ViewsState {
  return { ...state, views: state.views.map((v) => (v.id === id ? { ...v, mode } : v)) }
}

/** Pose la vue d'ouverture : une vue qui n'existe pas n'est pas retenue. `null` = dernière vue utilisée. */
export function setDefaultView(state: ViewsState, id: string | null): ViewsState {
  if (id !== null && !viewExists(state, id)) return state
  return { ...state, defaultId: id }
}

/** Pose la vue active ; une vue qui n'existe pas ramène à Complet. */
export function setActiveView(state: ViewsState, id: string): ViewsState {
  return { ...state, activeId: viewExists(state, id) ? id : DEFAULT_ACTIVE }
}

/**
 * LA VUE À APPLIQUER À L'OUVERTURE d'un rejeu de ce mode : la vue rattachée à ce mode d'abord, puis la vue
 * d'ouverture, sinon `null` (les réglages restent comme ils ont été laissés).
 */
export function openingViewId(state: ViewsState, mode: MatchModeKey): string | null {
  const linked = state.views.find((v) => v.mode === mode)
  if (linked) return linked.id
  if (state.defaultId !== null && viewExists(state, state.defaultId)) return state.defaultId
  return null
}

const SETTING_ID_SET: ReadonlySet<string> = new Set(SETTING_IDS)

function isSettingValue(v: unknown): v is SettingValue {
  return v === null || typeof v === 'boolean' || typeof v === 'string' || (typeof v === 'number' && Number.isFinite(v))
}

/** Une vue lue du stockage, validée : un réglage inconnu ou une valeur étrangère est écarté, jamais cru. */
function readView(raw: unknown): SavedView | null {
  if (typeof raw !== 'object' || raw === null) return null
  const r = raw as Record<string, unknown>
  const name = typeof r.name === 'string' ? cleanName(r.name) : null
  if (typeof r.id !== 'string' || r.id === '' || name === null) return null
  const values: Partial<Record<SettingId, SettingValue>> = {}
  if (typeof r.values === 'object' && r.values !== null) {
    for (const [k, v] of Object.entries(r.values)) {
      if (SETTING_ID_SET.has(k) && isSettingValue(v)) values[k as SettingId] = v
    }
  }
  const mode = (MATCH_MODE_KEYS as readonly unknown[]).includes(r.mode) ? (r.mode as MatchModeKey) : null
  return { id: r.id, name, values, mode }
}

/** Les vues lues du stockage : au plus `VIEW_LIMIT`, sans doublon d'identifiant, les invalides écartées. */
export function readViews(raw: unknown): readonly SavedView[] {
  if (!Array.isArray(raw)) return []
  const seen = new Set<string>()
  const out: SavedView[] = []
  for (const item of raw) {
    const v = readView(item)
    if (v === null || seen.has(v.id) || out.length >= VIEW_LIMIT) continue
    seen.add(v.id)
    out.push(v)
  }
  return out
}
