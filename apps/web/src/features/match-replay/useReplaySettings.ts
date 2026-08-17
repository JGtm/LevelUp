/**
 * useReplaySettings — préférences d'AFFICHAGE du rejeu : calques (visée, zones, noms, carte
 * de chaleur), effets d'événement (tirs, morts) et vitesse de lecture. Persistées comme le
 * son (replayPreferences.ts, patron né dans useReplaySound), depuis le tiroir de réglages
 * (décision utilisateur du 16/08).
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
const SHOW_SHOT_FX_KEY = 'replay-show-shot-fx'
const SHOW_KILL_FX_KEY = 'replay-show-kill-fx'
const SHOW_PLACEMENTS_KEY = 'replay-show-placements'
const SHOW_UNNAMED_PLACEMENTS_KEY = 'replay-show-unnamed-placements'

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

/**
 * LES DEUX EFFETS D'ÉVÉNEMENT, et leurs défauts OPPOSÉS (décision utilisateur du 16/08) :
 *
 *  - les ÉCLAIRS DE BOUCHE sont ALLUMÉS : ils disent où le match se joue, image après image,
 *    et c'est le calque que l'utilisateur a validé sans réserve ;
 *  - les EFFETS DE MORT sont ÉTEINTS : « optionnel, désactivé par défaut ». Le trait tueur ->
 *    victime affirme un couple complet ; il ne s'allume que si on le demande.
 *
 * Ce n'est PAS un demi-livrable au sens de CLAUDE.md n°11 (« pas de flag qui laisse une
 * feature OFF pour plus tard ») : les deux effets sont livrés, complets, et l'interrupteur
 * est un RÉGLAGE D'AFFICHAGE offert au lecteur — pas un interrupteur de chantier.
 */
const SHOW_SHOT_FX_DEFAULT = true
const SHOW_KILL_FX_DEFAULT = false

/**
 * LES DEUX BASCULES DES POSES D'ÉQUIPEMENT, et leurs défauts OPPOSÉS eux aussi (décision
 * utilisateur du 18/08) :
 *
 *  - les POSES sont ALLUMÉES : le mur et le capteur sont des objets du terrain qui changent
 *    la lecture d'un échange, au même titre qu'une zone de capture ;
 *  - les OBJETS NON IDENTIFIÉS sont ÉTEINTS, et ce n'est pas de la pudeur : sur un film de
 *    Big Team Battle, AUCUNE pose n'est nommée aujourd'hui (le seuil de nommage du 18/08
 *    laisse la palette « famille A » sans nom) — les allumer y poserait des centaines de
 *    points neutres. La bascule sert exactement à cela : voir ce que la mesure a trouvé sans
 *    savoir le nommer, quand on le demande.
 *
 * Ni l'une ni l'autre n'est un demi-livrable (CLAUDE.md n°11) : les deux rendus sont livrés
 * et complets, l'interrupteur est un RÉGLAGE D'AFFICHAGE offert au lecteur.
 */
const SHOW_PLACEMENTS_DEFAULT = true
const SHOW_UNNAMED_PLACEMENTS_DEFAULT = false

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
  /** Éclairs de bouche sur TOUS les tirs décodés. Allumés par défaut. */
  showShotFx: boolean
  toggleShotFx: () => void
  /** Trait orienté tueur -> victime sur les éliminations. ÉTEINT par défaut. */
  showKillFx: boolean
  toggleKillFx: () => void
  /** Calque des POSES d'équipement (mur, capteur). Allumé par défaut. */
  showPlacements: boolean
  togglePlacements: () => void
  /** Poses dont la nature n'est pas établie (famille `other`). ÉTEINT par défaut. */
  showUnnamedPlacements: boolean
  toggleUnnamedPlacements: () => void
  /** Multiplicateur de vitesse courant — toujours une valeur de SPEED_MULTIPLIERS. */
  speed: number
  setSpeed: (speed: number) => void
}

/**
 * usePersistedFlag — UN interrupteur persisté : son état, et la bascule qui l'écrit.
 *
 * CENTRALISÉ ICI (CLAUDE.md n°6, « à la 3e copie, centraliser ») : les calques de visée, de
 * zones, de noms, de carte de chaleur, d'effets de tirs et d'effets de mort partagent EXACTEMENT
 * le même corps — état initial lu du stockage, bascule qui persiste la nouvelle valeur. Six
 * copies de six lignes se seraient mises à diverger (l'une oubliant la persistance, l'autre
 * la forme fonctionnelle du setState).
 */
function usePersistedFlag(key: string, fallback: boolean): [boolean, () => void] {
  const [value, setValue] = useState(() => readStoredFlag(key, fallback))
  const toggle = useCallback(() => {
    setValue((prev) => {
      const next = !prev
      persistPreference(key, String(next))
      return next
    })
  }, [key])
  return [value, toggle]
}

export function useReplaySettings(): ReplaySettings {
  const [showAim, toggleAim] = usePersistedFlag(SHOW_AIM_KEY, true)
  const [showZones, toggleZones] = usePersistedFlag(SHOW_ZONES_KEY, true)
  const [showNames, toggleNames] = usePersistedFlag(SHOW_NAMES_KEY, true)
  const [showHeatmap, toggleHeatmap] = usePersistedFlag(SHOW_HEATMAP_KEY, SHOW_HEATMAP_DEFAULT)
  const [showShotFx, toggleShotFx] = usePersistedFlag(SHOW_SHOT_FX_KEY, SHOW_SHOT_FX_DEFAULT)
  const [showKillFx, toggleKillFx] = usePersistedFlag(SHOW_KILL_FX_KEY, SHOW_KILL_FX_DEFAULT)
  const [showPlacements, togglePlacements] = usePersistedFlag(
    SHOW_PLACEMENTS_KEY,
    SHOW_PLACEMENTS_DEFAULT,
  )
  const [showUnnamedPlacements, toggleUnnamedPlacements] = usePersistedFlag(
    SHOW_UNNAMED_PLACEMENTS_KEY,
    SHOW_UNNAMED_PLACEMENTS_DEFAULT,
  )
  const [heatmapMode, setHeatmapModeState] = useState(() =>
    readStoredChoice(HEATMAP_MODE_KEY, HEATMAP_MODE_DEFAULT, HEATMAP_MODES),
  )
  const [speed, setSpeedState] = useState(() =>
    readStoredNumber(SPEED_KEY, SPEED_DEFAULT, (v) => SPEED_MULTIPLIERS.includes(v)),
  )

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
    showShotFx,
    toggleShotFx,
    showKillFx,
    toggleKillFx,
    showPlacements,
    togglePlacements,
    showUnnamedPlacements,
    toggleUnnamedPlacements,
    speed,
    setSpeed,
  }
}
