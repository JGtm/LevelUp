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
import { useCallback, useEffect, useState } from 'react'

import type { HeatmapMode, HeatmapSpan } from './heatmapLayer'
import {
  persistPreference,
  readStoredChoice,
  readStoredFlag,
  readStoredNumber,
  subscribePreference,
} from './replayPreferences'

const SHOW_AIM_KEY = 'replay-show-aim'
const SHOW_ZONES_KEY = 'replay-show-zones'
const SHOW_NAMES_KEY = 'replay-show-names'
const SHOW_TRAIL_KEY = 'replay-show-trail'
const SPEED_KEY = 'replay-speed'
const SHOW_HEATMAP_KEY = 'replay-show-heatmap'
const HEATMAP_MODE_KEY = 'replay-heatmap-mode'
const HEATMAP_SPAN_KEY = 'replay-heatmap-span'
const SHOW_SHOT_FX_KEY = 'replay-show-shot-fx'
const SHOW_KILL_FX_KEY = 'replay-show-kill-fx'
const SHOW_PLACEMENTS_KEY = 'replay-show-placements'
const SHOW_UNNAMED_PLACEMENTS_KEY = 'replay-show-unnamed-placements'
const SHOW_DROPPED_PLACEMENTS_KEY = 'replay-show-dropped-placements'
const SHOW_WEAPON_PADS_KEY = 'replay-show-weapon-pads'
const SHOW_FLAG_CARRIES_KEY = 'replay-show-flag-carries'
const MARKER_COLORS_KEY = 'replay-marker-colors'

/** Multiplicateurs de vitesse proposés (repris du POC, réglés à l'écran). */
export const SPEED_MULTIPLIERS: readonly number[] = [0.5, 1, 2, 4]

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

/**
 * LES OBJETS DE PUISSANCE LÂCHÉS SONT ALLUMÉS PAR DÉFAUT (décision utilisateur du 2026-08-18 :
 * « hors Fiesta c'est tactiquement interessant »). Un surbouclier ou un capteur au sol est
 * RAMASSABLE : savoir qu'il est là change la lecture de l'échange suivant, exactement comme
 * savoir qu'une arme est encore sur son socle.
 *
 * LA BASCULE COMMANDE DANS TOUS LES MODES depuis le 2026-08-20. Une garde de mode l'annulait
 * en Fiesta au nom de la même décision du 18/08 ; elle masquait en fait 26 lâchers réels sur
 * le témoin Fiesta `000d5950`, et l'utilisateur veut les voir. Le tiroir affiche donc la
 * commande dès que le film porte de quoi la commander — même règle que le bouton Zones.
 *
 * Ce n'est pas un demi-livrable (CLAUDE.md n°11) : le calque est complet, l'interrupteur est un
 * RÉGLAGE D'AFFICHAGE offert au lecteur.
 */
const SHOW_DROPPED_PLACEMENTS_DEFAULT = true

/**
 * LES SOCLES D'ARME SONT ALLUMÉS PAR DÉFAUT (décision utilisateur du 18/08, W4 : « les infos
 * sont intéressantes à avoir »). Ce sont des lieux du terrain, pas des événements : savoir que
 * le fusil de précision est encore sur son socle change la lecture d'un échange, exactement
 * comme une zone de capture. Le film qui n'en porte aucun n'affiche ni calque ni bascule.
 */
const SHOW_WEAPON_PADS_DEFAULT = true

/**
 * LES DRAPEAUX SONT ALLUMÉS PAR DÉFAUT. C'est l'ENJEU du match en capture de drapeau : savoir
 * où est le drapeau, qui le porte et depuis quand est la lecture même du mode — un rejeu de CTF
 * qui s'ouvrirait sans lui montrerait huit points qui courent sans raison. Le film qui n'en
 * porte aucun (tout mode qui n'est pas de la capture) n'affiche ni calque ni bascule.
 *
 * Ce n'est pas un demi-livrable (CLAUDE.md n°11) : le calque est complet, l'interrupteur est un
 * RÉGLAGE D'AFFICHAGE offert au lecteur — un BTB de capture reste dense.
 */
const SHOW_FLAG_CARRIES_DEFAULT = true

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
  /** Calque des POSES d'équipement (mur, capteur). Allumé par défaut. */
  showPlacements: boolean
  togglePlacements: () => void
  /** Poses dont la nature n'est pas établie (famille `other`). ÉTEINT par défaut. */
  showUnnamedPlacements: boolean
  toggleUnnamedPlacements: () => void
  /**
   * Objets de PUISSANCE lâchés à la mort (power-ups, équipements déployables). ALLUMÉ par
   * défaut, et effectif dans TOUS les modes (cf. SHOW_DROPPED_PLACEMENTS_DEFAULT).
   */
  showDroppedPlacements: boolean
  toggleDroppedPlacements: () => void
  /** Calque des SOCLES D'ARME (schéma 11). Allumé par défaut (cf. SHOW_WEAPON_PADS_DEFAULT). */
  showWeaponPads: boolean
  toggleWeaponPads: () => void
  /**
   * Calque des DRAPEAUX de capture (schéma 15). Allumé par défaut : c'est l'enjeu du mode
   * (cf. SHOW_FLAG_CARRIES_DEFAULT). Un film hors capture n'en publie aucun.
   */
  showFlagCarries: boolean
  toggleFlagCarries: () => void
  /** Couleur des points des joueurs : par équipe (défaut) ou distincte par joueur. */
  markerColors: MarkerColorsMode
  setMarkerColors: (mode: MarkerColorsMode) => void
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
 *
 * L'ÉCRITURE NE SE FAIT JAMAIS DEPUIS L'UPDATER de `setValue`, et c'est un invariant, pas un
 * style. `persistPreference` prévient SYNCHRONEMENT tous les abonnés de la clé, qui appellent
 * `setValue` à leur tour : le faire depuis l'updater, c'est déclencher une mise à jour d'état
 * en pleine phase de rendu, ré-entrante sur ce hook — et sur l'AUTRE instance de la même clé
 * quand elle vit chez un ancêtre (défaut vécu sur la clé des fiches compactes, aujourd'hui
 * supprimée : le tiroir et la page lisaient la même clé). React y répondait en gardant
 * l'ancienne valeur une fois sur deux : la bascule « revenait en arrière » toute seule, et
 * StrictMode, qui invoque les updaters deux fois pour débusquer exactement cette impureté,
 * doublait le symptôme.
 *
 * Donc : `next` se calcule ICI, hors updater, à partir de la valeur rendue ; `setValue` reçoit
 * une valeur nue ; la persistance vient APRÈS, depuis le gestionnaire d'événement. `value` est
 * de ce fait une dépendance de la bascule — c'est le prix, et il est juste : une bascule qui ne
 * connaît pas la valeur qu'elle inverse n'existe pas.
 */
function usePersistedFlag(key: string, fallback: boolean): [boolean, () => void] {
  const [value, setValue] = useState(() => readStoredFlag(key, fallback))
  // L'ABONNEMENT REND LA CLÉ PARTAGEABLE : deux composants qui lisent la même préférence
  // bougent ensemble (cf. la note de `subscribePreference`). Sans lui, la bascule du tiroir
  // ne toucherait que sa propre copie de l'état.
  useEffect(() => subscribePreference(key, (raw) => setValue(raw === 'true')), [key])
  const toggle = useCallback(() => {
    const next = !value
    setValue(next)
    persistPreference(key, String(next))
  }, [key, value])
  return [value, toggle]
}

export function useReplaySettings(): ReplaySettings {
  const [showAim, toggleAim] = usePersistedFlag(SHOW_AIM_KEY, true)
  const [showZones, toggleZones] = usePersistedFlag(SHOW_ZONES_KEY, true)
  const [showNames, toggleNames] = usePersistedFlag(SHOW_NAMES_KEY, true)
  const [showTrail, toggleTrail] = usePersistedFlag(SHOW_TRAIL_KEY, true)
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
  const [showDroppedPlacements, toggleDroppedPlacements] = usePersistedFlag(
    SHOW_DROPPED_PLACEMENTS_KEY,
    SHOW_DROPPED_PLACEMENTS_DEFAULT,
  )
  const [showWeaponPads, toggleWeaponPads] = usePersistedFlag(
    SHOW_WEAPON_PADS_KEY,
    SHOW_WEAPON_PADS_DEFAULT,
  )
  const [showFlagCarries, toggleFlagCarries] = usePersistedFlag(
    SHOW_FLAG_CARRIES_KEY,
    SHOW_FLAG_CARRIES_DEFAULT,
  )
  const [heatmapMode, setHeatmapModeState] = useState(() =>
    readStoredChoice(HEATMAP_MODE_KEY, HEATMAP_MODE_DEFAULT, HEATMAP_MODES),
  )
  const [heatmapSpan, setHeatmapSpanState] = useState(() =>
    readStoredChoice(HEATMAP_SPAN_KEY, HEATMAP_SPAN_DEFAULT, HEATMAP_SPANS),
  )
  const [markerColors, setMarkerColorsState] = useState(() =>
    readStoredChoice(MARKER_COLORS_KEY, MARKER_COLORS_DEFAULT, MARKER_COLORS_MODES),
  )
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
    showNames,
    toggleNames,
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
    showPlacements,
    togglePlacements,
    showUnnamedPlacements,
    toggleUnnamedPlacements,
    showDroppedPlacements,
    toggleDroppedPlacements,
    showWeaponPads,
    toggleWeaponPads,
    showFlagCarries,
    toggleFlagCarries,
    markerColors,
    setMarkerColors,
    speed,
    setSpeed,
  }
}
