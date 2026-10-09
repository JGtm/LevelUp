/**
 * useCompositionOptions — les coéquipiers proposés au sélecteur de composition
 * (`SquadCompositionPicker`) hors de la page Escouade, l'annuaire qui traduit une composition en
 * xuids, et le xuid du joueur consulté. Règles de la liste : `compositionOptions.logic.ts`.
 *
 * Les lectures : joueurs croisés, top des coéquipiers des matchs avec amis (la source de l'Escouade),
 * amis déclarés, escouades enregistrées et groupes.
 *
 * MÊMES CLÉS DE CACHE QUE LES PAGES QUI LISENT DÉJÀ CES RÉPONSES (`careerEncounters` de la
 * Carrière, `playerFriends`, escouades et groupes de `useSquadPresets`) : une seule entrée de cache
 * par réponse, jamais deux requêtes pour une seule liste.
 */
import { useEffect, useMemo } from 'react'
import { useQuery } from '@tanstack/react-query'

import { usePlayerFriends } from '@/features/friends/queries'
import { useMyGroups } from '@/features/groups/queries'
import { useMySquads } from '@/features/prestige/hooks/useSquads'
import { api } from '@/lib/api/client'
import type { CareerEncountersResponse, SquadPageResponse, TeammateOption } from '@/lib/api/types'
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
  /**
   * TOUTES les lectures dont dépend l'annuaire ont répondu — joueurs croisés, amis, escouades et,
   * avec une identité liée, groupes. Un échec définitif compte comme une réponse (journalisé) :
   * l'annuaire s'en passe plutôt que d'attendre sans fin.
   */
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
  // La source de l'Escouade : sans `teammate`, le service ne lit que le top des coéquipiers des
  // matchs avec amis (`SquadService.GetSquadPage`), rien de plus.
  const avecAmis = useQuery({
    queryKey: queryKeys.teammatesTop(playerSlug, titleSlug),
    queryFn: () => api.get<SquadPageResponse>(`/players/${playerSlug}/pages/squad`),
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
      avecAmis: avecAmis.data?.top_teammates ?? [],
      amis: amis.data?.gamertags ?? [],
      identifies,
      joueur,
    }
    const connus = annuaireDeComposition(sources)
    return { options: coequipiersProposes(sources, connus), annuaire: connus }
  }, [rencontres.data, avecAmis.data, amis.data, escouades.data, groupes.data, profils, joueur])

  useEchecJournalise('rencontres', rencontres)
  useEchecJournalise('coequipiers_avec_amis', avecAmis)
  useEchecJournalise('amis', amis)
  useEchecJournalise('escouades', escouades)
  useEchecJournalise('groupes', groupes)
  // Sans identité liée, la lecture des groupes est désactivée : il n'y a rien à attendre.
  const chargees =
    reglee(rencontres) &&
    reglee(avecAmis) &&
    reglee(amis) &&
    reglee(escouades) &&
    (!avecIdentite || reglee(groupes))
  return { options, annuaire, chargees, joueurXuid: joueur.xuid }
}

interface EtatDeLecture {
  isSuccess: boolean
  isError: boolean
  error: unknown
}

/** reglee — la lecture a répondu, par un succès ou par un échec définitif. */
function reglee(lecture: EtatDeLecture): boolean {
  return lecture.isSuccess || lecture.isError
}

/** useEchecJournalise — dit l'échec définitif d'une source de l'annuaire, une fois par erreur. */
function useEchecJournalise(source: string, lecture: EtatDeLecture): void {
  const { isError, error } = lecture
  useEffect(() => {
    if (isError) {
      console.warn('[composition] source de l’annuaire en échec — la composition s’en passe', {
        source,
        err: error,
      })
    }
  }, [source, isError, error])
}
