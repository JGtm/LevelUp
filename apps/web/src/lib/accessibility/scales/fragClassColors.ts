/**
 * fragClassColors.ts — Identité couleur CATÉGORIELLE FIXE des classes d'arme du
 * sunburst « Répartition des frags » v2 (classe→rôle).
 *
 * EXCEPTION à la règle « zéro magic hex » (skill color-tokens, § Exceptions
 * tolérées), au même titre que rarity.ts et okabe-ito.ts : ces hex sont la RAISON
 * D'ÊTRE du fichier. Identité catégorielle fixe CVD-safe, HORS palette active,
 * garantit la distinction quelle que soit la palette.
 *
 * POURQUOI des hex fixes indépendants de la palette active : sous la palette DÉFAUT
 * de l'app (palettes/default.ts), les tokens chart-series-1..5 forment une rampe
 * indigo (all-pairs ΔE ~5.4, SOUS le plancher CVD) → 5 classes de frags
 * indistinguables. Donner aux classes de frags leur PROPRE jeu de couleurs
 * Okabe-Ito fixes — exactement le précédent des couleurs de rareté (rarity.ts) —
 * garantit la distinction CVD-safe QUELLE QUE SOIT la palette active.
 *
 * Teintes Okabe-Ito (Color Universal Design, 2008) choisies pour distance maximale
 * sous protanopie/deutéranopie. Validées CVD (validate_palette.js / guard test) :
 * worst all-pairs ΔE 11.0 en deutan (> cible validateur 8) ; normal-vision 15.6.
 * Le résidu « Non attribué » reçoit un neutre (rendu HACHURÉ côté chart, jamais une
 * couleur pleine).
 *
 * NE JAMAIS importer ce fichier depuis features/ ou components/ : la consommation
 * passe par fragClassColor()/fragRoleColor() (fragClass.ts).
 */
import type { FragClassKey } from './fragClass'

/**
 * Couleur hex fixe par classe. Indépendante de la palette active — la CVD-safety
 * est portée par ces teintes Okabe-Ito, jamais par un token de palette. 1 hex
 * distinct par classe de combat (zéro collision), neutre pour le résidu.
 */
export const FRAG_CLASS_HEX: Record<FragClassKey, string> = {
  shoulder: '#0072B2', // Blue         — Épaule
  sidearm: '#E69F00', // Orange        — Poing
  heavy: '#56B4E9', // Sky Blue        — Lourde
  melee: '#D55E00', // Vermillion      — Mêlée
  grenade: '#009E73', // Bluish Green   — Grenade
  spartan_ability: '#F0E442', // Yellow — Capacités spartanes (H5)
  unattributed: '#888888', // Gris neutre — Non attribué (hachuré côté chart)
}

/** Neutre de repli pour une clé de classe inconnue du front (jamais une couleur de combat). */
export const FRAG_CLASS_NEUTRAL_HEX = '#888888'

/** Hex fixe de la classe (fallback neutre si clé inconnue). Client ET SSR (aucun accès DOM). */
export function fragClassFixedHex(className: string | null | undefined): string {
  if (className != null && className in FRAG_CLASS_HEX) {
    return FRAG_CLASS_HEX[className as FragClassKey]
  }
  return FRAG_CLASS_NEUTRAL_HEX
}
