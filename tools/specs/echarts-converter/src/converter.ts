import type { ChartSpec, EChartsOption, ThemeDefault } from './types.js';
import { resolveHeight, resolveLegend, resolvePaletteToken, resolveI18nToken } from './loader.js';
import { convertBarDiverging } from './converters/bar-diverging.js';
import { convertStackedBar } from './converters/stacked-bar.js';
import { convertHeatmap } from './converters/heatmap.js';
import { convertGroupedBar } from './converters/grouped-bar.js';
import { convertGauge } from './converters/gauge.js';
import { convertLine } from './converters/line.js';
import { convertTableHtml } from './converters/table-html.js';
import { convertBullet } from './converters/bullet.js';
import { convertPie } from './converters/pie.js';
import { convertRadar } from './converters/radar.js';
import { convertKpiRow, convertCompositeBlock } from './converters/composite-ui.js';
import { convertHistogramTimeseries, convertScatterMatrix } from './converters/histogram.js';

/**
 * Construit le `grid` ECharts à partir de margin (avec override prioritaire).
 */
export function buildGrid(spec: ChartSpec): EChartsOption['grid'] {
  const margin = spec.layout.margin_override ?? spec.layout.margin ?? { l: 40, r: 20, t: 30, b: 60 };
  const yaxisAutomargin = spec.layout.yaxis?.automargin ?? false;
  const xaxisAutomargin = spec.layout.xaxis?.automargin ?? false;
  return {
    left: margin.l,
    right: margin.r,
    top: margin.t,
    bottom: margin.b,
    containLabel: yaxisAutomargin || xaxisAutomargin, // Plotly automargin → ECharts containLabel
  };
}

/**
 * Construit le bloc `legend` ECharts à partir du layout (override prioritaire) + thème.
 */
export function buildLegend(
  spec: ChartSpec,
  theme: ThemeDefault,
): Record<string, unknown> | false {
  if (spec.layout.showlegend === false) return false;
  // Override prioritaire (cas des charts 1, 2 où le caller modifie la légende)
  const legend = spec.layout.legend_override ?? resolveLegend(spec.layout.legend, theme);
  if (!legend) return false;
  // Mapping Plotly → ECharts
  const orientation = legend.orientation === 'h' ? 'horizontal' : 'vertical';
  const result: Record<string, unknown> = { orient: orientation };
  // Plotly y/yanchor → ECharts top/bottom
  if (typeof legend.y === 'number') {
    if (legend.y < 0) {
      // Sous le grid : ECharts bottom (positif)
      result.bottom = 5;
    } else if (legend.y > 1) {
      // Au-dessus : ECharts top
      result.top = 5;
    }
  }
  // Plotly x/xanchor → ECharts left/right
  if (legend.xanchor === 'center') {
    result.left = 'center';
  } else if (legend.xanchor === 'right' || legend.x === 1) {
    result.right = 0;
  } else {
    result.left = 0;
  }
  return result;
}

/**
 * Convertit la config d'axe Plotly en config xAxis/yAxis ECharts (héritage du thème).
 */
export function buildAxis(
  axis: ChartSpec['layout']['xaxis'] | ChartSpec['layout']['yaxis'],
  theme: ThemeDefault,
  options?: { categories?: unknown[]; isYAxis?: boolean },
): Record<string, unknown> {
  if (!axis) return {};
  const result: Record<string, unknown> = {};

  // Type
  if (axis.type) {
    if (axis.type === 'category') result.type = 'category';
    else if (axis.type === 'date') result.type = 'time';
    else if (axis.type === 'log') result.type = 'log';
    else result.type = 'value';
  }

  // Title
  if (axis.title !== null && axis.title !== undefined && axis.title !== '') {
    result.name = resolveI18nToken(axis.title);
  }

  // Range
  if (axis.range && Array.isArray(axis.range)) {
    result.min = axis.range[0];
    result.max = axis.range[1];
  }

  // autorange:'reversed' → inverse:true
  if (axis.autorange === 'reversed') {
    result.inverse = true;
  }

  // showticklabels=false → axisLabel.show=false
  if (axis.showticklabels === false) {
    result.axisLabel = { show: false };
  }

  // showgrid → splitLine
  if (axis.showgrid === false) {
    result.splitLine = { show: false };
  } else if (theme.axes_default.showgrid === false) {
    result.splitLine = { show: false };
  } else {
    result.splitLine = {
      show: true,
      lineStyle: { color: theme.axes_default.gridcolor },
    };
  }

  // tickangle
  if (typeof axis.tickangle === 'number' && axis.tickangle !== 0) {
    result.axisLabel = {
      ...((result.axisLabel as Record<string, unknown>) || {}),
      rotate: axis.tickangle,
    };
  }

  // tickformat (pour Y axes en %)
  if (axis.tickformat === '.0%') {
    result.axisLabel = {
      ...((result.axisLabel as Record<string, unknown>) || {}),
      formatter: '{value}%',
    };
  }

  // Catégories à fournir directement à l'axe catégoriel
  if (options?.categories && result.type === 'category') {
    result.data = options.categories;
  }

  return result;
}

/**
 * Construit le tooltip ECharts par défaut. Les hovertemplates par série
 * sont gérés au niveau série via `tooltip.formatter`.
 */
export function buildTooltip(spec: ChartSpec, theme: ThemeDefault): Record<string, unknown> {
  return {
    trigger: spec.chart_kind === 'heatmap' ? 'item' : 'axis',
    backgroundColor: theme.hoverlabel.bgcolor,
    borderColor: theme.hoverlabel.bordercolor,
    textStyle: { color: theme.font.color },
  };
}

/**
 * Point d'entrée principal — dispatche selon `chart_kind`.
 */
export function specToEChartsOption(
  spec: ChartSpec,
  theme: ThemeDefault,
  mockContext: Record<string, unknown> = {},
): EChartsOption {
  const warnings: string[] = [];
  const ctx = mockContext as Record<string, unknown>;
  const heightContext: Record<string, number> = {
    metrics: typeof ctx.metrics_count === 'number' ? ctx.metrics_count : 6, // duel chart : 6 métriques
    pivot_height: typeof ctx.pivot_height === 'number' ? ctx.pivot_height : 12,
  };
  // Les tables HTML n'ont pas de hauteur Plotly — passer 0 sentinel
  const height =
    spec.chart_kind === 'table_html' || !spec.layout?.height
      ? 0
      : resolveHeight(spec.layout.height, heightContext, theme, warnings);

  let option: EChartsOption;
  switch (spec.chart_kind) {
    case 'bar_diverging':
      option = convertBarDiverging(spec, theme, mockContext, warnings);
      break;
    case 'stacked_bar':
      option = convertStackedBar(spec, theme, mockContext, warnings);
      break;
    case 'heatmap':
      option = convertHeatmap(spec, theme, mockContext, warnings);
      break;
    case 'grouped_bar':
      option = convertGroupedBar(spec, theme, mockContext, warnings);
      break;
    case 'gauge':
      option = convertGauge(spec, theme, mockContext, warnings);
      break;
    case 'line':
      option = convertLine(spec, theme, mockContext, warnings);
      break;
    case 'table_html':
      option = convertTableHtml(spec, theme, mockContext, warnings);
      break;
    case 'pie':
      option = convertPie(spec, theme, mockContext, warnings);
      break;
    case 'radar':
      option = convertRadar(spec, theme, mockContext, warnings);
      break;
    case 'bullet':
      option = convertBullet(spec, theme, mockContext, warnings);
      break;
    case 'kpi_row':
      option = convertKpiRow(spec, theme, mockContext, warnings);
      break;
    case 'composite_block':
      option = convertCompositeBlock(spec, theme, mockContext, warnings);
      break;
    case 'histogram':
      option = convertHistogramTimeseries(spec, theme);
      break;
    case 'scatter_matrix':
      option = convertScatterMatrix(spec, theme);
      break;
    default:
      warnings.push(`chart_kind="${spec.chart_kind}" non implémenté — option vide`);
      option = { series: [] };
  }

  // Métadonnées injectées (pour le développeur, pas consommées par ECharts)
  option.__meta = {
    spec_id: spec.id,
    chart_kind: spec.chart_kind,
    source_function: spec.source_function,
    warnings,
    height,
  };

  return option;
}

/**
 * Helper partagé : couleurs résolues pour le thème global.
 */
export function applyThemeBase(
  option: EChartsOption,
  theme: ThemeDefault,
): EChartsOption {
  option.backgroundColor = theme.bgcolor.plot;
  option.textStyle = { color: theme.font.color, fontSize: theme.font.size };
  return option;
}

export { resolvePaletteToken, resolveI18nToken };
