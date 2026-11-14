/**
 * useReplayLivesFlags — LES RÉGLAGES DU GROUPE « VIES » du rejeu : trois calques (apparitions,
 * morts, tueurs) et deux affinages COMMUNS (qui, quand). Chacun est une préférence persistée
 * (`replayPreferences.ts`) lue au montage ; `useReplaySettings` en reprend l'objet tel quel.
 *
 * LES TROIS CALQUES SONT ÉTEINTS PAR DÉFAUT (sauf préréglage retenu qui les allume, cf. `livesDefault`), et
 * ce n'est pas un chantier laissé en plan (CLAUDE.md n°11) : « Complet » est l'état d'OUVERTURE de la page (cf. `viewPresetLogic`), et ces traces
 * s'ajoutent à des pions déjà nombreux. Ils sont complets ; les préréglages « Combat »
 * (morts, tueurs) et « Déplacements » (apparitions) les allument.
 *
 * LES AFFINAGES NE SONT PAS DES CALQUES : ils restreignent ce que les trois montrent, et un
 * préréglage ne les touche pas. Leurs défauts (tous, toute la partie) sont le comportement le plus
 * simple à lire. Une valeur écrite par une autre version, ou à la main, n'entre jamais dans l'état
 * (`readStoredChoice` la refuse).
 */
import { useCallback, useState } from 'react'

import {
  DEFAULT_LIVES_WHEN,
  DEFAULT_LIVES_WHO,
  LIVES_WHEN,
  LIVES_WHO,
  type LivesWhen,
  type LivesWho,
} from '../model/livesMarks'
import { persistPreference, readStoredChoice, readViewPreset } from './replayPreferences'
import { usePersistedFlag } from './usePersistedFlag'
import { PRESET_LAYERS, type LayerKey } from './viewPresetLogic'

const SHOW_SPAWNS_KEY = 'replay-show-spawns'
const SHOW_DEATHS_KEY = 'replay-show-deaths'
const SHOW_KILLERS_KEY = 'replay-show-killers'
const LIVES_WHO_KEY = 'replay-lives-who'
const LIVES_WHEN_KEY = 'replay-lives-when'

/**
 * LE DÉFAUT D'UN CALQUE DES VIES, tant que son interrupteur n'a jamais été écrit : éteint, SAUF si le
 * préréglage retenu l'allume.
 *
 * POURQUOI (relecture du 2026-11-14). Avant ces trois calques, « Combat » et « Déplacements » ne les
 * connaissaient pas : un lecteur qui avait choisi l'un d'eux a ses douze autres calques à l'état du
 * préréglage et ces trois clés ABSENTES. Éteintes, elles ne correspondent plus au préréglage, et la
 * vue active passait à « Personnalisé » sans que le lecteur n'ait rien touché. On applique donc au
 * préréglage retenu ce que la table lui donne pour les nouveaux calques : sa vue reste la sienne, et
 * c'est la plus petite règle qui ne recopie aucun état (la clé n'est pas écrite, une bascule à la main
 * la fixe ensuite comme toute préférence). Sans préréglage retenu, ou pour « Complet » et
 * « Objectif » : éteints, l'état d'ouverture de la page.
 */
function livesDefault(layer: LayerKey): boolean {
  const remembered = readViewPreset()
  return remembered !== undefined && PRESET_LAYERS[remembered].includes(layer)
}

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
}

export function useReplayLivesFlags(): ReplayLivesFlags {
  // Lus UNE fois, au montage : `usePersistedFlag` ne s'en sert qu'à l'initialisation.
  const [defaults] = useState(() => ({
    spawns: livesDefault('spawns'),
    deaths: livesDefault('deaths'),
    killers: livesDefault('killers'),
  }))
  const [showSpawns, toggleSpawns] = usePersistedFlag(SHOW_SPAWNS_KEY, defaults.spawns)
  const [showDeaths, toggleDeaths] = usePersistedFlag(SHOW_DEATHS_KEY, defaults.deaths)
  const [showKillers, toggleKillers] = usePersistedFlag(SHOW_KILLERS_KEY, defaults.killers)
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
    livesWho, setLivesWho, livesWhen, setLivesWhen,
  }
}
