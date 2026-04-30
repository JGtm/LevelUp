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
 * Convertit un chart `grouped_bar`. Deux modes :
 *
 * 1. **Mode "stacked + line Y2"** (synthesis.04 — `top_matches_by_week`) :
 *    - 2 traces bar empilées (ids: top, others) + 1 line sur axe Y2 (top_rate_line)
 *    - barmode "stack" sur les barres, line sur axe Y secondaire
 *    - Détection : présence d'au moins 1 trace avec id ∈ {top, others, top_rate_line}.
 *
 * 2. **Mode "generic per-bar coloring"** (match_view.03 expected_vs_actual,
 *    match_view.04 spree_headshots) :
 *    - N traces (chaque trace = 1 catégorie d'analyse : actual / expected / history_avg)
 *    - Chaque trace a `color_per_bar` (couleur DIFFÉRENTE par catégorie X au sein de la trace)
 *    - Chaque trace peut avoir `pattern: {shape, ...}` (hachuré '/' ou dotté '.')
 *    - barmode "group"
 *    - Détection : par défaut quand aucun id legacy n'est trouvé.
 */
export function convertGroupedBar(
  spec: ChartSpec,
  theme: ThemeDefault,
  mockCtx: Record<string, unknown>,
  warnings: string[],
): EChartsOption {
  // Dispatch précoce pour les charts dont la section `traces:` est un template
  // (objet, pas array) — sinon `spec.traces.some(...)` plus bas explose.
  if (spec.id === 'teammates.14') return convertPerMinuteStats(spec, theme);
  if (spec.id === 'teammates.17') return convertFirstEventsButterfly(spec, theme);
  if (spec.id === 'timeseries.28' || spec.id === 'win_loss.04') return convertTopByWeek(spec, theme);
  if (spec.id === 'session_compare.13') return convertParticipationTrend(spec, theme);
  if (spec.id === 'session_compare.11') return convertModesBreakdownSC(spec, theme);
  if (spec.id === 'session_compare.08') return convertSessionCompareBar(spec, theme);

  const legacyIds = new Set(['top', 'others', 'top_rate_line']);
  const isLegacyMode = spec.traces.some((t) => t.id && legacyIds.has(t.id));

  if (isLegacyMode) {
    return convertLegacyTopByWeek(spec, theme, mockCtx, warnings);
  }
  return convertGenericGroupedBar(spec, theme, mockCtx, warnings);
}

// =============================================================================
// MODE 1 — synthesis.04 "top by week"
// =============================================================================
function convertLegacyTopByWeek(
  spec: ChartSpec,
  theme: ThemeDefault,
  mockCtx: Record<string, unknown>,
  warnings: string[],
): EChartsOption {
  const periods =
    (mockCtx.periods as string[]) ??
    [
      '2026-02-02', '2026-02-09', '2026-02-16', '2026-02-23',
      '2026-03-02', '2026-03-09', '2026-03-16', '2026-03-23',
      '2026-03-30', '2026-04-06', '2026-04-13', '2026-04-20',
    ];
  const topCounts = (mockCtx.top_counts as number[]) ?? [3, 5, 2, 4, 6, 3, 4, 5, 7, 3, 4, 5];
  const otherCounts = (mockCtx.other_counts as number[]) ?? [12, 10, 14, 11, 9, 13, 12, 11, 8, 13, 12, 10];
  const topRates = topCounts.map((t, i) => +((t / (t + otherCounts[i])) * 100).toFixed(1));

  const series: Array<Record<string, unknown>> = [];
  for (const trace of spec.traces) {
    const color = resolvePaletteToken(trace.color, theme);
    const traceName = resolveI18nToken(trace.name);
    const safeName = JSON.stringify(traceName ?? '');

    if (trace.id === 'top') {
      series.push({
        type: 'bar', name: traceName, stack: 'total',
        data: topCounts.map((value, idx) => ({ value, top_rate: topRates[idx] })),
        itemStyle: { color, opacity: trace.opacity ?? 1 },
        label: { show: true, position: trace.text_data?.position ?? 'inside',
                 formatter: new Function('params', 'return String(params.value);') },
        tooltip: {
          formatter: new Function('params',
            `var period=params.name;return period+'<br>Top: '+params.data.value+'<br>i18n:viz_t.trace_top_rate: '+params.data.top_rate.toFixed(1)+'%';`),
        },
      });
    } else if (trace.id === 'others') {
      const labelFn = trace.text_data?.empty_when === 'value == 0'
        ? new Function('params', 'return params.value > 0 ? String(params.value) : "";')
        : new Function('params', 'return String(params.value);');
      series.push({
        type: 'bar', name: traceName, stack: 'total',
        data: otherCounts.map((value) => ({ value })),
        itemStyle: { color, opacity: trace.opacity ?? 1 },
        label: { show: true, position: trace.text_data?.position ?? 'inside', formatter: labelFn },
        tooltip: {
          formatter: new Function('params',
            `return params.name + '<br>' + ${safeName} + ': ' + params.value;`),
        },
      });
    } else if (trace.id === 'top_rate_line') {
      series.push({
        type: 'line', name: traceName, data: topRates, yAxisIndex: 1,
        lineStyle: { color, width: trace.line?.width ?? 2 },
        itemStyle: { color },
        symbol: trace.marker?.symbol ?? 'circle',
        symbolSize: trace.marker?.size ?? 6,
        smooth: false,
        tooltip: {
          formatter: new Function('params',
            `return params.name + '<br>' + ${safeName} + ': ' + params.value.toFixed(1) + '%';`),
        },
      });
    } else {
      warnings.push(`Trace id="${trace.id}" non reconnue pour mode legacy — skip`);
    }
  }

  const yAxes: Array<Record<string, unknown>> = [buildAxis(spec.layout.yaxis, theme)];
  if (spec.layout.yaxis2) {
    const y2 = buildAxis(spec.layout.yaxis2, theme);
    if (spec.layout.yaxis2.side === 'right') y2.position = 'right';
    if (spec.layout.yaxis2.showgrid === false) y2.splitLine = { show: false };
    yAxes.push(y2);
  }

  const xAxis = buildAxis(spec.layout.xaxis, theme, { categories: periods });

  const option: EChartsOption = {
    grid: buildGrid(spec),
    tooltip: buildTooltip(spec, theme),
    legend: buildLegend(spec, theme),
    xAxis,
    yAxis: yAxes,
    series,
  };
  return applyThemeBase(option, theme);
}

// =============================================================================
// MODE 2 — match_view.03 / match_view.04 "generic per-bar grouped bars"
// + match_view.10 (tug-of-war stacked) + match_view.11 (cadence stacked + MA)
// =============================================================================
function convertGenericGroupedBar(
  spec: ChartSpec,
  theme: ThemeDefault,
  mockCtx: Record<string, unknown>,
  warnings: string[],
): EChartsOption {
  // Cas spécifiques avec timeline temporelle (mm:ss buckets) et stacked bars
  if (spec.id === 'match_view.10') {
    return convertTugOfWar(spec, theme, mockCtx);
  }
  if (spec.id === 'match_view.11') {
    return convertCadenceHistogram(spec, theme, mockCtx);
  }
  if (spec.id === 'teammates.09') {
    return convertWeaponKillsMulti(spec, theme);
  }
  if (spec.id === 'teammates.13') {
    return convertMapPerfVsHistory(spec, theme);
  }
  if (spec.id === 'teammates.14') {
    return convertPerMinuteStats(spec, theme);
  }
  if (spec.id === 'teammates.17') {
    return convertFirstEventsButterfly(spec, theme);
  }

  // Catégories X selon le chart
  const categories = pickCategories(spec.id, mockCtx);

  // Mock values par trace + catégorie (data réaliste pour rendre les charts visibles)
  const mockMatrix = pickMockMatrix(spec.id, spec.traces.length, categories.length);

  const series: Array<Record<string, unknown>> = [];
  for (let traceIdx = 0; traceIdx < spec.traces.length; traceIdx++) {
    const trace = spec.traces[traceIdx];
    const traceName = resolveI18nToken(trace.name) ?? trace.id ?? `trace_${traceIdx}`;
    const safeName = JSON.stringify(traceName);

    // Résolution couleurs : color_per_bar (1 couleur par catégorie X) > color (1 couleur)
    const traceAsRec = trace as unknown as Record<string, unknown>;
    const colorsPerBar = resolveColorPerBar(traceAsRec, theme, categories.length);

    // Pattern (hachuré '/' ou dotté '.') → simulé via decal en ECharts 5.4+,
    // sinon via opacité réduite + bordure pour distinguer visuellement
    const pattern = (trace as { pattern?: { shape?: string } }).pattern;
    const opacity = trace.opacity ?? 1;

    const traceValues = mockMatrix[traceIdx] ?? new Array(categories.length).fill(0);

    // Construire data avec couleur per-item
    const data = traceValues.map((v, i) => ({
      value: v,
      itemStyle: buildItemStyle(colorsPerBar[i], opacity, pattern, traceAsRec),
    }));

    series.push({
      type: 'bar',
      name: traceName,
      data,
      label: { show: false },
      tooltip: {
        formatter: new Function(
          'params',
          `return params.name + ' (' + ${safeName} + '): ' + params.value;`,
        ),
      },
    });
  }

  const xAxis = buildAxis(spec.layout.xaxis, theme, { categories });
  const yAxis = buildAxis(spec.layout.yaxis, theme);
  // rangemode tozero → yAxis.min: 0
  if ((spec.layout.yaxis as { rangemode?: string })?.rangemode === 'tozero') {
    (yAxis as Record<string, unknown>).min = 0;
  }

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

// === Helpers ===

function pickCategories(chartId: string, mockCtx: Record<string, unknown>): string[] {
  if (mockCtx.categories) return mockCtx.categories as string[];
  if (chartId === 'match_view.03') return ['K', 'D', 'A'];
  if (chartId === 'match_view.04') return ['Killing Spree', 'Headshots', 'Perfect Kills'];
  return ['Cat A', 'Cat B', 'Cat C'];
}

/**
 * Génère une matrice [trace × category] de valeurs mock.
 * Pour match_view.03 et 04, on simule un cas typique :
 *   - actual : valeurs du match courant
 *   - expected : valeurs attendues (≈ actual ± 20%)
 *   - history_avg : valeurs historiques (≈ actual ± 10%)
 */
function pickMockMatrix(chartId: string, nTraces: number, nCats: number): number[][] {
  if (chartId === 'match_view.03') {
    // [actual, expected, history_avg] × [K, D, A]
    return [
      [18, 8, 5],          // actual
      [14, 11, 4],         // expected
      [16, 9, 4.5],        // history_avg
    ].slice(0, nTraces);
  }
  if (chartId === 'match_view.04') {
    // [actual, history_avg] × [Spree, Headshots, Perfect]
    return [
      [9, 6, 2],   // actual
      [6.5, 4.2, 1.1], // history_avg
    ].slice(0, nTraces);
  }
  // Fallback random
  return Array.from({ length: nTraces }, () =>
    Array.from({ length: nCats }, () => Math.round(Math.random() * 20)),
  );
}

function resolveColorPerBar(
  trace: Record<string, unknown>,
  theme: ThemeDefault,
  nCats: number,
): string[] {
  // 1) trace.color_per_bar = liste explicite
  const cpb = (trace as { color_per_bar?: string[] }).color_per_bar;
  if (Array.isArray(cpb) && cpb.length > 0) {
    return cpb.map((c) => resolvePaletteToken(c, theme) ?? c).slice(0, nCats);
  }
  // 2) Sinon : couleur unique pour toutes les barres
  const single = resolvePaletteToken((trace as { color?: string }).color, theme);
  return new Array(nCats).fill(single ?? '#33D6FF');
}

function buildItemStyle(
  color: string,
  opacity: number,
  pattern: { shape?: string } | undefined,
  trace: Record<string, unknown>,
): Record<string, unknown> {
  const style: Record<string, unknown> = { color, opacity };
  // Bordure pour le pattern hachuré '/' (cas "expected" du chart 03)
  const markerLine = (trace as { marker_line?: { width?: number; color_per_bar?: string[] } })
    .marker_line;
  if (markerLine?.width) {
    style.borderWidth = markerLine.width;
    // border color : prend la 1ère du color_per_bar si dispo, sinon défaut
    const blc = markerLine.color_per_bar?.[0];
    if (blc) style.borderColor = blc;
  }
  // ECharts 5.4+ supporte itemStyle.decal (équivalent du pattern Plotly).
  if (pattern?.shape === '/') {
    style.decal = {
      symbol: 'rect',
      symbolSize: 0.8,
      dashArrayX: [1, 0],
      dashArrayY: [4, 4],
      rotation: -Math.PI / 4,
      color: 'rgba(255,255,255,0.35)',
    };
  } else if (pattern?.shape === '.') {
    style.decal = {
      symbol: 'circle',
      symbolSize: 1,
      dashArrayX: [4, 8],
      dashArrayY: [4, 8],
      color: 'rgba(255,255,255,0.45)',
    };
  }
  return style;
}

// =============================================================================
// MODE 3 — match_view.10 "tug-of-war dominance"
// =============================================================================
// Reproduction fidèle du chart Plotly source (team_dominance_timeline.py) :
//
//   Y=160 ──── [Label streak alliée] ───────────  ← labels streaks
//   Y=143 ─────●─●─●─●──── ───●─●─●─●─●─ ──────  ← lane streaks alliées
//   Y=110 ─── annotations cumul mon équipe (vert si lead) ─
//   Y=100 ━━━━━━ haut barres dominance ━━━━━━━━
//        │█▌  ▐█│ █│ ▐█│  █▌█│  ▐█▌  ▌│  ▐█│
//        │  ▐█│  ▐█│ ▌█▌│ █▌  ▐██▌  ▌█▌  ▐█│
//   Y=50 ─── ligne parité (pointillée blanche) ──
//        │  ▐█│ █▌█│ █▌█│ ▐█▌  ▐█│ ▐█▌  █▌│
//   Y=0  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
//   Y=-10 ── annotations cumul équipe ennemie (vermillon si lead) ─
//   Y=0.65/lane ennemie (panel 2 simplifié en mono) ────●─●─●─●─
//   Y=-25 ── [Label streak ennemie]
//
// Y range = [-18, 170]. PAS y_max=100 comme un % bar simple — c'est un canvas
// où dominance% est la bande 0-100 et au-dessus/dessous on a des annotations.
// =============================================================================

interface MockStreak {
  player: string;
  isAlly: boolean;
  startBucketIdx: number;
  endBucketIdx: number;
  killsCount: number;
}

function convertTugOfWar(
  spec: ChartSpec,
  theme: ThemeDefault,
  _mockCtx: Record<string, unknown>,
): EChartsOption {
  // 20 buckets de 30s sur 10 min de match
  const N_BUCKETS = 20;
  const BUCKET_S = 30;
  const categories: string[] = [];

  // Mock COHÉRENT : on génère kills_per_bucket par équipe puis on dérive %, cumul.
  const myKillsPerBucket: number[] = [];
  const enemyKillsPerBucket: number[] = [];
  for (let i = 0; i < N_BUCKETS; i++) {
    const t_s = i * BUCKET_S + BUCKET_S / 2;
    const m = Math.floor(t_s / 60);
    const s = Math.floor(t_s % 60);
    categories.push(`${m}:${String(s).padStart(2, '0')}`);
    // Profil de match : alliés dominent en milieu, ennemis en début+fin
    const allyTendency = 0.5 + 0.4 * Math.sin((i / N_BUCKETS) * Math.PI * 1.4);
    // Cadence variable (1 à 5 kills par bucket)
    const intensity = 1 + Math.floor(Math.random() * 5);
    const myK = Math.round(intensity * allyTendency * 1.5);
    const enK = Math.round(intensity * (1 - allyTendency) * 1.5);
    myKillsPerBucket.push(Math.max(0, myK));
    enemyKillsPerBucket.push(Math.max(0, enK));
  }

  // Dominance % dérivée des kills réels
  const myValues = myKillsPerBucket.map((mk, i) => {
    const total = mk + enemyKillsPerBucket[i];
    return total === 0 ? 50 : Math.round((mk / total) * 100);
  });
  const enemyValues = myValues.map((v) => 100 - v);

  // Cumul = somme effective des kills par bucket (cohérent)
  const cumulMy: number[] = [];
  const cumulEnemy: number[] = [];
  let runMy = 0;
  let runEnemy = 0;
  for (let i = 0; i < N_BUCKETS; i++) {
    runMy += myKillsPerBucket[i];
    runEnemy += enemyKillsPerBucket[i];
    cumulMy.push(runMy);
    cumulEnemy.push(runEnemy);
  }

  const mockStreaks: MockStreak[] = [
    { player: 'JGtm',          isAlly: true,  startBucketIdx: 4,  endBucketIdx: 5,  killsCount: 4 },
    { player: 'NeoSpartan_42', isAlly: true,  startBucketIdx: 12, endBucketIdx: 13, killsCount: 3 },
    { player: 'XxShadowxX',    isAlly: false, startBucketIdx: 7,  endBucketIdx: 8,  killsCount: 5 },
    { player: 'BlazingFury',   isAlly: false, startBucketIdx: 15, endBucketIdx: 16, killsCount: 3 },
  ];

  // ============================================================================
  // STRUCTURE 2 GRIDS ECharts (multi-panel) :
  //   grid 0 (haut, 75%) : barres dominance + lane alliée + annotations cumul
  //   grid 1 (bas, 18%)  : lane ennemie (sous l'axe X du grid 0 en visuel)
  //
  //   xAxis 0 : catégoriel (mm:ss labels affichés en bas du grid 0)
  //   xAxis 1 : catégoriel (axe partagé, labels masqués)
  //   yAxis 0 : value [-12, 175], caché (sémantique mixte : % + lane allié + annotations)
  //   yAxis 1 : value [-2, 2], caché (lane ennemie centrée)
  // ============================================================================

  const series: Array<Record<string, unknown>> = [];

  // ── 1. Bars dominance dans grid 0 ──
  series.push({
    type: 'bar',
    stack: 'total',
    name: 'Mon équipe',
    data: myValues,
    itemStyle: { color: 'rgba(0, 114, 178, 0.80)' },
    barCategoryGap: '0%',                                    // /!\ barres collées (Plotly bargap=0)
    xAxisIndex: 0,
    yAxisIndex: 0,
    tooltip: {
      formatter: new Function('p', `return p.name + '<br>Mon équipe : ' + p.value + '%';`),
    },
  });
  series.push({
    type: 'bar',
    stack: 'total',
    name: 'Adversaires',
    data: enemyValues,
    itemStyle: { color: 'rgba(213, 94, 0, 0.80)' },
    barCategoryGap: '0%',
    xAxisIndex: 0,
    yAxisIndex: 0,
    tooltip: {
      formatter: new Function('p', `return p.name + '<br>Adversaires : ' + p.value + '%';`),
    },
  });

  // ── 2. Ligne CONTINUE alliée à y=143 sur toute la durée (grid 0) ──
  series.push({
    type: 'line',
    name: 'Lane alliée',
    data: Array.from({ length: N_BUCKETS }, (_, i) => [i, 143]),
    lineStyle: { color: 'rgba(0, 114, 178, 0.45)', width: 1.5, type: 'solid' },
    showSymbol: false,
    silent: true,
    tooltip: { show: false },
    legendHoverLink: false,
    xAxisIndex: 0,
    yAxisIndex: 0,
    z: 2,
  });

  // ── 3. Scatter markers alliés : 1 point par kill réel à y=143 (grid 0) ──
  // Cohérent avec myKillsPerBucket — 1 marker par kill, distribué dans le bucket.
  const myKillFeedPts: Array<[number, number]> = [];
  const myKillFeedTooltips: string[] = [];
  for (let b = 0; b < N_BUCKETS; b++) {
    const n = myKillsPerBucket[b];
    for (let k = 0; k < n; k++) {
      const x = b + (n > 1 ? (k / (n - 1)) * 0.6 + 0.2 : 0.5);
      myKillFeedPts.push([x, 143]);
      myKillFeedTooltips.push(`Allié — kill à ${categories[b]}`);
    }
  }
  series.push({
    type: 'scatter',
    name: 'Mes kills',
    data: myKillFeedPts.map((pt, i) => ({ value: pt, _tip: myKillFeedTooltips[i] })),
    symbol: 'circle',
    symbolSize: 7,
    itemStyle: { color: '#0072B2', opacity: 0.7 },
    legendHoverLink: false,
    xAxisIndex: 0,
    yAxisIndex: 0,
    tooltip: {
      formatter: new Function('p', `return p.data._tip || '';`),
    },
    z: 3,
  });

  // ── 4. Streaks alliées par-dessus (line+markers gros, label EN TOOLTIP) ──
  for (const streak of mockStreaks.filter((s) => s.isAlly)) {
    const xs: number[] = [];
    for (let b = streak.startBucketIdx; b <= streak.endBucketIdx; b++) xs.push(b);
    const color = '#0072B2';
    series.push({
      type: 'line',
      name: `Streak ${streak.player}`,
      data: xs.map((x) => ({
        value: [x, 143],
        _tip: `<b>${streak.player}</b> — Killing Spree ×${streak.killsCount} (${categories[x]})`,
      })),
      lineStyle: { color, width: 2.5 },
      itemStyle: { color, borderColor: 'rgba(255,255,255,0.6)', borderWidth: 1.5 },
      symbol: 'circle',
      symbolSize: 12,
      showSymbol: true,
      smooth: false,
      legendHoverLink: false,
      xAxisIndex: 0,
      yAxisIndex: 0,
      tooltip: {
        formatter: new Function('p', `return p.data._tip || '';`),
      },
      z: 5,
    });
  }

  // ── 5. Annotations cumul (panel 0) sur les bars principales ──
  const cumulMarkPoints: Array<Record<string, unknown>> = [];
  for (let i = 0; i < N_BUCKETS; i++) {
    const myLead = cumulMy[i] > cumulEnemy[i];
    const enemyLead = cumulEnemy[i] > cumulMy[i];
    cumulMarkPoints.push({
      coord: [i, 110],
      symbol: 'rect',
      symbolSize: [22, 14],
      itemStyle: {
        color: myLead ? 'rgba(0, 158, 115, 0.18)' : 'rgba(0,0,0,0)',
        borderColor: myLead ? '#009E73' : 'rgba(0,0,0,0)',
        borderWidth: myLead ? 1.5 : 0,
      },
      label: {
        show: true,
        formatter: String(cumulMy[i]),
        color: myLead ? '#009E73' : '#0072B2',
        fontSize: 10,
        fontWeight: 'bold',
      },
    });
    cumulMarkPoints.push({
      coord: [i, -7],
      symbol: 'rect',
      symbolSize: [22, 14],
      itemStyle: {
        color: enemyLead ? 'rgba(213, 94, 0, 0.18)' : 'rgba(0,0,0,0)',
        borderColor: enemyLead ? '#D55E00' : 'rgba(0,0,0,0)',
        borderWidth: enemyLead ? 1.5 : 0,
      },
      label: {
        show: true,
        formatter: String(cumulEnemy[i]),
        color: '#D55E00',
        fontSize: 10,
        fontWeight: 'bold',
      },
    });
  }
  (series[0] as Record<string, unknown>).markPoint = {
    silent: true,
    data: cumulMarkPoints,
  };
  // markLine y=50 (parité) sur la 1ère série bar
  (series[0] as Record<string, unknown>).markLine = {
    silent: true,
    symbol: 'none',
    data: [
      {
        yAxis: 50,
        lineStyle: { color: 'rgba(255, 255, 255, 0.35)', type: 'dotted', width: 1 },
        label: { show: false },
      },
    ],
  };

  // ── 6. Lane ennemie : ligne continue + scatter markers + streaks DANS GRID 1 ──
  // Ligne continue à y=0 sur toute la durée
  series.push({
    type: 'line',
    name: 'Lane ennemie',
    data: Array.from({ length: N_BUCKETS }, (_, i) => [i, 0]),
    lineStyle: { color: 'rgba(213, 94, 0, 0.45)', width: 1.5, type: 'solid' },
    showSymbol: false,
    silent: true,
    tooltip: { show: false },
    legendHoverLink: false,
    xAxisIndex: 1,
    yAxisIndex: 1,
    z: 2,
  });

  // Scatter markers ennemis : 1 point par kill réel (grid 1)
  const enemyKillFeedPts: Array<[number, number]> = [];
  const enemyKillFeedTooltips: string[] = [];
  for (let b = 0; b < N_BUCKETS; b++) {
    const n = enemyKillsPerBucket[b];
    for (let k = 0; k < n; k++) {
      const x = b + (n > 1 ? (k / (n - 1)) * 0.6 + 0.2 : 0.5);
      enemyKillFeedPts.push([x, 0]);
      enemyKillFeedTooltips.push(`Ennemi — kill à ${categories[b]}`);
    }
  }
  series.push({
    type: 'scatter',
    name: 'Kills ennemis',
    data: enemyKillFeedPts.map((pt, i) => ({ value: pt, _tip: enemyKillFeedTooltips[i] })),
    symbol: 'circle',
    symbolSize: 7,
    itemStyle: { color: '#D55E00', opacity: 0.7 },
    legendHoverLink: false,
    xAxisIndex: 1,
    yAxisIndex: 1,
    tooltip: {
      formatter: new Function('p', `return p.data._tip || '';`),
    },
    z: 3,
  });

  // Streaks ennemies par-dessus (grid 1)
  for (const streak of mockStreaks.filter((s) => !s.isAlly)) {
    const xs: number[] = [];
    for (let b = streak.startBucketIdx; b <= streak.endBucketIdx; b++) xs.push(b);
    const color = '#D55E00';
    series.push({
      type: 'line',
      name: `Streak ${streak.player}`,
      data: xs.map((x) => ({
        value: [x, 0],
        _tip: `<b>${streak.player}</b> — Killing Spree ×${streak.killsCount} (${categories[x]})`,
      })),
      lineStyle: { color, width: 2.5 },
      itemStyle: { color, borderColor: 'rgba(255,255,255,0.6)', borderWidth: 1.5 },
      symbol: 'circle',
      symbolSize: 12,
      showSymbol: true,
      smooth: false,
      legendHoverLink: false,
      xAxisIndex: 1,
      yAxisIndex: 1,
      tooltip: {
        formatter: new Function('p', `return p.data._tip || '';`),
      },
      z: 5,
    });
  }

  // ── Configuration multi-grid ECharts ──
  // Grid 0 (haut, ~75%) : barres + lane alliée + annotations cumul. yAxis [-12, 175]
  // Grid 1 (bas, ~18%)  : lane ennemie + scatter ennemis + streaks. yAxis [-2, 2]
  // L'axe X visible avec ses labels mm:ss est entre les deux grids (en bas du grid 0).
  const xAxis: Array<Record<string, unknown>> = [
    {
      // Grid 0 — axe X visible avec labels
      gridIndex: 0,
      type: 'category',
      data: categories,
      axisLine: { show: true, lineStyle: { color: 'rgba(255, 255, 255, 0.25)' } },
      axisTick: { show: true, alignWithLabel: true, length: 4 },
      axisLabel: {
        rotate: 0,
        fontSize: 9,
        color: 'rgba(245, 248, 255, 0.85)',
        interval: 1, // afficher 1 label sur 2 pour ne pas surcharger
        margin: 6,
      },
      splitLine: { show: false },
    },
    {
      // Grid 1 — axe X partagé (catégories identiques) mais labels masqués
      gridIndex: 1,
      type: 'category',
      data: categories,
      show: false,
      axisLine: { show: false },
      axisTick: { show: false },
      axisLabel: { show: false },
      splitLine: { show: false },
    },
  ];

  const yAxis: Array<Record<string, unknown>> = [
    {
      // Grid 0 — Y étendu pour bars [0..100] + cumul allié (y=110) + lane alliée (y=143)
      gridIndex: 0,
      type: 'value',
      min: -12,
      max: 175,
      show: false,
      splitLine: { show: false },
    },
    {
      // Grid 1 — Y centré sur la ligne ennemie (y=0)
      gridIndex: 1,
      type: 'value',
      min: -2,
      max: 2,
      show: false,
      splitLine: { show: false },
    },
  ];

  const option: EChartsOption = {
    grid: [
      // Grid 0 : panel haut (barres + alliés)
      { left: 14, right: 14, top: 30, height: '70%', containLabel: false },
      // Grid 1 : panel bas (ennemis), sous l'axe X du grid 0
      { left: 14, right: 14, top: '83%', height: '12%', containLabel: false },
    ] as unknown as EChartsOption['grid'],
    tooltip: {
      trigger: 'item',
      backgroundColor: 'rgba(15, 20, 35, 0.95)',
      borderColor: 'rgba(255, 255, 255, 0.15)',
      textStyle: { color: '#F5F8FF', fontSize: 11 },
    },
    legend: {
      orient: 'horizontal',
      top: 4,
      textStyle: { color: theme.font.color, fontSize: 11 },
      data: ['Mon équipe', 'Adversaires'],
    },
    xAxis: xAxis as unknown as EChartsOption['xAxis'],
    yAxis: yAxis as unknown as EChartsOption['yAxis'],
    series,
  };
  return applyThemeBase(option, theme);
}

// =============================================================================
// MODE 4 — match_view.11 "cadence histogram"
// =============================================================================
// Stacked bars (my_team + enemy kills par bucket de 15s) + 2 lines MA(3).
// Annotation arrow sur le pic max.
// =============================================================================
function convertCadenceHistogram(
  spec: ChartSpec,
  theme: ThemeDefault,
  _mockCtx: Record<string, unknown>,
): EChartsOption {
  // 30 buckets de 15s sur ~7.5 min
  const N_BUCKETS = 30;
  const BUCKET_S = 15;
  const categories: string[] = [];
  const myKills: number[] = [];
  const enemyKills: number[] = [];

  for (let i = 0; i < N_BUCKETS; i++) {
    const t_s = i * BUCKET_S + BUCKET_S / 2;
    const m = Math.floor(t_s / 60);
    const s = Math.floor(t_s % 60);
    categories.push(`${m}:${String(s).padStart(2, '0')}`);
    // Cadence variable avec un pic au milieu
    const intensity = 1.5 + 2.5 * Math.sin((i / N_BUCKETS) * Math.PI);
    myKills.push(Math.max(0, Math.round(intensity + (Math.random() - 0.5) * 1.8)));
    enemyKills.push(Math.max(0, Math.round(intensity * 0.85 + (Math.random() - 0.5) * 1.6)));
  }

  // Moyennes glissantes (window=3)
  const movingAvg = (arr: number[], window = 3): (number | null)[] => {
    const result: (number | null)[] = [];
    for (let i = 0; i < arr.length; i++) {
      if (i < window - 1) {
        result.push(null);
        continue;
      }
      let sum = 0;
      for (let j = i - window + 1; j <= i; j++) sum += arr[j];
      result.push(+(sum / window).toFixed(1));
    }
    return result;
  };
  const maMy = movingAvg(myKills);
  const maEnemy = movingAvg(enemyKills);

  const series: Array<Record<string, unknown>> = [
    {
      type: 'bar',
      stack: 'total',
      name: 'Mon équipe',
      data: myKills,
      itemStyle: {
        color: 'rgba(0, 114, 178, 0.80)',
        borderColor: '#0072B2',
        borderWidth: 1,
      },
    },
    {
      type: 'bar',
      stack: 'total',
      name: 'Adversaires',
      data: enemyKills,
      itemStyle: {
        color: 'rgba(213, 94, 0, 0.80)',
        borderColor: '#D55E00',
        borderWidth: 1,
      },
    },
    {
      type: 'line',
      name: 'MA Mon équipe',
      data: maMy,
      lineStyle: { color: 'rgba(0, 114, 178, 0.85)', width: 3 },
      itemStyle: { color: 'rgba(0, 114, 178, 0.85)' },
      symbol: 'none',
      smooth: true,
    },
    {
      type: 'line',
      name: 'MA Adversaires',
      data: maEnemy,
      lineStyle: { color: 'rgba(213, 94, 0, 0.85)', width: 3 },
      itemStyle: { color: 'rgba(213, 94, 0, 0.85)' },
      symbol: 'none',
      smooth: true,
    },
  ];

  // Annotation pic via markPoint sur la 1ère série
  const totalPerBucket = myKills.map((m, i) => m + enemyKills[i]);
  const peakIdx = totalPerBucket.indexOf(Math.max(...totalPerBucket));
  (series[0] as Record<string, unknown>).markPoint = {
    silent: true,
    symbolSize: 1,
    label: {
      show: true,
      formatter: 'PIC',
      position: 'top',
      color: '#E69F00',
      fontSize: 10,
      backgroundColor: 'rgba(0,0,0,0.5)',
      padding: [2, 4],
    },
    data: [{ coord: [peakIdx, totalPerBucket[peakIdx]] }],
  };

  const xAxis: Record<string, unknown> = {
    type: 'category',
    data: categories,
    name: 'Temps',
    nameTextStyle: { color: theme.font.color },
    axisLabel: { rotate: -45, fontSize: 9, color: 'rgba(245, 248, 255, 0.85)' },
    splitLine: { show: false },
  };
  const yAxis: Record<string, unknown> = {
    type: 'value',
    name: 'Kills',
    nameTextStyle: { color: theme.font.color },
    min: 0,
    splitLine: { show: true, lineStyle: { color: 'rgba(255, 255, 255, 0.07)' } },
  };

  const margin = spec.layout.margin ?? { l: 40, r: 20, t: 30, b: 60 };
  const option: EChartsOption = {
    grid: { left: margin.l, right: margin.r, top: margin.t + 20, bottom: margin.b + 20, containLabel: true },
    tooltip: { trigger: 'axis', axisPointer: { type: 'shadow' } },
    legend: { orient: 'horizontal', bottom: 5, textStyle: { color: theme.font.color } },
    xAxis,
    yAxis,
    series,
  };
  return applyThemeBase(option, theme);
}

// =============================================================================
// MODE 5 — teammates.09 "weapon kills multi-joueurs horizontal"
// =============================================================================
// Bar horizontal groupé : 1 série par joueur, Y = noms d'armes (triés ASC),
// X = nombre de kills (text outside avec valeur), bandes zebra background.
// =============================================================================
function convertWeaponKillsMulti(
  spec: ChartSpec,
  theme: ThemeDefault,
): EChartsOption {
  // 12 armes Halo Infinite typiques
  const weapons = [
    'Plasma Pistol', 'Sniper Rifle', 'Hydra', 'Skewer', 'Stalker Rifle',
    'Bandit', 'Pulse Carbine', 'Sidekick', 'BR75', 'Commando', 'AR (MA40)', 'Mêlée',
  ];
  const players = ['JGtm', 'NeoSpartan_42', 'BlazingFury', 'ShadowKnight'];
  const colors = ['#33D6FF', '#EF5350', '#FFCA28', '#26C6DA'];

  // Mock kills par joueur par arme (matrice [N_players × N_weapons])
  // Distribution biaisée : armes du milieu (BR/AR) plus utilisées
  const matrix: number[][] = players.map((_, pi) =>
    weapons.map((_, wi) => {
      const intensity = wi >= 7 ? 8 + Math.floor(Math.random() * 12) : Math.floor(Math.random() * 6);
      return Math.max(0, intensity + (pi === 0 ? 4 : 0));
    }),
  );

  const series: Array<Record<string, unknown>> = players.map((name, idx) => ({
    type: 'bar',
    name,
    data: matrix[idx].map((v) => ({
      value: [v, ''], // valeurs sur X, label vide sur Y géré par yAxis.data
    })).map((d, wi) => ({
      value: matrix[idx][wi],
      itemStyle: { color: colors[idx] },
    })),
    orientation: 'horizontal',
    itemStyle: { color: colors[idx] },
    label: {
      show: true,
      position: 'right',
      formatter: new Function('p', 'return p.value > 0 ? String(p.value) : "";'),
      color: '#fff',
      fontSize: 10,
      fontWeight: 'bold',
    },
    barCategoryGap: '35%',
    barGap: '8%',
  }));

  // Bandes zebra : markArea sur la 1ère série
  const zebraData: Array<[Record<string, unknown>, Record<string, unknown>]> = [];
  for (let i = 0; i < weapons.length; i++) {
    if (i % 2 === 0) {
      zebraData.push([
        { yAxis: i - 0.5, itemStyle: { color: 'rgba(255, 255, 255, 0.04)' } },
        { yAxis: i + 0.5 },
      ]);
    }
  }
  if (series[0]) {
    (series[0] as Record<string, unknown>).markArea = {
      silent: true,
      data: zebraData,
    };
  }

  const dynamicHeight = Math.max(350, weapons.length * 38);

  const option: EChartsOption = {
    grid: { left: 100, right: 80, top: 50, bottom: 30, containLabel: false },
    tooltip: { trigger: 'item' },
    legend: false, // panel flottant teammates.01 sert de légende
    xAxis: {
      type: 'value',
      show: false,
      splitLine: { show: false },
    },
    yAxis: {
      type: 'category',
      data: weapons,
      axisLabel: { color: 'rgba(245, 248, 255, 0.85)', fontSize: 10 },
      splitLine: { show: false },
    },
    series,
    title: {
      text: 'Kills par arme — escouade',
      left: 'center',
      top: 5,
      textStyle: { color: theme.font.color, fontSize: 13 },
    },
  };
  // @ts-expect-error - height meta pour render-html
  option.__height = dynamicHeight;
  // Background transparent (cf. YAML)
  option.backgroundColor = 'transparent';
  return option;
}

// =============================================================================
// MODE 6 — teammates.13 "perf vs historique" (barres horizontales groupées,
// session colorée selon palier vs historique gris)
// =============================================================================

function convertMapPerfVsHistory(spec: ChartSpec, theme: ThemeDefault): EChartsOption {
  // Mock : 12 cartes communes session/historique
  const maps = [
    'Aquarius', 'Recharge', 'Live Fire', 'Streets', 'Bazaar', 'Behemoth',
    'Catalyst', 'Argyle', 'Solitude', 'Empyrean', 'Forbidden', 'Cliffhanger',
  ];
  // Performances mock : session vs historique (échelle 0-100)
  const sessionPerf = [82, 71, 64, 55, 48, 38, 25, 67, 73, 58, 41, 50];
  const historyPerf = [70, 62, 58, 50, 45, 42, 35, 60, 65, 55, 48, 52];
  const sessionN = [4, 3, 5, 2, 6, 3, 1, 4, 3, 2, 5, 3];
  const historyN = [42, 38, 51, 28, 60, 35, 18, 44, 33, 25, 55, 31];

  // Couleur session par palier (SCORE_THRESHOLDS)
  const perfColor = (v: number): string => {
    if (v >= 75) return '#00e676';      // green
    if (v >= 60) return '#00b7eb';      // cyan
    if (v >= 45) return '#ffb300';      // amber
    if (v >= 30) return '#FF8C00';      // orange
    return '#e53935';                    // red
  };

  const sessionData = sessionPerf.map((v, i) => ({
    value: v,
    itemStyle: { color: perfColor(v), opacity: 0.85 },
    customN: sessionN[i],
  }));
  const historyData = historyPerf.map((v, i) => ({
    value: v,
    itemStyle: { color: 'rgba(120,120,120,0.45)' },
    customN: historyN[i],
  }));

  const dynamicHeight = Math.max(360, maps.length * 32 + 100);

  const option: EChartsOption = {
    backgroundColor: 'transparent',
    textStyle: { color: theme.font.color, fontSize: theme.font.size },
    grid: { left: 110, right: 30, top: 50, bottom: 70, containLabel: false },
    legend: {
      orient: 'horizontal',
      bottom: 10,
      textStyle: { color: 'rgba(245, 248, 255, 0.85)' },
      data: ['Historique', 'Session actuelle'],
    },
    tooltip: {
      trigger: 'item',
      backgroundColor: 'rgba(20, 30, 50, 0.92)',
      borderColor: 'rgba(120, 160, 220, 0.4)',
      textStyle: { color: '#fff', fontSize: 12 },
      formatter: new Function(
        'p',
        `return p.name + "<br>" + p.seriesName + " = " + p.value.toFixed(1) + " (N=" + (p.data.customN || 0) + ")";`,
      ) as unknown as Record<string, unknown>,
    },
    xAxis: {
      type: 'value',
      min: 0,
      max: 100,
      axisLine: { lineStyle: { color: 'rgba(245, 248, 255, 0.6)' } },
      axisLabel: { color: 'rgba(245, 248, 255, 0.85)' },
      splitLine: { lineStyle: { color: 'rgba(245, 248, 255, 0.08)' } },
    },
    yAxis: {
      type: 'category',
      data: maps,
      axisLabel: { color: 'rgba(245, 248, 255, 0.85)', fontSize: 11 },
      splitLine: { show: false },
    },
    series: [
      {
        name: 'Historique',
        type: 'bar',
        data: historyData,
        barGap: 0,
        barCategoryGap: '30%',
        markLine: {
          symbol: 'none',
          lineStyle: { type: 'dotted', color: 'rgba(180,180,180,0.6)', width: 1 },
          data: [{ xAxis: 0 }],
          label: { show: false },
        },
      },
      {
        name: 'Session actuelle',
        type: 'bar',
        data: sessionData,
        barGap: 0,
        barCategoryGap: '30%',
      },
    ],
    title: {
      text: 'Performance par carte — Session vs Historique',
      left: 'center',
      top: 8,
      textStyle: { color: theme.font.color, fontSize: 13 },
    },
  };
  // @ts-expect-error - height meta pour render-html
  option.__height = dynamicHeight;
  return option;
}

// =============================================================================
// MODE 7 — teammates.14 "stats par minute" (3 catégories X × N joueurs,
// morts en barres négatives sous l'axe X)
// =============================================================================

function convertPerMinuteStats(spec: ChartSpec, theme: ThemeDefault): EChartsOption {
  const categories = ['Frags/min', 'Morts/min', 'Assists/min'];
  // Mock : 4 joueurs avec leurs stats/min
  const players = [
    { name: 'JGtm', color: '#56B4E9', neg: '#ff6666', kpm: 1.42, dpm: 0.95, apm: 1.18 },
    { name: 'NeoSpartan_42', color: '#E69F00', neg: '#e63333', kpm: 1.18, dpm: 1.12, apm: 0.94 },
    { name: 'BlazingFury', color: '#009E73', neg: '#b31c1c', kpm: 0.98, dpm: 0.88, apm: 1.32 },
    { name: 'ShadowKnight', color: '#CC79A7', neg: '#7a1111', kpm: 0.74, dpm: 0.96, apm: 0.62 },
  ];

  const series: Array<Record<string, unknown>> = players.map((p) => ({
    name: p.name,
    type: 'bar',
    data: [
      { value: p.kpm, itemStyle: { color: p.color }, label: { show: true, position: 'top', formatter: p.kpm.toFixed(2) } },
      { value: -p.dpm, itemStyle: { color: p.neg }, label: { show: true, position: 'bottom', formatter: p.dpm.toFixed(2) } },
      { value: p.apm, itemStyle: { color: p.color }, label: { show: true, position: 'top', formatter: p.apm.toFixed(2) } },
    ],
    barCategoryGap: '40%',
    barGap: '10%',
  }));

  const option: EChartsOption = {
    backgroundColor: 'transparent',
    textStyle: { color: theme.font.color, fontSize: theme.font.size },
    grid: { left: 60, right: 30, top: 50, bottom: 70, containLabel: false },
    legend: false,
    tooltip: {
      trigger: 'item',
      formatter: new Function(
        'p',
        `return p.seriesName + " — " + p.name + " : " + Math.abs(p.value).toFixed(2);`,
      ) as unknown as Record<string, unknown>,
    },
    xAxis: {
      type: 'category',
      data: categories,
      axisLabel: { color: 'rgba(245, 248, 255, 0.85)', fontSize: 11 },
    },
    yAxis: {
      type: 'value',
      axisLine: { lineStyle: { color: 'rgba(255,255,255,0.75)', width: 2 } },
      axisLabel: { color: 'rgba(245, 248, 255, 0.85)', formatter: new Function('v', 'return Math.abs(v).toFixed(1);') as unknown as string },
      splitLine: { lineStyle: { color: 'rgba(245, 248, 255, 0.08)' } },
    },
    series,
    title: {
      text: 'Stats par minute — Frags / Morts / Assists',
      left: 'center',
      top: 8,
      textStyle: { color: theme.font.color, fontSize: 13 },
    },
  };
  // @ts-expect-error - height meta
  option.__height = 360;
  return option;
}

// =============================================================================
// MODE 8 — teammates.17 "first events butterfly" (bins de 15s × N joueurs,
// frags positifs / morts négatives)
// =============================================================================

function convertFirstEventsButterfly(spec: ChartSpec, theme: ThemeDefault): EChartsOption {
  // Bins de 15s : 0-15s, 15-30s, ..., 2m45s-3m
  const bins = ['15s', '30s', '45s', '1m00s', '1m15s', '1m30s', '1m45s', '2m00s', '2m15s', '2m30s', '2m45s', '3m00s'];
  const players = [
    { name: 'JGtm', color: '#56B4E9' },
    { name: 'NeoSpartan_42', color: '#E69F00' },
    { name: 'BlazingFury', color: '#009E73' },
    { name: 'ShadowKnight', color: '#CC79A7' },
  ];

  // Mock counts : kills (positifs), deaths (négatifs)
  const killCounts = [
    [1, 3, 5, 4, 3, 2, 1, 1, 0, 0, 0, 0],   // JGtm — agressif early
    [0, 1, 3, 4, 5, 3, 2, 1, 0, 0, 0, 0],   // NeoSpartan_42 — pic mid-early
    [0, 0, 2, 3, 4, 5, 3, 2, 1, 0, 0, 0],   // BlazingFury — mid game
    [0, 0, 1, 2, 2, 3, 4, 3, 2, 1, 1, 0],   // ShadowKnight — late game
  ];
  const deathCounts = [
    [0, 0, 1, 2, 3, 4, 4, 2, 1, 0, 0, 0],
    [1, 1, 2, 3, 4, 3, 2, 1, 0, 0, 0, 0],
    [1, 2, 3, 3, 2, 2, 1, 1, 0, 0, 0, 0],
    [0, 1, 2, 4, 3, 2, 2, 1, 1, 0, 0, 0],
  ];

  const series: Array<Record<string, unknown>> = [];
  players.forEach((p, idx) => {
    series.push({
      name: p.name,
      type: 'bar',
      stack: undefined,
      data: killCounts[idx],
      itemStyle: { color: p.color },
      barGap: '0%',
    });
    series.push({
      name: p.name + ' (morts)',
      type: 'bar',
      data: deathCounts[idx].map((v) => -v),
      itemStyle: { color: p.color, opacity: 0.85 },
      barGap: '0%',
    });
  });

  // Séparateurs verticaux : markLine sur la 1re série uniquement
  const seps = bins.slice(1).map((_, i) => ({ xAxis: i + 0.5 }));
  (series[0] as Record<string, unknown>).markLine = {
    symbol: 'none',
    silent: true,
    lineStyle: { type: 'dotted', color: 'rgba(255,255,255,0.18)', width: 1 },
    data: seps,
    label: { show: false },
  };

  const maxC = Math.max(
    ...killCounts.flat(),
    ...deathCounts.flat(),
  );
  const yMax = Math.ceil(maxC * 1.15);

  const option: EChartsOption = {
    backgroundColor: 'transparent',
    textStyle: { color: theme.font.color, fontSize: theme.font.size },
    grid: { left: 60, right: 30, top: 60, bottom: 80, containLabel: false },
    legend: false,
    tooltip: {
      trigger: 'item',
      formatter: new Function(
        'p',
        `var v = Math.abs(p.value); var label = p.value >= 0 ? "1er frag" : "1ère mort"; return "<b>" + p.seriesName.replace(" (morts)", "") + "</b><br>" + p.name + "<br>" + label + " : " + v + " matchs";`,
      ) as unknown as Record<string, unknown>,
    },
    xAxis: {
      type: 'category',
      data: bins,
      axisLabel: { color: 'rgba(245, 248, 255, 0.85)', fontSize: 10, rotate: 0 },
      splitLine: { show: false },
    },
    yAxis: {
      type: 'value',
      min: -yMax,
      max: yMax,
      axisLine: { lineStyle: { color: 'rgba(255,255,255,0.75)', width: 2 } },
      axisLabel: { color: 'rgba(245, 248, 255, 0.85)', formatter: new Function('v', 'return Math.abs(v);') as unknown as string },
      splitLine: { lineStyle: { color: 'rgba(245, 248, 255, 0.08)' } },
    },
    series,
    title: {
      text: 'Premier frag (haut) / Première mort (bas) — par tranche de 15 s',
      left: 'center',
      top: 8,
      textStyle: { color: theme.font.color, fontSize: 13 },
    },
  };
  // @ts-expect-error - height meta
  option.__height = 440;
  return option;
}

// =============================================================================
// timeseries.28 — Top matches by week (matchs Top vs Total par semaine)
// =============================================================================

function convertTopByWeek(spec: ChartSpec, theme: ThemeDefault): EChartsOption {
  const weeks = [
    '2026-02-02', '2026-02-09', '2026-02-16', '2026-02-23',
    '2026-03-02', '2026-03-09', '2026-03-16', '2026-03-23',
    '2026-03-30', '2026-04-06', '2026-04-13', '2026-04-20',
  ];
  const tops = [3, 5, 2, 4, 6, 3, 4, 5, 7, 3, 4, 5];
  const totals = [12, 15, 10, 13, 17, 14, 12, 16, 18, 11, 13, 14];
  const ratios = tops.map((t, i) => (t / totals[i]) * 100);

  const option: EChartsOption = {
    backgroundColor: 'transparent',
    textStyle: { color: theme.font.color, fontSize: theme.font.size },
    grid: { left: 60, right: 60, top: 60, bottom: 80, containLabel: false },
    legend: { bottom: 8, textStyle: { color: 'rgba(245,248,255,0.85)' } },
    tooltip: { trigger: 'axis', axisPointer: { type: 'shadow' } },
    xAxis: {
      type: 'category',
      data: weeks,
      axisLabel: { color: 'rgba(245,248,255,0.85)', rotate: -45, fontSize: 10 },
    },
    yAxis: [
      {
        type: 'value',
        name: 'Matchs',
        nameTextStyle: { color: 'rgba(245,248,255,0.85)' },
        axisLabel: { color: 'rgba(245,248,255,0.85)' },
        splitLine: { lineStyle: { color: 'rgba(245,248,255,0.05)' } },
      },
      {
        type: 'value',
        name: '% Top',
        max: 100,
        nameTextStyle: { color: 'rgba(245,248,255,0.85)' },
        axisLabel: { color: 'rgba(245,248,255,0.85)', formatter: '{value}%' },
        splitLine: { show: false },
      },
    ],
    series: [
      {
        name: 'Top (rank=1)',
        type: 'bar',
        data: tops,
        itemStyle: { color: '#FFD700' },
        barCategoryGap: '30%',
      },
      {
        name: 'Total semaine',
        type: 'bar',
        data: totals,
        itemStyle: { color: 'rgba(120,120,120,0.45)' },
        barCategoryGap: '30%',
      },
      {
        name: '% Top',
        type: 'line',
        yAxisIndex: 1,
        data: ratios.map((v) => parseFloat(v.toFixed(1))),
        smooth: true,
        symbol: 'circle',
        symbolSize: 6,
        lineStyle: { color: '#41d6ff', width: 2 },
      },
    ],
    title: {
      text: 'Top matches by week (rank=1) vs Total — % Top sur Y2',
      left: 'center',
      top: 8,
      textStyle: { color: theme.font.color, fontSize: 13 },
    },
  };
  // @ts-expect-error - height meta
  option.__height = 400;
  return option;
}

// =============================================================================
// session_compare.13 — Participation trend (6 axes A vs B en bars horizontales)
// =============================================================================

function convertParticipationTrend(spec: ChartSpec, theme: ThemeDefault): EChartsOption {
  const axes = ['Objectifs', 'Combat', 'Support', 'Score', 'Impact', 'Survie'];
  // Mock : Session A meilleure en combat/score, B meilleure en survie/support
  const valuesA = [55, 78, 42, 70, 65, 50];
  const valuesB = [48, 60, 65, 55, 50, 72];

  const option: EChartsOption = {
    backgroundColor: 'transparent',
    textStyle: { color: '#E0E0E0', fontSize: theme.font.size },
    grid: { left: 100, right: 30, top: 30, bottom: 70, containLabel: false },
    legend: { bottom: 10, textStyle: { color: '#E0E0E0' } },
    tooltip: { trigger: 'axis', axisPointer: { type: 'shadow' } },
    xAxis: {
      type: 'value',
      max: 110,
      name: '%',
      nameLocation: 'end',
      nameTextStyle: { color: '#E0E0E0' },
      axisLabel: { color: '#E0E0E0' },
      splitLine: { lineStyle: { color: 'rgba(255,255,255,0.1)' } },
    },
    yAxis: {
      type: 'category',
      data: axes,
      axisLabel: { color: '#E0E0E0' },
      splitLine: { show: false },
    },
    series: [
      { name: 'Session A', type: 'bar', data: valuesA, itemStyle: { color: '#E74C3C' }, barCategoryGap: '30%' },
      { name: 'Session B', type: 'bar', data: valuesB, itemStyle: { color: '#3498DB' }, barCategoryGap: '30%' },
    ],
    title: { text: 'Participation trend — 6 axes A vs B (% normalisés)', left: 'center', top: 8, textStyle: { color: theme.font.color, fontSize: 13 } },
  };
  // @ts-expect-error - height meta
  option.__height = 320;
  return option;
}

// =============================================================================
// session_compare.11 — Modes breakdown (bars horizontales A vs B par mode)
// =============================================================================

function convertModesBreakdownSC(spec: ChartSpec, theme: ThemeDefault): EChartsOption {
  const modes = ['BTB', 'CTF', 'King of the Hill', 'Land Grab', 'Oddball', 'Quickplay', 'Slayer', 'Strongholds'];
  const countsA = [3, 0, 2, 1, 0, 5, 4, 0];
  const countsB = [0, 3, 1, 0, 2, 4, 1, 1];
  const dynamicHeight = Math.max(280, modes.length * 48 + 80);

  const option: EChartsOption = {
    backgroundColor: 'transparent',
    textStyle: { color: '#E0E0E0', fontSize: theme.font.size },
    grid: { left: 140, right: 30, top: 40, bottom: 50, containLabel: false },
    legend: { top: 8, right: 30, textStyle: { color: '#E0E0E0' } },
    tooltip: { trigger: 'axis', axisPointer: { type: 'shadow' } },
    xAxis: {
      type: 'value',
      name: 'Parties',
      nameTextStyle: { color: '#E0E0E0' },
      axisLabel: { color: '#E0E0E0' },
      minInterval: 1,
      splitLine: { lineStyle: { color: 'rgba(255,255,255,0.08)' } },
    },
    yAxis: { type: 'category', data: modes, axisLabel: { color: '#E0E0E0' }, splitLine: { show: false } },
    series: [
      { name: 'Session A', type: 'bar', data: countsA, itemStyle: { color: '#E74C3C' }, barCategoryGap: '40%' },
      { name: 'Session B', type: 'bar', data: countsB, itemStyle: { color: '#3498DB' }, barCategoryGap: '40%' },
    ],
    title: { text: 'Modes breakdown — comparatif modes joués', left: 'center', top: 8, textStyle: { color: theme.font.color, fontSize: 13 } },
  };
  // @ts-expect-error - height meta
  option.__height = dynamicHeight;
  return option;
}

// =============================================================================
// session_compare.08 — Bar comparison A/B sur 2 axes Y (kd Y1 + win rate Y2)
// =============================================================================

function convertSessionCompareBar(spec: ChartSpec, theme: ThemeDefault): EChartsOption {
  const leftMetrics = ['F/D ratio', 'Précision', 'Score/min'];
  const rightMetric = 'Win rate';

  const aLeft = [1.55, 0.62, 1.18];
  const bLeft = [1.05, 0.51, 0.95];
  const aWr = 65;
  const bWr = 50;
  const histLeft = [1.20, 0.55, 1.05];
  const histWr = 55;

  const option: EChartsOption = {
    backgroundColor: 'transparent',
    textStyle: { color: '#E0E0E0', fontSize: theme.font.size },
    grid: { left: 60, right: 60, top: 40, bottom: 70, containLabel: false },
    legend: { bottom: 10, textStyle: { color: '#E0E0E0' } },
    tooltip: { trigger: 'axis', axisPointer: { type: 'shadow' } },
    xAxis: {
      type: 'category',
      data: [...leftMetrics, rightMetric],
      axisLabel: { color: '#E0E0E0', interval: 0 },
    },
    yAxis: [
      {
        type: 'value',
        name: 'Ratio',
        nameTextStyle: { color: '#E0E0E0' },
        axisLabel: { color: '#E0E0E0' },
        splitLine: { lineStyle: { color: 'rgba(255,255,255,0.08)' } },
      },
      {
        type: 'value',
        name: 'Win rate (%)',
        max: 100,
        nameTextStyle: { color: '#E0E0E0' },
        axisLabel: { color: '#E0E0E0', formatter: '{value}%' },
        splitLine: { show: false },
      },
    ],
    series: [
      { name: 'Session A', type: 'bar', data: [...aLeft, null], itemStyle: { color: '#E74C3C' }, barCategoryGap: '30%' },
      { name: 'Session B', type: 'bar', data: [...bLeft, null], itemStyle: { color: '#3498DB' }, barCategoryGap: '30%' },
      { name: 'Hist (n=8)', type: 'bar', data: [...histLeft, null], itemStyle: { color: '#9B59B6', decal: { symbol: 'rect', color: 'rgba(255,255,255,0.4)' } }, barCategoryGap: '30%' },
      { name: 'Session A — Win rate', type: 'bar', yAxisIndex: 1, data: [null, null, null, aWr], itemStyle: { color: '#E74C3C' }, barCategoryGap: '30%' },
      { name: 'Session B — Win rate', type: 'bar', yAxisIndex: 1, data: [null, null, null, bWr], itemStyle: { color: '#3498DB' }, barCategoryGap: '30%' },
      { name: 'Hist — Win rate', type: 'bar', yAxisIndex: 1, data: [null, null, null, histWr], itemStyle: { color: '#9B59B6', decal: { symbol: 'rect', color: 'rgba(255,255,255,0.4)' } }, barCategoryGap: '30%' },
    ],
    title: { text: 'Bar comparison — A/B (Y1) + Win rate (Y2) avec moyenne historique', left: 'center', top: 8, textStyle: { color: theme.font.color, fontSize: 13 } },
  };
  // @ts-expect-error - height meta
  option.__height = 360;
  return option;
}
