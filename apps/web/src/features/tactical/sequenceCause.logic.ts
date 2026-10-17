/**
 * sequenceCause.logic — CE QUE LA PAGE DIT D'UNE SÉQUENCE : pour une contribution choisie, TOUJOURS un
 * état — la séquence qui se joue, ou un état vide qui dit pourquoi elle manque — sauf quand la fonction
 * elle-même est fermée. C'est ce qui permet de distinguer un manque de fonction d'un manque de données.
 *
 * LES DEUX FAMILLES D'ABSENCE (contrat de `handlers/tactical_sequence.go`) :
 *
 *   FONCTION FERMÉE  le titre n'a pas de rejeu 2D, ou la route répond `replay_not_available` (garde du
 *                    rejeu : production, match que l'appelant n'a pas joué, démo hors figés). Aucun état :
 *                    la vue de séquence n'existe pas, exactement comme le lien de rejeu.
 *   DONNÉE ABSENTE   le match est celui du joueur, la séquence manque pour une raison de données : un code
 *                    machine propre (`sequence_*`), lu ici et écrit en texte (FR/EN) par `TacticalText`.
 *                    Une erreur réseau ou un code inconnu donne l'état générique `inconnue`, jamais rien.
 */
import { apiErrorCode } from '@/lib/api/client'
import type { TacticalContribution } from '@/lib/api/types'

/** Les codes machine servis par la route de séquence (`sequenceCauses` côté Go) et celui du refus. */
export const CODE_SEQUENCE = {
  refus: 'replay_not_available',
  analyseEnCours: 'sequence_analysis_pending',
  filmNonAnalyse: 'sequence_film_not_analysed',
  filmAReanalyser: 'sequence_film_to_reanalyse',
  carteSansRayon: 'sequence_map_unmeasured',
  sansPosition: 'sequence_no_death_position',
} as const

/** Pourquoi une séquence manque. `inconnue` : erreur réseau ou code non reconnu. */
export type CauseDeSequence =
  | 'analyse_en_cours'
  | 'film_non_analyse'
  | 'film_a_reanalyser'
  | 'carte_sans_rayon'
  | 'sans_position'
  | 'inconnue'

const CAUSE_PAR_CODE: Readonly<Record<string, CauseDeSequence>> = {
  [CODE_SEQUENCE.analyseEnCours]: 'analyse_en_cours',
  [CODE_SEQUENCE.filmNonAnalyse]: 'film_non_analyse',
  [CODE_SEQUENCE.filmAReanalyser]: 'film_a_reanalyser',
  [CODE_SEQUENCE.carteSansRayon]: 'carte_sans_rayon',
  [CODE_SEQUENCE.sansPosition]: 'sans_position',
}

/** causeDeLErreur — la cause d'une requête en échec ; `refus` quand la fonction est fermée. */
export function causeDeLErreur(erreur: unknown): CauseDeSequence | 'refus' {
  const code = apiErrorCode(erreur)
  if (code === CODE_SEQUENCE.refus) return 'refus'
  return (code && CAUSE_PAR_CODE[code]) || 'inconnue'
}

/** Ce que la zone « séquence » du plan montre pour la contribution choisie. */
export type EtatDeLaSequence =
  | { kind: 'chargement' }
  | { kind: 'servie' }
  | { kind: 'absente'; cause: CauseDeSequence }
  /** La contribution n'est pas une mort : seules les morts ont une séquence. */
  | { kind: 'sur_les_morts' }

/** L'état de la requête de séquence, tel que `useTacticalSequence` le rend. */
export interface RequeteDeSequence {
  /** La requête est partie (une mort avec sa victime nommée). */
  demandee: boolean
  enCours: boolean
  servie: boolean
  erreur: unknown
}

/**
 * etatDeLaSequence — l'état de la zone « séquence » du plan, ou `null` quand elle n'existe pas : rien
 * de choisi, titre sans rejeu 2D, ou route fermée par le garde du rejeu.
 */
export function etatDeLaSequence(entree: {
  contribution: TacticalContribution | undefined
  /** Le titre produit des rejeux 2D (capability `replay`). */
  rejeuDuTitre: boolean
  requete: RequeteDeSequence
}): EtatDeLaSequence | null {
  const { contribution: c, rejeuDuTitre, requete } = entree
  if (!c || !rejeuDuTitre) return null
  if (c.face !== 'mort') return { kind: 'sur_les_morts' }
  if (!requete.demandee) return { kind: 'absente', cause: 'inconnue' }
  if (requete.servie) return { kind: 'servie' }
  if (requete.erreur != null) {
    const cause = causeDeLErreur(requete.erreur)
    return cause === 'refus' ? null : { kind: 'absente', cause }
  }
  return requete.enCours ? { kind: 'chargement' } : { kind: 'absente', cause: 'inconnue' }
}
