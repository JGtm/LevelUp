/**
 * useCompositionOptions — les coéquipiers proposés au sélecteur de composition
 * (`SquadCompositionPicker`) hors de la page Escouade, l'annuaire qui traduit une composition en
 * xuids, et le xuid du joueur consulté. Règles de la liste : `compositionOptions.logic.ts`.
 *
 * MÊMES CLÉS DE CACHE QUE LES PAGES QUI LISENT DÉJÀ CES RÉPONSES (`careerEncounters` de la
 * Carrière, `playerFriends`, escouades et groupes de `useSquadPresets`) : une seule entrée de cache
 * par réponse, jamais deux requêtes pour une seule liste.
 */
import { useMemo } from 'react'
import { useQuery } from '@tanstack/react-query'

import { usePlayerFriends } from '@/features/friends/queries'
import { useMyGroups } from '@/features/groups/queries'
import { useMySquads } from '@/features/prestige/hooks/useSquads'
import { api } from '@/lib/api/client'
import type { CareerEncountersResponse, TeammateOption } from '@/lib/api/types'
import { queryKeys } from '@/lib/query/keys'
import { useAppShellStore } from '@/stores/appShellStore'

import {
  annuaireDeComposition,
  coequipiersProposes,
  type JoueurIdentifie,
} from './compositionOptions.logic'

/** Durée de fraîcheur des joueurs croisés — celle de la Carrière. */
const FRAICHEUR_RENCONTRES_MS = 5 * 60 * 1000

export interface CompositionOptions {
  /** La liste du sélecteur (amis déclarés, ou coéquipiers croisés), bots exclus. */
  options: TeammateOption[]
  /** Chaque joueur humain connu avec son xuid : traduit une composition en xuids. */
  annuaire: TeammateOption[]
  /** Les lectures dont dépend l'annuaire ont répondu (succès ou échec). */
  chargees: boolean
  /** Xuid du joueur consulté ; vide tant que son profil n'est pas connu. */
  joueurXuid: string
}

export function useCompositionOptions(playerSlug: string): CompositionOptions {
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  const profils = useAppShellStore((s) => s.availablePlayers)
  const avecIdentite = useAppShellStore((s) => !!s.linkedHaloIdentity)
  const rencontres = useQuery({
    queryKey: queryKeys.careerEncounters(playerSlug, titleSlug),
    queryFn: () =>
      api.get<CareerEncountersResponse>(`/players/${playerSlug}/pages/career/encounters`),
    enabled: !!playerSlug,
    staleTime: FRAICHEUR_RENCONTRES_MS,
  })
  const amis = usePlayerFriends(playerSlug)
  const escouades = useMySquads(playerSlug)
  const groupes = useMyGroups(avecIdentite)

  const joueur = useMemo(() => {
    const profil = profils.find((p) => p.player_slug.toLowerCase() === playerSlug.toLowerCase())
    return { gamertag: profil?.gamertag ?? playerSlug, xuid: profil?.xuid ?? '' }
  }, [profils, playerSlug])

  // MÉMOÏSÉ sur les réponses : un tableau neuf à chaque rendu recalculerait, sur la page
  // Tactique, la composition puis les paramètres et l'empreinte de chaque lecture.
  const { options, annuaire } = useMemo(() => {
    const identifies: JoueurIdentifie[] = [
      ...profils,
      ...(escouades.data?.squads ?? []).flatMap((s) => s.members),
      ...(groupes.data ?? []).flatMap((g) => g.members),
    ]
    const sources = {
      coequipiers: rencontres.data?.teammates ?? [],
      adversaires: rencontres.data?.enemies ?? [],
      amis: amis.data?.gamertags ?? [],
      identifies,
      joueur,
    }
    const connus = annuaireDeComposition(sources)
    return { options: coequipiersProposes(sources, connus), annuaire: connus }
  }, [rencontres.data, amis.data, escouades.data, groupes.data, profils, joueur])

  const reglee = (q: { isSuccess: boolean; isError: boolean }) => q.isSuccess || q.isError
  const chargees = reglee(rencontres) && reglee(amis) && reglee(escouades)
  return { options, annuaire, chargees, joueurXuid: joueur.xuid }
}
