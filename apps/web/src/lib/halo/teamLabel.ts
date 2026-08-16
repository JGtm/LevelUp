/**
 * teamLabel.ts — LE LIBELLÉ D'UNE ÉQUIPE, et il n'y en a qu'un.
 *
 * La cascade vivait en deux copies (`MatchScoreboard.tsx`, `MatchObjectivesSection.tsx`) ;
 * les fiches du rejeu 2D en auraient été la troisième (2026-08-16) — règle CLAUDE.md n°6 :
 * centraliser ET poser un garde-rail (`teamLabel.guard.test.ts` : `labelHasTeamWord(` ne
 * s'appelle plus hors de `lib/halo/`).
 *
 * L'ORDRE DE LA CASCADE, inchangé :
 *   1. libellé fourni par le backend (`team_name` — Halo 5 : « Rouge/Bleu » déjà localisé
 *      côté serveur depuis team_colors) ;
 *   2. sinon nom officiel résolu côté front par `team_side` (Halo Infinite : Eagle / Cobra…) ;
 *   3. un libellé qui contient DÉJÀ le mot « équipe/team » n'est PAS re-préfixé (sinon
 *      « Équipe Équipe Cobra ») ; un nom nu reçoit le préfixe localisé ;
 *   4. sinon « Équipe N » si l'id d'équipe est connu mais hors référentiel ;
 *   5. sinon « Équipe inconnue ».
 * Décision sur la DONNÉE, jamais sur le slug de titre.
 */
import type { MatchScoreboardRow } from '@/lib/api/types'

import { labelHasTeamWord, parseTeamSideID, resolveTeamName } from './teamNames'

/**
 * Les trois textes localisés dont la cascade a besoin. `MatchViewText` et le dictionnaire du
 * rejeu les portent tous deux ; ce type structurel évite un couplage entre features.
 */
export interface TeamLabelText {
  teamLabelFmt: (name: string) => string
  teamNumberedFmt: (n: number) => string
  teamUnknown: string
}

/** Lignes de scoreboard à considérer pour le libellé backend : celles de l'équipe visée. */
type RowsLike = ReadonlyArray<Pick<MatchScoreboardRow, 'team_name'>>

/**
 * resolveTeamLabel rend le libellé d'affichage d'une équipe.
 *
 * `rows` = les lignes de scoreboard DE CETTE ÉQUIPE (le premier `team_name` non vide gagne) ;
 * `teamSide` = le côté au format `t{N}` du backend, `null` si inconnu.
 */
export function resolveTeamLabel(
  rows: RowsLike,
  teamSide: string | null | undefined,
  text: TeamLabelText,
): string {
  const backendName = rows.find((r) => r.team_name)?.team_name ?? null
  const officialName = backendName ?? resolveTeamName(teamSide)
  if (officialName) {
    return labelHasTeamWord(officialName) ? officialName : text.teamLabelFmt(officialName)
  }
  const teamID = parseTeamSideID(teamSide)
  return teamID != null ? text.teamNumberedFmt(teamID) : text.teamUnknown
}
