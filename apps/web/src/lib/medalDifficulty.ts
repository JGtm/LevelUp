/**
 * medalDifficulty.ts — Couleurs des médailles Halo Infinite par niveau de difficulté.
 *
 * Les 4 niveaux correspondent aux couleurs officielles du jeu (PGCR) :
 * Normal=vert, Heroic=bleu, Legendary=violet, Mythic=rouge.
 *
 * Exception couleur autorisée : rarity/difficulty Halo (CLAUDE.md §20).
 * Utiliser dropShadowForDifficulty() sur <img> (drop-shadow suit la forme PNG transparente)
 * et boxShadowForDifficulty() sur les fallbacks texte (border-radius rectangulaire).
 */

export type MedalDifficulty = 'Normal' | 'Heroic' | 'Legendary' | 'Mythic'

// Couleurs RGBA alignées sur les teintes officielles Halo Infinite.
const DIFFICULTY_GLOW_COLOR: Record<MedalDifficulty, string> = {
  Normal:    'rgba(74,222,128,0.55)',   // green-400 — niveau Normal
  Heroic:    'rgba(56,189,248,0.55)',   // sky-400   — niveau Heroic
  Legendary: 'rgba(192,132,252,0.60)',  // purple-400 — niveau Legendary
  Mythic:    'rgba(244,63,94,0.65)',    // rose-500  — niveau Mythic
}

/** CSS filter drop-shadow pour un <img> à fond transparent (suit la forme réelle). */
export function dropShadowForDifficulty(difficulty: string | undefined): string | undefined {
  const color = DIFFICULTY_GLOW_COLOR[difficulty as MedalDifficulty]
  if (!color) return undefined
  return `drop-shadow(0 0 7px ${color})`
}

/** CSS box-shadow pour un conteneur rectangulaire (fallback initiale). */
export function boxShadowForDifficulty(difficulty: string | undefined): string | undefined {
  const color = DIFFICULTY_GLOW_COLOR[difficulty as MedalDifficulty]
  if (!color) return undefined
  return `0 0 9px -1px ${color}`
}
