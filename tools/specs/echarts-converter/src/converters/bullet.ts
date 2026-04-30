import type { ChartSpec, EChartsOption, ThemeDefault } from '../types.js';
import { applyThemeBase } from '../converter.js';

/**
 * Convertit un chart `bullet` (Plotly bullet — winrate session vs historique par carte).
 *
 * Structure : 4 traces overlay Plotly → 2 séries bar ECharts par carte (session + histo)
 * avec le pattern simplifié : on dessine UNE barre par catégorie pour chaque trace
 * (session colorée vert/ambre/rouge selon delta vs histo, histo rose Okabe), barmode=overlay.
 * + markLine x=0.5 (ligne parité) + entrées légende couleur.
 */
export function convertBullet(
  spec: ChartSpec,
  theme: ThemeDefault,
  _mockCtx: Record<string, unknown>,
  _warnings: string[],
): EChartsOption {
  // Mock : 8 cartes Halo Infinite avec winrate session + historique
  const mockData = [
    { map: 'Aquarius',  session: 0.72, history: 0.58 },         // session > histo (vert)
    { map: 'Behemoth',  session: 0.45, history: 0.62 },         // session < histo (rouge)
    { map: 'Streets',   session: 0.55, history: 0.58 },         // ≈ (ambre)
    { map: 'Recharge',  session: 0.80, history: 0.65 },         // vert
    { map: 'Live Fire', session: 0.42, history: 0.50 },         // rouge
    { map: 'Catalyst',  session: 0.60, history: 0.55 },         // ≈ (ambre)
    { map: 'Argyle',    session: 0.0,  history: 0.40 },         // 0% — marker spécial
    { map: 'Bazaar',    session: 0.75, history: 0.70 },         // ≈
  ];

  const THRESHOLD = 0.05;
  const palette = {
    green: '#3DFFB5',                                          // HALO_COLORS.green
    amber: '#FFBF00',                                          // HALO_COLORS.amber
    red:   '#FF4D6D',                                          // HALO_COLORS.red
    rose:  '#CC79A7',                                          // _OKABE_ROSE
  };

  const sessColor = (curr: number, hist: number): string => {
    if (curr >= hist + THRESHOLD) return palette.green;
    if (curr <= hist - THRESHOLD) return palette.red;
    return palette.amber;
  };

  const mapNames = mockData.map((d) => d.map);
  const sessionData = mockData.map((d) => ({
    value: d.session,
    itemStyle: { color: sessColor(d.session, d.history) },
  }));
  const historyData = mockData.map((d) => ({
    value: d.history,
    itemStyle: { color: palette.rose, opacity: 0.85 },
  }));

  // Markers x=0 pour les cartes à 0% session
  const zeroSessionMaps = mockData
    .filter((d) => d.session === 0)
    .map((d) => d.map);

  const series: Array<Record<string, unknown>> = [
    {
      type: 'bar',
      name: 'Historique',
      data: historyData,
      barCategoryGap: '20%',
      barGap: '-100%',                                          // /!\ overlay : 2 bars superposées
      label: { show: false },
      tooltip: {
        formatter: new Function(
          'p',
          `return p.name + '<br>Historique: ' + (p.value * 100).toFixed(1) + '%';`,
        ),
      },
      z: 1,
    },
    {
      type: 'bar',
      name: 'Session actuelle',
      data: sessionData,
      barCategoryGap: '20%',
      barGap: '-100%',
      label: { show: false },
      tooltip: {
        formatter: new Function(
          'p',
          `return p.name + '<br>Session: ' + (p.value * 100).toFixed(1) + '%';`,
        ),
      },
      z: 2,
    },
  ];

  // markLine x=0.5 (ligne de parité)
  (series[0] as Record<string, unknown>).markLine = {
    silent: true,
    symbol: 'none',
    label: { show: false },
    data: [
      {
        xAxis: 0.5,
        lineStyle: { color: 'rgba(180, 180, 180, 0.6)', type: 'dotted', width: 1.5 },
      },
    ],
  };

  // Markers cartes 0% (line-ns)
  if (zeroSessionMaps.length > 0) {
    series.push({
      type: 'scatter',
      name: '0% (toutes défaites)',
      data: zeroSessionMaps.map((m) => [0, m]),
      symbol: 'rect',
      symbolSize: [3, 18],
      itemStyle: { color: palette.red },
      legendHoverLink: false,
      tooltip: {
        formatter: new Function('p', `return p.value[1] + '<br>Session=0%';`),
      },
      z: 3,
    });
  }

  // 3 entrées légende couleur factices (séries bar invisibles avec name uniquement)
  const pctTxt = `±${Math.round(THRESHOLD * 100)}%`;
  for (const [color, label] of [
    [palette.green, `Session > hist. (+${Math.round(THRESHOLD * 100)}%)`],
    [palette.amber, `Session ≈ hist. (${pctTxt})`],
    [palette.red, `Session < hist. (-${Math.round(THRESHOLD * 100)}%)`],
  ]) {
    series.push({
      type: 'bar',
      name: label,
      data: [],
      itemStyle: { color },
      legendHoverLink: false,
      silent: true,
      tooltip: { show: false },
    });
  }

  const option: EChartsOption = {
    grid: { left: 100, right: 30, top: 30, bottom: 70, containLabel: true },
    tooltip: { trigger: 'item' },
    legend: {
      orient: 'horizontal',
      bottom: 5,
      textStyle: { color: theme.font.color, fontSize: 10 },
    },
    xAxis: {
      type: 'value',
      min: -0.05,
      max: 1.1,
      axisLabel: {
        formatter: new Function('val', `return Math.round(val * 100) + '%';`),
      },
      splitLine: { show: true, lineStyle: { color: 'rgba(255, 255, 255, 0.07)' } },
    },
    yAxis: {
      type: 'category',
      data: mapNames,
      axisLabel: { color: 'rgba(245, 248, 255, 0.85)', fontSize: 11 },
      inverse: true,                                            // 1ère carte en haut
    },
    series,
  };
  return applyThemeBase(option, theme);
}
