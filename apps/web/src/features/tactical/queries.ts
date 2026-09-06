/**
 * queries — les lectures TanStack Query de l'onglet Tactique.
 *
 * TROIS LECTURES, TROIS CLÉS, et elles ne s'invalident pas ensemble :
 *   - le PÉRIMÈTRE (les `match_id` de la sélection) dépend de la barre L2 ;
 *   - la GRILLE dépend du périmètre et de la composition ;
 *   - le FOND d'une carte ne dépend de rien : il est figé entre deux cuissons.
 *
 * ─── LE PÉRIMÈTRE EST RÉSOLU CÔTÉ CLIENT, COMME L'EXPLORATEUR ────────────────
 *
 * La barre produit un `FilterContextInput` ; `/filters/match-ids` le résout sur la
 * base JOUEUR (période OU sessions épinglées, contexte solo/escouade, cascade) et
 * rend les `match_id` ; l'onglet les poste en LISTE BLANCHE. Une seule définition du
 * périmètre dans l'app — et c'est la seule qui sache lire les sessions, que les
 * requêtes shared du lecteur tactique ne joignent pas.
 */
import { useQuery } from '@tanstack/react-query'

import { api } from '@/lib/api/client'
import type {
  CareerEncountersResponse,
  FilterContextInput,
  FilterMatchIdsResponse,
  TacticalMapsBody,
  TacticalMapsPage,
  TeammateOption,
} from '@/lib/api/types'
import { queryKeys } from '@/lib/query/keys'
import { useAppShellStore } from '@/stores/appShellStore'

/** FNV-1a 32 bits — même algorithme que `useFiltersPreview` et `computeHash`. */
export function hashFiltre(valeur: unknown): string {
  const s = JSON.stringify(valeur) ?? ''
  let h = 0x811c9dc5
  for (let i = 0; i < s.length; i++) {
    h ^= s.charCodeAt(i)
    h = Math.imul(h, 0x01000193) >>> 0
  }
  return h.toString(16).padStart(8, '0')
}

/**
 * useTacticalMatchIDs — le PÉRIMÈTRE : les `match_id` de la sélection courante.
 *
 * Même endpoint et même pipeline que le bouton « Voir les matchs » de l'omnibar :
 * `match_context`, sessions, période et cascade y sont tous honorés. Une liste VIDE
 * est une réponse légitime (le filtre ne retient rien) et NON une absence de
 * réponse : les lectures qui la consomment servent alors une grille vide.
 *
 * UN ÉCHEC N'EST PAS AVALÉ. Il est journalisé ici puis propagé : l'appelant doit
 * pouvoir DIRE que la lecture a échoué, là où une résolution muette laissait la page
 * afficher « aucune carte » pour toujours.
 */
export function useTacticalMatchIDs(playerSlug: string, contexte: FilterContextInput) {
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  return useQuery({
    queryKey: queryKeys.tacticalMatchIDs(playerSlug, titleSlug, hashFiltre(contexte)),
    queryFn: async () => {
      try {
        return await api.post<FilterMatchIdsResponse>(
          `/players/${playerSlug}/filters/match-ids`,
          contexte,
        )
      } catch (err) {
        console.error('[tactique] résolution du périmètre en échec', err)
        throw err
      }
    },
    enabled: !!playerSlug,
    staleTime: 2 * 60 * 1000,
  })
}

/**
 * useTacticalMaps — la grille des cartes jouées DANS LE PÉRIMÈTRE.
 *
 * POST : la liste de `match_id` ne tient pas dans une query string. La clé de cache
 * porte l'empreinte du périmètre ET de la composition — deux périmètres différents
 * ne doivent jamais se resservir l'un l'autre.
 *
 * `matchIDs` à `null` = le périmètre n'est pas encore résolu (ou une composition
 * n'est pas traduisible) : la requête N'EST PAS lancée, et l'appelant rend un état
 * d'ATTENTE — jamais un état vide, qui se lirait comme un résultat.
 *
 * Le corps est typé par le CONTRAT GÉNÉRÉ (`TacticalMapsBody`) : renommer un champ
 * côté Go doit casser `tsc` ici, pas se découvrir à l'exécution.
 */
export function useTacticalMaps(
  playerSlug: string,
  matchIDs: string[] | null,
  coequipiers: string[],
) {
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  const corps: TacticalMapsBody = { match_ids: matchIDs ?? [], coequipiers }
  return useQuery({
    queryKey: queryKeys.tacticalMaps(playerSlug, titleSlug, hashFiltre(corps)),
    queryFn: () => api.post<TacticalMapsPage>(`/players/${playerSlug}/tactical/maps`, corps),
    enabled: !!playerSlug && matchIDs !== null,
    staleTime: 2 * 60 * 1000,
  })
}

/**
 * useCoequipierOptions — les coéquipiers proposés au sélecteur de composition,
 * AVEC LEUR XUID (le serveur ne connaît que celui-là).
 *
 * Source : `/pages/career/encounters`, la liste des joueurs croisés le plus souvent
 * COMME COÉQUIPIERS — amis compris, contrairement à `top-encounters` qui les exclut
 * par construction et qui serait donc la mauvaise liste pour choisir une escouade.
 *
 * MÊME CLÉ DE CACHE QUE LA PAGE CARRIÈRE (`queryKeys.careerEncounters`) : c'est le
 * même endpoint et la même réponse, donc UNE seule entrée de cache et jamais deux
 * requêtes pour une seule liste. Le hook de la carrière n'est pas importé — ce
 * serait une dépendance croisée entre features pour trois lignes de projection.
 */
export function useCoequipierOptions(playerSlug: string): {
  options: TeammateOption[]
  chargees: boolean
} {
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  const { data, isSuccess } = useQuery({
    queryKey: queryKeys.careerEncounters(playerSlug, titleSlug),
    queryFn: () =>
      api.get<CareerEncountersResponse>(`/players/${playerSlug}/pages/career/encounters`),
    enabled: !!playerSlug,
    staleTime: 5 * 60 * 1000,
  })
  const options = (data?.teammates ?? []).map((t) => ({
    gamertag: t.gamertag,
    xuid: t.xuid,
    encounter_count: t.match_count,
  }))
  return { options, chargees: isSuccess }
}

/**
 * useTacticalMapBackgroundUrl — l'URL affichable du fond d'une carte, ou `null`.
 *
 * L'URL D'OBJET **EST** L'ENTRÉE DE CACHE, et c'est un choix, pas une facilité. Elle est
 * créée une fois dans la requête et gardée pour la session (`staleTime` et `gcTime`
 * infinis) : une image de carte ne change qu'à une re-cuisson, et revenir sur l'onglet doit
 * réafficher la grille sans re-télécharger. La contrepartie assumée est qu'aucune URL n'est
 * révoquée avant la fermeture de l'onglet ; créer l'URL au montage de chaque vignette
 * aurait imposé un `setState` dans un effet — cascade de rendus que le lint du dépôt
 * signale, pour un gain nul à cette échelle.
 *
 * LA BORNE EST LE NOMBRE DE CARTES DU TITRE, quel que soit le nombre de joueurs consultés
 * (revue R1, W4) : la clé de cache ne porte PAS le joueur. Une image de carte est une donnée
 * de référence du titre, identique pour tout le monde ; la mettre en cache par joueur
 * retenait N fois le même contenu. Le joueur reste dans l'URL de fetch, parce que la route
 * est derrière l'ownership.
 *
 * 404 = la carte n'a pas de fond figé : cas NOMINAL (toutes les cartes n'en ont pas). Pas
 * de nouvelle tentative, rien à dire à l'utilisateur — la vignette s'affiche sans image.
 */
export function useTacticalMapBackgroundUrl(playerSlug: string, mapId: string): string | null {
  const titleSlug = useAppShellStore((s) => s.currentTitleSlug)
  const { data } = useQuery({
    queryKey: queryKeys.tacticalMapBackground(titleSlug, mapId),
    queryFn: async () => {
      const blob = await api.getBlob(
        `/players/${playerSlug}/tactical/${encodeURIComponent(mapId)}/background.png`,
      )
      // `createObjectURL` manque dans certains environnements de test : l'absence de fond
      // n'est pas une panne de grille, la vignette s'affiche sans image.
      if (typeof URL.createObjectURL !== 'function') return null
      return URL.createObjectURL(blob)
    },
    staleTime: Infinity,
    gcTime: Infinity,
    enabled: !!playerSlug && !!mapId,
    retry: false,
  })
  return data ?? null
}
