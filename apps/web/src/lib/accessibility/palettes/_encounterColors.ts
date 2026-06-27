/**
 * _encounterColors.ts — Couleurs des badges de RELATION (Communauté > Relations
 * + Explorer / Match View / Compare).
 *
 * Set SOMBRE, DISTINCT par pill, PALETTE-INVARIANT : contraste WCAG AA garanti
 * sur TEXTE BLANC (`--color-badge-on-solid`) — validé dans wcagContrast.test.ts.
 *
 * Pourquoi invariant (et non recoloré par palette) : ces badges sont SOLIDES +
 * texte blanc → exigent un fond sombre, alors que les palettes daltonisme
 * (Okabe-Ito, Tol Bright) sont des teintes CLAIRES (pensées pour texte noir ou
 * séries de courbes) — le texte blanc y échouait l'AA. La distinction pour les
 * daltoniens repose donc sur le LABEL (toujours affiché), jamais la couleur
 * seule (WCAG 1.4.1) ; les teintes sont néanmoins choisies aussi écartées que
 * possible.
 */
import type { SemanticToken } from '../semantic-tokens'

type EncounterToken = Extract<SemanticToken, `narrative-encounter-${string}`>

export const ENCOUNTER_BADGE_COLORS: Record<EncounterToken, string> = {
  'narrative-encounter-ally-plus': '#166534', // vert       — Allié+
  'narrative-encounter-duo-gagnant': '#115E59', // teal       — Duo gagnant
  'narrative-encounter-tough-enemy': '#B91C1C', // rouge      — Dur à cuire
  'narrative-encounter-coriace': '#9D174D', // amarante   — Coriace
  'narrative-encounter-ordinal': '#1D4ED8', // cobalt     — ×N rencontres
  'narrative-encounter-cameleon': '#6D28D9', // améthyste  — Caméléon
  'narrative-encounter-de-longue-date': '#3730A3', // indigo     — Ancien
  'narrative-encounter-recrue': '#155E75', // cyan       — Recrue
  'narrative-encounter-proie-favorite': '#854D0E', // or         — Proie favorite
  'narrative-encounter-cross-game': '#475569', // ardoise    — {jeu} (cross-titre)
}
