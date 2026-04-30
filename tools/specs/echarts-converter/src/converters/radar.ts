import type { ChartSpec, EChartsOption, ThemeDefault } from '../types.js';
import { applyThemeBase, resolvePaletteToken, resolveI18nToken } from '../converter.js';

/**
 * Convertit un chart `radar` (Plotly Scatterpolar) en ECharts radar.
 *
 * Mapping :
 * - radar.axes[] → radar.indicator[{name, max=1.1}]
 * - radial_axis.tickvals/ticktext → radar.splitNumber + axisLabel.formatter
 * - polar_bgcolor → backgroundColor du composant radar
 * - traces[].fill='toself' → series.areaStyle
 * - customdata par axe → tooltip.formatter custom
 */
export function convertRadar(
  spec: ChartSpec,
  theme: ThemeDefault,
  mockCtx: Record<string, unknown>,
  warnings: string[],
): EChartsOption {
  // session_compare.07 : 3 axes F/D, Win%, Accuracy avec hist optional
  if (spec.id === 'session_compare.07') {
    return convertSessionCompareRadar(spec, theme);
  }

  // @ts-expect-error - section radar custom du YAML
  const radarSpec = spec.radar as Record<string, unknown> | undefined;
  if (!radarSpec) {
    warnings.push('chart_kind=radar mais section `radar:` absente du YAML');
    return { series: [] };
  }

  // Récupérer les axes
  const axes =
    ((radarSpec.axes as Array<Record<string, unknown>>) ?? []).map((a) => ({
      name: resolveI18nToken(a.label as string) ?? (a.id as string),
      id: a.id as string,
    }));
  if (axes.length === 0) {
    warnings.push('Section radar.axes vide ou absente');
    return { series: [] };
  }

  // Mock data : 6 valeurs normalisées (0-1) pour les 6 axes (cas du participation_radar)
  const mockNormValues = (mockCtx.values as number[]) ?? [0.72, 0.85, 0.45, 0.68, 0.55, 0.62];
  const mockRawValues = (mockCtx.raw_values as string[]) ?? [
    '1,420 pts',
    '2,100 pts',
    '880 pts',
    '4,400 pts',
    '12.5 pts/min',
    '62% de chances de survie',
  ];

  // Construire les indicators (1 par axe)
  const indicator = axes.map((a) => ({
    name: a.name,
    max: 1.0,
  }));

  // Résoudre la couleur de la trace
  const trace = spec.traces?.[0];
  const traceColor = (trace?.color as string) ?? '#636EFA'; // bleu Plotly default
  const resolvedColor = resolvePaletteToken(traceColor, theme) ?? traceColor;

  // Tooltip formatter : utilise customdata par axe
  const rawValuesJson = JSON.stringify(mockRawValues);
  const axisNamesJson = JSON.stringify(axes.map((a) => a.name));
  const tooltipFormatter = new Function(
    'params',
    `
    var rawValues = ${rawValuesJson};
    var axisNames = ${axisNamesJson};
    var lines = ['<b>' + params.name + '</b>'];
    if (Array.isArray(params.value)) {
      params.value.forEach(function(v, i) {
        lines.push(axisNames[i] + ': ' + (rawValues[i] || (Math.round(v * 100) + '%')));
      });
    }
    return lines.join('<br>');
    `,
  );

  // Series radar
  const series: Array<Record<string, unknown>> = [
    {
      type: 'radar',
      data: [
        {
          name: resolveI18nToken(trace?.name) ?? 'Profil',
          value: mockNormValues,
          areaStyle: { color: hexToRgba(resolvedColor, 0.5) },
          lineStyle: { color: resolvedColor, width: 2 },
          itemStyle: { color: resolvedColor },
          symbolSize: 4,
        },
      ],
      tooltip: {
        formatter: tooltipFormatter,
      },
    },
  ];

  // Configuration du composant radar
  const radial = (radarSpec.radial_axis as Record<string, unknown>) ?? {};
  const splitNumber =
    Array.isArray(radial.tickvals) ? (radial.tickvals as unknown[]).length : 4;

  const polarBg = (radarSpec.polar_bgcolor as string) ?? theme.bgcolor.plot;

  const option: EChartsOption = {
    grid: undefined as unknown as EChartsOption['grid'], // pas de grid pour les radars
    tooltip: {
      trigger: 'item',
      backgroundColor: theme.hoverlabel.bgcolor,
      borderColor: theme.hoverlabel.bordercolor,
      textStyle: { color: theme.font.color },
    },
    legend:
      (spec.layout?.showlegend ?? false) === false
        ? false
        : { show: true, bottom: 0, textStyle: { color: theme.font.color } },
    series,
  };

  // ECharts radar component
  (option as unknown as Record<string, unknown>).radar = {
    indicator,
    center: ['50%', '52%'],
    radius: '68%',
    splitNumber,
    shape: 'polygon',
    axisName: {
      color: 'rgba(245, 248, 255, 0.92)',
      fontSize: 12,
    },
    splitArea: {
      areaStyle: {
        color: ['transparent'], // pas de bandes alternées
      },
    },
    splitLine: {
      lineStyle: {
        color: 'rgba(255, 255, 255, 0.12)',
      },
    },
    axisLine: {
      lineStyle: {
        color: 'rgba(255, 255, 255, 0.12)',
      },
    },
    axisLabel: {
      show: true,
      color: 'rgba(255, 255, 255, 0.85)',
      fontSize: 10,
      fontWeight: 'bold',
      formatter: new Function('value', 'return Math.round(value * 100) + "%";'),
    },
  };

  applyThemeBase(option, theme);
  // Override : background polar spécifique
  option.backgroundColor = polarBg;

  return option;
}

/**
 * Convertit "#RRGGBB" en "rgba(R, G, B, alpha)".
 */
function hexToRgba(hex: string, alpha: number): string {
  if (!hex.startsWith('#') || (hex.length !== 7 && hex.length !== 4)) {
    return `rgba(99, 110, 250, ${alpha})`; // fallback bleu Plotly
  }
  let h = hex.slice(1);
  if (h.length === 3) {
    h = h
      .split('')
      .map((c) => c + c)
      .join('');
  }
  const r = parseInt(h.slice(0, 2), 16);
  const g = parseInt(h.slice(2, 4), 16);
  const b = parseInt(h.slice(4, 6), 16);
  return `rgba(${r}, ${g}, ${b}, ${alpha})`;
}

// =============================================================================
// session_compare.07 — Radar 3 axes A/B/hist (F/D, Win%, Accuracy)
// =============================================================================

function convertSessionCompareRadar(spec: ChartSpec, theme: ThemeDefault): EChartsOption {
  const indicators = [
    { name: 'F/D', max: 100 },
    { name: 'Win %', max: 100 },
    { name: 'Précision', max: 100 },
  ];

  // Mock : Session A bonne, Session B moyenne, hist faible
  const valuesA = [70, 65, 55]; // F/D 1.4 → 70, WR 65%, acc 55%
  const valuesB = [50, 50, 48]; // F/D 1.0 → 50, WR 50%, acc 48%
  const valuesHist = [55, 55, 50]; // moyenne historique (n=8)

  const option: EChartsOption = {
    backgroundColor: 'transparent',
    textStyle: { color: '#E0E0E0', fontSize: theme.font.size },
    tooltip: { trigger: 'item' },
    legend: {
      bottom: 8,
      textStyle: { color: '#E0E0E0' },
      data: ['Moyenne historique (n=8)', 'Session A', 'Session B'],
    },
    // @ts-expect-error - radar component
    radar: {
      indicator: indicators,
      shape: 'polygon',
      splitNumber: 5,
      center: ['50%', '52%'],
      radius: '65%',
      axisName: { color: '#E0E0E0', fontSize: 11 },
      splitLine: { lineStyle: { color: 'rgba(255,255,255,0.2)' } },
      splitArea: { areaStyle: { color: ['rgba(255,255,255,0.02)', 'rgba(255,255,255,0.05)'] } },
      axisLine: { lineStyle: { color: 'rgba(255,255,255,0.3)' } },
    },
    series: [
      {
        type: 'radar',
        data: [
          {
            name: 'Moyenne historique (n=8)',
            value: valuesHist,
            symbol: 'none',
            lineStyle: { color: '#9B59B6', type: 'dotted', width: 2 },
            areaStyle: { color: 'rgba(155, 89, 182, 0.2)' },
          },
          {
            name: 'Session A',
            value: valuesA,
            symbol: 'circle',
            symbolSize: 6,
            lineStyle: { color: '#E74C3C', width: 2 },
            areaStyle: { color: 'rgba(231, 76, 60, 0.3)' },
          },
          {
            name: 'Session B',
            value: valuesB,
            symbol: 'circle',
            symbolSize: 6,
            lineStyle: { color: '#3498DB', width: 2 },
            areaStyle: { color: 'rgba(52, 152, 219, 0.3)' },
          },
        ],
      },
    ],
  };
  // @ts-expect-error - height meta
  option.__height = 400;
  return applyThemeBase(option, theme);
}
