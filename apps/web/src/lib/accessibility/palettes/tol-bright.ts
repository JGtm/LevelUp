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
import { ENCOUNTER_BADGE_COLORS } from './_encounterColors'

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

// Tol Muted — teinte « wine », empruntée au schéma Muted du même auteur pour
// dissocier narrative-humiliation de outcome-dnf (tous deux sur TOL_PURPLE).
const TOL_MUTED_WINE = '#882255'

// Tol Vibrant — teinte « orange », empruntée au schéma Vibrant du même auteur pour
// dissocier narrative-debacle de outcome-loss (tous deux sur TOL_RED). Même motif
// que TOL_MUTED_WINE ci-dessus : une débâcle se superpose à une défaite, les deux
// ne peuvent pas porter exactement le même fond. L'orange aligne aussi tol-bright
// sur la sémantique des autres palettes (débâcle = orange/vermillon).
const TOL_VIBRANT_ORANGE = '#EE7733'

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

  // ── Joueurs d'escouade ─────────────────────────────────────────────────────
  // Les 7 teintes du schéma Bright servent déjà toutes un token outcome ou
  // narratif, et 3 d'entre elles (Cyan, Yellow, Grey) sont trop claires pour le
  // seuil de contraste 3:1 sur la surface claire. On puise donc dans les schémas
  // VIBRANT et MUTED du même auteur (mêmes garanties CVD) 4 teintes absentes du
  // reste de la palette : l'identité joueur ne collide avec aucun autre rôle.
  // ΔE min 17.9 en vision normale ; contraste ≥ 3.1:1 sur les deux surfaces.
  'squad-player-1': '#0077BB', // Tol Vibrant Blue   — joueur principal
  'squad-player-2': '#CC3311', // Tol Vibrant Red    — coéquipier 1
  'squad-player-3': '#AA4499', // Tol Muted Purple   — coéquipier 2
  'squad-player-4': '#117733', // Tol Muted Green    — coéquipier 3

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

  // ── Rareté — accent légendaire (encadré surbouclier du rejeu 2D, etc.) ──────
  // Le plus proche d'un "or" dans le set catégoriel à 7 teintes — déjà réutilisé
  // (outcome-draw, chart-series-4, warning) : même pattern de partage que le
  // reste de cette palette (TOL_BLUE seul sert déjà 7 rôles).
  'legendary': TOL_YELLOW, // #CCBB44

  // ── Extrême rare — Purple du schéma Bright, CVD-safe par construction ─────
  'extreme': TOL_PURPLE,

  // ── Badges narratifs ────────────────────────────────────────────────────────
  // Texte calculé pour contraste WCAG AA (test automatique Phase C)
  'narrative-dominant':              TOL_GREEN,  // vert UI — statut positif
  'narrative-dominant-text':         '#FFFFFF',
  // Wine (Tol Muted) et non TOL_PURPLE : ce dernier reste outcome-dnf, et deux
  // verdicts de match ne doivent pas partager exactement le même fond.
  'narrative-humiliation':           TOL_MUTED_WINE,
  'narrative-humiliation-text':      '#FFFFFF',
  'narrative-remontada':             TOL_BLUE,
  'narrative-remontada-text':        '#FFFFFF',
  'narrative-debacle':               TOL_VIBRANT_ORANGE,
  'narrative-debacle-text':          '#000000', // noir sur Tol Vibrant Orange (7.3) — blanc ne passe pas AA
  'narrative-contre-remontada':      TOL_CYAN,   // cyan clair
  'narrative-contre-remontada-text': '#000000',

  // ── Badges encounter ───────────────────────────────────────────────────────
  // (set sombre distinct AA-blanc, palette-invariant — cf. _encounterColors.ts ;
  //  labels disambiguent pour les daltoniens)
  ...ENCOUNTER_BADGE_COLORS,

  // ── Classes de frags — famille dédiée (2026-08-29) ─────────────────────────
  // Même motif que sur les autres palettes : 11 classes > 7 teintes Bright, on
  // puise dans les schémas Muted et Vibrant du même auteur (mêmes garanties CVD),
  // comme le font déjà squad-player-* et narrative-*. Garde-rail par palette :
  // fragClass.guard.test.ts (ΔE ≥ 8 ici).
  'frag-shoulder':        TOL_CYAN,           // cyan
  'frag-sidearm':         TOL_GREEN,          // vert
  'frag-heavy':           TOL_PURPLE,         // pourpre
  'frag-melee':           TOL_RED,            // rouge rosé
  'frag-grenade':         TOL_YELLOW,         // jaune
  'frag-spartan-ability': TOL_BLUE,           // bleu
  'frag-vehicle':         '#332288',          // Tol Muted Indigo — indigo profond
  'frag-turret':          TOL_VIBRANT_ORANGE, // orange brûlé
  'frag-equipment':       TOL_MUTED_WINE,     // wine — écho du fuchsia du défaut
  'frag-environmental':   '#004488',          // Tol High-contrast Blue — bleu profond
  //                       (le Vibrant Blue #0077BB valait ΔE 4,2 contre TOL_BLUE)
  'frag-unattributed':    NEUTRAL_GREY,       // gris neutre (résidu)

  // ── Heatmaps ────────────────────────────────────────────────────────────────
  'heatmap-cold':           TOL_RED,
  'heatmap-hot':            TOL_BLUE,
  'heatmap-divergent-low':  TOL_RED,
  'heatmap-divergent-high': TOL_BLUE,
  'heatmap-freq-low':       '#1A3550', // fréquence faible — bleu sombre (neutre, mono-teinte)
  'heatmap-freq-high':      TOL_CYAN, // fréquence forte  — cyan Tol Bright (neutre)

  // ── Équipes — axe Blue/Red Tol Bright ─────────────────────────────────────
  'team-ally':  TOL_BLUE, // overridable via settings accessibilité
  'team-enemy': TOL_RED,  // overridable via settings accessibilité
}
