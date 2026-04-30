import type { ChartSpec, EChartsOption, ThemeDefault, TraceSpec } from '../types.js';
import {
  buildGrid,
  buildLegend,
  buildAxis,
  buildTooltip,
  applyThemeBase,
  resolvePaletteToken,
  resolveI18nToken,
} from '../converter.js';

/**
 * Convertit un chart `bar_diverging` (cas du duel Solo vs Squad).
 *
 * Caractéristiques :
 * - 2 traces horizontales : x négatif à gauche, x positif à droite
 * - x est une valeur normalisée (% de la max de la paire), pas la vraie valeur
 * - text affiche la vraie valeur pré-formatée
 * - vline x=0 → markLine
 * - barmode "overlay" → series sans stack
 */
export function convertBarDiverging(
  spec: ChartSpec,
  theme: ThemeDefault,
  mockCtx: Record<string, unknown>,
  warnings: string[],
): EChartsOption {
  // Mock data : 6 métriques pour le duel
  const labels =
    (mockCtx.labels as string[]) ??
    [
      'Score de performance',
      'Vie moyenne',
      'KPM',
      'Précision',
      'Win Rate',
      'K/D',
    ]; // ordre = reversed(metrics) déjà appliqué côté API
  const soloValues = (mockCtx.solo_values as number[]) ?? [62.4, 145, 0.92, 41.5, 48.0, 1.18];
  const squadValues = (mockCtx.squad_values as number[]) ?? [71.2, 162, 1.05, 44.1, 56.0, 1.42];
  const soloTexts =
    (mockCtx.solo_texts as string[]) ??
    ['62.4', '145s', '0.92', '41.5%', '48.0%', '1.18'];
  const squadTexts =
    (mockCtx.squad_texts as string[]) ??
    ['71.2', '162s', '1.05', '44.1%', '56.0%', '1.42'];

  // Application du data_transform : x = ±(value / scale) * 100, scale = max(solo, squad, 1.0)
  const soloX: number[] = [];
  const squadX: number[] = [];
  for (let i = 0; i < labels.length; i++) {
    const scale = Math.max(soloValues[i] ?? 0, squadValues[i] ?? 0, 1.0);
    soloX.push(-(((soloValues[i] ?? 0) / scale) * 100));
    squadX.push(((squadValues[i] ?? 0) / scale) * 100);
  }

  // Construction des series
  const traces = spec.traces;
  const series: Array<Record<string, unknown>> = [];

  for (let t = 0; t < traces.length; t++) {
    const trace = traces[t];
    const isSolo = trace.id === 'solo' || t === 0;
    const xData = isSolo ? soloX : squadX;
    const textData = isSolo ? soloTexts : squadTexts;
    const color = resolvePaletteToken(trace.color, theme);

    // Position du label : `outside` Plotly horizontal :
    //   - barre négative → label à GAUCHE de la barre
    //   - barre positive → label à DROITE de la barre
    const labelPosition = isSolo ? 'left' : 'right';

    const seriesItem: Record<string, unknown> = {
      type: 'bar',
      name: resolveI18nToken(trace.name),
      data: xData.map((x, idx) => ({
        value: x,
        // Valeur additionnelle dans data pour le tooltip
        text: textData[idx],
        label: labels[idx],
      })),
      itemStyle: {
        color,
        opacity: trace.opacity ?? 1,
      },
      label: {
        show: true,
        position: labelPosition,
        formatter: new Function('params', 'return (params.data && params.data.text) ? params.data.text : "";'),
      },
      // cliponaxis: false → ECharts clip: false (laisser le label déborder)
      clip: trace.clip === false ? false : true,
      // Plotly barmode="overlay" + orientation horizontale = les deux séries
      // partagent la MÊME ligne pour chaque catégorie Y. En ECharts, sans cela,
      // les séries se groupent côte à côte. barGap:'-100%' superpose les barres.
      barGap: '-100%',
      // Hovertemplate Plotly → ECharts tooltip.formatter
      tooltip: {
        formatter: buildHoverFormatter(trace, isSolo),
      },
    };

    // markLine sur x=0 (vline) — à attacher à la 1ère série uniquement (sinon doublon)
    if (t === 0 && spec.layout.shapes && spec.layout.shapes.length > 0) {
      const markLines: Array<Record<string, unknown>> = [];
      for (const shape of spec.layout.shapes) {
        if (shape.kind === 'vline' && typeof shape.x === 'number') {
          markLines.push({
            xAxis: shape.x,
            lineStyle: {
              color: resolvePaletteToken(shape.line_color, theme),
              width: shape.line_width ?? 1,
              opacity: shape.opacity ?? 1,
              type: 'solid',
            },
            symbol: 'none',
            label: { show: false },
          });
        }
      }
      if (markLines.length > 0) {
        seriesItem.markLine = { silent: true, symbol: 'none', data: markLines };
      }
    }

    series.push(seriesItem);
  }

  // Axes
  const xAxis = buildAxis(spec.layout.xaxis, theme);
  const yAxis = buildAxis(spec.layout.yaxis, theme, {
    categories: labels,
  });

  const option: EChartsOption = {
    grid: buildGrid(spec),
    tooltip: buildTooltip(spec, theme),
    legend: buildLegend(spec, theme),
    xAxis,
    yAxis,
    series,
  };

  return applyThemeBase(option, theme);
}

/**
 * Construit un formatter de tooltip pour le duel chart.
 * Construit via `new Function(...)` pour inliner traceName dans le body
 * (sinon fn.toString() perd la closure côté HTML).
 */
function buildHoverFormatter(
  trace: TraceSpec,
  _isSolo: boolean,
): Function {
  const traceName = resolveI18nToken(trace.name) ?? 'Series';
  const safeName = JSON.stringify(traceName);
  const body = `
    var lbl = (params.data && params.data.label) ? params.data.label : params.name;
    var txt = (params.data && params.data.text) ? params.data.text : '';
    return ${safeName} + '<br>' + lbl + ': ' + txt;
  `;
  return new Function('params', body);
}
