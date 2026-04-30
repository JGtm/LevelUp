import type { ChartSpec, EChartsOption, ThemeDefault } from '../types.js';

/**
 * Spec d'un tableau HTML — pas une option ECharts.
 * Le renderer HTML produit une vraie <table> à partir de cette structure.
 */
export interface TableSpec {
  __kind: 'table_html';
  id: string;
  title: string;
  caption: { text: string; colspan: number };
  cssClasses: {
    wrapper: string;
    table: string;
    headerTeam: string;
    headerCell: string;
    row: string;
    cell: string;
  };
  inlineStyles?: { table?: string };
  columns: Array<{
    id: string;
    header: string;
    style?: string;
  }>;
  rows: Array<{
    cells: Array<{ html: string; style?: string }>;
    isDegraded?: boolean;
    degradedHtml?: string; // si la ligne est en mode dégradé (colspan large)
  }>;
  legend?: {
    enabled: boolean;
    html?: string; // HTML pré-construit
    placement: string;
  };
  warnings: string[];
}

/**
 * Convertit un chart `table_html` en TableSpec.
 * Génère du HTML à partir des spec YAML + données mock.
 */
export function convertTableHtml(
  spec: ChartSpec,
  theme: ThemeDefault,
  mockCtx: Record<string, unknown>,
  warnings: string[],
): EChartsOption {
  const tableSpec = buildTableSpec(spec, mockCtx, warnings);
  // Le converter générique attend un EChartsOption — on retourne un objet sentinel
  // que le renderer HTML détecte via __meta.chart_kind ou via une clé spéciale.
  return {
    series: [],
    __meta: {
      spec_id: spec.id,
      chart_kind: 'table_html',
      source_function: spec.source_function,
      warnings,
      height: 0, // pas pertinent pour une table
    },
    // @ts-expect-error - extension non standard pour transporter le table spec
    __table: tableSpec,
  };
}

function buildTableSpec(
  spec: ChartSpec,
  mockCtx: Record<string, unknown>,
  warnings: string[],
): TableSpec {
  // @ts-expect-error - section table custom du YAML
  const table = spec.table as Record<string, unknown> | undefined;
  if (!table) {
    warnings.push('Section `table:` absente du YAML — fallback table vide');
    return emptyTable(spec, warnings);
  }

  const css = (table.css_classes as Record<string, string>) ?? {
    wrapper: 'os-table-wrap os-sb-wrap',
    table: 'os-table os-scoreboard',
    header_team: 'os-sb-team',
    header_cell: 'os-sb-th',
    row: 'os-sb-row',
    cell: 'os-sb-td',
  };

  const captionRaw = (table.caption as Record<string, unknown> | undefined)?.text as
    | string
    | undefined;
  const caption = {
    text: resolveLabel(captionRaw, mockCtx),
    colspan: typeof (table.caption as { colspan?: number })?.colspan === 'number'
      ? ((table.caption as { colspan: number }).colspan)
      : (Array.isArray(table.columns) ? (table.columns as unknown[]).length : 6),
  };

  // Le YAML peut contenir une string "(idem chart X)" comme raccourci dans columns —
  // dans ce cas on fallback sur les colonnes par défaut connues pour ce spec id.
  let cols = Array.isArray(table.columns)
    ? (table.columns as Array<Record<string, unknown>>)
    : getDefaultColumnsFor(spec.id, mockCtx);
  if (!Array.isArray(cols) || cols.length === 0) {
    warnings.push(`Section table.columns absente ou non-array pour ${spec.id} — fallback default`);
    cols = getDefaultColumnsFor(spec.id, mockCtx);
  }

  // Mock rows par chart : on génère 5-10 lignes représentatives
  const mockRows = generateMockRows(spec.id, cols.length);

  return {
    __kind: 'table_html',
    id: spec.id,
    title: spec.title,
    caption,
    cssClasses: {
      wrapper: css.wrapper,
      table: css.table,
      headerTeam: css.header_team,
      headerCell: css.header_cell,
      row: css.row,
      cell: css.cell,
    },
    inlineStyles: {
      table: ((table.inline_styles as Record<string, string>) ?? {}).table,
    },
    columns: cols.map((c) => ({
      id: c.id as string,
      header: resolveLabel(c.header as string, mockCtx),
      style: c.style as string | undefined,
    })),
    rows: mockRows,
    legend:
      (table.legend as { enabled: boolean })?.enabled === true
        ? {
            enabled: true,
            html: buildMockLegend(spec.id),
            placement: ((table.legend as { placement?: string })?.placement) ?? 'below_table',
          }
        : { enabled: false, placement: 'none' },
    warnings,
  };
}

/**
 * Colonnes par défaut quand le YAML utilise une string raccourci ("(idem chart X)").
 * Permet de garder les YAML courts pour les variantes (career.06 vs 05, career.09 vs 08).
 */
function getDefaultColumnsFor(specId: string, _mockCtx: Record<string, unknown>): Array<Record<string, unknown>> {
  if (specId === 'career.05' || specId === 'career.06') {
    return [
      { id: 'match_id', header: '{{t.career_top_col_match_id}}' },
      { id: 'date', header: '{{t.career_top_col_date}}' },
      { id: 'mode', header: '{{t.career_top_col_mode}}' },
      { id: 'map', header: '{{t.career_top_col_map}}' },
      { id: 'score', header: '{{t.career_top_col_score}}', style: 'font-weight: 600' },
      { id: 'kda', header: '{{t.career_top_col_kda}}' },
      { id: 'kd_ratio', header: '{{t.career_top_col_kd}}' },
      { id: 'duration', header: '{{t.career_top_col_duration}}' },
      { id: 'badge', header: '' },
    ];
  }
  if (specId === 'career.10') {
    return [
      { id: 'date', header: 'Date' },
      { id: 'rank', header: 'Rang' },
      { id: 'rank_label', header: 'Titre' },
      { id: 'xp_total', header: 'XP total', style: 'text-align: right; font-weight: 600' },
    ];
  }
  if (specId === 'career.07') {
    return [
      { id: 'player', header: '{{t.col_player}}' },
      { id: 'role', header: '{{t.col_role}}' },
      { id: 'encounters', header: '{{t.col_encounters}}' },
      { id: 'wr_ally', header: '{{t.col_wr_ally}}' },
      { id: 'wr_enemy', header: '{{t.col_wr_enemy}}' },
      { id: 'kd_cross', header: '{{t.col_kd_cross}}' },
      { id: 'last_seen', header: '{{t.col_last_seen}}', style: 'color: #aaa; font-size: 0.85em' },
    ];
  }
  if (specId === 'career.08') {
    return [
      { id: 'rank', header: '#', style: 'text-align: left' },
      { id: 'player', header: '{{t.col_player}}', style: 'text-align: left' },
      { id: 'main_metric', header: '{{t.col_times_killed_by}}', style: 'text-align: center; font-weight: 700' },
      { id: 'sec_metric', header: '{{t.col_times_killed}}', style: 'text-align: center' },
      { id: 'net', header: '{{t.col_net_kills}}', style: 'text-align: center' },
      { id: 'matches_against', header: '{{t.col_matches_against}}', style: 'text-align: center; color: #aaa' },
    ];
  }
  if (specId === 'career.09') {
    return [
      { id: 'rank', header: '#', style: 'text-align: left' },
      { id: 'player', header: '{{t.col_player}}', style: 'text-align: left' },
      { id: 'main_metric', header: '{{t.col_times_killed}}', style: 'text-align: center; font-weight: 700' },
      { id: 'sec_metric', header: '{{t.col_times_killed_by}}', style: 'text-align: center' },
      { id: 'net', header: '{{t.col_net_kills}}', style: 'text-align: center' },
      { id: 'matches_against', header: '{{t.col_matches_against}}', style: 'text-align: center; color: #aaa' },
    ];
  }
  if (specId === 'teammates.10') {
    return [
      { id: 'weapon', header: 'Arme' },
      { id: 'kills', header: 'Frags', style: 'text-align: right; font-weight: 600' },
    ];
  }
  if (specId === 'teammates.11') {
    return [
      { id: 'app', header: 'Match' },
      { id: 'waypoint', header: 'HW' },
      { id: 'date', header: 'Date', style: 'white-space: nowrap' },
      { id: 'map', header: 'Carte' },
      { id: 'playlist', header: 'Playlist' },
      { id: 'mode', header: 'Mode' },
      { id: 'result', header: 'Résultat' },
      { id: 'win_rate_hist', header: 'WR hist' },
      { id: 'score', header: 'Score' },
      { id: 'team_mmr', header: 'MMR alliés' },
      { id: 'enemy_mmr', header: 'MMR ennemis' },
      { id: 'delta_mmr', header: 'Δ MMR' },
    ];
  }
  return [];
}

function emptyTable(spec: ChartSpec, warnings: string[]): TableSpec {
  return {
    __kind: 'table_html',
    id: spec.id,
    title: spec.title,
    caption: { text: spec.title, colspan: 1 },
    cssClasses: {
      wrapper: 'os-table-wrap',
      table: 'os-table',
      headerTeam: 'os-sb-team',
      headerCell: 'os-sb-th',
      row: 'os-sb-row',
      cell: 'os-sb-td',
    },
    columns: [],
    rows: [],
    warnings,
  };
}

/**
 * Résout grossièrement les tokens `{{t.xxx}}` ou `{{viz_t.xxx}}` en strings lisibles.
 * Pour un mock visuel, on traduit vers les labels FR explicites.
 */
function resolveLabel(s: string | undefined, _mockCtx: Record<string, unknown>): string {
  if (!s) return '';
  const m = s.match(/^\{\{(t|viz_t)\.(\w+)\}\}$/);
  if (!m) return s;
  const key = m[2];
  // Petite table de correspondance pour les clés utilisées dans les tableaux Career
  const labels: Record<string, string> = {
    career_top_best_title: 'Top 10 meilleurs matchs',
    career_top_worst_title: 'Top 10 pires matchs',
    career_encounters_header: 'Top 10 joueurs croisés',
    career_nemesis_header: 'Top 10 némésis',
    career_victims_header: 'Top 10 souffre-douleurs',
    career_top_col_match_id: 'Match',
    career_top_col_date: 'Date',
    career_top_col_mode: 'Mode',
    career_top_col_map: 'Carte',
    career_top_col_score: 'Score',
    career_top_col_kda: 'K/D/A',
    career_top_col_kd: 'K/D',
    career_top_col_duration: 'Durée',
    col_player: 'Joueur',
    col_role: 'Rôle',
    col_encounters: 'Rencontres',
    col_wr_ally: 'WR allié',
    col_wr_enemy: 'WR ennemi',
    col_kd_cross: 'K/D croisé',
    col_last_seen: 'Vu pour la dernière fois',
    col_times_killed: 'Tué',
    col_times_killed_by: 'Tué par',
    col_net_kills: 'Net',
    col_matches_against: 'Matchs',
  };
  return labels[key] ?? `i18n:${m[1]}.${key}`;
}

// === Mock rows generators ===

function generateMockRows(specId: string, nCols: number): TableSpec['rows'] {
  if (specId === 'career.05' || specId === 'career.06') {
    return mockTopMatchesRows(specId === 'career.05');
  }
  if (specId === 'career.07') {
    return mockEncountersRows();
  }
  if (specId === 'career.08' || specId === 'career.09') {
    return mockAntagonistRows(specId === 'career.08');
  }
  if (specId === 'match_view.06' || specId === 'teammates.10') {
    return mockWeaponKillsRows();
  }
  if (specId === 'match_view.08') {
    return mockScoreboardRows();
  }
  if (specId === 'teammates.11') {
    return mockFriendsHistoryRows();
  }
  if (specId === 'match_view.14') {
    return mockEncountersRows();
  }
  if (specId === 'career.10') {
    return mockXpSnapshotsRows();
  }
  // Fallback : 3 lignes vides
  return Array.from({ length: 3 }, () => ({
    cells: Array.from({ length: nCols }, () => ({ html: '—' })),
  }));
}

function mockXpSnapshotsRows(): TableSpec['rows'] {
  // 10 dernières snapshots career_progression (du plus récent au plus ancien)
  const snapshots = [
    { date: '2026-04-28 21:32', rank: 188, label: 'Onyx Diamond III · Champion', xp: 1842300 },
    { date: '2026-04-25 19:14', rank: 186, label: 'Onyx Diamond II · Major', xp: 1798450 },
    { date: '2026-04-22 22:01', rank: 184, label: 'Onyx Diamond I · Major', xp: 1755200 },
    { date: '2026-04-19 21:48', rank: 181, label: 'Onyx Platinum III · Major', xp: 1712100 },
    { date: '2026-04-15 20:22', rank: 178, label: 'Onyx Platinum II · Sergeant', xp: 1668900 },
    { date: '2026-04-12 22:35', rank: 175, label: 'Onyx Platinum I · Sergeant', xp: 1625400 },
    { date: '2026-04-08 19:55', rank: 172, label: 'Onyx Gold V · Corporal', xp: 1582300 },
    { date: '2026-04-05 21:08', rank: 169, label: 'Onyx Gold IV · Corporal', xp: 1539800 },
    { date: '2026-04-01 20:42', rank: 166, label: 'Onyx Gold III · Private', xp: 1497200 },
    { date: '2026-03-28 22:18', rank: 163, label: 'Onyx Gold II · Private', xp: 1454600 },
  ];
  return snapshots.map((s) => ({
    cells: [
      { html: `<span style="color:#aaa">${s.date}</span>` },
      { html: `<span style="font-weight:700;color:#41d6ff">${s.rank}</span>` },
      { html: s.label },
      { html: s.xp.toLocaleString('fr-FR'), style: 'text-align: right; font-weight: 600' },
    ],
  }));
}

function mockFriendsHistoryRows(): TableSpec['rows'] {
  const outcomes = [
    { label: 'V', color: '#3DFFB5' },
    { label: 'D', color: '#FF4D6D' },
    { label: 'V', color: '#3DFFB5' },
    { label: 'V', color: '#3DFFB5' },
    { label: 'D', color: '#FF4D6D' },
    { label: '–', color: '#888' },
  ];
  const dates = ['15/04/2026 21:32', '14/04/2026 19:48', '12/04/2026 22:05', '11/04/2026 20:14', '08/04/2026 19:30', '05/04/2026 21:55'];
  const maps = ['Aquarius', 'Behemoth', 'Streets', 'Recharge', 'Live Fire', 'Catalyst'];
  const playlists = ['Ranked Arena', 'Quickplay', 'Big Team Battle', 'Ranked Arena', 'Quickplay', 'Tactical Slayer'];
  const modes = ['Slayer', 'CTF', 'BTB Slayer', 'Strongholds', 'KOTH', 'Tactical Slayer'];
  const wrHist = [0.62, 0.45, 0.55, 0.71, 0.38, 0.52];
  const scores = ['50–32', '2–5', '127–98', '250–215', '32–50', '50–48'];
  const teamMMR = [1480.5, 1502.3, 1455.8, 1521.2, 1478.6, 1495.1];
  const enemyMMR = [1462.1, 1530.8, 1438.4, 1510.5, 1525.3, 1490.6];

  return outcomes.map((o, i) => {
    const dm = teamMMR[i] - enemyMMR[i];
    const dmColor = dm > 0 ? '#3DFFB5' : '#FF4D6D';
    return {
      cells: [
        { html: `<a href="#" style="color:#33d6ff;font-family:monospace;font-size:0.75em">a1b2c3d4…</a>` },
        { html: `<a href="#" style="color:#aaa;font-size:0.75em">HW</a>` },
        { html: dates[i], style: 'white-space: nowrap; font-size: 0.85em' },
        { html: maps[i] },
        { html: playlists[i], style: 'font-size: 0.85em' },
        { html: modes[i] },
        { html: o.label, style: `color: ${o.color}; font-weight: 700` },
        { html: `${(wrHist[i] * 100).toFixed(0)}%`, style: 'color: #aaa; font-size: 0.85em' },
        { html: scores[i], style: 'font-weight: 600' },
        { html: teamMMR[i].toFixed(1), style: 'font-size: 0.85em' },
        { html: enemyMMR[i].toFixed(1), style: 'font-size: 0.85em' },
        { html: `${dm > 0 ? '+' : ''}${dm.toFixed(1)}`, style: `color: ${dmColor}; font-weight: 600` },
      ],
    };
  });
}

function mockWeaponKillsRows(): TableSpec['rows'] {
  const data = [
    { weapon: 'MA40 AR', kills: 12 },
    { weapon: 'BR75', kills: 8 },
    { weapon: 'Sidekick', kills: 6 },
    { weapon: 'Plasma Carbine', kills: 4 },
    { weapon: 'Bulldog', kills: 3 },
    { weapon: 'Mêlée', kills: 2 },
    { weapon: 'Grenades', kills: 1 },
  ];
  return data.map((r) => ({
    cells: [
      { html: r.weapon },
      { html: String(r.kills), style: 'text-align: right; font-weight: 600' },
    ],
  }));
}

function mockScoreboardRows(): TableSpec['rows'] {
  // Mock simplifié : 4 joueurs avec 19 colonnes typiques
  // (teamId implicite — tous dans la même équipe, le multi-table sera géré par le renderer)
  const cols = (k: number, d: number, a: number, score: number, rowMod = ''): TableSpec['rows'][0] => ({
    cells: [
      {
        html: `<a href='#' style='color:#33d6ff;font-weight:600'>Player_${Math.floor(Math.random() * 9999)}</a>`,
      },
      { html: '15' },
      { html: String(score), style: 'font-weight: 700' },
      { html: String(k) },
      { html: String(d) },
      { html: String(a) },
      { html: `${(k / Math.max(d, 1)).toFixed(2)}` },
      { html: 'BR75' },
      { html: '7' },
      { html: '3' },
      { html: '1' },
      { html: '180' },
      { html: '54' },
      { html: '30%' },
      { html: '2' },
      { html: '4' },
      { html: '8500' },
      { html: '6300' },
      { html: '2:15' },
    ],
  });

  return [
    cols(18, 11, 4, 1850), // MVP-like
    cols(14, 12, 6, 1720),
    cols(11, 13, 3, 1480),
    cols(8, 15, 2, 1100), // LVP-like
  ];
}

function mockTopMatchesRows(best: boolean): TableSpec['rows'] {
  const data = best
    ? [
        { id: 'a1b2c3d4', date: '15/04/2026 21:32', mode: 'Slayer', map: 'Aquarius', score: '50 — 32', kda: '24/8/3', kd: '3.00', duration: '11:42', badge: 'DOMINATION' },
        { id: 'b2c3d4e5', date: '12/04/2026 19:18', mode: 'CTF', map: 'Behemoth', score: '5 — 2', kda: '18/12/7', kd: '1.50', duration: '14:15', badge: 'REMONTADA' },
        { id: 'c3d4e5f6', date: '08/04/2026 22:41', mode: 'Strongholds', map: 'Recharge', score: '250 — 180', kda: '21/9/4', kd: '2.33', duration: '13:08', badge: '' },
        { id: 'd4e5f6a7', date: '03/04/2026 20:55', mode: 'Slayer', map: 'Streets', score: '50 — 41', kda: '19/14/2', kd: '1.36', duration: '9:32', badge: '' },
        { id: 'e5f6a7b8', date: '28/03/2026 18:22', mode: 'Oddball', map: 'Catalyst', score: '100 — 78', kda: '15/11/9', kd: '1.36', duration: '12:48', badge: '' },
      ]
    : [
        { id: 'f6a7b8c9', date: '14/04/2026 22:12', mode: 'Slayer', map: 'Argyle', score: '50 — 18', kda: '6/22/3', kd: '0.27', duration: '8:42', badge: 'HUMILIATION' },
        { id: 'a7b8c9d0', date: '10/04/2026 20:33', mode: 'CTF', map: 'Live Fire', score: '1 — 5', kda: '8/19/4', kd: '0.42', duration: '11:55', badge: 'DÉBÂCLE' },
        { id: 'b8c9d0e1', date: '05/04/2026 21:08', mode: 'Strongholds', map: 'Aquarius', score: '120 — 250', kda: '11/18/2', kd: '0.61', duration: '10:28', badge: '' },
        { id: 'c9d0e1f2', date: '01/04/2026 19:44', mode: 'Slayer', map: 'Behemoth', score: '32 — 50', kda: '14/22/5', kd: '0.64', duration: '12:01', badge: '' },
        { id: 'd0e1f2a3', date: '25/03/2026 18:55', mode: 'Oddball', map: 'Streets', score: '78 — 100', kda: '13/19/3', kd: '0.68', duration: '11:14', badge: '' },
      ];

  const badgeColors: Record<string, { bg: string; fg: string }> = {
    DOMINATION: { bg: '#2e7d32', fg: '#e8f5e9' },
    REMONTADA: { bg: '#1565c0', fg: '#e3f2fd' },
    'CONTRE-REMONTADA': { bg: '#00695c', fg: '#e0f2f1' },
    HUMILIATION: { bg: '#6a1b9a', fg: '#f3e5f5' },
    'DÉBÂCLE': { bg: '#bf360c', fg: '#fbe9e7' },
  };

  const showBadgeCol = data.some((r) => r.badge);

  return data.map((r) => {
    const cells: Array<{ html: string; style?: string }> = [
      {
        html: `<a href='/?page=Explorer&match_id=${r.id}' target='_self' style='font-family:monospace;font-size:0.75em;color:#33d6ff;text-decoration:none;'>${r.id.slice(0, 8)}…</a>`,
      },
      { html: r.date, style: 'white-space: nowrap' },
      { html: r.mode },
      { html: r.map },
      { html: r.score, style: 'font-weight: 600' },
      { html: r.kda },
      { html: r.kd, style: kdRatioColor(r.kd) },
      { html: r.duration },
    ];
    if (showBadgeCol) {
      const bcol = badgeColors[r.badge];
      const badgeHtml = r.badge && bcol
        ? `<span style="padding:1px 6px;border-radius:3px;font-size:0.75em;font-weight:600;background:${bcol.bg};color:${bcol.fg}">${r.badge}</span>`
        : '';
      cells.push({ html: badgeHtml });
    }
    return { cells };
  });
}

function kdRatioColor(kd: string): string {
  const v = parseFloat(kd);
  if (Number.isNaN(v)) return '';
  if (v >= 1.5) return 'color: #3DFFB5; font-weight: 700;';
  if (v <= 0.67) return 'color: #FF4D6D; font-weight: 700;';
  return '';
}

function mockEncountersRows(): TableSpec['rows'] {
  const data = [
    { gt: 'XxShadowKnightxX', total: 14, ally: 5, enemy: 9, wr_ally: '60%', wr_enemy: '44%', kd: '23/19', last: 'il y a 2j' },
    { gt: 'BlazingFury', total: 11, ally: 8, enemy: 3, wr_ally: '75%', wr_enemy: '67%', kd: '12/8', last: 'il y a 5j' },
    { gt: 'NeoSpartan_42', total: 9, ally: 2, enemy: 7, wr_ally: '50%', wr_enemy: '29%', kd: '8/22', last: 'hier' },
    { gt: 'GlitchHunter', total: 7, ally: 4, enemy: 3, wr_ally: '50%', wr_enemy: '33%', kd: '11/9', last: 'il y a 3j' },
    { gt: 'NoScope_King', total: 5, ally: 1, enemy: 4, wr_ally: '100%', wr_enemy: '50%', kd: '7/15', last: 'il y a 1sem' },
  ];

  return data.map((r) => {
    const sideHtml = r.ally > r.enemy
      ? "<span style='background:#1b5e20;color:#a5d6a7;padding:2px 8px;border-radius:10px;font-size:0.75em'>allié</span>"
      : "<span style='background:#5d2828;color:#f5b7b7;padding:2px 8px;border-radius:10px;font-size:0.75em'>ennemi</span>";

    const wrColor = (wr: string): string => {
      const v = parseInt(wr, 10);
      if (v >= 60) return 'color:#3DFFB5;font-weight:700';
      if (v <= 40) return 'color:#FF9E6B;font-weight:700';
      return '';
    };

    return {
      cells: [
        {
          html: `<a href='/?page=Explorer&gamertag=${r.gt}' target='_self' style='color:#33d6ff;text-decoration:none;font-weight:600'>${r.gt}</a> <span style='color:#888;font-size:0.85em'>(${r.total}e)</span>`,
        },
        { html: sideHtml },
        {
          html: `${r.total} <span style='color:#888;font-size:0.8em;'>(A:${r.ally} | E:${r.enemy})</span>`,
        },
        { html: r.wr_ally, style: wrColor(r.wr_ally) },
        { html: r.wr_enemy, style: wrColor(r.wr_enemy) },
        { html: r.kd },
        { html: r.last, style: 'color: #aaa; font-size: 0.85em' },
      ],
    };
  });
}

function mockAntagonistRows(isNemesis: boolean): TableSpec['rows'] {
  const data = isNemesis
    ? [
        { gt: 'XxShadowKnightxX', main: 28, sec: 14, net: -14, matches: 9 },
        { gt: 'BlazingFury_007', main: 22, sec: 16, net: -6, matches: 7 },
        { gt: 'NeoSpartan_42', main: 19, sec: 12, net: -7, matches: 8 },
        { gt: 'GlitchHunter', main: 17, sec: 9, net: -8, matches: 6 },
        { gt: 'NoScope_King', main: 15, sec: 13, net: -2, matches: 5 },
      ]
    : [
        { gt: 'EasyTarget_99', main: 32, sec: 8, net: 24, matches: 11 },
        { gt: 'NewbieGamer', main: 28, sec: 11, net: 17, matches: 9 },
        { gt: 'JustLooking', main: 24, sec: 14, net: 10, matches: 8 },
        { gt: 'SlowReflexes', main: 21, sec: 13, net: 8, matches: 7 },
        { gt: 'ChillPlayer42', main: 19, sec: 12, net: 7, matches: 6 },
      ];

  const netStyle = (kills: number, killedBy: number): string => {
    if (killedBy === 0) return kills > 0 ? 'color:#33ffbf;font-weight:700;' : '';
    const ratio = kills / killedBy;
    if (ratio >= 1.5) return 'color:#33ffbf;font-weight:700;';
    if (ratio <= 0.5) return 'color:#ff9e6b;font-weight:700;';
    return '';
  };

  return data.map((r, i) => {
    // En mode nemesis : main=times_killed_by, sec=times_killed
    // En mode victim : main=times_killed, sec=times_killed_by
    const kills = isNemesis ? r.sec : r.main;
    const killedBy = isNemesis ? r.main : r.sec;
    const netSign = r.net > 0 ? '+' : '';
    const ns = netStyle(kills, killedBy);
    return {
      cells: [
        { html: String(i + 1), style: 'text-align: left' },
        {
          html: `<a href='/?page=Explorer&gamertag=${r.gt}' target='_self' style='color:#33d6ff;text-decoration:none;font-weight:600'>${r.gt}</a>`,
          style: 'text-align: left',
        },
        { html: String(r.main), style: 'text-align: center; font-weight: 700' },
        { html: String(r.sec), style: 'text-align: center' },
        { html: `${netSign}${r.net}`, style: `text-align: center; ${ns}` },
        { html: String(r.matches), style: 'text-align: center; color: #aaa' },
      ],
    };
  });
}

function buildMockLegend(specId: string): string {
  if (specId === 'career.05') {
    return `<div style="font-size:0.78em;opacity:0.72;margin-top:4px;line-height:1.8;">
      <span style="padding:1px 6px;border-radius:3px;font-size:0.75em;font-weight:600;background:#2e7d32;color:#e8f5e9">DOMINATION</span> Victoire écrasante (≥30 pts d'écart)<br>
      <span style="padding:1px 6px;border-radius:3px;font-size:0.75em;font-weight:600;background:#1565c0;color:#e3f2fd">REMONTADA</span> Victoire après être mené tard<br>
      <span style="padding:1px 6px;border-radius:3px;font-size:0.75em;font-weight:600;background:#00695c;color:#e0f2f1">CONTRE-REMONTADA</span> Empêcher l'adversaire de remonter
    </div>`;
  }
  if (specId === 'career.06') {
    return `<div style="font-size:0.78em;opacity:0.72;margin-top:4px;line-height:1.8;">
      <span style="padding:1px 6px;border-radius:3px;font-size:0.75em;font-weight:600;background:#6a1b9a;color:#f3e5f5">HUMILIATION</span> Défaite écrasante (≥30 pts d'écart)<br>
      <span style="padding:1px 6px;border-radius:3px;font-size:0.75em;font-weight:600;background:#bf360c;color:#fbe9e7">DÉBÂCLE</span> Défaite avec abandon
    </div>`;
  }
  return '';
}
