/**
 * useReplaySettings — préférences d'AFFICHAGE du rejeu : calques (visée, zones, noms, carte
 * de chaleur) et vitesse de lecture. Persistées comme le son (replayPreferences.ts, patron
 * né dans useReplaySound), depuis le tiroir de réglages (décision utilisateur du 16/08).
 *
 * DÉLIBÉRÉMENT SÉPARÉ DE useReplaySound : ce hook ne connaît RIEN au son (ni la piste, ni
 * Web Audio) — même séparation des responsabilités que les modules voisins. Il ne fait que
 * porter l'état des calques et de la vitesse ; ReplayCanvas les consomme sans savoir qu'ils
 * survivent à la page, ReplaySettingsDrawer les affiche sans savoir où ils vivent.
 */
import { useCallback, useState } from 'react'

import type { HeatmapMode } from './heatmapLayer'
import {
  persistPreference,
  readStoredChoice,
  readStoredFlag,
  readStoredNumber,
} from './replayPreferences'

const SHOW_AIM_KEY = 'replay-show-aim'
const SHOW_ZONES_KEY = 'replay-show-zones'
const SHOW_NAMES_KEY = 'replay-show-names'
const SPEED_KEY = 'replay-speed'
const SHOW_HEATMAP_KEY = 'replay-show-heatmap'
const HEATMAP_MODE_KEY = 'replay-heatmap-mode'

/** Multiplicateurs de vitesse proposés (repris du POC, réglés à l'écran). */
export const SPEED_MULTIPLIERS: readonly number[] = [0.5, 1, 2, 4]

const SPEED_DEFAULT = 1

/** Les deux lectures de la carte de chaleur, dans l'ordre où le tiroir les propose. */
export const HEATMAP_MODES: readonly HeatmapMode[] = ['presence', 'kills']

/**
 * La carte de chaleur est ÉTEINTE par défaut. Ce n'est pas un demi-livrable : c'est un
 * calque de synthèse qui recouvre le terrain, et le rejeu s'ouvre sur ce qui bouge — les
 * joueurs. Même règle que le son, allumé par l'utilisateur quand il le veut.
 */
const SHOW_HEATMAP_DEFAULT = false
const HEATMAP_MODE_DEFAULT: HeatmapMode = 'presence'

export interface ReplaySettings {
  /** Calque de visée (direction du regard). Allumé par défaut, comme aujourd'hui. */
  showAim: boolean
  toggleAim: () => void
  /** Calque des zones nommées. Allumé par défaut ; l'appelant le rend conditionnel à la
   *  présence de zones sur la carte (même règle que le bouton d'origine). */
  showZones: boolean
  toggleZones: () => void
  /**
   * Calque des NOMS sous les marqueurs (décision D4, 2026-08-16). Allumé par défaut : le nom
   * est ce qui distingue un coéquipier d'un autre, la couleur ne dit que le camp. Un BTB à
   * 24 joueurs doit néanmoins pouvoir l'éteindre.
   */
  showNames: boolean
  toggleNames: () => void
  /** Calque de la carte de chaleur. ÉTEINT par défaut (cf. SHOW_HEATMAP_DEFAULT). */
  showHeatmap: boolean
  toggleHeatmap: () => void
  /** Ce que la carte de chaleur mesure — toujours une valeur de HEATMAP_MODES. */
  heatmapMode: HeatmapMode
  setHeatmapMode: (mode: HeatmapMode) => void
  /** Multiplicateur de vitesse courant — toujours une valeur de SPEED_MULTIPLIERS. */
  speed: number
  setSpeed: (speed: number) => void
}

export function useReplaySettings(): ReplaySettings {
  const [showAim, setShowAim] = useState(() => readStoredFlag(SHOW_AIM_KEY, true))
  const [showZones, setShowZones] = useState(() => readStoredFlag(SHOW_ZONES_KEY, true))
  const [showNames, setShowNames] = useState(() => readStoredFlag(SHOW_NAMES_KEY, true))
  const [showHeatmap, setShowHeatmap] = useState(() =>
    readStoredFlag(SHOW_HEATMAP_KEY, SHOW_HEATMAP_DEFAULT),
  )
  const [heatmapMode, setHeatmapModeState] = useState(() =>
    readStoredChoice(HEATMAP_MODE_KEY, HEATMAP_MODE_DEFAULT, HEATMAP_MODES),
  )
  const [speed, setSpeedState] = useState(() =>
    readStoredNumber(SPEED_KEY, SPEED_DEFAULT, (v) => SPEED_MULTIPLIERS.includes(v)),
  )

  const toggleAim = useCallback(() => {
    setShowAim((prev) => {
      const next = !prev
      persistPreference(SHOW_AIM_KEY, String(next))
      return next
    })
  }, [])

  const toggleZones = useCallback(() => {
    setShowZones((prev) => {
      const next = !prev
      persistPreference(SHOW_ZONES_KEY, String(next))
      return next
    })
  }, [])

  const toggleNames = useCallback(() => {
    setShowNames((prev) => {
      const next = !prev
      persistPreference(SHOW_NAMES_KEY, String(next))
      return next
    })
  }, [])

  const toggleHeatmap = useCallback(() => {
    setShowHeatmap((prev) => {
      const next = !prev
      persistPreference(SHOW_HEATMAP_KEY, String(next))
      return next
    })
  }, [])

  const setHeatmapMode = useCallback((next: HeatmapMode) => {
    setHeatmapModeState(next)
    persistPreference(HEATMAP_MODE_KEY, next)
  }, [])

  const setSpeed = useCallback((next: number) => {
    setSpeedState(next)
    persistPreference(SPEED_KEY, String(next))
  }, [])

  return {
    showAim,
    toggleAim,
    showZones,
    toggleZones,
    showNames,
    toggleNames,
    showHeatmap,
    toggleHeatmap,
    heatmapMode,
    setHeatmapMode,
    speed,
    setSpeed,
  }
}
