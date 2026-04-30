import type { ChartSpec, EChartsOption, ThemeDefault } from '../types.js';

/**
 * Spec d'un composant UI composite — pas une option ECharts.
 * Le renderer HTML produit du HTML pur à partir de cette structure.
 */
export interface CompositeUiSpec {
  __kind: 'kpi_row' | 'composite_block';
  id: string;
  title: string;
  blocks: Array<KpiTileSpec | CompositeBlockTileSpec>;
  warnings: string[];
}

export interface KpiTileSpec {
  type: 'simple' | 'enriched';
  text?: string;
  scoreText?: string;
  scoreColor?: string;
  badge?: { label: string; bg: string; fg: string } | null;
}

export interface CompositeBlockTileSpec {
  type: 'image' | 'text_coloured' | 'raw_html';
  columnRatio: number;
  payload: string; // HTML/text
  color?: string;
}

/**
 * Convertit un YAML chart_kind=kpi_row en spec rendable.
 */
export function convertKpiRow(
  spec: ChartSpec,
  theme: ThemeDefault,
  mockCtx: Record<string, unknown>,
  warnings: string[],
): EChartsOption {
  // Dispatchs spécifiques (mocks fidèles)
  if (spec.id === 'timeseries.00' || spec.id === 'teammates.00') {
    return convertKpisSection(spec);
  }
  if (spec.id === 'teammates.01') {
    return convertSquadSessionHeader(spec);
  }
  if (spec.id === 'career.11') {
    return convertLusrRankCards(spec);
  }

  // @ts-expect-error - section kpi_row custom
  const kpi = spec.kpi_row as Record<string, unknown> | undefined;
  if (!kpi) {
    warnings.push('chart_kind=kpi_row mais section `kpi_row:` absente du YAML');
    return emptyComposite(spec, 'kpi_row', warnings);
  }

  const tiles = (kpi.tiles as Array<Record<string, unknown>>) ?? [];
  const blocks: KpiTileSpec[] = tiles.map((tile, idx) => {
    const tileType = (tile.type as string) ?? 'simple';

    // Mock data par tile selon l'index (cas du header_kpi_cards)
    const mockData: KpiTileSpec[] = [
      { type: 'simple', text: 'lun. 15 avr. 2026 — 21:32' },
      {
        type: 'enriched',
        scoreText: '50 — 32',
        scoreColor: '#4CAF50',
        badge: { label: 'DOMINATION', bg: '#2e7d32', fg: '#e8f5e9' },
      },
      { type: 'simple', text: 'Quickplay' },
      { type: 'simple', text: 'Slayer sur Aquarius' },
    ];

    if (idx < mockData.length) return mockData[idx];

    return {
      type: tileType as 'simple' | 'enriched',
      text: (tile.example as string) ?? '—',
    };
  });

  return {
    series: [],
    __meta: {
      spec_id: spec.id,
      chart_kind: 'kpi_row' as never,
      source_function: spec.source_function,
      warnings,
      height: 0,
    },
    // @ts-expect-error - extension non-standard
    __composite: {
      __kind: 'kpi_row',
      id: spec.id,
      title: spec.title,
      blocks,
      warnings,
    } as CompositeUiSpec,
  };
}

/**
 * Convertit un YAML chart_kind=composite_block en spec rendable.
 */
export function convertCompositeBlock(
  spec: ChartSpec,
  theme: ThemeDefault,
  mockCtx: Record<string, unknown>,
  warnings: string[],
): EChartsOption {
  // Cas spécifiques teammates : mocks dédiés
  if (spec.id === 'teammates.07') {
    return convertImpactTaquinerie(spec);
  }
  if (spec.id === 'teammates.12') {
    return convertTrioMedals(spec);
  }
  // Cas spécifique citations match_view (grille citations + médailles)
  if (spec.id === 'match_view.15') {
    return convertCitationsGrid(spec);
  }
  if (spec.id === 'match_view.16') {
    return convertMedalsGrid(spec);
  }

  // @ts-expect-error - section composite_block custom
  const cb = spec.composite_block as Record<string, unknown> | undefined;
  if (!cb) {
    warnings.push('chart_kind=composite_block mais section `composite_block:` absente du YAML');
    return emptyComposite(spec, 'composite_block', warnings);
  }

  const blocksRaw = (cb.blocks as Array<Record<string, unknown>>) ?? [];
  const blocks: CompositeBlockTileSpec[] = blocksRaw.map((b) => {
    const blockType = (b.type as string) ?? 'text_coloured';
    const ratio = (b.column_ratio as number) ?? 1;
    if (blockType === 'image') {
      // Mock : SVG inline représentant une miniature de carte
      return {
        type: 'image',
        columnRatio: ratio,
        payload: `<div style="width:100%;aspect-ratio:16/9;background:linear-gradient(135deg,#1a3a52 0%,#2d5a7e 50%,#1a3a52 100%);border-radius:6px;display:flex;align-items:center;justify-content:center;color:rgba(255,255,255,0.5);font-size:0.85em;">Map thumbnail (mock)</div>`,
      };
    }
    if (blockType === 'text_coloured') {
      return {
        type: 'text_coloured',
        columnRatio: ratio,
        payload: '78',
        color: '#3DFFB5',
      };
    }
    if (blockType === 'raw_html') {
      // Mock du rank HTML block
      return {
        type: 'raw_html',
        columnRatio: ratio,
        payload: `<div style="display:flex;align-items:center;gap:12px;padding:8px;background:rgba(255,255,255,0.03);border-radius:6px">
          <div style="width:64px;height:64px;background:radial-gradient(circle, #FFD700 0%, #c4a000 70%);border-radius:50%;display:flex;align-items:center;justify-content:center;color:#000;font-weight:700;font-size:0.85em">Onyx</div>
          <div style="flex:1">
            <div style="font-size:0.95em;font-weight:600">Onyx Diamond III</div>
            <div style="font-size:0.78em;color:#888;margin-top:2px"><span style="background:#00B7EB;color:#000;padding:1px 6px;border-radius:8px;font-size:0.92em;font-weight:600">LUSR</span> <span style="font-weight:700;font-size:1.1em">1842</span></div>
            <div style="font-size:0.85em;color:#3DFFB5;margin-top:2px">▲ +24</div>
          </div>
        </div>`,
      };
    }
    return { type: 'text_coloured', columnRatio: ratio, payload: '—' };
  });

  return {
    series: [],
    __meta: {
      spec_id: spec.id,
      chart_kind: 'composite_block' as never,
      source_function: spec.source_function,
      warnings,
      height: 0,
    },
    // @ts-expect-error - extension
    __composite: {
      __kind: 'composite_block',
      id: spec.id,
      title: spec.title,
      blocks,
      warnings,
    } as CompositeUiSpec,
  };
}

function emptyComposite(
  spec: ChartSpec,
  kind: 'kpi_row' | 'composite_block',
  warnings: string[],
): EChartsOption {
  return {
    series: [],
    __meta: {
      spec_id: spec.id,
      chart_kind: kind as never,
      source_function: spec.source_function,
      warnings,
      height: 0,
    },
    // @ts-expect-error
    __composite: {
      __kind: kind,
      id: spec.id,
      title: spec.title,
      blocks: [],
      warnings,
    },
  };
}

function makeCompositeOption(
  spec: ChartSpec,
  blocks: CompositeBlockTileSpec[],
): EChartsOption {
  return {
    series: [],
    __meta: {
      spec_id: spec.id,
      chart_kind: 'composite_block' as never,
      source_function: spec.source_function,
      warnings: [],
      height: 0,
    },
    // @ts-expect-error - extension
    __composite: {
      __kind: 'composite_block',
      id: spec.id,
      title: spec.title,
      blocks,
      warnings: [],
    } as CompositeUiSpec,
  };
}

/**
 * teammates.07 — Impact taquinerie (TABLEAU SCOREBOARD MATRICIEL)
 *
 * Structure réelle (cf. _render_impact_ranking_html) :
 *   Joueur | matchs 1..N (header coloré win/loss) | 8 cols emoji agrégat | Score | Badge
 *   1 ligne par joueur, triées par score décroissant.
 *   Cellule match×joueur = emojis events empilés 2 par ligne.
 *   Cellules agrégat colorées best (vert) / worst (rouge) selon extremum
 *   (inversion pour last_casualty / last_group_kill / first_group_death / false_brother).
 */
function convertImpactTaquinerie(spec: ChartSpec): EChartsOption {
  const EVENT_TO_EMOJI: Record<string, string> = {
    first_blood: '⚡',
    clutch_finisher: '🎯',
    last_casualty: '💀',
    last_group_kill: '🐌',
    first_group_death: '🪦',
    silent_hero: '🛡️',
    false_brother: '🗡️',
    top_killer: '💥',
  };
  const AGG_KEYS = Object.keys(EVENT_TO_EMOJI);
  const INVERTED = new Set(['last_casualty', 'last_group_kill', 'first_group_death', 'false_brother']);
  const OUTCOME_BG: Record<number, string> = {
    1: 'rgba(100,100,130,0.15)',
    2: 'rgba(0,158,115,0.30)',
    3: 'rgba(213,94,0,0.30)',
    4: 'rgba(100,100,130,0.15)',
  };

  // Mock dataset : 4 joueurs × 6 matchs avec events plausibles
  const players: Array<{
    gamertag: string;
    score: number;
    counts: Record<string, number>;
    matchEvents: Record<string, string[]>;
  }> = [
    {
      gamertag: 'JGtm',
      score: 8.5,
      counts: {
        first_blood: 2, clutch_finisher: 1, last_casualty: 0, last_group_kill: 0,
        first_group_death: 0, silent_hero: 1, false_brother: 0, top_killer: 1,
      },
      matchEvents: { m1: ['⚡', '💥'], m2: ['🎯'], m3: ['⚡'], m5: ['🛡️'] },
    },
    {
      gamertag: 'NeoSpartan_42',
      score: 3.5,
      counts: {
        first_blood: 1, clutch_finisher: 0, last_casualty: 0, last_group_kill: 1,
        first_group_death: 0, silent_hero: 1, false_brother: 0, top_killer: 1,
      },
      matchEvents: { m1: ['🛡️'], m4: ['⚡', '💥'], m6: ['🐌'] },
    },
    {
      gamertag: 'BlazingFury',
      score: 0.0,
      counts: {
        first_blood: 0, clutch_finisher: 1, last_casualty: 1, last_group_kill: 0,
        first_group_death: 1, silent_hero: 0, false_brother: 0, top_killer: 0,
      },
      matchEvents: { m2: ['💀'], m3: ['🎯'], m6: ['🪦'] },
    },
    {
      gamertag: 'ShadowKnight',
      score: -4.5,
      counts: {
        first_blood: 0, clutch_finisher: 0, last_casualty: 1, last_group_kill: 1,
        first_group_death: 0, silent_hero: 0, false_brother: 1, top_killer: 0,
      },
      matchEvents: { m4: ['💀', '🗡️'], m5: ['🐌'] },
    },
  ];

  const matchIds = ['m1', 'm2', 'm3', 'm4', 'm5', 'm6'];
  const matchOutcomes: Record<string, number> = { m1: 2, m2: 3, m3: 2, m4: 3, m5: 1, m6: 2 };

  // Calcul des extremums par colonne agrégat (≥2 valeurs distinctes non nulles)
  const extremes: Record<string, [number, number]> = {};
  for (const k of AGG_KEYS) {
    const vals = players.map((p) => p.counts[k]).filter((v) => v !== 0);
    if (vals.length >= 2) {
      const mn = Math.min(...vals);
      const mx = Math.max(...vals);
      if (mn !== mx) extremes[k] = [mn, mx];
    }
  }
  const scoreVals = players.map((p) => p.score).filter((s) => s !== 0);
  if (scoreVals.length >= 2) {
    const mn = Math.min(...scoreVals);
    const mx = Math.max(...scoreVals);
    if (mn !== mx) extremes.score = [mn, mx];
  }

  const tdClass = (key: string, val: number): string => {
    const ex = extremes[key];
    if (!ex || val === 0) return '';
    const [mn, mx] = ex;
    if (INVERTED.has(key)) {
      return val === mx ? ' os-sb-td--worst' : val === mn ? ' os-sb-td--best' : '';
    }
    return val === mx ? ' os-sb-td--best' : val === mn ? ' os-sb-td--worst' : '';
  };

  // Sorted players par score desc
  const sorted = [...players].sort((a, b) => b.score - a.score);
  const n = sorted.length;
  const lastScore = sorted[n - 1].score;

  // Style inline scoped (pas de feuille CSS dispo dans le mock)
  const style = `<style>
    .os-impact-table{border-collapse:collapse;width:100%;font-size:0.85em;color:#e8eef9;font-family:'Segoe UI',sans-serif}
    .os-impact-table .os-sb-team{padding:8px 10px;background:rgba(33,118,255,0.18);text-align:left;font-weight:700;letter-spacing:0.02em;border-bottom:1px solid rgba(255,255,255,0.08)}
    .os-impact-table .os-sb-th{padding:6px 8px;background:rgba(255,255,255,0.04);font-weight:600;text-align:center;border-bottom:1px solid rgba(255,255,255,0.08);color:#cdd6e6}
    .os-impact-table .os-sb-td{padding:6px 8px;border-bottom:1px solid rgba(255,255,255,0.04);text-align:center;color:#e8eef9}
    .os-impact-table .os-sb-td--best{background:rgba(0,158,115,0.22);color:#9be8c8;font-weight:700}
    .os-impact-table .os-sb-td--worst{background:rgba(213,94,0,0.22);color:#ffb98a;font-weight:700}
    .os-impact-table .os-sb-row--mvp td{background:rgba(255,215,0,0.10)}
    .os-impact-table .os-sb-row--lvp td{background:rgba(213,94,0,0.10)}
  </style>`;

  // Header
  const matchTh = matchIds
    .map((mid, i) => {
      const bg = OUTCOME_BG[matchOutcomes[mid] || 1];
      return `<th class="os-sb-th" style="width:34px;background:${bg}">${i + 1}</th>`;
    })
    .join('');
  const aggTh = AGG_KEYS.map((k) => `<th class="os-sb-th">${EVENT_TO_EMOJI[k]}</th>`).join('');
  const headRow = `<tr>
    <th class="os-sb-th" style="text-align:left">Joueur</th>
    ${matchTh}${aggTh}
    <th class="os-sb-th">Score</th>
    <th class="os-sb-th">Badge</th>
  </tr>`;
  const nTotal = 1 + matchIds.length + AGG_KEYS.length + 2;
  const captionRow = `<tr><th class="os-sb-team" colspan="${nTotal}">Classement Impact — Quel est l'effet de chacun ?</th></tr>`;
  const thead = `<tbody>${captionRow}${headRow}</tbody>`;

  // Body
  const bodyRows = sorted
    .map((p, idx) => {
      const rank = idx + 1;
      let rowClass = '';
      let badge = '';
      if (rank === 1) {
        rowClass = ' os-sb-row--mvp';
        badge = '🏆 Champion';
      } else if (rank === n && lastScore < 0) {
        rowClass = ' os-sb-row--lvp';
        badge = '🍌 Maillon faible';
      } else if (rank === n) {
        badge = '📉 Passager clandestin';
      }
      const scoreStr = p.score > 0 ? `+${p.score}` : `${p.score}`;
      const matchTds = matchIds
        .map((mid) => {
          const evs = p.matchEvents[mid] || [];
          const parts: string[] = [];
          evs.forEach((e, i) => {
            parts.push(e);
            if ((i + 1) % 2 === 0 && i < evs.length - 1) parts.push('<br/>');
          });
          return `<td class="os-sb-td">${parts.join('')}</td>`;
        })
        .join('');
      const aggTds = AGG_KEYS.map((k) => {
        const v = p.counts[k];
        return `<td class="os-sb-td${tdClass(k, v)}">${v ? v : '—'}</td>`;
      }).join('');
      return `<tbody><tr class="os-sb-player${rowClass}">
        <td class="os-sb-td" style="text-align:left;font-weight:600">${p.gamertag}</td>
        ${matchTds}${aggTds}
        <td class="os-sb-td${tdClass('score', p.score)}">${scoreStr}</td>
        <td class="os-sb-td">${badge}</td>
      </tr></tbody>`;
    })
    .join('');

  const html = `${style}<div style="overflow-x:auto;width:100%">
    <table class="os-impact-table">${thead}${bodyRows}</table>
  </div>`;
  return makeCompositeOption(spec, [{ type: 'raw_html', columnRatio: 1, payload: html }]);
}

/**
 * teammates.12 — Médailles escouade (4 expanders avec grille 6 cols × 2 rows).
 */
function convertTrioMedals(spec: ChartSpec): EChartsOption {
  const players = ['JGtm', 'NeoSpartan_42', 'BlazingFury', 'ShadowKnight'];
  const medalGrid = (seed: number): string => {
    const cells = Array.from({ length: 12 }, (_, i) => {
      const count = ((seed * 7 + i * 3) % 18) + 2;
      const hue = (seed * 60 + i * 30) % 360;
      return `<div style="display:flex;flex-direction:column;align-items:center;gap:2px;padding:4px">
        <div style="width:40px;height:40px;border-radius:50%;background:radial-gradient(circle,hsl(${hue},70%,55%) 0%,hsl(${hue},60%,30%) 70%);border:2px solid hsl(${hue},80%,40%);display:flex;align-items:center;justify-content:center;color:#fff;font-size:0.7em;font-weight:700">M${i + 1}</div>
        <div style="font-size:0.7em;color:#888">×${count}</div>
      </div>`;
    }).join('');
    return `<div style="display:grid;grid-template-columns:repeat(6,1fr);gap:4px;padding:8px">${cells}</div>`;
  };
  const expanders = players
    .map(
      (name, idx) => `<div style="background:rgba(255,255,255,0.03);border-radius:6px;overflow:hidden">
      <div style="padding:8px 12px;background:rgba(255,255,255,0.05);font-weight:700;font-size:0.9em;border-bottom:1px solid rgba(255,255,255,0.06)">▼ ${name}</div>
      ${medalGrid(idx + 1)}
    </div>`,
    )
    .join('');
  const html = `<div style="display:grid;grid-template-columns:repeat(4,1fr);gap:10px;width:100%">${expanders}</div>`;
  return makeCompositeOption(spec, [{ type: 'raw_html', columnRatio: 1, payload: html }]);
}

/**
 * match_view.15 — Citations progressées (grille de 6 anneaux conic-gradient).
 */
function convertCitationsGrid(spec: ChartSpec): EChartsOption {
  const citations = [
    { name: 'Marksman', level: 'Or', current: 1234, target: 2000, delta: 18, master: false },
    { name: 'Sharpshooter', level: 'Argent', current: 567, target: 1000, delta: 22, master: false },
    { name: 'Demolitionist', level: 'Maître', current: 5012, target: 5000, delta: 14, master: true },
    { name: 'Tactician', level: 'Bronze', current: 145, target: 500, delta: 8, master: false },
    { name: 'Spartan', level: 'Or', current: 1789, target: 2500, delta: 35, master: false },
    { name: 'Vigilant', level: 'Argent', current: 678, target: 1200, delta: 11, master: false },
  ];
  const tiles = citations
    .map((c) => {
      const ratio = Math.min(1, c.current / c.target);
      const ringColor = c.master ? '#d6b35a' : '#41d6ff';
      const angle = (ratio * 360).toFixed(0);
      const levelColor = c.master ? '#d6b35a' : '#bbb';
      return `<div style="display:flex;flex-direction:column;align-items:center;gap:4px;padding:6px">
        <div style="width:72px;height:72px;border-radius:50%;background:conic-gradient(${ringColor} 0deg ${angle}deg, rgba(255,255,255,0.06) ${angle}deg 360deg);display:flex;align-items:center;justify-content:center">
          <div style="width:56px;height:56px;border-radius:50%;background:#1a1a1a;display:flex;align-items:center;justify-content:center;color:${ringColor};font-size:0.7em;font-weight:700">${c.name.slice(0, 3).toUpperCase()}</div>
        </div>
        <div style="font-size:0.78em;font-weight:600;text-align:center">${c.name}</div>
        <div style="font-size:0.72em;color:${levelColor};font-weight:${c.master ? 700 : 500}">${c.level}</div>
        <div style="font-size:0.72em;color:#888">${c.current}/${c.target} <span style="color:#4CAF50;font-weight:700">+${c.delta}</span></div>
      </div>`;
    })
    .join('');
  const html = `<div style="display:grid;grid-template-columns:repeat(6,1fr);gap:8px;width:100%;padding:8px;justify-items:center">${tiles}</div>`;
  return makeCompositeOption(spec, [{ type: 'raw_html', columnRatio: 1, payload: html }]);
}

/**
 * match_view.16 — Médailles du match (grille centrée 8 cols × 12 médailles).
 */
function convertMedalsGrid(spec: ChartSpec): EChartsOption {
  const medals = [
    { name: 'Killing Spree', count: 4, hue: 200 },
    { name: 'Double Kill', count: 7, hue: 280 },
    { name: 'Triple Kill', count: 3, hue: 320 },
    { name: 'Headshot', count: 12, hue: 45 },
    { name: 'Perfection', count: 1, hue: 50 },
    { name: 'Sniper Kill', count: 5, hue: 150 },
    { name: 'Melee Kill', count: 2, hue: 0 },
    { name: 'Quick Draw', count: 6, hue: 180 },
    { name: 'Achilles', count: 1, hue: 30 },
    { name: 'From the Grave', count: 2, hue: 270 },
    { name: 'Pancake', count: 3, hue: 90 },
    { name: 'Splatter', count: 4, hue: 110 },
  ];
  const tiles = medals
    .map(
      (m) => `<div style="display:flex;flex-direction:column;align-items:center;gap:3px;padding:6px">
      <div style="width:56px;height:56px;border-radius:50%;background:radial-gradient(circle,hsl(${m.hue},70%,55%) 0%,hsl(${m.hue},60%,30%) 70%);border:2px solid hsl(${m.hue},80%,45%);display:flex;align-items:center;justify-content:center;color:#fff;font-size:0.7em;font-weight:700">${m.name.split(' ').map((w) => w[0]).join('')}</div>
      <div style="font-size:0.78em;font-weight:700;color:#fff">×${m.count}</div>
      <div style="font-size:0.7em;color:#aaa;text-align:center;max-width:72px;line-height:1.1">${m.name}</div>
    </div>`,
    )
    .join('');
  const html = `<div style="display:grid;grid-template-columns:repeat(8,1fr);gap:6px;width:100%;padding:8px;justify-items:center">${tiles}</div>`;
  return makeCompositeOption(spec, [{ type: 'raw_html', columnRatio: 1, payload: html }]);
}

// =============================================================================
// timeseries.00 / teammates.00 — KPI Section (8 cards avec trends)
// =============================================================================

function convertKpisSection(spec: ChartSpec): EChartsOption {
  type Trend = 'above' | 'near' | 'below' | 'none';
  const trendIcon = (t: Trend, color: { up: string; down: string; near: string }): string => {
    if (t === 'above') return `<span style="color:${color.up};font-size:0.85em">▲</span>`;
    if (t === 'below') return `<span style="color:${color.down};font-size:0.85em">▼</span>`;
    if (t === 'near') return `<span style="color:${color.near};font-size:0.85em">━</span>`;
    return '';
  };
  const tColor = { up: '#00C853', down: '#FF5252', near: '#888888' };

  const cards = [
    { label: 'Matchs', main: '24', sub: '08:42/match', trend: 'none' as Trend },
    { label: 'Durée totale', main: '3h 28min', sub: null, trend: 'none' as Trend },
    { label: 'Kills/match', main: '14.20', sub: '1.63/min', trend: 'above' as Trend },
    { label: 'Morts/match', main: '9.80', sub: '1.13/min', trend: 'above' as Trend },
    { label: 'Assists/match', main: '5.50', sub: '0.63/min', trend: 'near' as Trend },
    { label: 'Précision', main: '52.34%', sub: null, trend: 'near' as Trend },
    { label: 'Vie moy.', main: '1:18', sub: null, trend: 'below' as Trend },
  ];
  const cardHtml = cards
    .map(
      (c) => `<div style="background:rgba(255,255,255,0.04);border-radius:8px;padding:12px;display:flex;flex-direction:column;gap:4px;min-width:140px">
      <div style="font-size:0.78em;color:#A0A0A0;font-weight:500">${c.label}</div>
      <div style="display:flex;align-items:baseline;gap:6px">
        <span style="font-size:1.6em;font-weight:700;color:#fff">${c.main}</span>
        ${trendIcon(c.trend, tColor)}
      </div>
      ${c.sub ? `<div style="font-size:0.72em;color:#888">${c.sub}</div>` : ''}
    </div>`,
    )
    .join('');

  // Results bar 8e card "wide"
  const wins = 14;
  const losses = 9;
  const ties = 1;
  const dnf = 0;
  const total = wins + losses + ties + dnf;
  const seg = (count: number, color: string, label: string): string => {
    const pct = ((count / total) * 100).toFixed(1);
    return `<div style="flex:${count};background:${color};display:flex;align-items:center;justify-content:center;color:#000;font-weight:600;font-size:0.78em" title="${label}">${count > 0 ? count : ''}</div>`;
  };
  const resultsHtml = `<div style="background:rgba(255,255,255,0.04);border-radius:8px;padding:12px;display:flex;flex-direction:column;gap:6px;flex:2;min-width:280px">
    <div style="font-size:0.78em;color:#A0A0A0;font-weight:500">Résultats</div>
    <div style="display:flex;height:32px;border-radius:4px;overflow:hidden;border:1px solid rgba(255,255,255,0.08)">
      ${seg(wins, '#3DFF9A', `${wins} V`)}
      ${seg(losses, '#FF5C5C', `${losses} D`)}
      ${ties > 0 ? seg(ties, '#A855F7', `${ties} E`) : ''}
      ${dnf > 0 ? seg(dnf, 'rgba(182,196,214,0.45)', `${dnf} N/F`) : ''}
    </div>
    <div style="font-size:0.72em;color:#888;display:flex;gap:10px">
      <span><span style="color:#3DFF9A">●</span> ${wins} V</span>
      <span><span style="color:#FF5C5C">●</span> ${losses} D</span>
      ${ties > 0 ? `<span><span style="color:#A855F7">●</span> ${ties} E</span>` : ''}
    </div>
  </div>`;

  const html = `<div style="display:flex;flex-wrap:wrap;gap:10px;width:100%">${cardHtml}${resultsHtml}</div>`;
  return makeCompositeOption(spec, [{ type: 'raw_html', columnRatio: 1, payload: html }]);
}

// =============================================================================
// teammates.01 — Squad session header (team card grade + N joueurs)
// =============================================================================

function convertSquadSessionHeader(spec: ChartSpec): EChartsOption {
  const teamScore = 68;
  const baseAvg = 64;
  const bonus = teamScore - baseAvg;
  const grade = 'B';
  const players = [
    { name: 'JGtm', score: 78, isBetter: true },
    { name: 'NeoSpartan_42', score: 65, isBetter: false },
    { name: 'BlazingFury', score: 71, isBetter: true },
    { name: 'ShadowKnight', score: 58, isBetter: false },
  ];

  const scoreColor = (s: number): string => {
    if (s >= 75) return '#00e676';
    if (s >= 60) return '#41d6ff';
    if (s >= 45) return '#ffb300';
    if (s >= 30) return '#FF8C00';
    return '#e53935';
  };
  const scoreLabel = (s: number): string => {
    if (s >= 75) return 'Excellent';
    if (s >= 60) return 'Solide';
    if (s >= 45) return 'Correct';
    if (s >= 30) return 'Mauvais';
    return 'Pourri';
  };

  // Team card (col 0)
  const teamCard = `<div style="background:rgba(0,183,235,0.08);border:2px solid #00B7EB;border-radius:8px;padding:14px;display:flex;flex-direction:column;gap:6px;min-width:160px">
    <div style="font-size:0.8em;color:#A0A0A0;font-weight:500">Score escouade</div>
    <div style="display:flex;align-items:baseline;gap:10px">
      <span style="font-size:2em;font-weight:700;color:${scoreColor(teamScore)}">${teamScore}</span>
      <span style="font-size:1.6em;font-weight:700;color:${scoreColor(teamScore)};letter-spacing:0.05em">${grade}</span>
    </div>
    <div style="font-size:0.75em;color:#888">Score base ${baseAvg} ${bonus > 0 ? `<span style="color:#00C853">+${bonus} bonus</span>` : ''}</div>
  </div>`;

  // Player cards
  const playerCards = players
    .map((p) => {
      const badge = p.isBetter
        ? `<span style="color:#00C853;font-size:1em;margin-left:8px">▲</span>`
        : `<span style="color:#FF5252;font-size:1em;margin-left:8px">▼</span>`;
      return `<div style="background:rgba(255,255,255,0.04);border-radius:8px;padding:12px;display:flex;flex-direction:column;gap:4px;min-width:140px">
        <div style="font-size:0.8em;color:#A0A0A0;font-weight:500">${p.name}</div>
        <div style="display:flex;align-items:baseline">
          <span style="font-size:1.6em;font-weight:700;color:${scoreColor(p.score)}">${p.score}</span>
          ${badge}
        </div>
        <div style="font-size:0.75em;color:${scoreColor(p.score)}">${scoreLabel(p.score)}</div>
      </div>`;
    })
    .join('');

  const html = `<div style="display:flex;flex-wrap:wrap;gap:10px;width:100%">${teamCard}${playerCards}</div>`;
  return makeCompositeOption(spec, [{ type: 'raw_html', columnRatio: 1, payload: html }]);
}

// =============================================================================
// career.11 — LUSR rank cards (grille N playlists avec rating + delta)
// =============================================================================

function convertLusrRankCards(spec: ChartSpec): EChartsOption {
  const playlists = [
    { pg: 'ranked', icon: '🏆', label: 'Ranked', tier: 'Onyx Diamond III', rating: 1842, delta: 24, type: 'CSR' },
    { pg: 'slayer', icon: '⚔️', label: 'Slayer', tier: 'Platinum I', rating: 1456, delta: -12, type: 'LUSR' },
    { pg: 'btb', icon: '🎯', label: 'Big Team Battle', tier: 'Gold V', rating: 1283, delta: 8, type: 'LUSR' },
    { pg: 'quickplay', icon: '⚡', label: 'Quickplay', tier: 'Platinum III', rating: 1612, delta: 0, type: 'LUSR' },
    { pg: 'objectifs', icon: '🚩', label: 'Objectifs', tier: 'Gold II', rating: 1198, delta: 18, type: 'LUSR' },
    { pg: 'mixed', icon: '🎮', label: 'Mixed', tier: 'Silver IV', rating: 987, delta: -5, type: 'LUSR' },
  ];

  const rankImg = (rating: number): string => {
    // SVG simple : couronne stylisée avec gradient selon le rating
    const hue = Math.min(60, Math.max(0, ((rating - 800) / 1100) * 60)); // 0=red → 60=yellow
    return `<svg viewBox="0 0 90 90" width="90" height="90">
      <defs><linearGradient id="g${rating}" x1="0%" y1="0%" x2="0%" y2="100%">
        <stop offset="0%" stop-color="hsl(${hue + 60}, 80%, 65%)" />
        <stop offset="100%" stop-color="hsl(${hue}, 80%, 35%)" />
      </linearGradient></defs>
      <polygon points="45,10 60,30 90,30 65,55 75,80 45,65 15,80 25,55 0,30 30,30" fill="url(#g${rating})" stroke="rgba(0,0,0,0.4)" stroke-width="1.5"/>
      <text x="45" y="55" text-anchor="middle" fill="#fff" font-size="14" font-weight="bold">${Math.floor(rating / 100)}</text>
    </svg>`;
  };

  const cards = playlists
    .map((p) => {
      const badgeBg = p.type === 'CSR' ? '#FFD700' : '#00B7EB';
      const deltaHtml =
        p.delta === 0
          ? `<div style="color:#888888;font-size:0.85em;margin-top:2px">= 0</div>`
          : p.delta > 0
            ? `<div style="color:#00C853;font-size:0.85em;margin-top:2px">▲ +${p.delta}</div>`
            : `<div style="color:#FF5252;font-size:0.85em;margin-top:2px">▼ ${p.delta}</div>`;
      return `<div style="background:rgba(255,255,255,0.04);border:1px solid rgba(255,255,255,0.08);border-radius:8px;padding:12px;display:flex;flex-direction:column;align-items:center;gap:6px;min-width:200px">
        <div style="font-size:0.85em;color:#A0A0A0;font-weight:500">${p.icon} ${p.label}</div>
        ${rankImg(p.rating)}
        <div style="font-size:0.92em;font-weight:600;color:#fff">${p.tier}</div>
        <div style="display:flex;align-items:center;gap:8px">
          <span style="background:${badgeBg};color:#000;padding:1px 6px;border-radius:8px;font-size:0.78em;font-weight:700">${p.type}</span>
          <span style="font-weight:700;font-size:1.1em;color:#fff">${p.rating}</span>
        </div>
        ${deltaHtml}
      </div>`;
    })
    .join('');

  const html = `<div style="display:grid;grid-template-columns:repeat(3, 1fr);gap:10px;width:100%">${cards}</div>`;
  return makeCompositeOption(spec, [{ type: 'raw_html', columnRatio: 1, payload: html }]);
}
