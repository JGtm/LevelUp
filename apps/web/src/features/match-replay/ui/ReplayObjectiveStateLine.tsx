/**
 * ReplayObjectiveStateLine — LES LIGNES D'OBJECTIF QUI NE SONT PAS UNE ACTION NOMMÉE : la fin d'une
 * manche, le crâne d'Oddball (pris, perdu) et la colline (début de tenue). Elles prennent la coque
 * commune des objectifs (`ObjectiveBanner`) et ses règles : une seule ligne, sans retour, le libellé
 * tronque avant le nom.
 *
 * L'ENCRE. Le crâne se lit à l'encre de camp du porteur (l'allégeance du film), la colline à celle du
 * camp qui la tient (`teams.allyOf`, la même lecture que les en-têtes de camp), le nom du camp venant
 * de la cascade des colonnes de fiches (`teams.labelOf`). Une fin de manche n'a pas de camp : sa coque
 * est à l'encre neutre, et chaque camp y garde sa propre encre devant son score de manche.
 */
import { teamTokenCssVar } from '@/features/match-view/teamSeriesColor'
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import { displayPlayerName } from '@/lib/players/displayName'

import type { ReplayTeamCascades } from '../hooks/useTeamCascades'
import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'
import type {
  HillEventLine,
  ObjectiveStateLine,
  RoundEndEventLine,
  SkullEventLine,
} from '../model/feedEventLines'
import type { PlayerMarkKind } from '../../../lib/replay/playerMarks'
import { FeedName } from './ReplayFeedName'
import type { GamertagIndex, InkOf } from './ReplayFeedLines'
import { ObjectiveBanner } from './ReplayObjectiveLine'

interface Props {
  line: ObjectiveStateLine
  replayMs: number
  inkOf: InkOf
  xuidMeta: GamertagIndex
  marks: ReadonlyMap<string, PlayerMarkKind>
  teams: ReplayTeamCascades
  locale: ReplayLocale
}

export function ReplayObjectiveStateLine({ line, ...rest }: Props) {
  if ('index' in line) return <RoundEndLine line={line} {...rest} />
  if ('teamId' in line) return <HillLine line={line} {...rest} />
  return <SkullLine line={line} {...rest} />
}

/** Le crâne : le porteur, à l'encre de son camp, puis ce qu'il en fait. */
function SkullLine({
  line,
  replayMs,
  inkOf,
  xuidMeta,
  marks,
  locale,
}: Props & { line: SkullEventLine }) {
  const t = REPLAY_TEXT[locale].feedLines
  const color = inkOf(line.xuid)
  const name = displayPlayerName(xuidMeta.get(line.xuid)?.gamertag, line.xuid)
  const action = line.change === 'taken' ? t.skullTaken : t.skullLost
  return (
    <ObjectiveBanner glyph="skull" color={color} replayMs={replayMs} title={`${name} ${action}`}>
      <FeedName kind={marks.get(line.xuid)} color={color} locale={locale} className="font-extrabold" name={name} />
      <span className="min-w-0 truncate text-foreground">{action}</span>
    </ObjectiveBanner>
  )
}

/** La colline : le camp qui se met à la tenir, à son encre et sous son nom de colonne. */
function HillLine({ line, replayMs, teams, locale }: Props & { line: HillEventLine }) {
  const action = REPLAY_TEXT[locale].feedLines.hillTaken
  const color = teamTokenCssVar(teams.allyOf(line.teamId))
  const name = teams.labelOf(line.teamId)
  return (
    <ObjectiveBanner glyph="hill" color={color} replayMs={replayMs} title={`${name} ${action}`}>
      <FeedName kind={undefined} color={color} locale={locale} className="font-extrabold" name={name} />
      <span className="min-w-0 truncate text-foreground">{action}</span>
    </ObjectiveBanner>
  )
}

/** La fin d'une manche : son rang, puis le score de manche de chaque camp, nommé à son encre. */
function RoundEndLine({ line, replayMs, teams, locale }: Props & { line: RoundEndEventLine }) {
  const label = REPLAY_TEXT[locale].feedLines.roundEnd(line.index)
  const scores = line.scores.map((s) => ({ ...s, name: teams.labelOf(s.teamId) }))
  return (
    <ObjectiveBanner
      glyph="round"
      color={tokenCssVar('divergent-neutral')}
      replayMs={replayMs}
      title={`${label} : ${scores.map((s) => `${s.name} ${s.score}`).join(' – ')}`}
    >
      <span className="shrink-0 font-extrabold text-foreground">{label}</span>
      {scores.map((s, i) => (
        <span key={s.teamId} className="flex min-w-0 items-center gap-1">
          {i > 0 && (
            <span aria-hidden className="shrink-0 text-muted-foreground">
              –
            </span>
          )}
          <FeedName
            kind={undefined}
            color={teamTokenCssVar(teams.allyOf(s.teamId))}
            locale={locale}
            className="font-medium"
            name={s.name}
          />
          <span className="shrink-0 font-mono tabular-nums text-foreground">{s.score}</span>
        </span>
      ))}
    </ObjectiveBanner>
  )
}
