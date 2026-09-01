/**
 * MatchEquipmentUsageSection — LE BILAN D'ÉQUIPEMENT D'UN MATCH, par joueur et par équipe.
 *
 * CE QUE LA PAGE NE SAVAIT PAS DIRE. Le rejeu 2D montre chaque geste d'équipement à l'IMAGE où
 * il a lieu : un mur posé à 3:12, une traction de grappin à 5:40. Personne ne peut en tirer
 * « qui a posé six murs » en regardant six instants séparés de trois minutes. Cette section les
 * COMPTE — et c'est tout ce qu'elle ajoute : aucune donnée nouvelle, aucun appel de plus.
 *
 * ELLE VIT DANS `match-replay/` ET NON DANS `match-view/`, à la différence de la courbe de score
 * qui la précède dans l'onglet. La raison est le VOCABULAIRE : chaque libellé qu'elle écrit —
 * familles de pose, familles d'état actif, types de grenade, socles de bonus — appartient au
 * dictionnaire du rejeu (`REPLAY_TEXT`) et aux tables du document. Le poser dans `match-view`
 * aurait forcé soit un import du dictionnaire voisin, soit une seconde table de noms qui
 * divergerait au premier ajout du manifeste du titre. Le sens de l'import est déjà établi :
 * `MatchScoreCurveChart` lit `match-replay/queries` depuis `match-view`.
 *
 * DOUBLE PORTE, comme la courbe de score : pas d'artefact (le cas de la quasi-totalité des
 * matchs) OU aucune grandeur mesurée -> RIEN. Pas de cadre vide, pas de « bientôt disponible ».
 * La MÊME clé de cache que la courbe (`useMatchReplay`, gaté par `header.replay_available`) :
 * les deux blocs de l'onglet partagent un seul téléchargement.
 *
 * CE QUE L'ÉCRAN DIT DE SA PROPRE MESURE, et il doit le dire :
 *   - les états actifs (camouflage, surbouclier) sont un PROXY — le film mesure que l'effet
 *     court, pas d'où il vient. La réserve est en infobulle d'en-tête, pas dans un commentaire ;
 *   - les socles de bonus vidés sont ANONYMES par mesure : une ligne au niveau du MATCH, jamais
 *     une colonne de joueur ;
 *   - répulseur et propulseur n'ont aucune colonne, et une phrase dit pourquoi : une colonne de
 *     zéros se lirait « zéro utilisation » là où la vérité est « non mesuré ».
 *
 * Aucun calcul ici : tout vient de `equipmentUsageLogic` (les mesures) et
 * `equipmentUsageColumns` (les colonnes et leurs noms). Gabarit structurel :
 * `match-view/MatchObjectivesSection.tsx` — mêmes classes de tableau, mêmes tokens.
 */
import { useMemo } from 'react'

import type { MatchScoreboardRow } from '@/lib/api/types'
import { resolveTeamLabel } from '@/lib/halo/teamLabel'
import { HeaderLabelTooltip } from '@/lib/table/columnMeta'

import { equipmentFamilyLabel, usageColumnGroups, type UsageColumn } from './equipmentUsageColumns'
import {
  buildEquipmentUsage,
  tallyTotal,
  type EquipmentUsage,
  type EquipmentUsageTeam,
} from './equipmentUsageLogic'
import { REPLAY_TEXT, type ReplayLocale } from './i18n'
import type { ReplayText } from './i18nContract'
import { useMatchReplay } from './queries'

interface Props {
  playerSlug: string
  matchId: string
  /** `header.replay_available` — le même gate que le lien rejeu et que la courbe de score. */
  replayAvailable: boolean
  scoreboard: MatchScoreboardRow[] | null | undefined
  locale: ReplayLocale
}

export function MatchEquipmentUsageSection({
  playerSlug,
  matchId,
  replayAvailable,
  scoreboard,
  locale,
}: Props) {
  const t = REPLAY_TEXT[locale]
  const { data } = useMatchReplay(playerSlug, matchId, replayAvailable)
  const board = useMemo(() => scoreboard ?? [], [scoreboard])
  const usage = useMemo(() => (data ? buildEquipmentUsage(data, board) : null), [data, board])
  const groups = useMemo(
    () => (data && usage ? usageColumnGroups(usage, data, t, locale) : []),
    [data, usage, t, locale],
  )
  const leaves = useMemo(() => groups.flatMap((g) => g.columns), [groups])
  const meXUID = useMemo(() => board.find((r) => r.is_me)?.xuid ?? null, [board])

  // Double porte : pas d'artefact, ou rien de mesuré -> rien du tout.
  if (!usage?.hasData) return null

  return (
    <section className="rounded-lg border-2 border-border" aria-label={t.equipmentUsage.title}>
      <h3 className="px-3 py-2 text-sm font-bold uppercase tracking-wider text-foreground">
        {t.equipmentUsage.title}
      </h3>
      <div className="overflow-x-auto">
        <table className="w-full border-collapse text-3xs">
          <thead>
            <tr className="text-muted-foreground">
              <th rowSpan={2} className="border border-border border-b-2 px-2 py-1 text-left">
                {t.equipmentUsage.colPlayer}
              </th>
              {groups.map((g) => (
                <th
                  key={g.key}
                  colSpan={g.columns.length}
                  className="border border-border px-2 py-1 text-center font-semibold"
                >
                  <HeaderLabelTooltip text={g.hint} focusable>
                    <span>{g.label}</span>
                  </HeaderLabelTooltip>
                </th>
              ))}
            </tr>
            <tr className="text-muted-foreground">
              {leaves.map((c) => (
                <th key={c.key} className="border border-border border-b-2 px-2 py-1 text-right">
                  {c.label}
                </th>
              ))}
            </tr>
          </thead>
          {usage.byTeam.map((team) => (
            <UsageTeamBody
              key={team.side ?? 'sans-equipe'}
              team={team}
              board={board}
              leaves={leaves}
              meXUID={meXUID}
              t={t}
            />
          ))}
        </table>
      </div>
      <UsageFootnotes usage={usage} t={t} />
    </section>
  )
}

/**
 * UsageTeamBody — un camp : son en-tête, ses joueurs, son total.
 *
 * LE CAMP SANS NOM EST UN CAMP QUAND MÊME (`side` null) : ce sont les joueurs que le film a vus
 * vivre et que le scoreboard ignore. Ils gardent leur ligne sous « Sans équipe » — le trou se
 * montre, il ne se comble pas, et surtout on ne les verse pas dans un camp au hasard.
 */
function UsageTeamBody({
  team,
  board,
  leaves,
  meXUID,
  t,
}: {
  team: EquipmentUsageTeam
  board: MatchScoreboardRow[]
  leaves: UsageColumn[]
  meXUID: string | null
  t: ReplayText
}) {
  const rows = board.filter((r) => (r.team_side ?? '') === (team.side ?? ''))
  const label = resolveTeamLabel(team.side ? rows : [], team.side, t)
  return (
    <tbody>
      <tr>
        <th
          colSpan={leaves.length + 1}
          className="border border-border px-3 py-1 text-left text-xs font-semibold text-foreground"
        >
          {label}
        </th>
      </tr>
      {team.players.map((p) => (
        // Clé composite : un xuid peut manquer (entrée de film sans identité résolue) et
        // plusieurs lignes se percuteraient sur une clé vide.
        <tr key={`${p.xuid}||${p.name}`} className={p.xuid === meXUID ? 'bg-info/10' : ''}>
          <td className="border border-border px-2 py-1 text-left">{p.name}</td>
          {leaves.map((c) => (
            <td key={c.key} className="border border-border px-2 py-1 text-right tabular-nums">
              {c.cell(p)}
            </td>
          ))}
        </tr>
      ))}
      <tr className="font-semibold text-foreground">
        <td className="border border-border px-2 py-1 text-left">{t.equipmentUsage.teamTotal}</td>
        {leaves.map((c) => (
          <td key={c.key} className="border border-border px-2 py-1 text-right tabular-nums">
            {c.cell(team.total)}
          </td>
        ))}
      </tr>
    </tbody>
  )
}

/**
 * UsageFootnotes — la ligne ANONYME du match, les dénominateurs, et ce qui n'est pas mesuré.
 *
 * La ligne des socles de bonus est HORS DU TABLEAU, et ce n'est pas une question de place : le
 * ramasseur n'est pas AFFICHÉ ici (`padPickups[].xuid` est publié depuis le schéma 30, mais cet
 * écran n'a pas été repensé pour l'exploiter). Une colonne, même
 * intitulée « anonyme », finirait par se lire comme une grandeur de joueur.
 */
function UsageFootnotes({ usage, t }: { usage: EquipmentUsage; t: ReplayText }) {
  const u = t.equipmentUsage
  const cov = usage.coverage
  const orphelins = tallyTotal(usage.unattributed)
  const detail = Object.entries(usage.powerupPickups)
    .map(([family, n]) => `${equipmentFamilyLabel(family, t)} ${n}`)
    .join(' · ')
  return (
    <div className="space-y-1 px-3 pb-2 pt-2 text-[11px] text-muted-foreground">
      {usage.powerupPickupsTotal > 0 && (
        <p className="text-foreground">
          <HeaderLabelTooltip text={u.powerupPadsHint} focusable>
            <span>{u.powerupPads}</span>
          </HeaderLabelTooltip>
          {' : '}
          <span className="font-semibold tabular-nums">{usage.powerupPickupsTotal}</span>
          {detail ? ` (${detail})` : ''}
          {cov.powerupPads > 0 ? ` — ${u.powerupPadsDenomFmt(cov.powerupPads)}` : ''}
        </p>
      )}
      {usage.columns.episodes.length > 0 && cov.tracksTotal > 0 && (
        <p>{u.coverageActiveFmt(cov.tracksTotal)}</p>
      )}
      {usage.columns.grapple && cov.grapplePulls > 0 && (
        <p>{u.coverageGrappleFmt(cov.grapplePulls, cov.grapplePullLives)}</p>
      )}
      {orphelins > 0 && <p>{u.unattributedFmt(orphelins)}</p>}
      <p>{u.notMeasured}</p>
    </div>
  )
}
