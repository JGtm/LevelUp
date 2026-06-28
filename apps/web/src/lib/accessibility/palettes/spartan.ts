/**
 * Palette du personnalisateur Spartan (Halo 5).
 *
 * EXCEPTION couleurs-tokens (CLAUDE.md règle 20 / skill color-tokens) : ce sont des
 * couleurs de DONNÉES choisies par le joueur pour recoloriser ses masques d'emblème
 * et de nameplate (zones primaire / secondaire / tertiaire), PAS des couleurs
 * sémantiques d'UI. Même statut que les couleurs de rareté (`rarity.ts`). Les
 * composants ne hardcodent aucun hex : ils consomment cette palette + les tokens.
 *
 * « Large mais pas pléthorique » : un échantillon couvrant le spectre, sans noyer
 * l'utilisateur.
 */

export interface SpartanColor {
  /** id stable — sert de clé i18n / aria-label. */
  id: string
  hex: string
}

export const SPARTAN_PALETTE: readonly SpartanColor[] = [
  { id: 'red', hex: '#d62828' },
  { id: 'crimson', hex: '#8c1c2b' },
  { id: 'orange', hex: '#e8590c' },
  { id: 'gold', hex: '#f0a500' },
  { id: 'yellow', hex: '#f4d03f' },
  { id: 'lime', hex: '#82c91e' },
  { id: 'green', hex: '#2f9e44' },
  { id: 'teal', hex: '#0ca678' },
  { id: 'cyan', hex: '#22b8cf' },
  { id: 'blue', hex: '#1971c2' },
  { id: 'navy', hex: '#1c3d6e' },
  { id: 'indigo', hex: '#4263eb' },
  { id: 'purple', hex: '#7048e8' },
  { id: 'magenta', hex: '#c2255c' },
  { id: 'pink', hex: '#e64980' },
  { id: 'white', hex: '#f1f3f5' },
  { id: 'silver', hex: '#adb5bd' },
  { id: 'charcoal', hex: '#343a40' },
]

/**
 * Préselection par défaut : rouge (primaire) / bleu (secondaire) / or (tertiaire).
 * La tertiaire est la teinte de la PLAQUE sur la plupart des nameplates → un or visible
 * (et non un blanc délavé) pour que les vignettes et la bannière par défaut ne soient
 * pas « toutes blanches ». Le joueur reste libre de choisir blanc/argent.
 */
export const SPARTAN_DEFAULT_COLORS = {
  primary: '#d62828',
  secondary: '#1971c2',
  tertiary: '#f0a500',
} as const
