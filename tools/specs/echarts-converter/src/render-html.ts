#!/usr/bin/env node
/**
 * Génère un fichier HTML standalone qui rend toutes les charts d'une page
 * avec ECharts (via CDN). Ouvrir le fichier dans un navigateur pour voir
 * le résultat visuel.
 *
 * Usage : tsx src/render-html.ts <yaml-dir> [--out <output.html>]
 */

import { readdirSync, mkdirSync, writeFileSync } from 'node:fs';
import { resolve, basename, join, dirname } from 'node:path';
import { loadChartSpec, loadThemeDefault } from './loader.js';
import { specToEChartsOption } from './converter.js';
import { serializeJS } from './serialize.js';

interface RenderArgs {
  inputDir: string;
  outputPath: string;
}

function parseArgs(argv: string[]): RenderArgs {
  let inputDir = '';
  let outputPath = '';
  for (let i = 2; i < argv.length; i++) {
    const a = argv[i];
    if (a === '--out') outputPath = argv[++i];
    else if (!inputDir) inputDir = a;
  }
  if (!inputDir) {
    console.error('Usage : tsx src/render-html.ts <yaml-dir> [--out <output.html>]');
    process.exit(1);
  }
  return { inputDir, outputPath };
}

function findYamlsInDir(dir: string): string[] {
  return readdirSync(dir)
    .filter((e) => !e.startsWith('_') && !e.startsWith('.'))
    .filter((e) => e.endsWith('.yaml') || e.endsWith('.yml'))
    .map((e) => join(dir, e))
    .sort();
}

interface ChartItem {
  id: string;
  title: string;
  option: unknown;
  height: number;
  sourceFunc: string;
  isTable: boolean;
  isComposite: boolean;
}

function renderHTML(charts: ChartItem[], pageTitle: string): string {
  const chartBlocks = charts
    .map((c, idx) => {
      // Si c'est un composite UI (kpi_row, composite_block) : on rend du HTML pur
      if (c.isComposite) {
        const opt = c.option as { __composite?: import('./converters/composite-ui.js').CompositeUiSpec };
        const comp = opt.__composite;
        if (!comp) {
          return `
    <section class="chart-card">
      <h2>${c.id} — ${escapeHtml(c.title)}</h2>
      <div class="meta"><code>${escapeHtml(c.sourceFunc)}</code></div>
      <div class="error">Composite spec manquant.</div>
    </section>`;
        }
        return `
    <section class="chart-card chart-card--composite">
      <h2>${c.id} — ${escapeHtml(c.title)}</h2>
      <div class="meta">
        <code>${escapeHtml(c.sourceFunc)}</code>
      </div>
      ${renderCompositeUi(comp)}
    </section>`;
      }

      // Si c'est un tableau HTML : on rend la <table> directement, pas d'init ECharts.
      if (c.isTable) {
        const opt = c.option as { __table?: import('./converters/table-html.js').TableSpec };
        const tableSpec = opt.__table;
        if (!tableSpec) {
          return `
    <section class="chart-card">
      <h2>${c.id} — ${escapeHtml(c.title)}</h2>
      <div class="meta"><code>${escapeHtml(c.sourceFunc)}</code></div>
      <div class="error">Table spec manquant.</div>
    </section>`;
        }
        return `
    <section class="chart-card chart-card--table">
      <h2>${c.id} — ${escapeHtml(c.title)}</h2>
      <div class="meta">
        <code>${escapeHtml(c.sourceFunc)}</code>
      </div>
      ${renderTableHtml(tableSpec)}
    </section>`;
      }

      // Sinon : chart ECharts standard
      const optionJS = serializeJS(c.option, 2);
      return `
    <section class="chart-card">
      <h2>${c.id} — ${escapeHtml(c.title)}</h2>
      <div class="meta">
        <code>${escapeHtml(c.sourceFunc)}</code>
      </div>
      <div id="chart-${idx}" class="chart-canvas" style="height: ${c.height}px;"></div>
      <script>
        (function() {
          const option = ${optionJS};
          const dom = document.getElementById('chart-${idx}');
          const chart = echarts.init(dom, null, { renderer: 'canvas' });
          chart.setOption(option);
          window.addEventListener('resize', () => chart.resize());
        })();
      </script>
    </section>`;
    })
    .join('\n');

  return `<!DOCTYPE html>
<html lang="fr">
<head>
  <meta charset="UTF-8">
  <title>Mock ECharts — ${escapeHtml(pageTitle)}</title>
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <script src="https://cdn.jsdelivr.net/npm/echarts@5.5.1/dist/echarts.min.js"></script>
  <style>
    body {
      font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
      margin: 0;
      padding: 24px;
      background: rgb(20, 28, 33);
      color: rgba(245, 248, 255, 0.92);
    }
    header {
      max-width: 1400px;
      margin: 0 auto 24px;
      padding-bottom: 16px;
      border-bottom: 1px solid rgba(255, 255, 255, 0.12);
    }
    header h1 {
      margin: 0 0 8px;
      font-size: 24px;
      font-weight: 500;
    }
    header .info {
      font-size: 13px;
      color: rgba(182, 196, 214, 0.88);
    }
    main {
      max-width: 1400px;
      margin: 0 auto;
      display: grid;
      grid-template-columns: 1fr;
      gap: 24px;
    }
    @media (min-width: 1100px) {
      main {
        grid-template-columns: repeat(2, 1fr);
      }
      .chart-card.full-width {
        grid-column: 1 / -1;
      }
    }
    .chart-card {
      background: rgba(29, 35, 40, 1.0);
      border: 1px solid rgba(255, 255, 255, 0.08);
      border-radius: 8px;
      padding: 16px 16px 8px;
    }
    .chart-card h2 {
      margin: 0 0 4px;
      font-size: 16px;
      font-weight: 500;
      color: #33D6FF;
    }
    .chart-card .meta {
      font-size: 11px;
      color: rgba(182, 196, 214, 0.72);
      margin-bottom: 12px;
    }
    .chart-card .meta code {
      background: rgba(255, 255, 255, 0.04);
      padding: 2px 6px;
      border-radius: 3px;
      font-size: 11px;
    }
    .chart-canvas {
      width: 100%;
    }
    .chart-card--table .os-table-wrap {
      overflow-x: auto;
      margin-top: 8px;
    }
    table.os-table {
      width: 100%;
      border-collapse: collapse;
      font-size: 0.86em;
      background: rgba(29, 35, 40, 0.96);
    }
    table.os-table th, table.os-table td {
      padding: 6px 10px;
      border-bottom: 1px solid rgba(255, 255, 255, 0.05);
      text-align: left;
      vertical-align: middle;
    }
    table.os-table th.os-sb-team {
      background: rgba(51, 214, 255, 0.08);
      color: #33D6FF;
      font-weight: 600;
      font-size: 0.95em;
      padding: 8px 12px;
      text-align: left;
      border-bottom: 1px solid rgba(51, 214, 255, 0.2);
    }
    table.os-table th.os-sb-th {
      color: rgba(182, 196, 214, 0.88);
      font-size: 0.78em;
      font-weight: 500;
      text-transform: uppercase;
      letter-spacing: 0.04em;
      border-bottom: 1px solid rgba(255, 255, 255, 0.12);
    }
    table.os-table tr.os-sb-row:hover {
      background: rgba(255, 255, 255, 0.03);
    }
    table.os-table td.os-sb-td {
      color: rgba(245, 248, 255, 0.92);
    }
    .table-legend {
      margin-top: 6px;
      padding: 6px 10px;
      background: rgba(255, 255, 255, 0.02);
      border-radius: 4px;
    }
    .kpi-row {
      display: grid;
      grid-template-columns: repeat(4, 1fr);
      gap: 8px;
      margin-top: 8px;
    }
    .kpi-tile {
      border: 1px solid rgba(255, 255, 255, 0.12);
      border-radius: 6px;
      padding: 9px 15px;
      text-align: center;
      background: rgba(255, 255, 255, 0.02);
      min-height: 60px;
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
    }
    .kpi-tile-text {
      font-family: 'Segoe UI', sans-serif;
      font-size: 22px;
      font-weight: 400;
      letter-spacing: 0.02em;
      color: rgba(255, 255, 255, 0.98);
    }
    .composite-block {
      margin-top: 8px;
      padding: 4px 0;
    }
    .composite-block .cb-cell img {
      max-width: 100%;
      border-radius: 6px;
    }
    footer {
      max-width: 1400px;
      margin: 24px auto 0;
      padding-top: 16px;
      border-top: 1px solid rgba(255, 255, 255, 0.08);
      font-size: 12px;
      color: rgba(182, 196, 214, 0.72);
    }
  </style>
</head>
<body>
  <header>
    <h1>Mock ECharts — Page ${escapeHtml(pageTitle)}</h1>
    <div class="info">
      ${charts.length} charts · Données mock synthétiques · ECharts 5.5.1<br>
      Source : <code>origin/v7/cockpit</code> (Plotly Python) — généré le ${new Date().toISOString()}
    </div>
  </header>
  <main>${chartBlocks}
  </main>
  <footer>
    <strong>But</strong> : valider visuellement que les YAML de spec graphique
    (<code>.ai/charts_specs/${escapeHtml(pageTitle)}/*.yaml</code>) reproduisent fidèlement
    les charts Plotly d'origine. Comparer chaque chart ci-dessus avec son équivalent
    Streamlit/Plotly en lançant l'app Python (si dispo) ou en regardant un screenshot.
  </footer>
</body>
</html>
`;
}

function escapeHtml(s: string): string {
  return s.replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c] as string));
}

function renderCompositeUi(spec: import('./converters/composite-ui.js').CompositeUiSpec): string {
  if (spec.__kind === 'kpi_row') {
    // 4 tiles en grid
    const tilesHtml = spec.blocks
      .map((b) => {
        const t = b as import('./converters/composite-ui.js').KpiTileSpec;
        if (t.type === 'enriched') {
          const badgeHtml = t.badge
            ? `<div style="display:block;margin-top:1px;padding:3px 10px;border-radius:4px;font-size:0.875em;font-weight:600;background:${t.badge.bg};color:${t.badge.fg}">${escapeHtml(t.badge.label)}</div>`
            : '';
          return `<div class="kpi-tile">
            <div style="font-family:'Segoe UI',sans-serif;font-size:38px;font-weight:700;line-height:1;color:${t.scoreColor || '#fff'}">${escapeHtml(t.scoreText || '')}</div>
            ${badgeHtml}
          </div>`;
        }
        return `<div class="kpi-tile"><div class="kpi-tile-text">${escapeHtml(t.text || '—')}</div></div>`;
      })
      .join('');
    return `<div class="kpi-row">${tilesHtml}</div>`;
  }

  if (spec.__kind === 'composite_block') {
    const totalRatio = spec.blocks.reduce((sum, b) => sum + (b as import('./converters/composite-ui.js').CompositeBlockTileSpec).columnRatio, 0);
    const cellsHtml = spec.blocks
      .map((b) => {
        const tb = b as import('./converters/composite-ui.js').CompositeBlockTileSpec;
        const widthPct = ((tb.columnRatio / totalRatio) * 100).toFixed(2);
        if (tb.type === 'image') {
          return `<div class="cb-cell" style="flex:0 0 ${widthPct}%;padding:0 8px">${tb.payload}</div>`;
        }
        if (tb.type === 'text_coloured') {
          return `<div class="cb-cell" style="flex:0 0 ${widthPct}%;padding:0 8px;display:flex;align-items:center;justify-content:center"><div style="font-size:48px;font-weight:700;color:${tb.color || '#fff'}">${escapeHtml(tb.payload)}</div></div>`;
        }
        if (tb.type === 'raw_html') {
          return `<div class="cb-cell" style="flex:0 0 ${widthPct}%;padding:0 8px">${tb.payload}</div>`;
        }
        return '';
      })
      .join('');
    return `<div class="composite-block" style="display:flex;align-items:center;gap:8px">${cellsHtml}</div>`;
  }

  return '';
}

function renderTableHtml(spec: import('./converters/table-html.js').TableSpec): string {
  const cs = spec.cssClasses;
  const tableStyle = spec.inlineStyles?.table ? ` style="${spec.inlineStyles.table}"` : '';

  // Headers
  const captionRow = `<tr><th class="${cs.headerTeam}" colspan="${spec.caption.colspan}">${escapeHtml(spec.caption.text)}</th></tr>`;
  const headerCells = spec.columns
    .map((c) => {
      const style = c.style ? ` style="${c.style}"` : '';
      return `<th class="${cs.headerCell}"${style}>${escapeHtml(c.header)}</th>`;
    })
    .join('');
  const headerRow = `<tr>${headerCells}</tr>`;
  const thead = `<thead>${captionRow}${headerRow}</thead>`;

  // Body
  const bodyRows = spec.rows
    .map((row) => {
      if (row.degradedHtml) return row.degradedHtml;
      const cells = row.cells
        .map((cell) => {
          const style = cell.style ? ` style="${cell.style}"` : '';
          return `<td class="${cs.cell}"${style}>${cell.html}</td>`;
        })
        .join('');
      return `<tr class="${cs.row}">${cells}</tr>`;
    })
    .join('');
  const tbody = `<tbody>${bodyRows}</tbody>`;

  const legendBlock =
    spec.legend?.enabled && spec.legend.html
      ? `<div class="table-legend">${spec.legend.html}</div>`
      : '';

  return `
      <div class="${cs.wrapper}">
        <table class="${cs.table}"${tableStyle}>
          ${thead}
          ${tbody}
        </table>
      </div>
      ${legendBlock}`;
}

function main(): void {
  const args = parseArgs(process.argv);
  const dirAbs = resolve(args.inputDir);
  const yamls = findYamlsInDir(dirAbs);
  if (yamls.length === 0) {
    console.error(`Aucun YAML dans ${dirAbs}`);
    process.exit(1);
  }

  const pageTitle = basename(dirAbs);
  console.log(`\nRendu de ${yamls.length} charts depuis ${dirAbs}...\n`);

  const charts: ChartItem[] = [];
  for (const yp of yamls) {
    try {
      const { spec, sourcePath } = loadChartSpec(yp);
      const theme = loadThemeDefault(sourcePath);
      const option = specToEChartsOption(spec, theme, {});
      const meta = (option as { __meta?: { height: number; warnings: string[]; chart_kind?: string } }).__meta;
      const isTable = spec.chart_kind === 'table_html';
      const isComposite =
        spec.chart_kind === 'kpi_row' || spec.chart_kind === 'composite_block';
      // On retire __meta avant rendu pour ECharts (mais on conserve pour tables/composites)
      if (!isTable && !isComposite) {
        delete (option as unknown as Record<string, unknown>).__meta;
      }
      charts.push({
        id: spec.id,
        title: spec.title,
        option,
        height: meta?.height ?? 360,
        sourceFunc: spec.source_function,
        isTable,
        isComposite,
      });
      const kindMark = isTable ? ' (table)' : isComposite ? ' (composite)' : '';
      console.log(`  ✓ ${spec.id} — ${spec.title}${kindMark}`);
    } catch (err) {
      console.error(`  ✗ ${yp} : ${(err as Error).message}`);
    }
  }

  const outPath = args.outputPath
    ? resolve(args.outputPath)
    : join(dirname(dirAbs), '_generated', pageTitle, 'mock-echarts.html');
  mkdirSync(dirname(outPath), { recursive: true });
  writeFileSync(outPath, renderHTML(charts, pageTitle), 'utf-8');

  console.log(`\nHTML généré : ${outPath}`);
  console.log('Ouvre ce fichier dans ton navigateur pour voir les charts.\n');
}

main();
