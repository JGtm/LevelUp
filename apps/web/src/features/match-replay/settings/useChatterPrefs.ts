/**
 * useChatterPrefs — LES RÉGLAGES DES RÉPLIQUES des Spartans : trois interrupteurs d'affichage (bulle de texte, icône de voix,
 * répliques dans le fil), cinq familles et une fréquence. Chacun est une préférence persistée (`replayPreferences.ts`) lue au
 * montage ; `useReplaySettings` en reprend l'objet tel quel, et les lecteurs hors du panneau (le fil, la voix) rappellent ce
 * hook : les abonnés d'une clé font bouger toutes ses copies ensemble (`subscribePreference`).
 *
 * TOUT EST ALLUMÉ PAR DÉFAUT, toutes familles, toutes fréquences (décision du 2026-11-24) : ce n'est pas un chantier laissé
 * en plan (CLAUDE.md n°11), les affichages sont complets et leurs interrupteurs sont des réglages offerts au lecteur. La voix,
 * elle, est une catégorie du son (`useReplaySound`) : elle ne joue que si le son du rejeu est activé, et le son reste coupé
 * par défaut.
 *
 * AUCUN RÉGLAGE DE LANGUE : texte et voix suivent la langue de l'interface (`model/chatterTable.ts`).
 */
import { useCallback, useEffect, useMemo, useState } from 'react'

import { CHATTER_FILTER_ALL, CHATTER_FREQUENCIES, type ChatterFamily, type ChatterFilter, type ChatterFrequency } from '../model/chatterLines'
import { persistPreference, readStoredChoice, subscribePreference } from './replayPreferences'
import { usePersistedFlag } from './usePersistedFlag'

const SHOW_BUBBLES_KEY = 'replay-show-chatter-bubbles'
const SHOW_ICON_KEY = 'replay-show-chatter-icon'
const SHOW_FEED_KEY = 'replay-show-chatter-feed'
const FREQUENCY_KEY = 'replay-chatter-frequency'

/** La clé de chaque famille : une préférence par famille, allumée par défaut. */
const FAMILY_KEYS: Readonly<Record<ChatterFamily, string>> = {
  enemy: 'replay-chatter-family-enemy',
  shields: 'replay-chatter-family-shields',
  squadDown: 'replay-chatter-family-squad-down',
  praise: 'replay-chatter-family-praise',
  other: 'replay-chatter-family-other',
}

const DEFAULT_FREQUENCY: ChatterFrequency = 'all'

export interface ChatterPrefs {
  /** Une bulle de texte sur le joueur qui parle. */
  showBubbles: boolean
  toggleBubbles: () => void
  /** Une icône de voix sur le joueur qui parle (quand sa réplique a une voix dans la langue). */
  showIcon: boolean
  toggleIcon: () => void
  /** Les répliques rejoignent le fil du match. */
  showFeed: boolean
  toggleFeed: () => void
  /** Les familles montrées. */
  families: Readonly<Record<ChatterFamily, boolean>>
  toggleFamily: (family: ChatterFamily) => void
  /** Toutes les phrases, sans les plus fréquentes, ou les rares seulement. */
  frequency: ChatterFrequency
  setFrequency: (frequency: ChatterFrequency) => void
}

export function useChatterPrefs(): ChatterPrefs {
  const [showBubbles, toggleBubbles] = usePersistedFlag(SHOW_BUBBLES_KEY, true)
  const [showIcon, toggleIcon] = usePersistedFlag(SHOW_ICON_KEY, true)
  const [showFeed, toggleFeed] = usePersistedFlag(SHOW_FEED_KEY, true)
  const [enemy, toggleEnemy] = usePersistedFlag(FAMILY_KEYS.enemy, true)
  const [shields, toggleShields] = usePersistedFlag(FAMILY_KEYS.shields, true)
  const [squadDown, toggleSquadDown] = usePersistedFlag(FAMILY_KEYS.squadDown, true)
  const [praise, togglePraise] = usePersistedFlag(FAMILY_KEYS.praise, true)
  const [other, toggleOther] = usePersistedFlag(FAMILY_KEYS.other, true)
  const [frequency, setFrequencyState] = useState<ChatterFrequency>(() =>
    readStoredChoice(FREQUENCY_KEY, DEFAULT_FREQUENCY, CHATTER_FREQUENCIES),
  )
  // Le panneau et les lecteurs (fil, voix) lisent la même clé : l'abonnement les fait bouger ensemble.
  useEffect(
    () =>
      subscribePreference(FREQUENCY_KEY, (raw) => {
        setFrequencyState((CHATTER_FREQUENCIES as readonly string[]).includes(raw) ? (raw as ChatterFrequency) : DEFAULT_FREQUENCY)
      }),
    [],
  )
  const setFrequency = useCallback((next: ChatterFrequency) => {
    setFrequencyState(next)
    persistPreference(FREQUENCY_KEY, next)
  }, [])
  const families = useMemo(() => ({ enemy, shields, squadDown, praise, other }), [enemy, shields, squadDown, praise, other])
  const toggleFamily = useCallback(
    (family: ChatterFamily) => {
      const toggles: Record<ChatterFamily, () => void> = {
        enemy: toggleEnemy, shields: toggleShields, squadDown: toggleSquadDown, praise: togglePraise, other: toggleOther,
      }
      toggles[family]()
    },
    [toggleEnemy, toggleShields, toggleSquadDown, togglePraise, toggleOther],
  )
  return { showBubbles, toggleBubbles, showIcon, toggleIcon, showFeed, toggleFeed, families, toggleFamily, frequency, setFrequency }
}

/** Les filtres d'affichage (familles et fréquence) que les réglages demandent. Même objet tant que rien ne change. */
export function chatterFilterOf(prefs: Pick<ChatterPrefs, 'families' | 'frequency'>): ChatterFilter {
  const { families, frequency } = prefs
  if (frequency === CHATTER_FILTER_ALL.frequency && Object.values(families).every(Boolean)) return CHATTER_FILTER_ALL
  return { families, frequency }
}
