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
 *   5. sinon le DERNIER RECOURS de l'appelant : « Équipe inconnue » pour `resolveTeamLabel`
 *      (Vue match), « Équipe N » de l'identifiant que l'appelant CONNAÎT pour
 *      `resolveKnownTeamLabel` (rejeu : l'équipe y vient du film, elle a toujours un numéro).
 * Décision sur la DONNÉE, jamais sur le slug de titre.
 */
import type { MatchScoreboardRow } from '@/lib/api/types'

import { labelHasTeamWord, parseTeamSideID, resolveTeamName } from './teamNames'

/**
 * Les deux textes localisés qui NOMMENT une équipe. Le dictionnaire du rejeu les porte et rien
 * de plus : sur sa page, une équipe vient du film et n'est jamais « inconnue » (ADR 0034 D-9).
 */
export interface TeamNamingText {
  teamLabelFmt: (name: string) => string
  teamNumberedFmt: (n: number) => string
}

/**
 * Les trois textes de la cascade de la Vue match : les deux qui nomment, plus son dernier
 * recours. `MatchViewText` les porte ; ce type structurel évite un couplage entre features.
 */
export interface TeamLabelText extends TeamNamingText {
  teamUnknown: string
}

/** Lignes de scoreboard à considérer pour le libellé backend : celles de l'équipe visée. */
type RowsLike = ReadonlyArray<Pick<MatchScoreboardRow, 'team_name'>>

/**
 * nameFromSheet — les étapes 1 à 4 de la cascade, `null` quand aucune ne nomme l'équipe. Les
 * deux variantes exportées ne diffèrent que par leur dernier recours.
 */
function nameFromSheet(
  rows: RowsLike,
  teamSide: string | null | undefined,
  text: TeamNamingText,
): string | null {
  const backendName = rows.find((r) => r.team_name)?.team_name ?? null
  const officialName = backendName ?? resolveTeamName(teamSide)
  if (officialName) {
    return labelHasTeamWord(officialName) ? officialName : text.teamLabelFmt(officialName)
  }
  const teamID = parseTeamSideID(teamSide)
  return teamID != null ? text.teamNumberedFmt(teamID) : null
}

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
  return nameFromSheet(rows, teamSide, text) ?? text.teamUnknown
}

/**
 * resolveKnownTeamLabel — le libellé d'une équipe dont l'IDENTIFIANT est connu de l'appelant :
 * la même cascade, et pour dernier recours « Équipe N » de cet identifiant au lieu
 * d'« Équipe inconnue ». Le rejeu l'emploie partout : l'équipe y est le désignateur du film, et
 * un camp sans ligne de feuille garde un nom (« Équipe 1 ») plutôt qu'un aveu d'ignorance.
 */
export function resolveKnownTeamLabel(
  rows: RowsLike,
  teamSide: string | null | undefined,
  teamId: number,
  text: TeamNamingText,
): string {
  return nameFromSheet(rows, teamSide, text) ?? text.teamNumberedFmt(teamId)
}
