/**
 * settingsDraftStore — préférence locale showWaypointColumn (I19).
 *
 * Colonne « Ouvrir sur Halo Waypoint » des tableaux de matchs : préférence
 * purement locale (localStorage), jamais envoyée au backend. Défaut ON.
 */
import { afterEach, describe, expect, it } from 'vitest'

import { useSettingsDraftStore } from './settingsDraftStore'

const INITIAL_LOCAL_UI_PREFS = useSettingsDraftStore.getState().localUiPrefs

afterEach(() => {
  useSettingsDraftStore.setState({ localUiPrefs: INITIAL_LOCAL_UI_PREFS })
})

describe('settingsDraftStore — showWaypointColumn', () => {
  it('défaut à true', () => {
    expect(useSettingsDraftStore.getState().localUiPrefs.showWaypointColumn).toBe(true)
  })

  it('setShowWaypointColumn(false) met à jour uniquement ce champ', () => {
    useSettingsDraftStore.getState().setShowWaypointColumn(false)
    const prefs = useSettingsDraftStore.getState().localUiPrefs
    expect(prefs.showWaypointColumn).toBe(false)
    // Les autres préférences locales restent inchangées (merge partiel).
    expect(prefs.theme).toBe(INITIAL_LOCAL_UI_PREFS.theme)
    expect(prefs.colorPalette).toBe(INITIAL_LOCAL_UI_PREFS.colorPalette)
  })

  it('setShowWaypointColumn(true) réactive la colonne', () => {
    useSettingsDraftStore.getState().setShowWaypointColumn(false)
    useSettingsDraftStore.getState().setShowWaypointColumn(true)
    expect(useSettingsDraftStore.getState().localUiPrefs.showWaypointColumn).toBe(true)
  })
})
