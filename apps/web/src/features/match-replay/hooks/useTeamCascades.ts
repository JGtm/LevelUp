/**
 * useTeamCascades — DE QUEL CÔTÉ EST UN CAMP, ET COMMENT IL S'APPELLE.
 *
 * POURQUOI UN FICHIER À PART. `ReplayCanvas` porte un seuil de taille
 * (`max-lines` eslint, R5) dont la règle est écrite noir sur blanc : « le franchir
 * se corrige en extrayant, pas en relevant le nombre ». Ces deux dérivations appartiennent
 * aux équipes, pas au dessin de la carte — et un hook exporté depuis un fichier de
 * composant coûte un avertissement `react-refresh`, d'où ce module (même convention que
 * useSlotIdentity.ts).
 *
 * IL S'APPELAIT `useLeadMarks` JUSQU'AU 2026-08-28, et il rendait alors une troisième
 * sortie : les RETOURNEMENTS lus du calque de score (`leadChanges`). La piste DOMINANCE ne
 * lit plus ce calque — elle compte les FRAGS du fil (`buildFragDominance`, demande
 * utilisateur du même jour) — et ces retournements n'avaient donc plus de lecteur. On ne
 * garde pas une sortie « au cas où » : git a l'historique, et `leadChanges` vit toujours
 * dans `lib/replay/scoreTimeline` pour la courbe de score de la vue match, son seul
 * consommateur restant. Le nom suit la fonction : ce hook ne sait plus rien des meneurs, il
 * sait nommer et colorer une équipe.
 *
 * Les cascades employées sont celles du dépôt, sans troisième copie : `allyOfTeamId` pour le
 * camp (grammaire de xuidMeta) et `resolveTeamLabel` pour le nom (celle du scoreboard, des
 * objectifs et des colonnes du rejeu).
 */
import { useCallback, useMemo } from 'react'

import type { XuidMeta } from '@/features/match-view/xuidMeta'
import { resolveTeamLabel } from '@/lib/halo/teamLabel'
import { allyOfTeamId } from '@/lib/replay/scoreTimeline'
import type { MatchScoreboardRow } from '@/lib/api/types'

import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'

/**
 * Ce que la piste DOMINANCE reçoit sur les équipes : de quel côté est un camp, et son nom.
 * Les deux se lisent du scoreboard — seul endroit où le camp du film (`team_side` au format
 * `t{N}`) et les joueurs de la page coexistent.
 */
export interface ReplayTeamCascades {
  /** Camp du meneur, du point de vue du joueur de la page (`null` = inconnu). */
  allyOf: (teamId: number) => boolean | null
  /** Libellé de l'équipe qui passe devant, tel que la colonne l'écrit. */
  labelOf: (teamId: number) => string
}

export function useTeamCascades(
  scoreboard: MatchScoreboardRow[] | undefined,
  xuidMeta: XuidMeta | undefined,
  locale: ReplayLocale,
): ReplayTeamCascades {
  const t = REPLAY_TEXT[locale]
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
  return { allyOf, labelOf }
}
