/**
 * useVueDeMort — la vue d'une contribution choisie : la mort ou le frag (fiche, scène), la séquence
 * quand elle est servie (lecture animée, étapes) et l'ÉTAT de la zone « séquence » du plan, qui existe
 * pour toute contribution choisie dès que le titre a des rejeux 2D (cf. `sequenceCause.logic.ts`).
 *
 * LA REQUÊTE DE SÉQUENCE PART POUR TOUTE MORT AVEC SA VICTIME NOMMÉE, y compris quand la liste dit que le
 * rejeu du match n'est pas construit : seul le serveur sait dire POURQUOI (analyse en cours, film non
 * analysé, ...) et seule sa réponse distingue une fonction fermée (garde du rejeu) d'une donnée absente.
 *
 * `vue` est `null` sans mort ni frag choisi : la colonne de droite et le plan montrent la vue de la zone.
 */
import { useMemo } from 'react'

import { useCapability } from '@/lib/capabilities/capabilities'
import type { TacticalContribution } from '@/lib/api/types'
import { useMediaQuery } from '@/lib/hooks/useMediaQuery'

import { useTacticalSequence } from './queries'
import { etatDeLaSequence, type EtatDeLaSequence } from './sequenceCause.logic'
import { useLectureDeSequence } from './useLectureDeSequence'
import { contributionDeLaVue, sequenceDemandee, type VueDeMort } from './vueDeMort'

/** La préférence de mouvement réduit : la séquence ne démarre pas d'elle-même. */
const MOUVEMENT_REDUIT = '(prefers-reduced-motion: reduce)'

export function useVueDeMort(
  playerSlug: string,
  mapId: string,
  contributions: readonly TacticalContribution[],
  choisie: number | null,
): { vue: VueDeMort | null; etatSequence: EtatDeLaSequence | null } {
  const contribution = contributionDeLaVue(contributions, choisie)
  const demande = sequenceDemandee(contribution)
  const requete = useTacticalSequence(playerSlug, mapId, demande)
  const rejeuDuTitre = useCapability('replay')
  const mouvementReduit = useMediaQuery(MOUVEMENT_REDUIT)
  const donnees = demande ? (requete.data ?? null) : null
  const { lecture, etape, choisirEtape } = useLectureDeSequence(donnees, mouvementReduit)
  const choisieQuelconque = choisie === null ? undefined : contributions[choisie]
  const erreur = demande ? requete.error : null
  const aUneDemande = demande !== null
  const enCours = aUneDemande && requete.isPending === true
  const etatSequence = useMemo(
    () =>
      etatDeLaSequence({
        contribution: choisieQuelconque,
        rejeuDuTitre,
        requete: { demandee: aUneDemande, enCours, servie: donnees !== null, erreur },
      }),
    [choisieQuelconque, rejeuDuTitre, aUneDemande, enCours, donnees, erreur],
  )
  // Une référence stable tant que rien ne change : la scène du plan ne se repeint pas à chaque rendu.
  const vue = useMemo<VueDeMort | null>(() => {
    if (choisie === null || !contribution) return null
    return { index: choisie, contribution, sequence: donnees, etape, onEtape: choisirEtape, lecture }
  }, [choisie, contribution, donnees, etape, choisirEtape, lecture])
  return { vue, etatSequence }
}
