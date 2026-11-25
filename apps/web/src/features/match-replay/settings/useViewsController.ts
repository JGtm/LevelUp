/**
 * useViewsController — LES VUES DU PANNEAU « VUE » EN ACTION : appliquer un préréglage ou une vue, revenir à elle,
 * enregistrer l'état comme une vue, la mettre à jour, la renommer, la supprimer, choisir la vue d'ouverture et le
 * mode auquel une vue se rattache, épingler des réglages, réinitialiser. Toute la logique est dans les modules
 * purs (`viewPresets`, `savedViews`, `quickAccess`) ; ce hook ne fait que les brancher sur les réglages réels
 * (`SettingsBinding`) et sur la mémoire des vues (`settingsStore`).
 *
 * LA VUE ACTIVE N'EST PAS UN ÉTAT DE RÉGLAGES : elle ne porte que son nom. « Modifié » se déduit de la comparaison
 * des réglages à sa table (`changedFrom`), sur les seuls réglages proposés pour ce match ; rien n'est écrit pour lui.
 *
 * L'OUVERTURE : à l'ouverture d'un rejeu, la vue rattachée au mode du match s'applique, sinon la vue d'ouverture ;
 * sans choix, les réglages restent tels que le lecteur les a laissés. Une seule fois par rejeu ouvert, mais elle vaut aussi
 * pour les réglages que le match ne propose que plus tard (probabilité de réapparition, répliques), tant que le lecteur n'a
 * rien changé lui-même.
 *
 * RÉINITIALISER remet les réglages d'usine et la vue « Complet » ; Mes vues, la vue d'ouverture et les épingles
 * restent.
 */
import { useCallback, useEffect, useRef, useState } from 'react'

import type { MatchModeKey, SettingsContext } from './settingsAvailability'
import type { SettingsBinding } from './settingsBinding'
import { persistPinnedSettings, persistViewsState, readPinnedSettings, readViewsState } from './settingsStore'
import { factoryValues, SETTING_IDS, type SettingId, type SettingValues } from './settingsTable'
import { togglePin } from './quickAccess'
import {
  attachMode, DEFAULT_ACTIVE, deleteView, openingViewId, renameView, saveView, setActiveView, setDefaultView,
  updateView, userViewOf, valuesOfView, type SavedView, type ViewsState,
} from './savedViews'
import { changedFrom, snapshotValues, writesToReach, type ViewValues } from './viewPresets'

export interface ViewsController {
  state: ViewsState
  /** La vue active quand c'est une vue enregistrée ; `null` pour un préréglage. */
  activeUser: SavedView | null
  /** Les réglages qui diffèrent de la vue active : vide = la vue est telle quelle. */
  changed: readonly SettingId[]
  apply: (id: string) => void
  revert: () => void
  /** Enregistre l'état courant comme une vue ; `limit` et `name` disent pourquoi le geste n'a pas eu lieu. */
  saveAs: (name: string) => 'ok' | 'limit' | 'name'
  updateActive: () => void
  rename: (id: string, name: string) => void
  remove: (id: string) => void
  setDefault: (id: string | null) => void
  attach: (id: string, mode: MatchModeKey | null) => void
  pins: readonly SettingId[]
  /** Épingle ou retire ; `full` dit que les cinq places sont prises. */
  pin: (id: SettingId) => 'ok' | 'full'
  reset: () => void
}

interface Input {
  ctx: SettingsContext
  binding: SettingsBinding
  /** Le mode de jeu de ce match (lu du film) : désigne la vue rattachée à appliquer à l'ouverture. */
  mode: MatchModeKey
  /** L'identité du rejeu ouvert : l'ouverture ne s'applique qu'une fois par rejeu. */
  matchKey: string
  /** Fabrique l'identifiant d'une vue neuve (injecté : jamais d'horloge ni de hasard dans la logique). */
  newId?: () => string
}

let counter = 0
function defaultNewId(): string {
  counter += 1
  return `v${Date.now().toString(36)}${counter}`
}

export function useViewsController({ ctx, binding, mode, matchKey, newId = defaultNewId }: Input): ViewsController {
  const [state, setState] = useState(readViewsState)
  const [pins, setPins] = useState(readPinnedSettings)
  const { values, set } = binding

  const commit = useCallback((next: ViewsState) => {
    setState(next)
    persistViewsState(next)
  }, [])

  const write = useCallback(
    (writes: ReadonlyArray<readonly [SettingId, SettingValues[SettingId]]>) => {
      for (const [id, value] of writes) set(id, value)
    },
    [set],
  )

  // L'OUVERTURE EN ATTENTE : la table de la vue d'ouverture et le contexte avec lequel elle a été appliquée, tant que le lecteur n'a
  // rien changé lui-même à ce rejeu. `null` = rien n'attend (pas de vue d'ouverture, ou le lecteur a agi).
  const lateOpening = useRef<{ table: ViewValues; ctx: SettingsContext } | null>(null)

  const applyView = useCallback(
    (id: string, opening: boolean) => {
      const table = valuesOfView(state, id)
      if (table === null) return
      write(writesToReach(table, values, ctx))
      commit(setActiveView(state, id))
      // Une vue appliquée à la main met fin à l'attente ; l'ouverture, elle, la commence.
      lateOpening.current = opening ? { table, ctx } : null
    },
    [state, values, ctx, write, commit],
  )
  const apply = useCallback((id: string) => applyView(id, false), [applyView])

  // L'OUVERTURE : une fois par rejeu ouvert, avec l'état et le contexte de ce premier rendu.
  const openedFor = useRef<string | null>(null)
  useEffect(() => {
    if (openedFor.current === matchKey) return
    openedFor.current = matchKey
    const id = openingViewId(state, mode)
    // Appliquer une vue écrit des préférences : c'est une synchronisation avec le stockage à l'ouverture du rejeu, pas
    // un état dérivé que l'on pourrait calculer au rendu.
    // eslint-disable-next-line react-hooks/set-state-in-effect
    if (id !== null) applyView(id, true)
    else lateOpening.current = null
    // Seul le rejeu ouvert déclenche l'ouverture ; l'état d'alors est celui du rendu qui l'a monté.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [matchKey])

  // LES RÉGLAGES PROPOSÉS APRÈS L'OUVERTURE (la probabilité à la réponse du serveur, les répliques à l'arrivée de la table) :
  // la vue d'ouverture vaut pour eux aussi, tant que rien de ce qu'elle posait n'a été changé par le lecteur. Au premier écart
  // sur ce qui était déjà proposé, l'attente s'arrête pour de bon.
  useEffect(() => {
    const pending = lateOpening.current
    // Même rendu que l'ouverture : ses écritures ne sont pas encore relues, rien à comparer.
    if (!pending || pending.ctx === ctx) return
    if (changedFrom(pending.table, values, pending.ctx).length > 0) {
      lateOpening.current = null
      return
    }
    const late = writesToReach(pending.table, values, ctx)
    pending.ctx = ctx
    // Écrire une vue est une synchronisation avec les réglages persistés, comme à l'ouverture.
    // eslint-disable-next-line react-hooks/set-state-in-effect
    if (late.length > 0) write(late)
  })

  const activeTable = valuesOfView(state, state.activeId) ?? valuesOfView(state, DEFAULT_ACTIVE)
  const changed = activeTable === null ? [] : changedFrom(activeTable, values, ctx)

  return {
    state,
    activeUser: userViewOf(state, state.activeId),
    changed,
    apply,
    revert: () => apply(state.activeId),
    saveAs: (name) => {
      const r = saveView(state, name, snapshotValues(values), newId())
      if (!r.ok) return r.reason
      commit(r.state)
      return 'ok'
    },
    updateActive: () => {
      if (userViewOf(state, state.activeId) !== null) commit(updateView(state, state.activeId, snapshotValues(values)))
    },
    rename: (id, name) => commit(renameView(state, id, name)),
    remove: (id) => commit(deleteView(state, id)),
    setDefault: (id) => commit(setDefaultView(state, id)),
    attach: (id, m) => commit(attachMode(state, id, m)),
    pins,
    pin: (id) => {
      const r = togglePin(pins, id)
      if (r.full) return 'full'
      setPins(r.pins)
      persistPinnedSettings(r.pins)
      return 'ok'
    },
    reset: () => {
      const factory = factoryValues()
      // « Traits du relief » n'a pas de valeur d'usine unique : elle dépend du fond (allumés sur le Relief seulement).
      lateOpening.current = null
      write(SETTING_IDS.filter((id) => id !== 'reliefLines' && values[id] !== factory[id]).map((id) => [id, factory[id]] as const))
      commit(setActiveView(state, DEFAULT_ACTIVE))
    },
  }
}

