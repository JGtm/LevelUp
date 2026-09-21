/**
 * SessionPadControlCards — LES QUATRE CARTES DES ARMES SPÉCIALES de la page Sessions.
 *
 * UNE SEULE CARTE À QUATRE VUES EMPILÉES JUSQU'AU 2026-09-21 (D6). « Contrôle des armes
 * spéciales » portait, sous un unique bandeau, les parts par famille, les niveaux d'arme,
 * la piste du lobby et la bande de régularité : quatre questions différentes, une seule
 * aide pour les quatre, et une colonne qui s'allongeait sans fin. Elles deviennent quatre
 * cartes — exactement le découpage que Synthèse et Escouade portent déjà
 * (`EquipmentUsageSection`), avec les mêmes titres :
 *   1. « Contrôle des armes spéciales » — parts par famille, puis leur TOTAL ;
 *   2. « Contrôle des armes par niveau » — les mêmes prises rangées par niveau ;
 *   3. « Qui ramasse les armes spéciales » — la piste du lobby, découpée par joueur ;
 *   4. « Régularité match par match » — une case par match, teintée par l'écart.
 *
 * RANGÉES À DEUX COLONNES EN PLEINE PAGE, EMPILÉES EN COLONNE DIVISÉE (`compact`) : deux
 * demi-colonnes dans une demi-colonne ne se lisent pas.
 *
 * AUCUN BLOC D'UNE RANGÉE NE SE MASQUE (D8) : une carte sans mesure reste affichée et
 * NOMME sa cause — escamotée, elle laisse la rangée bancale et se lit comme un bug.
 *
 * Aucun calcul ici : projections dans `@/features/_shared/usage/`.
 */
import { useMemo } from 'react'

import { SectionCard } from '@/components/ui/section-card'

import { UsageBandLegend } from '@/features/_shared/usage/UsageBandLegend'
import { UsageEmptyNotice } from '@/features/_shared/usage/UsageEmptyNotice'
import { UsageGaugeGrid } from '@/features/_shared/usage/UsageForms'
import { UsageLobbyTrack } from '@/features/_shared/usage/UsageLobbyTrack'
import { UsageRegularityBand } from '@/features/_shared/usage/UsageRegularityBand'
import { buildGaugeRow, type UsageGaugeRowModel } from '@/features/_shared/usage/usageGaugeModel'
import { usageCardTitle } from '@/features/_shared/usage/usageCardTitle'
import { buildLobbyTrack } from '@/features/_shared/usage/usageLobbyTrackModel'
import {
  buildPadTierGaugeRows,
  isCollapsedTierRowKey,
  padTiersNotes,
} from '@/features/_shared/usage/usagePadTiersModel'
import { metricLabel, padMetric } from '@/features/_shared/usage/usageMetricKinds'
import { teamOfLobbyParityPct } from '@/features/_shared/usage/usageParity'
import { buildRegularityBand } from '@/features/_shared/usage/usageRegularityBandModel'

import { bandAboveCaption, metricGaugeRows, type CardProps } from './SessionUsageShared'

/** Le corps d'une carte : centré verticalement quand la voisine de rangée l'étire. */
function CardBody({ children }: { children: React.ReactNode }) {
  return <div className="flex flex-1 flex-col justify-center px-3 pb-3 pt-3">{children}</div>
}

/** Une rangée de deux cartes en pleine page, empilée en colonne divisée. */
function Row({ compact, children }: { compact: boolean; children: React.ReactNode }) {
  return <div className={`grid grid-cols-1 gap-4${compact ? '' : ' lg:grid-cols-2'}`}>{children}</div>
}

export function PadControlCards({ usage, meLabel, t, locale, compact }: CardProps) {
  const pad = useMemo(() => padMetric(usage.metrics), [usage.metrics])
  const squadPlayers = useMemo(() => usage.squad_players ?? [], [usage.squad_players])
  // LES FAMILLES D'ABORD, LE TOTAL ENSUITE (D6). `pad_pickups` est exactement la somme
  // des familles : posé en tête et à leur niveau, il se lisait comme une grandeur de
  // plus. En pied et marqué `isTotal`, il redevient ce qu'il est — un total.
  const gaugeRows = useMemo(() => {
    const teamOfLobby = teamOfLobbyParityPct(usage.team_size_avg, usage.lobby_size_avg)
    const rows = (usage.pad_families ?? []).map((fam) =>
      buildGaugeRow({
        key: `family-${fam.family_key}`,
        // Le NOM vient du serveur (catalogue d'armes du titre, résolu à la requête).
        // Absent = famille hors catalogue : la CLÉ s'affiche, jamais un nom approchant
        // (même règle que le catalogue du rejeu — D7).
        label: fam.family_label || t.padFamilyFmt(fam.family_key),
        shares: fam,
        teamParityPct: usage.team_parity_pct,
        lobbyParityPct: usage.lobby_parity_pct,
        teamOfLobbyParityPct: teamOfLobby,
        t,
        locale,
      }),
    )
    if (pad) {
      rows.push(
        ...metricGaugeRows([pad], usage, t, locale).map((row) => ({ ...row, isTotal: true })),
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
  // LES PARITÉS DU BLOC viennent du bloc lui-même : celles de la carte portent sur le
  // périmètre du résumé d'usage, qui n'est pas le sien (revue du 2026-09-14).
  const tierRows = useMemo(
    () => buildPadTierGaugeRows(usage.pad_tiers, { t, locale }),
    [usage.pad_tiers, t, locale],
  )
  const tierNotes = useMemo(() => padTiersNotes(usage.pad_tiers, t), [usage.pad_tiers, t])
  const bandCells = useMemo(
    () =>
      pad?.per_match != null
        ? buildRegularityBand(pad.per_match, usage.team_parity_pct, t, locale)
        : [],
    [pad, usage.team_parity_pct, t, locale],
  )
  // LES NIVEAUX COMPTENT DANS LA PORTE (revue du 2026-09-14) : ils viennent d'une AUTRE passe,
  // sur d'autres matchs. Sans eux dans cette condition, une session dont seuls les niveaux sont
  // mesurés ne rendait RIEN — une mesure existante avalée par la porte de sa voisine.
  if (pad == null && gaugeRows.length === 0 && tierRows.length === 0) return null

  const visibleTiers: UsageGaugeRowModel[] = []
  const collapsedTiers: UsageGaugeRowModel[] = []
  for (const row of tierRows) (isCollapsedTierRowKey(row.key) ? collapsedTiers : visibleTiers).push(row)

  return (
    <>
      <Row compact={compact}>
        <SectionCard
          title={t.blockPadControl}
          label={t.blockPadControl}
          titleAdornment={usageCardTitle(t.cardHintPadControl)}
        >
          <CardBody>
            {gaugeRows.length > 0 ? (
              <UsageGaugeGrid rows={gaugeRows} t={t} dense={compact} />
            ) : (
              <UsageEmptyNotice reason="no-pads" t={t} />
            )}
          </CardBody>
        </SectionCard>

        {/* LA RANGÉE DES NIVEAUX A SA PROPRE PASSE (revue du 2026-09-14) : sans bloc
            `pad_tiers`, la carte est ABSENTE — « pas encore mesuré » ne se dessine pas comme
            « aucune prise ». Bloc servi mais sans niveau : elle reste, et dit pourquoi (D8). */}
        {usage.pad_tiers != null && (
        <SectionCard
          title={t.blockPadTiers}
          label={t.blockPadTiers}
          titleAdornment={usageCardTitle(t.cardHintPadTiers, ...tierNotes)}
        >
          <CardBody>
            {tierRows.length > 0 ? (
              <UsageGaugeGrid
                rows={visibleTiers}
                collapsedRows={collapsedTiers}
                collapsedLabel={
                  collapsedTiers.length > 0
                    ? t.padTierBaseToggleFmt(collapsedTiers.length)
                    : undefined
                }
                t={t}
                dense={compact}
              />
            ) : (
              <UsageEmptyNotice reason="no-pads" t={t} />
            )}
          </CardBody>
        </SectionCard>
        )}
      </Row>

      <Row compact={compact}>
        <SectionCard title={t.viewLobbyTrack} label={t.viewLobbyTrack}>
          <CardBody>
            {track != null ? (
              <UsageLobbyTrack segments={track} label={t.viewLobbyTrack} t={t} dense={compact} />
            ) : (
              <UsageEmptyNotice reason="no-pads" t={t} />
            )}
          </CardBody>
        </SectionCard>

        <SectionCard
          title={t.viewRegularity}
          label={t.viewRegularity}
          titleAdornment={usageCardTitle(t.cardHintRegularity)}
        >
          <CardBody>
            {pad != null && bandCells.length > 0 ? (
              <>
                <UsageRegularityBand
                  label={metricLabel(pad.key, t)}
                  cells={bandCells}
                  caption={bandAboveCaption(pad, usage.matches_measured, t)}
                  dense={compact}
                />
                <UsageBandLegend t={t} />
              </>
            ) : (
              <UsageEmptyNotice reason="no-film" t={t} />
            )}
          </CardBody>
        </SectionCard>
      </Row>
    </>
  )
}
