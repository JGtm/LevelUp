/**
 * hudBand — LE BANDEAU HUD de la colonne de droite du rejeu (option 2a du handoff
 * 2026-08-27) : liseré gauche, dégradé qui s'éteint vers la droite, typo mono en capitales
 * espacées. DEUX consommateurs, UNE écriture (règle CLAUDE.md n°6) : l'en-tête d'équipe
 * (`ReplayTeamHeader`, aux couleurs de camp) et le titre du fil des éliminations
 * (`ReplayKillFeed`, en neutre). Fichier dédié : un fichier de composant n'exporte que des
 * composants (react-refresh/only-export-components).
 */
import type { CSSProperties } from 'react'

/** Part de la couleur de camp au départ du dégradé : assez pour teinter, jamais pour crier. */
const BAND_TINT_PCT = 22
/** Fond du bandeau NEUTRE (sans camp, titre du fil) : l'encre du thème, très diluée. */
const BAND_NEUTRAL_PCT = 10

/**
 * La classe du bandeau : coins droits côté liseré (`rounded-r-sm` = 0 4px 4px 0).
 * L'ENCRE ne vit pas ici : chaque usage la choisit (foreground pour un camp connu et pour
 * le titre du fil, muted-foreground pour un groupe sans camp).
 */
export const HUD_BAND_CLASS =
  'shrink-0 rounded-r-sm px-2 py-1 font-mono text-[10px] font-bold uppercase tracking-[.16em]'

/**
 * hudBandStyle — liseré gauche et dégradé du bandeau : la couleur de camp quand elle est
 * connue, les neutres du thème sinon. Le dégradé s'éteint vers la droite — le bandeau
 * teinte le départ de la ligne, jamais toute sa largeur.
 */
export function hudBandStyle(accent: string | null): CSSProperties {
  const tint = accent
    ? `color-mix(in srgb, ${accent} ${BAND_TINT_PCT}%, transparent)`
    : `color-mix(in srgb, var(--foreground) ${BAND_NEUTRAL_PCT}%, transparent)`
  return {
    borderLeft: `2px solid ${accent ?? 'var(--border)'}`,
    background: `linear-gradient(90deg, ${tint}, transparent)`,
  }
}
