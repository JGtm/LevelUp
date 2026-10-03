/**
 * ElevationCard — LA CARTE « DÉNIVELÉ » DE L'ONGLET RÉSUMÉ : le nuage T5 (décision D25).
 *
 * Elle a quitté `WeaponRangeSection.tsx` le 2026-09-22, qui franchissait les 500 lignes du
 * dépôt en l'accueillant. La frontière suit la carte : la portée par arme et ses tuiles
 * restent là-bas, tout ce qui est propre au nuage vit ici.
 *
 * CE COMPOSANT NE CALCULE RIEN : l'option ECharts, les bornes d'axes et les halos vivent
 * dans `_elevationCloudChart.ts` (pur, testé hors rendu) ; les quantiles et le SIGNE du
 * dénivelé viennent de Go. Ici : le chrome de carte, la légende, l'état vide et la couverture.
 */
import { useCallback, useMemo } from 'react'

import { ChartCard } from '@/components/charts/ChartCard'
import { ChartLegend, type ChartLegendItem } from '@/components/charts/ChartLegend'
import { SectionCard } from '@/components/ui/section-card'
import { resolveToken, tokenCssVar, type SemanticToken } from '@/lib/accessibility'
import type { ElevationCloudBlock, TimeseriesMatchRow } from '@/lib/api/types'
import { getEChartsThemeColors } from '@/lib/echarts/themeColors'
import type { ManifestLocale } from '@/lib/i18n/format'

import { buildElevationCloudOption, elevationMatchLabels } from './_elevationCloudChart'
import { titleWithHelp } from './titleWithHelp'
import type { RangeFormats, Translate } from './weaponRangeText'

/**
 * Encres du nuage : les MÊMES que la carte de portée (D25).
 *
 * Le nuage a deux côtés, pas trois classes. Leur donner une rampe de teinte propre (ce que
 * faisaient les barres empilées par arme) aurait fait dire à la couleur « d'en haut / d'en
 * bas » là où la position se lit déjà sur l'axe des ordonnées.
 */
const KILLS_TOKEN: SemanticToken = 'stat-kills'
const DEATHS_TOKEN: SemanticToken = 'stat-deaths'
const MEDIAN_TOKEN: SemanticToken = 'perf-tier-2'

/**
 * Légende du nuage : un frag, une mort, le halo p25-p75, les médianes, la bande à niveau.
 *
 * LE HALO ET LES MÉDIANES ONT LEUR ENTRÉE, et c'est le point : ce sont deux géométries que
 * l'œil voit sans savoir les nommer. La bande grise emprunte le gris des libellés d'axe
 * (`--muted-foreground`), la MÊME encre que son remplissage — elle n'a pas de token
 * d'accessibilité, d'où la variable CSS brute plutôt qu'un `tokenCssVar`.
 */
function elevationLegendItems(t: Translate): ChartLegendItem[] {
  return [
    { key: 'kill', label: t('synthesis.weapon_range.elev_point_kill'), color: tokenCssVar(KILLS_TOKEN) },
    { key: 'death', label: t('synthesis.weapon_range.elev_point_death'), color: tokenCssVar(DEATHS_TOKEN) },
    { key: 'quartiles', label: t('synthesis.weapon_range.elev_quartiles'), color: tokenCssVar(KILLS_TOKEN) },
    { key: 'medians', label: t('synthesis.weapon_range.elev_medians'), color: tokenCssVar(MEDIAN_TOKEN) },
    { key: 'level', label: t('synthesis.weapon_range.elev_level_band'), color: 'var(--muted-foreground)' },
  ]
}


/**
 * ElevationCard — le nuage « distance × dénivelé », sa légende et sa couverture.
 *
 * ÉTAT VIDE NOMMÉ, ET IL DIT SA CAUSE : le nuage ne dépend d'aucun seuil de publication (la
 * dimension arme ne compte pas, D23-b), donc une carte vide veut dire « aucun frag mesuré
 * sur la fenêtre » — un film non décodé, pas un seuil.
 *
 * AUCUNE PHRASE DE LECTEUR (loi D22) : les deux médianes sont écrites SUR le graphe, la
 * couverture en pied, et l'explication tient dans l'infobulle ⓘ du titre.
 */
export function ElevationCard({
  elevation,
  matchRows,
  height,
  locale,
  t,
  f,
}: {
  elevation: ElevationCloudBlock | null | undefined
  matchRows: readonly TimeseriesMatchRow[] | undefined
  height: number
  locale: ManifestLocale
  t: Translate
  f: RangeFormats
}) {
  const points = useMemo(
    () => [...(elevation?.kills ?? []), ...(elevation?.deaths ?? [])],
    [elevation],
  )
  const series = useMemo(() => [{ key: 'elevation-cloud', datapoints: points }], [points])
  const buildOption = useElevationOption(elevation, matchRows, locale, t, f)

  return (
    <SectionCard
      title={t('synthesis.weapon_range.elevation_subtitle')}
      label={t('synthesis.weapon_range.elevation_subtitle')}
      titleAdornment={titleWithHelp(t('synthesis.weapon_range.elevation_subtitle_detail'))}
      footer={<ElevationFooter elevation={elevation} t={t} />}
    >
      {points.length === 0 ? (
        <p className="px-3 pb-1 pt-2.5 text-xs text-muted-foreground">
          {t('synthesis.weapon_range.elevation_empty')}
        </p>
      ) : (
        <ChartCard
          series={series}
          buildOption={buildOption}
          height={height}
          frameless
          legend={
            <ChartLegend
              items={elevationLegendItems(t)}
              ariaLabel={t('synthesis.weapon_range.legend_elevation_label')}
            />
          }
        />
      )}
    </SectionCard>
  )
}

/** La couverture, en pied de carte : ce que le décodeur a su placer sur ce que j'ai fait. */
function ElevationFooter({
  elevation,
  t,
}: {
  elevation: ElevationCloudBlock | null | undefined
  t: Translate
}) {
  if (!elevation) return null
  return (
    <p className="border-t border-border px-3 py-1 text-xs text-muted-foreground">
      {t('synthesis.weapon_range.elev_coverage', {
        mk: elevation.measured_kills,
        tk: elevation.total_kills,
        md: elevation.measured_deaths,
        td: elevation.total_deaths,
      })}
    </p>
  )
}

/**
 * useElevationOption — le constructeur d'option du nuage, résolu au thème courant.
 *
 * Les tokens ne sont résolus qu'ICI, à l'appel : ECharts peint dans un canvas, qui n'accepte
 * pas de variable CSS, et `ChartCard` rappelle la fonction à chaque bascule de thème.
 *
 * LE DÉNIVELÉ N'EST PAS RECALCULÉ : les deux étiquettes de médiane FORMATENT la valeur reçue
 * (`delta_z_p50`), déjà signée du point de vue du joueur côté Go.
 */
function useElevationOption(
  elevation: ElevationCloudBlock | null | undefined,
  matchRows: readonly TimeseriesMatchRow[] | undefined,
  locale: ManifestLocale,
  t: Translate,
  f: RangeFormats,
) {
  return useCallback(() => {
    if (!elevation) return {}
    const mediane = (v: number | undefined, key: 'kills' | 'deaths') =>
      v == null
        ? ''
        : t(
            key === 'kills'
              ? 'synthesis.weapon_range.elev_median_kills'
              : 'synthesis.weapon_range.elev_median_deaths',
            { value: f.signedDistance(v) },
          )
    return buildElevationCloudOption({
      block: elevation,
      tc: getEChartsThemeColors(),
      colors: {
        kills: resolveToken(KILLS_TOKEN),
        deaths: resolveToken(DEATHS_TOKEN),
        card: getEChartsThemeColors().card,
      },
      matchLabels: elevationMatchLabels(matchRows ?? []),
      weaponLabels: elevationWeaponLabels(elevation, locale),
      fmt: { distance: f.distance, signedDistance: f.signedDistance },
      labels: {
        kills: t('synthesis.weapon_range.side_kills'),
        deaths: t('synthesis.weapon_range.side_deaths'),
        medianKills: mediane(elevation.kills_summary.delta_z_p50, 'kills'),
        medianDeaths: mediane(elevation.deaths_summary.delta_z_p50, 'deaths'),
        xAxis: t('synthesis.weapon_range.elev_axis_x'),
        yAxis: t('synthesis.weapon_range.elev_axis_y'),
        levelBand: t('synthesis.weapon_range.elev_level_band'),
      },
    })
  }, [elevation, matchRows, locale, t, f])
}

/**
 * elevationWeaponLabels — clé d'arme -> libellé DANS LA LANGUE DE LA PAGE.
 *
 * Repli sur la clé si la metadata ne connaît pas l'arme : c'est le contrat du bloc (Go
 * n'invente aucun nom). Repli sur le FR si l'EN manque, et inversement — un nom à moitié
 * traduit reste plus lisible qu'un identifiant.
 */
function elevationWeaponLabels(
  elevation: ElevationCloudBlock,
  locale: ManifestLocale,
): Record<string, string> {
  const out: Record<string, string> = {}
  for (const [key, l] of Object.entries(elevation.weapon_labels ?? {})) {
    const nom = locale === 'en' ? l.label_en || l.label : l.label || l.label_en
    if (nom) out[key] = nom
  }
  return out
}
