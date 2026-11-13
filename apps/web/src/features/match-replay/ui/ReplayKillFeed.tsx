/**
 * ReplayKillFeed — le fil des éliminations de la page de rejeu, SYNCHRONISÉ sur l'horloge du
 * rejeu : la liste, le menu de filtre et les états vides. Les formes de ligne vivent dans
 * `ReplayFeedLines.tsx`.
 *
 *  - FIL PERMANENT : toutes les lignes depuis le début du match, la plus récente en tête, dans
 *    une liste qui DÉFILE — les événements passés restent lisibles. On ne remonte en tête que si
 *    le lecteur y était déjà : une nouvelle ligne ne lui arrache pas sa position de lecture.
 *  - CHAQUE LIGNE : l'HORODATAGE EN TÊTE, dans une gouttière fixe ; puis TUEUR (encre de son
 *    camp), ICÔNE de l'arme, VICTIME (encre de SON camp), les MÉDAILLES du kill, puis l'ASSISTANCE
 *    quand elle est nommée. Médaille seule, mort neutre, entrée/sortie de partie : leurs propres
 *    formes.
 *  - L'ENCRE DE CAMP D'UN NOM EST L'ALLÉGEANCE DU FILM (`FilmAllegiance.ofXuid`, l'équipe du film
 *    de l'acteur comparée à celle du joueur regardé). Un acteur dont le film tait l'équipe — ou
 *    absent du film — prend l'encre neutre : aucun repli deviné.
 *  - LE FIL ALIGNÉ LUI EST DONNÉ, IL NE L'ASSEMBLE PLUS : `buildFeedEntries` est appelé UNE fois
 *    par la page, et sa sortie sert ici ET aux pistes de la frise — la garantie que les deux vues
 *    montrent LE MÊME fil.
 *  - SA HAUTEUR EST CELLE DE LA RANGÉE, JAMAIS LA SIENNE : le fil vit dans une colonne à côté de
 *    la carte, qui impose la hauteur ; il occupe ce qu'on lui donne et fait défiler à l'intérieur.
 *  - UNE LIGNE, UNE SEULE, ET SANS RETOUR : ce qui déborde est tronqué, jamais renvoyé à la ligne
 *    (cf. `model/feedRow.ts`). Aucun compteur total dans le titre.
 *
 * LE MENU DE FILTRE ET LES LIGNES D'ÉVÉNEMENT. Le titre est un menu (`ReplayFeedFilterMenu`), et
 * la liste AFFICHÉE est le fil complet (`entries`) plus les lignes d'événement du document
 * (objectifs en bandeaux, prises d'armes et de bonus, montées en véhicule), passé au filtre.
 * `entries` n'est JAMAIS modifié : la frise et les effets de mort lisent ce fil-là, entier. Deux
 * états vides distincts, en bloc pointillé : « Rien à cet instant » (le filtre a des lignes plus
 * loin dans le match) et « Aucune entrée pour ce match » (il n'en a aucune).
 */
import { useEffect, useMemo, useRef } from 'react'

import { EmptyStateNotice } from '@/components/ui/empty-state'
import { teamTokenCssVar } from '@/features/match-view/teamSeriesColor'
import type { MatchScoreboardRow } from '@/lib/api/types'
import type { FilmAllegiance } from '@/lib/replay/filmAllegiance'
import { normalizeGamertagKey } from '@/lib/players/displayName'

import type { KillerZoneOf } from '../hooks/useKillerZones'
import { useFeedEventEntries } from '../hooks/useFeedEventEntries'
import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'
import { withEventLines } from '../model/feedEventLines'
import {
  availableFeedFilters,
  effectiveFeedFilter,
  filterFeedEntries,
} from '../model/feedFilterLogic'
import { feedAt, type ReplayFeedEntry } from '../model/killFeedLogic'
import { focusTargetOf } from '../model/killFocusLogic'
import { NO_WEAPON_ICONS, type WeaponIconsByKey } from '../model/killSourceIcons'
import { usePlaybackFocus, type PlaybackStore } from '../model/playbackStore'
import { displayClockMs, type ReplayWindowBounds } from '../model/replayWindow'
import { useFeedFilter } from '../settings/useFeedFilter'
import { NO_MARKS, type PlayerMarkKind } from '../../../lib/replay/playerMarks'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'

/** Réexporté pour `assistMarkIcon.guard.test.ts` : le stem a déménagé, sa surface non. */
export { ASSIST_ICON_STEM }
import { ASSIST_ICON_STEM } from './ReplayAssistMark'
import { ReplayFeedFilterMenu } from './ReplayFeedFilterMenu'
import { FeedLine, type GamertagIndex, type InkOf } from './ReplayFeedLines'

/** Seuil sous lequel le lecteur est considéré « en tête de fil » (POC : 4 px). */
const AT_TOP_PX = 4

interface Props {
  /**
   * LE FIL ALIGNÉ, assemblé par la page (`buildFeedEntries`) : kills recalés sur l'axe du
   * rejeu, morts neutres et médailles seules, triés. Trié est un PRÉ-REQUIS — `feedAt` coupe
   * au premier élément postérieur à l'instant courant.
   */
  entries: ReplayFeedEntry[]
  /** Instant courant sur l'axe BRUT du film (il décide du VISIBLE) ; `playWindow` recale l'AFFICHÉ (D-A2). */
  nowMs: number
  playWindow: ReplayWindowBounds | null
  scoreboard: MatchScoreboardRow[] | null | undefined
  /** Les NOMS de la feuille, par xuid de base — jamais l'allégeance, qui vient du film. */
  xuidMeta: GamertagIndex
  /** L'allégeance lue dans le film, vue du joueur regardé (`model.allegiance`) : l'encre des noms. */
  allegiance: FilmAllegiance
  locale: ReplayLocale
  /**
   * Marques d'identité par xuid (« moi », « ami ») — la même grammaire que les fiches et la
   * carte. Absentes = aucun glyphe, jamais une marque devinée.
   */
  marks?: ReadonlyMap<string, PlayerMarkKind>
  /**
   * La zone nommée où se trouvait le tueur de chaque frag (`useKillerZones`), par clé de ligne.
   * Absent = la carte n'a aucune zone : les lignes de frag ne réservent pas de gouttière. Présent,
   * la gouttière suit l'horloge sur CHAQUE ligne de frag et reste vide quand la zone est inconnue.
   */
  zoneOf?: KillerZoneOf
  /**
   * Le document de rejeu, source des lignes d'événement (objectifs, prises, montées). Absent = le
   * fil ne porte que les éliminations, médailles, morts et présences de `entries`.
   */
  doc?: ReplayDocumentReady | null
  /**
   * Le magasin de lecture de la page : il rend les lignes de FRAG cliquables (le clic demande le
   * saut et pose la mise en valeur) et dit laquelle est choisie. Absent, ou sans `doc` où viser
   * une image : le fil reste en lecture seule.
   */
  playbackStore?: PlaybackStore
  /**
   * Les pictogrammes d'arme du titre par weapon_key (`useKillSourceIcons`) : la ligne « a détruit »
   * y prend le dessin de l'arme qui a détruit le véhicule. Absent ou vide (chargement, erreur) : les
   * lignes s'affichent sans pictogramme.
   */
  weaponIcons?: WeaponIconsByKey
}

export function ReplayKillFeed({
  entries, nowMs, playWindow, scoreboard, xuidMeta, allegiance, locale, marks, zoneOf, doc, playbackStore, weaponIcons = NO_WEAPON_ICONS,
}: Props) {
  const t = REPLAY_TEXT[locale]
  // UNE SEULE RÈGLE POUR TOUTES LES FORMES DE LIGNE : l'allégeance du film, `team-ally` /
  // `team-enemy`, l'encre neutre quand elle est inconnue (`teamTokenCssVar(null)`).
  const inkOf = useMemo<InkOf>(() => (xuid) => teamTokenCssVar(allegiance.ofXuid(xuid)), [allegiance])
  // L'ASSISTANT N'A PAS DE XUID dans l'événement du film : il n'est nommé que par son
  // gamertag (cf. `KillEvent`). Pour qu'il porte la MÊME marque que les autres, les marques
  // sont réindexées par gamertag normalisé — via le scoreboard, la seule table qui porte les
  // deux clés à la fois. Sans lui, l'assistant serait le seul nom du fil sans glyphe.
  const marksByGamertag = useMemo(() => {
    const byName = new Map<string, PlayerMarkKind>()
    for (const r of scoreboard ?? []) {
      const kind = (marks ?? NO_MARKS).get(r.xuid)
      if (kind) byName.set(normalizeGamertagKey(r.gamertag), kind)
    }
    return byName
  }, [scoreboard, marks])

  // LA LISTE AFFICHÉE : le fil complet plus les lignes d'événement, puis le filtre. `entries`
  // (le fil de la page) n'est jamais modifié.
  const events = useFeedEventEntries(doc, locale, weaponIcons)
  const fullFeed = useMemo(() => withEventLines(entries, events), [entries, events])
  const options = useMemo(() => availableFeedFilters(fullFeed), [fullFeed])
  const [storedFilter, chooseFilter] = useFeedFilter()
  const filter = effectiveFeedFilter(storedFilter, options)
  const shown = useMemo(() => filterFeedEntries(fullFeed, filter), [fullFeed, filter])
  const visibles = feedAt(shown, nowMs)
  const matchHasNone = (options.find((o) => o.id === filter)?.count ?? 0) === 0

  // LE CLIC SUR UNE LIGNE DE FRAG : la ligne choisie vient de la mise en valeur du magasin, et le
  // geste fait deux demandes — la mise en valeur d'abord, puis le saut (pause sur l'image du frag).
  // La cible se lit sur la ligne ORIGINALE : la copie affichée porte l'horloge recalée.
  const selectedKey = usePlaybackFocus(playbackStore)?.entryKey ?? null
  const selectKill = (entry: ReplayFeedEntry) => {
    const target = playbackStore && doc ? focusTargetOf(entry, doc) : null
    if (!playbackStore || !target) return
    playbackStore.focusKill(target)
    playbackStore.requestSeek(target.frame)
  }

  // POSITION DE LECTURE (règle du POC) : on ne colle en tête que si le lecteur y était.
  const listRef = useRef<HTMLUListElement>(null)
  const atTopRef = useRef(true)
  const count = visibles.length
  useEffect(() => {
    const el = listRef.current
    if (el && atTopRef.current) el.scrollTop = 0
  }, [count])

  return (
    // `relative` ancre la liste déroulante du menu ; le titre est un bandeau HUD neutre et les
    // lignes se séparent par leur filet, pas par une boîte.
    <div className="relative flex h-full min-h-0 flex-col">
      <ReplayFeedFilterMenu options={options} active={filter} onChange={chooseFilter} locale={locale} />
      <ul
        ref={listRef}
        onScroll={(e) => {
          atTopRef.current = e.currentTarget.scrollTop <= AT_TOP_PX
        }}
        className="mt-2 flex min-h-[4.5rem] flex-1 flex-col overflow-y-auto"
        aria-live="off"
      >
        {count === 0 && (
          <li className="shrink-0 px-1">
            <EmptyStateNotice
              size="compact"
              title={matchHasNone ? t.feedEmptyMatch : t.killFeedEmpty}
              description={matchHasNone ? t.feedEmptyMatchHint : t.killFeedEmptyHint}
            />
          </li>
        )}
        {visibles.map((entry) => (
          <FeedLine
            key={entry.key}
            entry={{ ...entry, replayMs: displayClockMs(entry.replayMs, playWindow) }}
            inkOf={inkOf}
            xuidMeta={xuidMeta}
            marks={marks ?? NO_MARKS}
            marksByGamertag={marksByGamertag}
            locale={locale}
            zoneOf={zoneOf}
            focus={
              playbackStore && doc
                ? { selected: selectedKey === entry.key, onSelect: () => selectKill(entry) }
                : undefined
            }
          />
        ))}
      </ul>
    </div>
  )
}
