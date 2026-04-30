import type { ChartSpec, EChartsOption, ThemeDefault } from '../types.js';
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
 * Convertit un chart `stacked_bar` (cas outcomes_by_map et outcomes_by_mode).
 *
 * Caractéristiques :
 * - Bars verticales (catégorie sur X, count sur Y), tickangle 45° sur X
 * - 2 traces toujours (Wins, Losses), 2 traces conditionnelles (Ties, DNF — show_when "sum > 0")
 * - barmode "stack" → series[].stack
 * - customdata sur Wins (win_rate) consommé par hovertemplate
 * - text inside, masqué pour 0 sur Ties/DNF
 */
export function convertStackedBar(
  spec: ChartSpec,
  theme: ThemeDefault,
  mockCtx: Record<string, unknown>,
  warnings: string[],
): EChartsOption {
  // Dispatch précoce pour les YAMLs qui n'ont pas de section `traces:` détaillée
  // (réutilisation d'autres specs). Ils tomberaient sinon dans le iterate-traces
  // ci-dessous avec une `series: []` vide.
  if (spec.id === 'timeseries.07' || spec.id === 'win_loss.02') {
    return convertMapModeBreakdownTwoCol(spec, theme);
  }
  // Win/Loss section 5 : streak chart (réutilise timeseries.06 logic)
  if (spec.id === 'win_loss.05') return convertStreakChartMock(spec, theme);
  // Win/Loss section 1 : outcomes over time (réutilise timeseries.05)
  if (spec.id === 'win_loss.01') return convertOutcomesOverTimeMock(spec, theme);

  // Mock data : 8 catégories (cartes ou modes)
  const categories =
    (mockCtx.categories as string[]) ??
    ['Aquarius', 'Behemoth', 'Bazaar', 'Recharge', 'Live Fire', 'Streets', 'Catalyst', 'Argyle'];
  const wins = (mockCtx.wins as number[]) ?? [12, 8, 15, 6, 11, 9, 4, 7];
  const losses = (mockCtx.losses as number[]) ?? [10, 14, 7, 11, 8, 12, 6, 9];
  const ties = (mockCtx.ties as number[]) ?? [0, 1, 0, 0, 2, 0, 0, 0];
  const lefts = (mockCtx.lefts as number[]) ?? [0, 0, 1, 0, 0, 0, 0, 1];
  const winRates = wins.map((w, i) => w / (w + losses[i] + ties[i] + lefts[i]));

  const dataByTrace: Record<string, number[]> = {
    wins,
    losses,
    ties,
    left: lefts,
  };

  const series: Array<Record<string, unknown>> = [];

  for (const trace of spec.traces) {
    const traceId = trace.id ?? '';
    const traceData = dataByTrace[traceId] ?? [];

    // show_when : "sum(values) > 0" → skip si tous à 0
    if (trace.show_when === 'sum(values) > 0') {
      const sum = traceData.reduce((a, b) => a + b, 0);
      if (sum <= 0) {
        continue; // trace omise
      }
    }

    const color = resolvePaletteToken(trace.color, theme);
    const traceName = resolveI18nToken(trace.name);

    // Construction du data avec eventuellement customdata (cas wins → win_rate)
    const data = traceData.map((value, idx) => {
      const item: Record<string, unknown> = { value };
      if (trace.customdata?.fields?.includes('win_rate')) {
        item.win_rate = winRates[idx];
      }
      return item;
    });

    const seriesItem: Record<string, unknown> = {
      type: 'bar',
      name: traceName,
      stack: 'total',
      data,
      itemStyle: {
        color,
        opacity: trace.opacity ?? 1,
      },
      label: {
        show: true,
        position: trace.text_data?.position ?? 'inside',
        // empty_when "value == 0" → label vide pour 0
        formatter:
          trace.text_data?.empty_when === 'value == 0'
            ? (params: { value: number }) => (params.value > 0 ? String(params.value) : '')
            : (params: { value: number }) => String(params.value),
      },
      tooltip: {
        formatter: buildStackedBarFormatter(trace, traceName ?? 'Series'),
      },
    };

    series.push(seriesItem);
  }

  const xAxis = buildAxis(spec.layout.xaxis, theme, { categories });
  const yAxis = buildAxis(spec.layout.yaxis, theme);

  const option: EChartsOption = {
    grid: buildGrid(spec),
    tooltip: buildTooltip(spec, theme),
    legend: buildLegend(spec, theme),
    xAxis,
    yAxis,
    series,
  };

  // bargap → barCategoryGap (ECharts)
  if (spec.layout.bargap !== null && spec.layout.bargap !== undefined) {
    // Plotly bargap=0.15 (15%) → ECharts barCategoryGap='15%'
    series.forEach((s) => {
      (s as Record<string, unknown>).barCategoryGap = `${(spec.layout.bargap ?? 0) * 100}%`;
    });
  }

  return applyThemeBase(option, theme);
}

/**
 * Formatter de tooltip pour les barres empilées.
 * Construit via `new Function(...)` pour inliner les constantes dans le body —
 * sinon `fn.toString()` côté HTML conserve les variables de closure et provoque ReferenceError.
 */
function buildStackedBarFormatter(
  trace: { id?: string; customdata?: { fields?: string[] } },
  traceName: string,
): Function {
  const hasWinRate = trace.customdata?.fields?.includes('win_rate') ?? false;
  const safeName = JSON.stringify(traceName); // échappe les guillemets
  const winRateBranch = hasWinRate
    ? `
    if (params.data && typeof params.data.win_rate === 'number') {
      var pct = (params.data.win_rate * 100).toFixed(1);
      result += '<br>i18n:viz_t.hover_win_rate: ' + pct + '%';
    }`
    : '';
  const body = `
    var cat = params.name;
    var val = (params.data && params.data.value !== undefined) ? params.data.value : params.value;
    var result = cat + '<br>' + ${safeName} + ': ' + val;${winRateBranch}
    return result;
  `;
  return new Function('params', body);
}

// =============================================================================
// timeseries.07 — Map / Mode breakdown (2 stacked bars en grille côte à côte)
// =============================================================================

function convertMapModeBreakdownTwoCol(spec: ChartSpec, theme: ThemeDefault): EChartsOption {
  const maps = ['Aquarius', 'Recharge', 'Live Fire', 'Streets', 'Bazaar', 'Behemoth', 'Catalyst', 'Argyle'];
  const modes = ['Slayer', 'CTF', 'Strongholds', 'Oddball', 'King of the Hill', 'Land Grab', 'Quickplay', 'BTB'];

  const mapsWins = [12, 9, 7, 6, 11, 4, 8, 5];
  const mapsLosses = [8, 11, 6, 9, 7, 12, 5, 8];
  const mapsTies = [1, 0, 0, 1, 0, 0, 0, 0];

  const modesWins = [15, 8, 6, 7, 5, 3, 12, 9];
  const modesLosses = [10, 9, 8, 6, 7, 4, 9, 11];
  const modesTies = [0, 1, 0, 0, 0, 0, 1, 0];

  const option: EChartsOption = {
    backgroundColor: 'transparent',
    textStyle: { color: theme.font.color, fontSize: theme.font.size },
    legend: {
      top: 10,
      right: 30,
      orient: 'horizontal',
      textStyle: { color: 'rgba(245,248,255,0.85)' },
      data: ['Wins', 'Losses', 'Ties'],
    },
    tooltip: { trigger: 'axis', axisPointer: { type: 'shadow' } },
    // @ts-expect-error - multi-grid
    grid: [
      { left: '5%', top: '20%', width: '40%', height: '70%', containLabel: true },
      { left: '55%', top: '20%', width: '40%', height: '70%', containLabel: true },
    ],
    xAxis: [
      { gridIndex: 0, type: 'category', data: maps, axisLabel: { color: 'rgba(245,248,255,0.7)', fontSize: 9, rotate: -30 }, name: 'Cartes (top 12)', nameLocation: 'middle', nameGap: 35, nameTextStyle: { color: 'rgba(245,248,255,0.85)' } },
      { gridIndex: 1, type: 'category', data: modes, axisLabel: { color: 'rgba(245,248,255,0.7)', fontSize: 9, rotate: -30 }, name: 'Modes', nameLocation: 'middle', nameGap: 35, nameTextStyle: { color: 'rgba(245,248,255,0.85)' } },
    ],
    yAxis: [
      { gridIndex: 0, type: 'value', axisLabel: { color: 'rgba(245,248,255,0.7)' }, splitLine: { lineStyle: { color: 'rgba(245,248,255,0.05)' } } },
      { gridIndex: 1, type: 'value', axisLabel: { color: 'rgba(245,248,255,0.7)' }, splitLine: { lineStyle: { color: 'rgba(245,248,255,0.05)' } } },
    ],
    series: [
      { name: 'Wins', type: 'bar', stack: 'maps', xAxisIndex: 0, yAxisIndex: 0, data: mapsWins, itemStyle: { color: '#00e676' } },
      { name: 'Losses', type: 'bar', stack: 'maps', xAxisIndex: 0, yAxisIndex: 0, data: mapsLosses, itemStyle: { color: '#e53935' } },
      { name: 'Ties', type: 'bar', stack: 'maps', xAxisIndex: 0, yAxisIndex: 0, data: mapsTies, itemStyle: { color: '#ab47bc' } },
      { name: 'Wins ', type: 'bar', stack: 'modes', xAxisIndex: 1, yAxisIndex: 1, data: modesWins, itemStyle: { color: '#00e676' } },
      { name: 'Losses ', type: 'bar', stack: 'modes', xAxisIndex: 1, yAxisIndex: 1, data: modesLosses, itemStyle: { color: '#e53935' } },
      { name: 'Ties ', type: 'bar', stack: 'modes', xAxisIndex: 1, yAxisIndex: 1, data: modesTies, itemStyle: { color: '#ab47bc' } },
    ],
    title: {
      text: 'V/D/T par carte (gauche, top 12) / par mode (droite)',
      left: 'center',
      top: 5,
      textStyle: { color: theme.font.color, fontSize: 13 },
    },
  };
  // @ts-expect-error - height meta
  option.__height = 460;
  return option;
}

// =============================================================================
// win_loss.01 — Outcomes over time (V/D par bucket, losses négatives)
// =============================================================================

function convertOutcomesOverTimeMock(spec: ChartSpec, theme: ThemeDefault): EChartsOption {
  const buckets = [
    'Sem 5', 'Sem 6', 'Sem 7', 'Sem 8', 'Sem 9',
    'Sem 10', 'Sem 11', 'Sem 12', 'Sem 13', 'Sem 14',
    'Sem 15', 'Sem 16',
  ];
  const wins = [12, 8, 14, 10, 16, 11, 9, 13, 15, 8, 12, 14];
  const losses = [-8, -11, -7, -10, -6, -9, -12, -8, -7, -11, -9, -7];
  const ties = [1, 0, 1, 0, 0, 1, 0, 0, 1, 0, 0, 1];

  const option: EChartsOption = {
    backgroundColor: 'transparent',
    textStyle: { color: theme.font.color, fontSize: theme.font.size },
    grid: { left: 50, right: 30, top: 50, bottom: 70, containLabel: false },
    legend: { bottom: 8, textStyle: { color: 'rgba(245,248,255,0.85)' } },
    tooltip: { trigger: 'axis', axisPointer: { type: 'shadow' } },
    xAxis: {
      type: 'category',
      data: buckets,
      axisLabel: { color: 'rgba(245,248,255,0.85)', fontSize: 10 },
    },
    yAxis: {
      type: 'value',
      axisLine: { lineStyle: { color: 'rgba(255,255,255,0.75)', width: 2 } },
      axisLabel: {
        color: 'rgba(245,248,255,0.85)',
        formatter: new Function('v', 'return Math.abs(v);') as unknown as string,
      },
      splitLine: { lineStyle: { color: 'rgba(245,248,255,0.05)' } },
    },
    series: [
      { name: 'Victoires', type: 'bar', data: wins, itemStyle: { color: '#00e676' }, barCategoryGap: '30%' },
      { name: 'Défaites', type: 'bar', data: losses, itemStyle: { color: '#e53935' }, barCategoryGap: '30%' },
      { name: 'Égalités', type: 'bar', data: ties, itemStyle: { color: '#ab47bc' }, barCategoryGap: '30%' },
    ],
    title: {
      text: 'Outcomes over time — V/D par semaine (losses négatives)',
      left: 'center',
      top: 8,
      textStyle: { color: theme.font.color, fontSize: 13 },
    },
  };
  // @ts-expect-error - height meta
  option.__height = 380;
  return option;
}

// =============================================================================
// win_loss.05 — Streak chart (séries V/D consécutives)
// =============================================================================

function convertStreakChartMock(spec: ChartSpec, theme: ThemeDefault): EChartsOption {
  // Séquence W/L alternées avec compteur cumulé en streak
  const sequence = ['W', 'W', 'W', 'L', 'L', 'W', 'W', 'L', 'L', 'L', 'L', 'W', 'W', 'W', 'W', 'W', 'L', 'L', 'W'];
  let counter = 0;
  let prev: string | null = null;
  const data: number[] = [];
  const colors: string[] = [];
  sequence.forEach((s) => {
    if (s !== prev) counter = 1;
    else counter++;
    prev = s;
    data.push(s === 'W' ? counter : -counter);
    colors.push(s === 'W' ? '#00e676' : '#e53935');
  });

  const xCats = sequence.map((_, i) => `m${i + 1}`);

  const option: EChartsOption = {
    backgroundColor: 'transparent',
    textStyle: { color: theme.font.color, fontSize: theme.font.size },
    grid: { left: 50, right: 30, top: 30, bottom: 70, containLabel: false },
    legend: false,
    tooltip: {
      trigger: 'item',
      formatter: new Function(
        'p',
        `var v = Math.abs(p.value); return p.name + "<br>Streak " + (p.value > 0 ? "victoire" : "défaite") + " : " + v;`,
      ) as unknown as Record<string, unknown>,
    },
    xAxis: {
      type: 'category',
      data: xCats,
      axisLabel: { color: 'rgba(245,248,255,0.7)', fontSize: 9, interval: 1 },
    },
    yAxis: {
      type: 'value',
      axisLine: { lineStyle: { color: 'rgba(255,255,255,0.75)', width: 2 } },
      axisLabel: {
        color: 'rgba(245,248,255,0.85)',
        formatter: new Function('v', 'return Math.abs(v);') as unknown as string,
      },
      splitLine: { lineStyle: { color: 'rgba(245,248,255,0.05)' } },
    },
    series: [
      {
        name: 'Streak',
        type: 'bar',
        data: data.map((v, i) => ({ value: v, itemStyle: { color: colors[i] } })),
        barCategoryGap: '15%',
      },
    ],
    title: {
      text: 'Streak chart — séries V/D consécutives (compteur cumulé dans la streak)',
      left: 'center',
      top: 5,
      textStyle: { color: theme.font.color, fontSize: 13 },
    },
  };
  // @ts-expect-error - height meta
  option.__height = 240;
  return option;
}
