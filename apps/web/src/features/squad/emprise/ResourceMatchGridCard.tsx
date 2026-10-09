/**
 * ResourceMatchGridCard — « Contrôle des ressources, match par match » (Escouade › Emprise, bloc
 * « Carte par carte » ; lot L5.2 du plan PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26, maquette de
 * l'onglet, `renderBand`).
 *
 * Colonnes = les matchs de la soirée : heure, carte, mode, pastille « Victoire 3–0 » (S9) et badge du
 * drapeau de dominance quand il existe et que l'appelant le demande (pas sur l'Escouade). La table elle-même (lignes, cases, râteliers repliés,
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
  /** Les libellés du drapeau de dominance ; absents : pas de badge de dominance sous les matchs. */
  dominanceLabels?: Record<DominanceValue, string>
  /**
   * Infobulle « Équipe : … » des seuls joueurs nommés (Escouade) : les prises des joueurs inconnus
   * n'y figurent pas. Faux : elles s'y lisent en « reste de l'équipe ».
   */
  namedOnly?: boolean
  outcomeLabels: Record<OutcomeValue, string>
  locale: Locale
  t: EmpriseText
  /**
   * Vue compacte du tiroir de comparaison de Sessions : en-tête réduit à l'heure, la carte et
   * l'initiale du résultat (score et dominance dans l'infobulle), table compacte.
   */
  compact?: boolean
}

export function ResourceMatchGridCard({
  grid,
  itemName,
  playerName,
  dominanceLabels,
  outcomeLabels,
  locale,
  t,
  compact = false,
  namedOnly = false,
}: Props) {
  const legend = useMemo(
    () => (
      <ObjectifLegend
        ariaLabel={t.grid.title}
        items={[
          { kind: 'square', label: t.grid.more, color: `color-mix(in oklab, ${PLUS_INK} 80%, var(--muted))` },
          { kind: 'square', label: t.grid.less, color: `color-mix(in oklab, ${MINUS_INK} 80%, var(--muted))` },
          { kind: 'square', label: t.grid.nothing, color: TRACK_INK },
        ]}
      />
    ),
    [t],
  )
  const columns: GridColumn[] = grid.columns.map((m) => ({
    key: m.matchId,
    head: compact ? (
      <CompactMatchHead m={m} outcomeLabels={outcomeLabels} locale={locale} />
    ) : (
      <MatchHead m={m} dominanceLabels={dominanceLabels} outcomeLabels={outcomeLabels} locale={locale} t={t} />
    ),
    tipHead: t.grid.matchHead(
      formatMatchTime(m.startTime, locale),
      m.map,
      m.mode,
      m.outcome ? `${inSentence(outcomeLabels[m.outcome])}${m.score ? ` ${m.score}` : ''}` : null,
    ),
  }))
  const whoText = (who: GridWho[]) =>
    who
      .map((w) => ({ name: (w.xuid && playerName(w.xuid)) || (namedOnly ? '' : t.grid.restLower), n: w.taken }))
      .filter((w) => w.name !== '')
      .sort((a, b) => b.n - a.n || a.name.localeCompare(b.name))
      .map((w) => `${w.name} ${w.n}`)
      .join(', ')
  return (
    <ObjectifFrame title={t.grid.title} info={t.grid.info} legend={legend} testId="emprise-grid">
      <ResourceGridTable columns={columns} sections={grid.sections} itemName={itemName} whoText={whoText} t={t} compact={compact} />
    </ObjectifFrame>
  )
}

/**
 * L'en-tête compact d'un match : l'heure, la carte (tronquée), l'initiale du résultat dans sa couleur
 * d'issue (l'initiale du libellé du titre : « V », « D » ; « W », « L » en anglais).
 */
function CompactMatchHead({ m, outcomeLabels, locale }: { m: EmpriseMatchInfo; outcomeLabels: Record<OutcomeValue, string>; locale: Locale }) {
  return (
    <div className="min-w-0 pb-[3px] text-center text-[10.5px] leading-tight text-muted-foreground" data-testid={`emprise-grid-head-${m.matchId}`}>
      <b className="block font-medium text-foreground">{formatMatchTime(m.startTime, locale)}</b>
      <span className="block truncate">{m.map}</span>
      {m.outcome && (
        <b className="block font-bold" style={{ color: tokenCssVar(OUTCOME_TOKENS[m.outcome]) }} data-testid={`emprise-grid-result-${m.matchId}`}>
          {outcomeLabels[m.outcome].charAt(0).toLocaleUpperCase(locale)}
        </b>
      )}
    </div>
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
  dominanceLabels?: Record<DominanceValue, string>
  outcomeLabels: Record<OutcomeValue, string>
  locale: Locale
  t: EmpriseText
}) {
  const result = m.outcome ? `${outcomeLabels[m.outcome]}${m.score ? ` ${m.score}` : ''}` : null
  const dom = m.dominance && dominanceLabels ? dominanceLabels[m.dominance] : null
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
