import type { ChartSpec, EChartsOption, ThemeDefault } from '../types.js';

/**
 * Histogramme avec dispatch par YAML id (timeseries.03/09/11).
 *
 * - timeseries.03 (KDA distribution KDE + rug + médiane)
 * - timeseries.09 (6 sub-charts en grille 2×3)
 * - timeseries.11 (first event distribution — 2 hist superposés)
 */
export function convertHistogramTimeseries(spec: ChartSpec, theme: ThemeDefault): EChartsOption {
  if (spec.id === 'timeseries.09') return convertDistributionsGrid(spec, theme);
  if (spec.id === 'timeseries.11') return convertFirstEventOverlay(spec, theme);

  // Fallback générique (KDA distribution déjà rendu via line, mais au cas où)
  return convertGenericKDE(spec, theme);
}

export function convertScatterMatrix(spec: ChartSpec, theme: ThemeDefault): EChartsOption {
  if (spec.id === 'timeseries.10') return convertCorrelationsScatter(spec, theme);
  return { series: [], textStyle: { color: theme.font.color } };
}

// =============================================================================
// timeseries.09 — Grille 2×3 de 6 histogrammes (mock condensé en 1 chart représentatif)
// =============================================================================

function convertDistributionsGrid(spec: ChartSpec, theme: ThemeDefault): EChartsOption {
  // 6 sub-distributions mockées avec couleurs HALO_COLORS
  const subs = [
    { name: 'Accuracy', color: '#41d6ff', data: gaussianData(150, 45, 12) },
    { name: 'Kills', color: '#00e676', data: gaussianData(150, 14, 5) },
    { name: 'Average life (s)', color: '#ffb300', data: gaussianData(150, 22, 8) },
    { name: 'Performance', color: '#ab47bc', data: gaussianData(150, 58, 14) },
    { name: 'Score / min', color: '#ffb300', data: gaussianData(150, 1.2, 0.4) },
    { name: 'Rolling WR', color: '#00e676', data: gaussianData(150, 0.55, 0.15) },
  ];

  // Multi-grid : 2 lignes × 3 colonnes
  const grids: Array<Record<string, unknown>> = [];
  const xAxes: Array<Record<string, unknown>> = [];
  const yAxes: Array<Record<string, unknown>> = [];
  const series: Array<Record<string, unknown>> = [];

  subs.forEach((s, i) => {
    const row = Math.floor(i / 3);
    const col = i % 3;
    const left = `${5 + col * 33}%`;
    const top = row === 0 ? '8%' : '54%';
    const width = '28%';
    const height = '38%';

    grids.push({ left, top, width, height, containLabel: true });

    // Histogram bins
    const { binEdges, counts } = histogramBins(s.data, 18);

    xAxes.push({
      gridIndex: i,
      type: 'category',
      data: binEdges.slice(0, -1).map((e) => e.toFixed(1)),
      axisLabel: { color: 'rgba(245,248,255,0.7)', fontSize: 8, interval: 2 },
      name: s.name,
      nameLocation: 'middle',
      nameGap: 18,
      nameTextStyle: { color: 'rgba(245,248,255,0.85)', fontSize: 10 },
    });

    yAxes.push({
      gridIndex: i,
      type: 'value',
      axisLabel: { color: 'rgba(245,248,255,0.6)', fontSize: 8 },
      splitLine: { lineStyle: { color: 'rgba(245,248,255,0.05)' } },
    });

    series.push({
      name: s.name,
      type: 'bar',
      xAxisIndex: i,
      yAxisIndex: i,
      data: counts,
      barCategoryGap: '5%',
      itemStyle: { color: s.color, opacity: 0.7 },
    });

    // KDE line superposée (smoothed)
    const kde = smooth(counts, 3);
    series.push({
      name: s.name + ' KDE',
      type: 'line',
      xAxisIndex: i,
      yAxisIndex: i,
      data: kde,
      smooth: true,
      symbol: 'none',
      lineStyle: { color: s.color, width: 2 },
      areaStyle: { color: s.color, opacity: 0.1 },
    });
  });

  const option: EChartsOption = {
    backgroundColor: 'transparent',
    textStyle: { color: theme.font.color, fontSize: theme.font.size },
    legend: false,
    tooltip: { trigger: 'item' },
    // @ts-expect-error - multi-grid
    grid: grids,
    xAxis: xAxes,
    yAxis: yAxes,
    series,
    title: {
      text: 'Distributions — 6 histogrammes (KDA, accuracy, life, perf, score/min, rolling WR)',
      left: 'center',
      top: 0,
      textStyle: { color: theme.font.color, fontSize: 13 },
    },
  };
  // @ts-expect-error - height meta
  option.__height = 560;
  return option;
}

// =============================================================================
// timeseries.10 — 5 scatter plots de corrélation en grille 3×2 (last full-width)
// =============================================================================

function convertCorrelationsScatter(spec: ChartSpec, theme: ThemeDefault): EChartsOption {
  const N = 80; // matchs mock

  // Génère N points avec corrélation + outcome (couleur)
  function genCorrelated(slope: number, intercept: number, noise: number, xRange: [number, number]) {
    return Array.from({ length: N }, () => {
      const x = xRange[0] + Math.random() * (xRange[1] - xRange[0]);
      const y = slope * x + intercept + (Math.random() - 0.5) * noise;
      const outcome = Math.random() > 0.5 ? 'win' : 'loss';
      return { x, y, outcome };
    });
  }

  const subs = [
    { name: 'Life vs Kills', x: 'Life (s)', y: 'Kills', data: genCorrelated(0.5, 5, 8, [10, 40]) },
    { name: 'Accuracy vs FDA', x: 'Accuracy %', y: 'FDA', data: genCorrelated(0.05, -0.5, 1.5, [20, 70]) },
    { name: 'Life vs Deaths', x: 'Life (s)', y: 'Deaths', data: genCorrelated(-0.2, 18, 6, [10, 40]) },
    { name: 'Kills vs Deaths', x: 'Kills', y: 'Deaths', data: genCorrelated(0.3, 8, 5, [5, 30]) },
    { name: 'Team MMR vs Enemy MMR', x: 'Team MMR', y: 'Enemy MMR', data: genCorrelated(0.95, 50, 80, [1200, 1900]) },
  ];

  const grids: Array<Record<string, unknown>> = [];
  const xAxes: Array<Record<string, unknown>> = [];
  const yAxes: Array<Record<string, unknown>> = [];
  const series: Array<Record<string, unknown>> = [];

  subs.forEach((s, i) => {
    const isLast = i === 4;
    let left: string, top: string, width: string, height: string;
    if (isLast) {
      left = '6%';
      top = '70%';
      width = '88%';
      height = '24%';
    } else {
      const row = Math.floor(i / 2);
      const col = i % 2;
      left = `${6 + col * 47}%`;
      top = row === 0 ? '8%' : '38%';
      width = '40%';
      height = '24%';
    }
    grids.push({ left, top, width, height, containLabel: true });

    xAxes.push({
      gridIndex: i,
      type: 'value',
      name: s.x,
      nameLocation: 'middle',
      nameGap: 18,
      nameTextStyle: { color: 'rgba(245,248,255,0.7)', fontSize: 9 },
      axisLabel: { color: 'rgba(245,248,255,0.6)', fontSize: 8 },
      splitLine: { lineStyle: { color: 'rgba(245,248,255,0.05)' } },
    });
    yAxes.push({
      gridIndex: i,
      type: 'value',
      name: s.y,
      nameLocation: 'middle',
      nameGap: 28,
      nameTextStyle: { color: 'rgba(245,248,255,0.7)', fontSize: 9 },
      axisLabel: { color: 'rgba(245,248,255,0.6)', fontSize: 8 },
      splitLine: { lineStyle: { color: 'rgba(245,248,255,0.05)' } },
    });

    // Wins (vert) + Losses (rouge) en 2 series
    const wins = s.data.filter((p) => p.outcome === 'win').map((p) => [p.x, p.y]);
    const losses = s.data.filter((p) => p.outcome === 'loss').map((p) => [p.x, p.y]);
    series.push({
      name: s.name + ' — wins',
      type: 'scatter',
      xAxisIndex: i,
      yAxisIndex: i,
      data: wins,
      itemStyle: { color: '#00e676', opacity: 0.7 },
      symbolSize: 6,
    });
    series.push({
      name: s.name + ' — losses',
      type: 'scatter',
      xAxisIndex: i,
      yAxisIndex: i,
      data: losses,
      itemStyle: { color: '#e53935', opacity: 0.7 },
      symbolSize: 6,
    });

    // Trendline OLS sur tout le set
    const allXs = s.data.map((p) => p.x);
    const allYs = s.data.map((p) => p.y);
    const trend = olsLine(allXs, allYs);
    series.push({
      name: s.name + ' — trend',
      type: 'line',
      xAxisIndex: i,
      yAxisIndex: i,
      data: trend,
      symbol: 'none',
      lineStyle: { color: 'rgba(255,255,255,0.5)', width: 1.5, type: 'dashed' },
    });
  });

  const option: EChartsOption = {
    backgroundColor: 'transparent',
    textStyle: { color: theme.font.color, fontSize: theme.font.size },
    legend: false,
    tooltip: { trigger: 'item' },
    // @ts-expect-error - multi-grid
    grid: grids,
    xAxis: xAxes,
    yAxis: yAxes,
    series,
    title: {
      text: 'Corrélations — 5 scatter plots colorés par outcome (vert/rouge) + trendline OLS',
      left: 'center',
      top: 0,
      textStyle: { color: theme.font.color, fontSize: 13 },
    },
  };
  // @ts-expect-error - height meta
  option.__height = 720;
  return option;
}

// =============================================================================
// timeseries.11 — First event : 2 histogrammes superposés (kills+/deaths-)
// =============================================================================

function convertFirstEventOverlay(spec: ChartSpec, theme: ThemeDefault): EChartsOption {
  const killsSec = gaussianData(120, 38, 14);
  const deathsSec = gaussianData(120, 52, 18);

  const allValues = [...killsSec, ...deathsSec].filter((v) => v >= 0 && v <= 180);
  const min = 0;
  const max = 180;
  const nbins = 20;
  const binWidth = (max - min) / nbins;
  const binLabels = Array.from({ length: nbins }, (_, i) => (min + i * binWidth).toFixed(0) + 's');

  const killsCounts = histogramBinsByEdges(killsSec, min, max, nbins);
  const deathsCounts = histogramBinsByEdges(deathsSec, min, max, nbins);

  const meanKill = mean(killsSec);
  const meanDeath = mean(deathsSec);
  const meanKillIdx = Math.round((meanKill - min) / binWidth);
  const meanDeathIdx = Math.round((meanDeath - min) / binWidth);

  const option: EChartsOption = {
    backgroundColor: 'transparent',
    textStyle: { color: theme.font.color, fontSize: theme.font.size },
    grid: { left: 60, right: 30, top: 60, bottom: 70, containLabel: false },
    legend: { bottom: 10, textStyle: { color: 'rgba(245,248,255,0.85)' } },
    tooltip: { trigger: 'axis', axisPointer: { type: 'shadow' } },
    xAxis: {
      type: 'category',
      data: binLabels,
      axisLabel: { color: 'rgba(245,248,255,0.85)', fontSize: 10, interval: 1 },
      name: 'Temps (secondes)',
      nameLocation: 'middle',
      nameGap: 28,
      nameTextStyle: { color: 'rgba(245,248,255,0.7)' },
    },
    yAxis: {
      type: 'value',
      axisLabel: { color: 'rgba(245,248,255,0.6)' },
      splitLine: { lineStyle: { color: 'rgba(245,248,255,0.05)' } },
    },
    series: [
      {
        name: '1er frag',
        type: 'bar',
        data: killsCounts,
        itemStyle: { color: '#00e676', opacity: 0.7 },
        barGap: '-100%',
        markLine: {
          symbol: 'none',
          silent: true,
          data: [{ xAxis: meanKillIdx }],
          lineStyle: { color: '#00e676', type: 'dashed', width: 2 },
          label: {
            show: true,
            formatter: `Moy. ${meanKill.toFixed(0)}s`,
            position: 'insideEndTop',
            color: '#00e676',
            fontSize: 11,
            fontWeight: 'bold' as const,
          },
        },
      },
      {
        name: '1ère mort',
        type: 'bar',
        data: deathsCounts,
        itemStyle: { color: '#e53935', opacity: 0.6 },
        barGap: '-100%',
        markLine: {
          symbol: 'none',
          silent: true,
          data: [{ xAxis: meanDeathIdx }],
          lineStyle: { color: '#e53935', type: 'dashed', width: 2 },
          label: {
            show: true,
            formatter: `Moy. ${meanDeath.toFixed(0)}s`,
            position: 'insideEndBottom',
            color: '#e53935',
            fontSize: 11,
            fontWeight: 'bold' as const,
          },
        },
      },
    ],
    title: {
      text: 'Premier événement — distribution kills (vert) vs deaths (rouge)',
      left: 'center',
      top: 8,
      textStyle: { color: theme.font.color, fontSize: 13 },
    },
  };
  // @ts-expect-error - height meta
  option.__height = 400;
  // Touche allValues pour éviter l'unused warning (pas critique)
  void allValues;
  return option;
}

// =============================================================================
// Generic KDE fallback (timeseries.03 ne devrait pas tomber ici, mais au cas où)
// =============================================================================

function convertGenericKDE(spec: ChartSpec, theme: ThemeDefault): EChartsOption {
  const data = gaussianData(150, 1.5, 0.6);
  const { binEdges, counts } = histogramBins(data, 25);
  const kde = smooth(counts, 3);

  const option: EChartsOption = {
    backgroundColor: 'transparent',
    textStyle: { color: theme.font.color, fontSize: theme.font.size },
    grid: { left: 50, right: 30, top: 50, bottom: 50, containLabel: false },
    legend: false,
    tooltip: { trigger: 'axis' },
    xAxis: {
      type: 'category',
      data: binEdges.slice(0, -1).map((e) => e.toFixed(2)),
      axisLabel: { color: 'rgba(245,248,255,0.7)', fontSize: 9, interval: 3 },
    },
    yAxis: {
      type: 'value',
      axisLabel: { color: 'rgba(245,248,255,0.7)' },
      splitLine: { lineStyle: { color: 'rgba(245,248,255,0.05)' } },
    },
    series: [
      {
        type: 'line',
        data: kde,
        smooth: true,
        symbol: 'none',
        lineStyle: { color: '#41d6ff', width: 2 },
        areaStyle: { color: 'rgba(53,208,255,0.18)' },
      },
    ],
    title: {
      text: spec.title,
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
// Helpers : génération de données synthétiques
// =============================================================================

function gaussianData(n: number, mean: number, std: number): number[] {
  const result: number[] = [];
  for (let i = 0; i < n; i++) {
    // Box-Muller
    const u1 = Math.random();
    const u2 = Math.random();
    const z = Math.sqrt(-2 * Math.log(u1)) * Math.cos(2 * Math.PI * u2);
    result.push(mean + z * std);
  }
  return result;
}

function histogramBins(values: number[], nbins: number): { binEdges: number[]; counts: number[] } {
  const min = Math.min(...values);
  const max = Math.max(...values);
  const width = (max - min) / nbins;
  const binEdges = Array.from({ length: nbins + 1 }, (_, i) => min + i * width);
  const counts = new Array(nbins).fill(0);
  values.forEach((v) => {
    const idx = Math.min(nbins - 1, Math.floor((v - min) / width));
    counts[idx]++;
  });
  return { binEdges, counts };
}

function histogramBinsByEdges(values: number[], min: number, max: number, nbins: number): number[] {
  const width = (max - min) / nbins;
  const counts = new Array(nbins).fill(0);
  values.forEach((v) => {
    if (v < min || v > max) return;
    const idx = Math.min(nbins - 1, Math.floor((v - min) / width));
    counts[idx]++;
  });
  return counts;
}

function smooth(values: number[], window: number): number[] {
  return values.map((_, i) => {
    const start = Math.max(0, i - Math.floor(window / 2));
    const end = Math.min(values.length, i + Math.ceil(window / 2));
    const slice = values.slice(start, end);
    return slice.reduce((a, b) => a + b, 0) / slice.length;
  });
}

function mean(values: number[]): number {
  return values.reduce((a, b) => a + b, 0) / values.length;
}

function olsLine(xs: number[], ys: number[]): Array<[number, number]> {
  const n = xs.length;
  const xMean = mean(xs);
  const yMean = mean(ys);
  let num = 0;
  let den = 0;
  for (let i = 0; i < n; i++) {
    num += (xs[i] - xMean) * (ys[i] - yMean);
    den += (xs[i] - xMean) ** 2;
  }
  const slope = num / den;
  const intercept = yMean - slope * xMean;
  const xMin = Math.min(...xs);
  const xMax = Math.max(...xs);
  return [
    [xMin, slope * xMin + intercept],
    [xMax, slope * xMax + intercept],
  ];
}
