import { readFileSync, existsSync } from 'node:fs';
import { resolve, dirname, join } from 'node:path';
import { parse } from 'yaml';
import type { ChartSpec, ThemeDefault } from './types.js';

/**
 * Charge un fichier YAML de spec graphique.
 */
export function loadChartSpec(yamlPath: string): { spec: ChartSpec; sourcePath: string } {
  const absPath = resolve(yamlPath);
  const raw = readFileSync(absPath, 'utf-8');
  const spec = parse(raw) as ChartSpec;
  if (!spec.id || !spec.chart_kind) {
    throw new Error(
      `YAML malformé : ${absPath} — champs requis manquants (id, chart_kind).`,
    );
  }
  return { spec, sourcePath: absPath };
}

/**
 * Charge le `_theme_default.yaml`. Cherche d'abord dans le même dossier parent que le YAML chart,
 * remonte jusqu'à `.ai/charts_specs/`.
 */
export function loadThemeDefault(chartYamlPath: string): ThemeDefault {
  let dir = dirname(resolve(chartYamlPath));
  const tried: string[] = [];
  for (let i = 0; i < 6; i++) {
    const candidate = join(dir, '_theme_default.yaml');
    tried.push(candidate);
    if (existsSync(candidate)) {
      try {
        const raw = readFileSync(candidate, 'utf-8');
        return parse(raw) as ThemeDefault;
      } catch (err) {
        throw new Error(
          `_theme_default.yaml trouvé à ${candidate} mais parsing échoué : ${(err as Error).message}`,
        );
      }
    }
    const parent = dirname(dir);
    if (parent === dir) break;
    dir = parent;
  }
  throw new Error(
    `_theme_default.yaml introuvable. Tentés :\n  ${tried.join('\n  ')}`,
  );
}

/**
 * Résout les tokens de palette `{{palette.X}}` ou `{{palette.okabe_ito.Y}}`
 * vers leur valeur hex. Laisse les tokens inconnus tels quels (sera signalé en warning).
 */
export function resolvePaletteToken(
  token: string | null | undefined,
  theme: ThemeDefault,
): string | null {
  if (!token) return null;
  const m = token.match(/^\{\{(palette|special_colors)\.([\w.]+)\}\}$/);
  if (!m) {
    // Soit déjà une couleur hex (#RRGGBB), soit token inconnu — retour as-is
    return token;
  }
  const [, source, path] = m;
  const root = source === 'special_colors' ? theme.special_colors : theme.palette;
  const parts = path.split('.');
  let current: unknown = root;
  for (const part of parts) {
    if (current && typeof current === 'object' && part in (current as object)) {
      current = (current as Record<string, unknown>)[part];
    } else {
      return token; // token non résolvable, retour as-is
    }
  }
  return typeof current === 'string' ? current : token;
}

/**
 * Résout une référence d'i18n `{{viz_t.key}}` ou `{{t.key}}` vers une clé brute
 * lisible humainement, en attendant un vrai système d'i18n côté client.
 *
 * Exemple : `{{viz_t.trace_wins}}` → `i18n:viz_t.trace_wins` (placeholder lisible).
 */
export function resolveI18nToken(token: string | null | undefined): string | null {
  if (!token) return null;
  const m = token.match(/^\{\{(viz_t|t)\.(\w+)\}\}$/);
  if (!m) return token;
  return `i18n:${m[1]}.${m[2]}`;
}

/**
 * Résout une légende qui hérite : si `legend.inherits === "legend_horizontal_bottom"`,
 * récupère la config depuis le thème.
 */
export function resolveLegend(
  legend: unknown,
  theme: ThemeDefault,
): Record<string, unknown> | null {
  if (!legend) return null;
  if (typeof legend !== 'object') return null;
  const lg = legend as Record<string, unknown>;
  if (typeof lg.inherits === 'string') {
    const themeKey = lg.inherits as keyof ThemeDefault;
    const inherited = theme[themeKey];
    if (inherited && typeof inherited === 'object') {
      return inherited as Record<string, unknown>;
    }
    return null;
  }
  return lg;
}

/**
 * Évalue une expression simple `max(320, 70 * len(metrics))` avec un contexte donné.
 * Implémentation très limitée : juste les patterns rencontrés dans les YAML actuels.
 *
 * Limites : pas un évaluateur Python complet. Ne reconnaît que :
 *   - max(a, b * len(name))      → renvoie le plus grand entre a et b * ctx[name]
 *   - max(a, b)                   → simple
 *   - une valeur littérale numérique
 */
export function evaluateHeightExpression(
  expr: string | null | undefined,
  context: Record<string, number>,
): number | null {
  if (!expr) return null;
  const trimmed = expr.trim();

  // Pattern : max(<n1>, <n2> * len(<var>))
  const m1 = trimmed.match(/^max\(\s*(\d+)\s*,\s*(\d+)\s*\*\s*len\((\w+)\)\s*\)$/);
  if (m1) {
    const a = parseInt(m1[1], 10);
    const b = parseInt(m1[2], 10);
    const varName = m1[3];
    const len = context[varName] ?? 0;
    return Math.max(a, b * len);
  }
  // Pattern : max(<n1>, <n2>)
  const m2 = trimmed.match(/^max\(\s*(\d+)\s*,\s*(\d+)\s*\)$/);
  if (m2) {
    return Math.max(parseInt(m2[1], 10), parseInt(m2[2], 10));
  }
  // Pattern : nombre seul
  const m3 = trimmed.match(/^\d+$/);
  if (m3) return parseInt(trimmed, 10);

  return null;
}

/**
 * Détermine la hauteur finale du chart à partir du champ `layout.height`,
 * qui peut être : `value` literal, `expression`, ou `branches` (if-else).
 */
export function resolveHeight(
  height: ChartSpec['layout']['height'],
  context: Record<string, number>,
  theme: ThemeDefault,
  warnings: string[],
): number {
  if (!height) return theme.heights.default;
  // Priorité : value literal > expression > branches > default
  if (typeof height.value === 'number' && height.value > 0) {
    return height.value;
  }
  if (height.expression) {
    const evaluated = evaluateHeightExpression(height.expression, context);
    if (evaluated !== null) return evaluated;
    warnings.push(
      `height.expression non évaluable : "${height.expression}" — fallback sur theme.default (${theme.heights.default})`,
    );
    return theme.heights.default;
  }
  if (height.branches && height.branches.length > 0) {
    for (const branch of height.branches) {
      // 'else: true' = catch-all
      if (branch.else === true) return branch.height;
      if (typeof branch.when === 'string') {
        // Évaluation très limitée : "pivot.height > 10" avec context.pivot_height
        const m = branch.when.match(/^pivot\.height\s*>\s*(\d+)$/);
        if (m && context.pivot_height !== undefined) {
          if (context.pivot_height > parseInt(m[1], 10)) return branch.height;
        }
        // Sinon, on ne sait pas évaluer — passer
      }
    }
    // Aucun match — prendre la première branche
    warnings.push(
      `height.branches : aucune condition évaluée — fallback sur la 1ère branche`,
    );
    return height.branches[0].height;
  }
  return theme.heights.default;
}
