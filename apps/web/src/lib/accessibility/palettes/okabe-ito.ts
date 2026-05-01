/**
 * palettes/okabe-ito.ts — Palette Okabe-Ito (2008).
 *
 * 8 couleurs distinguables par les principaux types de daltonisme
 * (protanopie, deutéranopie, tritanopie).
 *
 * Référence : Okabe M. & Ito K. (2008) "Color Universal Design (CUD)"
 * https://jfly.uni-koeln.de/color/
 *
 * Couleurs brutes :
 *   Black        #000000
 *   Orange       #E69F00
 *   Sky Blue     #56B4E9
 *   Bluish Green #009E73
 *   Yellow       #F0E442
 *   Blue         #0072B2
 *   Vermillion   #D55E00
 *   Reddish Pur  #CC79A7
 *
 * ⚠️  Ce fichier est la SEULE exception à la règle "zéro magic hex" —
 *     c'est sa raison d'être. Ne jamais importer ce fichier depuis un composant.
 */
import type { Palette } from '../semantic-tokens'

export const okabePalette: Palette = {
  // ── Perf tiers — ordinal 5 niveaux ─────────────────────────────────────────
  // Combine teinte ET luminosité pour rester lisible en monochrome
  'perf-tier-1': '#009E73', // Bluish Green — excellent
  'perf-tier-2': '#56B4E9', // Sky Blue
  'perf-tier-3': '#F0E442', // Yellow — milieu neutre
  'perf-tier-4': '#E69F00', // Orange
  'perf-tier-5': '#D55E00', // Vermillion — pire

  // ── Outcomes ────────────────────────────────────────────────────────────────
  'outcome-win':  '#009E73', // Bluish Green
  'outcome-loss': '#D55E00', // Vermillion
  'outcome-draw': '#56B4E9', // Sky Blue
  'outcome-dnf':  '#CC79A7', // Reddish Purple

  // ── Divergent ───────────────────────────────────────────────────────────────
  'divergent-pos':     '#009E73', // Bluish Green
  'divergent-neutral': '#888888', // Gris neutre (pas de connotation directionnelle)
  'divergent-neg':     '#D55E00', // Vermillion

  // ── Statuts UI ─────────────────────────────────────────────────────────────
  'success':     '#009E73', // Bluish Green
  'warning':     '#E69F00', // Orange
  'info':        '#56B4E9', // Sky Blue
  'destructive': '#D55E00', // Vermillion

  // ── Comparaisons ───────────────────────────────────────────────────────────
  'compare-a': '#0072B2', // Blue
  'compare-b': '#CC79A7', // Reddish Purple

  // ── Chart series — 7 couleurs OI (sans Black) + gris clair ─────────────────
  // Black (#000000) retiré : invisible sur fond sombre (défaut de l'app)
  // Remplacé par #BBBBBB (gris clair — visible sur fond sombre et clair)
  'chart-series-1': '#E69F00', // Orange
  'chart-series-2': '#56B4E9', // Sky Blue
  'chart-series-3': '#009E73', // Bluish Green
  'chart-series-4': '#F0E442', // Yellow
  'chart-series-5': '#0072B2', // Blue
  'chart-series-6': '#D55E00', // Vermillion
  'chart-series-7': '#CC79A7', // Reddish Purple
  'chart-series-8': '#BBBBBB', // Gris clair (substitut de Black)

  // ── Badges narratifs ────────────────────────────────────────────────────────
  // Texte calculé pour contraste WCAG AA sur le fond correspondant
  'narrative-dominant':              '#009E73', // Bluish Green
  'narrative-dominant-text':         '#000000', // noir sur vert clair
  'narrative-humiliation':           '#CC79A7', // Reddish Purple
  'narrative-humiliation-text':      '#000000', // noir (luminosité 0.62 > 0.45)
  'narrative-remontada':             '#56B4E9', // Sky Blue (remplace navy #0072B2 risqué)
  'narrative-remontada-text':        '#000000', // noir sur bleu ciel
  'narrative-debacle':               '#D55E00', // Vermillion
  'narrative-debacle-text':          '#FFFFFF', // blanc sur fond foncé
  'narrative-contre-remontada':      '#E69F00', // Orange (remplace cyan #33D6FF trop proche)
  'narrative-contre-remontada-text': '#000000', // noir sur orange

  // ── Badges encounter — Okabe-Ito daltonisme-safe ────────────────────────────
  'narrative-encounter-ally-plus':    '#009E73', // Bluish Green — allié positif
  'narrative-encounter-tough-enemy':  '#D55E00', // Vermillion   — ennemi dangereux
  'narrative-encounter-ordinal':      '#56B4E9', // Sky Blue     — compteur rencontres

  // ── Heatmaps ────────────────────────────────────────────────────────────────
  'heatmap-cold':           '#D55E00', // Vermillion — mauvais
  'heatmap-hot':            '#009E73', // Bluish Green — bon
  'heatmap-divergent-low':  '#D55E00', // K/D bas
  'heatmap-divergent-high': '#009E73', // K/D haut
}
