/**
 * ResourceMatchGridCard — « Contrôle des ressources, match par match » (Escouade › Emprise, bloc
 * « Carte par carte » ; lot L5.2 du plan PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26, maquette de
 * l'onglet, `renderBand`).
 *
 * Colonnes = les matchs de la soirée : heure, carte, mode, pastille « Victoire 3–0 » (S9) et badge du
 * drapeau de dominance quand il existe. La table elle-même (lignes, cases, râteliers repliés,
 * infobulles) est `ResourceGridTable`, partagée avec la grille par carte des Séries temporelles.
 */
import { useMemo } from 'react'

import { Tooltip } from '@/components/ui/tooltip'
import type { DominanceValue, OutcomeValue } from '@/components/charts/outcomeSequence'
import { tokenCssVar } from '@/lib/accessibility'
import type { Locale } from '@/lib/i18n/locale'
import { DOMINANCE_COLOR_TOKENS } from '@/lib/narrative/dominance'

import { MINUS_INK, PLUS_INK, TRACK_INK } from '../formes/colors'
import { formatMatchTime } from '../formes/format'
import { ObjectifFrame, ObjectifLegend } from '../objectif/ObjectifFrame'
import type { EmpriseMatchInfo, GridRow, GridWho, MatchGrid } from './emprise.logic'
import type { EmpriseText } from './empriseStrings'
import { ResourceGridTable, type GridColumn } from './ResourceGridTable'
import { TipText } from './TipText'
import { inSentence } from './useOutcomeLabels'

const OUTCOME_TOKENS = {
  win: 'outcome-win',
  loss: 'outcome-loss',
  tie: 'outcome-draw',
  dnf: 'outcome-dnf',
} as const

interface Props {
  grid: MatchGrid
  /** Le nom d'un objet (bonus nommé par le web, arme par le titre). */
  itemName: (row: GridRow) => string
  /** Le nom d'un joueur de l'escouade par xuid (vide = inconnu). */
  playerName: (xuid: string) => string
  dominanceLabels: Record<DominanceValue, string>
  outcomeLabels: Record<OutcomeValue, string>
  locale: Locale
  t: EmpriseText
}

export function ResourceMatchGridCard({ grid, itemName, playerName, dominanceLabels, outcomeLabels, locale, t }: Props) {
  const legend = useMemo(
    () => (
      <ObjectifLegend
        ariaLabel={t.grid.title}
        items={[
          { kind: 'square', label: t.grid.more, color: `color-mix(in oklab, ${PLUS_INK} 80%, var(--muted))` },
          { kind: 'square', label: t.grid.less, color: `color-mix(in oklab, ${MINUS_INK} 80%, var(--muted))` },
          { kind: 'square', label: t.grid.nothing, color: TRACK_INK },
          { kind: 'hatch', label: t.grid.noFilm },
        ]}
      />
    ),
    [t],
  )
  const columns: GridColumn[] = grid.columns.map((m) => ({
    key: m.matchId,
    head: <MatchHead m={m} dominanceLabels={dominanceLabels} outcomeLabels={outcomeLabels} locale={locale} t={t} />,
    tipHead: t.grid.matchHead(
      formatMatchTime(m.startTime, locale),
      m.map,
      m.mode,
      m.outcome ? `${inSentence(outcomeLabels[m.outcome])}${m.score ? ` ${m.score}` : ''}` : null,
    ),
  }))
  const whoText = (who: GridWho[]) =>
    who
      .map((w) => ({ name: w.xuid ? playerName(w.xuid) || t.grid.restLower : t.grid.restLower, n: w.taken }))
      .sort((a, b) => b.n - a.n || a.name.localeCompare(b.name))
      .map((w) => `${w.name} ${w.n}`)
      .join(', ')
  return (
    <ObjectifFrame title={t.grid.title} info={t.grid.info} legend={legend} testId="emprise-grid">
      <ResourceGridTable columns={columns} sections={grid.sections} itemName={itemName} whoText={whoText} t={t} />
    </ObjectifFrame>
  )
}

function MatchHead({
  m,
  dominanceLabels,
  outcomeLabels,
  locale,
  t,
}: {
  m: EmpriseMatchInfo
  dominanceLabels: Record<DominanceValue, string>
  outcomeLabels: Record<OutcomeValue, string>
  locale: Locale
  t: EmpriseText
}) {
  const result = m.outcome ? `${outcomeLabels[m.outcome]}${m.score ? ` ${m.score}` : ''}` : null
  const dom = m.dominance ? dominanceLabels[m.dominance] : null
  const domInk = m.dominance ? tokenCssVar(DOMINANCE_COLOR_TOKENS[m.dominance]) : ''
  return (
    <div className="pb-[3px] text-center text-[11px] leading-tight text-muted-foreground" data-testid={`emprise-grid-head-${m.matchId}`}>
      {formatMatchTime(m.startTime, locale)}
      <b className="block text-xs font-medium text-foreground">{m.map}</b>
      {m.mode}
      {result && m.outcome && (
        <div>
          <span
            className="mt-[3px] inline-block rounded-full px-[7px] text-[10.5px] font-bold text-white"
            style={{ backgroundColor: tokenCssVar(OUTCOME_TOKENS[m.outcome]) }}
            data-testid={`emprise-grid-result-${m.matchId}`}
          >
            {result}
          </span>
        </div>
      )}
      {dom && (
        <div>
          <Tooltip content={<TipText text={t.grid.dominanceTip(dom)} />}>
            <span
              className="mt-[3px] inline-block rounded-full px-[7px] text-[10.5px] font-semibold text-foreground"
              style={{
                backgroundColor: `color-mix(in oklab, ${domInk} 22%, var(--card))`,
                boxShadow: `inset 0 0 0 1.5px ${domInk}`,
              }}
              data-testid={`emprise-grid-dominance-${m.matchId}`}
            >
              {dom}
            </span>
          </Tooltip>
        </div>
      )}
    </div>
  )
}
