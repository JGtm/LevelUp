/**
 * palettes/tol-bright.ts — Palette Tol Bright (Paul Tol 2018).
 *
 * Palette catégorielle 7 couleurs optimisée daltonisme — alternative à
 * Okabe-Ito quand on dépasse 4 séries simultanées (plus discriminable
 * en deutéranopie/protanopie sur les paires lointaines).
 *
 * Référence : Tol P. (2018) "Colour schemes",
 *   technical note SRON/EPS/TN/09-002.
 *   https://personal.sron.nl/~pault/data/colourschemes.pdf
 *
 * Couleurs brutes (Bright scheme) :
 *   Blue    #4477AA
 *   Cyan    #66CCEE
 *   Green   #228833
 *   Yellow  #CCBB44
 *   Red     #EE6677
 *   Purple  #AA3377
 *   Grey    #BBBBBB
 *
 * Pour les ramps ordinaux (perf-tier), on utilise Tol's "Sunset" sequential
 * scheme (5 échantillons), également CVD-safe par construction.
 *
 * ⚠️  Ce fichier est la SEULE exception à la règle "zéro magic hex" —
 *     c'est sa raison d'être. Ne jamais importer ce fichier depuis un composant.
 */
import type { Palette } from '../semantic-tokens'

// Tol Bright — catégorielles
const TOL_BLUE   = '#4477AA'
const TOL_CYAN   = '#66CCEE'
const TOL_GREEN  = '#228833'
const TOL_YELLOW = '#CCBB44'
const TOL_RED    = '#EE6677'
const TOL_PURPLE = '#AA3377'
const TOL_GREY   = '#BBBBBB'

// Tol Sunset — séquentielle 5 niveaux (CVD-safe, monotone en L*)
const SUNSET_1 = '#364B9A' // bleu profond — meilleur
const SUNSET_2 = '#4A7BB7'
const SUNSET_3 = '#98C2E0' // bleu clair
const SUNSET_4 = '#FAB57F' // orange clair
const SUNSET_5 = '#DC4A38' // rouge — pire

// Couleurs neutres / extension 8e série (Tol Bright n'en a que 7)
const NEUTRAL_GREY = '#888888'
const TOL_BLACK = '#000000'

export const tolBrightPalette: Palette = {
  // ── Perf tiers — Tol Sunset (séquentielle CVD-safe) ───────────────────────
  'perf-tier-1': SUNSET_1, // bleu profond — excellent
  'perf-tier-2': SUNSET_2,
  'perf-tier-3': SUNSET_3, // bleu clair (milieu)
  'perf-tier-4': SUNSET_4, // orange clair
  'perf-tier-5': SUNSET_5, // rouge — pire

  // ── Outcomes — axe blue/red Tol Bright ─────────────────────────────────────
  'outcome-win':  TOL_BLUE,
  'outcome-loss': TOL_RED,
  'outcome-draw': TOL_YELLOW,
  'outcome-dnf':  TOL_PURPLE,

  // ── Divergent ───────────────────────────────────────────────────────────────
  'divergent-pos':     TOL_BLUE,
  'divergent-neutral': NEUTRAL_GREY,
  'divergent-neg':     TOL_RED,

  // ── Statuts UI ─────────────────────────────────────────────────────────────
  'success':     TOL_GREEN,  // vert UI conventionnel (statut, pas binaire)
  'warning':     TOL_YELLOW,
  'info':        TOL_CYAN,
  'destructive': TOL_RED,

  // ── Comparaisons ───────────────────────────────────────────────────────────
  'compare-a': TOL_BLUE,
  'compare-b': TOL_PURPLE,
  'compare-c': TOL_GREEN,

  // ── Chart series — Tol Bright 7 + black en s8 ─────────────────────────────
  // Ordre Tol officiel : Blue, Red, Green, Yellow, Cyan, Purple, Grey + extension
  'chart-series-1': TOL_BLUE,
  'chart-series-2': TOL_RED,
  'chart-series-3': TOL_GREEN,
  'chart-series-4': TOL_YELLOW,
  'chart-series-5': TOL_CYAN,
  'chart-series-6': TOL_PURPLE,
  'chart-series-7': TOL_GREY,
  'chart-series-8': TOL_BLACK,

  // ── Bonus (assistances) — pourpre Tol, standout cohérent avec le défaut,
  //     distinct des joueurs squad (bleu/vert/bleu-clair).
  'bonus': TOL_PURPLE, // #AA3377

  // ── Badges narratifs ────────────────────────────────────────────────────────
  // Texte calculé pour contraste WCAG AA (test automatique Phase C)
  'narrative-dominant':              TOL_GREEN,  // vert UI — statut positif
  'narrative-dominant-text':         '#FFFFFF',
  'narrative-humiliation':           TOL_PURPLE,
  'narrative-humiliation-text':      '#FFFFFF',
  'narrative-remontada':             TOL_BLUE,
  'narrative-remontada-text':        '#FFFFFF',
  'narrative-debacle':               TOL_RED,
  'narrative-debacle-text':          '#000000', // noir sur Tol Red (6.8) — blanc ne passait pas AA (3.09)
  'narrative-contre-remontada':      TOL_CYAN,   // cyan clair
  'narrative-contre-remontada-text': '#000000',

  // ── Badges encounter ───────────────────────────────────────────────────────
  'narrative-encounter-ally-plus':    TOL_BLUE,
  'narrative-encounter-tough-enemy':  TOL_RED,    // K/D contre nous mauvais
  'narrative-encounter-coriace':      TOL_PURPLE, // winrate vs lui mauvais (Coriace)
  'narrative-encounter-ordinal':      TOL_CYAN,
  // Badges « solid » du hub Relations : vert = positif, bleu/cyan = neutre.
  'narrative-encounter-duo-gagnant':    TOL_GREEN,
  'narrative-encounter-cameleon':       TOL_BLUE,
  'narrative-encounter-de-longue-date': TOL_BLUE,
  'narrative-encounter-recrue':         TOL_CYAN,
  'narrative-encounter-proie-favorite': TOL_GREEN,

  // ── Heatmaps ────────────────────────────────────────────────────────────────
  'heatmap-cold':           TOL_RED,
  'heatmap-hot':            TOL_BLUE,
  'heatmap-divergent-low':  TOL_RED,
  'heatmap-divergent-high': TOL_BLUE,

  // ── Équipes — axe Blue/Red Tol Bright ─────────────────────────────────────
  'team-ally':  TOL_BLUE, // overridable via settings accessibilité
  'team-enemy': TOL_RED,  // overridable via settings accessibilité
}
