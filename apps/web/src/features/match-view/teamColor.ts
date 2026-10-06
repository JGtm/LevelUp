/**
 * teamColor.ts — LA TEINTE D'UN HABILLAGE D'ÉQUIPE, écrite une fois pour la vue match et le rejeu.
 *
 * La couleur qu'on y verse est déjà résolue par l'appelant — les jetons d'allégeance
 * `team-ally` / `team-enemy` (`teamSeriesColor.teamTokenCssVar`), que les réglages
 * d'accessibilité surchargent. La CASCADE D'IDENTITÉ qui vivait ici (`team_color` du backend,
 * puis couleur officielle par `team_id`, puis le jeton) n'avait plus d'appelant depuis que le
 * fil des éliminations du rejeu, son dernier lecteur, prend l'allégeance du film (2026-10-06) :
 * elle est retirée, avec la table des couleurs officielles qui ne servait qu'à elle.
 */

/**
 * LA RECETTE DE TEINTE D'UN HABILLAGE D'ÉQUIPE : un fond, un trait, un accent plein.
 *
 * Elle vivait en toutes lettres dans l'en-tête d'équipe du scoreboard ; l'écran de victoire du
 * rejeu en aurait été la deuxième copie (2026-08-26). Deux surfaces qui écrivent la MÊME
 * identité d'équipe avec deux dosages différents, ce sont deux teintes pour la même équipe sur
 * deux pages du même match. Règle CLAUDE.md n°6 : centraliser ET poser un garde-rail
 * (`teamTint.guard.test.ts`, qui interdit le littéral de fond hors de ce fichier).
 *
 * LES DOSAGES SONT CEUX DU SCOREBOARD, INCHANGÉS : 22 % pour teinter un fond sans disputer la
 * lisibilité du texte, 55 % pour un trait qui se voit sans crier, la couleur PLEINE pour
 * l'accent qui porte l'identité. `oklab` et non `srgb` : le mélange y garde la teinte perçue
 * d'une couleur vive (le jaune Valor ne vire pas au vert olive en s'éclaircissant).
 *
 * CE QUE LA RECETTE NE DIT PAS, ET C'EST VOULU : ni l'ÉPAISSEUR des traits (2 px sous l'en-tête
 * du scoreboard, 4 px sur son bord gauche — deux rôles, deux mesures), ni la couleur du TEXTE,
 * qui reste `--foreground` partout. Une couleur d'identité peut être très claire ou très vive ;
 * l'encre du thème est le seul contraste garanti.
 */
export interface TeamTintStyles {
  /** Aplat de fond : la couleur d'équipe à peine posée, pour teinter une surface. */
  background: string
  /** COULEUR d'un trait (bordure, soulignement) — l'épaisseur reste au point d'appel. */
  border: string
  /** La couleur d'identité PLEINE : liserés marqués, pastilles, filets. */
  accent: string
}

/** Part de couleur d'équipe dans un fond teinté. */
const TINT_BACKGROUND_PCT = 22
/** Part de couleur d'équipe dans un trait. */
const TINT_BORDER_PCT = 55

/** teamTintStyles applique la recette ci-dessus à une couleur d'identité déjà résolue. */
export function teamTintStyles(color: string): TeamTintStyles {
  return {
    background: `color-mix(in oklab, ${color} ${TINT_BACKGROUND_PCT}%, transparent)`,
    border: `color-mix(in oklab, ${color} ${TINT_BORDER_PCT}%, transparent)`,
    accent: color,
  }
}
