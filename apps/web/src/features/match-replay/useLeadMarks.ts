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
 * CE QU'IL ALIMENTE A CHANGÉ LE 2026-08-28, PAS CE QU'IL CALCULE. Il servait `ReplayLeadMarks`,
 * qui posait un trait sur la frise à chaque retournement ; ce composant est SUPPRIMÉ — la piste
 * DOMINANCE de la nouvelle frise montre les mêmes retournements en DURÉES (`buildDominance`),
 * ce qui dit davantage et occupe la même hauteur.
 *
 * IL A MAIGRI AVEC SON CONSOMMATEUR (revue R1, même jour). Il rendait sept champs, taillés pour
 * les props d'un composant qui n'existe plus : la piste Dominance n'en lit que TROIS — les
 * changements, le camp, le libellé. Les quatre autres (`frameCount`, `frameIntervalMs`,
 * `playWindow`, `locale`) étaient recopiés pour personne, et l'échelle comme l'horloge sont
 * désormais l'affaire de `useReplayTimeline`, qui les tient de première main. On ne garde pas
 * des sorties « au cas où » : git a l'historique.
 *
 * Les cascades employées sont celles du dépôt, sans troisième copie : `allyOfTeamId` pour le
 * camp (grammaire de xuidMeta) et `resolveTeamLabel` pour le nom (celle du scoreboard, des
 * objectifs et des colonnes du rejeu).
 */
import { useCallback, useMemo } from 'react'

import type { XuidMeta } from '@/features/match-view/xuidMeta'
import { resolveTeamLabel } from '@/lib/halo/teamLabel'
import { allyOfTeamId, leadChanges, scoreTimelineOf, type LeadChange } from '@/lib/replay/scoreTimeline'
import type { MatchScoreboardRow } from '@/lib/api/types'

import { REPLAY_TEXT, type ReplayLocale } from './i18n'
import type { ReplayDocumentReady } from './replayNormalize'

/**
 * Ce que la piste DOMINANCE reçoit des retournements : quand le meneur change, et qui il est.
 * Le type vivait dans `ReplayLeadMarks.tsx` tant que ce composant existait ; il a suivi son
 * seul producteur à sa suppression, puis s'est réduit à ce que son seul lecteur consomme.
 */
export interface ReplayLeadMarks {
  /** Les instants où le meneur change (cf. `leadChanges`). Vide = aucun retournement mesuré. */
  changes: readonly LeadChange[]
  /** Camp du meneur, du point de vue du joueur de la page (`null` = inconnu). */
  allyOf: (teamId: number) => boolean | null
  /** Libellé de l'équipe qui passe devant, tel que la colonne l'écrit. */
  labelOf: (teamId: number) => string
}

export function useLeadMarks(
  doc: ReplayDocumentReady,
  scoreboard: MatchScoreboardRow[] | undefined,
  xuidMeta: XuidMeta | undefined,
  locale: ReplayLocale,
): ReplayLeadMarks {
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
  return { changes, allyOf, labelOf }
}
