/**
 * MatchObjectivesSection — section « Objectifs » de la page match (V72-03).
 *
 * CE QU'ELLE MONTRE, ET SOUS QUELLE FORME. Les statistiques d'objectif du mode joué (drapeau /
 * zones — Bastion et King of the Hill / crâne / réserve / extraction / VIP / Assaut), en DEUX VUES
 * empilées depuis le 2026-09-03 (retours utilisateur : le tableau était hors style et illisible
 * d'un coup d'œil) :
 *   1. « Actions d'objectif par joueur » — la grille partagée `components/charts/ValueGrid` :
 *      lignes = joueurs camp par camp, colonnes = grandeurs du mode, chacune avec SON échelle ;
 *   2. « Total d'objectif par équipe » — un face-à-face, zéro au centre, une échelle par ligne :
 *      la longueur d'un côté dit l'AVANTAGE sur cette grandeur, pas sa valeur absolue.
 *
 * RIEN N'EST ÉCRIT EN DUR SUR LE MODE. Les colonnes viennent de `objectiveColsFor(mode)` et les
 * totaux d'équipe de `objectiveTeamTotal` (cumul, ou MAXIMUM pour un « meilleur temps » —
 * additionner des meilleurs temps n'a aucun sens). Les deux vues encaissent donc de 3 à 5
 * grandeurs sans une ligne de code par mode. Détail de la projection : `objectivesChart.ts`.
 *
 * Double porte d'affichage, inchangée :
 *   - capability `objective_stats` (titre) via useCapability — masquée sur un titre qui ne la
 *     déclare pas (Halo 5) ;
 *   - data-driven : aucune ligne ne porte de bloc objectif (Slayer) → mode == null → rien
 *     affiché, pas de section vide.
 *
 * L'ASSAUT EST DANS LES DEUX VUES DEPUIS LE 2026-09-04, et cet en-tête a porté l'inverse : ses
 * quatre grandeurs ne viennent PAS de l'API — qui n'en publie aucune — mais du FILM, reconstruites
 * à la cuisson et servies par `match_bomb_stats_latest` sous la capability `film.bomb_stats`. Rien
 * de particulier n'est écrit ici pour elles : le mode se détecte comme les autres
 * (`detectObjectiveMode`) et ses colonnes sortent d'`objectiveColsFor`.
 *
 * COULEURS D'ÉQUIPE : les jetons `team-ally` / `team-enemy` via `teamTokenCssVar`, donc la
 * palette d'accessibilité réglée par l'utilisateur. C'est un changement assumé du 2026-09-03 :
 * la section employait la cascade d'IDENTITÉ (`teamColorResolver`, couleur officielle du jeu)
 * tant qu'elle était un tableau à bandeaux d'équipe. En devenant un GRAPHE elle suit la règle
 * des graphes — la frontière entre les deux familles est écrite en tête de `teamSeriesColor.ts`.
 */
import { useMemo } from 'react'

import { ChartLegend } from '@/components/charts/ChartLegend'
import { ValueGrid } from '@/components/charts/ValueGrid'
import { SectionCard } from '@/components/ui/section-card'
import { Tooltip } from '@/components/ui/tooltip'
import type { MatchScoreboardRow } from '@/lib/api/types'
import { useCapability } from '@/lib/capabilities/capabilities'
import { resolveTeamLabel } from '@/lib/halo/teamLabel'
import { HeaderLabelTooltip } from '@/lib/table/columnMeta'

import type { MatchViewText } from './i18n'
import {
  buildObjectiveDuel,
  buildObjectiveGrid,
  type ObjectiveDuelRow,
  type ObjectiveDuelSide,
} from './objectivesChart'
import { teamTokenCssVar } from './teamSeriesColor'
import {
  detectObjectiveMode,
  objectiveColsFor,
  type ObjectiveColSpec,
} from './MatchScoreboard.logic'

interface Props {
  rows: MatchScoreboardRow[]
  /** team_side distincts du lobby (ex: "t0", "t1"), déjà triés par le parent. */
  teams: string[]
  /** team_side du joueur principal (is_me) — pour le repli couleur ally/enemy. */
  myTeamSide: string | null
  t: MatchViewText
}

export function MatchObjectivesSection({ rows, teams, myTeamSide, t }: Props) {
  const hasObjectiveCap = useCapability('objective_stats')
  const mode = useMemo(() => detectObjectiveMode(rows), [rows])

  // Les lignes PORTEUSES d'un bloc d'objectif, groupées dans l'ordre des camps du lobby : un
  // joueur se lit en ligne d'une colonne à l'autre, et les camps ne s'entrelacent pas.
  const withObjective = useMemo(
    () =>
      teams.flatMap((side) =>
        rows.filter((r) => (r.team_side ?? '') === side && r.objective != null),
      ),
    [rows, teams],
  )
  // Les camps qui ont effectivement des lignes — un camp vide ne prend pas de place à l'écran.
  const sides = useMemo(
    () => teams.filter((side) => withObjective.some((r) => (r.team_side ?? '') === side)),
    [teams, withObjective],
  )

  // Double porte : capability titre absente OU aucun bloc objectif → rien.
  if (!hasObjectiveCap || mode == null || withObjective.length === 0) return null

  // Sous la porte : la projection est une poignée de multiplications sur huit lignes — la
  // mémoriser coûterait plus de dépendances à tenir juste qu'elle ne fait économiser.
  const cols = objectiveColsFor(mode)
  const colLabel = (col: ObjectiveColSpec) =>
    t.objectives.cols[String(col.key)]?.label ?? String(col.key)
  const teamLabel = (side: string) =>
    resolveTeamLabel(
      rows.filter((r) => (r.team_side ?? '') === side),
      side,
      t,
    )
  // « Allié » = du côté du joueur de la page. Sans `is_me` au tableau des scores, le camp est
  // INCONNU : encre neutre, jamais l'une des deux couleurs (cf. `teamTokenCssVar`).
  const teamColor = (side: string) =>
    teamTokenCssVar(myTeamSide == null || side === '' ? null : side === myTeamSide)
  const grid = buildObjectiveGrid({
    rows: withObjective,
    cols,
    colLabel,
    teamLabel,
    teamColor,
    tipFmt: t.objectives.gridTipFmt,
  })

  return (
    <SectionCard title={t.objectives.title} label={t.objectives.title}>
      <div className="space-y-5 px-3 pb-3 pt-3">
        <section aria-label={t.objectives.viewByPlayer}>
          <ViewTitle>{t.objectives.viewByPlayer}</ViewTitle>
          <ValueGrid model={grid} />
        </section>
        {/* LE FACE-À-FACE N'A DE SENS QU'À DEUX CAMPS. Un lobby qui n'en présente pas exactement
            deux (donnée partielle, mode à plus de deux camps) garde la vue par joueur seule :
            mieux vaut une vue de moins qu'un « face-à-face » qui en tairait un troisième. */}
        {sides.length === 2 && (
          <section aria-label={t.objectives.viewTeamTotals}>
            <ViewTitle>{t.objectives.viewTeamTotals}</ViewTitle>
            <ObjectiveDuel
              duel={buildObjectiveDuel({
                rows: withObjective,
                teams: [sides[0], sides[1]],
                cols,
                colLabel,
                teamLabel,
                teamColor,
                tipFmt: t.objectives.duelTipFmt,
              })}
              cols={cols}
              t={t}
            />
            <ChartLegend
              className="pt-3"
              items={sides.map((side) => ({
                key: side,
                label: teamLabel(side),
                color: teamColor(side),
              }))}
            />
          </section>
        )}
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
 * ObjectiveDuel — le face-à-face : une ligne par grandeur, le zéro au centre.
 *
 * Le libellé central porte l'aide de la colonne (`cols[].tooltip` de l'i18n) : c'est le seul
 * endroit où la grandeur est nommée en toutes lettres au milieu de l'écran, donc le seul où son
 * explication se cherche.
 */
function ObjectiveDuel({
  duel,
  cols,
  t,
}: {
  duel: ObjectiveDuelRow[]
  cols: ObjectiveColSpec[]
  t: MatchViewText
}) {
  return (
    <div className="overflow-x-auto">
      <div className="min-w-[420px] space-y-2.5">
        {duel.map((row, i) => (
          <div key={row.key} className="grid grid-cols-[1fr_168px_1fr] items-center gap-3">
            <DuelSide side={row.left} align="end" />
            <div className="text-center text-3xs text-muted-foreground">
              <HeaderLabelTooltip text={t.objectives.cols[String(cols[i].key)]?.tooltip} focusable>
                <span>{row.label}</span>
              </HeaderLabelTooltip>
            </div>
            <DuelSide side={row.right} align="start" />
          </div>
        ))}
      </div>
    </div>
  )
}

/** Un côté du face-à-face : la valeur au bout de la barre, la barre vers le centre. */
function DuelSide({ side, align }: { side: ObjectiveDuelSide; align: 'start' | 'end' }) {
  const value = (
    <span className="text-xs tabular-nums">{side.text}</span>
  )
  // La barre part du CENTRE : le côté gauche colle sa barre à droite de sa cellule, le côté
  // droit à gauche de la sienne — c'est ce qui met le zéro au milieu de la ligne.
  const bar = (
    <Tooltip
      content={side.tooltip}
      className={`min-w-0 flex-1${align === 'end' ? ' justify-end' : ''}`}
    >
      <div
        className="h-[18px]"
        tabIndex={0}
        role="img"
        aria-label={side.tooltip}
        style={{ backgroundColor: side.color, width: `${side.fraction * 100}%` }}
      />
    </Tooltip>
  )
  return (
    <div className={`flex items-center gap-2 ${align === 'end' ? 'justify-end' : ''}`}>
      {align === 'end' ? value : bar}
      {align === 'end' ? bar : value}
    </div>
  )
}
