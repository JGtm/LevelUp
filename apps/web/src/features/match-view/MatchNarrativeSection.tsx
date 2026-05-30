/**
 * MatchNarrativeSection — section narrative du MatchView V2.
 *
 * Rend les nouveaux champs livrés par MV1-MV4 :
 *   - DominanceBadge (header — chunk MV1)
 *   - Cadence intra-match (chunk MV2 — wrapper <BarStackedChart>)
 *   - Impact 8 rôles (chunk MV2 — liste de badges narratifs)
 *   - Radar 6 axes (chunk MV4.B — wrapper <RadarChart>)
 *
 * Composant standalone : peut être inséré dans MatchViewPage.tsx ou utilisé
 * indépendamment. Les sections sans données ne sont pas rendues
 * (dégradation gracieuse).
 */
import { BarStackedChart } from '@/components/charts/BarStackedChart'
import type { ChartPointStacked } from '@/components/charts/BarStackedChart'
import type { ChartSeries } from '@/components/charts/ChartCard'
import { RadarChart, type RadarSeriesPayload } from '@/components/charts/RadarChart'
import { tokenCssVar } from '@/lib/accessibility'
import { displayPlayerName } from '@/lib/players/displayName'

import type {
  MatchViewCadence,
  MatchViewHeader,
  MatchViewImpactRole,
  MatchViewRadarSeries,
} from '@/lib/api/types'

export interface MatchNarrativeSectionProps {
  header: MatchViewHeader
  impactRoles?: MatchViewImpactRole[]
  cadence?: MatchViewCadence | null
  radar?: MatchViewRadarSeries[]
  /** Labels passés par le caller (résolus via i18n manifest match_view.toml). */
  labels: MatchNarrativeLabels
  /** Mapping xuid -> gamertag pour rendre les rôles d'impact lisibles. */
  gamertagByXUID?: Record<string, string>
}

export interface MatchNarrativeLabels {
  sectionTitle: string
  cadenceTitle: string
  impactSectionTitle: string
  radarSectionTitle: string
  /** Résolveur i18n pour les LabelKey narrative.role.* / narrative.dominance.* */
  resolveLabelKey: (key: string) => string
  /** Résolveur axes radar (combat -> "Combat" etc.). */
  resolveAxisLabel: (axis: string) => string
}

export function MatchNarrativeSection({
  header,
  impactRoles,
  cadence,
  radar,
  labels,
  gamertagByXUID,
}: MatchNarrativeSectionProps) {
  const hasContent =
    !!header.dominance_badge ||
    !!cadence ||
    (impactRoles && impactRoles.length > 0) ||
    (radar && radar.length > 0)

  if (!hasContent) {
    return null
  }

  return (
    <section
      className="flex flex-col gap-4 rounded-lg border border-border bg-card p-4"
      data-testid="match-narrative-section"
    >
      <h2 className="text-base font-semibold">{labels.sectionTitle}</h2>

      {header.dominance_badge && (
        <DominanceBadgePill
          flag={header.dominance_badge.flag}
          labelKey={header.dominance_badge.label_key}
          colorToken={header.dominance_badge.color_token}
          resolveLabelKey={labels.resolveLabelKey}
        />
      )}

      {impactRoles && impactRoles.length > 0 && (
        <ImpactRolesList
          roles={impactRoles}
          title={labels.impactSectionTitle}
          resolveLabelKey={labels.resolveLabelKey}
          gamertagByXUID={gamertagByXUID}
        />
      )}

      {cadence && cadence.datapoints.length > 0 && (
        <BarStackedChart
          title={labels.cadenceTitle}
          series={[asCadenceSeries(cadence)]}
          height={220}
        />
      )}

      {radar && radar.length > 0 && (
        <RadarChart
          title={labels.radarSectionTitle}
          series={radar.map((s) => asRadarPayload(s, labels.resolveAxisLabel))}
          height={300}
          axisLabels={radarAxisLabelsFromResolver(labels.resolveAxisLabel)}
        />
      )}
    </section>
  )
}

function asCadenceSeries(cadence: MatchViewCadence): ChartSeries<ChartPointStacked> {
  return {
    key: cadence.key,
    labelKey: cadence.label_key,
    datapoints: cadence.datapoints,
    meta: cadence.meta,
  }
}

function asRadarPayload(
  s: MatchViewRadarSeries,
  resolveAxisLabel: (axis: string) => string,
): RadarSeriesPayload {
  return {
    key: `match.radar.${s.xuid}`,
    axes: s.axes.map((ax) => ({
      axis: resolveAxisLabel(ax.Axis),
      value: ax.Value,
      raw: ax.Raw,
    })),
    meta: { gamertag: s.gamertag, mode_family: s.mode_family },
  }
}

/**
 * radarAxisLabelsFromResolver : produit le map { axis_canon : label_localise }
 * attendu par RadarChart.axisLabels. Couvre les 6 axes canoniques.
 */
function radarAxisLabelsFromResolver(
  resolve: (axis: string) => string,
): Record<string, string> {
  return {
    combat: resolve('combat'),
    survival: resolve('survival'),
    support: resolve('support'),
    score: resolve('score'),
    objective: resolve('objective'),
    impact: resolve('impact'),
  }
}

interface DominanceBadgePillProps {
  flag: number
  labelKey: string
  colorToken: string
  resolveLabelKey: (key: string) => string
}

function DominanceBadgePill({
  labelKey,
  colorToken,
  resolveLabelKey,
}: DominanceBadgePillProps) {
  return (
    <div
      className="inline-flex w-fit items-center gap-2 rounded-full px-3 py-1 text-sm font-medium"
      style={{
        backgroundColor: `color-mix(in oklab, ${tokenCssVar(colorToken as never)} 20%, transparent)`,
        color: tokenCssVar(colorToken as never),
        borderColor: tokenCssVar(colorToken as never),
        borderWidth: 1,
      }}
      data-testid="match-narrative-dominance-badge"
    >
      <span>{resolveLabelKey(labelKey)}</span>
    </div>
  )
}

interface ImpactRolesListProps {
  roles: MatchViewImpactRole[]
  title: string
  resolveLabelKey: (key: string) => string
  gamertagByXUID?: Record<string, string>
}

function ImpactRolesList({
  roles,
  title,
  resolveLabelKey,
  gamertagByXUID,
}: ImpactRolesListProps) {
  return (
    <div className="flex flex-col gap-2">
      <h3 className="text-sm font-medium">{title}</h3>
      <div className="flex flex-wrap gap-2" data-testid="match-narrative-impact-roles">
        {roles.map((role, idx) => {
          const gt = displayPlayerName(gamertagByXUID?.[role.xuid], role.xuid)
          return (
            <div
              key={`${role.xuid}-${role.role_key}-${idx}`}
              className="inline-flex items-center gap-2 rounded-md border px-2 py-1 text-xs"
              style={{
                backgroundColor: `color-mix(in oklab, ${tokenCssVar(role.color_token as never)} 12%, transparent)`,
                borderColor: tokenCssVar(role.color_token as never),
              }}
            >
              <span className="font-medium">{gt}</span>
              <span className="text-muted-foreground">·</span>
              <span style={{ color: tokenCssVar(role.color_token as never) }}>
                {resolveLabelKey(role.label_key)}
              </span>
            </div>
          )
        })}
      </div>
    </div>
  )
}
