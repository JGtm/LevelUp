import type { ChartSpec, EChartsOption, ThemeDefault } from '../types.js';
import { applyThemeBase, resolvePaletteToken, resolveI18nToken } from '../converter.js';

/**
 * Convertit un chart `gauge` (Plotly Indicator gauge+number) en ECharts gauge.
 *
 * Caractéristiques principales :
 * - axisLine.lineStyle.color = liste [pct, color] (équivalent Plotly steps[])
 * - progress.itemStyle.color = couleur dynamique selon valeur (équivalent Plotly bar.color)
 * - pointer = équivalent threshold Plotly
 * - detail.formatter = nombre central avec suffixe %
 * - title affiché en haut (avec subtitle dans une seconde ligne)
 */
export function convertGauge(
  spec: ChartSpec,
  theme: ThemeDefault,
  mockCtx: Record<string, unknown>,
  warnings: string[],
): EChartsOption {
  // Mock value pour la preview (ex: 55% pour rang, 12% pour Hero)
  const mockValue =
    typeof mockCtx.value === 'number'
      ? mockCtx.value
      : spec.id === 'career.02'
        ? 12.4
        : 55.0;
  const mockTitle =
    (mockCtx.title as string) ??
    (spec.id === 'career.01' ? 'Capitaine III' : 'Progression vers Héros');
  const mockSubtitle =
    (mockCtx.subtitle as string) ??
    (spec.id === 'career.01' ? '125,400 / 226,000 XP' : '1,156,420 / 9,319,350 XP');

  // Détermine la couleur de la barre selon la valeur (cf. _progress_bar_color)
  const barColor = pickProgressBarColor(mockValue);

  // Steps Plotly → axisLine.lineStyle.color
  // Pas direct dans le YAML ; on hardcode la palette du gauge Halo
  // (cf. _build_progress_gauge dans career_progress_circle.py)
  const axisLineColor: Array<[number, string]> = [
    [0.25, 'rgba(255, 102, 102, 0.25)'], // rouge clair (alpha 0.08 dans Plotly, on remonte un peu pour ECharts)
    [0.5, 'rgba(255, 170, 0, 0.25)'], // orange
    [0.75, 'rgba(51, 214, 255, 0.25)'], // cyan
    [1, 'rgba(0, 255, 136, 0.25)'], // vert
  ];

  const series: Array<Record<string, unknown>> = [
    {
      type: 'gauge',
      min: 0,
      max: 100,
      startAngle: 215,
      endAngle: -35, // gauge demi-circulaire en haut
      radius: '90%',
      center: ['50%', '60%'],
      data: [
        {
          value: mockValue,
          name: '',
        },
      ],
      // axisLine = anneau d'arrière-plan (équivalent Plotly steps)
      axisLine: {
        lineStyle: {
          width: 18,
          color: axisLineColor,
        },
      },
      // progress = la barre dynamique (équivalent Plotly bar)
      progress: {
        show: true,
        width: 18,
        roundCap: false,
        itemStyle: {
          color: barColor,
        },
      },
      // pointer = équivalent threshold Plotly
      pointer: {
        show: true,
        length: '60%',
        width: 4,
        itemStyle: {
          color: '#ffffff',
        },
      },
      // axisTick = invisible (Plotly tickwidth:0)
      axisTick: {
        show: false,
      },
      // splitLine = graduations tous les 25% (Plotly dtick:25)
      splitLine: {
        show: true,
        distance: 0,
        length: 6,
        lineStyle: {
          color: 'rgba(102,102,102,0.5)',
          width: 1,
        },
      },
      // axisLabel = ticks de pourcentage (Plotly tickfont)
      axisLabel: {
        show: true,
        distance: 22,
        color: '#666',
        fontSize: 10,
        formatter: new Function('value', 'return Math.round(value).toString();'),
      },
      // title = nom du rang ou label "Progression Héros"
      title: {
        show: true,
        offsetCenter: [0, '-32%'],
        color: '#ffffff',
        fontSize: 14,
        fontWeight: 'bold',
      },
      // detail = nombre central (le pourcentage avec suffixe)
      detail: {
        show: true,
        offsetCenter: [0, '24%'],
        color: '#ffffff',
        fontSize: 30,
        fontWeight: 'bold',
        formatter: new Function('value', 'return Math.round(value) + "%";'),
      },
    },
  ];

  const option: EChartsOption = {
    grid: undefined as unknown as EChartsOption['grid'],
    tooltip: {
      formatter: new Function(
        'params',
        `return ${JSON.stringify(mockTitle)} + '<br>' + ${JSON.stringify(mockSubtitle)} + '<br>' + Math.round(params.value) + '%';`,
      ),
    },
    legend: false,
    series,
    title: {
      text: mockTitle,
      subtext: mockSubtitle,
      left: 'center',
      top: '85%',
      textStyle: {
        color: '#ffffff',
        fontSize: 14,
      },
      subtextStyle: {
        color: '#aaa',
        fontSize: 11,
      },
    } as Record<string, unknown> as EChartsOption['title'],
  };

  applyThemeBase(option, theme);
  // Override : le gauge utilise font.color: white spécifique
  option.textStyle = { color: '#ffffff', fontSize: 13 };

  return option;
}

/**
 * Reproduit la fonction `_progress_bar_color` du Python :
 *   pct >= 75 → vert, >= 50 → cyan, >= 25 → orange, sinon rouge.
 */
function pickProgressBarColor(pct: number): string {
  if (pct >= 75) return '#00ff88';
  if (pct >= 50) return '#33d6ff';
  if (pct >= 25) return '#ffaa00';
  return '#ff6666';
}
