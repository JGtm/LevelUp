/**
 * SquadFragSection — regroupe les 3 graphes « frags » de l'Escouade.
 *
 * Relocalisé depuis l'onglet Contributions (SquadContributionsPage +
 * SquadPerformanceCharts) vers l'onglet Synergies. Comportement identique,
 * simplement déplacé :
 *   1. Répartition des frags (barres empilées par classe) — ChartCard +
 *      buildFragBreakdownOption.
 *   2. Outils de destruction (armes gun + détail non-arme depuis frag_classes) —
 *      SquadWeaponKillsChart alimenté par buildSquadFragTools.
 *   3. Précision par rôle (native Halo 5, gaté DATA sur weapon_accuracy) —
 *      SquadWeaponAccuracyBarsChart.
 */
import { useCallback, useMemo } from 'react'
import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import { formatMessage } from '@/lib/i18n/format'
import { fragsManifest } from '@/lib/i18n/generated/frags'
import type { FragClassEntry, SquadWeaponAccuracy, SquadWeaponKills } from '@/lib/api/types'
import { buildFragBreakdownOption } from './charts/squadFragBreakdownChart'
import { buildSquadFragTools, SQUAD_TOOLS_TOP_GUNS } from './charts/squadFragTools'
import { SquadWeaponKillsChart } from './SquadWeaponKillsChart'
import { SquadWeaponAccuracyBarsChart } from './SquadWeaponAccuracyBarsChart'
import type { SquadLocale, SquadText } from './i18n'

/** Hauteur de la carte « Répartition des frags » (identique à SquadPerformanceCharts). */
const FRAG_BREAKDOWN_HEIGHT = 280

interface SquadFragSectionProps {
  /** Répartition des frags PAR CLASSE (D8) par gamertag (agrégat serveur). */
  fragClassesByPlayer: Record<string, FragClassEntry[]>
  weaponKills: SquadWeaponKills | null | undefined
  weaponAccuracy: SquadWeaponAccuracy | null | undefined
  /** gamertag → couleur hex (getSquadPlayerColors). */
  playerColors: Record<string, string>
  /** Ordre stable des joueurs (main d'abord, puis coéquipiers). */
  playerOrder: string[]
  locale: SquadLocale
  t: SquadText
}

export function SquadFragSection({
  fragClassesByPlayer,
  weaponKills,
  weaponAccuracy,
  playerColors,
  playerOrder,
  locale,
  t,
}: SquadFragSectionProps) {
  // Série sentinelle pour l'empty-state du ChartCard : non vide dès qu'un joueur
  // a au moins une classe de frags (buildFragBreakdownOption filtre lui-même sur
  // playerOrder). Dérivée de frag_classes (et non de la performance_series comme
  // dans SquadPerformanceCharts) car c'est la source de ce chart en standalone.
  const fragSeries = useMemo<ChartSeries<FragClassEntry>[]>(() => {
    const merged = playerOrder.flatMap((p) => fragClassesByPlayer[p] ?? [])
    return merged.length > 0 ? [{ key: 'frag-flat', datapoints: merged }] : []
  }, [playerOrder, fragClassesByPlayer])

  const buildFragBreakdown = useCallback(
    () =>
      buildFragBreakdownOption(fragClassesByPlayer, {
        playerOrder,
        classLabel: (c: string) => formatMessage(fragsManifest, `frags.class.${c}` as never, locale),
      }),
    [fragClassesByPlayer, playerOrder, locale],
  )

  // « Outils de destruction » = version multi-joueurs de buildFragDetailBreakdown :
  // armes gun (top-N) + détail Assassinat/Corps-à-corps/Coup au sol/Charge spartane/
  // Grenade tiré de frag_classes, SANS « Spartan »/unattributed.
  const weaponTools = useMemo(
    () =>
      buildSquadFragTools(weaponKills, fragClassesByPlayer, {
        roleLabel: (r) => formatMessage(fragsManifest, `frags.role.${r}` as never, locale),
        classLabel: (c) => formatMessage(fragsManifest, `frags.class.${c}` as never, locale),
        otherWeaponsLabel: t.weaponKills.otherWeapons,
        topGuns: SQUAD_TOOLS_TOP_GUNS,
      }),
    [weaponKills, fragClassesByPlayer, locale, t.weaponKills.otherWeapons],
  )

  return (
    <section className="space-y-4">
      {/* Rangée 1 : Répartition des frags | Précision par rôle (précision native H5, gate
          DATA sur weapon_accuracy → sur Infinite la Répartition occupe seule la largeur). */}
      <div className={weaponAccuracy ? 'grid grid-cols-1 gap-4 lg:grid-cols-2' : 'grid grid-cols-1 gap-4'}>
        {/* fluid/fillHeight : les 2 blocs de la rangée s'étirent à la hauteur de la ligne
            (grid align-items:stretch) → hauteurs alignées quelle que soit la donnée. */}
        <ChartCard
          title={t.performanceCharts.fragBreakdownTitle}
          series={fragSeries}
          buildOption={buildFragBreakdown}
          height={FRAG_BREAKDOWN_HEIGHT}
          emptyMessage={t.empty.noBlockData}
          fluid
        />
        {weaponAccuracy && (
          <SquadWeaponAccuracyBarsChart
            title={t.weaponAccuracy.title}
            emptyMessage={t.empty.noBlockData}
            data={weaponAccuracy}
            colorByPlayer={playerColors}
            roleLabel={(r) => formatMessage(fragsManifest, `frags.role.${r}` as never, locale)}
            shotsLabel={t.weaponAccuracy.shotsLabel}
            fillHeight
          />
        )}
      </div>

      {/* Rangée 2 : Outils de destruction, seul sur sa rangée (pleine largeur). */}
      <SquadWeaponKillsChart
        title={t.weaponKills.title}
        emptyMessage={t.empty.noBlockData}
        data={weaponTools}
        colorByPlayer={playerColors}
      />
    </section>
  )
}
