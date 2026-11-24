/**
 * useReplayLivesFlags — LES RÉGLAGES DU GROUPE « VIES » du rejeu : trois calques (apparitions,
 * morts, tueurs) et deux affinages COMMUNS (qui, quand). Chacun est une préférence persistée
 * (`replayPreferences.ts`) lue au montage ; `useReplaySettings` en reprend l'objet tel quel.
 *
 * LES TROIS CALQUES SONT ÉTEINTS PAR DÉFAUT, et ce n'est pas un chantier laissé en plan (CLAUDE.md n°11) : ces traces
 * s'ajoutent à des pions déjà nombreux. Ils sont complets ; les préréglages « Combat » (morts, tueurs) et
 * « Déplacements » (apparitions) les allument.
 *
 * LES AFFINAGES NE SONT PAS DES CALQUES : ils restreignent ce que les trois montrent, et un
 * préréglage ne les touche pas. Leurs défauts (tous, toute la partie) sont le comportement le plus
 * simple à lire. Une valeur écrite par une autre version, ou à la main, n'entre jamais dans l'état
 * (`readStoredChoice` la refuse).
 */
import { useCallback, useState } from 'react'

import {
  DEFAULT_FORECAST_DRAW,
  FORECAST_DRAWS,
  type ForecastDraw,
} from '../model/respawnForecast.logic'
import {
  DEFAULT_LIVES_WHEN,
  DEFAULT_LIVES_WHO,
  LIVES_WHEN,
  LIVES_WHO,
  type LivesWhen,
  type LivesWho,
} from '../model/livesMarks'
import { persistPreference, readStoredChoice } from './replayPreferences'
import { usePersistedFlag } from './usePersistedFlag'

const SHOW_SPAWNS_KEY = 'replay-show-spawns'
const SHOW_DEATHS_KEY = 'replay-show-deaths'
const SHOW_KILLERS_KEY = 'replay-show-killers'
const LIVES_WHO_KEY = 'replay-lives-who'
const LIVES_WHEN_KEY = 'replay-lives-when'
const SHOW_RESPAWN_FORECAST_KEY = 'replay-show-respawn-forecast'
const RESPAWN_FORECAST_DRAW_KEY = 'replay-respawn-forecast-draw'

/**
 * LA PROBABILITÉ DE RÉAPPARITION EST ALLUMÉE PAR DÉFAUT (maquette validée le 2026-11-16) : elle n'existe
 * que pendant le compte à rebours d'un joueur mort, puis s'efface — elle ne charge pas la carte le reste
 * du temps — et c'est elle qu'on vient lire. Allumée aussi dans « Complet », l'état d'ouverture.
 */
const SHOW_RESPAWN_FORECAST_DEFAULT = true

/** Les traces des vies (apparitions, morts, tueurs) sont éteintes tant que le lecteur ne les a pas allumées. */
const SHOW_TRACE_DEFAULT = false

/** Les interrupteurs et affinages du groupe « Vies » : un état et sa bascule (ou son setter). */
export interface ReplayLivesFlags {
  /** Apparitions : une goutte creuse orientée à la première image de chaque vie. */
  showSpawns: boolean
  toggleSpawns: () => void
  /** Morts : une croix au lieu de la mort. */
  showDeaths: boolean
  toggleDeaths: () => void
  /** Tueurs : un triangle creux au tueur et un trait vers la victime. */
  showKillers: boolean
  toggleKillers: () => void
  /** Affinage commun « qui » — toujours une valeur de `LIVES_WHO`. */
  livesWho: LivesWho
  setLivesWho: (who: LivesWho) => void
  /** Affinage commun « quand » — toujours une valeur de `LIVES_WHEN`. */
  livesWhen: LivesWhen
  setLivesWhen: (when: LivesWhen) => void
  /** Probabilité de réapparition : zones teintées et point réel pendant le compte à rebours d'un joueur mort. */
  showRespawnForecast: boolean
  toggleRespawnForecast: () => void
  /** Ce qu'elle dessine — toujours une valeur de `FORECAST_DRAWS` (défaut : zones et points). */
  respawnForecastDraw: ForecastDraw
  setRespawnForecastDraw: (draw: ForecastDraw) => void
}

export function useReplayLivesFlags(): ReplayLivesFlags {
  const [showSpawns, toggleSpawns] = usePersistedFlag(SHOW_SPAWNS_KEY, SHOW_TRACE_DEFAULT)
  const [showDeaths, toggleDeaths] = usePersistedFlag(SHOW_DEATHS_KEY, SHOW_TRACE_DEFAULT)
  const [showKillers, toggleKillers] = usePersistedFlag(SHOW_KILLERS_KEY, SHOW_TRACE_DEFAULT)
  const [showRespawnForecast, toggleRespawnForecast] = usePersistedFlag(
    SHOW_RESPAWN_FORECAST_KEY,
    SHOW_RESPAWN_FORECAST_DEFAULT,
  )
  const [respawnForecastDraw, setDrawState] = useState(() =>
    readStoredChoice(RESPAWN_FORECAST_DRAW_KEY, DEFAULT_FORECAST_DRAW, FORECAST_DRAWS),
  )
  const setRespawnForecastDraw = useCallback((next: ForecastDraw) => {
    setDrawState(next)
    persistPreference(RESPAWN_FORECAST_DRAW_KEY, next)
  }, [])
  const [livesWho, setWhoState] = useState(() => readStoredChoice(LIVES_WHO_KEY, DEFAULT_LIVES_WHO, LIVES_WHO))
  const [livesWhen, setWhenState] = useState(() => readStoredChoice(LIVES_WHEN_KEY, DEFAULT_LIVES_WHEN, LIVES_WHEN))
  const setLivesWho = useCallback((next: LivesWho) => {
    setWhoState(next)
    persistPreference(LIVES_WHO_KEY, next)
  }, [])
  const setLivesWhen = useCallback((next: LivesWhen) => {
    setWhenState(next)
    persistPreference(LIVES_WHEN_KEY, next)
  }, [])
  return {
    showSpawns, toggleSpawns, showDeaths, toggleDeaths, showKillers, toggleKillers,
    livesWho, setLivesWho, livesWhen, setLivesWhen, showRespawnForecast, toggleRespawnForecast,
    respawnForecastDraw, setRespawnForecastDraw,
  }
}
