import type { ChartSpec, EChartsOption, ThemeDefault } from '../types.js';
import { applyThemeBase, resolvePaletteToken, resolveI18nToken } from '../converter.js';

/**
 * Convertit un chart `pie` (Plotly Pie) en ECharts pie.
 *
 * Caractéristiques mappées :
 * - hole=0.35 → radius: ['35%', '70%'] (donut)
 * - sort=false → ne PAS retrier (passer data déjà ordonné)
 * - textinfo=percent → label.formatter: '{d}%'
 * - textposition=inside → label.position: 'inside'
 * - hovertemplate avec %{label}, %{value}, %{percent} → tooltip.formatter custom
 * - color_palette → data[i].itemStyle.color par index (cycle si N items > N couleurs)
 * - legend verticale droite → legend.orient='vertical', legend.right=0
 */
export function convertPie(
  spec: ChartSpec,
  theme: ThemeDefault,
  mockCtx: Record<string, unknown>,
  warnings: string[],
): EChartsOption {
  // session_compare.03 : 2 donuts W/L/T/DNF côte à côte (multi-grid)
  if (spec.id === 'session_compare.03') {
    return convertOutcomesDonutsSC(spec, theme);
  }

  // @ts-expect-error - section pie custom du YAML
  const pieSpec = spec.pie as Record<string, unknown> | undefined;
  if (!pieSpec) {
    warnings.push('chart_kind=pie mais section `pie:` absente du YAML');
    return { series: [] };
  }

  // Mock data : 5 armes typiques
  const labels =
    (mockCtx.labels as string[]) ??
    ['MA40 AR', 'BR75', 'Sidekick', 'Plasma Carbine', 'Bulldog'];
  const values = (mockCtx.values as number[]) ?? [12, 8, 6, 4, 3];

  // Résoudre la palette depuis le YAML
  const trace = spec.traces?.[0];
  const palette =
    ((trace as { color_palette?: string[] })?.color_palette as string[] | undefined) ?? [
      '#29B6F6',
      '#FF7043',
      '#66BB6A',
      '#FFA726',
      '#AB47BC',
      '#26C6DA',
      '#EC407A',
      '#8D6E63',
    ];

  // Cycle la palette si N items > N couleurs
  const sliceColors = labels.map((_, i) => {
    const c = palette[i % palette.length];
    return resolvePaletteToken(c, theme) ?? c;
  });

  // Convertir hole (0..1) en radius ECharts
  const hole = (pieSpec.hole as number) ?? 0;
  const radiusInner = `${Math.round(hole * 100)}%`;
  const radiusOuter = '70%';

  // Construire data avec couleur per-item
  const data = labels.map((label, i) => ({
    name: label,
    value: values[i],
    itemStyle: { color: sliceColors[i] },
  }));

  // Position du label
  const labelPosition = (pieSpec.textposition as string) ?? 'inside';
  // Format du label (textinfo)
  const textinfo = (pieSpec.textinfo as string) ?? 'percent';
  const labelFormatter =
    textinfo === 'percent'
      ? new Function('p', 'return Math.round(p.percent) + "%";')
      : textinfo === 'value'
        ? new Function('p', 'return String(p.value);')
        : new Function('p', 'return p.name;');

  // Tooltip avec la structure typique du hovertemplate Plotly
  const tooltipFormatter = new Function(
    'p',
    `return p.name + '<br>' + p.value + ' frags (' + Math.round(p.percent) + '%)';`,
  );

  const series: Array<Record<string, unknown>> = [
    {
      type: 'pie',
      name: resolveI18nToken(trace?.name) ?? 'pie',
      radius: [radiusInner, radiusOuter],
      center: ['40%', '50%'], // léger décalage à gauche pour laisser place à la légende droite
      data,
      label: {
        show: true,
        position: labelPosition,
        color: '#fff',
        fontWeight: 'bold',
        formatter: labelFormatter,
      },
      labelLine: {
        show: labelPosition === 'outside',
      },
      // sort:false → on passe data déjà trié, mais ECharts a un comportement par défaut :
      // les slices commencent à 12h en SENS HORAIRE par ordre d'apparition dans data.
      // Pour préserver l'ordre du DataFrame d'origine, c'est exactement ce que l'on veut.
      tooltip: {
        formatter: tooltipFormatter,
      },
    },
  ];

  // Légende verticale à droite (par défaut pour les pies)
  const legendBlock = (() => {
    const layoutLegend = spec.layout?.legend as Record<string, unknown> | undefined;
    const orientation = (layoutLegend?.orientation as string) ?? 'v';
    const isVertical = orientation === 'v';
    return {
      show: true,
      orient: isVertical ? 'vertical' : 'horizontal',
      right: 0,
      top: 'middle',
      textStyle: { color: '#dce8ff', fontSize: 11 },
      type: 'scroll', // si trop d'items
    };
  })();

  // Marges très serrées pour laisser place au donut
  const margin = spec.layout?.margin ?? { l: 10, r: 10, t: 10, b: 10 };

  const option: EChartsOption = {
    grid: undefined as unknown as EChartsOption['grid'], // pas de grid pour les pies
    tooltip: {
      trigger: 'item',
      backgroundColor: theme.hoverlabel.bgcolor,
      borderColor: theme.hoverlabel.bordercolor,
      textStyle: { color: theme.font.color },
    },
    legend: legendBlock,
    series,
  };

  // grid n'est pas utilisé par les pies, mais on garde les marges via le container CSS
  (option as unknown as Record<string, unknown>).grid = {
    left: margin.l,
    right: margin.r * 6, // espace pour la légende verticale à droite
    top: margin.t,
    bottom: margin.b,
    containLabel: false,
  };

  return applyThemeBase(option, theme);
}

// =============================================================================
// session_compare.03 — 2 donuts Outcomes côte à côte (Session A | Session B)
// =============================================================================

function convertOutcomesDonutsSC(spec: ChartSpec, theme: ThemeDefault): EChartsOption {
  // Mock : Session A 6W/4L/1T, Session B 5W/6L/0T
  const dataA = [
    { name: 'Wins', value: 6, itemStyle: { color: '#00e676' } },
    { name: 'Losses', value: 4, itemStyle: { color: '#e53935' } },
    { name: 'Ties', value: 1, itemStyle: { color: '#ab47bc' } },
  ];
  const dataB = [
    { name: 'Wins', value: 5, itemStyle: { color: '#00e676' } },
    { name: 'Losses', value: 6, itemStyle: { color: '#e53935' } },
  ];
  const totalA = dataA.reduce((s, d) => s + d.value, 0);
  const winsA = dataA[0].value;
  const totalB = dataB.reduce((s, d) => s + d.value, 0);
  const winsB = dataB[0].value;

  const option: EChartsOption = {
    backgroundColor: 'transparent',
    textStyle: { color: '#E0E0E0', fontSize: theme.font.size },
    legend: false,
    tooltip: { trigger: 'item', formatter: '{b}: {c} partie(s) ({d}%)' },
    title: [
      { text: 'Session A', left: '25%', top: 8, textAlign: 'center', textStyle: { color: '#E74C3C', fontSize: 13, fontWeight: 'bold' as const } },
      { text: 'Session B', left: '75%', top: 8, textAlign: 'center', textStyle: { color: '#3498DB', fontSize: 13, fontWeight: 'bold' as const } },
      { text: `${winsA}/${totalA}`, left: '25%', top: '50%', textAlign: 'center', textVerticalAlign: 'middle', textStyle: { color: '#E0E0E0', fontSize: 14, fontWeight: 'bold' as const } },
      { text: `${winsB}/${totalB}`, left: '75%', top: '50%', textAlign: 'center', textVerticalAlign: 'middle', textStyle: { color: '#E0E0E0', fontSize: 14, fontWeight: 'bold' as const } },
    ] as unknown as EChartsOption['title'],
    series: [
      {
        type: 'pie',
        center: ['25%', '50%'],
        radius: ['45%', '70%'],
        data: dataA,
        label: { color: '#E0E0E0', formatter: '{b}: {d}%', fontSize: 10 },
        itemStyle: { borderColor: 'rgba(0,0,0,0.25)', borderWidth: 1 },
      },
      {
        type: 'pie',
        center: ['75%', '50%'],
        radius: ['45%', '70%'],
        data: dataB,
        label: { color: '#E0E0E0', formatter: '{b}: {d}%', fontSize: 10 },
        itemStyle: { borderColor: 'rgba(0,0,0,0.25)', borderWidth: 1 },
      },
    ],
  };
  // @ts-expect-error - height meta
  option.__height = 280;
  return applyThemeBase(option, theme);
}
