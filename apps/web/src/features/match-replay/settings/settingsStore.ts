/**
 * settingsStore — LA MÉMOIRE DES VUES du panneau « Vue » : Mes vues, la vue d'ouverture, la vue active et les
 * réglages épinglés. Retenus dans CE navigateur, jamais côté serveur ; chaque lecture est VALIDÉE (valeur absente,
 * étrangère ou écrite par une autre version -> la valeur par défaut, jamais une erreur ni une valeur crue).
 *
 * Les réglages eux-mêmes gardent leurs clés d'avant (`replayPreferences`, les hooks de calques) : la vue active ne
 * porte que SON NOM — l'état « modifié » se déduit de la comparaison des réglages à sa table
 * (`viewPresets.changedFrom`), rien n'est écrit pour lui.
 *
 * LA VUE ACTIVE vit sous `replay-view-preset`, la clé d'avant (« dernier préréglage choisi ») : un préréglage s'y lit
 * comme avant ; une vue enregistrée s'y lit par son identifiant.
 */
import { persistPreference, readStoredChoice, readStoredRaw } from '@/lib/replay/preferenceStore'

import { DEFAULT_ACTIVE, readViews, viewExists, type ViewsState } from './savedViews'
import { DEFAULT_PINS, readPins } from './quickAccess'
import { RUBRIC_IDS, type RubricId, type SettingId } from './settingsTable'

export const VIEW_ACTIVE_KEY = 'replay-view-preset'
export const VIEWS_KEY = 'replay-views'
export const VIEW_DEFAULT_KEY = 'replay-view-default'
export const PINS_KEY = 'replay-pins'
export const VIEW_TAB_KEY = 'replay-view-tab'

function readJson(key: string): unknown {
  const raw = readStoredRaw(key)
  if (raw === null) return undefined
  try {
    return JSON.parse(raw)
  } catch {
    return undefined
  }
}

/** Mes vues, la vue d'ouverture et la vue active, relues et validées contre les vues qui existent. */
export function readViewsState(): ViewsState {
  const views = readViews(readJson(VIEWS_KEY))
  const base: ViewsState = { views, defaultId: null, activeId: DEFAULT_ACTIVE }
  const defaultId = readStoredRaw(VIEW_DEFAULT_KEY)
  const activeId = readStoredRaw(VIEW_ACTIVE_KEY)
  return {
    views,
    defaultId: defaultId !== null && viewExists(base, defaultId) ? defaultId : null,
    activeId: activeId !== null && viewExists(base, activeId) ? activeId : DEFAULT_ACTIVE,
  }
}

/** Retient Mes vues, la vue d'ouverture et la vue active. Stockage refusé : les choix valent pour la session. */
export function persistViewsState(state: ViewsState): void {
  persistPreference(VIEWS_KEY, JSON.stringify(state.views))
  persistPreference(VIEW_DEFAULT_KEY, state.defaultId ?? '')
  persistPreference(VIEW_ACTIVE_KEY, state.activeId)
}

/** Les réglages épinglés, relus et validés ; sans choix retenu, ceux d'usine. */
export function readPinnedSettings(): readonly SettingId[] {
  return readPins(readJson(PINS_KEY) ?? DEFAULT_PINS)
}

/** Retient les réglages épinglés. */
export function persistPinnedSettings(pins: readonly SettingId[]): void {
  persistPreference(PINS_KEY, JSON.stringify(pins))
}

/** La rubrique du panneau ouverte en dernier, ou la première. */
export function readViewTab(): RubricId {
  return readStoredChoice(VIEW_TAB_KEY, RUBRIC_IDS[0], RUBRIC_IDS)
}

/** Retient la rubrique ouverte. */
export function persistViewTab(rubric: RubricId): void {
  persistPreference(VIEW_TAB_KEY, rubric)
}
