/**
 * usageInks.ts — LES ENCRES ET TEXTURES du bloc « usages d'équipement, armes spéciales et
 * objectifs », en un seul endroit.
 *
 * EXTRAIT DE `UsageForms.tsx` le 2026-09-21, parce que la LÉGENDE de ces textures
 * (`UsageHatchLegend`) est posée PAR les formes elles-mêmes : la légende doit peindre
 * exactement la même hachure que la jauge qu'elle explique, et un import croisé entre la
 * forme et sa légende ferait un cycle. Les deux lisent donc la même source (CLAUDE.md n°6 —
 * une légende qui recopie sa texture finit par mentir sur ce qu'elle légende).
 *
 * COULEURS — jetons sémantiques uniquement (cf. en-tête de `UsageForms.tsx` pour la
 * grammaire complète : « nous » coloré, « eux » jamais).
 */
import type { CSSProperties } from 'react'

import { tokenCssVar } from '@/lib/accessibility'

/** L'encre du trait de parité — jeton distinct, jamais une teinte de donnée. */
export const PARITY_INK = tokenCssVar('warning')
/** L'encre de « nous » (le camp du joueur), surchargeable par l'accessibilité. */
export const ALLY_INK = tokenCssVar('team-ally')
/**
 * LES DEUX REPÈRES DE TAUX DANS LA TRANCHE (P7, §3.2, étape E4) : « reste de mon
 * équipe » reprend le jeton `team-ally` (la référence EST mon camp) ; « eux » n'a
 * PAS de jeton de donnée — un trait pointillé neutre, comme la hachure de la piste
 * du lobby (règle du bloc usage : l'adversaire n'est jamais coloré).
 */
export const TEAMMATES_REF_INK = tokenCssVar('team-ally')
export const OPPONENTS_REF_INK = 'var(--muted-foreground)'

/**
 * La hachure anonyme de « eux » : motif neutre du thème, jamais un jeton d'équipe.
 *
 * ELLE NE PORTE PLUS L'OPACITÉ DU SEGMENT (2026-09-21). Posée sur le segment lui-même,
 * `opacity: 0.45` délavait AUSSI l'étiquette blanche posée dessus, et le segment se lisait
 * plus mince que ses voisins pleins (des rayures sur un fond de carte contre un aplat).
 * Elle est désormais un CALQUE `aria-hidden` sous l'étiquette, sur une assise `--muted` :
 * même rectangle, même épaisseur, étiquette à pleine encre.
 */
export const ENEMY_HATCH: CSSProperties = {
  backgroundColor: 'var(--muted)',
  backgroundImage:
    'repeating-linear-gradient(45deg, transparent 0px, transparent 4px, var(--muted-foreground) 4px, var(--muted-foreground) 6px)',
}

/**
 * LA MARQUE DU DÉNOMINATEUR « LOBBY » (ajustement pré-v7.5, 2026-09-13) — les deux
 * colonnes rapportées au LOBBY (« Mon équipe dans le lobby », « Ma part dans le
 * lobby ») portent la hachure neutre par-dessus leur tranche ; la colonne rapportée
 * à MON ÉQUIPE (« Ma part dans mon équipe ») reste un aplat. Les deux se
 * distinguent donc au premier coup d'œil, hachure contre aplat.
 *
 * POURQUOI UNE TEXTURE ET PAS UNE SECONDE TEINTE, mesure à l'appui. Sur les
 * grandeurs du bilan d'équipement les deux jauges portent la MÊME pile d'issues par
 * construction (`usageGaugeModel`, un seul `outcomes` par grandeur) : une teinte de
 * remplissage n'y changerait rien. Et pour les autres grandeurs, aucun jeton
 * existant ne tient : le balayage des jetons de `palettes/*` (13/09, validateur
 * `dataviz/scripts/validate_palette.js`) ne laisse, contre `team-ally` ET contre le
 * `warning` du trait de parité, sur les QUATRE palettes livrées, que la famille
 * rouge/vermillon — c'est-à-dire `team-enemy`, que la grammaire de ce bloc interdit
 * (« l'adversaire n'est jamais coloré »). La hachure, elle, se lit sous toute forme
 * de daltonisme, en impression et en contraste forcé — et elle dit déjà « le lobby,
 * anonyme » sur la piste du lobby juste à côté.
 */
export const LOBBY_HATCH: CSSProperties = {
  backgroundImage:
    'repeating-linear-gradient(45deg, transparent 0px, transparent 3px, var(--background) 3px, var(--background) 5px)',
  opacity: 0.55,
}

/** Les quatre encres de la bande de régularité — source unique des cases ET de leur légende. */
export const BAND_TONE_INKS = {
  above: tokenCssVar('divergent-pos'),
  near: tokenCssVar('divergent-neutral'),
  below: tokenCssVar('divergent-neg'),
  /** Non mesuré : AUCUNE encre — la case reste sur le fond `bg-muted`, ce n'est pas une donnée. */
  unmeasured: undefined,
} as const
