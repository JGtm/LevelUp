import type { ChartSpec, EChartsOption, ThemeDefault } from '../types.js';
import { buildGrid, buildAxis, applyThemeBase, resolvePaletteToken, resolveI18nToken } from '../converter.js';

/**
 * Convertit un chart `heatmap` (cas winrate jour × heure).
 *
 * Caractéristiques :
 * - 7 jours × 24 heures = 168 cellules (grille reconstituée)
 * - z = win_rate (0.0–1.0), text = count par cellule (vide si 0)
 * - colorscale 3-stops (red 0% → amber 50% → green 100%)
 * - autorange:'reversed' sur Y → Lun en haut
 * - cellules avec count<min_matches → null (visualMap mask)
 */
export function convertHeatmap(
  spec: ChartSpec,
  theme: ThemeDefault,
  mockCtx: Record<string, unknown>,
  warnings: string[],
): EChartsOption {
  // Dispatchs avec mocks complets en amont — pas de dépendance à `spec.heatmap`
  if (spec.id === 'teammates.03') {
    return convertSquadMapHeatmap(spec, theme);
  }
  if (spec.id === 'teammates.15') {
    return convertSquadIntensityHeatmap(spec, theme);
  }
  // timeseries.21 : intensity heatmap solo (réutilise teammates.15 logic)
  if (spec.id === 'timeseries.21') {
    return convertSquadIntensityHeatmap(spec, theme);
  }
  // timeseries.27 / win_loss.03 : WL heatmap (jour × heure)
  if (spec.id === 'timeseries.27' || spec.id === 'win_loss.03') {
    return convertWlHeatmap(spec, theme);
  }

  if (!spec.heatmap) {
    warnings.push('chart_kind=heatmap mais section `heatmap:` absente du YAML');
    return { series: [] };
  }
  const hm = spec.heatmap;

  // Labels pour X (heures) et Y (jours)
  const xLabels = hm.x_labels?.values ?? Array.from({ length: 24 }, (_, i) => `${String(i).padStart(2, '0')}h`);
  const lang = (mockCtx.lang as string) ?? 'fr';
  const yLabels =
    (hm.y_labels?.values as string[]) ??
    (lang === 'fr'
      ? hm.y_labels?.values_fr ?? ['Lun', 'Mar', 'Mer', 'Jeu', 'Ven', 'Sam', 'Dim']
      : hm.y_labels?.values_en ?? ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun']);

  // Mock data 7×24 (synthétique : pic entre 18h et 23h, plus calme matin)
  const heatmapData: Array<[number, number, number | null, number]> = [];
  for (let d = 0; d < 7; d++) {
    for (let h = 0; h < 24; h++) {
      // count synthétique
      let count = 0;
      if (h >= 18 && h <= 23) count = Math.floor(2 + Math.random() * 5);
      else if (h >= 12 && h <= 17) count = Math.floor(Math.random() * 3);
      else count = Math.floor(Math.random() * 2);

      // win_rate synthétique
      const winRate = count > 0 ? Math.random() : null;

      // ECharts heatmap : data = [[xIdx, yIdx, value, count?], ...]
      heatmapData.push([h, d, winRate, count]);
    }
  }

  // visualMap (équivalent colorscale Plotly)
  const stops = hm.colorscale.stops.map((s) => ({
    value: s.value,
    color: resolvePaletteToken(s.color, theme),
  }));
  const visualMap: Record<string, unknown> = {
    min: hm.z_range?.[0] ?? 0,
    max: hm.z_range?.[1] ?? 1,
    calculable: false,
    show: hm.colorbar?.visible !== false,
    orient: 'vertical',
    right: 10,
    top: 'center',
    inRange: {
      // Ordonner les couleurs selon les stops (du min au max)
      color: stops.map((s) => s.color as string),
    },
    // Format pourcentage si tickformat ".0%"
    formatter:
      hm.colorbar?.tickformat === '.0%'
        ? new Function('val', 'return (val * 100).toFixed(0) + "%";')
        : undefined,
    text: hm.colorbar?.title ? [resolveI18nToken(hm.colorbar.title) ?? '', ''] : undefined,
  };

  // Formatters construits via new Function pour inliner xLabels/yLabels (sinon perte de closure)
  const xLabelsJson = JSON.stringify(xLabels);
  const yLabelsJson = JSON.stringify(yLabels);

  const labelFormatter =
    hm.cell_label?.empty_when === 'count == 0'
      ? new Function(
          'params',
          'return params.data && params.data.count > 0 ? String(params.data.count) : "";',
        )
      : new Function('params', 'return params.data && params.data.count !== undefined ? String(params.data.count) : "";');

  const tooltipFormatter = new Function(
    'params',
    `
    var xLabels = ${xLabelsJson};
    var yLabels = ${yLabelsJson};
    var v = params.data.value;
    var hIdx = v[0], dIdx = v[1], wr = v[2];
    var day = yLabels[dIdx];
    var hour = xLabels[hIdx];
    var wrFormatted = (wr === null || wr === undefined) ? 'n/a' : ((wr * 100).toFixed(1) + '%');
    return day + ' ' + hour + '<br>i18n:viz_t.hover_win_rate: ' + wrFormatted + '<br>i18n:viz_t.trace_matches: ' + params.data.count;
    `,
  );

  // Series heatmap
  const series: Array<Record<string, unknown>> = [
    {
      type: 'heatmap',
      data: heatmapData.map(([x, y, value, count]) => ({
        value: [x, y, value],
        count,
      })),
      label: {
        show: true,
        fontSize: hm.cell_label?.font_size ?? 10,
        formatter: labelFormatter,
      },
      itemStyle: {},
      emphasis: {
        itemStyle: { shadowBlur: 8 },
      },
      tooltip: {
        formatter: tooltipFormatter,
      },
    },
  ];

  const xAxis = buildAxis(spec.layout.xaxis, theme, { categories: xLabels });
  const yAxis = buildAxis(spec.layout.yaxis, theme, { categories: yLabels });

  const option: EChartsOption = {
    grid: buildGrid(spec),
    tooltip: { trigger: 'item' }, // override : heatmap a un tooltip per-item, pas axis
    legend: false, // les heatmaps n'ont pas de légende
    xAxis,
    yAxis,
    visualMap,
    series,
  };

  return applyThemeBase(option, theme);
}

/**
 * Mock spécifique teammates.03 — heatmap joueur × carte (top 15) avec perf 0-100.
 * Colorscale DISCRÈTE (pieces) au lieu d'un gradient continu.
 */
function convertSquadMapHeatmap(spec: ChartSpec, theme: ThemeDefault): EChartsOption {
  const players = ['JGtm', 'NeoSpartan_42', 'BlazingFury', 'ShadowKnight'];
  const maps = [
    'Aquarius', 'Behemoth', 'Streets', 'Recharge', 'Live Fire', 'Catalyst',
    'Argyle', 'Bazaar', 'Solitude', 'Empyrean', 'Forbidden', 'Oasis',
    'Origin', 'Banished', 'Chasm',
  ];

  // Génération matrice perf [n_players × n_maps], certaines cells à null
  const data: Array<{ value: [number, number, number | null]; player: string; map: string }> = [];
  for (let p = 0; p < players.length; p++) {
    for (let m = 0; m < maps.length; m++) {
      // 15% des cells sont null (joueur n'a pas joué cette carte)
      if (Math.random() < 0.12) {
        data.push({ value: [m, p, null], player: players[p], map: maps[m] });
      } else {
        // Perf entre 30 et 95 avec biais selon joueur
        const baseMid = 50 + (p === 0 ? 15 : (p % 2 === 0 ? 5 : -5));
        const perf = Math.max(30, Math.min(95, Math.round(baseMid + (Math.random() - 0.5) * 30)));
        data.push({ value: [m, p, perf], player: players[p], map: maps[m] });
      }
    }
  }

  // Colorscale DISCRÈTE par paliers (cf. _discrete_perf_colorscale)
  const visualMap: Record<string, unknown> = {
    type: 'piecewise',
    pieces: [
      { gte: 0,  lt: 30,  color: '#FF4444', label: 'Faible' },
      { gte: 30, lt: 50,  color: '#FF8C00', label: 'Sous-moyenne' },
      { gte: 50, lt: 65,  color: '#FFBF00', label: 'Moyenne' },
      { gte: 65, lt: 80,  color: '#00B7EB', label: 'Bonne' },
      { gte: 80, lte: 100, color: '#50C878', label: 'Excellente' },
    ],
    show: true,
    orient: 'vertical',
    right: 5,
    top: 'center',
    textStyle: { color: theme.font.color, fontSize: 10 },
  };

  const series: Array<Record<string, unknown>> = [
    {
      type: 'heatmap',
      data: data.map((d) => ({ value: d.value, _player: d.player, _map: d.map })),
      label: { show: false },
      emphasis: { itemStyle: { shadowBlur: 6 } },
      tooltip: {
        formatter: new Function(
          'p',
          `if (p.value[2] === null) return p.data._player + ' — ' + p.data._map + '<br>(non joué)';
           return p.data._player + ' — ' + p.data._map + '<br>Perf=' + p.value[2].toFixed(1);`,
        ),
      },
    },
  ];

  const dynamicHeight = Math.max(180, players.length * 60 + 100);

  const option: EChartsOption = {
    grid: { left: 80, right: 90, top: 30, bottom: 100, containLabel: false },
    tooltip: { trigger: 'item' },
    legend: false,
    xAxis: {
      type: 'category',
      data: maps,
      axisLabel: { rotate: -35, fontSize: 9, color: 'rgba(245, 248, 255, 0.85)' },
      splitLine: { show: false },
    },
    yAxis: {
      type: 'category',
      data: players,
      axisLabel: { color: 'rgba(245, 248, 255, 0.92)', fontSize: 11 },
      splitLine: { show: false },
    },
    visualMap,
    series,
  };
  // @ts-expect-error - height meta pour render-html
  option.__height = dynamicHeight;
  return applyThemeBase(option, theme);
}

// =============================================================================
// teammates.15 — Heatmap d'intensité matchs × phases (10 phases normalisées)
// Colorscale cyan→jaune→ambre→orange→rouge (5 stops). Toggle joueur en amont
// (ne fait pas partie du chart lui-même, géré par segmented_control Streamlit).
// =============================================================================

function convertSquadIntensityHeatmap(spec: ChartSpec, theme: ThemeDefault): EChartsOption {
  // Mock : 12 matchs × 10 phases (normalisées 0-1)
  const phases = ['0-10%', '10-20%', '20-30%', '30-40%', '40-50%', '50-60%', '60-70%', '70-80%', '80-90%', '90-100%'];
  const matches = [
    'Aquarius — 28 avr.',
    'Recharge — 28 avr.',
    'Live Fire — 27 avr.',
    'Streets — 27 avr.',
    'Bazaar — 26 avr.',
    'Behemoth — 26 avr.',
    'Catalyst — 25 avr.',
    'Argyle — 25 avr.',
    'Solitude — 24 avr.',
    'Empyrean — 24 avr.',
    'Forbidden — 23 avr.',
    'Cliffhanger — 23 avr.',
  ];

  // Mock z-data : 12 matchs × 10 phases, valeurs 0..1, motifs variés
  const data: Array<[number, number, number]> = [];
  for (let m = 0; m < matches.length; m++) {
    for (let p = 0; p < phases.length; p++) {
      // Profil synthétique : pics vers le mid-late game (phases 4-7)
      let v: number;
      if (m % 4 === 0) {
        // pic early
        v = Math.max(0, 1 - Math.abs(p - 2) * 0.25 + (Math.random() - 0.5) * 0.2);
      } else if (m % 4 === 1) {
        // pic mid
        v = Math.max(0, 1 - Math.abs(p - 5) * 0.22 + (Math.random() - 0.5) * 0.2);
      } else if (m % 4 === 2) {
        // pic late
        v = Math.max(0, 1 - Math.abs(p - 7) * 0.2 + (Math.random() - 0.5) * 0.2);
      } else {
        // distribué
        v = 0.4 + (Math.random() - 0.5) * 0.5;
      }
      v = Math.min(1, Math.max(0, v));
      data.push([p, m, parseFloat(v.toFixed(2))]);
    }
  }

  const dynamicHeight = Math.max(380, matches.length * 28 + 120);

  const visualMap: Record<string, unknown> = {
    min: 0,
    max: 1,
    calculable: false,
    show: true,
    orient: 'vertical',
    right: 10,
    top: 'middle',
    inRange: {
      color: ['#38C8C8', '#FFFF00', '#FFB300', '#FF5500', '#FF1A00'],
    },
    text: ['intense', 'calme'],
    textStyle: { color: 'rgba(245, 248, 255, 0.85)' },
  };

  const option: EChartsOption = {
    backgroundColor: 'transparent',
    textStyle: { color: theme.font.color, fontSize: theme.font.size },
    grid: { left: 160, right: 80, top: 60, bottom: 60, containLabel: false },
    legend: false,
    tooltip: {
      trigger: 'item',
      formatter: new Function(
        'p',
        `return "Phase " + p.value[0] + " (" + ${JSON.stringify(phases)}[p.value[0]] + ")<br>" + ${JSON.stringify(matches)}[p.value[1]] + "<br>Intensité : " + p.value[2].toFixed(2);`,
      ) as unknown as Record<string, unknown>,
    },
    xAxis: {
      type: 'category',
      data: phases,
      axisLabel: { color: 'rgba(245, 248, 255, 0.85)', fontSize: 10 },
      splitArea: { show: true },
    },
    yAxis: {
      type: 'category',
      data: matches,
      axisLabel: { color: 'rgba(245, 248, 255, 0.85)', fontSize: 11 },
      splitArea: { show: true },
    },
    visualMap,
    series: [
      {
        name: 'Intensité',
        type: 'heatmap',
        data,
        emphasis: { itemStyle: { borderColor: '#fff', borderWidth: 1 } },
      },
    ],
    title: {
      text: "Heatmap d'intensité — kills par phase de match",
      left: 'center',
      top: 8,
      textStyle: { color: theme.font.color, fontSize: 13 },
    },
  };
  // @ts-expect-error - height meta
  option.__height = dynamicHeight;
  return applyThemeBase(option, theme);
}

// =============================================================================
// timeseries.27 — Win/Loss heatmap jour × heure
// =============================================================================

function convertWlHeatmap(spec: ChartSpec, theme: ThemeDefault): EChartsOption {
  const days = ['Lun', 'Mar', 'Mer', 'Jeu', 'Ven', 'Sam', 'Dim'];
  const hours = Array.from({ length: 24 }, (_, h) => `${String(h).padStart(2, '0')}h`);

  // Mock : winrate par cellule (jour, heure)
  const data: Array<[number, number, number | null, number]> = [];
  for (let d = 0; d < 7; d++) {
    for (let h = 0; h < 24; h++) {
      let count = 0;
      if (h >= 18 && h <= 23) count = Math.floor(2 + Math.random() * 6);
      else if (h >= 13 && h <= 17) count = Math.floor(Math.random() * 3);
      else if (h >= 8 && h <= 12) count = Math.floor(Math.random() * 2);
      const wr = count > 0 ? 0.3 + Math.random() * 0.6 : null;
      data.push([h, d, wr, count]);
    }
  }

  const option: EChartsOption = {
    backgroundColor: 'transparent',
    textStyle: { color: theme.font.color, fontSize: theme.font.size },
    grid: { left: 60, right: 60, top: 60, bottom: 50, containLabel: false },
    legend: false,
    tooltip: {
      trigger: 'item',
      formatter: new Function(
        'p',
        `if (p.value[2] === null || p.value[2] === undefined) return p.value[0] + 'h ' + ${JSON.stringify(days)}[p.value[1]] + '<br>Aucun match';
return p.value[0] + 'h ' + ${JSON.stringify(days)}[p.value[1]] + '<br>WR: ' + (p.value[2] * 100).toFixed(0) + '% (' + p.value[3] + ' matchs)';`,
      ) as unknown as Record<string, unknown>,
    },
    visualMap: {
      min: 0,
      max: 1,
      show: true,
      orient: 'vertical',
      right: 10,
      top: 'middle',
      inRange: { color: ['#e53935', '#ffb300', '#00e676'] },
      text: ['Bon', 'Mauvais'],
      textStyle: { color: 'rgba(245,248,255,0.85)' },
    },
    xAxis: {
      type: 'category',
      data: hours,
      axisLabel: { color: 'rgba(245,248,255,0.85)', fontSize: 9, interval: 1 },
      splitArea: { show: true },
    },
    yAxis: {
      type: 'category',
      data: days,
      inverse: true,
      axisLabel: { color: 'rgba(245,248,255,0.85)' },
      splitArea: { show: true },
    },
    series: [
      {
        name: 'WL heatmap',
        type: 'heatmap',
        data,
        emphasis: { itemStyle: { borderColor: '#fff', borderWidth: 1 } },
      },
    ],
    title: {
      text: 'Win Rate par jour × heure',
      left: 'center',
      top: 8,
      textStyle: { color: theme.font.color, fontSize: 13 },
    },
  };
  // @ts-expect-error - height meta
  option.__height = 380;
  return applyThemeBase(option, theme);
}
