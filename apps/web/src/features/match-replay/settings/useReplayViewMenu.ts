/**
 * useReplayViewMenu — LE BOUTON « VUE » et son panneau, assemblés en un objet.
 *
 * Le canvas garde le DESSIN ; ce hook porte le formulaire. Il ne réinvente aucun réglage : les états et leur
 * persistance restent dans `useReplaySettings` (calques, effets, chaleur, couleur des points), `useReplayRelief` (fond,
 * traits, étage), `useReplayOrientation`, `useReplaySound` (son, volume, catégories) et le filtre du fil. Il range ces
 * états sous la table des réglages (`bindSettings`), compose ce que le match propose (`SettingsContext`), et branche
 * les vues (`useViewsController`) : préréglages souples, Mes vues, épingles.
 *
 * LA VUE ACTIVE N'EST PAS UN ÉTAT DE RÉGLAGES : elle se déduit en comparant les réglages à la table de la vue
 * (« modifié » sur les seuls réglages que le match propose). Une bascule à la main ne l'efface pas, elle la dit modifiée.
 *
 * LES « available » NE SONT PAS DÉCORATIFS : c'est la règle du dépôt « pas de commande quand il n'y a rien à
 * commander » (un film sans socle publié ne montre pas la bascule des socles), et c'est ce qui interdit à un
 * préréglage d'écrire un réglage que le match ne porte pas.
 *
 * LE PANNEAU EST FERMÉ PAR DÉFAUT et s'ouvre par le bouton « Vue » de la carte. Les réglages qu'il porte SURVIVENT à
 * la page (persistés) ; seule l'ouverture repart de zéro.
 */
import { useCallback, useMemo, useRef, useState, type RefObject } from 'react'

import type { MapBackgroundVariant } from '@/lib/replay/mapBackgroundVariant'

import { useFeedFilterChoice } from '../hooks/useFeedFilterChoice'
import type { ReplayRelief } from '../hooks/useReplayRelief'
import type { ReplayLocale } from '../i18n/i18n'
import type { FilmAllegiance } from '../../../lib/replay/filmAllegiance'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import type { ReplayFeedEntry } from '../model/killFeedLogic'
import type { ReplaySound } from '../sound/useReplaySound'
import type { FilmLayerAvailability } from './filmLayerAvailability'
import { matchModeKey, type SettingsContext } from './settingsAvailability'
import { bindSettings } from './settingsBinding'
import { activeViewLabel, type ActiveViewLabel } from './viewLabel'
import { useReplayDimension, type ReplayDimensionState } from './useReplayDimension'
import type { ReplayOrientation } from './useReplayOrientation'
import type { ReplaySettings } from './useReplaySettings'
import { useViewsController } from './useViewsController'
import type { ReplayViewPanelModel } from './viewPanelModel'
import { REPLAY_TEXT } from '../i18n/i18n'

/** Ce que le canvas prête au panneau : les réglages, ce que le film porte, et les sorties. */
export interface ReplayViewMenuOptions {
  settings: ReplaySettings
  /** L'orientation de la carte, retenue par carte (`useReplayOrientation`). */
  orientation: ReplayOrientation
  sound: ReplaySound
  /** Ce que le film porte réellement, calque par calque. */
  available: FilmLayerAvailability
  /** Le fond de carte et son relief (`useReplayRelief`) : fonds proposés, fond retenu, traits, commande d'étage. */
  relief: ReplayRelief
  /** Le fond RÉELLEMENT posé : le plan porte déjà les murs, leur calque n'est alors pas proposé. */
  drawn: MapBackgroundVariant
  /** Le cadre pose le fil des éliminations sur le plan : « Fil posé sur le plan » n'est proposé qu'alors. */
  planFeedAvailable: boolean
  /** Le cadre peut poser la barre de lecture sur le plan : la barre fine n'est proposée qu'alors. */
  thinBarAvailable?: boolean
  /** La lecture de la carte de chaleur RÉELLEMENT servie, et si les morts sont localisables. */
  heat: { mode: 'presence' | 'kills'; killsAvailable: boolean }
  locale: ReplayLocale
  /** Ce qu'il faut pour compter les lignes de chaque filtre du fil : le film, le fil aligné, l'allégeance. */
  feed: { doc: ReplayDocumentReady; entries: readonly ReplayFeedEntry[]; allegiance: FilmAllegiance }
  /** L'identité du rejeu ouvert : la vue d'ouverture ne s'applique qu'une fois par rejeu. */
  matchKey: string
}

/** Ce que le canvas reçoit : l'état du panneau, ce qui l'ouvre, et de quoi le peindre. */
export interface ReplayViewMenuState {
  open: boolean
  /** Bascule d'ouverture, posée sur le bouton « Vue ». */
  toggle: () => void
  /** Ferme le panneau et rend le focus au bouton. */
  close: () => void
  /** Le bouton qui ouvre le panneau : exclu du « clic dehors », il reprend le focus. */
  buttonRef: RefObject<HTMLButtonElement | null>
  /** Le nom de la vue active, et son état modifié : le bouton les porte. */
  label: ActiveViewLabel
  /** Tout ce que le panneau et l'accès rapide lisent. */
  model: ReplayViewPanelModel
  /** Le commutateur 2D / 3D posé à gauche du bouton : n'existe que si le serveur offre la vue 3D. */
  dimension: ReplayDimensionState
}

/** Ce que le match, le cadre et le fond proposent, composé des disponibilités publiées par le canvas. */
function settingsContextOf(o: ReplayViewMenuOptions, feedFilters: readonly string[]): SettingsContext {
  const { relief } = o
  return {
    film: o.available,
    planFeedAvailable: o.planFeedAvailable,
    thinBarAvailable: o.thinBarAvailable ?? false,
    drawn: o.drawn,
    backgroundOptions: relief.options,
    reliefLinesAvailable: relief.lines.available,
    floor: relief.floor.state.state,
    orientationAvailable: o.orientation.available,
    soundAvailable: o.sound.available,
    heatKillsAvailable: o.heat.killsAvailable,
    feedFilters,
  }
}

export function useReplayViewMenu(o: ReplayViewMenuOptions): ReplayViewMenuState {
  const [open, setOpen] = useState(false)
  const buttonRef = useRef<HTMLButtonElement>(null)
  const dimension = useReplayDimension()
  const feed = useFeedFilterChoice(o.feed.doc, o.locale, o.feed.allegiance, o.feed.entries)
  const feedFilters = useMemo(() => feed.options.map((f) => f.id), [feed.options])
  const ctx = settingsContextOf(o, feedFilters)
  const binding = bindSettings({
    settings: o.settings, relief: o.relief, orientation: o.orientation, sound: o.sound,
    feed: { value: feed.value, retained: feed.retained, choose: feed.choose }, heatMode: o.heat.mode, followedKnown: o.available.lives.followedKnown,
  })
  const views = useViewsController({ ctx, binding, mode: matchModeKey(o.available), matchKey: o.matchKey })
  const toggle = useCallback(() => setOpen((v) => !v), [])
  // LA FERMETURE REPREND LE FOCUS : sans cela le focus retomberait au document et la navigation au clavier repartirait
  // du haut de la page.
  const close = useCallback(() => {
    setOpen(false)
    buttonRef.current?.focus({ preventScroll: true })
  }, [])
  const model: ReplayViewPanelModel = {
    locale: o.locale, ctx, values: binding.values, set: binding.set, views,
    floor: o.relief.floor, meshPercent: o.relief.meshPercent, feedOptions: feed.options,
  }
  const label = activeViewLabel(REPLAY_TEXT[o.locale].settings, views)
  return { open, toggle, close, buttonRef, label, model, dimension }
}
