/**
 * MatchLivesCard — « Isolement, par joueur » (Vue match, carte I ; plan PLAN_MATCHVIEW_EMPRISE_2026-10-06,
 * D8, D14, A1).
 *
 * Une ligne par joueur de l'équipe, dans l'ordre des fiches de « Prises par joueur » : la ligne de la
 * carte des Séries temporelles (`LivesNearTeammateRow`), son nom à la pastille du joueur (palette du
 * match). Seuls les joueurs qui ont une vie rangée ont une ligne (`buildMatchLives`).
 */
import { useMemo, useRef } from 'react'

import { useSegmentLabelFit } from '@/components/charts/segmentLabelFit'
import { ObjectifFrame } from '@/features/squad/objectif/ObjectifFrame'
import { LivesAxis, LivesLegend } from '@/features/timeseries/usages/LivesNearTeammateCard'
import { LivesNearTeammateRow } from '@/features/timeseries/usages/LivesNearTeammateRow'
import type { UsagesCardsText } from '@/features/timeseries/usages/usagesCardsText'

import type { MatchLives, MatchLivesRow } from './matchEmprise.logic'

interface Props {
  lives: MatchLives
  /** xuid → encre du joueur (palette du match). */
  inkOf: (xuid: string) => string
  /** xuid du joueur de la page : son nom en gras. */
  meXUID: string | null
  ut: UsagesCardsText
}

export function MatchLivesCard({ lives, inkOf, meXUID, ut }: Props) {
  const l = ut.lives
  const ref = useRef<HTMLDivElement | null>(null)
  const hidden = useSegmentLabelFit(ref, lives)
  const legend = useMemo(() => <LivesLegend ut={ut} />, [ut])
  return (
    <ObjectifFrame
      title={l.title}
      info={l.info}
      legend={legend}
      testId="match-emprise-lives"
    >
      <div ref={ref} className="flex flex-col gap-3.5">
        {lives.rows.map((row) => {
          const label = <PlayerLabel row={row} ink={inkOf(row.xuid)} me={row.xuid === meXUID} />
          return <LivesNearTeammateRow key={row.xuid} model={row.model} label={label} hidden={hidden} idPrefix={`${row.xuid}-`} ut={ut} />
        })}
        <LivesAxis ut={ut} />
      </div>
    </ObjectifFrame>
  )
}

function PlayerLabel({ row, ink, me }: { row: MatchLivesRow; ink: string; me: boolean }) {
  return (
    <span className="inline-flex min-w-0 items-center">
      <span className="mr-1.5 inline-block h-[9px] w-[9px] shrink-0 rounded-[2px]" style={{ backgroundColor: ink }} aria-hidden />
      <span className={`truncate ${me ? 'font-semibold' : ''}`}>{row.gamertag}</span>
    </span>
  )
}
