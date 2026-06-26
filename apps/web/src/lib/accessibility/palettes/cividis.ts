/**
 * palettes/cividis.ts — Palette Cividis (Nuñez et al. 2018).
 *
 * Palette séquentielle perceptuellement uniforme, conçue **explicitement**
 * pour la déficience visuelle des couleurs (CVD-safe). Monotone en luminance
 * (L*), idéale pour les ramps ordinaux (perf-tiers, heatmaps).
 *
 * Référence : Nuñez J. R., Anderton C. R., Renslow R. S. (2018)
 * "Optimizing colormaps with consideration for color vision deficiency
 *  to enable accurate interpretation of scientific data", PLOS ONE 13(7).
 *
 * Pour les couples binaires (win/loss, divergent, encounter, heatmap),
 * on utilise l'axe **blue/vermillion** — Cividis ne définit pas de
 * sémantique binaire, on emprunte la même bonne pratique que Okabe-Ito
 * (Wong 2011, Nature Methods).
 *
 * ⚠️  Ce fichier est la SEULE exception à la règle "zéro magic hex" —
 *     c'est sa raison d'être. Ne jamais importer ce fichier depuis un composant.
 */
import type { Palette } from '../semantic-tokens'

// Échantillons Cividis aux positions t = 0.00, 0.10, 0.25, 0.40, 0.50, 0.60, 0.75, 0.90, 1.00
// Source : nuñez/cividis lookup table (PLOS ONE 2018, supplément S1)
const CIVIDIS_DARKEST = '#00224E' // t=0.00
const CIVIDIS_T10     = '#1B3158' // t=0.10
const CIVIDIS_T25     = '#3F4A6B' // t=0.25
const CIVIDIS_T40     = '#575C6D' // t=0.40
const CIVIDIS_MID     = '#7C7B78' // t=0.50
const CIVIDIS_T60     = '#928D6B' // t=0.60
const CIVIDIS_T75     = '#B6A855' // t=0.75
const CIVIDIS_T90     = '#E1C752' // t=0.90
const CIVIDIS_LIGHTEST = '#FEE838' // t=1.00

// Couples binaires — axe blue/vermillion daltonisme-safe (Okabe-Ito source)
const SAFE_BLUE = '#0072B2'
const SAFE_VERMILLION = '#D55E00'
const SAFE_GREY = '#888888'

export const cividisPalette: Palette = {
  // ── Perf tiers — ramp Cividis monotone en L* (foncé = excellent) ───────────
  'perf-tier-1': CIVIDIS_DARKEST, // t=0.00 — excellent
  'perf-tier-2': CIVIDIS_T25,     // t=0.25
  'perf-tier-3': CIVIDIS_MID,     // t=0.50
  'perf-tier-4': CIVIDIS_T75,     // t=0.75
  'perf-tier-5': CIVIDIS_LIGHTEST, // t=1.00 — pire

  // ── Outcomes — axe blue/vermillion ────────────────────────────────────────
  'outcome-win':  SAFE_BLUE,
  'outcome-loss': SAFE_VERMILLION,
  'outcome-draw': CIVIDIS_LIGHTEST, // jaune Cividis (point neutre lumineux)
  'outcome-dnf':  CIVIDIS_T40,      // gris-bleu Cividis

  // ── Divergent ───────────────────────────────────────────────────────────────
  'divergent-pos':     SAFE_BLUE,
  'divergent-neutral': SAFE_GREY,
  'divergent-neg':     SAFE_VERMILLION,

  // ── Statuts UI ─────────────────────────────────────────────────────────────
  'success':     SAFE_BLUE,
  'warning':     CIVIDIS_T75, // ocre Cividis (entre jaune et orange)
  'info':        CIVIDIS_T40, // gris-bleu
  'destructive': SAFE_VERMILLION,

  // ── Comparaisons ───────────────────────────────────────────────────────────
  'compare-a': CIVIDIS_DARKEST,
  'compare-b': CIVIDIS_LIGHTEST,
  'compare-c': CIVIDIS_MID,     // t=0.50 — gris-brun neutre (troisième joueur)

  // ── Chart series — alterne extrémités Cividis pour distinguer 2 séries voisines
  'chart-series-1': CIVIDIS_DARKEST,  // t=0.00
  'chart-series-2': CIVIDIS_LIGHTEST, // t=1.00
  'chart-series-3': CIVIDIS_T25,      // t=0.25
  'chart-series-4': CIVIDIS_T75,      // t=0.75
  'chart-series-5': CIVIDIS_T40,      // t=0.40
  'chart-series-6': CIVIDIS_T60,      // t=0.60
  'chart-series-7': CIVIDIS_T10,      // t=0.10
  'chart-series-8': CIVIDIS_T90,      // t=0.90

  // ── Bonus (assistances) — ocre Cividis : meilleur compromis distinct du set
  //     joueurs {navy, jaune, gris, bleu} et non-vermillon (≠ morts). Ramp
  //     séquentielle bleu→jaune : pas de pourpre possible (limite CVD assumée).
  'bonus': CIVIDIS_T75, // #B6A855 (ocre)

  // ── Badges narratifs ────────────────────────────────────────────────────────
  // Texte calculé pour contraste WCAG AA (test automatique Phase C)
  'narrative-dominant':              CIVIDIS_LIGHTEST, // jaune clair
  'narrative-dominant-text':         '#000000',
  'narrative-humiliation':           CIVIDIS_DARKEST,  // bleu très foncé
  'narrative-humiliation-text':      '#FFFFFF',
  'narrative-remontada':             CIVIDIS_T25,      // bleu-gris foncé
  'narrative-remontada-text':        '#FFFFFF',
  'narrative-debacle':               SAFE_VERMILLION,  // vermillion (couple binaire)
  'narrative-debacle-text':          '#000000',        // noir sur vermillion (5.4) — blanc ne passait pas AA
  'narrative-contre-remontada':      CIVIDIS_T75,      // ocre
  'narrative-contre-remontada-text': '#000000',

  // ── Badges encounter — axe blue/vermillion ────────────────────────────────
  'narrative-encounter-ally-plus':    SAFE_BLUE,
  'narrative-encounter-tough-enemy':  SAFE_VERMILLION,
  'narrative-encounter-coriace':      CIVIDIS_T75, // ocre — winrate vs lui mauvais
  'narrative-encounter-ordinal':      CIVIDIS_T40,
  // Badges « solid » du hub Relations : positif = SAFE_BLUE (axe « bon » Cividis),
  // neutre = tons Cividis (pas de vert dans cette palette monochrome).
  'narrative-encounter-duo-gagnant':    SAFE_BLUE,
  'narrative-encounter-cameleon':       CIVIDIS_T40,
  'narrative-encounter-de-longue-date': CIVIDIS_T25,
  'narrative-encounter-recrue':         CIVIDIS_T60,
  'narrative-encounter-proie-favorite': SAFE_BLUE,

  // ── Heatmaps — axe blue/vermillion ────────────────────────────────────────
  'heatmap-cold':           SAFE_VERMILLION,
  'heatmap-hot':            SAFE_BLUE,
  'heatmap-divergent-low':  SAFE_VERMILLION,
  'heatmap-divergent-high': SAFE_BLUE,

  // ── Équipes — axe blue/vermillion CVD-safe ─────────────────────────────────
  'team-ally':  SAFE_BLUE,       // overridable via settings accessibilité
  'team-enemy': SAFE_VERMILLION, // overridable via settings accessibilité
}
