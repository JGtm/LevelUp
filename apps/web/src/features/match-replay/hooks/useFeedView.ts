/**
 * useFeedView — LA LISTE DU FIL VUE À L'INSTANT LU, sans aucun rendu : tout affichage du fil (la
 * colonne du Standard, un fil posé sur le plan, un extrait dans la fiche) lit ici la MÊME liste, au lieu
 * de la recomposer chacun de son côté.
 *
 * CE QU'ELLE COMPOSE. Le fil de la page (`entries`, jamais modifié : la frise et les effets de mort
 * le lisent entier) plus les lignes d'événement du document (objectifs, prises, véhicules), passé au
 * filtre APPLIQUÉ — le choix retenu dans le navigateur s'il est proposé pour ce match, Tout sinon —,
 * puis coupé à l'instant lu. Le filtre est une préférence partagée : tout affichage qui la lit suit
 * le choix fait dans un autre.
 *
 * L'ORDRE EST CELUI DU FIL : de la plus récente à la plus ancienne. Un affichage qui veut la plus
 * récente en bas renverse la liste ; il ne la refiltre pas.
 *
 * LA LIGNE CHOISIE vient de la mise en valeur du magasin de lecture (`selectedKey`) ; `select` est le
 * geste de clic — la mise en valeur d'abord, puis le saut (pause sur l'image du frag). Il vaut `null`
 * quand le fil est en lecture seule : pas de magasin, ou pas de document où viser une image.
 */
import { useCallback, useMemo } from 'react'

import type { FilmAllegiance } from '../../../lib/replay/filmAllegiance'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import type { ReplayLocale } from '../i18n/i18n'
import { withEventLines } from '../model/feedEventLines'
import {
  availableFeedFilters,
  effectiveFeedFilter,
  filterFeedEntries,
  type FeedFilterId,
  type FeedFilterOption,
} from '../model/feedFilterLogic'
import { chatterShownFor } from '../model/chatterLines'
import { feedAt, type ReplayFeedEntry } from '../model/killFeedLogic'
import { focusTargetOf } from '../model/killFocusLogic'
import { NO_WEAPON_ICONS, type WeaponIconsByKey } from '../model/killSourceIcons'
import { usePlaybackFocus, type PlaybackStore } from '../model/playbackStore'
import type { HiddenPlayers } from '../model/playerPanelLogic'
import { useFeedFilter } from '../settings/useFeedFilter'
import { useChatterFeedLines } from './useChatterLines'
import { useFeedEventEntries } from './useFeedEventEntries'

export interface FeedViewInput {
  /** Le fil aligné de la page (`buildFeedEntries`), TRIÉ : la coupe à l'instant lu s'arrête au premier postérieur. */
  entries: ReplayFeedEntry[]
  /** L'instant lu, sur l'axe BRUT du film : il décide de ce qui est visible. */
  nowMs: number
  /** Le document de rejeu, source des lignes d'événement ; absent : le fil ne porte que `entries`. */
  doc?: ReplayDocumentReady | null
  locale: ReplayLocale
  /** L'allégeance du film : ses camps écrivent le score des lignes de fin de manche. */
  allegiance: FilmAllegiance
  /** Les pictogrammes d'arme du titre, pour les lignes de destruction de véhicule. */
  weaponIcons?: WeaponIconsByKey
  /** Le magasin de lecture de la page : sans lui, aucune ligne n'est choisie ni cliquable. */
  playbackStore?: PlaybackStore
  /** Les joueurs masqués (« Seul », « Masquer ») : une réplique est à son joueur, elle se retire du fil avec lui. Absent = tout se montre. */
  hiddenPlayers?: HiddenPlayers | null
}

export interface FeedView {
  /** Les lignes visibles à l'instant lu, filtrées, de la plus récente à la plus ancienne. */
  lines: ReplayFeedEntry[]
  /** Les filtres proposés pour ce match, avec leur nombre de lignes dans tout le match. */
  options: FeedFilterOption[]
  /** Le filtre appliqué, et le geste qui le change (et le retient). */
  filter: FeedFilterId
  chooseFilter: (id: FeedFilterId) => void
  /** Vrai quand le filtre appliqué n'a AUCUNE ligne dans tout le match (et pas seulement à cet instant). */
  matchHasNone: boolean
  /** La clé de la ligne mise en valeur, ou `null`. */
  selectedKey: string | null
  /** Le clic sur une ligne de frag ; `null` quand le fil est en lecture seule. */
  select: ((entry: ReplayFeedEntry) => void) | null
}

export function useFeedView({
  entries,
  nowMs,
  doc,
  locale,
  allegiance,
  weaponIcons = NO_WEAPON_ICONS,
  playbackStore,
  hiddenPlayers = null,
}: FeedViewInput): FeedView {
  const campIds = useMemo(() => allegiance.camps.map((c) => c.team), [allegiance])
  const chatter = useChatterFeedLines(doc, locale)
  const events = useFeedEventEntries(doc, locale, weaponIcons, campIds, chatter)
  const fullFeed = useMemo(() => withEventLines(entries, events), [entries, events])
  const options = useMemo(() => availableFeedFilters(fullFeed), [fullFeed])
  const [storedFilter, chooseFilter] = useFeedFilter()
  const filter = effectiveFeedFilter(storedFilter, options)
  const shown = useMemo(() => {
    const filtered = filterFeedEntries(fullFeed, filter)
    const chatterShown = chatterShownFor(hiddenPlayers)
    // Une réplique est à son joueur : masqué, il emporte sa ligne du fil (les autres lignes ne changent pas).
    return chatterShown
      ? filtered.filter((e) => e.event?.family !== 'replies' || chatterShown({ xuid: e.event.xuid }))
      : filtered
  }, [fullFeed, filter, hiddenPlayers])
  const lines = useMemo(() => feedAt(shown, nowMs), [shown, nowMs])
  const matchHasNone = (options.find((o) => o.id === filter)?.count ?? 0) === 0

  const selectedKey = usePlaybackFocus(playbackStore)?.entryKey ?? null
  const selectable = playbackStore !== undefined && doc != null
  const select = useCallback(
    (entry: ReplayFeedEntry) => {
      const target = playbackStore && doc ? focusTargetOf(entry, doc) : null
      if (!playbackStore || !target) return
      playbackStore.focusKill(target)
      playbackStore.requestSeek(target.frame)
    },
    [playbackStore, doc],
  )

  return { lines, options, filter, chooseFilter, matchHasNone, selectedKey, select: selectable ? select : null }
}
