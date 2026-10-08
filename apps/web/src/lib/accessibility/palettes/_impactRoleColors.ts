/**
 * _impactRoleColors.ts — Les deux rampes des RÔLES D'IMPACT (Escouade › Contributions,
 * « Points d'impact par soirée et par rôle »), en thème clair et en thème sombre.
 *
 * Rampes ORDINALES, PALETTE-INVARIANTES : un rôle s'y lit à sa clarté (le pas 1, le plus fort
 * barème, est posé contre l'axe zéro et le plus contrasté sur la carte) et au côté de l'axe
 * (gains au-dessus, pertes en dessous), jamais à la teinte seule ; son nom est dans l'infobulle
 * et la légende. Les écarts entre voisins d'empilement tiennent en vision normale ET sous
 * protanopie / deutéranopie (impactRoleTokens.test.ts) : une palette daltonienne n'a donc pas
 * de raison de les recolorer. Teintes : celles de `divergent-pos` / `divergent-neg` de la
 * palette par défaut (vert, rouge).
 *
 * Seuls jetons dont la valeur change avec le THÈME : en sombre, la rampe s'inverse en clarté
 * pour que le pas le plus fort reste le plus contrasté (`paletteForTheme`). Valeurs de la
 * maquette validée par l'utilisateur.
 */
import type { SemanticToken } from '../semantic-tokens'

type ImpactRoleToken = Extract<SemanticToken, `impact-${string}`>

/** Thème clair (valeur portée par chaque palette). */
export const IMPACT_ROLE_COLORS: Record<ImpactRoleToken, string> = {
  'impact-gain-1': '#023212', // Finisseur +2
  'impact-gain-2': '#0A6031', // Premier sang +2
  'impact-gain-3': '#159257', // Héros silencieux +1,5
  'impact-gain-4': '#22CE89', // Bourreau +1
  'impact-loss-1': '#A30E22', // Boulet −2
  'impact-loss-2': '#F2393E', // Faux-frère −1,5
  'impact-loss-3': '#FD9A8F', // Première victime, Touriste, Kamikaze, Voleur −1
}

/** Thème sombre (appliqué par-dessus la palette active, cf. `paletteForTheme`). */
export const IMPACT_ROLE_COLORS_DARK: Record<ImpactRoleToken, string> = {
  'impact-gain-1': '#7DF861',
  'impact-gain-2': '#1FC447',
  'impact-gain-3': '#14904C',
  'impact-gain-4': '#095E3A',
  'impact-loss-1': '#FC8F84',
  'impact-loss-2': '#E82D34',
  'impact-loss-3': '#970C1E',
}
