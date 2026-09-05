/**
 * MatchEquipmentUsageSection — LE BILAN D'ÉQUIPEMENT D'UN MATCH, par joueur et par équipe.
 *
 * CE QUE LA PAGE NE SAVAIT PAS DIRE. Le rejeu 2D montre chaque geste d'équipement à l'IMAGE où
 * il a lieu : un mur posé à 3:12, une traction de grappin à 5:40. Personne ne peut en tirer
 * « qui a posé six murs » en regardant six instants séparés de trois minutes. Cette section les
 * COMPTE — et c'est tout ce qu'elle ajoute : aucune donnée nouvelle, aucun appel de plus.
 *
 * DEUX VUES, PLUS UN TABLEAU (2026-09-03, retours utilisateur sur l'onglet Chronologie). Le
 * tableau à deux niveaux d'en-tête donnait la bonne mesure dans la mauvaise forme : une grille
 * de chiffres où l'œil ne trouve ni le geste dominant ni le camp qui s'en est servi. À sa place,
 * dans la même carte :
 *   1. « Nombre de gestes par joueur » — la grille partagée `components/charts/ValueGrid` :
 *      lignes = joueurs dans l'ordre du roster, camp par camp, filet entre les deux camps ;
 *      colonnes = grandeurs, CHACUNE AVEC SON ÉCHELLE (un mur se compare à un mur) ;
 *   2. « Part de chaque équipe, geste par geste » — une barre 100 % par famille de geste, le
 *      compte brut ET le pourcentage écrits dans le segment.
 * Les colonnes restent DÉCIDÉES PAR LA DONNÉE (`usageColumnGroups`) : aucune liste en dur.
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
 *     court, pas d'où il vient. La réserve est en infobulle, portée par le NOM DE LA FAMILLE
 *     dans la vue 2 (une famille y a exactement une ligne, donc exactement un endroit où sa
 *     réserve se lit) ;
 *   - les socles de bonus vidés sont ANONYMES par mesure : une ligne au niveau du MATCH, jamais
 *     une colonne de joueur ;
 *   - répulseur et propulseur n'ont aucune colonne, et une phrase dit pourquoi — mais pas pour
 *     la même raison depuis le 2026-09-03. Le RÉPULSEUR reste non mesuré (neuf canaux du film
 *     fouillés, négatif) : une colonne de zéros se lirait « zéro utilisation » là où la vérité
 *     est « non mesuré ». Le PROPULSEUR, lui, EST mesuré (schéma 38) et n'a toujours pas de
 *     colonne : décision utilisateur — le geste dure une demi-seconde, il se lit sur la carte
 *     du rejeu (le dash du pion), pas dans un compte de tableau.
 *
 * COULEURS. Les familles de geste prennent la table d'encres de `equipmentUsageChart` (jetons
 * sémantiques, jamais un hex) ; les camps prennent `teamTokenCssVar` — les jetons `team-ally` /
 * `team-enemy` que les réglages d'accessibilité surchargent, et NON la cascade d'identité de
 * `teamColor.ts` (cf. l'en-tête de `match-view/teamSeriesColor.ts` pour la frontière).
 *
 * Aucun calcul ici : tout vient de `equipmentUsageLogic` (les mesures),
 * `equipmentUsageColumns` (les colonnes et leurs noms) et `equipmentUsageChart` (la projection).
 */
import { useCallback, useMemo } from 'react'

import { ChartLegend } from '@/components/charts/ChartLegend'
import { ValueGrid } from '@/components/charts/ValueGrid'
import { SectionCard } from '@/components/ui/section-card'
import { Tooltip } from '@/components/ui/tooltip'
import { teamTokenCssVar } from '@/features/match-view/teamSeriesColor'
import type { MatchScoreboardRow } from '@/lib/api/types'
import { resolveTeamLabel } from '@/lib/halo/teamLabel'
import { HeaderLabelTooltip } from '@/lib/table/columnMeta'

import {
  buildUsageGrid,
  buildUsageShares,
  usageGroupColor,
  type UsageShareRow,
} from './equipmentUsageChart'
import { equipmentFamilyLabel, usageColumnGroups } from './equipmentUsageColumns'
import { buildEquipmentUsage, tallyTotal, type EquipmentUsage } from './equipmentUsageLogic'
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
  const meRow = useMemo(() => board.find((r) => r.is_me), [board])
  const meSide = meRow?.team_side ?? null

  const teamLabel = useCallback(
    (side: string | null) =>
      resolveTeamLabel(
        side ? board.filter((r) => (r.team_side ?? '') === side) : [],
        side,
        t,
      ),
    [board, t],
  )
  // « Allié » = du côté du joueur de la page. Sans `is_me` au tableau des scores, ou pour un
  // joueur que le film a vu vivre sans ligne de scoreboard, le camp est INCONNU (null) : encre
  // neutre, jamais l'une des deux couleurs d'équipe (même règle que `ReplayTeamHeader`).
  const teamAccent = useCallback(
    (side: string | null) =>
      teamTokenCssVar(side == null || meSide == null ? null : side === meSide),
    [meSide],
  )

  const grid = useMemo(
    () =>
      buildUsageGrid({
        teams: usage?.byTeam ?? [],
        groups,
        meXUID: meRow?.xuid ?? null,
        teamLabel,
        teamAccent,
        tipFmt: t.equipmentUsage.gridTipFmt,
      }),
    [usage, groups, meRow, teamLabel, teamAccent, t],
  )
  const shares = useMemo(
    () => buildUsageShares({ teams: usage?.byTeam ?? [], groups, teamLabel, teamAccent }),
    [usage, groups, teamLabel, teamAccent],
  )

  // Double porte : pas d'artefact, ou rien de mesuré -> rien du tout.
  if (!usage?.hasData) return null

  return (
    <SectionCard
      title={t.equipmentUsage.title}
      label={t.equipmentUsage.title}
      footer={<UsageFootnotes usage={usage} t={t} />}
    >
      <div className="space-y-5 px-3 pb-3 pt-3">
        <section aria-label={t.equipmentUsage.viewByPlayer}>
          <ViewTitle>{t.equipmentUsage.viewByPlayer}</ViewTitle>
          <ValueGrid model={grid} />
          <ChartLegend
            className="pt-2"
            items={groups.map((g) => ({
              key: g.key,
              label: g.label,
              color: usageGroupColor(g.key),
            }))}
          />
        </section>
        <section aria-label={t.equipmentUsage.viewTeamShare}>
          <ViewTitle>{t.equipmentUsage.viewTeamShare}</ViewTitle>
          <UsageTeamShares rows={shares} t={t} />
        </section>
      </div>
    </SectionCard>
  )
}

/** Le titre d'une vue à l'intérieur de la carte : les deux vues répondent à deux questions. */
function ViewTitle({ children }: { children: string }) {
  return (
    <h4 className="mb-2 text-3xs font-semibold uppercase tracking-wider text-muted-foreground">
      {children}
    </h4>
  )
}

/**
 * UsageTeamShares — la vue 2 : une barre 100 % par famille de geste, un segment par camp.
 *
 * LE COMPTE BRUT ET LE POURCENTAGE SONT ÉCRITS DANS LE SEGMENT (demande utilisateur du
 * 2026-09-03) : une part sans son compte laisse croire qu'un 4-1 et un 40-10 racontent la même
 * partie. À gauche, le nom de la famille, son total, et sa pastille — la même encre que la
 * colonne correspondante de la vue 1.
 */
function UsageTeamShares({ rows, t }: { rows: UsageShareRow[]; t: ReplayText }) {
  return (
    <div className="overflow-x-auto">
      <div className="min-w-[420px] space-y-2.5">
        {rows.map((row) => (
          <div key={row.key} className="grid grid-cols-[158px_1fr] items-center gap-3.5">
            <div className="flex items-center justify-end gap-2 text-xs">
              <HeaderLabelTooltip text={row.hint} focusable>
                <span className="truncate text-right">{row.label}</span>
              </HeaderLabelTooltip>
              <span className="text-muted-foreground tabular-nums">{row.total}</span>
              <span
                className="h-2.5 w-2.5 flex-none"
                style={{ backgroundColor: row.color }}
                aria-hidden="true"
              />
            </div>
            <div className="flex h-[22px] gap-0.5">
              {row.segments.map((seg) => {
                const tip = t.equipmentUsage.shareTipFmt(seg.label, row.label, seg.count, row.total)
                // `text-white` : le libellé est posé SUR l'aplat du camp, quelle que soit la
                // palette réglée — ce n'est pas une couleur sémantique mais le contraste d'un
                // texte dans un aplat (même usage que `MatchNemesisCards`).
                return (
                  <Tooltip key={seg.side ?? 'sans-equipe'} content={tip} className="h-full">
                    <div
                      className="flex h-full items-center justify-center overflow-hidden whitespace-nowrap px-1 text-3xs font-semibold text-white"
                      style={{ backgroundColor: seg.accent, flexGrow: seg.count, flexBasis: 0 }}
                      tabIndex={0}
                      role="img"
                      aria-label={tip}
                    >
                      {`${seg.count} · ${seg.percent} %`}
                    </div>
                  </Tooltip>
                )
              })}
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}

/**
 * UsageFootnotes — la ligne ANONYME du match, les dénominateurs, et ce qui n'est pas mesuré.
 *
 * La ligne des socles de bonus est HORS DES DEUX VUES, et ce n'est pas une question de place :
 * le ramasseur n'est pas AFFICHÉ ici (`padPickups[].xuid` est publié depuis le schéma 30, mais
 * cet écran n'a pas été repensé pour l'exploiter — il l'est par `MatchPadControlSection`, juste
 * en dessous dans l'onglet). Une colonne, même intitulée « anonyme », finirait par se lire comme
 * une grandeur de joueur.
 */
function UsageFootnotes({ usage, t }: { usage: EquipmentUsage; t: ReplayText }) {
  const u = t.equipmentUsage
  const cov = usage.coverage
  const orphelins = tallyTotal(usage.unattributed)
  const detail = Object.entries(usage.powerupPickups)
    .map(([family, n]) => `${equipmentFamilyLabel(family, t)} ${n}`)
    .join(' · ')
  return (
    <div className="space-y-1 border-t border-border px-3 pb-2 pt-2 text-[11px] text-muted-foreground">
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
