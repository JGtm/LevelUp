/**
 * useReplaySettings — préférences d'AFFICHAGE du rejeu : calques (visée, zones, noms, carte
 * de chaleur), effets d'événement (tirs, morts) et vitesse de lecture. Persistées comme le
 * son (replayPreferences.ts, patron né dans useReplaySound), depuis le tiroir de réglages
 * (décision utilisateur du 16/08).
 *
 * DÉLIBÉRÉMENT SÉPARÉ DE useReplaySound : ce hook ne connaît RIEN au son (ni la piste, ni
 * Web Audio) — même séparation des responsabilités que les modules voisins. Il ne fait que
 * porter l'état des calques et de la vitesse ; ReplayCanvas les consomme sans savoir qu'ils
 * survivent à la page, ReplayViewPanel les affiche sans savoir où ils vivent.
 */
import { useCallback, useState } from 'react'

import type { HeatmapMode, HeatmapSpan } from '../../../lib/replay/heatPaint'
import { persistPreference, readStoredChoice, readStoredNumber } from './replayPreferences'
import { usePersistedFlag } from './usePersistedFlag'
import { useReplayTerrainFlags, type ReplayTerrainFlags } from './useReplayTerrainFlags'

const SHOW_AIM_KEY = 'replay-show-aim'
const SHOW_ZONES_KEY = 'replay-show-zones'
const SHOW_TRAIL_KEY = 'replay-show-trail'
const SPEED_KEY = 'replay-speed'
const SHOW_HEATMAP_KEY = 'replay-show-heatmap'
const HEATMAP_MODE_KEY = 'replay-heatmap-mode'
const HEATMAP_SPAN_KEY = 'replay-heatmap-span'
const SHOW_SHOT_FX_KEY = 'replay-show-shot-fx'
const SHOW_KILL_FX_KEY = 'replay-show-kill-fx'
const MARKER_COLORS_KEY = 'replay-marker-colors'

/**
 * LA FRISE EST-ELLE DÉPLIÉE ? Réglage du LECTEUR et non d'un calque : il ne passe donc pas
 * par le tiroir, mais par un chevron sur la frise elle-même (retour utilisateur du
 * 2026-08-28) — replié, il ne reste que la barre de progression et ses horloges. La clé vit
 * avec les autres préférences du rejeu : c'est leur registre, et `useReplayTimeline` la lit.
 */
export const TIMELINE_EXPANDED_KEY = 'replay-timeline-expanded'

/**
 * LA LECTURE DÉMARRE-T-ELLE TOUTE SEULE ? (demande utilisateur du 2026-08-29, point 22 :
 * « lecture automatique dans les réglages, avec persistance du choix »).
 *
 * EXPORTÉE POUR LA MÊME RAISON QUE LA CLÉ DE LA FRISE : c'est `useReplayPlayback` qui la lit,
 * parce que lui seul tient l'état « en lecture », et il ne passe pas par le tiroir. La bascule,
 * elle, vit bien dans le tiroir comme les autres réglages — d'où la clé partagée plutôt qu'une
 * seconde copie du corps de `usePersistedFlag`.
 *
 * ÉTEINTE PAR DÉFAUT (décision utilisateur du 2026-08-29, dans le même échange que la demande) :
 * le rejeu s'ouvre EN PAUSE, cadré au coup d'envoi, et attend le bouton Lecture. C'est un
 * CHANGEMENT DE COMPORTEMENT assumé — la lecture partait toute seule au montage depuis l'origine
 * du rejeu — et il se tient : on arrive sur cette page par un lien de match, souvent pour lire
 * d'abord le rappel du match et le tableau de scores, pendant que le film, lui, courait déjà.
 *
 * CE N'EST PAS UN DEMI-LIVRABLE (CLAUDE.md n°11) : les deux comportements sont complets et
 * l'interrupteur est un RÉGLAGE offert au lecteur, pas un interrupteur de chantier.
 */
export const AUTOPLAY_KEY = 'replay-autoplay'
export const AUTOPLAY_DEFAULT = false

/** Multiplicateurs de vitesse proposés (repris du POC, réglés à l'écran). */
export const SPEED_MULTIPLIERS: readonly number[] = [0.5, 1, 1.5, 2, 4]

const SPEED_DEFAULT = 1

/** Les deux lectures de la carte de chaleur, dans l'ordre où le tiroir les propose. */
export const HEATMAP_MODES: readonly HeatmapMode[] = ['presence', 'kills']

/** Les deux portées de temps, dans l'ordre où le tiroir les propose (V2, 2026-08-18). */
export const HEATMAP_SPANS: readonly HeatmapSpan[] = ['match', 'live']

/**
 * La carte de chaleur est ÉTEINTE par défaut. Ce n'est pas un demi-livrable : c'est un
 * calque de synthèse qui recouvre le terrain, et le rejeu s'ouvre sur ce qui bouge — les
 * joueurs. Même règle que le son, allumé par l'utilisateur quand il le veut.
 */
const SHOW_HEATMAP_DEFAULT = false
const HEATMAP_MODE_DEFAULT: HeatmapMode = 'presence'

/**
 * TOUTE LA PARTIE PAR DÉFAUT, et c'est le comportement inchangé depuis le 16/08 : la carte du
 * rejeu était DÉJÀ celle du match entier (mesuré avant de coder — `accumulatePresence` ne
 * connaissait aucune borne). La portée `live` est ce que ce lot AJOUTE ; en faire le défaut
 * changerait sous les pieds de l'utilisateur la seule lecture qu'il ait validée.
 */
const HEATMAP_SPAN_DEFAULT: HeatmapSpan = 'match'

/**
 * LES DEUX EFFETS D'ÉVÉNEMENT SONT ALLUMÉS PAR DÉFAUT.
 *
 *  - les ÉCLAIRS DE BOUCHE le sont depuis le 16/08 : ils disent où le match se joue, image
 *    après image, et c'est le calque que l'utilisateur a validé sans réserve ;
 *  - les EFFETS DE MORT les rejoignent le 2026-08-20. Ils étaient éteints au titre du 16/08
 *    (« optionnel, désactivé par défaut »), et c'est cette réserve qui s'est retournée contre
 *    eux : l'utilisateur les CHERCHAIT sans les trouver, puisqu'il fallait déjà savoir qu'ils
 *    existaient pour aller les allumer. Un trait tueur -> victime au moment de l'élimination
 *    est précisément ce qu'on vient lire sur un rejeu.
 *
 * Ce n'est PAS un demi-livrable au sens de CLAUDE.md n°11 (« pas de flag qui laisse une
 * feature OFF pour plus tard ») : les deux effets sont livrés, complets, et l'interrupteur
 * est un RÉGLAGE D'AFFICHAGE offert au lecteur — pas un interrupteur de chantier.
 */
const SHOW_SHOT_FX_DEFAULT = true
const SHOW_KILL_FX_DEFAULT = true


/** Les deux lectures de couleur des points, dans l'ordre où le tiroir les propose. */
export type MarkerColorsMode = 'team' | 'player'
export const MARKER_COLORS_MODES: readonly MarkerColorsMode[] = ['team', 'player']

/**
 * LA COULEUR DES POINTS DIT L'ÉQUIPE PAR DÉFAUT (doctrine D1 du 2026-08-16 : la couleur dit
 * le camp). L'option « distinctes par joueur » (proposition utilisateur du 2026-08-24) donne
 * à chaque joueur une couleur de série stable — suivre QUELQU'UN dans la mêlée — et le camp
 * reste dit par les fiches, le fil et le bandeau. Ce n'est pas un demi-livrable (CLAUDE.md
 * n°11) : les deux lectures sont complètes, l'option est un réglage d'affichage.
 */
const MARKER_COLORS_DEFAULT: MarkerColorsMode = 'team'

// LE RÉGLAGE « FICHES COMPACTES » EST SUPPRIMÉ (décision utilisateur du 2026-08-24 :
// « fiches compactes va devenir la seule et unique option donc on va virer le réglage ») :
// la fiche compacte est devenue LA fiche, la variante longue est partie avec sa clé.

export interface ReplaySettings extends ReplayTerrainFlags {
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
  /**
   * Calque de la TRAÎNÉE (V1, retour utilisateur du 2026-08-18 : « avoir la traînée en
   * option »). ALLUMÉE par défaut, comme aujourd'hui : elle fait partie du marqueur validé
   * le 16/08, et c'est elle qui dit le SENS d'un déplacement. Ce n'est pas un demi-livrable
   * (CLAUDE.md n°11) : le calque est complet, l'interrupteur est un réglage d'affichage.
   */
  showTrail: boolean
  toggleTrail: () => void
  /** Calque de la carte de chaleur. ÉTEINT par défaut (cf. SHOW_HEATMAP_DEFAULT). */
  showHeatmap: boolean
  toggleHeatmap: () => void
  /** Ce que la carte de chaleur mesure — toujours une valeur de HEATMAP_MODES. */
  heatmapMode: HeatmapMode
  setHeatmapMode: (mode: HeatmapMode) => void
  /** Sur QUELLE PORTÉE elle le mesure — toujours une valeur de HEATMAP_SPANS. */
  heatmapSpan: HeatmapSpan
  setHeatmapSpan: (span: HeatmapSpan) => void
  /** Éclairs de bouche sur TOUS les tirs décodés. Allumés par défaut. */
  showShotFx: boolean
  toggleShotFx: () => void
  /** Trait orienté tueur -> victime sur les éliminations. ALLUMÉ par défaut. */
  showKillFx: boolean
  toggleKillFx: () => void
  /** Couleur des points des joueurs : par équipe (défaut) ou distincte par joueur. */
  markerColors: MarkerColorsMode
  setMarkerColors: (mode: MarkerColorsMode) => void
  /**
   * LA LECTURE DÉMARRE-T-ELLE SEULE à l'ouverture du rejeu ? ÉTEINTE par défaut (cf.
   * AUTOPLAY_DEFAULT). Ce n'est PAS une commande de transport : basculer ce réglage ne met
   * ni en lecture ni en pause le rejeu ouvert — il ne décide que de l'état de DÉPART, lu une
   * fois au montage par `useReplayPlayback`. « Lecture » et « Pause » restent à la barre.
   */
  autoPlay: boolean
  toggleAutoPlay: () => void
  /** Multiplicateur de vitesse courant — toujours une valeur de SPEED_MULTIPLIERS. */
  speed: number
  setSpeed: (speed: number) => void
}


export function useReplaySettings(): ReplaySettings {
  const [showAim, toggleAim] = usePersistedFlag(SHOW_AIM_KEY, true)
  const [showZones, toggleZones] = usePersistedFlag(SHOW_ZONES_KEY, true)
  const [showTrail, toggleTrail] = usePersistedFlag(SHOW_TRAIL_KEY, true)
  const [showHeatmap, toggleHeatmap] = usePersistedFlag(SHOW_HEATMAP_KEY, SHOW_HEATMAP_DEFAULT)
  const [showShotFx, toggleShotFx] = usePersistedFlag(SHOW_SHOT_FX_KEY, SHOW_SHOT_FX_DEFAULT)
  const [showKillFx, toggleKillFx] = usePersistedFlag(SHOW_KILL_FX_KEY, SHOW_KILL_FX_DEFAULT)
  const terrain = useReplayTerrainFlags()
  const [heatmapMode, setHeatmapModeState] = useState(() =>
    readStoredChoice(HEATMAP_MODE_KEY, HEATMAP_MODE_DEFAULT, HEATMAP_MODES),
  )
  const [heatmapSpan, setHeatmapSpanState] = useState(() =>
    readStoredChoice(HEATMAP_SPAN_KEY, HEATMAP_SPAN_DEFAULT, HEATMAP_SPANS),
  )
  const [markerColors, setMarkerColorsState] = useState(() =>
    readStoredChoice(MARKER_COLORS_KEY, MARKER_COLORS_DEFAULT, MARKER_COLORS_MODES),
  )
  const [autoPlay, toggleAutoPlay] = usePersistedFlag(AUTOPLAY_KEY, AUTOPLAY_DEFAULT)
  const [speed, setSpeedState] = useState(() =>
    readStoredNumber(SPEED_KEY, SPEED_DEFAULT, (v) => SPEED_MULTIPLIERS.includes(v)),
  )

  const setHeatmapMode = useCallback((next: HeatmapMode) => {
    setHeatmapModeState(next)
    persistPreference(HEATMAP_MODE_KEY, next)
  }, [])

  const setHeatmapSpan = useCallback((next: HeatmapSpan) => {
    setHeatmapSpanState(next)
    persistPreference(HEATMAP_SPAN_KEY, next)
  }, [])

  const setSpeed = useCallback((next: number) => {
    setSpeedState(next)
    persistPreference(SPEED_KEY, String(next))
  }, [])

  const setMarkerColors = useCallback((next: MarkerColorsMode) => {
    setMarkerColorsState(next)
    persistPreference(MARKER_COLORS_KEY, next)
  }, [])

  return {
    showAim,
    toggleAim,
    showZones,
    toggleZones,
    showTrail,
    toggleTrail,
    showHeatmap,
    toggleHeatmap,
    heatmapMode,
    setHeatmapMode,
    heatmapSpan,
    setHeatmapSpan,
    showShotFx,
    toggleShotFx,
    showKillFx,
    toggleKillFx,
    ...terrain,
    markerColors,
    setMarkerColors,
    autoPlay,
    toggleAutoPlay,
    speed,
    setSpeed,
  }
}
