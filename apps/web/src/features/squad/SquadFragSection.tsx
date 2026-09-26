/**
 * SquadFragSection — regroupe les 3 graphes « frags » de l'Escouade.
 *
 * Relocalisé depuis l'onglet Contributions (SquadContributionsPage +
 * SquadPerformanceCharts) vers Synergies, puis vers l'onglet Usages
 * (SquadUsagesPage, lot 3 « sections » du 2026-09-22). Les trois graphes :
 *   1. Répartition des frags (barres empilées par classe, rendu DOM) —
 *      SquadFragBreakdownCard : comptes dans les segments, repli au-dessus de la barre,
 *      total au bout (lot L2 du plan PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26).
 *   2. Outils de destruction (chaque frag nommé, D8) — SquadWeaponKillsChart alimenté par
 *      buildSquadToolRows sur `weapon_tools` : légende des joueurs, compte au bout de chaque
 *      barre, pastille de classe devant l'outil, part du joueur en infobulle.
 *   3. Précision par rôle (native Halo 5, gaté DATA sur weapon_accuracy) —
 *      SquadWeaponAccuracyBarsChart.
 */
import { useCallback, useMemo } from 'react'
import { ChartLegend } from '@/components/charts/ChartLegend'
import { InfoTooltip } from '@/components/ui/info-tooltip'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest } from '@/lib/i18n/generated/common'
import { fragsManifest } from '@/lib/i18n/generated/frags'
import type { FragClassEntry, SquadWeaponAccuracy, SquadWeaponTools } from '@/lib/api/types'
import { buildSquadToolRows, type SquadToolKindLabels } from './charts/squadFragTools'
import { SquadFragBreakdownCard } from './SquadFragBreakdownCard'
import { SquadWeaponKillsChart } from './SquadWeaponKillsChart'
import { SquadWeaponAccuracyBarsChart } from './SquadWeaponAccuracyBarsChart'
import type { SquadText } from './i18n'
import type { Locale } from '@/lib/i18n/locale'

interface SquadFragSectionProps {
  /** Répartition des frags PAR CLASSE par gamertag (agrégat serveur). */
  fragClassesByPlayer: Record<string, FragClassEntry[]>
  weaponTools: SquadWeaponTools | null | undefined
  weaponAccuracy: SquadWeaponAccuracy | null | undefined
  /** gamertag → couleur hex (getSquadPlayerColors). */
  playerColors: Record<string, string>
  /** Ordre stable des joueurs (main d'abord, puis coéquipiers). */
  playerOrder: string[]
  locale: Locale
  t: SquadText
}

export function SquadFragSection({
  fragClassesByPlayer,
  weaponTools,
  weaponAccuracy,
  playerColors,
  playerOrder,
  locale,
  t,
}: SquadFragSectionProps) {
  const classLabel = useCallback(
    (c: string) => formatMessage(fragsManifest, `frags.class.${c}` as never, locale),
    [locale],
  )

  // « Outils de destruction » : les natures sans nom de registre sont nommées ici (le
  // serveur n'écrit aucun libellé) — manifeste `frags` pour la mêlée, les mécaniques et le
  // résidu, i18n Escouade pour les deux catégories de source du film.
  const toolRows = useMemo(() => {
    const role = (r: string) => formatMessage(fragsManifest, `frags.role.${r}` as never, locale)
    const labels: SquadToolKindLabels = {
      melee: classLabel('melee'),
      assassination: role('assassination'),
      ground_pound: role('ground_pound'),
      shoulder_bash: role('shoulder_bash'),
      explosive_object: t.weaponKills.explosiveObject,
      environment: t.weaponKills.environment,
      unattributed: classLabel('unattributed'),
    }
    return buildSquadToolRows(weaponTools, { locale, labels })
  }, [weaponTools, locale, classLabel, t.weaponKills.explosiveObject, t.weaponKills.environment])

  const playerLegend = useMemo(
    () => (toolRows?.players ?? []).map((p) => ({ key: p, label: p, color: playerColors[p] ?? '' })),
    [toolRows, playerColors],
  )

  const breakdownCard = (
    <SquadFragBreakdownCard
      fragClassesByPlayer={fragClassesByPlayer}
      playerOrder={playerOrder}
      playerColors={playerColors}
      classLabel={classLabel}
      emptyTitle={formatMessage(commonManifest, 'common.charts.empty_title', locale)}
      t={t}
    />
  )

  return (
    <section className="space-y-4">
      {/* Rangée 1 :
          - weaponAccuracy (Précision native, Halo 5) → [Répartition | Précision] ;
          - sinon → Répartition pleine largeur. */}
      {weaponAccuracy ? (
        <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
          {breakdownCard}
          <SquadWeaponAccuracyBarsChart
            title={t.weaponAccuracy.title}
            emptyMessage={t.empty.noBlockData}
            data={weaponAccuracy}
            colorByPlayer={playerColors}
            roleLabel={(r) => formatMessage(fragsManifest, `frags.role.${r}` as never, locale)}
            shotsLabel={t.weaponAccuracy.shotsLabel}
            fillHeight
          />
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-4">{breakdownCard}</div>
      )}

      {/* Rangée 2 : Outils de destruction, seul sur sa rangée (pleine largeur). */}
      <SquadWeaponKillsChart
        title={
          <span className="flex items-center gap-1.5">
            {t.weaponKills.title}
            <InfoTooltip content={t.weaponKills.info} />
          </span>
        }
        emptyMessage={t.empty.noBlockData}
        data={toolRows}
        colorByPlayer={playerColors}
        valueText={t.weaponKills.killsShare}
        legend={
          playerLegend.length > 0 ? <ChartLegend items={playerLegend} ariaLabel={t.weaponKills.title} /> : undefined
        }
      />
    </section>
  )
}
