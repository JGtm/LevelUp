/**
 * useReplaySettings — préférences d'AFFICHAGE du rejeu : calques (visée, zones) et vitesse
 * de lecture. Persistées comme le son (replayPreferences.ts, patron né dans
 * useReplaySound), depuis le tiroir de réglages (décision utilisateur du 16/08).
 *
 * DÉLIBÉRÉMENT SÉPARÉ DE useReplaySound : ce hook ne connaît RIEN au son (ni la piste, ni
 * Web Audio) — même séparation des responsabilités que les modules voisins. Il ne fait que
 * porter l'état des calques et de la vitesse ; ReplayCanvas les consomme sans savoir qu'ils
 * survivent à la page, ReplaySettingsDrawer les affiche sans savoir où ils vivent.
 */
import { useCallback, useState } from 'react'

import { persistPreference, readStoredFlag, readStoredNumber } from './replayPreferences'

const SHOW_AIM_KEY = 'replay-show-aim'
const SHOW_ZONES_KEY = 'replay-show-zones'
const SPEED_KEY = 'replay-speed'

/** Multiplicateurs de vitesse proposés (repris du POC, réglés à l'écran). */
export const SPEED_MULTIPLIERS: readonly number[] = [0.5, 1, 2, 4]

const SPEED_DEFAULT = 1

export interface ReplaySettings {
  /** Calque de visée (direction du regard). Allumé par défaut, comme aujourd'hui. */
  showAim: boolean
  toggleAim: () => void
  /** Calque des zones nommées. Allumé par défaut ; l'appelant le rend conditionnel à la
   *  présence de zones sur la carte (même règle que le bouton d'origine). */
  showZones: boolean
  toggleZones: () => void
  /** Multiplicateur de vitesse courant — toujours une valeur de SPEED_MULTIPLIERS. */
  speed: number
  setSpeed: (speed: number) => void
}

export function useReplaySettings(): ReplaySettings {
  const [showAim, setShowAim] = useState(() => readStoredFlag(SHOW_AIM_KEY, true))
  const [showZones, setShowZones] = useState(() => readStoredFlag(SHOW_ZONES_KEY, true))
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

  const setSpeed = useCallback((next: number) => {
    setSpeedState(next)
    persistPreference(SPEED_KEY, String(next))
  }, [])

  return { showAim, toggleAim, showZones, toggleZones, speed, setSpeed }
}
