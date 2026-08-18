/**
 * ReplayTeamHeader — LE TITRE D'UNE COLONNE D'ÉQUIPE, dans les fiches du rejeu.
 *
 * CE QU'IL CORRIGE (demande utilisateur du 2026-08-16 : « chaque équipe devra retrouver son
 * nom (Équipe Cobra / Équipe Eagle sur le scoreboard, sans réinventer la roue ») : la colonne
 * affichait `t0` / `t1` bruts, c'est-à-dire l'identifiant de transport du backend. Le libellé
 * vient désormais de `resolveTeamLabel` — LA cascade du dépôt, celle du scoreboard et des
 * objectifs — sans qu'une troisième copie soit écrite ici (règle CLAUDE.md n°6).
 *
 * LA COULEUR EST CELLE DES DEUX AUTRES PANNEAUX (décision D1 amendée) : `team-ally` /
 * `team-enemy`, les tokens que les réglages d'accessibilité peuvent surcharger. Un point bleu
 * sur la carte et un titre rouge pour la même équipe seraient une page cassée. Les tokens de
 * COMPARAISON qui servaient ici (`compare-a/b`) disaient « ceci n'est pas cela » et rien de
 * plus : ils sont supprimés avec la fonction qui les cyclait.
 *
 * UN GROUPE SANS CAMP CONNU N'EMPRUNTE AUCUNE DES DEUX COULEURS : encre et liseré neutres.
 * Le camp est une information, pas un défaut d'affichage à combler.
 *
 * LE SCORE QUI TIQUE (schéma 12, lot A phase 2). La colonne portait le nom de l'équipe et
 * son effectif ; elle porte maintenant son SCORE À L'INSTANT LU — la seule grandeur que le
 * rejeu pouvait montrer et ne montrait pas. C'est le TOTAL du match qui s'affiche en grand,
 * parce que c'est le nombre que le jeu affiche et celui que l'oracle a validé (43/50 sur le
 * témoin Slayer, 200/121 sur l'Oddball). Quand le mode a plusieurs manches, un second
 * repère, discret, rappelle la manche en cours et SA valeur : sans lui, « 200 » sur un
 * Oddball ne dit pas que la manche en est à 100.
 *
 * ZÉRO N'EST PAS UNE LACUNE ICI, et c'est le seul endroit du dossier où c'est vrai : une
 * équipe qui n'a jamais marqué n'émet aucun palier (témoin CTF 3-0 : une seule série pour
 * deux camps). Son score EST nul. Mais si le film ne publie AUCUNE série d'équipe — mode
 * sans compteur, identité non résolue — la colonne n'affiche rien du tout : « 0 » se lirait
 * alors comme une mesure, alors que personne n'a compté.
 */
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import type { XuidMeta } from '@/features/match-view/xuidMeta'
import { resolveTeamLabel } from '@/lib/halo/teamLabel'
import {
  roundAtFrame,
  teamIdOfSide,
  teamScoreAtFrame,
  teamSeriesFor,
  type ReplayScoreTimelineReady,
} from '@/lib/replay/scoreTimeline'
import type { MatchScoreboardRow } from '@/lib/api/types'

import { REPLAY_TEXT, type ReplayLocale } from './i18n'
import type { ReplayPlayer } from './rosterLogic'

/** Part du fond qu'occupe la couleur d'équipe : assez pour teinter, jamais pour crier. */
const TINT_PCT = 14

/**
 * allyOfGroup dit de quel côté est un groupe : `true` allié, `false` adverse, `null` quand
 * on ne sait pas — camp non transmis, ou aucun joueur du groupe reconnu au scoreboard.
 * « Allié » veut dire « du côté du joueur dont on regarde la page » (cf. xuidMeta.ts).
 */
function allyOfGroup(
  players: readonly ReplayPlayer[],
  side: string | null,
  xuidMeta: XuidMeta | undefined,
): boolean | null {
  if (side === null || !xuidMeta) return null
  const known = players.filter((p) => xuidMeta.has(p.xuid))
  if (known.length === 0) return null
  return known.some((p) => xuidMeta.get(p.xuid)?.ally === true)
}

interface Props {
  /** Les joueurs de CE groupe : leurs lignes de scoreboard portent le libellé du backend. */
  players: readonly ReplayPlayer[]
  side: string | null
  xuidMeta?: XuidMeta
  locale: ReplayLocale
  /**
   * Le calque de score du film, déjà passé par la garde d'horloge (`scoreTimelineOf`).
   * Absent = aucun score affiché ; c'est le cas de tout artefact antérieur au schéma 12.
   */
  scoreTimeline?: ReplayScoreTimelineReady
  /** Image de lecture courante — le score est lu À CE frame, pas à la fin du match. */
  frame: number
}

export function ReplayTeamHeader({
  players, side, xuidMeta, locale, scoreTimeline, frame,
}: Props) {
  const t = REPLAY_TEXT[locale]
  const rows = players
    .map((p) => p.board)
    .filter((r): r is MatchScoreboardRow => r !== undefined)
  const label = resolveTeamLabel(rows, side, t)
  const ally = allyOfGroup(players, side, xuidMeta)
  const accent = ally === null ? null : tokenCssVar(ally ? 'team-ally' : 'team-enemy')
  const teamId = teamIdOfSide(side)
  // Aucune série publiée = personne n'a compté : on n'écrit pas « 0 » (cf. en-tête).
  const scored = (scoreTimeline?.teams.length ?? 0) > 0 && teamId != null
  const score = scored ? teamScoreAtFrame(scoreTimeline, teamId, frame) : null
  const round = scored ? roundAtFrame(teamSeriesFor(scoreTimeline, teamId), frame) : null
  return (
    <h3
      className={`mb-2 flex shrink-0 items-baseline justify-between gap-1 rounded-sm px-1.5 py-1 text-[11px] font-semibold uppercase tracking-wide ${
        accent ? 'text-foreground' : 'border-l-[3px] border-border text-muted-foreground'
      }`}
      style={
        accent
          ? {
              borderLeft: `3px solid ${accent}`,
              background: `color-mix(in srgb, ${accent} ${TINT_PCT}%, transparent)`,
            }
          : undefined
      }
    >
      <span className="min-w-0 flex-1 truncate">{label}</span>
      {score !== null && (
        <span
          className="shrink-0 font-mono text-[13px] font-bold tabular-nums"
          style={accent ? { color: accent } : undefined}
          title={t.scoreLive}
        >
          {score}
        </span>
      )}
      {/* La MANCHE en cours, seulement quand il y en a plusieurs : sur un mode à manche
          unique elle répéterait le total. Discrète — c'est un rappel, pas le score. */}
      {round && round.count > 1 && (
        <span
          className="shrink-0 font-mono text-[9px] font-normal tabular-nums text-muted-foreground"
          title={t.roundLabelFmt(round.index, round.count, round.value)}
        >
          {t.roundShortFmt(round.index)}&nbsp;{round.value}
        </span>
      )}
      <span className="shrink-0 font-mono text-[10px] font-normal tabular-nums text-muted-foreground">
        {players.length}
      </span>
    </h3>
  )
}
