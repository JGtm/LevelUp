/**
 * ReplayKillFeed — le fil des éliminations de la page de rejeu, SYNCHRONISÉ sur l'horloge du rejeu :
 * la liste, le menu de filtre et les états vides. Les formes de ligne vivent dans `ReplayFeedLines.tsx`
 * et la liste filtrée dans `hooks/useFeedView.ts`, que tout autre affichage du fil lit aussi.
 *
 *  - FIL PERMANENT : toutes les lignes depuis le début du match, la plus récente en tête, dans
 *    une liste qui DÉFILE — les événements passés restent lisibles. On ne remonte en tête que si
 *    le lecteur y était déjà : une nouvelle ligne ne lui arrache pas sa position de lecture.
 *  - CHAQUE LIGNE FAIT 28 PX, sans fond ni bordure. L'élimination est une grille à colonnes fixes :
 *    INSTANT, LIEU de la victime, TUEUR (encre de son camp), ICÔNE de l'arme, VICTIME (encre de SON
 *    camp), puis les MÉDAILLES. L'assistance et le porteur du drapeau sont dans l'infobulle. Médaille
 *    seule, mort neutre, entrée/sortie de partie, événements : leurs propres formes, sur la même
 *    gouttière d'instant (cotes : `model/cotesFeedLine.ts`).
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
 * LE MENU DE FILTRE. Le titre est un menu (`ReplayFeedFilterMenu`). Deux états vides distincts, en
 * bloc pointillé : « Rien à cet instant » (le filtre a des lignes plus loin dans le match) et
 * « Aucune entrée pour ce match » (il n'en a aucune).
 */
import { useEffect, useMemo, useRef } from 'react'

import { EmptyStateNotice } from '@/components/ui/empty-state'
import { teamTokenCssVar } from '@/features/match-view/teamSeriesColor'
import type { MatchScoreboardRow } from '@/lib/api/types'
import type { FilmAllegiance } from '@/lib/replay/filmAllegiance'

import type { FeedZoneOf } from '../hooks/useFeedZones'
import { useFeedView } from '../hooks/useFeedView'
import { useTeamCascades } from '../hooks/useTeamCascades'
import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'
import type { ReplayFeedEntry } from '../model/killFeedLogic'
import { NO_WEAPON_ICONS, type WeaponIconsByKey } from '../model/killSourceIcons'
import type { PlaybackStore } from '../model/playbackStore'
import { displayClockMs, type ReplayWindowBounds } from '../model/replayWindow'
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
   * La zone nommée où se trouvait la victime de chaque frag (`useFeedZones`), par clé de ligne.
   * Absent = la carte n'a aucune zone : les lignes de frag ne réservent pas de colonne de lieu.
   * Présent, la colonne suit l'instant sur CHAQUE ligne de frag et reste vide quand la zone est inconnue.
   */
  zoneOf?: FeedZoneOf
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
  const teams = useTeamCascades(scoreboard ?? undefined, allegiance, locale)
  const { lines, options, filter, chooseFilter, matchHasNone, selectedKey, select } = useFeedView({
    entries, nowMs, doc, locale, allegiance, weaponIcons, playbackStore,
  })

  // POSITION DE LECTURE (règle du POC) : on ne colle en tête que si le lecteur y était.
  const listRef = useRef<HTMLUListElement>(null)
  const atTopRef = useRef(true)
  const count = lines.length
  useEffect(() => {
    const el = listRef.current
    if (el && atTopRef.current) el.scrollTop = 0
  }, [count])

  return (
    // `relative` ancre la liste déroulante du menu ; le titre est un bandeau HUD neutre.
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
        {lines.map((entry) => (
          <FeedLine
            key={entry.key}
            // La copie affichée porte l'horloge recalée ; la cible d'un clic se lit sur la ligne ORIGINALE.
            entry={{ ...entry, replayMs: displayClockMs(entry.replayMs, playWindow) }}
            inkOf={inkOf}
            xuidMeta={xuidMeta}
            marks={marks ?? NO_MARKS}
            locale={locale}
            zoneOf={zoneOf}
            teams={teams}
            focus={select ? { selected: selectedKey === entry.key, onSelect: () => select(entry) } : undefined}
          />
        ))}
      </ul>
    </div>
  )
}
