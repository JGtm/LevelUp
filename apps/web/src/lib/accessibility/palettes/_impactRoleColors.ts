/**
 * _impactRoleColors.ts — Les deux rampes des RÔLES D'IMPACT (Escouade › Contributions,
 * « Points d'impact par soirée et par rôle »), par palette, en thème clair et en thème sombre.
 *
 * Rampes ORDINALES : un rôle s'y lit à sa clarté (le pas 1, le plus fort barème, est posé contre
 * l'axe zéro et le plus contrasté sur la carte) et au côté de l'axe (gains au-dessus, pertes en
 * dessous), jamais à la teinte seule ; son nom est dans l'infobulle et la légende. Chaque palette
 * prend ses rampes dans les deux teintes qu'elle donne déjà à `divergent-pos` / `divergent-neg` :
 * vert / rouge pour la palette par défaut, bleu / vermillon pour Okabe-Ito et Cividis (même axe
 * binaire), bleu / rouge de Tol Bright pour Tol Bright.
 *
 * Contraintes, tenues dans les deux thèmes (impactRoleTokens.test.ts) : contraste décroissant du
 * pas 1 au dernier, nuance la plus pâle à ≥ 2:1 sur la carte, voisins d'empilement à ΔE OKLab
 * ≥ 15 en vision normale et sous protanopie / deutéranopie (et tritanopie pour les palettes
 * daltoniennes).
 *
 * Seuls jetons dont la valeur change avec le THÈME : en sombre, la rampe s'inverse en clarté pour
 * que le pas le plus fort reste le plus contrasté. La palette porte la rampe claire ;
 * `paletteForTheme` pose la rampe sombre jumelle (`IMPACT_ROLE_RAMPS`).
 */
import type { SemanticToken } from '../semantic-tokens'

export type ImpactRoleToken = Extract<SemanticToken, `impact-${string}`>
export type ImpactRoleRamp = Record<ImpactRoleToken, string>

/** Une rampe claire et sa jumelle sombre. */
export interface ImpactRoleRamps {
  light: ImpactRoleRamp
  dark: ImpactRoleRamp
}

/**
 * Palette par défaut — vert / rouge sur les teintes de `divergent-pos` (#22C55E, OKLCH 149,6°) et
 * `divergent-neg` (#EF4444, 25,3°), que le reste de l'application emploie : chaque pas garde la
 * teinte de son ancre (écart < 1°), seules la clarté et la chroma varient. Chroma 0,08 à 0,16, sous
 * celle des ancres (0,19 et 0,21) : plus de vert fluo ni de rouge saturé (retour du 2026-10-17).
 *
 * Le pas 1 des gains clairs reste un vert profond (clarté 0,28) : le ΔE ≥ 15 sous deutéranopie /
 * protanopie entre quatre verts ne tient qu'avec des pas de 0,15 de clarté entre ce pas et le plus
 * pâle (0,74, plafonné par le contraste 2:1 sur la carte) ; le relever exigerait de relâcher le seuil.
 */
export const IMPACT_GREEN_RED: ImpactRoleRamps = {
  light: {
    'impact-gain-1': '#003412', // Finisseur +2
    'impact-gain-2': '#1D6131', // Premier sang +2
    'impact-gain-3': '#3C9253', // Héros silencieux +1,5
    'impact-gain-4': '#5DC677', // Bourreau +1
    'impact-loss-1': '#901E20', // Boulet −2
    'impact-loss-2': '#C45852', // Faux-frère −1,5
    'impact-loss-3': '#F98F86', // Première victime, Touriste, Kamikaze, Voleur −1
  },
  dark: {
    'impact-gain-1': '#9DF6AF',
    'impact-gain-2': '#6DBF7F',
    'impact-gain-3': '#3E8B51',
    'impact-gain-4': '#085A27',
    'impact-loss-1': '#FE9B93',
    'impact-loss-2': '#C9605A',
    'impact-loss-3': '#942223',
  },
}

/**
 * Okabe-Ito et Cividis — bleu (teinte OKLCH 244°, Blue #0072B2) / vermillon (47°, Vermillion
 * #D55E00), l'axe binaire que ces deux palettes partagent.
 */
export const IMPACT_BLUE_VERMILLION: ImpactRoleRamps = {
  light: {
    'impact-gain-1': '#072842',
    'impact-gain-2': '#005485',
    'impact-gain-3': '#0681D5',
    'impact-gain-4': '#4EBBFF',
    'impact-loss-1': '#88350A',
    'impact-loss-2': '#CB5F0B',
    'impact-loss-3': '#F59C78',
  },
  dark: {
    'impact-gain-1': '#C6E2FA',
    'impact-gain-2': '#49B3FF',
    'impact-gain-3': '#007BCC',
    'impact-gain-4': '#195175',
    'impact-loss-1': '#F8BFA6',
    'impact-loss-2': '#DA6A0A',
    'impact-loss-3': '#853101',
  },
}

/** Tol Bright — bleu (teinte OKLCH 250°, Blue #4477AA) / rouge (15°, Red #EE6677). */
export const IMPACT_TOL_BLUE_RED: ImpactRoleRamps = {
  light: {
    'impact-gain-1': '#082841',
    'impact-gain-2': '#1A5481',
    'impact-gain-3': '#0077E0',
    'impact-gain-4': '#6CB7FF',
    'impact-loss-1': '#9B0035',
    'impact-loss-2': '#E73850',
    'impact-loss-3': '#FF92A2',
  },
  dark: {
    'impact-gain-1': '#C8E1FF',
    'impact-gain-2': '#58B0FE',
    'impact-gain-3': '#0073D9',
    'impact-gain-4': '#184F7A',
    'impact-loss-1': '#FABBC2',
    'impact-loss-2': '#F14C60',
    'impact-loss-3': '#961431',
  },
}

/** Toutes les paires de rampes : `paletteForTheme` y retrouve la jumelle sombre d'une palette. */
export const IMPACT_ROLE_RAMPS: readonly ImpactRoleRamps[] = [
  IMPACT_GREEN_RED,
  IMPACT_BLUE_VERMILLION,
  IMPACT_TOL_BLUE_RED,
]
