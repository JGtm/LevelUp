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
import { useCallback, useMemo, type ReactNode } from 'react'
import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import { formatMessage } from '@/lib/i18n/format'
import { fragsManifest } from '@/lib/i18n/generated/frags'
import type { FragClassEntry, SquadWeaponAccuracy, SquadWeaponKills } from '@/lib/api/types'
import { buildFragBreakdownOption } from './charts/squadFragBreakdownChart'
import { buildSquadFragTools, SQUAD_TOOLS_TOP_GUNS } from './charts/squadFragTools'
import { SquadWeaponKillsChart } from './SquadWeaponKillsChart'
import { SquadWeaponAccuracyBarsChart } from './SquadWeaponAccuracyBarsChart'
import type { SquadText } from './i18n'
import type { Locale } from '@/lib/i18n/locale'

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
  locale: Locale
  t: SquadText
  /**
   * Slot optionnel monté À GAUCHE de « Répartition des frags » (rangée 1) —
   * carte « Écart cumulé au FDA attendu » sur Infinite (gate `expected_stats`
   * côté parent). Mutuellement exclusif avec `weaponAccuracy` (natif Halo 5) :
   * les titres se distinguent par capability, jamais les deux à la fois.
   */
  leftOfBreakdown?: ReactNode
}

export function SquadFragSection({
  fragClassesByPlayer,
  weaponKills,
  weaponAccuracy,
  playerColors,
  playerOrder,
  locale,
  t,
  leftOfBreakdown,
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

  // Carte « Répartition des frags » définie UNE seule fois (≤ 2 copies), puis
  // placée selon la composition de la rangée 1. fluid : s'étire à la hauteur de
  // la ligne (grid align-items:stretch) → alignée avec la cellule voisine.
  const fragBreakdownCard = (
    <ChartCard
      title={t.performanceCharts.fragBreakdownTitle}
      series={fragSeries}
      buildOption={buildFragBreakdown}
      height={FRAG_BREAKDOWN_HEIGHT}
      emptyMessage={t.empty.noBlockData}
      fluid
    />
  )

  return (
    <section className="space-y-4">
      {/* Rangée 1 — composition mutuellement exclusive :
          - leftOfBreakdown (Écart cumulé FDA, Infinite) → [Écart | Répartition] ;
          - sinon weaponAccuracy (Précision native, Halo 5) → [Répartition | Précision] ;
          - sinon → Répartition pleine largeur.
          Les capabilities `expected_stats` (Infinite) et `weapon_accuracy` (H5)
          sont disjointes : si jamais les deux coexistaient (cas impossible
          aujourd'hui), la branche leftOfBreakdown l'emporte. */}
      {leftOfBreakdown ? (
        <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
          {leftOfBreakdown}
          {fragBreakdownCard}
        </div>
      ) : weaponAccuracy ? (
        <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
          {fragBreakdownCard}
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
        <div className="grid grid-cols-1 gap-4">{fragBreakdownCard}</div>
      )}

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
