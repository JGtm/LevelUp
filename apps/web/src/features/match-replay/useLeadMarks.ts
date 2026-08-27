/**
 * useLeadMarks — ce que la frise a besoin de savoir sur les retournements.
 *
 * POURQUOI UN FICHIER À PART. `ReplayCanvas` porte un cliquet de taille
 * (`placementFamily.guard.test.ts`) dont la règle est écrite noir sur blanc : « le franchir
 * se corrige en extrayant, pas en relevant le nombre ». Ces trois dérivations appartiennent
 * aux retournements, pas au dessin de la carte — et un hook exporté depuis un fichier de
 * composant coûte un avertissement `react-refresh`, d'où ce module (même convention que
 * useSlotIdentity.ts).
 *
 * Les cascades employées sont celles du dépôt, sans troisième copie : `allyOfTeamId` pour le
 * camp (grammaire de xuidMeta) et `resolveTeamLabel` pour le nom (celle du scoreboard, des
 * objectifs et des colonnes du rejeu).
 */
import { useCallback, useMemo } from 'react'

import type { XuidMeta } from '@/features/match-view/xuidMeta'
import { resolveTeamLabel } from '@/lib/halo/teamLabel'
import { allyOfTeamId, leadChanges, scoreTimelineOf } from '@/lib/replay/scoreTimeline'
import type { MatchScoreboardRow } from '@/lib/api/types'

import { REPLAY_TEXT, type ReplayLocale } from './i18n'
import type { ReplayLeadMarksProps } from './ReplayLeadMarks'
import type { ReplayDocumentReady } from './replayNormalize'
import type { ReplayWindowBounds } from './replayWindow'

export function useLeadMarks(
  doc: ReplayDocumentReady,
  scoreboard: MatchScoreboardRow[] | undefined,
  xuidMeta: XuidMeta | undefined,
  locale: ReplayLocale,
  playWindow: ReplayWindowBounds | null,
): ReplayLeadMarksProps {
  const t = REPLAY_TEXT[locale]
  // Le balayage des paliers ne dépend QUE du document : la frise ne se recalcule pas
  // soixante fois par seconde de lecture.
  const changes = useMemo(() => leadChanges(scoreTimelineOf(doc)), [doc])
  const board = useMemo(() => scoreboard ?? [], [scoreboard])
  const allyOf = useCallback(
    (teamId: number) => allyOfTeamId(board, xuidMeta, teamId),
    [board, xuidMeta],
  )
  const labelOf = useCallback(
    (teamId: number) =>
      resolveTeamLabel(
        board.filter((r) => r.team_side === `t${teamId}`),
        `t${teamId}`,
        t,
      ),
    [board, t],
  )
  return {
    changes,
    frameCount: doc.frameCount,
    frameIntervalMs: doc.frameIntervalMs,
    playWindow,
    allyOf,
    labelOf,
    locale,
  }
}
