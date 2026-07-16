/**
 * SessionSummaryCard — bande KPI plate des performances d'une session.
 *
 * Look aligné sur la section "Mes stats sur cette session" des pages Solo/Escouade
 * (`_shared/SessionBriefing/KpiGrid`) : grille de petites cards plates
 * (`rounded border bg-card`), pas de Card englobante, pas de titre. L'identité de la
 * session (label, catégorie, nb de matchs, durée) vit dans le header L3 / drawer via
 * `SessionParamPills`.
 *
 * Les KPIs redondants avec le donut d'issues (Victoires/Défaites, Taux de victoire)
 * et avec le tableau (KDR) ont été retirés. À la place : Rendement/Résistance (OC/DR,
 * même composite que la home) et Durée de vie moyenne (même rendu que "Mes stats").
 */
import { CombatYieldDisplay } from '@/components/ui/combat-yield-display'
import { KpiCard } from '@/components/cards/KpiCard'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { tokenCssVar, type SemanticToken } from '@/lib/accessibility'
import { accuracyScale, kdScale, lifespanScale, perfScale } from '@/lib/accessibility/scales'
import { formatDurationMMSS, formatDurationMShort, displayRatingLabel } from '@/lib/formatters'
import { combatYieldToken } from '@/lib/formatters/combatYield'
import { useProvidesDamageTaken } from '@/lib/damage/effectiveHp'
import type { SessionCompareEntry } from '@/lib/api/types'
import { useFieldMappings } from '@/lib/i18n/fieldMappings'

import { formatNumber, formatPercent, formatRankDelta, rankDeltaToken, useSessionT } from './_shared'
import { log } from './_logger'

interface Props {
  entry: SessionCompareEntry | null
  /** Colonne divisée (drawer compare ouvert) : libellés longs abrégés pour tenir sur une ligne. */
  compact?: boolean
}

// Accent perf via la source unique perfScale (80/65/50/35, alignés sur
// analysis.PerfTier côté Go). null → undefined (pas d'accent). Fin adaptateur, pas
// une échelle recopiée (le mapping seuils→tiers reste dans perfScale).
function perfAccent(score: number | null): SemanticToken | undefined {
  return score == null ? undefined : perfScale(score)
}

export function SessionSummaryCard({ entry, compact = false }: Props) {
  const { data: fieldMappings } = useFieldMappings()
  const labelOf = (key: string): string => fieldMappings?.fields[key]?.label ?? key
  const t = useSessionT()
  // false (Halo 5) → DR non calculable : on neutralise la Résistance dans l'accent
  // (sinon DR=0 tirerait l'accent vers le rouge) ; le composite reste visible dès
  // que le Rendement (OC) existe, et affiche N/A pour la Résistance.
  const providesDamageTaken = useProvidesDamageTaken()

  if (!entry) {
    return (
      <EmptyStateNotice
        title={t('session.detail.summary_unavailable_title')}
        description={t('session.detail.summary_unavailable_description')}
      />
    )
  }

  const drForToken = providesDamageTaken ? entry.avg_dr : null
  const hasOffDef = entry.avg_oc != null || (providesDamageTaken && entry.avg_dr != null)
  // Observabilité : KPI Rendement/Résistance à "—" = OC/DR absents (pas de dégâts).
  if (!hasOffDef) {
    log.warn(
      `offdef_missing:${entry.session_label ?? ''}`,
      'KPI Rendement/Résistance vide : avg_oc/avg_dr absents (aucun match avec dégâts ?)',
    )
  }

  return (
    // gap serré : la bande doit tenir sur une ligne même en colonne divisée (drawer
    // compare ouvert). overflow-x-auto reste un filet pour les cas extrêmes (6 tuiles
    // sur très petit écran).
    <div className="flex gap-1.5 overflow-x-auto pb-0.5">
      {/* Précision moyenne de la session (le KDA est désormais au centre du donut F/D/A). */}
      <KpiStat
        label={labelOf('accuracy')}
        value={entry.avg_accuracy != null ? formatPercent(entry.avg_accuracy * 100) : '—'}
        accent={entry.avg_accuracy != null ? accuracyScale(entry.avg_accuracy * 100) : undefined}
      />
      <KpiStat
        label={t('session.detail.stat_kills_per_match')}
        value={formatNumber(entry.kills_per_match, 1)}
        accent={kdScale(entry.kdr)}
      />
      <KpiStat
        label={t('session.detail.stat_avg_life')}
        value={entry.avg_life_seconds != null ? formatDurationMMSS(entry.avg_life_seconds) : '—'}
        // Tooltip désambiguïsant le MM:SS (ex. "1:05" → "1m05s"), même formatter canonique
        // que la tuile Durée de vie du SessionBriefing.
        valueTitle={entry.avg_life_seconds != null ? formatDurationMShort(entry.avg_life_seconds) : undefined}
        accent={entry.avg_life_seconds != null ? lifespanScale(entry.avg_life_seconds) : undefined}
      />
      <KpiStat
        label={t('session.detail.stat_perf_score')}
        value={formatNumber(entry.performance_score, 1)}
        token={perfAccent(entry.performance_score)}
        accent={perfAccent(entry.performance_score)}
      />
      {/* Rendement / Résistance : tile plus large (barre composite responsive).
          Accent = qualité combinée vs référence (rendement ≥ 100% & résistance
          ≥ +0% → vert ; les deux en-dessous → rouge ; mixte → neutre). */}
      <KpiCard
        accent={combatYieldToken(entry.avg_oc, drForToken)}
        className="flex-[2] min-w-[9.5rem]"
      >
        <div className="px-3 py-2">
          <p className="text-3xs font-medium uppercase tracking-wide text-muted-foreground">
            {t(compact ? 'session.detail.stat_off_def_short' : 'session.detail.stat_off_def')}
          </p>
          <div className="mt-1.5">
            {hasOffDef ? (
              <CombatYieldDisplay
                className="w-full"
                offensiveConversion={entry.avg_oc}
                defensiveResistance={entry.avg_dr}
                dmgPerKill={entry.dmg_per_kill}
                dmgPerDeath={entry.dmg_per_death}
                align="start"
              />
            ) : (
              <p className="text-lg font-bold text-muted-foreground">—</p>
            )}
          </div>
        </div>
      </KpiCard>
      {entry.skill_rating_delta != null && entry.skill_rating_type ? (
        <KpiStat
          label={`Δ ${displayRatingLabel(entry.skill_rating_type) ?? ''}`}
          value={formatRankDelta(entry.skill_rating_delta, entry.skill_rating_type)}
          token={rankDeltaToken(entry.skill_rating_delta)}
          accent={rankDeltaToken(entry.skill_rating_delta)}
        />
      ) : null}
    </div>
  )
}

function KpiStat({
  label,
  value,
  token,
  accent,
  valueTitle,
}: {
  label: string
  value: string
  token?: SemanticToken
  /** Barre d'accent du catalogue (qualité de la métrique). */
  accent?: SemanticToken
  /** Tooltip natif (attribut title) au survol — lève l'ambiguïté d'un MM:SS. */
  valueTitle?: string
}) {
  return (
    <KpiCard accent={accent} className="flex-1 min-w-[5rem]">
      <div className="px-3 py-2" title={valueTitle}>
        <p className="text-3xs font-medium uppercase tracking-wide text-muted-foreground">{label}</p>
        <p className="mt-0.5 text-lg font-bold text-foreground" style={token ? { color: tokenCssVar(token) } : undefined}>
          {value}
        </p>
      </div>
    </KpiCard>
  )
}
