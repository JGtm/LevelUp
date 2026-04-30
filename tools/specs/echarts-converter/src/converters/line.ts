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
 * Convertit un chart `line` (Plotly Scatter mode lines/lines+markers) en ECharts line.
 *
 * Couvre :
 * - Multi-traces avec couleurs distinctes
 * - mode lines vs lines+markers (Plotly) → showSymbol true/false (ECharts)
 * - line.dash (solid/dot/dash/dashdot) → lineStyle.type
 * - fill='tozeroy' / 'tonexty' (CI ribbons) → areaStyle
 * - visible: 'legendonly' → legend.selected[name]:false
 * - hrect (zones de tier) → markArea
 * - hline (seuil) → markLine
 */
export function convertLine(
  spec: ChartSpec,
  theme: ThemeDefault,
  mockCtx: Record<string, unknown>,
  warnings: string[],
): EChartsOption {
  // teammates.16 : famille de 6 sub-charts en cascade — on rend le 1er
  // (KD combined butterfly time-series) comme représentant visuel
  if (spec.id === 'teammates.16') {
    return convertTrioKdCombined(spec, theme);
  }
  // timeseries.01 : form_score (réutilise teammates.05 logic — courbe perf
  // avec areaStyle positif/négatif et highlight session courante)
  if (spec.id === 'timeseries.01') {
    return convertFormScoreSelf(spec, theme);
  }
  // win_loss.06 : personal_score bars amber + smoothing rolling 10
  if (spec.id === 'win_loss.06') {
    return convertPersonalScoreMock(spec, theme);
  }
  // session_compare.10 : K/D progression A vs B + accuracy Y2
  if (spec.id === 'session_compare.10') {
    return convertSessionCompareKD(spec, theme);
  }
  // session_compare.09 : cumulative comparison (line + hline 0)
  if (spec.id === 'session_compare.09') {
    return convertSessionCompareCumulative(spec, theme);
  }

  // Mock data : N points temporels selon le chart
  const dates =
    (mockCtx.dates as string[]) ??
    (spec.id === 'career.03'
      ? generateMockDates(20, 14)
      : generateMockDates(30, 7));

  const series: Array<Record<string, unknown>> = [];
  const legendSelected: Record<string, boolean> = {};

  // Pour match_view.09 : pré-calcul des data des 2 courbes + algo de level GLOBAL
  // (étiquettes des courbes kills + deaths considérées ensemble pour éviter superposition).
  let impactGlobalLevels: Map<string, number> | null = null;
  let impactKillsData: Array<[number, number]> | null = null;
  let impactDeathsData: Array<[number, number]> | null = null;
  if (spec.id === 'match_view.09') {
    impactKillsData = generateMatchClockMock('kills_cum', 600_000);
    impactDeathsData = generateMatchClockMock('deaths_cum', 600_000);
    impactGlobalLevels = computeGlobalImpactLevels(impactKillsData, impactDeathsData, 600_000);
  }

  // Palette pour les amis (cf. _OTHER_PLAYERS_COLORS dans career_charts.py:21)
  const FRIENDS_PALETTE = [
    '#EF5350', // rouge
    '#29B6F6', // bleu clair
    '#FFCA28', // ambre
    '#26C6DA', // cyan
    '#FF7043', // orange-rouge
    '#AB47BC', // violet foncé
  ];
  // Mock : 3 amis pour la preview career.03 (au lieu de 6 pour pas surcharger)
  const MOCK_FRIENDS =
    spec.id === 'career.03'
      ? [
          { gamertag: 'JGtm', color: FRIENDS_PALETTE[0] },
          { gamertag: 'NeoSpartan_42', color: FRIENDS_PALETTE[1] },
          { gamertag: 'BlazingFury', color: FRIENDS_PALETTE[2] },
        ]
      : [];

  let traceIdx = -1;
  for (const trace of spec.traces) {
    traceIdx++;
    // Skip le template (pas une trace réelle, juste un descripteur de pattern)
    if (trace.id === 'friends_traces_template') {
      // Génère ici les traces des amis (jusqu'à N_FRIENDS × 4 sous-traces)
      for (const friend of MOCK_FRIENDS) {
        const subTraces = generateFriendSubTraces(friend, dates, spec.id);
        for (const sub of subTraces) {
          series.push(sub.series);
          if (sub.legendOnly) legendSelected[sub.name] = false;
        }
      }
      continue;
    }

    // match_view.13 : 1 trace template → N joueurs mock (jusqu'à 8 joueurs)
    if (trace.id === 'player_traces_template') {
      const mockPlayers = generateMockPlayers(8);
      for (const p of mockPlayers) {
        series.push({
          type: 'line',
          name: p.name,
          data: p.data, // [[time_ms, kd_diff], ...]
          itemStyle: { color: p.color },
          lineStyle: { color: p.color, width: p.isMain ? 3 : 1.5, type: 'solid' },
          symbol: 'circle',
          symbolSize: p.isMain ? 6 : 4,
          showSymbol: false, // trop de markers sinon
          opacity: p.isMain ? 1.0 : 0.65,
          smooth: false,
        });
      }
      continue;
    }

    // teammates.08 : 1 template → 4 joueurs mock avec bars groupées par match
    if (trace.id === 'player_metric_bars_template') {
      const players = ['JGtm', 'NeoSpartan_42', 'BlazingFury', 'ShadowKnight'];
      const colors = ['#33D6FF', '#EF5350', '#FFCA28', '#26C6DA'];
      players.forEach((name, idx) => {
        const data = Array.from({ length: 12 }, (_, mi) => {
          // Mock killing spree : 0-12 par match, avec biais joueur
          const base = idx === 0 ? 5 : 3 + (idx % 2);
          return [`#${mi + 1}`, Math.max(0, Math.round(base + (Math.random() - 0.4) * 5))];
        });
        series.push({
          type: 'bar',
          name,
          data,
          itemStyle: { color: colors[idx] },
        });
      });
      continue;
    }

    // show_when : pour la preview mock, on rend TOUTES les traces (sauf le template).
    // Les conditions documentent l'affichage runtime mais ne filtrent pas le mock.

    const traceName = resolveI18nToken(trace.name) ?? trace.id ?? 'series';
    const color = resolvePaletteToken(trace.color, theme) ?? '#33D6FF';

    // Mode → showSymbol (ECharts)
    const showSymbol = trace.mode?.includes('markers') ?? false;

    // Dash → lineStyle.type
    const lineDash = trace.line?.dash;
    const lineType =
      lineDash === 'dot'
        ? 'dotted'
        : lineDash === 'dash'
          ? 'dashed'
          : lineDash === 'dashdot'
            ? 'dashed' // ECharts n'a pas dashdot direct
            : 'solid';

    // Mock data : timeline-aware selon le chart_id
    let seriesData: Array<[string | number, number]>;
    if (spec.id === 'career.03') {
      const segMock = generateXpTimelineMock(trace.id ?? 'main_xp', 0);
      seriesData = segMock.dates.map((d, i) => [d, segMock.values[i]]);
    } else if (spec.id === 'match_view.09') {
      // Réutiliser les data pré-calculées pour cohérence avec computeGlobalImpactLevels
      if (trace.id === 'kills_cum' && impactKillsData) seriesData = impactKillsData;
      else if (trace.id === 'deaths_cum' && impactDeathsData) seriesData = impactDeathsData;
      else seriesData = generateMatchClockMock(trace.id ?? 'kills_cum', 600_000);
    } else if (spec.id === 'match_view.10') {
      // Team dominance : x = time_ms, y = pct dominance par bucket de 30s
      seriesData = generateDominanceMock(trace.id ?? '', 600_000);
    } else if (spec.id === 'teammates.04') {
      // Squad timeline : 8 sessions avec perf, winrate, mmr
      seriesData = generateSquadTimelineMock(trace.id ?? '', traceIdx);
    } else if (spec.id === 'teammates.05') {
      // Form score : 1 série par membre escouade (4 mocks players)
      seriesData = generateFormScoreMock(trace.id ?? '', traceIdx);
    } else {
      const values = generateMockValues(dates.length, trace.id ?? 'main', spec.id);
      seriesData = dates.map((d, i) => [d, values[i]]);
    }

    // /!\ Respect du type Plotly de chaque trace : go.Bar → 'bar' ECharts, sinon 'line'
    const isBar = trace.type === 'go.Bar';
    const seriesItem: Record<string, unknown> = isBar
      ? {
          type: 'bar',
          name: traceName,
          data: applyPerBarColoring(seriesData, trace, color),
          itemStyle: { color, opacity: trace.opacity ?? 0.85 },
          // secondary_y → yAxisIndex (cf. plus bas)
        }
      : {
          type: 'line',
          name: traceName,
          data: seriesData,
          itemStyle: { color },
          lineStyle: {
            color,
            width: trace.line?.width ?? 2,
            type: lineType,
          },
          symbol: trace.marker?.symbol ?? 'circle',
          symbolSize: trace.marker?.size ?? 6,
          showSymbol,
          smooth: false,
        };

    // secondary_y (axe Y2) — applicable aux 2 types
    if (trace.secondary_y === true) {
      seriesItem.yAxisIndex = 1;
    }

    // Fill (CI ribbon)
    if (trace.fill === 'tonexty' || trace.fill === 'tozeroy') {
      seriesItem.areaStyle = {
        color: (trace as { fillcolor?: string }).fillcolor ?? 'rgba(0,183,235,0.12)',
        opacity: 1,
      };
    }

    // visible: legendonly → masqué par défaut dans la légende
    const visibility = (trace as { visible?: string }).visible;
    if (visibility === 'legendonly') {
      legendSelected[traceName] = false;
    }

    // match_view.09 : annotations d'impact events sur la série kills ou deaths.
    // Levels GLOBAUX (pré-calculés sur les 2 courbes confondues) → pas de chevauchement entre courbes.
    if (spec.id === 'match_view.09' && Array.isArray(seriesData) && impactGlobalLevels) {
      const curveType = trace.id === 'kills_cum' ? 'kills' : trace.id === 'deaths_cum' ? 'deaths' : null;
      if (curveType) {
        const { markPoints, markLineData } = buildImpactAnnotations(
          curveType,
          seriesData as Array<[number, number]>,
          600_000,
          impactGlobalLevels,
        );
        if (markPoints.length > 0) {
          seriesItem.markPoint = { silent: true, data: markPoints };
        }
        if (markLineData.length > 0) {
          seriesItem.markLine = {
            silent: true,
            symbol: ['none', 'none'],
            label: { show: false },
            data: markLineData,
          };
        }
      }
    }

    series.push(seriesItem);
  }

  // markArea (hrect) — zones de tier pour LUSR
  if (spec.layout.shapes && spec.layout.shapes.length > 0 && series.length > 0) {
    for (const shape of spec.layout.shapes) {
      if (shape.kind === 'hline' && typeof shape.y === 'number') {
        // hline → markLine sur la 1ère série
        const sLine = (series[0] as Record<string, unknown>).markLine as
          | Record<string, unknown>
          | undefined;
        const data = (sLine?.data as Array<Record<string, unknown>>) ?? [];
        data.push({
          yAxis: shape.y,
          lineStyle: {
            color: shape.line_color,
            width: shape.line_width ?? 1,
            type: 'dotted',
            opacity: shape.opacity ?? 1,
          },
          label: {
            show: !!shape.annotation,
            formatter: shape.annotation
              ? resolveI18nToken((shape.annotation as { text?: string }).text ?? '') ?? ''
              : '',
            position: 'insideStartTop',
            color: 'rgba(255, 215, 0, 0.5)',
            fontSize: 10,
          },
          symbol: 'none',
        });
        (series[0] as Record<string, unknown>).markLine = {
          silent: true,
          symbol: 'none',
          data,
        };
      }
    }
  }

  // Build legend — appliquer selected si visible:legendonly
  const legend = buildLegend(spec, theme);
  if (legend && typeof legend === 'object' && Object.keys(legendSelected).length > 0) {
    (legend as Record<string, unknown>).selected = legendSelected;
  }

  const xAxis = buildAxis(spec.layout.xaxis, theme);
  // Pour les charts time-series, X est de type 'time' ou 'category'
  if (spec.layout.xaxis?.type === 'date') {
    (xAxis as Record<string, unknown>).type = 'time';
  } else if (spec.id === 'career.04') {
    // LUSR utilise apply_chrono_xaxis qui rend une catégorie chronologique
    (xAxis as Record<string, unknown>).type = 'category';
    (xAxis as Record<string, unknown>).data = dates;
  } else if (spec.id === 'teammates.04') {
    // Squad timeline : axe X catégoriel (sessions)
    (xAxis as Record<string, unknown>).type = 'category';
    (xAxis as Record<string, unknown>).data = ['Sess 1', 'Sess 2', 'Sess 3', 'Sess 4', 'Sess 5', 'Sess 6', 'Sess 7', 'Sess 8'];
  } else if (spec.id === 'teammates.05') {
    // Form score : axe X temporel (dates)
    (xAxis as Record<string, unknown>).type = 'time';
  } else if (spec.id === 'teammates.08') {
    // Metric bars multi-joueurs : axe X catégoriel (numéros de matchs)
    (xAxis as Record<string, unknown>).type = 'category';
    (xAxis as Record<string, unknown>).data = Array.from({ length: 12 }, (_, i) => `#${i + 1}`);
  } else if (
    spec.id === 'match_view.09' ||
    spec.id === 'match_view.10' ||
    spec.id === 'match_view.13'
  ) {
    // Match clock en ms — axe linéaire avec formatter mm:ss
    (xAxis as Record<string, unknown>).type = 'value';
    (xAxis as Record<string, unknown>).min = 0;
    (xAxis as Record<string, unknown>).max = 600_000; // 10 min de match par défaut
    (xAxis as Record<string, unknown>).axisLabel = {
      formatter: new Function(
        'val',
        `var m = Math.floor(val / 60000); var s = Math.floor((val % 60000) / 1000); return m + ':' + (s < 10 ? '0' + s : s);`,
      ),
    };
  }
  const yAxis = buildAxis(spec.layout.yaxis, theme);

  // Pour LUSR, ajouter le rendu des "tier bands" en arrière-plan via markArea
  // (simplification : on n'implémente que la trame, pas les bandes par tier)
  if (spec.id === 'career.04' && series.length > 0) {
    // Les bandes de tier seraient idéalement des séries dédiées de type 'line' avec
    // areaStyle, mais ECharts les supporte via markArea sur la 1ère série.
    // Implémentation minimaliste pour la preview.
    const tierBands: Array<[Record<string, unknown>, Record<string, unknown>]> = [
      [
        { yAxis: 0, itemStyle: { color: 'rgba(205,127,50,0.08)' } },
        { yAxis: 800 },
      ],
      [
        { yAxis: 800, itemStyle: { color: 'rgba(192,192,192,0.08)' } },
        { yAxis: 1200 },
      ],
      [
        { yAxis: 1200, itemStyle: { color: 'rgba(255,215,0,0.10)' } },
        { yAxis: 1600 },
      ],
      [
        { yAxis: 1600, itemStyle: { color: 'rgba(0,206,209,0.08)' } },
        { yAxis: 2000 },
      ],
      [
        { yAxis: 2000, itemStyle: { color: 'rgba(185,242,255,0.10)' } },
        { yAxis: 2200 },
      ],
    ];
    (series[0] as Record<string, unknown>).markArea = {
      silent: true,
      data: tierBands,
    };
  }

  // yAxis2 si une trace utilise secondary_y (cas teammates.04 MMR sur Y2)
  const hasY2 = spec.traces.some((t) => t.secondary_y === true);
  const finalYAxis: unknown = hasY2
    ? [
        yAxis,
        {
          type: 'value',
          name:
            spec.layout.yaxis2?.title
              ? resolveI18nToken(spec.layout.yaxis2.title) ?? ''
              : 'MMR',
          position: 'right',
          splitLine: { show: false },
          min: spec.layout.yaxis2?.range?.[0],
          max: spec.layout.yaxis2?.range?.[1],
        },
      ]
    : yAxis;

  const option: EChartsOption = {
    grid: buildGrid(spec),
    tooltip: buildTooltip(spec, theme),
    legend,
    xAxis,
    yAxis: finalYAxis as EChartsOption['yAxis'],
    series,
  };

  return applyThemeBase(option, theme);
}

// === Helpers mock ===

const XP_HERO_TOTAL = 9_319_350;
const CAREER_XP_LAUNCH = '2023-06-20';
const FIRST_SYNC_DATE = '2024-09-15'; // mock : date du 1er snapshot DB
const LAST_REAL_DATE = '2026-04-01'; // mock : dernier snapshot connu

function generateMockDates(count: number, dayStep: number): string[] {
  const dates: string[] = [];
  const start = new Date('2025-08-01');
  for (let i = 0; i < count; i++) {
    const d = new Date(start);
    d.setDate(d.getDate() + i * dayStep);
    dates.push(d.toISOString().slice(0, 10));
  }
  return dates;
}

/**
 * Génère un mock de timeline XP avec 3 segments distincts :
 *   - estimé pré-sync (CAREER_XP_LAUNCH → FIRST_SYNC_DATE) : courbe rétro-estimée
 *   - réel (FIRST_SYNC_DATE → LAST_REAL_DATE) : snapshots DB
 *   - projections (LAST_REAL_DATE → atteinte de 9.3M XP) : 2 courbes (normale + optimiste)
 *
 * Retourne (dates, values) selon le `traceId` demandé.
 */
function generateXpTimelineMock(traceId: string, baseXp = 0): {
  dates: string[];
  values: number[];
} {
  const dates: string[] = [];
  const values: number[] = [];

  if (traceId === 'estimated_xp' || traceId === 'friend_estimated') {
    // Segment estimé : du 20/06/2023 jusqu'au 1er sync (15/09/2024 ≈ 15 mois)
    let d = new Date(CAREER_XP_LAUNCH);
    let xp = 0;
    const targetXp = baseXp || 3_500_000; // first_sync_xp
    const endDate = new Date(FIRST_SYNC_DATE);
    const totalDays = (endDate.getTime() - d.getTime()) / 86_400_000;
    // Distribution non-linéaire : on simule une montée plus rapide au début (joueur actif au lancement)
    const N_POINTS = 25;
    for (let i = 0; i <= N_POINTS; i++) {
      const ratio = i / N_POINTS;
      // Easing pour simuler accélération puis stabilisation
      const eased = Math.pow(ratio, 1.3);
      const dayOffset = totalDays * ratio;
      const cur = new Date(d);
      cur.setDate(cur.getDate() + Math.round(dayOffset));
      xp = Math.round(targetXp * eased);
      dates.push(cur.toISOString().slice(0, 10));
      values.push(xp);
    }
    return { dates, values };
  }

  if (traceId === 'main_xp' || traceId === 'friend_real') {
    // Segment réel : du 1er sync (15/09/2024) au dernier (01/04/2026)
    let d = new Date(FIRST_SYNC_DATE);
    const endDate = new Date(LAST_REAL_DATE);
    const totalDays = (endDate.getTime() - d.getTime()) / 86_400_000;
    let xp = baseXp || 3_500_000;
    const N_POINTS = 18;
    for (let i = 0; i <= N_POINTS; i++) {
      const ratio = i / N_POINTS;
      const dayOffset = totalDays * ratio;
      const cur = new Date(d);
      cur.setDate(cur.getDate() + Math.round(dayOffset));
      xp = Math.round((baseXp || 3_500_000) + (4_500_000 - 0) * ratio + Math.random() * 50_000);
      dates.push(cur.toISOString().slice(0, 10));
      values.push(xp);
    }
    return { dates, values };
  }

  if (traceId === 'projection_normal' || traceId === 'friend_projection_normal') {
    // Projection normale : ~3000 XP/jour active rate (typique sans boost)
    const startXp = baseXp || 8_000_000;
    const startDate = new Date(LAST_REAL_DATE);
    const xpPerDay = 3_000;
    const remaining = XP_HERO_TOTAL - startXp;
    const daysToHero = Math.min(remaining / xpPerDay, 365 * 3);
    const N_POINTS = Math.min(Math.ceil(daysToHero / 30), 30);
    for (let i = 0; i <= N_POINTS; i++) {
      const ratio = i / N_POINTS;
      const dayOffset = daysToHero * ratio;
      const cur = new Date(startDate);
      cur.setDate(cur.getDate() + Math.round(dayOffset));
      dates.push(cur.toISOString().slice(0, 10));
      values.push(Math.min(startXp + xpPerDay * dayOffset, XP_HERO_TOTAL));
    }
    return { dates, values };
  }

  if (traceId === 'projection_optimistic' || traceId === 'friend_projection_optimistic') {
    // Projection optimiste : (rythme + 636) × 2 ≈ 7300 XP/jour
    const startXp = baseXp || 8_000_000;
    const startDate = new Date(LAST_REAL_DATE);
    const xpPerDay = 7_300;
    const remaining = XP_HERO_TOTAL - startXp;
    const daysToHero = Math.min(remaining / xpPerDay, 365 * 3);
    const N_POINTS = Math.min(Math.ceil(daysToHero / 30), 30);
    for (let i = 0; i <= N_POINTS; i++) {
      const ratio = i / N_POINTS;
      const dayOffset = daysToHero * ratio;
      const cur = new Date(startDate);
      cur.setDate(cur.getDate() + Math.round(dayOffset));
      dates.push(cur.toISOString().slice(0, 10));
      values.push(Math.min(startXp + xpPerDay * dayOffset, XP_HERO_TOTAL));
    }
    return { dates, values };
  }

  return { dates: [], values: [] };
}

function generateMockValues(_count: number, traceId: string, chartId: string): number[] {
  if (chartId === 'career.03') {
    // /!\ on retourne juste les VALUES — les DATES sont gérées séparément dans le seriesItem
    return generateXpTimelineMock(traceId).values;
  }
  if (chartId === 'career.04') {
    // LUSR : valeurs autour de 1500 avec variations
    const values: number[] = [];
    let r = 1400;
    for (let i = 0; i < _count; i++) {
      r += -30 + Math.random() * 60;
      values.push(Math.round(Math.max(800, Math.min(2400, r))));
    }
    return values;
  }
  const values: number[] = [];
  for (let i = 0; i < _count; i++) {
    values.push(Math.round(Math.random() * 100));
  }
  return values;
}

// =============================================================================
// Helpers mock pour Match View (timeline match_clock en ms)
// =============================================================================

/**
 * Génère des données pour les charts match_view.09 (KD cumul du joueur).
 * x = time_ms (0 à durationMs), y = compteur incrémental selon le type de trace.
 * NB : les "impact events" ne sont PAS une série mais des markPoints (cf. addImpactMarkPoints).
 */
function generateMatchClockMock(
  traceId: string,
  durationMs: number,
): Array<[number, number]> {
  const data: Array<[number, number]> = [];
  let nEvents: number;
  if (traceId === 'kills_cum') nEvents = 14;
  else if (traceId === 'deaths_cum') nEvents = 9;
  else nEvents = 10;

  let cumul = 0;
  for (let i = 0; i < nEvents; i++) {
    const ratio = (i + 1) / nEvents;
    const jitter = (Math.random() - 0.5) * 0.05;
    const t_ms = Math.max(1000, Math.round(durationMs * (ratio + jitter)));
    cumul++;
    data.push([t_ms, cumul]);
  }
  return data.sort((a, b) => a[0] - b[0]);
}

/**
 * Définition d'un impact event mock (correspondance avec impact_events.example_events du YAML 09).
 */
const IMPACT_EVENTS_MOCK = [
  { type: 'first_blood',     icon: '🩸', label: 'First Blood',     player: 'JGtm',         isMe: true,  curve: 'kills',  ratio: 0.05 },
  { type: 'top_gun',         icon: '🎯', label: 'Top Gun',         player: 'JGtm',         isMe: true,  curve: 'kills',  ratio: 0.95 },
  { type: 'clutch_finisher', icon: '⚡', label: 'Clutch Finisher', player: 'JGtm',         isMe: true,  curve: 'kills',  ratio: 0.78 },
  // 2 events proches MÊME courbe → level monte
  { type: 'multikill',       icon: '🔥', label: 'Multi Kill',      player: 'JGtm',         isMe: true,  curve: 'kills',  ratio: 0.82 },
  { type: 'tourist',         icon: '🧳', label: 'Touriste',        player: 'BotPlayer42',  isMe: false, curve: 'deaths', ratio: 0.10 },
  // 1 event proche d'un kill MAIS sur la courbe deaths → l'algo global doit lever ce death
  { type: 'first_group_death', icon: '💀', label: 'Premier mort équipe', player: 'BotPlayer99', isMe: false, curve: 'deaths', ratio: 0.80 },
  { type: 'last_casualty',   icon: '🪦', label: 'Last Casualty',   player: 'JGtm',         isMe: true,  curve: 'deaths', ratio: 0.92 },
];

/**
 * Construit les annotations d'impact events :
 *   - 2 markPoints par event (cercle ancrage + boîte étiquette flottante)
 *   - 1 markLine reliant les 2 (ligne fine de même couleur)
 *
 * Réplique du comportement Plotly add_annotation(showarrow=True, ay=-40/-90/-140).
 * Retourne {markPoints, markLineData} prêts à attacher sur la série.
 */
// 3 niveaux verticaux, threshold proximité (cf. plot_match_kill_death_timeline)
const IMPACT_PROXIMITY_THRESHOLD_MS = 75_000;
const IMPACT_Y_LEVELS = [3.5, 6.5, 9.5];

/**
 * Calcule les levels GLOBALEMENT pour TOUS les events (kills + deaths confondus).
 * Algo Plotly source : trie tous events par time_ms, puis pour chaque event
 * compte combien d'events précédents sont dans la fenêtre de proximité —
 * ça force la montée de niveau quel que soit la courbe d'ancrage.
 *
 * Retourne un Map<event.type, levelIdx>. Réutilisé par les 2 courbes pour
 * que les étiquettes ne se chevauchent JAMAIS, même entre courbes différentes.
 */
function computeGlobalImpactLevels(
  killsData: Array<[number, number]>,
  deathsData: Array<[number, number]>,
  durationMs: number,
): Map<string, number> {
  type EventWithPos = (typeof IMPACT_EVENTS_MOCK)[0] & {
    arrowX: number;
    levelIdx?: number;
  };
  const allPositioned: EventWithPos[] = IMPACT_EVENTS_MOCK.map((ev) => {
    const curveData = ev.curve === 'kills' ? killsData : deathsData;
    if (!curveData.length) return { ...ev, arrowX: Math.round(durationMs * ev.ratio) };
    const t_ms = Math.round(durationMs * ev.ratio);
    const firstT = curveData[0][0];
    const lastT = curveData[curveData.length - 1][0];
    const arrowX = Math.max(firstT, Math.min(t_ms, lastT));
    return { ...ev, arrowX };
  });

  // Tri global par arrowX (TOUS events confondus, peu importe la courbe)
  allPositioned.sort((a, b) => a.arrowX - b.arrowX);

  for (let i = 0; i < allPositioned.length; i++) {
    const ev = allPositioned[i];
    let levelIdx = 0;
    for (let j = 0; j < i; j++) {
      const prev = allPositioned[j];
      if (
        prev.levelIdx !== undefined &&
        Math.abs(ev.arrowX - prev.arrowX) < IMPACT_PROXIMITY_THRESHOLD_MS
      ) {
        levelIdx = Math.max(levelIdx, prev.levelIdx + 1);
      }
    }
    levelIdx = levelIdx % IMPACT_Y_LEVELS.length;
    ev.levelIdx = levelIdx;
  }

  const map = new Map<string, number>();
  for (const ev of allPositioned) {
    map.set(ev.type, ev.levelIdx ?? 0);
  }
  return map;
}

function buildImpactAnnotations(
  curve: 'kills' | 'deaths',
  curveData: Array<[number, number]>,
  durationMs: number,
  globalLevels: Map<string, number>,
): {
  markPoints: Array<Record<string, unknown>>;
  markLineData: Array<Array<Record<string, unknown>>>;
} {
  if (!curveData.length) return { markPoints: [], markLineData: [] };
  const events = IMPACT_EVENTS_MOCK.filter((e) => e.curve === curve);
  const markPoints: Array<Record<string, unknown>> = [];
  const markLineData: Array<Array<Record<string, unknown>>> = [];

  for (const ev of events) {
    const t_ms = Math.round(durationMs * ev.ratio);
    const firstT = curveData[0][0];
    const lastT = curveData[curveData.length - 1][0];
    const arrowX = Math.max(firstT, Math.min(t_ms, lastT));
    const yVal = curveData.filter((p) => p[0] <= arrowX).length;

    const lineColor = ev.isMe ? '#3DFFB5' : '#E69F00';
    const bgColor = ev.isMe ? 'rgba(61, 255, 181, 0.92)' : 'rgba(255, 183, 3, 0.92)';
    const levelIdx = globalLevels.get(ev.type) ?? 0;
    const yOffset = IMPACT_Y_LEVELS[levelIdx];
    const labelY = yVal + yOffset;

    // 1. Cercle ancrage sur la courbe
    markPoints.push({
      coord: [arrowX, yVal],
      symbol: 'circle',
      symbolSize: 9,
      itemStyle: { color: lineColor, borderColor: '#fff', borderWidth: 1.5 },
      label: { show: false },
    });

    // 2. Étiquette flottante (boîte arrondie avec label centré)
    markPoints.push({
      coord: [arrowX, labelY],
      symbol: 'roundRect',
      symbolSize: [120, 32],
      itemStyle: { color: bgColor, borderColor: lineColor, borderWidth: 1.5 },
      label: {
        show: true,
        formatter: `${ev.icon} ${ev.label}\n${ev.player}`,
        color: '#0a0e14',
        fontSize: 10,
        fontWeight: 'bold',
        align: 'center',
        verticalAlign: 'middle',
        lineHeight: 13,
      },
    });

    // 3. Ligne reliant ancrage et étiquette
    markLineData.push([
      {
        coord: [arrowX, yVal],
        lineStyle: { color: lineColor, width: 1.5, type: 'solid', opacity: 0.9 },
      },
      { coord: [arrowX, labelY] },
    ]);
  }
  return { markPoints, markLineData };
}

/**
 * Mock pour match_view.10 (dominance d'équipe par bucket de 30s).
 * Génère 20 buckets sur 10 min avec valeurs alternant entre les 2 équipes.
 */
function generateDominanceMock(
  traceId: string,
  durationMs: number,
): Array<[number, number]> {
  const data: Array<[number, number]> = [];
  const bucketMs = 30_000;
  const nBuckets = Math.floor(durationMs / bucketMs);
  for (let i = 0; i < nBuckets; i++) {
    const t_ms = i * bucketMs + bucketMs / 2; // centre du bucket
    let pct: number;
    if (traceId.includes('my_team') || traceId === 'my_team_dominance_bars') {
      // Mon équipe domine plutôt en milieu de match
      const base = 50 + 20 * Math.sin((i / nBuckets) * Math.PI);
      pct = Math.max(10, Math.min(90, base + (Math.random() - 0.5) * 15));
    } else if (traceId.includes('enemy') || traceId === 'enemy_dominance_bars') {
      // Équipe adverse domine début + fin
      const base = 50 - 20 * Math.sin((i / nBuckets) * Math.PI);
      pct = Math.max(10, Math.min(90, base + (Math.random() - 0.5) * 15));
    } else {
      // points kill feed (my_kills_points / enemy_kills_points / kill_feed_per_player)
      pct = 50 + (Math.random() - 0.5) * 10;
    }
    data.push([t_ms, Math.round(pct)]);
  }
  return data;
}

/**
 * Mock pour match_view.13 (frags différentiel cumulé tous joueurs).
 * Génère N joueurs avec des courbes K-D différentielles distinctes.
 */
function generateMockPlayers(nPlayers: number): Array<{
  name: string;
  color: string;
  isMain: boolean;
  data: Array<[number, number]>;
}> {
  const colors = [
    '#33D6FF', // joueur principal — cyan
    '#EF5350',
    '#29B6F6',
    '#FFCA28',
    '#26C6DA',
    '#FF7043',
    '#AB47BC',
    '#66BB6A',
  ];
  const names = [
    'JGtm', // joueur principal
    'NeoSpartan_42',
    'BlazingFury',
    'ShadowKnight',
    'GlitchHunter',
    'VortexMaster',
    'NightHawk',
    'CrimsonRage',
  ];
  const players: Array<{
    name: string;
    color: string;
    isMain: boolean;
    data: Array<[number, number]>;
  }> = [];

  const durationMs = 600_000;
  const nPoints = 30;
  for (let p = 0; p < nPlayers; p++) {
    const isMain = p === 0;
    const data: Array<[number, number]> = [];
    let kd = 0;
    // Tendance globale par joueur (positif ou négatif)
    const trend = isMain ? 0.6 : (Math.random() - 0.4) * 1.2;
    for (let i = 0; i <= nPoints; i++) {
      const t_ms = Math.round((i / nPoints) * durationMs);
      // Marche aléatoire avec biais
      kd += (Math.random() < 0.5 + trend * 0.15 ? 1 : -1) * (Math.random() < 0.7 ? 1 : 0);
      data.push([t_ms, kd]);
    }
    players.push({
      name: names[p] ?? `Player ${p + 1}`,
      color: colors[p] ?? '#888888',
      isMain,
      data,
    });
  }
  return players;
}

/**
 * Génère les 4 sous-traces ECharts pour un ami (réelle + estimée + 2 projections).
 * Toutes en `legendOnly` (visibles dans la légende mais masquées par défaut).
 */
function generateFriendSubTraces(
  friend: { gamertag: string; color: string },
  _datesUnused: string[],
  chartId: string,
): Array<{ series: Record<string, unknown>; name: string; legendOnly: boolean }> {
  if (chartId !== 'career.03') return [];

  // Mock : XP de cet ami différent du joueur courant pour visualiser la diversité
  const friendBaseXp = 2_000_000 + Math.random() * 4_000_000;
  const result: Array<{ series: Record<string, unknown>; name: string; legendOnly: boolean }> = [];

  // 1. Trace réelle (lines+markers, 1.5px, solid)
  const real = generateXpTimelineMock('friend_real', friendBaseXp);
  const realName = `XP ${friend.gamertag}`;
  result.push({
    series: {
      type: 'line',
      name: realName,
      data: real.dates.map((d, i) => [d, real.values[i]]),
      itemStyle: { color: friend.color },
      lineStyle: { color: friend.color, width: 1.5, type: 'solid' },
      symbol: 'circle',
      symbolSize: 5,
      showSymbol: true,
      smooth: false,
    },
    name: realName,
    legendOnly: true,
  });

  // 2. Trace estimée (lines, 1.5px, dot)
  const est = generateXpTimelineMock('friend_estimated', friendBaseXp);
  const estName = `XP estimé ${friend.gamertag}`;
  result.push({
    series: {
      type: 'line',
      name: estName,
      data: est.dates.map((d, i) => [d, est.values[i]]),
      itemStyle: { color: friend.color },
      lineStyle: { color: friend.color, width: 1.5, type: 'dotted' },
      symbol: 'none',
      showSymbol: false,
      smooth: false,
    },
    name: estName,
    legendOnly: true,
  });

  // 3. Projection normale (lines, 1.5px, dash)
  const lastRealXp = real.values[real.values.length - 1];
  const projN = generateXpTimelineMock('friend_projection_normal', lastRealXp);
  const projNName = `Projection ${friend.gamertag}`;
  result.push({
    series: {
      type: 'line',
      name: projNName,
      data: projN.dates.map((d, i) => [d, projN.values[i]]),
      itemStyle: { color: friend.color },
      lineStyle: { color: friend.color, width: 1.5, type: 'dashed' },
      symbol: 'none',
      showSymbol: false,
      smooth: false,
    },
    name: projNName,
    legendOnly: true,
  });

  // 4. Projection optimiste (lines, 1.5px, dashdot — ECharts: dashed)
  const projO = generateXpTimelineMock('friend_projection_optimistic', lastRealXp);
  const projOName = `Projection optimiste ${friend.gamertag}`;
  result.push({
    series: {
      type: 'line',
      name: projOName,
      data: projO.dates.map((d, i) => [d, projO.values[i]]),
      itemStyle: { color: friend.color },
      lineStyle: { color: friend.color, width: 1.5, type: 'dashed' },
      symbol: 'none',
      showSymbol: false,
      smooth: false,
    },
    name: projOName,
    legendOnly: true,
  });

  return result;
}


/**
 * Per-bar coloring pour les bars Plotly avec marker_color = list[str].
 * Cas teammates.04 squad_perf_bars : couleur selon SCORE_THRESHOLDS (perf_value).
 */
function applyPerBarColoring(
  seriesData: Array<[string | number, number]>,
  trace: import('../types.js').TraceSpec,
  defaultColor: string,
): Array<{ value: [string | number, number]; itemStyle: Record<string, unknown> }> {
  const colorPerBarFn = (val: number): string => {
    if (trace.id === 'squad_perf_bars') {
      // _bar_color(v) : palette excellent→good→average→below→poor
      if (val >= 80) return '#50C878'; // excellent (vert)
      if (val >= 65) return '#00B7EB'; // good (cyan)
      if (val >= 50) return '#FFBF00'; // average (ambre)
      if (val >= 30) return '#FF8C00'; // below (orange)
      return '#FF4444'; // poor (rouge)
    }
    return defaultColor;
  };
  return seriesData.map((d) => ({
    value: d,
    itemStyle: { color: colorPerBarFn(d[1]) },
  }));
}

// =============================================================================
// Helpers mock pour Teammates
// =============================================================================

const SQUAD_SESSIONS = ["Sess 1", "Sess 2", "Sess 3", "Sess 4", "Sess 5", "Sess 6", "Sess 7", "Sess 8"];
const SQUAD_PLAYERS = ["JGtm", "NeoSpartan_42", "BlazingFury", "ShadowKnight"];
const SQUAD_PLAYER_COLORS = ["#33D6FF", "#EF5350", "#FFCA28", "#26C6DA"];

/**
 * Mock teammates.04 — squad timeline (sessions × perf|winrate|mmr).
 */
function generateSquadTimelineMock(
  traceId: string,
  _traceIdx: number,
): Array<[string | number, number]> {
  const result: Array<[string | number, number]> = [];
  if (traceId === "squad_perf_bars") {
    // perf 0-100 par session
    SQUAD_SESSIONS.forEach((s, i) => {
      const perf = Math.round(50 + 25 * Math.sin((i / 8) * Math.PI) + (Math.random() - 0.5) * 18);
      result.push([s, Math.max(20, Math.min(95, perf))]);
    });
  } else if (traceId === "winrate_line") {
    SQUAD_SESSIONS.forEach((s, i) => {
      const wr = Math.round(45 + 20 * Math.sin((i / 8) * Math.PI) + (Math.random() - 0.5) * 14);
      result.push([s, Math.max(20, Math.min(85, wr))]);
    });
  } else if (traceId === "team_mmr_line") {
    SQUAD_SESSIONS.forEach((s, i) => {
      const mmr = Math.round(1500 + 100 * Math.sin((i / 8) * Math.PI) + (Math.random() - 0.5) * 80);
      result.push([s, mmr]);
    });
  }
  return result;
}

/**
 * Mock teammates.05 — form score history (1 ligne par membre).
 * traceIdx 0/1 = fill traces (single mode), 2+ = player lines.
 */
function generateFormScoreMock(
  traceId: string,
  traceIdx: number,
): Array<[string | number, number]> {
  // Indices 0+ = lignes de joueurs (1 trace par membre escouade)
  const playerIdx = Math.max(0, Math.min(SQUAD_PLAYERS.length - 1, traceIdx));
  const N_POINTS = 25;
  const result: Array<[string | number, number]> = [];
  const start = new Date("2026-03-01");
  // Form score oscille autour de 0 avec une tendance par joueur
  const trend = (playerIdx === 0 ? 1 : -1) * (Math.random() * 0.3 + 0.1);
  let val = 0;
  for (let i = 0; i < N_POINTS; i++) {
    val += trend + (Math.random() - 0.5) * 1.5;
    val = Math.max(-5, Math.min(5, val));
    const date = new Date(start);
    date.setDate(date.getDate() + i * 2);
    result.push([date.toISOString().slice(0, 10), +val.toFixed(2)]);
  }
  return result;
}

// =============================================================================
// teammates.16 — Trio performance (KD combined butterfly time-series)
//
// 1 sub-chart représentatif : kills↑ (couleur joueur) / morts↓ (couleur _negative).
// Les 5 autres sub-charts (assists, KDA, accuracy, avg_life, performance) sont
// documentés dans le YAML mais non rendus individuellement (1 chart suffit pour
// la fidélité visuelle de la famille).
// =============================================================================

function convertTrioKdCombined(spec: ChartSpec, theme: ThemeDefault): EChartsOption {
  const N_MATCHES = 18;
  const xCats = Array.from({ length: N_MATCHES }, (_, i) => `m${i + 1}`);

  const players = [
    { name: 'JGtm', color: '#56B4E9', neg: '#ff6666' },
    { name: 'NeoSpartan_42', color: '#E69F00', neg: '#e63333' },
    { name: 'BlazingFury', color: '#009E73', neg: '#b31c1c' },
    { name: 'ShadowKnight', color: '#CC79A7', neg: '#7a1111' },
  ];

  const smooth = (arr: number[]): number[] => {
    const w = 7;
    return arr.map((_, i) => {
      const start = Math.max(0, i - Math.floor(w / 2));
      const end = Math.min(arr.length, i + Math.ceil(w / 2));
      const slice = arr.slice(start, end);
      return slice.reduce((a, b) => a + b, 0) / slice.length;
    });
  };

  const series: Array<Record<string, unknown>> = [];
  players.forEach((p, idx) => {
    const kills = Array.from({ length: N_MATCHES }, (_, i) =>
      Math.max(0, Math.round(8 + 4 * Math.sin(i / 2 + idx) + (Math.random() - 0.5) * 3)),
    );
    const deaths = Array.from({ length: N_MATCHES }, (_, i) =>
      Math.max(0, Math.round(7 + 3 * Math.cos(i / 2.3 + idx * 0.7) + (Math.random() - 0.5) * 2)),
    );
    const killsSmooth = smooth(kills);
    const deathsSmoothNeg = smooth(deaths).map((v) => -v);

    series.push({
      name: p.name,
      type: 'bar',
      stack: `stack_${idx}`,
      data: kills,
      itemStyle: { color: p.color, opacity: 0.7 },
      barCategoryGap: '10%',
    });
    series.push({
      name: p.name + ' (morts)',
      type: 'bar',
      stack: `stack_${idx}`,
      data: deaths.map((v) => -v),
      itemStyle: { color: p.neg, opacity: 0.7 },
      barCategoryGap: '10%',
    });
    series.push({
      name: p.name + ' (kills lissé)',
      type: 'line',
      data: killsSmooth.map((v) => parseFloat(v.toFixed(2))),
      itemStyle: { color: p.color },
      lineStyle: { color: p.color, width: 2 },
      symbol: 'circle',
      symbolSize: 4,
      smooth: true,
    });
    series.push({
      name: p.name + ' (morts lissé)',
      type: 'line',
      data: deathsSmoothNeg.map((v) => parseFloat(v.toFixed(2))),
      itemStyle: { color: p.neg },
      lineStyle: { color: p.neg, width: 2 },
      symbol: 'circle',
      symbolSize: 4,
      smooth: true,
    });
  });

  const option: EChartsOption = {
    backgroundColor: 'transparent',
    textStyle: { color: theme.font.color, fontSize: theme.font.size },
    grid: { left: 60, right: 30, top: 60, bottom: 60, containLabel: false },
    legend: false,
    tooltip: { trigger: 'axis', axisPointer: { type: 'shadow' } },
    xAxis: {
      type: 'category',
      data: xCats,
      axisLabel: { color: 'rgba(245, 248, 255, 0.85)', fontSize: 10 },
    },
    yAxis: {
      type: 'value',
      axisLine: { lineStyle: { color: 'rgba(255,255,255,0.75)', width: 2 } },
      axisLabel: {
        color: 'rgba(245, 248, 255, 0.85)',
        formatter: new Function('v', 'return Math.abs(v);') as unknown as string,
      },
      splitLine: { lineStyle: { color: 'rgba(245, 248, 255, 0.08)' } },
    },
    series,
    title: {
      text: 'Kills (haut) / Morts (bas) — escouade lissés (rolling 7)',
      left: 'center',
      top: 8,
      textStyle: { color: theme.font.color, fontSize: 13 },
    },
  };
  // @ts-expect-error - height meta
  option.__height = 420;
  return applyThemeBase(option, theme);
}

// =============================================================================
// timeseries.01 — Form score solo (joueur principal) avec highlight session
// =============================================================================

function convertFormScoreSelf(spec: ChartSpec, theme: ThemeDefault): EChartsOption {
  const N = 30;
  const buckets = Array.from({ length: N }, (_, i) => `S${i + 1}`);

  // Form score : oscille autour de 0, avec une session courante mise en avant
  const sessionStartIdx = N - 5;
  const data: number[] = [];
  for (let i = 0; i < N; i++) {
    const base = 5 * Math.sin(i / 4) + 3 * Math.cos(i / 7);
    const noise = (Math.random() - 0.5) * 6;
    let v = base + noise;
    if (i >= sessionStartIdx) v += 8; // session récente meilleure
    data.push(parseFloat(v.toFixed(1)));
  }
  const dataPos = data.map((v) => (v >= 0 ? v : 0));
  const dataNeg = data.map((v) => (v < 0 ? v : 0));

  const baseline = data.slice(0, sessionStartIdx).reduce((a, b) => a + b, 0) / sessionStartIdx;
  const current = data.slice(sessionStartIdx).reduce((a, b) => a + b, 0) / (N - sessionStartIdx);
  const sessionColor = current >= baseline ? 'rgba(0,200,100,0.18)' : 'rgba(255,80,80,0.18)';

  const option: EChartsOption = {
    backgroundColor: 'transparent',
    textStyle: { color: theme.font.color, fontSize: theme.font.size },
    grid: { left: 50, right: 30, top: 50, bottom: 60, containLabel: false },
    legend: { bottom: 8, textStyle: { color: 'rgba(245,248,255,0.85)' } },
    tooltip: { trigger: 'axis' },
    xAxis: {
      type: 'category',
      data: buckets,
      axisLabel: { color: 'rgba(245,248,255,0.7)', fontSize: 9, interval: 2 },
    },
    yAxis: {
      type: 'value',
      axisLine: { lineStyle: { color: 'rgba(255,255,255,0.5)', width: 1 } },
      axisLabel: { color: 'rgba(245,248,255,0.7)' },
      splitLine: { lineStyle: { color: 'rgba(245,248,255,0.05)' } },
    },
    series: [
      {
        name: 'Form score (positif)',
        type: 'line',
        data: dataPos,
        smooth: true,
        symbol: 'none',
        lineStyle: { color: '#00e676', width: 2 },
        areaStyle: { color: 'rgba(0,200,100,0.25)', origin: 'start' },
      },
      {
        name: 'Form score (négatif)',
        type: 'line',
        data: dataNeg,
        smooth: true,
        symbol: 'none',
        lineStyle: { color: '#e53935', width: 2 },
        areaStyle: { color: 'rgba(213,94,0,0.25)', origin: 'start' },
      },
      {
        name: 'Form score',
        type: 'line',
        data: data,
        smooth: true,
        symbol: 'circle',
        symbolSize: 5,
        lineStyle: { color: '#41d6ff', width: 2 },
        markArea: {
          silent: true,
          itemStyle: { color: sessionColor },
          data: [[{ xAxis: buckets[sessionStartIdx], name: 'Session courante' }, { xAxis: buckets[N - 1] }]],
        },
        markLine: {
          symbol: 'none',
          silent: true,
          data: [{ yAxis: 0 }],
          lineStyle: { color: 'rgba(255,255,255,0.4)', type: 'dashed', width: 1 },
          label: { show: false },
        },
      },
    ],
    title: {
      text: 'Form score — Session vs Historique (zone colorée = session courante)',
      left: 'center',
      top: 8,
      textStyle: { color: theme.font.color, fontSize: 13 },
    },
  };
  // @ts-expect-error - height meta
  option.__height = 380;
  return applyThemeBase(option, theme);
}

// =============================================================================
// win_loss.06 — Personal score bars amber + smoothing rolling 10 (line violet)
// =============================================================================

function convertPersonalScoreMock(spec: ChartSpec, theme: ThemeDefault): EChartsOption {
  const N = 30;
  const xCats = Array.from({ length: N }, (_, i) => `m${i + 1}`);
  const scores = Array.from({ length: N }, () => Math.floor(2500 + Math.random() * 4500));
  // Smoothing rolling 10
  const smoothed = scores.map((_, i) => {
    const start = Math.max(0, i - 5);
    const end = Math.min(N, i + 5);
    const slice = scores.slice(start, end);
    return slice.reduce((a, b) => a + b, 0) / slice.length;
  });

  const option: EChartsOption = {
    backgroundColor: 'transparent',
    textStyle: { color: theme.font.color, fontSize: theme.font.size },
    grid: { left: 60, right: 30, top: 50, bottom: 70, containLabel: false },
    legend: { bottom: 8, textStyle: { color: 'rgba(245,248,255,0.85)' } },
    tooltip: { trigger: 'axis', axisPointer: { type: 'shadow' } },
    xAxis: {
      type: 'category',
      data: xCats,
      axisLabel: { color: 'rgba(245,248,255,0.7)', fontSize: 9, interval: 2 },
    },
    yAxis: {
      type: 'value',
      axisLabel: { color: 'rgba(245,248,255,0.85)' },
      splitLine: { lineStyle: { color: 'rgba(245,248,255,0.05)' } },
    },
    series: [
      {
        name: 'Personal score',
        type: 'bar',
        data: scores,
        itemStyle: { color: '#ffb300', opacity: 0.85 },
        barCategoryGap: '20%',
      },
      {
        name: 'Smoothing (rolling 10)',
        type: 'line',
        data: smoothed.map((v) => parseFloat(v.toFixed(0))),
        smooth: true,
        symbol: 'circle',
        symbolSize: 5,
        lineStyle: { color: '#ab47bc', width: 2 },
      },
    ],
    title: {
      text: 'Personal score par match — bars amber + smoothing rolling 10 (line violet)',
      left: 'center',
      top: 8,
      textStyle: { color: theme.font.color, fontSize: 13 },
    },
  };
  // @ts-expect-error - height meta
  option.__height = 380;
  return applyThemeBase(option, theme);
}

// =============================================================================
// session_compare.10 — KD progression A vs B + accuracy Y2
// =============================================================================

function convertSessionCompareKD(spec: ChartSpec, theme: ThemeDefault): EChartsOption {
  const N = 12;
  const xCats = Array.from({ length: N }, (_, i) => `${i + 1}`);

  const kdA = [1.4, 1.8, 1.2, 2.1, 1.5, 1.9, 1.3, 2.4, 1.7, 1.6, 2.0, 1.8];
  const kdB = [0.9, 1.1, 0.8, 1.3, 1.0, 1.2, 0.7, 1.4, 1.1, 0.9, 1.0, 1.2];
  const accA = [55, 62, 50, 68, 58, 65, 53, 70, 60, 57, 64, 62];
  const accB = [48, 52, 45, 55, 50, 53, 47, 56, 51, 49, 52, 54];

  const option: EChartsOption = {
    backgroundColor: 'transparent',
    textStyle: { color: '#E0E0E0', fontSize: theme.font.size },
    grid: { left: 60, right: 60, top: 30, bottom: 80, containLabel: false },
    legend: { bottom: 10, textStyle: { color: '#E0E0E0' } },
    tooltip: { trigger: 'axis', axisPointer: { type: 'cross' } },
    xAxis: {
      type: 'category',
      data: xCats,
      name: 'Match',
      nameLocation: 'middle',
      nameGap: 30,
      nameTextStyle: { color: '#E0E0E0' },
      axisLabel: { color: '#E0E0E0' },
    },
    yAxis: [
      {
        type: 'value',
        name: 'F/D',
        nameTextStyle: { color: '#E0E0E0' },
        axisLabel: { color: '#E0E0E0' },
        splitLine: { lineStyle: { color: 'rgba(255,255,255,0.08)' } },
      },
      {
        type: 'value',
        name: 'Précision (%)',
        min: 0,
        max: 100,
        nameTextStyle: { color: '#E0E0E0' },
        axisLabel: { color: '#E0E0E0', formatter: '{value}%' },
        splitLine: { show: false },
      },
    ],
    series: [
      { name: 'Session A — F/D', type: 'line', data: kdA, lineStyle: { color: '#E74C3C', width: 2 }, itemStyle: { color: '#E74C3C' }, symbol: 'circle', symbolSize: 7,
        markLine: { symbol: 'none', silent: true, data: [{ yAxis: 1.0 }], lineStyle: { type: 'dotted', color: 'rgba(255,255,255,0.25)' }, label: { show: true, formatter: '1.0', position: 'end', color: 'rgba(255,255,255,0.4)' } } },
      { name: 'Session B — F/D', type: 'line', data: kdB, lineStyle: { color: '#3498DB', width: 2 }, itemStyle: { color: '#3498DB' }, symbol: 'circle', symbolSize: 7 },
      { name: 'Session A — Précision', type: 'line', yAxisIndex: 1, data: accA, lineStyle: { color: '#E74C3C', width: 1.5, type: 'dashed' }, itemStyle: { color: '#E74C3C' }, symbol: 'emptyCircle', symbolSize: 5, opacity: 0.75 },
      { name: 'Session B — Précision', type: 'line', yAxisIndex: 1, data: accB, lineStyle: { color: '#3498DB', width: 1.5, type: 'dashed' }, itemStyle: { color: '#3498DB' }, symbol: 'emptyCircle', symbolSize: 5, opacity: 0.75 },
    ],
    title: { text: 'K/D progression A vs B + Précision (Y2 dashed)', left: 'center', top: 8, textStyle: { color: theme.font.color, fontSize: 13 } },
  };
  // @ts-expect-error - height meta
  option.__height = 360;
  return applyThemeBase(option, theme);
}

// =============================================================================
// session_compare.09 — Cumulative net score A vs B + hline 0
// =============================================================================

function convertSessionCompareCumulative(spec: ChartSpec, theme: ThemeDefault): EChartsOption {
  const N = 14;
  const xCats = Array.from({ length: N }, (_, i) => `${i + 1}`);
  const cumulA = [3, 7, 4, 11, 9, 14, 17, 13, 18, 22, 20, 25, 28, 30];
  const cumulB = [-2, -4, -1, 1, -3, -6, -2, 0, -2, 1, -3, -1, 2, 0];

  const option: EChartsOption = {
    backgroundColor: 'transparent',
    textStyle: { color: '#E0E0E0', fontSize: theme.font.size },
    grid: { left: 60, right: 30, top: 40, bottom: 80, containLabel: false },
    legend: { bottom: 10, textStyle: { color: '#E0E0E0' } },
    tooltip: { trigger: 'axis' },
    xAxis: {
      type: 'category',
      data: xCats,
      name: 'Match',
      nameLocation: 'middle',
      nameGap: 30,
      nameTextStyle: { color: '#E0E0E0' },
      axisLabel: { color: '#E0E0E0' },
    },
    yAxis: {
      type: 'value',
      name: 'Net score cumulé',
      nameTextStyle: { color: '#E0E0E0' },
      axisLabel: { color: '#E0E0E0' },
      splitLine: { lineStyle: { color: 'rgba(255,255,255,0.08)' } },
    },
    series: [
      {
        name: 'Session A',
        type: 'line',
        data: cumulA,
        smooth: true,
        symbol: 'circle',
        symbolSize: 6,
        lineStyle: { color: '#E74C3C', width: 2 },
        itemStyle: { color: '#E74C3C' },
        markLine: {
          symbol: 'none',
          silent: true,
          data: [{ yAxis: 0 }],
          lineStyle: { color: 'rgba(255,255,255,0.75)', width: 2 },
          label: { show: true, formatter: 'Équilibre', position: 'end', color: 'rgba(255,255,255,0.6)' },
        },
      },
      {
        name: 'Session B',
        type: 'line',
        data: cumulB,
        smooth: true,
        symbol: 'circle',
        symbolSize: 6,
        lineStyle: { color: '#3498DB', width: 2 },
        itemStyle: { color: '#3498DB' },
      },
    ],
    title: { text: 'Net score cumulé — Session A vs Session B', left: 'center', top: 8, textStyle: { color: theme.font.color, fontSize: 13 } },
  };
  // @ts-expect-error - height meta
  option.__height = 380;
  return applyThemeBase(option, theme);
}

