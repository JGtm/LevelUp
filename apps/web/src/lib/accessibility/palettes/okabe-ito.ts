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
import { ENCOUNTER_BADGE_COLORS } from './_encounterColors'

export const okabePalette: Palette = {
  // ── Perf tiers — ordinal 5 niveaux, ramp divergent bleu→jaune→vermillion ──
  // Re-mappé v2 : axe blue/orange (Wong 2011, Nature Methods) — l'ancien ramp
  // green→...→vermillion collapsait sur l'axe de confusion deutan.
  'perf-tier-1': '#0072B2', // Blue       — excellent
  'perf-tier-2': '#56B4E9', // Sky Blue
  'perf-tier-3': '#F0E442', // Yellow     — milieu (point neutre)
  'perf-tier-4': '#E69F00', // Orange
  'perf-tier-5': '#D55E00', // Vermillion — pire

  // ── Outcomes — axe blue/vermillion daltonisme-safe ─────────────────────────
  // Trade-off conscient : "win" est bleu (pas vert) sur cette palette ;
  // les composants conservent des indices secondaires (icônes, signes).
  'outcome-win':  '#0072B2', // Blue
  'outcome-loss': '#D55E00', // Vermillion
  'outcome-draw': '#F0E442', // Yellow (libère Sky Blue trop proche du Blue)
  // DNF = pas de résultat → teinte désaturée, comme dans Cividis (#575C6D).
  // Okabe-Ito n'a pas de gris canonique ; on éclaircit ce gris-bleu pour tenir
  // ≥ 3:1 sur les deux surfaces. Libère le Reddish Purple, qui servait aussi à
  // narrative-humiliation : deux verdicts de match ne peuvent plus être confondus.
  'outcome-dnf':  '#6E7B8B', // Gris-bleu désaturé

  // ── Divergent — axe blue/vermillion (Brewer RdBu inversé) ──────────────────
  'divergent-pos':     '#0072B2', // Blue
  'divergent-neutral': '#888888', // Gris (pas de connotation directionnelle)
  'divergent-neg':     '#D55E00', // Vermillion

  // ── Statuts UI ─────────────────────────────────────────────────────────────
  'success':     '#009E73', // Bluish Green (statut UI conventionnel — non binaire)
  'warning':     '#E69F00', // Orange
  'info':        '#56B4E9', // Sky Blue
  'destructive': '#D55E00', // Vermillion

  // ── Comparaisons ───────────────────────────────────────────────────────────
  'compare-a': '#0072B2', // Blue
  'compare-b': '#CC79A7', // Reddish Purple
  'compare-c': '#009E73', // Bluish Green

  // ── Joueurs d'escouade ─────────────────────────────────────────────────────
  // Contrainte du garde-rail (squadPlayerTokens.test.ts) : contraste ≥ 3:1 sur
  // les DEUX surfaces produit + ΔE OKLab toutes-paires ≥ 15 en vision normale
  // et ≥ 8 sous protanopie/deutéranopie. Deux teintes brutes ne peuvent pas
  // servir telles quelles, on garde la teinte et on baisse la luminosité :
  //   - Orange #E69F00 est trop clair (2.21 seulement sur la surface claire) ;
  //     Vermillion, qui passerait, est déjà outcome-loss / team-enemy → un
  //     coéquipier prendrait la couleur du camp adverse dans la vue Match ;
  //   - Reddish Purple #CC79A7 brut tombe à ΔE deutan 7.5 contre Bluish Green.
  // Le set retenu vaut ΔE min 17.4 (normal) / 9.6 (protan) / 10.5 (deutan) et
  // n'entre en collision avec aucun token outcome/narratif de cette palette.
  'squad-player-1': '#0072B2', // Blue                      — joueur principal
  'squad-player-2': '#B77E00', // Orange assombri  (× 0.60) — coéquipier 1
  'squad-player-3': '#8E5374', // Reddish Purple assombri   — coéquipier 2
  'squad-player-4': '#009E73', // Bluish Green              — coéquipier 3

  // ── Chart series — réordonnées pour distance maximale en deutan ────────────
  // 2 premières séries = paire la plus discriminable (Blue/Orange).
  // À 4 séries : Blue/Orange/SkyBlue/Vermillion — toutes paires ≥ 30 ΔE en deutan.
  // Black retiré (invisible sur fond sombre) → remplacé par gris clair en s8.
  'chart-series-1': '#0072B2', // Blue
  'chart-series-2': '#E69F00', // Orange
  'chart-series-3': '#56B4E9', // Sky Blue
  'chart-series-4': '#D55E00', // Vermillion
  'chart-series-5': '#009E73', // Bluish Green
  'chart-series-6': '#F0E442', // Yellow
  'chart-series-7': '#CC79A7', // Reddish Purple
  'chart-series-8': '#BBBBBB', // Gris clair (substitut de Black)

  // ── Bonus (assistances) — Reddish Purple : distinct du perf-tier-3 jaune et
  //     des joueurs squad (bleu/vert). Pas de collision en Okabe-Ito.
  'bonus': '#CC79A7', // Reddish Purple

  // ── Badges narratifs ────────────────────────────────────────────────────────
  // Texte calculé pour contraste WCAG AA sur le fond correspondant
  'narrative-dominant':              '#009E73', // Bluish Green
  'narrative-dominant-text':         '#000000', // noir sur vert clair
  'narrative-humiliation':           '#CC79A7', // Reddish Purple
  'narrative-humiliation-text':      '#000000', // noir (luminosité 0.62 > 0.45)
  'narrative-remontada':             '#56B4E9', // Sky Blue (remplace navy #0072B2 risqué)
  'narrative-remontada-text':        '#000000', // noir sur bleu ciel
  // Vermillion BRUT est déjà outcome-loss / team-enemy / divergent-neg : une
  // débâcle est un verdict qui se superpose à une défaite, les deux ne peuvent
  // pas porter exactement le même fond (même raison que outcome-dnf ↔
  // narrative-humiliation ci-dessus). Okabe-Ito n'ayant que 8 teintes toutes
  // affectées, on garde la teinte et on baisse la luminosité — technique déjà
  // employée pour squad-player-2 (« Orange assombri × 0.60 »). × 0.75 ici.
  'narrative-debacle':               '#A04700', // Vermillion assombri (× 0.75)
  'narrative-debacle-text':          '#FFFFFF', // blanc sur vermillion sombre (6.2) — noir tombe à 3.40
  'narrative-contre-remontada':      '#E69F00', // Orange (remplace cyan #33D6FF trop proche)
  'narrative-contre-remontada-text': '#000000', // noir sur orange

  // ── Badges encounter — axe blue/vermillion daltonisme-safe ────────────────
  // (set sombre distinct AA-blanc, palette-invariant — cf. _encounterColors.ts ;
  //  les teintes Okabe claires échouaient le texte blanc ; labels disambiguent)
  ...ENCOUNTER_BADGE_COLORS,

  // ── Heatmaps — axe blue/vermillion ────────────────────────────────────────
  'heatmap-cold':           '#D55E00', // Vermillion — mauvais
  'heatmap-hot':            '#0072B2', // Blue       — bon
  'heatmap-divergent-low':  '#D55E00', // K/D bas
  'heatmap-divergent-high': '#0072B2', // K/D haut
  'heatmap-freq-low':       '#013A63', // fréquence faible — bleu sombre (neutre, mono-teinte)
  'heatmap-freq-high':      '#56B4E9', // fréquence forte  — Sky Blue    (neutre, mono-teinte)

  // ── Équipes — axe Blue/Vermillion daltonisme-safe ──────────────────────────
  'team-ally':  '#0072B2', // Blue       — overridable via settings accessibilité
  'team-enemy': '#D55E00', // Vermillion — overridable via settings accessibilité
}
