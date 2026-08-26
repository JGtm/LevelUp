/**
 * teamColor.ts — LA COULEUR D'IDENTITÉ D'UNE ÉQUIPE, et il n'y en a qu'une.
 *
 * Centralisée ici quand le rejeu 2D a eu besoin de la même cascade que la vue match : deux
 * cascades divergentes, ce sont deux couleurs pour la même équipe sur deux pages du même
 * match. Scoreboard, objectifs et fil des frags du rejeu appellent tous cette fonction, et
 * rien d'autre.
 *
 * L'ORDRE DE LA CASCADE reproduit celui de l'en-tête du scoreboard (`MatchScoreboard.tsx`) :
 * couleur fournie par le backend (`team_color`, Halo 5) d'abord, sinon la couleur officielle
 * par `team_id` (Halo Infinite : Eagle bleu, Cobra rouge...), sinon le token sémantique
 * allié/ennemi — surchargeable par les réglages d'accessibilité.
 *
 * Aucun hex n'est écrit ici : ils vivent dans `lib/halo/`, référentiel de couleurs de jeu au
 * même titre que `rarity.ts`.
 */
import { tokenCssVar } from '@/lib/accessibility'
import { parseTeamSideID, resolveTeamColorFromID } from '@/lib/halo/teamNames'
import type { MatchScoreboardRow } from '@/lib/api/types'

/** Résout la couleur d'identité d'une équipe, à partir de son id et de son camp. */
export type TeamColorResolver = (teamID: number | null, ally: boolean) => string

/**
 * teamColorResolver construit le résolveur pour un scoreboard donné.
 *
 * Le scoreboard est indexé UNE fois : un feed de 80 kills ne doit pas le reparcourir à
 * chaque ligne.
 */
export function teamColorResolver(
  // `readonly` : le résolveur ne fait que LIRE le scoreboard, et ses appelants le tiennent
  // souvent en lecture seule (props de composant). Élargir le paramètre ne coûte rien et
  // évite une copie défensive au point d'appel.
  scoreboard: readonly MatchScoreboardRow[] | null | undefined,
): TeamColorResolver {
  const backendByTeamID = new Map<number, string>()
  for (const r of scoreboard ?? []) {
    const id = parseTeamSideID(r.team_side)
    if (id != null && r.team_color && !backendByTeamID.has(id)) {
      backendByTeamID.set(id, r.team_color)
    }
  }
  return (teamID, ally) => {
    if (teamID != null) {
      const backend = backendByTeamID.get(teamID)
      if (backend) return backend
      const official = resolveTeamColorFromID(teamID)
      if (official) return official
    }
    return tokenCssVar(ally ? 'team-ally' : 'team-enemy')
  }
}

/**
 * LA RECETTE DE TEINTE D'UN HABILLAGE D'ÉQUIPE : un fond, un trait, un accent plein.
 *
 * Elle vivait en toutes lettres dans l'en-tête d'équipe du scoreboard ; l'écran de victoire du
 * rejeu en aurait été la deuxième copie (2026-08-26). Deux surfaces qui écrivent la MÊME
 * identité d'équipe avec deux dosages différents, ce sont deux bleus Eagle sur deux pages du
 * même match — le motif exact qui a fait centraliser la CASCADE juste au-dessus. Règle
 * CLAUDE.md n°6 : centraliser ET poser un garde-rail (`teamTint.guard.test.ts`, qui interdit le
 * littéral de fond hors de ce fichier).
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
