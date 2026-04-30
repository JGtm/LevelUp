#!/usr/bin/env node
import { readdirSync, mkdirSync, writeFileSync } from 'node:fs';
import { resolve, basename, join, dirname } from 'node:path';
import { loadChartSpec, loadThemeDefault } from './loader.js';
import { specToEChartsOption } from './converter.js';

interface CliArgs {
  paths: string[];
  all: boolean;
  outputDir?: string;
  data?: string;
}

function parseArgs(argv: string[]): CliArgs {
  const args: CliArgs = { paths: [], all: false };
  for (let i = 2; i < argv.length; i++) {
    const a = argv[i];
    if (a === '--all') args.all = true;
    else if (a === '--out') args.outputDir = argv[++i];
    else if (a === '--data') args.data = argv[++i];
    else args.paths.push(a);
  }
  return args;
}

function findYamlsInDir(dir: string): string[] {
  const entries = readdirSync(dir);
  const result: string[] = [];
  for (const e of entries) {
    if (e.startsWith('_') || e.startsWith('.')) continue; // skip _index.md, _theme, _schema
    if (e.endsWith('.yaml') || e.endsWith('.yml')) {
      result.push(join(dir, e));
    }
  }
  return result.sort();
}

function processOne(yamlPath: string, outputDir: string | undefined): { ok: boolean; warnings: string[] } {
  console.log(`\n→ ${yamlPath}`);
  try {
    const { spec, sourcePath } = loadChartSpec(yamlPath);
    const theme = loadThemeDefault(sourcePath);
    const option = specToEChartsOption(spec, theme, {});

    const warnings = option.__meta?.warnings ?? [];
    if (warnings.length > 0) {
      console.log(`  ⚠  ${warnings.length} warning(s) :`);
      for (const w of warnings) console.log(`     - ${w}`);
    } else {
      console.log(`  ✓ Pas de warning`);
    }

    // Stats rapides
    const seriesCount = Array.isArray(option.series) ? option.series.length : 0;
    console.log(`  Chart kind : ${spec.chart_kind}`);
    console.log(`  Series    : ${seriesCount}`);
    console.log(`  Height    : ${option.__meta?.height} px`);

    // Écriture
    const targetDir = outputDir ?? join(dirname(sourcePath), '..', '_generated', spec.page);
    mkdirSync(targetDir, { recursive: true });
    const outPath = join(
      targetDir,
      basename(yamlPath, yamlPath.endsWith('.yaml') ? '.yaml' : '.yml') + '.option.json',
    );
    writeFileSync(outPath, JSON.stringify(option, replacer, 2), 'utf-8');
    console.log(`  → ${outPath}`);

    return { ok: true, warnings };
  } catch (err) {
    console.error(`  ✗ Erreur : ${(err as Error).message}`);
    return { ok: false, warnings: [(err as Error).message] };
  }
}

/**
 * Replacer pour JSON.stringify — les fonctions formatter sont remplacées
 * par leur source string (les vrais consumers ECharts auront besoin de fonctions
 * mais on émet du JSON, donc on documente).
 */
function replacer(_key: string, value: unknown): unknown {
  if (typeof value === 'function') {
    return `[Function: ${value.toString().slice(0, 80).replace(/\n/g, ' ')}...]`;
  }
  return value;
}

function main(): void {
  const args = parseArgs(process.argv);

  if (args.paths.length === 0) {
    console.error('Usage : tsx src/cli.ts <yaml-path> [<yaml-path>...] [--out dir] [--all]');
    console.error('   ou : tsx src/cli.ts --all <directory>');
    process.exit(1);
  }

  const yamls: string[] = [];
  for (const p of args.paths) {
    const abs = resolve(p);
    try {
      const stat = readdirSync(abs); // si c'est un dir
      if (Array.isArray(stat)) {
        if (args.all) {
          yamls.push(...findYamlsInDir(abs));
        } else {
          console.error(`Argument "${p}" est un dossier — utiliser --all`);
          process.exit(1);
        }
      }
    } catch {
      // pas un dir, traité comme fichier
      yamls.push(abs);
    }
  }

  if (yamls.length === 0) {
    console.error('Aucun YAML à traiter.');
    process.exit(1);
  }

  console.log(`\n${yamls.length} YAML(s) à convertir.\n`);

  let totalWarnings = 0;
  let okCount = 0;
  for (const y of yamls) {
    const r = processOne(y, args.outputDir);
    if (r.ok) okCount++;
    totalWarnings += r.warnings.length;
  }

  console.log(`\n=== Résumé ===`);
  console.log(`OK       : ${okCount} / ${yamls.length}`);
  console.log(`Warnings : ${totalWarnings}`);
  process.exit(okCount === yamls.length ? 0 : 1);
}

main();
