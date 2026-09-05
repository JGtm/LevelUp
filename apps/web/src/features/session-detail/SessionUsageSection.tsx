/**
 * SessionUsageSection — LES TROIS BLOCS « usages d'équipement, socles et objectifs »
 * de la page Sessions (chantier session-usage S3, GO utilisateur 2026-09-05 sur la
 * voie de repli : grammaire du handoff §1 + modèle de la vue match livrée).
 *
 * LA DONNÉE ARRIVE DANS LA RÉPONSE EXISTANTE (`SessionPageResponse.usage`, attaché
 * par le service Go — aucune query nouvelle, aucun appel de plus). Le contexte
 * Solo/Escouade est résolu EN AMONT par le serveur : le bloc porte `squad_players`
 * et des lignes `squad` par grandeur — vides en solo. À l'écran, l'escouade ajoute
 * une ligne par coéquipier dans les grilles et découpe la piste du lobby par
 * joueur ; le solo garde moi + les agrégats.
 *
 * TROIS CARTES (le gabarit `SectionCard` de la vue match) :
 *   1. « Usages d'équipement » — grille alignée des cadences (ValueGrid), jauges de
 *      parts avec parité et étendue, bandes de régularité match par match ;
 *   2. « Contrôle des armes spéciales » — jauges des prises de socle, PISTE DU
 *      LOBBY découpée par joueur (hachuré = eux, anonyme), jauges par famille
 *      d'arme, bande de régularité ; socles de bonus ANONYMES en pied ;
 *   3. « Objectifs par rôle et par famille » — jauges par rôle, grille famille ×
 *      rôle, et (escouade) grille joueur × rôle. Son scope est INDÉPENDANT de la
 *      couverture des films.
 *
 * « Matchs mesurés N/M » est TOUJOURS visible (bandeau de titre) ; le corollaire du
 * §4 (épisode actif ≠ socle de bonus vidé, jamais additionnés) est écrit dans le
 * pied des deux blocs concernés. Bloc indisponible → carte d'état vide avec la
 * raison, jamais un crash ni un bloc fantôme.
 *
 * Aucun calcul ici : projections dans `usageLogic.ts` / `usageGrids.ts`, formes
 * dans `SessionUsageForms.tsx`.
 */
import { useMemo } from 'react'

import { ValueGrid } from '@/components/charts/ValueGrid'
import { SectionCard } from '@/components/ui/section-card'
import { tokenCssVar } from '@/lib/accessibility'
import type { SessionUsageBlock, SessionUsageMetric } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'
import { useAppShellStore } from '@/stores/appShellStore'

import { UsageGaugeGrid, UsageLobbyTrack, UsageRegularityBand } from './SessionUsageForms'
import { USAGE_TEXT, powerupLabel, roleLabel, type UsageText } from './usageI18n'
import {
  buildCadenceGrid,
  buildObjectiveFamilyGrid,
  buildSquadRoleGrid,
  usagePlayerInk,
  type UsageGridInks,
} from './usageGrids'
import {
  USAGE_METRIC_TOKENS,
  buildGaugeRow,
  buildLobbyTrack,
  buildRegularityBand,
  equipmentMetrics,
  formatUsageRate,
  metricKind,
  metricLabel,
  padMetric,
  roleToken,
  sortRoles,
  teamOfLobbyParityPct,
  usageAvailability,
  type UsageGaugeRowModel,
} from './usageLogic'

interface Props {
  /** Le bloc `usage` de la réponse — absent (vieux serveur) : rien ne se rend. */
  usage: SessionUsageBlock | null | undefined
  /** Libellé du joueur de la page (slug de la route — l'identité des lignes « moi »). */
  meLabel: string
}

/** Le titre d'une vue à l'intérieur d'une carte (même gabarit que la vue match). */
function ViewTitle({ children }: { children: string }) {
  return (
    <h4 className="mb-2 text-3xs font-semibold uppercase tracking-wider text-muted-foreground">
      {children}
    </h4>
  )
}

/** Le bandeau « Matchs mesurés N/M », posé à droite du titre de carte. */
function measuredAdornment(text: string) {
  return (label: string) => (
    <span className="flex items-baseline gap-2">
      <span>{label}</span>
      <span className="text-3xs font-medium normal-case text-muted-foreground tabular-nums">
        {text}
      </span>
    </span>
  )
}

/** Les encres partagées des grilles : colonnes par famille/rôle, lignes par joueur. */
function useGridInks(): UsageGridInks {
  return useMemo(
    () => ({
      columnColor: (key: string) => tokenCssVar(USAGE_METRIC_TOKENS[metricKind(key)]),
      rowAccent: (kind, squadIndex) =>
        kind === 'aggregate' ? undefined : usagePlayerInk(kind, squadIndex),
    }),
    [],
  )
}

export function SessionUsageSection({ usage, meLabel }: Props) {
  const locale = useAppShellStore((s) => s.locale)
  const t = USAGE_TEXT[locale]
  const availability = usageAvailability(usage, t)

  if (availability.kind === 'hidden' || usage == null) return null
  if (availability.kind === 'empty') {
    return (
      <>
        <SectionCard
          title={t.blockUnavailableTitle}
          label={t.blockUnavailableTitle}
          titleAdornment={
            usage.available
              ? measuredAdornment(t.measuredFmt(usage.matches_measured, usage.matches_total))
              : undefined
          }
        >
          <p className="px-3 pb-3 pt-3 text-sm text-muted-foreground">{availability.message}</p>
        </SectionCard>
        {/* Les objectifs vivent HORS films : ils s'affichent même sans match mesuré. */}
        {usage.available && <ObjectivesCard usage={usage} meLabel={meLabel} t={t} locale={locale} />}
      </>
    )
  }

  return (
    <>
      <EquipmentCard usage={usage} meLabel={meLabel} t={t} locale={locale} />
      <PadControlCard usage={usage} meLabel={meLabel} t={t} locale={locale} />
      <ObjectivesCard usage={usage} meLabel={meLabel} t={t} locale={locale} />
    </>
  )
}

interface CardProps {
  usage: SessionUsageBlock
  meLabel: string
  t: UsageText
  locale: Locale
}

/** Les jauges d'une liste de métriques (parités du bloc + étendue publiée). */
function metricGaugeRows(
  metrics: SessionUsageMetric[],
  usage: SessionUsageBlock,
  t: UsageText,
  locale: Locale,
): UsageGaugeRowModel[] {
  const teamOfLobby = teamOfLobbyParityPct(usage.team_size_avg, usage.lobby_size_avg)
  return metrics.map((m) =>
    buildGaugeRow({
      key: m.key,
      label: metricLabel(m.key, t),
      shares: m,
      teamParityPct: usage.team_parity_pct,
      lobbyParityPct: usage.lobby_parity_pct,
      teamOfLobbyParityPct: teamOfLobby,
      rangeMinPct: m.player_share_of_team_min_pct,
      rangeMaxPct: m.player_share_of_team_max_pct,
      t,
      locale,
    }),
  )
}

/** La légende de comptage d'une bande : matchs au-dessus de chaque parité. */
function bandAboveCaption(m: SessionUsageMetric, measured: number, t: UsageText): string {
  const team = m.matches_above_team_parity == null ? t.notMeasured : `${m.matches_above_team_parity}/${measured}`
  return t.bandAboveFmt(team, `${m.matches_above_lobby_parity}/${measured}`)
}

/** Bloc 1 — usages d'équipement : cadences, parts, régularité. */
function EquipmentCard({ usage, meLabel, t, locale }: CardProps) {
  const inks = useGridInks()
  const metrics = useMemo(() => equipmentMetrics(usage.metrics), [usage.metrics])
  const squadPlayers = useMemo(() => usage.squad_players ?? [], [usage.squad_players])
  const cadenceGrid = useMemo(
    () => buildCadenceGrid({ metrics, squadPlayers, meLabel, t, locale, ...inks }),
    [metrics, squadPlayers, meLabel, t, locale, inks],
  )
  const gaugeRows = useMemo(() => metricGaugeRows(metrics, usage, t, locale), [metrics, usage, t, locale])
  if (metrics.length === 0) return null

  return (
    <SectionCard
      title={t.blockEquipment}
      label={t.blockEquipment}
      titleAdornment={measuredAdornment(t.measuredFmt(usage.matches_measured, usage.matches_total))}
      footer={
        <div className="space-y-1 border-t border-border px-3 pb-2 pt-2 text-[11px] text-muted-foreground">
          <p>{t.corollaryEquipment}</p>
        </div>
      }
    >
      <div className="space-y-5 px-3 pb-3 pt-3">
        {cadenceGrid && (
          <section aria-label={t.viewCadences}>
            <ViewTitle>{t.viewCadences}</ViewTitle>
            <ValueGrid model={cadenceGrid} />
          </section>
        )}
        <section aria-label={t.viewShares}>
          <ViewTitle>{t.viewShares}</ViewTitle>
          <UsageGaugeGrid rows={gaugeRows} t={t} />
        </section>
        <section aria-label={t.viewRegularity}>
          <ViewTitle>{t.viewRegularity}</ViewTitle>
          <div className="space-y-1.5">
            {metrics.map((m) => (
              <UsageRegularityBand
                key={m.key}
                label={metricLabel(m.key, t)}
                cells={buildRegularityBand(m.per_match, usage.team_parity_pct, t, locale)}
                caption={bandAboveCaption(m, usage.matches_measured, t)}
              />
            ))}
          </div>
          <p className="pt-1.5 text-[11px] text-muted-foreground">{t.bandCaption}</p>
        </section>
      </div>
    </SectionCard>
  )
}

/** Bloc 2 — contrôle des armes spéciales : socles nommés, familles, bonus anonymes. */
function PadControlCard({ usage, meLabel, t, locale }: CardProps) {
  const pad = useMemo(() => padMetric(usage.metrics), [usage.metrics])
  const squadPlayers = useMemo(() => usage.squad_players ?? [], [usage.squad_players])
  const gaugeRows = useMemo(() => {
    const rows = pad ? metricGaugeRows([pad], usage, t, locale) : []
    const teamOfLobby = teamOfLobbyParityPct(usage.team_size_avg, usage.lobby_size_avg)
    for (const fam of usage.pad_families ?? []) {
      rows.push(
        buildGaugeRow({
          key: `family-${fam.family_key}`,
          label: t.padFamilyFmt(fam.family_key),
          shares: fam,
          teamParityPct: usage.team_parity_pct,
          lobbyParityPct: usage.lobby_parity_pct,
          teamOfLobbyParityPct: teamOfLobby,
          t,
          locale,
        }),
      )
    }
    return rows
  }, [pad, usage, t, locale])
  const track = useMemo(
    () =>
      pad
        ? buildLobbyTrack({ meLabel, shares: pad, squadPlayers, squadShares: pad.squad, t, locale })
        : null,
    [pad, meLabel, squadPlayers, t, locale],
  )
  const powerups = usage.powerup_pickups ?? []
  if (pad == null && gaugeRows.length === 0 && powerups.length === 0) return null

  return (
    <SectionCard
      title={t.blockPadControl}
      label={t.blockPadControl}
      titleAdornment={measuredAdornment(t.measuredFmt(usage.matches_measured, usage.matches_total))}
      footer={<PadControlFootnotes usage={usage} pad={pad} t={t} locale={locale} />}
    >
      <div className="space-y-5 px-3 pb-3 pt-3">
        {gaugeRows.length > 0 && (
          <section aria-label={t.viewShares}>
            <ViewTitle>{t.viewShares}</ViewTitle>
            <UsageGaugeGrid rows={gaugeRows} t={t} />
          </section>
        )}
        {track && (
          <section aria-label={t.viewLobbyTrack}>
            <ViewTitle>{t.viewLobbyTrack}</ViewTitle>
            <UsageLobbyTrack segments={track} label={t.viewLobbyTrack} />
          </section>
        )}
        {pad?.per_match != null && pad.per_match.length > 0 && (
          <section aria-label={t.viewRegularity}>
            <ViewTitle>{t.viewRegularity}</ViewTitle>
            <UsageRegularityBand
              label={metricLabel(pad.key, t)}
              cells={buildRegularityBand(pad.per_match, usage.team_parity_pct, t, locale)}
              caption={bandAboveCaption(pad, usage.matches_measured, t)}
            />
            <p className="pt-1.5 text-[11px] text-muted-foreground">{t.bandCaption}</p>
          </section>
        )}
      </div>
    </SectionCard>
  )
}

/** Le pied du bloc 2 : anonymes, bonus, cadences — les dénominateurs d'honnêteté. */
function PadControlFootnotes({
  usage,
  pad,
  t,
  locale,
}: {
  usage: SessionUsageBlock
  pad: SessionUsageMetric | null
  t: UsageText
  locale: Locale
}) {
  const powerups = usage.powerup_pickups ?? []
  const detail = powerups
    .map((p) =>
      t.powerupDetailFmt(
        powerupLabel(p.family_key, t),
        p.occupations,
        p.per_10min == null ? null : formatUsageRate(p.per_10min, locale),
      ),
    )
    .join(' · ')
  return (
    <div className="space-y-1 border-t border-border px-3 pb-2 pt-2 text-[11px] text-muted-foreground">
      {(usage.pad_unnamed_total ?? 0) > 0 && (
        <p className="text-foreground">{t.padUnnamedFmt(usage.pad_unnamed_total ?? 0)}</p>
      )}
      {powerups.length > 0 && (
        <p className="text-foreground">{`${t.powerupLine} : ${detail}`}</p>
      )}
      {pad != null && (
        <p>
          {t.padCadenceFmt(
            formatUsageRate(pad.player_per_10min, locale),
            formatUsageRate(pad.team_per_10min, locale),
            formatUsageRate(pad.lobby_per_10min, locale),
          )}
        </p>
      )}
      <p>{t.corollaryPadControl}</p>
    </div>
  )
}

/** Bloc 3 — objectifs par rôle et famille (scope indépendant des films). */
function ObjectivesCard({ usage, meLabel, t, locale }: CardProps) {
  const inks = useGridInks()
  const obj = usage.objectives
  const roles = useMemo(() => sortRoles(obj?.roles), [obj])
  const squadPlayers = useMemo(() => usage.squad_players ?? [], [usage.squad_players])
  const roleInks = useMemo(
    () => ({ ...inks, columnColor: (key: string) => tokenCssVar(roleToken(key)) }),
    [inks],
  )
  const gaugeRows = useMemo(() => {
    if (obj == null) return []
    const teamOfLobby = teamOfLobbyParityPct(obj.team_size_avg, obj.lobby_size_avg)
    return roles.map((r) =>
      buildGaugeRow({
        key: r.role,
        label: roleLabel(r.role, t),
        shares: r,
        isDuration: r.is_duration === true,
        teamParityPct: obj.team_parity_pct,
        lobbyParityPct: obj.lobby_parity_pct,
        teamOfLobbyParityPct: teamOfLobby,
        t,
        locale,
      }),
    )
  }, [obj, roles, t, locale])
  const familyGrid = useMemo(
    () => buildObjectiveFamilyGrid({ families: obj?.families ?? [], t, locale, ...roleInks }),
    [obj, t, locale, roleInks],
  )
  const squadGrid = useMemo(
    () => buildSquadRoleGrid({ roles, squadPlayers, meLabel, t, locale, ...roleInks }),
    [roles, squadPlayers, meLabel, t, locale, roleInks],
  )
  if (obj == null || obj.matches_with_objectives <= 0 || roles.length === 0) return null
  const hasDuration = roles.some((r) => r.is_duration === true)

  return (
    <SectionCard
      title={t.blockObjectives}
      label={t.blockObjectives}
      titleAdornment={measuredAdornment(
        t.objectivesScopeFmt(obj.matches_with_objectives, usage.matches_total),
      )}
      footer={
        hasDuration ? (
          <div className="space-y-1 border-t border-border px-3 pb-2 pt-2 text-[11px] text-muted-foreground">
            <p>{t.durationNote}</p>
          </div>
        ) : undefined
      }
    >
      <div className="space-y-5 px-3 pb-3 pt-3">
        <section aria-label={t.viewRoles}>
          <ViewTitle>{t.viewRoles}</ViewTitle>
          <UsageGaugeGrid rows={gaugeRows} t={t} />
        </section>
        {familyGrid && (
          <section aria-label={t.viewFamilies}>
            <ViewTitle>{t.viewFamilies}</ViewTitle>
            <ValueGrid model={familyGrid} />
          </section>
        )}
        {squadGrid && (
          <section aria-label={t.viewSquadRoles}>
            <ViewTitle>{t.viewSquadRoles}</ViewTitle>
            <ValueGrid model={squadGrid} />
          </section>
        )}
      </div>
    </SectionCard>
  )
}
