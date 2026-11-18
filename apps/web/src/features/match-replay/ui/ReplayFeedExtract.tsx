/**
 * ReplayFeedExtract — LE PIED « ÉLIMINATIONS » DE LA FICHE DU STANDARD : le fil du match réduit à ses deux
 * dernières lignes, avec le lien « Fil complet » qui rend la colonne.
 *
 * LA MÊME LISTE QUE LA COLONNE. Les lignes sont celles de `useFeedView` : tous les joueurs, le filtre courant
 * du fil, coupées à l'instant lu, de la plus récente à la plus ancienne ; l'extrait n'en garde que les
 * premières (`COTES_FICHE_STANDARD.pied.lignes`) et ne refiltre rien. Chaque ligne a le format de la ligne du
 * fil (`FeedLine`), cliquable comme elle ; le joueur de la fiche y est en gras là où son nom apparaît.
 *
 * Un filtre qui n'a aucune ligne à cet instant laisse la liste vide : le titre et le lien restent, aucune
 * ligne n'est inventée.
 */
import { useMemo, type ComponentProps } from 'react'

import { teamTokenCssVar } from '@/features/match-view/teamSeriesColor'
import type { ReplayPlayer } from '@/lib/replay/rosterLogic'

import { useFeedView } from '../hooks/useFeedView'
import { useTeamCascades } from '../hooks/useTeamCascades'
import { REPLAY_TEXT } from '../i18n/i18n'
import { COTES_FICHE_STANDARD } from '../model/cotesFicheStandard'
import { NO_WEAPON_ICONS } from '../model/killSourceIcons'
import { displayClockMs } from '../model/replayWindow'
import { NO_MARKS } from '../../../lib/replay/playerMarks'
import type { ReplayKillFeed } from './ReplayKillFeed'
import { FeedLine, type InkOf } from './ReplayFeedLines'
import { PRESS_GUARDS } from './replayPressGuards'

/** Ce que l'extrait lit du fil de la page : exactement ce que reçoit le fil de la colonne. */
type FeedSource = ComponentProps<typeof ReplayKillFeed>

export interface ReplayFeedExtractProps extends FeedSource {
  /** Le joueur de la fiche : son nom est en gras sur les lignes où il figure. */
  player: ReplayPlayer
  /** « Fil complet » : rend la colonne des équipes et du fil. */
  onFullFeed: () => void
}

const P = COTES_FICHE_STANDARD.pied

export function ReplayFeedExtract({
  player, onFullFeed, entries, nowMs, playWindow, scoreboard, xuidMeta, allegiance, locale, marks, zoneOf, doc,
  playbackStore, weaponIcons = NO_WEAPON_ICONS,
}: ReplayFeedExtractProps) {
  const t = REPLAY_TEXT[locale]
  const inkOf = useMemo<InkOf>(() => (xuid) => teamTokenCssVar(allegiance.ofXuid(xuid)), [allegiance])
  const teams = useTeamCascades(scoreboard ?? undefined, allegiance, locale)
  const { lines, selectedKey, select } = useFeedView({ entries, nowMs, doc, locale, allegiance, weaponIcons, playbackStore })
  // Le fil nomme un joueur par son xuid de base, le film par sa clé : les deux le désignent.
  const strong = useMemo(
    () => new Set([player.xuid, ...(player.board?.xuid ? [player.board.xuid] : [])]),
    [player],
  )

  return (
    <section
      aria-label={t.playerPanel.extractTitle}
      data-replay-fiche-extract=""
      className="flex shrink-0 flex-col border-t border-border"
      style={{ paddingTop: P.margeHautePx }}
    >
      <div
        className="flex items-baseline text-[11px] font-semibold uppercase tracking-[.2px] text-muted-foreground"
        style={{ marginBottom: P.ecartTitrePx }}
      >
        {t.playerPanel.extractTitle}
        <button
          type="button"
          className="ml-auto font-normal normal-case text-muted-foreground hover:text-foreground"
          style={{ fontSize: P.tailleLienPx }}
          {...PRESS_GUARDS}
          onClick={onFullFeed}
        >
          {t.playerPanel.fullFeed}
        </button>
      </div>
      <ul className="flex flex-col" aria-live="off">
        {lines.slice(0, P.lignes).map((entry) => (
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
            strongXuids={strong}
          />
        ))}
      </ul>
    </section>
  )
}
