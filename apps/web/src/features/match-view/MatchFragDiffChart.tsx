/**
 * MatchFragDiffChart — match_view.13.
 *
 * Frags différentiel cumulé pour tous les joueurs du match.
 * Chaque kill = +1, chaque mort = -1. Une courbe par joueur, lignes droites
 * sans marqueurs. Couleurs : main player en `compare-a`, alliés sur palette
 * cool, ennemis sur palette warm (cf. `colors.ts`).
 *
 * Source : `combat_tab.highlight_events` (kill+death horodatés).
 *
 * Résolution gamertag : la table `xuidToGamertag` est construite depuis
 * scoreboard + roster + killer_victim, parce que certains xuids présents
 * dans `highlight_events` peuvent ne pas être au scoreboard mais sont
 * résolus dans les kvPairs (cf. `buildXUIDToGamertagMap`).
 */
import { Card, CardContent } from '@/components/ui/card'
import { TimeseriesLineChart } from '@/components/charts/TimeseriesLineChart'
import type {
  MatchHighlightEvent,
  MatchKillerVictimPair,
  MatchRosterRow,
  MatchScoreboardRow,
} from '@/lib/api/types'
import { allPlayersFragDiffSeries, formatBinSeconds } from './_chartSeries'
import { buildMatchPlayerColors, buildXUIDToGamertagMap } from './colors'

interface Props {
  events: MatchHighlightEvent[]
  scoreboard: MatchScoreboardRow[]
  roster?: MatchRosterRow[]
  pairs?: MatchKillerVictimPair[]
  meXUID: string | null
  /** Gamertags amis (page Squad) — bonus visuel : couleurs squad pour les amis alliés. */
  friendGamertags?: readonly string[]
}

export function MatchFragDiffChart({
  events,
  scoreboard,
  roster,
  pairs,
  meXUID,
  friendGamertags,
}: Props) {
  const xuidToGamertag = buildXUIDToGamertagMap(scoreboard, pairs, roster)
  const colors = buildMatchPlayerColors(scoreboard, meXUID, friendGamertags, roster)
  const series = allPlayersFragDiffSeries(events, xuidToGamertag, meXUID, colors.tokenByXUID)
  if (series.length === 0) {
    return (
      <Card>
        <CardContent className="py-8 text-center text-sm text-muted-foreground">
          Pas d'events kill/death disponibles pour tracer le différentiel cumulé.
        </CardContent>
      </Card>
    )
  }
  // Pré-résolution hex par xuid : on extrait le suffixe xuid de la clé de
  // chaque série et on lit `colors.hexByXUID`. Évite que ECharts retombe sur
  // sa palette interne (1ères entrées = bleu) si le token n'a pas pu être
  // résolu juste-à-temps côté wrapper.
  const FRAG_KEY_PREFIX = 'match_view.combat.frag_diff.'
  const colorResolver = (s: { key: string }): string | undefined => {
    const xu = s.key.startsWith(FRAG_KEY_PREFIX) ? s.key.slice(FRAG_KEY_PREFIX.length) : ''
    return colors.hexByXUID.get(xu)
  }
  return (
    <Card>
      <CardContent className="py-4">
        <TimeseriesLineChart
          title="Frags différentiel cumulé — tous les joueurs"
          height={360}
          xAxisType="value"
          timeAxis={false}
          outcomeMarkers={false}
          showSymbol={false}
          xAxisLabelFormatter={(v) => formatBinSeconds(Number(v))}
          series={series}
          seriesColorResolver={colorResolver}
        />
      </CardContent>
    </Card>
  )
}
